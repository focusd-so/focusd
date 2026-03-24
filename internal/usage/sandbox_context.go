package usage

import (
	"time"

	"golang.org/x/net/publicsuffix"
)

type sandboxUsageMetadata struct {
	AppName        string `json:"appName"`
	Title          string `json:"title"`
	Hostname       string `json:"hostname"`
	Path           string `json:"path"`
	Domain         string `json:"domain"`
	URL            string `json:"url"`
	Classification string `json:"classification"`
}

type sandboxUsageInsights struct {
	DistractingMinutes         int `json:"distractingMinutes"`
	BlockedCount               int `json:"blockedCount"`
	MinutesSinceLastBlock      int `json:"minutesSinceLastBlock"`
	MinutesUsedSinceLastBlock  int `json:"minutesUsedSinceLastBlock"`
	LastBlockedDurationMinutes int `json:"lastBlockedDurationMinutes"`
}

type sandboxUsageContext struct {
	Metadata sandboxUsageMetadata `json:"metadata"`
	Insights sandboxUsageInsights `json:"insights"`
}

type sandboxInsightsToday struct {
	ProductiveMinutes  int `json:"productiveMinutes"`
	DistractingMinutes int `json:"distractingMinutes"`
	IdleMinutes        int `json:"idleMinutes"`
	OtherMinutes       int `json:"otherMinutes"`
	FocusScore         int `json:"focusScore"`
	DistractionCount   int `json:"distractionCount"`
	BlockedCount       int `json:"blockedCount"`
}

type sandboxInsightsContext struct {
	Today                  sandboxInsightsToday              `json:"today"`
	TopDistractions        map[string]int                    `json:"topDistractions"`
	TopBlocked             map[string]int                    `json:"topBlocked"`
	ProjectBreakdown       map[string]int                    `json:"projectBreakdown"`
	CommunicationBreakdown map[string]CommunicationBreakdown `json:"communicationBreakdown"`
}

// sandboxContext provides context for the current rule execution including usage data and helper functions.
type sandboxContext struct {
	Usage    sandboxUsageContext    `json:"usage"`
	Insights sandboxInsightsContext `json:"insights"`

	// Helper functions
	Now                 func(loc *time.Location) time.Time                                   `json:"-"`
	MinutesUsedInPeriod func(appName, hostname string, durationMinutes int64) (int64, error) `json:"-"`
}

type sandboxContextOption func(*sandboxContext)

func WithAppNameContext(appName string) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.Usage.Metadata.AppName = appName
	}
}

func WithWindowTitleContext(title string) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.Usage.Metadata.Title = title
	}
}

func WithBrowserURLContext(url string) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.Usage.Metadata.URL = url

		u, err := parseURLNormalized(url)
		if err == nil {
			ctx.Usage.Metadata.Hostname = u.Hostname()
			ctx.Usage.Metadata.Path = u.Path
			ctx.Usage.Metadata.Domain, _ = publicsuffix.EffectiveTLDPlusOne(u.Hostname())
		}
	}
}

func WithNowContext(now time.Time) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.Now = func(loc *time.Location) time.Time {
			return now.In(loc)
		}
	}
}

func WithMinutesUsedInPeriodContext(minutesUsedInPeriod func(appName, hostname string, durationMinutes int64) (int64, error)) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.MinutesUsedInPeriod = minutesUsedInPeriod
	}
}

func WithMinutesSinceLastBlockContext(minutesSinceLastBlock int) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.Usage.Insights.MinutesSinceLastBlock = minutesSinceLastBlock
	}
}

func WithMinutesUsedSinceLastBlockContext(minutesUsedSinceLastBlock int) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.Usage.Insights.MinutesUsedSinceLastBlock = minutesUsedSinceLastBlock
	}
}

func WithClassificationContext(classification Classification) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.Usage.Metadata.Classification = string(classification)
	}
}

func NewSandboxContext(opts ...sandboxContextOption) sandboxContext {
	ctx := sandboxContext{
		Usage: sandboxUsageContext{
			Insights: sandboxUsageInsights{
				MinutesSinceLastBlock:      -1,
				MinutesUsedSinceLastBlock:  -1,
				LastBlockedDurationMinutes: -1,
			},
		},
		Insights: sandboxInsightsContext{
			TopDistractions:        make(map[string]int),
			TopBlocked:             make(map[string]int),
			ProjectBreakdown:       make(map[string]int),
			CommunicationBreakdown: make(map[string]CommunicationBreakdown),
		},
	}

	for _, opt := range opts {
		opt(&ctx)
	}

	return ctx
}
