package usage

import (
	"math"
	"time"
)

func (s *Service) GetDayInsights(date time.Time) (DayInsights, error) {
	// if date.IsZero() {
	// 	date = time.Now()
	// }

	// usages, err := s.GetUsageList(GetUsageListOptions{Date: &date})
	// if err != nil {
	// 	return DayInsights{}, fmt.Errorf("failed to get usage list: %w", err)
	// }

	// score := ProductivityScore{}
	// hourly := make(ProductivityPerHourBreakdown)

	// insights := DayInsights{
	// 	ProductivityScore:            score,
	// 	ProductivityPerHourBreakdown: hourly,
	// 	TopDistractions:              make(map[string]int),
	// 	TopBlocked:                   make(map[string]int),
	// 	ProjectBreakdown:             make(map[string]int),
	// 	CommunicationBreakdown:       make(map[string]CommunicationBreakdown),
	// }

	// for i, usage := range usages {
	// 	end := resolveEndTime(usage, usages, i)
	// 	if end <= usage.StartedAt {
	// 		continue
	// 	}

	// 	dur := int(end - usage.StartedAt)
	// 	// isIdle := usage.Application.Name == IdleApplicationName
	// 	// score.addSeconds(usage.Classification, dur, isIdle)

	// 	for hour, secs := range splitSecondsPerHour(usage.StartedAt, end) {
	// 		entry := hourly[hour]
	// 		// entry.addSeconds(usage.Classification, secs, isIdle)
	// 		hourly[hour] = entry
	// 	}

	// 	if usage.IsCommunicationUsage() {
	// 		key := fmt.Sprintf("%s:%s", usage.Application.Name, usage.CommunicationChannel())

	// 		entry := insights.CommunicationBreakdown[key]
	// 		entry.Name = usage.Application.Name
	// 		entry.Channel = usage.CommunicationChannel()
	// 		entry.DurationSeconds += dur
	// 		insights.CommunicationBreakdown[key] = entry
	// 	}

	// 	if usage.EnforcementAction == EnforcementActionBlock {
	// 		insights.TopBlocked[usageDisplayName(usage)] += dur
	// 	}

	// 	if usage.Classification == ClassificationDistracting && usage.EnforcementAction != EnforcementActionBlock {
	// 		insights.TopDistractions[usageDisplayName(usage)] += dur
	// 	}

	// 	if usage.HasDetectedProject() {
	// 		insights.ProjectBreakdown[usage.GetDetectedProject()] += dur
	// 	}
	// }

	// score.ProductivityScore = calculateProductivityScore(score.ProductiveSeconds, score.DistractingSeconds)
	// for hour, s := range hourly {
	// 	s.ProductivityScore = calculateProductivityScore(s.ProductiveSeconds, s.DistractingSeconds)
	// 	hourly[hour] = s
	// }

	// insights.ProductivityScore = score
	// insights.ProductivityPerHourBreakdown = hourly

	// var summary LLMDailySummary
	// if err := s.db.Where("date = ?", date.Format("2006-01-02")).First(&summary).Error; err == nil {
	// 	insights.LLMDailySummary = &summary
	// }

	return DayInsights{}, nil
}

// func resolveEndTime(usage ApplicationUsage, usages []ApplicationUsage, i int) int64 {
// 	if usage.EndedAt != nil {
// 		return *usage.EndedAt
// 	}
// 	if i+1 < len(usages) {
// 		return usages[i+1].StartedAt
// 	}
// 	return 0
// }

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

func calculateProductivityScore(productiveSeconds, distractingSeconds int) int {
	totalSeconds := productiveSeconds + distractingSeconds

	if totalSeconds == 0 {
		return 0
	}

	return int(math.Round((float64(productiveSeconds) / float64(totalSeconds)) * 100))
}

// func usageDisplayName(usage ApplicationUsage) string {
// 	if usage.Application.Hostname != nil && *usage.Application.Hostname != "" {
// 		return *usage.Application.Hostname
// 	}
// 	return usage.Application.Name
// }
