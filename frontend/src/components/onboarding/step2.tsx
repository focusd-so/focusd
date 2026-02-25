import { useCallback, useEffect, useState } from "react";
import {
  CheckAccessibility,
  RequestAccessibility,
  RequestAutomation,
  OpenSettings,
} from "../../../bindings/github.com/focusd-so/focusd/internal/native/nativeservice";

type PermissionStatus = "pending" | "granted" | "denied";

interface PermissionCard {
  id: string;
  title: string;
  description: string;
  status: PermissionStatus;
}

interface Step2Props {
  onAllGranted: (allGranted: boolean) => void;
}

export function Step2({ onAllGranted }: Step2Props) {
  const [accessibility, setAccessibility] =
    useState<PermissionStatus>("pending");
  const [systemEvents, setSystemEvents] =
    useState<PermissionStatus>("pending");
  const [browsers, setBrowsers] = useState<PermissionStatus>("pending");

  // Check accessibility on mount
  useEffect(() => {
    CheckAccessibility().then((granted) => {
      if (granted) setAccessibility("granted");
    });
  }, []);

  // Notify parent when all granted
  useEffect(() => {
    const allGranted =
      accessibility === "granted" &&
      systemEvents === "granted" &&
      browsers === "granted";
    onAllGranted(allGranted);
  }, [accessibility, systemEvents, browsers, onAllGranted]);

  const handleGrant = useCallback(
    async (id: string) => {
      switch (id) {
        case "accessibility": {
          const granted = await RequestAccessibility();
          setAccessibility(granted ? "granted" : "denied");
          break;
        }
        case "system-events": {
          const granted = await RequestAutomation("com.apple.systemevents");
          setSystemEvents(granted ? "granted" : "denied");
          break;
        }
        case "browsers": {
          const chrome = await RequestAutomation("com.google.Chrome");
          const safari = await RequestAutomation("com.apple.Safari");
          setBrowsers(chrome || safari ? "granted" : "denied");
          break;
        }
      }
    },
    []
  );

  const cards: PermissionCard[] = [
    {
      id: "accessibility",
      title: "Accessibility",
      description:
        "Lets Focusd see which app you're using so it can keep you on track.",
      status: accessibility,
    },
    {
      id: "system-events",
      title: "System Events",
      description:
        "Allows Focusd to minimize distracting apps when you're in a focus session.",
      status: systemEvents,
    },
    {
      id: "browsers",
      title: "Browsers",
      description:
        "Enables Focusd to detect distracting websites and redirect you back to work.",
      status: browsers,
    },
  ];

  return (
    <div className="flex flex-col items-center w-full max-w-lg">
      <h1
        className="text-4xl font-bold mb-2 tracking-tight"
        style={{ color: "#e0e0e0" }}
      >
        Set up permissions
      </h1>
      <p
        className="text-base mb-8 text-center leading-relaxed"
        style={{ color: "#999" }}
      >
        Focusd needs a few permissions to work properly.
      </p>

      <div className="flex flex-col gap-3 w-full">
        {cards.map((card) => (
          <div
            key={card.id}
            className="flex items-center gap-4 rounded-xl px-5 py-4"
            style={{
              background: "rgba(255,255,255,0.04)",
              border: "1px solid rgba(255,255,255,0.07)",
            }}
          >
            {/* Status dot */}
            <div
              className="shrink-0 w-2.5 h-2.5 rounded-full transition-colors duration-300"
              style={{
                background:
                  card.status === "granted"
                    ? "#4ade80"
                    : card.status === "denied"
                      ? "#f87171"
                      : "#555",
              }}
            />

            {/* Text */}
            <div className="flex-1 min-w-0">
              <div
                className="text-sm font-semibold mb-0.5"
                style={{ color: "#d4d4d4" }}
              >
                {card.title}
              </div>
              <div className="text-xs leading-snug" style={{ color: "#888" }}>
                {card.description}
              </div>
            </div>

            {/* Action */}
            {card.status === "granted" ? (
              <span
                className="text-xs font-medium shrink-0"
                style={{ color: "#4ade80" }}
              >
                Granted
              </span>
            ) : card.status === "denied" ? (
              <button
                className="text-xs font-medium shrink-0 cursor-pointer hover:underline"
                style={{
                  color: "#f87171",
                  background: "none",
                  border: "none",
                  padding: 0,
                }}
                onClick={() => OpenSettings()}
              >
                Open Settings
              </button>
            ) : (
              <button
                className="text-xs font-medium px-3 py-1.5 rounded-lg shrink-0 cursor-pointer transition-colors"
                style={{
                  background: "rgba(255,255,255,0.08)",
                  color: "#d4d4d4",
                  border: "1px solid rgba(255,255,255,0.1)",
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = "rgba(255,255,255,0.14)";
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = "rgba(255,255,255,0.08)";
                }}
                onClick={() => handleGrant(card.id)}
              >
                Grant
              </button>
            )}
          </div>
        ))}
      </div>

      {/* Privacy note */}
      <p
        className="text-xs mt-6 text-center leading-relaxed max-w-sm"
        style={{ color: "#666" }}
      >
        Your data is never stored, shared, sold, or used for training. It's
        processed through foundational AI models and immediately discarded.
      </p>
    </div>
  );
}
