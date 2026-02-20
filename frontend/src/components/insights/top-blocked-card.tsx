import { IconShield } from "@tabler/icons-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { BlockedAttempt } from "@/lib/mock-data";

interface TopBlockedCardProps {
  blockedAttempts: BlockedAttempt[];
}

export function TopBlockedCard({ blockedAttempts }: TopBlockedCardProps) {
  const totalBlocked = blockedAttempts.reduce((sum, b) => sum + b.count, 0);
  const maxCount = Math.max(...blockedAttempts.map((b) => b.count), 1);

  // Show top 5
  const topBlocked = blockedAttempts.slice(0, 5);

  return (
    <Card className="bg-gradient-to-br from-cyan-500/10 to-blue-600/5 border-cyan-500/20">
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <IconShield className="w-4 h-4 text-cyan-400" />
            <span className="text-cyan-400">Blocked Today</span>
          </CardTitle>
          <span className="text-xs text-muted-foreground">
            {totalBlocked} total
          </span>
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        {topBlocked.length === 0 ? (
          <p className="text-xs text-muted-foreground text-center py-4">
            No blocked attempts today
          </p>
        ) : (
          topBlocked.map((attempt) => {
            const widthPct = (attempt.count / maxCount) * 100;
            return (
              <div key={attempt.id} className="space-y-1">
                <div className="flex items-center justify-between text-xs">
                  <span className="truncate max-w-[140px]">{attempt.hostname}</span>
                  <span className="text-muted-foreground font-mono">
                    {attempt.count}x
                  </span>
                </div>
                <div className="h-1.5 bg-cyan-500/10 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-cyan-500/60 rounded-full transition-all"
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
