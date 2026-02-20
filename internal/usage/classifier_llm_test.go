package usage

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/genai"
)

// This is probably controvertial tests suite to hit LLM as part of the CI, but at least it will
// give us some confidence that the LLM classification is not completely broken after prompt or
// code changes. It will depend on flakiness of the LLM, but I think it's worth it to have some
// confidence that the LLM is working as expected.

// TODO: mock http client to return specific responses for open graph and main content vs real http requests

func TestClassify_Website_Youtube(t *testing.T) {
	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: os.Getenv("GEMINI_API_KEY"),
	})

	require.NoError(t, err, "failed to create genai client")

	s, _ := setUpService(t, WithGenaiClient(genaiClient))

	t.Run("neutral - music video", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://www.youtube.com/watch?v=i0YnfhP3ogE&list=RDi0YnfhP3ogE&start_radio=1", "Tobu - Dancing in the Moonlight (feat Syndec)")
		require.NoError(t, err, "failed to classify youtube website")

		assert.Equal(t, ClassificationNeutral, response.Classification)
		assert.Contains(t, response.Tags, "music")
	})

	t.Run("distracting - entertainment video", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://www.youtube.com/watch?v=qKf9sgKFQLU", "OpenAI just dropped their Cursor killer")

		t.Logf("reasoning: %s", response.Reasoning)

		require.NoError(t, err, "failed to classify youtube website")

		assert.Equal(t, ClassificationDistracting, response.Classification)
	})
}

func TestClassify_Website_Reddit(t *testing.T) {
	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: os.Getenv("GEMINI_API_KEY"),
	})

	require.NoError(t, err, "failed to create genai client")

	s, _ := setUpService(t, WithGenaiClient(genaiClient))

	t.Run("productive - golang discussion", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://www.reddit.com/r/golang/comments/1fu1z4v/whats_the_state_of_go_web_frameworks_today/", "What's the state of Go web frameworks today?")
		require.NoError(t, err, "failed to classify reddit website")

		assert.Equal(t, ClassificationProductive, response.Classification)
	})

	t.Run("distracting - stock discussion", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://www.reddit.com/r/UnityStock/comments/1qrjrpw/unitys_moat_in_case_of_world_models/", "Unity's Moat in Case of World Models")
		require.NoError(t, err, "failed to classify reddit website")

		assert.Equal(t, ClassificationDistracting, response.Classification)
	})
}

func TestClassify_Website_LinkedIn(t *testing.T) {
	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: os.Getenv("GEMINI_API_KEY"),
	})

	require.NoError(t, err, "failed to create genai client")

	s, _ := setUpService(t, WithGenaiClient(genaiClient))

	t.Run("productive - technical article", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://www.linkedin.com/pulse/how-i-speed-up-microservice-development-golang-fiber-charith-rajitha-dyuhc/", "How I Speed Up Microservice Development with Golang Fiber Framework")
		require.NoError(t, err, "failed to classify linkedin website")

		assert.Equal(t, ClassificationProductive, response.Classification)
	})

	t.Run("distracting - opinion post", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://www.linkedin.com/posts/kyleszives_ai-assisted-coding-slows-mastery-this-week-share-7423115331218337792-DlYT/", "AI-assisted coding: trade-off between speed and mastery | Kyle Szives posted on the topic | LinkedIn")
		require.NoError(t, err, "failed to classify linkedin website")

		t.Log(response.Reasoning)

		assert.Equal(t, ClassificationDistracting, response.Classification)
	})
}

func TestClassify_Website_Medium(t *testing.T) {
	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: os.Getenv("GEMINI_API_KEY"),
	})

	require.NoError(t, err, "failed to create genai client")

	s, _ := setUpService(t, WithGenaiClient(genaiClient))

	t.Run("productive - technical article", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://medium.com/@Sarahcollins05/golang-web-frameworks-top-picks-for-modern-development-8b85d9000921", "Golang Web Frameworks: Top Picks for Modern Development")
		require.NoError(t, err, "failed to classify medium website")

		assert.Equal(t, ClassificationProductive, response.Classification)
	})

	t.Run("distracting - career clickbait", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://medium.com/@kakamber07/i-interviewed-50-devs-laid-off-in-2025-one-mistake-was-universal-15711c87f192", "I Interviewed 50 Devs Laid Off in 2025: One Mistake Was Universal")
		require.NoError(t, err, "failed to classify medium website")

		assert.Equal(t, ClassificationDistracting, response.Classification)
	})
}

func TestClassify_Website_Twitter(t *testing.T) {
	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: os.Getenv("GEMINI_API_KEY"),
	})

	require.NoError(t, err, "failed to create genai client")

	s, _ := setUpService(t, WithGenaiClient(genaiClient))

	t.Run("productive - technical announcement", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://x.com/golang/status/1818029690722304258", "Go on X: \"Go 1.23 is released! Release notes: https://go.dev/doc/go1.23\"")
		require.NoError(t, err, "failed to classify twitter website")

		assert.Equal(t, ClassificationProductive, response.Classification)
	})

	t.Run("distracting - hot take engagement bait", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://x.com/random_user/status/1234567890", "Hot take: nobody actually needs microservices. Fight me. 🔥")
		require.NoError(t, err, "failed to classify twitter website")

		assert.Equal(t, ClassificationDistracting, response.Classification)
	})
}

func TestClassify_Website_HackerNews(t *testing.T) {
	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: os.Getenv("GEMINI_API_KEY"),
	})

	require.NoError(t, err, "failed to create genai client")

	s, _ := setUpService(t, WithGenaiClient(genaiClient))

	t.Run("productive - technical article", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://news.ycombinator.com/item?id=39232976", "Writing a SQL database from scratch in Go")
		require.NoError(t, err, "failed to classify hackernews website")

		assert.Equal(t, ClassificationProductive, response.Classification)
	})

	t.Run("distracting - career opinion thread", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://news.ycombinator.com/item?id=39171900", "Ask HN: What's your salary and how happy are you with it?")
		require.NoError(t, err, "failed to classify hackernews website")

		assert.Equal(t, ClassificationDistracting, response.Classification)
	})
}

func TestClassify_Website_Substack(t *testing.T) {
	// genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
	// 	APIKey: os.Getenv("GEMINI_API_KEY"),
	// })

	// require.NoError(t, err, "failed to create genai client")

	// s, _ := setUpService(t, WithGenaiClient(genaiClient))

	// t.Run("productive - engineering newsletter", func(t *testing.T) {
	// 	response, err := s.classifyWebsite(context.Background(), "https://newsletter.systemdesign.one/p/cell-based-architecture", "Cell-Based Architecture: How to Build Scalable Distributed Systems")
	// 	require.NoError(t, err, "failed to classify substack website")

	// 	assert.Equal(t, ClassificationProductive, response.Classification)
	// })

	// t.Run("distracting - self-help opinion", func(t *testing.T) {
	// 	response, err := s.classifyWebsite(context.Background(), "https://www.theguardianofperspective.substack.com/p/why-you-should-quit-social-media", "Why You Should Quit Social Media and Reclaim Your Life")
	// 	require.NoError(t, err, "failed to classify substack website")

	// 	assert.Equal(t, ClassificationDistracting, response.Classification)
	// })
}

func TestClassify_Website_Slack(t *testing.T) {
	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: os.Getenv("GEMINI_API_KEY"),
	})

	require.NoError(t, err, "failed to create genai client")

	s, _ := setUpService(t, WithGenaiClient(genaiClient))

	t.Run("productive - engineering channel", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://app.slack.com/client/T12345/C67890", "#engineering - Acme Corp Slack")
		require.NoError(t, err, "failed to classify slack website")

		assert.Equal(t, ClassificationProductive, response.Classification)
		assert.NotEmpty(t, response.DetectedCommunicationChannel)
	})

	t.Run("productive - incident channel", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://app.slack.com/client/T12345/C11111", "#incidents - Acme Corp Slack")
		require.NoError(t, err, "failed to classify slack website")

		assert.Equal(t, ClassificationProductive, response.Classification)
		assert.NotEmpty(t, response.DetectedCommunicationChannel)
	})

	t.Run("neutral - general channel", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://app.slack.com/client/T12345/C22222", "#general - Acme Corp Slack")
		require.NoError(t, err, "failed to classify slack website")

		assert.Equal(t, ClassificationNeutral, response.Classification)
		assert.NotEmpty(t, response.DetectedCommunicationChannel)
	})

	t.Run("neutral - announcements channel", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://app.slack.com/client/T12345/C33333", "#announcements - Acme Corp Slack")
		require.NoError(t, err, "failed to classify slack website")

		assert.Equal(t, ClassificationNeutral, response.Classification)
		assert.NotEmpty(t, response.DetectedCommunicationChannel)
	})

	t.Run("distracting - random channel", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://app.slack.com/client/T12345/C99999", "#random - Acme Corp Slack")
		require.NoError(t, err, "failed to classify slack website")

		assert.Equal(t, ClassificationDistracting, response.Classification)
		assert.NotEmpty(t, response.DetectedCommunicationChannel)
	})

	t.Run("distracting - fun channel", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://app.slack.com/client/T12345/C88888", "#fun-memes - Acme Corp Slack")
		require.NoError(t, err, "failed to classify slack website")

		assert.Equal(t, ClassificationDistracting, response.Classification)
		assert.NotEmpty(t, response.DetectedCommunicationChannel)
	})

	t.Run("distracting - browsing channels", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://app.slack.com/client/T12345/browse-channels", "Browse channels - Acme Corp Slack")
		require.NoError(t, err, "failed to classify slack website")

		assert.Equal(t, ClassificationNeutral, response.Classification)
	})
}

func TestClassify_Website_Generic(t *testing.T) {
	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: os.Getenv("GEMINI_API_KEY"),
	})

	require.NoError(t, err, "failed to create genai client")

	s, _ := setUpService(t, WithGenaiClient(genaiClient))

	t.Run("productive - documentation", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://gorm.io/docs/index.html", "GORM Guides | GORM - The fantastic ORM library for Golang, aims to be developer friendly.")
		require.NoError(t, err, "failed to classify generic website")

		assert.Equal(t, ClassificationProductive, response.Classification)
	})

	t.Run("distracting - news site", func(t *testing.T) {
		response, err := s.classifyWebsite(context.Background(), "https://www.tmz.com/", "TMZ")
		require.NoError(t, err, "failed to classify generic website")

		assert.Equal(t, ClassificationDistracting, response.Classification)
	})
}
