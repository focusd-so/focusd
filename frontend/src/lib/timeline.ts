import { Event } from "../../bindings/github.com/focusd-so/focusd/internal/timeline/models";
import type {
  AllowUsagePayload,
  Application,
  ApplicationUsagePayload,
  CustomRulesTracePayload,
  PauseProtectionPayload,
} from "../../bindings/github.com/focusd-so/focusd/internal/usage/models";

// Timeline event type identifiers (mirrors usage/types.go).
export const EventType = {
  ProtectionStatusChanged: "protection_status_changed",
  AllowUsage: "allow_usage",
  CustomRulesTrace: "custom_rules_trace",
  UsageChanged: "usage_changed",
  UserIdleChanged: "user_idle_changed",
} as const;

export type TimelineEvent = Event;

// parsePayload safely decodes the JSON payload of a timeline event into the
// requested struct. Returns null when the payload is missing or malformed.
export function parsePayload<T>(event: Event | null | undefined): T | null {
  if (!event || !event.payload) return null;
  try {
    return JSON.parse(event.payload) as T;
  } catch {
    return null;
  }
}

// isEventActive returns true when the event is still ongoing (no finishedAt or
// finishedAt is in the future).
export function isEventActive(event: Event | null | undefined): boolean {
  if (!event) return false;
  if (event.ended_at == null) return true;
  return event.ended_at * 1000 > Date.now();
}

export function safeHostname(rawURL: string | undefined): string | undefined {
  if (!rawURL) return undefined;
  try {
    return new URL(rawURL).hostname || undefined;
  } catch {
    return undefined;
  }
}

export function pickEnforced(payload: ApplicationUsagePayload | null | undefined) {
  const r = payload?.enforcement_result;
  return r?.StandardEnforcementResult ?? r?.CustomRulesEnforcementResult ?? undefined;
}

export function pickClassificationTags(
  payload: ApplicationUsagePayload | null | undefined,
): string[] {
  const c = payload?.classification_result;
  return (
    c?.custom_rules_classification_result?.tags ??
    c?.llm_classification_result?.tags ??
    c?.obviously_classification_result?.tags ??
    []
  );
}

export function pickClassificationSandbox(
  payload: ApplicationUsagePayload | null | undefined,
) {
  const custom = payload?.classification_result?.custom_rules_classification_result;
  return pickSandboxFields(custom as unknown as Record<string, unknown> | null | undefined);
}

export function pickEnforcementSandbox(
  payload: ApplicationUsagePayload | null | undefined,
) {
  const custom = payload?.enforcement_result?.CustomRulesEnforcementResult;
  return pickSandboxFields(custom as unknown as Record<string, unknown> | null | undefined);
}

export interface UsageSandboxData {
  context?: string;
  response?: string;
  logs?: string;
}

function pickSandboxFields(
  source: Record<string, unknown> | null | undefined,
): UsageSandboxData | undefined {
  if (!source) return undefined;

  const context = pickString(source, [
    "sandbox_context",
    "SandboxContext",
    "context",
    "Context",
  ]);
  const response = pickString(source, [
    "sandbox_output",
    "SanboxOutput",
    "SandboxOutput",
    "output",
    "Output",
  ]);
  const logs = pickLogs(source, [
    "sandbox_logs",
    "SandboLogs",
    "SandboxLogs",
    "logs",
    "Logs",
  ]);

  if (!context && !response && !logs) return undefined;

  return { context, response, logs };
}

function pickString(source: Record<string, unknown>, keys: string[]): string | undefined {
  for (const key of keys) {
    const value = source[key];
    if (typeof value === "string") {
      return value;
    }
  }
  return undefined;
}

function pickLogs(source: Record<string, unknown>, keys: string[]): string | undefined {
  for (const key of keys) {
    const value = source[key];
    if (Array.isArray(value)) {
      return value.length > 0 ? JSON.stringify(value) : undefined;
    }
    if (typeof value === "string") {
      return value;
    }
  }
  return undefined;
}

// Re-export payload shapes for convenience.
export type {
  AllowUsagePayload,
  Application,
  ApplicationUsagePayload,
  CustomRulesTracePayload,
  PauseProtectionPayload,
};
