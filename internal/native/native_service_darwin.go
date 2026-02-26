//go:build darwin
// +build darwin

package native

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices -framework ServiceManagement

#include <ApplicationServices/ApplicationServices.h>

#import <ServiceManagement/ServiceManagement.h>

static Boolean checkAccessibility(Boolean prompt) {
	NSDictionary *options = @{(__bridge NSString *)kAXTrustedCheckOptionPrompt: @(prompt)};
	return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options);
}

// enableLoginItem registers the current app as a macOS Login Item via SMAppService (macOS 13+).
// Returns 0 on success, 1 on failure.
static int enableLoginItem(void) {
	if (@available(macOS 13.0, *)) {
		NSError *error = nil;
		BOOL success = [[SMAppService mainAppService] registerAndReturnError:&error];
		if (!success) {
			NSLog(@"[focusd] Failed to register login item: %@", error);
			return 1;
		}
		return 0;
	}
	return 1;
}

// disableLoginItem unregisters the current app as a macOS Login Item.
// Returns 0 on success, 1 on failure.
static int disableLoginItem(void) {
	if (@available(macOS 13.0, *)) {
		NSError *error = nil;
		BOOL success = [[SMAppService mainAppService] unregisterAndReturnError:&error];
		if (!success) {
			NSLog(@"[focusd] Failed to unregister login item: %@", error);
			return 1;
		}
		return 0;
	}
	return 1;
}

// loginItemEnabled checks whether the app is currently registered as a Login Item.
// Returns 1 if enabled, 0 otherwise.
static int loginItemEnabled(void) {
	if (@available(macOS 13.0, *)) {
		SMAppServiceStatus status = [[SMAppService mainAppService] status];
		return (status == SMAppServiceStatusEnabled) ? 1 : 0;
	}
	return 0;
}
*/
import "C"
import (
	"fmt"
	"sync"
)

// NativeService is a Wails-bound service for managing macOS permissions
// during the onboarding flow.
type NativeService struct {
	mu      sync.Mutex
	started bool
}

func NewNativeService() *NativeService {
	return &NativeService{}
}

// CheckAccessibility returns whether Accessibility permission is currently granted,
// without prompting the user.
func (s *NativeService) CheckAccessibility() bool {
	return C.checkAccessibility(C.Boolean(0)) != 0
}

// RequestAccessibility prompts the user for Accessibility permission via the
// macOS system dialog. Returns true if already granted or the user grants it.
func (s *NativeService) RequestAccessibility() bool {
	return C.checkAccessibility(C.Boolean(1)) != 0
}

// RequestAutomation triggers the macOS TCC prompt for a specific app bundle ID.
// Returns true if permission was granted (or was already granted).
func (s *NativeService) RequestAutomation(bundleID string) bool {
	return RequestAutomationPermission(bundleID)
}

// OpenSettings opens System Settings → Privacy & Security → Automation.
func (s *NativeService) OpenSettings() {
	OpenAutomationSettings()
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

// EnableLoginItem registers the app to open at login.
func (s *NativeService) EnableLoginItem() error {
	if C.enableLoginItem() != 0 {
		return fmt.Errorf("failed to register login item")
	}
	return nil
}

// DisableLoginItem unregisters the app from opening at login.
func (s *NativeService) DisableLoginItem() error {
	if C.disableLoginItem() != 0 {
		return fmt.Errorf("failed to unregister login item")
	}
	return nil
}

// LoginItemEnabled returns whether the app is currently registered to open at login.
func (s *NativeService) LoginItemEnabled() bool {
	return C.loginItemEnabled() != 0
}
