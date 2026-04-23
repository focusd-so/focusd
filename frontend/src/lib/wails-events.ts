import { Events } from "@wailsio/runtime";
import type { QueryClient } from "@tanstack/react-query";

import { Event as TimelineEvent } from "../../bindings/github.com/focusd-so/focusd/internal/timeline/models";
import { EventType } from "./timeline";
import { queryKeys } from "./query-keys";

// bootstrapWailsEvents wires the Wails-emitted timeline events into the React
// Query cache so consumers stay in sync without polling. It is meant to be
// called exactly once at app start (before the React tree mounts) so the
// subscription lives for the full app lifetime — no useEffect required.
export function bootstrapWailsEvents(queryClient: QueryClient): () => void {
  const unsubscribers: Array<() => void> = [];

  unsubscribers.push(
    Events.On(EventType.ProtectionStatusChanged, (event) => {
      const tlEvent = toTimelineEvent(event.data);
      // The backend emits both create and update for the active pause event.
      // Update the status cache directly and let history refetch lazily.
      queryClient.setQueryData(queryKeys.protectionStatus, tlEvent ?? null);
      queryClient.invalidateQueries({ queryKey: queryKeys.pauseHistoryAll });
    }),
  );

  unsubscribers.push(
    Events.On(EventType.UsageChanged, (event) => {
      const tlEvent = toTimelineEvent(event.data);
      if (!tlEvent) return;

      queryClient.setQueryData<TimelineEvent[] | undefined>(
        queryKeys.recentUsages,
        (prev) => {
          const next = prev ? prev.filter((e) => e.id !== tlEvent.id) : [];

          if (next.length > 0 && !next[0].ended_at) {
            next[0] = TimelineEvent.createFrom({
              ...next[0],
              ended_at: tlEvent.occurred_at
            });
          }

          next.unshift(tlEvent);
          return next.slice(0, 100);
        },
      );

      // Aggregations / insights derived from usage need a soft refresh.
      queryClient.invalidateQueries({ queryKey: queryKeys.usageListAll });
      queryClient.invalidateQueries({ queryKey: queryKeys.usageAggregationAll });
      queryClient.invalidateQueries({ queryKey: queryKeys.dayInsightsAll });
      queryClient.invalidateQueries({ queryKey: queryKeys.allowList });
    }),
  );

  unsubscribers.push(
    Events.On(EventType.UserIdleChanged, () => {
      // Idle transitions also flip whatever active usage row is showing.
      queryClient.invalidateQueries({ queryKey: queryKeys.recentUsages });
    }),
  );

  return () => {
    for (const off of unsubscribers) off();
  };
}

function toTimelineEvent(data: unknown): TimelineEvent | null {
  if (!data) return null;
  // Wails delivers payloads as already-decoded objects matching the Go struct.
  return TimelineEvent.createFrom(data as object);
}
