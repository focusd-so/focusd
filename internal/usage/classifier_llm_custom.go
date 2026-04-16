package usage

import (
	"context"
	"fmt"
)

func classifySlackActivity(ctx context.Context, input string) (*LLMClassificationResult, error) {
	const instructionSlackClassification = `
You classify Slack activity for software engineers.

Task:
- Classify activity as one of: "productive", "neutral", "distracting".
- Extract "detected_communication_channel" from the title when possible.

Guidelines:
- productive: project/work/engineering/incident/release/support communication.
- neutral: general organization/announcements/company-wide communication, or unclear context.
- distracting: social/fun/off-topic/non-work communication.

Channel extraction:
- Use the visible title text to infer the channel or DM/thread name.
- Return only the channel or DM name, without extra prose.
- If no channel or DM can be reliably inferred, return an empty string.

Return JSON only with exactly these keys:
{
  "classification": "productive|neutral|distracting",
  "reasoning": "short reason",
  "detected_communication_channel": "channel name or empty string"
}
`
	response, err := classify(ctx, instructionSlackClassification, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify Slack activity: %w", err)
	}

	return response, nil
}
