import { useCallback, useMemo } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  GetApplicationByID,
  GetApplicationList,
} from "../../bindings/github.com/focusd-so/focusd/internal/usage/service";
import type { Application } from "../../bindings/github.com/focusd-so/focusd/internal/usage/models";
import { queryKeys } from "@/lib/query-keys";

const APPLICATION_STALE_TIME_MS = 5 * 60_000;

function upsertApplication(
  applications: Application[] | undefined,
  nextApplication: Application,
): Application[] {
  if (!applications || applications.length === 0) {
    return [nextApplication];
  }

  let replaced = false;
  const next = applications.map((application) => {
    if (application.id !== nextApplication.id) return application;
    replaced = true;
    return nextApplication;
  });

  return replaced ? next : [nextApplication, ...next];
}

export function useApplicationList() {
  return useQuery({
    queryKey: queryKeys.applicationList,
    queryFn: () => GetApplicationList(),
    staleTime: APPLICATION_STALE_TIME_MS,
  });
}

export function useApplicationStore() {
  const queryClient = useQueryClient();
  const applicationListQuery = useApplicationList();

  const applications = applicationListQuery.data ?? [];
  const applicationsById = useMemo(() => {
    const map = new Map<number, Application>();
    for (const application of applications) {
      if (application?.id) {
        map.set(application.id, application);
      }
    }
    return map;
  }, [applications]);

  const getApplicationByID = useCallback(
    async (id: number): Promise<Application | null> => {
      if (!id) return null;

      const cachedList = queryClient.getQueryData<Application[]>(queryKeys.applicationList);
      const cached = cachedList?.find((application) => application.id === id);
      if (cached) {
        return cached;
      }

      const fetched =
        (await queryClient.fetchQuery({
          queryKey: queryKeys.applicationById(id),
          queryFn: () => GetApplicationByID(id),
          staleTime: APPLICATION_STALE_TIME_MS,
        })) ?? null;

      if (!fetched) {
        return null;
      }

      queryClient.setQueryData<Application[]>(
        queryKeys.applicationList,
        (previous) => upsertApplication(previous, fetched),
      );

      return fetched;
    },
    [queryClient],
  );

  return {
    ...applicationListQuery,
    applications,
    applicationsById,
    getApplicationByID,
  };
}
