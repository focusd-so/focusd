package usage

import (
	"fmt"
	"log/slog"
	"math"
	"time"
)

func (s *Service) GetDayInsights(date time.Time) (DayInsights, error) {

	slog.Info("[INGISHTS]: requestung for", "date", date)

	if date.IsZero() {
		date = time.Now()
	}

	listOpts := GetUsageListOptions{
		Date: &date,
	}

	usages, err := s.GetUsageList(listOpts)
	if err != nil {
		return DayInsights{}, fmt.Errorf("failed to get usage list: %w", err)
	}

	slog.Info("[INGISHTS]: number of usages", "count", len(usages))

	var (
		productivityScore            = ProductivityScore{}
		productivityPerHourBreakdown = make(ProductivityPerHourBreakdown)
	)

	for i, usage := range usages {
		// calculate overall summary
		end := fromPtr(usage.EndedAt)

		if end == 0 {
			if len(usages) > i+1 {
				end = usages[i+1].StartedAt
			}

			if end == 0 {
				continue
			}
		}

		durationSeconds := end - usage.StartedAt
		if durationSeconds <= 0 {
			continue
		}

		slog.Info("[INGISHTS]: calculated duration in seconds", "duration", durationSeconds)

		switch usage.Classification {
		case ClassificationProductive:
			productivityScore.ProductiveSeconds += int(durationSeconds)
		case ClassificationDistracting:
			productivityScore.DistractiveSeconds += int(durationSeconds)
		default:
			productivityScore.OtherSeconds += int(durationSeconds)
		}

	}

	slog.Info("[INGISHTS]: total  seconds", "productivity", productivityScore.ProductiveSeconds, "distracting", productivityScore.DistractiveSeconds)

	// Calculate overall score
	productivityScore.ProductivityScore = calculateProductivityScore(productivityScore.ProductiveSeconds, productivityScore.DistractiveSeconds)

	// Calculate hourly scores
	for hour, score := range productivityPerHourBreakdown {
		score.ProductivityScore = calculateProductivityScore(score.ProductiveSeconds, score.DistractiveSeconds)
		productivityPerHourBreakdown[hour] = score
	}

	return DayInsights{
		ProductivityScore:            productivityScore,
		ProductivityPerHourBreakdown: productivityPerHourBreakdown,
	}, nil
}

func calculateProductivityScore(productiveSeconds, distractiveSeconds int) int {
	totalSeconds := productiveSeconds + distractiveSeconds

	if totalSeconds == 0 {
		return 0
	}

	return int(math.Round((float64(productiveSeconds) / float64(totalSeconds)) * 100))
}
