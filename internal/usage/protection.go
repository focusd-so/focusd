package usage

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/focusd-so/focusd/internal/timeline"
)

type PauseProtectionPayload struct {
	ResumeReason string `json:"resume_reason"`
	PauseReason  string `json:"pause_reason"`
}

type AllowUsagePayload struct {
	AppName  string `json:"app_name,omitempty"`
	URL      string `json:"url,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

// ProtectionPause temporarily disables focus protection for the specified duration.
// It creates a new timeline event of type EventTypeProtectionPause or updates
// an existing active pause event to extend its duration.
//
// Parameters:
//   - durationSeconds: The duration to pause protection in seconds.
//   - reason: A user-provided reason for the pause.
//
// Returns an error if the timeline event creation or update fails.
func (s *Service) ProtectionPause(durationSeconds int, reason string) error {
	dur := time.Duration(durationSeconds) * time.Second
	willEndAt := time.Now().Add(dur)

	event, err := s.timelineService.GetActiveEventOfType(EventTypeProtectionStatusChanged)
	if err != nil {
		return err
	}

	if event != nil {
		event.FinishedAt = new(willEndAt.Unix())

		if err := s.timelineService.UpdateEvent(event); err != nil {
			return fmt.Errorf("updating event: %w", err)
		}

		return nil
	}

	_, err = s.timelineService.CreateEvent(
		EventTypeProtectionStatusChanged,
		timeline.WithFinishedAt(willEndAt),
		timeline.WithPayload(PauseProtectionPayload{PauseReason: reason}),
	)
	if err != nil {
		return err
	}

	return nil
}

// ProtectionResume ends any active protection pause early.
// It looks for an active timeline event of type EventTypeProtectionPause
// and sets its end time to now.
//
// Parameters:
//   - reason: A user-provided reason for resuming protection early.
//
// Returns an error if the timeline event update fails.
func (s *Service) ProtectionResume(reason string) error {
	event, err := s.timelineService.GetActiveEventOfType(EventTypeProtectionStatusChanged)
	if err != nil {
		return err
	}

	// no active protection pause to resume
	if event == nil {
		return nil
	}

	event.FinishedAt = new(time.Now().Unix())

	if err := s.timelineService.UpdateEvent(event); err != nil {
		return fmt.Errorf("updating event: %w", err)
	}

	return nil
}

// ProtectionGetStatus retrieves the current protection pause status.
// It queries the timeline service for an active event of type EventTypeProtectionPause.
//
// Returns the active event if found, or nil if protection is currently active.
func (s *Service) ProtectionGetStatus() (*timeline.Event, error) {
	return s.timelineService.GetActiveEventOfType(EventTypeProtectionStatusChanged)
}

// PauseGetHistory retrieves the history of protection pauses within the specified number of days.
//
// Parameters:
//   - days: The number of days to look back.
//
// Returns a slice of timeline events ordered by age.
func (s *Service) PauseGetHistory(days int) ([]*timeline.Event, error) {
	return s.timelineService.ListEvents(
		timeline.ByTypes(EventTypeProtectionStatusChanged),
		timeline.ByAge(days),
	)
}

// AllowApp temporarily allows usage of a specific application for the specified duration.
// It creates or updates an active timeline event of type EventTypeAllowUsage.
//
// Parameters:
//   - appname: The application name to allow.
//   - duration: The duration to allow the usage.
//
// Returns an error if the timeline event creation or update fails.
func (s *Service) AllowApp(appname string, duration time.Duration) error {
	return s.allowUsage(AllowUsagePayload{AppName: appname}, duration)
}

// AllowURL temporarily allows usage of a specific URL for the specified duration.
// It creates or updates an active timeline event of type EventTypeAllowUsage.
//
// Parameters:
//   - rawURL: The URL to allow.
//   - duration: The duration to allow the usage.
//
// Returns an error if the timeline event creation or update fails.
func (s *Service) AllowURL(rawURL string, duration time.Duration) error {
	parsed, _, _ := parseURLNormalized(new(rawURL))
	if parsed == nil {
		return fmt.Errorf("failed to parse URL: %s", rawURL)
	}

	return s.allowUsage(AllowUsagePayload{URL: parsed.String()}, duration)
}

// AllowHostname temporarily allows usage of a specific hostname for the specified duration.
// It creates or updates an active timeline event of type EventTypeAllowUsage.
//
// Parameters:
//   - rawURL: The URL whose hostname will be allowed.
//   - duration: The duration to allow the usage.
//
// Returns an error if the timeline event creation or update fails.
func (s *Service) AllowHostname(rawURL string, duration time.Duration) error {
	_, hostname, _ := parseURLNormalized(new(rawURL))
	return s.allowUsage(AllowUsagePayload{Hostname: hostname}, duration)
}

// AllowGetAll returns all active allowed usage events that haven't expired.
//
// Returns a slice of active timeline events of type EventTypeAllowUsage.
func (s *Service) AllowGetAll() ([]*timeline.Event, error) {
	return s.timelineService.ListEvents(
		timeline.ByTypes(EventTypeAllowUsage),
		timeline.ActiveOnly(),
	)
}

// AllowRemove removes an allowed usage event by its database ID.
//
// Parameters:
//   - id: The ID of the event to remove.
//
// Returns an error if the deletion fails.
func (s *Service) AllowRemove(id int64) error {
	allowed, err := s.timelineService.ListEvents(
		timeline.ByTypes(EventTypeAllowUsage),
		timeline.ActiveOnly(),
	)
	if err != nil {
		return err
	}

	for _, event := range allowed {
		if event.ID != id {
			continue
		}

		return s.timelineService.EventFinished(event)
	}

	return nil
}

// allowUsage is an internal helper that creates or updates an allowed usage event.
// It checks for an existing active event with the same payload and extends its
// duration if found. Otherwise, it creates a new timeline event.
//
// Parameters:
//   - req: The payload containing the app name, URL, or hostname to allow.
//   - duration: The duration to allow the usage.
//
// Returns an error if the timeline event creation or update fails.
func (s *Service) allowUsage(req AllowUsagePayload, duration time.Duration) error {
	allowed, err := s.timelineService.ListEvents(
		timeline.ByTypes(EventTypeAllowUsage),
		timeline.ActiveOnly(),
	)

	if err != nil {
		return err
	}

	willEndAt := time.Now().Add(duration)

	for _, event := range allowed {
		payload := AllowUsagePayload{}
		if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
			return err
		}

		if payload.AppName == req.AppName && payload.URL == req.URL && payload.Hostname == req.Hostname {
			event.FinishedAt = new(willEndAt.Unix())
			if err := s.timelineService.UpdateEvent(event); err != nil {
				return fmt.Errorf("updating event: %w", err)
			}
			return nil
		}
	}

	_, err = s.timelineService.CreateEvent(
		EventTypeAllowUsage,
		timeline.WithFinishedAt(willEndAt),
		timeline.WithPayload(req),
	)

	return err
}
