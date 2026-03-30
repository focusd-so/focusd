package usage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/focusd-so/focusd/internal/sandbox"
	"github.com/focusd-so/focusd/internal/settings"
	v8 "rogchap.com/v8go"
)

// classificationResult is returned from the classify function.
type classificationResult struct {
	Classification          string   `json:"classification"`
	ClassificationReasoning string   `json:"classificationReasoning"`
	Tags                    []string `json:"tags"`
}

func (s *Service) ClassifyCustomRules(ctx context.Context, opts ...sandboxContextOption) (*ClassificationResponse, error) {
	slog.Info("classifying application usage with custom rules")

	return s.classifyCustomRulesWithSandbox(ctx, NewSandboxContext(opts...))
}

func (s *Service) classifyCustomRulesWithSandbox(ctx context.Context, sandboxCtx sandboxContext) (*ClassificationResponse, error) {
	s.enrichSandboxContext(&sandboxCtx)

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

func (s *Service) classifySandbox(ctx context.Context, sandboxCtx sandboxContext) (decision *classificationResult, logs []string, err error) {
	// Get the latest custom rules code
	customRules := settings.GetCustomRulesJS()
	if customRules == "" {
		return nil, nil, nil
	}

	sb, err := sandbox.New()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create sandbox: %w", err)
	}
	defer sb.Close()

	if sandboxCtx.MinutesUsedInPeriod != nil {
		err = sb.RegisterGlobal("__minutesUsedInPeriod", func(info *v8.FunctionCallbackInfo) *v8.Value {
			args := info.Args()
			if len(args) < 3 {
				val, _ := v8.NewValue(info.Context().Isolate(), int32(0))
				return val
			}

			appName := args[0].String()
			hostname := args[1].String()
			minutes := int64(args[2].Integer())

			result, err := sandboxCtx.MinutesUsedInPeriod(appName, hostname, minutes)
			if err != nil {
				slog.Debug("failed to query minutes used", "error", err)
				val, _ := v8.NewValue(info.Context().Isolate(), int32(0))
				return val
			}

			val, _ := v8.NewValue(info.Context().Isolate(), int32(result))
			return val
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to register minutes query func: %w", err)
		}
	}

	result, err := sb.Execute(customRules, "__classify_wrapper", sandboxCtx)
	if err != nil {
		return nil, result.Logs, err
	}

	if result.Output == "" || result.Output == "null" || result.Output == "undefined" {
		return nil, result.Logs, nil
	}

	var d classificationResult
	if err := json.Unmarshal([]byte(result.Output), &d); err != nil {
		return nil, result.Logs, fmt.Errorf("failed to parse classification decision: %w", err)
	}

	return &d, result.Logs, nil
}
