package timeline

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_CreateEvent_CallsSubscribers(t *testing.T) {
	h := NewHarness(t)

	var received []*Event
	h.service.On("focus", func(e *Event) {
		received = append(received, e)
	})

	created, err := h.service.CreateEvent("focus", WithPayload(map[string]string{"a": "b"}))
	require.NoError(t, err)

	require.Len(t, received, 1)
	require.NotNil(t, received[0])
	assert.Equal(t, "focus", received[0].Type)
	assert.NotZero(t, received[0].ID, "subscriber should fire after DB commit assigns an ID")
	assert.Equal(t, created.ID, received[0].ID)
}

func TestService_CreateEvent_OnlyCallsMatchingTypeSubscribers(t *testing.T) {
	h := NewHarness(t)

	var focusCalls, breakCalls int
	h.service.On("focus", func(e *Event) { focusCalls++ })
	h.service.On("break", func(e *Event) { breakCalls++ })

	_, err := h.service.CreateEvent("focus")
	require.NoError(t, err)

	assert.Equal(t, 1, focusCalls)
	assert.Equal(t, 0, breakCalls)
}

func TestService_CreateEvent_CallsMultipleSubscribersForSameType(t *testing.T) {
	h := NewHarness(t)

	var order []int
	h.service.On("focus", func(e *Event) { order = append(order, 1) })
	h.service.On("focus", func(e *Event) { order = append(order, 2) })

	_, err := h.service.CreateEvent("focus")
	require.NoError(t, err)

	assert.Equal(t, []int{1, 2}, order)
}

func TestService_UpdateEvent_CallsSubscribers(t *testing.T) {
	h := NewHarness(t)

	created, err := h.service.CreateEvent("focus")
	require.NoError(t, err)

	var received []*Event
	h.service.On("focus", func(e *Event) {
		received = append(received, e)
	})

	require.NoError(t, h.service.UpdateEvent(&created, WithFinishedAt(time.Now())))

	require.Len(t, received, 1)
	require.NotNil(t, received[0])
	assert.Equal(t, "focus", received[0].Type)
	require.NotNil(t, received[0].FinishedAt)
	assert.Equal(t, created.ID, received[0].ID)
}

func TestService_UpdateEvent_FiresForEachCallSite(t *testing.T) {
	h := NewHarness(t)

	var calls int
	h.service.On("focus", func(e *Event) {
		assert.Equal(t, "focus", e.Type)
		calls++
	})

	created, err := h.service.CreateEvent("focus")
	require.NoError(t, err)

	require.NoError(t, h.service.UpdateEvent(&created, WithPayload(map[string]string{"k": "v"})))
	require.NoError(t, h.service.UpdateEvent(&created, WithFinishedAt(time.Now())))

	assert.Equal(t, 3, calls, "expected 1 create + 2 update invocations")
}
