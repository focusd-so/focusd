import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { Browser } from "@wailsio/runtime";
import {
  IconWorld,
  IconAppWindow,
  IconTerminal,
  IconChevronDown,
  IconChevronRight,
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

function IgnoredRuleBanner({
  shouldRender,
  customRulesAction,
  checkoutLink,
}: {
  shouldRender: boolean;
  customRulesAction?:
    | (typeof EnforcementAction)[keyof typeof EnforcementAction]
    | null;
  checkoutLink?: string | null;
}) {
  if (!shouldRender) return null;

  const wouldBlock =
    customRulesAction === EnforcementAction.EnforcementActionBlock;

  return (
    <div className="self-start text-[9px] text-amber-200/90 bg-amber-500/10 px-2 py-1 rounded border border-amber-400/20 mt-1 flex items-center gap-1.5 animate-in fade-in slide-in-from-left-1 duration-500">
      <IconTerminal className="w-2.5 h-2.5 shrink-0 text-amber-300/80" />
      <span className="truncate">
        Custom rules would have{" "}
        <span className="font-semibold uppercase text-amber-200">
          {wouldBlock ? "blocked" : "allowed"}
        </span>{" "}
        this action. <span className="text-amber-200/70">Upgrade to Plus to enforce custom rules.</span>
      </span>
      {checkoutLink && (
        <button
          onClick={(e) => {
            e.stopPropagation();
            Browser.OpenURL(checkoutLink);
          }}
          className="underline hover:text-amber-100 font-semibold whitespace-nowrap"
        >
          Upgrade to Plus
        </button>
      )}
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
  isIgnoredRule,
  customRulesAction,
}: {
  payload: ApplicationUsagePayload | null;
  application?: Application | null;
  hostname?: string;
  isWeb: boolean;
  startedAt?: number | null;
  durationSeconds: number | null;
  isDistractingEvent: boolean;
  isIgnoredRule: boolean;
  customRulesAction?:
    | (typeof EnforcementAction)[keyof typeof EnforcementAction]
    | null;
}) {
  const checkoutLink = useAccountStore((state) => state.checkoutLink);

  return (
    <div className="flex min-w-0 flex-1 flex-col">
      <div className="flex items-center gap-2 min-w-0">
        <TruncatedLabel className="text-xs font-semibold text-foreground truncate leading-tight">
          {hostname || application?.name || "Unknown"}
        </TruncatedLabel>
        <span className="text-[10px] text-muted-foreground/50 tabular-nums leading-none shrink-0">
          at {formatSmartDate(startedAt)}
        </span>
      </div>
      <div className="flex items-center gap-1.5 mt-0.5">
        <span className="text-[10px] font-medium uppercase tracking-widest opacity-70">
          {payload?.classification
            ? formatClassificationLabel(payload.classification)
            : isDistractingEvent
              ? "Distracting"
              : "Productive"}
        </span>
        {payload?.classification_source ===
          ClassificationSource.ClassificationSourceCustomRules && (
          <Link
            to="/settings"
            search={{ tab: "rules" }}
            className="inline-flex items-center gap-0.5 text-[9px] font-medium px-1 py-px rounded bg-muted/40 text-muted-foreground/50 border border-muted-foreground/10 hover:bg-muted/70 hover:text-muted-foreground/80 hover:border-muted-foreground/25 transition-colors"
            onClick={(e) => e.stopPropagation()}
          >
            ⚡️ custom rules
          </Link>
        )}
        <span className="text-muted-foreground/40 text-[10px]">—</span>
        <TruncatedLabel className="text-[10px] text-muted-foreground truncate max-w-[130px] sm:max-w-[210px] lg:max-w-[250px]">
          {payload?.window_title || (isWeb ? "Browsing" : "Using app")}
        </TruncatedLabel>
        {durationSeconds != null && durationSeconds >= 60 && (
          <span className="text-[10px] text-muted-foreground/50 tabular-nums shrink-0 font-medium">
            · {formatDuration(durationSeconds)}
          </span>
        )}
      </div>
      <IgnoredRuleBanner
        shouldRender={isIgnoredRule}
        customRulesAction={customRulesAction}
        checkoutLink={checkoutLink}
      />
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

  const onResume = () => {
    resumeMutation.mutate("user manually resumed");
  };

  return (
    <div
      className={`flex flex-col p-2.5 rounded-lg border transition-all ${theme.container}`}
    >
      <div className="flex items-center justify-between w-full gap-2 min-w-0">
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
            isIgnoredRule={isIgnoredRule}
            customRulesAction={customRulesAction}
          />
        </div>

        <div className="flex min-w-0 items-center gap-2 pl-1">
          <div className="flex min-w-0 flex-col items-end gap-1">
            <div className="flex flex-wrap items-center justify-end gap-1">
              <Badge
                variant="outline"
                className={`px-1.5 py-0 text-[9px] font-bold rounded-full ${theme.badge}`}
              >
                {isWeb ? "web" : "app"}
              </Badge>
              {tags?.map((tag) => (
                <Badge
                  key={tag}
                  variant="outline"
                  className={`px-1.5 py-0 text-[9px] font-bold rounded-full ${theme.badge}`}
                >
                  {tag}
                </Badge>
              ))}
            </div>

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

          {hasAnySandbox && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                setShowLogs((prev) => !prev);
              }}
              className={`px-1.5 py-1 rounded-md border transition-all flex items-center justify-center gap-1 ${showLogs
                ? "bg-muted/40 border-border/60 text-foreground"
                : "bg-muted/10 border-border/30 text-muted-foreground/60 hover:bg-muted/30 hover:border-border/60 hover:text-foreground"
                }`}
              title="Show sandbox trace (classification/enforcement)"
              aria-label="Toggle sandbox trace"
            >
              <span className="hidden md:inline text-[9px] font-semibold uppercase tracking-wider">
                Trace
              </span>
              {showLogs ? (
                <IconChevronDown className="w-4 h-4" />
              ) : (
                <IconChevronRight className="w-4 h-4" />
              )}
            </button>
          )}
        </div>
      </div>

      {showLogs && hasAnySandbox && (
        <UsageItemSandboxPanel
          classificationSandbox={classificationSandbox}
          enforcementSandbox={enforcementSandbox}
        />
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
