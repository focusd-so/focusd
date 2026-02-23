import { createFileRoute } from "@tanstack/react-router";
import { CustomRules } from "@/components/custom-rules";
import { useSettingsStore } from "@/stores/settings-store";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

type SettingsSearch = {
  tab?: string;
};

export const Route = createFileRoute("/settings")({
  validateSearch: (search: Record<string, unknown>): SettingsSearch => {
    return {
      tab: typeof search.tab === "string" ? search.tab : "general",
    };
  },
  loader: () => useSettingsStore.getState().fetchSettings(),
  component: SettingsPage,
});

function SettingsPage() {
  const { tab } = Route.useSearch();
  const navigate = Route.useNavigate();

  return (
    <div className="flex flex-col h-full p-4 overflow-hidden">
      <Tabs
        value={tab}
        onValueChange={(value) => navigate({ search: { tab: value }, replace: true })}
        className="flex flex-col h-full w-full overflow-hidden"
      >
        <div>
          <TabsList className="w-fit">
            <TabsTrigger value="general">General</TabsTrigger>
            <TabsTrigger value="rules">Rules</TabsTrigger>
          </TabsList>
        </div>
        <TabsContent value="general" className="flex-1 overflow-y-auto mt-4">
          <div className="flex flex-col gap-4">
            <h3 className="text-lg font-medium">General Settings</h3>
            <p className="text-sm text-muted-foreground">
              General application settings will go here.
            </p>
          </div>
        </TabsContent>
        <TabsContent
          value="rules"
          className="flex-1 min-h-0 mt-4 data-[state=active]:flex data-[state=active]:flex-col"
        >
          <CustomRules />
        </TabsContent>
      </Tabs>
    </div>
  );
}
