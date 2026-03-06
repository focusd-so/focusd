package usage

import (
	"fmt"
	"math"
	"sort"
	"time"
)

type usageInterval struct {
	start int64
	end   int64
}

func mergeIntervals(intervals []usageInterval) []usageInterval {
	if len(intervals) == 0 {
		return nil
	}
	sort.Slice(intervals, func(i, j int) bool {
		if intervals[i].start == intervals[j].start {
			return intervals[i].end < intervals[j].end
		}
		return intervals[i].start < intervals[j].start
	})

	merged := make([]usageInterval, 0, len(intervals))
	merged = append(merged, intervals[0])

	for i := 1; i < len(intervals); i++ {
		last := &merged[len(merged)-1]
		current := intervals[i]

		if current.start <= last.end {
			if current.end > last.end {
				last.end = current.end
			}
		} else {
			merged = append(merged, current)
		}
	}
	return merged
}

func sumIntervals(intervals []usageInterval) int {
	total := 0
	for _, inv := range intervals {
		total += int(inv.end - inv.start)
	}
	return total
}

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

	// Group intervals by classification and CLIP to day boundary
	intervalsByClass := make(map[Classification][]usageInterval)

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)
	startUnix := startOfDay.Unix()
	endUnix := endOfDay.Unix()

	for _, usage := range usages {
		if usage.DurationSeconds == nil || *usage.DurationSeconds <= 0 {
			continue
		}

		uStart := usage.StartedAt
		uEnd := usage.StartedAt + int64(*usage.DurationSeconds)

		// Clip to day boundary
		if uStart < startUnix {
			uStart = startUnix
		}
		if uEnd > endUnix {
			uEnd = endUnix
		}

		if uStart < uEnd {
			intervalsByClass[usage.Classification] = append(intervalsByClass[usage.Classification], usageInterval{
				start: uStart,
				end:   uEnd,
			})
		}
	}

	// Merge intervals for total daily wall-clock time
	mergedProductive := mergeIntervals(intervalsByClass[ClassificationProductive])
	mergedDistractive := mergeIntervals(intervalsByClass[ClassificationDistracting])

	// 'Other' includes neutral, system, and unknown/error
	var otherRaw []usageInterval
	otherRaw = append(otherRaw, intervalsByClass[ClassificationNeutral]...)
	otherRaw = append(otherRaw, intervalsByClass[ClassificationSystem]...)
	otherRaw = append(otherRaw, intervalsByClass[ClassificationError]...)
	mergedOther := mergeIntervals(otherRaw)

	productivityScore := ProductivityScore{
		ProductiveSeconds:  sumIntervals(mergedProductive),
		DistractiveSeconds: sumIntervals(mergedDistractive),
		OtherSeconds:       sumIntervals(mergedOther),
	}
	productivityScore.ProductivityScore = calculateProductivityScore(productivityScore.ProductiveSeconds, productivityScore.DistractiveSeconds)

	// Hourly breakdown
	productivityPerHourBreakdown := make(ProductivityPerHourBreakdown)

	// Intersect the merged daily intervals with each hour bucket to ensure no double-counting within hours
	for i := 0; i < 24; i++ {
		hourStart := startOfDay.Add(time.Duration(i) * time.Hour)
		hourEnd := hourStart.Add(time.Hour)
		hStartUnix := hourStart.Unix()
		hEndUnix := hourEnd.Unix()

		intersect := func(intervals []usageInterval) []usageInterval {
			var result []usageInterval
			for _, inv := range intervals {
				// Check for overlap with [hStartUnix, hEndUnix]
				s := inv.start
				if s < hStartUnix {
					s = hStartUnix
				}
				e := inv.end
				if e > hEndUnix {
					e = hEndUnix
				}

				if s < e {
					result = append(result, usageInterval{start: s, end: e})
				}
			}
			// Re-merge in case the clipping created overlaps somehow (unlikely with already merged input)
			return mergeIntervals(result)
		}

		hProd := intersect(mergedProductive)
		hDist := intersect(mergedDistractive)
		hOther := intersect(mergedOther)

		bucket := ProductivityScore{
			ProductiveSeconds:  sumIntervals(hProd),
			DistractiveSeconds: sumIntervals(hDist),
			OtherSeconds:       sumIntervals(hOther),
		}

		if bucket.ProductiveSeconds > 0 || bucket.DistractiveSeconds > 0 || bucket.OtherSeconds > 0 {
			bucket.ProductivityScore = calculateProductivityScore(bucket.ProductiveSeconds, bucket.DistractiveSeconds)
			productivityPerHourBreakdown[hourStart] = bucket
		}
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
