import { IconAlertTriangle } from "@tabler/icons-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatMinutes, type DistractionItem } from "@/lib/mock-data";

interface TopDistractionsCardProps {
  distractions: DistractionItem[];
}

export function TopDistractionsCard({ distractions }: TopDistractionsCardProps) {
  const totalMinutes = distractions.reduce((sum, d) => sum + d.minutes, 0);
  const maxMinutes = Math.max(...distractions.map((d) => d.minutes), 1);

  // Show top 5
  const topDistractions = distractions.slice(0, 5);

  return (
    <Card className="bg-gradient-to-br from-rose-500/10 to-orange-600/5 border-rose-500/20">
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <IconAlertTriangle className="w-4 h-4 text-rose-400" />
            <span className="text-rose-400">Time Lost To</span>
          </CardTitle>
          <span className="text-xs text-muted-foreground">
            {formatMinutes(totalMinutes)}
          </span>
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        {topDistractions.length === 0 ? (
          <p className="text-xs text-muted-foreground text-center py-4">
            No distractions recorded
          </p>
        ) : (
          topDistractions.map((distraction, index) => {
            const widthPct = (distraction.minutes / maxMinutes) * 100;
            return (
              <div key={distraction.id} className="space-y-1">
                <div className="flex items-center justify-between text-xs">
                  <span className="flex items-center gap-2">
                    <span className="text-muted-foreground w-4">{index + 1}.</span>
                    <span className="truncate max-w-[100px]">{distraction.name}</span>
                    <span className="text-[10px] text-muted-foreground bg-muted/30 px-1.5 py-0.5 rounded">
                      {distraction.category}
                    </span>
                  </span>
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
