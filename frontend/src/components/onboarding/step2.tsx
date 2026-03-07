import { useCallback, useEffect, useState } from "react";
import {
  Eye,
  MousePointer2,
  Globe,
  Settings,
  CheckCircle2,
} from "lucide-react";
import {
  CheckAccessibility,
  CheckAutomation,
  RequestAccessibility,
  RequestAutomation,
  OpenSettings,
  GetInstalledBrowsers,
} from "../../../bindings/github.com/focusd-so/focusd/internal/native/nativeservice";
import { PrivacyNote } from "./privacy-note";
import { OnboardingHeader } from "./onboarding-header";
import { OnboardingCard } from "./onboarding-card";

type PermissionStatus = "pending" | "granted" | "denied";

interface PermissionCard {
  id: string;
  title: string;
  description: string;
  status: PermissionStatus;
  icon: React.ReactNode;
}

interface BrowserEntry {
  bundleID: string;
  name: string;
  status: PermissionStatus;
}

const MANDATORY_BROWSERS = new Set([
  "com.apple.Safari",
  "com.google.Chrome",
]);

interface Step2Props {
  onAllGranted: (allGranted: boolean) => void;
  entered: boolean;
}

export function Step2({ onAllGranted, entered }: Step2Props) {
  const [isVisible, setIsVisible] = useState(false);
  const [accessibility, setAccessibility] =
    useState<PermissionStatus>("pending");
  const [systemEvents, setSystemEvents] = useState<PermissionStatus>("pending");
  const [browsers, setBrowsers] = useState<BrowserEntry[]>([]);

  useEffect(() => {
    const t = setTimeout(() => setIsVisible(true), 50);

    CheckAccessibility().then((granted) => {
      if (granted) setAccessibility("granted");
    });

    CheckAutomation("com.apple.systemevents").then((granted) => {
      console.log(granted)
      if (granted) setSystemEvents("granted");
    });

    GetInstalledBrowsers().then(async (installed) => {
      const entries: BrowserEntry[] = await Promise.all(
        installed.map(async (b) => {
          const granted = await CheckAutomation(b.bundleID);
          return {
            bundleID: b.bundleID,
            name: b.name,
            status: granted ? ("granted" as const) : ("pending" as const),
          };
        }),
      );
      setBrowsers(entries);
    });

    return () => clearTimeout(t);
  }, []);

  const mandatoryBrowsersGranted = browsers
    .filter((b) => MANDATORY_BROWSERS.has(b.bundleID))
    .every((b) => b.status === "granted");

  useEffect(() => {
    const allGranted =
      accessibility === "granted" &&
      systemEvents === "granted" &&
      mandatoryBrowsersGranted;
    onAllGranted(allGranted);
  }, [accessibility, systemEvents, mandatoryBrowsersGranted, onAllGranted]);

  const handleGrant = useCallback(async (id: string) => {
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
    }
  }, []);

  const handleBrowserGrant = useCallback(async (bundleID: string) => {
    const granted = await RequestAutomation(bundleID);
    setBrowsers((prev) =>
      prev.map((b) =>
        b.bundleID === bundleID
          ? { ...b, status: granted ? "granted" : "denied" }
          : b,
      ),
    );
  }, []);

  const handleAllowAll = useCallback(async () => {
    for (const browser of browsers) {
      if (browser.status === "granted") continue;
      const granted = await RequestAutomation(browser.bundleID);
      setBrowsers((prev) =>
        prev.map((b) =>
          b.bundleID === browser.bundleID
            ? { ...b, status: granted ? "granted" : "denied" }
            : b,
        ),
      );
    }
  }, [browsers]);

  const allBrowsersGranted =
    browsers.length > 0 && browsers.every((b) => b.status === "granted");
  const anyBrowserDenied = browsers.some((b) => b.status === "denied");
  const anyBrowserPending = browsers.some((b) => b.status === "pending");

  const cards: PermissionCard[] = [
    {
      id: "accessibility",
      title: "Screen Content",
      description:
        "Detects which application is active to help you stay focused on your work.",
      status: accessibility,
      icon: <Eye className="w-5 h-5" />,
    },
    {
      id: "system-events",
      title: "System Control",
      description:
        "Automatically minimizes distracting applications during your focus sessions.",
      status: systemEvents,
      icon: <MousePointer2 className="w-5 h-5" />,
    },
  ];

  const renderAction = (status: PermissionStatus, onGrant: () => void) => {
    if (status === "granted") {
      return (
        <div className="flex items-center gap-1.5 text-green-400 text-sm font-medium animate-in fade-in zoom-in duration-500">
          Granted
        </div>
      );
    }
    if (status === "denied") {
      return (
        <button
          className="flex items-center gap-2 px-4 py-2 rounded-xl bg-zinc-800 hover:bg-zinc-700 active:scale-95 text-red-400 text-sm font-medium transition-all duration-300 border border-red-500/20"
          onClick={() => OpenSettings()}
        >
          <Settings className="w-4 h-4" />
          Fix in Settings
        </button>
      );
    }
    return (
      <button
        className="px-4 py-1.5 rounded-lg bg-white hover:bg-zinc-200 active:scale-95 text-black text-xs font-bold shadow-lg shadow-white/5 transition-all duration-300"
        onClick={onGrant}
      >
        Allow
      </button>
    );
  };

  return (
    <div className="flex flex-col items-center w-full max-w-4xl">
      <OnboardingHeader
        title="Perfect your workspace"
        subtitle="Grant a few permissions so Focusd can protect your focus."
        entered={entered}
      />

      <div className="w-full max-w-lg flex flex-col items-center">
        <div
          className="flex flex-col gap-4 w-full mb-10"
          style={{
            opacity: isVisible ? 1 : 0,
            transform: isVisible ? "translateY(0)" : "translateY(8px)",
            transition: "opacity 0.7s ease 0.4s, transform 0.7s ease 0.4s",
          }}
        >
          {cards.map((card) => (
            <OnboardingCard
              key={card.id}
              status={card.status}
              icon={
                card.status === "granted" ? (
                  <CheckCircle2 className="w-5 h-5 animate-in zoom-in duration-500" />
                ) : (
                  card.icon
                )
              }
              title={card.title}
              description={card.description}
              action={renderAction(card.status, () => handleGrant(card.id))}
            />
          ))}

          {browsers.length > 0 && (
            <div className="flex flex-col gap-2 mt-2">
              <div className="flex flex-col gap-1 mb-1">
                <div className="flex items-center justify-between px-4">
                  <div className="flex items-center gap-2">
                    <Globe className="w-4 h-4 text-zinc-400" />
                    <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">
                      Web Browsing
                    </span>
                  </div>
                  {allBrowsersGranted ? (
                    <div className="flex items-center gap-1.5 text-green-400 text-xs font-medium animate-in fade-in zoom-in duration-500">
                      <CheckCircle2 className="w-3.5 h-3.5" />
                      All Granted
                    </div>
                  ) : anyBrowserPending ? (
                    <button
                      className="px-3 py-1 rounded-lg bg-white hover:bg-zinc-200 active:scale-95 text-black text-xs font-bold shadow-lg shadow-white/5 transition-all duration-300"
                      onClick={handleAllowAll}
                    >
                      Allow All
                    </button>
                  ) : anyBrowserDenied ? (
                    <button
                      className="flex items-center gap-1.5 px-3 py-1 rounded-lg bg-zinc-800 hover:bg-zinc-700 active:scale-95 text-red-400 text-xs font-medium transition-all duration-300 border border-red-500/20"
                      onClick={() => OpenSettings()}
                    >
                      <Settings className="w-3.5 h-3.5" />
                      Fix in Settings
                    </button>
                  ) : null}
                </div>
                <p className="text-xs text-zinc-500 leading-snug">
                  Allow Focusd to monitor and block distracting websites in your
                  browsers.
                </p>
              </div>

              <div
                className="rounded-xl overflow-hidden"
                style={{
                  background: "rgba(255, 255, 255, 0.03)",
                  border: "1px solid rgba(255, 255, 255, 0.08)",
                  backdropFilter: "blur(12px)",
                }}
              >
                {browsers.map((browser, i) => {
                  const isMandatory = MANDATORY_BROWSERS.has(browser.bundleID);
                  return (
                    <div
                      key={browser.bundleID}
                      className={`flex items-center justify-between py-2.5 px-4 ${i > 0 ? "border-t border-white/5" : ""}`}
                    >
                      <div className="flex items-center gap-2.5">
                        {browser.status === "granted" ? (
                          <CheckCircle2 className="w-4 h-4 text-green-400 shrink-0 animate-in zoom-in duration-500" />
                        ) : (
                          <Globe className="w-4 h-4 text-zinc-500 shrink-0" />
                        )}
                        <span className="text-sm text-zinc-200">
                          {browser.name}
                        </span>
                        {isMandatory && browser.status !== "granted" && (
                          <span className="text-[10px] font-medium text-zinc-500 uppercase tracking-wider">
                            Required
                          </span>
                        )}
                      </div>
                      {browser.status === "granted" ? (
                        <span className="text-xs text-green-400 font-medium">
                          Granted
                        </span>
                      ) : browser.status === "denied" ? (
                        <button
                          className="flex items-center gap-1.5 px-3 py-1 rounded-lg bg-zinc-800 hover:bg-zinc-700 active:scale-95 text-red-400 text-xs font-medium transition-all duration-300 border border-red-500/20"
                          onClick={() => OpenSettings()}
                        >
                          <Settings className="w-3.5 h-3.5" />
                          Fix
                        </button>
                      ) : isMandatory ? (
                        <button
                          className="px-3 py-1 rounded-lg bg-white hover:bg-zinc-200 active:scale-95 text-black text-xs font-bold shadow-lg shadow-white/5 transition-all duration-300"
                          onClick={() => handleBrowserGrant(browser.bundleID)}
                        >
                          Allow
                        </button>
                      ) : (
                        <button
                          className="px-3 py-1 rounded-lg bg-white/10 hover:bg-white/15 active:scale-95 text-zinc-200 text-xs font-medium transition-all duration-300"
                          onClick={() => handleBrowserGrant(browser.bundleID)}
                        >
                          Allow
                        </button>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          )}
        </div>

        <PrivacyNote
          style={{
            opacity: isVisible ? 1 : 0,
            transform: isVisible ? "translateY(0)" : "translateY(8px)",
            transition: "opacity 0.7s ease 0.55s, transform 0.7s ease 0.55s",
          }}
        />
      </div>
    </div>
  );
}
