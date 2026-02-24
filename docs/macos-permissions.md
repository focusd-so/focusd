# macOS Permissions Implementation Details

This document outlines the architecture for handling macOS Accessibility and Automation permissions within Focusd, specifically regarding the native observer in `darwin.go`.

## 1. The "Accessibility" Permission (One toggle rules them all)

The `darwin.go` implementation uses low-level C Mac APIs (`AXUIElement` and `AXObserver`) to monitor the focused window and title globally across the operating system.

Because this code runs under the **Accessibility** permission framework, the application only needs to ask the user to enable a single toggle in `System Settings -> Privacy & Security -> Accessibility`.

Once granted, Focusd has global permission to read the titles of **every single app on their computer**. It will _never_ prompt the user again when they switch to a new app (like Slack, Spotify, or Xcode).

## 2. The "Automation" Permission (Domain-specific access)

Focusd uses AppleScript (`osascript`) for three specific interactions. AppleScript relies on the **Automation** permission framework, which requests permission _per-app_. However, the requested applications are very specific:

- **Minimizing Apps:** The `minimiseApp` function executes the command: `tell application "System Events"`.
  - _Result:_ macOS asks for permission to control **System Events**, _not_ the target app being minimized. Once permission for "System Events" is granted, the AppleScript can minimize _any_ app via System Events without prompting the user again.
- **Reading Browser URLs:** The `getBrowserURLAndTitle` function specifically targets supported browsers (e.g., `tell application id "com.google.Chrome"`).
  - _Result:_ The user will only be prompted for permission for the specific browsers they use (Chrome, Safari, Brave, etc.).

## 3. Ideal Onboarding Flow

Because of this architecture, the onboarding flow only needs to trigger permissions for three domains to achieve full control:

1.  **Accessibility Permission:** For global active window title monitoring.
2.  **System Events Automation:** To minimize distracting apps or switch focus.
3.  **Browser Automation:** To extract specific URLs and apply blocks in supported browsers.

### Pre-authorizing during Onboarding

To prevent permission prompts from appearing unexpectedly during a focus session, the application can trigger these prompts intentionally during the onboarding flow.

This is done by executing a harmless AppleScript command against the target application. This explicitly forces the macOS TCC (Transparency, Consent, and Control) prompt to appear when the user is expecting it.

```go
// RequestAutomationPermission runs a harmless AppleScript to intentionally
// trigger the macOS TCC prompt right away during the onboarding UI.
func RequestAutomationPermission(bundleID string) bool {
	// A harmless command that forces macOS to ask for permission.
	// We ask for the 'version' property which exists on most macOS apps.
	script := fmt.Sprintf("tell application id \"%s\" to get version", bundleID)
	cmd := exec.Command("osascript", "-e", script)

	err := cmd.Run()

	// err == nil means the user clicked "OK" (or had already granted permission).
	// err != nil means they clicked "Don't Allow" (or the app doesn't exist).
	return err == nil
}
```

**Usage in Onboarding:**
When the user clicks "Connect Browsers" or "Enable App Blocking", the application should call `RequestAutomationPermission` for the relevant bundle IDs:

```go
// Trigger this when they connect browsers
RequestAutomationPermission("com.google.Chrome")
RequestAutomationPermission("com.apple.Safari")

// Trigger this when they enable system-wide app blocking
RequestAutomationPermission("com.apple.systemevents")
```

### 4. Customizing the Prompt Message

To increase the opt-in rate, the actual macOS permission prompt must explain why the permission is needed. This string is defined in the application's `Info.plist` file.

```xml
<key>NSAppleEventsUsageDescription</key>
<string>Focusd needs access to Safari and Chrome so it can read the active tab URL and block distracting websites during your focus sessions.</string>
```

### 5. Handling Denied Permissions

If `RequestAutomationPermission()` returns `false`, macOS will not prompt the user again for that specific app. The user must be guided to System Settings to manually enable the permission.

```go
// OpenAutomationSettings opens the macOS System Settings directly to the Automation page
func OpenAutomationSettings() {
	// Opens System Settings -> Privacy & Security -> Automation
	exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation").Run()
}
```
