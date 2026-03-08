import { IconMessages, IconArrowRight } from "@tabler/icons-react";
import { Link } from "@tanstack/react-router";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatMinutes } from "@/lib/mock-data";
import type { CommunicationBreakdown } from "@/../bindings/github.com/focusd-so/focusd/internal/usage/models";

interface CommunicationCardProps {
  channels: CommunicationBreakdown[];
}

const channelTextColors: Record<string, string> = {
  Slack: "text-purple-400",
  Email: "text-blue-400",
  Zoom: "text-sky-400",
  Discord: "text-indigo-400",
  Teams: "text-violet-400",
};

const channelBarColors: Record<string, string> = {
  Slack: "bg-purple-500/60",
  Email: "bg-blue-500/60",
  Zoom: "bg-sky-500/60",
  Discord: "bg-indigo-500/60",
  Teams: "bg-violet-500/60",
};

export function CommunicationCard({ channels }: CommunicationCardProps) {
  const totalMinutes = channels.reduce((sum, c) => sum + c.minutes, 0);
  const maxMinutes = Math.max(...channels.map((c) => c.minutes), 1);

  return (
    <Card className="border-border/50">
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <IconMessages className="w-4 h-4 text-muted-foreground" />
            Communication
          </CardTitle>
          <Link
            to="/screen-time/screentime"
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            {formatMinutes(totalMinutes)} total
            <IconArrowRight className="w-3 h-3" />
          </Link>
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        {channels.length === 0 ? (
          <p className="text-xs text-muted-foreground text-center py-4">
            No communication activity
          </p>
        ) : (
          channels.slice(0, 3).map((channel, index) => {
            const textColor = channelTextColors[channel.name] || "text-muted-foreground";
            const widthPct = (channel.minutes / maxMinutes) * 100;

            return (
              <div key={index} className="space-y-1">
                <div className="flex items-center justify-between text-xs">
                  <span className="truncate max-w-[140px]">{channel.name}</span>
                  <span className={`font-mono ${textColor}`}>
                    {formatMinutes(channel.minutes)}
                  </span>
                </div>
                <div className="h-1.5 bg-muted/20 rounded-full overflow-hidden">
                  <div
                    className={`h-full rounded-full transition-all ${channelBarColors[channel.name] || "bg-muted-foreground/40"}`}
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
