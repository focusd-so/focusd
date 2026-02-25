//go:build linux
// +build linux

package native

// PermissionsService is a Wails-bound service for managing OS permissions.
// On Linux, these are unimplemented stubs.
type PermissionsService struct{}

func NewPermissionsService() *PermissionsService {
	return &PermissionsService{}
}

func (s *PermissionsService) CheckAccessibility() bool {
	return true
}

func (s *PermissionsService) RequestAccessibility() bool {
	return true
}

func (s *PermissionsService) RequestAutomation(bundleID string) bool {
	return true
}

func (s *PermissionsService) OpenSettings() {
}
