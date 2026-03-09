package usage

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/genai"
	"gorm.io/gorm"

	"github.com/focusd-so/focusd/internal/settings"
)

type Service struct {
	// external services and dependencies
	db          *gorm.DB
	genaiClient *genai.Client

	// internal services and dependencies
	settingsService *settings.Service

	appBlocker func(appName, title, reason string, tags []string, browserURL *string)

	// events
	onProtectionPaused     func(pause ProtectionPause)
	onProtectionResumed    func(pause ProtectionPause)
	onLLMDailySummaryReady func(summary LLMDailySummary)

	// mu serializes title change processing to prevent race conditions
	// when multiple events fire concurrently
	mu sync.Mutex

	// channel to receive usage updates
	UsageUpdates chan *ApplicationUsage
}

func NewService(ctx context.Context, db *gorm.DB, options ...Option) (*Service, error) {
	if err := db.AutoMigrate(
		&Application{},
		&ApplicationUsage{},
		&ApplicationUsageTags{},
		&ProtectionPause{},
		&ProtectionWhitelist{},
		&SandboxExecutionLog{},
		&LLMDailySummary{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate usage tables: %w", err)
	}

	service := &Service{
		db:           db,
		UsageUpdates: make(chan *ApplicationUsage, 10), // buffer of 10 to prevent blocking
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
	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)

	return s.db.Where("created_at < ?", sevenDaysAgo).Delete(&SandboxExecutionLog{}).Error
}
