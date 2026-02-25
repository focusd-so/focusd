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
        return <Step2 onAllGranted={handlePermissionsChange} />;

      default:
        return <Step1 entered={entered} />;
    }
  };

  return (
    <div
      className="flex flex-col h-screen text-foreground p-4 select-none relative overflow-hidden"
      style={{ background: "#1c1c1c" }}
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

      {/* Animated step content */}
      <div
        className="flex-grow flex flex-col items-center justify-center text-center px-8 relative z-10"
        style={{
          opacity: animating ? 0 : 1,
          transform: animating
            ? direction === "forward"
              ? "translateY(12px)"
              : "translateY(-12px)"
            : "translateY(0)",
          filter: animating ? "blur(4px)" : "blur(0)",
          transition: animating
            ? "none"
            : "opacity 0.35s cubic-bezier(0.16,1,0.3,1), transform 0.35s cubic-bezier(0.16,1,0.3,1), filter 0.35s ease",
        }}
      >
        {renderStep()}
      </div>

      {/* Bottom bar */}
      <div className="flex items-center justify-between relative z-10">
        {/* Step indicator dots */}
        <div className="flex items-center gap-2">
          {Array.from({ length: TOTAL_STEPS }).map((_, i) => (
            <div
              key={i}
              className="h-2 rounded-full transition-all duration-300 cursor-pointer"
              onClick={() => goToStep(i)}
              style={{
                width: i === step ? 28 : 8,
                background:
                  i === step
                    ? "#d4d4d4"
                    : "var(--color-secondary)",
                opacity: i < step ? 0.4 : 1,
              }}
            />
          ))}
        </div>

        <Button
          variant="default"
          onClick={handleNext}
          disabled={isNextDisabled}
          style={{
            opacity: isNextDisabled ? 0.4 : 1,
            cursor: isNextDisabled ? "not-allowed" : "pointer",
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