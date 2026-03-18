package usage_test

import (
	"context"
	"fmt"
	urlpkg "net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/focusd-so/focusd/internal/usage"
)

func memoryDSN(t *testing.T) string {
	t.Helper()

	return fmt.Sprintf("file:%s?mode=memory&cache=shared", urlpkg.QueryEscape(t.Name()))
}

func setUpService(t *testing.T, options ...usage.Option) (*usage.Service, *gorm.DB) {
	db, _ := gorm.Open(sqlite.Open(memoryDSN(t)), &gorm.Config{})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	service, err := usage.NewService(ctx, db, options...)
	require.NoError(t, err, "failed to create service")

	return service, db
}

type usageDriver struct {
	t       *testing.T
	service *usage.Service
	db      *gorm.DB
}

func newUsageDriver(t *testing.T, options ...usage.Option) *usageDriver {
	service, db := setUpService(t, options...)
	return &usageDriver{
		t:       t,
		service: service,
		db:      db,
	}
}

func (d *usageDriver) EnterIdle() *usageDriver {
	err := d.service.IdleChanged(context.Background(), true)
	require.NoError(d.t, err, "failed to enter idle")
	return d
}

func (d *usageDriver) ExitIdle() *usageDriver {
	err := d.service.IdleChanged(context.Background(), false)
	require.NoError(d.t, err, "failed to exit idle")
	return d
}

func (d *usageDriver) TitleChanged(path, bundleID, name, title string) *usageDriver {
	err := d.service.TitleChanged(context.Background(), path, bundleID, name, title, nil, nil, nil)
	require.NoError(d.t, err, "failed to change title")
	return d
}

func (d *usageDriver) AssertApplicationCount(expected int) *usageDriver {
	apps, err := d.service.GetApplicationList()
	require.NoError(d.t, err, "failed to get application list")
	require.Len(d.t, apps, expected, "unexpected application count")
	return d
}

func (d *usageDriver) AssertUsageCount(appName string, expected int) *usageDriver {
	var app usage.Application
	err := d.db.Where("name = ?", appName).First(&app).Error
	require.NoError(d.t, err, "failed to find application %s", appName)

	usages, err := d.service.GetUsageList(usage.GetUsageListOptions{
		ApplicationID: &app.ID,
	})
	require.NoError(d.t, err, "failed to get usage list for %s", appName)
	require.Len(d.t, usages, expected, "unexpected usage count for %s", appName)
	return d
}

func (d *usageDriver) AssertLastUsage(appName string, check func(*usage.ApplicationUsage)) *usageDriver {
	var app usage.Application
	err := d.db.Where("name = ?", appName).First(&app).Error
	require.NoError(d.t, err, "failed to find application %s", appName)

	var lastUsage usage.ApplicationUsage
	err = d.db.Where("application_id = ?", app.ID).Order("started_at desc").First(&lastUsage).Error
	require.NoError(d.t, err, "failed to find last usage for %s", appName)

	check(&lastUsage)
	return d
}

func TestService_IdleChanged_TransitionsBetweenIdleAndApplicationUsage(t *testing.T) {
	d := newUsageDriver(t)

	updateCalls := 0
	d.service.OnUsageUpdated(func(_ usage.ApplicationUsage) {
		updateCalls++
	})

	// Initial idle entry should create an idle usage record with no classification
	d.EnterIdle().
		AssertApplicationCount(1).
		AssertUsageCount(usage.IdleApplicationName, 1).
		AssertLastUsage(usage.IdleApplicationName, func(u *usage.ApplicationUsage) {
			require.Equal(t, usage.ClassificationNone, u.Classification)
			require.Nil(t, u.EndedAt)
		})

	// Ensure classification update callback was called once for the idle usage
	require.Equal(t, 1, updateCalls)

	// Re-entering idle should be idempotent
	d.EnterIdle().
		AssertApplicationCount(1).
		AssertUsageCount(usage.IdleApplicationName, 1)

	time.Sleep(2 * time.Second)

	// Switching from Idle to an application should close the Idle usage and
	// create a new application usage
	d.TitleChanged("/Applications/Slack.app/Contents/MacOS/Slack", "com.tinyspeck.slackmacgap", "Slack", "General").
		AssertApplicationCount(2).
		AssertUsageCount(usage.IdleApplicationName, 1).
		AssertLastUsage(usage.IdleApplicationName, func(u *usage.ApplicationUsage) {
			require.NotNil(t, u.EndedAt)
			require.NotNil(t, u.DurationSeconds)
			require.Equal(t, 2, *u.DurationSeconds)
		}).
		AssertUsageCount("Slack", 1).
		AssertLastUsage("Slack", func(u *usage.ApplicationUsage) {
			require.Nil(t, u.EndedAt)
		})

	// Switching from an application back to Idle should close the application
	// usage and create a new Idle usage
	time.Sleep(3 * time.Second)
	d.EnterIdle().
		AssertUsageCount(usage.IdleApplicationName, 2).
		AssertUsageCount("Slack", 1).
		AssertLastUsage("Slack", func(u *usage.ApplicationUsage) {
			require.NotNil(t, u.EndedAt)
			require.NotNil(t, u.DurationSeconds)
			require.Equal(t, 3, *u.DurationSeconds)
		})
}

func TestService_TitleChanged(t *testing.T) {
	d := newUsageDriver(t)

	d.TitleChanged("/Applications/Slack.app/Contents/MacOS/Slack", "com.tinyspeck.slackmacgap", "Slack", "General").
		AssertApplicationCount(1).
		AssertUsageCount("Slack", 1).
		AssertLastUsage("Slack", func(u *usage.ApplicationUsage) {
			require.Nil(t, u.EndedAt)
		})
}

func TestService_TitleChanged_WhenSameApplication_ContinueCurrentApplicationUsage(t *testing.T) {
	service, db := setUpService(t)

	ctx := context.Background()

	// create a new application usage
	applicationUsage := usage.ApplicationUsage{
		StartedAt:   time.Now().Unix(),
		WindowTitle: "Slack",
		Application: usage.Application{Name: "Slack"},
	}
	if err := db.Create(&applicationUsage).Error; err != nil {
		t.Fatalf("failed to create application usage: %v", err)
	}

	// change the title of the current application
	err := service.TitleChanged(ctx, "/Applications/Slack.app/Contents/MacOS/Slack", "Slack", "Slack", "", nil, nil, nil)
	require.NoError(t, err, "failed to change title")

	// read the application usage
	var readApplicationUsage usage.ApplicationUsage
	if err := db.Where("id = ?", applicationUsage.ID).First(&readApplicationUsage).Error; err != nil {
		t.Fatalf("failed to find application usage: %v", err)
	}

	require.NotEqual(t, 0, readApplicationUsage.StartedAt)
	require.Nil(t, readApplicationUsage.EndedAt)
	require.Nil(t, readApplicationUsage.DurationSeconds)
}

func TestService_TitleChanged_WhenDifferentApplication_CloseCurrentApplicationUsage(t *testing.T) {
	service, db := setUpService(t)

	ctx := context.Background()

	// create a new application usage
	applicationUsage := usage.ApplicationUsage{
		StartedAt:   time.Now().Unix(),
		WindowTitle: "Slack",
		Application: usage.Application{Name: "Slack"},
	}
	if err := db.Create(&applicationUsage).Error; err != nil {
		t.Fatalf("failed to create application usage: %v", err)
	}

	// change the title of the current application
	err := service.TitleChanged(ctx, "com.apple.Safari", "Safari", "New Tab", "", nil, nil, nil)
	require.NoError(t, err, "failed to change title")

	// read the application usage
	var readApplicationUsage usage.ApplicationUsage
	if err := db.Where("id = ?", applicationUsage.ID).First(&readApplicationUsage).Error; err != nil {
		t.Fatalf("failed to find application usage: %v", err)
	}

	require.NotNil(t, readApplicationUsage.EndedAt)
}

func TestService_TitleChanged_ClassificationErrorStored(t *testing.T) {
	// setup a service with a mock settings service that returns invalid custom rules to trigger a classification error
	usageService, db := setUpServiceWithSettings(t, "invalid custom rules")

	err := usageService.TitleChanged(context.Background(), "com.apple.Safari", "Safari", "New Tab", "", nil, nil, nil)
	require.Nil(t, err)

	var readApplicationUsage usage.ApplicationUsage
	if err := db.Where("id = ?", 1).First(&readApplicationUsage).Error; err != nil {
		t.Fatalf("failed to find application usage: %v", err)
	}

	require.NotNil(t, readApplicationUsage.ClassificationError)
	require.Contains(t, *readApplicationUsage.ClassificationError, "failed to classify application usage with custom rules")
}

func TestService_TitleChanged_StripsWWWInURLComparison(t *testing.T) {
	service, db := setUpService(t)

	app := usage.Application{Name: "Google Chrome"}
	require.NoError(t, db.Create(&app).Error)

	initialURL := "https://youtube.com/watch?v=abc"
	applicationUsage := usage.ApplicationUsage{
		StartedAt:     time.Now().Unix(),
		WindowTitle:   "YouTube",
		BrowserURL:    &initialURL,
		ApplicationID: app.ID,
	}
	require.NoError(t, db.Create(&applicationUsage).Error)

	newURL := "https://www.youtube.com/watch?v=abc"
	err := service.TitleChanged(context.Background(), "Google Chrome", "YouTube", "Google Chrome", "", nil, &newURL, nil)
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&usage.ApplicationUsage{}).Count(&count).Error)
	require.Equal(t, int64(1), count)

	var readUsage usage.ApplicationUsage
	require.NoError(t, db.Where("id = ?", applicationUsage.ID).First(&readUsage).Error)
	require.Nil(t, readUsage.EndedAt)
	require.NotNil(t, readUsage.BrowserURL)
	require.Equal(t, "https://www.youtube.com/watch?v=abc", *readUsage.BrowserURL)
}

func TestService_TitleChanged_PreservesRawURLForBlocking(t *testing.T) {
	service, db := setUpService(t)

	newURL := "https://www.focusd.so/blocked?d=123"
	err := service.TitleChanged(context.Background(), "Google Chrome", "Blocked", "Google Chrome", "", nil, &newURL, nil)
	require.NoError(t, err)

	var readUsage usage.ApplicationUsage
	require.NoError(t, db.Where("id = ?", 1).First(&readUsage).Error)
	require.NotNil(t, readUsage.BrowserURL)
	require.Equal(t, "https://www.focusd.so/blocked?d=123", *readUsage.BrowserURL)
}
