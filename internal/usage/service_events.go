package usage

// OnUsageUpdated subscribes a callback to the usage updated event.
func (s *Service) OnUsageUpdated(fn func(usage ApplicationUsage)) {
	s.eventsMu.Lock()
	defer s.eventsMu.Unlock()
	s.onUsageUpdated = append(s.onUsageUpdated, fn)
}

// OnLLMDailySummaryReady subscribes a callback to the daily summary ready event.
func (s *Service) OnLLMDailySummaryReady(fn func(summary LLMDailySummary)) {
	s.eventsMu.Lock()
	defer s.eventsMu.Unlock()
	s.onLLMDailySummaryReady = append(s.onLLMDailySummaryReady, fn)
}

// OnProtectionPause subscribes a callback to the protection paused event.
func (s *Service) OnProtectionPause(fn func(pause ProtectionPause)) {
	s.eventsMu.Lock()
	defer s.eventsMu.Unlock()
	s.onProtectionPaused = append(s.onProtectionPaused, fn)
}

// OnProtectionResumed subscribes a callback to the protection resumed event.
func (s *Service) OnProtectionResumed(fn func(pause ProtectionPause)) {
	s.eventsMu.Lock()
	defer s.eventsMu.Unlock()
	s.onProtectionResumed = append(s.onProtectionResumed, fn)
}
