package usage

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
	"golang.org/x/sync/errgroup"
)

func (s *Service) classifyWebsite(ctx context.Context, url *url.URL, title string) (*LLMClassificationResult, error) {
	if url == nil {
		return nil, fmt.Errorf("url is nil")
	}

	hostname := url.Hostname()
	domain, _ := publicsuffix.EffectiveTLDPlusOne(hostname)

	switch domain {
	case "youtube.com", "youtu.be", "yt.be":
		return classifyYoutube(ctx, url.String(), title)
	case "reddit.com":
		return classifyReddit(ctx, url.String(), title)
	case "linkedin.com":
		return classifyLinkedin(ctx, url.String(), title)
	case "medium.com":
		return classifyMedium(ctx, url.String(), title)
	case "x.com", "twitter.com":
		return classifyTwitter(ctx, url.String(), title)
	case "ycombinator.com":
		return classifyHackerNews(ctx, url.String(), title)
	case "substack.com":
		return classifySubstack(ctx, url.String(), title)
	case "slack.com":
		return classifySlack(ctx, url.String(), title)
	}

	return classifyGeneric(ctx, url, title)
}

func classifyGeneric(ctx context.Context, url *url.URL, title string) (*LLMClassificationResult, error) {
	// Attempt to fetch main content for better context, but proceed even if it fails
	// or if the site blocks scrapers.
	var mainContent string
	if content, err := fetchMainContent(ctx, url.String()); err == nil {
		// Truncate content to avoid excessive token usage while keeping enough context
		if len(content) > 5000 {
			mainContent = content[:5000] + "..."
		} else {
			mainContent = content
		}
	} else {
		// Log error but don't fail classification
		slog.Debug("failed to fetch main content for generic classification", "url", url, "error", err)
	}

	if isDeterministicCriticalNoBlockURL(url) {
		return &LLMClassificationResult{
			BasicClassificationResult: BasicClassificationResult{
				Classification:       ClassificationNeutral,
				ClassificationReason: "Payment/booking flow detected - safety override to avoid interruption",
				Tags:                 []string{"other"},
			},
			ConfidenceScore: 1.0,
		}, nil
	}

	suspiciousCriticalContext := isSuspiciousCriticalContext(url, title, mainContent)

	var (
		instructions = instructionGenericWebsiteClassification
		inputTmpl    = `
The user is currently visiting a website. Classify the page based on the following information:

URL: %s
Title: %s
Main Content (truncated): %s
`
	)

	input := fmt.Sprintf(inputTmpl, url, title, mainContent)

	response, err := classify(ctx, instructions, input)
	if err != nil {
		if suspiciousCriticalContext {
			return &LLMClassificationResult{
				BasicClassificationResult: BasicClassificationResult{
					Classification:       ClassificationNeutral,
					ClassificationReason: "Potential payment/booking flow with uncertain model result - safety override to avoid interruption",
					Tags:                 []string{"other"},
				},
				ConfidenceScore: 1.0,
			}, nil
		}

		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	if suspiciousCriticalContext && response.ConfidenceScore < 0.75 {
		return &LLMClassificationResult{
			BasicClassificationResult: BasicClassificationResult{
				Classification:       ClassificationNeutral,
				ClassificationReason: "Potential payment/booking flow with low-confidence classification - safety override to avoid interruption",
				Tags:                 []string{"other"},
			},
			ConfidenceScore: 1.0,
		}, nil
	}

	return response, nil
}

func classifyYoutube(ctx context.Context, url, title string) (*LLMClassificationResult, error) {
	httpClient := &http.Client{Timeout: 3000 * time.Millisecond}

	metaData, err := extractOpenGraph(httpClient, url)
	if err != nil {
		slog.Error("failed to extract open graph", "error", err)
	}

	var (
		description string
		tags        []string
	)

	for _, meta := range metaData {
		switch meta.Property {
		case "title":
			title = meta.Content
		case "description":
			description = meta.Content
		case "video:tag":
			tags = append(tags, meta.Content)
		}
	}

	instructions := instructionYouTubeWebsiteClassification

	input := fmt.Sprintf(
		`Classify the following YouTube video:\nURL: %s\nTitle: %s\nDescription: %s\nTags: %s`,
		url, title, description, strings.Join(tags, ", "),
	)

	response, err := classify(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

func classifyReddit(ctx context.Context, url, title string) (*LLMClassificationResult, error) {
	mainContent, err := fetchMainContent(ctx, url)
	if err != nil {
		slog.Error("failed to fetch main content", "error", err)
	}

	instructions := instructionRedditWebsiteClassification

	inputTmpl := `
The user is currently visiting a Reddit post. Classify the post based on the following information:

URL: %s
Title: %s
Main Content (if available): %s
`

	input := fmt.Sprintf(inputTmpl, url, title, mainContent)

	response, err := classify(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

func classifyLinkedin(ctx context.Context, url, title string) (*LLMClassificationResult, error) {

	var (
		ogTitle       = title
		ogDescription string
		mainContent   string
		g             = errgroup.Group{}
	)

	httpClient := &http.Client{Timeout: 3000 * time.Millisecond}

	g.Go(func() error {
		content, err := fetchMainContent(ctx, url)
		if err != nil {
			slog.Error("failed to fetch main content", "error", err)
		}
		mainContent = content

		return nil
	})

	g.Go(func() error {
		metaData, err := extractOpenGraph(httpClient, url)
		if err != nil {
			slog.Error("failed to extract open graph", "error", err)
			return nil
		}
		for _, meta := range metaData {
			switch meta.Property {
			case "title":
				ogTitle = meta.Content
			case "description":
				ogDescription = meta.Content
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		slog.Error("failed to classify with Gemini", "error", err)
	}

	var (
		instructions = instructionLinkedInWebsiteClassification
		inputTmpl    = `
The user is currently visiting a LinkedIn page. Classify the page based on the following information:

URL: %s
Title: %s
Description: %s
Main Content (if available): %s
`
	)

	input := fmt.Sprintf(inputTmpl, url, ogTitle, ogDescription, mainContent)

	response, err := classify(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

// classifyMedium classifies Medium articles using only the URL and browser tab title.
// Medium is behind Cloudflare managed challenges so server-side content fetching is not possible.
// The URL slug and title are usually descriptive enough for accurate classification.
func classifyMedium(ctx context.Context, url, title string) (*LLMClassificationResult, error) {
	var (
		instructions = instructionMediumWebsiteClassification
		inputTmpl    = `
The user is currently visiting a Medium article. Classify the article based on the following information:

URL: %s
Title: %s
`
	)

	input := fmt.Sprintf(inputTmpl, url, title)

	response, err := classify(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

// classifyTwitter classifies X/Twitter posts using only the URL and browser tab title.
// X/Twitter is behind auth walls for server-side fetching, so OpenGraph and content extraction
// are not reliable. The browser tab title contains the tweet text for status pages, which is
// a strong signal for classification.
func classifyTwitter(ctx context.Context, url, title string) (*LLMClassificationResult, error) {
	var (
		instructions = instructionTwitterWebsiteClassification
		inputTmpl    = `
The user is currently visiting an X/Twitter page. Classify the page based on the following information:

URL: %s
Title: %s
`
	)

	input := fmt.Sprintf(inputTmpl, url, title)

	response, err := classify(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

func classifyHackerNews(ctx context.Context, url, title string) (*LLMClassificationResult, error) {
	mainContent, err := fetchMainContent(ctx, url)
	if err != nil {
		slog.Error("failed to fetch main content", "error", err)
	}

	instructions := instructionHackerNewsWebsiteClassification

	inputTmpl := `
The user is currently visiting a Hacker News page. Classify the page based on the following information:

URL: %s
Title: %s
Main Content (if available): %s
`

	input := fmt.Sprintf(inputTmpl, url, title, mainContent)

	response, err := classify(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

// classifySubstack classifies Substack newsletter articles using OpenGraph metadata and content extraction.
// Substack pages are generally scrapable without heavy Cloudflare challenges.
func classifySubstack(ctx context.Context, url, title string) (*LLMClassificationResult, error) {
	httpClient := &http.Client{Timeout: 3000 * time.Millisecond}

	var (
		ogTitle       = title
		ogDescription string
		mainContent   string
		g             = errgroup.Group{}
	)

	g.Go(func() error {
		content, err := fetchMainContent(ctx, url)
		if err != nil {
			slog.Error("failed to fetch main content", "error", err)
		}
		mainContent = content
		return nil
	})

	g.Go(func() error {
		metaData, err := extractOpenGraph(httpClient, url)
		if err != nil {
			slog.Error("failed to extract open graph", "error", err)
			return nil
		}
		for _, meta := range metaData {
			switch meta.Property {
			case "title":
				ogTitle = meta.Content
			case "description":
				ogDescription = meta.Content
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		slog.Error("failed to fetch substack content", "error", err)
	}

	var (
		instructions = instructionSubstackWebsiteClassification
		inputTmpl    = `
The user is currently visiting a Substack article. Classify the article based on the following information:

URL: %s
Title: %s
Description: %s
Main Content (if available): %s
`
	)

	input := fmt.Sprintf(inputTmpl, url, ogTitle, ogDescription, mainContent)

	response, err := classify(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

// classifySlack classifies Slack activity using the browser tab title (or window title for desktop).
// No HTTP fetching is needed — the title contains the channel/DM name which is the primary signal.
// Titles typically look like: "#engineering - Acme Corp Slack" or "Slack | #random | My Workspace"
func classifySlack(ctx context.Context, url, title string) (*LLMClassificationResult, error) {
	input := fmt.Sprintf("Slack web title: %s\nURL: %s", title, url)

	return classifySlackActivity(ctx, input)
}
