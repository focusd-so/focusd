package usage

type Application struct {
	ID             int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name           string `json:"name"`
	ExecutablePath string `json:"executable_path"`
	Icon           string `json:"icon"` // either app icon or favicon

	Hostname *string `json:"hostname" gorm:"uniqueIndex:idx_bundle_id"`
	Domain   *string `json:"domain"`

	// darwin only
	BundleID *string `json:"bundle_id" gorm:"uniqueIndex:idx_bundle_id;nullable"`
}

type ApplicationUsage struct {
	ID          int64   `gorm:"primaryKey;autoIncrement" json:"id"`
	WindowTitle string  `json:"window_title"`
	BrowserURL  *string `json:"browser_url" gorm:"type:text;nullable"`

	StartedAt       int64  `json:"started_at"`
	EndedAt         *int64 `json:"ended_at" gorm:"nullable"`
	DurationSeconds *int   `json:"duration_seconds" gorm:"nullable"`

	Classification           Classification       `gorm:"index:idx_classification" json:"classification"`
	ClassificationReasoning  string               `json:"classification_reasoning"`
	ClassificationError      *string              `gorm:"index:idx_classification_error" json:"classification_error"`
	ClassificationConfidence float32              `json:"classification_confidence"`
	ClassificationSource     ClassificationSource `json:"classification_source"`

	DetectedProject              string `gorm:"index:idx_detected_project" json:"detected_project"`
	DetectedCommunicationChannel string `gorm:"index:idx_detected_communication_channel" json:"detected_communication_channel"`

	TerminationMode      TerminationMode       `json:"termination_mode"`
	TerminationReasoning string                `json:"termination_reasoning"`
	TerminationSource    TerminationModeSource `json:"termination_mode_source"`
	TerminationError     string                `gorm:"index:idx_termination_mode_error" json:"termination_mode_error"`

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
	if url != nil && a.BrowserURL != nil && *a.BrowserURL == *url {
		return true
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
	ID             int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	ExecutablePath string `gorm:"uniqueIndex:idx_allow_usage_identity" json:"executable_path"`
	Hostname       string `gorm:"uniqueIndex:idx_allow_usage_identity" json:"hostname"`
	URL            string `gorm:"uniqueIndex:idx_allow_usage_identity" json:"url"`
	ExpiresAt      int64  `json:"expires_at"` // nil = allow indefinitely, otherwise Unix timestamp
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
}
