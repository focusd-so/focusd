package usage_test

import (
	"context"
	"fmt"
	urlpkg "net/url"
	"testing"
	"time"

	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/focusd-so/focusd/internal/usage"
)

func memoryDSN(t *testing.T) string {
	t.Helper()

	return fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", urlpkg.QueryEscape(t.Name()))
}

func setUpService(t *testing.T, options ...usage.Option) (*usage.Service, *gorm.DB) {
	db, _ := gorm.Open(sqlite.Open(memoryDSN(t)), &gorm.Config{})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	service, err := usage.NewService(ctx, db, options...)
	require.NoError(t, err, "failed to create service")

	return service, db
}

func TestService_TransitionsBetweenIdleAndApplicationUsage(t *testing.T) {
	h := newUsageHarness(t)

	h.
		EnterIdle().
		AssertLastUsage(
			assertUsageOpened(t),
			assertUsageApplicationName(t, usage.IdleApplicationName),
			assertUsageClassification(t, usage.ClassificationNone),
		).
		AssertUpdateEventsCount(1).
		AssertUsagesCount(1).
		Await(1 * time.Second).
		EnterIdle().
		AssertUpdateEventsCount(1).
		AssertUsagesCount(1)

	h.
		EnterIdle().
		TitleChanged("Chrome", "Github", withPtr("https://github.com")).
		AssertLastUsage(
			assertUsageOpened(t),
			assertUsageApplicationName(t, "Chrome"),
			assertUsageHostname(t, "github.com"),
			assertUsageClassification(t, usage.ClassificationProductive),
		).
		AssertPreviousUsage(assertUsageClosed(t)).
		AssertUpdateEventsCount(4).
		AssertUsagesCount(2)

	h.
		Await(3 * time.Second).
		EnterIdle().
		AssertLastUsage(assertUsageOpened(t)).
		AssertPreviousUsage(assertUsageClosed(t)).
		AssertUpdateEventsCount(6).
		AssertUsagesCount(3)
}

func TestService_ProtectionPauseAndWhitelisting(t *testing.T) {
	h := newUsageHarness(t)

	// user opens amazon, gets blocked by obviously classifier since it's a distraction
	h.
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertUsageClosed(t),
			assertUsageApplicationName(t, "Chrome"),
			assertUsageHostname(t, "amazon.com"),
			assertUsageClassification(t, usage.ClassificationDistracting),
			assertEnforcementAction(t, usage.EnforcementActionBlock),
			assertEnforcementSource(t, usage.EnforcementSourceApplication),
		).
		AssertUpdateEventsCount(2)

	// user pauses all protection and opens amazon and linkedin, should not be blocked
	h.
		ResetUsageEvents().
		Pause(5, "test").
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertUsageApplicationName(t, "Chrome"),
			assertUsageHostname(t, "amazon.com"),
			assertUsageClassification(t, usage.ClassificationDistracting),
			assertEnforcementAction(t, usage.EnforcementActionAllow),
			assertEnforcementSource(t, usage.EnforcementSourcePaused),
		).
		AssertUpdateEventsCount(2).
		TitleChanged("Chrome", "Linkedin", withPtr("https://www.linkedin.com")).
		AssertLastUsage(
			assertUsageApplicationName(t, "Chrome"),
			assertUsageHostname(t, "linkedin.com"),
			assertUsageClassification(t, usage.ClassificationDistracting),
			assertEnforcementAction(t, usage.EnforcementActionAllow),
			assertEnforcementSource(t, usage.EnforcementSourcePaused),
		).
		AssertUpdateEventsCount(5)

	// pause duration has collapsed, so user gets blocked again
	h.
		ResetUsageEvents().
		Await(6*time.Second).
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertPreviousUsage(assertUsageClosed(t)).
		AssertLastUsage(
			assertUsageApplicationName(t, "Chrome"),
			assertUsageHostname(t, "amazon.com"),
			assertUsageClassification(t, usage.ClassificationDistracting),
			assertEnforcementAction(t, usage.EnforcementActionBlock),
			assertEnforcementSource(t, usage.EnforcementSourceApplication),
		).
		AssertUpdateEventsCount(3)

	// user whitelists amazon and opens it again, should not be blocked, linkedin is still blocked
	h.
		Whitelist("Chrome", "https://www.amazon.com", 3*time.Second).
		TitleChanged("Chrome", "Amazon", withPtr("https://amazon.com")).
		AssertLastUsage(
			assertUsageHostname(t, "amazon.com"),
			assertUsageClassification(t, usage.ClassificationDistracting),
			assertEnforcementAction(t, usage.EnforcementActionAllow),
			assertEnforcementSource(t, usage.EnforcementSourceWhitelist),
		).
		TitleChanged("Chrome", "Linkedin", withPtr("https://www.linkedin.com")).
		AssertPreviousUsage(assertUsageClosed(t)).
		AssertLastUsage(
			assertUsageHostname(t, "linkedin.com"),
			assertUsageClassification(t, usage.ClassificationDistracting),
			assertEnforcementAction(t, usage.EnforcementActionBlock),
			assertEnforcementSource(t, usage.EnforcementSourceApplication),
		).
		Await(time.Second*4).
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertUsageHostname(t, "amazon.com"),
			assertUsageClassification(t, usage.ClassificationDistracting),
			assertEnforcementAction(t, usage.EnforcementActionBlock),
			assertEnforcementSource(t, usage.EnforcementSourceApplication),
		)

	// 2. Manual Pause Resumption
	h.
		Pause(10, "test early resume").
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertEnforcementAction(t, usage.EnforcementActionAllow),
			assertEnforcementSource(t, usage.EnforcementSourcePaused),
		).
		Await(time.Second).
		Resume("user clicked resume").
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertEnforcementAction(t, usage.EnforcementActionBlock),
			assertEnforcementSource(t, usage.EnforcementSourceApplication),
		)

	// 3. Whitelist Overwriting / Extension
	h.
		Whitelist("Chrome", "https://www.amazon.com", 2*time.Second).
		Await(time.Second).
		Whitelist("Chrome", "https://www.amazon.com", 4*time.Hour).
		Await(time.Second*3).
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertEnforcementAction(t, usage.EnforcementActionAllow),
			assertEnforcementSource(t, usage.EnforcementSourceWhitelist),
		)

	// 4. Cross-Browser Whitelist
	h.
		TitleChanged("Safari", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertUsageApplicationName(t, "Safari"),
			assertEnforcementAction(t, usage.EnforcementActionAllow),
			assertEnforcementSource(t, usage.EnforcementSourceWhitelist),
		)

	// 5. Manual Whitelist Removal
	h.
		RemoveActiveWhitelists().
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertEnforcementAction(t, usage.EnforcementActionBlock),
			assertEnforcementSource(t, usage.EnforcementSourceApplication),
		)

	// 6. Pause Expiry While Whitelist Is Still Active
	h.
		Await(250*time.Millisecond).
		ResetUsageEvents().
		Pause(3, "pause shorter than whitelist").
		Whitelist("Chrome", "https://www.amazon.com", 7*time.Second).
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertEnforcementAction(t, usage.EnforcementActionAllow),
			assertEnforcementSource(t, usage.EnforcementSourcePaused),
		).
		Await(4*time.Second).
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertEnforcementAction(t, usage.EnforcementActionAllow),
			assertEnforcementSource(t, usage.EnforcementSourceWhitelist),
		)

	// 7. Whitelist Expiry While Pause Is Still Active
	h.
		ResetUsageEvents().
		Pause(8, "pause longer than whitelist").
		Whitelist("Chrome", "https://www.amazon.com", 2*time.Second).
		Await(3*time.Second).
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertEnforcementAction(t, usage.EnforcementActionAllow),
			assertEnforcementSource(t, usage.EnforcementSourcePaused),
		).
		Await(6*time.Second).
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertEnforcementAction(t, usage.EnforcementActionBlock),
			assertEnforcementSource(t, usage.EnforcementSourceApplication),
		)

	// 8. Manual Resume Does Not Clear Active Whitelist
	h.
		Await(250*time.Millisecond).
		ResetUsageEvents().
		Whitelist("Chrome", "https://www.amazon.com", 10*time.Second).
		Pause(10, "manual resume should preserve whitelist").
		Resume("user resumed protection manually").
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertEnforcementAction(t, usage.EnforcementActionAllow),
			assertEnforcementSource(t, usage.EnforcementSourceWhitelist),
		).
		TitleChanged("Chrome", "Linkedin", withPtr("https://www.linkedin.com")).
		AssertLastUsage(
			assertEnforcementAction(t, usage.EnforcementActionBlock),
			assertEnforcementSource(t, usage.EnforcementSourceApplication),
		)

	// 9. Quick-Allow Input Shape Parity (hostname vs full URL)
	h.
		Await(250*time.Millisecond).
		ResetUsageEvents().
		RemoveActiveWhitelists().
		Whitelist("Chrome", "amazon.com", 6*time.Second).
		TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
		AssertLastUsage(
			assertEnforcementAction(t, usage.EnforcementActionAllow),
			assertEnforcementSource(t, usage.EnforcementSourceWhitelist),
		).
		TitleChanged("Chrome", "Amazon", withPtr("https://amazon.com")).
		AssertLastUsage(
			assertEnforcementAction(t, usage.EnforcementActionAllow),
			assertEnforcementSource(t, usage.EnforcementSourceWhitelist),
		)
}

func TestService_Classification(t *testing.T) {
	customRulesOverrideAmazon := `
export function classify(context: Context): Classify | undefined {
	if (context.usage.metadata.domain === "amazon.com") {
		return {
			classification: Classification.Productive,
			classificationReasoning: "Amazon is productive for procurement work",
			tags: ["custom", "procurement"],
		}
	}

	return undefined;
}
`

	t.Run("custom rules override obvious classification for paid tier", func(t *testing.T) {
		h := newUsageHarness(t,
			withAccountTier(apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PLUS),
			withCustomRulesJS(customRulesOverrideAmazon),
		)

		h.
			TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
			AssertLastUsage(
				assertUsageClassification(t, usage.ClassificationProductive),
				assertUsageClassificationSource(t, usage.ClassificationSourceCustomRules),
				assertUsageClassificationReasoning(t, "Amazon is productive for procurement work"),
				assertUsageClassificationConfidence(t, 1),
				assertUsageTags(t, "custom", "procurement"),
			)
	})

	t.Run("obvious classification wins when tier cannot execute custom rules", func(t *testing.T) {
		h := newUsageHarness(t,
			withAccountTier(apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE),
			withCustomRulesJS(customRulesOverrideAmazon),
		)

		h.
			TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
			AssertLastUsage(
				assertUsageClassification(t, usage.ClassificationDistracting),
				assertUsageClassificationSource(t, usage.ClassificationSourceObviously),
				assertClassificationSandboxRecorded(t),
			)
	})

	t.Run("obvious classification applies when custom rules do not match", func(t *testing.T) {
		h := newUsageHarness(t,
			withAccountTier(apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PLUS),
			withCustomRulesJS(`
export function classify(context: Context): Classify | undefined {
	if (context.usage.metadata.domain === "not-amazon.com") {
		return {
			classification: Classification.Productive,
			classificationReasoning: "Unreachable rule",
			tags: ["custom"],
		}
	}

	return undefined;
}
`),
		)

		h.
			TitleChanged("Chrome", "Amazon", withPtr("https://www.amazon.com")).
			AssertLastUsage(
				assertUsageClassification(t, usage.ClassificationDistracting),
				assertUsageClassificationSource(t, usage.ClassificationSourceObviously),
			)
	})

	t.Run("LLM fallback applies when custom and obvious do not classify", func(t *testing.T) {
		h := newUsageHarness(t,
			withAccountTier(apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE),
			withDummyLLMResponse(usage.ClassificationResponse{
				Classification:       usage.ClassificationNeutral,
				ClassificationSource: usage.ClassificationSourceCloudLLMOpenAI,
				Reasoning:            "LLM fallback for unknown website",
				ConfidenceScore:      0.77,
				Tags:                 []string{"llm", "fallback"},
			}),
		)

		h.
			TitleChanged("Chrome", "Unknown", withPtr("https://niche-unknown-example.com/rare/page")).
			AssertLastUsage(
				assertUsageClassification(t, usage.ClassificationNeutral),
				assertUsageClassificationSource(t, usage.ClassificationSourceCloudLLMOpenAI),
				assertUsageClassificationReasoning(t, "LLM fallback for unknown website"),
				assertUsageClassificationConfidence(t, 0.77),
				assertUsageTags(t, "llm", "fallback"),
			)
	})
}
