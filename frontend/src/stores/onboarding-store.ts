import { create } from "zustand";
import { persist } from "zustand/middleware";

interface OnboardingState {
    step: number;
    completed: boolean;

    nextStep: () => void;
    goToStep: (step: number) => void;
    complete: () => void;
    reset: () => void;
}

export const useOnboardingStore = create<OnboardingState>()(
    persist(
        (set) => ({
            step: 0,
            completed: false,

            nextStep: () =>
                set((state) => ({ step: state.step + 1 })),

            goToStep: (step: number) => set({ step }),

            complete: () => set({ completed: true }),

            reset: () => set({ step: 0, completed: false }),
        }),
        {
            name: "focusd:onboarding",
        }
    )
);
