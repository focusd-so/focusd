package usage

import (
	"context"
	"fmt"
	"strings"
)

type AppClassifier func(ctx context.Context, appName, title string, bundleID, appCategory *string) (*ClassificationResponse, error)

var classifiers = map[string]AppClassifier{
	"slack": classifySlackApp,
}

func (s *Service) classifyApplication(ctx context.Context, appName, title string, bundleID, appCategory *string) (*ClassificationResponse, error) {
	if classifier, ok := classifiers[strings.ToLower(appName)]; ok {
		return classifier(ctx, appName, title, bundleID, appCategory)
	}

	return classifyGenericApplication(ctx, appName, title, bundleID, appCategory)
}

func classifyGenericApplication(ctx context.Context, appName, title string, bundleID, appCategory *string) (*ClassificationResponse, error) {
	instructions := instructionGenericApplicationClassification

	var (
		inputTmpl = `
The user is currently using an application. Classify the activity based on the following information:

Application Name: %s
Window Title: %s
Bundle ID: %s
App Store Category: %s
`
	)

	bundleIDValue := fromPtr(bundleID)
	appCategoryValue := fromPtr(appCategory)

	input := fmt.Sprintf(inputTmpl, appName, title, bundleIDValue, appCategoryValue)

	response, err := classify(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

// classifySlackApp classifies Slack desktop application activity using the window title.
// Titles typically look like: "Slack | #engineering | Acme Corp" or "Slack - #random - Workspace"
func classifySlackApp(ctx context.Context, appName, title string, _, _ *string) (*ClassificationResponse, error) {

	return classifySlackActivity(ctx, "Analyse Slack desktop activity from the following title: "+title)
}
