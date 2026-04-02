const DEFAULT_CUSTOM_RULES_TS = `import {
  productive,
  distracting,
  block,
  Timezone,
  runtime,
  type Classify,
  type Enforce,
} from "@focusd/runtime";

/**
 * Classify determines whether the current app or website is productive or distracting.
 * It is called every time your usage changes.
 */
export function classify(): Classify | undefined {
  const { domain, app, current } = runtime.usage;

  // --- EXAMPLES ---
  //
  // 1. Mark specific domains as productive
  // if (domain === "github.com") return productive("Working on code");
  //
  // 2. Mark distracting based on today's stats & current usage
  // if (domain === "twitter.com" && current.usedToday > 30) {
  //   return distracting("Daily limit reached");
  // }
  //
  // 3. Use current hour stats to mark distraction
  // if (runtime.hour.distractingMinutes > 20 && current.last(60) > 15) {
  //   return distracting("Too much distraction this hour");
  // }

  return undefined;
}

/**
 * Enforcement determines whether or not to block the current app or website.
 * It is called when the current usage has been classified as distracting.
 */
export function enforcement(): Enforce | undefined {
  const { domain } = runtime.usage;

  // --- EXAMPLES ---
  //
  // 1. Block if daily distraction limit is reached
  // if (runtime.today.distractingMinutes > 60) {
  //   return block("Daily distraction limit reached");
  // }
  //
  // 2. Block after a specific time (e.g., 10 PM in London)
  // const hour = runtime.time.now(Timezone.Europe_London).getHours();
  // if (domain === "youtube.com" && hour >= 22) {
  //   return block("Late night blocking");
  // }

  return undefined;
}
`;

function normalizeRules(source: string): string {
  return source
    .replace(/\r\n/g, "\n")
    .split("\n")
    .map((line) => line.trimEnd())
    .join("\n")
    .trim();
}

export function hasNonDefaultCustomRules(customRules: string | null | undefined): boolean {
  if (!customRules) return false;

  const normalizedCurrent = normalizeRules(customRules);
  if (!normalizedCurrent) return false;

  const normalizedDefault = normalizeRules(DEFAULT_CUSTOM_RULES_TS);
  return normalizedCurrent !== normalizedDefault;
}
