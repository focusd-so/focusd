import { create } from "zustand";
import {
  GetAll,
  GetVersionHistory,
  Save,
} from "../../bindings/github.com/focusd-so/focusd/internal/settings/service";
import type { Settings } from "../../bindings/github.com/focusd-so/focusd/internal/settings/models";
import { SettingsKey } from "../../bindings/github.com/focusd-so/focusd/internal/settings/models";

interface SettingsState {
  settings: Settings[];
  customRules: string;
  idleThreshold: string;
  historyRetention: string;
  distractionAllowance: string;
  autoUpdate: boolean;
  customRulesHistory: Settings[];
  isLoading: boolean;
  error: string | null;

  fetchSettings: () => Promise<void>;
  fetchCustomRulesHistory: (limit?: number) => Promise<void>;
  updateSetting: (key: string, value: string) => Promise<void>;
  getSettingValue: (key: string) => string | undefined;
}

function parseBooleanSetting(value: string | undefined, fallback: boolean) {
  if (value == null || value === "") {
    return fallback;
  }

  return value.toLowerCase() !== "false";
}

export const useSettingsStore = create<SettingsState>()((set, get) => ({
  settings: [],
  customRules: "",
  idleThreshold: "120", // default 120
  historyRetention: "7", // default 7
  distractionAllowance: "0", // default 0 / unlimited
  autoUpdate: true,
  customRulesHistory: [],
  isLoading: false,
  error: null,

  fetchSettings: async () => {
    set({ isLoading: true, error: null });
    try {
      const settings = await GetAll();
      const customRules =
        settings?.find((s) => s.key === SettingsKey.SettingsKeyCustomRules)
          ?.value || "";
      const idleThreshold =
        settings?.find((s) => s.key === SettingsKey.SettingsKeyIdleThreshold)
          ?.value || "120";
      const historyRetention =
        settings?.find((s) => s.key === SettingsKey.SettingsKeyHistoryRetention)
          ?.value || "7";
      const distractionAllowance =
        settings?.find((s) => s.key === SettingsKey.SettingsKeyDistractionAllowance)
          ?.value || "0";
      const autoUpdate = parseBooleanSetting(
        settings?.find((s) => s.key === SettingsKey.SettingsKeyAutoUpdate)?.value,
        true
      );

      set({
        settings: settings || [],
        customRules,
        idleThreshold,
        historyRetention,
        distractionAllowance,
        autoUpdate,
        isLoading: false
      });
    } catch (error) {
      console.error("Failed to fetch settings:", error);
      set({ error: String(error), isLoading: false });
    }
  },

  fetchCustomRulesHistory: async (limit = 10) => {
    try {
      const history = await GetVersionHistory(
        SettingsKey.SettingsKeyCustomRules,
        limit
      );
      set({ customRulesHistory: history || [] });
    } catch (error) {
      console.error("Failed to fetch custom rules history:", error);
      set({ error: String(error) });
    }
  },

  updateSetting: async (key, value) => {
    try {
      await Save(key as SettingsKey, value);

      // Update local state optimistically
      if (key === SettingsKey.SettingsKeyCustomRules) {
        set({ customRules: value });
      } else if (key === SettingsKey.SettingsKeyIdleThreshold) {
        set({ idleThreshold: value });
      } else if (key === SettingsKey.SettingsKeyHistoryRetention) {
        set({ historyRetention: value });
      } else if (key === SettingsKey.SettingsKeyDistractionAllowance) {
        set({ distractionAllowance: value });
      } else if (key === SettingsKey.SettingsKeyAutoUpdate) {
        set({ autoUpdate: parseBooleanSetting(value, true) });
      }

      // Refresh to get the updated version from backend
      await get().fetchSettings();

      // Refresh history if updating custom rules
      if (key === SettingsKey.SettingsKeyCustomRules) {
        await get().fetchCustomRulesHistory();
      }
    } catch (error) {
      console.error("Failed to update setting:", error);
      set({ error: String(error) });
    }
  },

  getSettingValue: (key) => {
    return get().settings.find((s) => s.key === key)?.value;
  },
}));

// Initialize on module load
useSettingsStore.getState().fetchSettings();
