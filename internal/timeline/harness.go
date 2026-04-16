package timeline

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type timelineHarness struct {
	t       *testing.T
	service *Service
	db      *gorm.DB
}

func NewHarness(t *testing.T) *timelineHarness {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(memoryDSNForTimelineHarness(t)), &gorm.Config{})
	require.NoError(t, err)

	service, err := NewService(db)
	require.NoError(t, err)

	return &timelineHarness{
		t:       t,
		service: service,
		db:      db,
	}
}

func memoryDSNForTimelineHarness(t *testing.T) string {
	t.Helper()

	return fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", url.QueryEscape(t.Name()))
}

func (h *timelineHarness) ListEvents(opts ...EventFilterOption) []*Event {
	h.t.Helper()

	var (
		events []*Event
		err    error
	)

	h.retryLocked(func() error {
		events, err = h.service.ListEvents(opts...)
		return err
	})

	return events
}

func (h *timelineHarness) LastEvent(opts ...EventFilterOption) *Event {
	h.t.Helper()

	queryOpts := make([]EventFilterOption, 0, len(opts)+2)
	queryOpts = append(queryOpts, opts...)
	queryOpts = append(queryOpts, OrderByOccurredAtDesc(), Limit(1))

	events := h.ListEvents(queryOpts...)
	if len(events) == 0 {
		return nil
	}

	return events[0]
}

func (h *timelineHarness) AssertEventsCount(expected int, opts ...EventFilterOption) *timelineHarness {
	h.t.Helper()

	require.Len(h.t, h.ListEvents(opts...), expected)
	return h
}

func (h *timelineHarness) AssertLastEventByFilters(filterOpts []EventFilterOption, check ...func(*Event)) *timelineHarness {
	h.t.Helper()

	event := h.LastEvent(filterOpts...)
	for _, c := range check {
		c(event)
	}

	return h
}

func (h *timelineHarness) retryLocked(fn func() error) {
	h.t.Helper()

	deadline := time.Now().Add(1500 * time.Millisecond)
	for {
		err := fn()
		if err == nil {
			return
		}

		if !strings.Contains(err.Error(), "database table is locked") {
			require.NoError(h.t, err)
		}

		if time.Now().After(deadline) {
			require.NoError(h.t, err)
		}

		time.Sleep(20 * time.Millisecond)
	}
}

func AssertEventType(t *testing.T, expected string) func(*Event) {
	t.Helper()

	return func(e *Event) {
		require.NotNil(t, e)
		require.Equal(t, expected, e.Type)
	}
}

func AssertEventTags(t *testing.T, expected ...string) func(*Event) {
	t.Helper()

	return func(e *Event) {
		require.NotNil(t, e)
		require.ElementsMatch(t, expected, e.TagsSlice())
	}
}

func AssertEventOccurredBetween(t *testing.T, from, to time.Time) func(*Event) {
	t.Helper()

	return func(e *Event) {
		require.NotNil(t, e)
		occurredAt := time.Unix(e.OccurredAt, 0).UTC()
		require.False(t, occurredAt.Before(from.UTC()))
		require.False(t, occurredAt.After(to.UTC()))
	}
}

func AssertNoEvent(t *testing.T) func(*Event) {
	t.Helper()

	return func(e *Event) {
		require.Nil(t, e)
	}
}
