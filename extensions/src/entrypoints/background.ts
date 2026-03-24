const NATIVE_HOST_NAME = "app.focusd.so";
const RECONNECT_BASE_DELAY_MS = 750;
const RECONNECT_MAX_DELAY_MS = 12_000;
const CONNECTION_TIMEOUT_MS = 5_000;

type NativeHostRequest = {
  type: "get_connection_info";
  application_name: string;
};

type NativeHostResponse = {
  type: "connection_info" | "error";
  ws_url?: string;
  api_key?: string;
  application_name?: string;
  version?: string;
  error?: string;
};

type ConnectionInfo = {
  wsUrl: string;
  apiKey: string;
  applicationName: string;
};

let reconnectDelay = RECONNECT_BASE_DELAY_MS;

void startBridge();

async function startBridge() {
  while (true) {
    try {
      const appName = detectApplicationName();
      const connectionInfo = await requestConnectionInfo(appName);
      await connectWebSocket(connectionInfo);
      reconnectDelay = RECONNECT_BASE_DELAY_MS;
    } catch (error) {
      console.warn("focusd bridge reconnect", error);
    }

    await sleep(reconnectDelay);
    reconnectDelay = Math.min(reconnectDelay * 2, RECONNECT_MAX_DELAY_MS);
  }
}

function requestConnectionInfo(applicationName: string): Promise<ConnectionInfo> {
  return new Promise((resolve, reject) => {
    const port = browser.runtime.connectNative(NATIVE_HOST_NAME);

    const cleanup = () => {
      port.onMessage.removeListener(onMessage);
      port.onDisconnect.removeListener(onDisconnect);
      clearTimeout(timeout);
      try {
        port.disconnect();
      } catch {
        // Ignore disconnect errors.
      }
    };

    const onMessage = (message: NativeHostResponse) => {
      cleanup();

      if (!message || message.type !== "connection_info") {
        reject(new Error(message?.error || "native host returned an invalid response"));
        return;
      }

      if (!message.ws_url || !message.api_key) {
        reject(new Error("native host response missing ws_url or api_key"));
        return;
      }

      resolve({
        wsUrl: message.ws_url,
        apiKey: message.api_key,
        applicationName: message.application_name || applicationName
      });
    };

    const onDisconnect = () => {
      const disconnectError = browser.runtime.lastError?.message;
      cleanup();
      reject(new Error(disconnectError || "native host disconnected before sending a response"));
    };

    const timeout = setTimeout(() => {
      cleanup();
      reject(new Error("native host connection timed out"));
    }, CONNECTION_TIMEOUT_MS);

    port.onMessage.addListener(onMessage);
    port.onDisconnect.addListener(onDisconnect);

    const request: NativeHostRequest = {
      type: "get_connection_info",
      application_name: applicationName
    };
    port.postMessage(request);
  });
}

function connectWebSocket(info: ConnectionInfo): Promise<void> {
  return new Promise((resolve) => {
    const wsURL = new URL(info.wsUrl);
    wsURL.searchParams.set("api_key", info.apiKey);
    wsURL.searchParams.set("application_name", info.applicationName);

    const ws = new WebSocket(wsURL.toString());

    ws.onopen = () => {
      reconnectDelay = RECONNECT_BASE_DELAY_MS;
    };

    ws.onclose = () => {
      resolve();
    };

    ws.onerror = () => {
      ws.close();
    };
  });
}

function detectApplicationName() {
  const userAgent = navigator.userAgent.toLowerCase();
  if (userAgent.includes("firefox")) {
    return "Firefox";
  }

  if (userAgent.includes("edg/")) {
    return "Microsoft Edge";
  }

  if (userAgent.includes("brave")) {
    return "Brave";
  }

  return "Chrome";
}

function sleep(durationMs: number) {
  return new Promise<void>((resolve) => {
    setTimeout(resolve, durationMs);
  });
}
