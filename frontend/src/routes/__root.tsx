import {
  Outlet,
  createRootRoute,
  useNavigate,
  useRouterState,
  redirect,
} from "@tanstack/react-router";
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { Toaster } from "@/components/ui/sonner";
import { AppSidebar } from "@/components/app-sidebar";
import { useAppVisibilityStore } from "@/stores/app-visibility-store";
import { useOnboardingStore } from "@/stores/onboarding-store";
import { useEffect } from "react";
import { StartObserver, EnableLoginItem } from "../../bindings/github.com/focusd-so/focusd/internal/native/nativeservice";
import { AccountStatus } from "@/components/account-status";

const routeTitles: Record<string, string> = {
  "/activity": "Smart Blocking",
  "/screen-time": "Overview",
  "/screen-time/screentime": "Screen Time",
  "/screen-time/trends": "Trends",
  "/screen-time/projects": "Projects",
  "/screen-time/deep-work": "Deep Work",
  "/screen-time/share": "Share",
  "/settings": "Settings",
};

function RootLayout() {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const title = routeTitles[pathname] || "";
  const navigate = useNavigate();

  const { shouldRedirectToSmartBlocking, resetRedirectFlag } =
    useAppVisibilityStore();
  const { completed } = useOnboardingStore();

  useEffect(() => {
    if (completed) {
      StartObserver();
      EnableLoginItem();
    }
  }, [completed]);

  // Handle redirect to smart blocking screen when window is reopened after timeout
  if (shouldRedirectToSmartBlocking && pathname !== "/activity") {
    queueMicrotask(() => {
      navigate({ to: "/activity" });
      resetRedirectFlag();
    });
  } else if (shouldRedirectToSmartBlocking && pathname === "/activity") {
    queueMicrotask(() => {
      resetRedirectFlag();
    });
  }

  // Render onboarding without sidebar/header
  if (pathname === "/onboarding") {
    return (
      <>
        <Toaster />
        <Outlet />
      </>
    );
  }

  return (
    <SidebarProvider>
      <AppSidebar />
      <Toaster />
      <SidebarInset className="overflow-hidden h-full">
        <header className="flex h-12 shrink-0 items-center justify-between border-b px-4 sticky top-0 bg-background z-10">
          <div className="flex items-center gap-4">
            <SidebarTrigger />
            {title && (
              <h1 className="text-sm font-semibold text-muted-foreground">
                {title}
              </h1>
            )}
          </div>
          <AccountStatus />
        </header>
        <div className="flex flex-1 flex-col h-full overflow-hidden">
          <Outlet />
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}

export const Route = createRootRoute({
  beforeLoad: ({ location }) => {
    // Skip guard when already on the onboarding page
    if (location.pathname === "/onboarding") return;

    const { completed } = useOnboardingStore.getState();
    if (!completed) {
      throw redirect({ to: "/onboarding" });
    }
  },
  component: RootLayout,
});
