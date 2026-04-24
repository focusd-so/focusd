package usage

import (
	"encoding/json"
	"time"

	"github.com/focusd-so/focusd/internal/timeline"
)

type CommunicationBreakdown struct {
	ApplicationID   int64  `json:"application_id"`
	ChannelName     string `json:"channel_name"`
	DurationSeconds int64  `json:"duration_seconds"`
}

type ProjectBreakdown struct {
	ApplicationID   int64  `json:"application_id"`
	ProjectName     string `json:"project_name"`
	DurationSeconds int64  `json:"duration_seconds"`
}

type DailySummary struct {
	// time durations
	TotalProductivityDuration time.Duration `json:"total_productivity_duration"`
	TotalDistractionDuration  time.Duration `json:"total_distraction_duration"`
	TotalIdleDuration         time.Duration `json:"total_idle_duration"`
	TotalUsageDuration        time.Duration `json:"total_usage_duration"`

	ProductivityScore float64 `json:"productivity_score"` // 0-100

	TopDistractions map[string]int `json:"top_distractions"`
	TopBlocked      map[string]int `json:"top_blocked"`

	ProjectBreakdown       []ProjectBreakdown       `json:"project_breakdown"`
	CommunicationBreakdown []CommunicationBreakdown `json:"communication_breakdown"`

	LLMDailySummary *LLMDailySummary `json:"llm_daily_summary,omitempty"`

	HourlyBreakdown map[string]*DailySummary `json:"hourly_breakdown"`
}

func (s *Service) DailySummary(date time.Time) DailySummary {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	events, err := s.timelineService.ListEvents(
		timeline.ByStartTime(dayStart, dayEnd),
		timeline.ByTypes(EventTypeUsageChanged, EventTypeUserIdleChanged),
		timeline.OrderByOccurredAtAsc(),
	)
	if err != nil {
		return DailySummary{}
	}

	summary := DailySummary{
		TopDistractions:        make(map[string]int),
		TopBlocked:             make(map[string]int),
		HourlyBreakdown:        make(map[string]*DailySummary),
		ProjectBreakdown:       []ProjectBreakdown{},
		CommunicationBreakdown: []CommunicationBreakdown{},
	}

	// Internal maps for accumulation
	projects := make(map[projectKey]int64)
	comms := make(map[commKey]int64)

	for i, event := range events {
		start := time.Unix(event.OccurredAt, 0).In(date.Location())
		var end time.Time

		if i < len(events)-1 {
			end = time.Unix(events[i+1].OccurredAt, 0).In(date.Location())
		} else if event.FinishedAt != nil {
			end = time.Unix(*event.FinishedAt, 0).In(date.Location())
		} else {
			// If it's the last event and not finished, use Now() capped to day end
			now := time.Now().In(date.Location())
			if now.After(dayEnd) {
				end = dayEnd
			} else {
				end = now
			}
		}

		// Ensure we stay within the requested day boundaries
		if start.Before(dayStart) {
			start = dayStart
		}
		if end.After(dayEnd) {
			end = dayEnd
		}
		if !end.After(start) {
			continue
		}

		s.processEvent(&summary, event, start, end, projects, comms)
	}

	// Finalize summary
	s.finalizeSummary(&summary, projects, comms)

	// Fetch LLM summary if exists
	var llmSummary LLMDailySummary
	if err := s.db.Where("date = ?", date.Format("2006-01-02")).First(&llmSummary).Error; err == nil {
		summary.LLMDailySummary = &llmSummary
	}

	return summary
}

type projectKey struct {
	AppID int64
	Name  string
}

type commKey struct {
	AppID int64
	Name  string
}

func (s *Service) processEvent(summary *DailySummary, event *timeline.Event, start, end time.Time, projects map[projectKey]int64, comms map[commKey]int64) {
	duration := end.Sub(start)
	summary.TotalUsageDuration += duration

	var payload ApplicationUsagePayload
	if event.Type == EventTypeUsageChanged {
		_ = json.Unmarshal([]byte(event.Payload), &payload)
	}

	isIdle := event.Type == EventTypeUserIdleChanged
	classification := payload.Classification

	// Accumulate durations
	if isIdle {
		summary.TotalIdleDuration += duration
	} else {
		switch classification {
		case ClassificationProductive:
			summary.TotalProductivityDuration += duration
		case ClassificationDistracting:
			summary.TotalDistractionDuration += duration
		}
	}

	// Breakdown data (only for the main summary, or we can pass it down)
	// For simplicity, we accumulate breakdowns at the main level.
	// If hourly breakdown needs its own ProjectBreakdown etc., we should handle it.
	// Looking at the struct, HourlyBreakdown is map[string]*DailySummary, so it HAS these fields.
	
	if !isIdle && payload.ApplicationID != 0 {
		// We need app name for TopDistractions
		var app Application
		if err := s.db.First(&app, payload.ApplicationID).Error; err == nil {
			if classification == ClassificationDistracting {
				summary.TopDistractions[app.Name] += int(duration.Seconds())
			}
			
			// Projects and Comms from ClassificationResult
			if res := payload.ClassificationResult; res != nil && res.LLMClassificationResult != nil {
				if proj := res.LLMClassificationResult.DetectedProject; proj != "" {
					projects[projectKey{AppID: app.ID, Name: proj}] += int64(duration.Seconds())
				}
				if comm := res.LLMClassificationResult.DetectedCommunicationChannel; comm != "" {
					comms[commKey{AppID: app.ID, Name: comm}] += int64(duration.Seconds())
				}
			}
		}
	}

	// Hourly breakdown splitting
	s.splitIntoHours(summary, event, start, end)
}

func (s *Service) splitIntoHours(summary *DailySummary, event *timeline.Event, start, end time.Time) {
	curr := start
	for curr.Before(end) {
		hourStart := curr
		hourEnd := curr.Truncate(time.Hour).Add(time.Hour)
		if hourEnd.After(end) {
			hourEnd = end
		}

		hourKey := hourStart.Format("15:04") // or "15:00"
		hourKey = hourStart.Truncate(time.Hour).Format("15:04")

		hourSummary, ok := summary.HourlyBreakdown[hourKey]
		if !ok {
			hourSummary = &DailySummary{
				TopDistractions: make(map[string]int),
				HourlyBreakdown: make(map[string]*DailySummary),
			}
			summary.HourlyBreakdown[hourKey] = hourSummary
		}

		// Accumulate for this hour (minimal version, mainly durations)
		duration := hourEnd.Sub(hourStart)
		hourSummary.TotalUsageDuration += duration

		var payload ApplicationUsagePayload
		if event.Type == EventTypeUsageChanged {
			_ = json.Unmarshal([]byte(event.Payload), &payload)
		}

		if event.Type == EventTypeUserIdleChanged {
			hourSummary.TotalIdleDuration += duration
		} else {
			switch payload.Classification {
			case ClassificationProductive:
				hourSummary.TotalProductivityDuration += duration
			case ClassificationDistracting:
				hourSummary.TotalDistractionDuration += duration
			}
		}

		curr = hourEnd
	}
}

func (s *Service) finalizeSummary(summary *DailySummary, projects map[projectKey]int64, comms map[commKey]int64) {
	// Productivity Score: (Prod / (Prod + Dist)) * 100
	calcScore := func(prod, dist time.Duration) float64 {
		total := prod + dist
		if total == 0 {
			return 0
		}
		return (float64(prod) / float64(total)) * 100
	}

	summary.ProductivityScore = calcScore(summary.TotalProductivityDuration, summary.TotalDistractionDuration)

	for _, hourSummary := range summary.HourlyBreakdown {
		hourSummary.ProductivityScore = calcScore(hourSummary.TotalProductivityDuration, hourSummary.TotalDistractionDuration)
	}

	// Convert maps to slices
	for k, v := range projects {
		summary.ProjectBreakdown = append(summary.ProjectBreakdown, ProjectBreakdown{
			ApplicationID:   k.AppID,
			ProjectName:     k.Name,
			DurationSeconds: v,
		})
	}

	for k, v := range comms {
		summary.CommunicationBreakdown = append(summary.CommunicationBreakdown, CommunicationBreakdown{
			ApplicationID:   k.AppID,
			ChannelName:     k.Name,
			DurationSeconds: v,
		})
	}
}

