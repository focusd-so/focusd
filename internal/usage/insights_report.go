package usage

import (
	"fmt"
	"math"
	"sort"
	"time"
)

func (s *Service) GetDayInsights(date time.Time) (DayInsights, error) {
	if date.IsZero() {
		date = time.Now()
	}

	usages, err := s.GetUsageList(GetUsageListOptions{Date: &date})
	if err != nil {
		return DayInsights{}, fmt.Errorf("failed to get usage list: %w", err)
	}

	score := ProductivityScore{}
	hourly := make(ProductivityPerHourBreakdown)

	for i, usage := range usages {
		end := resolveEndTime(usage, usages, i)
		if end <= usage.StartedAt {
			continue
		}

		dur := int(end - usage.StartedAt)
		score.addSeconds(usage.Classification, dur)

		for hour, secs := range splitSecondsPerHour(usage.StartedAt, end) {
			entry := hourly[hour]
			entry.addSeconds(usage.Classification, secs)
			hourly[hour] = entry
		}
	}

	score.ProductivityScore = calculateProductivityScore(score.ProductiveSeconds, score.DistractiveSeconds)
	for hour, s := range hourly {
		s.ProductivityScore = calculateProductivityScore(s.ProductiveSeconds, s.DistractiveSeconds)
		hourly[hour] = s
	}

	insights := DayInsights{
		ProductivityScore:            score,
		ProductivityPerHourBreakdown: hourly,
		TopDistractions:              buildDistractionBreakdown(usages),
		TopBlocked:                   buildBlockedBreakdown(usages),
		ProjectBreakdown:             buildProjectBreakdown(usages),
		CommunicationBreakdown:       buildCommunicationBreakdown(usages),
	}

	var summary LLMDailySummary
	if err := s.db.Where("date = ?", date.Format("2006-01-02")).First(&summary).Error; err == nil {
		insights.LLMDailySummary = &summary
	}

	return insights, nil
}

func resolveEndTime(usage ApplicationUsage, usages []ApplicationUsage, i int) int64 {
	if usage.EndedAt != nil {
		return *usage.EndedAt
	}
	if i+1 < len(usages) {
		return usages[i+1].StartedAt
	}
	return 0
}

func splitSecondsPerHour(startUnix, endUnix int64) map[int]int {
	start := time.Unix(startUnix, 0)
	end := time.Unix(endUnix, 0)
	result := make(map[int]int)

	for cursor := start; cursor.Before(end); {
		hour := cursor.Hour()
		nextHour := time.Date(cursor.Year(), cursor.Month(), cursor.Day(), hour+1, 0, 0, 0, time.Local)

		segmentEnd := end
		if nextHour.Before(end) {
			segmentEnd = nextHour
		}

		result[hour] += int(segmentEnd.Sub(cursor).Seconds())
		cursor = segmentEnd
	}

	return result
}

func calculateProductivityScore(productiveSeconds, distractiveSeconds int) int {
	totalSeconds := productiveSeconds + distractiveSeconds

	if totalSeconds == 0 {
		return 0
	}

	return int(math.Round((float64(productiveSeconds) / float64(totalSeconds)) * 100))
}

func usageDisplayName(usage ApplicationUsage) string {
	if usage.Application.Hostname != nil && *usage.Application.Hostname != "" {
		return *usage.Application.Hostname
	}
	return usage.Application.Name
}

func buildDistractionBreakdown(usages []ApplicationUsage) []DistractionBreakdown {
	seconds := make(map[string]int)
	for i, u := range usages {
		if u.Classification != ClassificationDistracting {
			continue
		}
		if u.TerminationMode == TerminationModeBlock {
			continue
		}
		end := resolveEndTime(u, usages, i)
		if end <= u.StartedAt {
			continue
		}
		seconds[usageDisplayName(u)] += int(end - u.StartedAt)
	}

	out := make([]DistractionBreakdown, 0, len(seconds))
	for name, secs := range seconds {
		out = append(out, DistractionBreakdown{Name: name, Minutes: secs / 60})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Minutes > out[j].Minutes })
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

func buildBlockedBreakdown(usages []ApplicationUsage) []BlockedBreakdown {
	counts := make(map[string]int)
	for _, u := range usages {
		if u.TerminationMode != TerminationModeBlock {
			continue
		}
		counts[usageDisplayName(u)]++
	}

	out := make([]BlockedBreakdown, 0, len(counts))
	for name, count := range counts {
		out = append(out, BlockedBreakdown{Name: name, Count: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

func buildProjectBreakdown(usages []ApplicationUsage) []ProjectBreakdown {
	seconds := make(map[string]int)
	for i, u := range usages {
		if u.DetectedProject == nil || *u.DetectedProject == "" {
			continue
		}
		end := resolveEndTime(u, usages, i)
		if end <= u.StartedAt {
			continue
		}
		seconds[*u.DetectedProject] += int(end - u.StartedAt)
	}

	out := make([]ProjectBreakdown, 0, len(seconds))
	for name, secs := range seconds {
		out = append(out, ProjectBreakdown{Name: name, Minutes: secs / 60})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Minutes > out[j].Minutes })
	return out
}

func hasTag(tags []ApplicationUsageTags, tag string) bool {
	for _, t := range tags {
		if t.Tag == tag {
			return true
		}
	}
	return false
}

func buildCommunicationBreakdown(usages []ApplicationUsage) []CommunicationBreakdown {
	seconds := make(map[string]int)
	for i, u := range usages {
		isCommunication := (u.DetectedCommunicationChannel != nil && *u.DetectedCommunicationChannel != "") ||
			hasTag(u.Tags, "communication")
		if !isCommunication {
			continue
		}
		end := resolveEndTime(u, usages, i)
		if end <= u.StartedAt {
			continue
		}
		seconds[usageDisplayName(u)] += int(end - u.StartedAt)
	}

	out := make([]CommunicationBreakdown, 0, len(seconds))
	for name, secs := range seconds {
		out = append(out, CommunicationBreakdown{Name: name, Minutes: secs / 60})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Minutes > out[j].Minutes })
	return out
}
