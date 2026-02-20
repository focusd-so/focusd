import { createFileRoute } from "@tanstack/react-router";
import { CustomRules } from "@/components/custom-rules";
import { useSettingsStore } from "@/stores/settings-store";

export const Route = createFileRoute("/settings")({
  loader: () => useSettingsStore.getState().fetchSettings(),
  component: SettingsPage,
});

function SettingsPage() {
  return (
    <div className="flex flex-col h-full p-4 overflow-hidden">
      <div className="flex-1 min-h-0 w-full">
        <CustomRules />
      </div>
    </div>
  );
}
