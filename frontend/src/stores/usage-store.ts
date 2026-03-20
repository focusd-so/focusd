import { create } from "zustand";
import { Events } from "@wailsio/runtime";
import type {
  ApplicationUsage,
  ProtectionWhitelist,
  ProtectionPause,
  DayInsights,
  UsageAggregation,
} from "../../bindings/github.com/focusd-so/focusd/internal/usage/models";
import { EnforcementAction, GetUsageListOptions } from "../../bindings/github.com/focusd-so/focusd/internal/usage/models";
import {
  GetWhitelist,
  Whitelist,
  RemoveWhitelist,
  GetProtectionStatus,
  PauseProtection,
  ResumeProtection,
  GetPauseHistory,
  GetUsageList,
  GetDayInsights,
  GetUsageAggregation,
} from "../../bindings/github.com/focusd-so/focusd/internal/usage/service";
import { Duration } from "../../bindings/time/models";



// ── Helpers ─────────────────────────────────────────────────────────────────

export const isToday = (date: Date): boolean => {
  const now = new Date();
  return (
    date.getDate() === now.getDate() &&
    date.getMonth() === now.getMonth() &&
    date.getFullYear() === now.getFullYear()
  );
};

const getYesterday = (): Date => {
  const yesterday = new Date();
  yesterday.setDate(yesterday.getDate() - 1);
  return yesterday;
};

// ── Store interface ─────────────────────────────────────────────────────────

interface UsageState {
  // Activity tracking
  recentUsages: ApplicationUsage[];
  blockedItems: Map<string, { usage: ApplicationUsage; count: number }>;
  isSubscribed: boolean;

  // Whitelist (allowed items)
  allowedItems: ProtectionWhitelist[];

  // Protection
  currentPause: ProtectionPause | null;
  pauseHistory: ProtectionPause[] | null;

  // Insights
  selectedDate: Date;
  overview: DayInsights | null;
  isLoading: boolean;
  error: string | null;

  // ScreenTime
  screenTimeUsages: ApplicationUsage[];
  screenTimeAggregation: UsageAggregation[];
  screenTimeFilters: GetUsageListOptions;

  // Activity actions
  addUsage: (usage: ApplicationUsage) => void;
  initSubscription: () => void;
  fetchRecentUsages: () => Promise<void>;
  getBlockedItemsList: () => { usage: ApplicationUsage; count: number }[];
  getActiveUsages: () => ApplicationUsage[];

  // Whitelist actions
  fetchWhitelist: () => Promise<void>;
  addToWhitelist: (appname: string, hostname: string, durationMinutes: number) => Promise<void>;
  removeFromWhitelist: (id: number) => Promise<void>;

  // Protection actions
  initProtectionStore: () => Promise<void>;
  pauseProtection: (durationMinutes: number) => Promise<void>;
  resumeProtection: () => Promise<void>;
  getPauseHistory: (days: number) => Promise<ProtectionPause[]>;

  // Insights actions
  setSelectedDate: (date: Date) => void;
  fetchOverview: (date?: Date) => Promise<void>;
  goToPrevDay: () => void;
  goToNextDay: () => void;
  goToToday: () => void;
  // ScreenTime actions
  setScreenTimeFilters: (filters: Partial<GetUsageListOptions>) => void;
  fetchScreenTimeUsages: (append?: boolean) => Promise<void>;
  fetchScreenTimeAggregation: () => Promise<void>;
}

// ── Store implementation ────────────────────────────────────────────────────

export const useUsageStore = create<UsageState>()((set, get) => ({
  // ── Activity state ──────────────────────────────────────────────────────
  recentUsages: [],
  blockedItems: new Map(),
  isSubscribed: false,
  allowedItems: [],
  isLoading: false,
  error: null,
  screenTimeUsages: [],
  screenTimeAggregation: [],
  screenTimeFilters: new GetUsageListOptions({
    Page: 0,
    PageSize: 50,
  }),

  addUsage: (usage) => {
    set((state) => {
      const filtered = state.recentUsages.filter((u) => u.id !== usage.id);
      const updated = [usage, ...filtered].slice(0, 100);

      const blocked = new Map(state.blockedItems);
      if (usage.enforcement_action === EnforcementAction.EnforcementActionBlock) {
        const key =
          usage.application?.hostname ||
          usage.application?.bundle_id ||
          String(usage.id);
        const existing = blocked.get(key);
        let newCount = existing ? existing.count : 1;
        if (existing && existing.usage.id !== usage.id) {
          newCount = existing.count + 1;
        }

        blocked.set(key, {
          usage,
          count: newCount,
        });
      }

      return { recentUsages: updated, blockedItems: blocked };
    });
  },

  initSubscription: () => {
    if (get().isSubscribed) return;

    Events.On("usage:update", (event) => {
      if (!event.data) return;
      console.log("usage update", event.data);
      get().addUsage(event.data);
    });

    Events.On("protection:status", (event) => {
      if (!event.data) return;
      console.log("protection status update", event.data);
      set({ currentPause: event.data.id > 0 ? event.data : null });
    });

    set({ isSubscribed: true });
  },

  getBlockedItemsList: () => {
    return Array.from(get().blockedItems.values()).sort(
      (a, b) => (b.usage.started_at ?? 0) - (a.usage.started_at ?? 0)
    );
  },

  getActiveUsages: () => {
    return get().recentUsages;
  },

  fetchRecentUsages: async () => {
    try {
      const recentUsagesOptions = new GetUsageListOptions({
        Date: new Date(),
        Page: 0,
        PageSize: 100,
      });
      const blockedItemsOptions = new GetUsageListOptions({
        Date: new Date(),
        EnforcementAction: EnforcementAction.EnforcementActionBlock,
      });
      const [usages, blockedItems] = await Promise.all([
        GetUsageList(recentUsagesOptions),
        GetUsageList(blockedItemsOptions),
      ]);
      const blockedItemsMap = new Map<string, { usage: ApplicationUsage; count: number }>();
      blockedItems.forEach((usage: ApplicationUsage) => {
        const key = usage.application?.hostname || usage.application?.bundle_id || String(usage.id);
        const existing = blockedItemsMap.get(key);
        const existingStarted = existing?.usage.started_at ?? 0;
        const keepExisting = existing && existingStarted > (usage.started_at ?? 0);

        blockedItemsMap.set(key, {
          usage: keepExisting ? existing.usage : usage,
          count: existing ? existing.count + 1 : 1,
        });
      });
      set({ blockedItems: blockedItemsMap });
      set({ recentUsages: usages });
    } catch (err) {
      console.error("Failed to fetch recent usages:", err);
    }
  },

  // ── Whitelist actions ───────────────────────────────────────────────────

  fetchWhitelist: async () => {
    try {
      const items = await GetWhitelist();
      set({ allowedItems: items });
    } catch (err) {
      console.error("Failed to fetch whitelist:", err);
    }
  },

  addToWhitelist: async (appname: string, hostname: string, durationMinutes: number) => {
    try {
      await Whitelist(appname, hostname, durationMinutes * Duration.Minute);
      await get().fetchWhitelist();
    } catch (err) {
      console.error("Failed to add to whitelist:", err);
    }
  },

  removeFromWhitelist: async (id) => {
    try {
      await RemoveWhitelist(id);
      await get().fetchWhitelist();
    } catch (err) {
      console.error("Failed to remove from whitelist:", err);
    }
  },

  // ── Protection state & actions ──────────────────────────────────────────

  currentPause: null,
  pauseHistory: null,

  initProtectionStore: async () => {
    try {
      const pause = await GetProtectionStatus();
      set({ currentPause: pause.id > 0 ? pause : null });
    } catch (err) {
      console.error("Failed to get protection status:", err);
    }
  },

  pauseProtection: async (durationMinutes) => {
    try {
      const pause = await PauseProtection(durationMinutes * 60, "user manually paused");
      set({ currentPause: pause });
    } catch (err) {
      console.error("Failed to pause protection:", err);
    }
  },

  resumeProtection: async () => {
    try {
      await ResumeProtection("user manually resumed");
      set({ currentPause: null });
    } catch (err) {
      console.error("Failed to resume protection:", err);
    }
  },

  getPauseHistory: async (days) => {
    try {
      const history = await GetPauseHistory(days);
      set({ pauseHistory: history });
      return history;
    } catch (err) {
      console.error("Failed to get pause history:", err);
      return [];
    }
  },

  // ── Insights state & actions (stubbed – no backend endpoint yet) ────────

  selectedDate: getYesterday(),
  overview: null,

  setSelectedDate: (date) => {
    set({ selectedDate: date });
  },

  fetchOverview: async (date?: Date) => {
    const targetDate = date ?? get().selectedDate;

    try {
      set({ isLoading: true, error: null });
      const overview = await GetDayInsights(targetDate);
      set({ overview, isLoading: false, error: null });
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to fetch overview data";
      set({ overview: null, isLoading: false, error: message });
      throw err;
    }
  },

  goToPrevDay: () => {
    const current = get().selectedDate;
    const newDate = new Date(current);
    newDate.setDate(newDate.getDate() - 1);
    get().setSelectedDate(newDate);
  },

  goToNextDay: () => {
    const current = get().selectedDate;
    if (isToday(current)) return;

    const newDate = new Date(current);
    newDate.setDate(newDate.getDate() + 1);
    get().setSelectedDate(newDate);
  },

  goToToday: () => {
    get().setSelectedDate(new Date());
  },

  // ── ScreenTime Actions ──────────────────────────────────────────────────

  setScreenTimeFilters: (filters) => {
    set((state) => ({
      screenTimeFilters: new GetUsageListOptions({
        ...state.screenTimeFilters,
        ...filters,
      }),
    }));
  },

  fetchScreenTimeUsages: async (append = false) => {
    try {
      const options = get().screenTimeFilters;
      const usages = await GetUsageList(options);
      set((state) => ({
        screenTimeUsages: append ? [...state.screenTimeUsages, ...usages] : usages,
      }));
    } catch (err) {
      console.error("Failed to fetch screen time usages:", err);
    }
  },

  fetchScreenTimeAggregation: async () => {
    try {
      const options = get().screenTimeFilters;
      const aggregation = await GetUsageAggregation(options);
      set({ screenTimeAggregation: aggregation });
    } catch (err) {
      console.error("Failed to fetch screen time aggregation:", err);
    }
  },
}));

// Initialize event subscription and load recent usages on module load
useUsageStore.getState().initSubscription();
useUsageStore.getState().initProtectionStore();
useUsageStore.getState().fetchRecentUsages();
useUsageStore.getState().fetchWhitelist();
