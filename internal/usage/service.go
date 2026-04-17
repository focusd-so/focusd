package usage

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/focusd-so/focusd/internal/timeline"
)

type Service struct {
	// external services and dependencies
	timelineService *timeline.Service

	appBlocker func(appName, title, reason string, tags []string, browserURL *string)

	// mu serializes title change processing to prevent race conditions
	// when multiple events fire concurrently
	mu sync.Mutex

	db *gorm.DB
}

func NewService(ctx context.Context, db *gorm.DB, timelineService *timeline.Service, options ...Option) (*Service, error) {
	service := &Service{timelineService: timelineService, db: db}

	if err := db.AutoMigrate(&Application{}); err != nil {
		return nil, fmt.Errorf("failed to migrate application: %w", err)
	}

	for _, option := range options {
		option(service)
	}

	go service.scheduleJobs(ctx)

	return service, nil
}

func (s *Service) scheduleJobs(ctx context.Context) {
	fn := func(ctx context.Context) error {
		if err := s.removeOldSandboxExecutionLogs(ctx); err != nil {
			slog.Error("failed to remove old sandbox execution logs", "error", err)
		}
		if err := s.GenerateLLMDailySummaryIfNeeded(ctx); err != nil {
			slog.Error("failed to generate daily summary", "error", err)
		}

		return nil
	}

	fn(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Hour):
				fn(ctx)
			}
		}
	}()
}

// removeOldSandboxExecutionLogs deletes sandbox execution logs older than 7 days.
func (s *Service) removeOldSandboxExecutionLogs(ctx context.Context) error {
	// TODO: delete using timeline service instead of accessing db directly

	// sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
	// return s.db.Where("created_at < ?", sevenDaysAgo).Delete(&SandboxExecutionLog{}).Error

	return nil
}

func (s *Service) CloseLastActiveUsageEvent() error {
	lastEvent, err := s.timelineService.GetActiveEventOfTypes(EventTypeUsageChanged, EventTypeUserIdleChanged)
	if err != nil {
		return err
	}

	if lastEvent != nil {
		return s.timelineService.EventFinished(lastEvent)
	}

	return nil
}
