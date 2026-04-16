package usage

import (
	"context"
	"fmt"
	"strings"
)

type AppClassifier func(ctx context.Context, appName, title string, appCategory *string) (*LLMClassificationResult, error)

var classifiers = map[string]AppClassifier{
	"slack": classifySlackApp,
}

func (s *Service) classifyApplication(ctx context.Context, appName, title string, appCategory *string) (*LLMClassificationResult, error) {
	if classifier, ok := classifiers[strings.ToLower(appName)]; ok {
		return classifier(ctx, appName, title, appCategory)
	}

	return classifyGenericApplication(ctx, appName, title, appCategory)
}

func classifyGenericApplication(ctx context.Context, appName, title string, appCategory *string) (*LLMClassificationResult, error) {
	instructions := instructionGenericApplicationClassification

	var (
		inputTmpl = `
The user is currently using an application. Classify the activity based on the following information:

Application Name: %s
Window Title: %s
App Store Category: %s
`
	)

	appCategoryValue := fromPtr(appCategory)

	input := fmt.Sprintf(inputTmpl, appName, title, appCategoryValue)

	response, err := classify(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

// classifySlackApp classifies Slack desktop application activity using the window title.
// Titles typically look like: "Slack | #engineering | Acme Corp" or "Slack - #random - Workspace"
func classifySlackApp(ctx context.Context, appName, title string, _ *string) (*LLMClassificationResult, error) {
	return classifySlackActivity(ctx, "Analyse Slack desktop activity from the following title: "+title)
}
