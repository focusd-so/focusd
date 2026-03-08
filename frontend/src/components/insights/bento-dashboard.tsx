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
import type { UsagePerHourBreakdown } from "@/stores/usage-store";
import { LLMInsightCard } from "./ai-insight-card";
import { TopBlockedCard } from "./top-blocked-card";
import { TopDistractionsCard } from "./top-distractions-card";
import { CategoriesCard } from "./categories-card";
import { CommunicationCard } from "./communication-card";

const MIN_SECONDS_FOR_INSIGHTS = 3600;

// Hourly breakdown chart component
function HourlyBreakdownChart({
  hourlyData,
}: {
  hourlyData: UsagePerHourBreakdown[];
}) {
  // Fixed 1-hour scale for Y-axis
  const maxMinutes = 60;

  return (
    <TooltipProvider>
      <div className="flex">
        {/* Y-axis labels - fixed 1 hour scale */}
        <div className="flex flex-col justify-between h-20 mr-2 text-[10px] text-muted-foreground text-right min-w-fit">
          <span>1h</span>
          <span>30m</span>
          <span>0</span>
        </div>
        {/* Bars */}
        <div className="flex items-end gap-[2px] h-20 flex-1 overflow-hidden">
          {hourlyData.map((hour) => {
            // Only use Productive + Distractive
            const totalSeconds =
              hour.ProductiveSeconds +
              hour.DistractiveSeconds;

            const totalMinutes = totalSeconds / 60;
            const height = maxMinutes > 0 ? (totalMinutes / maxMinutes) * 100 : 0;
            const prodPct =
              totalSeconds > 0 ? (hour.ProductiveSeconds / totalSeconds) * 100 : 0;
            const disPct =
              totalSeconds > 0 ? (hour.DistractiveSeconds / totalSeconds) * 100 : 0;

            if (totalMinutes === 0) {
              return (
                <div
                  key={hour.HourLabel}
                  className="flex-1 h-full flex items-end"
                >
                  <div className="w-full h-1 bg-muted/20 rounded-t" />
                </div>
              );
            }

            return (
              <Tooltip key={hour.HourLabel}>
                <TooltipTrigger asChild>
                  <div
                    className="flex-1 flex flex-col"
                    style={{ height: `${Math.min(100, height)}%` }}
                  >
                    {/* Distractive (top - red) */}
                    {disPct > 0 && (
                      <div
                        className="w-full bg-rose-500/80"
                        style={{ height: `${disPct}%` }}
                      />
                    )}

                    {/* Productive (bottom - emerald) */}
                    {prodPct > 0 && (
                      <div
                        className="w-full bg-emerald-500/80 rounded-b"
                        style={{ height: `${prodPct}%` }}
                      />
                    )}
                  </div>
                </TooltipTrigger>
                <TooltipContent>
                  <div className="text-xs space-y-1">
                    <p className="font-medium">
                      {hour.HourLabel}
                    </p>
                    <p className="text-emerald-400">
                      Productive: {Math.round(hour.ProductiveSeconds / 60)}m
                    </p>

                    <p className="text-rose-400">
                      Distractive: {Math.round(hour.DistractiveSeconds / 60)}m
                    </p>
                  </div>
                </TooltipContent>
              </Tooltip>
            );
          })}
        </div>
      </div>
      {/* Hour labels */}
      <div className="flex justify-between mt-1 text-[10px] text-muted-foreground ml-8">
        <span>12am</span>
        <span>6am</span>
        <span>12pm</span>
        <span>6pm</span>
        <span>11pm</span>
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

  const productiveSeconds = overview?.UsageOverview?.ProductiveSeconds ?? 0;
  const distractiveSeconds = overview?.UsageOverview?.DistractiveSeconds ?? 0;
  const totalTrackedSeconds = productiveSeconds + distractiveSeconds;
  const hasEnoughData = totalTrackedSeconds >= MIN_SECONDS_FOR_INSIGHTS;

  const focusScore = Math.round(overview?.UsageOverview?.ProductivityScore ?? 0);
  const productiveMinutes = Math.round(productiveSeconds / 60);
  const distractiveMinutes = Math.round(distractiveSeconds / 60);

  // Get hourly breakdown from backend (already in UsagePerHourBreakdown format with seconds)
  // Filter out any null values that may come from the backend
  const hourlyBreakdown = (overview?.UsagePerHourBreakdown ?? []).filter(
    (item): item is UsagePerHourBreakdown => item !== null
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
      {overview?.DailyUsageSummary && overview.DailyUsageSummary.headline && (
        <LLMInsightCard
          dailyUsageSummary={overview.DailyUsageSummary}
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
              <div className="flex items-center gap-4 text-[10px]">
                <span className="flex items-center gap-1">
                  <span className="w-2 h-2 bg-emerald-500 rounded-full" />
                  Productive
                </span>
                <span className="flex items-center gap-1">
                  <span className="w-2 h-2 bg-rose-500 rounded-full" />
                  Distractive
                </span>
              </div>
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
        <TopDistractionsCard distractions={overview?.TopDistractions ?? []} />
        <TopBlockedCard blockedAttempts={overview?.TopBlocked ?? []} />
      </div>

      {/* Row 4: Projects + Communication */}
      <div className="grid grid-cols-2 gap-4">
        <CategoriesCard projects={overview?.ProjectBreakdown ?? []} />
        <CommunicationCard channels={overview?.CommunicationBreakdown ?? []} />
      </div>
    </div>
  );
}