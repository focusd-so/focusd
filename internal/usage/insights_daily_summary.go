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
	"github.com/focusd-so/focusd/internal/settings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/openai"
	"google.golang.org/api/option"
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

func (s *Service) GenerateLLMDailySummaryIfNeeded(ctx context.Context) error {
	if !dailySummaryLLMConfigured() {
		return nil
	}

	now := time.Now()
	if now.Hour() < summaryGenerationHour {
		return nil
	}

	yesterday := now.AddDate(0, 0, -1)
	startDate := s.resolveBackfillStart(yesterday)

	for d := startDate; !d.After(yesterday); d = d.AddDate(0, 0, 1) {
		if err := s.generateDailySummaryForDate(ctx, d); err != nil {
			slog.Error("failed to generate daily summary",
				"date", d.Format("2006-01-02"), "error", err)
		}
	}
	return nil
}

// resolveBackfillStart returns the first date that needs a summary.
// If previous summaries exist, returns the day after the most recent one.
// On first run (empty table), returns yesterday so only one day is processed.
func (s *Service) resolveBackfillStart(yesterday time.Time) time.Time {
	var last LLMDailySummary
	if err := s.db.Order("date DESC").First(&last).Error; err != nil {
		return yesterday
	}

	lastDate, err := time.Parse("2006-01-02", last.Date)
	if err != nil {
		return yesterday
	}

	return lastDate.AddDate(0, 0, 1)
}

func (s *Service) generateDailySummaryForDate(ctx context.Context, date time.Time) error {
	dateStr := date.Format("2006-01-02")

	input, err := s.computeLLMDaySummaryInput(date)
	if err != nil {
		return fmt.Errorf("failed to compute summary input: %w", err)
	}

	var summary = LLMDailySummary{
		Headline:            "Not Enough Data",
		DayVibe:             "insufficient-data",
		Date:                dateStr,
		CreatedAt:           time.Now().Unix(),
		ContextSwitchCount:  input.ContextSwitchCount,
		LongestFocusMinutes: input.LongestFocusStretchMin,
		DeepWorkMinutes:     input.DeepWorkTotalMinutes,
		BlockedAttemptCount: input.BlockedAttemptCount,
	}

	if input.hasMinimumData() {
		tempSummary, err := s.generateLLMDailySummary(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to generate LLM summary: %w", err)
		}

		summary = tempSummary
	}

	if err := s.db.Create(&summary).Error; err != nil {
		return fmt.Errorf("failed to save daily summary: %w", err)
	}

	slog.Info("daily summary generated", "date", dateStr, "headline", summary.Headline)

	if summary.DayVibe != "insufficient-data" {
		s.eventsMu.RLock()
		for _, fn := range s.onLLMDailySummaryReady {
			fn(summary)
		}
		s.eventsMu.RUnlock()
	}

	return nil
}

// computeLLMDaySummaryInput builds the structured input for the LLM daily summary.
// It aggregates app usage for a given date into productivity metrics, deep work sessions,
// distraction cascades, and per-app/per-hour breakdowns — all fed to the LLM for narrative generation.
func (s *Service) computeLLMDaySummaryInput(date time.Time) (LLMDaySummaryInput, error) {

	// Get usages and sort them by started at
	usages, err := s.GetUsageList(GetUsageListOptions{Date: &date})
	if err != nil {
		return LLMDaySummaryInput{}, fmt.Errorf("failed to get usage list: %w", err)
	}
	sort.Slice(usages, func(i, j int) bool { return usages[i].StartedAt < usages[j].StartedAt })

	var (
		productiveSecs  int
		distractingSecs int
		contextSwitches int // productive↔distracting transitions
		prevClass       Classification

		appProductiveSecs  = make(map[string]int)
		appProductiveVisit = make(map[string]int)
		appDistractSecs    = make(map[string]int)
		appDistractVisit   = make(map[string]int)
		hourProductiveSecs = make(map[int]int)
		hourDistractSecs   = make(map[int]int)

		deep    deepWorkTracker // emits sessions ≥ 25 min
		focus   focusStretchTracker
		cascade cascadeTracker
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
			deep.processProductive(u.StartedAt, dur, appName)
			focus.addProductive(dur)
			cascade.endCascade(u.StartedAt)

		case ClassificationDistracting:
			distractingSecs += dur
			appDistractSecs[appName] += dur
			appDistractVisit[appName]++
			hourDistractSecs[startHour] += dur
			deep.processDistracting(u.StartedAt)
			focus.reset()
			cascade.addDistracting(u.StartedAt, dur, appName)
		}

		if u.Classification.IsProductiveOrDistracting() {
			if prevClass != "" && u.Classification != prevClass {
				contextSwitches++
			}
			prevClass = u.Classification
		}
	}

	// Flush trackers for any in-progress sessions at end of day
	lastEnd := time.Now()
	if len(usages) > 0 {
		last := usages[len(usages)-1]
		if e := resolveEndTime(last, usages, len(usages)-1); e > 0 {
			lastEnd = time.Unix(e, 0)
		}
	}
	deep.flush(lastEnd)
	cascade.flush()

	input := LLMDaySummaryInput{
		Date:                    date.Format("2006-01-02"),
		TotalProductiveMinutes:  productiveSecs / 60,
		TotalDistractingMinutes: distractingSecs / 60,
		FocusScore:              calculateProductivityScore(productiveSecs, distractingSecs),
		ContextSwitchCount:      contextSwitches,
		LongestFocusStretchMin:  focus.longestMinutes(),
		DeepWorkSessions:        deep.sessions,
		DeepWorkTotalMinutes:    deep.totalMinutes(),
		DistractionCascades:     cascade.cascades,
		TopDistractions:         topApps(appDistractSecs, appDistractVisit, 5),
		TopProductiveApps:       topApps(appProductiveSecs, appProductiveVisit, 5),
		MostProductiveHours:     peakHour(hourProductiveSecs),
		MostDistractingHours:    peakHour(hourDistractSecs),
	}

	s.enrichWithDBStats(&input, date)

	return input, nil
}

func (s *Service) enrichWithDBStats(input *LLMDaySummaryInput, date time.Time) {
	blockMode := EnforcementActionBlock
	blockedUsages, err := s.GetUsageList(GetUsageListOptions{Date: &date, EnforcementAction: &blockMode})
	if err == nil {
		input.BlockedAttemptCount = len(blockedUsages)
	}

	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local).Unix()
	dayEnd := dayStart + 86400
	var pauseCount int64
	s.db.Model(&ProtectionPause{}).Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).Count(&pauseCount)
	input.ProtectionPauseCount = int(pauseCount)

	input.AvgFocusScoreLast7Days, input.FocusScoreTrend = s.computeFocusTrend(date)
}

func (s *Service) generateLLMDailySummary(ctx context.Context, input LLMDaySummaryInput) (LLMDailySummary, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return LLMDailySummary{}, fmt.Errorf("failed to marshal input: %w", err)
	}

	text, err := generateDailySummary(ctx, llmDailySummaryPrompt, string(inputJSON))
	if err != nil {
		return LLMDailySummary{}, err
	}

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

func dailySummaryLLMConfigured() bool {
	switch settings.GetConfig().ClassificationLLMProvider {
	case settings.LLMProviderGoogle, settings.LLMProviderOpenAI, settings.LLMProviderAnthropic, settings.LLMProviderGroq:
		return true
	default:
		return false
	}
}

func generateDailySummary(ctx context.Context, systemPrompt, input string) (string, error) {
	switch settings.GetConfig().ClassificationLLMProvider {
	case settings.LLMProviderGoogle:
		return generateDailySummaryWithGemini(ctx, systemPrompt, input)
	case settings.LLMProviderOpenAI:
		return generateDailySummaryWithOpenAI(ctx, systemPrompt, input)
	case settings.LLMProviderAnthropic:
		return generateDailySummaryWithAnthropic(ctx, systemPrompt, input)
	case settings.LLMProviderGroq:
		return generateDailySummaryWithGrok(ctx, systemPrompt, input)
	default:
		return "", errors.New("unsupported LLM provider")
	}
}

func selectDailySummaryModel(models map[apiv1.DeviceHandshakeResponse_AccountTier]string) string {
	tier := identity.GetAccountTier()
	if model, ok := models[tier]; ok {
		return model
	}

	slog.Warn("unsupported account tier for daily summary model, using free tier model", "tier", tier)

	if model, ok := models[apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE]; ok {
		return model
	}

	return models[apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_UNSPECIFIED]
}

func generateDailySummaryWithGemini(ctx context.Context, systemPrompt, input string) (string, error) {
	// Use the strongest current non-reasoning Gemini model.
	models := map[apiv1.DeviceHandshakeResponse_AccountTier]string{
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_UNSPECIFIED: "gemini-2.5-flash-lite",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE:        "gemini-2.5-flash-lite",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_TRIAL:       "gemini-2.5-flash-lite",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PLUS:        "gemini-2.5-flash-lite",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO:         "gemini-2.5-flash-lite",
	}

	withGoogleAIEndpoint := func(endpoint string) googleai.Option {
		return func(opts *googleai.Options) {
			opts.ClientOptions = append(opts.ClientOptions, option.WithEndpoint(endpoint))
		}
	}

	model, err := googleai.New(ctx,
		googleai.WithDefaultModel(selectDailySummaryModel(models)),
		googleai.WithHTTPClient(newSignedLLMHTTPClient()),
		withGoogleAIEndpoint(settings.APIBaseURL()+"/api/v1/gemini"),
	)
	if err != nil {
		return "", err
	}

	return generateDailySummaryWithLLM(ctx, model, systemPrompt, input)
}

func generateDailySummaryWithOpenAI(ctx context.Context, systemPrompt, input string) (string, error) {
	models := map[apiv1.DeviceHandshakeResponse_AccountTier]string{
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_UNSPECIFIED: "gpt-4.1",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE:        "gpt-4.1",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_TRIAL:       "gpt-4.1",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PLUS:        "gpt-4.1",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO:         "gpt-4.1",
	}

	model, err := openai.New(
		openai.WithModel(selectDailySummaryModel(models)),
		openai.WithToken("stubbed"),
		openai.WithHTTPClient(newSignedLLMHTTPClient()),
		openai.WithBaseURL(settings.APIBaseURL()+"/api/v1/openai/v1"),
	)
	if err != nil {
		return "", err
	}

	return generateDailySummaryWithLLM(ctx, model, systemPrompt, input)
}

func generateDailySummaryWithAnthropic(ctx context.Context, systemPrompt, input string) (string, error) {
	models := map[apiv1.DeviceHandshakeResponse_AccountTier]string{
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_UNSPECIFIED: "claude-sonnet-4-5",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE:        "claude-sonnet-4-5",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_TRIAL:       "claude-sonnet-4-5",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PLUS:        "claude-sonnet-4-5",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO:         "claude-sonnet-4-5",
	}

	model, err := anthropic.New(
		anthropic.WithModel(selectDailySummaryModel(models)),
		anthropic.WithToken("stubbed"),
		anthropic.WithHTTPClient(newSignedLLMHTTPClient()),
		anthropic.WithBaseURL(settings.APIBaseURL()+"/api/v1/anthropic/v1"),
	)
	if err != nil {
		return "", err
	}

	return generateDailySummaryWithLLM(ctx, model, systemPrompt, input)
}

func generateDailySummaryWithGrok(ctx context.Context, systemPrompt, input string) (string, error) {
	models := map[apiv1.DeviceHandshakeResponse_AccountTier]string{
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_UNSPECIFIED: "grok-4.20-beta-latest-non-reasoning",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE:        "grok-4.20-beta-latest-non-reasoning",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_TRIAL:       "grok-4.20-beta-latest-non-reasoning",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PLUS:        "grok-4.20-beta-latest-non-reasoning",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO:         "grok-4.20-beta-latest-non-reasoning",
	}

	model, err := openai.New(
		openai.WithModel(selectDailySummaryModel(models)),
		openai.WithToken("stubbed"),
		openai.WithHTTPClient(newSignedLLMHTTPClient()),
		openai.WithBaseURL(settings.APIBaseURL()+"/api/v1/grok/v1"),
	)
	if err != nil {
		return "", err
	}

	return generateDailySummaryWithLLM(ctx, model, systemPrompt, input)
}

func generateDailySummaryWithLLM(ctx context.Context, model llms.Model, systemPrompt, input string) (string, error) {
	resp, err := model.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	})
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("empty response from LLM")
	}

	text := resp.Choices[0].Content
	text = strings.NewReplacer("```json", "", "```", "", "`", "").Replace(text)

	return text, nil
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
		if insights.ProductivityScore.ProductiveSeconds+insights.ProductivityScore.DistractingSeconds < minSecondsForSummary {
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
