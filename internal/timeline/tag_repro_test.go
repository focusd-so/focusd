package timeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression test: UpdateEvent must look up each tag by Name+Type, not just
// call FirstOrCreate with no Where clause. Previously every tag pointer was
// overwritten with the first row in timeline_tag, collapsing many2many inserts
// to a single join row.
func TestService_UpdateEvent_PersistsMultipleTags(t *testing.T) {
	h := NewHarness(t)

	created, err := h.service.CreateEvent("focus")
	require.NoError(t, err)

	created.Tags = append(created.Tags,
		NewTag("coding", "classification_tag"),
		NewTag("terminal", "classification_tag"),
		NewTag("productive", "classification"),
	)

	require.NoError(t, h.service.UpdateEvent(&created))

	events := h.ListEvents()
	require.Len(t, events, 1)

	assert.ElementsMatch(t,
		[]string{"coding", "terminal", "productive"},
		events[0].TagsSlice(),
	)
}
