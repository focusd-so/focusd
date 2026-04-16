package usage_test

import (
	"testing"
	"time"

	"github.com/focusd-so/focusd/internal/timeline"
	"github.com/focusd-so/focusd/internal/usage"
	"github.com/stretchr/testify/require"
)

func TestIdleChanged_Transitions(t *testing.T) {
	h := newHarness(t)

	t.Run("user enters idle after usage changed", func(t *testing.T) {
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
			})
	})
}
