package usage

import "time"

type GetUsageListOptions struct {
	Date            *time.Time
	Page            *int
	PageSize        *int
	StartedAt       *time.Time
	EndedAt         *time.Time
	TerminationMode *TerminationMode
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
	if err := query.Find(&usages).Error; err != nil {
		return nil, err
	}
	return usages, nil
}
