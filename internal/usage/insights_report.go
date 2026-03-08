package usage

import (
	"fmt"
	"math"
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

	return DayInsights{ProductivityScore: score, ProductivityPerHourBreakdown: hourly}, nil
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
