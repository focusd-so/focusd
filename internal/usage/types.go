package usage

import "github.com/focusd-so/focusd/internal/identity"

type (
	EnforcementAction    string
	EnforcementSource    string
	Classification       string
	ClassificationSource string
)

const (
	EnforcementActionNone   EnforcementAction = "none"
	EnforcementActionBlock  EnforcementAction = "block"
	EnforcementActionPaused EnforcementAction = "paused"
	EnforcementActionAllow  EnforcementAction = "allow"

	EnforcementSourceApplication EnforcementSource = "application"
	EnforcementSourceCustomRules EnforcementSource = "custom_rules"
	EnforcementSourceAllowed     EnforcementSource = "allowed"
	EnforcementSourcePaused      EnforcementSource = "paused"

	ClassificationSourceNone         ClassificationSource = "none"
	ClassificationSourceObviously    ClassificationSource = "obviously"
	ClassificationSourceCustomRules  ClassificationSource = "custom_rules"
	ClassificationSourceLLMGemini    ClassificationSource = "llm_gemini"
	ClassificationSourceLLMOpenAI    ClassificationSource = "llm_openai"
	ClassificationSourceLLMGroq      ClassificationSource = "llm_grok"
	ClassificationSourceLLMAnthropic ClassificationSource = "llm_anthropic"

	ClassificationNone        Classification = "none"
	ClassificationProductive  Classification = "productive"
	ClassificationDistracting Classification = "distracting"
	ClassificationNeutral     Classification = "neutral"
	ClassificationSystem      Classification = "system"

	TagTypeClassificationTag = "classification_tag"
	TagTypeClassification    = "classification"

	EventTypeProtectionStatusChanged = "protection_status_changed"
	EventTypeAllowUsage              = "allow_usage"
	EventTypeUsageChanged            = "usage_changed"
	EventTypeUserIdleChanged         = "user_idle_changed"
)

func (c Classification) IsProductiveOrDistracting() bool {
	return c == ClassificationProductive || c == ClassificationDistracting
}

type BasicClassificationResult struct {
	Classification       Classification `json:"classification"`
	ClassificationReason string         `json:"classification_reason"`
	Tags                 []string       `json:"tags"`
}

type CustomRulesClassificationResult struct {
	BasicClassificationResult

	// Custom rules classification only
	SandboLogs     []string `json:"sandbox_logs"`
	SandboxContext string   `json:"sandbox_context"`
	SanboxOutput   *string  `json:"sandbox_output,omitempty"`
	SandboxError   *string  `json:"sandbox_error,omitempty"`
}

type LLMClassificationResult struct {
	BasicClassificationResult

	ConfidenceScore float32 `json:"confidence_score"`
	// DetectedProject is inferred by the LLM from the window title or channel name.
	// For coding apps (VS Code, Xcode, etc.), it extracts the workspace/project name from the title format.
	// For communication apps (Slack), it extracts the project/team context if strongly implied by the channel name.
	DetectedProject string `json:"detected_project"`
	// DetectedCommunicationChannel is inferred by the LLM from the window title for communication apps.
	// E.g., for Slack it extracts "engineering" from "Slack | #engineering | Acme Corp".
	// This is only populated when the "communication" tag is assigned.
	DetectedCommunicationChannel string `json:"detected_communication_channel"`

	ClassificationSource ClassificationSource `json:"classification_source"`
}

type ObviouslyClassificationResult struct {
	BasicClassificationResult
}

type ClassificationResult struct {
	CustomRulesClassificationResult *CustomRulesClassificationResult `json:"custom_rules_classification_result,omitempty"`
	LLMClassificationResult         *LLMClassificationResult         `json:"llm_classification_result,omitempty"`
	ObviouslyClassificationResult   *ObviouslyClassificationResult   `json:"obviously_classification_result,omitempty"`
}

func (cr ClassificationResult) Classification() Classification {
	if cr.CustomRulesClassificationResult != nil {
		if identity.HasPremiumFeatures() {
			return cr.CustomRulesClassificationResult.Classification
		}
	}

	if cr.ObviouslyClassificationResult != nil {
		return cr.ObviouslyClassificationResult.Classification
	}

	if cr.LLMClassificationResult != nil {
		return cr.LLMClassificationResult.Classification
	}

	return ClassificationNone
}

func (cr ClassificationResult) ClassificationSource() ClassificationSource {
	if cr.CustomRulesClassificationResult != nil {
		if identity.HasPremiumFeatures() {
			return ClassificationSourceCustomRules
		}
	}

	if cr.ObviouslyClassificationResult != nil {
		return ClassificationSourceObviously
	}

	if cr.LLMClassificationResult != nil {
		return cr.LLMClassificationResult.ClassificationSource
	}

	return ClassificationSourceNone
}

func (cr ClassificationResult) ClassificationReason() string {
	switch cr.ClassificationSource() {
	case ClassificationSourceCustomRules:
		return cr.CustomRulesClassificationResult.ClassificationReason
	case ClassificationSourceObviously:
		return cr.ObviouslyClassificationResult.ClassificationReason
	default:
		return cr.LLMClassificationResult.ClassificationReason
	}
}

func (cr ClassificationResult) Tags() []string {
	switch cr.ClassificationSource() {
	case ClassificationSourceCustomRules:
		return cr.CustomRulesClassificationResult.Tags
	case ClassificationSourceObviously:
		return cr.ObviouslyClassificationResult.Tags
	default:
		return cr.LLMClassificationResult.Tags
	}
}

// Application represents an application that has been used by the user.
// It tracks both native applications and browser-based applications.
type Application struct {
	// mandatory fields
	ID         int64  `json:"id" gorm:"primaryKey;autoIncrement;not null"`
	Name       string `json:"name" gorm:"index:idx_application_name;not null"`
	LastUsedAt int64  `json:"last_used_at" gorm:"index:idx_application_last_used_at;not null;default:0"`

	Icon   *string `json:"icon"` // either app icon or favicon if host is present
	Domain *string `json:"domain"`

	// darwin only
	AppCategory *string `json:"app_category"` // LSApplicationCategoryType, e.g. "public.app-category.developer-tools"
}

func (a Application) TableName() string {
	return "application"
}

type DayInsights struct {
	ProductivityScore            ProductivityScore                 `json:"productivity_score"`
	ProductivityPerHourBreakdown ProductivityPerHourBreakdown      `json:"productivity_per_hour_breakdown"`
	LLMDailySummary              *LLMDailySummary                  `json:"llm_daily_summary"`
	TopDistractions              map[string]int                    `json:"top_distractions"`
	TopBlocked                   map[string]int                    `json:"top_blocked"`
	ProjectBreakdown             map[string]int                    `json:"project_breakdown"`
	CommunicationBreakdown       map[string]CommunicationBreakdown `json:"communication_breakdown"`
}

// type ProjectBreakdown struct {
// 	Name            string `json:"name"`
// 	DurationSeconds int    `json:"duration_seconds"`
// }

// type CommunicationBreakdown struct {
// 	ApplicationID   int64  `json:"application_id"`
// 	Channel         string `json:"channel"`
// 	DurationSeconds int    `json:"duration_seconds"`
// }

type ProductivityScore struct {
	ProductiveSeconds  int `json:"productive_seconds"`
	DistractingSeconds int `json:"distracting_seconds"`
	IdleSeconds        int `json:"idle_seconds"`
	OtherSeconds       int `json:"other_seconds"`
	ProductivityScore  int `json:"productivity_score"`
}

func (p *ProductivityScore) addSeconds(classification Classification, seconds int, isIdle bool) {
	if isIdle {
		p.IdleSeconds += seconds
		return
	}
	switch classification {
	case ClassificationProductive:
		p.ProductiveSeconds += seconds
	case ClassificationDistracting:
		p.DistractingSeconds += seconds
	default:
		p.OtherSeconds += seconds
	}
}

type ProductivityPerHourBreakdown map[int]ProductivityScore
