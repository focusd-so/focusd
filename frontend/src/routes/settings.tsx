import { createFileRoute } from "@tanstack/react-router"
import { CustomRules } from "@/components/custom-rules";
import { useSettingsStore } from "@/stores/settings-store";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { GeneralSettings } from "@/components/settings/general-settings";
import { ExtensionsSettings } from "@/components/settings/extensions-settings";
import { AboutSettings } from "@/components/settings/about-settings";
import { DevSettings } from "@/components/settings/dev-settings";
import { AccountSettings } from "@/components/settings/account-settings";
import { z } from "zod";

const tabValues = ["general", "account", "rules", "integrations", "about", ...(import.meta.env.DEV ? ["dev"] : [])] as const;

const settingsSearchSchema = z.object({
  tab: z.enum(tabValues as unknown as [string, ...string[]]).optional().catch("general"),
});

export const Route = createFileRoute("/settings")({
  validateSearch: (search) => settingsSearchSchema.parse(search),
  loader: () => useSettingsStore.getState().fetchSettings(),
  component: SettingsPage,
});

function SettingsPage() {
  const { tab } = Route.useSearch();
  const navigate = Route.useNavigate();

  const handleTabChange = (value: string) => {
    navigate({ search: (prev) => ({ ...prev, tab: value as any }) });
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
          <TabsTrigger value="integrations">Integrations</TabsTrigger>
          <TabsTrigger value="account">Account</TabsTrigger>
          <TabsTrigger value="about">About</TabsTrigger>
          {import.meta.env.DEV && (
            <TabsTrigger value="dev">Development</TabsTrigger>
          )}
        </TabsList>

        <TabsContent value="general" className="flex-1 mt-0 overflow-auto">
          <GeneralSettings />
        </TabsContent>


        <TabsContent value="rules" className="flex-1 mt-0 min-h-0">
          <CustomRules />
        </TabsContent>

        <TabsContent value="integrations" className="flex-1 mt-0 overflow-auto">
          <ExtensionsSettings />
        </TabsContent>

        <TabsContent value="account" className="flex-1 mt-0 overflow-auto">
          <AccountSettings />
        </TabsContent>

        <TabsContent value="about" className="flex-1 mt-0 overflow-auto">
          <AboutSettings />
        </TabsContent>

        {import.meta.env.DEV && (
          <TabsContent value="dev" className="flex-1 mt-0 overflow-auto">
            <DevSettings />
          </TabsContent>
        )}
      </Tabs>

    </div>
  );
}
