package usage

import "time"

type GetUsageListOptions struct {
	Date            *time.Time
	Page            *int
	PageSize        *int
	StartedAt       *time.Time
	EndedAt         *time.Time
	TerminationMode *TerminationMode
	Classification  *Classification
	ApplicationID   *int64
	ApplicationName *string
	Hostname        *string
	BundleID        *string
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
func (s *Service) GetUsageList(options GetUsageListOptions) ([]ApplicationUsage, error) {
	var usages []ApplicationUsage
	query := s.db.Preload("Application").Preload("Tags").Order("started_at DESC")
	if options.Page != nil && options.PageSize != nil {
		query = query.Offset(*options.Page * *options.PageSize).Limit(*options.PageSize)
	}
	if options.Date != nil {
		query = query.Where("date(started_at, 'unixepoch') = ?", options.Date.Format("2006-01-02"))
	}
	if options.StartedAt != nil {
		query = query.Where("started_at >= ?", options.StartedAt.Unix())
	}
	if options.EndedAt != nil {
		query = query.Where("ended_at <= ?", options.EndedAt.Unix())
	}
	if options.TerminationMode != nil {
		query = query.Where("termination_mode = ?", *options.TerminationMode)
	}
	if options.Classification != nil {
		query = query.Where("classification = ?", *options.Classification)
	}
	if options.ApplicationID != nil {
		query = query.Where("application_id = ?", *options.ApplicationID)
	}

	if options.Hostname != nil || options.BundleID != nil || options.ApplicationName != nil {
		query = query.Joins("JOIN application ON application.id = application_usage.application_id")
		if options.Hostname != nil {
			query = query.Where("application.hostname = ?", *options.Hostname)
		}
		if options.BundleID != nil {
			query = query.Where("application.bundle_id = ?", *options.BundleID)
		}
		if options.ApplicationName != nil {
			query = query.Where("application.name = ?", *options.ApplicationName)
		}
	}

	if err := query.Find(&usages).Error; err != nil {
		return nil, err
	}
	return usages, nil
}

func (s *Service) GetUsageAggregation(options GetUsageListOptions) ([]UsageAggregation, error) {
	var results []struct {
		ApplicationID int64
		TotalDuration int
		UsageCount    int
	}

	query := s.db.Model(&ApplicationUsage{}).
		Select("application_id, sum(duration_seconds) as total_duration, count(*) as usage_count").
		Group("application_id").
		Order("total_duration DESC")

	if options.Date != nil {
		query = query.Where("date(started_at, 'unixepoch') = ?", options.Date.Format("2006-01-02"))
	}
	if options.StartedAt != nil {
		query = query.Where("started_at >= ?", options.StartedAt.Unix())
	}
	if options.EndedAt != nil {
		query = query.Where("ended_at <= ?", options.EndedAt.Unix())
	}
	if options.TerminationMode != nil {
		query = query.Where("termination_mode = ?", *options.TerminationMode)
	}
	if options.Classification != nil {
		query = query.Where("classification = ?", *options.Classification)
	}

	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	var aggregation []UsageAggregation
	for _, res := range results {
		var app Application
		if err := s.db.First(&app, res.ApplicationID).Error; err != nil {
			continue
		}
		aggregation = append(aggregation, UsageAggregation{
			Application:   app,
			TotalDuration: res.TotalDuration,
			UsageCount:    res.UsageCount,
		})
	}

	return aggregation, nil
}
