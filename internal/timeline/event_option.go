package timeline

import (
	"encoding/json"
	"time"
)

type EventOption func(e *Event) error

func WithStartedAt(t time.Time) EventOption {
	return func(e *Event) error {
		if t.IsZero() {
			t = time.Now()
		}

		e.StartedAt = t.UTC().Unix()

		return nil
	}
}

func WithEndedAt(t time.Time) EventOption {
	return func(e *Event) error {
		if !t.IsZero() {
			ts := t.UTC().Unix()

			e.EndedAt = &ts
		}

		return nil
	}
}

func WithPayload(payload any) EventOption {
	return func(e *Event) error {
		switch x := payload.(type) {
		case string:
			e.Payload = x
		default:
			b, err := json.Marshal(payload)
			if err != nil {
				return err
			}

			e.Payload = string(b)
		}

		return nil
	}
}

func WithTags(tags ...Tag) EventOption {
	return func(e *Event) error {
		e.Tags = tags

		return nil
	}
}
