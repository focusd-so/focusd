package usage

import (
	"google.golang.org/genai"

	"github.com/focusd-so/focusd/internal/settings"
)

// Option is a function that configures a Service.
type Option func(*Service)

// WithAppBlocker configures the Service with a function to call when an application is blocked.
func WithAppBlocker(appBlocker func(appName, title, reason string, tags []string, browserURL *string)) Option {
	return func(s *Service) {
		s.appBlocker = appBlocker
	}
}

// WithSettingsService configures the Service with a SettingsService interface
// to allow accessing settings from the database.
func WithSettingsService(settingsService *settings.Service) Option {
	return func(s *Service) {
		s.settingsService = settingsService
	}
}

// WithGenaiClient configures the Service with a GenaiClient interface
// to allow using the Genai client.
func WithGenaiClient(genaiClient *genai.Client) Option {
	return func(s *Service) {
		s.genaiClient = genaiClient
	}
}

// WithProtectionPaused configures the Service with a function to call when protection is paused.
func WithProtectionPaused(onProtectionPaused func(pause ProtectionPause)) Option {
	return func(s *Service) {
		s.onProtectionPaused = onProtectionPaused
	}
}

// WithProtectionResumed configures the Service with a function to call when protection is resumed.
func WithProtectionResumed(onProtectionResumed func(pause ProtectionPause)) Option {
	return func(s *Service) {
		s.onProtectionResumed = onProtectionResumed
	}
}

// WithLLMDailySummaryReady configures the Service with a function to call when a daily LLM summary is generated.
func WithLLMDailySummaryReady(fn func(summary LLMDailySummary)) Option {
	return func(s *Service) {
		s.onLLMDailySummaryReady = fn
	}
}
