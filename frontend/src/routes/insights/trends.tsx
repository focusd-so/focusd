import { createFileRoute } from "@tanstack/react-router";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  IconTrendingUp,
  IconTrendingDown,
  IconMinus,
  IconFlame,
  IconTargetArrow,
} from "@tabler/icons-react";
import { getWeeklyStats, formatMinutes } from "@/lib/mock-data";

export const Route = createFileRoute("/insights/trends")({
  component: TrendsPage,
});

function TrendsPage() {
  const weeklyStats = getWeeklyStats();
  const dayLabels = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

  // Calculate averages
  const avgFocusScore = Math.round(
    weeklyStats.reduce((sum, d) => sum + d.focusScore, 0) / weeklyStats.length
  );
  const avgProductiveMinutes = Math.round(
    weeklyStats.reduce((sum, d) => sum + d.productiveMinutes, 0) / weeklyStats.length
  );
  const totalDeepWorkSessions = weeklyStats.reduce(
    (sum, d) => sum + d.deepWorkSessions,
    0
  );
  const totalBlocked = weeklyStats.reduce((sum, d) => sum + d.blockedAttempts, 0);

  // Calculate week-over-week change (mock)
  const prevWeekAvgFocus = avgFocusScore - 5 + Math.floor(Math.random() * 10);
  const focusChange = avgFocusScore - prevWeekAvgFocus;

  // Find best and worst days
  const bestDay = weeklyStats.reduce((best, current) =>
    current.focusScore > best.focusScore ? current : best
  );
  const worstDay = weeklyStats.reduce((worst, current) =>
    current.focusScore < worst.focusScore ? current : worst
  );

  const maxProductive = Math.max(...weeklyStats.map((d) => d.productiveMinutes));
  const maxDistracting = Math.max(...weeklyStats.map((d) => d.distractingMinutes));

  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold">Trends</h1>
        <p className="text-muted-foreground text-sm mt-1">
          Your productivity patterns over time
        </p>
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-4 gap-4">
        <Card className="bg-gradient-to-br from-blue-500/10 to-blue-600/5 border-blue-500/20">
          <CardContent className="pt-6">
            <p className="text-xs font-bold uppercase tracking-widest text-blue-400">
              Avg Focus Score
            </p>
            <div className="flex items-baseline gap-2 mt-1">
              <p className="text-3xl font-bold text-blue-400">{avgFocusScore}%</p>
              <TrendBadge value={focusChange} suffix="%" />
            </div>
            <p className="text-xs text-muted-foreground mt-2">This week</p>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-emerald-500/10 to-emerald-600/5 border-emerald-500/20">
          <CardContent className="pt-6">
            <p className="text-xs font-bold uppercase tracking-widest text-emerald-400">
              Avg Productive
            </p>
            <p className="text-3xl font-bold text-emerald-400 mt-1">
              {formatMinutes(avgProductiveMinutes)}
            </p>
            <p className="text-xs text-muted-foreground mt-2">Per day</p>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-violet-500/10 to-violet-600/5 border-violet-500/20">
          <CardContent className="pt-6">
            <p className="text-xs font-bold uppercase tracking-widest text-violet-400">
              Deep Work
            </p>
            <p className="text-3xl font-bold text-violet-400 mt-1">
              {totalDeepWorkSessions}
            </p>
            <p className="text-xs text-muted-foreground mt-2">Sessions this week</p>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-rose-500/10 to-rose-600/5 border-rose-500/20">
          <CardContent className="pt-6">
            <p className="text-xs font-bold uppercase tracking-widest text-rose-400">
              Blocked
            </p>
            <p className="text-3xl font-bold text-rose-400 mt-1">{totalBlocked}</p>
            <p className="text-xs text-muted-foreground mt-2">Distractions this week</p>
          </CardContent>
        </Card>
      </div>

      {/* Weekly Focus Score Chart */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Focus Score by Day</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-end gap-2 h-40">
            {weeklyStats.map((day, i) => {
              const date = new Date(day.date * 1000);
              const dayOfWeek = dayLabels[date.getDay()];
              const isBest = day === bestDay;
              const isWorst = day === worstDay;

              return (
                <div key={i} className="flex-1 flex flex-col items-center gap-2">
                  <div className="relative w-full h-32 flex items-end justify-center">
                    <div
                      className={`w-full max-w-12 rounded-t transition-all ${isBest
                          ? "bg-emerald-500"
                          : isWorst
                            ? "bg-rose-500/70"
                            : "bg-blue-500/70"
                        }`}
                      style={{ height: `${day.focusScore}%` }}
                    />
                    {isBest && (
                      <div className="absolute -top-6 text-emerald-400">
                        <IconFlame className="w-4 h-4" />
                      </div>
                    )}
                    {isWorst && (
                      <div className="absolute -top-6 text-rose-400">
                        <IconTargetArrow className="w-4 h-4" />
                      </div>
                    )}
                  </div>
                  <div className="text-center">
                    <p className="text-xs font-medium">{day.focusScore}%</p>
                    <p className="text-[10px] text-muted-foreground">{dayOfWeek}</p>
                  </div>
                </div>
              );
            })}
          </div>
          <div className="flex items-center justify-center gap-6 mt-4 text-xs text-muted-foreground">
            <span className="flex items-center gap-1">
              <IconFlame className="w-3 h-3 text-emerald-400" /> Best day
            </span>
            <span className="flex items-center gap-1">
              <IconTargetArrow className="w-3 h-3 text-rose-400" /> Needs improvement
            </span>
          </div>
        </CardContent>
      </Card>

      {/* Productive vs Distracting Chart */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Productive vs Distracting Time</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {weeklyStats.map((day, i) => {
              const date = new Date(day.date * 1000);
              const dayOfWeek = dayLabels[date.getDay()];
              const productivePct = (day.productiveMinutes / maxProductive) * 100;
              const distractingPct = (day.distractingMinutes / maxDistracting) * 100;

              return (
                <div key={i} className="grid grid-cols-[60px_1fr] gap-4 items-center">
                  <div className="text-sm text-muted-foreground text-right">
                    {dayOfWeek}
                  </div>
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <div
                        className="h-3 bg-emerald-500/80 rounded"
                        style={{ width: `${productivePct}%` }}
                      />
                      <span className="text-xs text-muted-foreground min-w-[50px]">
                        {formatMinutes(day.productiveMinutes)}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <div
                        className="h-3 bg-rose-500/60 rounded"
                        style={{ width: `${distractingPct}%` }}
                      />
                      <span className="text-xs text-muted-foreground min-w-[50px]">
                        {formatMinutes(day.distractingMinutes)}
                      </span>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
          <div className="flex items-center justify-center gap-6 mt-4 text-xs text-muted-foreground">
            <span className="flex items-center gap-1">
              <span className="w-3 h-3 bg-emerald-500 rounded" /> Productive
            </span>
            <span className="flex items-center gap-1">
              <span className="w-3 h-3 bg-rose-500 rounded" /> Distracting
            </span>
          </div>
        </CardContent>
      </Card>

      {/* <Card className="bg-gradient-to-br from-amber-500/5 to-amber-600/5 border-amber-500/20">
        <CardHeader>
          <CardTitle className="text-base text-amber-400">Weekly Insights</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <p>
            📈 Your best day was{" "}
            <strong>{dayLabels[new Date(bestDay.date * 1000).getDay()]}</strong> with a{" "}
            {bestDay.focusScore}% focus score and{" "}
            {formatMinutes(bestDay.productiveMinutes)} of productive time.
          </p>
          <p>
            🎯 You had <strong>{totalDeepWorkSessions} deep work sessions</strong> this
            week. Consider scheduling more uninterrupted blocks.
          </p>
          <p>
            🛡️ focusd blocked <strong>{totalBlocked} distractions</strong> - that's
            roughly {Math.round(totalBlocked * 5)} minutes of focus protected!
          </p>
        </CardContent>
      </Card> */}
    </div>
  );
}

function TrendBadge({ value, suffix = "" }: { value: number; suffix?: string }) {
  if (value > 0) {
    return (
      <Badge
        variant="outline"
        className="border-emerald-500/50 text-emerald-400 text-xs"
      >
        <IconTrendingUp className="w-3 h-3 mr-1" />+{value}
        {suffix}
      </Badge>
    );
  }
  if (value < 0) {
    return (
      <Badge variant="outline" className="border-rose-500/50 text-rose-400 text-xs">
        <IconTrendingDown className="w-3 h-3 mr-1" />
        {value}
        {suffix}
      </Badge>
    );
  }
  return (
    <Badge variant="outline" className="border-muted-foreground/50 text-xs">
      <IconMinus className="w-3 h-3 mr-1" />0{suffix}
    </Badge>
  );
}
