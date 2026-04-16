package usage

import (
	"log/slog"
	"net/url"
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
	Now func(loc *time.Location) time.Time `json:"-"`
}

type sandboxContextOption func(*sandboxContext)

func (s *Service) createSandboxContext(opts ...sandboxContextOption) sandboxContext {
	ctx := sandboxContext{}

	for _, opt := range opts {
		opt(&ctx)
	}

	if ctx.Now == nil {
		ctx.Now = func(loc *time.Location) time.Time {
			return time.Now().In(loc)
		}
	}

	if err := s.populateInsightsContext(&ctx); err != nil {
		slog.Debug("failed to populate sandbox insights context", "error", err)
	}

	if err := s.populateCurrentUsageContext(&ctx); err != nil {
		slog.Debug("failed to populate sandbox current-usage context", "error", err)
	}

	return ctx
}

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

func WithBrowserURLContext(url *url.URL) sandboxContextOption {
	return func(ctx *sandboxContext) {
		if url == nil {
			return
		}

		ctx.Usage.Meta.URL = url.String()
		ctx.Usage.Meta.Host = url.Hostname()
		ctx.Usage.Meta.Path = url.Path
		ctx.Usage.Meta.Domain, _ = publicsuffix.EffectiveTLDPlusOne(url.Hostname())
	}
}

func WithNowContext(now time.Time) sandboxContextOption {
	return func(ctx *sandboxContext) {
		ctx.Now = func(loc *time.Location) time.Time {
			return now.In(loc)
		}
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
