package usage_test

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/focusd-so/focusd/internal/usage"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

var customRulesApps = `
/**
* Custom classification logic.
* Return a ClassificationDecision to override the default, or undefined to keep the default.
*/
export function classify(context: UsageContext): ClassificationDecision | undefined {
	console.log("should capture this");

	if (now().getHours() == 10 && now().getMinutes() > 0 && now().getMinutes() < 30) {
		return {
			classification: Classification.Productive,
			classificationReasoning: "Work-related activity",
			tags: ["work", "productivity"],
		}
	}

	console.log("and this too");

	if (context.appName == "Slack") {
		return {
			classification: Classification.Neutral,
			classificationReasoning: "Slack is a neutral app",
			tags: ["communication", "work"],
		}
	}

	console.log("also this");

	if (context.minutesSinceLastBlock >= 20 && context.minutesUsedSinceLastBlock < 5 && context.appName == "Discord") {
		return {
			classification: Classification.Neutral,
			classificationReasoning: "Allow using 5 mins every 20 mins",
			tags: ["resting", "relaxing"],
		}
	}

	return undefined;
}

/**
* Custom termination logic (blocking).
* Return a EnforcementDecision to override the default, or undefined to keep the default.
*/
export function enforcementDecision(context: UsageContext): EnforcementDecision | undefined {
	
}
`

var customRulesWithMinutesUsedInPeriod = `
export function classify(context: UsageContext): ClassificationDecision | undefined {
	const minutesUsed = context.minutesUsedInPeriod(60);
	
	if (minutesUsed > 30) {
		return {
			classification: Classification.Distracting,
			classificationReasoning: "Too much time spent: " + minutesUsed + " minutes",
			tags: ["limit-exceeded"],
		}
	}
	
	return {
		classification: Classification.Neutral,
		classificationReasoning: "Under limit: " + minutesUsed + " minutes",
		tags: ["within-limit"],
	}
}
`

var customRulesWebsite = `
export function classify(context: UsageContext): ClassificationDecision | undefined {
	// Match by domain
	if (context.domain === "youtube.com") {
		return {
			classification: Classification.Distracting,
			classificationReasoning: "YouTube is distracting",
			tags: ["video", "entertainment"],
		}
	}
	
	// Match by hostname (subdomain-aware)
	if (context.hostname === "docs.google.com") {
		return {
			classification: Classification.Productive,
			classificationReasoning: "Google Docs is productive",
			tags: ["docs", "work"],
		}
	}
	
	// Match by path
	if (context.hostname === "github.com" && context.path.startsWith("/pulls")) {
		return {
			classification: Classification.Productive,
			classificationReasoning: "Reviewing pull requests",
			tags: ["code-review", "work"],
		}
	}
	
	// Match by full URL
	if (context.url === "https://twitter.com/home") {
		return {
			classification: Classification.Distracting,
			classificationReasoning: "Twitter home feed",
			tags: ["social-media"],
		}
	}
	
	return undefined;
}
`

func TestClassifyCustomRules_Application(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(customRulesApps))
	viper.SetDefault("custom_rules_js", []string{encoded})

	service, _ := setUpService(t)

	// match Slack app
	response, err := service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("Slack"))
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, usage.ClassificationNeutral, response.Classification)
	require.Equal(t, "Slack is a neutral app", response.Reasoning)
	require.Equal(t, []string{"communication", "work"}, response.Tags)

	// no match Steam app
	response, err = service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("Steam"))
	require.NoError(t, err)
	require.Nil(t, response)
}

func TestClassifyCustomRules_Application_Time(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(customRulesApps))
	viper.SetDefault("custom_rules_js", []string{encoded})

	service, _ := setUpService(t)

	now := time.Date(2026, 1, 1, 10, 15, 0, 0, time.Local)

	response, err := service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("Discord"), usage.WithNowContext(now))

	require.NoError(t, err)
	require.Equal(t, usage.ClassificationProductive, response.Classification)
	require.Equal(t, "Work-related activity", response.Reasoning)
	require.Equal(t, []string{"work", "productivity"}, response.Tags)

	// no match Productive time

	now = time.Date(2026, 1, 1, 10, 45, 0, 0, time.Local)

	response, err = service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("Discord"), usage.WithNowContext(now))
	require.NoError(t, err)
	require.Nil(t, response)

	// no match Productive hour

	now = time.Date(2026, 1, 1, 11, 0, 0, 0, time.Local)
	response, err = service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("Discord"), usage.WithNowContext(now))
	require.NoError(t, err)
	require.Nil(t, response)
}

func TestClassifyCustomRules_Application_MinutesSinceLastBlock(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(customRulesApps))
	viper.SetDefault("custom_rules_js", []string{encoded})

	service, _ := setUpService(t)

	minutesSinceLastBlock := 21
	minutesUsedSinceLastBlock := 3

	// match MinutesSinceLastBlock rule
	response, err := service.ClassifyCustomRules(context.Background(),
		usage.WithAppNameContext("Discord"),
		usage.WithMinutesSinceLastBlockContext(minutesSinceLastBlock),
		usage.WithMinutesUsedSinceLastBlockContext(minutesUsedSinceLastBlock),
	)
	require.NotNil(t, response)
	require.NoError(t, err)
	require.Equal(t, usage.ClassificationNeutral, response.Classification)
	require.Equal(t, "Allow using 5 mins every 20 mins", response.Reasoning)
	require.Equal(t, []string{"resting", "relaxing"}, response.Tags)
}

func TestClassifyCustomRules_ExecutionLogs(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(customRulesApps))
	viper.SetDefault("custom_rules_js", []string{encoded})

	service, db := setUpService(t)

	service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("Discord"))

	// read execution logs
	var log usage.SandboxExecutionLog
	if err := db.Order("id DESC").Limit(1).First(&log).Error; err != nil {
		t.Fatalf("failed to find execution log: %v", err)
	}

	require.Equal(t, `["should capture this","and this too","also this"]`, strings.Trim(log.Logs, "\n"))
}

func TestClassifyCustomRules_MinutesUsedInPeriod(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(customRulesWithMinutesUsedInPeriod))
	viper.SetDefault("custom_rules_js", []string{encoded})

	service, _ := setUpService(t)

	var calledWithMinutes int64

	response, err := service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("YouTube"), usage.WithMinutesUsedInPeriodContext(func(_, _ string, durationMinutes int64) (int64, error) {
		calledWithMinutes = durationMinutes
		return 45, nil // Return 45 minutes (exceeds 30 limit)
	}))
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, usage.ClassificationDistracting, response.Classification)
	require.Equal(t, "Too much time spent: 45 minutes", response.Reasoning)
	require.Equal(t, []string{"limit-exceeded"}, response.Tags)

	// Verify callback was called with correct duration parameter
	require.Equal(t, int64(60), calledWithMinutes)

	response, err = service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("YouTube"), usage.WithMinutesUsedInPeriodContext(func(_, _ string, durationMinutes int64) (int64, error) {
		return 15, nil // Return 15 minutes (under 30 limit)
	}))
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, usage.ClassificationNeutral, response.Classification)
	require.Equal(t, "Under limit: 15 minutes", response.Reasoning)
	require.Equal(t, []string{"within-limit"}, response.Tags)

	response, err = service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("YouTube"), usage.WithMinutesUsedInPeriodContext(func(_, _ string, durationMinutes int64) (int64, error) {
		return 30, nil // Return exactly 30 minutes (at limit, should be Neutral)
	}))
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, usage.ClassificationNeutral, response.Classification)
	require.Equal(t, "Under limit: 30 minutes", response.Reasoning)
}

func TestClassifyCustomRules_MinutesUsedInPeriod_NilFunction(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(customRulesWithMinutesUsedInPeriod))
	viper.SetDefault("custom_rules_js", []string{encoded})

	service, _ := setUpService(t)

	response, err := service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("YouTube"), usage.WithMinutesUsedInPeriodContext(nil))
	require.NoError(t, err)
	require.NotNil(t, response)
	// When MinutesUsedInPeriod is nil, the JS fallback returns 0, so it should be under limit
	require.Equal(t, usage.ClassificationNeutral, response.Classification)
	require.Equal(t, "Under limit: 0 minutes", response.Reasoning)
	require.Equal(t, []string{"within-limit"}, response.Tags)
}

func TestClassifyCustomRules_Website(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(customRulesWebsite))
	viper.SetDefault("custom_rules_js", []string{encoded})

	service, _ := setUpService(t)

	response, err := service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("Chrome"), usage.WithBrowserURLContext("https://www.youtube.com/watch?v=abc123"))
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, usage.ClassificationDistracting, response.Classification)
	require.Equal(t, "YouTube is distracting", response.Reasoning)
	require.Equal(t, []string{"video", "entertainment"}, response.Tags)

	response, err = service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("Chrome"), usage.WithBrowserURLContext("https://docs.google.com/document/d/123"))
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, usage.ClassificationProductive, response.Classification)
	require.Equal(t, "Google Docs is productive", response.Reasoning)
	require.Equal(t, []string{"docs", "work"}, response.Tags)

	response, err = service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("Chrome"), usage.WithBrowserURLContext("https://github.com/pulls"))
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, usage.ClassificationProductive, response.Classification)
	require.Equal(t, "Reviewing pull requests", response.Reasoning)
	require.Equal(t, []string{"code-review", "work"}, response.Tags)

	response, err = service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("Chrome"), usage.WithBrowserURLContext("https://github.com/pulls/assigned"))
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, usage.ClassificationProductive, response.Classification)
	require.Equal(t, "Reviewing pull requests", response.Reasoning)

	response, err = service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("Chrome"), usage.WithBrowserURLContext("https://twitter.com/home"))
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, usage.ClassificationDistracting, response.Classification)
	require.Equal(t, "Twitter home feed", response.Reasoning)
	require.Equal(t, []string{"social-media"}, response.Tags)

	response, err = service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("Chrome"), usage.WithBrowserURLContext("https://example.com/"))
	require.NoError(t, err)
	require.Nil(t, response)

	response, err = service.ClassifyCustomRules(context.Background(), usage.WithAppNameContext("Chrome"), usage.WithBrowserURLContext("https://github.com/user/repo"))
	require.NoError(t, err)
	require.Nil(t, response)
}
