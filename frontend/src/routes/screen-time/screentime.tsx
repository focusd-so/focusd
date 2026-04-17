import React, { useMemo, useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import {
  IconCalendar,
  IconChartBar,
  IconListSearch,
  IconShield,
  IconClock,
  IconFilter,
  IconAppWindow,
  IconWorld,
} from "@tabler/icons-react";
import { format } from "date-fns";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { UsageItem, formatDuration } from "@/components/usage-item";
import { cn } from "@/lib/utils";
import { useUsageStore } from "@/stores/usage-store";
import {
  useApplicationList,
  useRecentUsages,
  useUsageAggregation,
  useUsingDevFallbackData,
} from "@/hooks/queries/use-usage";
import {
  buildApplicationsById,
  toUsageItemView,
  type UsageItemView,
} from "@/lib/usage-view";
import { parsePayload, type ApplicationUsagePayload } from "@/lib/timeline";
import { EnforcementAction } from "../../../bindings/github.com/focusd-so/focusd/internal/usage/models";

export interface UsageAggregation {
  application: {
    id: number;
    name: string;
    icon: string | null;
    hostname: string | null;
    domain: string | null;
  };
  total_duration: number;
  usage_count: number;
}

export const Route = createFileRoute("/screen-time/screentime")({
  component: ScreenTimePage,
});

function ScreenTimePage() {
  const { screenTimeFilters, setScreenTimeFilters } = useUsageStore();
  const date = screenTimeFilters.date;
  const setDate = (d: Date) => setScreenTimeFilters({ date: d });

  const [activeTab, setActiveTab] = useState("activity");

  const { data: recentEvents = [], isLoading: usagesLoading } = useRecentUsages();
  const { data: aggregation = [], isLoading: aggLoading } = useUsageAggregation(screenTimeFilters);
  const { data: applications } = useApplicationList();
  const applicationsById = useMemo(
    () => buildApplicationsById(applications),
    [applications],
  );

  const usingFallback = useUsingDevFallbackData();
  const isLoading = usagesLoading || aggLoading;

  const filteredViews: UsageItemView[] = useMemo(() => {
    return recentEvents
      .map((event) => {
        const payload = parsePayload<ApplicationUsagePayload>(event);
        const app = payload?.application_id
          ? applicationsById.get(payload.application_id)
          : undefined;
        return toUsageItemView(event, app);
      })
      .filter((view) => {
        if (
          screenTimeFilters.classification &&
          view.classification !== screenTimeFilters.classification
        ) {
          return false;
        }
        if (
          screenTimeFilters.enforcementAction &&
          view.enforcement_action !== screenTimeFilters.enforcementAction
        ) {
          return false;
        }
        return true;
      });
  }, [recentEvents, applicationsById, screenTimeFilters]);

  const aggregationItems = aggregation as UsageAggregation[];

  return (
    <div className="flex flex-col h-full bg-background text-foreground">
      <div className="flex flex-col gap-4 p-6 border-b border-white/5 bg-white/[0.02]">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">Screen Time</h1>
            <p className="text-sm text-muted-foreground mt-1">
              Analyze your digital habits and distractions.
            </p>
          </div>

          <div className="flex items-center gap-2">
            <Popover>
              <PopoverTrigger asChild>
                <Button
                  variant="outline"
                  className={cn(
                    "justify-start text-left font-normal w-[240px]",
                    !date && "text-muted-foreground"
                  )}
                >
                  <IconCalendar className="mr-2 h-4 w-4" />
                  {date ? format(date, "PPP") : <span>Pick a date</span>}
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-auto p-0" align="end">
                <Calendar
                  mode="single"
                  selected={date}
                  onSelect={(d) => d && setDate(d)}
                  initialFocus
                />
              </PopoverContent>
            </Popover>
          </div>
        </div>

        {usingFallback && <SampleDataBanner />}

        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 bg-white/5 p-1 rounded-lg border border-white/10">
            <Select
              value={screenTimeFilters.classification || "all"}
              onValueChange={(v) =>
                setScreenTimeFilters({
                  classification: v === "all" ? undefined : v,
                })
              }
            >
              <SelectTrigger className="w-[160px] h-8 bg-transparent border-0 text-xs focus:ring-0">
                <div className="flex items-center gap-2">
                  <IconFilter className="w-3.5 h-3.5 opacity-50" />
                  <SelectValue placeholder="All Activity" />
                </div>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Activity</SelectItem>
                <SelectItem value="productive">Productive</SelectItem>
                <SelectItem value="distracting">Distracting</SelectItem>
                <SelectItem value="neutral">Neutral</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <Button
            variant="outline"
            size="sm"
            className={cn(
              "h-8 text-xs gap-2 transition-all",
              screenTimeFilters.enforcementAction === EnforcementAction.EnforcementActionBlock
                ? "bg-red-500/10 border-red-500/30 text-red-500 hover:bg-red-500/20"
                : "opacity-60"
            )}
            onClick={() =>
              setScreenTimeFilters({
                enforcementAction:
                  screenTimeFilters.enforcementAction === EnforcementAction.EnforcementActionBlock
                    ? undefined
                    : EnforcementAction.EnforcementActionBlock,
              })
            }
          >
            <IconShield className="w-3.5 h-3.5" />
            Blocked Only
          </Button>

          {isLoading && (
            <div className="flex items-center gap-2 text-[10px] text-muted-foreground animate-pulse ml-auto">
              <span className="w-1.5 h-1.5 rounded-full bg-blue-500" />
              Loading data...
            </div>
          )}
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-hidden bg-background">
        <Tabs
          value={activeTab}
          onValueChange={setActiveTab}
          className="h-full flex flex-col p-6"
        >
          <TabsList className="w-full max-w-[400px] mb-6 grid grid-cols-2 bg-white/5 border border-white/10">
            <TabsTrigger value="activity" className="gap-2 text-xs">
              <IconListSearch className="w-3.5 h-3.5" />
              Activity Feed
            </TabsTrigger>
            <TabsTrigger value="aggregation" className="gap-2 text-xs">
              <IconChartBar className="w-3.5 h-3.5" />
              Aggregation
            </TabsTrigger>
          </TabsList>

          <TabsContent value="activity" className="flex-1 min-h-0 mt-0 focus-visible:ring-0">
            <ScrollArea className="h-full pr-4 -mr-4 [&_[data-radix-scroll-area-scrollbar]]:opacity-0 hover:[&_[data-radix-scroll-area-scrollbar]]:opacity-100">
              {filteredViews.length === 0 ? (
                <EmptyState icon={<IconListSearch className="w-10 h-10" />} message="No activity found for this filter." />
              ) : (
                <div className="space-y-4">
                  {filteredViews.map((view) => (
                    <UsageItem key={view.id} usage={view} />
                  ))}
                </div>
              )}
            </ScrollArea>
          </TabsContent>

          <TabsContent value="aggregation" className="flex-1 min-h-0 mt-0 focus-visible:ring-0">
            <ScrollArea className="h-full pr-4 -mr-4 [&_[data-radix-scroll-area-scrollbar]]:opacity-0 hover:[&_[data-radix-scroll-area-scrollbar]]:opacity-100">
              {aggregationItems.length === 0 ? (
                <EmptyState icon={<IconChartBar className="w-10 h-10" />} message="No usage data to aggregate." />
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {aggregationItems.map((agg) => (
                    <AggregationCard key={agg.application.id} aggregation={agg} />
                  ))}
                </div>
              )}
            </ScrollArea>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}

function SampleDataBanner() {
  return (
    <div className="rounded-md border border-amber-500/20 bg-amber-500/10 px-3 py-2 text-[11px] text-amber-200/90">
      Showing sample data — backend endpoint pending timeline rewrite.
    </div>
  );
}

function EmptyState({ icon, message }: { icon: React.ReactNode; message: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-24 text-muted-foreground border border-dashed rounded-2xl border-white/10 bg-white/[0.01]">
      <div className="opacity-20 mb-4">{icon}</div>
      <p className="text-sm font-medium">{message}</p>
    </div>
  );
}

function AggregationCard({ aggregation }: { aggregation: UsageAggregation }) {
  const { application, total_duration, usage_count } = aggregation;
  const isWeb = !!application.hostname;

  return (
    <div className="group flex items-center gap-4 p-4 rounded-xl border border-white/10 bg-white/[0.02] hover:bg-white/[0.04] hover:border-white/20 transition-all cursor-default">
      <div className="w-12 h-12 rounded-lg flex items-center justify-center overflow-hidden shrink-0 bg-white/5 ring-1 ring-white/10 group-hover:ring-white/20 transition-all">
        {application.icon ? (
          <img
            src={
              application.icon.startsWith("data:")
                ? application.icon
                : `data:image/png;base64,${application.icon}`
            }
            alt={application.hostname || application.name}
            className="w-10 h-10 object-contain"
          />
        ) : isWeb ? (
          <IconWorld className="w-8 h-8 opacity-40 text-blue-400" />
        ) : (
          <IconAppWindow className="w-8 h-8 opacity-40 text-purple-400" />
        )}
      </div>

      <div className="flex flex-col min-w-0 flex-1">
        <h3 className="text-sm font-bold truncate leading-tight group-hover:text-foreground transition-colors">
          {application.hostname || application.name || "Unknown"}
        </h3>
        <div className="flex items-center gap-2 mt-1">
          <Badge variant="outline" className="h-5 px-1.5 text-[9px] font-bold uppercase tracking-wider bg-white/5 border-white/10 text-white/40">
            {isWeb ? "Web" : "App"}
          </Badge>
          <span className="text-white/20">·</span>
          <span className="text-[10px] text-muted-foreground font-mono">
            {usage_count} {usage_count === 1 ? 'session' : 'sessions'}
          </span>
        </div>
      </div>

      <div className="flex flex-col items-end gap-1">
        <div className="flex items-center gap-1.5 text-blue-400">
          <IconClock className="w-3.5 h-3.5 opacity-60" />
          <span className="text-sm font-bold font-mono">
            {formatDuration(total_duration)}
          </span>
        </div>
      </div>
    </div>
  );
}
