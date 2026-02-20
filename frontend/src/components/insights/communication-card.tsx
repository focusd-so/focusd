import { IconMessages } from "@tabler/icons-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatMinutes, type CommunicationChannel } from "@/lib/mock-data";

interface CommunicationCardProps {
  channels: CommunicationChannel[];
}

// Background colors for each channel
const channelColors: Record<string, string> = {
  Slack: "bg-purple-500/20 border-purple-500/30",
  Email: "bg-blue-500/20 border-blue-500/30",
  Zoom: "bg-sky-500/20 border-sky-500/30",
  Discord: "bg-indigo-500/20 border-indigo-500/30",
  Teams: "bg-violet-500/20 border-violet-500/30",
};

const channelTextColors: Record<string, string> = {
  Slack: "text-purple-400",
  Email: "text-blue-400",
  Zoom: "text-sky-400",
  Discord: "text-indigo-400",
  Teams: "text-violet-400",
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
          <span className="text-xs text-muted-foreground">
            {formatMinutes(totalMinutes)} total
          </span>
        </div>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-5 gap-2">
          {channels.map((channel) => {
            const bgColor = channelColors[channel.name] || "bg-muted/20 border-muted/30";
            const textColor = channelTextColors[channel.name] || "text-muted-foreground";
            const heightPct = Math.max(20, (channel.minutes / maxMinutes) * 100);

            return (
              <div
                key={channel.id}
                className={`relative rounded-lg border ${bgColor} p-2 flex flex-col items-center justify-end transition-all hover:scale-105`}
                style={{ minHeight: "80px" }}
              >
                {/* Visual bar indicator */}
                <div className="absolute bottom-0 left-0 right-0 rounded-b-lg bg-white/5" style={{ height: `${heightPct}%` }} />

                {/* Content */}
                <div className="relative z-10 text-center">
                  <span className="text-lg">{channel.icon}</span>
                  <p className="text-[10px] font-medium mt-1 truncate max-w-full">
                    {channel.name}
                  </p>
                  <p className={`text-xs font-bold ${textColor}`}>
                    {formatMinutes(channel.minutes)}
                  </p>
                </div>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}
