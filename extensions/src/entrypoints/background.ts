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

type PageTitleChangedMessage = {
  type: "page_title_changed";
  payload: {
    tabId: number;
    title: string;
    windowId: number;
    url?: string;
    timestamp: string;
  };
};

type PageTitleClassifiedMessage = {
  type: "page_title_classified";
  payload: {
    tabId: number;
    title: string;
    usage: Record<string, unknown>;
  };
};

type PageTitleErrorMessage = {
  type: "page_title_error";
  payload: {
    tabId: number;
    error: string;
  };
};

type IncomingWSMessage = PageTitleClassifiedMessage | PageTitleErrorMessage;

let reconnectDelay = RECONNECT_BASE_DELAY_MS;
const latestUsageByTabId = new Map<number, Record<string, unknown>>();

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
    let stopTabTitleListener: (() => void) | null = null;

    ws.onopen = () => {
      reconnectDelay = RECONNECT_BASE_DELAY_MS;
      stopTabTitleListener = startTabTitleListener(ws);
    };

    ws.onmessage = (event) => {
      if (typeof event.data !== "string") {
        return;
      }

      let message: IncomingWSMessage;
      try {
        message = JSON.parse(event.data) as IncomingWSMessage;
      } catch {
        return;
      }

      if (message.type === "page_title_classified") {
        latestUsageByTabId.set(message.payload.tabId, message.payload.usage);
        return;
      }

      if (message.type === "page_title_error") {
        console.warn("focusd page title classification error", message.payload);
      }
    };

    ws.onclose = () => {
      stopTabTitleListener?.();
      stopTabTitleListener = null;
      resolve();
    };

    ws.onerror = () => {
      ws.close();
    };
  });
}

function startTabTitleListener(ws: WebSocket) {
  const onTabUpdated: Parameters<typeof browser.tabs.onUpdated.addListener>[0] = (
    tabId,
    changeInfo,
    tab
  ) => {
    if (!changeInfo.title || ws.readyState !== WebSocket.OPEN) {
      return;
    }

    const message: PageTitleChangedMessage = {
      type: "page_title_changed",
      payload: {
        tabId,
        title: changeInfo.title,
        windowId: tab.windowId,
        url: tab.url,
        timestamp: new Date().toISOString()
      }
    };

    try {
      ws.send(JSON.stringify(message));
    } catch {
      ws.close();
    }
  };

  browser.tabs.onUpdated.addListener(onTabUpdated);

  return () => {
    browser.tabs.onUpdated.removeListener(onTabUpdated);
  };
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
