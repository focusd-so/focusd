import { createFileRoute, redirect } from "@tanstack/react-router";
import { Onboarding } from "@/components/onboarding";
import { useOnboardingStore } from "@/stores/onboarding-store";

export const Route = createFileRoute("/onboarding")({
  beforeLoad: () => {
    const { completed } = useOnboardingStore.getState();
    if (completed) {
      throw redirect({ to: "/activity" });
    }
  },
  component: Onboarding,
});
