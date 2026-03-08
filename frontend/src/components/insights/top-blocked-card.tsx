import { IconShield, IconArrowRight } from "@tabler/icons-react";
import { Link } from "@tanstack/react-router";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { BlockedBreakdown } from "@/../bindings/github.com/focusd-so/focusd/internal/usage/models";

interface TopBlockedCardProps {
  blockedAttempts: BlockedBreakdown[];
}

export function TopBlockedCard({ blockedAttempts }: TopBlockedCardProps) {
  const totalBlocked = blockedAttempts.reduce((sum, b) => sum + b.count, 0);
  const maxCount = Math.max(...blockedAttempts.map((b) => b.count), 1);

  const topBlocked = blockedAttempts.slice(0, 5);

  return (
    <Card className="bg-gradient-to-br from-cyan-500/10 to-blue-600/5 border-cyan-500/20">
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <IconShield className="w-4 h-4 text-cyan-400" />
            <span className="text-cyan-400">Blocked Today</span>
          </CardTitle>
          <Link
            to="/screen-time/screentime"
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-cyan-400 transition-colors"
          >
            {totalBlocked} total
            <IconArrowRight className="w-3 h-3" />
          </Link>
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        {topBlocked.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-6 text-center">
            <IconShield className="w-6 h-6 text-cyan-400/20 mb-2" />
            <p className="text-xs text-muted-foreground">No blocked attempts today</p>
            <p className="text-[10px] text-muted-foreground/60 mt-1">Your blocklist is ready to protect your focus</p>
          </div>
        ) : (
          topBlocked.map((attempt, index) => {
            const widthPct = (attempt.count / maxCount) * 100;
            return (
              <div key={index} className="space-y-1">
                <div className="flex items-center justify-between text-xs">
                  <span className="truncate max-w-[140px]">{attempt.name}</span>
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
