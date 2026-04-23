import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { Browser } from "@wailsio/runtime";
import {
  IconWorld,
  IconAppWindow,
  IconChevronDown,
  IconChevronUp,
} from "@tabler/icons-react";
import { Badge } from "@/components/ui/badge";
import { useResumeProtection, useIsProtectionPaused } from "@/hooks/queries/use-protection";
import { useAccountStore } from "@/stores/account-store";
import type { Event as TimelineEvent } from "../../../bindings/github.com/focusd-so/focusd/internal/timeline/models";
import {
  ClassificationSource,
  EnforcementAction,
  EnforcementSource,
  type Application,
  type ApplicationUsagePayload,
} from "../../../bindings/github.com/focusd-so/focusd/internal/usage/models";
import {
  pickClassificationSandbox,
  pickEnforcementSandbox,
  pickClassificationTags,
  pickEnforced,
  safeHostname,
} from "@/lib/timeline";
import {
  formatClassificationLabel,
  formatClassificationSource,
  formatDuration,
  formatEnforcementSource,
  formatSmartDate,
  hasSandboxData,
  isDistracting,
  isNeutralOrSystem,
} from "@/components/usage-item/formatters";
import { ClassificationReasoningLabel } from "@/components/usage-item/reasoning-label";
import { UsageItemSandboxPanel } from "@/components/usage-item/sandbox-panel";
import { TruncatedLabel } from "@/components/usage-item/truncated-label";

function getThemeClasses(
  isYellowTheme: boolean,
  isDistractingEvent: boolean,
  isGrayTheme: boolean,
) {
  if (isYellowTheme) {
    return {
      container:
        "bg-yellow-500/5 border-yellow-500/20 text-yellow-400 hover:bg-yellow-500/10",
      iconBg: "bg-yellow-500/10",
      badge: "border border-yellow-500/30 text-yellow-400",
    };
  }
  if (isDistractingEvent) {
    return {
      container: "bg-red-500/5 border-red-500/20 text-red-400 hover:bg-red-500/10",
      iconBg: "bg-red-500/10",
      badge: "border border-red-500/30 text-red-400",
    };
  }
  if (isGrayTheme) {
    return {
      container:
        "bg-zinc-500/5 border-zinc-500/20 text-zinc-400 hover:bg-zinc-500/10",
      iconBg: "bg-zinc-500/10",
      badge: "border border-zinc-500/30 text-zinc-400",
    };
  }
  return {
    container: "bg-green-500/5 border-green-500/20 text-green-400 hover:bg-green-500/10",
    iconBg: "bg-green-500/10",
    badge: "border border-green-500/30 text-green-400",
  };
}

function UsageAvatar({
  application,
  hostname,
  isWeb,
  iconBgClass,
}: {
  application?: Application | null;
  hostname?: string;
  isWeb: boolean;
  iconBgClass: string;
}) {
  return (
    <div
      className={`w-8 h-8 rounded-md flex items-center justify-center overflow-hidden shrink-0 ${iconBgClass}`}
    >
      {application?.icon ? (
        <img
          src={
            application.icon.startsWith("data:")
              ? application.icon
              : `data:image/png;base64,${application.icon}`
          }
          alt={hostname || application.name}
          className="w-8 h-8 object-contain"
        />
      ) : isWeb ? (
        <IconWorld className="w-8 h-8" />
      ) : (
        <IconAppWindow className="w-8 h-8" />
      )}
    </div>
  );
}

function RuleOutcomeNote({
  isIgnoredRule,
  customRulesAction,
  checkoutLink,
}: {
  isIgnoredRule: boolean;
  customRulesAction?:
    | (typeof EnforcementAction)[keyof typeof EnforcementAction]
    | null;
  checkoutLink?: string | null;
}) {
  if (!isIgnoredRule) return null;

  const wouldBlock =
    customRulesAction === EnforcementAction.EnforcementActionBlock;
  const verb = wouldBlock ? "block" : "allow";

  return (
    <div className="inline-flex min-w-0 items-center gap-1.5 text-[10px] font-medium text-amber-300/85">
      <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-amber-400/70" />
      <span className="truncate">
        Your rule would {verb} this
        {checkoutLink ? " — " : ""}
        {checkoutLink && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              Browser.OpenURL(checkoutLink);
            }}
            className="whitespace-nowrap font-medium text-amber-200 underline decoration-amber-200/40 underline-offset-2 transition-colors hover:text-amber-100"
          >
            upgrade →
          </button>
        )}
      </span>
    </div>
  );
}

function UsageMainInfo({
  payload,
  application,
  hostname,
  isWeb,
  startedAt,
  durationSeconds,
  isDistractingEvent,
}: {
  payload: ApplicationUsagePayload | null;
  application?: Application | null;
  hostname?: string;
  isWeb: boolean;
  startedAt?: number | null;
  durationSeconds: number | null;
  isDistractingEvent: boolean;
}) {
  return (
    <div className="flex min-w-0 flex-1 flex-col gap-1">
      <div className="flex min-w-0 items-center gap-1.5">
        <TruncatedLabel className="text-xs font-semibold text-foreground truncate leading-tight">
          {hostname || application?.name || "Unknown"}
        </TruncatedLabel>
        <span className="shrink-0 rounded border border-border/40 bg-muted/25 px-1 py-px text-[9px] font-medium text-muted-foreground/75">
          {payload?.classification
            ? formatClassificationLabel(payload.classification)
            : isDistractingEvent
              ? "Distracting"
              : "Productive"}
        </span>
        <span className="text-[10px] text-muted-foreground/50 tabular-nums leading-none shrink-0">
          at {formatSmartDate(startedAt)}
        </span>
      </div>
      <div className="flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-1">
        {payload?.classification_source ===
          ClassificationSource.ClassificationSourceCustomRules && (
          <Link
            to="/settings"
            search={{ tab: "rules" }}
            className="inline-flex items-center rounded border border-border/35 bg-muted/20 px-1 py-px text-[9px] text-muted-foreground/65 transition-colors hover:border-border/60 hover:bg-muted/35 hover:text-muted-foreground"
            onClick={(e) => e.stopPropagation()}
          >
            rules
          </Link>
        )}
        <TruncatedLabel className="max-w-[130px] truncate text-[10px] text-muted-foreground sm:max-w-[210px] lg:max-w-[250px]">
          {payload?.window_title || (isWeb ? "Browsing" : "Using app")}
        </TruncatedLabel>
        {durationSeconds != null && durationSeconds >= 60 && (
          <span className="text-[10px] font-medium text-muted-foreground/50 tabular-nums shrink-0">
            · {formatDuration(durationSeconds)}
          </span>
        )}
      </div>
    </div>
  );
}

export function UsageItem({
  event,
  payload,
  application,
}: {
  event: TimelineEvent;
  payload: ApplicationUsagePayload | null;
  application?: Application | null;
}) {
  const [showLogs, setShowLogs] = useState(false);
  const enforced = pickEnforced(payload);
  const resumeMutation = useResumeProtection();
  const isCurrentlyPaused = useIsProtectionPaused();
  const checkoutLink = useAccountStore((state) => state.checkoutLink);

  const browserURL = payload?.browser_url || undefined;
  const hostname = safeHostname(browserURL) ?? application?.domain ?? undefined;
  const isWeb = !!hostname;
  const tags = pickClassificationTags(payload);

  const classification = payload?.classification;
  const classificationSource = payload?.classification_source;
  const isDistractingEvent = isDistracting(classification);
  const isGrayTheme = isNeutralOrSystem(classification);

  const enforcementAction = enforced?.Action;
  const enforcementSource = enforced?.Source;

  const isPausedDistraction =
    isDistractingEvent &&
    enforcementAction === EnforcementAction.EnforcementActionPaused;

  const isAllowedDistraction =
    isDistractingEvent &&
    enforcementAction === EnforcementAction.EnforcementActionAllow &&
    (enforcementSource === EnforcementSource.EnforcementSourceCustomRules ||
      enforcementSource === EnforcementSource.EnforcementSourceAllowed);

  const theme = getThemeClasses(
    isPausedDistraction || isAllowedDistraction,
    isDistractingEvent,
    isGrayTheme,
  );

  const sourceMeta = formatClassificationSource(
    classificationSource,
    classification,
    payload?.classification_reason,
  );
  const enforcementSourceMeta = formatEnforcementSource(
    enforcementSource,
    payload?.classification_reason,
  );

  const startedAt = event.occurred_at;
  const endedAt = event.ended_at;
  const durationSeconds = endedAt && startedAt ? endedAt - startedAt : null;

  const classificationSandbox = pickClassificationSandbox(payload);
  const enforcementSandbox = pickEnforcementSandbox(payload);
  const hasClassificationSandbox = hasSandboxData(classificationSandbox);
  const hasEnforcementSandbox = hasSandboxData(enforcementSandbox);
  const hasAnySandbox = hasClassificationSandbox || hasEnforcementSandbox;

  const standardAction = payload?.enforcement_result?.StandardEnforcementResult?.Action;
  const customRulesAction = payload?.enforcement_result?.CustomRulesEnforcementResult?.Action;
  const isCustomRulesApplied =
    enforcementSource === EnforcementSource.EnforcementSourceCustomRules;

  const hasComparableActualAction =
    standardAction === EnforcementAction.EnforcementActionBlock ||
    standardAction === EnforcementAction.EnforcementActionAllow;
  const hasComparableCustomRulesAction =
    customRulesAction === EnforcementAction.EnforcementActionBlock ||
    customRulesAction === EnforcementAction.EnforcementActionAllow;

  const isIgnoredRule =
    hasComparableActualAction &&
    hasComparableCustomRulesAction &&
    !isCustomRulesApplied &&
    standardAction !== customRulesAction;

  const visibleTags = tags?.slice(0, 2) ?? [];
  const hiddenTagCount = Math.max(0, (tags?.length ?? 0) - visibleTags.length);

  const onResume = () => {
    resumeMutation.mutate("user manually resumed");
  };

  const showFooter = isIgnoredRule || hasAnySandbox;

  return (
    <div
      className={`flex flex-col p-2.5 rounded-lg border transition-all ${theme.container}`}
    >
      <div className="flex items-start justify-between w-full gap-2 min-w-0">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <UsageAvatar
            application={application}
            hostname={hostname}
            isWeb={isWeb}
            iconBgClass={theme.iconBg}
          />

          <UsageMainInfo
            payload={payload}
            application={application}
            hostname={hostname}
            isWeb={isWeb}
            startedAt={startedAt}
            durationSeconds={durationSeconds}
            isDistractingEvent={isDistractingEvent}
          />
        </div>

        <div className="flex shrink-0 flex-col items-end gap-1 pl-1">
          <div className="flex max-w-[170px] flex-wrap items-center justify-end gap-1 sm:max-w-[240px]">
            <Badge
              variant="outline"
              className={`rounded-full px-1.5 py-0 text-[9px] font-medium ${theme.badge} opacity-90`}
            >
              {isWeb ? "web" : "app"}
            </Badge>
            {visibleTags.map((tag) => (
              <Badge
                key={tag}
                variant="outline"
                className={`rounded-full px-1.5 py-0 text-[9px] font-medium ${theme.badge} opacity-80`}
              >
                {tag}
              </Badge>
            ))}
            {hiddenTagCount > 0 && (
              <Badge
                variant="outline"
                className={`rounded-full px-1.5 py-0 text-[9px] font-medium ${theme.badge} opacity-65`}
              >
                +{hiddenTagCount}
              </Badge>
            )}
          </div>

          <div className="flex max-w-[170px] justify-end sm:max-w-[260px]">
            <ClassificationReasoningLabel
              payload={payload}
              sourceMeta={sourceMeta}
              isAllowedDistraction={isAllowedDistraction}
              isPausedDistraction={isPausedDistraction}
              isCurrentlyPaused={isCurrentlyPaused}
              onResume={onResume}
              enforcementSourceMeta={
                enforcementSourceMeta?.label === "custom rules"
                  ? enforcementSourceMeta
                  : null
              }
            />
          </div>
        </div>
      </div>

      {showFooter && (
        <div className="-mx-2.5 -mb-2.5 mt-2.5 flex flex-col overflow-hidden rounded-b-lg border-t border-border/10 bg-black/15">
          <div className="flex items-center justify-between px-2.5 py-1.5">
            <div className="min-w-0 flex-1">
              {hasAnySandbox && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    setShowLogs((prev) => !prev);
                  }}
                  className="flex items-center gap-1 text-[10px] font-semibold text-muted-foreground/60 hover:text-foreground transition-colors group"
                  aria-label="Toggle sandbox trace"
                  aria-expanded={showLogs}
                >
                  <span>Custom rules trace</span>
                  {showLogs ? (
                    <IconChevronUp className="h-3 w-3 transition-transform group-hover:-translate-y-0.5" />
                  ) : (
                    <IconChevronDown className="h-3 w-3 transition-transform group-hover:translate-y-0.5" />
                  )}
                </button>
              )}
            </div>

            <div className="shrink-0">
              <RuleOutcomeNote
                isIgnoredRule={isIgnoredRule}
                customRulesAction={customRulesAction}
                checkoutLink={checkoutLink}
              />
            </div>
          </div>

          {showLogs && hasAnySandbox && (
            <div className="border-t border-border/10 px-2.5 py-2.5">
              <UsageItemSandboxPanel
                classificationSandbox={classificationSandbox}
                enforcementSandbox={enforcementSandbox}
              />
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export {
  TruncatedLabel,
  formatClassificationSource,
  formatDuration,
  formatEnforcementSource,
  formatSmartDate,
};
