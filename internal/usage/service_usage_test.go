package usage_test

import (
	"context"
	"sync"
	"testing"
	"time"

	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
	"github.com/focusd-so/focusd/internal/sandbox"
	"github.com/focusd-so/focusd/internal/timeline"
	"github.com/focusd-so/focusd/internal/usage"
	"github.com/stretchr/testify/require"
)

var registerUsageSandboxContributorOnce sync.Once

func TestIdleChanged_Transitions(t *testing.T) {
	h := newHarness(t)

	t.Run("user enters idle after being active", func(t *testing.T) {
		h.TitleChanged("Google Chrome", "Google", new("https://www.google.com")).
			AssertApplicationCount(1).
			AssertApplicationExists("google.com").
			AssertLastActiveEvent([]string{usage.EventTypeUsageChanged, usage.EventTypeUserIdle}, func(e *timeline.Event) {
				require.Equal(t, usage.EventTypeUsageChanged, e.Type)
				require.Nil(t, e.FinishedAt)
			}).
			Await(1*time.Second).
			EnterIdle().
			AssertPreviousEvent([]string{usage.EventTypeUsageChanged, usage.EventTypeUserIdle}, func(e *timeline.Event) {
				require.Equal(t, usage.EventTypeUsageChanged, e.Type)
				require.NotNil(t, e.FinishedAt)
			}).
			AssertLastActiveEvent([]string{usage.EventTypeUsageChanged, usage.EventTypeUserIdle}, func(e *timeline.Event) {
				require.Equal(t, usage.EventTypeUserIdle, e.Type)
				require.Nil(t, e.FinishedAt)
			}).
			Await(1*time.Second).
			TitleChanged("Google Chrome", "Google", new("https://www.linkedin.com")).
			AssertApplicationExists("linkedin.com").
			AssertPreviousEvent([]string{usage.EventTypeUserIdle, usage.EventTypeUsageChanged}, func(e *timeline.Event) {
				require.Equal(t, usage.EventTypeUserIdle, e.Type)
				require.NotNil(t, e.FinishedAt)
			}).
			AssertLastActiveEvent([]string{usage.EventTypeUsageChanged, usage.EventTypeUserIdle}, func(e *timeline.Event) {
				require.Equal(t, usage.EventTypeUsageChanged, e.Type)
				require.Nil(t, e.FinishedAt)
			})
	})

	t.Run("user idle event triggered while user was idle, check idempotency", func(t *testing.T) {
		h.TitleChanged("Google Chrome", "Google", new("https://www.google.com")).
			Await(1*time.Second).
			EnterIdle().
			AssertLastActiveEvent([]string{usage.EventTypeUserIdle, usage.EventTypeUsageChanged}, func(e *timeline.Event) {
				require.Equal(t, usage.EventTypeUserIdle, e.Type)
				require.Nil(t, e.FinishedAt)
			}).
			EnterIdle().
			AssertLastActiveEvent([]string{usage.EventTypeUserIdle, usage.EventTypeUsageChanged}, func(e *timeline.Event) {
				require.Equal(t, usage.EventTypeUserIdle, e.Type)
				require.Nil(t, e.FinishedAt)
			}).
			AssertPreviousEvent([]string{usage.EventTypeUserIdle, usage.EventTypeUsageChanged}, func(e *timeline.Event) {
				require.Equal(t, usage.EventTypeUsageChanged, e.Type)
				require.NotNil(t, e.FinishedAt)
			})
	})

	t.Run("user idle event triggered while no active event, check no-op", func(t *testing.T) {
		h := newHarness(t)
		h.TitleChanged("Google Chrome", "Google", new("https://www.google.com"))

		active, err := h.timelineService.GetActiveEventOfTypes(
			[]string{usage.EventTypeUsageChanged, usage.EventTypeUserIdle},
		)
		require.NoError(t, err)
		require.NotNil(t, active)
		require.NoError(t, h.timelineService.EventFinished(active))

		before, err := h.timelineService.ListEvents(timeline.ByTypes(usage.EventTypeUserIdle))
		require.NoError(t, err)

		err = h.service.IdleChanged(context.Background(), true)
		require.NoError(t, err)

		after, err := h.timelineService.ListEvents(timeline.ByTypes(usage.EventTypeUserIdle))
		require.NoError(t, err)
		require.Len(t, after, len(before)) // no new idle event
	})
}

func TestTitleChanged_ClassificationCustomRules(t *testing.T) {
	customRules := `import { runtime, Timezone, productive, distracting, type Classify } from "@focusd/runtime";

export function classify(): Classify | undefined {
	const now = runtime.time.now(Timezone.UTC);
	if (!now) {
		return undefined;
	}

	if (!runtime.usage.url && runtime.usage.app === "Slack") {
		return productive("native app rule matched", ["native-app", "time-checked"]);
	}

	if (runtime.usage.url && runtime.usage.host === "youtube.com") {
		return distracting("website rule matched", ["website", "time-checked"]);
	}

	return undefined;
}
`

	t.Run("native app uses appName in custom rules", func(t *testing.T) {
		h := newHarness(
			t,
			withAccountTier(apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO),
			withCustomRulesJS(customRules),
		)
		registerUsageSandboxContributorOnce.Do(func() {
			sandbox.Register(usage.NewUsageContributor(h.service))
		})

		h.TitleChanged("Slack", "Acme | #engineering", nil).
			AssertApplicationExists("Slack")

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)

		require.Equal(t, usage.ClassificationProductive, payload.Classification)
		require.Equal(t, usage.ClassificationSourceCustomRules, payload.ClassificationSource)
		require.NotNil(t, payload.ClassificationResult)
		require.NotNil(t, payload.ClassificationResult.CustomRulesClassificationResult)
		require.Equal(t, []string{"native-app", "time-checked"}, payload.ClassificationResult.CustomRulesClassificationResult.Tags)
		require.Equal(t, "native app rule matched", payload.ClassificationResult.CustomRulesClassificationResult.ClassificationReason)
	})

	t.Run("browser url uses host in custom rules", func(t *testing.T) {
		h := newHarness(
			t,
			withAccountTier(apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO),
			withCustomRulesJS(customRules),
		)
		registerUsageSandboxContributorOnce.Do(func() {
			sandbox.Register(usage.NewUsageContributor(h.service))
		})

		h.TitleChanged("Google Chrome", "YouTube", new("https://youtube.com/watch?v=abc")).
			AssertApplicationExists("youtube.com")

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)

		require.Equal(t, usage.ClassificationDistracting, payload.Classification)
		require.Equal(t, usage.ClassificationSourceCustomRules, payload.ClassificationSource)
		require.NotNil(t, payload.ClassificationResult)
		require.NotNil(t, payload.ClassificationResult.CustomRulesClassificationResult)
		require.Equal(t, []string{"website", "time-checked"}, payload.ClassificationResult.CustomRulesClassificationResult.Tags)
		require.Equal(t, "website rule matched", payload.ClassificationResult.CustomRulesClassificationResult.ClassificationReason)
	})

	t.Run("falls back to obviously when custom rule does not match", func(t *testing.T) {
		h := newHarness(
			t,
			withAccountTier(apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO),
			withCustomRulesJS(customRules),
		)
		registerUsageSandboxContributorOnce.Do(func() {
			sandbox.Register(usage.NewUsageContributor(h.service))
		})

		h.TitleChanged("Google Chrome", "Facebook", new("https://www.facebook.com/")).
			AssertApplicationExists("facebook.com")

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)

		require.Equal(t, usage.ClassificationDistracting, payload.Classification)
		require.Equal(t, usage.ClassificationSourceObviously, payload.ClassificationSource)
		require.NotNil(t, payload.ClassificationResult)
		require.Nil(t, payload.ClassificationResult.CustomRulesClassificationResult)
		require.NotNil(t, payload.ClassificationResult.ObviouslyClassificationResult)
	})

	t.Run("custom rules ignored for free tier", func(t *testing.T) {
		customRules := `
import { runtime, productive, type Classify } from "@focusd/runtime";
export function classify(): Classify | undefined {
  if (runtime.usage.host === "facebook.com") {
    return productive("custom says productive", ["custom"]);
  }
  return undefined;
}
`
		h := newHarness(
			t,
			withAccountTier(apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE),
			withCustomRulesJS(customRules),
		)
		registerUsageSandboxContributorOnce.Do(func() {
			sandbox.Register(usage.NewUsageContributor(h.service))
		})

		h.TitleChanged("Google Chrome", "Facebook", new("https://www.facebook.com/")).
			AssertApplicationExists("facebook.com")

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)

		// Effective result should ignore custom rules for free tier.
		require.Equal(t, usage.ClassificationDistracting, payload.Classification)
		require.Equal(t, usage.ClassificationSourceObviously, payload.ClassificationSource)
	})
}

func TestTitleChanged_EnforcementCalculation(t *testing.T) {
	t.Run("distracting classification results in standard block enforcement", func(t *testing.T) {
		h := newHarness(t)

		h.TitleChanged("Google Chrome", "Facebook", new("https://www.facebook.com/")).
			AssertApplicationExists("facebook.com")

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)

		require.NotNil(t, payload.EnforcementResult.StandardEnforcementResult)
		require.Equal(t, usage.EnforcementActionBlock, payload.EnforcementResult.StandardEnforcementResult.Action)
		require.Equal(t, usage.EnforcementSourceApplication, payload.EnforcementResult.StandardEnforcementResult.Source)
		require.Equal(t, "distracting usage, focus protection is enabled", payload.EnforcementResult.StandardEnforcementResult.Reason)
		require.Nil(t, payload.EnforcementResult.CustomRulesEnforcementResult)
	})

	t.Run("non distracting classification results in standard allow enforcement", func(t *testing.T) {
		h := newHarness(t)

		h.TitleChanged("Google Chrome", "Google", new("https://www.google.com")).
			AssertApplicationExists("google.com")

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)

		require.NotNil(t, payload.EnforcementResult.StandardEnforcementResult)
		require.Equal(t, usage.EnforcementActionAllow, payload.EnforcementResult.StandardEnforcementResult.Action)
		require.Equal(t, usage.EnforcementSourceApplication, payload.EnforcementResult.StandardEnforcementResult.Source)
		require.Equal(t, "non distracting usage", payload.EnforcementResult.StandardEnforcementResult.Reason)
		require.Nil(t, payload.EnforcementResult.CustomRulesEnforcementResult)
	})

	t.Run("AllowApp takes precedence over custom rules and standard enforcement", func(t *testing.T) {
		customRules := `import { block, type Enforce } from "@focusd/runtime";
export function enforcement(): Enforce | undefined {
  return block("custom enforcement block");
}
`

		h := newHarness(
			t,
			withAccountTier(apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO),
			withCustomRulesJS(customRules),
		)
		registerUsageSandboxContributorOnce.Do(func() {
			sandbox.Register(usage.NewUsageContributor(h.service))
		})

		h.AllowApp("Google Chrome", time.Minute).
			TitleChanged("Google Chrome", "Facebook", new("https://www.facebook.com/")).
			AssertApplicationExists("facebook.com")

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)

		require.NotNil(t, payload.EnforcementResult.StandardEnforcementResult)
		require.Equal(t, usage.EnforcementActionAllow, payload.EnforcementResult.StandardEnforcementResult.Action)
		require.Equal(t, usage.EnforcementSourceAllowed, payload.EnforcementResult.StandardEnforcementResult.Source)
		require.Equal(t, "temporarily allowed usage by user", payload.EnforcementResult.StandardEnforcementResult.Reason)
		require.Nil(t, payload.EnforcementResult.CustomRulesEnforcementResult)
	})

	t.Run("AllowWebsite takes precedence over custom rules and standard enforcement", func(t *testing.T) {
		customRules := `import { block, type Enforce } from "@focusd/runtime";
export function enforcement(): Enforce | undefined {
  return block("custom enforcement block");
}
`

		h := newHarness(
			t,
			withAccountTier(apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO),
			withCustomRulesJS(customRules),
		)
		registerUsageSandboxContributorOnce.Do(func() {
			sandbox.Register(usage.NewUsageContributor(h.service))
		})

		h.AllowWebsite("https://www.facebook.com/", time.Minute).
			TitleChanged("Google Chrome", "Facebook", new("https://www.facebook.com/")).
			AssertApplicationExists("facebook.com")

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)

		require.NotNil(t, payload.EnforcementResult.StandardEnforcementResult)
		require.Equal(t, usage.EnforcementActionAllow, payload.EnforcementResult.StandardEnforcementResult.Action)
		require.Equal(t, usage.EnforcementSourceAllowed, payload.EnforcementResult.StandardEnforcementResult.Source)
		require.Equal(t, "temporarily allowed usage by user", payload.EnforcementResult.StandardEnforcementResult.Reason)
		require.Nil(t, payload.EnforcementResult.CustomRulesEnforcementResult)
	})

	t.Run("AllowURL takes precedence over custom rules and standard enforcement", func(t *testing.T) {
		customRules := `import { block, type Enforce } from "@focusd/runtime";
export function enforcement(): Enforce | undefined {
  return block("custom enforcement block");
}
`

		h := newHarness(
			t,
			withAccountTier(apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO),
			withCustomRulesJS(customRules),
		)
		registerUsageSandboxContributorOnce.Do(func() {
			sandbox.Register(usage.NewUsageContributor(h.service))
		})

		h.AllowURL("https://www.facebook.com/", time.Minute).
			TitleChanged("Google Chrome", "Facebook", new("https://www.facebook.com/")).
			AssertApplicationExists("facebook.com")

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)

		require.NotNil(t, payload.EnforcementResult.StandardEnforcementResult)
		require.Equal(t, usage.EnforcementActionAllow, payload.EnforcementResult.StandardEnforcementResult.Action)
		require.Equal(t, usage.EnforcementSourceAllowed, payload.EnforcementResult.StandardEnforcementResult.Source)
		require.Equal(t, "temporarily allowed usage by user", payload.EnforcementResult.StandardEnforcementResult.Reason)
		require.Nil(t, payload.EnforcementResult.CustomRulesEnforcementResult)
	})

	t.Run("PauseProtection takes precedence over allow overrides, custom rules, and standard enforcement", func(t *testing.T) {
		customRules := `import { block, type Enforce } from "@focusd/runtime";
export function enforcement(): Enforce | undefined {
  return block("custom enforcement block");
}
`

		h := newHarness(
			t,
			withAccountTier(apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO),
			withCustomRulesJS(customRules),
		)
		registerUsageSandboxContributorOnce.Do(func() {
			sandbox.Register(usage.NewUsageContributor(h.service))
		})

		h.
			Pause(60, "user manually paused focus protection").
			TitleChanged("Google Chrome", "Facebook", new("https://www.facebook.com/")).
			AssertApplicationExists("facebook.com")

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)

		require.NotNil(t, payload.EnforcementResult.StandardEnforcementResult)
		require.Equal(t, usage.EnforcementActionPaused, payload.EnforcementResult.StandardEnforcementResult.Action)
		require.Equal(t, usage.EnforcementSourcePaused, payload.EnforcementResult.StandardEnforcementResult.Source)
		require.Equal(t, "focus protection is temporarily paused by user", payload.EnforcementResult.StandardEnforcementResult.Reason)
		require.Nil(t, payload.EnforcementResult.CustomRulesEnforcementResult)
	})

	t.Run("paused enforcement expires and standard enforcement resumes", func(t *testing.T) {
		h := newHarness(t)

		h.Pause(1, "brief pause for test").
			TitleChanged("Google Chrome", "Facebook", new("https://www.facebook.com/")).
			AssertApplicationExists("facebook.com")

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)
		require.NotNil(t, payload.EnforcementResult.StandardEnforcementResult)
		require.Equal(t, usage.EnforcementActionPaused, payload.EnforcementResult.StandardEnforcementResult.Action)
		require.Equal(t, usage.EnforcementSourcePaused, payload.EnforcementResult.StandardEnforcementResult.Source)

		h.Await(2*time.Second).
			TitleChanged("Google Chrome", "Facebook", new("https://www.facebook.com/"))

		events, err = h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err = timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)
		require.NotNil(t, payload.EnforcementResult.StandardEnforcementResult)
		require.Equal(t, usage.EnforcementActionBlock, payload.EnforcementResult.StandardEnforcementResult.Action)
		require.Equal(t, usage.EnforcementSourceApplication, payload.EnforcementResult.StandardEnforcementResult.Source)
		require.Equal(t, "distracting usage, focus protection is enabled", payload.EnforcementResult.StandardEnforcementResult.Reason)
	})

	t.Run("allowed app enforcement expires and standard enforcement resumes", func(t *testing.T) {
		h := newHarness(t)

		h.AllowApp("Google Chrome", time.Second).
			TitleChanged("Google Chrome", "Facebook", new("https://www.facebook.com/")).
			AssertApplicationExists("facebook.com")

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)
		require.NotNil(t, payload.EnforcementResult.StandardEnforcementResult)
		require.Equal(t, usage.EnforcementActionAllow, payload.EnforcementResult.StandardEnforcementResult.Action)
		require.Equal(t, usage.EnforcementSourceAllowed, payload.EnforcementResult.StandardEnforcementResult.Source)

		h.Await(2*time.Second).
			TitleChanged("Google Chrome", "Facebook", new("https://www.facebook.com/"))

		events, err = h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err = timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)
		require.NotNil(t, payload.EnforcementResult.StandardEnforcementResult)
		require.Equal(t, usage.EnforcementActionBlock, payload.EnforcementResult.StandardEnforcementResult.Action)
		require.Equal(t, usage.EnforcementSourceApplication, payload.EnforcementResult.StandardEnforcementResult.Source)
		require.Equal(t, "distracting usage, focus protection is enabled", payload.EnforcementResult.StandardEnforcementResult.Reason)
	})

	t.Run("manual resume ends paused enforcement and standard enforcement resumes", func(t *testing.T) {
		h := newHarness(t)

		h.Pause(60, "long pause for test").
			TitleChanged("Google Chrome", "Facebook", new("https://www.facebook.com/")).
			AssertApplicationExists("facebook.com")

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)
		require.NotNil(t, payload.EnforcementResult.StandardEnforcementResult)
		require.Equal(t, usage.EnforcementActionPaused, payload.EnforcementResult.StandardEnforcementResult.Action)

		h.Resume("resume for test").
			Await(1100*time.Millisecond).
			TitleChanged("Google Chrome", "Facebook", new("https://www.facebook.com/"))

		events, err = h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err = timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)
		require.NotNil(t, payload.EnforcementResult.StandardEnforcementResult)
		require.Equal(t, usage.EnforcementActionBlock, payload.EnforcementResult.StandardEnforcementResult.Action)
		require.Equal(t, usage.EnforcementSourceApplication, payload.EnforcementResult.StandardEnforcementResult.Source)
	})

	t.Run("manual allow removal ends allowed enforcement and standard enforcement resumes", func(t *testing.T) {
		h := newHarness(t)

		h.AllowApp("Google Chrome", time.Minute).
			TitleChanged("Google Chrome", "Facebook", new("https://www.facebook.com/")).
			AssertApplicationExists("facebook.com")

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)
		require.NotNil(t, payload.EnforcementResult.StandardEnforcementResult)
		require.Equal(t, usage.EnforcementActionAllow, payload.EnforcementResult.StandardEnforcementResult.Action)
		require.Equal(t, usage.EnforcementSourceAllowed, payload.EnforcementResult.StandardEnforcementResult.Source)

		allowEvents, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeAllowUsage),
			timeline.ActiveOnly(),
		)
		require.NoError(t, err)
		require.Len(t, allowEvents, 1)

		err = h.service.AllowRemove(allowEvents[0].ID)
		require.NoError(t, err)

		h.Await(1100 * time.Millisecond)

		h.TitleChanged("Google Chrome", "Facebook", new("https://www.facebook.com/"))

		events, err = h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err = timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)
		require.NotNil(t, payload.EnforcementResult.StandardEnforcementResult)
		require.Equal(t, usage.EnforcementActionBlock, payload.EnforcementResult.StandardEnforcementResult.Action)
		require.Equal(t, usage.EnforcementSourceApplication, payload.EnforcementResult.StandardEnforcementResult.Source)
	})

	t.Run("custom rules enforcement decision is persisted", func(t *testing.T) {
		customRules := `import { runtime, distracting, block, type Classify, type Enforce } from "@focusd/runtime";

export function classify(): Classify | undefined {
	if (runtime.usage.host === "youtube.com") {
		return distracting("custom distracting rule", ["website"]);
	}

	return undefined;
}

export function enforcement(): Enforce | undefined {
	if (runtime.usage.classification === "distracting") {
		return { enforcementAction: "block", enforcementReason: "custom enforcement block" };
	}

	return block("custom fallback block");
}
`

		h := newHarness(
			t,
			withAccountTier(apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO),
			withCustomRulesJS(customRules),
		)
		registerUsageSandboxContributorOnce.Do(func() {
			sandbox.Register(usage.NewUsageContributor(h.service))
		})

		h.TitleChanged("Google Chrome", "YouTube", new("https://youtube.com/watch?v=abc")).
			AssertApplicationExists("youtube.com")

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)

		require.NotNil(t, payload.EnforcementResult.StandardEnforcementResult)
		require.Equal(t, usage.EnforcementActionBlock, payload.EnforcementResult.StandardEnforcementResult.Action)
		require.Equal(t, usage.EnforcementSourceApplication, payload.EnforcementResult.StandardEnforcementResult.Source)

		require.NotNil(t, payload.EnforcementResult.CustomRulesEnforcementResult)
		require.Equal(t, usage.EnforcementActionBlock, payload.EnforcementResult.CustomRulesEnforcementResult.Action)
		require.Equal(t, usage.EnforcementSourceCustomRules, payload.EnforcementResult.CustomRulesEnforcementResult.Source)
		require.Equal(t, "custom enforcement block", payload.EnforcementResult.CustomRulesEnforcementResult.Reason)
	})

	t.Run("custom enforcement helper block shape", func(t *testing.T) {
		customRules := `import { block, type Enforce } from "@focusd/runtime";
export function enforcement(): Enforce | undefined {
  return block("helper says block");
}
`
		h := newHarness(
			t,
			withAccountTier(apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO),
			withCustomRulesJS(customRules),
		)
		registerUsageSandboxContributorOnce.Do(func() {
			sandbox.Register(usage.NewUsageContributor(h.service))
		})

		// Non-distracting by default, so standard would likely allow.
		h.TitleChanged("Google Chrome", "Google", new("https://www.google.com"))

		events, err := h.timelineService.ListEvents(
			timeline.ByTypes(usage.EventTypeUsageChanged),
			timeline.Limit(1),
			timeline.OrderByOccurredAtDesc(),
		)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, err := timeline.UnmarshalPayload[usage.ApplicationUsagePayload](events[0])
		require.NoError(t, err)

		require.NotNil(t, payload.EnforcementResult.CustomRulesEnforcementResult)
		require.Equal(t, usage.EnforcementActionBlock, payload.EnforcementResult.CustomRulesEnforcementResult.Action)
		require.Equal(t, "helper says block", payload.EnforcementResult.CustomRulesEnforcementResult.Reason)
	})
}
