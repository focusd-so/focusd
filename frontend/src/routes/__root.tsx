import {
  Outlet,
  createRootRoute,
  useNavigate,
  useRouterState,
} from "@tanstack/react-router";
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { Toaster } from "@/components/ui/sonner";
import { AppSidebar } from "@/components/app-sidebar";
import { AccountStatus } from "@/components/account-status";
import { useAppVisibilityStore } from "@/stores/app-visibility-store";

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

  // Handle redirect to smart blocking screen when window is reopened after timeout
  // This runs during render when the flag is set, navigates, and resets
  if (shouldRedirectToSmartBlocking && pathname !== "/activity") {
    // Use queueMicrotask to avoid navigation during render
    queueMicrotask(() => {
      navigate({ to: "/activity" });
      resetRedirectFlag();
    });
  } else if (shouldRedirectToSmartBlocking && pathname === "/activity") {
    // Already on activity page, just reset the flag
    queueMicrotask(() => {
      resetRedirectFlag();
    });
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
        <script src="https://cdn.jsdelivr.net/npm/@polar-sh/checkout@0.1/dist/embed.global.js" defer data-auto-init></script>
        <div className="flex flex-1 flex-col h-full overflow-hidden">
          <Outlet />
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}

export const Route = createRootRoute({
  component: RootLayout,
});
