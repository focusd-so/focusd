package timeline

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB

	onEvent map[string][]func(*Event)
}

func NewService(db *gorm.DB) (*Service, error) {
	if err := db.AutoMigrate(&Event{}, &Tag{}, &EventTag{}); err != nil {
		return nil, err
	}

	return &Service{
		db:      db,
		onEvent: make(map[string][]func(*Event)),
	}, nil
}

func (s Service) CreateEvent(eventType string, opts ...EventOption) (Event, error) {
	event := Event{
		OccurredAt: time.Now().UTC().Unix(),
		Type:       eventType,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if err := opt(&event); err != nil {
			return event, fmt.Errorf("failed to apply event option: %w", err)
		}
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return event, fmt.Errorf("failed to start create event transaction: %w", tx.Error)
	}

	if len(event.Tags) > 0 {
		for i := range event.Tags {
			tag := &event.Tags[i]
			lookup := Tag{Name: tag.Name, Type: tag.Type}
			if err := tx.Where(&lookup).FirstOrCreate(tag).Error; err != nil {
				tx.Rollback()
				return event, fmt.Errorf("failed to upsert tag %q (%q): %w", tag.Name, tag.Type, err)
			}
		}
	}

	if err := tx.Create(&event).Error; err != nil {
		tx.Rollback()
		return event, fmt.Errorf("failed to create event: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return event, fmt.Errorf("failed to commit create event transaction: %w", err)
	}

	for _, sub := range s.onEvent[eventType] {
		sub(&event)
	}

	return event, nil
}

func (s Service) EventFinished(event *Event) error {
	return s.UpdateEvent(event, WithFinishedAt(time.Now()))
}

func (s Service) UpdateEvent(event *Event, opts ...EventOption) error {
	for _, opt := range opts {
		opt(event)
	}

	for i := range event.Tags {
		tag := &event.Tags[i]
		lookup := Tag{Name: tag.Name, Type: tag.Type}
		if err := s.db.Where(&lookup).FirstOrCreate(tag).Error; err != nil {
			return fmt.Errorf("failed to upsert tag %q (%q): %w", tag.Name, tag.Type, err)
		}
	}

	if err := s.db.Save(event).Error; err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	for _, sub := range s.onEvent[event.Type] {
		sub(event)
	}

	return nil
}

func (s Service) ListEvents(opts ...EventFilterOption) ([]*Event, error) {
	query := s.db.Model(&Event{}).Preload("Tags")
	query = applyEventFilter(query, opts...)

	var events []*Event
	if err := query.Find(&events).Error; err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	return events, nil
}

func (s Service) GetActiveEventOfTypes(eventTypes ...string) (*Event, error) {
	events, err := s.ListEvents(ByTypes(eventTypes...), ActiveOnly())
	if err != nil {
		return nil, fmt.Errorf("failed to get active event of types %q: %w", eventTypes, err)
	}

	if len(events) == 0 {
		return nil, nil
	}

	return events[0], nil
}

func (s Service) GetActiveEventOfType(eventType string) (*Event, error) {
	return s.GetActiveEventOfTypes(eventType)
}

func (s Service) LastEventOfTypes(eventTypes ...string) (*Event, error) {
	events, err := s.ListEvents(ByTypes(eventTypes...), ActiveOnly(), Limit(1))
	if err != nil {
		return nil, fmt.Errorf("failed to get last event of types %q: %w", eventTypes, err)
	}

	if len(events) == 0 {
		return nil, nil
	}

	return events[0], nil
}

func (s Service) LastEventOfType(eventType string) (*Event, error) {
	return s.LastEventOfTypes(eventType)
}

func (s Service) On(eventType string, fn func(event *Event)) {
	s.onEvent[eventType] = append(s.onEvent[eventType], fn)
}
