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

func (s *NativeService) CheckAutomation(appID string) bool {
	return true
}

func (s *NativeService) RequestAutomation(appID string) bool {
	return true
}

func (s *NativeService) OpenSettings() {
}

func (s *NativeService) StartObserver() {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}

	s.started = true
	s.mu.Unlock()

	go startObserver()
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

func (s *NativeService) GetInstalledBrowsers() []InstalledBrowser {
	return []InstalledBrowser{}
}
