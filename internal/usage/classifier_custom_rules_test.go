package usage

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/focusd-so/focusd/internal/settings"
)

func setupSettingsService(t *testing.T) *settings.Service {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	service, err := settings.NewService(db)
	require.NoError(t, err, "failed to create settings service")

	return service
}

func setUpService(t *testing.T, options ...Option) (*Service, *gorm.DB) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	service, err := NewService(context.Background(), db, options...)
	require.NoError(t, err, "failed to create service")

	return service, db
}

var customRulesApps = `
/**
* Custom classification logic.
* Return a ClassificationDecision to override the default, or undefined to keep the default.
*/
export function classify(ctx: Context): ClassificationDecision | undefined {
	console.log("should capture this");

	if (now().getHours() == 10 && now().getMinutes() > 0 && now().getMinutes() < 30) {
		return {
			classification: Classification.Productive,
			reasoning: "Work-related activity",
			tags: ["work", "productivity"],
		}
	}

	console.log("and this too");

	if (ctx.appName == "Slack") {
		return {
			classification: Classification.Neutral,
			reasoning: "Slack is a neutral app",
			tags: ["communication", "work"],
		}
	}

	console.log("also this");

	if (ctx.minutesSinceLastBlock >= 20 && ctx.minutesUsedSinceLastBlock < 5 && ctx.appName == "Discord") {
		return {
			classification: Classification.Neutral,
			reasoning: "Allow using 5 mins every 20 mins",
			tags: ["resting", "relaxing"],
		}
	}

	return undefined;
}

/**
* Custom termination logic (blocking).
* Return a TerminationDecision to override the default, or undefined to keep the default.
*/
export function terminationMode(ctx: Context): TerminationDecision | undefined {
	
}
`

var customRulesWithMinutesUsedInPeriod = `
export function classify(ctx: Context): ClassificationDecision | undefined {
	const minutesUsed = ctx.minutesUsedInPeriod(60);
	
	if (minutesUsed > 30) {
		return {
			classification: Classification.Distracting,
			reasoning: "Too much time spent: " + minutesUsed + " minutes",
			tags: ["limit-exceeded"],
		}
	}
	
	return {
		classification: Classification.Neutral,
		reasoning: "Under limit: " + minutesUsed + " minutes",
		tags: ["within-limit"],
	}
}
`

var customRulesWebsite = `
export function classify(ctx: Context): ClassificationDecision | undefined {
	// Match by domain
	if (ctx.domain === "youtube.com") {
		return {
			classification: Classification.Distracting,
			reasoning: "YouTube is distracting",
			tags: ["video", "entertainment"],
		}
	}
	
	// Match by hostname (subdomain-aware)
	if (ctx.hostname === "docs.google.com") {
		return {
			classification: Classification.Productive,
			reasoning: "Google Docs is productive",
			tags: ["docs", "work"],
		}
	}
	
	// Match by path
	if (ctx.hostname === "github.com" && ctx.path.startsWith("/pulls")) {
		return {
			classification: Classification.Productive,
			reasoning: "Reviewing pull requests",
			tags: ["code-review", "work"],
		}
	}
	
	// Match by full URL
	if (ctx.url === "https://twitter.com/home") {
		return {
			classification: Classification.Distracting,
			reasoning: "Twitter home feed",
			tags: ["social-media"],
		}
	}
	
	return undefined;
}
`

func TestClassifyCustomRules_Application(t *testing.T) {
	settingsService := setupSettingsService(t)
	require.NoError(t, settingsService.Save(settings.SettingsKeyCustomRules, customRulesApps))

	service, _ := setUpService(t, WithSettingsService(settingsService))

	// match Slack app
	response, err := service.ClassifyCustomRules(context.Background(), "Slack", "/Applications/Slack.app/Contents/MacOS/Slack", nil)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, ClassificationNeutral, response.Classification)
	require.Equal(t, "Slack is a neutral app", response.Reasoning)
	require.Equal(t, []string{"communication", "work"}, response.Tags)

	// no match Steam app
	response, err = service.ClassifyCustomRules(context.Background(), "Steam", "/Applications/Steam.app/Contents/MacOS/Steam", nil)
	require.NoError(t, err)
	require.Nil(t, response)
}

func TestClassifyCustomRules_Application_Time(t *testing.T) {
	settingsService := setupSettingsService(t)
	require.NoError(t, settingsService.Save(settings.SettingsKeyCustomRules, customRulesApps))

	service, _ := setUpService(t, WithSettingsService(settingsService))

	sandboxCtx := sandboxContext{
		AppName:        "Discord",
		ExecutablePath: "com.hnc.discord",
		Now: func(loc *time.Location) time.Time {
			return time.Date(2026, 1, 1, 10, 15, 0, 0, time.Local)
		},
	}

	// match Productive time
	response, err := service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)
	require.NoError(t, err)
	require.Equal(t, ClassificationProductive, response.Classification)
	require.Equal(t, "Work-related activity", response.Reasoning)
	require.Equal(t, []string{"work", "productivity"}, response.Tags)

	// no match Productive time
	sandboxCtx.Now = func(loc *time.Location) time.Time {
		return time.Date(2026, 1, 1, 10, 45, 0, 0, time.Local)
	}
	response, err = service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)
	require.NoError(t, err)
	require.Nil(t, response)

	// no match Productive hour
	sandboxCtx.Now = func(loc *time.Location) time.Time {
		return time.Date(2026, 1, 1, 11, 0, 0, 0, time.Local)
	}
	response, err = service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)
	require.NoError(t, err)
	require.Nil(t, response)
}

func TestClassifyCustomRules_Application_MinutesSinceLastBlock(t *testing.T) {
	settingsService := setupSettingsService(t)
	require.NoError(t, settingsService.Save(settings.SettingsKeyCustomRules, customRulesApps))

	service, _ := setUpService(t, WithSettingsService(settingsService))

	minutesSinceLastBlock := 21
	minutesUsedSinceLastBlock := 3

	sandboxCtx := sandboxContext{
		AppName:                   "Discord",
		ExecutablePath:            "/Applications/Discord.app/Contents/MacOS/Discord",
		MinutesSinceLastBlock:     &minutesSinceLastBlock,
		MinutesUsedSinceLastBlock: &minutesUsedSinceLastBlock,
	}

	// match MinutesSinceLastBlock rule
	response, err := service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)
	require.NotNil(t, response)
	require.NoError(t, err)
	require.Equal(t, ClassificationNeutral, response.Classification)
	require.Equal(t, "Allow using 5 mins every 20 mins", response.Reasoning)
	require.Equal(t, []string{"resting", "relaxing"}, response.Tags)
}

func TestClassifyCustomRules_ExecutionLogs(t *testing.T) {
	settingsService := setupSettingsService(t)
	require.NoError(t, settingsService.Save(settings.SettingsKeyCustomRules, customRulesApps))

	service, db := setUpService(t, WithSettingsService(settingsService))

	// match Productive time
	sandboxCtx := sandboxContext{
		AppName:        "Discord",
		ExecutablePath: "/Applications/Discord.app/Contents/MacOS/Discord",
	}

	service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)

	// read execution logs
	var log SandboxExecutionLog
	if err := db.Order("id DESC").Limit(1).First(&log).Error; err != nil {
		t.Fatalf("failed to find execution log: %v", err)
	}

	require.Equal(t, `["should capture this","and this too","also this"]`, strings.Trim(log.Logs, "\n"))
}

func TestClassifyCustomRules_MinutesUsedInPeriod(t *testing.T) {
	settingsService := setupSettingsService(t)
	require.NoError(t, settingsService.Save(settings.SettingsKeyCustomRules, customRulesWithMinutesUsedInPeriod))

	service, _ := setUpService(t, WithSettingsService(settingsService))

	var calledWithMinutes int64

	// Test case 1: Minutes used exceeds limit (> 30)
	sandboxCtx := sandboxContext{
		AppName:        "YouTube",
		ExecutablePath: "/Applications/YouTube.app",
		Hostname:       "youtube.com",
		MinutesUsedInPeriod: func(_, _ string, durationMinutes int64) (int64, error) {
			calledWithMinutes = durationMinutes
			return 45, nil // Return 45 minutes (exceeds 30 limit)
		},
	}

	response, err := service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, ClassificationDistracting, response.Classification)
	require.Equal(t, "Too much time spent: 45 minutes", response.Reasoning)
	require.Equal(t, []string{"limit-exceeded"}, response.Tags)

	// Verify callback was called with correct duration parameter
	require.Equal(t, int64(60), calledWithMinutes)

	// Test case 2: Minutes used under limit (<= 30)
	sandboxCtx.MinutesUsedInPeriod = func(bundleID, hostname string, durationMinutes int64) (int64, error) {
		return 15, nil // Return 15 minutes (under 30 limit)
	}

	response, err = service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, ClassificationNeutral, response.Classification)
	require.Equal(t, "Under limit: 15 minutes", response.Reasoning)
	require.Equal(t, []string{"within-limit"}, response.Tags)

	// Test case 3: Minutes used exactly at limit (30)
	sandboxCtx.MinutesUsedInPeriod = func(bundleID, hostname string, durationMinutes int64) (int64, error) {
		return 30, nil // Return exactly 30 minutes (at limit, should be Neutral)
	}

	response, err = service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, ClassificationNeutral, response.Classification)
	require.Equal(t, "Under limit: 30 minutes", response.Reasoning)
}

func TestClassifyCustomRules_MinutesUsedInPeriod_NilFunction(t *testing.T) {
	settingsService := setupSettingsService(t)
	require.NoError(t, settingsService.Save(settings.SettingsKeyCustomRules, customRulesWithMinutesUsedInPeriod))

	service, _ := setUpService(t, WithSettingsService(settingsService))

	// Test with nil MinutesUsedInPeriod - should default to 0 and classify as Neutral
	sandboxCtx := sandboxContext{
		AppName:             "YouTube",
		ExecutablePath:      "/Applications/YouTube.app",
		Hostname:            "youtube.com",
		MinutesUsedInPeriod: nil, // Explicitly nil
	}

	response, err := service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)
	require.NoError(t, err)
	require.NotNil(t, response)
	// When MinutesUsedInPeriod is nil, the JS fallback returns 0, so it should be under limit
	require.Equal(t, ClassificationNeutral, response.Classification)
	require.Equal(t, "Under limit: 0 minutes", response.Reasoning)
	require.Equal(t, []string{"within-limit"}, response.Tags)
}

func TestClassifyCustomRules_Website(t *testing.T) {
	settingsService := setupSettingsService(t)
	require.NoError(t, settingsService.Save(settings.SettingsKeyCustomRules, customRulesWebsite))

	service, _ := setUpService(t, WithSettingsService(settingsService))

	// Test case 1: Domain matching - youtube.com from www.youtube.com
	sandboxCtx := sandboxContext{
		AppName:        "Chrome",
		ExecutablePath: "/Applications/Google Chrome.app",
		Hostname:       "www.youtube.com",
		Domain:         "youtube.com",
		Path:           "/watch",
		URL:            "https://www.youtube.com/watch?v=abc123",
	}

	response, err := service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, ClassificationDistracting, response.Classification)
	require.Equal(t, "YouTube is distracting", response.Reasoning)
	require.Equal(t, []string{"video", "entertainment"}, response.Tags)

	// Test case 2: Hostname matching - docs.google.com (subdomain)
	sandboxCtx = sandboxContext{
		AppName:        "Chrome",
		ExecutablePath: "/Applications/Google Chrome.app",
		Hostname:       "docs.google.com",
		Domain:         "google.com",
		Path:           "/document/d/123",
		URL:            "https://docs.google.com/document/d/123",
	}

	response, err = service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, ClassificationProductive, response.Classification)
	require.Equal(t, "Google Docs is productive", response.Reasoning)
	require.Equal(t, []string{"docs", "work"}, response.Tags)

	// Test case 3: Path matching - github.com/pulls
	sandboxCtx = sandboxContext{
		AppName:        "Chrome",
		ExecutablePath: "/Applications/Google Chrome.app",
		Hostname:       "github.com",
		Domain:         "github.com",
		Path:           "/pulls",
		URL:            "https://github.com/pulls",
	}

	response, err = service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, ClassificationProductive, response.Classification)
	require.Equal(t, "Reviewing pull requests", response.Reasoning)
	require.Equal(t, []string{"code-review", "work"}, response.Tags)

	// Test case 4: Path matching with deeper path - github.com/pulls/assigned
	sandboxCtx = sandboxContext{
		AppName:        "Chrome",
		ExecutablePath: "/Applications/Google Chrome.app",
		Hostname:       "github.com",
		Domain:         "github.com",
		Path:           "/pulls/assigned",
		URL:            "https://github.com/pulls/assigned",
	}

	response, err = service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, ClassificationProductive, response.Classification)
	require.Equal(t, "Reviewing pull requests", response.Reasoning)

	// Test case 5: Full URL matching - twitter.com/home
	sandboxCtx = sandboxContext{
		AppName:        "Chrome",
		ExecutablePath: "/Applications/Google Chrome.app",
		Hostname:       "twitter.com",
		Domain:         "twitter.com",
		Path:           "/home",
		URL:            "https://twitter.com/home",
	}

	response, err = service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, ClassificationDistracting, response.Classification)
	require.Equal(t, "Twitter home feed", response.Reasoning)
	require.Equal(t, []string{"social-media"}, response.Tags)

	// Test case 6: No match - random website
	sandboxCtx = sandboxContext{
		AppName:        "Chrome",
		ExecutablePath: "/Applications/Google Chrome.app",
		Hostname:       "example.com",
		Domain:         "example.com",
		Path:           "/",
		URL:            "https://example.com/",
	}

	response, err = service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)
	require.NoError(t, err)
	require.Nil(t, response)

	// Test case 7: github.com but NOT /pulls path - should not match
	sandboxCtx = sandboxContext{
		AppName:        "Chrome",
		ExecutablePath: "/Applications/Google Chrome.app",
		Hostname:       "github.com",
		Domain:         "github.com",
		Path:           "/user/repo",
		URL:            "https://github.com/user/repo",
	}

	response, err = service.ClassifyCustomRulesWithSandbox(context.Background(), sandboxCtx)
	require.NoError(t, err)
	require.Nil(t, response)
}
