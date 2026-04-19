//go:build darwin
// +build darwin

package native

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices -framework ServiceManagement

#include <stdlib.h>
#include <ApplicationServices/ApplicationServices.h>

#import <Cocoa/Cocoa.h>
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

// getInstalledAppName returns the display name of the app with the given application id,
// or NULL if the app is not installed.
static const char* getInstalledAppName(const char* appID) {
	@autoreleasepool {
		NSString *bid = [NSString stringWithUTF8String:appID];
		NSURL *appURL = [[NSWorkspace sharedWorkspace] URLForApplicationWithBundleIdentifier:bid];
		if (appURL == nil) {
			return NULL;
		}
		NSBundle *bundle = [NSBundle bundleWithURL:appURL];
		NSString *name = [bundle objectForInfoDictionaryKey:@"CFBundleDisplayName"];
		if (name == nil) {
			name = [bundle objectForInfoDictionaryKey:@"CFBundleName"];
		}
		if (name == nil) {
			name = [[appURL lastPathComponent] stringByDeletingPathExtension];
		}
		return strdup([name UTF8String]);
	}
}
*/
import "C"
import (
	"fmt"
	"sort"
	"sync"
	"unsafe"
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

// CheckAutomation silently checks whether automation permission for a given
// application id is already granted, without triggering a TCC prompt.
func (s *NativeService) CheckAutomation(appID string) bool {
	return CheckAutomationPermission(appID)
}

// RequestAutomation triggers the macOS TCC prompt for a specific app application id.
// Returns true if permission was granted (or was already granted).
func (s *NativeService) RequestAutomation(appID string) bool {
	return RequestAutomationPermission(appID)
}

// OpenSettings opens System Settings → Privacy & Security → Automation.
func (s *NativeService) OpenSettings() {
	OpenAutomationSettings()
}

func (s *NativeService) StartObserver() {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}

	s.started = true
	s.mu.Unlock()

	// The observer enters a long-lived native run loop, so start it
	// asynchronously after releasing the service lock.
	go startObserver()
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

var browserPriority = map[string]int{
	"com.apple.Safari":           0,
	"com.google.Chrome":          1,
	"com.brave.Browser":          2,
	"com.microsoft.edgemac":      3,
	"com.operasoftware.Opera":    4,
	"com.vivaldi.Vivaldi":        5,
	"company.thebrowser.Browser": 6,
}

// GetInstalledBrowsers returns all known browsers that are installed on the system,
// sorted with popular browsers first.
func (s *NativeService) GetInstalledBrowsers() []InstalledBrowser {
	allAppIDs := make([]string, 0, len(chromeBaseAppIDs)+len(safariBasedAppIDs))
	allAppIDs = append(allAppIDs, safariBasedAppIDs...)
	allAppIDs = append(allAppIDs, chromeBaseAppIDs...)

	var installed []InstalledBrowser
	for _, bid := range allAppIDs {
		cBid := C.CString(bid)
		cName := C.getInstalledAppName(cBid)
		C.free(unsafe.Pointer(cBid))
		if cName == nil {
			continue
		}
		name := C.GoString(cName)
		C.free(unsafe.Pointer(cName))
		installed = append(installed, InstalledBrowser{AppID: bid, Name: name})
	}

	sort.SliceStable(installed, func(i, j int) bool {
		pi, okI := browserPriority[installed[i].AppID]
		pj, okJ := browserPriority[installed[j].AppID]
		if okI && okJ {
			return pi < pj
		}
		if okI {
			return true
		}
		if okJ {
			return false
		}
		return installed[i].Name < installed[j].Name
	})

	return installed
}
