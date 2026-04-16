package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/focusd-so/focusd/internal/sandbox"
	"github.com/focusd-so/focusd/internal/settings"
)

// classificationResult is returned from the classify function.
type classificationResult struct {
	Classification          string   `json:"classification"`
	ClassificationReasoning string   `json:"classificationReasoning"`
	Tags                    []string `json:"tags"`
}

func (s *Service) ClassifyCustomRules(ctx context.Context, opts ...sandboxContextOption) (*CustomRulesClassificationResult, error) {
	sandboxCtx := s.createSandboxContext(opts...)

	contextJSON, err := json.Marshal(sandboxCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sandbox context: %w", err)
	}

	result, err := s.executeClassificationCustomRules(ctx, sandboxCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to classify sandbox: %w", err)
	}

	if result == nil || result.Output == "" || result.Output == "null" || result.Output == "undefined" {
		return nil, nil
	}

	var decision classificationResult
	if err := json.Unmarshal([]byte(result.Output), &decision); err != nil {
		return nil, fmt.Errorf("failed to parse classification decision: %w", err)
	}

	resp := CustomRulesClassificationResult{
		BasicClassificationResult: BasicClassificationResult{
			Classification:       Classification(decision.Classification),
			ClassificationReason: decision.ClassificationReasoning,
			Tags:                 decision.Tags,
		},
		SandboLogs:     result.Logs,
		SanboxOutput:   &result.Output,
		SandboxContext: string(contextJSON),
	}

	return &resp, nil
}

func (s *Service) executeClassificationCustomRules(ctx context.Context, sandboxCtx sandboxContext) (*sandbox.Result, error) {
	// Get the latest custom rules code
	customRules := settings.GetCustomRulesJS()
	if customRules == "" {
		return nil, nil
	}

	sb, err := sandbox.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox: %w", err)
	}
	defer sb.Close()

	return sb.Execute(customRules, "__classify_wrapper", sandboxCtx)
}

// TestClassifyCustomRules is exposed to Wails specifically for the Test Rules UI.
// It parses standard JSON arguments from the frontend and converts them into sandbox options.
func (s *Service) TestClassifyCustomRules(appName string, url *string, simulatedTimeISO *string) (*CustomRulesClassificationResult, error) {
	opts := []sandboxContextOption{
		WithAppNameContext(appName),
	}

	u, _, _ := parseURLNormalized(url)

	if u != nil {
		opts = append(opts, WithBrowserURLContext(u))
	}

	if simulatedTimeISO != nil && *simulatedTimeISO != "" {
		t, err := time.Parse(time.RFC3339, *simulatedTimeISO)
		if err == nil {
			opts = append(opts, WithNowContext(t))
		}
	}

	return s.ClassifyCustomRules(context.Background(), opts...)
}
