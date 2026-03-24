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
	appName := ctx.Usage.Metadata.AppName
	hostname := ctx.Usage.Metadata.Hostname
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

	if ctx.Usage.Insights.MinutesSinceLastBlock < 0 {
		ctx.Usage.Insights.MinutesSinceLastBlock = int(time.Since(time.Unix(lastBlocked.StartedAt, 0)).Minutes())
	}

	if ctx.Usage.Insights.LastBlockedDurationMinutes < 0 {
		if lastBlocked.DurationSeconds != nil {
			ctx.Usage.Insights.LastBlockedDurationMinutes = *lastBlocked.DurationSeconds / 60
		} else {
			ctx.Usage.Insights.LastBlockedDurationMinutes = 0
		}
	}

	if ctx.Usage.Insights.MinutesUsedSinceLastBlock < 0 {
		var totalSeconds int64
		sumErr := s.scopedUsageIdentityQuery(appName, hostname).
			Where("application_usage.started_at > ?", lastBlocked.StartedAt).
			Select("COALESCE(SUM(COALESCE(application_usage.duration_seconds, 0)), 0)").
			Scan(&totalSeconds).Error
		if sumErr != nil {
			return sumErr
		}

		ctx.Usage.Insights.MinutesUsedSinceLastBlock = int(totalSeconds / 60)
	}

	return nil
}

func (s *Service) populateInsightsContext(ctx *sandboxContext) error {
	now := time.Now()
	insights, err := s.GetDayInsights(now)
	if err != nil {
		return err
	}

	ctx.Insights.Today.ProductiveMinutes = insights.ProductivityScore.ProductiveSeconds / 60
	ctx.Insights.Today.DistractingMinutes = insights.ProductivityScore.DistractiveSeconds / 60
	ctx.Insights.Today.IdleMinutes = insights.ProductivityScore.IdleSeconds / 60
	ctx.Insights.Today.OtherMinutes = insights.ProductivityScore.OtherSeconds / 60
	ctx.Insights.Today.FocusScore = insights.ProductivityScore.ProductivityScore

	ctx.Insights.TopDistractions = insights.TopDistractions
	ctx.Insights.TopBlocked = insights.TopBlocked
	ctx.Insights.ProjectBreakdown = insights.ProjectBreakdown
	ctx.Insights.CommunicationBreakdown = insights.CommunicationBreakdown

	todayKey := now.Format("2006-01-02")

	var distractionCount int64
	if err := s.db.Model(&ApplicationUsage{}).
		Where("date(started_at, 'unixepoch') = ?", todayKey).
		Where("classification = ?", ClassificationDistracting).
		Count(&distractionCount).Error; err != nil {
		return err
	}
	ctx.Insights.Today.DistractionCount = int(distractionCount)

	var blockedCount int64
	if err := s.db.Model(&ApplicationUsage{}).
		Where("date(started_at, 'unixepoch') = ?", todayKey).
		Where("enforcement_action = ?", EnforcementActionBlock).
		Count(&blockedCount).Error; err != nil {
		return err
	}
	ctx.Insights.Today.BlockedCount = int(blockedCount)

	appName := ctx.Usage.Metadata.AppName
	hostname := ctx.Usage.Metadata.Hostname
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
	ctx.Usage.Insights.DistractingMinutes = int(currentDistractingSeconds / 60)

	var currentBlockedCount int64
	if err := s.scopedUsageIdentityQuery(appName, hostname).
		Where("date(application_usage.started_at, 'unixepoch') = ?", todayKey).
		Where("application_usage.enforcement_action = ?", EnforcementActionBlock).
		Count(&currentBlockedCount).Error; err != nil {
		return err
	}
	ctx.Usage.Insights.BlockedCount = int(currentBlockedCount)

	return nil
}
