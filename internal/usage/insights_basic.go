package usage

import (
	"time"

	"gorm.io/gorm"
)

type GetUsageListOptions struct {
	Date              *time.Time
	Page              *int
	PageSize          *int
	StartedAt         *time.Time
	EndedAt           *time.Time
	EnforcementAction *EnforcementAction
	Classification    *Classification
	ApplicationID     *int64
	ApplicationName   *string
	Hostname          *string
}

type UsageAggregation struct {
	Application   Application `json:"application"`
	TotalDuration int         `json:"total_duration"` // in seconds
	UsageCount    int         `json:"usage_count"`
}

// GetUsageList returns a list of application usages for the given date and pagination options
//
// Parameters:
//   - options: GetUsageListOptions
//   - Date: The date to get usages for - useful for detailed analysis of a specific day usage
//   - Page: The page number to get
//   - PageSize: The number of usages per page
//
// Returns:
//   - []ApplicationUsage: The list of application usages
// func (s *Service) GetUsageList(options GetUsageListOptions) ([]ApplicationUsage, error) {
// 	var usages []ApplicationUsage
// 	query := s.db.Preload("Application").Preload("Tags").Order("started_at DESC")
// 	if options.Page != nil && options.PageSize != nil {
// 		query = query.Offset(*options.Page * *options.PageSize).Limit(*options.PageSize)
// 	}
// 	if options.Date != nil {
// 		query = query.Where("date(started_at, 'unixepoch') = ?", options.Date.Format("2006-01-02"))
// 	}
// 	if options.StartedAt != nil {
// 		query = query.Where("started_at >= ?", options.StartedAt.Unix())
// 	}
// 	if options.EndedAt != nil {
// 		query = query.Where("ended_at <= ?", options.EndedAt.Unix())
// 	}
// 	if options.EnforcementAction != nil {
// 		query = query.Where("enforcement_action = ?", *options.EnforcementAction)
// 	}
// 	if options.Classification != nil {
// 		query = query.Where("classification = ?", *options.Classification)
// 	}
// 	if options.ApplicationID != nil {
// 		query = query.Where("application_id = ?", *options.ApplicationID)
// 	}

// 	if options.Hostname != nil || options.ApplicationName != nil {
// 		query = query.Joins("JOIN application ON application.id = application_usage.application_id")
// 		if options.Hostname != nil {
// 			query = query.Where("application.hostname = ?", *options.Hostname)
// 		}
// 		if options.ApplicationName != nil {
// 			query = query.Where("application.name = ?", *options.ApplicationName)
// 		}
// 	}

// 	if err := query.Find(&usages).Error; err != nil {
// 		return nil, err
// 	}
// 	return usages, nil
// }

func (s *Service) GetApplicationList() ([]Application, error) {
	var applications []Application
	if err := s.db.
		Order("last_used_at DESC").
		Limit(50).
		Find(&applications).Error; err != nil {
		return nil, err
	}
	return applications, nil
}

func (s *Service) GetApplicationByID(id int64) (*Application, error) {
	var application Application
	if err := s.db.First(&application, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &application, nil
}

// func (s *Service) GetUsageAggregation(options GetUsageListOptions) ([]UsageAggregation, error) {
// 	var usages []ApplicationUsage

// 	// Reuse GetUsageList logic to get the raw sessions for the period
// 	query := s.db.Preload("Application").Order("started_at ASC")
// 	if options.Date != nil {
// 		query = query.Where("date(started_at, 'unixepoch') = ?", options.Date.Format("2006-01-02"))
// 	}
// 	if options.StartedAt != nil {
// 		query = query.Where("started_at >= ?", options.StartedAt.Unix())
// 	}
// 	if options.EndedAt != nil {
// 		query = query.Where("ended_at <= ?", options.EndedAt.Unix())
// 	}
// 	if options.EnforcementAction != nil {
// 		query = query.Where("enforcement_action = ?", *options.EnforcementAction)
// 	}
// 	if options.Classification != nil {
// 		query = query.Where("classification = ?", *options.Classification)
// 	}

// 	if err := query.Find(&usages).Error; err != nil {
// 		return nil, err
// 	}

// 	// Group intervals by application
// 	type appInterval struct {
// 		start int64
// 		end   int64
// 	}
// 	intervalsByApp := make(map[int64][]appInterval)
// 	apps := make(map[int64]Application)
// 	sessionsCount := make(map[int64]int)

// 	for _, u := range usages {
// 		if u.DurationSeconds == nil || *u.DurationSeconds <= 0 {
// 			continue
// 		}
// 		intervalsByApp[u.ApplicationID] = append(intervalsByApp[u.ApplicationID], appInterval{
// 			start: u.StartedAt,
// 			end:   u.StartedAt + int64(*u.DurationSeconds),
// 		})
// 		apps[u.ApplicationID] = u.Application
// 		sessionsCount[u.ApplicationID]++
// 	}

// 	var aggregation []UsageAggregation
// 	for appID, intervals := range intervalsByApp {
// 		// Merge overlapping intervals for this specific app
// 		if len(intervals) == 0 {
// 			continue
// 		}
// 		sort.Slice(intervals, func(i, j int) bool {
// 			return intervals[i].start < intervals[j].start
// 		})

// 		merged := []appInterval{intervals[0]}
// 		for i := 1; i < len(intervals); i++ {
// 			last := &merged[len(merged)-1]
// 			current := intervals[i]
// 			if current.start <= last.end {
// 				if current.end > last.end {
// 					last.end = current.end
// 				}
// 			} else {
// 				merged = append(merged, current)
// 			}
// 		}

// 		totalDuration := 0
// 		for _, inv := range merged {
// 			totalDuration += int(inv.end - inv.start)
// 		}

// 		aggregation = append(aggregation, UsageAggregation{
// 			Application:   apps[appID],
// 			TotalDuration: totalDuration,
// 			UsageCount:    sessionsCount[appID],
// 		})
// 	}

// 	// Sort by duration descending
// 	sort.Slice(aggregation, func(i, j int) bool {
// 		return aggregation[i].TotalDuration > aggregation[j].TotalDuration
// 	})

// 	return aggregation, nil
// }
