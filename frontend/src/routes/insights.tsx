import { BentoDashboard } from "@/components/insights/bento-dashboard";
import { createFileRoute, Outlet, useMatchRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/insights")({
  component: ScreenTimeLayout,
});

function ScreenTimeLayout() {
  const matchRoute = useMatchRoute();

  // If we're at exactly /insights (not a child route), show the dashboard
  const isExactMatch = !!matchRoute({ to: "/insights", fuzzy: false });

  if (isExactMatch) {
    return <BentoDashboard />;
  }

  // Otherwise render the child route
  return <Outlet />;
}
