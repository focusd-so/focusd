import React, { useState, useMemo } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";
import {
  IconWorld,
  IconAppWindow,
  IconSparkles,
  IconShield,
} from "@tabler/icons-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useUsageStore } from "@/stores/usage-store";
import { SmartBlockingStatus } from "@/components/smart-blocking-status";
import { AllowConfirmationDialog } from "@/components/allow-confirmation-dialog";
import {
  UsageItem,
  TruncatedLabel,
  formatSmartDate,
  formatClassificationSource,
  formatTerminationModeSource,
} from "@/components/usage-item";
import type { ApplicationUsage } from "../../bindings/github.com/focusd-so/focusd/internal/usage/models";

// Extended blocked item for UI display with allowed status
interface BlockedUsageDisplay {
  usage: ApplicationUsage;
  count: number;
  isAllowed: boolean;
  expiresAt: number | null;
  whitelistId: number | null;
}

export const Route = createFileRoute("/activity")({
  component: ActivityPage,
});


function ClassificationSourceBadge({
  source,
  classification,
  reasoning,
  variant = "default",
  isAllowedDistraction,
  className = "",
  maxWidth = "max-w-[300px]",
}: {
  source?: string | null;
  classification?: string | null;
  variant?: "default" | "red" | "yellow" | "green";
  reasoning?: string | null;
  isAllowedDistraction?: boolean;
  className?: string;
  maxWidth?: string;
}) {
  if (!source) return null;

  const { label, icon, isLink } = formatClassificationSource(
    source,
    classification,
    reasoning
  );

  const shouldBeLink = isLink || isAllowedDistraction;

  const colorClasses = {
    default: "bg-white/5 text-white/60 border-white/10",
    red: "bg-red-500/10 text-red-400 border-red-500/20",
    yellow: "bg-yellow-500/10 text-yellow-400 border-yellow-500/20",
    green: "bg-green-500/10 text-green-400 border-green-500/20",
  };

  const badgeContent = (
    <span
      className={`inline-flex items-center gap-0.5 px-1.5 py-0 text-[9px] font-medium rounded border ${shouldBeLink ? "cursor-pointer hover:opacity-80" : "cursor-help"
        } ${colorClasses[variant]} ${className}`}
    >
      <span className="text-[8px]">{icon}</span>
      <TruncatedLabel className={maxWidth}>{label}</TruncatedLabel>
    </span>
  );

  if (shouldBeLink) {
    return (
      <Link
        to="/settings"
        search={{ tab: "rules" }}
        className="no-underline inline-flex items-center"
      >
        {badgeContent}
      </Link>
    );
  }

  return badgeContent;
}


function ActivityPage() {
  const { recentUsages, getBlockedItemsList, allowedItems } = useUsageStore();

  // Get active usages
  const activeUsages = useMemo(
    () => recentUsages,
    [recentUsages]
  );

  // Combine blocked items with allowed items for display
  const blockedUsagesDisplay = useMemo(() => {
    const blockedItems = getBlockedItemsList();
    const result: BlockedUsageDisplay[] = [];

    // Process blocked items
    blockedItems.forEach((item) => {
      // Check if this item is in the whitelist
      // For web content (has hostname): match by hostname only
      // For native apps (no hostname): match by bundle_id only
      const whitelistEntry = allowedItems.find((w) => {
        const itemHostname = item.usage.application?.hostname;
        const itemExePath = item.usage.application?.executable_path;

        if (w.hostname) {
          // Whitelist entry is for a website - match by hostname only
          return w.hostname === itemHostname;
        } else if (w.executable_path) {
          // Whitelist entry is for a native app - match by executable_path only
          return w.executable_path === itemExePath;
        }
        return false;
      });

      result.push({
        usage: item.usage,
        count: item.count,
        isAllowed: !!whitelistEntry,
        expiresAt: whitelistEntry?.expires_at || null,
        whitelistId: whitelistEntry?.id || null,
      });
    });

    // Add allowed items that aren't in blocked list
    allowedItems.forEach((allowed) => {
      // Use same matching logic as above
      const alreadyInList = result.some((r) => {
        if (allowed.hostname) {
          return r.usage.application?.hostname === allowed.hostname;
        } else if (allowed.executable_path) {
          return r.usage.application?.executable_path === allowed.executable_path;
        }
        return false;
      });

      if (!alreadyInList) {
        // Find a recent usage for this allowed item to get display info
        const recentUsage = recentUsages.find((u) => {
          if (allowed.hostname) {
            return u.application?.hostname === allowed.hostname;
          } else if (allowed.executable_path) {
            return u.application?.executable_path === allowed.executable_path;
          }
          return false;
        });

        if (recentUsage) {
          result.push({
            usage: recentUsage,
            count: 0,
            isAllowed: true,
            expiresAt: allowed.expires_at,
            whitelistId: allowed.id,
          });
        }
      }
    });

    // Sort by recency (latest first)
    return result.sort((a, b) => (b.usage.started_at ?? 0) - (a.usage.started_at ?? 0));
  }, [getBlockedItemsList, allowedItems, recentUsages]);

  return (
    <div className="flex flex-col gap-6 p-4 flex-1 min-h-0 overflow-hidden">
      <div className="flex flex-col gap-4 shrink-0">
        <SmartBlockingStatus />
      </div>

      {blockedUsagesDisplay.length > 0 && (
        <div className="flex flex-col gap-2 min-h-0 max-h-[40%]">
          <div className="flex items-center justify-between shrink-0">
            <p className="text-xs font-bold text-red-500/80 uppercase tracking-widest flex items-center gap-2">
              <IconShield className="w-3 h-3" />
              Blocked Distractions Today
            </p>
            <Badge
              variant="outline"
              className="border-red-500/20 text-red-500/60 text-[9px] px-1.5 h-4"
            >
              {blockedUsagesDisplay.filter((b) => !b.isAllowed).length} PREVENTED
            </Badge>
          </div>
          <ScrollArea className="flex-1 min-h-0 [&_[data-radix-scroll-area-scrollbar]]:hidden">
            <div className="flex flex-col gap-2">
              {blockedUsagesDisplay.map((item) => (
                <BlockedUsageItem
                  key={item.usage.application?.hostname || item.usage.application?.bundle_id || ""}
                  item={item}
                />
              ))}
            </div>
          </ScrollArea>
        </div>
      )}

      <div className="flex flex-col gap-3 flex-1 min-h-0">
        <div className="flex items-center justify-between shrink-0">
          <p className="text-xs font-medium text-white/40 uppercase tracking-wider">
            Recent Activity
          </p>
        </div>

        <ScrollArea className="flex-1 min-h-0 [&_[data-radix-scroll-area-scrollbar]]:hidden">
          <div className="flex flex-col gap-3">
            {activeUsages.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground border border-dashed rounded-xl opacity-50">
                <IconSparkles className="w-8 h-8 mb-2" />
                <p>No activity recorded yet</p>
              </div>
            ) : (
              <div className="space-y-1.5">
                {activeUsages.map((usage) => (
                  <UsageItem key={usage.id} usage={usage} />
                ))}
              </div>
            )}
          </div>
        </ScrollArea>
      </div>
    </div>
  );
}


function BlockedUsageItem({ item }: { item: BlockedUsageDisplay }) {
  const { usage, count, isAllowed, expiresAt, whitelistId } = item;
  const isWeb = !!usage.application?.hostname;
  const [showAllowDialog, setShowAllowDialog] = useState(false);
  const [timeLeft, setTimeLeft] = useState<number | null>(null);
  const { addToWhitelist, removeFromWhitelist } = useUsageStore();

  // Timer effect for countdown
  React.useEffect(() => {
    if (!isAllowed || !expiresAt) {
      setTimeLeft(null);
      return;
    }

    const updateTimer = () => {
      const remaining = expiresAt - Math.floor(Date.now() / 1000);
      if (remaining <= 0) {
        setTimeLeft(null);
        // Remove from whitelist when time expires to return to blocked state
        if (whitelistId) {
          removeFromWhitelist(whitelistId);
        }
      } else {
        setTimeLeft(remaining);
      }
    };

    updateTimer();
    const interval = setInterval(updateTimer, 1000);
    return () => clearInterval(interval);
  }, [isAllowed, expiresAt, whitelistId, removeFromWhitelist]);

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, "0")}`;
  };

  const handleAllowWithDuration = async (durationMinutes: number) => {
    const executablePath = usage.application?.executable_path || "";
    const hostname = usage.application?.hostname || "";
    await addToWhitelist(executablePath, hostname, durationMinutes);
  };

  const handleUnallow = async () => {
    if (whitelistId) {
      await removeFromWhitelist(whitelistId);
    }
  };

  const borderColor = isAllowed ? "border-yellow-500/20" : "border-red-500/20";
  const bgColor = isAllowed
    ? "bg-yellow-500/5 hover:bg-yellow-500/10"
    : "bg-red-500/5 hover:bg-red-500/10";
  const iconBgColor = isAllowed
    ? "bg-yellow-500/10 ring-yellow-500/20 group-hover:ring-yellow-500/30"
    : "bg-red-500/10 ring-red-500/20 group-hover:ring-red-500/30";
  const textColor = isAllowed
    ? "group-hover:text-yellow-500"
    : "group-hover:text-red-500";
  const statusColor = isAllowed ? "text-yellow-500" : "text-red-500";
  const iconColor = isAllowed ? "text-yellow-500/60" : "text-red-500/60";

  const termSource = formatTerminationModeSource(
    usage.termination_mode_source,
    usage.classification_reasoning
  );

  return (
    <div
      className={`flex items-center justify-between p-3 rounded-xl border ${borderColor} ${bgColor} transition-all group`}
    >
      <div className="flex items-center gap-3">
        <div
          className={`relative w-10 h-10 rounded-lg flex items-center justify-center overflow-hidden shrink-0 ${iconBgColor} ring-1 transition-all`}
        >
          {usage.application?.icon ? (
            <img
              src={
                usage.application.icon.startsWith("data:")
                  ? usage.application.icon
                  : `data:image/png;base64,${usage.application.icon}`
              }
              alt={usage.application?.hostname || usage.application?.name}
              className={`w-10 h-10 object-contain ${isAllowed ? "" : "grayscale opacity-70"} group-hover:grayscale-0 group-hover:opacity-100 transition-all`}
            />
          ) : isWeb ? (
            <IconWorld className={`w-6 h-6 ${iconColor}`} />
          ) : (
            <IconAppWindow className={`w-6 h-6 ${iconColor}`} />
          )}
        </div>

        <div className="flex flex-col">
          <TruncatedLabel
            className={`text-sm font-bold text-foreground/90 truncate leading-tight ${textColor} transition-colors`}
          >
            {usage.application?.hostname || usage.application?.name || "Unknown"}
          </TruncatedLabel>
          {usage.window_title && (
            <TruncatedLabel className="text-[10px] text-muted-foreground/60 truncate max-w-[200px]">
              {usage.window_title}
            </TruncatedLabel>
          )}
          {termSource && !isAllowed && (
            <span className="text-[9px] text-muted-foreground/50 flex items-center gap-1 mt-0.5">
              <span>{termSource.icon}</span>
              {termSource.isLink ? (
                <TruncatedLabel className="max-w-[200px]">
                  <Link
                    to="/settings"
                    search={{ tab: "rules" }}
                    className="hover:text-foreground hover:underline transition-colors"
                  >
                    {termSource.label}
                  </Link>
                </TruncatedLabel>
              ) : (
                <TruncatedLabel className="max-w-[200px]">
                  {termSource.label}
                </TruncatedLabel>
              )}
            </span>
          )}
          <div className="flex items-center gap-2 mt-1">
            {isAllowed ? (
              <>
                <div className="flex items-center gap-1.5">
                  <span
                    className={`text-[10px] font-bold ${statusColor} uppercase tracking-wider`}
                  >
                    ALLOWED
                  </span>
                  {timeLeft !== null && (
                    <>
                      <span className="text-yellow-500/40 text-[10px]">•</span>
                      <span className="text-xs text-yellow-500 font-mono font-semibold">
                        {formatTime(timeLeft)} left
                      </span>
                    </>
                  )}
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleUnallow}
                  className="h-6 px-2 text-[10px] font-medium border-yellow-500/20 text-yellow-500/70 hover:bg-yellow-500/10 hover:border-yellow-500/40 hover:text-yellow-500 transition-all gap-1"
                >
                  <IconShield className="w-3 h-3" />
                  Resume Protection
                </Button>
              </>
            ) : (
              <>
                <span
                  className={`text-[10px] font-bold ${statusColor} uppercase tracking-wider`}
                >
                  BLOCKED
                </span>
              </>
            )}
          </div>
        </div>
      </div>

      <div className="flex flex-col items-end gap-1.5">
        <div className="flex items-center gap-2">
          <span className="text-[9px] text-muted-foreground/40 font-mono">
            {formatSmartDate(usage.started_at)}
          </span>
          {usage.tags?.map((usageTag) => (
            <Badge
              key={usageTag.tag}
              variant="outline"
              className={`px-1.5 py-0 text-[9px] font-bold rounded-full border ${isAllowed ? "border-yellow-500/30 text-yellow-400" : "border-red-500/30 text-red-400"}`}
            >
              {usageTag.tag}
            </Badge>
          ))}

          {!isAllowed && count > 1 && (
            <TooltipProvider>
              <Tooltip delayDuration={300}>
                <TooltipTrigger asChild>
                  <Badge
                    variant="outline"
                    className="px-1.5 py-0 text-[9px] font-bold rounded-full bg-red-500/10 border border-red-500/30 text-red-400 cursor-help"
                  >
                    {count}x
                  </Badge>
                </TooltipTrigger>
                <TooltipContent
                  side="bottom"
                  className="text-xs bg-red-950 border-red-500/30 text-red-200"
                >
                  <p>Prevented {count} access attempts</p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          )}

          {!isAllowed && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowAllowDialog(true)}
              className="h-7 px-2.5 text-[10px] font-semibold rounded-lg text-muted-foreground/40 hover:text-yellow-400 border border-transparent hover:border-yellow-500/30 hover:bg-yellow-500/10 transition-all"
            >
              Allow
            </Button>
          )}
        </div>

        <ClassificationSourceBadge
          source={usage.classification_source}
          classification={usage.classification}
          reasoning={usage.classification_reasoning}
          variant={isAllowed ? "yellow" : "red"}
          isAllowedDistraction={isAllowed}
          maxWidth="max-w-[500px]"
        />
      </div>

      <AllowConfirmationDialog
        open={showAllowDialog}
        onOpenChange={setShowAllowDialog}
        appName={usage.application?.hostname || usage.application?.name || "Unknown"}
        appIcon={usage.application?.icon}
        onConfirm={handleAllowWithDuration}
      />
    </div>
  );
}
