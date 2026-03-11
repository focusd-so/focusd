package usage

type (
	TerminationMode       string
	TerminationModeSource string
	Classification        string
	ClassificationSource  string
)

const (
	TerminationModeNone   TerminationMode = "none"
	TerminationModeBlock  TerminationMode = "block"
	TerminationModePaused TerminationMode = "paused"
	TerminationModeAllow  TerminationMode = "allow"

	TerminationModeSourceApplication TerminationModeSource = "application"
	TerminationModeSourceCustomRules TerminationModeSource = "custom_rules"
	TerminationModeSourceWhitelist   TerminationModeSource = "whitelist"
	TerminationModeSourcePaused      TerminationModeSource = "paused"

	ClassificationSourceUserSet        ClassificationSource = "user_set"
	ClassificationSourceObviously      ClassificationSource = "obviously"
	ClassificationSourceCustomRules    ClassificationSource = "custom_rules"
	ClassificationSourceCloudLLMGemini ClassificationSource = "llm_gemini"

	IdleApplicationName = "Idle"

	ClassificationNone        Classification = "none"
	ClassificationProductive  Classification = "productive"
	ClassificationDistracting Classification = "distracting"
	ClassificationNeutral     Classification = "neutral"
	ClassificationSystem      Classification = "system"
)

func (c Classification) IsProductiveOrDistracting() bool {
	return c == ClassificationProductive || c == ClassificationDistracting
}

type TerminationDecision struct {
	Mode      TerminationMode
	Reasoning string
	Source    TerminationModeSource
}

type ClassificationResponse struct {
	Classification               Classification       `json:"classification"`
	ClassificationSource         ClassificationSource `json:"classification_source"`
	Reasoning                    string               `json:"reasoning"`
	ConfidenceScore              float32              `json:"confidence_score"`
	DetectedProject              string               `json:"detected_project"`
	DetectedCommunicationChannel string               `json:"detected_communication_channel"`
	Tags                         []string             `json:"tags"`

	SandboxContext  string  `json:"sandbox_context"`
	SandboxResponse *string `json:"sandbox_response"`
	SandboxLogs     string  `json:"sandbox_logs"`
}

type ClassifyRequest struct {
	AppName        string         `json:"app_name"`
	ExecutablePath string         `json:"executable_path"`
	Hostname       string         `json:"hostname"`
	URL            string         `json:"url"`
	Classification Classification `json:"classification"`
}

type ApplicationTagsSlice []ApplicationUsageTags

func (a ApplicationTagsSlice) Tags() []string {
	tags := make([]string, len(a))

	for i, tag := range a {
		tags[i] = tag.Tag
	}
	return tags
}
