// types_db.go contains the database models for the usage service
//
// The general rule is to use pointers for optional fields and non-pointers for mandatory fields
//
// This is to ensure consistency in the database schema and to avoid bugs related to nil vs empty
// string vs zero value. This will also make sure to properly think-through nil values in the code.

package usage

import (
	"net/url"
	"time"
)

type (
	ExecutionLogType string
)

const (
	ExecutionLogTypeClassification  ExecutionLogType = "classification"
	ExecutionLogTypeTerminationMode ExecutionLogType = "termination_mode"
)

// Application represents a unique application that has been used by the user
// Application is unique by name and hostname that is
//   - if the application is not a browser it is unique by name
//   - if the application is a browser it is unique by name and domain
//     eg. Chrome + google.com != Chrome + youtube.com, each of them will have its own application
type Application struct {
	// mandatory fields
	ID   int64  `json:"id" gorm:"primaryKey;autoIncrement;not null"`
	Name string `json:"name" gorm:"uniqueIndex:idx_name_hostname_id;not null"`

	// optional fields
	Icon           *string `json:"icon"` // either app icon or favicon if host is present
	Hostname       *string `json:"hostname" gorm:"uniqueIndex:idx_name_hostname_id"`
	Domain         *string `json:"domain"`
	ExecutablePath string  `json:"executable_path"`

	// darwin only
	BundleID    *string `json:"bundle_id"`
	AppCategory *string `json:"app_category"` // LSApplicationCategoryType, e.g. "public.app-category.developer-tools"
}

func (a Application) TableName() string {
	return "application"
}

func (a Application) NewUsage(windowTitle string, browserURL *string) ApplicationUsage {
	return ApplicationUsage{
		ApplicationID:   a.ID,
		Application:     a,
		StartedAt:       time.Now().Unix(),
		Classification:  ClassificationNone,
		TerminationMode: TerminationModeNone,
		WindowTitle:     windowTitle,
		BrowserURL:      browserURL,
	}
}

func NewIdleApplication() Application {
	return Application{
		Name:           IdleApplicationName,
		ExecutablePath: "com.system.idle",
	}
}

type ApplicationUsage struct {
	// mandatory fields
	ID              int64           `json:"id" gorm:"primaryKey;autoIncrement;not null"`
	WindowTitle     string          `json:"window_title" gorm:"not null"`
	StartedAt       int64           `json:"started_at" gorm:"not null"`
	Classification  Classification  `json:"classification" gorm:"index:idx_classification"`
	TerminationMode TerminationMode `json:"termination_mode" gorm:"not null"`

	// optional fields
	BrowserURL      *string `json:"browser_url" gorm:"type:text"`
	EndedAt         *int64  `json:"ended_at"`
	DurationSeconds *int    `json:"duration_seconds"`

	ClassificationError      *string               `gorm:"index:idx_classification_error" json:"classification_error"`
	ClassificationConfidence *float32              `json:"classification_confidence"`
	ClassificationReasoning  *string               `json:"classification_reasoning"`
	ClassificationSource     *ClassificationSource `json:"classification_source"`

	DetectedProject              *string `gorm:"index:idx_detected_project" json:"detected_project"`
	DetectedCommunicationChannel *string `gorm:"index:idx_detected_communication_channel" json:"detected_communication_channel"`

	TerminationReasoning *string                `json:"termination_reasoning"`
	TerminationSource    *TerminationModeSource `json:"termination_mode_source"`
	TerminationError     *string                `gorm:"index:idx_termination_mode_error" json:"termination_mode_error"`

	SandboxContext  *string `json:"sandbox_context" gorm:"type:text;nullable"`
	SandboxResponse *string `json:"sandbox_response" gorm:"type:text;nullable"`
	SandboxLogs     *string `json:"sandbox_logs" gorm:"type:text;nullable"`

	// relations
	Tags          []ApplicationUsageTags `gorm:"foreignKey:UsageID" json:"tags"`
	ApplicationID int64                  `json:"application_id"`
	Application   Application            `gorm:"foreignKey:ApplicationID" json:"application"`
}

func (a *ApplicationUsage) TableName() string {
	return "application_usage"
}

func (a *ApplicationUsage) IsCommunicationUsage() bool {
	if fromPtr(a.DetectedCommunicationChannel) != "" {
		return true
	}

	for _, tag := range a.Tags {
		if tag.Tag == "communication" {
			return true
		}
	}

	return false
}

func (a *ApplicationUsage) CommunicationChannel() string {
	return fromPtr(a.DetectedCommunicationChannel)
}

func (a *ApplicationUsage) HasDetectedProject() bool {
	return a.GetDetectedProject() != ""
}

func (a *ApplicationUsage) GetDetectedProject() string {
	return fromPtr(a.DetectedProject)
}

// Same returns true if the application usage is the same as the given application usage
func (a ApplicationUsage) Same(a1 ApplicationUsage) bool {
	if a.ID != 0 && a1.ID != 0 && a.ID == a1.ID {
		return true
	}

	return a.String() == a1.String()
}

func (a *ApplicationUsage) String() string {
	if a.Application.Name == IdleApplicationName {
		return IdleApplicationName
	}

	vals := url.Values{}
	vals.Set("app", a.Application.Name)
	vals.Set("title", a.WindowTitle)
	vals.Set("url", fromPtr(a.BrowserURL))

	return vals.Encode()
}

type ApplicationUsageTags struct {
	ID      int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Tag     string `json:"tag"`
	UsageID int64  `json:"usage_id" gorm:"index:idx_usage_id"`
}

func (a *ApplicationUsageTags) TableName() string {
	return "application_usage_tag"
}

type ProtectionWhitelist struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	// ExpiresAt should be pre-calculated and set to the time when the whitelist expires
	ExpiresAt int64 `json:"expires_at"`

	AppName  string  `gorm:"uniqueIndex:idx_allow_usage_identity" json:"appname"`
	Hostname *string `gorm:"uniqueIndex:idx_allow_usage_identity" json:"hostname"`
}

func (p *ProtectionWhitelist) TableName() string {
	return "protection_whitelist"
}

type ProtectionPause struct {
	ID                       int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	RequestedDurationSeconds int    `json:"requested_duration_seconds"`
	ActualDurationSeconds    int    `json:"actual_duration_seconds"`
	ResumedAt                int64  `json:"resumed_at"`
	ResumedReason            string `json:"resumed_reason"`
	CreatedAt                int64  `json:"created_at"`
	Reason                   string `json:"reason"`
}

func (p *ProtectionPause) TableName() string {
	return "protection_pause"
}

type SandboxExecutionLog struct {
	ID         int64   `gorm:"primaryKey;autoIncrement" json:"id"`
	Context    string  `json:"context" gorm:"type:text;nullable"`
	Response   *string `json:"response" gorm:"type:text;nullable"`
	Logs       string  `json:"logs" gorm:"type:text;nullable"`
	CreatedAt  int64   `json:"created_at" gorm:"index:idx_created_at"`
	FinishedAt *int64  `json:"finished_at" gorm:"index:idx_finished_at;nullable"`
	Error      *string `json:"error" gorm:"type:text;nullable"`
	Type       string  `json:"type" gorm:"index:idx_type"`
}

func (s *SandboxExecutionLog) TableName() string {
	return "sandbox_execution_log"
}
