import { useMemo, useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import {
  Classification,
  type ApplicationUsage,
} from "../../../bindings/github.com/focusd-so/focusd/internal/usage/models";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { UsageItem } from "@/components/usage-item";

export const Route = createFileRoute("/screen-time/screentime")({
  component: ScreenTimePage,
});

const PAGE_SIZE = 25;

const APP_FIXTURES = [
  { id: 1, name: "Visual Studio Code", bundle_id: "com.microsoft.VSCode", icon: "VS", hostname: "" },
  { id: 2, name: "Slack", bundle_id: "com.tinyspeck.slackmacgap", icon: "SL", hostname: "" },
  { id: 3, name: "Google Chrome", bundle_id: "com.google.Chrome", icon: "CH", hostname: "chrome.google.com" },
  { id: 4, name: "Notion", bundle_id: "notion.id", icon: "NO", hostname: "" },
  { id: 5, name: "YouTube", bundle_id: "com.google.Chrome", icon: "YT", hostname: "youtube.com" },
  { id: 6, name: "Linear", bundle_id: "com.linear", icon: "LI", hostname: "" },
  { id: 7, name: "Figma", bundle_id: "com.figma.Desktop", icon: "FI", hostname: "" },
];

const CLASSIFICATION_CYCLE: Classification[] = [
  Classification.ClassificationProductive,
  Classification.ClassificationProductive,
  Classification.ClassificationNeutral,
  Classification.ClassificationDistracting,
  Classification.ClassificationSystem,
];

function getDayStartEpochDaysAgo(daysAgo: number): number {
  const now = new Date();
  now.setDate(now.getDate() - daysAgo);
  now.setHours(0, 0, 0, 0);
  return Math.floor(now.getTime() / 1000);
}

function generateMockUsageData(count = 180): ApplicationUsage[] {
  const rows: ApplicationUsage[] = [];
  const startOfToday = getDayStartEpochDaysAgo(0);
  const lookbackDays = 6;

  for (let i = 0; i < count; i += 1) {
    const app = APP_FIXTURES[i % APP_FIXTURES.length];
    const dayOffset = i % (lookbackDays + 1);
    const startsAt = startOfToday - dayOffset * 24 * 60 * 60 + (i * 29 * 60) % (23 * 60 * 60);
    const durationSeconds = 4 * 60 + ((i * 137) % (65 * 60));

    rows.push({
      id: i + 1,
      started_at: startsAt,
      ended_at: startsAt + durationSeconds,
      duration_seconds: durationSeconds,
      window_title: `${app.name} Session ${i + 1}`,
      browser_url: app.hostname ? `https://${app.hostname}` : null,
      classification: CLASSIFICATION_CYCLE[i % CLASSIFICATION_CYCLE.length],
      classification_reasoning: "",
      classification_error: null,
      classification_confidence: 0.86,
      classification_source: "custom_rules",
      detected_project: "",
      detected_communication_channel: "",
      termination_mode: "none",
      termination_reasoning: "",
      termination_mode_source: "application",
      termination_mode_error: "",
      tags: [],
      application_id: app.id,
      application: {
        id: app.id,
        name: app.name,
        executable_path: `/Applications/${app.name}.app`,
        icon: app.icon,
        hostname: app.hostname || null,
        domain: app.hostname ? app.hostname : null,
        bundle_id: app.bundle_id,
      },
      sandbox_context: "",
      sandbox_response: null,
      sandbox_logs: "",
    });
  }

  return rows.sort((a, b) => (b.started_at ?? 0) - (a.started_at ?? 0));
}

function formatDuration(seconds: number | null | undefined): string {
  if (!seconds || seconds <= 0) return "0m";
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

function ScreenTimePage() {
  const [activeTab, setActiveTab] = useState<"timeline" | "aggregation">("timeline");
  const [timelinePage, setTimelinePage] = useState(1);
  const [aggregationPage, setAggregationPage] = useState(1);

  const allUsageRows = useMemo(() => generateMockUsageData(), []);

  const timelineVisibleRows = useMemo(
    () => allUsageRows.slice(0, PAGE_SIZE * timelinePage),
    [allUsageRows, timelinePage]
  );

  type AggregatedUsageRow = {
    appKey: string;
    appName: string;
    totalDurationSeconds: number;
    launchCount: number;
    avgSessionSeconds: number;
    classifications: Set<Classification>;
  };

  const aggregatedRows = useMemo(() => {
    const grouped = new Map<string, AggregatedUsageRow>();

    allUsageRows.forEach((row) => {
      const appName = row.application?.name ?? "Unknown App";
      const appKey = row.application?.bundle_id || row.application?.hostname || `app-${row.application_id}`;
      const duration = row.duration_seconds ?? 0;

      if (!grouped.has(appKey)) {
        grouped.set(appKey, {
          appKey,
          appName,
          totalDurationSeconds: 0,
          launchCount: 0,
          avgSessionSeconds: 0,
          classifications: new Set<Classification>(),
        });
      }

      const entry = grouped.get(appKey);
      if (!entry) return;

      entry.totalDurationSeconds += duration;
      entry.launchCount += 1;
      entry.classifications.add(row.classification);
    });

    const rows = Array.from(grouped.values()).map((item) => ({
      ...item,
      avgSessionSeconds: item.launchCount > 0 ? Math.floor(item.totalDurationSeconds / item.launchCount) : 0,
    }));

    rows.sort((a, b) => b.totalDurationSeconds - a.totalDurationSeconds);

    return rows;
  }, [allUsageRows]);

  const aggregationVisibleRows = useMemo(
    () => aggregatedRows.slice(0, PAGE_SIZE * aggregationPage),
    [aggregatedRows, aggregationPage]
  );

  const canLoadMoreTimeline = timelineVisibleRows.length < allUsageRows.length;
  const canLoadMoreAggregation = aggregationVisibleRows.length < aggregatedRows.length;

  return (
    <div className="p-6 space-y-4">
      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as "timeline" | "aggregation")}>
        <TabsList>
          <TabsTrigger value="timeline">Timeline</TabsTrigger>
          <TabsTrigger value="aggregation">Aggregation</TabsTrigger>
        </TabsList>

        <TabsContent value="timeline" className="space-y-3">
          <div className="text-sm text-muted-foreground">{allUsageRows.length} sessions</div>
          <div
            className="max-h-[560px] overflow-auto rounded-md"
            onScroll={(event) => {
              const target = event.currentTarget;
              if (!canLoadMoreTimeline) return;
              const threshold = 56;
              const reachedBottom = target.scrollTop + target.clientHeight >= target.scrollHeight - threshold;
              if (reachedBottom) {
                setTimelinePage((page) => page + 1);
              }
            }}
          >
            <div className="space-y-1.5">
              {timelineVisibleRows.map((row) => (
                <UsageItem key={row.id} usage={row} />
              ))}
            </div>
            {timelineVisibleRows.length === 0 && (
              <div className="p-6 text-center text-sm text-muted-foreground">No sessions available.</div>
            )}
          </div>
        </TabsContent>

        <TabsContent value="aggregation" className="space-y-3">
          <div className="text-sm text-muted-foreground">{aggregatedRows.length} apps</div>
          <div
            className="max-h-[560px] overflow-auto rounded-md border"
            onScroll={(event) => {
              const target = event.currentTarget;
              if (!canLoadMoreAggregation) return;
              const threshold = 56;
              const reachedBottom = target.scrollTop + target.clientHeight >= target.scrollHeight - threshold;
              if (reachedBottom) {
                setAggregationPage((page) => page + 1);
              }
            }}
          >
            <Table>
              <TableHeader className="sticky top-0 bg-background">
                <TableRow>
                  <TableHead>App</TableHead>
                  <TableHead>Total Duration</TableHead>
                  <TableHead>Launch Count</TableHead>
                  <TableHead>Avg Session</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {aggregationVisibleRows.map((row) => (
                  <TableRow key={row.appKey}>
                    <TableCell className="font-medium">{row.appName}</TableCell>
                    <TableCell>{formatDuration(row.totalDurationSeconds)}</TableCell>
                    <TableCell>{row.launchCount}</TableCell>
                    <TableCell>{formatDuration(row.avgSessionSeconds)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            {aggregationVisibleRows.length === 0 && (
              <div className="p-6 text-center text-sm text-muted-foreground">No apps available.</div>
            )}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
