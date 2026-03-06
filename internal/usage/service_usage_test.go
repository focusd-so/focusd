package usage_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/focusd-so/focusd/internal/usage"
)

func setUpService(t *testing.T, options ...usage.Option) (*usage.Service, *gorm.DB) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	service, err := usage.NewService(ctx, db, options...)
	require.NoError(t, err, "failed to create service")

	return service, db
}

func TestService_WhenEnterIdle_ContinueCurrentIdlePeriod(t *testing.T) {
	service, db := setUpService(t)

	// create a new idle period
	idlePeriod := usage.IdlePeriod{
		StartedAt: time.Now().Unix(),
	}
	if err := db.Create(&idlePeriod).Error; err != nil {
		t.Fatalf("failed to create idle period: %v", err)
	}

	// wait 3 seconds
	time.Sleep(3 * time.Second)

	// enter idle
	err := service.IdleChanged(context.Background(), true)
	require.NoError(t, err, "failed to enter idle")

	// read the idle period
	var readIdlePeriod usage.IdlePeriod
	if err := db.Where("id = ?", idlePeriod.ID).First(&readIdlePeriod).Error; err != nil {
		t.Fatalf("failed to find idle period: %v", err)
	}

	require.NotEqual(t, int64(0), readIdlePeriod.StartedAt)
	require.Equal(t, readIdlePeriod.ID, idlePeriod.ID)
	require.Nil(t, readIdlePeriod.EndedAt)
	require.Nil(t, readIdlePeriod.DurationSeconds)
}

func TestService_WhenEnterIdle_CreateNewIdlePeriod(t *testing.T) {
	service, db := setUpService(t)

	now := time.Now().Unix()
	expectedDurationSeconds := 3

	// create a closed idle period
	closedIdlePeriod := usage.IdlePeriod{
		StartedAt:       time.Now().Unix(),
		EndedAt:         &now,
		DurationSeconds: &expectedDurationSeconds,
	}
	if err := db.Create(&closedIdlePeriod).Error; err != nil {
		t.Fatalf("failed to create closed idle period: %v", err)
	}

	// enter idle
	err := service.IdleChanged(context.Background(), true)
	require.NoError(t, err, "failed to enter idle")

	// read the idle period
	var readIdlePeriod usage.IdlePeriod
	if err := db.Where("ended_at IS NULL").Limit(1).Order("started_at desc").First(&readIdlePeriod).Error; err != nil {
		t.Fatalf("failed to find idle period: %v", err)
	}

	require.NotEqual(t, int64(0), readIdlePeriod.StartedAt)
	require.NotEqual(t, readIdlePeriod.ID, closedIdlePeriod.ID)
	require.Nil(t, readIdlePeriod.EndedAt)
	require.Nil(t, readIdlePeriod.DurationSeconds)
}

func TestService_WhenExitIdle_CloseCurrentIdlePeriod(t *testing.T) {
	service, db := setUpService(t)

	// create a new idle period
	idlePeriod := usage.IdlePeriod{
		StartedAt: time.Now().Unix(),
	}

	if err := db.Create(&idlePeriod).Error; err != nil {
		t.Fatalf("failed to create idle period: %v", err)
	}

	// wait 3 seconds
	time.Sleep(3 * time.Second)

	// exit idle
	err := service.IdleChanged(context.Background(), false)
	require.NoError(t, err, "failed to exit idle")

	// read the idle period
	var readIdlePeriod usage.IdlePeriod
	if err := db.Where("id = ?", idlePeriod.ID).First(&readIdlePeriod).Error; err != nil {
		t.Fatalf("failed to find idle period: %v", err)
	}

	require.NotEqual(t, int64(0), readIdlePeriod.StartedAt)
	require.NotEqual(t, int64(0), readIdlePeriod.EndedAt)
	require.NotNil(t, readIdlePeriod.DurationSeconds)
	require.Equal(t, 3, *readIdlePeriod.DurationSeconds)
}

func TestService_CloseCurrentApplicationUsageWhenEnterIdle(t *testing.T) {
	service, db := setUpService(t)

	// create a new application usage
	applicationUsage := usage.ApplicationUsage{
		StartedAt:       time.Now().Unix(),
		EndedAt:         nil,
		DurationSeconds: nil,
	}
	if err := db.Create(&applicationUsage).Error; err != nil {
		t.Fatalf("failed to create application usage: %v", err)
	}

	// wait 3 seconds
	time.Sleep(3 * time.Second)

	err := service.IdleChanged(context.Background(), true)
	require.NoError(t, err, "failed to enter idle")

	// read the application usage
	var readApplicationUsage usage.ApplicationUsage
	if err := db.Where("id = ?", applicationUsage.ID).First(&readApplicationUsage).Error; err != nil {
		t.Fatalf("failed to find application usage: %v", err)
	}

	require.NotEqual(t, 0, readApplicationUsage.StartedAt)
	require.NotEqual(t, 0, readApplicationUsage.EndedAt)
	require.NotNil(t, readApplicationUsage.DurationSeconds)
	require.Equal(t, 3, *readApplicationUsage.DurationSeconds)
}

func TestService_TitleChanged_WhenSameApplication_ContinueCurrentApplicationUsage(t *testing.T) {
	service, db := setUpService(t)

	ctx := context.Background()

	// create a new application usage
	applicationUsage := usage.ApplicationUsage{
		StartedAt:   time.Now().Unix(),
		WindowTitle: "Slack",
		Application: usage.Application{Name: "Slack"},
	}
	if err := db.Create(&applicationUsage).Error; err != nil {
		t.Fatalf("failed to create application usage: %v", err)
	}

	// change the title of the current application
	err := service.TitleChanged(ctx, "/Applications/Slack.app/Contents/MacOS/Slack", "Slack", "Slack", "", nil, nil)
	require.NoError(t, err, "failed to change title")

	// read the application usage
	var readApplicationUsage usage.ApplicationUsage
	if err := db.Where("id = ?", applicationUsage.ID).First(&readApplicationUsage).Error; err != nil {
		t.Fatalf("failed to find application usage: %v", err)
	}

	require.NotEqual(t, 0, readApplicationUsage.StartedAt)
	require.Nil(t, readApplicationUsage.EndedAt)
	require.Nil(t, readApplicationUsage.DurationSeconds)
}

func TestService_TitleChanged_WhenDifferentApplication_CloseCurrentApplicationUsage(t *testing.T) {
	service, db := setUpService(t)

	ctx := context.Background()

	// create a new application usage
	applicationUsage := usage.ApplicationUsage{
		StartedAt:   time.Now().Unix(),
		WindowTitle: "Slack",
		Application: usage.Application{Name: "Slack"},
	}
	if err := db.Create(&applicationUsage).Error; err != nil {
		t.Fatalf("failed to create application usage: %v", err)
	}

	// change the title of the current application
	err := service.TitleChanged(ctx, "com.apple.Safari", "Safari", "New Tab", "", nil, nil)
	require.NoError(t, err, "failed to change title")

	// read the application usage
	var readApplicationUsage usage.ApplicationUsage
	if err := db.Where("id = ?", applicationUsage.ID).First(&readApplicationUsage).Error; err != nil {
		t.Fatalf("failed to find application usage: %v", err)
	}

	require.NotNil(t, readApplicationUsage.EndedAt)
}

func TestService_TitleChanged_ClassificationErrorStored(t *testing.T) {
	// setup a service with a mock settings service that returns invalid custom rules to trigger a classification error
	usageService, db := setUpServiceWithSettings(t, "invalid custom rules")

	err := usageService.TitleChanged(context.Background(), "com.apple.Safari", "Safari", "New Tab", "", nil, nil)
	require.Nil(t, err)

	var readApplicationUsage usage.ApplicationUsage
	if err := db.Where("id = ?", 1).First(&readApplicationUsage).Error; err != nil {
		t.Fatalf("failed to find application usage: %v", err)
	}

	require.NotNil(t, readApplicationUsage.ClassificationError)
	require.Contains(t, *readApplicationUsage.ClassificationError, "failed to classify application usage with custom rules")
}

// roundTripFunc is an adapter to allow the use of ordinary functions as http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestService_TitleChanged_PropogateClassificationFromLLM(t *testing.T) {
	// Classification response that the mocked LLM returns.
	// Note: "classification_source" is intentionally omitted here because the real
	// LLM prompt does not ask for it. The backend (classifyWithGemini) must set it
	// explicitly to ClassificationSourceCloudLLM after unmarshalling.
	classificationJSON := `{"classification":"productive","reasoning":"Productive work communication","confidence_score":0.95,"tags":["work","communication"]}`

	// Wrap in a valid Gemini API response envelope (candidates[].content.parts[].text)
	geminiResponse := fmt.Sprintf(`{"candidates":[{"content":{"parts":[{"text":%q}],"role":"model"}}]}`, classificationJSON)

	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  "test-key",
		Backend: genai.BackendGeminiAPI,
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(geminiResponse)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			}),
		},
	})
	require.NoError(t, err, "failed to create genai client")

	service, db := setUpService(t, usage.WithGenaiClient(genaiClient))

	url := "https://example.com/docs"
	err = service.TitleChanged(context.Background(),
		"/Applications/Safari.app/Contents/MacOS/Safari",
		"Safari",
		"Example Docs",
		"",
		nil,
		&url,
	)
	require.NoError(t, err, "failed to change title")

	// Read the application usage and verify classification was propagated from LLM
	var readApplicationUsage usage.ApplicationUsage
	require.NoError(t, db.Preload("Tags").Where("ended_at IS NULL").First(&readApplicationUsage).Error)

	require.Equal(t, usage.ClassificationProductive, readApplicationUsage.Classification)
	require.Equal(t, usage.ClassificationSourceCloudLLM, readApplicationUsage.ClassificationSource)
	require.Equal(t, "Productive work communication", readApplicationUsage.ClassificationReasoning)
	require.Equal(t, float32(0.95), readApplicationUsage.ClassificationConfidence)

	// Verify tags were propagated
	require.Len(t, readApplicationUsage.Tags, 2)
	require.Equal(t, "work", readApplicationUsage.Tags[0].Tag)
	require.Equal(t, "communication", readApplicationUsage.Tags[1].Tag)
}
