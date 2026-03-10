import { useState, useMemo } from "react";
import {
  IconChevronLeft,
  IconChevronRight,
  IconHistory,
} from "@tabler/icons-react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  formatMinutes,
  formatDate,
} from "@/lib/mock-data";
import { useUsageStore, isToday } from "@/stores/usage-store";
import type { ProductivityScore, CommunicationBreakdown } from "@/../bindings/github.com/focusd-so/focusd/internal/usage/models";
import { LLMInsightCard } from "./ai-insight-card";
import { TopBlockedCard } from "./top-blocked-card";
import { TopDistractionsCard } from "./top-distractions-card";
import { CategoriesCard } from "./categories-card";
import { CommunicationCard } from "./communication-card";

const MIN_SECONDS_FOR_INSIGHTS = 3600;

const SERIES = [
  { key: "productive", field: "productive_seconds", label: "Productive", bg: "bg-emerald-500/80", dot: "bg-emerald-500", text: "text-emerald-400" },
  { key: "distractive", field: "distractive_seconds", label: "Distractive", bg: "bg-rose-500/80", dot: "bg-rose-500", text: "text-rose-400" },
  { key: "idle", field: "idle_seconds", label: "Idle", bg: "bg-zinc-400/60", dot: "bg-zinc-400", text: "text-zinc-400" },
  { key: "other", field: "other_seconds", label: "Other", bg: "bg-amber-400/60", dot: "bg-amber-400", text: "text-amber-400" },
] as const;

type SeriesField = (typeof SERIES)[number]["field"];

interface HourlySlot extends Record<SeriesField, number> {
  HourLabel: string;
}

const formatHourLabel = (hour: number): string => {
  const suffix = hour >= 12 ? "pm" : "am";
  const normalizedHour = hour % 12 === 0 ? 12 : hour % 12;
  return `${normalizedHour}${suffix}`;
};

const buildHourlySlots = (breakdown: Record<string, ProductivityScore> | null | undefined): HourlySlot[] => {
  const map = breakdown ?? {};
  return Array.from({ length: 24 }, (_, hour) => {
    const score = map[String(hour)];
    return {
      HourLabel: formatHourLabel(hour),
      productive_seconds: score?.productive_seconds ?? 0,
      distractive_seconds: score?.distractive_seconds ?? 0,
      idle_seconds: score?.idle_seconds ?? 0,
      other_seconds: score?.other_seconds ?? 0,
    };
  });
};

/**
 * Converts the backend communication map to a sorted array.
 * Groups by channel name (A-Z) and then sorts by minutes (descending).
 */
const buildSortedChannels = (
  breakdown: Record<string, CommunicationBreakdown | undefined> | null | undefined
): CommunicationBreakdown[] => {
  return Object.values(breakdown ?? {})
    .filter((c): c is CommunicationBreakdown => c != null)
    .sort((a, b) => a.channel.localeCompare(b.channel) || b.minutes - a.minutes);
};

type SeriesKey = (typeof SERIES)[number]["key"];

function HourlyBreakdownChart({
  hourlyData,
}: {
  hourlyData: HourlySlot[];
}) {
  const [visible, setVisible] = useState<Record<SeriesKey, boolean>>({
    productive: true,
    distractive: true,
    idle: false,
    other: false,
  });

  const maxMinutes = 60;

  const toggle = (key: SeriesKey) =>
    setVisible((prev) => ({ ...prev, [key]: !prev[key] }));

  const activeSeries = SERIES.filter((s) => visible[s.key]);

  return (
    <TooltipProvider>
      <div className="flex">
        <div className="flex flex-col justify-between h-20 mr-2 text-[10px] text-muted-foreground text-right min-w-fit">
          <span>1h</span>
          <span>30m</span>
          <span>0</span>
        </div>
        <div className="flex items-end gap-[2px] h-20 flex-1 overflow-hidden">
          {hourlyData.map((hour) => {
            const totalSeconds = activeSeries.reduce(
              (sum, s) => sum + (hour[s.field] ?? 0),
              0
            );
            const totalMinutes = totalSeconds / 60;
            const rawHeight = maxMinutes > 0 ? (totalMinutes / maxMinutes) * 100 : 0;
            const height = totalMinutes > 0 ? Math.max(rawHeight, 5) : 0;

            if (totalMinutes === 0) {
              return (
                <Tooltip key={hour.HourLabel}>
                  <TooltipTrigger asChild>
                    <div className="flex-1 h-full flex items-end cursor-default">
                      <div className="w-full h-full border border-dashed border-muted-foreground/10 rounded bg-muted/5" />
                    </div>
                  </TooltipTrigger>
                  <TooltipContent className="bg-popover/80 backdrop-blur-md border-muted/20 shadow-xl px-3 py-2">
                    <div className="flex flex-col gap-1">
                      <span className="text-[10px] text-muted-foreground font-semibold uppercase tracking-wider">
                        {hour.HourLabel}
                      </span>
                      <span className="text-xs text-muted-foreground/80 font-medium">
                        No activity tracked
                      </span>
                    </div>
                  </TooltipContent>
                </Tooltip>
              );
            }

            // Stacking order top-to-bottom: distractive, other, idle, productive
            const segments = [
              { ...SERIES[1], seconds: visible.distractive ? hour.distractive_seconds : 0 },
              { ...SERIES[3], seconds: visible.other ? hour.other_seconds : 0 },
              { ...SERIES[2], seconds: visible.idle ? hour.idle_seconds : 0 },
              { ...SERIES[0], seconds: visible.productive ? hour.productive_seconds : 0 },
            ].filter((seg) => seg.seconds > 0);

            return (
              <Tooltip key={hour.HourLabel}>
                <TooltipTrigger asChild>
                  <div className="flex-1 h-full flex flex-col">
                    {/* Dashed placeholder for untracked portion of the hour */}
                    <div
                      className="w-full border border-dashed border-muted-foreground/10 rounded-t bg-muted/5"
                      style={{ height: `${Math.max(0, 100 - Math.min(100, height))}%` }}
                    />
                    {/* Actual activity segments */}
                    <div
                      className="w-full flex flex-col"
                      style={{ height: `${Math.min(100, height)}%` }}
                    >
                      {segments.map((seg, i) => (
                        <div
                          key={seg.key}
                          className={`w-full ${seg.bg} ${i === segments.length - 1 ? "rounded-b" : ""}`}
                          style={{ height: `${(seg.seconds / totalSeconds) * 100}%` }}
                        />
                      ))}
                    </div>
                  </div>
                </TooltipTrigger>
                <TooltipContent className="bg-popover/90 backdrop-blur-md border-muted/20 shadow-2xl px-3 py-2.5 min-w-[140px]">
                  <div className="space-y-2">
                    <p className="text-[10px] text-muted-foreground font-semibold uppercase tracking-wider border-b border-muted/10 pb-1.5">
                      {hour.HourLabel}
                    </p>
                    <div className="space-y-1.5">
                      {SERIES.map((s) => {
                        const val = hour[s.field] ?? 0;
                        if (val === 0) return null;
                        return (
                          <div key={s.key} className="flex items-center justify-between gap-4">
                            <div className="flex items-center gap-2">
                              <div className={`w-1.5 h-1.5 rounded-full ${s.dot}`} />
                              <span className="text-xs font-medium text-foreground/90">
                                {s.label}
                              </span>
                            </div>
                            <span className={`text-xs font-mono font-medium ${s.text}`}>
                              {Math.round(val / 60)}m
                            </span>
                          </div>
                        );
                      })}
                      {(() => {
                        const trackedMinutes = SERIES.reduce(
                          (sum, s) => sum + (hour[s.field] ?? 0) / 60,
                          0
                        );
                        const untrackedMinutes = Math.round(60 - trackedMinutes);
                        if (untrackedMinutes <= 1) return null;
                        return (
                          <div className="flex items-center justify-between gap-4 pt-1 border-t border-muted/5 mt-1">
                            <div className="flex items-center gap-2">
                              <div className="w-1.5 h-1.5 rounded-full bg-muted/40" />
                              <span className="text-xs font-medium text-muted-foreground">
                                Untracked
                              </span>
                            </div>
                            <span className="text-xs font-mono font-medium text-muted-foreground/80">
                              {untrackedMinutes}m
                            </span>
                          </div>
                        );
                      })()}
                    </div>
                  </div>
                </TooltipContent>
              </Tooltip>
            );
          })}
        </div>
      </div>
      <div className="flex gap-[2px] mt-1 text-[10px] text-muted-foreground ml-8">
        {hourlyData.filter((_, i) => i % 2 === 0).map((hour) => (
          <div key={hour.HourLabel} className="flex-1 min-w-0 text-center truncate">
            {hour.HourLabel}
          </div>
        ))}
      </div>
      {/* Legend toggles */}
      <div className="flex items-center gap-3 mt-2 ml-8">
        {SERIES.map((s) => (
          <button
            key={s.key}
            type="button"
            onClick={() => toggle(s.key)}
            className={`flex items-center gap-1 text-[10px] transition-opacity ${visible[s.key] ? "opacity-100" : "opacity-40"}`}
          >
            <span className={`w-2 h-2 rounded-full ${visible[s.key] ? s.dot : "bg-muted-foreground/40"}`} />
            {s.label}
          </button>
        ))}
      </div>
    </TooltipProvider>
  );
}

// Helper to check if a date is yesterday
function isYesterday(date: Date): boolean {
  const now = new Date();
  const yesterday = new Date(now);
  yesterday.setDate(yesterday.getDate() - 1);
  return (
    date.getDate() === yesterday.getDate() &&
    date.getMonth() === yesterday.getMonth() &&
    date.getFullYear() === yesterday.getFullYear()
  );
}

export function BentoDashboard() {
  const {
    selectedDate,
    overview,
    isLoading: isStoreLoading,
    error,
    fetchOverview,
    goToPrevDay,
    goToNextDay,
    goToToday,
  } = useUsageStore();

  const selectedDateKey = `${selectedDate.getFullYear()}-${selectedDate.getMonth()}-${selectedDate.getDate()}`;

  const { isLoading: isQueryLoading } = useQuery({
    queryKey: ["day-insights", selectedDateKey],
    queryFn: () => fetchOverview(selectedDate),
    retry: false,
  });

  const isLoading = isStoreLoading || isQueryLoading;

  const productiveSeconds = overview?.productivity_score?.productive_seconds ?? 0;
  const distractiveSeconds = overview?.productivity_score?.distractive_seconds ?? 0;
  const totalTrackedSeconds = productiveSeconds + distractiveSeconds;
  const hasEnoughData = totalTrackedSeconds >= MIN_SECONDS_FOR_INSIGHTS;

  const focusScore = Math.round(overview?.productivity_score?.productivity_score ?? 0);
  const productiveMinutes = Math.round(productiveSeconds / 60);
  const distractiveMinutes = Math.round(distractiveSeconds / 60);

  // Build 24-slot hourly breakdown from the backend's per-hour map
  const hourlyBreakdown = useMemo(
    () => buildHourlySlots(overview?.productivity_per_hour_breakdown as Record<string, ProductivityScore> | undefined),
    [overview?.productivity_per_hour_breakdown]
  );

  const canGoNext = !isToday(selectedDate);

  // Show loading overlay if data is loading
  if (isLoading && !overview) {
    return (
      <div className="p-6 flex items-center justify-center h-full">
        <div className="text-muted-foreground">Loading insights...</div>
      </div>
    );
  }

  if (error && !overview) {
    return (
      <div className="p-6 flex items-center justify-center h-full">
        <div className="text-destructive text-sm">Failed to load insights: {error}</div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      {/* Date Picker Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="icon"
            onClick={goToPrevDay}
            className="h-8 w-8"
          >
            <IconChevronLeft className="w-4 h-4" />
          </Button>
          <h2 className="text-lg font-semibold min-w-[180px] text-center">
            {formatDate(selectedDate)}
          </h2>
          <Button
            variant="ghost"
            size="icon"
            onClick={goToNextDay}
            disabled={!canGoNext}
            className="h-8 w-8"
          >
            <IconChevronRight className="w-4 h-4" />
          </Button>
        </div>
        {!isToday(selectedDate) && (
          <Button variant="outline" size="sm" onClick={goToToday}>
            Today
          </Button>
        )}
      </div>

      {/* Row 0: LLM Summary (At the top if it exists) */}
      {overview?.llm_daily_summary && (
        <LLMInsightCard
          dailyUsageSummary={overview.llm_daily_summary}
          isYesterday={isYesterday(selectedDate)}
        />
      )}

      {/* Row 1: Hero Stats */}
      <div className="grid grid-cols-4 gap-4">
        {/* Focus Score - Large */}
        <Card className="col-span-2 bg-gradient-to-br from-blue-500/10 to-blue-600/5 border-blue-500/20">
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs font-bold uppercase tracking-widest text-blue-400">
                  Focus Score
                </p>
                {hasEnoughData ? (
                  <>
                    <p className="text-5xl font-bold text-blue-400 mt-1">
                      {focusScore}%
                    </p>
                    <p className="text-xs text-muted-foreground mt-2">
                      When I chose between work and distraction, I won {focusScore}% of the time
                    </p>
                  </>
                ) : (
                  <>
                    <p className="text-3xl font-bold text-muted-foreground/40 mt-1">--</p>
                    <p className="text-xs text-muted-foreground mt-2">
                      Requires at least 1h of activity
                    </p>
                  </>
                )}
              </div>
              {hasEnoughData && (
                <div className="w-24 h-24">
                  <svg viewBox="0 0 100 100" className="transform -rotate-90">
                    <circle
                      cx="50"
                      cy="50"
                      r="40"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="8"
                      className="text-blue-500/20"
                    />
                    <circle
                      cx="50"
                      cy="50"
                      r="40"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="8"
                      strokeDasharray={`${focusScore * 2.51} 251`}
                      strokeLinecap="round"
                      className="text-blue-500"
                    />
                  </svg>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Productive Time */}
        <Card className="bg-gradient-to-br from-emerald-500/10 to-emerald-600/5 border-emerald-500/20">
          <CardContent className="pt-6">
            <p className="text-xs font-bold uppercase tracking-widest text-emerald-400">
              Productive
            </p>
            <p className="text-3xl font-bold text-emerald-400 mt-1">
              {formatMinutes(productiveMinutes)}
            </p>
            <p className="text-xs text-muted-foreground mt-2">
              Deep focus time
            </p>
          </CardContent>
        </Card>

        {/* Distractive Hours */}
        <Card className="bg-gradient-to-br from-rose-500/10 to-rose-600/5 border-rose-500/20">
          <CardContent className="pt-6">
            <p className="text-xs font-bold uppercase tracking-widest text-rose-400">
              Distractive
            </p>
            <p className="text-3xl font-bold text-rose-400 mt-1">
              {formatMinutes(distractiveMinutes)}
            </p>
            <p className="text-xs text-muted-foreground mt-2">
              Time lost
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Row 2: Full-width Hourly Breakdown */}
      <Card className="border-border/50">
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-medium">
              Activity Throughout the Day
            </CardTitle>
            <div className="flex items-center gap-4">
              <Link
                to="/screen-time/screentime"
                className="flex items-center gap-1 text-xs text-violet-400 hover:text-violet-300 transition-colors"
              >
                <IconHistory className="w-3.5 h-3.5" />
                History
              </Link>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <HourlyBreakdownChart hourlyData={hourlyBreakdown} />
        </CardContent>
      </Card>

      {/* Row 3: Time Lost To + Blocked Today */}
      <div className="grid grid-cols-2 gap-4">
        <TopDistractionsCard distractions={overview?.top_distractions ?? []} />
        <TopBlockedCard blockedAttempts={overview?.top_blocked ?? []} />
      </div>

      {/* Row 4: Projects + Communication */}
      <div className="grid grid-cols-2 gap-4">
        <CategoriesCard projects={overview?.project_breakdown ?? []} />
        <CommunicationCard channels={buildSortedChannels(overview?.communication_breakdown)} />
      </div>
    </div>
  );
}