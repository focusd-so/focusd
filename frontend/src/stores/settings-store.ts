import { create } from "zustand";
import {
  GetConfig,
  SaveConfig,
} from "../../bindings/github.com/focusd-so/focusd/internal/settings/service";
import { AppConfig, LLMProvider } from "../../bindings/github.com/focusd-so/focusd/internal/settings/models";

interface SettingsState {
  customRules: string;
  isLoading: boolean;
  error: string | null;

  fetchSettings: () => Promise<void>;
  updateSetting: (key: string, value: string) => Promise<void>;
}

const SETTINGS_KEY = "custom_rules";

function encodeBase64(value: string): string {
  const bytes = new TextEncoder().encode(value);
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary);
}

function decodeBase64(value: string): string {
  const binary = atob(value);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i);
  }
  return new TextDecoder().decode(bytes);
}

function toAppConfig(config: AppConfig | null): AppConfig {
  if (config) {
    return config;
  }

  return new AppConfig({
    idle_threshold_seconds: 120,
    history_retention_days: 30,
    distraction_allowance_minutes: 60,
    custom_rules_js: [],
    classification_llm_provider: LLMProvider.LLMProviderGoogle,
  });
}

export const useSettingsStore = create<SettingsState>()((set, get) => ({
  customRules: "",
  isLoading: false,
  error: null,

  fetchSettings: async () => {
    set({ isLoading: true, error: null });
    try {
      const config = await GetConfig();
      const appConfig = toAppConfig(config);

      const encodedCurrentRules = appConfig.custom_rules_js?.[0] ?? "";
      let customRules = "";

      if (encodedCurrentRules) {
        try {
          customRules = decodeBase64(encodedCurrentRules);
        } catch {
          customRules = "";
        }
      }

      set({
        customRules,
        isLoading: false,
      });
    } catch (error) {
      console.error("Failed to fetch settings:", error);
      set({ error: String(error), isLoading: false });
    }
  },

  updateSetting: async (key, value) => {
    try {
      if (key !== SETTINGS_KEY) {
        return;
      }

      const config = await GetConfig();
      const appConfig = toAppConfig(config);
      const encodedRules = encodeBase64(value);

      await SaveConfig(
        new AppConfig({
          ...appConfig,
          custom_rules_js: [encodedRules, ...(appConfig.custom_rules_js ?? [])],
        })
      );

      set({ customRules: value });

      // Refresh to get the updated version from backend
      await get().fetchSettings();
    } catch (error) {
      console.error("Failed to update setting:", error);
      set({ error: String(error) });
    }
  },
}));

// Initialize on module load
useSettingsStore.getState().fetchSettings();
