package usage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
	"golang.org/x/sync/errgroup"
	"google.golang.org/genai"
)

func (s *Service) classifyWebsite(ctx context.Context, url, title string) (*ClassificationResponse, error) {
	hostname, _ := parseURL(url)
	domain, _ := publicsuffix.EffectiveTLDPlusOne(hostname)

	switch domain {
	case "youtube.com", "youtu.be", "yt.be":
		return s.classifyYoutube(ctx, url, title)
	case "reddit.com":
		return s.classifyReddit(ctx, url, title)
	case "linkedin.com":
		return s.classifyLinkedin(ctx, url, title)
	case "medium.com":
		return s.classifyMedium(ctx, url, title)
	case "x.com", "twitter.com":
		return s.classifyTwitter(ctx, url, title)
	case "ycombinator.com":
		return s.classifyHackerNews(ctx, url, title)
	case "substack.com":
		return s.classifySubstack(ctx, url, title)
	case "slack.com":
		return s.classifySlack(ctx, url, title)
	}

	return s.classifyGeneric(ctx, url, title)
}

func (s *Service) classifyGeneric(ctx context.Context, url, title string) (*ClassificationResponse, error) {
	// Attempt to fetch main content for better context, but proceed even if it fails
	// or if the site blocks scrapers.
	var mainContent string
	if content, err := fetchMainContent(ctx, url); err == nil {
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

	var (
		instructions = `
You are a General Software Engineer Intent Classifier. Your job is to determine if the user is actively doing work related to their software engineering job or seeking entertainment/distraction.

# Classification Logic

## **productive**
**Criteria:** Content that directly supports software engineering tasks: coding, debugging, learning, researching, planning, or deploying.
**Signals to detect:**
- **Documentation:** Official docs for languages, frameworks, libraries, APIs, clouds (AWS, GCP, Azure).
- **Repositories:** GitHub, GitLab, Bitbucket (code viewing, PRs, issues).
- **Q&A/Forums:** Stack Overflow, GitHub Discussions, technical forums.
- **Tools:** Issue trackers (Jira, Linear), design tools (Figma), cloud consoles, CI/CD dashboards, localhost servers.
- **Learning:** Technical tutorials, courses, engineering blogs, research papers.

## **distracting**
**Criteria:** Content irrelevant to work or primarily for entertainment/consumption.
**Includes:**
- Social media (Facebook, Instagram, TikTok, generic Twitter/X usage).
- News, politics, sports, celebrity gossip.
- Video streaming (Netflix, Hulu, HBO) and gaming.
- Shopping (Amazon, eBay) unless clearly for work equipment.
- General interest browsing not related to tech.

## **neutral**
**Criteria:** Ambiguous utilities or general organizational tools.
**Includes:**
- Email, Calendar (unless clearly personal).
- General search (Google home page).
- Banking, personal admin (borderline, but often necessary during work day).

Return a JSON object with the following keys:
1. "classification": "productive", "neutral", or "distracting".
2. "reasoning": Brief explanation.
3. "tags": Array of strings from ONLY these options: ["coding", "docs", "debug", "communication", "planning", "learning", "entertainment", "news", "social", "shopping", "other"].
4. "confidence_score": Float (0.0 - 1.0)
`
		inputTmpl = `
The user is currently visiting a website. Classify the page based on the following information:

URL: %s
Title: %s
Main Content (truncated): %s
`
	)

	input := fmt.Sprintf(inputTmpl, url, title, mainContent)

	response, err := s.classifyWithGemini(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

func (s *Service) classifyYoutube(ctx context.Context, url, title string) (*ClassificationResponse, error) {
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

	var (
		instructions = `
You are a STRICT YouTube Software Engineering Intent Classifier.

Your job is NOT to detect whether a video is about tech.
Your job is to determine whether the user is actively doing work or learning job-relevant software engineering skills.

You must be extremely conservative.

If there is ANY ambiguity, classify as "distracting".

# Classification Logic

## CORE PRINCIPLE

Tech topic ≠ Productive.

A video is "productive" ONLY if it clearly contains:
- Step-by-step instruction
- Hands-on coding
- Technical walkthrough
- Debugging session
- Deep system design explanation
- Practical implementation details

If the video is:
- Opinion-based
- News-style
- Reaction-style
- Influencer commentary
- Product hype
- Drama framing
- Industry gossip
- “X just killed Y”
- “This changes everything”
- “You won’t believe…”
- Tool announcement coverage
- AI model launch discussion without hands-on demo

→ It is "distracting".

## HARD RULES

1. Reviews, comparisons, or "vs" videos are ONLY productive if:
   - They include real code examples, benchmarks, architecture diagrams, or implementation walkthroughs.
   - Otherwise they are distracting.

2. Any video framed around:
   - hype
   - shock value
   - “killer”
   - “destroyed”
   - “insane”
   - “crazy”
   - “just dropped”
   - “game changer”
   - “you need to see this”
   is automatically classified as "distracting".

3. Influencer-style commentary, even about technical tools, is "distracting".

4. AI news coverage is "distracting" unless it teaches how to implement or use the tool with code.

5. Productive requires clear evidence of active skill development.

## **productive**
**Criteria:** Content that directly helps the user with software engineering work: tutorials, 
debugging guides, technology comparisons, coding walkthroughs, system design explanations, or 
learning job-relevant technical skills.

## **neutral**
**Criteria:** Content that is audio-centric, calming, or passive background noise designed to help the user focus.
**Keywords to detect:**
- "lofi", "music", "ambient", "instrumental", "jazz", "classical"
- "study", "focus", "relax", "sleep", "meditation"
- "rain", "white noise", "binaural", "soundscape", "soundtrack"
- "radio", "beats", "playlist" (when combined with above terms)

## **distracting**
**Criteria:** **EVERYTHING ELSE.** If it requires visual attention for entertainment, follows a non-educational narrative, or is not work-related technical content.
**Includes:** Vlogs, gaming/gameplay, non-tech reviews, news, podcasts (non-technical), comedy, reactions, shorts, and livestreams 
(unless specifically 24/7 music radio or technical content).

Return a JSON object with the following keys:
1. "classification": "productive" (work-related technical content), "neutral" (focus aids), or "distracting" (everything else).
2. "reasoning": Brief explanation.
3. "tags": Array of strings from ONLY these options: ["music", "ambient", "entertainment", "news", "edu", "learning", "content-consumption", "time-sink"].
4. "confidence_score": Float (0.0 - 1.0)
`

		inputTmpl = `
The user is currently visiting a YouTube video page. Classify the website based on the following information:

URL: %s
Title: %s
Description: %s
Tags: %s
`
	)

	input := fmt.Sprintf(inputTmpl, url, title, description, strings.Join(tags, ", "))

	response, err := s.classifyWithGemini(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

func (s *Service) classifyReddit(ctx context.Context, url, title string) (*ClassificationResponse, error) {
	mainContent, err := fetchMainContent(ctx, url)
	if err != nil {
		slog.Error("failed to fetch main content", "error", err)
	}

	var (
		instructions = `
You are a Reddit Software Engineer Intent Classifier. Your job is to determine if the user is actively doing work related to their software engineering job or seeking entertainment/distraction.

# Classification Logic

## **productive**
**Criteria:** Content that directly helps the user with software engineering work: solving technical problems, learning job-relevant skills, system design, architecture, DevOps, debugging, code reviews, or technical leadership.
**Signals to detect:**
- **Subreddits:** "programming", "webdev", "golang", "rust", "python", "javascript", "typescript", "devops", "docker", "kubernetes", "linux", "vim", "neovim", "aws", "azure", "gcp", "database", "sql", "react", "svelte", "vue", "node", "backend", "frontend", "softwareengineering", "experienceddevs", "sre", "netsec", "security", "terraform", "ansible", "graphql", "grpc", "microservices", "systemdesign"
- **Title patterns:** "How to", "Help with", "Debug", "Error", "Fix", "Solved", "Issue with", "Problem:", "[Question]", "Why does", "Best practice", "Architecture", "Design review", "Code review", "Production issue", "Outage", "Incident"
- **Content signals:** Stack traces, code snippets, error messages, technical discussions, architecture diagrams, system design questions, API discussions, infrastructure problems, deployment issues, monitoring/observability, security vulnerabilities, performance optimization

## **distracting**
**Criteria:** **EVERYTHING ELSE.** If the content is for entertainment, general browsing, memes, news, politics, gaming, or passive consumption, it is distracting.
**Includes:**
- Meme subreddits (even tech memes like "ProgrammerHumor" — entertainment, not work)
- News, politics, world events
- Gaming, hobbies, lifestyle content
- General discussion threads without a specific technical problem
- Career rants, salary discussions, workplace drama, interview stories
- "What's your favorite X" or opinion polls
- Showcase posts without technical depth
- Venting about managers, coworkers, or company culture

Return a JSON object with the following keys:
1. "classification": "productive" (actively doing work) or "distracting" (everything else).
2. "reasoning": Brief explanation.
3. "tags": Array of strings from ONLY these options: ["music", "ambient", "entertainment", "news", "edu", "learning", "content-consumption", "time-sink"].
4. "confidence_score": Float (0.0 - 1.0)
`
	)

	inputTmpl := `
The user is currently visiting a Reddit post. Classify the post based on the following information:

URL: %s
Title: %s
Main Content (if available): %s
`

	input := fmt.Sprintf(inputTmpl, url, title, mainContent)

	response, err := s.classifyWithGemini(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

func (s *Service) classifyLinkedin(ctx context.Context, url, title string) (*ClassificationResponse, error) {

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
		instructions = `
You are a STRICT LinkedIn Software Engineering Intent Classifier.

Your job is NOT to detect whether a post is about tech.
Your job is to determine whether the user is actively doing work or learning job-relevant software engineering skills.

You must be extremely conservative.

If there is ANY ambiguity, classify as "distracting".

# Classification Logic

## CORE PRINCIPLE

Tech topic ≠ Productive.

A video is "productive" ONLY if it clearly contains:
- Step-by-step instruction
- Hands-on coding
- Technical walkthrough
- Debugging session
- Deep system design explanation
- Practical implementation details

If the video is:
- Opinion-based
- News-style
- Reaction-style
- Influencer commentary
- Product hype
- Drama framing
- Industry gossip
- “X just killed Y”
- “This changes everything”
- “You won’t believe…”
- Tool announcement coverage
- AI model launch discussion without hands-on demo

→ It is "distracting".

## HARD RULES

1. Reviews, comparisons, or "vs" videos are ONLY productive if:
   - They include real code examples, benchmarks, architecture diagrams, or implementation walkthroughs.
   - Otherwise they are distracting.

2. Any video framed around:
   - hype
   - shock value
   - “killer”
   - “destroyed”
   - “insane”
   - “crazy”
   - “just dropped”
   - “game changer”
   - “you need to see this”
   is automatically classified as "distracting".

3. Influencer-style commentary, even about technical tools, is "distracting".

4. AI news coverage is "distracting" unless it teaches how to implement or use the tool with code.

5. Productive requires clear evidence of active skill development.


# Classification Logic

## **productive**
**Criteria:** ONLY content that directly helps with software engineering work. Must be clearly technical and actionable.
**Signals to detect:**
- Technical how-to guides, tutorials, or walkthroughs (coding, architecture, DevOps, tooling)
- Technology or framework comparisons (e.g. "X vs Y", "Choosing between...")
- System design, architecture, or engineering blog posts with technical depth
- Code-related discussions, debugging, performance, security, or infrastructure
- LinkedIn Learning or similar courses on relevant tech (when identifiable from title/description)
- Research or deep-dives on languages, frameworks, APIs, or platforms

## **distracting**
**Criteria:** EVERYTHING ELSE. When in doubt, classify as distracting.
**Includes:**
- Career advice, salary discussions, interview stories, job-search content
- Motivational or hustle-culture posts, influencer content, engagement bait
- Networking, recruiter messages, profile browsing, company PR or news
- Personal stories, "hot takes", memes, rants about managers or workplace culture
- Generic business or leadership content without technical depth
- Any content that is not clearly and specifically about doing technical work

Return a JSON object with the following keys:
1. "classification": "productive" (only clearly work-related technical content) or "distracting" (everything else).
2. "reasoning": Brief explanation.
3. "tags": Array of strings from ONLY these options: ["music", "ambient", "entertainment", "news", "edu", "learning", "content-consumption", "time-sink"].
4. "confidence_score": Float (0.0 - 1.0)
`
		inputTmpl = `
The user is currently visiting a LinkedIn page. Classify the page based on the following information:

URL: %s
Title: %s
Description: %s
Main Content (if available): %s
`
	)

	input := fmt.Sprintf(inputTmpl, url, ogTitle, ogDescription, mainContent)

	response, err := s.classifyWithGemini(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

// classifyMedium classifies Medium articles using only the URL and browser tab title.
// Medium is behind Cloudflare managed challenges so server-side content fetching is not possible.
// The URL slug and title are usually descriptive enough for accurate classification.
func (s *Service) classifyMedium(ctx context.Context, url, title string) (*ClassificationResponse, error) {
	var (
		instructions = `
You are a Medium Software Engineer Intent Classifier. Your job is to determine if the user is actively doing work related to their software engineering job or seeking entertainment/distraction. Be RESTRICTIVE: only clearly work-related technical content is productive.

NOTE: You will only receive the URL and the browser tab title. Medium articles encode meaningful keywords in their URL slugs (e.g. "golang-web-frameworks-top-picks-for-modern-development"). Use both the URL slug and the title to infer the article's topic.

# Classification Logic

## **productive**
**Criteria:** ONLY content that directly helps with software engineering work. Must be clearly technical and actionable.
**Signals to detect:**
- URL slugs or titles containing technical terms: programming languages (golang, rust, python, javascript, typescript, etc.), frameworks (react, django, gin, fiber, etc.), tools (docker, kubernetes, terraform, etc.)
- Technical blog posts, engineering deep-dives, tutorials, how-to guides
- System design, architecture, or engineering articles with technical depth
- DevOps write-ups, postmortems, infrastructure, deployment, monitoring
- Code snippets, debugging, performance, security, APIs, databases
- Author paths or slugs suggesting technical publications: better-programming, level-up-coding, towards-data-science, aws-in-plain-english, etc.

## **distracting**
**Criteria:** EVERYTHING ELSE. When in doubt, classify as distracting.
**Includes:**
- Self-help, productivity porn, hustle culture, life advice, motivational content
- Opinion pieces, listicles without technical depth, "top 10" non-technical lists
- Personal stories, career advice, interview stories, salary discussions
- Politics, lifestyle content, relationship or wellness advice
- Publications that often host non-technical content: The Startup, Personal Growth, etc.
- Any content that is not clearly and specifically about doing technical work

Return a JSON object with the following keys:
1. "classification": "productive" (only clearly work-related technical content) or "distracting" (everything else).
2. "reasoning": Brief explanation.
3. "tags": Array of strings from ONLY these options: ["music", "ambient", "entertainment", "news", "edu", "learning", "content-consumption", "time-sink"].
4. "confidence_score": Float (0.0 - 1.0)
`
		inputTmpl = `
The user is currently visiting a Medium article. Classify the article based on the following information:

URL: %s
Title: %s
`
	)

	input := fmt.Sprintf(inputTmpl, url, title)

	response, err := s.classifyWithGemini(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

// classifyTwitter classifies X/Twitter posts using only the URL and browser tab title.
// X/Twitter is behind auth walls for server-side fetching, so OpenGraph and content extraction
// are not reliable. The browser tab title contains the tweet text for status pages, which is
// a strong signal for classification.
func (s *Service) classifyTwitter(ctx context.Context, url, title string) (*ClassificationResponse, error) {
	var (
		instructions = `
You are an X/Twitter Software Engineer Intent Classifier. Your job is to determine if the user is actively doing work related to their software engineering job or seeking entertainment/distraction. Be RESTRICTIVE: only clearly work-related technical content is productive.

NOTE: You will only receive the URL and the browser tab title. For individual tweets, the browser tab title contains the tweet text. Use both the URL structure and the title to infer the content's purpose.

# Classification Logic

## **productive**
**Criteria:** ONLY content that directly helps with software engineering work. Must be clearly technical and actionable.
**Signals to detect:**
- Technical threads about programming, engineering, architecture, DevOps, security, or infrastructure
- Library, framework, or language release announcements and changelogs (e.g. "Go 1.23 released", "React 19 is out")
- Incident postmortems and production war stories with technical lessons
- Links to or discussions of engineering blog posts, technical papers, or documentation
- Open source project announcements with technical depth
- Code snippets, debugging discussions, performance analysis, security advisories
- Profiles of well-known technical authors when viewing their technical content
- Technical conference announcements, talk summaries, or slide decks

## **distracting**
**Criteria:** EVERYTHING ELSE. When in doubt, classify as distracting.
**Includes:**
- Hot takes, opinions, and engagement bait (even about tech topics like "X is dead", "Y is overrated")
- Tech memes, jokes, and humorous content
- Career advice, salary discussions, job search content, interview stories
- Political content, news, world events
- Personal stories, life updates, motivational content
- Recruiter outreach, hiring announcements, company PR
- Celebrity or influencer content
- Rants about managers, workplace culture, or industry drama
- "Unpopular opinion" or "hot take" threads
- Any content that is not clearly and specifically about doing technical work

Return a JSON object with the following keys:
1. "classification": "productive" (only clearly work-related technical content) or "distracting" (everything else).
2. "reasoning": Brief explanation.
3. "tags": Array of strings from ONLY these options: ["music", "ambient", "entertainment", "news", "edu", "learning", "content-consumption", "time-sink"].
4. "confidence_score": Float (0.0 - 1.0)
`
		inputTmpl = `
The user is currently visiting an X/Twitter page. Classify the page based on the following information:

URL: %s
Title: %s
`
	)

	input := fmt.Sprintf(inputTmpl, url, title)

	response, err := s.classifyWithGemini(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

func (s *Service) classifyHackerNews(ctx context.Context, url, title string) (*ClassificationResponse, error) {
	mainContent, err := fetchMainContent(ctx, url)
	if err != nil {
		slog.Error("failed to fetch main content", "error", err)
	}

	var (
		instructions = `
You are a Hacker News Software Engineer Intent Classifier. Your job is to determine if the user is actively doing work related to their software engineering job or seeking entertainment/distraction. Be RESTRICTIVE: only clearly work-related technical content is productive.

# Classification Logic

## **productive**
**Criteria:** ONLY content that directly helps with software engineering work. Must be clearly technical and actionable.
**Signals to detect:**
- Technical articles about programming languages, frameworks, tools, or infrastructure
- "Show HN" posts with genuine technical depth: new libraries, tools, architectures, or engineering solutions
- "Ask HN" posts asking about specific technical problems: debugging, architecture decisions, tool selection, performance
- Engineering blog posts: postmortems, system design, scaling, security, monitoring, deployment
- Deep-dives into algorithms, data structures, distributed systems, databases, networking
- Open source project announcements with technical substance
- Language/framework release notes and migration guides
- Security advisories, CVE discussions, vulnerability analysis
- DevOps, SRE, and infrastructure discussions with technical depth

## **distracting**
**Criteria:** EVERYTHING ELSE. When in doubt, classify as distracting.
**Includes:**
- Political threads, policy discussions, regulation debates
- Career advice, salary discussions, "Who is hiring?" threads, interview experiences
- "Ask HN: What's your favorite X?" or opinion polls without technical depth
- Lifestyle content, productivity tips, work-life balance discussions
- Startup drama, company news without technical relevance, funding announcements
- Non-technical "Show HN" (e.g. games, art projects, lifestyle apps without technical depth)
- Meta-HN discussions about moderation, community, or voting
- Science or academic content not directly related to software engineering
- General interest articles (history, philosophy, economics) even if intellectually stimulating
- Comment threads that have devolved into off-topic debate

Return a JSON object with the following keys:
1. "classification": "productive" (only clearly work-related technical content) or "distracting" (everything else).
2. "reasoning": Brief explanation.
3. "tags": Array of strings from ONLY these options: ["music", "ambient", "entertainment", "news", "edu", "learning", "content-consumption", "time-sink"].
4. "confidence_score": Float (0.0 - 1.0)
`
	)

	inputTmpl := `
The user is currently visiting a Hacker News page. Classify the page based on the following information:

URL: %s
Title: %s
Main Content (if available): %s
`

	input := fmt.Sprintf(inputTmpl, url, title, mainContent)

	response, err := s.classifyWithGemini(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

// classifySubstack classifies Substack newsletter articles using OpenGraph metadata and content extraction.
// Substack pages are generally scrapable without heavy Cloudflare challenges.
func (s *Service) classifySubstack(ctx context.Context, url, title string) (*ClassificationResponse, error) {
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
		instructions = `
You are a Substack Software Engineer Intent Classifier. Your job is to determine if the user is actively doing work related to their software engineering job or seeking entertainment/distraction. Be RESTRICTIVE: only clearly work-related technical content is productive.

# Classification Logic

## **productive**
**Criteria:** ONLY content that directly helps with software engineering work. Must be clearly technical and actionable.
**Signals to detect:**
- Technical newsletters about programming, system design, architecture, DevOps, security
- Engineering deep-dives: database internals, distributed systems, networking, performance
- Language or framework tutorials, guides, and best practices
- Security advisories, vulnerability analysis, threat modeling
- Infrastructure and cloud engineering content
- Open source project updates with technical substance
- Technical book reviews or paper summaries relevant to software engineering
- Newsletter authors known for technical content (e.g. engineering-focused Substacks)
- URL subdomains or slugs containing technical terms: programming languages, frameworks, tools

## **distracting**
**Criteria:** EVERYTHING ELSE. When in doubt, classify as distracting.
**Includes:**
- Self-help, productivity advice, hustle culture, motivational content
- Career advice, job market analysis, salary discussions, interview tips
- Opinion pieces, hot takes, and thought leadership without technical depth
- Political commentary, news analysis, cultural criticism
- Lifestyle content, wellness, personal finance, relationship advice
- General interest newsletters (history, philosophy, economics)
- Tech industry gossip, company drama, funding news without technical relevance
- "State of the industry" pieces that are opinion-heavy and light on technical content
- Any content that is not clearly and specifically about doing technical work

Return a JSON object with the following keys:
1. "classification": "productive" (only clearly work-related technical content) or "distracting" (everything else).
2. "reasoning": Brief explanation.
3. "tags": Array of strings from ONLY these options: ["music", "ambient", "entertainment", "news", "edu", "learning", "content-consumption", "time-sink"].
4. "confidence_score": Float (0.0 - 1.0)
`
		inputTmpl = `
The user is currently visiting a Substack article. Classify the article based on the following information:

URL: %s
Title: %s
Description: %s
Main Content (if available): %s
`
	)

	input := fmt.Sprintf(inputTmpl, url, ogTitle, ogDescription, mainContent)

	response, err := s.classifyWithGemini(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

// classifySlack classifies Slack activity using the browser tab title (or window title for desktop).
// No HTTP fetching is needed — the title contains the channel/DM name which is the primary signal.
// Titles typically look like: "#engineering - Acme Corp Slack" or "Slack | #random | My Workspace"
func (s *Service) classifySlack(ctx context.Context, url, title string) (*ClassificationResponse, error) {
	var (
		instructions = `
You are a Slack Software Engineer Intent Classifier. Your job is to determine if the user is engaged in productive work communication, general organizational activity, or distracted by non-essential chatter. You will primarily receive the browser tab title (or desktop window title) — this is your main signal.

# Classification Logic

## **productive**
**Criteria:** Direct work communication, project discussions, incident response, technical collaboration, or focused async work.
**Indicators:**
- Work-related channels: #engineering, #product, #design, #support, #incidents, #deploys, #standup, #sprint-*, #dev-*, #backend, #frontend, #infra, #security, #ops, #platform, #release-*, #bug-*, #feature-*, #project-*, #review-*, #ci-*, #monitoring, #alerts, #on-call
- DMs discussing work tasks (inferred from title context)
- Threads with technical discussions
- Huddles or calls (likely meetings)
- Canvas or document collaboration
- Any channel name that clearly relates to a specific project, team function, or work task

## **neutral**
**Criteria:** General organizational communication that is neither clearly productive nor distracting. Company-wide or team-wide channels used for announcements and coordination.
**Indicators:**
- General channels: #general, #announcements, #company, #all-hands, #team-*, #org-*, #office-*, #hr, #it-support, #helpdesk, #onboarding, #welcome
- Channels that serve organizational purposes without being directly about project work
- Workspace home page or search without specific channel context
- Browsing channel list without engaging (title contains "Browse channels" or similar)

## **distracting**
**Criteria:** Social channels, watercooler chat, entertainment, or passive browsing without clear work purpose.
**Indicators:**
- Social/fun channels: #random, #watercooler, #pets, #food, #memes, #off-topic, #fun-*, #chat-*, #social-*, #music, #gaming, #sports, #books, #movies, #travel, #fitness, #jokes, #dogs, #cats, #photos
- Content consumption channels: #links, #articles, #videos, #interesting-*, #cool-*
- Any channel name that suggests entertainment, socializing, or non-work activity

# Output Format

Return a JSON object with the following keys:
1. "classification": "productive" (work communication), "neutral" (general org channels), or "distracting" (social/entertainment).
2. "reasoning": Brief explanation.
3. "tags": Array of strings from ONLY these options: ["communication", "entertainment", "time-sink", "content-consumption"].
4. "confidence_score": Float (0.0 - 1.0)
5. "detected_communication_channel": The channel or DM name extracted from the title (e.g., "#engineering", "#random", "DM with John", "thread in #product"). Return empty string if unable to determine.
`
		inputTmpl = `
The user is currently using Slack. Classify their activity based on the following information:

URL (if available): %s
Title: %s
`
	)

	input := fmt.Sprintf(inputTmpl, url, title)

	response, err := s.classifyWithGemini(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

func (s *Service) classifyWithGemini(ctx context.Context, instructions, input string) (*ClassificationResponse, error) {
	if s.genaiClient == nil {
		return nil, errors.New("genai client not configured")
	}

	resp, err := s.genaiClient.Models.GenerateContent(ctx, "gemini-3-flash-preview", []*genai.Content{
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

	replacer := strings.NewReplacer("```json", "", "`", "")
	text = replacer.Replace(text)

	var response ClassificationResponse

	if err := json.Unmarshal([]byte(text), &response); err != nil {
		return nil, err
	}

	response.ClassificationSource = ClassificationSourceCloudLLM

	return &response, nil
}
