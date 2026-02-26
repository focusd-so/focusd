import { useOnboardingStore } from "@/stores/onboarding-store";
import { Button } from "@/components/ui/button";
import { useState } from "react";

export function DevSettings() {
  const reset = useOnboardingStore((s) => s.reset);
  const [resetDone, setResetDone] = useState(false);

  const handleResetOnboarding = () => {
    reset();
    setResetDone(true);
    setTimeout(() => setResetDone(false), 2000);
  };

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-sm font-semibold mb-1">Onboarding</h3>
        <p className="text-xs text-muted-foreground mb-3">
          Reset the onboarding flag so it shows again on next launch.
        </p>
        <Button
          variant="outline"
          size="sm"
          onClick={handleResetOnboarding}
        >
          {resetDone ? "✓ Reset" : "Reset Onboarding"}
        </Button>
      </div>
    </div>
  );
}
