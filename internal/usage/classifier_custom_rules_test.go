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
import {
	productive,
	neutral,
	runtime,
	type Classify,
	type Enforce,
} from "@focusd/runtime";

export function classify(): Classify | undefined {
	const { app, current } = runtime.usage;

	console.log("should capture this");

	if (runtime.time.now().getHours() == 10 && runtime.time.now().getMinutes() > 0 && runtime.time.now().getMinutes() < 30) {
		return productive("Work-related activity", ["work", "productivity"]);
	}

	console.log("and this too");

	if (app == "Slack") {
		return neutral("Slack is a neutral app", ["communication", "work"]);
	}

	console.log("also this");

	if (current.sinceBlock >= 20 && current.usedSinceBlock < 5 && app == "Discord") {
		return neutral("Allow using 5 mins every 20 mins", ["resting", "relaxing"]);
	}

	return undefined;
}

export function enforcement(): Enforce | undefined {
	
}
`

var customRulesWithMinutesUsedInPeriod = `
import { distracting, neutral, runtime, type Classify } from "@focusd/runtime";

export function classify(): Classify | undefined {
	const { current } = runtime.usage;
	const minutesUsed = current.last(60);
	
	if (minutesUsed > 30) {
		return distracting("Too much time spent: " + minutesUsed + " minutes", ["limit-exceeded"]);
	}
	
	return neutral("Under limit: " + minutesUsed + " minutes", ["within-limit"]);
}
`

var customRulesWebsite = `
import { productive, distracting, runtime, type Classify } from "@focusd/runtime";

export function classify(): Classify | undefined {
	const { domain, host, path, url } = runtime.usage;

	if (domain === "youtube.com") {
		return distracting("YouTube is distracting", ["video", "entertainment"]);
	}
	
	if (host === "docs.google.com") {
		return productive("Google Docs is productive", ["docs", "work"]);
	}
	
	if (host === "github.com" && path.startsWith("/pulls")) {
		return productive("Reviewing pull requests", ["code-review", "work"]);
	}
	
	if (url === "https://twitter.com/home") {
		return distracting("Twitter home feed", ["social-media"]);
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
