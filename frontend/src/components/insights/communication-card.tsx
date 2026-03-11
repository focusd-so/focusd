import { IconMessages, IconArrowRight, IconInfoCircle } from "@tabler/icons-react";
import { Link } from "@tanstack/react-router";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { formatDuration } from "@/lib/mock-data";
import type { CommunicationBreakdown } from "@/../bindings/github.com/focusd-so/focusd/internal/usage/models";


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

export function CommunicationCard({ channels }: { channels: CommunicationBreakdown[] }) {
  // const totalMinutes = channels.reduce((sum, c) => sum + c.minutes, 0);
  const maxSeconds = Math.max(...channels.map((c) => c.duration_seconds), 1);

  return (
    <Card className="border-border/50">
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <IconMessages className="w-4 h-4 text-muted-foreground" />
            Communication
            <TooltipProvider>
              <Tooltip delayDuration={300}>
                <TooltipTrigger asChild>
                  <IconInfoCircle className="w-3.5 h-3.5 text-muted-foreground/50 hover:text-muted-foreground cursor-help transition-colors" />
                </TooltipTrigger>
                <TooltipContent className="max-w-[250px] text-xs text-muted-foreground bg-popover/90 backdrop-blur-md px-3 py-2 border-muted/20 shadow-xl">
                  Channels and conversation names are automatically inferred by AI from your messaging apps (e.g., Slack).
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </CardTitle>
          <Link
            to="/screen-time/screentime"
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            {/* {formatMinutes(totalMinutes)} total */}
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
          channels.slice(0, 5).map((channel, index) => {
            const textColor = channelTextColors[channel.channel] || "text-muted-foreground";
            const widthPct = (channel.duration_seconds / maxSeconds) * 100;

            return (
              <div key={index} className="space-y-1">
                <div className="flex items-center justify-between text-xs">
                  <span
                    className="truncate max-w-[300px]"
                    title={`${channel.name} | ${channel.channel}`}
                  >
                    {channel.name} | {channel.channel}
                  </span>
                  <span className={`font-mono ${textColor}`}>
                    {formatDuration(channel.duration_seconds)}
                  </span>
                </div>
                <div className="h-1.5 bg-muted/20 rounded-full overflow-hidden">
                  <div
                    className={`h-full rounded-full transition-all ${channelBarColors[channel.channel] || "bg-muted-foreground/40"}`}
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
