package usage_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/focusd-so/focusd/internal/usage"
)

// helper to create an int pointer
func intPtr(v int) *int { return &v }

// helper to create an int64 pointer
func int64Ptr(v int64) *int64 { return &v }

// baseDate returns 2025-06-15 00:00:00 UTC – a fixed date used as the reference day for all tests.
func baseDate() time.Time {
	return time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
}

// hourOf returns the beginning of a given hour on baseDate (UTC).
func hourOf(hour int) time.Time {
	return time.Date(2025, 6, 15, hour, 0, 0, 0, time.UTC)
}

func TestGetDayInsights_UsageSpanningTwoHours(t *testing.T) {
	service, db := setUpService(t)

	// Chrome used 10:54 – 11:15 (21 minutes = 1260 seconds)
	// Expected: 360s in hour 10, 900s in hour 11
	startAt := time.Date(2025, 6, 15, 10, 54, 0, 0, time.UTC)
	duration := 21 * 60 // 1260 seconds

	appUsage := usage.ApplicationUsage{
		StartedAt:       startAt.Unix(),
		EndedAt:         int64Ptr(startAt.Unix() + int64(duration)),
		DurationSeconds: intPtr(duration),
		Classification:  usage.ClassificationProductive,
		Application: usage.Application{
			Name:           "Google Chrome",
			ExecutablePath: "/Applications/Google Chrome.app",
		},
	}
	require.NoError(t, db.Create(&appUsage).Error)

	insights, err := service.GetDayInsights(baseDate())
	require.NoError(t, err)

	// Total productivity
	require.Equal(t, 1260, insights.ProductivityScore.ProductiveSeconds)
	require.Equal(t, 0, insights.ProductivityScore.DistractiveSeconds)
	require.Equal(t, 100, insights.ProductivityScore.ProductivityScore)

	// Hour 10 bucket: 6 minutes = 360 seconds
	bucket10, ok := insights.ProductivityPerHourBreakdown[hourOf(10)]
	require.True(t, ok, "expected a bucket for hour 10")
	require.Equal(t, 360, bucket10.ProductiveSeconds)
	require.Equal(t, 0, bucket10.DistractiveSeconds)

	// Hour 11 bucket: 15 minutes = 900 seconds
	bucket11, ok := insights.ProductivityPerHourBreakdown[hourOf(11)]
	require.True(t, ok, "expected a bucket for hour 11")
	require.Equal(t, 900, bucket11.ProductiveSeconds)
	require.Equal(t, 0, bucket11.DistractiveSeconds)

	// No other hour buckets should exist
	require.Len(t, insights.ProductivityPerHourBreakdown, 2)
}

func TestGetDayInsights_UsageSpanningThreeHours(t *testing.T) {
	service, db := setUpService(t)

	// Usage from 09:30 to 11:45 (2h15m = 8100 seconds)
	// Expected: 1800s in hour 9, 3600s in hour 10, 2700s in hour 11
	startAt := time.Date(2025, 6, 15, 9, 30, 0, 0, time.UTC)
	duration := 2*3600 + 15*60 // 8100 seconds

	appUsage := usage.ApplicationUsage{
		StartedAt:       startAt.Unix(),
		EndedAt:         int64Ptr(startAt.Unix() + int64(duration)),
		DurationSeconds: intPtr(duration),
		Classification:  usage.ClassificationDistracting,
		Application: usage.Application{
			Name:           "YouTube",
			ExecutablePath: "/Applications/Safari.app",
		},
	}
	require.NoError(t, db.Create(&appUsage).Error)

	insights, err := service.GetDayInsights(baseDate())
	require.NoError(t, err)

	require.Equal(t, 8100, insights.ProductivityScore.DistractiveSeconds)

	// Hour 9: 30 min = 1800s
	bucket9 := insights.ProductivityPerHourBreakdown[hourOf(9)]
	require.Equal(t, 1800, bucket9.DistractiveSeconds)

	// Hour 10: full hour = 3600s
	bucket10 := insights.ProductivityPerHourBreakdown[hourOf(10)]
	require.Equal(t, 3600, bucket10.DistractiveSeconds)

	// Hour 11: 45 min = 2700s
	bucket11 := insights.ProductivityPerHourBreakdown[hourOf(11)]
	require.Equal(t, 2700, bucket11.DistractiveSeconds)

	require.Len(t, insights.ProductivityPerHourBreakdown, 3)
}

func TestGetDayInsights_UsageWithinSingleHour(t *testing.T) {
	service, db := setUpService(t)

	// Usage from 14:10 to 14:25 (15 min = 900 seconds) – entirely within hour 14
	startAt := time.Date(2025, 6, 15, 14, 10, 0, 0, time.UTC)
	duration := 15 * 60

	appUsage := usage.ApplicationUsage{
		StartedAt:       startAt.Unix(),
		EndedAt:         int64Ptr(startAt.Unix() + int64(duration)),
		DurationSeconds: intPtr(duration),
		Classification:  usage.ClassificationNeutral,
		Application: usage.Application{
			Name:           "Finder",
			ExecutablePath: "/System/Library/CoreServices/Finder.app",
		},
	}
	require.NoError(t, db.Create(&appUsage).Error)

	insights, err := service.GetDayInsights(baseDate())
	require.NoError(t, err)

	require.Equal(t, 900, insights.ProductivityScore.OtherSeconds)

	bucket14, ok := insights.ProductivityPerHourBreakdown[hourOf(14)]
	require.True(t, ok, "expected a bucket for hour 14")
	require.Equal(t, 900, bucket14.OtherSeconds)

	require.Len(t, insights.ProductivityPerHourBreakdown, 1)
}

func TestGetDayInsights_MultipleUsagesSameHour(t *testing.T) {
	service, db := setUpService(t)

	// Two usages both within hour 8 – seconds should accumulate in the same bucket
	// Usage 1: productive 08:00-08:20 (1200s)
	// Usage 2: distracting 08:30-08:45 (900s)
	u1Start := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	u2Start := time.Date(2025, 6, 15, 8, 30, 0, 0, time.UTC)

	usages := []usage.ApplicationUsage{
		{
			StartedAt:       u1Start.Unix(),
			EndedAt:         int64Ptr(u1Start.Unix() + 1200),
			DurationSeconds: intPtr(1200),
			Classification:  usage.ClassificationProductive,
			Application:     usage.Application{Name: "VSCode", ExecutablePath: "/usr/bin/code"},
		},
		{
			StartedAt:       u2Start.Unix(),
			EndedAt:         int64Ptr(u2Start.Unix() + 900),
			DurationSeconds: intPtr(900),
			Classification:  usage.ClassificationDistracting,
			Application:     usage.Application{Name: "Twitter", ExecutablePath: "/Applications/Safari.app"},
		},
	}
	for _, u := range usages {
		require.NoError(t, db.Create(&u).Error)
	}

	insights, err := service.GetDayInsights(baseDate())
	require.NoError(t, err)

	bucket8, ok := insights.ProductivityPerHourBreakdown[hourOf(8)]
	require.True(t, ok)
	require.Equal(t, 1200, bucket8.ProductiveSeconds)
	require.Equal(t, 900, bucket8.DistractiveSeconds)

	// Productivity score for the hour: 1200 / (1200+900) = 57%
	require.Equal(t, 57, bucket8.ProductivityScore)

	require.Len(t, insights.ProductivityPerHourBreakdown, 1)
}

func TestGetDayInsights_NilAndZeroDurationSkipped(t *testing.T) {
	service, db := setUpService(t)

	// Usage with nil duration – should be skipped
	u1 := usage.ApplicationUsage{
		StartedAt:       time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC).Unix(),
		DurationSeconds: nil,
		Classification:  usage.ClassificationProductive,
		Application:     usage.Application{Name: "App1", ExecutablePath: "/bin/app1"},
	}
	// Usage with 0 duration – should be skipped
	u2 := usage.ApplicationUsage{
		StartedAt:       time.Date(2025, 6, 15, 11, 0, 0, 0, time.UTC).Unix(),
		DurationSeconds: intPtr(0),
		Classification:  usage.ClassificationProductive,
		Application:     usage.Application{Name: "App2", ExecutablePath: "/bin/app2"},
	}
	require.NoError(t, db.Create(&u1).Error)
	require.NoError(t, db.Create(&u2).Error)

	insights, err := service.GetDayInsights(baseDate())
	require.NoError(t, err)

	require.Equal(t, 0, insights.ProductivityScore.ProductiveSeconds)
	require.Len(t, insights.ProductivityPerHourBreakdown, 0)
}

func TestGetDayInsights_UsageExactlyOnHourBoundary(t *testing.T) {
	service, db := setUpService(t)

	// Usage starts exactly at 13:00 and lasts exactly 1 hour (3600s)
	// Should land entirely in the hour-13 bucket
	startAt := time.Date(2025, 6, 15, 13, 0, 0, 0, time.UTC)

	appUsage := usage.ApplicationUsage{
		StartedAt:       startAt.Unix(),
		EndedAt:         int64Ptr(startAt.Unix() + 3600),
		DurationSeconds: intPtr(3600),
		Classification:  usage.ClassificationProductive,
		Application:     usage.Application{Name: "IDE", ExecutablePath: "/bin/ide"},
	}
	require.NoError(t, db.Create(&appUsage).Error)

	insights, err := service.GetDayInsights(baseDate())
	require.NoError(t, err)

	bucket13, ok := insights.ProductivityPerHourBreakdown[hourOf(13)]
	require.True(t, ok)
	require.Equal(t, 3600, bucket13.ProductiveSeconds)

	require.Len(t, insights.ProductivityPerHourBreakdown, 1)
}

func TestGetDayInsights_PerHourProductivityScore(t *testing.T) {
	service, db := setUpService(t)

	// Hour 15: 40 min productive (2400s) + 20 min distracting crossing into hour 16
	// Productive: 15:00 – 15:40 (2400s, entirely in hour 15)
	// Distracting: 15:50 – 16:10 (1200s, 600s in hour 15, 600s in hour 16)
	u1Start := time.Date(2025, 6, 15, 15, 0, 0, 0, time.UTC)
	u2Start := time.Date(2025, 6, 15, 15, 50, 0, 0, time.UTC)

	usages := []usage.ApplicationUsage{
		{
			StartedAt:       u1Start.Unix(),
			EndedAt:         int64Ptr(u1Start.Unix() + 2400),
			DurationSeconds: intPtr(2400),
			Classification:  usage.ClassificationProductive,
			Application:     usage.Application{Name: "Code", ExecutablePath: "/bin/code"},
		},
		{
			StartedAt:       u2Start.Unix(),
			EndedAt:         int64Ptr(u2Start.Unix() + 1200),
			DurationSeconds: intPtr(1200),
			Classification:  usage.ClassificationDistracting,
			Application:     usage.Application{Name: "Reddit", ExecutablePath: "/Applications/Safari.app"},
		},
	}
	for _, u := range usages {
		require.NoError(t, db.Create(&u).Error)
	}

	insights, err := service.GetDayInsights(baseDate())
	require.NoError(t, err)

	// Hour 15: 2400s productive + 600s distracting => score = 2400/3000 = 80%
	bucket15 := insights.ProductivityPerHourBreakdown[hourOf(15)]
	require.Equal(t, 2400, bucket15.ProductiveSeconds)
	require.Equal(t, 600, bucket15.DistractiveSeconds)
	require.Equal(t, 80, bucket15.ProductivityScore)

	// Hour 16: 600s distracting only => score = 0%
	bucket16 := insights.ProductivityPerHourBreakdown[hourOf(16)]
	require.Equal(t, 0, bucket16.ProductiveSeconds)
	require.Equal(t, 600, bucket16.DistractiveSeconds)
	require.Equal(t, 0, bucket16.ProductivityScore)
}
