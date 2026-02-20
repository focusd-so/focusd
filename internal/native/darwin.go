//go:build darwin
// +build darwin

package native

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices -framework CoreGraphics

#include <Cocoa/Cocoa.h>
#include <ApplicationServices/ApplicationServices.h>

// Forward declaration of Go callback
extern void goOnTitleChange(int pid, char* bundleID, char* title, char* appName, char* executablePath, char* appIcon);

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

    const char* bundleIDStr = [app.bundleIdentifier UTF8String] ?: "unknown";
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

    goOnTitleChange((int)pid, (char*)bundleIDStr, (char*)titleStr, (char*)appNameStr, (char*)execPathStr, (char*)appIconStr);
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

func StartObserver() {
	fmt.Println("--- Accessibility Title Watcher ---")
	fmt.Println("Ensure this app has Accessibility permissions.")

	go func() {
		for {
			// CGEventSourceSecondsSinceLastEventType returns seconds since last input
			idle := float64(C.CGEventSourceSecondsSinceLastEventType(C.kCGEventSourceStateHIDSystemState, C.kCGAnyInputEventType))
			fmt.Printf("\rIdle: %.2f sec", idle)

			onIdleChange(idle)

			time.Sleep(1 * time.Second)
		}
	}()

	C.startWorkspaceWatcher()
	C.runLoop()
}

//export goOnTitleChange
func goOnTitleChange(cPID C.int, cBundleID *C.char, cTitle *C.char, cAppName *C.char, cExecutablePath *C.char, cAppIcon *C.char) {
	// Copy C strings to Go strings synchronously (C memory may be freed after return)
	pid := int(cPID)
	bundleID := C.GoString(cBundleID)
	title := C.GoString(cTitle)
	appName := C.GoString(cAppName)
	executablePath := C.GoString(cExecutablePath)
	appIcon := C.GoString(cAppIcon)

	go func() {
		// Resolve browser URL if applicable (uses osascript, may block)
		browserURL := getBrowserURL(bundleID)

		onTitleChange(NativeEvent{
			Type:           AxEventTypeTitle,
			PID:            pid,
			ExecutablePath: executablePath,
			AppName:        appName,
			BundleID:       bundleID,
			Icon:           appIcon,
			Title:          title,
			AppIcon:        appIcon,
			URL:            browserURL,
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

// package native

// /*
// #cgo CFLAGS: -fobjc-arc -isysroot /Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk
// #cgo LDFLAGS: -framework AppKit -framework ApplicationServices -framework Foundation
// #include <stdlib.h>
// #include "ax_observer.h"
// #include "overlay.h"

// extern void goAXEventCallback(AXEventData* data);

// static void cGoCallbackWrapper(AXEventData* data) {
//     goAXEventCallback(data);
// }

// static int startObserverWithGoCallback() {
//     return AXObserverStart(cGoCallbackWrapper);
// }
// */
// import "C"

// import (
// 	"context"
// 	"crypto/sha256"
// 	"encoding/base64"
// 	"encoding/hex"
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"log/slog"
// 	"net/url"
// 	"os/exec"
// 	"runtime"
// 	"slices"
// 	"strings"
// 	"sync"
// 	"time"
// 	"unsafe"

// 	"github.com/progrium/darwinkit/macos/appkit"
// )

// // Global callback storage
// var (
// 	titleCallback   func(AxEvent)
// 	idleCallback    func(AxEvent)
// 	axCallbackMu    sync.Mutex
// 	observerOnce    sync.Once
// 	observerStarted bool

// 	// Browser polling state
// 	pollingMu       sync.Mutex
// 	stopPolling     chan struct{}
// 	currentBundleID string
// 	lastTitle       string
// 	lastURL         string
// )

// //export goAXEventCallback
// func goAXEventCallback(data *C.AXEventData) {
// 	axCallbackMu.Lock()
// 	tFn := titleCallback
// 	iFn := idleCallback
// 	axCallbackMu.Unlock()

// 	// COPY DATA SYNCHRONOUSLY
// 	eventType := int(data.event_type)
// 	isIdle := data.is_idle == 1
// 	idleSeconds := float64(data.idle_seconds)
// 	pid := int(data.pid)
// 	appName := C.GoString(data.app_name)
// 	bundleID := C.GoString(data.bundle_id)
// 	initialTitle := C.GoString(data.title)
// 	appIcon := C.GoString(data.app_icon)

// 	// Process in a goroutine
// 	go func() {
// 		if eventType == C.AX_EVENT_TYPE_IDLE {
// 			if iFn != nil {
// 				iFn(AxEvent{
// 					Type:        AxEventTypeIdle,
// 					IsIdle:      isIdle,
// 					IdleSeconds: idleSeconds,
// 				})
// 			}
// 			return
// 		}

// 		if tFn == nil {
// 			return
// 		}

// 		title := initialTitle
// 		if (eventType == C.AX_EVENT_TYPE_APP || title == "") && pid > 0 {
// 			title = getProcessTitle(int32(pid))
// 		}

// 		url := getBrowserURL(bundleID)

// 		axEventType := AxEventTypeTitle
// 		if eventType == C.AX_EVENT_TYPE_APP {
// 			axEventType = AxEventTypeApp

// 			// Handle browser polling lifecycle
// 			manageBrowserPolling(pid, appName, bundleID, title, url, tFn)
// 		}

// 		tFn(AxEvent{
// 			Type:     axEventType,
// 			PID:      pid,
// 			AppName:  appName,
// 			BundleID: bundleID,
// 			Title:    title,
// 			URL:      url,
// 			AppIcon:  appIcon,
// 		})
// 	}()
// }

// func manageBrowserPolling(pid int, appName, bundleID, title, url string, fn func(AxEvent)) {
// 	pollingMu.Lock()
// 	defer pollingMu.Unlock()

// 	// If we are already polling this browser, just update the last known values
// 	if IsBrowser(bundleID) && bundleID == currentBundleID && stopPolling != nil {
// 		lastTitle = title
// 		lastURL = url
// 		return
// 	}

// 	// Stop previous polling if exists
// 	if stopPolling != nil {
// 		close(stopPolling)
// 		stopPolling = nil
// 	}

// 	// Start polling if it's a browser app
// 	if IsBrowser(bundleID) {
// 		currentBundleID = bundleID
// 		stopPolling = make(chan struct{})
// 		lastTitle = title
// 		lastURL = url
// 		stopChan := stopPolling

// 		go func() {
// 			ticker := time.NewTicker(1 * time.Second)
// 			defer ticker.Stop()

// 			for {
// 				select {
// 				case <-ticker.C:
// 					updatedTitle := getProcessTitle(int32(pid))
// 					updatedURL := getBrowserURL(bundleID)

// 					pollingMu.Lock()
// 					// Only notify if title or URL changed
// 					if updatedTitle != lastTitle || updatedURL != lastURL {
// 						lastTitle = updatedTitle
// 						lastURL = updatedURL
// 						pollingMu.Unlock()

// 						fn(AxEvent{
// 							Type:     AxEventTypeTitle,
// 							PID:      pid,
// 							AppName:  appName,
// 							BundleID: bundleID,
// 							Title:    updatedTitle,
// 							URL:      updatedURL,
// 						})

// 						slog.Debug("Browser app update - values changed",
// 							slog.String("app_name", appName),
// 							slog.String("title", updatedTitle),
// 							slog.String("url", updatedURL))
// 					} else {
// 						pollingMu.Unlock()
// 					}

// 				case <-stopChan:
// 					return
// 				}
// 			}
// 		}()

// 		slog.Info("Started polling for browser app", slog.String("app_name", appName), slog.String("bundle_id", bundleID))
// 	} else {
// 		currentBundleID = ""
// 		lastTitle = ""
// 		lastURL = ""
// 		slog.Info("Stopped polling - non-browser app active", slog.String("app_name", appName), slog.String("bundle_id", bundleID))
// 	}
// }

// func OnTitleChangeCallback(ctx context.Context, fn func(AxEvent)) {
// 	axCallbackMu.Lock()
// 	titleCallback = fn
// 	axCallbackMu.Unlock()
// 	startObserverOnce()
// }

// func OnIdleChangeCallback(ctx context.Context, fn func(AxEvent)) {
// 	axCallbackMu.Lock()
// 	idleCallback = fn
// 	axCallbackMu.Unlock()
// 	startObserverOnce()
// }

// func startObserverOnce() {
// 	observerOnce.Do(func() {
// 		go func() {
// 			runtime.LockOSThread()
// 			app := appkit.Application_SharedApplication()
// 			result := C.startObserverWithGoCallback()
// 			if result == 0 {
// 				log.Println("Failed to start AX observer - check accessibility permissions")
// 				return
// 			}
// 			observerStarted = true
// 			log.Println("Listening for active application, title and idle changes...")
// 			app.Run()
// 		}()
// 	})
// }

// func OnActiveAppTitleChange(ctx context.Context, fn func(event AxEvent)) {
// 	OnTitleChangeCallback(ctx, fn)
// 	// For backward compatibility, we'll also hook idle if they use this
// 	OnIdleChangeCallback(ctx, fn)

// 	// Block as before
// 	for {
// 		time.Sleep(1 * time.Second)
// 		if ctx.Err() != nil {
// 			return
// 		}
// 	}
// }

// func StopObserver() {
// 	C.AXObserverStop()

// 	axCallbackMu.Lock()
// 	titleCallback = nil
// 	idleCallback = nil
// 	axCallbackMu.Unlock()
// }

// func ShowWarningOverlay(appName, subtitle string, seconds int) {
// 	cAppName := C.CString(appName)
// 	defer C.free(unsafe.Pointer(cAppName))
// 	cSubtitle := C.CString(subtitle)
// 	defer C.free(unsafe.Pointer(cSubtitle))
// 	C.ShowOverlay(cAppName, cSubtitle, C.int(seconds))
// }

// func UpdateWarningOverlay(seconds int) {
// 	C.UpdateOverlay(C.int(seconds))
// }

// func HideWarningOverlay() {
// 	C.HideOverlay()
// }

// func CloseApp(ctx context.Context, app AxEvent) error {
// 	if IsBrowser(app.BundleID) {
// 		slog.Info("Closing browser tab", slog.String("app_name", app.AppName), slog.String("bundle_id", app.BundleID))
// 		return closeBrowserTab(app.BundleID)
// 	}

// 	slog.Info("Closing application", slog.String("app_name", app.AppName), slog.String("bundle_id", app.BundleID))
// 	return closeApplication(app.BundleID)
// }

// func closeBrowserTab(bundleID string) error {
// 	var script string

// 	if slices.Contains(chromeBaseBundleIDs, bundleID) {
// 		script = fmt.Sprintf(`tell app id "%s" to close active tab of front window`, bundleID)
// 	} else if slices.Contains(safariBasedBundleIDs, bundleID) {
// 		script = fmt.Sprintf(`tell app id "%s" to close current tab of front window`, bundleID)
// 	} else {
// 		return fmt.Errorf("unsupported browser: %s", bundleID)
// 	}

// 	cmd := exec.Command("osascript", "-e", script)
// 	output, err := cmd.CombinedOutput()
// 	if err != nil {
// 		return fmt.Errorf("failed to close browser tab: %v\nOutput: %s", err, output)
// 	}

// 	return nil
// }

// func closeApplication(bundleID string) error {
// 	script := fmt.Sprintf(`tell app id "%s" to quit`, bundleID)

// 	cmd := exec.Command("osascript", "-e", script)
// 	output, err := cmd.CombinedOutput()
// 	if err != nil {
// 		return fmt.Errorf("failed to close application: %v\nOutput: %s", err, output)
// 	}

// 	return nil
// }

// func TerminateApp(pid int) {
// 	// For backward compatibility, just log
// 	slog.Info("TERMINATING APP (DEPRECATED, use CloseApp)", "pid", pid)
// }

// func getProcessTitle(pid int32) string {
// 	osascript := fmt.Sprintf(`tell application "System Events" to get the title of front window of (first process whose unix id is %d)`, pid)
// 	cmd := exec.Command("osascript", "-e", osascript)

// 	output, err := cmd.CombinedOutput()
// 	if err != nil {
// 		slog.Error("Failed to get process title", slog.String("error", err.Error()), slog.Int("pid", int(pid)))
// 	}

// 	return strings.TrimSpace(string(output))
// }

func getBrowserURL(bundleID string) string {
	// Check if this is a supported browser first
	if !IsBrowser(bundleID) {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var script string

	if slices.Contains(safariBasedBundleIDs, bundleID) {
		script = fmt.Sprintf(`tell app id "%s" to get URL of front document`, bundleID)
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
                    return URL of active tab of w
                end if
            end if
        end try
    end repeat
    return ""
end tell`, bundleID)
	}

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		slog.Error("failed to get browser URL", "bundleID", bundleID, "error", err)
		return ""
	}

	return strings.TrimSpace(string(output))
}

// // Suppress unused import warning
// var _ = unsafe.Pointer(nil)

// IsBrowser checks if the given bundleID is a known browser
func IsBrowser(bundleID string) bool {
	return slices.Contains(chromeBaseBundleIDs, bundleID) || slices.Contains(safariBasedBundleIDs, bundleID) || bundleID == "com.apple.Safari"
}

func getBundleID(appName string) (string, error) {
	script := fmt.Sprintf(`tell application "%s" to get bundle identifier`, appName)
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get bundle ID: %w", err)
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

func BlockURL(targetURL, title, reason string, tags []string, appName string) error {

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

	cmd := exec.Command("osascript", "-e", appleScript)
	if err := cmd.Run(); err != nil {
		slog.Error("Failed to block URL", "appName", appName, "error", err)
	}

	return nil
}

func minimiseApp(bundleID string) {
	appleScript := fmt.Sprintf(`set bid to "%s"
tell application "System Events"
  tell (first process whose bundle identifier is bid)
    repeat with w in windows
      try
        set value of attribute "AXMinimized" of w to true
      end try
    end repeat
  end tell
end tell`, bundleID)

	cmd := exec.Command("osascript", "-e", appleScript)
	if err := cmd.Run(); err != nil {
		slog.Error("Failed to minimise app", "bundleID", bundleID, "error", err)
	}
}

func BlockApp(appName, title, reason string, tags []string) error {
	minimiseApp(appName)

	return nil
}

var chromeBaseBundleIDs = []string{
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

var safariBasedBundleIDs = []string{
	"com.apple.Safari",
	"com.apple.SafariTechnologyPreview",
}
