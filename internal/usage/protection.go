package usage

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
	"github.com/focusd-so/focusd/internal/identity"
	"github.com/focusd-so/focusd/internal/settings"
)

// PauseProtection temporarily disables focus protection for the specified duration.
//
// The function starts a background goroutine that automatically resumes protection
// after the duration expires. The pause state is persisted to the database, allowing
// the application to recover pause state across restarts.
//
// Parameters:
//   - duration: The duration to pause protection (must be > 0)
//
// Returns:
//   - error: Returns an error if protection is already paused
//
// Side effects:
//   - Creates a ProtectionPause record in the database
//   - Emits a state update via the state channel
//   - Spawns a goroutine that calls ResumeProtection after the duration
func (s *Service) PauseProtection(durationSeconds int, reason string) (ProtectionPause, error) {
	protectionPause, err := s.GetProtectionStatus()
	if err != nil {
		return ProtectionPause{}, err
	}

	if protectionPause.ID != 0 {
		return protectionPause, nil
	}

	now := time.Now().Unix()
	projectedResumedAt := now + int64(durationSeconds)

	protectionPause = ProtectionPause{
		RequestedDurationSeconds: durationSeconds,
		ResumedAt:                projectedResumedAt,
		CreatedAt:                now,
		ActualDurationSeconds:    durationSeconds,
		ResumedReason:            fmt.Sprintf("protection paused for %ds expired", durationSeconds),
		Reason:                   reason,
	}

	if err := s.db.Create(&protectionPause).Error; err != nil {
		return ProtectionPause{}, err
	}

	s.eventsMu.RLock()
	for _, fn := range s.onProtectionPaused {
		fn(protectionPause)
	}
	s.eventsMu.RUnlock()

	return protectionPause, nil
}

// ResumeProtection re-enables focus protection and records the reason for resumption.
//
// This function is called either automatically when a pause duration expires,
// or manually by the user to end a pause early. The reason is persisted to the
// database for auditing and analytics purposes.
//
// Parameters:
//   - reason: A human-readable explanation for why protection was resumed
//     (e.g., "protection paused for 5m0s expired" or "user manually resumed")
//
// Returns:
//   - error: Returns an error if protection is not currently paused
//
// Side effects:
//   - Updates the ProtectionPause record in the database with ResumedAt timestamp
//   - Clears the pause state and emits a state update via the state channel
func (s *Service) ResumeProtection(reason string) (ProtectionPause, error) {
	protectionPause, err := s.GetProtectionStatus()
	if err != nil {
		return ProtectionPause{}, err
	}

	if protectionPause.ID == 0 {
		return ProtectionPause{}, fmt.Errorf("protection not paused")
	}

	now := time.Now().Unix()

	// precalculate the resumed at timestamp
	protectionPause.ResumedAt = now
	protectionPause.ResumedReason = reason
	protectionPause.ActualDurationSeconds = int(now - protectionPause.CreatedAt)

	if err := s.db.Save(protectionPause).Error; err != nil {
		return ProtectionPause{}, err
	}

	s.eventsMu.RLock()
	for _, fn := range s.onProtectionResumed {
		fn(protectionPause)
	}
	s.eventsMu.RUnlock()

	return protectionPause, nil
}

// GetProtectionStatus retrieves the current protection pause status.
//
// It queries for an active ProtectionPause record where ResumedAt is greater than
// the current time (indicating the pause is still active and hasn't been resumed yet).
// When a pause is created, ResumedAt is set to a future timestamp representing when
// the pause should automatically expire. When manually resumed, ResumedAt is updated
// to the current time, making it no longer match this query.
//
// Returns:
//   - ProtectionPause: The active pause record if protection is currently paused,
//     or an empty ProtectionPause (ID == 0) if protection is active (not paused)
//   - error: Database error if the query fails
func (s *Service) GetProtectionStatus() (ProtectionPause, error) {
	var protectionPause ProtectionPause
	if err := s.db.Where("resumed_at > ?", time.Now().Unix()).Limit(1).First(&protectionPause).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ProtectionPause{}, nil
		}
		return ProtectionPause{}, err
	}

	return protectionPause, nil
}

// GetPauseHistory retrieves the history of protection pauses within the specified number of days.
//
// Parameters:
//   - days: The number of days to look back (e.g., 7 for one week)
//
// Returns:
//   - []ProtectionPause: A slice of pause records ordered by creation time (newest first)
//   - error: Database error if the query fails
func (s *Service) GetPauseHistory(days int) ([]ProtectionPause, error) {
	var pauses []ProtectionPause

	cutoff := time.Now().AddDate(0, 0, -days).Unix()

	if err := s.db.Where("created_at >= ?", cutoff).Order("created_at DESC").Find(&pauses).Error; err != nil {
		return nil, err
	}

	return pauses, nil
}

// Whitelist temporarily allows a specific blocked usage (by bundle ID and hostname) for the specified duration.
//
// This function creates a ProtectionWhitelist entry that allows the specified application or website
// to bypass focus protection for a limited time. This enables users to temporarily access blocked
// content without pausing all protection. The whitelist entry is persisted to the database and can
// be checked during termination decision evaluation.
//
// Parameters:
//   - bundleID: The application bundle identifier (e.g., "com.example.app")
//   - hostname: The website hostname (e.g., "example.com") - empty string for non-browser apps
//   - duration: The duration to allow the usage (defaults to 1 hour if 0)
//
// Returns:
//   - error: Database error if the whitelist entry creation fails
//
// Side effects:
//   - Creates a ProtectionWhitelist record in the database with expiration timestamp
func (s *Service) Whitelist(appname string, hostname string, duration time.Duration) error {
	if duration < 5*time.Minute {
		return fmt.Errorf("duration must be at least 5 minutes")
	}

	hostname = normalizeHostname(hostname)

	now := time.Now().Unix()
	expiresAt := now + int64(duration.Seconds())

	if duration == 0 {
		duration = time.Hour
	}

	// delete any existing whitelist entries for the app and hostname
	if hostname == "" {
		if err := s.db.Where("app_name = ? AND (hostname IS NULL OR hostname = '')", appname).Delete(&ProtectionWhitelist{}).Error; err != nil {
			return err
		}
	} else {
		if err := s.db.Where("app_name = ? AND (hostname = ? OR hostname = ?)", appname, hostname, "www."+hostname).Delete(&ProtectionWhitelist{}).Error; err != nil {
			return err
		}
	}

	whitelist := ProtectionWhitelist{
		AppName:   appname,
		ExpiresAt: expiresAt,
	}

	if hostname != "" {
		whitelist.Hostname = &hostname
	}

	return s.db.Create(&whitelist).Error
}

// GetWhitelist returns all active whitelist entries that haven't expired.
//
// Returns:
//   - []ProtectionWhitelist: A slice of active whitelist entries
//   - error: Database error if the query fails
func (s *Service) GetWhitelist() ([]ProtectionWhitelist, error) {
	var whitelist []ProtectionWhitelist
	now := time.Now().Unix()
	if err := s.db.Where("expires_at > ? OR expires_at = 0", now).Find(&whitelist).Error; err != nil {
		return nil, err
	}
	return whitelist, nil
}

// RemoveWhitelist removes a whitelist entry by ID.
//
// Parameters:
//   - id: The ID of the whitelist entry to remove
//
// Returns:
//   - error: Database error if the deletion fails
func (s *Service) RemoveWhitelist(id int64) error {
	return s.db.Delete(&ProtectionWhitelist{}, id).Error
}

// CalculateTerminationMode determines whether an application or website should be blocked, allowed, or paused based on classification, custom rules, protection status, and whitelist entries.
//
// This function evaluates multiple factors in order of priority:
// 1. Custom rules (if configured) - highest priority
// 2. Classification - non-distracting usage is always allowed
// 3. Protection pause status - if protection is paused, usage is allowed
// 4. Whitelist entries - temporarily whitelisted bundle ID/hostname combinations are allowed
// 5. Default blocking - distracting usage is blocked when protection is active
//
// Parameters:
//   - bundleID: The application bundle identifier (e.g., "com.example.app")
//   - hostname: The website hostname (e.g., "example.com") - empty string for non-browser apps
//   - domain: The domain name extracted from the URL
//   - url: The full URL being accessed
//   - classification: The classification result indicating whether the usage is distracting
//   - terminationMode: The requested termination mode (may be overridden by custom rules)
//
// Returns:
//   - TerminationDecision: A decision containing the mode (Allow/Block/Paused), reasoning, and source
//   - error: Database error if protection status or whitelist lookup fails
func (s *Service) CalculateTerminationMode(ctx context.Context, appUsage *ApplicationUsage) (TerminationDecision, error) {
	classification := appUsage.Classification

	customRulesDecision, err := s.calculateTerminationModeWithCustomRules(ctx, appUsage)
	if err != nil {
		return TerminationDecision{}, fmt.Errorf("failed to calculate termination mode with custom rules: %w", err)
	}

	if customRulesDecision.Mode != "" && customRulesDecision.Mode != TerminationModeNone {
		tier := identity.GetAccountTier()
		if tier != apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE {
			return customRulesDecision, nil
		}
	}

	if classification != ClassificationDistracting {
		return TerminationDecision{
			Mode:      TerminationModeAllow,
			Reasoning: "non distracting usage",
			Source:    TerminationModeSourceApplication,
		}, nil
	}

	protectionPause, err := s.GetProtectionStatus()
	if err != nil {
		return TerminationDecision{}, err
	}

	if protectionPause.ID > 0 {
		return TerminationDecision{
			Mode:      TerminationModePaused,
			Reasoning: "focus protection has been paused by the user",
			Source:    TerminationModeSourcePaused,
		}, nil
	}

	// get all whitelist entries for the bundle ID and hostname
	var whitelist ProtectionWhitelist
	hostname := normalizeHostname(fromPtr(appUsage.Application.Hostname))
	query := s.db.Where("app_name = ? AND expires_at > ?", appUsage.Application.Name, time.Now().Unix())
	if hostname == "" {
		query = query.Where("(hostname IS NULL OR hostname = '')")
	} else {
		query = query.Where("(hostname = ? OR hostname = ?)", hostname, "www."+hostname)
	}

	if err := query.Limit(1).First(&whitelist).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return TerminationDecision{}, err
		}
	}

	if whitelist.ID > 0 {
		return TerminationDecision{
			Mode:      TerminationModeAllow,
			Reasoning: "temporarily allowed usage by user",
			Source:    TerminationModeSourceWhitelist,
		}, nil
	}

	return TerminationDecision{
		Mode:      TerminationModeBlock,
		Reasoning: "distracting usage, focus protection is enabled",
		Source:    TerminationModeSourceApplication,
	}, nil
}

func (s *Service) calculateTerminationModeWithCustomRules(_ context.Context, appUsage *ApplicationUsage) (TerminationDecision, error) {
	sandboxCtx := createSandboxContext(appUsage.Application.Name, appUsage.BrowserURL)
	sandboxCtx.Classification = string(appUsage.Classification)

	customRules := settings.GetCustomRulesJS()
	if customRules == "" {
		return TerminationDecision{Mode: TerminationModeNone}, nil
	}

	// Create a new sandbox with the custom rules code
	sb, err := newSandbox(customRules)
	if err != nil {
		return TerminationDecision{}, err
	}

	decision, err := sb.invokeTerminationMode(sandboxCtx)
	if err != nil {
		return TerminationDecision{}, err
	}

	if decision == nil {
		return TerminationDecision{Mode: TerminationModeNone}, nil
	}

	return TerminationDecision{
		Mode:      TerminationMode(decision.TerminationMode),
		Reasoning: decision.TerminationReasoning,
		Source:    TerminationModeSourceCustomRules,
	}, nil
}
