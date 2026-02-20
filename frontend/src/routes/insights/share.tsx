import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { IconDownload, IconBrandTwitter, IconCheck } from "@tabler/icons-react";
import {
  mockTodayStats,
  mockBlockedAttempts,
  formatMinutes,
} from "@/lib/mock-data";

export const Route = createFileRoute("/insights/share")({
  component: SharePage,
});

function SharePage() {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    // In real implementation, this would copy the card image
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      <div>
        <h1 className="text-2xl font-bold">Share Your Progress</h1>
        <p className="text-sm text-muted-foreground">
          Generate beautiful cards to share on social media
        </p>
      </div>

      <div className="grid grid-cols-2 gap-6">
        {/* Preview Card */}
        <div className="space-y-4">
          <h2 className="text-sm font-medium text-muted-foreground uppercase tracking-wider">
            Preview
          </h2>

          {/* The shareable card */}
          <Card className="bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 border-slate-700 overflow-hidden relative">
            <div className="absolute inset-0 bg-gradient-to-br from-blue-500/10 via-transparent to-emerald-500/10" />
            <CardContent className="pt-8 pb-6 px-8 relative">
              {/* Header */}
              <div className="flex items-center gap-2 mb-6">
                <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-emerald-500 flex items-center justify-center">
                  <span className="text-white font-bold text-sm">F</span>
                </div>
                <span className="text-white/60 text-sm font-medium">
                  focusd
                </span>
                <Badge
                  variant="outline"
                  className="ml-auto border-white/20 text-white/60 text-xs"
                >
                  Today
                </Badge>
              </div>

              {/* Main stat */}
              <div className="text-center mb-8">
                <p className="text-6xl font-bold text-white mb-2">
                  {mockTodayStats.focusScore}%
                </p>
                <p className="text-white/60">Focus Score</p>
              </div>

              {/* Stats grid */}
              <div className="grid grid-cols-3 gap-4 text-center">
                <div>
                  <p className="text-2xl font-bold text-emerald-400">
                    {formatMinutes(mockTodayStats.productiveMinutes)}
                  </p>
                  <p className="text-xs text-white/40">Productive</p>
                </div>
                <div>
                  <p className="text-2xl font-bold text-violet-400">
                    {mockTodayStats.deepWorkSessions}
                  </p>
                  <p className="text-xs text-white/40">Deep sessions</p>
                </div>
                <div>
                  <p className="text-2xl font-bold text-rose-400">
                    {mockTodayStats.blockedAttempts}
                  </p>
                  <p className="text-xs text-white/40">Blocked</p>
                </div>
              </div>

              {/* Villain */}
              <div className="mt-6 pt-6 border-t border-white/10 text-center">
                <p className="text-xs text-white/40 mb-1">
                  Top villain defeated
                </p>
                <p className="text-white font-medium">
                  {mockBlockedAttempts[0].hostname}
                </p>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Actions */}
        <div className="space-y-4">
          <h2 className="text-sm font-medium text-muted-foreground uppercase tracking-wider">
            Actions
          </h2>

          <div className="space-y-3">
            <Button
              className="w-full justify-start gap-3"
              size="lg"
              onClick={handleCopy}
            >
              {copied ? (
                <IconCheck className="w-5 h-5" />
              ) : (
                <IconDownload className="w-5 h-5" />
              )}
              {copied ? "Copied to clipboard!" : "Download as Image"}
            </Button>

            <Button
              variant="outline"
              className="w-full justify-start gap-3"
              size="lg"
            >
              <IconBrandTwitter className="w-5 h-5" />
              Share on Twitter
            </Button>
          </div>

          <Card className="border-dashed mt-6">
            <CardHeader className="pb-2">
              <CardTitle className="text-sm">More cards coming soon</CardTitle>
            </CardHeader>
            <CardContent className="text-sm text-muted-foreground">
              <ul className="space-y-1">
                <li>Weekly Wrapped</li>
                <li>Streak Card</li>
                <li>Achievement Cards</li>
                <li>Focus Receipt</li>
              </ul>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
