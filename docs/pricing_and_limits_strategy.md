# Focusd Pricing & AI Rate Limiting Strategy

This document summarizes the strategic decisions, trade-offs, and implementation details regarding Focusd's pricing model and AI usage limits.

## 1. The Core Dilemma: Freemium vs. 7-Day Trial

We debated whether to keep a permanent Freemium tier or switch to a strict 7-day trial.

- **The "Why Keep Free Users?" Argument:** If users won't pay $8 to solve their distraction problems, are they worth keeping around? Focusd is an indie app—we cannot absorb infinite API costs like VC-backed giants (OpenAI, Anthropic). Comparing our API burn rate to theirs is comparing "apples to a passing asteroid."
- **The Value of Freemium:** A permanent free tier drives top-of-funnel growth, word-of-mouth marketing, and provides a larger dataset for product feedback.
- **The Compromise:** We keep the free tier, but strictly cap the most expensive feature—server-side AI classifications via Gemini.

## 2. Choosing the Limit: Behavior-Driven Constraints

When deciding how many AI classifications to give Free users, we evaluated several caps:

- **15 distractions:** Felt completely arbitrary.
- **10 distractions:** Still too generous. Psychologically, if a user is getting distracted 10 times, they have a serious problem and are failing to focus.
- **Final Decision: 5 Distractions per Rolling Hour.**

### Why a Rolling Hour?

1. **Behavioral Coaching:** Instead of resetting at the top of the clock (e.g., 3:00 PM), a rolling hour ensures that if a user is highly distractible for a continuous period, they are put on a "cooldown." Their allowance gradually returns as their older distractions age past the 60-minute mark. This incentivizes sustained focus.
2. **Preventing Exploits:** It prevents "boundary bursts" where a user gets distracted 5 times at 2:55 PM, and immediately gets 5 more at 3:00 PM.

## 3. Server-Side Enforcement & Security

- **No Client-Side Trust:** The rate limits must be enforced on the backend proxy server. If limits were enforced locally on the desktop app, users could trivially bypass them by clearing local storage or intercepting DB calls.
- **Generic LLM Handling:** The proxy is built to intercept requests for any LLM provider (starting with Gemini), validate the user's JWT token, and check their tier against the central database before passing the request along.

## 4. Telemetry and Future Adjustments

We recognized that setting the limit to "5 distractions" is currently based on intuition. To make data-driven decisions later, we implemented extensive server-side telemetry:

- We log **ALL** requests (both Free and Pro).
- We track the **Classification Result** (`productive` vs `distracting`).
- We count the exact **Input and Output Tokens**.

By storing this raw data now, we can calculate our exact API costs per user tier and confidently adjust the "5 distractions" limit up or down in the future based on real-world usage patterns.
