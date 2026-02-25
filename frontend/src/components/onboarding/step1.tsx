import { useEffect, useState } from "react";
import { Zap, Brain, Code } from "lucide-react";
import { PrivacyNote } from "./privacy-note";
import { OnboardingHeader } from "./onboarding-header";
import { OnboardingCard } from "./onboarding-card";

interface Step1Props {
  entered: boolean;
}

export function Step1({ entered }: Step1Props) {
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    const t = setTimeout(() => setIsVisible(true), 50);
    return () => clearTimeout(t);
  }, []);

  const features = [
    {
      icon: <Brain className="w-5 h-5 text-violet-400" />,
      title: "AI-Powered Classification",
      desc: "Automatically detects distracting apps and websites in real time.",
    },
    {
      icon: <Zap className="w-5 h-5 text-amber-400" />,
      title: "Instant Protection",
      desc: "Minimizes distractions the moment they appear — zero effort.",
    },
    {
      icon: <Code className="w-5 h-5 text-emerald-400" />,
      title: "Scriptable Rules in TypeScript",
      desc: "Write your own blocking logic — time-based, app-aware, fully programmable.",
    },
  ];

  return (
    <div className="flex flex-col items-center w-full max-w-4xl">
      <OnboardingHeader
        title="Claim your focus"
        subtitle="More intentional Mac experience starts here."
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
          {features.map((item, i) => (
            <OnboardingCard
              key={i}
              icon={item.icon}
              title={item.title}
              description={item.desc}
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
