// types_db.go contains the database models for the usage service
//
// The general rule is to use pointers for optional fields and non-pointers for mandatory fields
//
// This is to ensure consistency in the database schema and to avoid bugs related to nil vs empty
// string vs zero value. This will also make sure to properly think-through nil values in the code.

package usage

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
	ID   int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `json:"name" gorm:"uniqueIndex:idx_name_hostname_id;nullable"`

	// optional fields
	Icon     *string `json:"icon"` // either app icon or favicon if host is present
	Hostname *string `json:"hostname" gorm:"uniqueIndex:idx_name_hostname_id;nullable"`
	Domain   *string `json:"domain"`

	// darwin only
	BundleID *string `json:"bundle_id"`
}

type ApplicationUsage struct {
	// mandatory fields
	ID              int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	WindowTitle     string          `json:"window_title"`
	StartedAt       int64           `json:"started_at"`
	Classification  Classification  `gorm:"index:idx_classification" json:"classification"`
	TerminationMode TerminationMode `json:"termination_mode"`

	// optional fields
	BrowserURL      *string `json:"browser_url" gorm:"type:text;nullable"`
	EndedAt         *int64  `json:"ended_at" gorm:"nullable"`
	DurationSeconds *int    `json:"duration_seconds" gorm:"nullable"`

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

// Same returns true if the application usage is the same as the given application usage
//
// Application usage is considered the same if:
//   - The url is not nil and the url is the same
//   - The application name and title are the same,
//     eg. Slack + general discussion != Slack + funny memes
func (a *ApplicationUsage) Same(windowTitle, appName string, bundleID, url *string) bool {
	// Consistently handle nil vs empty string for URL comparison
	aURL := a.BrowserURL
	if aURL != nil && *aURL == "" {
		aURL = nil
	}
	newURL := url
	if newURL != nil && *newURL == "" {
		newURL = nil
	}

	if aURL != nil && newURL != nil && *aURL == *newURL {
		return true
	}

	// If one is nil and the other isn't, they are different (e.g. switching between tabs and native app)
	if (aURL == nil) != (newURL == nil) {
		return false
	}

	if appName == a.Application.Name && windowTitle == a.WindowTitle {
		return true
	}

	return false
}

type ApplicationUsageTags struct {
	ID      int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Tag     string `json:"tag"`
	UsageID int64  `json:"usage_id" gorm:"index:idx_usage_id"`
}

type ProtectionWhitelist struct {
	ID        int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	ExpiresAt int64 `json:"expires_at"`

	ExecutablePath string  `gorm:"uniqueIndex:idx_allow_usage_identity" json:"executable_path"`
	Hostname       *string `gorm:"uniqueIndex:idx_allow_usage_identity" json:"hostname"`
	URL            *string `gorm:"uniqueIndex:idx_allow_usage_identity" json:"url"`
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

type IdlePeriod struct {
	ID              int64  `gorm:"primaryKey;autoIncrement"`
	StartedAt       int64  `json:"started_at"`
	EndedAt         *int64 `json:"end_at" gorm:"index:idx_ended_at"`
	DurationSeconds *int   `json:"duration_seconds"`
	Reason          string `json:"reason"`
}

func (i *IdlePeriod) TableName() string {
	return "idle_period"
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
