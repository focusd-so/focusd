import React, { useState } from "react";
import { Link } from "@tanstack/react-router";
import {
  IconWorld,
  IconAppWindow,
  IconPlayerPause,
  IconTerminal,
  IconChevronDown,
  IconChevronRight,
} from "@tabler/icons-react";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useUsageStore } from "@/stores/usage-store";
import type { ApplicationUsage } from "../../bindings/github.com/focusd-so/focusd/internal/usage/models";
import { EnforcementAction } from "../../bindings/github.com/focusd-so/focusd/internal/usage/models";

export function isDistracting(classification?: string | null): boolean {
  if (!classification) return false;
  const lower = classification.toLowerCase();
  return lower.includes("distract") || lower === "distracting";
}

export function isNeutralOrSystem(classification?: string | null): boolean {
  if (!classification) return false;
  const lower = classification.toLowerCase();
  return lower === "neutral" || lower === "system";
}

export function formatSmartDate(unixSeconds: number | null | undefined): string {
  if (!unixSeconds) return "";
  const date = new Date(unixSeconds * 1000);
  const now = new Date();
  const isToday = date.toDateString() === now.toDateString();

  if (isToday) {
    // Time only: "2:34pm"
    const hours = date.getHours();
    const mins = date.getMinutes();
    const h = hours % 12 || 12;
    const ampm = hours < 12 ? "am" : "pm";
    return `${h}:${mins.toString().padStart(2, "0")}${ampm}`;
  }

  // Short date: "Jan 15"
  return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

function tryParseJSON(str: string | null | undefined): string {
  if (!str) return "—";
  try {
    return JSON.stringify(JSON.parse(str), null, 2);
  } catch {
    return str;
  }
}

function formatSandboxLogs(logsStr: string | null | undefined): string {
  if (!logsStr) return "";
  try {
    const logs = JSON.parse(logsStr);
    if (Array.isArray(logs)) {
      return logs.join("\n");
    }
    return logsStr;
  } catch {
    return logsStr;
  }
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

export function formatClassificationSource(
  source?: string | null,
  classification?: string | null,
  reasoning?: string | null
): {
  label: string;
  icon?: string;
  description: string;
  isLink?: boolean;
} {
  if (reasoning) {
    return {
      label: reasoning,
      description: reasoning,
    };
  }

  switch (source) {
    case "obviously":
      if (classification === "productive") {
        return {
          label: "work",
          icon: "⚡️",
          description: "It's obviously productive",
        };
      }
      return {
        label: "duh",
        icon: "🙄",
        description: "C'mon... it's obviously distracting",
      };
    case "cloud_llm":
      return {
        label: "ai",
        icon: "✨",
        description: "Classified by AI based on context",
      };
    case "custom_rules":
      return {
        label: "custom rules",
        icon: "📋",
        description: "Matched your custom blocking rules",
        isLink: true,
      };
    default:
      return {
        label: source || "",
        icon: "❓",
        description: "Unknown classification source",
      };
  }
}

export function formatEnforcementSource(
  source?: string | null,
  reasoning?: string | null
): {
  label: string;
  icon: string;
  description: string;
  isLink?: boolean;
} | null {
  if (!source || source === "application") return null;

  switch (source) {
    case "custom_rules":
      return {
        label: "custom rules",
        icon: "⚡️",
        description: reasoning || "Action determined by your custom rules",
        isLink: true,
      };
    case "whitelist":
      return {
        label: "allowed",
        icon: "✓",
        description: "Temporarily allowed by you",
      };
    default:
      return {
        label: source,
        icon: "❓",
        description: "Unknown termination source",
      };
  }
}

// Helper function to extract text content from React nodes
function extractTextContent(node: React.ReactNode): string {
  if (typeof node === "string" || typeof node === "number") {
    return String(node);
  }
  if (node === null || node === undefined) {
    return "";
  }
  if (Array.isArray(node)) {
    return node.map(extractTextContent).join("");
  }
  if (typeof node === "object" && "props" in node) {
    const element = node as React.ReactElement<{ children?: React.ReactNode }>;
    return extractTextContent(element.props?.children);
  }
  return "";
}

export function TruncatedLabel({
  children,
  className = "",
}: {
  children: React.ReactNode;
  className?: string;
}) {
  const [isTruncated, setIsTruncated] = useState(false);
  const textRef = React.useRef<HTMLSpanElement | null>(null);

  const textContent = extractTextContent(children);

  React.useEffect(() => {
    const checkTruncation = () => {
      if (textRef.current) {
        setIsTruncated(
          textRef.current.scrollWidth > textRef.current.clientWidth
        );
      }
    };

    checkTruncation();
    const timeoutId = setTimeout(checkTruncation, 0);
    window.addEventListener("resize", checkTruncation);

    return () => {
      clearTimeout(timeoutId);
      window.removeEventListener("resize", checkTruncation);
    };
  }, [children]);

  const content = (
    <span
      ref={textRef}
      className={`truncate inline-block align-middle ${className}`}
    >
      {children}
    </span>
  );

  if (isTruncated) {
    return (
      <TooltipProvider>
        <Tooltip delayDuration={300}>
          <TooltipTrigger asChild>{content}</TooltipTrigger>
          <TooltipContent side="top" className="max-w-[400px] break-words">
            <p>{textContent}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  return content;
}

export function ClassificationReasoningLabel({
  usage,
  icon,
  description,
  isLink,
  isAllowedDistraction,
  isPausedDistraction,
  isCurrentlyPaused,
  onResume,
  enforcementSource,
}: {
  usage: ApplicationUsage;
  icon?: string;
  description: string;
  isLink?: boolean;
  isAllowedDistraction: boolean;
  isPausedDistraction?: boolean;
  isCurrentlyPaused?: boolean;
  onResume?: () => void;
  enforcementSource?: {
    label: string;
    icon: string;
    description: string;
    isLink?: boolean;
  } | null;
}) {
  if (!usage.classification_source) return null;

  const isCustomRulesAllow =
    usage.enforcement_action === EnforcementAction.EnforcementActionAllow &&
    usage.enforcement_source === "custom_rules";

  const isCustomRulesClassification =
    usage.classification_reasoning && usage.enforcement_source === "custom_rules";

  // Determine the label text based on priority
  const getLabelText = (): string => {
    if (isPausedDistraction) {
      return "was paused by user";
    }
    if (isCustomRulesAllow) {
      return usage.enforcement_reason || "set by custom rules";
    }
    if (isCustomRulesClassification) {
      return usage.classification_reasoning!;
    }
    if (isAllowedDistraction) {
      return usage.enforcement_reason || "user allowed distraction";
    }
    return description;
  };

  // Determine if it should be a link
  const shouldBeLink = isCustomRulesAllow || isCustomRulesClassification || isLink;

  const labelText = getLabelText();
  const prefixLabel = enforcementSource?.label;
  const displayText = prefixLabel ? `${prefixLabel}: ${labelText}` : labelText;
  const displayIcon = isPausedDistraction ? <IconPlayerPause className="w-3 h-3 text-yellow-500/70" /> : <span className="text-[10px] opacity-70">{enforcementSource?.icon || icon}</span>;

  // Handle paused distraction display with optional resume button
  if (isPausedDistraction) {
    return (
      <span className="text-[10px] text-yellow-500/70 flex items-center gap-1.5">
        {displayIcon}
        <span>{labelText}</span>
        {isCurrentlyPaused && (
          <>
            <span className="text-yellow-500/40">·</span>
            <button
              onClick={(e) => {
                e.stopPropagation();
                onResume?.();
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
      {displayIcon}
      {shouldBeLink || enforcementSource?.isLink ? (
        <TruncatedLabel className="max-w-[300px]">
          <Link
            to="/settings"
            search={{ tab: "rules" }}
            className="hover:text-muted-foreground hover:underline transition-colors"
          >
            {displayText}
          </Link>
        </TruncatedLabel>
      ) : (
        <TruncatedLabel className="max-w-[300px]">{displayText}</TruncatedLabel>
      )}
    </span>
  );
}

function getThemeClasses(
  isYellowTheme: boolean,
  isDistractingEvent: boolean,
  isGrayTheme: boolean
) {
  if (isYellowTheme) {
    return {
      container: "bg-yellow-500/5 border-yellow-500/20 text-yellow-400 hover:bg-yellow-500/10",
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
      container: "bg-zinc-500/5 border-zinc-500/20 text-zinc-400 hover:bg-zinc-500/10",
      iconBg: "bg-zinc-500/10",
      badge: "border border-zinc-500/30 text-zinc-400",
    };
  }
  // Default: productive green
  return {
    container: "bg-green-500/5 border-green-500/20 text-green-400 hover:bg-green-500/10",
    iconBg: "bg-green-500/10",
    badge: "border border-green-500/30 text-green-400",
  };
}

export function UsageItem({ usage }: { usage: ApplicationUsage }) {
  const isWeb = !!usage.application?.hostname;
  const isDistractingEvent = isDistracting(usage.classification);
  const isGrayTheme = isNeutralOrSystem(usage.classification);
  const resumeProtection = useUsageStore((state) => state.resumeProtection);
  const currentPause = useUsageStore((state) => state.currentPause);
  const [showLogs, setShowLogs] = useState(false);

  // Check if there's currently an active pause
  const isCurrentlyPaused = !!(currentPause && currentPause.id > 0);

  // Check if this is a paused distraction
  const isPausedDistraction =
    isDistractingEvent &&
    usage.enforcement_action === EnforcementAction.EnforcementActionPaused;

  const isAllowedDistraction =
    isDistractingEvent &&
    usage.enforcement_action === EnforcementAction.EnforcementActionAllow &&
    (usage.enforcement_source === "custom_rules" || usage.enforcement_source === "whitelist");

  // Combined flag for yellow styling
  const isYellowTheme = isPausedDistraction || isAllowedDistraction;

  const theme = getThemeClasses(isYellowTheme, isDistractingEvent, isGrayTheme);

  const { description, icon, isLink } = formatClassificationSource(
    usage.classification_source,
    usage.classification,
    usage.classification_reasoning
  );

  const termSource = formatEnforcementSource(
    usage.enforcement_source,
    usage.classification_reasoning
  );

  // Duration display: show duration for ended items, or elapsed time for ongoing items
  const durationSeconds =
    usage.ended_at && usage.started_at
      ? usage.ended_at - usage.started_at
      : null;

  // Detect script execution that was ignored (Free tier)
  let sandboxDecision: any = null;
  try {
    if (usage.sandbox_response && usage.sandbox_response !== "no response") {
      sandboxDecision = JSON.parse(usage.sandbox_response);
    }
  } catch (e) {
    // ignore
  }

  const isIgnoredRule =
    !!sandboxDecision &&
    usage.classification_source !== "custom_rules" &&
    usage.enforcement_source !== "custom_rules";

  return (
    <div
      className={`flex flex-col p-1.5 rounded-lg border transition-all ${theme.container}`}
    >
      <div className="flex items-center justify-between w-full">
        <div className="flex items-center gap-2 truncate">
          {/* Icon Container */}
          <div
            className={`w-8 h-8 rounded-md flex items-center justify-center overflow-hidden shrink-0 ${theme.iconBg}`}
          >
            {usage.application?.icon ? (
              <img
                src={
                  usage.application.icon.startsWith("data:")
                    ? usage.application.icon
                    : `data:image/png;base64,${usage.application.icon}`
                }
                alt={usage.application?.hostname || usage.application?.name}
                className="w-8 h-8 object-contain"
              />
            ) : isWeb ? (
              <IconWorld className="w-8 h-8" />
            ) : (
              <IconAppWindow className="w-8 h-8" />
            )}
          </div>

          {/* Text Content */}
          <div className="flex flex-col truncate">
            <TruncatedLabel className="text-xs font-semibold text-foreground truncate leading-tight">
              {usage.application?.hostname || usage.application?.name || "Unknown"}
            </TruncatedLabel>
            <div className="flex items-center gap-1.5 mt-0.5">
              <span className="text-[10px] font-medium uppercase tracking-widest opacity-70">
                {usage.classification ||
                  (isDistractingEvent ? "Distracting" : "Productive")}
              </span>
              {usage.classification_source == "custom_rules" && (
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
              <TruncatedLabel className="text-[10px] text-muted-foreground truncate max-w-[250px]">
                {usage.window_title || (isWeb ? "Browsing" : "Using app")}
              </TruncatedLabel>
            </div>
          </div>
        </div>

        {/* Right Side Group */}
        <div className="flex items-center gap-2 shrink-0">
          <div className="flex flex-col items-end gap-1">
            <div className="flex items-center gap-1.5">
              {durationSeconds != null && durationSeconds >= 0 && (
                <div className="flex items-center gap-1.5">
                  <span className="text-xs font-semibold text-foreground/90 tabular-nums">
                    {formatDuration(durationSeconds)}
                  </span>
                </div>
              )}

              <Badge
                variant="outline"
                className={`px-1.5 py-0 text-[9px] font-bold rounded-full ${theme.badge}`}
              >
                {isWeb ? "web" : "app"}
              </Badge>

              {usage.tags?.map((usageTag) => (
                <Badge
                  key={usageTag.tag}
                  variant="outline"
                  className={`px-1.5 py-0 text-[9px] font-bold rounded-full ${theme.badge}`}
                >
                  {usageTag.tag}
                </Badge>
              ))}
            </div>

            <span className="text-[10px] text-muted-foreground/50 tabular-nums leading-none">
              at {formatSmartDate(usage.started_at)}
            </span>

            <ClassificationReasoningLabel
              usage={usage}
              icon={icon}
              description={description}
              isLink={isLink}
              isAllowedDistraction={isAllowedDistraction}
              isPausedDistraction={isPausedDistraction}
              isCurrentlyPaused={isCurrentlyPaused}
              onResume={resumeProtection}
              enforcementSource={
                termSource?.label === "custom rules" ? termSource : null
              }
            />
            {isIgnoredRule && (
              <div className="text-[9px] text-purple-400/80 bg-purple-500/5 px-1.5 py-0.5 rounded border border-purple-500/10 mt-1 flex items-center gap-1 animate-in fade-in slide-in-from-right-1 duration-500">
                <IconTerminal className="w-2.5 h-2.5 shrink-0" />
                <span className="truncate">
                  Script would have{" "}
                  <span className="font-semibold uppercase text-purple-400">
                    {sandboxDecision.enforcementAction && sandboxDecision.enforcementAction !== "none"
                      ? sandboxDecision.enforcementAction
                      : sandboxDecision.classification}
                  </span>{" "}
                  this.{" "}
                  <Link
                    to="/settings"
                    search={{ tab: "rules" }}
                    className="underline hover:text-purple-300 font-medium"
                    onClick={(e) => e.stopPropagation()}
                  >
                    Upgrade to Pro
                  </Link>
                </span>
              </div>
            )}
          </div>

          {/* Sandbox Logs Toggle */}
          {(usage.sandbox_context || usage.sandbox_response || usage.sandbox_logs) && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                setShowLogs(!showLogs);
              }}
              className={`p-1 rounded-md border transition-all flex items-center justify-center ${showLogs
                ? "bg-muted/40 border-border/60 text-foreground"
                : "bg-muted/10 border-border/30 text-muted-foreground/60 hover:bg-muted/30 hover:border-border/60 hover:text-foreground"
                }`}
              title="Show sandbox execution logs"
            >
              {showLogs ? (
                <IconChevronDown className="w-4 h-4" />
              ) : (
                <IconChevronRight className="w-4 h-4" />
              )}
            </button>
          )}
        </div>
      </div>

      {/* Expanded Logs */}
      {showLogs && (
        <div className="w-full mt-2 pt-2 border-t border-border/20 space-y-3">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div>
              <span className="text-[9px] font-bold uppercase tracking-wider text-muted-foreground/40 mb-1 block">
                Context
              </span>
              <pre className="text-[10px] text-muted-foreground/70 bg-background/30 rounded p-1.5 overflow-x-auto max-h-[150px] overflow-y-auto font-mono whitespace-pre-wrap break-all border border-border/10">
                {tryParseJSON(usage.sandbox_context)}
              </pre>
            </div>
            <div>
              <span className="text-[9px] font-bold uppercase tracking-wider text-muted-foreground/40 mb-1 block">
                Response
              </span>
              <pre className="text-[10px] text-green-400/60 bg-background/30 rounded p-1.5 overflow-x-auto max-h-[150px] overflow-y-auto font-mono whitespace-pre-wrap break-all border border-border/10">
                {tryParseJSON(usage.sandbox_response)}
              </pre>
            </div>
          </div>

          {usage.sandbox_logs && usage.sandbox_logs !== "null" && (
            <div>
              <span className="text-[9px] font-bold uppercase tracking-wider text-muted-foreground/40 mb-1 flex items-center gap-1">
                <IconTerminal className="w-3 h-3" />
                Console Logs
              </span>
              <pre className="text-[10px] text-yellow-400/60 bg-background/30 rounded p-1.5 overflow-x-auto max-h-[150px] overflow-y-auto font-mono whitespace-pre-wrap break-all border border-border/10">
                {formatSandboxLogs(usage.sandbox_logs)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
