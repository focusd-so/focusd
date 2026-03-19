package usage_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/focusd-so/focusd/internal/usage"
)

// timePtr returns a pointer to the given time.
func timePtr(t time.Time) *time.Time { return &t }

func terminationModePtr(mode usage.TerminationMode) *usage.TerminationMode { return &mode }

func TestGetUsageList_NoFilters(t *testing.T) {
	service, db := setUpService(t)

	// Seed four usages across the day.
	starts := []time.Time{
		time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 18, 0, 0, 0, time.UTC),
	}
	durations := []int{1800, 2700, 1200, 3000}
	names := []string{"Chrome", "Slack", "Finder", "Safari"}
	for i := range starts {
		endAt := starts[i].Unix() + int64(durations[i])
		u := usage.ApplicationUsage{
			StartedAt:       starts[i].Unix(),
			EndedAt:         &endAt,
			DurationSeconds: &durations[i],
			Classification:  usage.ClassificationProductive,
			Application:     usage.Application{Name: names[i]},
		}
		require.NoError(t, db.Create(&u).Error)
	}

	result, err := service.GetUsageList(usage.GetUsageListOptions{})
	require.NoError(t, err)
	require.Len(t, result, 4, "all four usages should be returned when no filters are set")
}

func TestGetUsageList_StartedAtFilter(t *testing.T) {
	service, db := setUpService(t)

	// Seed usages at 08:00, 10:00, 14:00, 18:00
	starts := []time.Time{
		time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 18, 0, 0, 0, time.UTC),
	}
	durations := []int{1800, 2700, 1200, 3000}
	for i := range starts {
		endAt := starts[i].Unix() + int64(durations[i])
		u := usage.ApplicationUsage{
			StartedAt:       starts[i].Unix(),
			EndedAt:         &endAt,
			DurationSeconds: &durations[i],
			Classification:  usage.ClassificationProductive,
			Application:     usage.Application{Name: "App"},
		}
		require.NoError(t, db.Create(&u).Error)
	}

	// Filter: startedAt >= 12:00 → should return usages at 14:00 and 18:00
	startedAt := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	result, err := service.GetUsageList(usage.GetUsageListOptions{
		StartedAt: timePtr(startedAt),
	})
	require.NoError(t, err)
	require.Len(t, result, 2, "only usages starting at or after 12:00 should be returned")

	// Results are ordered by started_at DESC, so 18:00 first, then 14:00
	require.Equal(t, starts[3].Unix(), result[0].StartedAt)
	require.Equal(t, starts[2].Unix(), result[1].StartedAt)
}

func TestGetUsageList_EndedAtFilter(t *testing.T) {
	service, db := setUpService(t)

	// Seed usages at 08:00, 10:00, 14:00, 18:00
	starts := []time.Time{
		time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 18, 0, 0, 0, time.UTC),
	}
	durations := []int{1800, 2700, 1200, 3000}
	for i := range starts {
		endAt := starts[i].Unix() + int64(durations[i])
		u := usage.ApplicationUsage{
			StartedAt:       starts[i].Unix(),
			EndedAt:         &endAt,
			DurationSeconds: &durations[i],
			Classification:  usage.ClassificationProductive,
			Application:     usage.Application{Name: "App"},
		}
		require.NoError(t, db.Create(&u).Error)
	}

	// u0 ends at 08:30, u1 ends at 10:45, u2 ends at 14:20, u3 ends at 18:50
	// Filter: endedAt <= 11:00 → should return u0 (ends 08:30) and u1 (ends 10:45)
	endedAt := time.Date(2025, 6, 15, 11, 0, 0, 0, time.UTC)
	result, err := service.GetUsageList(usage.GetUsageListOptions{
		EndedAt: timePtr(endedAt),
	})
	require.NoError(t, err)
	require.Len(t, result, 2, "only usages ending at or before 11:00 should be returned")

	// Results ordered by started_at DESC: u1 (10:00) first, then u0 (08:00)
	require.Equal(t, starts[1].Unix(), result[0].StartedAt)
	require.Equal(t, starts[0].Unix(), result[1].StartedAt)
}

func TestGetUsageList_StartedAtAndEndedAtCombined(t *testing.T) {
	service, db := setUpService(t)

	// Seed usages at 08:00, 10:00, 14:00, 18:00
	starts := []time.Time{
		time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 18, 0, 0, 0, time.UTC),
	}
	durations := []int{1800, 2700, 1200, 3000}
	for i := range starts {
		endAt := starts[i].Unix() + int64(durations[i])
		u := usage.ApplicationUsage{
			StartedAt:       starts[i].Unix(),
			EndedAt:         &endAt,
			DurationSeconds: &durations[i],
			Classification:  usage.ClassificationProductive,
			Application:     usage.Application{Name: "App"},
		}
		require.NoError(t, db.Create(&u).Error)
	}

	// u0: 08:00–08:30, u1: 10:00–10:45, u2: 14:00–14:20, u3: 18:00–18:50
	// Filter: startedAt >= 09:00 AND endedAt <= 15:00
	// → u1 (starts 10:00, ends 10:45) and u2 (starts 14:00, ends 14:20)
	startedAt := time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC)
	endedAt := time.Date(2025, 6, 15, 15, 0, 0, 0, time.UTC)

	result, err := service.GetUsageList(usage.GetUsageListOptions{
		StartedAt: timePtr(startedAt),
		EndedAt:   timePtr(endedAt),
	})
	require.NoError(t, err)
	require.Len(t, result, 2, "only usages within the [09:00, 15:00] window should be returned")

	// Ordered by started_at DESC: u2 (14:00) first, then u1 (10:00)
	require.Equal(t, starts[2].Unix(), result[0].StartedAt)
	require.Equal(t, starts[1].Unix(), result[1].StartedAt)
}

func TestGetUsageList_StartedAtExactBoundary(t *testing.T) {
	service, db := setUpService(t)

	// Single usage starting at exactly 10:00
	startAt := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	endAt := startAt.Unix() + 1800
	dur := 1800
	u := usage.ApplicationUsage{
		StartedAt:       startAt.Unix(),
		EndedAt:         &endAt,
		DurationSeconds: &dur,
		Classification:  usage.ClassificationProductive,
		Application:     usage.Application{Name: "App"},
	}
	require.NoError(t, db.Create(&u).Error)

	// startedAt filter equals the exact start time → should include the usage (>=)
	result, err := service.GetUsageList(usage.GetUsageListOptions{
		StartedAt: timePtr(startAt),
	})
	require.NoError(t, err)
	require.Len(t, result, 1, "usage starting at exact boundary should be included (>=)")
	require.Equal(t, startAt.Unix(), result[0].StartedAt)
}

func TestGetUsageList_EndedAtExactBoundary(t *testing.T) {
	service, db := setUpService(t)

	// Single usage ending at exactly 10:30
	startAt := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	endAt := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	endAtUnix := endAt.Unix()
	dur := 1800
	u := usage.ApplicationUsage{
		StartedAt:       startAt.Unix(),
		EndedAt:         &endAtUnix,
		DurationSeconds: &dur,
		Classification:  usage.ClassificationProductive,
		Application:     usage.Application{Name: "App"},
	}
	require.NoError(t, db.Create(&u).Error)

	// endedAt filter equals the exact end time → should include the usage (<=)
	result, err := service.GetUsageList(usage.GetUsageListOptions{
		EndedAt: timePtr(endAt),
	})
	require.NoError(t, err)
	require.Len(t, result, 1, "usage ending at exact boundary should be included (<=)")
}

func TestGetUsageList_OrderDescByStartedAt(t *testing.T) {
	service, db := setUpService(t)

	// Insert usages in ascending order; the function should return them in descending order.
	starts := []time.Time{
		time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC),
	}
	dur := 600
	for _, s := range starts {
		endAt := s.Unix() + int64(dur)
		u := usage.ApplicationUsage{
			StartedAt:       s.Unix(),
			EndedAt:         &endAt,
			DurationSeconds: &dur,
			Classification:  usage.ClassificationProductive,
			Application:     usage.Application{Name: "App"},
		}
		require.NoError(t, db.Create(&u).Error)
	}

	result, err := service.GetUsageList(usage.GetUsageListOptions{})
	require.NoError(t, err)
	require.Len(t, result, 3)

	// Verify descending order
	require.Equal(t, starts[2].Unix(), result[0].StartedAt)
	require.Equal(t, starts[1].Unix(), result[1].StartedAt)
	require.Equal(t, starts[0].Unix(), result[2].StartedAt)
}

func TestGetUsageList_PreloadsApplication(t *testing.T) {
	service, db := setUpService(t)

	startAt := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	endAt := startAt.Unix() + 1800
	dur := 1800
	u := usage.ApplicationUsage{
		StartedAt:       startAt.Unix(),
		EndedAt:         &endAt,
		DurationSeconds: &dur,
		Classification:  usage.ClassificationProductive,
		Application:     usage.Application{Name: "Visual Studio Code"},
	}
	require.NoError(t, db.Create(&u).Error)

	result, err := service.GetUsageList(usage.GetUsageListOptions{})
	require.NoError(t, err)
	require.Len(t, result, 1)

	// Application relation should be eagerly loaded
	require.Equal(t, "Visual Studio Code", result[0].Application.Name)
}

func TestGetUsageList_PreloadsTags(t *testing.T) {
	service, db := setUpService(t)

	startAt := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	endAt := startAt.Unix() + 1800
	dur := 1800
	u := usage.ApplicationUsage{
		StartedAt:       startAt.Unix(),
		EndedAt:         &endAt,
		DurationSeconds: &dur,
		Classification:  usage.ClassificationProductive,
		Application:     usage.Application{Name: "Slack"},
	}
	require.NoError(t, db.Create(&u).Error)

	// Attach tags to the usage
	tags := []usage.ApplicationUsageTags{
		{UsageID: u.ID, Tag: "work"},
		{UsageID: u.ID, Tag: "communication"},
	}
	for _, tag := range tags {
		require.NoError(t, db.Create(&tag).Error)
	}

	result, err := service.GetUsageList(usage.GetUsageListOptions{})
	require.NoError(t, err)
	require.Len(t, result, 1)

	// Tags relation should be eagerly loaded
	require.Len(t, result[0].Tags, 2)
	tagNames := []string{result[0].Tags[0].Tag, result[0].Tags[1].Tag}
	require.Contains(t, tagNames, "work")
	require.Contains(t, tagNames, "communication")
}

func TestGetUsageList_DateFilter(t *testing.T) {
	service, db := setUpService(t)

	// Seed usages on two different days
	day1 := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC)
	dur := 1800

	for _, s := range []time.Time{day1, day2} {
		endAt := s.Unix() + int64(dur)
		u := usage.ApplicationUsage{
			StartedAt:       s.Unix(),
			EndedAt:         &endAt,
			DurationSeconds: &dur,
			Classification:  usage.ClassificationProductive,
			Application:     usage.Application{Name: "App"},
		}
		require.NoError(t, db.Create(&u).Error)
	}

	// Filter by day1 only
	filterDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	result, err := service.GetUsageList(usage.GetUsageListOptions{
		Date: timePtr(filterDate),
	})
	require.NoError(t, err)
	require.Len(t, result, 1, "only usages on 2025-06-15 should be returned")
	require.Equal(t, day1.Unix(), result[0].StartedAt)
}

func TestGetUsageList_DateFilterNoMatches(t *testing.T) {
	service, db := setUpService(t)

	// Usage on June 15
	startAt := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	endAt := startAt.Unix() + 1800
	dur := 1800
	u := usage.ApplicationUsage{
		StartedAt:       startAt.Unix(),
		EndedAt:         &endAt,
		DurationSeconds: &dur,
		Classification:  usage.ClassificationProductive,
		Application:     usage.Application{Name: "App"},
	}
	require.NoError(t, db.Create(&u).Error)

	// Filter by a date with no usages
	noMatchDate := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	result, err := service.GetUsageList(usage.GetUsageListOptions{
		Date: timePtr(noMatchDate),
	})
	require.NoError(t, err)
	require.Len(t, result, 0, "no usages should match a date with no data")
}

func TestGetUsageList_PaginationPageAndPageSize(t *testing.T) {
	service, db := setUpService(t)

	// Create 5 usages, each 30 minutes apart
	baseStart := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	dur := 600
	for i := 0; i < 5; i++ {
		s := baseStart.Add(time.Duration(i) * 30 * time.Minute)
		endAt := s.Unix() + int64(dur)
		u := usage.ApplicationUsage{
			StartedAt:       s.Unix(),
			EndedAt:         &endAt,
			DurationSeconds: &dur,
			Classification:  usage.ClassificationProductive,
			Application:     usage.Application{Name: "App"},
		}
		require.NoError(t, db.Create(&u).Error)
	}

	page := 0
	pageSize := 2

	// Page 0, size 2 → first 2 results (DESC order: index 4, 3)
	result, err := service.GetUsageList(usage.GetUsageListOptions{
		Page:     &page,
		PageSize: &pageSize,
	})
	require.NoError(t, err)
	require.Len(t, result, 2, "Page 0 should return 2 items")

	// Page 1, size 2 → offset=2, skips first 2 (DESC order: returns index 2, 1)
	page = 1
	result, err = service.GetUsageList(usage.GetUsageListOptions{
		Page:     &page,
		PageSize: &pageSize,
	})
	require.NoError(t, err)
	require.Len(t, result, 2, "page 1 with pageSize 2 should skip 2 and return next 2")

	// Page 2, size 2 → offset=4, skips first 4 (returns index 0 only)
	page = 2
	result, err = service.GetUsageList(usage.GetUsageListOptions{
		Page:     &page,
		PageSize: &pageSize,
	})
	require.NoError(t, err)
	require.Len(t, result, 1, "page 2 with pageSize 2 should skip 4 and return last 1")

	// Page 3, size 2 → offset=6, beyond all data
	page = 3
	result, err = service.GetUsageList(usage.GetUsageListOptions{
		Page:     &page,
		PageSize: &pageSize,
	})
	require.NoError(t, err)
	require.Len(t, result, 0, "page beyond data should return empty")
}

func TestGetUsageList_PaginationRequiresBothPageAndPageSize(t *testing.T) {
	service, db := setUpService(t)

	// Create 3 usages
	baseStart := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	dur := 600
	for i := 0; i < 3; i++ {
		s := baseStart.Add(time.Duration(i) * time.Hour)
		endAt := s.Unix() + int64(dur)
		u := usage.ApplicationUsage{
			StartedAt:       s.Unix(),
			EndedAt:         &endAt,
			DurationSeconds: &dur,
			Classification:  usage.ClassificationProductive,
			Application:     usage.Application{Name: "App"},
		}
		require.NoError(t, db.Create(&u).Error)
	}

	page := 1
	pageSize := 1

	// Only Page set, no PageSize → pagination should NOT apply
	result, err := service.GetUsageList(usage.GetUsageListOptions{
		Page: &page,
	})
	require.NoError(t, err)
	require.Len(t, result, 3, "pagination should not apply when only Page is set")

	// Only PageSize set, no Page → pagination should NOT apply
	result, err = service.GetUsageList(usage.GetUsageListOptions{
		PageSize: &pageSize,
	})
	require.NoError(t, err)
	require.Len(t, result, 3, "pagination should not apply when only PageSize is set")
}

func TestGetUsageList_DateCombinedWithStartedAtAndEndedAt(t *testing.T) {
	service, db := setUpService(t)

	// Two usages on the same day, at different times
	// u0: 08:00–08:30, u1: 14:00–14:30
	day := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	starts := []time.Time{
		time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
	}
	// Also one on a different day
	otherDay := time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC)
	allStarts := append(starts, otherDay)
	dur := 1800

	for _, s := range allStarts {
		endAt := s.Unix() + int64(dur)
		u := usage.ApplicationUsage{
			StartedAt:       s.Unix(),
			EndedAt:         &endAt,
			DurationSeconds: &dur,
			Classification:  usage.ClassificationProductive,
			Application:     usage.Application{Name: "App"},
		}
		require.NoError(t, db.Create(&u).Error)
	}

	// Filter: date=June 15 AND startedAt >= 10:00
	// → only u1 (14:00) should match; u0 is before 10:00 and other-day is excluded by Date
	startedAt := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	result, err := service.GetUsageList(usage.GetUsageListOptions{
		Date:      timePtr(day),
		StartedAt: timePtr(startedAt),
	})
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, starts[1].Unix(), result[0].StartedAt)
}

func TestGetUsageList_EmptyDatabase(t *testing.T) {
	service, _ := setUpService(t)

	result, err := service.GetUsageList(usage.GetUsageListOptions{})
	require.NoError(t, err)
	require.NotNil(t, result, "should return empty slice, not nil")
	require.Len(t, result, 0)
}

func TestGetUsageList_NoMatchesWithNarrowWindow(t *testing.T) {
	service, db := setUpService(t)

	// Usages at 08:00 and 18:00
	starts := []time.Time{
		time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 18, 0, 0, 0, time.UTC),
	}
	durations := []int{1800, 3000}
	for i := range starts {
		endAt := starts[i].Unix() + int64(durations[i])
		u := usage.ApplicationUsage{
			StartedAt:       starts[i].Unix(),
			EndedAt:         &endAt,
			DurationSeconds: &durations[i],
			Classification:  usage.ClassificationProductive,
			Application:     usage.Application{Name: "App"},
		}
		require.NoError(t, db.Create(&u).Error)
	}

	// Window 12:00–13:00: no usages start after 12:00 with ended_at before 13:00
	startedAt := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	endedAt := time.Date(2025, 6, 15, 13, 0, 0, 0, time.UTC)

	result, err := service.GetUsageList(usage.GetUsageListOptions{
		StartedAt: timePtr(startedAt),
		EndedAt:   timePtr(endedAt),
	})
	require.NoError(t, err)
	require.Len(t, result, 0, "no usages should match a narrow window with no data")
}

func TestGetUsageList_TerminationModeFilter(t *testing.T) {
	service, db := setUpService(t)

	starts := []time.Time{
		time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
	}
	modes := []usage.TerminationMode{
		usage.TerminationModeNone,
		usage.TerminationModeBlock,
		usage.TerminationModeAllow,
	}
	dur := 1800
	for i := range starts {
		endAt := starts[i].Unix() + int64(dur)
		u := usage.ApplicationUsage{
			StartedAt:       starts[i].Unix(),
			EndedAt:         &endAt,
			DurationSeconds: &dur,
			Classification:  usage.ClassificationProductive,
			TerminationMode: modes[i],
			Application:     usage.Application{Name: "App"},
		}
		require.NoError(t, db.Create(&u).Error)
	}

	result, err := service.GetUsageList(usage.GetUsageListOptions{
		TerminationMode: terminationModePtr(usage.TerminationModeBlock),
	})
	require.NoError(t, err)
	require.Len(t, result, 1, "only blocked items should be returned")
	require.Equal(t, usage.TerminationModeBlock, result[0].TerminationMode)
	require.Equal(t, starts[1].Unix(), result[0].StartedAt)
}

func TestGetUsageList_TerminationModeFilterNoMatches(t *testing.T) {
	service, db := setUpService(t)

	startAt := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	endAt := startAt.Unix() + 1800
	dur := 1800
	u := usage.ApplicationUsage{
		StartedAt:       startAt.Unix(),
		EndedAt:         &endAt,
		DurationSeconds: &dur,
		Classification:  usage.ClassificationProductive,
		TerminationMode: usage.TerminationModeNone,
		Application:     usage.Application{Name: "App"},
	}
	require.NoError(t, db.Create(&u).Error)

	result, err := service.GetUsageList(usage.GetUsageListOptions{
		TerminationMode: terminationModePtr(usage.TerminationModeBlock),
	})
	require.NoError(t, err)
	require.Len(t, result, 0, "no rows should match an unused termination mode")
}

func TestGetUsageList_TerminationModeFilterCombinedWithDateRange(t *testing.T) {
	service, db := setUpService(t)

	type seedUsage struct {
		start time.Time
		mode  usage.TerminationMode
	}
	seed := []seedUsage{
		{start: time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC), mode: usage.TerminationModeBlock},
		{start: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC), mode: usage.TerminationModeNone},
		{start: time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC), mode: usage.TerminationModeBlock},
		{start: time.Date(2025, 6, 15, 18, 0, 0, 0, time.UTC), mode: usage.TerminationModeBlock},
	}
	dur := 1800
	for _, row := range seed {
		endAt := row.start.Unix() + int64(dur)
		u := usage.ApplicationUsage{
			StartedAt:       row.start.Unix(),
			EndedAt:         &endAt,
			DurationSeconds: &dur,
			Classification:  usage.ClassificationProductive,
			TerminationMode: row.mode,
			Application:     usage.Application{Name: "App"},
		}
		require.NoError(t, db.Create(&u).Error)
	}

	startedAt := time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC)
	endedAt := time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC)
	result, err := service.GetUsageList(usage.GetUsageListOptions{
		StartedAt:       timePtr(startedAt),
		EndedAt:         timePtr(endedAt),
		TerminationMode: terminationModePtr(usage.TerminationModeBlock),
	})
	require.NoError(t, err)
	require.Len(t, result, 1, "combined filters should return only the matching intersection")
	require.Equal(t, usage.TerminationModeBlock, result[0].TerminationMode)
	require.Equal(t, time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC).Unix(), result[0].StartedAt)
}

func TestGetUsageList_TerminationModeNilIgnored(t *testing.T) {
	service, db := setUpService(t)

	starts := []time.Time{
		time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 15, 11, 0, 0, 0, time.UTC),
	}
	modes := []usage.TerminationMode{
		usage.TerminationModeBlock,
		usage.TerminationModeNone,
	}
	dur := 1200
	for i := range starts {
		endAt := starts[i].Unix() + int64(dur)
		u := usage.ApplicationUsage{
			StartedAt:       starts[i].Unix(),
			EndedAt:         &endAt,
			DurationSeconds: &dur,
			Classification:  usage.ClassificationProductive,
			TerminationMode: modes[i],
			Application:     usage.Application{Name: "App"},
		}
		require.NoError(t, db.Create(&u).Error)
	}

	result, err := service.GetUsageList(usage.GetUsageListOptions{
		TerminationMode: nil,
	})
	require.NoError(t, err)
	require.Len(t, result, 2, "nil termination mode should not filter results")
}
