import { useEffect } from "react";
import { useRouterState } from "@tanstack/react-router";
import { usePageSearchStore } from "@/stores/page-search-store";

interface UsePageSearchOptions {
  enabled: boolean;
  placeholder?: string;
}

export function usePageSearch(options: UsePageSearchOptions) {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const registerPageSearch = usePageSearchStore((state) => state.registerPageSearch);
  const unregisterPageSearch = usePageSearchStore((state) => state.unregisterPageSearch);
  const query = usePageSearchStore((state) => state.queries[pathname] ?? "");
  const setQuery = usePageSearchStore((state) => state.setQuery);

  useEffect(() => {
    if (!options.enabled) {
      unregisterPageSearch(pathname);
      return;
    }

    registerPageSearch(pathname, {
      enabled: true,
      placeholder: options.placeholder,
    });

    return () => {
      unregisterPageSearch(pathname);
    };
  }, [options.enabled, options.placeholder, pathname, registerPageSearch, unregisterPageSearch]);

  return {
    query,
    setQuery: (value: string) => setQuery(pathname, value),
  };
}
