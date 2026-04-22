import {
  Classification,
  ClassificationSource,
  EnforcementSource,
} from "../../../bindings/github.com/focusd-so/focusd/internal/usage/models";

export function isDistracting(
  classification?: (typeof Classification)[keyof typeof Classification] | null,
): boolean {
  return classification === Classification.ClassificationDistracting;
}

export function isNeutralOrSystem(
  classification?: (typeof Classification)[keyof typeof Classification] | null,
): boolean {
  return (
    classification === Classification.ClassificationNeutral ||
    classification === Classification.ClassificationSystem
  );
}

export function formatSmartDate(unixSeconds: number | null | undefined): string {
  if (!unixSeconds) return "";
  const date = new Date(unixSeconds * 1000);
  const now = new Date();
  const isToday = date.toDateString() === now.toDateString();

  if (isToday) {
    const hours = date.getHours();
    const mins = date.getMinutes();
    const h = hours % 12 || 12;
    const ampm = hours < 12 ? "am" : "pm";
    return `${h}:${mins.toString().padStart(2, "0")}${ampm}`;
  }

  return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

export function formatDuration(seconds: number): string {
  if (seconds <= 0) return "0s";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m`;
  return `${s}s`;
}

export function formatClassificationLabel(
  classification?: (typeof Classification)[keyof typeof Classification] | null,
): string {
  switch (classification) {
    case Classification.ClassificationDistracting:
      return "Distracting";
    case Classification.ClassificationProductive:
      return "Productive";
    case Classification.ClassificationNeutral:
      return "Neutral";
    case Classification.ClassificationSystem:
      return "System";
    default:
      return "Productive";
  }
}

export function formatClassificationSource(
  source?: (typeof ClassificationSource)[keyof typeof ClassificationSource] | null,
  classification?: (typeof Classification)[keyof typeof Classification] | null,
  reasoning?: string | null,
) {
  if (reasoning) {
    return {
      label: reasoning,
      description: reasoning,
      icon: "",
      isLink: false,
    };
  }

  switch (source) {
    case ClassificationSource.ClassificationSourceObviously:
      if (classification === Classification.ClassificationProductive) {
        return {
          label: "work",
          icon: "⚡️",
          description: "It's obviously productive",
          isLink: false,
        };
      }
      return {
        label: "duh",
        icon: "🙄",
        description: "C'mon... it's obviously distracting",
        isLink: false,
      };
    case ClassificationSource.ClassificationSourceLLMGemini:
    case ClassificationSource.ClassificationSourceLLMOpenAI:
    case ClassificationSource.ClassificationSourceLLMGroq:
    case ClassificationSource.ClassificationSourceLLMAnthropic:
      return {
        label: "ai",
        icon: "✨",
        description: "Classified by AI based on context",
        isLink: false,
      };
    case ClassificationSource.ClassificationSourceCustomRules:
      return {
        label: "custom rules",
        icon: "📋",
        description: "Matched your custom blocking rules",
        isLink: true,
      };
    default:
      return {
        label: source || "unknown",
        icon: "❓",
        description: "Unknown classification source",
        isLink: false,
      };
  }
}

export function formatEnforcementSource(
  source?: (typeof EnforcementSource)[keyof typeof EnforcementSource] | null,
  reasoning?: string | null,
) {
  if (!source || source === EnforcementSource.EnforcementSourceApplication) {
    return null;
  }

  switch (source) {
    case EnforcementSource.EnforcementSourceCustomRules:
      return {
        label: "custom rules",
        icon: "⚡️",
        description: reasoning || "Action determined by your custom rules",
        isLink: true,
      };
    case EnforcementSource.EnforcementSourceAllowed:
      return {
        label: "allowed",
        icon: "✓",
        description: "Temporarily allowed by you",
        isLink: false,
      };
    case EnforcementSource.EnforcementSourcePaused:
      return {
        label: "paused",
        icon: "⏸",
        description: "Focus protection is temporarily paused",
        isLink: false,
      };
    default:
      return {
        label: source,
        icon: "❓",
        description: "Unknown enforcement source",
        isLink: false,
      };
  }
}

export function hasSandboxResult(response?: string | null): boolean {
  if (!response) return false;
  const normalized = response.trim().toLowerCase();
  return (
    normalized !== "" &&
    normalized !== "no response" &&
    normalized !== "null" &&
    normalized !== "undefined"
  );
}

export function hasSandboxData(sandbox?: {
  context?: string;
  response?: string;
  logs?: string;
} | null): boolean {
  if (!sandbox) return false;

  const hasContext = hasSandboxResult(sandbox.context);
  const hasResponse = hasSandboxResult(sandbox.response);

  if (hasContext || hasResponse) return true;

  if (!hasSandboxResult(sandbox.logs)) return false;

  try {
    const parsed = JSON.parse(sandbox.logs as string);
    if (Array.isArray(parsed)) return parsed.length > 0;
  } catch {
    return true;
  }

  return true;
}

export function tryParseJSON(str: string | null | undefined): string {
  if (!str) return "—";
  try {
    return JSON.stringify(JSON.parse(str), null, 2);
  } catch {
    return str;
  }
}

export function formatSandboxLogs(logsStr: string | null | undefined): string {
  if (!logsStr) return "";
  try {
    const logs = JSON.parse(logsStr);
    if (Array.isArray(logs)) return logs.join("\n");
  } catch {
    return logsStr;
  }
  return logsStr;
}
