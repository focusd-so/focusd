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
  RequestAccessibility,
  RequestAutomation,
  OpenSettings,
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

interface Step2Props {
  onAllGranted: (allGranted: boolean) => void;
  entered: boolean;
}

export function Step2({ onAllGranted, entered }: Step2Props) {
  const [isVisible, setIsVisible] = useState(false);
  const [accessibility, setAccessibility] =
    useState<PermissionStatus>("pending");
  const [systemEvents, setSystemEvents] =
    useState<PermissionStatus>("pending");
  const [browsers, setBrowsers] = useState<PermissionStatus>("pending");

  // Check accessibility on mount
  useEffect(() => {
    const t = setTimeout(() => setIsVisible(true), 50);
    CheckAccessibility().then((granted) => {
      if (granted) setAccessibility("granted");
    });
    return () => clearTimeout(t);
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
    {
      id: "browsers",
      title: "Web Browsing",
      description:
        "Detects distracting websites in your browser and redirects you to productivity.",
      status: browsers,
      icon: <Globe className="w-5 h-5" />,
    },
  ];

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
              action={
                card.status === "granted" ? (
                  <div className="flex items-center gap-1.5 text-green-400 text-sm font-medium animate-in fade-in zoom-in duration-500">
                    Granted
                  </div>
                ) : card.status === "denied" ? (
                  <button
                    className="flex items-center gap-2 px-4 py-2 rounded-xl bg-zinc-800 hover:bg-zinc-700 active:scale-95 text-red-400 text-sm font-medium transition-all duration-300 border border-red-500/20"
                    onClick={() => OpenSettings()}
                  >
                    <Settings className="w-4 h-4" />
                    Fix in Settings
                  </button>
                ) : (
                  <button
                    className="px-4 py-1.5 rounded-lg bg-white hover:bg-zinc-200 active:scale-95 text-black text-xs font-bold shadow-lg shadow-white/5 transition-all duration-300"
                    onClick={() => handleGrant(card.id)}
                  >
                    Allow
                  </button>
                )
              }
            />
          ))}
        </div>

        {/* Privacy note */}
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
