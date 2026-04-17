package usage

// PayloadShapes exists solely so the wails3 binding generator emits
// TypeScript types for the structs serialized into timeline.Event.Payload.
// It must not be called at runtime; the frontend ignores it.
func (s *Service) PayloadShapes() (
	ApplicationUsagePayload,
	AllowUsagePayload,
	PauseProtectionPayload,
	CustomRulesTracePayload,
) {
	return ApplicationUsagePayload{}, AllowUsagePayload{}, PauseProtectionPayload{}, CustomRulesTracePayload{}
}
