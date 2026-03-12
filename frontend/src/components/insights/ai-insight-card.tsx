import { useState, useMemo } from "react";
import {
  IconSparkles,
  IconChevronDown,
  IconChevronUp,
  IconBulb,
  IconTrophy,
  IconEye,
  IconArrowsShuffle,
  IconTarget,
} from "@tabler/icons-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import type { LLMDailySummary } from "@/../bindings/github.com/focusd-so/focusd/internal/usage/models";

interface LLMInsightCardProps {
  dailyUsageSummary: LLMDailySummary;
  isYesterday?: boolean;
}

export function LLMInsightCard({
  dailyUsageSummary,
  isYesterday = false,
}: LLMInsightCardProps) {
  const [isOpen, setIsOpen] = useState<boolean>(
    isYesterday || !!dailyUsageSummary,
  );
  const isInsufficientData =
    dailyUsageSummary?.day_vibe === "insufficient-data";

  const wins = useMemo(() => {
    if (!dailyUsageSummary?.wins) return [];
    try {
      return JSON.parse(dailyUsageSummary.wins) as string[];
    } catch (e) {
      console.error("Failed to parse wins:", e);
      return [];
    }
  }, [dailyUsageSummary?.wins]);

  const headline = dailyUsageSummary?.headline || "Daily Insight";
  const narrative = dailyUsageSummary?.narrative || "";
  const keyPattern = dailyUsageSummary?.key_pattern || "";
  const suggestion = dailyUsageSummary?.suggestion || "";
  const dayVibe = dailyUsageSummary?.day_vibe || "";

  return (
    <Card className="bg-gradient-to-br from-violet-500/10 to-purple-600/5 border-violet-500/20">
      <Collapsible open={isOpen} onOpenChange={setIsOpen}>
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <IconSparkles className="w-4 h-4 text-violet-400" />
              <span className="text-violet-400">{headline}</span>
            </CardTitle>
            <div className="flex items-center gap-2">
              {dayVibe && !isInsufficientData && (
                <span className="text-[10px] px-2 py-0.5 rounded-full bg-violet-500/20 text-violet-300 font-medium uppercase tracking-wider">
                  {dayVibe}
                </span>
              )}
              <CollapsibleTrigger asChild>
                <Button variant="ghost" size="sm" className="h-7 px-2">
                  {isOpen ? (
                    <IconChevronUp className="w-4 h-4" />
                  ) : (
                    <IconChevronDown className="w-4 h-4" />
                  )}
                </Button>
              </CollapsibleTrigger>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {isInsufficientData ? (
            <p className="text-sm text-muted-foreground leading-relaxed">
              Not enough activity was tracked. Use your computer a bit more for
              a better AI summary.
            </p>
          ) : (
            <div className="space-y-3">
              <p className="text-sm text-muted-foreground leading-relaxed">
                {narrative}
              </p>

              {/* Stat badges */}
              <div className="flex gap-3 flex-wrap">
                {dailyUsageSummary.context_switch_count > 0 && (
                  <span className="flex items-center gap-1 text-[10px] px-2 py-1 rounded-md bg-muted/50 text-muted-foreground">
                    <IconArrowsShuffle className="w-3 h-3" />
                    {dailyUsageSummary.context_switch_count} context switches
                  </span>
                )}
                {dailyUsageSummary.deep_work_minutes > 0 && (
                  <span className="flex items-center gap-1 text-[10px] px-2 py-1 rounded-md bg-emerald-500/10 text-emerald-400">
                    <IconTarget className="w-3 h-3" />
                    {dailyUsageSummary.deep_work_minutes}m deep work
                  </span>
                )}
                {dailyUsageSummary.longest_focus_minutes > 0 && (
                  <span className="flex items-center gap-1 text-[10px] px-2 py-1 rounded-md bg-blue-500/10 text-blue-400">
                    <IconTarget className="w-3 h-3" />
                    {dailyUsageSummary.longest_focus_minutes}m longest focus
                  </span>
                )}
              </div>

              <CollapsibleContent className="space-y-4">
                <div className="border-t border-violet-500/20 pt-3 space-y-4">
                  {/* Key Pattern */}
                  {keyPattern && (
                    <div className="space-y-1">
                      <p className="text-[10px] uppercase tracking-wider text-muted-foreground flex items-center gap-1">
                        <IconEye className="w-3 h-3 text-violet-400" />
                        Key Pattern
                      </p>
                      <p className="text-sm text-violet-300/90">{keyPattern}</p>
                    </div>
                  )}

                  {/* Wins */}
                  {wins.length > 0 && (
                    <div className="space-y-2">
                      <p className="text-[10px] uppercase tracking-wider text-muted-foreground flex items-center gap-1">
                        <IconTrophy className="w-3 h-3 text-amber-400" />
                        Wins
                      </p>
                      <ul className="space-y-1">
                        {wins.map((win, i) => (
                          <li
                            key={i}
                            className="text-sm text-emerald-400/90 flex items-start gap-2"
                          >
                            <span className="mt-1.5 w-1 h-1 rounded-full bg-emerald-400 shrink-0" />
                            {win}
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}

                  {/* Suggestion */}
                  {suggestion && (
                    <div className="bg-violet-500/10 rounded-lg p-3">
                      <p className="text-[10px] uppercase tracking-wider text-violet-400 mb-1 flex items-center gap-1">
                        <IconBulb className="w-3 h-3" />
                        Coach's Suggestion
                      </p>
                      <p className="text-xs text-muted-foreground italic">
                        "{suggestion}"
                      </p>
                    </div>
                  )}
                </div>
              </CollapsibleContent>
            </div>
          )}
        </CardContent>
      </Collapsible>
    </Card>
  );
}
