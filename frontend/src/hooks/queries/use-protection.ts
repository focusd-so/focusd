import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  PauseGetHistory,
  ProtectionGetStatus,
  ProtectionPause,
  ProtectionResume,
} from "../../../bindings/github.com/focusd-so/focusd/internal/usage/service";
import type { Event as TimelineEvent } from "../../../bindings/github.com/focusd-so/focusd/internal/timeline/models";
import { queryKeys } from "@/lib/query-keys";
import { isEventActive } from "@/lib/timeline";

// useProtectionStatus returns the active protection-pause event when one is in
// flight, otherwise null. The bridge keeps this cache in sync via
// `protection_status_changed` events.
export function useProtectionStatus() {
  return useQuery({
    queryKey: queryKeys.protectionStatus,
    queryFn: async () => (await ProtectionGetStatus()) ?? null,
    staleTime: 60_000,
    select: (event) => event ?? null,
  });
}

// useIsProtectionPaused is a convenience selector returning true when a pause
// event is in flight and not yet expired.
export function useIsProtectionPaused(): boolean {
  const { data } = useProtectionStatus();
  return isEventActive(data ?? null);
}

export function usePauseHistory(days: number, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.pauseHistory(days),
    queryFn: async () => (await PauseGetHistory(days)).filter(Boolean) as TimelineEvent[],
    enabled: options?.enabled ?? true,
    staleTime: 60_000,
  });
}

export function usePauseProtection() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ durationSeconds, reason }: { durationSeconds: number; reason: string }) =>
      ProtectionPause(durationSeconds, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.protectionStatus });
      queryClient.invalidateQueries({ queryKey: queryKeys.pauseHistoryAll });
    },
  });
}

export function useResumeProtection() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (reason: string) => ProtectionResume(reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.protectionStatus });
      queryClient.invalidateQueries({ queryKey: queryKeys.pauseHistoryAll });
    },
  });
}
