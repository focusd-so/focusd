package insights

// import (
// 	"time"

// 	"github.com/focusd-so/focusd/internal/usage"
// )

// type Application struct {
// 	ID       int64  `gorm:"primaryKey;autoIncrement" json:"id"`
// 	Name     string `json:"name"`
// 	BundleID string `json:"bundle_id" gorm:"uniqueIndex:idx_bundle_id"`
// 	Hostname string `json:"hostname" gorm:"uniqueIndex:idx_bundle_id"`
// 	Domain   string `json:"domain"`
// 	Icon     string `json:"icon"` // base64 encoded image
// }

// type ApplicationUsage struct {
// 	ID                           int64                       `gorm:"primaryKey;autoIncrement" json:"id"`
// 	WindowTitle                  string                      `json:"window_title"`
// 	BrowserURL                   string                      `json:"browser_url"`
// 	IsIdle                       bool                        `json:"is_idle"`
// 	StartedAt                    int64                       `json:"started_at"`
// 	EndedAt                      int64                       `json:"ended_at"`
// 	Duration                     int64                       `json:"duration"`
// 	Classification               usage.Classification        `gorm:"index:idx_classification" json:"classification"`
// 	ClassificationReasoning      string                      `json:"classification_reasoning"`
// 	ClassificationError          string                      `gorm:"index:idx_classification_error" json:"classification_error"`
// 	ClassificationConfidence     float32                     `json:"classification_confidence"`
// 	ClassificationSource         usage.ClassificationSource  `json:"classification_source"`
// 	DetectedProject              string                      `gorm:"index:idx_detected_project" json:"detected_project"`
// 	DetectedCommunicationChannel string                      `gorm:"index:idx_detected_communication_channel" json:"detected_communication_channel"`
// 	TerminationMode              usage.TerminationMode       `json:"termination_mode"`
// 	TerminationReasoning         string                      `json:"termination_reasoning"`
// 	TerminationModeSource        usage.TerminationModeSource `json:"termination_mode_source"`
// 	TerminationModeError         string                      `gorm:"index:idx_termination_mode_error" json:"termination_mode_error"`

// 	Tags          []ApplicationUsageTags `gorm:"foreignKey:UsageID" json:"tags"`
// 	ApplicationID int64                  `json:"application_id"`
// 	Application   Application            `gorm:"foreignKey:ApplicationID" json:"application"`
// }

// type ApplicationUsageTags struct {
// 	ID      int64  `gorm:"primaryKey;autoIncrement" json:"id"`
// 	Tag     string `json:"tag"`
// 	UsageID int64  `json:"usage_id" gorm:"index:idx_usage_id"`
// }

// // last 7 days of stats excluding today
// type UsageStats struct {
// 	ProductiveMinutes  int64   `json:"productive_minutes"`
// 	OtherMinutes       int64   `json:"supportive_minutes"`
// 	DistractiveMinutes int64   `json:"distractive_minutes"`
// 	ProductivityScore  float64 `json:"productivity_score"`
// }

// type DailyScore struct {
// 	Date  int64   `json:"date" gorm:"uniqueIndex:idx_date"`
// 	Score float64 `json:"score"`
// }

// type UsageOverview struct {
// 	ProductiveSeconds  int `gorm:"column:productive"`
// 	OtherSeconds       int `gorm:"column:other"`
// 	DistractiveSeconds int `gorm:"column:distracting"`
// 	ProductivityScore  int `gorm:"column:productivity_score"`
// }

// type UsagePerHourBreakdown struct {
// 	HourLabel          string `gorm:"column:hour_label"`
// 	ProductiveSeconds  int    `gorm:"column:productive"`
// 	OtherSeconds       int    `gorm:"column:other"`
// 	DistractiveSeconds int    `gorm:"column:distracting"`
// 	ProductivityScore  int    `gorm:"column:score"`
// }

// type DailyOverview struct {
// 	Date                  time.Time
// 	UsageOverview         UsageOverview
// 	UsagePerHourBreakdown []*UsagePerHourBreakdown
// 	DailyUsageSummary     DailyUsageSummary
// }

// type LLMDailySummaryRequest struct {
// 	Date                  time.Time
// 	UsageOverview         UsageOverview // Pre-calculated stats
// 	UsagePerHourBreakdown []*UsagePerHourBreakdown
// 	ContextSwitchCount    int                // You calculate this
// 	TopApps               []ApplicationUsage // Optional: top 5 apps
// }

// type LLMDailySummaryResponse struct {
// 	Headline   string   `json:"headline"`
// 	Summary    string   `json:"summary"`
// 	Wins       []string `json:"wins"`
// 	Suggestion string   `json:"suggestion"`
// 	DayVibe    string   `json:"day_vibe,omitempty"`
// }

// // DB model
// type DailyUsageSummary struct {
// 	ID            int64  `json:"id" gorm:"primaryKey;autoIncrement"`
// 	Date          string `json:"date" gorm:"uniqueIndex:idx_date"`
// 	Headline      string `json:"headline"`
// 	Summary       string `json:"summary"`
// 	Wins          string `json:"wins"`
// 	Suggestion    string `json:"suggestion"`
// 	DayVibe       string `json:"day_vibe"`
// 	DailyOverview string `json:"daily_overview"`
// }
