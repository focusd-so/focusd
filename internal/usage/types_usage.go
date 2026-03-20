package usage

type (
	TerminationMode       string
	TerminationModeSource string
	Classification        string
	ClassificationSource  string
)

const (
	TerminationModeNone  TerminationMode = "none"
	TerminationModeBlock TerminationMode = "block"
	TerminationModeAllow TerminationMode = "allow"

	TerminationModeSourceApplication TerminationModeSource = "application"
	TerminationModeSourceCustomRules TerminationModeSource = "custom_rules"
	TerminationModeSourceWhitelist   TerminationModeSource = "whitelist"
	TerminationModeSourcePaused      TerminationModeSource = "paused"

	ClassificationSourceUserSet           ClassificationSource = "user_set"
	ClassificationSourceObviously         ClassificationSource = "obviously"
	ClassificationSourceCustomRules       ClassificationSource = "custom_rules"
	ClassificationSourceCloudLLMGemini    ClassificationSource = "llm_gemini"
	ClassificationSourceCloudLLMOpenAI    ClassificationSource = "llm_openai"
	ClassificationSourceCloudLLMGroq      ClassificationSource = "llm_grok"
	ClassificationSourceCloudLLMAnthropic ClassificationSource = "llm_anthropic"

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
	Classification       Classification       `json:"classification"`
	ClassificationSource ClassificationSource `json:"classification_source"`
	Reasoning            string               `json:"reasoning"`
	ConfidenceScore      float32              `json:"confidence_score"`
	// DetectedProject is inferred by the LLM from the window title or channel name.
	// For coding apps (VS Code, Xcode, etc.), it extracts the workspace/project name from the title format.
	// For communication apps (Slack), it extracts the project/team context if strongly implied by the channel name.
	DetectedProject string `json:"detected_project"`

	// DetectedCommunicationChannel is inferred by the LLM from the window title for communication apps.
	// E.g., for Slack it extracts "engineering" from "Slack | #engineering | Acme Corp".
	// This is only populated when the "communication" tag is assigned.
	DetectedCommunicationChannel string `json:"detected_communication_channel"`

	Tags []string `json:"tags"`

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
