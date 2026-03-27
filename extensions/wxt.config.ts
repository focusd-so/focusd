import { defineConfig } from "wxt";

export default defineConfig({
  srcDir: "src",
  extensionApi: "webextension-polyfill",
  manifest: {
    name: "Focusd Bridge",
    description: "Connects browser activity to the local Focusd app",
    version: "0.0.1",
    permissions: ["nativeMessaging", "tabs"],
    host_permissions: ["<all_urls>", "http://127.0.0.1:50533/*", "ws://127.0.0.1:50533/*"],
    browser_specific_settings: {
      gecko: {
        id: "focusd@focusd.so"
      }
    }
  }
});
