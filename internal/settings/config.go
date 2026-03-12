package settings

type LLMProvider string

const (
	LLMProviderGoogle    LLMProvider = "google"
	LLMProviderOpenAI    LLMProvider = "openai"
	LLMProviderGroq      LLMProvider = "groq"
	LLMProviderAnthropic LLMProvider = "anthropic"
	LLMProviderDummy     LLMProvider = "dummy"
)

// AppConfig represents all configurable options for the application
type AppConfig struct {
	IdleThresholdSeconds        int         `json:"idle_threshold_seconds" mapstructure:"idle_threshold_seconds"`
	HistoryRetentionDays        int         `json:"history_retention_days" mapstructure:"history_retention_days"`
	DistractionAllowanceMinutes int         `json:"distraction_allowance_minutes" mapstructure:"distraction_allowance_minutes"`
	CustomRulesJS               []string    `json:"custom_rules_js" mapstructure:"custom_rules_js"`
	ClassificationLLMProvider   LLMProvider `json:"classification_llm_provider" mapstructure:"classification_llm_provider"`
}

// DefaultConfig returns the default application configuration
func DefaultConfig() AppConfig {
	return AppConfig{
		IdleThresholdSeconds:        120, // 2 minutes
		HistoryRetentionDays:        30,  // 30 days
		DistractionAllowanceMinutes: 60,  // 1 hour
		CustomRulesJS:               []string{},
		ClassificationLLMProvider:   LLMProviderGoogle,
	}
}

func IsProductionBuild() bool {
	return isProductionBuild
}

func APIBaseURL() string {
	if IsProductionBuild() {
		return "https://api.focusd.so"
	}
	return "http://localhost:8089"
}
