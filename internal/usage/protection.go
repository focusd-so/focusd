package usage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/focusd-so/focusd/internal/identity"
	"github.com/focusd-so/focusd/internal/sandbox"
	"github.com/focusd-so/focusd/internal/settings"
	"github.com/focusd-so/focusd/internal/timeline"
)

// enforcement is returned from the enforcement function.
type enforcement struct {
	EnforcementAction string `json:"enforcementAction"`
	EnforcementReason string `json:"enforcementReason"`
}

type PauseProtectionPayload struct {
	ResumeReason string `json:"resume_reason"`
	PauseReason  string `json:"pause_reason"`
}

type AllowUsagePayload struct {
	AppName  string `json:"app_name,omitempty"`
	URL      string `json:"url,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

func (s *Service) ProtectionPause(durationSeconds int, reason string) error {
	dur := time.Duration(durationSeconds) * time.Second
	willEndAt := time.Now().Add(dur)

	event, err := s.timelineService.GetActiveEventOfType(EventTypeProtectionPause)
	if err != nil {
		return err
	}

	if event != nil {
		event.EndedAt = withPtr(willEndAt.Unix())

		if err := s.timelineService.UpdateEvent(event); err != nil {
			return fmt.Errorf("updating event: %w", err)
		}

		return nil
	}

	_, err = s.timelineService.CreateEvent(
		EventTypeProtectionPause,
		timeline.WithEndedAt(willEndAt),
		timeline.WithPayload(PauseProtectionPayload{PauseReason: reason}),
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) ProtectionResume(reason string) error {
	event, err := s.timelineService.GetActiveEventOfType(EventTypeProtectionPause)
	if err != nil {
		return err
	}

	// no active protection pause to resume
	if event == nil {
		return nil
	}

	event.EndedAt = withPtr(time.Now().Unix())

	if s.timelineService.UpdateEvent(event); err != nil {
		return fmt.Errorf("updating event: %w", err)
	}

	return nil
}

func (s *Service) ProtectionGetStatus() (*timeline.Event, error) {
	return s.timelineService.GetActiveEventOfType(EventTypeProtectionPause)
}

func (s *Service) PauseGetHistory(days int) ([]*timeline.Event, error) {
	return s.timelineService.ListEvents(
		timeline.ByTypes(EventTypeProtectionPause),
		timeline.ByAge(days),
	)
}

func (s *Service) AllowApp(appname string, duration time.Duration) error {
	return s.allowUsage(AllowUsagePayload{AppName: appname}, duration)
}

func (s *Service) AllowURL(rawURL string, duration time.Duration) error {
	parsed, _ := parseURLNormalized(rawURL)
	return s.allowUsage(AllowUsagePayload{URL: parsed.String()}, duration)
}

func (s *Service) AllowHostname(rawURL string, duration time.Duration) error {
	parsed, _ := parseURLNormalized(rawURL)
	return s.allowUsage(AllowUsagePayload{Hostname: parsed.Hostname()}, duration)
}

func (s *Service) allowUsage(req AllowUsagePayload, duration time.Duration) error {
	allowed, err := s.timelineService.ListEvents(
		timeline.ByTypes(EventTypeAllowUsage),
		timeline.ActiveOnly(),
	)

	if err != nil {
		return err
	}

	willEndAt := time.Now().Add(duration)

	for _, event := range allowed {
		payload := AllowUsagePayload{}
		if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
			return err
		}

		if payload.AppName == req.AppName && payload.URL == req.URL && payload.Hostname == req.Hostname {
			event.EndedAt = withPtr(willEndAt.Unix())
			if err := s.timelineService.UpdateEvent(event); err != nil {
				return fmt.Errorf("updating event: %w", err)
			}
			return nil
		}
	}

	_, err = s.timelineService.CreateEvent(
		EventTypeAllowUsage,
		timeline.WithEndedAt(willEndAt),
		timeline.WithPayload(req),
	)

	return err
}

func (s *Service) AllowGetAll() ([]*timeline.Event, error) {
	return s.timelineService.ListEvents(
		timeline.ByTypes(EventTypeAllowUsage),
		timeline.ActiveOnly(),
	)
}

func (s *Service) Whitelist(appname string, url string, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}

	now := time.Now().Unix()
	expiresAt := now + int64(duration.Seconds())

	var hostname string
	if normalized, err := parseURLNormalized(url); err == nil && normalized != nil {
		hostname = normalized.Hostname()
	} else if url != "" {
		hostname = strings.ToLower(strings.TrimSpace(url))
		hostname = strings.TrimSuffix(hostname, ".")
		hostname = strings.TrimPrefix(hostname, "www.")
	}

	// delete any existing whitelist entries for the app and hostname
	if hostname == "" {
		if err := s.db.Where("app_name = ? AND (hostname IS NULL OR hostname = '')", appname).Delete(&ProtectionWhitelist{}).Error; err != nil {
			return err
		}
	} else {
		if err := s.db.Where("hostname = ?", hostname).Delete(&ProtectionWhitelist{}).Error; err != nil {
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

// CalculateEnforcementDecision determines whether usage should be blocked, allowed, or paused.
//
// This function evaluates multiple factors in order of priority:
// 1. Custom rules (if configured) - highest priority
// 2. Classification - non-distracting usage is always allowed
// 3. Protection pause status - if protection is paused, usage is allowed
// 4. Whitelist entries - temporarily whitelisted bundle ID/hostname combinations are allowed
// 5. Default blocking - distracting usage is blocked when protection is active
//
// Parameter:
//   - appUsage: Usage details for the current app or site event
//
// Returns:
//   - EnforcementDecision: A decision containing the action (Allow/Block/Paused), reasoning, and source
//   - error: Database error if protection status or whitelist lookup fails
func (s *Service) CalculateEnforcementDecision(ctx context.Context, appUsage *ApplicationUsage) (EnforcementDecision, error) {
	classification := appUsage.Classification

	customRulesDecision, err := s.calculateEnforcementDecisionWithCustomRules(ctx, appUsage)
	if err != nil {
		return EnforcementDecision{}, fmt.Errorf("failed to calculate enforcement decision with custom rules: %w", err)
	}

	if customRulesDecision.Action != "" && customRulesDecision.Action != EnforcementActionNone {
		tier := identity.GetAccountTier()
		if hasCustomRulesExecutionAccess(tier) {
			return customRulesDecision, nil
		}
	}

	if classification != ClassificationDistracting {
		return EnforcementDecision{
			Action: EnforcementActionAllow,
			Reason: "non distracting usage",
			Source: EnforcementSourceApplication,
		}, nil
	}

	protectionPause, err := s.ProtectionGetStatus()
	if err != nil {
		return EnforcementDecision{}, err
	}

	if protectionPause.ID > 0 {
		return EnforcementDecision{
			Action: EnforcementActionAllow,
			Reason: "focus protection has been paused by the user",
			Source: EnforcementSourcePaused,
		}, nil
	}

	// get all whitelist entries for the bundle ID and hostname
	var whitelist ProtectionWhitelist
	hostname := fromPtr(appUsage.Application.Hostname)
	query := s.db.Where("expires_at > ?", time.Now().Unix())
	if hostname == "" {
		query = query.Where("app_name = ? AND (hostname IS NULL OR hostname = '')", appUsage.Application.Name)
	} else {
		query = query.Where("(hostname = ? OR hostname = ?)", hostname, "www."+hostname)
	}

	if err := query.Limit(1).First(&whitelist).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return EnforcementDecision{}, err
		}
	}

	if whitelist.ID > 0 {
		return EnforcementDecision{
			Action: EnforcementActionAllow,
			Reason: "temporarily allowed usage by user",
			Source: EnforcementSourceWhitelist,
		}, nil
	}

	return EnforcementDecision{
		Action: EnforcementActionBlock,
		Reason: "distracting usage, focus protection is enabled",
		Source: EnforcementSourceApplication,
	}, nil
}

func (s *Service) calculateEnforcementDecisionWithCustomRules(_ context.Context, appUsage *ApplicationUsage) (EnforcementDecision, error) {
	sandboxCtx := createSandboxContext(appUsage.Application.Name, appUsage.BrowserURL)
	sandboxCtx.Usage.Meta.Title = appUsage.WindowTitle
	sandboxCtx.Usage.Meta.Classification = string(appUsage.Classification)

	customRules := settings.GetCustomRulesJS()
	if customRules == "" {
		return EnforcementDecision{Action: EnforcementActionNone}, nil
	}

	s.enrichSandboxContext(&sandboxCtx)

	contextJSON, err := json.Marshal(sandboxCtx)
	if err != nil {
		return EnforcementDecision{}, err
	}

	executionLog := SandboxExecutionLog{
		Context:   string(contextJSON),
		CreatedAt: time.Now().Unix(),
		Type:      string(ExecutionLogTypeEnforcementAction),
	}

	if err := s.db.Create(&executionLog).Error; err != nil {
		return EnforcementDecision{}, err
	}

	finalizeExecutionLog := func(decision *enforcement, logs []string, invokeErr error) error {
		if invokeErr != nil {
			errMsg := invokeErr.Error()
			executionLog.Error = &errMsg
		}

		if decision != nil {
			respJSON, marshalErr := json.Marshal(decision)
			if marshalErr != nil {
				errMsg := fmt.Errorf("failed to marshal response: %w", marshalErr).Error()
				executionLog.Error = &errMsg
			} else {
				respJSONStr := string(respJSON)
				executionLog.Response = &respJSONStr
			}
		} else {
			txt := "no response"
			executionLog.Response = &txt
		}

		finishedAt := time.Now().Unix()
		executionLog.FinishedAt = &finishedAt

		b := new(bytes.Buffer)
		if err := json.NewEncoder(b).Encode(logs); err != nil {
			return err
		}
		executionLog.Logs = b.String()

		appUsage.EnforcementSandboxContext = withPtr(executionLog.Context)
		appUsage.EnforcementSandboxResponse = executionLog.Response
		appUsage.EnforcementSandboxLogs = withPtr(executionLog.Logs)

		if err := s.db.Save(&executionLog).Error; err != nil {
			return err
		}

		return nil
	}

	// Create a new sandbox with the custom rules code
	sb, err := sandbox.New()
	if err != nil {
		if logErr := finalizeExecutionLog(nil, nil, err); logErr != nil {
			return EnforcementDecision{}, logErr
		}
		return EnforcementDecision{}, err
	}
	defer sb.Close()

	execResult, err := sb.Execute(customRules, "__enforcement_wrapper", sandboxCtx)
	if err != nil {
		if logErr := finalizeExecutionLog(nil, execResult.Logs, err); logErr != nil {
			return EnforcementDecision{}, logErr
		}
		return EnforcementDecision{}, err
	}

	if execResult.Output == "" || execResult.Output == "null" || execResult.Output == "undefined" {
		if logErr := finalizeExecutionLog(nil, execResult.Logs, nil); logErr != nil {
			return EnforcementDecision{}, logErr
		}
		return EnforcementDecision{Action: EnforcementActionNone}, nil
	}

	var decision enforcement
	if err := json.Unmarshal([]byte(execResult.Output), &decision); err != nil {
		if logErr := finalizeExecutionLog(nil, execResult.Logs, err); logErr != nil {
			return EnforcementDecision{}, logErr
		}
		return EnforcementDecision{}, err
	}

	if logErr := finalizeExecutionLog(&decision, execResult.Logs, nil); logErr != nil {
		return EnforcementDecision{}, logErr
	}

	return EnforcementDecision{
		Action: EnforcementAction(decision.EnforcementAction),
		Reason: EnforcementReason(decision.EnforcementReason),
		Source: EnforcementSourceCustomRules,
	}, nil
}
