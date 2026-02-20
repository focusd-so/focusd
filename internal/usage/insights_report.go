package usage

import (
	"fmt"
	"time"
)

func (s *Service) GetDayInsights(date time.Time) (DayInsights, error) {
	listOpts := GetUsageListOptions{}

	if date.IsZero() {
		date = time.Now()
	}
	listOpts.Date = &date

	usages, err := s.GetUsageList(listOpts)
	if err != nil {
		return DayInsights{}, fmt.Errorf("failed to get usage list: %w", err)
	}

	var (
		productivityScore = ProductivityScore{
			ProductiveSeconds:  0,
			DistractiveSeconds: 0,
			OtherSeconds:       0,
		}
		productivityPerHourBreakdown = make(ProductivityPerHourBreakdown)
	)

	for _, usage := range usages {
		if usage.DurationSeconds == nil || *usage.DurationSeconds <= 0 {
			continue
		}

		durationSec := int(*usage.DurationSeconds)

		// Accumulate total productivity score
		if usage.Classification == ClassificationProductive {
			productivityScore.ProductiveSeconds += durationSec
		} else if usage.Classification == ClassificationDistracting {
			productivityScore.DistractiveSeconds += durationSec
		} else {
			productivityScore.OtherSeconds += durationSec
		}

		// Split usage across hour boundaries for per-hour breakdown
		start := time.Unix(usage.StartedAt, 0).UTC()
		end := start.Add(time.Duration(durationSec) * time.Second)
		cursor := start

		for cursor.Before(end) {
			hourStart := cursor.Truncate(time.Hour)
			hourEnd := hourStart.Add(time.Hour)

			segmentEnd := end
			if hourEnd.Before(end) {
				segmentEnd = hourEnd
			}

			overlapSeconds := int(segmentEnd.Sub(cursor).Seconds())

			bucket, ok := productivityPerHourBreakdown[hourStart]
			if !ok {
				bucket = ProductivityScore{}
			}

			if usage.Classification == ClassificationProductive {
				bucket.ProductiveSeconds += overlapSeconds
			} else if usage.Classification == ClassificationDistracting {
				bucket.DistractiveSeconds += overlapSeconds
			} else {
				bucket.OtherSeconds += overlapSeconds
			}

			productivityPerHourBreakdown[hourStart] = bucket
			cursor = hourEnd
		}
	}

	productivityScore.ProductivityScore = calculateProductivityScore(productivityScore.ProductiveSeconds, productivityScore.DistractiveSeconds)

	// Calculate productivity score for each hour bucket
	for hourKey, bucket := range productivityPerHourBreakdown {
		bucket.ProductivityScore = calculateProductivityScore(bucket.ProductiveSeconds, bucket.DistractiveSeconds)
		productivityPerHourBreakdown[hourKey] = bucket
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

	return int((float64(productiveSeconds) / float64(totalSeconds)) * 100)
}
