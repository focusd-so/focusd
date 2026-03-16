package usage_test

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/focusd-so/focusd/internal/usage"
)

func TestProtection_PauseProtection(t *testing.T) {
	service, db := setUpService(t)

	protectionPause, err := service.PauseProtection(10, "test")
	require.NoError(t, err)
	require.NotEqual(t, int64(0), protectionPause.ID)

	var (
		requestedDurationSeconds      = 10
		expectedActualDurationSeconds = requestedDurationSeconds
		expectedResumedAt             = protectionPause.CreatedAt + int64(requestedDurationSeconds)
	)

	var readProtectionPause usage.ProtectionPause
	if err := db.Where("id = ?", protectionPause.ID).First(&readProtectionPause).Error; err != nil {
		t.Fatalf("failed to find protection pause: %v", err)
	}

	require.NotEqual(t, int64(0), readProtectionPause.ID)
	require.Equal(t, requestedDurationSeconds, readProtectionPause.RequestedDurationSeconds)
	require.Equal(t, expectedActualDurationSeconds, readProtectionPause.ActualDurationSeconds)
	require.Equal(t, expectedResumedAt, readProtectionPause.ResumedAt)
	require.Equal(t, "protection paused for 10s expired", readProtectionPause.ResumedReason)

	// wait 3 seconds

	time.Sleep(3 * time.Second)

	// resume protection
	_, err = service.ResumeProtection("just because")
	require.NoError(t, err)

	var readProtectionPause2 usage.ProtectionPause
	if err := db.Where("id = ?", protectionPause.ID).First(&readProtectionPause2).Error; err != nil {
		t.Fatalf("failed to find protection pause: %v", err)
	}

	require.NotEqual(t, int64(0), readProtectionPause2.ID)
	require.Equal(t, requestedDurationSeconds, readProtectionPause2.RequestedDurationSeconds)
	require.Equal(t, 3, readProtectionPause2.ActualDurationSeconds)
	require.Equal(t, protectionPause.CreatedAt+3, readProtectionPause2.ResumedAt)
	require.Equal(t, "just because", readProtectionPause2.ResumedReason)
}

func TestProtection_PauseProtectionIgnoreWhenAlreadyPaused(t *testing.T) {
	service, _ := setUpService(t)

	protectionPause, err := service.PauseProtection(10, "test")
	require.NoError(t, err)
	require.NotEqual(t, int64(0), protectionPause.ID)
	require.Equal(t, "test", protectionPause.Reason)

	protectionPause2, err := service.PauseProtection(10, "test")
	require.NoError(t, err)
	require.NotEqual(t, int64(0), protectionPause2.ID)
	require.Equal(t, "test", protectionPause2.Reason)

	require.Equal(t, protectionPause.ID, protectionPause2.ID)
}

func TestProtection_PauseProtectionEventsFired(t *testing.T) {
	var (
		onProtectionPausedCalled  = false
		onProtectionResumedCalled = false
	)

	service, _ := setUpService(t)

	service.OnProtectionPause(func(pause usage.ProtectionPause) {
		onProtectionPausedCalled = true
		require.NotEqual(t, int64(0), pause.ResumedAt)
		// this will still be initial requested duration
		require.Equal(t, 10, pause.ActualDurationSeconds)
		require.Equal(t, "protection paused for 10s expired", pause.ResumedReason)
	})

	service.OnProtectionResumed(func(pause usage.ProtectionPause) {
		onProtectionResumedCalled = true
		require.NotEqual(t, int64(0), pause.ResumedAt)
		// this is the actual duration calculated after the resume
		require.Equal(t, 3, pause.ActualDurationSeconds)
		require.Equal(t, "just because", pause.ResumedReason)
	})

	_, err := service.PauseProtection(10, "test")

	// wait 3 seconds
	time.Sleep(3 * time.Second)

	_, err = service.ResumeProtection("just because")

	require.NoError(t, err)
	require.True(t, onProtectionPausedCalled, "onProtectionPaused should be called")
	require.True(t, onProtectionResumedCalled, "onProtectionResumed should be called")
}

func TestProtection_GetPauseHistory_ReturnsWithinRange(t *testing.T) {
	service, db := setUpService(t)

	now := time.Now()

	// 1. Pause created today (should be returned)
	pause1 := usage.ProtectionPause{
		CreatedAt:             now.Unix(),
		ActualDurationSeconds: 60,
	}
	require.NoError(t, db.Create(&pause1).Error)

	// 2. Pause created 3 days ago (should be returned for 7 day window)
	pause2 := usage.ProtectionPause{
		CreatedAt:             now.AddDate(0, 0, -3).Unix(),
		ActualDurationSeconds: 120,
	}
	require.NoError(t, db.Create(&pause2).Error)

	// 3. Pause created 10 days ago (should NOT be returned for 7 day window)
	pause3 := usage.ProtectionPause{
		CreatedAt:             now.AddDate(0, 0, -10).Unix(),
		ActualDurationSeconds: 300,
	}
	require.NoError(t, db.Create(&pause3).Error)

	history, err := service.GetPauseHistory(7)
	require.NoError(t, err)
	require.Len(t, history, 2)

	// Ordered by created_at DESC (newest first)
	require.Equal(t, pause1.ID, history[0].ID)
	require.Equal(t, pause2.ID, history[1].ID)
}

func TestProtection_GetPauseHistory_ReturnsEmpty(t *testing.T) {
	service, _ := setUpService(t)

	history, err := service.GetPauseHistory(7)
	require.NoError(t, err)
	require.Empty(t, history)
}

func TestProtection_Whitelist_CreatesEntry(t *testing.T) {
	service, db := setUpService(t)

	err := service.Whitelist("/usr/bin/app", "example.com", 30*time.Minute)
	require.NoError(t, err)

	var entry usage.ProtectionWhitelist
	err = db.First(&entry).Error
	require.NoError(t, err)

	require.Equal(t, "/usr/bin/app", entry.AppName)
	require.NotNil(t, entry.Hostname)
	require.Equal(t, "example.com", *entry.Hostname)
	require.InDelta(t, time.Now().Add(30*time.Minute).Unix(), entry.ExpiresAt, 5)
}

func TestProtection_Whitelist_ReplacesExistingEntry(t *testing.T) {
	service, db := setUpService(t)

	// Create initial entry
	err := service.Whitelist("/usr/bin/app", "example.com", 10*time.Minute)
	require.NoError(t, err)

	// Create replacement entry
	err = service.Whitelist("/usr/bin/app", "example.com", 1*time.Hour)
	require.NoError(t, err)

	var entries []usage.ProtectionWhitelist
	err = db.Find(&entries).Error
	require.NoError(t, err)

	require.Len(t, entries, 1)
	require.Equal(t, "/usr/bin/app", entries[0].AppName)
	require.NotNil(t, entries[0].Hostname)
	require.Equal(t, "example.com", *entries[0].Hostname)
	require.InDelta(t, time.Now().Add(1*time.Hour).Unix(), entries[0].ExpiresAt, 5)
}

func TestProtection_Whitelist_NormalizesWWW(t *testing.T) {
	service, db := setUpService(t)

	err := service.Whitelist("Google Chrome", "www.youtube.com", 30*time.Minute)
	require.NoError(t, err)

	var entry usage.ProtectionWhitelist
	err = db.First(&entry).Error
	require.NoError(t, err)
	require.NotNil(t, entry.Hostname)
	require.Equal(t, "youtube.com", *entry.Hostname)
}

func TestProtection_Whitelist_ZeroDurationBug(t *testing.T) {
	service, _ := setUpService(t)

	err := service.Whitelist("/usr/bin/app", "", 0)
	require.Error(t, err, "duration must be at least 5 minutes")
}

func TestProtection_GetWhitelist_ReturnsActiveEntries(t *testing.T) {
	service, db := setUpService(t)
	now := time.Now().Unix()

	// 1. Future expiration (Active)
	active := usage.ProtectionWhitelist{
		AppName:   "/bin/active",
		ExpiresAt: now + 3600,
	}
	require.NoError(t, db.Create(&active).Error)

	// 2. Past expiration (Inactive)
	inactive := usage.ProtectionWhitelist{
		AppName:   "/bin/inactive",
		ExpiresAt: now - 3600,
	}
	require.NoError(t, db.Create(&inactive).Error)

	whitelist, err := service.GetWhitelist()
	require.NoError(t, err)
	require.Len(t, whitelist, 1)
	require.Equal(t, active.ID, whitelist[0].ID)
}

func TestProtection_GetWhitelist_ReturnsIndefiniteEntries(t *testing.T) {
	service, db := setUpService(t)

	// Indefinite expiration (ExpiresAt = 0)
	indefinite := usage.ProtectionWhitelist{
		AppName:   "/bin/indefinite",
		ExpiresAt: 0,
	}
	require.NoError(t, db.Create(&indefinite).Error)

	whitelist, err := service.GetWhitelist()
	require.NoError(t, err)
	require.Len(t, whitelist, 1)
	require.Equal(t, indefinite.ID, whitelist[0].ID)
}

func TestProtection_GetWhitelist_ReturnsEmpty(t *testing.T) {
	service, _ := setUpService(t)

	whitelist, err := service.GetWhitelist()
	require.NoError(t, err)
	require.Empty(t, whitelist)
}

func TestProtection_RemoveWhitelist_DeletesEntry(t *testing.T) {
	service, db := setUpService(t)

	entry := usage.ProtectionWhitelist{
		AppName: "/bin/app",
	}
	require.NoError(t, db.Create(&entry).Error)

	err := service.RemoveWhitelist(entry.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&usage.ProtectionWhitelist{}).Where("id = ?", entry.ID).Count(&count)
	require.Equal(t, int64(0), count)
}

func TestProtection_RemoveWhitelist_NonExistentID(t *testing.T) {
	service, _ := setUpService(t)

	err := service.RemoveWhitelist(9999)
	require.NoError(t, err)
}

func setUpServiceWithSettings(t *testing.T, customRules string) (*usage.Service, *gorm.DB) {
	t.Helper()

	db, _ := gorm.Open(sqlite.Open(memoryDSN(t)), &gorm.Config{})

	viper.SetDefault("custom_rules_js", []string{customRules})

	service, err := usage.NewService(context.Background(), db)
	require.NoError(t, err)

	return service, db
}

func TestProtection_CalculateTerminationMode_CustomRules(t *testing.T) {

	t.Run("ctx.appName is accessible", func(t *testing.T) {
		customRules := `
export function terminationMode(ctx) {
	if (ctx.appName == "Slack") {
		return {
			terminationMode: TerminationMode.Block,
			terminationReasoning: "Slack is blocked by custom rule",
		}
	}
	return undefined;
}
`
		service, _ := setUpServiceWithSettings(t, customRules)

		decision, err := service.CalculateTerminationMode(context.Background(), &usage.ApplicationUsage{
			Classification: usage.ClassificationNeutral,
			Application:    usage.Application{Name: "Slack"},
		})
		require.NoError(t, err)
		require.Equal(t, usage.TerminationModeBlock, decision.Mode)
		require.Equal(t, usage.TerminationModeSourceCustomRules, decision.Source)
		require.Equal(t, "Slack is blocked by custom rule", decision.Reasoning)
	})

	t.Run("ctx.classification is accessible", func(t *testing.T) {
		customRules := `
export function terminationMode(ctx) {
	if (ctx.classification == "distracting") {
		return {
			terminationMode: TerminationMode.Block,
			terminationReasoning: "distracting classification detected",
		}
	}
	return undefined;
}
`
		service, _ := setUpServiceWithSettings(t, customRules)

		decision, err := service.CalculateTerminationMode(context.Background(), &usage.ApplicationUsage{
			Classification: usage.ClassificationDistracting,
			Application:    usage.Application{Name: "YouTube"},
		})
		require.NoError(t, err)
		require.Equal(t, usage.TerminationModeBlock, decision.Mode)
		require.Equal(t, usage.TerminationModeSourceCustomRules, decision.Source)
		require.Equal(t, "distracting classification detected", decision.Reasoning)
	})

	t.Run("ctx.hostname and ctx.domain are accessible", func(t *testing.T) {
		// Note: parseURL strips "www." prefix, so "docs.google.com" stays as-is
		// while domain is extracted via publicsuffix as "google.com"
		customRules := `
export function terminationMode(ctx) {
	if (ctx.hostname == "docs.google.com" && ctx.domain == "google.com") {
		return {
			terminationMode: TerminationMode.Block,
			terminationReasoning: "Google Docs blocked via hostname/domain",
		}
	}
	return undefined;
}
`
		service, _ := setUpServiceWithSettings(t, customRules)

		url := "https://docs.google.com/document/d/123"
		decision, err := service.CalculateTerminationMode(context.Background(), &usage.ApplicationUsage{
			Classification: usage.ClassificationDistracting,
			BrowserURL:     &url,
			Application:    usage.Application{Name: "Chrome"},
		})
		require.NoError(t, err)
		require.Equal(t, usage.TerminationModeBlock, decision.Mode)
		require.Equal(t, usage.TerminationModeSourceCustomRules, decision.Source)
		require.Equal(t, "Google Docs blocked via hostname/domain", decision.Reasoning)
	})

	t.Run("ctx.url and ctx.path are accessible", func(t *testing.T) {
		customRules := `
export function terminationMode(ctx) {
	if (ctx.url == "https://github.com/pulls" && ctx.path == "/pulls") {
		return {
			terminationMode: TerminationMode.Allow,
			terminationReasoning: "PR reviews are allowed",
		}
	}
	return undefined;
}
`
		service, _ := setUpServiceWithSettings(t, customRules)

		url := "https://github.com/pulls"
		decision, err := service.CalculateTerminationMode(context.Background(), &usage.ApplicationUsage{
			Classification: usage.ClassificationDistracting,
			BrowserURL:     &url,
			Application:    usage.Application{Name: "Chrome"},
		})
		require.NoError(t, err)
		require.Equal(t, usage.TerminationModeAllow, decision.Mode)
		require.Equal(t, usage.TerminationModeSourceCustomRules, decision.Source)
		require.Equal(t, "PR reviews are allowed", decision.Reasoning)
	})

	t.Run("returns undefined falls through to default", func(t *testing.T) {
		customRules := `
export function terminationMode(ctx) {
	return undefined;
}
`
		service, _ := setUpServiceWithSettings(t, customRules)

		decision, err := service.CalculateTerminationMode(context.Background(), &usage.ApplicationUsage{
			Classification: usage.ClassificationNeutral,
			Application:    usage.Application{Name: "VSCode"},
		})
		require.NoError(t, err)
		// When custom rules return undefined, falls through to default logic.
		// Non-distracting classification results in Allow from the application source.
		require.Equal(t, usage.TerminationModeAllow, decision.Mode)
		require.Equal(t, usage.TerminationModeSourceApplication, decision.Source)
		require.Equal(t, "non distracting usage", decision.Reasoning)
	})
}

func TestProtection_CalculateTerminationMode_ProtectionPaused(t *testing.T) {
	service, _ := setUpService(t)

	protectionPause, err := service.PauseProtection(1000, "test")
	require.NoError(t, err)
	require.NotEqual(t, int64(0), protectionPause.ID)

	decision, err := service.CalculateTerminationMode(context.Background(), &usage.ApplicationUsage{
		Classification: usage.ClassificationDistracting,
		Application:    usage.Application{Name: "YouTube"},
	})
	require.NoError(t, err)
	require.Equal(t, usage.TerminationModePaused, decision.Mode)
	require.Equal(t, usage.TerminationModeSourcePaused, decision.Source)
	require.Equal(t, "focus protection has been paused by the user", decision.Reasoning)
}

func TestProtection_AllowedByWhitelist(t *testing.T) {
	t.Run("whitelisted by executable path", func(t *testing.T) {
		service, _ := setUpService(t)

		err := service.Whitelist("/usr/bin/app", "", 30*time.Minute)
		require.NoError(t, err)

		decision, err := service.CalculateTerminationMode(context.Background(), &usage.ApplicationUsage{
			Classification: usage.ClassificationDistracting,
			Application:    usage.Application{Name: "/usr/bin/app"},
		})
		require.NoError(t, err)
		require.Equal(t, usage.TerminationModeAllow, decision.Mode)
		require.Equal(t, usage.TerminationModeSourceWhitelist, decision.Source)
		require.Equal(t, "temporarily allowed usage by user", decision.Reasoning)
	})

	t.Run("whitelisted by executable path and hostname", func(t *testing.T) {
		service, _ := setUpService(t)

		err := service.Whitelist("/usr/bin/app", "example.com", 30*time.Minute)
		require.NoError(t, err)

		hostname := "example.com"
		decision, err := service.CalculateTerminationMode(context.Background(), &usage.ApplicationUsage{
			Classification: usage.ClassificationDistracting,
			Application:    usage.Application{Name: "/usr/bin/app", Hostname: &hostname},
		})
		require.NoError(t, err)
		require.Equal(t, usage.TerminationModeAllow, decision.Mode)
		require.Equal(t, usage.TerminationModeSourceWhitelist, decision.Source)
		require.Equal(t, "temporarily allowed usage by user", decision.Reasoning)
	})

	t.Run("does not allow other hostnames in same browser", func(t *testing.T) {
		service, _ := setUpService(t)

		err := service.Whitelist("Google Chrome", "www.amazon.com", 30*time.Minute)
		require.NoError(t, err)

		youtube := "youtube.com"
		decision, err := service.CalculateTerminationMode(context.Background(), &usage.ApplicationUsage{
			Classification: usage.ClassificationDistracting,
			Application:    usage.Application{Name: "Google Chrome", Hostname: &youtube},
		})
		require.NoError(t, err)
		require.Equal(t, usage.TerminationModeBlock, decision.Mode)
		require.Equal(t, usage.TerminationModeSourceApplication, decision.Source)
	})

	t.Run("strips www when evaluating whitelist", func(t *testing.T) {
		service, _ := setUpService(t)

		err := service.Whitelist("Google Chrome", "www.youtube.com", 30*time.Minute)
		require.NoError(t, err)

		hostname := "youtube.com"
		decision, err := service.CalculateTerminationMode(context.Background(), &usage.ApplicationUsage{
			Classification: usage.ClassificationDistracting,
			Application:    usage.Application{Name: "Google Chrome", Hostname: &hostname},
		})
		require.NoError(t, err)
		require.Equal(t, usage.TerminationModeAllow, decision.Mode)
		require.Equal(t, usage.TerminationModeSourceWhitelist, decision.Source)
	})

	t.Run("subdomains remain distinct", func(t *testing.T) {
		service, _ := setUpService(t)

		err := service.Whitelist("Google Chrome", "mail.google.com", 30*time.Minute)
		require.NoError(t, err)

		drive := "drive.google.com"
		decision, err := service.CalculateTerminationMode(context.Background(), &usage.ApplicationUsage{
			Classification: usage.ClassificationDistracting,
			Application:    usage.Application{Name: "Google Chrome", Hostname: &drive},
		})
		require.NoError(t, err)
		require.Equal(t, usage.TerminationModeBlock, decision.Mode)
	})
}
