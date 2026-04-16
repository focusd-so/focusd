package usage

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
	"github.com/focusd-so/focusd/internal/api"
	"github.com/focusd-so/focusd/internal/identity"
	"github.com/focusd-so/focusd/internal/settings"
	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/openai"
	"google.golang.org/api/option"
)

func (s *Service) ClassifyWithLLM(ctx context.Context, appName, title string, url *url.URL, appCategory *string) (*LLMClassificationResult, error) {
	if url != nil {
		return s.classifyWebsite(ctx, url, title)
	}

	return s.classifyApplication(ctx, appName, title, appCategory)
}

func classify(ctx context.Context, instructions, input string) (*LLMClassificationResult, error) {
	switch settings.GetConfig().ClassificationLLMProvider {
	case settings.LLMProviderGoogle:
		return classifyWithGemini(ctx, instructions, input)
	case settings.LLMProviderOpenAI:
		return classifyWithOpenAI(ctx, instructions, input)
	case settings.LLMProviderAnthropic:
		return classifyWithAnthropic(ctx, instructions, input)
	case settings.LLMProviderGroq:
		return classifyWithGrok(ctx, instructions, input)
	case settings.LLMProviderDummy:
		resp := viper.GetString("dummy_classification_response")
		if resp == "" {
			return nil, errors.New("dummy_classification_response is not set")
		}
		var classification LLMClassificationResult
		if err := json.Unmarshal([]byte(resp), &classification); err != nil {
			return nil, err
		}
		return &classification, nil
	}

	return nil, errors.New("unsupported LLM provider")
}

func selectClassificationModel(models map[apiv1.DeviceHandshakeResponse_AccountTier]string) string {
	tier := identity.GetAccountTier()
	if !slices.Contains([]apiv1.DeviceHandshakeResponse_AccountTier{apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE, apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PLUS}, tier) {
		slog.Warn("tier is not pro, using free model", "tier", tier)

		tier = apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE
	}

	return models[tier]
}

func newSignedLLMHTTPClient() *http.Client {
	return &http.Client{
		Transport: api.NewSigningRoundTripper(nil),
	}
}

func classifyWithGemini(ctx context.Context, instructions, input string) (*LLMClassificationResult, error) {
	// Use the strongest current non-reasoning Gemini model.
	models := map[apiv1.DeviceHandshakeResponse_AccountTier]string{
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_UNSPECIFIED: "gemini-2.5-flash-lite",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE:        "gemini-2.5-flash-lite",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_TRIAL:       "gemini-2.5-flash-lite",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PLUS:        "gemini-2.5-flash-lite",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO:         "gemini-2.5-flash-lite",
	}

	withGoogleAIEndpoint := func(endpoint string) googleai.Option {
		return func(opts *googleai.Options) {
			opts.ClientOptions = append(opts.ClientOptions, option.WithEndpoint(endpoint))
		}
	}

	selectedModel := selectClassificationModel(models)
	model, err := googleai.New(ctx,
		googleai.WithDefaultModel(selectedModel),
		googleai.WithHTTPClient(newSignedLLMHTTPClient()),
		withGoogleAIEndpoint(settings.APIBaseURL()+"/api/v1/gemini"),
	)
	if err != nil {
		return nil, err
	}

	return classifyWithLLM(ctx, model, instructions, input)
}

func classifyWithOpenAI(ctx context.Context, instructions, input string) (*LLMClassificationResult, error) {
	// Use the strongest current non-reasoning OpenAI model.
	models := map[apiv1.DeviceHandshakeResponse_AccountTier]string{
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_UNSPECIFIED: "gpt-4.1",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE:        "gpt-4.1",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_TRIAL:       "gpt-4.1",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PLUS:        "gpt-4.1",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO:         "gpt-4.1",
	}

	model, err := openai.New(
		openai.WithModel(selectClassificationModel(models)),
		openai.WithToken("stubbed"),
		openai.WithHTTPClient(newSignedLLMHTTPClient()),
		openai.WithBaseURL(settings.APIBaseURL()+"/api/v1/openai/v1"),
	)
	if err != nil {
		return nil, err
	}

	return classifyWithLLM(ctx, model, instructions, input)
}

func classifyWithAnthropic(ctx context.Context, instructions, input string) (*LLMClassificationResult, error) {
	// Use the strongest current non-reasoning Claude model.
	models := map[apiv1.DeviceHandshakeResponse_AccountTier]string{
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_UNSPECIFIED: "claude-sonnet-4-5",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE:        "claude-sonnet-4-5",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_TRIAL:       "claude-sonnet-4-5",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PLUS:        "claude-sonnet-4-5",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO:         "claude-sonnet-4-5",
	}

	model, err := anthropic.New(
		anthropic.WithModel(selectClassificationModel(models)),
		anthropic.WithToken("stubbed"),
		anthropic.WithHTTPClient(newSignedLLMHTTPClient()),
		anthropic.WithBaseURL(settings.APIBaseURL()+"/api/v1/anthropic/v1"),
	)
	if err != nil {
		return nil, err
	}

	return classifyWithLLM(ctx, model, instructions, input)
}

func classifyWithGrok(ctx context.Context, instructions, input string) (*LLMClassificationResult, error) {
	// Use the strongest current non-reasoning Grok model.
	models := map[apiv1.DeviceHandshakeResponse_AccountTier]string{
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_UNSPECIFIED: "grok-4.20-beta-latest-non-reasoning",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE:        "grok-4.20-beta-latest-non-reasoning",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_TRIAL:       "grok-4.20-beta-latest-non-reasoning",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PLUS:        "grok-4.20-beta-latest-non-reasoning",
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO:         "grok-4.20-beta-latest-non-reasoning",
	}

	model, err := openai.New(
		openai.WithModel(selectClassificationModel(models)),
		openai.WithToken("stubbed"),
		openai.WithHTTPClient(newSignedLLMHTTPClient()),
		openai.WithBaseURL(settings.APIBaseURL()+"/api/v1/grok/v1"),
	)
	if err != nil {
		return nil, err
	}

	return classifyWithLLM(ctx, model, instructions, input)
}

func classifyWithLLM(ctx context.Context, model llms.Model, instructions, input string) (*LLMClassificationResult, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := model.GenerateContent(reqCtx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, instructions),
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("empty response from LLM")
	}

	text := resp.Choices[0].Content

	replacer := strings.NewReplacer("```json", "", "```", "", "`", "")
	text = replacer.Replace(text)

	slog.Info("LLM classification response", "resp", text)

	var response LLMClassificationResult

	if err := json.Unmarshal([]byte(text), &response); err != nil {
		return nil, err
	}

	switch settings.GetConfig().ClassificationLLMProvider {
	case settings.LLMProviderGoogle:
		response.ClassificationSource = ClassificationSourceLLMGemini
	case settings.LLMProviderOpenAI:
		response.ClassificationSource = ClassificationSourceLLMOpenAI
	case settings.LLMProviderAnthropic:
		response.ClassificationSource = ClassificationSourceLLMAnthropic
	case settings.LLMProviderGroq:
		response.ClassificationSource = ClassificationSourceLLMGroq
	}

	return &response, nil
}
