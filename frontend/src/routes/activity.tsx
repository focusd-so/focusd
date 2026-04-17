import React, { useState, useMemo } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { Browser } from "@wailsio/runtime";
import {
  IconWorld,
  IconAppWindow,
  IconSparkles,
  IconShield,
  IconChevronDown,
  IconClock,
  IconCalendar,
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useSettingsStore } from "@/stores/settings-store";
import { useAccountStore } from "@/stores/account-store";
import { SmartBlockingStatus } from "@/components/smart-blocking-status";
import {
  UsageItem,
  TruncatedLabel,
  formatSmartDate,
  formatClassificationSource,
  formatEnforcementSource,
} from "@/components/usage-item";
import { AllowCustomDialog } from "@/components/allow-custom-dialog";
import { hasNonDefaultCustomRules } from "@/lib/rules/default-rules";
import {
  useApplicationList,
  useBlockedItems,
  useRecentUsages,
  type BlockedItemView,
} from "@/hooks/queries/use-usage";
import {
  useAllowApp,
  useAllowHostname,
  useAllowList,
  useAllowURL,
  useRemoveAllow,
  type AllowedItem,
} from "@/hooks/queries/use-allow";
import {
  buildApplicationsById,
  toUsageItemView,
  type UsageItemView,
} from "@/lib/usage-view";
import { parsePayload, type ApplicationUsagePayload } from "@/lib/timeline";
import { DeviceHandshakeResponse_AccountTier } from "../../bindings/github.com/focusd-so/focusd/gen/api/v1/models";
import { usePageSearch } from "@/hooks/use-page-search";

interface BlockedUsageDisplay {
  view: UsageItemView;
  count: number;
  isAllowed: boolean;
  expiresAt: number | null;
  allowId: number | null;
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

  const cursorClass = shouldBeLink ? "cursor-pointer hover:opacity-80" : "cursor-help";

  const badgeContent = (
    <span
      className={[
        "inline-flex items-center gap-0.5 py-0 text-[9px] font-medium rounded border",
        cursorClass,
        colorClasses[variant],
        className,
      ].join(" ")}
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

function matchesAllowed(view: UsageItemView, allowed: AllowedItem): boolean {
  if (allowed.hostname && view.application?.hostname) {
    return allowed.hostname === view.application.hostname;
  }
  if (allowed.url && view.browser_url) {
    return allowed.url === view.browser_url;
  }
  if (allowed.app_name && view.application?.name) {
    return allowed.app_name === view.application.name;
  }
  return false;
}

function ActivityPage() {
  const { data: recentEvents = [] } = useRecentUsages();
  const blockedRaw = useBlockedItems();
  const { items: allowedItems } = useAllowList();
  const { data: applications } = useApplicationList();
  const applicationsById = useMemo(
    () => buildApplicationsById(applications),
    [applications],
  );

  const customRules = useSettingsStore((state) => state.customRules);
  const { checkoutLink, fetchAccountTier } = useAccountStore();
  const { data: accountTier } = useQuery({
    queryKey: ["accountTier"],
    queryFn: () => fetchAccountTier(),
  });
  const isFreeTier =
    accountTier ===
    DeviceHandshakeResponse_AccountTier.DeviceHandshakeResponse_ACCOUNT_TIER_FREE;
  const showNonEnforcedCustomRulesNote =
    isFreeTier && hasNonDefaultCustomRules(customRules);
  const { query: searchQuery } = usePageSearch({
    enabled: true,
    placeholder: "Search blocked + recent...",
  });

  const [renderCount, setRenderCount] = useState(15);

  React.useEffect(() => {
    if (recentEvents.length > 15) {
      const timer = setTimeout(() => setRenderCount(100), 50);
      return () => clearTimeout(timer);
    }
  }, [recentEvents.length]);

  const activeUsages: UsageItemView[] = useMemo(
    () =>
      recentEvents.map((event) => {
        const payload = parsePayload<ApplicationUsagePayload>(event);
        const app = payload?.application_id
          ? applicationsById.get(payload.application_id)
          : undefined;
        return toUsageItemView(event, app);
      }),
    [recentEvents, applicationsById],
  );

  const blockedItemViews = useMemo<BlockedUsageDisplay[]>(() => {
    return blockedRaw.map(({ event, count }: BlockedItemView) => {
      const payload = parsePayload<ApplicationUsagePayload>(event);
      const app = payload?.application_id
        ? applicationsById.get(payload.application_id)
        : undefined;
      const view = toUsageItemView(event, app);
      const allowMatch = allowedItems.find((a) => matchesAllowed(view, a));
      return {
        view,
        count,
        isAllowed: !!allowMatch,
        expiresAt: allowMatch?.expires_at ?? null,
        allowId: allowMatch?.id ?? null,
      };
    });
  }, [blockedRaw, allowedItems, applicationsById]);

  const allowedOnlyDisplay = useMemo<BlockedUsageDisplay[]>(() => {
    const seen = new Set<string>();
    for (const item of blockedItemViews) {
      seen.add(`${item.view.application?.hostname ?? ""}|${item.view.application?.name ?? ""}`);
    }
    const out: BlockedUsageDisplay[] = [];
    for (const allowed of allowedItems) {
      const key = `${allowed.hostname ?? ""}|${allowed.app_name ?? ""}`;
      if (seen.has(key)) continue;

      const matchView = activeUsages.find((u) =>
        matchesAllowed(u, allowed),
      );
      if (!matchView) continue;

      out.push({
        view: matchView,
        count: 0,
        isAllowed: true,
        expiresAt: allowed.expires_at,
        allowId: allowed.id,
      });
    }
    return out;
  }, [allowedItems, activeUsages, blockedItemViews]);

  const blockedUsagesDisplay = useMemo(
    () =>
      [...blockedItemViews, ...allowedOnlyDisplay].sort(
        (a, b) => (b.view.started_at ?? 0) - (a.view.started_at ?? 0),
      ),
    [blockedItemViews, allowedOnlyDisplay],
  );

  const filteredBlockedUsages = useMemo(() => {
    if (!searchQuery) return blockedUsagesDisplay;
    const q = searchQuery.toLowerCase();
    return blockedUsagesDisplay.filter(({ view }) => {
      const name = view.application?.name?.toLowerCase() || "";
      const host = view.application?.hostname?.toLowerCase() || "";
      const title = view.window_title?.toLowerCase() || "";
      const tags = view.tags.join(" ").toLowerCase();
      return name.includes(q) || host.includes(q) || title.includes(q) || tags.includes(q);
    });
  }, [blockedUsagesDisplay, searchQuery]);

  const filteredActiveUsages = useMemo(() => {
    if (!searchQuery) return activeUsages;
    const q = searchQuery.toLowerCase();
    return activeUsages.filter((view) => {
      const name = view.application?.name?.toLowerCase() || "";
      const host = view.application?.hostname?.toLowerCase() || "";
      const title = view.window_title?.toLowerCase() || "";
      const tags = view.tags.join(" ").toLowerCase();
      return name.includes(q) || host.includes(q) || title.includes(q) || tags.includes(q);
    });
  }, [activeUsages, searchQuery]);

  return (
    <div className="flex flex-col gap-6 p-4 flex-1 min-h-0 overflow-hidden">
      <div className="flex flex-col gap-4 shrink-0">
        <SmartBlockingStatus />
      </div>

      {blockedUsagesDisplay.length > 0 && (
        <div className="flex flex-col gap-2 min-h-0 max-h-[40%]">
          <div className="flex items-center justify-between shrink-0 gap-4">
            <p className="text-xs font-bold text-red-500/80 uppercase tracking-widest flex items-center gap-2 whitespace-nowrap">
              <IconShield className="w-3 h-3" />
              Blocked Distractions Today
            </p>
            <div className="flex items-center gap-3">
              <Badge
                variant="outline"
                className="border-red-500/20 text-red-500/60 text-[9px] px-1.5 h-4 shrink-0 transition-all"
              >
                {filteredBlockedUsages.filter((b) => !b.isAllowed).length} PREVENTED
              </Badge>
            </div>
          </div>
          <ScrollArea className="flex-1 min-h-0 [&_[data-radix-scroll-area-scrollbar]]:hidden">
            <div className="flex flex-col gap-2">
              {filteredBlockedUsages.length === 0 ? (
                <div className="text-xs text-center text-white/30 py-4 italic">No matches found</div>
              ) : (
                filteredBlockedUsages.map((item) => (
                  <BlockedUsageItem
                    key={item.view.application?.hostname || item.view.application?.name || String(item.view.id)}
                    item={item}
                  />
                ))
              )}
            </div>
          </ScrollArea>
        </div>
      )}

      <div className="flex flex-col gap-3 flex-1 min-h-0">
        <div className="flex items-center justify-between shrink-0">
          <p className="text-xs font-medium text-white/40 uppercase tracking-wider">
            Recent Activity
          </p>
          {showNonEnforcedCustomRulesNote && (
            <div className="flex items-center gap-2 rounded-md border border-amber-500/20 bg-amber-500/10 px-2 py-1 text-[10px] text-amber-200/90">
              <span className="truncate">Custom rules are preview-only on Free.</span>
              {checkoutLink && (
                <button
                  onClick={() => Browser.OpenURL(checkoutLink)}
                  className="whitespace-nowrap font-semibold underline hover:text-amber-100"
                >
                  Upgrade
                </button>
              )}
            </div>
          )}
        </div>

        <ScrollArea className="flex-1 min-h-0 [&_[data-radix-scroll-area-scrollbar]]:hidden">
          <div className="flex flex-col gap-3">
            {activeUsages.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground border border-dashed rounded-xl opacity-50">
                <IconSparkles className="w-8 h-8 mb-2" />
                <p>No activity recorded yet</p>
              </div>
            ) : filteredActiveUsages.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground border border-dashed rounded-xl opacity-50">
                <IconSparkles className="w-8 h-8 mb-2" />
                <p>No recent activity matches your search</p>
              </div>
            ) : (
              <div className="space-y-1.5">
                {filteredActiveUsages.slice(0, renderCount).map((view) => (
                  <UsageItem key={view.id} usage={view} />
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
  const { view, count, isAllowed, expiresAt, allowId } = item;
  const isWeb = !!view.application?.hostname;
  const [timeLeft, setTimeLeft] = useState<number | null>(null);
  const [isCustomDialogOpen, setIsCustomDialogOpen] = useState(false);
  const allowAppMutation = useAllowApp();
  const allowHostnameMutation = useAllowHostname();
  const allowURLMutation = useAllowURL();
  const removeAllowMutation = useRemoveAllow();

  const isRecentlyBlocked = React.useMemo(() => {
    if (!view.started_at || isAllowed) return false;
    const now = Math.floor(Date.now() / 1000);
    return (now - view.started_at) < 300;
  }, [view.started_at, isAllowed]);

  React.useEffect(() => {
    if (!isAllowed || !expiresAt) {
      setTimeLeft(null);
      return;
    }

    const updateTimer = () => {
      const remaining = expiresAt - Math.floor(Date.now() / 1000);
      if (remaining <= 0) {
        setTimeLeft(null);
        if (allowId) removeAllowMutation.mutate(allowId);
      } else {
        setTimeLeft(remaining);
      }
    };

    updateTimer();
    const interval = setInterval(updateTimer, 1000);
    return () => clearInterval(interval);
  }, [isAllowed, expiresAt, allowId, removeAllowMutation]);

  const formatTime = (seconds: number) => {
    const hours = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;

    if (hours > 0) {
      return `${hours}:${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
    }
    return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
  };

  const handleAllowSite = async (durationMinutes: number) => {
    if (isWeb && view.browser_url) {
      await allowHostnameMutation.mutateAsync({
        rawURL: view.browser_url,
        durationMinutes,
      });
    } else if (view.application?.name) {
      await allowAppMutation.mutateAsync({
        appName: view.application.name,
        durationMinutes,
      });
    }
  };

  const handleAllowExactURL = async (durationMinutes: number) => {
    if (!view.browser_url) return;
    await allowURLMutation.mutateAsync({
      rawURL: view.browser_url,
      durationMinutes,
    });
  };

  const handleExtendDuration = async (addedMinutes: number) => {
    if (!expiresAt) return;
    const now = Math.floor(Date.now() / 1000);
    const currentRemaining = Math.max(0, expiresAt - now);
    const totalMinutes = Math.floor(currentRemaining / 60) + addedMinutes;
    await handleAllowSite(totalMinutes);
  };

  const handleUnallow = async () => {
    if (allowId) await removeAllowMutation.mutateAsync(allowId);
  };

  const borderColor = isAllowed ? "border-yellow-500/20" : isRecentlyBlocked ? "border-red-500/40 shadow-[0_0_15px_rgba(239,68,68,0.1)]" : "border-red-500/20";
  const bgColor = isAllowed
    ? "bg-yellow-500/5 hover:bg-yellow-500/10"
    : isRecentlyBlocked ? "bg-red-500/10 hover:bg-red-500/15" : "bg-red-500/5 hover:bg-red-500/10";
  const iconBgColor = isAllowed
    ? "bg-yellow-500/10 ring-yellow-500/20 group-hover:ring-yellow-500/30"
    : isRecentlyBlocked ? "bg-red-500/20 ring-red-500/40" : "bg-red-500/10 ring-red-500/20 group-hover:ring-red-500/30";
  const textColor = isAllowed
    ? "group-hover:text-yellow-500"
    : "group-hover:text-red-500";
  const statusColor = isAllowed ? "text-yellow-500" : "text-red-500";
  const iconColor = isAllowed ? "text-yellow-500/60" : "text-red-500/60";

  const termSource = formatEnforcementSource(
    view.enforcement_source,
    view.classification_reason
  );

  const allowSiteLabel = isWeb ? "this site" : "this app";

  return (
    <div
      className={`flex items-center gap-3 px-3 py-2.5 rounded-xl border transition-all group ${borderColor} ${bgColor}`}
    >
      <div
        className={`relative w-10 h-10 rounded-lg flex items-center justify-center overflow-hidden shrink-0 ring-1 ${iconBgColor} transition-all`}
      >
        {view.application?.icon ? (
          <img
            src={
              view.application.icon.startsWith("data:")
                ? view.application.icon
                : `data:image/png;base64,${view.application.icon}`
            }
            alt={view.application?.hostname || view.application?.name}
            className={`w-8 h-8 object-contain ${isAllowed ? "" : "grayscale opacity-60 group-hover:grayscale-0 group-hover:opacity-100"} transition-all`}
          />
        ) : isWeb ? (
          <IconWorld className={`w-8 h-8 ${iconColor}`} />
        ) : (
          <IconAppWindow className={`w-8 h-8 ${iconColor}`} />
        )}
      </div>

      <div className="flex flex-col min-w-0 flex-1 justify-center gap-0.5">
        <div className="flex items-center gap-2">
          <TruncatedLabel
            className={`text-xs font-semibold truncate ${textColor} transition-colors`}
          >
            {view.application?.hostname || view.application?.name || "Unknown"}
          </TruncatedLabel>
          <span className={`text-[9px] font-bold ${statusColor} uppercase tracking-wider opacity-90 shrink-0`}>
            {isAllowed ? "ALLOWED" : "BLOCKED"}
          </span>
        </div>

        <div className="flex items-start gap-1.5 overflow-hidden text-left">
          {termSource && !isAllowed && (
            <span className="text-[9px] text-white/40 flex items-center gap-0.5 shrink-0">
              {termSource.icon && <span className="opacity-70 text-[8px]">{termSource.icon}</span>}

              {termSource.isLink ? (
                <Link
                  to="/settings"
                  search={{ tab: "rules" }}
                  className="hover:text-white/70 hover:underline transition-colors"
                >
                  {termSource.label}
                </Link>
              ) : (
                <span>{termSource.label}</span>
              )}
            </span>
          )}
          {termSource && !isAllowed && view.window_title && (
            <span className="text-white/20 text-[9px] shrink-0">—</span>
          )}
          {view.window_title && (
            <TruncatedLabel className="text-[9px] text-white/35 truncate max-w-[240px]">
              {view.window_title}
            </TruncatedLabel>
          )}
          {!termSource && !view.window_title && (
            <ClassificationSourceBadge
              source={view.classification_source}
              classification={view.classification}
              reasoning={view.classification_reason}
              variant={isAllowed ? "yellow" : "red"}
              isAllowedDistraction={isAllowed}
              className="!bg-transparent !border-transparent px-0 py-0 opacity-75 hover:opacity-100 transition-opacity font-normal"
              maxWidth="max-w-[280px]"
            />
          )}
        </div>

        {(termSource || view.window_title) && view.classification_source && (
          <ClassificationSourceBadge
            source={view.classification_source}
            classification={view.classification}
            reasoning={view.classification_reason}
            variant={isAllowed ? "yellow" : "red"}
            isAllowedDistraction={isAllowed}
            className="!bg-transparent !border-transparent px-0 py-0 opacity-75 hover:opacity-100 transition-opacity font-normal self-start text-left"
            maxWidth="max-w-[280px]"
          />
        )}
      </div>

      <div className="flex items-center gap-1.5 shrink-0">
        <span className="text-[9px] text-muted-foreground/35 font-mono">
          {formatSmartDate(view.started_at)}
        </span>

        {view.tags.map((tag) => (
          <Badge
            key={tag}
            variant="outline"
            className={`px-1 py-0 text-[8px] font-bold rounded-sm border ${isAllowed
              ? "border-yellow-500/30 text-yellow-500/70 bg-yellow-500/5"
              : "border-red-500/30 text-red-500/70 bg-red-500/5"
              }`}
          >
            {tag}
          </Badge>
        ))}

        {!isAllowed && count > 1 && (
          <TooltipProvider>
            <Tooltip delayDuration={300}>
              <TooltipTrigger asChild>
                <Badge
                  variant="outline"
                  className="px-1 py-0 text-[8px] font-bold rounded-sm bg-red-500/10 border border-red-500/30 text-red-400 cursor-help"
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
      </div>

      <div className="shrink-0">
        {!isAllowed && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                className="h-7 px-3 text-[10px] font-bold bg-transparent border-white/10 text-white/50 hover:text-white/80 hover:bg-white/5 hover:border-white/20 transition-all gap-1.5 rounded-lg"
              >
                <IconClock className="w-3 h-3" />
                Allow
                <IconChevronDown className="w-2.5 h-2.5 opacity-60" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="end"
              className="w-44 bg-neutral-950 border-white/10 text-white"
            >
              <DropdownMenuLabel className="text-[9px] font-semibold text-white/30 uppercase tracking-widest px-2 py-1">
                Allow {allowSiteLabel} for…
              </DropdownMenuLabel>
              <DropdownMenuSeparator className="bg-white/8" />
              <DropdownMenuItem
                onClick={() => handleAllowSite(15)}
                className="text-sm text-white/80 hover:text-white focus:text-white focus:bg-white/8 cursor-pointer gap-2 justify-start"
              >
                <IconClock className="w-3.5 h-3.5 opacity-60" />
                15 minutes
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => handleAllowSite(30)}
                className="text-sm text-white/80 hover:text-white focus:text-white focus:bg-white/8 cursor-pointer gap-2 justify-start"
              >
                <IconClock className="w-3.5 h-3.5 opacity-60" />
                30 minutes
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => handleAllowSite(60)}
                className="text-sm text-white/80 hover:text-white focus:text-white focus:bg-white/8 cursor-pointer gap-2 justify-start"
              >
                <IconClock className="w-3.5 h-3.5 opacity-60" />
                1 hour
              </DropdownMenuItem>
              <DropdownMenuSeparator className="bg-white/8" />
              <DropdownMenuItem
                onClick={() => setIsCustomDialogOpen(true)}
                className="text-sm text-white/80 hover:text-white focus:text-white focus:bg-white/8 cursor-pointer gap-2 justify-start"
              >
                <IconCalendar className="w-3.5 h-3.5 opacity-60" />
                Custom time...
              </DropdownMenuItem>
              {isWeb && view.browser_url && (
                <>
                  <DropdownMenuSeparator className="bg-white/8" />
                  <DropdownMenuLabel className="text-[9px] font-semibold text-white/30 uppercase tracking-widest px-2 py-1">
                    Allow this URL only
                  </DropdownMenuLabel>
                  <DropdownMenuItem
                    onClick={() => handleAllowExactURL(15)}
                    className="text-sm text-white/80 hover:text-white focus:text-white focus:bg-white/8 cursor-pointer gap-2 justify-start"
                  >
                    <IconClock className="w-3.5 h-3.5 opacity-60" />
                    15 minutes
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={() => handleAllowExactURL(60)}
                    className="text-sm text-white/80 hover:text-white focus:text-white focus:bg-white/8 cursor-pointer gap-2 justify-start"
                  >
                    <IconClock className="w-3.5 h-3.5 opacity-60" />
                    1 hour
                  </DropdownMenuItem>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        )}

        {isAllowed && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                className="h-7 px-3 text-[10px] font-bold bg-transparent border-yellow-500/20 text-yellow-500/70 hover:text-yellow-400 hover:bg-yellow-500/10 hover:border-yellow-500/40 transition-all gap-1.5 rounded-lg tabular-nums"
              >
                <IconShield className="w-3 h-3" />
                {timeLeft !== null ? `${formatTime(timeLeft)} left` : "Allowed"}
                <IconChevronDown className="w-2.5 h-2.5 opacity-60 ml-0.5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="end"
              className="w-36 bg-neutral-950 border-white/10 text-white"
            >
              <DropdownMenuLabel className="text-[9px] font-semibold text-white/30 uppercase tracking-widest px-2 py-1">
                Extend by…
              </DropdownMenuLabel>
              <DropdownMenuSeparator className="bg-white/8" />
              <DropdownMenuItem
                onClick={() => handleExtendDuration(15)}
                className="text-sm text-white/80 hover:text-white focus:text-white focus:bg-white/8 cursor-pointer gap-2 justify-start"
              >
                <IconClock className="w-3.5 h-3.5 opacity-60" />
                +15 minutes
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => handleExtendDuration(30)}
                className="text-sm text-white/80 hover:text-white focus:text-white focus:bg-white/8 cursor-pointer gap-2 justify-start"
              >
                <IconClock className="w-3.5 h-3.5 opacity-60" />
                +30 minutes
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => handleExtendDuration(60)}
                className="text-sm text-white/80 hover:text-white focus:text-white focus:bg-white/8 cursor-pointer gap-2 justify-start"
              >
                <IconClock className="w-3.5 h-3.5 opacity-60" />
                +1 hour
              </DropdownMenuItem>
              <DropdownMenuSeparator className="bg-white/8" />
              <DropdownMenuItem
                onClick={() => setIsCustomDialogOpen(true)}
                className="text-sm text-white/80 hover:text-white focus:text-white focus:bg-white/8 cursor-pointer gap-2 justify-start"
              >
                <IconCalendar className="w-3.5 h-3.5 opacity-60" />
                Custom time...
              </DropdownMenuItem>
              <DropdownMenuSeparator className="bg-white/8" />
              <DropdownMenuItem
                onClick={handleUnallow}
                className="text-sm text-red-500 hover:text-red-400 focus:text-red-400 focus:bg-red-500/10 cursor-pointer gap-2 justify-start font-medium"
              >
                <IconShield className="w-3.5 h-3.5 opacity-80" />
                Block now
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      <AllowCustomDialog
        open={isCustomDialogOpen}
        onOpenChange={setIsCustomDialogOpen}
        onConfirm={handleAllowSite}
        appName={view.application?.hostname || view.application?.name || "App"}
        defaultDate={expiresAt ? new Date(expiresAt * 1000) : undefined}
      />
    </div>
  );
}
