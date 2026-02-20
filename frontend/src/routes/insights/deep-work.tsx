import { createFileRoute } from "@tanstack/react-router";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { IconFlame, IconBrain } from "@tabler/icons-react";
import {
  mockDeepWorkSessions,
  mockTodayStats,
  formatMinutes,
  formatRelativeTime,
} from "@/lib/mock-data";

export const Route = createFileRoute("/insights/deep-work")({
  component: DeepWorkPage,
});

function DeepWorkPage() {
  const totalDeepWorkMinutes = mockDeepWorkSessions.reduce(
    (sum, s) => sum + s.durationMinutes,
    0
  );

  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      {/* Hero Stats */}
      <div className="grid grid-cols-3 gap-4">
        <Card className="bg-gradient-to-br from-violet-500/10 to-violet-600/5 border-violet-500/20">
          <CardContent className="pt-6">
            <div className="flex items-center gap-2 mb-2">
              <IconBrain className="w-5 h-5 text-violet-400" />
              <p className="text-xs font-bold uppercase tracking-widest text-violet-400">
                Sessions Today
              </p>
            </div>
            <p className="text-4xl font-bold">{mockDeepWorkSessions.length}</p>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-emerald-500/10 to-emerald-600/5 border-emerald-500/20">
          <CardContent className="pt-6">
            <p className="text-xs font-bold uppercase tracking-widest text-emerald-400 mb-2">
              Total Deep Work
            </p>
            <p className="text-4xl font-bold">
              {formatMinutes(totalDeepWorkMinutes)}
            </p>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-orange-500/10 to-orange-600/5 border-orange-500/20">
          <CardContent className="pt-6">
            <div className="flex items-center gap-2 mb-2">
              <IconFlame className="w-5 h-5 text-orange-400" />
              <p className="text-xs font-bold uppercase tracking-widest text-orange-400">
                Longest Session
              </p>
            </div>
            <p className="text-4xl font-bold">
              {formatMinutes(mockTodayStats.longestSessionMinutes)}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Session List */}
      <div>
        <h2 className="text-lg font-semibold mb-4">Today's Sessions</h2>
        <div className="space-y-3">
          {mockDeepWorkSessions.map((session, index) => (
            <Card
              key={session.id}
              className="hover:border-border transition-colors"
            >
              <CardContent className="py-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-violet-500/20 to-purple-500/20 flex items-center justify-center">
                      <span className="text-2xl font-bold text-violet-400">
                        #{mockDeepWorkSessions.length - index}
                      </span>
                    </div>
                    <div>
                      <p className="font-semibold">{session.projectName}</p>
                      <p className="text-sm text-muted-foreground">
                        {session.app} · {formatRelativeTime(session.startTime)}
                      </p>
                    </div>
                  </div>
                  <div className="text-right">
                    <Badge
                      variant={
                        session.durationMinutes >= 60 ? "default" : "secondary"
                      }
                      className="mb-1"
                    >
                      {session.durationMinutes >= 60
                        ? "Great session"
                        : "Good start"}
                    </Badge>
                    <p className="text-xl font-bold">
                      {formatMinutes(session.durationMinutes)}
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>

      {/* Tip */}
      <Card className="border-dashed">
        <CardContent className="py-4 text-center">
          <p className="text-sm text-muted-foreground">
            <span className="font-medium">Tip:</span> Deep work sessions are
            detected when you stay focused on one project for 25+ minutes
            without distractions.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
