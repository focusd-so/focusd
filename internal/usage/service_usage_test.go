package usage_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/focusd-so/focusd/internal/usage"
)

func setUpService(t *testing.T, options ...usage.Option) (*usage.Service, *gorm.DB) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	service, err := usage.NewService(ctx, db, options...)
	require.NoError(t, err, "failed to create service")

	return service, db
}

func TestService_WhenEnterIdle_ContinueCurrentIdlePeriod(t *testing.T) {
	service, db := setUpService(t)

	// create the Idle application
	idleApp := usage.Application{Name: usage.IdleApplicationName}
	require.NoError(t, db.Create(&idleApp).Error)

	// create an open idle usage
	idleUsage := usage.ApplicationUsage{
		ApplicationID:   idleApp.ID,
		StartedAt:       time.Now().Unix(),
		Classification:  usage.ClassificationNone,
		TerminationMode: usage.TerminationModeNone,
		WindowTitle:     usage.IdleApplicationName,
		ExecutablePath:  "idle",
	}
	require.NoError(t, db.Create(&idleUsage).Error)

	// wait 3 seconds
	time.Sleep(3 * time.Second)

	// enter idle
	err := service.IdleChanged(context.Background(), true)
	require.NoError(t, err, "failed to enter idle")

	// read the idle usage — it should still be open
	var readIdleUsage usage.ApplicationUsage
	require.NoError(t, db.Where("id = ?", idleUsage.ID).First(&readIdleUsage).Error)

	require.NotEqual(t, int64(0), readIdleUsage.StartedAt)
	require.Equal(t, readIdleUsage.ID, idleUsage.ID)
	require.Nil(t, readIdleUsage.EndedAt)
	require.Nil(t, readIdleUsage.DurationSeconds)
}

func TestService_WhenEnterIdle_CreateNewIdlePeriod(t *testing.T) {
	service, db := setUpService(t)

	// create the Idle application
	idleApp := usage.Application{Name: usage.IdleApplicationName}
	require.NoError(t, db.Create(&idleApp).Error)

	now := time.Now().Unix()
	expectedDurationSeconds := 3

	// create a closed idle usage
	closedIdleUsage := usage.ApplicationUsage{
		ApplicationID:   idleApp.ID,
		StartedAt:       time.Now().Unix(),
		EndedAt:         &now,
		DurationSeconds: &expectedDurationSeconds,
		Classification:  usage.ClassificationNone,
		TerminationMode: usage.TerminationModeNone,
		WindowTitle:     usage.IdleApplicationName,
		ExecutablePath:  "idle",
	}
	require.NoError(t, db.Create(&closedIdleUsage).Error)

	// enter idle
	err := service.IdleChanged(context.Background(), true)
	require.NoError(t, err, "failed to enter idle")

	// read the new idle usage
	var readIdleUsage usage.ApplicationUsage
	require.NoError(t, db.Joins("Application").
		Where("application.name = ? AND application_usage.ended_at IS NULL", usage.IdleApplicationName).
		Limit(1).Order("application_usage.started_at DESC").
		First(&readIdleUsage).Error)

	require.NotEqual(t, int64(0), readIdleUsage.StartedAt)
	require.NotEqual(t, readIdleUsage.ID, closedIdleUsage.ID)
	require.Nil(t, readIdleUsage.EndedAt)
	require.Nil(t, readIdleUsage.DurationSeconds)
}

func TestService_WhenExitIdle_CloseCurrentIdlePeriod(t *testing.T) {
	service, db := setUpService(t)

	// create the Idle application
	idleApp := usage.Application{Name: usage.IdleApplicationName}
	require.NoError(t, db.Create(&idleApp).Error)

	// create an open idle usage
	idleUsage := usage.ApplicationUsage{
		ApplicationID:   idleApp.ID,
		StartedAt:       time.Now().Unix(),
		Classification:  usage.ClassificationNone,
		TerminationMode: usage.TerminationModeNone,
		WindowTitle:     usage.IdleApplicationName,
		ExecutablePath:  "idle",
	}
	require.NoError(t, db.Create(&idleUsage).Error)

	// wait 3 seconds
	time.Sleep(3 * time.Second)

	// exit idle
	err := service.IdleChanged(context.Background(), false)
	require.NoError(t, err, "failed to exit idle")

	// read the idle usage
	var readIdleUsage usage.ApplicationUsage
	require.NoError(t, db.Where("id = ?", idleUsage.ID).First(&readIdleUsage).Error)

	require.NotEqual(t, int64(0), readIdleUsage.StartedAt)
	require.NotNil(t, readIdleUsage.EndedAt)
	require.NotNil(t, readIdleUsage.DurationSeconds)
	require.Equal(t, 3, *readIdleUsage.DurationSeconds)
}

func TestService_CloseCurrentApplicationUsageWhenEnterIdle(t *testing.T) {
	service, db := setUpService(t)

	// create a new application usage
	applicationUsage := usage.ApplicationUsage{
		StartedAt:       time.Now().Unix(),
		EndedAt:         nil,
		DurationSeconds: nil,
	}
	if err := db.Create(&applicationUsage).Error; err != nil {
		t.Fatalf("failed to create application usage: %v", err)
	}

	// wait 3 seconds
	time.Sleep(3 * time.Second)

	err := service.IdleChanged(context.Background(), true)
	require.NoError(t, err, "failed to enter idle")

	// read the application usage
	var readApplicationUsage usage.ApplicationUsage
	if err := db.Where("id = ?", applicationUsage.ID).First(&readApplicationUsage).Error; err != nil {
		t.Fatalf("failed to find application usage: %v", err)
	}

	require.NotEqual(t, 0, readApplicationUsage.StartedAt)
	require.NotEqual(t, 0, readApplicationUsage.EndedAt)
	require.NotNil(t, readApplicationUsage.DurationSeconds)
	require.Equal(t, 3, *readApplicationUsage.DurationSeconds)
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

// roundTripFunc is an adapter to allow the use of ordinary functions as http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
