package usage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/focusd-so/focusd/internal/timeline"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDailySummary(t *testing.T) {
	ctx := context.Background()
	dbFile := "test_daily_summary.db"
	_ = os.Remove(dbFile)
	defer os.Remove(dbFile)

	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	require.NoError(t, err)

	timelineService, err := timeline.NewService(db)
	require.NoError(t, err)

	usageService, err := NewService(ctx, db, timelineService)
	require.NoError(t, err)

	t.Run("HourlyBreakdown", func(t *testing.T) {
		date := time.Date(2024, 4, 24, 0, 0, 0, 0, time.UTC)
		app := Application{Name: "TestApp"}
		db.Create(&app)

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

		createUsageEvent(date.Add(10*time.Hour+50*time.Minute), ClassificationProductive)
		createUsageEvent(date.Add(11*time.Hour+30*time.Minute), ClassificationDistracting)
		
		lastTime := date.Add(12*time.Hour+15*time.Minute).Unix()
		events, _ := timelineService.ListEvents(timeline.OrderByOccurredAtDesc(), timeline.Limit(1))
		require.NotEmpty(t, events)
		events[0].FinishedAt = &lastTime
		db.Save(events[0])

		summary := usageService.DailySummary(date)

		require.Equal(t, 40*time.Minute, summary.TotalProductivityDuration)
		require.Equal(t, 45*time.Minute, summary.TotalDistractionDuration)
		require.Equal(t, 85*time.Minute, summary.TotalUsageDuration)

		require.NotNil(t, summary.HourlyBreakdown["10:00"])
		require.Equal(t, 10*time.Minute, summary.HourlyBreakdown["10:00"].TotalProductivityDuration)
		require.NotNil(t, summary.HourlyBreakdown["11:00"])
		require.Equal(t, 30*time.Minute, summary.HourlyBreakdown["11:00"].TotalProductivityDuration)
		require.Equal(t, 30*time.Minute, summary.HourlyBreakdown["11:00"].TotalDistractionDuration)
		require.NotNil(t, summary.HourlyBreakdown["12:00"])
		require.Equal(t, 15*time.Minute, summary.HourlyBreakdown["12:00"].TotalDistractionDuration)
	})

	t.Run("Complex", func(t *testing.T) {
		date := time.Date(2024, 4, 25, 0, 0, 0, 0, time.UTC)
		app := Application{Name: "IDE"}
		db.Create(&app)

		_, _ = timelineService.CreateEvent(EventTypeUsageChanged,
			timeline.WithOccurredAt(date.Add(9*time.Hour)),
			timeline.WithPayload(ApplicationUsagePayload{
				ApplicationID: app.ID,
				Classification: ClassificationProductive,
				ClassificationResult: &ClassificationResult{
					LLMClassificationResult: &LLMClassificationResult{
						DetectedProject: "focusd",
					},
				},
			}),
		)

		_, _ = timelineService.CreateEvent(EventTypeUserIdleChanged,
			timeline.WithOccurredAt(date.Add(9*time.Hour+30*time.Minute)),
		)

		_, _ = timelineService.CreateEvent(EventTypeUsageChanged,
			timeline.WithOccurredAt(date.Add(10*time.Hour)),
			timeline.WithPayload(ApplicationUsagePayload{
				ApplicationID: app.ID,
				Classification: ClassificationProductive,
				ClassificationResult: &ClassificationResult{
					LLMClassificationResult: &LLMClassificationResult{
						DetectedProject: "focusd",
					},
				},
			}),
		)

		_, _ = timelineService.CreateEvent(EventTypeUsageChanged,
			timeline.WithOccurredAt(date.Add(10*time.Hour+15*time.Minute)),
			timeline.WithPayload(ApplicationUsagePayload{
				ApplicationID: app.ID,
				Classification: ClassificationDistracting,
			}),
		)

		lastTime := date.Add(10*time.Hour+20*time.Minute).Unix()
		events, _ := timelineService.ListEvents(timeline.OrderByOccurredAtDesc(), timeline.Limit(1))
		require.NotEmpty(t, events)
		events[0].FinishedAt = &lastTime
		db.Save(events[0])

		summary := usageService.DailySummary(date)

		require.Equal(t, 45*time.Minute, summary.TotalProductivityDuration)
		require.Equal(t, 5*time.Minute, summary.TotalDistractionDuration)
		require.Equal(t, 30*time.Minute, summary.TotalIdleDuration)
		require.Equal(t, 80*time.Minute, summary.TotalUsageDuration)

		require.Len(t, summary.ProjectBreakdown, 1)
		require.Equal(t, "focusd", summary.ProjectBreakdown[0].ProjectName)
		require.Equal(t, int64(45*60), summary.ProjectBreakdown[0].DurationSeconds)
		require.Equal(t, 5*60, summary.TopDistractions["IDE"])
		require.InDelta(t, 90.0, summary.ProductivityScore, 0.01)
	})

	t.Run("Empty", func(t *testing.T) {
		date := time.Date(2024, 4, 26, 0, 0, 0, 0, time.UTC)
		summary := usageService.DailySummary(date)
		require.Equal(t, time.Duration(0), summary.TotalUsageDuration)
		require.Len(t, summary.HourlyBreakdown, 0)
	})

	t.Run("Ongoing", func(t *testing.T) {
		date := time.Date(2024, 4, 27, 0, 0, 0, 0, time.UTC)
		
		// Event at 10:00, no end time.
		// Since 2024-04-27 is in the past, DailySummary will cap it at dayEnd (00:00 tomorrow).
		_, _ = timelineService.CreateEvent(EventTypeUsageChanged,
			timeline.WithOccurredAt(date.Add(10*time.Hour)),
			timeline.WithPayload(ApplicationUsagePayload{
				Classification: ClassificationProductive,
			}),
		)

		summary := usageService.DailySummary(date)
		// 10:00 to 00:00 (next day) = 14 hours
		require.Equal(t, 14*time.Hour, summary.TotalProductivityDuration)
	})
}
