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

func NewService(db *gorm.DB) *Service {
	return &Service{
		db:        db,
		mu:        &sync.Mutex{},
		onCreated: make(map[string][]func(*Event)),
		onUpdated: make(map[string][]func(*Event)),
	}
}

func (s Service) OnEventCreated(eventType string, fn func(event *Event)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.onCreated[eventType] = append(s.onCreated[eventType], fn)
}

func (s Service) CreateEvent(eventType string, opts ...EventOption) (Event, error) {
	event := NewEvent(eventType, opts...)

	if err := s.db.Create(&event).Error; err != nil {
		return event, fmt.Errorf("failed to create event: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, sub := range s.onCreated[eventType] {
		sub(&event)
	}

	return event, nil
}

func (s Service) UpdateEvent(e *Event) error {
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
		Where("ended_at IS NULL OR ended_at > ?", time.Now().UTC().Unix()).
		Order("started_at DESC").
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
