package settings

import "encoding/base64"

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
		CustomRulesJS:               []string{base64.StdEncoding.EncodeToString([]byte(DefaultCustomRulesJS))},
		ClassificationLLMProvider:   LLMProviderGoogle,
	}
}

const DefaultCustomRulesJS = `import {
  productive,
  distracting,
  block,
  Timezone,
  runtime,
  type Classify,
  type Enforce,
} from "@focusd/runtime";

/**
 * Classify determines whether the current app or website is productive or distracting.
 * It is called every time your usage changes.
 */
export function classify(): Classify | undefined {
  const { domain, app, current } = runtime.usage;
  
  // --- EXAMPLES ---
  //
  // 1. Mark specific domains as productive
  // if (domain === "github.com") return productive("Working on code");
  // 
  // 2. Mark distracting based on today's stats & current usage
  // if (domain === "twitter.com" && current.usedToday > 30) {
  //   return distracting("Daily limit reached");
  // }
  //
  // 3. Use current hour stats to mark distraction
  // if (runtime.hour.distractingMinutes > 20 && current.last(60) > 15) {
  //   return distracting("Too much distraction this hour");
  // }

  return undefined;
}

/**
 * Enforcement determines whether or not to block the current app or website.
 * It is called when the current usage has been classified as distracting.
 */
export function enforcement(): Enforce | undefined {
  const { domain } = runtime.usage;

  // --- EXAMPLES ---
  //
  // 1. Block if daily distraction limit is reached
  // if (runtime.today.distractingMinutes > 60) {
  //   return block("Daily distraction limit reached");
  // }
  //
  // 2. Block after a specific time (e.g., 10 PM in London)
  // const hour = runtime.time.now(Timezone.Europe_London).getHours();
  // if (domain === "youtube.com" && hour >= 22) {
  //   return block("Late night blocking");
  // }

  return undefined;
}
`

func IsProductionBuild() bool {
	return isProductionBuild
}

func APIBaseURL() string {
	if IsProductionBuild() {
		return "https://api.focusd.so"
	}
	return "http://localhost:8089"
}
