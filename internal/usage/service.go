package usage

import (
	"context"
	"fmt"
	"log/slog"
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
	onProtectionPaused  func(pause ProtectionPause)
	onProtectionResumed func(pause ProtectionPause)

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
		&IdlePeriod{},
		&SandboxExecutionLog{},
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
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Hour):
				if err := s.removeOldSandboxExecutionLogs(ctx); err != nil {
					slog.Error("failed to cleanup sandbox execution logs", "error", err)
				}
			}
		}
	}()
}

// cleanupSandboxExecutionLogs deletes sandbox execution logs older than two weeks.
func (s *Service) removeOldSandboxExecutionLogs(ctx context.Context) error {
	twoWeeksAgo := time.Now().Add(-2 * 7 * 24 * time.Hour)

	return s.db.Where("created_at < ?", twoWeeksAgo).Delete(&SandboxExecutionLog{}).Error
}
