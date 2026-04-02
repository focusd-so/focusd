import { create } from "zustand";

interface PageSearchConfig {
  enabled: boolean;
  placeholder?: string;
}

interface PageSearchState {
  configs: Record<string, PageSearchConfig>;
  queries: Record<string, string>;
  registerPageSearch: (path: string, config: PageSearchConfig) => void;
  unregisterPageSearch: (path: string) => void;
  setQuery: (path: string, query: string) => void;
}

export const usePageSearchStore = create<PageSearchState>()((set) => ({
  configs: {},
  queries: {},

  registerPageSearch: (path, config) => {
    set((state) => ({
      configs: {
        ...state.configs,
        [path]: config,
      },
      queries: {
        ...state.queries,
        [path]: state.queries[path] ?? "",
      },
    }));
  },

  unregisterPageSearch: (path) => {
    set((state) => {
      const nextConfigs = { ...state.configs };
      const nextQueries = { ...state.queries };

      delete nextConfigs[path];
      delete nextQueries[path];

      return {
        configs: nextConfigs,
        queries: nextQueries,
      };
    });
  },

  setQuery: (path, query) => {
    set((state) => ({
      queries: {
        ...state.queries,
        [path]: query,
      },
    }));
  },
}));
