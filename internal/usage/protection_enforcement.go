package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/focusd-so/focusd/internal/sandbox"
	"github.com/focusd-so/focusd/internal/settings"
	"github.com/focusd-so/focusd/internal/timeline"
)

type BasicEnforcementResult struct {
	Action EnforcementAction
	Reason string
	Source EnforcementSource
}

type CustomRulesEnforcementResult struct {
	BasicEnforcementResult

	// Custom rules classification only
	SandboLogs     []string `json:"-"`
	SandboxContext *string  `json:"-"`
	SanboxOutput   *string  `json:"-"`
	SandboxError   *string  `json:"-"`
}

type EnforcementResult struct {
	CustomRulesEnforcementResult *CustomRulesEnforcementResult
	StandardEnforcementResult    *BasicEnforcementResult
}

func (s *Service) CalculateEnforcement(
	ctx context.Context,
	appName, windowTitle string,
	browserURL *url.URL,
	classification Classification,
) (EnforcementResult, error) {
	// global pause takes precedence over any other decision.
	if paused, _ := s.isProtectionPaused(); paused {
		return EnforcementResult{
			StandardEnforcementResult: &BasicEnforcementResult{
				Action: EnforcementActionPaused,
				Reason: "focus protection is temporarily paused by user",
				Source: EnforcementSourcePaused,
			},
		}, nil
	}

	// when a usage is explicitly allowed, allow it!
	if allowed, _ := s.isAllowed(appName, browserURL); allowed {
		return EnforcementResult{
			StandardEnforcementResult: &BasicEnforcementResult{
				Action: EnforcementActionAllow,
				Reason: "temporarily allowed usage by user",
				Source: EnforcementSourceAllowed,
			},
		}, nil
	}

	sandboxCtx := s.createSandboxContext(
		WithAppNameContext(appName),
		WithWindowTitleContext(windowTitle),
		WithBrowserURLContext(browserURL),
		WithClassificationContext(classification),
	)

	customRulesDecision, err := s.calculateEnforcementCustomRules(sandboxCtx)
	if err != nil {
		slog.Warn("failed to calculate enforcement decision with custom rules", "error", err)
	}

	standardEnforcementResult := &BasicEnforcementResult{
		Action: EnforcementActionBlock,
		Reason: "distracting usage, focus protection is enabled",
		Source: EnforcementSourceApplication,
	}

	if classification != ClassificationDistracting {
		standardEnforcementResult = &BasicEnforcementResult{
			Action: EnforcementActionAllow,
			Reason: "non distracting usage",
			Source: EnforcementSourceApplication,
		}
	}

	return EnforcementResult{
		CustomRulesEnforcementResult: customRulesDecision,
		StandardEnforcementResult:    standardEnforcementResult,
	}, nil
}

func (s *Service) isProtectionPaused() (bool, error) {
	event, err := s.timelineService.GetActiveEventOfType(EventTypeProtectionStatusChanged)
	if err != nil {
		return false, fmt.Errorf("failed to get active protection pause event: %w", err)
	}

	return event != nil, nil
}

func (s *Service) isAllowed(appName string, browserURL *url.URL) (bool, error) {
	events, err := s.timelineService.ListEvents(
		timeline.ByTypes(EventTypeAllowUsage),
		timeline.ActiveOnly(),
	)
	if err != nil {
		return false, fmt.Errorf("failed to list allowed events: %w", err)
	}

	payloads, err := timeline.UnmarshalPayloads[AllowUsagePayload](events)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal allowed event payloads: %w", err)
	}

	for _, payload := range payloads {
		if payload.AppName != "" && payload.AppName == appName {
			return true, nil
		}
		if payload.URL != "" && browserURL != nil && payload.URL == browserURL.String() {
			return true, nil
		}
		if payload.Hostname != "" && browserURL != nil && payload.Hostname == browserURL.Hostname() {
			return true, nil
		}
	}

	return false, nil
}

func (s *Service) calculateEnforcementCustomRules(sandboxCtx sandboxContext) (*CustomRulesEnforcementResult, error) {
	contextJSON, err := json.Marshal(sandboxCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sandbox context: %w", err)
	}

	customRulesEvent, err := s.timelineService.CreateEvent(
		EventTypeCustomRulesTrace,
		timeline.WithPayload(CustomRulesTracePayload{
			Context: string(contextJSON),
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create custom rules trace event: %w", err)
	}

	customRulesTracePayload := CustomRulesTracePayload{
		Context: string(contextJSON),
	}

	result, err := s.executeEnforcementCustomRules(sandboxCtx)

	if result != nil {
		customRulesTracePayload.Logs = result.Logs
		customRulesTracePayload.Output = result.Output
	}

	defer func() {
		s.timelineService.UpdateEvent(
			&customRulesEvent,
			timeline.WithPayload(customRulesTracePayload),
			timeline.WithFinishedAt(time.Now()),
		)
	}()

	if err != nil {
		customRulesTracePayload.Error = err.Error()

		return nil, err
	}

	if result == nil || result.Output == "" || result.Output == "null" || result.Output == "undefined" {
		return nil, nil
	}

	customRulesTracePayload.Logs = result.Logs
	customRulesTracePayload.Output = result.Output

	var decision struct {
		Action string `json:"enforcementAction"`
		Reason string `json:"enforcementReason"`
	}
	if err := json.Unmarshal([]byte(result.Output), &decision); err != nil {
		return nil, fmt.Errorf("failed to unmarshal enforcement decision: %w", err)
	}

	return &CustomRulesEnforcementResult{
		BasicEnforcementResult: BasicEnforcementResult{
			Action: EnforcementAction(decision.Action),
			Reason: decision.Reason,
			Source: EnforcementSourceCustomRules,
		},
	}, nil
}

func (s *Service) executeEnforcementCustomRules(sandboxCtx sandboxContext) (*sandbox.Result, error) {
	customRules := settings.GetCustomRulesJS()
	if customRules == "" {
		return nil, nil
	}

	sb, err := sandbox.New()
	if err != nil {
		return nil, err
	}
	defer sb.Close()

	return sb.Execute(customRules, "__enforcement_wrapper", sandboxCtx)
}
