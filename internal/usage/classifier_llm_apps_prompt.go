package usage

var (
	instructionGenericApplicationClassification = mustLoadPrompt("app", "app_generic.txt")
	instructionSlackApplicationClassification   = mustLoadPrompt("app", "app_slack.txt")
)
