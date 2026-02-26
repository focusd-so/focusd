import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { Button } from "@/components/ui/button";
import { useOnboardingStore } from "@/stores/onboarding-store";
import { Step1 } from "@/components/onboarding/step1";
import { Step2 } from "@/components/onboarding/step2";

const TOTAL_STEPS = 2;

export function Onboarding() {
  const navigate = useNavigate();
  const { step, nextStep, goToStep, complete } = useOnboardingStore();

  // Track the previous step to determine animation direction
  const [displayStep, setDisplayStep] = useState(step);
  const [animating, setAnimating] = useState(false);
  const [direction, setDirection] = useState<"forward" | "back">("forward");
  const [entered, setEntered] = useState(false);

  // Step 2 blocks progression until all permissions are granted
  const [permissionsGranted, setPermissionsGranted] = useState(false);

  // Entrance animation on mount
  useEffect(() => {
    const t = setTimeout(() => setEntered(true), 50);
    return () => clearTimeout(t);
  }, []);

  useEffect(() => {
    if (step !== displayStep) {
      setDirection(step > displayStep ? "forward" : "back");
      setAnimating(true);
      const t = setTimeout(() => {
        setDisplayStep(step);
        setAnimating(false);
      }, 280);
      return () => clearTimeout(t);
    }
  }, [step, displayStep]);

  const handleNext = () => {
    if (step < TOTAL_STEPS - 1) {
      nextStep();
    } else {
      complete();
      navigate({ to: "/activity" });
    }
  };

  const handlePermissionsChange = useCallback((allGranted: boolean) => {
    setPermissionsGranted(allGranted);
  }, []);

  const isLastStep = step === TOTAL_STEPS - 1;

  // Disable Next on step 2 until permissions are granted
  const isNextDisabled = displayStep === 1 && !permissionsGranted;

  const renderStep = () => {
    switch (displayStep) {
      case 0:
        return <Step1 entered={entered} />;
      case 1:
        return <Step2 onAllGranted={handlePermissionsChange} entered={entered} />;

      default:
        return <Step1 entered={entered} />;
    }
  };

  return (
    <div
      className="flex flex-col h-screen text-foreground select-none relative overflow-hidden"
      style={{
        background:
          "radial-gradient(circle at 50% 120%, rgba(30, 30, 45, 1) 0%, rgba(9, 9, 11, 1) 100%)",
      }}
    >
      <style>{`
        @keyframes shimmer {
          0% { background-position: 100% center; }
          100% { background-position: 0% center; }
        }
      `}</style>

      {/* Draggable title-bar area */}
      <div
        className="absolute top-0 left-0 w-full h-10 z-50"
        style={{ WebkitAppRegion: "drag" } as React.CSSProperties}
      />

      {/* step content */}
      <div
        className={`flex-grow flex flex-col items-center justify-center text-center px-12 relative z-10 transition-all duration-700 ease-in-out ${animating
          ? "opacity-0 invisible blur-lg " +
          (direction === "forward" ? "translate-y-3" : "-translate-y-3")
          : "opacity-100 visible blur-0 translate-y-0"
          }`}
      >
        {renderStep()}
      </div>

      {/* Bottom bar */}
      <div className="flex items-center justify-between relative z-20 mt-auto w-full px-10 pb-4">
        {/* Step indicator dots */}
        <div className="flex items-center gap-2.5">
          {Array.from({ length: TOTAL_STEPS }).map((_, i) => (
            <div
              key={i}
              className="h-1.5 rounded-full transition-all duration-700 ease-out cursor-pointer"
              onClick={() => goToStep(i)}
              style={{
                width: i === step ? 32 : 6,
                background: i === step ? "white" : "rgba(255,255,255,0.15)",
                opacity: i < step ? 0.4 : 1,
              }}
            />
          ))}
        </div>

        <Button
          variant="default"
          onClick={handleNext}
          disabled={isNextDisabled}
          className="h-11 px-8 rounded-xl font-bold text-sm tracking-tight transition-all duration-500 ease-in-out"
          style={{
            background: isNextDisabled ? "rgba(255,255,255,0.06)" : "white",
            color: isNextDisabled ? "rgba(255,255,255,0.2)" : "black",
            opacity: isNextDisabled ? 0.5 : 1,
            cursor: isNextDisabled ? "not-allowed" : "pointer",
            border: "none",
          }}
        >
          {step === 0
            ? "Get Started"
            : isLastStep
              ? "Start 7-day trial"
              : "Next"}
        </Button>
      </div>
    </div>
  );
}