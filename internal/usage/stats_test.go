package usage

import (
	"context"
	"testing"
	"time"

	"github.com/focusd-so/focusd/internal/timeline"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDailySummary_HourlyBreakdown(t *testing.T) {
	t.Log("STARTING TEST")
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	timelineService, err := timeline.NewService(db)
	require.NoError(t, err)

	usageService, err := NewService(ctx, db, timelineService)
	require.NoError(t, err)

	// Use a fixed date: 2024-04-24
	date := time.Date(2024, 4, 24, 0, 0, 0, 0, time.UTC)
	
	// Create an application
	app := Application{Name: "TestApp"}
	db.Create(&app)

	// Helper to create events
	createUsageEvent := func(occurredAt time.Time, classification Classification) {
		payload := ApplicationUsagePayload{
			ApplicationID:  app.ID,
			Classification: classification,
		}
		_, err := timelineService.CreateEvent(
			EventTypeUsageChanged,
			timeline.WithOccurredAt(occurredAt),
			timeline.WithPayload(payload),
		)
		require.NoError(t, err)
	}

	// 10:50 - 11:30 Productive (40 mins) -> 10m in 10:00, 30m in 11:00
	// 11:30 - 12:15 Distracting (45 mins) -> 30m in 11:00, 15m in 12:00
	createUsageEvent(date.Add(10*time.Hour+50*time.Minute), ClassificationProductive)
	createUsageEvent(date.Add(11*time.Hour+30*time.Minute), ClassificationDistracting)
	
	// Finalize last event at 12:15
	lastTime := date.Add(12*time.Hour+15*time.Minute).Unix()
	events, _ := timelineService.ListEvents(timeline.OrderByOccurredAtDesc(), timeline.Limit(1))
	lastEvent := events[0]
	lastEvent.FinishedAt = &lastTime
	db.Save(lastEvent)

	summary := usageService.DailySummary(date)

	// Verify Daily Summary
	require.Equal(t, 40*time.Minute, summary.TotalProductivityDuration)
	require.Equal(t, 45*time.Minute, summary.TotalDistractionDuration)
	require.Equal(t, 85*time.Minute, summary.TotalUsageDuration)

	// Verify Hourly Breakdown
	// 10:00 slot
	require.NotNil(t, summary.HourlyBreakdown["10:00"])
	require.Equal(t, 10*time.Minute, summary.HourlyBreakdown["10:00"].TotalProductivityDuration)
	require.Equal(t, time.Duration(0), summary.HourlyBreakdown["10:00"].TotalDistractionDuration)

	// 11:00 slot
	require.NotNil(t, summary.HourlyBreakdown["11:00"])
	require.Equal(t, 30*time.Minute, summary.HourlyBreakdown["11:00"].TotalProductivityDuration)
	require.Equal(t, 30*time.Minute, summary.HourlyBreakdown["11:00"].TotalDistractionDuration)

	// 12:00 slot
	require.NotNil(t, summary.HourlyBreakdown["12:00"])
	require.Equal(t, time.Duration(0), summary.HourlyBreakdown["12:00"].TotalProductivityDuration)
	require.Equal(t, 15*time.Minute, summary.HourlyBreakdown["12:00"].TotalDistractionDuration)
}
