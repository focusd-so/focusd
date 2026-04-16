package usage

type CustomRulesTracePayload struct {
	Context string   `json:"context"`
	Logs    []string `json:"logs"`
	Output  string   `json:"resp"`
	Error   string   `json:"error"`
}
