package usage

import (
	"context"
	"fmt"
)

func (s *Service) classifyApplication(ctx context.Context, appName, title string, url *string) (*ClassificationResponse, error) {
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
3. "tags": Array of strings from ONLY these options: ["coding", "docs", "debug", "communication", "planning", "learning", "entertainment", "news", "social", "shopping", "other"].
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
