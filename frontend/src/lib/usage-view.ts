import type { Event as TimelineEvent } from "../../bindings/github.com/focusd-so/focusd/internal/timeline/models";
import type {
  Application,
  CustomRulesClassificationResult,
  EnforcementResult,
} from "../../bindings/github.com/focusd-so/focusd/internal/usage/models";
import { parsePayload, type ApplicationUsagePayload } from "@/lib/timeline";

export interface UsageItemView {
  id: number;
  application: {
    id?: number;
    name?: string;
    hostname?: string;
    icon?: string | null;
  } | null;
  window_title: string;
  browser_url?: string;
  classification?: string;
  classification_source?: string;
  classification_reason?: string;
  enforcement_action?: string;
  enforcement_source?: string;
  enforcement_reason?: string;
  tags: string[];
  started_at: number;
  ended_at: number | null;
  classification_sandbox?: {
    context?: string;
    response?: string;
    logs?: string;
  };
  enforcement_sandbox?: {
    context?: string;
    response?: string;
    logs?: string;
  };
}

function safeHostname(rawURL: string | undefined): string | undefined {
  if (!rawURL) return undefined;
  try {
    return new URL(rawURL).hostname || undefined;
  } catch {
    return undefined;
  }
}

function pickEnforced(result: EnforcementResult | undefined) {
  if (!result) return undefined;
  return result.StandardEnforcementResult ?? result.CustomRulesEnforcementResult ?? undefined;
}

function classificationSandbox(custom: CustomRulesClassificationResult | null | undefined) {
  if (!custom) return undefined;
  return {
    context: custom.sandbox_context,
    response: custom.sandbox_output ?? undefined,
    logs:
      custom.sandbox_logs && custom.sandbox_logs.length > 0
        ? JSON.stringify(custom.sandbox_logs)
        : undefined,
  };
}

export function toUsageItemView(
  event: TimelineEvent,
  application?: Application | null,
): UsageItemView {
  const payload = parsePayload<ApplicationUsagePayload>(event);
  const enforced = pickEnforced(payload?.enforcement_result);
  const hostname = safeHostname(payload?.browser_url);

  return {
    id: event.id,
    application: payload
      ? {
          id: payload.application_id,
          name: application?.name,
          hostname: hostname ?? application?.domain ?? undefined,
          icon: application?.icon ?? null,
        }
      : null,
    window_title: payload?.window_title ?? "",
    browser_url: payload?.browser_url || undefined,
    classification: payload?.classification,
    classification_source: payload?.classification_source,
    classification_reason: payload?.classification_reason,
    enforcement_action: enforced?.Action,
    enforcement_source: enforced?.Source,
    enforcement_reason: enforced?.Reason,
    tags: payload?.tags ?? [],
    started_at: event.occurred_at,
    ended_at: event.ended_at,
    classification_sandbox: classificationSandbox(
      payload?.classification_result?.custom_rules_classification_result,
    ),
    // The backend currently does not surface enforcement-sandbox traces inline
    // on the usage event; see internal/usage/protection_enforcement.go where
    // the trace is logged as a separate timeline event. Keep the field for
    // forward compatibility.
    enforcement_sandbox: undefined,
  };
}

export function buildApplicationsById(applications: Application[] | undefined): Map<number, Application> {
  const map = new Map<number, Application>();
  for (const app of applications ?? []) {
    if (app?.id) map.set(app.id, app);
  }
  return map;
}
