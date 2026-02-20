package classification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
	"google.golang.org/genai"
)

type ContentGenerator interface {
	GenerateContent(ctx context.Context, instruction, prompt string) (string, error)
}

type Classification struct {
	client *genai.Client
}

func NewClassification(client *genai.Client) *Classification {
	return &Classification{client: client}
}

// classifyWithInstructions is the shared helper that all classifiers use.
// It builds the input from the request, calls Gemini with the given instructions,
// and parses the JSON response.
func (c *Classification) classifyWithInstructions(ctx context.Context, instructions string, req *apiv1.LLMClassifyRequest) (*apiv1.LLMClassifyResponse, error) {
	input := buildInput(req)

	resp, err := c.client.Models.GenerateContent(ctx, "gemini-2.5-flash", []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText(input),
			},
		},
	}, &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				genai.NewPartFromText(instructions),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("empty response from Gemini")
	}

	text := resp.Candidates[0].Content.Parts[0].Text

	var response apiv1.LLMClassifyResponse
	if err := json.Unmarshal([]byte(text), &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// buildInput constructs the input string from the request fields for web classification.
func buildInput(req *apiv1.LLMClassifyRequest) string {
	input := fmt.Sprintf("URL: %s\nTitle: %s", req.WebsiteUrl, req.WebsiteTitle)

	if req.WebsiteDescription != "" {
		input += fmt.Sprintf("\nDescription: %s", req.WebsiteDescription)
	}

	if req.WebsiteContentSnapshot != "" {
		input += fmt.Sprintf("\nContent Snapshot: %s", req.WebsiteContentSnapshot)
	}

	return input
}

// buildAppInput constructs the input string from the request fields for app classification.
func buildAppInput(req *apiv1.LLMClassifyRequest) string {
	return fmt.Sprintf("Bundle ID: %s\nWindow Title: %s", req.BundleId, req.WindowTitle)
}

// classifyAppWithInstructions is the shared helper that all app classifiers use.
// It builds the input from the request, calls Gemini with the given instructions,
// and parses the JSON response.
func (c *Classification) classifyAppWithInstructions(ctx context.Context, instructions string, req *apiv1.LLMClassifyRequest) (*apiv1.LLMClassifyResponse, error) {
	input := buildAppInput(req)

	resp, err := c.client.Models.GenerateContent(ctx, "gemini-2.5-flash", []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText(input),
			},
		},
	}, &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				genai.NewPartFromText(instructions),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("empty response from Gemini")
	}

	text := resp.Candidates[0].Content.Parts[0].Text

	var response apiv1.LLMClassifyResponse
	if err := json.Unmarshal([]byte(text), &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Classification) Classify(ctx context.Context, req *apiv1.LLMClassifyRequest) (*apiv1.LLMClassifyResponse, error) {
	if req.WebsiteUrl != "" {
		return c.classifyWebsite(ctx, req)
	} else if req.BundleId != "" {
		return c.classifyApplication(ctx, req)
	}

	return nil, errors.New("no classification method found")
}

func (c *Classification) classifyWebsite(ctx context.Context, req *apiv1.LLMClassifyRequest) (*apiv1.LLMClassifyResponse, error) {
	parsedURL, err := url.Parse(req.WebsiteUrl)
	if err != nil {
		return nil, err
	}

	hostname := strings.TrimPrefix(strings.ToLower(parsedURL.Hostname()), "www.")

	switch hostname {
	case "youtube.com", "youtu.be", "yt.be", "music.youtube.com":
		return c.ClassifyYoutube(ctx, req)
	case "reddit.com", "old.reddit.com", "new.reddit.com":
		return c.ClassifyReddit(ctx, req)
	case "linkedin.com":
		return c.ClassifyLinkedin(ctx, req)
	case "x.com", "twitter.com", "mobile.twitter.com", "mobile.x.com":
		return c.ClassifyX(ctx, req)
	case "slack.com", "app.slack.com":
		return c.ClassifySlack(ctx, req)
	case "discord.com", "discordapp.com", "discord.gg":
		return c.ClassifyDiscord(ctx, req)
	default:
		return c.ClassifyGenericWebsite(ctx, req)
	}
}

func (c *Classification) classifyApplication(ctx context.Context, req *apiv1.LLMClassifyRequest) (*apiv1.LLMClassifyResponse, error) {
	bundleID := strings.ToLower(req.BundleId)

	switch {
	case strings.Contains(bundleID, "slack"):
		return c.ClassifySlackApp(ctx, req)
	case strings.Contains(bundleID, "discord"):
		return c.ClassifyDiscordApp(ctx, req)
	case isCodeEditor(bundleID):
		return c.ClassifyCodeEditor(ctx, req)
	default:
		return c.ClassifyGenericApp(ctx, req)
	}
}
