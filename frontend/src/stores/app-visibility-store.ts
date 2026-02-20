import { create } from "zustand";
import { Events } from "@wailsio/runtime";

const SMART_BLOCKING_TIMEOUT_MS = 5 * 60 * 1000; // 5 minutes

interface AppVisibilityState {
  hiddenAt: number | null;
  isSubscribed: boolean;
  shouldRedirectToSmartBlocking: boolean;

  initSubscription: () => void;
  resetRedirectFlag: () => void;
}

export const useAppVisibilityStore = create<AppVisibilityState>()((set, get) => ({
  hiddenAt: null,
  isSubscribed: false,
  shouldRedirectToSmartBlocking: false,

  initSubscription: () => {
    if (get().isSubscribed) return;

    // Listen for window hidden event
    Events.On("window:hidden", () => {
      set({ hiddenAt: Date.now() });
    });

    // Listen for window shown event
    Events.On("window:shown", () => {
      const { hiddenAt } = get();

      if (hiddenAt !== null) {
        const elapsed = Date.now() - hiddenAt;

        if (elapsed >= SMART_BLOCKING_TIMEOUT_MS) {
          set({ shouldRedirectToSmartBlocking: true });
        }
      }

      // Reset hiddenAt after processing
      set({ hiddenAt: null });
    });

    set({ isSubscribed: true });
  },

  resetRedirectFlag: () => {
    set({ shouldRedirectToSmartBlocking: false });
  },
}));

// Initialize subscription when module loads
useAppVisibilityStore.getState().initSubscription();
