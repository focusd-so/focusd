package usage

import (
	"time"

	"golang.org/x/net/publicsuffix"
)

// sandboxContext provides context for the current rule execution including usage data and helper functions
type sandboxContext struct {
	// Input data
	AppName string `json:"appName"`
	Title   string `json:"title"`

	Hostname       string `json:"hostname"`
	Path           string `json:"path"`
	Domain         string `json:"domain"`
	URL            string `json:"url"`
	Classification string `json:"classification"`

	// Helper pre-computed values
	MinutesSinceLastBlock     *int `json:"minutesSinceLastBlock"`
	MinutesUsedSinceLastBlock *int `json:"minutesUsedSinceLastBlock"`

	// Helper functions
	Now                 func(loc *time.Location) time.Time                                    `json:"-"`
	MinutesUsedInPeriod func(bundleID, hostname string, durationMinutes int64) (int64, error) `json:"-"`
}

type sandboxContextOption func(*sandboxContext)

func WithAppNameContext(appName string) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.AppName = appName
	}
}

func WithWindowTitleContext(title string) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.Title = title
	}
}

func WithBrowserURLContext(url string) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.URL = url

		hostname, path := parseURL(url)
		ctx.Hostname = hostname
		ctx.Path = path
		ctx.Domain, _ = publicsuffix.EffectiveTLDPlusOne(hostname)
	}
}

func WithNowContext(now time.Time) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.Now = func(loc *time.Location) time.Time {
			return now.In(loc)
		}
	}
}

func WithMinutesUsedInPeriodContext(minutesUsedInPeriod func(bundleID, hostname string, durationMinutes int64) (int64, error)) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.MinutesUsedInPeriod = minutesUsedInPeriod
	}
}

func WithMinutesSinceLastBlockContext(minutesSinceLastBlock int) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.MinutesSinceLastBlock = &minutesSinceLastBlock
	}
}

func WithMinutesUsedSinceLastBlockContext(minutesUsedSinceLastBlock int) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.MinutesUsedSinceLastBlock = &minutesUsedSinceLastBlock
	}
}

func NewSandboxContext(opts ...sandboxContextOption) sandboxContext {
	ctx := sandboxContext{}

	for _, opt := range opts {
		opt(&ctx)
	}

	return ctx
}
