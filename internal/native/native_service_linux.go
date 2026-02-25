//go:build linux
// +build linux

package native

// NativeService is a Wails-bound service for managing OS permissions.
// On Linux, these are unimplemented stubs.
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
