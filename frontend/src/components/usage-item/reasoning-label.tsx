import { Link } from "@tanstack/react-router";
import { IconPlayerPause } from "@tabler/icons-react";
import {
  ClassificationSource,
  EnforcementAction,
  EnforcementSource,
  type ApplicationUsagePayload,
} from "../../../bindings/github.com/focusd-so/focusd/internal/usage/models";
import { pickEnforced } from "@/lib/timeline";
import { TruncatedLabel } from "@/components/usage-item/truncated-label";
import {
  formatClassificationSource,
  formatEnforcementSource,
} from "@/components/usage-item/formatters";

export function ClassificationReasoningLabel({
  payload,
  sourceMeta,
  isAllowedDistraction,
  isPausedDistraction,
  isCurrentlyPaused,
  onResume,
  enforcementSourceMeta,
}: {
  payload: ApplicationUsagePayload | null;
  sourceMeta: ReturnType<typeof formatClassificationSource>;
  isAllowedDistraction: boolean;
  isPausedDistraction: boolean;
  isCurrentlyPaused: boolean;
  onResume: () => void;
  enforcementSourceMeta: ReturnType<typeof formatEnforcementSource>;
}) {
  if (!payload?.classification_source) return null;

  const enforced = pickEnforced(payload);
  const enforcementAction = enforced?.Action;
  const enforcementSrc = enforced?.Source;
  const enforcementReason = enforced?.Reason;
  const classificationReason = payload.classification_reason;

  const isCustomRulesAllow =
    enforcementAction === EnforcementAction.EnforcementActionAllow &&
    enforcementSrc === EnforcementSource.EnforcementSourceCustomRules;

  const isCustomRulesClassification =
    payload.classification_source ===
      ClassificationSource.ClassificationSourceCustomRules &&
    !!classificationReason;

  const labelText = isPausedDistraction
    ? "was paused by user"
    : isCustomRulesAllow
      ? enforcementReason || "set by custom rules"
      : isCustomRulesClassification
        ? classificationReason
        : isAllowedDistraction
          ? enforcementReason || "user allowed distraction"
          : sourceMeta.description;

  const shouldBeLink =
    isCustomRulesAllow ||
    isCustomRulesClassification ||
    sourceMeta.isLink ||
    enforcementSourceMeta?.isLink;

  const displayText = enforcementSourceMeta?.label
    ? `${enforcementSourceMeta.label}: ${labelText}`
    : labelText;

  if (isPausedDistraction) {
    return (
      <span className="text-[10px] text-yellow-500/70 flex items-center gap-1.5">
        <IconPlayerPause className="w-3 h-3 text-yellow-500/70" />
        <span>{labelText}</span>
        {isCurrentlyPaused && (
          <>
            <span className="text-yellow-500/40">·</span>
            <button
              onClick={(e) => {
                e.stopPropagation();
                onResume();
              }}
              className="text-yellow-500 hover:text-yellow-400 hover:underline transition-colors font-medium"
            >
              resume
            </button>
          </>
        )}
      </span>
    );
  }

  return (
    <span className="text-[10px] text-muted-foreground/50 flex items-center gap-1.5">
      <span className="text-[10px] opacity-70">
        {enforcementSourceMeta?.icon || sourceMeta.icon}
      </span>
      {shouldBeLink ? (
        <TruncatedLabel className="max-w-[170px] sm:max-w-[260px]">
          <Link
            to="/settings"
            search={{ tab: "rules" }}
            className="hover:text-muted-foreground hover:underline transition-colors"
          >
            {displayText}
          </Link>
        </TruncatedLabel>
      ) : (
        <TruncatedLabel className="max-w-[170px] sm:max-w-[260px]">
          {displayText}
        </TruncatedLabel>
      )}
    </span>
  );
}
