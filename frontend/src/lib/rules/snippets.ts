export interface RuleSnippet {
  id: string;
  title: string;
  description: string;
  code: string;
}

export const RULE_SNIPPETS: RuleSnippet[] = [
  {
    id: "import-runtime",
    title: "Import Runtime SDK",
    description: "Insert the recommended import line.",
    code: `import { productive, distracting, neutral, block, Timezone, runtime, type Classify, type Enforce } from "@focusd/runtime";`,
  },
  {
    id: "productive-domain",
    title: "Classify Productive Domain",
    description: "Mark a domain as productive.",
    code: `if (runtime.usage.domain === "github.com") {
  return productive("GitHub is productive work");
}`,
  },
  {
    id: "hour-budget",
    title: "Hourly Usage Budget",
    description: "Mark as distracting after a threshold.",
    code: `if (runtime.usage.current.last(60) > 30) {
  return distracting("Exceeded hourly usage budget");
}`,
  },
  {
    id: "late-night-block",
    title: "Late Night Block",
    description: "Block social media after 10 PM.",
    code: `if (runtime.usage.domain === "twitter.com" && runtime.time.now(Timezone.Europe_London).getHours() >= 22) {
  return block("Blocked after 10 PM");
}`,
  },
  {
    id: "insights-trigger",
    title: "Insights-Based Enforcement",
    description: "Block when today's distraction is high with repeated attempts.",
    code: `if (runtime.today.distractingMinutes >= 90 && runtime.usage.current.blocks >= 3) {
  return block("High distraction day with repeated attempts");
}`,
  },
  {
    id: "cooldown-after-block",
    title: "Cooldown After Block",
    description: "Allow brief usage after a cooldown period since last block.",
    code: `if (runtime.usage.current.sinceBlock != null && runtime.usage.current.sinceBlock >= 20 && (runtime.usage.current.usedSinceBlock ?? 0) < 5) {
  return neutral("Allow 5 mins every 20 mins after block");
}`,
  },
];
