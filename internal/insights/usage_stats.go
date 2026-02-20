package insights

// import (
// 	"time"

// 	"github.com/focusd-so/focusd/internal/usage"
// )

// const weeklyStatsQuery = `
// SELECT
// 	classification,
// 	SUM(duration) as duration
// FROM application_usages au
// WHERE start_at >= strftime('%s', date('now', '-6 days'))
// GROUP BY classification, date(start_at, 'unixepoch')
// ORDER BY date(start_at, 'unixepoch')
// `

// const dailyUsageStatsQuery = `
// SELECT
// 	classification,
// 	SUM(duration) as duration
// FROM application_usages au
// WHERE date(start_at, 'unixepoch') = ?
// `

// // GetWeeklyStats returns daily stats for the last 7 days (including today)
// func (s *Service) GetWeeklyStats() (UsageStats, error) {
// 	type dbRow struct {
// 		Classification usage.Classification
// 		Duration       int64
// 	}

// 	var results []dbRow

// 	if err := s.db.Raw(weeklyStatsQuery).Scan(&results).Error; err != nil {
// 		return UsageStats{}, err
// 	}

// 	stats := UsageStats{
// 		ProductiveMinutes:  0,
// 		OtherMinutes:       0,
// 		DistractiveMinutes: 0,
// 		ProductivityScore:  0,
// 	}

// 	for _, result := range results {
// 		switch result.Classification {
// 		case usage.ClassificationProductive:
// 			stats.ProductiveMinutes += result.Duration
// 		case usage.ClassificationNeutral:
// 			stats.OtherMinutes += result.Duration
// 		case usage.ClassificationDistracting:
// 			stats.DistractiveMinutes += result.Duration
// 		}
// 	}

// 	totalMinutes := stats.ProductiveMinutes + stats.OtherMinutes + stats.DistractiveMinutes
// 	if totalMinutes > 0 {
// 		stats.ProductivityScore = float64(stats.ProductiveMinutes) / float64(totalMinutes) * 100
// 	}

// 	return stats, nil
// }

// func (s *Service) GetDailyUsageStats(date time.Time) (UsageStats, error) {
// 	type dbRow struct {
// 		Classification usage.Classification
// 		Duration       int64
// 	}

// 	var results []dbRow

// 	if err := s.db.Raw(dailyUsageStatsQuery, date.Format("2006-01-02")).Scan(&results).Error; err != nil {
// 		return UsageStats{}, err
// 	}

// 	stats := UsageStats{
// 		ProductiveMinutes:  0,
// 		OtherMinutes:       0,
// 		DistractiveMinutes: 0,
// 		ProductivityScore:  0,
// 	}

// 	for _, result := range results {
// 		switch result.Classification {
// 		case usage.ClassificationProductive:
// 			stats.ProductiveMinutes += result.Duration
// 		case usage.ClassificationNeutral:
// 			stats.OtherMinutes += result.Duration
// 		case usage.ClassificationDistracting:
// 			stats.DistractiveMinutes += result.Duration
// 		}
// 	}

// 	totalMinutes := stats.ProductiveMinutes + stats.OtherMinutes + stats.DistractiveMinutes
// 	if totalMinutes > 0 {
// 		stats.ProductivityScore = float64(stats.ProductiveMinutes) / float64(totalMinutes) * 100
// 	}

// 	return stats, nil
// }
