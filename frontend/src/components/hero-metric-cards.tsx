import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Progress } from "@/components/ui/progress";

// Defined locally - screentime service was removed from backend
interface UsageStats {
  productive_minutes: number;
  neutral_minutes: number;
  distractive_minutes: number;
  productivity_score: number;
}

// Type Definitions
export type ComparisonMode = "7-day-avg" | "yesterday";

export type DailyStats = {
  date: number;
  productive_minutes: number;
  neutral_minutes: number;
  distractive_minutes: number;
};

interface MetricCardProps {
  title: string;
  minutes: number;
  colorScheme: "emerald" | "amber" | "rose" | "slate" | "blue";
  contextLabel: string;
}

interface FocusScoreCardProps {
  score: number; // 0-100
  contextLabel: string;
}

// Utility Functions
function formatMinutesToHoursMinutes(minutes: number): string {
  const roundedMinutes = Math.round(minutes);
  const hours = Math.floor(roundedMinutes / 60);
  const mins = roundedMinutes % 60;
  if (hours > 0) {
    return `${hours}h ${mins}m`;
  }
  return `${mins}m`;
}

function getContextLabel(comparisonMode: ComparisonMode): string {
  switch (comparisonMode) {
    case "7-day-avg":
      return "7-day avg";
    case "yesterday":
      return "Yesterday";
  }
}

function calculateFocusScore(productive: number, distractive: number): number {
  const total = productive + distractive;
  if (total === 0) return 100; // No activity = perfect focus (no distractions)
  return (productive / total) * 100;
}

// Color scheme configurations with gradients and hover states
const colorSchemes = {
  emerald: {
    card: "bg-gradient-to-br from-emerald-500/15 to-emerald-600/5 border-emerald-500/30 hover:border-emerald-500/50 hover:shadow-emerald-500/10",
    title: "text-emerald-500",
    value: "text-emerald-500",
    progress: "bg-emerald-500",
  },
  amber: {
    card: "bg-gradient-to-br from-amber-500/15 to-amber-600/5 border-amber-500/30 hover:border-amber-500/50 hover:shadow-amber-500/10",
    title: "text-amber-500",
    value: "text-amber-500",
    progress: "bg-amber-500",
  },
  rose: {
    card: "bg-gradient-to-br from-rose-500/15 to-rose-600/5 border-rose-500/30 hover:border-rose-500/50 hover:shadow-rose-500/10",
    title: "text-rose-500",
    value: "text-rose-500",
    progress: "bg-rose-500",
  },
  slate: {
    card: "bg-gradient-to-br from-slate-500/15 to-slate-600/5 border-slate-500/30 hover:border-slate-500/50 hover:shadow-slate-500/10",
    title: "text-slate-400",
    value: "text-slate-400",
    progress: "bg-slate-500",
  },
  blue: {
    card: "bg-gradient-to-br from-blue-500/15 to-blue-600/5 border-blue-500/30 hover:border-blue-500/50 hover:shadow-blue-500/10",
    title: "text-blue-500",
    value: "text-blue-500",
    progress: "bg-blue-500",
  },
};

function MetricCard({
  title,
  minutes,
  colorScheme,
  contextLabel,
}: MetricCardProps) {
  const colors = colorSchemes[colorScheme];
  const formattedTime = formatMinutesToHoursMinutes(minutes);

  // Build tooltip content
  const tooltipContent = `${Math.round(minutes)} minutes · ${contextLabel}`;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Card
          className={cn(
            "border cursor-default transition-all duration-200 hover:shadow-lg",
            colors.card
          )}
        >
          <CardContent className="pt-6">
            <div className="flex flex-col gap-2">
              {/* Category Label */}
              <span
                className={cn(
                  "text-xs font-bold uppercase tracking-widest",
                  colors.title
                )}
              >
                {title}
              </span>

              {/* Time Display */}
              <span className={cn("text-4xl font-bold", colors.value)}>
                {formattedTime}
              </span>

              {/* Context Label */}
              <div className="flex items-center gap-1">
                <span className="text-xs text-muted-foreground">
                  {contextLabel}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>
      </TooltipTrigger>
      <TooltipContent>
        <p>{tooltipContent}</p>
      </TooltipContent>
    </Tooltip>
  );
}

function FocusScoreCard({
  score,
  contextLabel,
}: FocusScoreCardProps) {
  const colors = colorSchemes.blue;

  // Build tooltip content
  const tooltipContent = `${score.toFixed(1)}% focus score · ${contextLabel}`;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Card
          className={cn(
            "border cursor-default transition-all duration-200 hover:shadow-lg",
            colors.card
          )}
        >
          <CardContent className="pt-6">
            <div className="flex flex-col gap-2">
              <span
                className={cn(
                  "text-xs font-bold uppercase tracking-widest",
                  colors.title
                )}
              >
                Focus Score
              </span>

              <span className={cn("text-4xl font-bold", colors.value)}>
                {score.toFixed(0)}%
              </span>

              {/* Progress bar for visual representation */}
              <Progress
                value={score}
                className="h-1.5 bg-blue-500/20"
              />

              <div className="flex items-center gap-1">
                <span className="text-xs text-muted-foreground">
                  {contextLabel}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>
      </TooltipTrigger>
      <TooltipContent>
        <p>{tooltipContent}</p>
      </TooltipContent>
    </Tooltip>
  );
}

interface HeroMetricCardsProps {
  stats: UsageStats | null;
  comparisonMode?: ComparisonMode;
}

export function HeroMetricCards({
  stats,
  comparisonMode = "7-day-avg",
}: HeroMetricCardsProps) {
  if (!stats) {
    return null;
  }

  const contextLabel = getContextLabel(comparisonMode);

  // Calculate totals for footer
  const totalActiveMinutes =
    stats.productive_minutes +
    stats.neutral_minutes +
    stats.distractive_minutes;

  const focusScore = Number.isFinite(stats.productivity_score)
    ? stats.productivity_score
    : calculateFocusScore(stats.productive_minutes, stats.distractive_minutes);

  return (
    <div className="space-y-3">
      <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Productive"
          minutes={stats.productive_minutes}
          colorScheme="emerald"
          contextLabel={contextLabel}
        />
        <MetricCard
          title="Distractive"
          minutes={stats.distractive_minutes}
          colorScheme="rose"
          contextLabel={contextLabel}
        />
        <MetricCard
          title="Other"
          minutes={stats.neutral_minutes}
          colorScheme="amber"
          contextLabel={contextLabel}
        />
        <FocusScoreCard
          score={focusScore}
          contextLabel={contextLabel}
        />
      </div>
      <div className="flex justify-center">
        <div className="inline-flex items-center gap-3 rounded-full bg-muted/50 border border-border/50 px-4 py-1.5 text-sm text-muted-foreground">
          <span>
            {formatMinutesToHoursMinutes(totalActiveMinutes)} tracked{" "}
            {contextLabel.toLowerCase()}
          </span>
        </div>
      </div>
    </div>
  );
}
