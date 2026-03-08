package usage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
	"github.com/focusd-so/focusd/internal/identity"

	"google.golang.org/genai"
	"gorm.io/gorm"
)

const (
	minSecondsForSummary  = 3600 // 1 hour of tracked activity required
	deepWorkThresholdSecs = 25 * 60
	summaryGenerationHour = 9
)

const llmDailySummaryPrompt = `You are a personal productivity coach analyzing a user's computer usage for one day.
Your job is to write a brief, insightful summary that helps the user understand
behavioral patterns and improve their focus habits.

RULES:
- Be conversational and warm, like a supportive coach -- not a report generator
- Lead with wins before addressing problems
- NEVER restate numbers in prose ("you spent 2h on VS Code") -- add insight the numbers alone can't convey
- Focus on cause-and-effect chains and behavioral patterns
- The narrative must be 3-4 sentences max
- The suggestion must be specific and actionable, referencing actual apps, times, or patterns from this day -- never generic ("try to focus more")
- key_pattern should surface something the user likely didn't notice themselves

HARD_RULES:
- Don't invent wins or key patterns that aren't present in the data
- Accuracy expontentially important than speed
- Avoid generic suggestions and day vibes, only valuable insights that are specific to the user's data

OUTPUT (strict JSON, no markdown fences):
{
  "headline": "5-8 word punchy summary",
  "narrative": "3-4 sentence story of the day",
  "key_pattern": "single most important behavioral pattern",
  "wins": ["1-3 specific wins"],
  "suggestion": "one actionable suggestion referencing today's data",
  "day_vibe": "locked-in | productive | balanced | scattered | recovering | rough"
}`

// GenerateLLMDailySummaryIfNeeded checks if it's time to generate yesterday's summary
// and produces one if it doesn't already exist.
func (s *Service) GenerateLLMDailySummaryIfNeeded(ctx context.Context) error {
	if s.genaiClient == nil {
		return nil
	}

	now := time.Now()
	if now.Hour() < summaryGenerationHour {
		return nil
	}

	yesterday := now.AddDate(0, 0, -1)
	dateStr := yesterday.Format("2006-01-02")

	var existing LLMDailySummary
	if err := s.db.Where("date = ?", dateStr).First(&existing).Error; err == nil {
		return nil // already generated
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check existing summary: %w", err)
	}

	input, err := s.computeLLMDaySummaryInput(yesterday)
	if err != nil {
		return fmt.Errorf("failed to compute summary input: %w", err)
	}

	totalTracked := (input.TotalProductiveMinutes + input.TotalDistractiveMinutes) * 60
	if totalTracked < minSecondsForSummary {
		slog.Info("not enough data for daily summary", "date", dateStr, "tracked_minutes", input.TotalProductiveMinutes+input.TotalDistractiveMinutes)
		return nil
	}

	summary, err := s.generateLLMDailySummary(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to generate LLM summary: %w", err)
	}

	summary.Date = dateStr
	summary.ContextSwitchCount = input.ContextSwitchCount
	summary.LongestFocusMinutes = input.LongestFocusStretchMin
	summary.DeepWorkMinutes = input.DeepWorkTotalMinutes
	summary.BlockedAttemptCount = input.BlockedAttemptCount
	summary.CreatedAt = time.Now().Unix()

	if err := s.db.Create(&summary).Error; err != nil {
		return fmt.Errorf("failed to save daily summary: %w", err)
	}

	slog.Info("daily summary generated", "date", dateStr, "headline", summary.Headline)

	if s.onLLMDailySummaryReady != nil {
		s.onLLMDailySummaryReady(summary)
	}

	return nil
}

func (s *Service) computeLLMDaySummaryInput(date time.Time) (LLMDaySummaryInput, error) {
	usages, err := s.GetUsageList(GetUsageListOptions{Date: &date})
	if err != nil {
		return LLMDaySummaryInput{}, fmt.Errorf("failed to get usage list: %w", err)
	}

	// GetUsageList returns DESC order, we need ASC for chronological walking
	sort.Slice(usages, func(i, j int) bool {
		return usages[i].StartedAt < usages[j].StartedAt
	})

	input := LLMDaySummaryInput{
		Date: date.Format("2006-01-02"),
	}

	var (
		productiveSecs  int
		distractiveSecs int
		contextSwitches int
		prevClass       Classification

		// deep work tracking
		currentDeepStart int64
		currentDeepSecs  int
		currentDeepApp   string
		deepSessions     []LLMDeepWorkSession

		// longest focus stretch
		currentFocusSecs int
		longestFocusSecs int

		// distraction cascade tracking
		cascadeStart   int64
		cascadeApps    []string
		cascadeSecs    int
		cascadeTrigger string
		cascades       []LLMDistractionCascade

		// per-app aggregation
		appProductiveSecs  = make(map[string]int)
		appProductiveVisit = make(map[string]int)
		appDistractSecs    = make(map[string]int)
		appDistractVisit   = make(map[string]int)

		// per-hour aggregation
		hourProductiveSecs = make(map[int]int)
		hourDistractSecs   = make(map[int]int)
	)

	for i, u := range usages {
		end := resolveEndTime(u, usages, i)
		if end <= u.StartedAt {
			continue
		}
		dur := int(end - u.StartedAt)
		appName := u.Application.Name

		startHour := time.Unix(u.StartedAt, 0).Hour()

		switch u.Classification {
		case ClassificationProductive:
			productiveSecs += dur
			appProductiveSecs[appName] += dur
			appProductiveVisit[appName]++
			hourProductiveSecs[startHour] += dur

			// deep work tracking: extend or start
			if currentDeepStart == 0 {
				currentDeepStart = u.StartedAt
				currentDeepApp = appName
			}
			currentDeepSecs += dur

			// focus stretch
			currentFocusSecs += dur
			if currentFocusSecs > longestFocusSecs {
				longestFocusSecs = currentFocusSecs
			}

			// end any distraction cascade
			if len(cascadeApps) > 0 {
				cascades = append(cascades, LLMDistractionCascade{
					TriggerTime:    time.Unix(cascadeStart, 0).Format("3:04pm"),
					TriggerApp:     cascadeTrigger,
					CascadeApps:    uniqueStrings(cascadeApps),
					TotalMinutes:   cascadeSecs / 60,
					ReturnedToWork: time.Unix(u.StartedAt, 0).Format("3:04pm"),
				})
				cascadeApps = nil
				cascadeSecs = 0
			}

		case ClassificationDistracting:
			distractiveSecs += dur
			appDistractSecs[appName] += dur
			appDistractVisit[appName]++
			hourDistractSecs[startHour] += dur

			// flush deep work session if it met the threshold
			if currentDeepSecs >= deepWorkThresholdSecs {
				deepSessions = append(deepSessions, LLMDeepWorkSession{
					Start:   time.Unix(currentDeepStart, 0).Format("3:04pm"),
					End:     time.Unix(u.StartedAt, 0).Format("3:04pm"),
					App:     currentDeepApp,
					Minutes: currentDeepSecs / 60,
				})
			}
			currentDeepStart = 0
			currentDeepSecs = 0
			currentDeepApp = ""

			// reset focus stretch
			currentFocusSecs = 0

			// distraction cascade tracking
			if len(cascadeApps) == 0 {
				cascadeStart = u.StartedAt
				cascadeTrigger = appName
			}
			cascadeApps = append(cascadeApps, appName)
			cascadeSecs += dur
		}

		// context switches (only between productive <-> distracting)
		if prevClass != "" && u.Classification != prevClass &&
			(u.Classification == ClassificationProductive || u.Classification == ClassificationDistracting) &&
			(prevClass == ClassificationProductive || prevClass == ClassificationDistracting) {
			contextSwitches++
		}
		if u.Classification == ClassificationProductive || u.Classification == ClassificationDistracting {
			prevClass = u.Classification
		}
	}

	// flush final deep work session
	if currentDeepSecs >= deepWorkThresholdSecs {
		lastEnd := time.Now()
		if len(usages) > 0 {
			last := usages[len(usages)-1]
			e := resolveEndTime(last, usages, len(usages)-1)
			if e > 0 {
				lastEnd = time.Unix(e, 0)
			}
		}
		deepSessions = append(deepSessions, LLMDeepWorkSession{
			Start:   time.Unix(currentDeepStart, 0).Format("3:04pm"),
			End:     lastEnd.Format("3:04pm"),
			App:     currentDeepApp,
			Minutes: currentDeepSecs / 60,
		})
	}

	// flush final cascade
	if len(cascadeApps) > 1 {
		cascades = append(cascades, LLMDistractionCascade{
			TriggerTime:  time.Unix(cascadeStart, 0).Format("3:04pm"),
			TriggerApp:   cascadeTrigger,
			CascadeApps:  uniqueStrings(cascadeApps),
			TotalMinutes: cascadeSecs / 60,
		})
	}

	deepWorkTotal := 0
	for _, ds := range deepSessions {
		deepWorkTotal += ds.Minutes
	}

	input.TotalProductiveMinutes = productiveSecs / 60
	input.TotalDistractiveMinutes = distractiveSecs / 60
	input.FocusScore = calculateProductivityScore(productiveSecs, distractiveSecs)
	input.ContextSwitchCount = contextSwitches
	input.LongestFocusStretchMin = longestFocusSecs / 60
	input.DeepWorkSessions = deepSessions
	input.DeepWorkTotalMinutes = deepWorkTotal
	input.DistractionCascades = cascades
	input.TopDistractions = topApps(appDistractSecs, appDistractVisit, 5)
	input.TopProductiveApps = topApps(appProductiveSecs, appProductiveVisit, 5)
	input.MostProductiveHours = peakHour(hourProductiveSecs)
	input.MostDistractiveHours = peakHour(hourDistractSecs)

	// blocked attempts
	blockMode := TerminationModeBlock
	blockedUsages, err := s.GetUsageList(GetUsageListOptions{Date: &date, TerminationMode: &blockMode})
	if err == nil {
		input.BlockedAttemptCount = len(blockedUsages)
	}

	// protection pauses
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local).Unix()
	dayEnd := dayStart + 86400
	var pauseCount int64
	s.db.Model(&ProtectionPause{}).Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).Count(&pauseCount)
	input.ProtectionPauseCount = int(pauseCount)

	// 7-day trend
	input.AvgFocusScoreLast7Days, input.FocusScoreTrend = s.computeFocusTrend(date)

	return input, nil
}

func (s *Service) generateLLMDailySummary(ctx context.Context, input LLMDaySummaryInput) (LLMDailySummary, error) {
	if s.genaiClient == nil {
		return LLMDailySummary{}, errors.New("genai client not configured")
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return LLMDailySummary{}, fmt.Errorf("failed to marshal input: %w", err)
	}

	// Use a more capable model for summaries since this runs once per day
	models := map[apiv1.DeviceHandshakeResponse_AccountTier]string{
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_UNSPECIFIED: "gemini-2.5-flash",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE:        "gemini-2.5-flash",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_TRIAL:       "gemini-2.5-flash",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PLUS:        "gemini-2.5-flash",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO:         "gemini-2.5-flash",
	}

	tier := identity.GetAccountTier()
	model, ok := models[tier]
	if !ok {
		model = "gemini-2.5-flash"
	}

	resp, err := s.genaiClient.Models.GenerateContent(ctx, model, []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText(string(inputJSON)),
			},
		},
	}, &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				genai.NewPartFromText(llmDailySummaryPrompt),
			},
		},
	})
	if err != nil {
		return LLMDailySummary{}, fmt.Errorf("gemini call failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return LLMDailySummary{}, errors.New("empty response from Gemini")
	}

	text := resp.Candidates[0].Content.Parts[0].Text
	text = strings.NewReplacer("```json", "", "`", "").Replace(text)

	var parsed llmDailySummaryResponse
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return LLMDailySummary{}, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	winsJSON, _ := json.Marshal(parsed.Wins)

	return LLMDailySummary{
		Headline:   parsed.Headline,
		Narrative:  parsed.Narrative,
		KeyPattern: parsed.KeyPattern,
		Wins:       string(winsJSON),
		Suggestion: parsed.Suggestion,
		DayVibe:    parsed.DayVibe,
	}, nil
}

// computeFocusTrend looks at the last 7 days and returns the average score + trend direction.
func (s *Service) computeFocusTrend(referenceDate time.Time) (avgScore int, trend string) {
	var scores []int
	for i := 1; i <= 7; i++ {
		d := referenceDate.AddDate(0, 0, -i)
		insights, err := s.GetDayInsights(d)
		if err != nil {
			continue
		}
		if insights.ProductivityScore.ProductiveSeconds+insights.ProductivityScore.DistractiveSeconds < minSecondsForSummary {
			continue
		}
		scores = append(scores, insights.ProductivityScore.ProductivityScore)
	}

	if len(scores) == 0 {
		return 0, "stable"
	}

	total := 0
	for _, sc := range scores {
		total += sc
	}
	avgScore = total / len(scores)

	// Compare first half vs second half for trend
	if len(scores) >= 4 {
		firstHalf := 0
		for _, sc := range scores[:len(scores)/2] {
			firstHalf += sc
		}
		secondHalf := 0
		for _, sc := range scores[len(scores)/2:] {
			secondHalf += sc
		}
		firstAvg := firstHalf / (len(scores) / 2)
		secondAvg := secondHalf / (len(scores) - len(scores)/2)

		// "scores" is ordered recent-first, so firstHalf = more recent days
		if firstAvg > secondAvg+5 {
			trend = "improving"
		} else if firstAvg < secondAvg-5 {
			trend = "declining"
		} else {
			trend = "stable"
		}
	} else {
		trend = "stable"
	}

	return avgScore, trend
}

func topApps(secsByApp map[string]int, visitsByApp map[string]int, limit int) []LLMAppTimeSummary {
	type entry struct {
		app  string
		secs int
	}
	var entries []entry
	for app, secs := range secsByApp {
		entries = append(entries, entry{app, secs})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].secs > entries[j].secs
	})
	if len(entries) > limit {
		entries = entries[:limit]
	}

	result := make([]LLMAppTimeSummary, len(entries))
	for i, e := range entries {
		result[i] = LLMAppTimeSummary{
			App:     e.app,
			Minutes: e.secs / 60,
			Visits:  visitsByApp[e.app],
		}
	}
	return result
}

func peakHour(hourSecs map[int]int) string {
	if len(hourSecs) == 0 {
		return ""
	}
	maxHour := -1
	maxSecs := 0
	for h, s := range hourSecs {
		if s > maxSecs {
			maxSecs = s
			maxHour = h
		}
	}
	if maxHour < 0 {
		return ""
	}
	return fmt.Sprintf("%s-%s", formatHour(maxHour), formatHour(maxHour+1))
}

func formatHour(h int) string {
	h = h % 24
	if h == 0 {
		return "12am"
	}
	if h == 12 {
		return "12pm"
	}
	if h < 12 {
		return fmt.Sprintf("%dam", h)
	}
	return fmt.Sprintf("%dpm", h-12)
}

func uniqueStrings(s []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
