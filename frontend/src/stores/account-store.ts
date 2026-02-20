import { create } from "zustand";
import { Events } from "@wailsio/runtime";
import { GetAccountTier, CheckoutLink } from "../../bindings/github.com/focusd-so/focusd/internal/identity/service";
import { DeviceHandshakeResponse_AccountTier } from "../../bindings/github.com/focusd-so/focusd/gen/api/v1/models";

interface AccountState {
  accountTier: DeviceHandshakeResponse_AccountTier | null;
  checkoutLink: string | null;
  isLoading: boolean;
  isSubscribed: boolean;

  fetchAccountTier: () => Promise<void>;
  fetchCheckoutLink: () => Promise<void>;
  initSubscription: () => void;
}

export const useAccountStore = create<AccountState>()((set, get) => ({
  accountTier: null,
  checkoutLink: null,
  isLoading: true,
  isSubscribed: false,

  fetchAccountTier: async () => {
    try {
      set({ isLoading: true });
      const tier = await GetAccountTier();
      set({ accountTier: tier, isLoading: false });
    } catch (error) {
      console.error("Failed to fetch account tier:", error);
      set({ isLoading: false });
    }
  },

  fetchCheckoutLink: async () => {
    try {
      const link = await CheckoutLink();
      set({ checkoutLink: link });
    } catch (error) {
      console.error("Failed to fetch checkout link:", error);
    }
  },

  initSubscription: () => {
    if (get().isSubscribed) return;

    // Listen for auth context changes from the backend
    Events.On("authctx:changed", () => {
      get().fetchAccountTier();
    });

    set({ isSubscribed: true });
  },
}));

// Initialize on module load
useAccountStore.getState().initSubscription();
useAccountStore.getState().fetchAccountTier();
useAccountStore.getState().fetchCheckoutLink();
