package timeline

import (
	"time"
)

type Event struct {
	ID         int64  `json:"id" gorm:"primaryKey;autoIncrement;not null"`
	OccurredAt int64  `json:"occurred_at" gorm:"index:idx_timeline_event_occurred_at,not null"`
	Type       string `json:"type" gorm:"index:idx_timeline_event_type,not null"`
	Payload    string `json:"payload" gorm:"not null,default:'{}'"`
	TraceID    string `json:"trace_id" gorm:"index:idx_timeline_event_trace_id"`

	ParentID   *int64  `json:"parent_id" gorm:"index:idx_timeline_event_parent_id"`
	FinishedAt *int64  `json:"ended_at" gorm:"index:idx_timeline_event_ended_at"`
	Key        *string `json:"key" gorm:"index:idx_timeline_event_key"`

	// Tags are optional
	Tags []Tag `json:"tags,omitempty" gorm:"many2many:timeline_event_tag;"`
}

func (e Event) TableName() string {
	return "timeline_event"
}

func NewEvent(eventType string, opts ...EventOption) Event {
	event := Event{
		OccurredAt: time.Now().UTC().Unix(),
		Type:       eventType,
	}

	for _, opt := range opts {
		opt(&event)
	}

	return event
}

func (e Event) TagsSlice() []string {
	tags := make([]string, 0)

	for _, tag := range e.Tags {
		tags = append(tags, tag.Name)
	}

	return tags
}

type Tag struct {
	ID   int64  `json:"id" gorm:"primaryKey;autoIncrement;not null"`
	Name string `json:"name" gorm:"not null;uniqueIndex:idx_timeline_tag_name_type,priority:1"`
	Type string `json:"type" gorm:"not null;index:idx_timeline_tag_type;uniqueIndex:idx_timeline_tag_name_type,priority:2"`

	Events []Event `json:"events,omitempty" gorm:"many2many:timeline_event_tag;"`
}

func NewTag(name, tagType string) Tag {
	return Tag{Name: name, Type: tagType}
}

func (e Tag) TableName() string {
	return "timeline_tag"
}

type EventTag struct {
	EventID int64 `json:"event_id" gorm:"primaryKey;not null"`
	TagID   int64 `json:"tag_id" gorm:"primaryKey;not null"`
}

func (e EventTag) TableName() string {
	return "timeline_event_tag"
}
