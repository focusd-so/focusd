package usage

import (
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

func (s *Service) enrichSandboxContext(ctx *sandboxContext) {
	if ctx.Now == nil {
		ctx.Now = func(loc *time.Location) time.Time {
			return time.Now().In(loc)
		}
	}

	if ctx.MinutesUsedInPeriod == nil {
		ctx.MinutesUsedInPeriod = s.minutesUsedInPeriod
	}

	if err := s.populateInsightsContext(ctx); err != nil {
		slog.Debug("failed to populate sandbox insights context", "error", err)
	}

	if err := s.populateCurrentUsageContext(ctx); err != nil {
		slog.Debug("failed to populate sandbox current-usage context", "error", err)
	}
}

func (s *Service) scopedUsageIdentityQuery(appName, hostname string) *gorm.DB {
	query := s.db.Model(&ApplicationUsage{}).
		Joins("JOIN application ON application.id = application_usage.application_id").
		Where("application.name = ?", appName)

	if hostname == "" {
		return query.Where("(application.hostname IS NULL OR application.hostname = '')")
	}

	return query.Where("(application.hostname = ? OR application.hostname = ?)", hostname, "www."+hostname)
}

func (s *Service) minutesUsedInPeriod(appName, hostname string, durationMinutes int64) (int64, error) {
	if appName == "" || durationMinutes <= 0 {
		return 0, nil
	}

	cutoff := time.Now().Add(-time.Duration(durationMinutes) * time.Minute).Unix()
	query := s.scopedUsageIdentityQuery(appName, hostname).
		Where("application_usage.started_at >= ?", cutoff)

	var totalSeconds int64
	if err := query.Select("COALESCE(SUM(COALESCE(application_usage.duration_seconds, 0)), 0)").
		Scan(&totalSeconds).Error; err != nil {
		return 0, err
	}

	return totalSeconds / 60, nil
}

func (s *Service) populateCurrentUsageContext(ctx *sandboxContext) error {
	appName := ctx.Usage.Meta.AppName
	hostname := ctx.Usage.Meta.Host
	if appName == "" {
		return nil
	}

	var lastBlocked ApplicationUsage
	err := s.scopedUsageIdentityQuery(appName, hostname).
		Where("application_usage.enforcement_action = ?", EnforcementActionBlock).
		Order("application_usage.started_at DESC").
		Limit(1).
		First(&lastBlocked).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	if ctx.Usage.Insights.Current.Duration.SinceLastBlock == nil {
		minutesSinceLastBlock := int(time.Since(time.Unix(lastBlocked.StartedAt, 0)).Minutes())
		ctx.Usage.Insights.Current.Duration.SinceLastBlock = &minutesSinceLastBlock
	}

	if ctx.Usage.Insights.Current.Duration.LastBlocked == nil {
		lastBlockedDurationMinutes := 0
		if lastBlocked.DurationSeconds != nil {
			lastBlockedDurationMinutes = *lastBlocked.DurationSeconds / 60
		}
		ctx.Usage.Insights.Current.Duration.LastBlocked = &lastBlockedDurationMinutes
	}

	if ctx.Usage.Insights.Current.Duration.UsedSinceLastBlock == nil {
		var totalSeconds int64
		sumErr := s.scopedUsageIdentityQuery(appName, hostname).
			Where("application_usage.started_at > ?", lastBlocked.StartedAt).
			Select("COALESCE(SUM(COALESCE(application_usage.duration_seconds, 0)), 0)").
			Scan(&totalSeconds).Error
		if sumErr != nil {
			return sumErr
		}

		minutesUsedSinceLastBlock := int(totalSeconds / 60)
		ctx.Usage.Insights.Current.Duration.UsedSinceLastBlock = &minutesUsedSinceLastBlock
	}

	return nil
}

func (s *Service) populateInsightsContext(ctx *sandboxContext) error {
	now := time.Now()
	insights, err := s.GetDayInsights(now)
	if err != nil {
		return err
	}

	ctx.Usage.Insights.Today.ProductiveMinutes = insights.ProductivityScore.ProductiveSeconds / 60
	ctx.Usage.Insights.Today.DistractingMinutes = insights.ProductivityScore.DistractingSeconds / 60
	ctx.Usage.Insights.Today.FocusScore = insights.ProductivityScore.ProductivityScore

	hourly := insights.ProductivityPerHourBreakdown[now.Hour()]
	ctx.Usage.Insights.Hour.ProductiveMinutes = hourly.ProductiveSeconds / 60
	ctx.Usage.Insights.Hour.DistractingMinutes = hourly.DistractingSeconds / 60
	ctx.Usage.Insights.Hour.FocusScore = hourly.ProductivityScore

	todayKey := now.Format("2006-01-02")

	appName := ctx.Usage.Meta.AppName
	hostname := ctx.Usage.Meta.Host
	if appName == "" {
		return nil
	}

	var currentDistractingSeconds int64
	if err := s.scopedUsageIdentityQuery(appName, hostname).
		Where("date(application_usage.started_at, 'unixepoch') = ?", todayKey).
		Where("application_usage.classification = ?", ClassificationDistracting).
		Select("COALESCE(SUM(COALESCE(application_usage.duration_seconds, 0)), 0)").
		Scan(&currentDistractingSeconds).Error; err != nil {
		return err
	}
	ctx.Usage.Insights.Current.Duration.Today = int(currentDistractingSeconds / 60)

	var currentBlockedCount int64
	if err := s.scopedUsageIdentityQuery(appName, hostname).
		Where("date(application_usage.started_at, 'unixepoch') = ?", todayKey).
		Where("application_usage.enforcement_action = ?", EnforcementActionBlock).
		Count(&currentBlockedCount).Error; err != nil {
		return err
	}
	ctx.Usage.Insights.Current.Blocks.Count = int(currentBlockedCount)

	return nil
}
