import { createFileRoute, Outlet, useMatchRoute } from "@tanstack/react-router";
import { BentoDashboard } from "@/components/insights/bento-dashboard";

export const Route = createFileRoute("/screen-time")({
  component: ScreenTimeLayout,
});

function ScreenTimeLayout() {
  const matchRoute = useMatchRoute();

  // If we're at exactly /screen-time (not a child route), show the dashboard
  const isExactMatch = !!matchRoute({ to: "/screen-time", fuzzy: false });

  if (isExactMatch) {
    return <BentoDashboard />;
  }

  // Otherwise render the child route
  return <Outlet />;
}
