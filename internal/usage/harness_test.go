package usage_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
	"github.com/focusd-so/focusd/gen/api/v1/apiv1connect"
	"github.com/focusd-so/focusd/internal/identity"
	"github.com/focusd-so/focusd/internal/settings"
	"github.com/focusd-so/focusd/internal/timeline"
	"github.com/focusd-so/focusd/internal/usage"
)

type usageHarness struct {
	t       *testing.T
	service *usage.Service
	db      *gorm.DB

	timelineService *timeline.Service
}

type usageHarnessConfig struct {
	customRulesJS  string
	llmResponse    usage.ClassificationResult
	llmResponseRaw *string
	accountTier    *apiv1.DeviceHandshakeResponse_AccountTier
}

type usageHarnessOption func(*usageHarnessConfig)

func withCustomRulesJS(customRules string) usageHarnessOption {
	return func(cfg *usageHarnessConfig) {
		cfg.customRulesJS = customRules
	}
}

func withDummyLLMResponse(resp usage.ClassificationResult) usageHarnessOption {
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

func newHarness(t *testing.T, opts ...usageHarnessOption) *usageHarness {
	t.Helper()

	cfg := usageHarnessConfig{
		llmResponse: usage.ClassificationResult{
			LLMClassificationResult: &usage.LLMClassificationResult{
				BasicClassificationResult: usage.BasicClassificationResult{
					Classification:       usage.ClassificationNone,
					ClassificationReason: "dummy integration classification",
					Tags:                 []string{"other"},
				},
				ClassificationSource: usage.ClassificationSourceLLMOpenAI,
				ConfidenceScore:      1,
			},
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

	db.Migrator().AutoMigrate(&usage.Application{}, &usage.LLMDailySummary{})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	timelineService, err := timeline.NewService(db)
	require.NoError(t, err)

	h := &usageHarness{t: t, db: db, timelineService: timelineService}

	h.service, err = usage.NewService(ctx, db, timelineService)
	require.NoError(t, err)

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

	err := h.service.TitleChanged(context.Background(), appName, windowTitle, "", browserURL, nil)
	require.NoError(h.t, err)

	return h
}

func (h *usageHarness) Await(dur time.Duration) *usageHarness {
	h.t.Helper()
	time.Sleep(dur)
	return h
}

func (h *usageHarness) EnterIdle() *usageHarness {
	h.t.Helper()
	err := h.service.IdleChanged(context.Background(), true)
	require.NoError(h.t, err)

	return h
}

func (h *usageHarness) Pause(durationSeconds int, reason string) *usageHarness {
	h.t.Helper()
	err := h.service.ProtectionPause(durationSeconds, reason)
	require.NoError(h.t, err)

	return h
}

func (h *usageHarness) Resume(reason string) *usageHarness {
	h.t.Helper()
	err := h.service.ProtectionResume(reason)
	require.NoError(h.t, err)

	return h
}

func (h *usageHarness) AllowApp(appName string, duration time.Duration) *usageHarness {
	h.t.Helper()
	err := h.service.AllowApp(appName, duration)
	require.NoError(h.t, err)

	return h
}

func (h *usageHarness) AllowWebsite(rawURL string, duration time.Duration) *usageHarness {
	h.t.Helper()
	err := h.service.AllowHostname(rawURL, duration)
	require.NoError(h.t, err)

	return h
}

func (h *usageHarness) AllowURL(rawURL string, duration time.Duration) *usageHarness {
	h.t.Helper()
	err := h.service.AllowURL(rawURL, duration)
	require.NoError(h.t, err)

	return h
}

func (h *usageHarness) AssertApplicationCount(expected int) *usageHarness {
	h.t.Helper()

	var count int64
	err := h.db.Model(&usage.Application{}).Count(&count).Error
	require.NoError(h.t, err)

	require.Equal(h.t, int64(expected), count)

	return h
}

func (h *usageHarness) AssertApplicationExists(appName string) *usageHarness {
	h.t.Helper()

	var app usage.Application
	err := h.db.Where("name = ?", appName).First(&app).Error
	require.NoError(h.t, err)

	return h
}

func (h *usageHarness) AssertLastActiveEvent(types []string, fn func(*timeline.Event)) *usageHarness {
	h.t.Helper()

	event, err := h.timelineService.GetActiveEventOfTypes(types...)
	require.NoError(h.t, err)

	fn(event)

	return h
}

func (h *usageHarness) AssertPreviousEvent(types []string, fn func(*timeline.Event)) *usageHarness {
	h.t.Helper()

	events, err := h.timelineService.ListEvents(timeline.ByTypes(types...), timeline.Limit(2), timeline.OrderByOccurredAtDesc())
	require.NoError(h.t, err)

	if len(events) < 2 {
		fn(nil)
		return h
	}

	slog.Info("events", "events0", *events[0], "events1", *events[1])

	fn(events[1])

	return h
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

func fromPtr[T any](v *T) T {
	if v == nil {
		return *new(T)
	}

	return *v
}
