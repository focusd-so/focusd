package insights

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"log/slog"
// 	"sort"
// 	"time"

// 	"gorm.io/gorm"
// )

// const DailySummarySystemPrompt = `

// You are a supportive productivity coach analyzing daily screen time data for the Focusd app.
// Your role is to provide encouraging, actionable insights that help users understand their work patterns without being judgmental.

// RULES:
// - Be concise and conversational, like a friendly coach
// - Focus on patterns and behaviors, not raw numbers (user already sees those)
// - Always find at least one positive thing, even on rough days
// - Make suggestions specific and actionable
// - Use "you" language to be personal
// - Keep headlines punchy (5-8 words max)
// - NEVER invent or guess statistics - only reference what's in the data
// - If the data shows a bad day, be empathetic not critical

// TONE: Supportive, insightful, motivating. Like a friend who happens to be a productivity expert.

// OUTPUT FORMAT: Respond with valid JSON matching this exact structure:
// {
//   "headline": "string (catchy, no numbers, 5-8 words)",
//   "summary": "string (2-3 sentences about the day's patterns)",
//   "wins": ["string", "string"] (1-3 positive observations),
//   "suggestion": "string (one specific, actionable improvement)",
//   "day_vibe": "string (one word: focused, scattered, balanced, grinding, recovering, distracted)"
// }`

// const DailySummaryUserPrompt = `Analyze this productivity data for %s:
// 📊 DAILY STATS (pre-calculated, use these as reference):
// - Productivity Score: %d/100
// - Productive Time: %s
// - Distractive Time: %s
// - Other/Neutral Time: %s
// 📈 HOURLY BREAKDOWN:
// %s
// 🔝 TOP APPS BY TIME:
// %s
// Based on this data, generate a daily summary that helps the user understand their work patterns and how to improve.`

// // scheduleSummaryGeneration runs every hour and checks if the summary for the previous day has been generated
// // if not, it generates the summary for the previous day
// // if it has been generated, it checks if the summary is outdated
// // if it is outdated, it regenerates the summary for the previous day
// func (s *Service) scheduleSummaryGeneration(ctx context.Context) {

// 	fn := func() {
// 		yesterday := time.Now().AddDate(0, 0, -1)

// 		var summary DailyUsageSummary
// 		if err := s.db.Where("date = ?", yesterday.Format("2006-01-02")).First(&summary).Error; err != nil {
// 			if err == gorm.ErrRecordNotFound {
// 				s.GenerateSummary(yesterday)
// 			}
// 		}
// 	}

// 	fn()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		case <-time.After(1 * time.Hour):
// 			fn()
// 		}
// 	}
// }

// func (s *Service) GenerateSummary(date time.Time) (DailyUsageSummary, error) {
// 	usages, err := getUsage(s.db, date)
// 	if err != nil {
// 		return DailyUsageSummary{}, err
// 	}

// 	overview, err := s.getOverview(date, usages)
// 	if err != nil {
// 		return DailyUsageSummary{}, err
// 	}

// 	// jsonify overview
// 	overviewJSON, err := json.Marshal(overview)
// 	if err != nil {
// 		return DailyUsageSummary{}, err
// 	}

// 	// Build the user prompt with actual data
// 	userPrompt := fmt.Sprintf(DailySummaryUserPrompt,
// 		date.Format("Monday, January 2, 2006"),
// 		overview.UsageOverview.ProductivityScore,
// 		formatDuration(overview.UsageOverview.ProductiveSeconds),
// 		formatDuration(overview.UsageOverview.DistractiveSeconds),
// 		formatDuration(overview.UsageOverview.OtherSeconds),
// 		formatHourlyBreakdown(overview.UsagePerHourBreakdown),
// 		formatTopApps(usages),
// 	)

// 	// Generate summary using LLM
// 	var response LLMDailySummaryResponse
// 	if err := s.contentGenerator.GenerateContent(
// 		context.Background(),
// 		DailySummarySystemPrompt,
// 		userPrompt,
// 		&response,
// 	); err != nil {
// 		slog.Error("failed to generate LLM summary", "error", err)
// 		return DailyUsageSummary{}, fmt.Errorf("failed to generate LLM summary: %w", err)
// 	}

// 	// Convert wins slice to JSON string for storage
// 	winsJSON, err := json.Marshal(response.Wins)
// 	if err != nil {
// 		return DailyUsageSummary{}, err
// 	}

// 	dailySummary := DailyUsageSummary{
// 		Date:          date.Format("2006-01-02"),
// 		DailyOverview: string(overviewJSON),
// 		Headline:      response.Headline,
// 		Summary:       response.Summary,
// 		Wins:          string(winsJSON),
// 		Suggestion:    response.Suggestion,
// 		DayVibe:       response.DayVibe,
// 	}

// 	return dailySummary, s.db.Create(&dailySummary).Error
// }

// // formatDuration converts seconds to a human-readable duration string
// func formatDuration(seconds int) string {
// 	hours := seconds / 3600
// 	minutes := (seconds % 3600) / 60

// 	if hours > 0 {
// 		return fmt.Sprintf("%dh %dm", hours, minutes)
// 	}
// 	return fmt.Sprintf("%dm", minutes)
// }

// // formatHourlyBreakdown creates a visual representation of hourly productivity
// func formatHourlyBreakdown(breakdown []*UsagePerHourBreakdown) string {
// 	var result string
// 	for _, hour := range breakdown {
// 		totalSeconds := hour.ProductiveSeconds + hour.DistractiveSeconds + hour.OtherSeconds
// 		if totalSeconds == 0 {
// 			continue // Skip hours with no activity
// 		}

// 		productivePercent := 0
// 		if totalSeconds > 0 {
// 			productivePercent = (hour.ProductiveSeconds * 100) / totalSeconds
// 		}

// 		// Create a simple bar visualization
// 		filledBars := productivePercent / 10
// 		emptyBars := 10 - filledBars
// 		bar := ""
// 		for i := 0; i < filledBars; i++ {
// 			bar += "█"
// 		}
// 		for i := 0; i < emptyBars; i++ {
// 			bar += "░"
// 		}

// 		result += fmt.Sprintf("%s:00 - %s %d%% productive\n", hour.HourLabel, bar, productivePercent)
// 	}
// 	return result
// }

// // formatTopApps aggregates and formats the top apps by usage time
// func formatTopApps(usages []ApplicationUsage) string {
// 	// Aggregate by application
// 	appDurations := make(map[string]struct {
// 		duration       int64
// 		classification string
// 	})

// 	for _, usage := range usages {
// 		appName := usage.Application.Name
// 		if appName == "" {
// 			appName = usage.Application.BundleID
// 		}
// 		if appName == "" {
// 			continue
// 		}

// 		existing := appDurations[appName]
// 		existing.duration += usage.Duration
// 		if existing.classification == "" {
// 			existing.classification = string(usage.Classification)
// 		}
// 		appDurations[appName] = existing
// 	}

// 	// Sort by duration
// 	type appEntry struct {
// 		name           string
// 		duration       int64
// 		classification string
// 	}
// 	var apps []appEntry
// 	for name, data := range appDurations {
// 		apps = append(apps, appEntry{name, data.duration, data.classification})
// 	}
// 	sort.Slice(apps, func(i, j int) bool {
// 		return apps[i].duration > apps[j].duration
// 	})

// 	// Take top 5
// 	var result string
// 	limit := 5
// 	if len(apps) < limit {
// 		limit = len(apps)
// 	}
// 	for i := 0; i < limit; i++ {
// 		result += fmt.Sprintf("%d. %s (%s) - %s\n",
// 			i+1,
// 			apps[i].name,
// 			apps[i].classification,
// 			formatDuration(int(apps[i].duration)),
// 		)
// 	}
// 	return result
// }
