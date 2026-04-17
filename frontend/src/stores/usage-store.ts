import { create } from "zustand";

// usage-store now only holds pure client-side UI state. All backend-derived
// state (protection status, allow list, pause history, recent usages, screen
// time aggregates, day insights) lives in the React Query cache and is fed by
// the wails-events bridge. This file used to drive the entire data layer; the
// React Query hooks under `frontend/src/hooks/queries/` are the new source of
// truth.

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

export interface ScreenTimeFilters {
  date: Date;
  enforcementAction?: string;
  classification?: string;
  applicationId?: number;
  page: number;
  pageSize: number;
}

interface UsageState {
  // Insights date selector
  selectedDate: Date;
  setSelectedDate: (date: Date) => void;
  goToPrevDay: () => void;
  goToNextDay: () => void;
  goToToday: () => void;

  // Screen Time filters
  screenTimeFilters: ScreenTimeFilters;
  setScreenTimeFilters: (filters: Partial<ScreenTimeFilters>) => void;
}

export const useUsageStore = create<UsageState>()((set, get) => ({
  selectedDate: getYesterday(),
  setSelectedDate: (date) => set({ selectedDate: date }),
  goToPrevDay: () => {
    const next = new Date(get().selectedDate);
    next.setDate(next.getDate() - 1);
    set({ selectedDate: next });
  },
  goToNextDay: () => {
    const current = get().selectedDate;
    if (isToday(current)) return;
    const next = new Date(current);
    next.setDate(next.getDate() + 1);
    set({ selectedDate: next });
  },
  goToToday: () => set({ selectedDate: new Date() }),

  screenTimeFilters: {
    date: new Date(),
    page: 0,
    pageSize: 50,
  },
  setScreenTimeFilters: (patch) =>
    set((state) => ({
      screenTimeFilters: { ...state.screenTimeFilters, ...patch },
    })),
}));
