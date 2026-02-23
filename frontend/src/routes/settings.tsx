import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { CustomRules } from "@/components/custom-rules";
import { useSettingsStore } from "@/stores/settings-store";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { GeneralSettings } from "@/components/settings/general-settings";
import { z } from "zod";

const settingsSearchSchema = z.object({
  tab: z.enum(["general", "rules"]).optional().catch("general"),
});

export const Route = createFileRoute("/settings")({
  validateSearch: (search) => settingsSearchSchema.parse(search),
  loader: () => useSettingsStore.getState().fetchSettings(),
  component: SettingsPage,
});

function SettingsPage() {
  const { tab } = Route.useSearch();
  const navigate = useNavigate();

  const handleTabChange = (value: string) => {
    navigate({ search: { tab: value as any } });
  };

  return (
    <div className="flex flex-col h-full p-4 overflow-hidden space-y-4">
      <Tabs
        value={tab || "general"}
        onValueChange={handleTabChange}
        className="flex-1 flex flex-col min-h-0"
      >
        <TabsList className="mb-2">
          <TabsTrigger value="general">General</TabsTrigger>
          <TabsTrigger value="rules">Custom Rules</TabsTrigger>
        </TabsList>

        <TabsContent value="general" className="flex-1 mt-0 overflow-auto">
          <GeneralSettings />
        </TabsContent>

        <TabsContent value="rules" className="flex-1 mt-0 min-h-0">
          <CustomRules />
        </TabsContent>
      </Tabs>
    </div>
  );
}
