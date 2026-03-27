package usage_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
	"github.com/focusd-so/focusd/gen/api/v1/apiv1connect"
	"github.com/focusd-so/focusd/internal/identity"
	"github.com/focusd-so/focusd/internal/settings"
	"github.com/focusd-so/focusd/internal/usage"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type usageHarness struct {
	t       *testing.T
	service *usage.Service
	db      *gorm.DB

	mu            sync.Mutex
	usageEvents   []*usage.ApplicationUsage
	pausedEvents  []usage.ProtectionPause
	resumedEvents []usage.ProtectionPause
}

type usageHarnessConfig struct {
	customRulesJS  string
	llmResponse    usage.ClassificationResponse
	llmResponseRaw *string
	accountTier    *apiv1.DeviceHandshakeResponse_AccountTier
}

type usageHarnessOption func(*usageHarnessConfig)

func withCustomRulesJS(customRules string) usageHarnessOption {
	return func(cfg *usageHarnessConfig) {
		cfg.customRulesJS = customRules
	}
}

func withDummyLLMResponse(resp usage.ClassificationResponse) usageHarnessOption {
	return func(cfg *usageHarnessConfig) {
		cfg.llmResponse = resp
	}
}

func withDummyLLMResponseRaw(resp string) usageHarnessOption {
	return func(cfg *usageHarnessConfig) {
		cfg.llmResponseRaw = &resp
	}
}

func withAccountTier(tier apiv1.DeviceHandshakeResponse_AccountTier) usageHarnessOption {
	return func(cfg *usageHarnessConfig) {
		cfg.accountTier = &tier
	}
}

func newUsageHarness(t *testing.T, opts ...usageHarnessOption) *usageHarness {
	t.Helper()

	cfg := usageHarnessConfig{
		llmResponse: usage.ClassificationResponse{
			Classification:       usage.ClassificationNone,
			ClassificationSource: usage.ClassificationSourceCloudLLMOpenAI,
			Reasoning:            "dummy integration classification",
			ConfidenceScore:      1,
			Tags:                 []string{"other"},
		},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	overrideTestConfig(t, cfg)
	stubFaviconFetcher(t)
	if cfg.accountTier != nil {
		setTestAccountTier(t, *cfg.accountTier)
	}

	db, err := gorm.Open(sqlite.Open(memoryDSNForHarness(t)), &gorm.Config{})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	h := &usageHarness{t: t, db: db}

	h.service, err = usage.NewService(ctx, db)
	require.NoError(t, err)

	h.service.OnUsageUpdated(func(appUsage *usage.ApplicationUsage) {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.usageEvents = append(h.usageEvents, appUsage)
	})

	h.service.OnProtectionPause(func(pause usage.ProtectionPause) {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.pausedEvents = append(h.pausedEvents, pause)
	})

	h.service.OnProtectionResumed(func(pause usage.ProtectionPause) {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.resumedEvents = append(h.resumedEvents, pause)
	})

	return h
}

type testAPIService struct {
	apiv1connect.UnimplementedApiServiceHandler
	tier apiv1.DeviceHandshakeResponse_AccountTier
}

func (m testAPIService) DeviceHandshake(_ context.Context, _ *connect.Request[apiv1.DeviceHandshakeRequest]) (*connect.Response[apiv1.DeviceHandshakeResponse], error) {
	return connect.NewResponse(&apiv1.DeviceHandshakeResponse{
		UserId:       1,
		SessionToken: "test-session-token",
		AccountTier:  m.tier,
	}), nil
}

func setTestAccountTier(t *testing.T, tier apiv1.DeviceHandshakeResponse_AccountTier) {
	t.Helper()

	prevTier := identity.GetAccountTier()

	mux := http.NewServeMux()
	_, handler := apiv1connect.NewApiServiceHandler(testAPIService{tier: tier})
	mux.Handle("/", handler)

	server := httptest.NewServer(mux)
	defer server.Close()

	client := apiv1connect.NewApiServiceClient(server.Client(), server.URL)
	require.NoError(t, identity.PerformHandshake(context.Background(), client))

	t.Cleanup(func() {
		mux := http.NewServeMux()
		_, handler := apiv1connect.NewApiServiceHandler(testAPIService{tier: prevTier})
		mux.Handle("/", handler)

		server := httptest.NewServer(mux)
		defer server.Close()

		client := apiv1connect.NewApiServiceClient(server.Client(), server.URL)
		require.NoError(t, identity.PerformHandshake(context.Background(), client))
	})
}

func memoryDSNForHarness(t *testing.T) string {
	t.Helper()

	return fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", url.QueryEscape(t.Name()))
}

func (h *usageHarness) TitleChanged(appName, windowTitle string, browserURL *string) *usageHarness {
	h.t.Helper()
	h.retryLocked(func() error {
		_, err := h.service.TitleChanged(context.Background(), appName, windowTitle, appName, "", nil, browserURL, nil)
		return err
	})
	return h
}

func (h *usageHarness) Await(dur time.Duration) *usageHarness {
	h.t.Helper()
	time.Sleep(dur)
	return h
}

func (h *usageHarness) TitleChangedRaw(executablePath, windowTitle, appName, icon string, bundleID, browserURL, appCategory *string) *usageHarness {
	h.t.Helper()
	h.retryLocked(func() error {
		_, err := h.service.TitleChanged(context.Background(), executablePath, windowTitle, appName, icon, bundleID, browserURL, appCategory)
		return err
	})
	return h
}

func (h *usageHarness) EnterIdle() *usageHarness {
	h.t.Helper()
	h.IdleChanged(true)
	return h
}

func (h *usageHarness) IdleChanged(isIdle bool) *usageHarness {
	h.t.Helper()
	h.retryLocked(func() error {
		_, err := h.service.IdleChanged(context.Background(), isIdle)
		return err
	})
	return h
}

func (h *usageHarness) Pause(durationSeconds int, reason string) *usageHarness {
	h.t.Helper()
	h.retryLocked(func() error {
		_, err := h.service.PauseProtection(durationSeconds, reason)
		return err
	})
	return h
}

func (h *usageHarness) Resume(reason string) *usageHarness {
	h.t.Helper()
	h.retryLocked(func() error {
		_, err := h.service.ResumeProtection(reason)
		return err
	})
	return h
}

func (h *usageHarness) Whitelist(appName, hostname string, duration time.Duration) *usageHarness {
	h.t.Helper()
	h.retryLocked(func() error {
		return h.service.Whitelist(appName, hostname, duration)
	})
	return h
}

func (h *usageHarness) RemoveActiveWhitelists() *usageHarness {
	h.t.Helper()
	h.retryLocked(func() error {
		wls, err := h.service.GetWhitelist()
		if err != nil {
			return err
		}
		for _, wl := range wls {
			if err := h.service.RemoveWhitelist(wl.ID); err != nil {
				return err
			}
		}
		return nil
	})
	return h
}

func (h *usageHarness) UsageList() []usage.ApplicationUsage {
	h.t.Helper()
	var (
		usages []usage.ApplicationUsage
		err    error
	)
	h.retryLocked(func() error {
		usages, err = h.service.GetUsageList(usage.GetUsageListOptions{})
		return err
	})
	return usages
}

func (h *usageHarness) OpenUsage() usage.ApplicationUsage {
	h.t.Helper()

	var appUsage usage.ApplicationUsage
	h.retryLocked(func() error {
		return h.db.Preload("Application").Preload("Tags").Where("ended_at IS NULL").Order("started_at DESC").First(&appUsage).Error
	})
	return appUsage
}

func (h *usageHarness) LastUsageByTitle(windowTitle string) usage.ApplicationUsage {
	h.t.Helper()

	var appUsage usage.ApplicationUsage
	h.retryLocked(func() error {
		return h.db.Preload("Application").Preload("Tags").Where("window_title = ?", windowTitle).Order("started_at DESC").First(&appUsage).Error
	})
	return appUsage
}

func (h *usageHarness) AssertLastUsage(check ...func(*usage.ApplicationUsage)) *usageHarness {
	h.t.Helper()

	var appUsage usage.ApplicationUsage
	h.retryLocked(func() error {
		return h.db.Preload("Application").Preload("Tags").Order("id DESC").First(&appUsage).Error
	})

	for _, c := range check {
		c(&appUsage)
	}

	return h
}

func (h *usageHarness) AssertPreviousUsage(check ...func(*usage.ApplicationUsage)) *usageHarness {
	h.t.Helper()

	var appUsages []usage.ApplicationUsage
	h.retryLocked(func() error {
		return h.db.Preload("Application").Preload("Tags").Order("id DESC").Limit(2).Find(&appUsages).Error
	})

	if len(appUsages) < 2 {
		for _, c := range check {
			c(nil)
		}
	} else {
		for _, c := range check {
			c(&appUsages[1])
		}
	}

	return h
}

func (h *usageHarness) AssertUpdateEventsCount(count int) *usageHarness {
	h.t.Helper()

	require.Equal(h.t, count, len(h.UsageEvents()))
	return h
}

func (h *usageHarness) AssertUsagesCount(count int) *usageHarness {
	h.t.Helper()

	require.Equal(h.t, count, h.CountUsages())
	return h
}

func (h *usageHarness) CountUsages() int {
	h.t.Helper()

	var count int64
	h.retryLocked(func() error {
		return h.db.Model(&usage.ApplicationUsage{}).Count(&count).Error
	})
	return int(count)
}

func (h *usageHarness) retryLocked(fn func() error) {
	h.t.Helper()

	deadline := time.Now().Add(1500 * time.Millisecond)
	for {
		err := fn()
		if err == nil {
			return
		}

		if !strings.Contains(err.Error(), "database table is locked") {
			require.NoError(h.t, err)
		}

		if time.Now().After(deadline) {
			require.NoError(h.t, err)
		}

		time.Sleep(20 * time.Millisecond)
	}
}

func (h *usageHarness) UsageEvents() []*usage.ApplicationUsage {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]*usage.ApplicationUsage(nil), h.usageEvents...)
}

func (h *usageHarness) ResetUsageEvents() *usageHarness {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.usageEvents = nil
	return h
}

func assertUsageApplicationName(t *testing.T, appName string) func(*usage.ApplicationUsage) {
	t.Helper()

	return func(u *usage.ApplicationUsage) {
		require.Equal(t, appName, u.Application.Name)
	}
}

func assertUsageHostname(t *testing.T, hostname string) func(*usage.ApplicationUsage) {
	t.Helper()

	return func(u *usage.ApplicationUsage) {
		require.Equal(t, hostname, fromPtr(u.Application.Hostname))
	}
}

func assertUsageClassification(t *testing.T, classification usage.Classification) func(*usage.ApplicationUsage) {
	t.Helper()

	return func(u *usage.ApplicationUsage) {
		require.Equal(t, classification, u.Classification)
	}
}

func assertUsageClassificationSource(t *testing.T, source usage.ClassificationSource) func(*usage.ApplicationUsage) {
	t.Helper()

	return func(u *usage.ApplicationUsage) {
		require.Equal(t, source, fromPtr(u.ClassificationSource))
	}
}

func assertUsageClassificationReasoning(t *testing.T, reasoning string) func(*usage.ApplicationUsage) {
	t.Helper()

	return func(u *usage.ApplicationUsage) {
		require.Equal(t, reasoning, fromPtr(u.ClassificationReasoning))
	}
}

func assertUsageClassificationConfidence(t *testing.T, confidence float32) func(*usage.ApplicationUsage) {
	t.Helper()

	return func(u *usage.ApplicationUsage) {
		require.Equal(t, confidence, fromPtr(u.ClassificationConfidence))
	}
}

func assertUsageTags(t *testing.T, tags ...string) func(*usage.ApplicationUsage) {
	t.Helper()

	return func(u *usage.ApplicationUsage) {
		actualTags := make([]string, len(u.Tags))
		for i, tag := range u.Tags {
			actualTags[i] = tag.Tag
		}
		require.ElementsMatch(t, tags, actualTags)
	}
}

func assertClassificationSandboxRecorded(t *testing.T) func(*usage.ApplicationUsage) {
	t.Helper()

	return func(u *usage.ApplicationUsage) {
		require.NotNil(t, u.ClassificationSandboxContext)
		require.NotNil(t, u.ClassificationSandboxResponse)
		require.NotNil(t, u.ClassificationSandboxLogs)
	}
}

func assertEnforcementAction(t *testing.T, mode usage.EnforcementAction) func(*usage.ApplicationUsage) {
	t.Helper()

	return func(u *usage.ApplicationUsage) {
		require.Equal(t, mode, u.EnforcementAction)
	}
}

func assertEnforcementSource(t *testing.T, source usage.EnforcementSource) func(*usage.ApplicationUsage) {
	t.Helper()

	return func(u *usage.ApplicationUsage) {
		require.Equal(t, source, fromPtr(u.EnforcementSource))
	}
}

func assertUsageOpened(t *testing.T) func(*usage.ApplicationUsage) {
	t.Helper()

	return func(u *usage.ApplicationUsage) {
		require.Nil(t, u.EndedAt)
	}
}

func assertUsageClosed(t *testing.T) func(*usage.ApplicationUsage) {
	t.Helper()

	return func(u *usage.ApplicationUsage) {
		require.NotNil(t, u.EndedAt)
	}
}

func overrideTestConfig(t *testing.T, cfg usageHarnessConfig) {
	t.Helper()

	type viperValue struct {
		isSet bool
		value any
	}

	keys := []string{"classification_llm_provider", "dummy_classification_response", "custom_rules_js"}
	snapshot := map[string]viperValue{}

	for _, key := range keys {
		snapshot[key] = viperValue{isSet: viper.IsSet(key), value: viper.Get(key)}
	}

	llmRespJSON := ""
	if cfg.llmResponseRaw != nil {
		llmRespJSON = *cfg.llmResponseRaw
	} else {
		respJSON, err := json.Marshal(cfg.llmResponse)
		require.NoError(t, err)
		llmRespJSON = string(respJSON)
	}

	viper.Set("classification_llm_provider", settings.LLMProviderDummy)
	viper.Set("dummy_classification_response", llmRespJSON)

	if cfg.customRulesJS == "" {
		viper.Set("custom_rules_js", []string{})
	} else {
		viper.Set("custom_rules_js", []string{base64.StdEncoding.EncodeToString([]byte(cfg.customRulesJS))})
	}

	t.Cleanup(func() {
		for key, val := range snapshot {
			if val.isSet {
				viper.Set(key, val.value)
				continue
			}
			viper.Set(key, nil)
		}
	})
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func stubFaviconFetcher(t *testing.T) {
	t.Helper()

	oldTransport := http.DefaultClient.Transport

	http.DefaultClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "www.google.com" && req.URL.Path == "/s2/favicons" {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("ico")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}

		if oldTransport != nil {
			return oldTransport.RoundTrip(req)
		}

		return http.DefaultTransport.RoundTrip(req)
	})

	t.Cleanup(func() {
		http.DefaultClient.Transport = oldTransport
	})
}

func withPtr[T any](v T) *T {
	// check if v is zero value
	if reflect.ValueOf(v).IsZero() {
		return nil
	}

	return &v
}

func fromPtr[T any](v *T) T {
	if v == nil {
		return *new(T)
	}

	return *v
}
