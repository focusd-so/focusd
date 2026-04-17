import { Event } from "../../bindings/github.com/focusd-so/focusd/internal/timeline/models";
import type {
  AllowUsagePayload,
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

// Re-export payload shapes for convenience.
export type {
  AllowUsagePayload,
  ApplicationUsagePayload,
  CustomRulesTracePayload,
  PauseProtectionPayload,
};
