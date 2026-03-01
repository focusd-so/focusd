package nativemessaging

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	hostName        = "app.focusd.so"
	hostDescription = "Focusd Native Messaging Host"
)

var (
	defaultChromeExtensionIDs  = []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	defaultFirefoxExtensionIDs = []string{"focusd@focusd.so"}
)

type hostManifest struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Path              string   `json:"path"`
	Type              string   `json:"type"`
	AllowedOrigins    []string `json:"allowed_origins,omitempty"`
	AllowedExtensions []string `json:"allowed_extensions,omitempty"`
}

// EnsureHostManifests creates or updates Chrome and Firefox native messaging
// host manifests for the current user.
func EnsureHostManifests() error {
	if runtime.GOOS != "darwin" {
		return nil
	}

	if os.Getenv("FOCUSD_DISABLE_NATIVE_MESSAGING_MANIFESTS") == "1" {
		return nil
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		// Keep the original path if symlink resolution fails.
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve user home directory: %w", err)
	}

	chromeIDs := readListEnv("FOCUSD_CHROME_EXTENSION_IDS", defaultChromeExtensionIDs)
	firefoxIDs := readListEnv("FOCUSD_FIREFOX_EXTENSION_IDS", defaultFirefoxExtensionIDs)

	chromeManifest := hostManifest{
		Name:           hostName,
		Description:    hostDescription,
		Path:           execPath,
		Type:           "stdio",
		AllowedOrigins: toChromeOrigins(chromeIDs),
	}
	firefoxManifest := hostManifest{
		Name:              hostName,
		Description:       hostDescription,
		Path:              execPath,
		Type:              "stdio",
		AllowedExtensions: firefoxIDs,
	}

	chromePath := filepath.Join(
		home,
		"Library",
		"Application Support",
		"Google",
		"Chrome",
		"NativeMessagingHosts",
		hostName+".json",
	)
	firefoxPath := filepath.Join(
		home,
		"Library",
		"Application Support",
		"Mozilla",
		"NativeMessagingHosts",
		hostName+".json",
	)

	slog.Info("installing native messaging host manifests", "chromePath", chromePath, "firefoxPath", firefoxPath)

	if err := writeManifest(chromePath, chromeManifest); err != nil {
		return fmt.Errorf("write chrome native messaging manifest: %w", err)
	}
	if err := writeManifest(firefoxPath, firefoxManifest); err != nil {
		return fmt.Errorf("write firefox native messaging manifest: %w", err)
	}

	return nil
}

func readListEnv(key string, fallback []string) []string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		out := make([]string, 0, len(fallback))
		out = append(out, fallback...)
		return out
	}

	parts := strings.Split(val, ",")
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		item := strings.TrimSpace(p)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}

	if len(out) == 0 {
		out = append(out, fallback...)
	}

	return out
}

func toChromeOrigins(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, "chrome-extension://"+id+"/")
	}
	return out
}

func writeManifest(path string, manifest hostManifest) error {
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	content = append(content, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create manifest directory: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return fmt.Errorf("write temp manifest: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("move temp manifest into place: %w", err)
	}

	return nil
}
