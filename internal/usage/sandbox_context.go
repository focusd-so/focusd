package usage

import (
	"time"

	"golang.org/x/net/publicsuffix"
)

type sandboxUsageMetadata struct {
	AppName        string `json:"appName"`
	Title          string `json:"title"`
	Host           string `json:"host"`
	Path           string `json:"path"`
	Domain         string `json:"domain"`
	URL            string `json:"url"`
	Classification string `json:"classification"`
}

type sandboxUsageBlocked struct {
	Count int  `json:"count"`
	Since *int `json:"since"`
	Used  *int `json:"used"`
	Last  *int `json:"last"`
}

type sandboxUsageDuration struct {
	Today              int  `json:"today"`
	SinceLastBlock     *int `json:"sinceLastBlock"`
	UsedSinceLastBlock *int `json:"usedSinceLastBlock"`
	LastBlocked        *int `json:"lastBlocked"`
}

type sandboxPeriodSummary struct {
	FocusScore         int `json:"focusScore"`
	ProductiveMinutes  int `json:"productiveMinutes"`
	DistractingMinutes int `json:"distractingMinutes"`
}

type sandboxUsageCurrentInsights struct {
	Duration sandboxUsageDuration `json:"duration"`
	Blocks   sandboxUsageBlocked  `json:"blocks"`
}

type sandboxUsageInsights struct {
	Today   sandboxPeriodSummary        `json:"today"`
	Hour    sandboxPeriodSummary        `json:"hour"`
	Current sandboxUsageCurrentInsights `json:"current"`
}

type sandboxUsageContext struct {
	Meta     sandboxUsageMetadata `json:"meta"`
	Insights sandboxUsageInsights `json:"insights"`
}

// sandboxContext provides context for the current rule execution including usage data and helper functions.
type sandboxContext struct {
	Usage sandboxUsageContext `json:"usage"`

	// Helper functions
	Now                 func(loc *time.Location) time.Time                                   `json:"-"`
	MinutesUsedInPeriod func(appName, hostname string, durationMinutes int64) (int64, error) `json:"-"`
}

type sandboxContextOption func(*sandboxContext)

func WithAppNameContext(appName string) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.Usage.Meta.AppName = appName
	}
}

func WithWindowTitleContext(title string) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.Usage.Meta.Title = title
	}
}

func WithBrowserURLContext(url string) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.Usage.Meta.URL = url

		u, err := parseURLNormalized(url)
		if err == nil {
			ctx.Usage.Meta.Host = u.Hostname()
			ctx.Usage.Meta.Path = u.Path
			ctx.Usage.Meta.Domain, _ = publicsuffix.EffectiveTLDPlusOne(u.Hostname())
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
		ctx.Usage.Insights.Current.Duration.SinceLastBlock = &minutesSinceLastBlock
	}
}

func WithMinutesUsedSinceLastBlockContext(minutesUsedSinceLastBlock int) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.Usage.Insights.Current.Duration.UsedSinceLastBlock = &minutesUsedSinceLastBlock
	}
}

func WithClassificationContext(classification Classification) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.Usage.Meta.Classification = string(classification)
	}
}

func NewSandboxContext(opts ...sandboxContextOption) sandboxContext {
	ctx := sandboxContext{}

	for _, opt := range opts {
		opt(&ctx)
	}

	return ctx
}
