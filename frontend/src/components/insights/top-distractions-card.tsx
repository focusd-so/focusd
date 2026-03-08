import { IconAlertTriangle, IconArrowRight } from "@tabler/icons-react";
import { Link } from "@tanstack/react-router";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatMinutes } from "@/lib/mock-data";
import type { DistractionBreakdown } from "@/../bindings/github.com/focusd-so/focusd/internal/usage/models";

interface TopDistractionsCardProps {
  distractions: DistractionBreakdown[];
}

export function TopDistractionsCard({ distractions }: TopDistractionsCardProps) {
  const totalMinutes = distractions.reduce((sum, d) => sum + d.minutes, 0);
  const maxMinutes = Math.max(...distractions.map((d) => d.minutes), 1);

  const topDistractions = distractions.slice(0, 5);

  return (
    <Card className="bg-gradient-to-br from-rose-500/10 to-orange-600/5 border-rose-500/20">
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <IconAlertTriangle className="w-4 h-4 text-rose-400" />
            <span className="text-rose-400">Time Lost To</span>
          </CardTitle>
          <Link
            to="/screen-time/screentime"
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-rose-400 transition-colors"
          >
            {formatMinutes(totalMinutes)}
            <IconArrowRight className="w-3 h-3" />
          </Link>
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        {topDistractions.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-6 text-center">
            <IconAlertTriangle className="w-6 h-6 text-rose-400/20 mb-2" />
            <p className="text-xs text-muted-foreground">No distractions recorded</p>
            <p className="text-[10px] text-muted-foreground/60 mt-1">Stay focused and keep it that way</p>
          </div>
        ) : (
          topDistractions.map((distraction, index) => {
            const widthPct = (distraction.minutes / maxMinutes) * 100;
            return (
              <div key={index} className="space-y-1">
                <div className="flex items-center justify-between text-xs">
                  <span className="truncate max-w-[140px]">{distraction.name}</span>
                  <span className="text-muted-foreground font-mono">
                    {formatMinutes(distraction.minutes)}
                  </span>
                </div>
                <div className="h-1.5 bg-rose-500/10 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-rose-500/60 rounded-full transition-all"
                    style={{ width: `${widthPct}%` }}
                  />
                </div>
              </div>
            );
          })
        )}
      </CardContent>
    </Card>
  );
}
