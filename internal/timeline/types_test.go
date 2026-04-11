package timeline_test

import (
	"testing"

	"github.com/focusd-so/focusd/internal/timeline"
	"github.com/stretchr/testify/assert"
)

func TestWithTags_NormalizesAndAppends(t *testing.T) {

	event := timeline.NewEvent(
		"test",
		timeline.WithTags(timeline.NewTag("work", "default"), timeline.NewTag("deep", "default")),
	)

	assert.Equal(t, []string{"work", "deep"}, event.TagsSlice())
}
