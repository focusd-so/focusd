//go:build windows
// +build windows

package native

import "sync"

// NativeService is a Wails-bound service for managing OS permissions.
// On Windows, these are unimplemented stubs.
type NativeService struct {
	mu      sync.Mutex
	started bool
}

func NewNativeService() *NativeService {
	return &NativeService{}
}

func (s *NativeService) CheckAccessibility() bool {
	return true
}

func (s *NativeService) RequestAccessibility() bool {
	return true
}

func (s *NativeService) CheckAutomation(bundleID string) bool {
	return true
}

func (s *NativeService) RequestAutomation(bundleID string) bool {
	return true
}

func (s *NativeService) OpenSettings() {
}

func (s *NativeService) StartObserver() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return
	}

	s.started = true
	startObserver()
}

func (s *NativeService) EnableLoginItem() error {
	return nil
}

func (s *NativeService) DisableLoginItem() error {
	return nil
}

func (s *NativeService) LoginItemEnabled() bool {
	return false
}
