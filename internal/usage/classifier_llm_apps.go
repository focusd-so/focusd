package usage

import (
	"context"
	"fmt"
	"strings"
)

func (s *Service) classifyApplication(ctx context.Context, appName, title string, url *string) (*ClassificationResponse, error) {
	switch strings.ToLower(appName) {
	case "slack":
		return s.classifySlackApp(ctx, appName, title)
	}

	var (
		instructions = `
You are a Software Engineering Application Intent Classifier. Your job is to determine if the user is actively doing work related to their software engineering job or seeking entertainment/distraction.

# Classification Logic

## **productive**
**Criteria:** Applications that directly support software engineering tasks: coding, debugging, learning, researching, planning, or deploying.
**Signals to detect:**
- **Editors/IDEs:** VS Code, Xcode, IntelliJ, PyCharm, GoLand, etc.
- **Terminals/CLIs:** Terminal, iTerm, shells, build tools, git clients.
- **Developer Tools:** API clients, DB clients, profilers, debuggers.
- **Documentation/Research apps:** PDF readers for technical docs, reference tools.
- **Project Management:** Jira, Linear, Trello, Notion when used for work.

## **distracting**
**Criteria:** Apps primarily for entertainment or non-work consumption.
**Includes:**
- Social media, chat apps used socially (Discord, WhatsApp, Facebook Messenger).
- Streaming video/music (YouTube Music, Spotify, Netflix).
- Games, gaming platforms.
- Shopping, news, celebrity gossip.

## **neutral**
**Criteria:** Ambiguous utilities or general organizational tools.
**Includes:**
- Email, Calendar, Notes, general file managers.
- General communication apps without context (Slack, Teams).
- System settings or OS utilities.

Return a JSON object with the following keys:
1. "classification": "productive", "neutral", or "distracting".
2. "reasoning": Brief explanation.
3. "tags": Array of strings from ONLY these options: ["coding", "docs", "debug", "communication", "terminal", "planning", "learning", "entertainment", "news", "social", "shopping", "terminal", "other"].
   - IMPORTANT: Be extremely conservative with the "communication" tag. Only use it when there is actual messaging, emailing, or chatting happening. Do NOT use it for reading code reviews, terminal multiplexers, or project management.
4. "confidence_score": Float (0.0 - 1.0)
`
		inputTmpl = `
The user is currently using an application. Classify the activity based on the following information:

Application Name: %s
Window Title: %s
Executable Path: %s
`
	)

	urlValue := ""
	if url != nil {
		urlValue = *url
	}

	input := fmt.Sprintf(inputTmpl, appName, title, urlValue)

	response, err := s.classifyWithGemini(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}

// classifySlackApp classifies Slack desktop application activity using the window title.
// Titles typically look like: "Slack | #engineering | Acme Corp" or "Slack - #random - Workspace"
func (s *Service) classifySlackApp(ctx context.Context, appName, title string) (*ClassificationResponse, error) {
	var (
		instructions = `
You are a Slack Software Engineer Intent Classifier. Your job is to determine if the user is engaged in productive work communication, general organizational activity, or distracted by non-essential chatter. You will receive the desktop window title — this is your main signal.

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
5. "detected_communication_channel": The channel or DM name extracted from the title 
`
		inputTmpl = `
The user is currently using the Slack desktop application. Classify their activity based on the following information:

Application Name: %s
Window Title: %s
`
	)

	input := fmt.Sprintf(inputTmpl, appName, title)

	response, err := s.classifyWithGemini(ctx, instructions, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify with Gemini: %w", err)
	}

	return response, nil
}
