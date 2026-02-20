package classification

import (
	"context"
	"strings"

	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
)

// codeEditorBundlePatterns contains bundle ID patterns for known code editors and IDEs.
var codeEditorBundlePatterns = []string{
	"com.microsoft.vscode",
	"com.microsoft.vscodium",
	"com.todesktop.230313mzl4w4u92", // Cursor
	"com.cursor",
	"com.jetbrains",
	"com.sublimetext",
	"com.sublimehq",
	"com.apple.dt.xcode",
	"com.github.atom",
	"io.zed",
	"dev.zed",
	"com.panic.nova",
	"com.barebones.bbedit",
	"com.barebones.textwrangler",
	"abnerworks.typora",
	"com.coteditor.coteditor",
}

// isCodeEditor checks if a bundle ID matches known code editor patterns.
func isCodeEditor(bundleID string) bool {
	bundleID = strings.ToLower(bundleID)
	for _, pattern := range codeEditorBundlePatterns {
		if strings.Contains(bundleID, pattern) {
			return true
		}
	}
	return false
}

const codeEditorInstructions = `
You are a Code Editor Focus Classifier. Your job is to analyze code editor and IDE window titles to extract project information and classify the activity.

Input: Bundle ID, Window Title.
Output: A single, raw JSON object (no markdown, no explanations).

---

## JSON Schema
Return exactly these keys:
1. "classification": "productive" (always productive for code editors)
2. "reasoning": Brief explanation.
3. "tags": Array containing "work" and "code-editor", plus any additional relevant tags.
4. "detected_project": (string | null) - The inferred project/workspace name from the window title.
5. "detected_communication_channel": null (not applicable for code editors)
6. "confidence_score": Float (0.0 - 1.0)

Allowed additional tags:
["research", "learning", "other"]

---

## Project Detection Rules

Infer the project name from common window title patterns used by code editors.

### Common patterns to detect:
- "project-name — file.ext"
- "project-name - file.ext"  
- "file.ext — project-name"
- "file.ext - project-name"
- "project-name"
- "folder-name (Workspace)"
- "folder-name [SSH]"
- "folder-name — Visual Studio Code"
- "folder-name — Cursor"
- "[project-name]" (JetBrains style)

### Heuristics:
- Prefer **project/folder/workspace name** over file name
- Strip file extensions from the project name
- Ignore editor branding ("Visual Studio Code", "IntelliJ IDEA", "Cursor", "GoLand", etc.)
- Ignore temporary labels like "•", "*", "modified", "Edited"
- Ignore common non-project indicators like "Welcome", "Settings", "Extensions"
- If multiple candidates exist, choose the most stable workspace-level name
- If no reliable project name is found, return null

### Terminal-specific patterns:
- "zsh — projectname" or "bash — projectname"
- "user@host: ~/projects/projectname"
- Look for directory paths and extract the project folder name

---

## Examples

### Example 1 — VS Code with project
**Input**
- Bundle ID: "com.microsoft.VSCode"
- Window Title: "focusd-backend — main.go"

**Output**
{
  "classification": "productive",
  "reasoning": "Actively editing backend source code.",
  "tags": ["work", "code-editor"],
  "detected_project": "focusd-backend",
  "detected_communication_channel": null,
  "confidence_score": 0.95
}

### Example 2 — Cursor with project
**Input**
- Bundle ID: "com.todesktop.230313mzl4w4u92"
- Window Title: "auth-service - handler.go — Cursor"

**Output**
{
  "classification": "productive",
  "reasoning": "Backend service development in Cursor IDE.",
  "tags": ["work", "code-editor"],
  "detected_project": "auth-service",
  "detected_communication_channel": null,
  "confidence_score": 0.90
}

### Example 3 — JetBrains GoLand
**Input**
- Bundle ID: "com.jetbrains.goland"
- Window Title: "[api-gateway] – main.go"

**Output**
{
  "classification": "productive",
  "reasoning": "Go development work in JetBrains IDE.",
  "tags": ["work", "code-editor"],
  "detected_project": "api-gateway",
  "detected_communication_channel": null,
  "confidence_score": 0.92
}

### Example 4 — VS Code Welcome screen
**Input**
- Bundle ID: "com.microsoft.VSCode"
- Window Title: "Welcome — Visual Studio Code"

**Output**
{
  "classification": "productive",
  "reasoning": "Code editor open but no active project.",
  "tags": ["work", "code-editor"],
  "detected_project": null,
  "detected_communication_channel": null,
  "confidence_score": 1.0
}

### Example 5 — Terminal with project path
**Input**
- Bundle ID: "com.apple.Terminal"
- Window Title: "zsh — ~/dev/focusd-wails"

**Output**
{
  "classification": "productive",
  "reasoning": "Terminal session in project directory.",
  "tags": ["work", "code-editor"],
  "detected_project": "focusd-wails",
  "detected_communication_channel": null,
  "confidence_score": 0.85
}

### Example 6 — iTerm2 running tests
**Input**
- Bundle ID: "com.googlecode.iterm2"
- Window Title: "npm test — auth-service"

**Output**
{
  "classification": "productive",
  "reasoning": "Running tests in terminal for project.",
  "tags": ["work", "code-editor"],
  "detected_project": "auth-service",
  "detected_communication_channel": null,
  "confidence_score": 0.88
}

### Example 7 — Xcode
**Input**
- Bundle ID: "com.apple.dt.Xcode"
- Window Title: "MyiOSApp — ContentView.swift"

**Output**
{
  "classification": "productive",
  "reasoning": "iOS development in Xcode.",
  "tags": ["work", "code-editor"],
  "detected_project": "MyiOSApp",
  "detected_communication_channel": null,
  "confidence_score": 0.95
}

---

REMINDER: Output must be a valid JSON object with no markdown fences and no explanations.
`

func (c *Classification) ClassifyCodeEditor(ctx context.Context, req *apiv1.LLMClassifyRequest) (*apiv1.LLMClassifyResponse, error) {
	return c.classifyAppWithInstructions(ctx, codeEditorInstructions, req)
}
