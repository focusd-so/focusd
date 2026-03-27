export const STARTER_RULES_TS = `import {
  productive,
  distracting,
  block,
  Timezone,
  runtime,
  type Classify,
  type Enforce,
} from "@focusd/runtime";

export function classify(): Classify | undefined {
  const { domain, current } = runtime.usage;

  if (domain === "github.com") {
    return productive("GitHub is productive work");
  }

  if (runtime.hour.distractingMinutes > 30 && current.last(60) > 45) {
    return distracting("High distraction this hour");
  }

  return undefined;
}

export function enforcement(): Enforce | undefined {
  const { domain, current } = runtime.usage;

  if (domain === "twitter.com" && runtime.time.now(Timezone.Europe_London).getHours() >= 22) {
    return block("Blocked after 10 PM in London");
  }

  if (runtime.today.distractingMinutes >= 90 && current.blocks >= 3) {
    return block("High distraction day with repeated attempts");
  }

  return undefined;
}
`;
