//go:build darwin
// +build darwin

package native

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices -framework CoreGraphics

#include <stdlib.h>
#include <Cocoa/Cocoa.h>
#include <ApplicationServices/ApplicationServices.h>

// checkAutomationPermission uses AEDeterminePermissionToAutomateTarget to
// silently check if automation permission for a application id is granted.
// Returns 1 if granted, 0 otherwise.
static int checkAutomationPermission(const char* appID) {
    NSString* bid = [NSString stringWithUTF8String:appID];

    NSAppleEventDescriptor* targetDesc = [NSAppleEventDescriptor descriptorWithBundleIdentifier:bid];

    OSStatus status = AEDeterminePermissionToAutomateTarget(
        targetDesc.aeDesc,
        typeWildCard,
        typeWildCard,
        false  // askUserIfNeeded = false -> don't prompt
    );

    return (status == noErr) ? 1 : 0;
}

// Forward declaration of Go callback
extern void goOnTitleChange(int pid, char* appID, char* title, char* appName, char* executablePath, char* appIcon, char* appCategory);

// Global state
static AXObserverRef gObserver = NULL;
static AXUIElementRef gAppElement = NULL;

// Forward declaration
static void hookFocusedWindow(BOOL emitTitle);

// Get window title from an AXUIElement
static NSString* getWindowTitle(AXUIElementRef windowElement) {
    if (!windowElement) return @"";

    CFTypeRef titleValue = NULL;
    if (AXUIElementCopyAttributeValue(windowElement, kAXTitleAttribute, &titleValue) == kAXErrorSuccess && titleValue) {
        NSString* title = (__bridge_transfer NSString*)titleValue;
        return title ?: @"";
    }
    return @"";
}

// Emit title change to Go
static void emitTitleChange(pid_t pid, NSString* title) {
    if (title.length == 0) return;  // Skip empty titles

    NSRunningApplication* app = [NSRunningApplication runningApplicationWithProcessIdentifier:pid];
    if (!app) return;

    NSString* appIDValue = @"unknown";
    if (app.bundleURL) {
        NSBundle* appBundle = [NSBundle bundleWithURL:app.bundleURL];
        if (appBundle) {
            NSString* id = [appBundle objectForInfoDictionaryKey:@"CFBundleIdentifier"];
            if (id) appIDValue = id;
        }
    }
    const char* appIDStr = [appIDValue UTF8String] ?: "unknown";
    const char* titleStr = [title UTF8String] ?: "";
    const char* appNameStr = [app.localizedName UTF8String] ?: "";
    const char* execPathStr = [app.executableURL.path UTF8String] ?: "";

    // Get app icon as base64 PNG
    NSString* appIconBase64 = @"";
    NSImage* icon = app.icon;
    if (icon) {
        NSBitmapImageRep* rep = [[NSBitmapImageRep alloc] initWithData:[icon TIFFRepresentation]];
        if (rep) {
            NSData* pngData = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
            if (pngData) {
                appIconBase64 = [pngData base64EncodedStringWithOptions:0];
            }
        }
    }
    const char* appIconStr = [appIconBase64 UTF8String] ?: "";

    // Get app category from Info.plist (LSApplicationCategoryType)
    NSString* appCategoryStr = @"";
    if (app.bundleURL) {
        NSBundle* appBundle = [NSBundle bundleWithURL:app.bundleURL];
        if (appBundle) {
            NSString* cat = [appBundle objectForInfoDictionaryKey:@"LSApplicationCategoryType"];
            if (cat) appCategoryStr = cat;
        }
    }
    const char* appCategoryC = [appCategoryStr UTF8String] ?: "";

    goOnTitleChange((int)pid, (char*)appIDStr, (char*)titleStr, (char*)appNameStr, (char*)execPathStr, (char*)appIconStr, (char*)appCategoryC);
}

// AXObserver callback
static void axObserverCallback(AXObserverRef observer, AXUIElementRef element, CFStringRef notification, void* refcon) {
    (void)observer;
    (void)refcon;

    @autoreleasepool {
        // Focused window changed within app - re-hook and emit title
        if (CFStringCompare(notification, kAXFocusedWindowChangedNotification, 0) == kCFCompareEqualTo) {
            hookFocusedWindow(YES);
            return;
        }

        // Window title changed
        if (CFStringCompare(notification, kAXTitleChangedNotification, 0) == kCFCompareEqualTo) {
            pid_t pid = 0;
            AXUIElementGetPid(element, &pid);
            NSString* title = getWindowTitle(element);
            emitTitleChange(pid, title);
            return;
        }

        // Window destroyed - re-hook to new focused window
        if (CFStringCompare(notification, kAXUIElementDestroyedNotification, 0) == kCFCompareEqualTo) {
            hookFocusedWindow(YES);
            return;
        }
    }
}

// Hook the focused window for title changes
static void hookFocusedWindow(BOOL emitTitle) {
    if (!gAppElement || !gObserver) return;

    CFTypeRef focusedWindow = NULL;
    if (AXUIElementCopyAttributeValue(gAppElement, kAXFocusedWindowAttribute, &focusedWindow) == kAXErrorSuccess && focusedWindow) {
        AXUIElementRef windowElement = (AXUIElementRef)focusedWindow;

        // Watch for title changes and window destruction
        AXObserverAddNotification(gObserver, windowElement, kAXTitleChangedNotification, NULL);
        AXObserverAddNotification(gObserver, windowElement, kAXUIElementDestroyedNotification, NULL);

        // Emit current title if requested
        if (emitTitle) {
            pid_t pid = 0;
            AXUIElementGetPid(windowElement, &pid);
            NSString* title = getWindowTitle(windowElement);
            emitTitleChange(pid, title);
        }

        CFRelease(windowElement);
    }
}

// Hook an application for title watching
static void hookApp(pid_t pid) {
    // Cleanup previous observer
    if (gObserver) {
        CFRunLoopRemoveSource(CFRunLoopGetCurrent(), AXObserverGetRunLoopSource(gObserver), kCFRunLoopDefaultMode);
        CFRelease(gObserver);
        gObserver = NULL;
    }
    if (gAppElement) {
        CFRelease(gAppElement);
        gAppElement = NULL;
    }

    gAppElement = AXUIElementCreateApplication(pid);

    AXObserverRef observer = NULL;
    if (AXObserverCreate(pid, axObserverCallback, &observer) != kAXErrorSuccess || !observer) {
        printf("--- CGO: Failed to create observer for PID %d ---\n", pid);
        return;
    }
    gObserver = observer;

    CFRunLoopAddSource(CFRunLoopGetCurrent(), AXObserverGetRunLoopSource(gObserver), kCFRunLoopDefaultMode);

    // Watch for focused window changes at app level
    AXObserverAddNotification(gObserver, gAppElement, kAXFocusedWindowChangedNotification, NULL);

    // Hook the current focused window (emit title on app switch)
    hookFocusedWindow(YES);

    printf("--- CGO: Watching PID %d ---\n", pid);
}

// Start workspace watcher for app switching
static void startWorkspaceWatcher() {
    // Hook current frontmost app
    NSRunningApplication* front = [[NSWorkspace sharedWorkspace] frontmostApplication];
    if (front) {
        hookApp(front.processIdentifier);
    }

    // Watch for app activation changes
    [[[NSWorkspace sharedWorkspace] notificationCenter]
        addObserverForName:NSWorkspaceDidActivateApplicationNotification
        object:nil
        queue:[NSOperationQueue mainQueue]
        usingBlock:^(NSNotification* note) {
            NSRunningApplication* app = [note.userInfo objectForKey:NSWorkspaceApplicationKey];
            if (app) {
                hookApp(app.processIdentifier);
            }
        }];
}

// Run the main loop (blocking)
static void runLoop() {
    CFRunLoopRun();
}
*/
import "C"

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os/exec"
	"slices"
	"strings"
	"time"
	"unsafe"
)

var (
	onTitleChange func(event NativeEvent)
	onIdleChange  func(idleSeconds float64)
)

func OnTitleChange(callback func(event NativeEvent)) {
	onTitleChange = callback
}

func OnIdleChange(callback func(idleSeconds float64)) {
	onIdleChange = callback
}

func startObserver() {
	fmt.Println("--- Accessibility Title Watcher ---")
	fmt.Println("Ensure this app has Accessibility permissions.")

	go func() {
		for {
			// CGEventSourceSecondsSinceLastEventType returns seconds since last input
			idle := float64(C.CGEventSourceSecondsSinceLastEventType(C.kCGEventSourceStateHIDSystemState, C.kCGAnyInputEventType))
			if idle > 120 {
				fmt.Printf("\rIdle: %.2f sec", idle)
			}

			onIdleChange(idle)

			time.Sleep(5 * time.Second)
		}
	}()

	C.startWorkspaceWatcher()
	C.runLoop()
}

//export goOnTitleChange
func goOnTitleChange(cPID C.int, cAppID *C.char, cTitle *C.char, cAppName *C.char, cExecutablePath *C.char, cAppIcon *C.char, cAppCategory *C.char) {
	// Copy C strings to Go strings synchronously (C memory may be freed after return)
	pid := int(cPID)
	appID := C.GoString(cAppID)
	title := C.GoString(cTitle)
	appName := C.GoString(cAppName)
	executablePath := C.GoString(cExecutablePath)
	appIcon := C.GoString(cAppIcon)
	appCategory := C.GoString(cAppCategory)

	go func() {
		var browserURL string

		// If it's a browser, resolve URL & precise title synchronously via AppleScript
		if IsBrowser(appID) {
			bURL, bTitle := getBrowserURLAndTitle(appID)
			browserURL = bURL

			// Override the accessibility API title with the browser's true active tab title.
			// This eliminates race conditions where the OS accessibility tree (which provides the 'title')
			// lags behind the browser's internal data model (which provides the 'URL'),
			// guaranteeing they perfectly match.
			if bTitle != "" && bURL != "" {
				title = bTitle
			}
		}

		onTitleChange(NativeEvent{
			Type:           AxEventTypeTitle,
			PID:            pid,
			ExecutablePath: executablePath,
			AppName:        appName,
			AppID:          appID,
			Icon:           appIcon,
			Title:          title,
			AppIcon:        appIcon,
			URL:            browserURL,
			AppCategory:    appCategory,
		})
	}()
}

func GetIdentity() (string, error) {
	uuid, err := getPlatformUUID()
	if err != nil {
		return "", err
	}
	serial, err := getPlatformSerial()
	if err != nil {
		return "", err
	}

	appSalt := hex.EncodeToString(sha256.New().Sum(nil))

	return fmt.Sprintf("%s:%s:%s", uuid, serial, appSalt), nil
}

func getBrowserURLAndTitle(appID string) (string, string) {
	// Check if this is a supported browser first
	if !IsBrowser(appID) {
		return "", ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var script string

	if slices.Contains(safariBasedAppIDs, appID) {
		script = fmt.Sprintf(`tell app id "%s"
	try
		set docURL to URL of front document
		set docTitle to name of front document
		return docURL & "|||" & docTitle
	on error
		return ""
	end try
end tell`, appID)
	} else {
		// For Chromium-based browsers, use a more robust script that filters out
		// automation/headless windows by checking visibility, mode, and dimensions
		script = fmt.Sprintf(`tell application id "%s"
    repeat with w in windows
        try
            set wVisible to visible of w
            set wMode to mode of w
            if wVisible is true and wMode is "normal" then
                set wBounds to bounds of w
                set wWidth to (item 3 of wBounds) - (item 1 of wBounds)
                set wHeight to (item 4 of wBounds) - (item 2 of wBounds)
                -- Filter out tiny windows (automation/headless often have small dimensions)
                if wWidth > 200 and wHeight > 200 then
                    set tabURL to URL of active tab of w
                    set tabTitle to title of active tab of w
                    return tabURL & "|||" & tabTitle
                end if
            end if
        end try
    end repeat
    return ""
end tell`, appID)
	}

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		slog.Error("failed to get browser URL and title", "appID", appID, "error", err)
		return "", ""
	}

	res := strings.TrimSpace(string(output))
	if res == "" {
		return "", ""
	}
	parts := strings.Split(res, "|||")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	// fallback if it just returned one thing or empty
	if len(parts) == 1 {
		return parts[0], ""
	}
	return "", ""
}

// // Suppress unused import warning
// var _ = unsafe.Pointer(nil)

// IsBrowser checks if the given appID is a known browser
func IsBrowser(appID string) bool {
	return slices.Contains(chromeBaseAppIDs, appID) || slices.Contains(safariBasedAppIDs, appID) || appID == "com.apple.Safari"
}

func getAppID(appName string) (string, error) {
	script := fmt.Sprintf(`tell application "%s" to get application id`, appName)
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get application id: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func getPlatformUUID() (string, error) {
	uuidCmd := "ioreg -d2 -c IOPlatformExpertDevice | awk -F\\\" '/IOPlatformUUID/{print $(NF-1)}'"
	uuidOut, err := exec.Command("bash", "-c", uuidCmd).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(uuidOut)), nil
}

func getPlatformSerial() (string, error) {
	serialCmd := "ioreg -d2 -c IOPlatformExpertDevice | awk -F\\\" '/IOPlatformSerialNumber/{print $(NF-1)}'"
	serialOut, err := exec.Command("bash", "-c", serialCmd).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(serialOut)), nil
}

func BlockURL(targetURL, title string, reason string, tags []string, appName string) error {

	data := struct {
		URL        string `json:"url"`
		Title      string `json:"title"`
		Descr      string `json:"descr"`
		Categories string `json:"categories"`
	}{
		URL:        targetURL,
		Title:      title,
		Descr:      reason,
		Categories: strings.Join(tags, ","),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Error("Failed to marshal blocked data", "error", err)
		return fmt.Errorf("failed to marshal blocked data: %w", err)
	}

	base64Data := base64.StdEncoding.EncodeToString(jsonData)
	encodedData := url.QueryEscape(base64Data)

	appleScript := fmt.Sprintf(`tell application "%s"
    repeat with w in every window
        repeat with t in every tab of w
            if the URL of t is "%s" then
                set URL of t to "https://focusd.so/blocked?d=%s"
            end if
        end repeat
    end repeat
end tell`, appName, targetURL, encodedData)

	slog.Info("block url", "script", appleScript)

	cmd := exec.Command("osascript", "-e", appleScript)
	if err := cmd.Run(); err != nil {
		slog.Error("permissionsservice", "appName", appName, "error", err)
	}

	return nil
}

func minimiseApp(appName string) {
	appleScript := fmt.Sprintf(`set appName to "%s"
tell application "System Events"
  tell (first process whose name is appName)
    repeat with w in windows
      try
        set value of attribute "AXMinimized" of w to true
      end try
    end repeat
  end tell
end tell`, appName)

	cmd := exec.Command("osascript", "-e", appleScript)
	if err := cmd.Run(); err != nil {
		slog.Error("Failed to minimise app", "appName", appName, "error", err)
	}
}

func BlockApp(appName, title, reason string, tags []string) error {
	minimiseApp(appName)

	return nil
}

// CheckAutomationPermission checks whether automation permission for a given
// application id is already granted, without triggering a TCC prompt.
// Returns true if granted, false if denied or not yet asked.
func CheckAutomationPermission(appID string) bool {
	cAppID := C.CString(appID)
	defer C.free(unsafe.Pointer(cAppID))
	return C.checkAutomationPermission(cAppID) != 0
}

// RequestAutomationPermission runs a harmless AppleScript to intentionally
// trigger the macOS TCC prompt right away during your onboarding UI.
func RequestAutomationPermission(appID string) bool {
	// We ask for the 'version' property which exists on almost all macOS apps.
	script := fmt.Sprintf(`tell application id "%s" to get version`, appID)
	cmd := exec.Command("osascript", "-e", script)

	err := cmd.Run()

	// err == nil means the user clicked "OK" (or had already granted permission).
	// err != nil means they clicked "Don't Allow" (or the app doesn't exist).
	return err == nil
}

// OpenAutomationSettings opens the macOS System Settings directly to the Automation page
func OpenAutomationSettings() {
	// Opens System Settings -> Privacy & Security -> Automation
	cmd := exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation")
	err := cmd.Run()
	if err != nil {
		slog.Error("Failed to open automation settings", "error", err)
	}
}

var chromeBaseAppIDs = []string{
	"com.google.Chrome",
	"com.google.Chrome.beta",
	"com.google.Chrome.dev",
	"com.google.Chrome.canary",
	"com.brave.Browser",
	"com.brave.Browser.beta",
	"com.brave.Browser.nightly",
	"com.microsoft.edgemac",
	"com.microsoft.edgemac.Beta",
	"com.microsoft.edgemac.Dev",
	"com.microsoft.edgemac.Canary",
	"com.mighty.app",
	"com.ghostbrowser.gb1",
	"com.bookry.wavebox",
	"com.pushplaylabs.sidekick",
	"com.operasoftware.Opera",
	"com.operasoftware.OperaNext",
	"com.operasoftware.OperaDeveloper",
	"com.operasoftware.OperaGX",
	"com.vivaldi.Vivaldi",
	"company.thebrowser.Browser",
}

var safariBasedAppIDs = []string{
	"com.apple.Safari",
	"com.apple.SafariTechnologyPreview",
}
