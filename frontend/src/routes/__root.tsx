import { IconSearch } from "@tabler/icons-react";
import {
  Outlet,
  createRootRoute,
  redirect,
  useNavigate,
  useRouterState,
} from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { AccountStatus } from "@/components/account-status";
import { AppSidebar } from "@/components/app-sidebar";
import { Input } from "@/components/ui/input";
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { Toaster } from "@/components/ui/sonner";
import { useAppVisibilityStore } from "@/stores/app-visibility-store";
import { useOnboardingStore } from "@/stores/onboarding-store";
import { usePageSearchStore } from "@/stores/page-search-store";
import {
  EnableLoginItem,
  StartObserver,
} from "../../bindings/github.com/focusd-so/focusd/internal/native/nativeservice";

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
  const searchConfig = usePageSearchStore((state) => state.configs[pathname]);
  const searchQuery = usePageSearchStore((state) => state.queries[pathname] ?? "");
  const setSearchQuery = usePageSearchStore((state) => state.setQuery);
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const isSearchEnabled = !!searchConfig?.enabled;

  const { shouldRedirectToSmartBlocking, resetRedirectFlag } =
    useAppVisibilityStore();
  const { completed } = useOnboardingStore();

  useEffect(() => {
    if (completed) {
      StartObserver();
      EnableLoginItem();
    }
  }, [completed]);

  useEffect(() => {
    if (!isSearchEnabled) {
      setIsSearchOpen(false);
      return;
    }

    setIsSearchOpen(searchQuery.length > 0);
  }, [isSearchEnabled, pathname, searchQuery]);

  useEffect(() => {
    if (!isSearchOpen) return;
    searchInputRef.current?.focus();
  }, [isSearchOpen]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const isFindShortcut = (event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "f";
      if (!isFindShortcut || !isSearchEnabled) return;

      event.preventDefault();
      setIsSearchOpen(true);
      requestAnimationFrame(() => {
        searchInputRef.current?.focus();
        searchInputRef.current?.select();
      });
    };

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [isSearchEnabled]);

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
            {title && <h1 className="text-sm font-semibold text-muted-foreground">{title}</h1>}
          </div>
          <div className="flex items-center gap-2">
            {isSearchEnabled && (
              <div className="flex items-center">
                <button
                  type="button"
                  aria-label="Open search"
                  onClick={() => setIsSearchOpen(true)}
                  className="inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground/70 hover:text-foreground hover:bg-muted/60 transition-colors"
                >
                  <IconSearch className="h-4 w-4" />
                </button>
                <div
                  className={`overflow-hidden transition-all duration-200 ease-out ${
                    isSearchOpen ? "w-56 ml-2 opacity-100" : "w-0 ml-0 opacity-0"
                  }`}
                >
                  <div className="relative">
                    <IconSearch className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground/60" />
                    <Input
                      ref={searchInputRef}
                      type="search"
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(pathname, e.target.value)}
                      onBlur={() => {
                        if (!searchQuery.trim()) {
                          setIsSearchOpen(false);
                        }
                      }}
                      onKeyDown={(e) => {
                        if (e.key === "Escape") {
                          e.preventDefault();
                          setSearchQuery(pathname, "");
                          setIsSearchOpen(false);
                          searchInputRef.current?.blur();
                        }
                      }}
                      placeholder={searchConfig.placeholder || "Search..."}
                      className="h-8 pl-8 text-xs"
                    />
                  </div>
                </div>
              </div>
            )}
            <AccountStatus />
          </div>
        </header>
        <div className="flex flex-1 flex-col min-h-0 overflow-hidden">
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
