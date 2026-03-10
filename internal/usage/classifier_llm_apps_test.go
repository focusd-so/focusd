package usage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

// roundTripFunc is an adapter to allow the use of ordinary functions as http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestService_ClassifyApplication_Slack(t *testing.T) {
	// Classification response that the mocked LLM returns.
	classificationJSON := `{"classification":"productive","reasoning":"Productive work communication","confidence_score":0.95,"tags":["communication"],"detected_communication_channel":"#engineering"}`

	// Wrap in a valid Gemini API response envelope
	geminiResponse := fmt.Sprintf(`{"candidates":[{"content":{"parts":[{"text":%q}],"role":"model"}}]}`, classificationJSON)

	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  "test-key",
		Backend: genai.BackendGeminiAPI,
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				// Verify that the instructions contain Slack-specific keywords
				body, _ := io.ReadAll(req.Body)
				bodyStr := string(body)
				assert.Contains(t, bodyStr, "Slack Software Engineer Intent Classifier")
				assert.Contains(t, bodyStr, "#engineering")
				
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(geminiResponse)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			}),
		},
	})
	require.NoError(t, err, "failed to create genai client")

	s := &Service{
		genaiClient: genaiClient,
	}

	t.Run("Slack productive", func(t *testing.T) {
		response, err := s.classifyApplication(context.Background(), "Slack", "#engineering | Acme Corp", nil)
		require.NoError(t, err)

		assert.Equal(t, ClassificationProductive, response.Classification)
		assert.Equal(t, "#engineering", response.DetectedCommunicationChannel)
	})
}

func TestService_ClassifyApplication_Generic(t *testing.T) {
	// Classification response for a generic app
	classificationJSON := `{"classification":"productive","reasoning":"Coding in VS Code","confidence_score":0.99,"tags":["coding"]}`
	geminiResponse := fmt.Sprintf(`{"candidates":[{"content":{"parts":[{"text":%q}],"role":"model"}}]}`, classificationJSON)

	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  "test-key",
		Backend: genai.BackendGeminiAPI,
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				// Verify that generic instructions are used
				body, _ := io.ReadAll(req.Body)
				bodyStr := string(body)
				assert.Contains(t, bodyStr, "Software Engineering Application Intent Classifier")
				assert.NotContains(t, bodyStr, "Slack Software Engineer Intent Classifier")

				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(geminiResponse)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			}),
		},
	})
	require.NoError(t, err)

	s := &Service{
		genaiClient: genaiClient,
	}

	t.Run("VS Code productive", func(t *testing.T) {
		response, err := s.classifyApplication(context.Background(), "Code", "main.go - focusd", nil)
		require.NoError(t, err)

		assert.Equal(t, ClassificationProductive, response.Classification)
	})
}
