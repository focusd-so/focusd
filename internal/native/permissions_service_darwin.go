//go:build darwin
// +build darwin

package native

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices

#include <ApplicationServices/ApplicationServices.h>

static Boolean checkAccessibility(Boolean prompt) {
	NSDictionary *options = @{(__bridge NSString *)kAXTrustedCheckOptionPrompt: @(prompt)};
	return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options);
}
*/
import "C"

// PermissionsService is a Wails-bound service for managing macOS permissions
// during the onboarding flow.
type PermissionsService struct{}

func NewPermissionsService() *PermissionsService {
	return &PermissionsService{}
}

// CheckAccessibility returns whether Accessibility permission is currently granted,
// without prompting the user.
func (s *PermissionsService) CheckAccessibility() bool {
	return C.checkAccessibility(C.Boolean(0)) != 0
}

// RequestAccessibility prompts the user for Accessibility permission via the
// macOS system dialog. Returns true if already granted or the user grants it.
func (s *PermissionsService) RequestAccessibility() bool {
	return C.checkAccessibility(C.Boolean(1)) != 0
}

// RequestAutomation triggers the macOS TCC prompt for a specific app bundle ID.
// Returns true if permission was granted (or was already granted).
func (s *PermissionsService) RequestAutomation(bundleID string) bool {
	return RequestAutomationPermission(bundleID)
}

// OpenSettings opens System Settings → Privacy & Security → Automation.
func (s *PermissionsService) OpenSettings() {
	OpenAutomationSettings()
}
