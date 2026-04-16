package timeline

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type EventFilterOption func(*eventFilter)

type eventFilter struct {
	types          []string
	tags           []string
	occurredAtFrom *int64
	occurredAtTo   *int64
	endedAtFrom    *int64
	endedAtTo      *int64
	activeOnly     bool
	limit          int
	offset         int
	orderBy        string
}

func newEventFilter(opts ...EventFilterOption) eventFilter {
	filter := eventFilter{
		orderBy: "occurred_at DESC",
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		opt(&filter)
	}

	return filter
}

func ByTypes(eventTypes ...string) EventFilterOption {
	return func(filter *eventFilter) {
		filter.types = normalizeList(eventTypes)
	}
}

func ByTags(tags ...string) EventFilterOption {
	return func(filter *eventFilter) {
		filter.tags = normalizeList(tags)
	}
}

func ByStartTime(from, to time.Time) EventFilterOption {
	return func(filter *eventFilter) {
		if !from.IsZero() {
			ts := from.UTC().Unix()
			filter.occurredAtFrom = &ts
		}

		if !to.IsZero() {
			ts := to.UTC().Unix()
			filter.occurredAtTo = &ts
		}
	}
}

func ByEndTime(from, to time.Time) EventFilterOption {
	return func(filter *eventFilter) {
		if !from.IsZero() {
			ts := from.UTC().Unix()
			filter.endedAtFrom = &ts
		}

		if !to.IsZero() {
			ts := to.UTC().Unix()
			filter.endedAtTo = &ts
		}
	}
}

func ByAge(days int) EventFilterOption {
	return func(filter *eventFilter) {
		if days >= 0 {
			ts := time.Now().UTC().AddDate(0, 0, -days).Unix()
			filter.occurredAtFrom = &ts
		}
	}
}

func ActiveOnly() EventFilterOption {
	return func(filter *eventFilter) {
		filter.activeOnly = true
	}
}

func Limit(limit int) EventFilterOption {
	return func(filter *eventFilter) {
		if limit > 0 {
			filter.limit = limit
		}
	}
}

func Offset(offset int) EventFilterOption {
	return func(filter *eventFilter) {
		if offset >= 0 {
			filter.offset = offset
		}
	}
}

func OrderByOccurredAtAsc() EventFilterOption {
	return func(filter *eventFilter) {
		filter.orderBy = "occurred_at ASC"
	}
}

func OrderByOccurredAtDesc() EventFilterOption {
	return func(filter *eventFilter) {
		filter.orderBy = "occurred_at DESC"
	}
}

func applyEventFilter(query *gorm.DB, filterOpts ...EventFilterOption) *gorm.DB {
	filter := newEventFilter(filterOpts...)
	eventTable := modelTableName(query, &Event{})
	tagTable := modelTableName(query, &Tag{})

	if eventTable == "" {
		eventTable = "events"
	}

	if tagTable == "" {
		tagTable = "tags"
	}

	eventTypeColumn := eventTable + ".type"
	occurredAtColumn := eventTable + ".occurred_at"
	endedAtColumn := eventTable + ".finished_at"

	if len(filter.types) > 0 {
		query = query.Where(eventTypeColumn+" IN ?", filter.types)
	}

	if filter.occurredAtFrom != nil {
		query = query.Where(occurredAtColumn+" >= ?", *filter.occurredAtFrom)
	}

	if filter.occurredAtTo != nil {
		query = query.Where(occurredAtColumn+" <= ?", *filter.occurredAtTo)
	}

	if filter.endedAtFrom != nil {
		query = query.Where(endedAtColumn+" IS NOT NULL").Where(endedAtColumn+" >= ?", *filter.endedAtFrom)
	}

	if filter.endedAtTo != nil {
		query = query.Where(endedAtColumn+" IS NOT NULL").Where(endedAtColumn+" <= ?", *filter.endedAtTo)
	}

	if filter.activeOnly {
		query = query.Where("("+endedAtColumn+" IS NULL OR "+endedAtColumn+" > ?)", time.Now().UTC().Unix())
	}

	if len(filter.tags) > 0 {
		tagJoin := fmt.Sprintf("JOIN timeline_event_tag ON timeline_event_tag.event_id = %s.id", eventTable)
		tagTableJoin := fmt.Sprintf("JOIN %s ON %s.id = timeline_event_tag.tag_id", tagTable, tagTable)
		tagNameColumn := tagTable + ".name"

		query = query.Joins(tagJoin).
			Joins(tagTableJoin).
			Where(tagNameColumn+" IN ?", filter.tags).
			Distinct()
	}

	if filter.orderBy != "" {
		query = query.Order(normalizeOrderBy(filter.orderBy, eventTable))
	}

	if filter.limit > 0 {
		query = query.Limit(filter.limit)
	}

	if filter.offset > 0 {
		query = query.Offset(filter.offset)
	}

	return query
}

func modelTableName(db *gorm.DB, model any) string {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return ""
	}

	if stmt.Schema == nil {
		return ""
	}

	return stmt.Schema.Table
}

func normalizeOrderBy(orderBy, eventTable string) string {
	value := strings.TrimSpace(strings.ToUpper(orderBy))
	switch value {
	case "OCCURRED_AT ASC":
		return eventTable + ".occurred_at ASC"
	case "OCCURRED_AT DESC":
		return eventTable + ".occurred_at DESC"
	default:
		return eventTable + ".occurred_at DESC"
	}
}

func normalizeList(input []string) []string {
	if len(input) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))

	for _, value := range input {
		n := strings.TrimSpace(value)
		if n == "" {
			continue
		}

		if _, ok := seen[n]; ok {
			continue
		}

		seen[n] = struct{}{}
		normalized = append(normalized, n)
	}

	if len(normalized) == 0 {
		return nil
	}

	return normalized
}
