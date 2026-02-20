package insights

// import (
// 	"context"
// 	"fmt"
// 	"sort"
// 	"time"

// 	"gorm.io/gorm"
// )

// type ContentGenerator interface {
// 	GenerateContent(ctx context.Context, instructions, input string, out any) error
// }

// type Service struct {
// 	db               *gorm.DB
// 	contentGenerator ContentGenerator
// }

// func NewService(db *gorm.DB, contentGenerator ContentGenerator) (*Service, error) {
// 	if err := db.AutoMigrate(&DailyUsageSummary{}); err != nil {
// 		return nil, err
// 	}

// 	s := &Service{db: db, contentGenerator: contentGenerator}

// 	go s.scheduleSummaryGeneration(context.Background())

// 	return s, nil
// }

// func (s *Service) GetOverview(date time.Time) (DailyOverview, error) {
// 	usages, err := getUsage(s.db, date)
// 	if err != nil {
// 		return DailyOverview{}, err
// 	}

// 	return s.getOverview(date, usages)
// }

// func (s *Service) GetUsageList(date time.Time) ([]ApplicationUsage, error) {
// 	return getUsage(s.db, date)
// }

// func (s *Service) getOverview(date time.Time, usages []ApplicationUsage) (DailyOverview, error) {
// 	productivityOverview := getDailyOverview(usages)
// 	perHourBreakdown, err := getPerHourBreakdown(usages)

// 	if err != nil {
// 		return DailyOverview{}, err
// 	}

// 	// read daily summary from db
// 	var summary DailyUsageSummary
// 	err = s.db.Where("date = ?", date.Format("2006-01-02")).First(&summary).Error

// 	dailyOverview := DailyOverview{
// 		Date:                  date,
// 		UsageOverview:         productivityOverview,
// 		UsagePerHourBreakdown: perHourBreakdown,
// 	}

// 	if err == nil {
// 		dailyOverview.DailyUsageSummary = summary
// 	}

// 	return dailyOverview, nil
// }

// func getUsage(db *gorm.DB, date time.Time) ([]ApplicationUsage, error) {
// 	var usages []ApplicationUsage
// 	if err := db.Preload("Application").Preload("Tags").
// 		Where("date(start_at, 'unixepoch', 'localtime') = ? AND is_idle = 0", date.Format("2006-01-02")).
// 		Order("start_at ASC").
// 		Find(&usages).Error; err != nil {
// 		return nil, err
// 	}

// 	return usages, nil
// }

// func getDailyOverview(usages []ApplicationUsage) UsageOverview {
// 	var overview UsageOverview

// 	for _, usage := range usages {
// 		if usage.Classification == "productive" {
// 			overview.ProductiveSeconds += int(usage.Duration)
// 		} else if usage.Classification == "distracting" {
// 			overview.DistractiveSeconds += int(usage.Duration)
// 		} else {
// 			overview.OtherSeconds += int(usage.Duration)
// 		}
// 	}

// 	overview.ProductivityScore = calculateProductivityScore(overview.ProductiveSeconds, overview.DistractiveSeconds)

// 	return overview
// }

// func getPerHourBreakdown(usages []ApplicationUsage) ([]*UsagePerHourBreakdown, error) {
// 	perHourBreakdown := make(map[string]*UsagePerHourBreakdown)

// 	for i := 0; i < 24; i++ {
// 		perHourBreakdown[fmt.Sprintf("%02d", i)] = &UsagePerHourBreakdown{
// 			HourLabel:          fmt.Sprintf("%02d", i),
// 			ProductiveSeconds:  0,
// 			DistractiveSeconds: 0,
// 			ProductivityScore:  0,
// 		}
// 	}

// 	for _, usage := range usages {
// 		timeToday := time.Unix(usage.StartAt, 0)
// 		hourKey := timeToday.Format("15")

// 		switch usage.Classification {
// 		case "productive":
// 			perHourBreakdown[hourKey].ProductiveSeconds += int(usage.Duration)
// 		case "distracting":
// 			perHourBreakdown[hourKey].DistractiveSeconds += int(usage.Duration)
// 		default:
// 			perHourBreakdown[hourKey].OtherSeconds += int(usage.Duration)
// 		}
// 	}

// 	// Calculate Productivity Score per Hour and convert to slice
// 	result := make([]*UsagePerHourBreakdown, 0, len(perHourBreakdown))
// 	for _, usage := range perHourBreakdown {
// 		if usage.ProductiveSeconds > 0 {
// 			usage.ProductivityScore = calculateProductivityScore(usage.ProductiveSeconds, usage.DistractiveSeconds)
// 		}
// 		result = append(result, usage)
// 	}

// 	sort.Slice(result, func(i, j int) bool {
// 		return result[i].HourLabel < result[j].HourLabel
// 	})

// 	return result, nil
// }

// func calculateProductivityScore(productiveDuration, distractingDuration int) int {
// 	sum := productiveDuration + distractingDuration

// 	if sum == 0 {
// 		return 0
// 	}

// 	return (productiveDuration * 100.0) / sum
// }

// //func getPerHourBreakdown(db *gorm.DB, date time.Time) ([]UsagePerHourBreakdown, error) {
// //	query := `
// //	WITH RECURSIVE hours(h) AS (
// //		SELECT 0
// //		UNION ALL
// //		SELECT h + 1 FROM hours WHERE h < 23
// //	)
// //	SELECT
// //		printf('%02d:00', hours.h) as hour_label,
// //		COALESCE(SUM(CASE WHEN classification = 'productive' THEN duration ELSE 0 END), 0) as productive,
// //		COALESCE(SUM(CASE WHEN classification = 'distracting' THEN duration ELSE 0 END), 0) as distracting,
// //		COALESCE(SUM(CASE WHEN classification NOT IN ('productive', 'distracting') THEN duration ELSE 0 END), 0) as other,
// //		-- Optional: Productivity Score per Hour
// //		CASE
// //			WHEN SUM(CASE WHEN classification IN ('productive', 'distracting') THEN duration ELSE 0 END) > 0
// //			THEN ROUND((SUM(CASE WHEN classification = 'productive' THEN duration ELSE 0 END) * 100.0) /
// //				SUM(CASE WHEN classification IN ('productive', 'distracting') THEN duration ELSE 0 END), 1)
// //			ELSE 0
// //		END as score
// //	FROM hours
// //	LEFT JOIN application_usages ON
// //		strftime('%H', start_at, 'unixepoch', 'localtime') = printf('%02d', hours.h)
// //		AND date(start_at, 'unixepoch', 'localtime') = ?
// //	GROUP BY hours.h
// //	ORDER BY hours.h ASC;
// //	`
// //
// //	var productivityOverviews []UsagePerHourBreakdown
// //	if err := db.Raw(query, date.Format("2006-01-02")).Scan(&productivityOverviews).Error; err != nil {
// //		return nil, err
// //	}
// //
// //	return productivityOverviews, nil
// //}
