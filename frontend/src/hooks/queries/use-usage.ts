import { useMemo } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import {
  GetApplicationList,
  GetDayInsights,
  GetSandboxExecutionLogs,
} from "../../../bindings/github.com/focusd-so/focusd/internal/usage/service";
import type { Event as TimelineEvent } from "../../../bindings/github.com/focusd-so/focusd/internal/timeline/models";
import { queryKeys } from "@/lib/query-keys";
import {
  mockDayInsights,
  mockRecentUsageEvents,
  mockSandboxLogEvents,
  mockUsageAggregation,
} from "@/lib/mock-data";

const useDevFallback = import.meta.env.DEV;

// useRecentUsages reads the recent timeline events that the wails-events bridge
// appends in real time. Until the backend implements a paginated GetUsageList
// over the timeline, we seed the cache with mock data in dev so the UI keeps
// working.
export function useRecentUsages() {
  const queryClient = useQueryClient();

  const query = useQuery<TimelineEvent[]>({
    queryKey: queryKeys.recentUsages,
    queryFn: async () => {
      // GetUsageList no longer exists on the backend (timeline rewrite pending).
      // Seed with mocks in dev so screens keep rendering; otherwise return [].
      const seed = useDevFallback ? mockRecentUsageEvents() : [];
      queryClient.setQueryData(queryKeys.recentUsages, seed);
      return seed;
    },
    staleTime: Infinity,
  });

  return query;
}

export function useUsageAggregation(_filters: unknown) {
  return useQuery({
    queryKey: queryKeys.usageAggregation(_filters),
    queryFn: async () => (useDevFallback ? mockUsageAggregation() : []),
    staleTime: Infinity,
  });
}

export function useDayInsights(date: Date) {
  const dateKey = date.toISOString().slice(0, 10);
  return useQuery({
    queryKey: queryKeys.dayInsights(dateKey),
    queryFn: async () => {
      try {
        const result = await GetDayInsights(date);
        if (result && hasInsightContent(result)) return result;
      } catch (err) {
        if (!useDevFallback) throw err;
        console.warn("GetDayInsights failed, falling back to mock", err);
      }
      return useDevFallback ? mockDayInsights() : null;
    },
  });
}

export function useApplicationList() {
  return useQuery({
    queryKey: queryKeys.applicationList,
    queryFn: () => GetApplicationList(),
    staleTime: 5 * 60_000,
  });
}

export function useSandboxLogs(logType: string, search: string) {
  return useQuery<TimelineEvent[]>({
    queryKey: queryKeys.sandboxLogs(logType, search),
    queryFn: async () => {
      try {
        const events = (await GetSandboxExecutionLogs(logType, search, 0, 50)).filter(
          Boolean,
        ) as TimelineEvent[];
        if (events.length > 0) return events;
      } catch (err) {
        if (!useDevFallback) throw err;
        console.warn("GetSandboxExecutionLogs failed, falling back to mock", err);
      }
      return useDevFallback ? mockSandboxLogEvents(logType, search) : [];
    },
  });
}

export function useUsingDevFallbackData(): boolean {
  return useDevFallback;
}

function hasInsightContent(result: { productivity_score?: { productive_seconds?: number; distracting_seconds?: number } }): boolean {
  const score = result.productivity_score;
  if (!score) return false;
  return (score.productive_seconds ?? 0) > 0 || (score.distracting_seconds ?? 0) > 0;
}

// useDerivedFromRecent gives consumers a memoised "blocked items" map derived
// from the recent timeline events.
export function useBlockedItems() {
  const { data: events = [] } = useRecentUsages();
  return useMemo(() => deriveBlockedItems(events), [events]);
}

export interface BlockedItemView {
  event: TimelineEvent;
  count: number;
}

import { parsePayload, type ApplicationUsagePayload } from "@/lib/timeline";

function deriveBlockedItems(events: TimelineEvent[]): BlockedItemView[] {
  const map = new Map<string, BlockedItemView>();
  for (const event of events) {
    const payload = parsePayload<ApplicationUsagePayload>(event);
    const enforced = payload?.enforcement_result?.StandardEnforcementResult;
    if (!enforced || enforced.Action !== "block") continue;
    const key = String(payload.application_id ?? event.id);
    const existing = map.get(key);
    if (existing) {
      existing.count += 1;
      // Keep the most recent event for display.
      if (event.occurred_at > existing.event.occurred_at) existing.event = event;
    } else {
      map.set(key, { event, count: 1 });
    }
  }
  return Array.from(map.values()).sort(
    (a, b) => b.event.occurred_at - a.event.occurred_at,
  );
}
