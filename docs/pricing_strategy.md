# Focusd Pricing Strategy

This document summarizes the strategic decisions and implementation details regarding Focusd's pricing model.

## 1. The Core Philosophy: Free Core, Paid Power Features

After evaluating different models, we have adopted a **Product-Led Growth (PLG)** approach:

- **The Core Value is Free:** AI classification of activities and the core distraction blocking engine are free forever. The models used for this (e.g., Gemini Flash) are inexpensive enough that we can offer the foundational value of the app without arbitrary limits.
- **Why Free Forever?:** A permanent, robust free tier drives top-of-funnel growth and word-of-mouth marketing. It allows users to build a daily habit and dependency on Focusd, while also providing us with a large dataset for ongoing product improvement.
- **Monetizing Professional Value:** If the core engine blocks distractions, the paid tier is designed to _actively facilitate productivity_. We monetize power features aimed at professionals who need seamless workflow integration.

## 2. The Paid Tier: Power Features & Integrations

Users upgrade to the paid tier to context-switch less, customize their AI, and get more structured work done. Future premium features will include:

- **Deep Integrations:** Connecting to GitHub, Jira, Slack, Notion, etc.
- **Intelligent Workflows:** Fetching classified, structured to-dos directly into the user's workspace based on their active tasks.
- **Custom AI Prompts:** Allowing users to define entirely custom classification rules for specific apps or domains (e.g., "On youtube.com, only allow videos with 'GoLang' or 'Tutorial' in the title").
- **Custom Rules & Analytics:** Granular control over what constitutes a "distraction" and deep insights into focus habits over time.

Instead of paying to remove a limitation, the user is paying to add a high-value workflow capability.

## 3. Server Architecture & Telemetry

Even with a generous free tier, we must protect our APIs and make data-driven decisions:

- **Aggressive Caching:** We use intensive server-side caching for previously classified URLs and window titles. If 1,000 users visit the same popular site, we only pay for the LLM inference once, driving our effective API costs close to zero for common activities.
- **Fair Use & Security:** All API access is routed through our backend proxy server to prevent abuse and enforce "fair use" ceilings (to prevent malicious exploitation of the free tier).
- **Telemetry:** We log ALL requests (both Free and Pro).
  - We track the **Classification Result** (`productive` vs `distracting`).
  - We count the exact **Input and Output Tokens**.

By combining caching and telemetry, we can calculate exact API costs per cohort, ensure the economics of the free tier remain sustainable, and confidently scale the platform based on real-world usage patterns.
