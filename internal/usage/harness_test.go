package usage_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

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

	mu               sync.Mutex
	usageEvents      []*usage.ApplicationUsage
	pausedEvents     []usage.ProtectionPause
	resumedEvents    []usage.ProtectionPause
	appBlockerEvents []appBlockerEvent
}

type appBlockerEvent struct {
	AppName    string
	Title      string
	Reason     string
	Tags       []string
	BrowserURL *string
}

type usageHarnessConfig struct {
	customRulesJS string
	llmResponse   usage.ClassificationResponse
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

func newUsageHarness(t *testing.T, opts ...usageHarnessOption) *usageHarness {
	t.Helper()

	cfg := usageHarnessConfig{
		llmResponse: usage.ClassificationResponse{
			Classification:  usage.ClassificationNone,
			Reasoning:       "dummy integration classification",
			ConfidenceScore: 1,
			Tags:            []string{"other"},
		},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	overrideTestConfig(t, cfg)
	stubFaviconFetcher(t)

	db, err := gorm.Open(sqlite.Open(memoryDSNForHarness(t)), &gorm.Config{})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	h := &usageHarness{t: t, db: db}

	h.service, err = usage.NewService(
		ctx,
		db,
		usage.WithAppBlocker(func(appName, title, reason string, tags []string, browserURL *string) {
			h.mu.Lock()
			defer h.mu.Unlock()

			h.appBlockerEvents = append(h.appBlockerEvents, appBlockerEvent{
				AppName:    appName,
				Title:      title,
				Reason:     reason,
				Tags:       append([]string(nil), tags...),
				BrowserURL: browserURL,
			})
		}),
	)
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

func (h *usageHarness) AssertBlockerEventsCount(count int) *usageHarness {
	h.t.Helper()

	require.Equal(h.t, count, len(h.BlockerEvents()))
	return h
}

func (h *usageHarness) AssertBlockerLastEvent(check ...func(event *appBlockerEvent)) *usageHarness {
	h.t.Helper()

	for _, c := range check {
		c(&h.appBlockerEvents[len(h.appBlockerEvents)-1])
	}

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

func (h *usageHarness) ResetBlockerEvents() *usageHarness {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.appBlockerEvents = nil
	return h
}

func (h *usageHarness) BlockerEvents() []appBlockerEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]appBlockerEvent(nil), h.appBlockerEvents...)
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

func assertTerminationMode(t *testing.T, mode usage.TerminationMode) func(*usage.ApplicationUsage) {
	t.Helper()

	return func(u *usage.ApplicationUsage) {
		require.Equal(t, mode, u.TerminationMode)
	}
}

func assertTerminationModeSource(t *testing.T, source usage.TerminationModeSource) func(*usage.ApplicationUsage) {
	t.Helper()

	return func(u *usage.ApplicationUsage) {
		require.Equal(t, source, fromPtr(u.TerminationSource))
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

	llmRespJSON, err := json.Marshal(cfg.llmResponse)
	require.NoError(t, err)

	viper.Set("classification_llm_provider", settings.LLMProviderDummy)
	viper.Set("dummy_classification_response", string(llmRespJSON))

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
