package timeline

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
	mu *sync.Mutex

	onCreated map[string][]func(*Event)
	onUpdated map[string][]func(*Event)
}

func NewService(db *gorm.DB) (*Service, error) {
	if err := db.AutoMigrate(&Event{}, &Tag{}, &EventTag{}); err != nil {
		return nil, err
	}

	return &Service{
		db:        db,
		mu:        &sync.Mutex{},
		onCreated: make(map[string][]func(*Event)),
		onUpdated: make(map[string][]func(*Event)),
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

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, sub := range s.onCreated[eventType] {
		sub(&event)
	}

	return event, nil
}

func (s Service) EventFinished(event *Event) error {
	return s.UpdateEvent(event, WithFinishedAt(time.Now()))
}

func (s Service) UpdateEvent(e *Event, opts ...EventOption) error {
	for _, opt := range opts {
		opt(e)
	}

	if err := s.db.Save(e).Error; err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, sub := range s.onUpdated[e.Type] {
		sub(e)
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

// GetActiveEventOfType returns the last active event of requested type.
func (s Service) GetActiveEventOfType(eventType string) (*Event, error) {
	return s.GetActiveEventOfTypes([]string{eventType})
}

// GetActiveEventOfTypes returns the last active event cross requested types.
func (s Service) GetActiveEventOfTypes(eventTypes []string) (*Event, error) {
	if len(eventTypes) == 0 {
		return nil, nil
	}

	var event Event

	err := s.db.
		Preload("Tags").
		Where("type IN ?", eventTypes).
		Where("finished_at IS NULL OR finished_at > ?", time.Now().UTC().Unix()).
		Order("occurred_at DESC").
		Limit(1).
		First(&event).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to find active event: %w", err)
	}

	return &event, nil
}

func (s Service) OnEventCreated(eventType string, fn func(event *Event)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.onCreated[eventType] = append(s.onCreated[eventType], fn)
}
