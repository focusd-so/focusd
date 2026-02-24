package usage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/focusd-so/focusd/internal/settings"
)

func (s *Service) ClassifyCustomRules(ctx context.Context, appName string, executablePath string, url *string, nowTime *time.Time) (*ClassificationResponse, error) {
	if s.settingsService == nil {
		slog.Warn("settings service is nil, skipping custom rules classification")

		return nil, nil
	}

	slog.Info("classifying application usage with custom rules")

	sandboxCtx := createSandboxContext(appName, executablePath, url)

	if nowTime != nil {
		t := *nowTime
		sandboxCtx.Now = func(loc *time.Location) time.Time {
			return t.In(loc)
		}
	}

	return s.ClassifyCustomRulesWithSandbox(ctx, sandboxCtx)
}

func (s *Service) ClassifyCustomRulesWithSandbox(ctx context.Context, sandboxCtx sandboxContext) (*ClassificationResponse, error) {
	// Serialize the context to JSON
	contextJSON, err := json.Marshal(sandboxCtx)
	if err != nil {
		return nil, err
	}

	// Create a new execution log
	executionLog := SandboxExecutionLog{
		Context:   string(contextJSON),
		CreatedAt: time.Now().Unix(),
		Type:      string(ExecutionLogTypeClassification),
	}

	if err := s.db.Create(&executionLog).Error; err != nil {
		return nil, err
	}

	if sandboxCtx.Now == nil {
		sandboxCtx.Now = func(loc *time.Location) time.Time {
			return time.Now().In(loc)
		}
	}

	if sandboxCtx.MinutesUsedInPeriod == nil {
		sandboxCtx.MinutesUsedInPeriod = func(bundleID, hostname string, durationMinutes int64) (int64, error) {
			return 0, nil
		}
	}

	resp, logs, err := s.classifySandbox(ctx, sandboxCtx)

	if err != nil {
		errMsg := fmt.Errorf("failed to classify sandbox: %w", err).Error()
		executionLog.Error = &errMsg
	}

	if resp != nil {
		respJSON, err := json.Marshal(resp)
		if err != nil {
			errMsg := fmt.Errorf("failed to marshal response: %w", err).Error()
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
		return nil, err
	}
	executionLog.Logs = b.String()

	if err := s.db.Save(&executionLog).Error; err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, nil
	}

	return &ClassificationResponse{
		Classification:       Classification(resp.Classification),
		ClassificationSource: ClassificationSourceCustomRules,
		Reasoning:            resp.ClassificationReasoning,
		ConfidenceScore:      1.0,
		Tags:                 resp.Tags,

		SandboxContext:  executionLog.Context,
		SandboxResponse: executionLog.Response,
		SandboxLogs:     executionLog.Logs,
	}, nil
}

func (s *Service) classifySandbox(ctx context.Context, sandboxCtx sandboxContext) (desicion *classificationDecision, logs []string, err error) {
	// Get the latest custom rules code
	customRules, err := s.settingsService.GetLatest(settings.SettingsKeyCustomRules)
	if err != nil {
		return nil, nil, err
	}

	if customRules == nil || customRules.Value == "" {
		return nil, nil, nil
	}

	// Create a new sandbox with the custom rules code
	sb, err := newSandbox(customRules.Value)
	if err != nil {
		return nil, nil, err
	}

	return sb.invokeClassify(sandboxCtx)
}
