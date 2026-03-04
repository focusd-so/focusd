import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/screen-time/screentime")({
  component: ScreenTimePage,
});

function ScreenTimePage() {
  return (
    <div className="p-6">
      {/* Blank for now */}
    </div>
  );
}
