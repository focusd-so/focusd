package usage_test

import (
	"context"
	"fmt"
	urlpkg "net/url"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/focusd-so/focusd/internal/usage"
)

func memoryDSN(t *testing.T) string {
	t.Helper()

	return fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", urlpkg.QueryEscape(t.Name()))
}

func setUpService(t *testing.T, options ...usage.Option) (*usage.Service, *gorm.DB) {
	db, _ := gorm.Open(sqlite.Open(memoryDSN(t)), &gorm.Config{})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	service, err := usage.NewService(ctx, db, options...)
	require.NoError(t, err, "failed to create service")

	return service, db
}

func TestService_TransitionsBetweenIdleAndApplicationUsage(t *testing.T) {
	h := newUsageHarness(t)

	h.
		EnterIdle().
		AssertLastUsage(
			assertUsageOpened(t),
			assertUsageApplicationName(t, usage.IdleApplicationName),
			assertUsageClassification(t, usage.ClassificationNone),
		).
		AssertUpdateEventsCount(1).
		AssertUsagesCount(1).
		Await(1 * time.Second).
		EnterIdle().
		AssertUpdateEventsCount(1).
		AssertUsagesCount(1)

	h.
		EnterIdle().
		TitleChanged("Chrome", "Github", withPtr("https://github.com")).
		AssertLastUsage(
			assertUsageOpened(t),
			assertUsageApplicationName(t, "Chrome"),
			assertUsageHostname(t, "github.com"),
			assertUsageClassification(t, usage.ClassificationProductive),
		).
		AssertPreviousUsage(assertUsageClosed(t)).
		AssertUpdateEventsCount(4).
		AssertUsagesCount(2)

	h.
		Await(3 * time.Second).
		EnterIdle().
		AssertLastUsage(assertUsageOpened(t)).
		AssertPreviousUsage(assertUsageClosed(t)).
		AssertUpdateEventsCount(6).
		AssertUsagesCount(3)
}

func TestService_ProtectionPauseAll(t *testing.T) {
	h := newUsageHarness(t)

	// user opens amazon, gets blocked by obviously classifier since it's a distraction
	h.
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertUsageClosed(t),
			assertUsageApplicationName(t, "Chrome"),
			assertUsageHostname(t, "amazon.com"),
			assertUsageClassification(t, usage.ClassificationDistracting),
			assertTerminationMode(t, usage.TerminationModeBlock),
			assertTerminationModeSource(t, usage.TerminationModeSourceApplication),
		).
		AssertUpdateEventsCount(2).
		AssertBlockerEventsCount(1)

	// user pauses all protection and opens amazon again, should not be blocked
	h.
		ResetBlockerEvents().
		ResetUsageEvents().
		Pause(5, "test").
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertUsageApplicationName(t, "Chrome"),
			assertUsageHostname(t, "amazon.com"),
			assertUsageClassification(t, usage.ClassificationDistracting),
			assertTerminationMode(t, usage.TerminationModePaused),
			assertTerminationModeSource(t, usage.TerminationModeSourcePaused),
		).
		AssertUpdateEventsCount(2).
		AssertBlockerEventsCount(0)

	// wait 6 seconds to collapse and see the distraction blocked again
	h.
		ResetBlockerEvents().
		ResetUsageEvents().
		Await(6*time.Second).
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertUsageApplicationName(t, "Chrome"),
			assertUsageHostname(t, "amazon.com"),
			assertUsageClassification(t, usage.ClassificationDistracting),
			assertTerminationMode(t, usage.TerminationModeBlock),
			assertTerminationModeSource(t, usage.TerminationModeSourceApplication),
		).
		AssertUpdateEventsCount(1).
		AssertBlockerEventsCount(1)

}

func TestService_TitleChanged_WhenSameApplication_ContinueCurrentApplicationUsage(t *testing.T) {
	h := newUsageHarness(t)

	// create a new application usage
	applicationUsage := usage.ApplicationUsage{
		StartedAt:   time.Now().Unix(),
		WindowTitle: "Slack",
		Application: usage.Application{Name: "Slack"},
	}
	if err := h.db.Create(&applicationUsage).Error; err != nil {
		t.Fatalf("failed to create application usage: %v", err)
	}

	// change the title of the current application
	h.TitleChangedRaw("/Applications/Slack.app/Contents/MacOS/Slack", "Slack", "Slack", "", nil, nil, nil)

	// read the application usage
	var readApplicationUsage usage.ApplicationUsage
	if err := h.db.Where("id = ?", applicationUsage.ID).First(&readApplicationUsage).Error; err != nil {
		t.Fatalf("failed to find application usage: %v", err)
	}

	require.NotEqual(t, 0, readApplicationUsage.StartedAt)
	require.Nil(t, readApplicationUsage.EndedAt)
	require.Nil(t, readApplicationUsage.DurationSeconds)
}

func TestService_TitleChanged_WhenDifferentApplication_CloseCurrentApplicationUsage(t *testing.T) {
	h := newUsageHarness(t)

	// create a new application usage
	applicationUsage := usage.ApplicationUsage{
		StartedAt:   time.Now().Unix(),
		WindowTitle: "Slack",
		Application: usage.Application{Name: "Slack"},
	}
	if err := h.db.Create(&applicationUsage).Error; err != nil {
		t.Fatalf("failed to create application usage: %v", err)
	}

	// change the title of the current application
	h.TitleChangedRaw("com.apple.Safari", "Safari", "New Tab", "", nil, nil, nil)

	// read the application usage
	var readApplicationUsage usage.ApplicationUsage
	if err := h.db.Where("id = ?", applicationUsage.ID).First(&readApplicationUsage).Error; err != nil {
		t.Fatalf("failed to find application usage: %v", err)
	}

	require.NotNil(t, readApplicationUsage.EndedAt)
}

func TestService_TitleChanged_ClassificationErrorStored(t *testing.T) {
	h := newUsageHarness(t)
	viper.Set("dummy_classification_response", "{")

	h.TitleChangedRaw("com.apple.Safari", "Safari", "New Tab", "", nil, nil, nil)

	var readApplicationUsage usage.ApplicationUsage
	if err := h.db.Where("id = ?", 1).First(&readApplicationUsage).Error; err != nil {
		t.Fatalf("failed to find application usage: %v", err)
	}

	require.NotNil(t, readApplicationUsage.ClassificationError)
	require.Contains(t, *readApplicationUsage.ClassificationError, "failed to classify application usage with LLM")
}

func TestService_TitleChanged_StripsWWWInURLComparison(t *testing.T) {
	h := newUsageHarness(t)

	app := usage.Application{Name: "Google Chrome"}
	h.retryLocked(func() error {
		return h.db.Create(&app).Error
	})

	initialURL := "https://youtube.com/watch?v=abc"
	applicationUsage := usage.ApplicationUsage{
		StartedAt:     time.Now().Unix(),
		WindowTitle:   "YouTube",
		BrowserURL:    &initialURL,
		ApplicationID: app.ID,
	}
	h.retryLocked(func() error {
		return h.db.Create(&applicationUsage).Error
	})

	newURL := "https://www.youtube.com/watch?v=abc"
	h.TitleChangedRaw("Google Chrome", "YouTube", "Google Chrome", "", nil, &newURL, nil)

	var count int64
	h.retryLocked(func() error {
		return h.db.Model(&usage.ApplicationUsage{}).Count(&count).Error
	})
	require.Equal(t, int64(1), count)

	var readUsage usage.ApplicationUsage
	h.retryLocked(func() error {
		return h.db.Where("id = ?", applicationUsage.ID).First(&readUsage).Error
	})
	require.Nil(t, readUsage.EndedAt)
	require.NotNil(t, readUsage.BrowserURL)
	require.Equal(t, "https://youtube.com/watch?v=abc", *readUsage.BrowserURL)
}

func TestService_TitleChanged_PreservesRawURLForBlocking(t *testing.T) {
	h := newUsageHarness(t)

	newURL := "https://www.focusd.so/blocked?d=123"
	h.TitleChangedRaw("Google Chrome", "Blocked", "Google Chrome", "", nil, &newURL, nil)

	var readUsage usage.ApplicationUsage
	h.retryLocked(func() error {
		return h.db.Where("id = ?", 1).First(&readUsage).Error
	})
	require.NotNil(t, readUsage.BrowserURL)
	require.Equal(t, "https://www.focusd.so/blocked?d=123", *readUsage.BrowserURL)
}
