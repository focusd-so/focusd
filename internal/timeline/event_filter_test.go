package timeline

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEventFilters(t *testing.T) {
	harness := newTimelineHarness(t)
	now := time.Now().UTC()

	// Seed data
	eventsData := []struct {
		eventType string
		opts      []EventOption
	}{
		{
			eventType: "app_usage",
			opts: []EventOption{
				WithStartedAt(now.Add(-10 * time.Hour)),
				WithEndedAt(now.Add(-9 * time.Hour)),
				WithTags(NewTag("vscode", "default"), NewTag("productivity", "default")),
			},
		},
		{
			eventType: "app_usage",
			opts: []EventOption{
				WithStartedAt(now.Add(-8 * time.Hour)),
				WithEndedAt(now.Add(-7 * time.Hour)),
				WithTags(NewTag("slack", "default"), NewTag("communication", "default")),
			},
		},
		{
			eventType: "website_visit",
			opts: []EventOption{
				WithStartedAt(now.Add(-6 * time.Hour)),
				WithEndedAt(now.Add(-5 * time.Hour)),
				WithTags(NewTag("github", "default"), NewTag("productivity", "default")),
			},
		},
		{
			eventType: "focus_session",
			opts: []EventOption{
				WithStartedAt(now.Add(-2 * time.Hour)), // Active event
				WithTags(NewTag("deep_work", "default")),
			},
		},
		{
			eventType: "focus_session",
			opts: []EventOption{
				WithStartedAt(now.Add(-1 * time.Hour)), // Active event
				WithTags(NewTag("learning", "default")),
			},
		},
	}

	for _, data := range eventsData {
		_, err := harness.service.CreateEvent(data.eventType, data.opts...)
		require.NoError(t, err)
	}

	t.Run("ByTypes", func(t *testing.T) {
		harness.AssertEventsCount(2, ByTypes("app_usage"))
		harness.AssertEventsCount(1, ByTypes("website_visit"))
		harness.AssertEventsCount(3, ByTypes("app_usage", "website_visit"))
		harness.AssertEventsCount(0, ByTypes("unknown"))
	})

	t.Run("ByTags", func(t *testing.T) {
		harness.AssertEventsCount(2, ByTags("productivity"))
		harness.AssertEventsCount(1, ByTags("vscode"))
		harness.AssertEventsCount(2, ByTags("slack", "deep_work")) // Actually gives us events with *either* tag because IN is used
		harness.AssertEventsCount(0, ByTags("non_existent"))
	})

	t.Run("ByStartTime", func(t *testing.T) {
		harness.AssertEventsCount(2, ByStartTime(now.Add(-11*time.Hour), now.Add(-7*time.Hour)))
		harness.AssertEventsCount(3, ByStartTime(time.Time{}, now.Add(-4*time.Hour)))
		harness.AssertEventsCount(2, ByStartTime(now.Add(-3*time.Hour), time.Time{}))
	})

	t.Run("ByEndTime", func(t *testing.T) {
		// Only 3 events have EndedAt
		harness.AssertEventsCount(2, ByEndTime(now.Add(-10*time.Hour), now.Add(-6*time.Hour)))
		harness.AssertEventsCount(3, ByEndTime(time.Time{}, now.Add(-4*time.Hour)))
		harness.AssertEventsCount(1, ByEndTime(now.Add(-6*time.Hour), time.Time{}))
	})

	t.Run("ActiveOnly", func(t *testing.T) {
		harness.AssertEventsCount(2, ActiveOnly())
		harness.AssertEventsCount(1, ActiveOnly(), ByTags("deep_work"))
	})

	t.Run("Limit and Offset", func(t *testing.T) {
		// Default order is DESC (newest first)
		// 1. focus_session (-1h)
		// 2. focus_session (-2h)
		// 3. website_visit (-6h)
		// 4. app_usage (-8h)
		// 5. app_usage (-10h)

		eventsLimit2 := harness.ListEvents(Limit(2))
		require.Len(t, eventsLimit2, 2)
		require.Equal(t, "focus_session", eventsLimit2[0].Type)
		require.Equal(t, []string{"learning"}, eventsLimit2[0].TagsSlice())
		require.Equal(t, "focus_session", eventsLimit2[1].Type)
		require.Equal(t, []string{"deep_work"}, eventsLimit2[1].TagsSlice())

		eventsOffset2Limit1 := harness.ListEvents(Offset(2), Limit(1))
		require.Len(t, eventsOffset2Limit1, 1)
		require.Equal(t, "website_visit", eventsOffset2Limit1[0].Type)
	})

	t.Run("OrderBy", func(t *testing.T) {
		ascEvents := harness.ListEvents(OrderByStartedAtAsc(), Limit(1))
		require.Len(t, ascEvents, 1)
		require.Equal(t, "app_usage", ascEvents[0].Type)
		require.Equal(t, []string{"vscode", "productivity"}, ascEvents[0].TagsSlice())

		descEvents := harness.ListEvents(OrderByStartedAtDesc(), Limit(1))
		require.Len(t, descEvents, 1)
		require.Equal(t, "focus_session", descEvents[0].Type)
		require.Equal(t, []string{"learning"}, descEvents[0].TagsSlice())
	})

	t.Run("Combined Filters", func(t *testing.T) {
		harness.AssertEventsCount(1,
			ByTypes("app_usage"),
			ByTags("communication"),
			ByStartTime(now.Add(-10*time.Hour), now.Add(-5*time.Hour)),
		)
	})
}
