import { create } from "zustand";
import { Events } from "@wailsio/runtime";
import { GetAccountTier, CheckoutLink } from "../../bindings/github.com/focusd-so/focusd/internal/identity/service";
import { DeviceHandshakeResponse_AccountTier } from "../../bindings/github.com/focusd-so/focusd/gen/api/v1/models";

interface AccountState {
  checkoutLink: string | null;
  isLoadingAccountTier: boolean;
  isSubscribed: boolean;

  fetchAccountTier: (retryCount?: number) => Promise<DeviceHandshakeResponse_AccountTier | null>;
  fetchCheckoutLink: (retryCount?: number) => Promise<void>;
  initSubscription: () => void;
}

let accountTierTimeout: ReturnType<typeof setTimeout> | undefined;
let checkoutLinkTimeout: ReturnType<typeof setTimeout> | undefined;

export const useAccountStore = create<AccountState>()((set, get) => ({
  checkoutLink: null,
  isLoadingAccountTier: true,
  isSubscribed: false,

  fetchAccountTier: async (retryCount = 0) => {
    if (accountTierTimeout) clearTimeout(accountTierTimeout);

    try {
      // Only set globally to loading on the first explicit attempt
      if (retryCount === 0) set({ isLoadingAccountTier: true });
      const tier = await GetAccountTier();
      set({ isLoadingAccountTier: false });
      return tier;
    } catch (error) {
      console.error(`Failed to fetch account tier (attempt ${retryCount + 1}):`, error);
      set({ isLoadingAccountTier: false });

      const delay = Math.min(2000 * Math.pow(1.5, retryCount), 30000);
      return new Promise<DeviceHandshakeResponse_AccountTier | null>((resolve) => {
        accountTierTimeout = setTimeout(() => {
          resolve(get().fetchAccountTier(retryCount + 1));
        }, delay);
      });
    }
  },

  fetchCheckoutLink: async (retryCount = 0) => {
    if (checkoutLinkTimeout) clearTimeout(checkoutLinkTimeout);

    try {
      const link = await CheckoutLink();
      set({ checkoutLink: link });
    } catch (error) {
      console.error(`Failed to fetch checkout link (attempt ${retryCount + 1}):`, error);

      const delay = Math.min(2000 * Math.pow(1.5, retryCount), 30000);
      checkoutLinkTimeout = setTimeout(() => {
        get().fetchCheckoutLink(retryCount + 1);
      }, delay);
    }
  },

  initSubscription: () => {
    if (get().isSubscribed) return;

    // Listen for auth context changes from the backend
    Events.On("authctx:changed", () => {
      window.dispatchEvent(new Event("authctx:updated"));
    });

    // Fired after a successful checkout handshake (deep-link callback)
    Events.On("authctx:updated", (event: { data: any }) => {
      window.dispatchEvent(new CustomEvent("authctx:updated", { detail: event.data }));
    });

    Events.On("window:shown", () => {
      // Wails doesn't trigger the browser's native "focus" event, so we dispatch
      // it manually so React Query's refetchOnWindowFocus can work.
      window.dispatchEvent(new Event("focus"));
    });

    set({ isSubscribed: true });
  },
}));

// Initialize on module load
useAccountStore.getState().initSubscription();
useAccountStore.getState().fetchAccountTier();
useAccountStore.getState().fetchCheckoutLink();
