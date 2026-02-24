package usage

import (
	"time"
)

// GetSandboxExecutionLogs retrieves paginated sandbox execution logs from the database.
// It supports fuzzy search on context and response fields, and filtering by log type.
// Only returns logs from the last 7 days.
func (s *Service) GetSandboxExecutionLogs(logType string, search string, page, pageSize int) ([]SandboxExecutionLog, error) {
	if pageSize <= 0 {
		pageSize = 50
	}
	if page < 0 {
		page = 0
	}

	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour).Unix()

	query := s.db.Where("created_at >= ?", sevenDaysAgo)

	if logType != "" {
		query = query.Where("type = ?", logType)
	}

	if search != "" {
		likePattern := "%" + search + "%"
		query = query.Where("context LIKE ? OR response LIKE ? OR logs LIKE ?", likePattern, likePattern, likePattern)
	}

	var logs []SandboxExecutionLog
	err := query.
		Order("created_at DESC").
		Offset(page * pageSize).
		Limit(pageSize).
		Find(&logs).Error

	if err != nil {
		return nil, err
	}

	return logs, nil
}
