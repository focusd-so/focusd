package native

import (
	"log/slog"
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// AxEventType represents the type of accessibility event
type AxEventType int

const (
	// AxEventTypeTitle is fired when the frontmost window's title changes
	AxEventTypeTitle AxEventType = 1
	// AxEventTypeIdle is fired when user idle state changes
	AxEventTypeIdle AxEventType = 2
)

// NativeEvent represents an accessibility event from the observer
type NativeEvent struct {
	Type           AxEventType
	PID            int
	ExecutablePath string
	AppName        string
	AppID          string
	Icon           string
	Title          string
	AppIcon        string // base64 encoded PNG
	URL            string
	AppCategory    string // LSApplicationCategoryType from Info.plist, e.g. "public.app-category.developer-tools"
}

func (e *NativeEvent) BrowserHostname() string {
	if e.URL == "" {
		return ""
	}

	u, err := url.Parse(e.URL)
	if err != nil {
		slog.Error("failed to parse URL", "error", err)

		return ""
	}

	hostname := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")

	return hostname
}

type InstalledBrowser struct {
	AppID string `json:"appID"`
	Name  string `json:"name"`
}

func (e *NativeEvent) Domain() string {
	if e.URL == "" {
		return ""
	}

	domain, _ := publicsuffix.PublicSuffix(e.BrowserHostname())

	return domain
}
