//go:build windows
// +build windows

package native

// NativeService is a Wails-bound service for managing OS permissions.
// On Windows, these are unimplemented stubs.
type NativeService struct{}

func NewNativeService() *NativeService {
	return &NativeService{}
}

func (s *NativeService) CheckAccessibility() bool {
	return true
}

func (s *NativeService) RequestAccessibility() bool {
	return true
}

func (s *NativeService) RequestAutomation(bundleID string) bool {
	return true
}

func (s *NativeService) OpenSettings() {
}

func (s *NativeService) StartObserver() {
	startObserver()
}
