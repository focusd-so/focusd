import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  AllowApp,
  AllowGetAll,
  AllowHostname,
  AllowRemove,
  AllowURL,
} from "../../../bindings/github.com/focusd-so/focusd/internal/usage/service";
import type { Event as TimelineEvent } from "../../../bindings/github.com/focusd-so/focusd/internal/timeline/models";
import { Duration } from "../../../bindings/time/models";
import { queryKeys } from "@/lib/query-keys";
import { parsePayload, type AllowUsagePayload } from "@/lib/timeline";

// AllowedItem is the frontend-friendly view of an allow_usage timeline event.
export interface AllowedItem {
  id: number;
  app_name?: string;
  url?: string;
  hostname?: string;
  expires_at: number | null;
  occurred_at: number;
}

type AllowPayloadShape = {
  app_name?: string;
  url?: string;
  hostname?: string;
};

function eventToAllowedItem(event: TimelineEvent): AllowedItem {
  const payload = (parsePayload<AllowUsagePayload>(event) ?? {}) as AllowPayloadShape;
  return {
    id: event.id,
    app_name: payload.app_name || undefined,
    url: payload.url || undefined,
    hostname: payload.hostname || undefined,
    expires_at: event.ended_at,
    occurred_at: event.occurred_at,
  };
}

// useAllowList returns the active allow_usage events (after expiry filtering by
// the backend), normalised to AllowedItem.
export function useAllowList() {
  const query = useQuery({
    queryKey: queryKeys.allowList,
    queryFn: async () => (await AllowGetAll()).filter(Boolean) as TimelineEvent[],
    staleTime: 30_000,
  });

  const items = useMemo<AllowedItem[]>(
    () => (query.data ?? []).map(eventToAllowedItem),
    [query.data],
  );

  return { ...query, items } as const;
}

function minutesToDuration(minutes: number): number {
  // Duration values in the generated bindings are nanoseconds (Go time.Duration).
  return minutes * Number(Duration.Minute);
}

export function useAllowApp() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ appName, durationMinutes }: { appName: string; durationMinutes: number }) =>
      AllowApp(appName, minutesToDuration(durationMinutes)),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.allowList }),
  });
}

export function useAllowHostname() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ rawURL, durationMinutes }: { rawURL: string; durationMinutes: number }) =>
      AllowHostname(rawURL, minutesToDuration(durationMinutes)),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.allowList }),
  });
}

export function useAllowURL() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ rawURL, durationMinutes }: { rawURL: string; durationMinutes: number }) =>
      AllowURL(rawURL, minutesToDuration(durationMinutes)),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.allowList }),
  });
}

export function useRemoveAllow() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => AllowRemove(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.allowList }),
  });
}
