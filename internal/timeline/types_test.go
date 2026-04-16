package timeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithTags_NormalizesAndAppends(t *testing.T) {

	event := NewEvent(
		"test",
		WithTags(NewTag("work", "default"), NewTag("deep", "default")),
	)

	assert.Equal(t, []string{"work", "deep"}, event.TagsSlice())
}
