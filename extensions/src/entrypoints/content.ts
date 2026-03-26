const OVERLAY_ID = "focusd-site-block-overlay";
const BLOCKED_EVENTS: Array<keyof WindowEventMap> = [
  "click",
  "dblclick",
  "mousedown",
  "mouseup",
  "keydown",
  "keypress",
  "keyup",
  "submit",
  "contextmenu",
  "dragstart",
  "touchstart"
];

type TabBlockPayload = {
  title: string;
  classificationReasoning: string;
  tags: string[];
};

type TabControlMessage = { type: "focusd:block"; payload: TabBlockPayload } | { type: "focusd:unblock" };

let isBlocked = false;
let overlayElement: HTMLDivElement | null = null;
let hadInlineOverflow = false;
let previousOverflow = "";
let overlayTitleElement: HTMLParagraphElement | null = null;
let overlayReasonElement: HTMLParagraphElement | null = null;
let overlayTagsElement: HTMLParagraphElement | null = null;

export default defineContentScript({
  matches: ["<all_urls>"],
  runAt: "document_start",
  main() {
    browser.runtime.onMessage.addListener((message: unknown) => {
      if (!isTabControlMessage(message)) {
        return;
      }

      if (message.type === "focusd:block") {
        applyBlock(message.payload);
        return;
      }

      clearBlock();
    });
  }
});

function isTabControlMessage(message: unknown): message is TabControlMessage {
  if (!message || typeof message !== "object") {
    return false;
  }

  const maybeType = (message as { type?: unknown }).type;
  if (maybeType === "focusd:unblock") {
    return true;
  }

  if (maybeType !== "focusd:block") {
    return false;
  }

  const payload = (message as { payload?: unknown }).payload;
  if (!payload || typeof payload !== "object") {
    return false;
  }

  const maybePayload = payload as Partial<TabBlockPayload>;
  return (
    typeof maybePayload.title === "string" &&
    typeof maybePayload.classificationReasoning === "string" &&
    Array.isArray(maybePayload.tags)
  );
}

function applyBlock(payload: TabBlockPayload) {
  if (isBlocked) {
    mountOverlay(payload);
    pauseAllMedia();
    return;
  }

  isBlocked = true;
  lockScrolling();
  pauseAllMedia();
  mountOverlay(payload);

  for (const eventName of BLOCKED_EVENTS) {
    window.addEventListener(eventName, preventInteraction, { capture: true, passive: false });
  }

  document.addEventListener("play", handlePlayEvent, true);
}

function clearBlock() {
  if (!isBlocked) {
    return;
  }

  isBlocked = false;

  for (const eventName of BLOCKED_EVENTS) {
    window.removeEventListener(eventName, preventInteraction, true);
  }

  document.removeEventListener("play", handlePlayEvent, true);
  restoreScrolling();
  unmountOverlay();
}

function preventInteraction(event: Event) {
  if (!isBlocked) {
    return;
  }

  event.preventDefault();
  event.stopImmediatePropagation();
}

function handlePlayEvent(event: Event) {
  if (!isBlocked) {
    return;
  }

  const target = event.target;
  if (!(target instanceof HTMLMediaElement)) {
    return;
  }

  target.pause();
}

function pauseAllMedia() {
  const mediaElements = document.querySelectorAll("video, audio");
  for (const mediaElement of mediaElements) {
    if (!(mediaElement instanceof HTMLMediaElement)) {
      continue;
    }

    mediaElement.pause();
  }
}

function lockScrolling() {
  const root = document.documentElement;
  if (!root) {
    return;
  }

  hadInlineOverflow = root.style.overflow !== "";
  previousOverflow = root.style.overflow;
  root.style.overflow = "hidden";
}

function restoreScrolling() {
  const root = document.documentElement;
  if (!root) {
    return;
  }

  if (hadInlineOverflow) {
    root.style.overflow = previousOverflow;
    return;
  }

  root.style.removeProperty("overflow");
}

function mountOverlay(payload: TabBlockPayload) {
  if (overlayElement?.isConnected) {
    updateOverlayContent(payload);
    return;
  }

  const root = document.documentElement;
  if (!root) {
    return;
  }

  const overlay = document.createElement("div");
  overlay.id = OVERLAY_ID;
  overlay.setAttribute("aria-live", "polite");
  overlay.style.position = "fixed";
  overlay.style.inset = "0";
  overlay.style.zIndex = "2147483647";
  overlay.style.background = "rgba(255, 255, 255, 0.98)";
  overlay.style.display = "flex";
  overlay.style.alignItems = "center";
  overlay.style.justifyContent = "center";
  overlay.style.padding = "24px";
  overlay.style.pointerEvents = "auto";

  const panel = document.createElement("div");
  panel.style.maxWidth = "620px";
  panel.style.textAlign = "left";
  panel.style.color = "#111827";
  panel.style.fontFamily = '-apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif';
  panel.style.background = "#ffffff";
  panel.style.border = "1px solid #e5e7eb";
  panel.style.borderRadius = "12px";
  panel.style.padding = "20px";
  panel.style.boxShadow = "0 12px 32px rgba(17, 24, 39, 0.14)";

  const heading = document.createElement("h1");
  heading.textContent = "Focusd paused this page";
  heading.style.margin = "0 0 10px 0";
  heading.style.fontSize = "28px";
  heading.style.fontWeight = "700";
  heading.style.textAlign = "center";

  const body = document.createElement("p");
  body.textContent = "This tab is temporarily blocked. It will unlock when your session resumes.";
  body.style.margin = "0 0 14px 0";
  body.style.fontSize = "16px";
  body.style.lineHeight = "1.5";
  body.style.textAlign = "center";

  const title = document.createElement("p");
  title.style.margin = "0 0 8px 0";
  title.style.fontSize = "14px";
  title.style.lineHeight = "1.45";
  title.style.color = "#374151";

  const reasoning = document.createElement("p");
  reasoning.style.margin = "0 0 8px 0";
  reasoning.style.fontSize = "14px";
  reasoning.style.lineHeight = "1.45";
  reasoning.style.color = "#374151";

  const tags = document.createElement("p");
  tags.style.margin = "0";
  tags.style.fontSize = "14px";
  tags.style.lineHeight = "1.45";
  tags.style.color = "#1f2937";

  panel.append(heading, body, title, reasoning, tags);
  overlay.append(panel);
  root.append(overlay);

  overlayTitleElement = title;
  overlayReasonElement = reasoning;
  overlayTagsElement = tags;
  overlayElement = overlay;
  updateOverlayContent(payload);
}

function unmountOverlay() {
  overlayElement?.remove();
  overlayElement = null;
  overlayTitleElement = null;
  overlayReasonElement = null;
  overlayTagsElement = null;
}

function updateOverlayContent(payload: TabBlockPayload) {
  if (overlayTitleElement) {
    overlayTitleElement.textContent = `Page title: ${payload.title || "Unknown"}`;
  }

  if (overlayReasonElement) {
    overlayReasonElement.textContent = payload.classificationReasoning
      ? `Reasoning: ${payload.classificationReasoning}`
      : "Reasoning: unavailable";
  }

  if (overlayTagsElement) {
    overlayTagsElement.textContent = payload.tags.length ? `Tags: ${payload.tags.join(", ")}` : "Tags: none";
  }
}
