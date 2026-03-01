import { create } from "zustand";
import { Events } from "@wailsio/runtime";
import type {
  ApplicationUsage,
  ProtectionWhitelist,
  ProtectionPause,
} from "../../bindings/github.com/focusd-so/focusd/internal/usage/models";
import { TerminationMode, GetUsageListOptions } from "../../bindings/github.com/focusd-so/focusd/internal/usage/models";
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
} from "../../bindings/github.com/focusd-so/focusd/internal/usage/service";
import type {
  DayInsights,
  ProductivityScore,
} from "../../bindings/github.com/focusd-so/focusd/internal/usage/models";
import { Duration } from "../../bindings/time/models";

export interface BlockedItem {
  usage: ApplicationUsage;
  count: number;
}

interface UsageOverview {
  ProductivityScore: number;
  ProductiveSeconds: number;
  DistractiveSeconds: number;
  SupportiveSeconds: number;
}

export interface UsagePerHourBreakdown {
  HourLabel: string;
  ProductiveSeconds: number;
  DistractiveSeconds: number;
  SupportiveSeconds: number;
}

interface DailyUsageSummary {
  headline: string;
  summary: string;
  suggestion: string;
  day_vibe: string;
  wins: string;
}

export interface DailyOverview {
  UsageOverview: UsageOverview | null;
  UsagePerHourBreakdown: UsagePerHourBreakdown[] | null;
  DailyUsageSummary: DailyUsageSummary | null;
}

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

const formatHourLabel = (hour: number): string => {
  const suffix = hour >= 12 ? "pm" : "am";
  const normalizedHour = hour % 12 === 0 ? 12 : hour % 12;
  return `${normalizedHour}${suffix}`;
};

const normalizeProductivityScore = (
  score: ProductivityScore | null | undefined
): ProductivityScore => {
  return {
    ProductiveSeconds: score?.ProductiveSeconds ?? 0,
    DistractiveSeconds: score?.DistractiveSeconds ?? 0,
    OtherSeconds: score?.OtherSeconds ?? 0,
    ProductivityScore: score?.ProductivityScore ?? 0,
  };
};

const mapDayInsightsToOverview = (
  insights: DayInsights
): DailyOverview => {
  const usageOverviewScore = normalizeProductivityScore(insights.ProductivityScore);
  const hourlyTotals = new Map<number, ProductivityScore>();

  const hourlyBreakdown = insights.ProductivityPerHourBreakdown ?? {};
  Object.entries(hourlyBreakdown).forEach(([hourKey, score]) => {
    const parsedHour = new Date(hourKey).getHours();
    const safeScore = normalizeProductivityScore(score);

    const current = hourlyTotals.get(parsedHour);
    if (!current) {
      hourlyTotals.set(parsedHour, safeScore);
      return;
    }

    hourlyTotals.set(parsedHour, {
      ProductiveSeconds: current.ProductiveSeconds + safeScore.ProductiveSeconds,
      DistractiveSeconds: current.DistractiveSeconds + safeScore.DistractiveSeconds,
      OtherSeconds: current.OtherSeconds + safeScore.OtherSeconds,
      ProductivityScore: 0,
    });
  });

  const usagePerHourBreakdown: UsagePerHourBreakdown[] = Array.from(
    { length: 24 },
    (_, hourIndex) => {
      const score = normalizeProductivityScore(hourlyTotals.get(hourIndex));

      return {
        HourLabel: formatHourLabel(hourIndex),
        ProductiveSeconds: score.ProductiveSeconds,
        DistractiveSeconds: score.DistractiveSeconds,
        SupportiveSeconds: score.OtherSeconds,
      };
    }
  );

  return {
    UsageOverview: {
      ProductivityScore: usageOverviewScore.ProductivityScore,
      ProductiveSeconds: usageOverviewScore.ProductiveSeconds,
      DistractiveSeconds: usageOverviewScore.DistractiveSeconds,
      SupportiveSeconds: usageOverviewScore.OtherSeconds,
    },
    UsagePerHourBreakdown: usagePerHourBreakdown,
    DailyUsageSummary: null,
  };
};

// ── Store interface ─────────────────────────────────────────────────────────

interface UsageState {
  // Activity tracking
  recentUsages: ApplicationUsage[];
  blockedItems: Map<string, BlockedItem>;
  isSubscribed: boolean;

  // Whitelist (allowed items)
  allowedItems: ProtectionWhitelist[];

  // Protection
  currentPause: ProtectionPause | null;
  pauseHistory: ProtectionPause[] | null;

  // Insights
  selectedDate: Date;
  overview: DailyOverview | null;
  isLoading: boolean;
  error: string | null;

  // Activity actions
  addUsage: (usage: ApplicationUsage) => void;
  initSubscription: () => void;
  fetchRecentUsages: () => Promise<void>;
  getBlockedItemsList: () => BlockedItem[];
  getActiveUsages: () => ApplicationUsage[];

  // Whitelist actions
  fetchWhitelist: () => Promise<void>;
  addToWhitelist: (
    executablePath: string,
    hostname: string,
    durationMinutes: number
  ) => Promise<void>;
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
}

// ── Store implementation ────────────────────────────────────────────────────

export const useUsageStore = create<UsageState>()((set, get) => ({
  // ── Activity state ──────────────────────────────────────────────────────
  recentUsages: [],
  blockedItems: new Map(),
  isSubscribed: false,
  allowedItems: [],

  addUsage: (usage) => {
    set((state) => {
      const filtered = state.recentUsages.filter((u) => u.id !== usage.id);
      const updated = [usage, ...filtered].slice(0, 100);

      const blocked = new Map(state.blockedItems);
      if (usage.termination_mode === TerminationMode.TerminationModeBlock) {
        const key =
          usage.application?.hostname ||
          usage.application?.bundle_id ||
          String(usage.id);
        const existing = blocked.get(key);
        blocked.set(key, {
          usage,
          count: existing ? existing.count + 1 : 1,
        });
      }

      return { recentUsages: updated, blockedItems: blocked };
    });
  },

  initSubscription: () => {
    if (get().isSubscribed) return;

    Events.On("usage:update", (event: { data: ApplicationUsage }) => {
      console.log("usage update", event.data);
      get().addUsage(event.data);
    });

    Events.On("protection:status", (event: { data: ProtectionPause }) => {
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
        TerminationMode: TerminationMode.TerminationModeBlock,
      });
      const [usages, blockedItems] = await Promise.all([
        GetUsageList(recentUsagesOptions),
        GetUsageList(blockedItemsOptions),
      ]);
      const blockedItemsMap = new Map<string, BlockedItem>();
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

  addToWhitelist: async (executablePath, hostname, durationMinutes) => {
    try {
      await Whitelist(executablePath, hostname, durationMinutes * Duration.Minute);
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
  isLoading: false,
  error: null,

  setSelectedDate: (date) => {
    set({ selectedDate: date });
  },

  fetchOverview: async (date?: Date) => {
    const targetDate = date ?? get().selectedDate;

    try {
      set({ isLoading: true, error: null });
      const insights = await GetDayInsights(targetDate);
      const overview = mapDayInsightsToOverview(insights);
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
}));

// Initialize event subscription and load recent usages on module load
useUsageStore.getState().initSubscription();
useUsageStore.getState().initProtectionStore();
useUsageStore.getState().fetchRecentUsages();
useUsageStore.getState().fetchWhitelist();

