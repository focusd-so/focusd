import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useSettingsStore } from "@/stores/settings-store";
import { SettingsKey } from "../../../bindings/github.com/focusd-so/focusd/internal/settings/models";

export function GeneralSettings() {
  const { idleThreshold, historyRetention, distractionAllowance, updateSetting } = useSettingsStore();

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Preferences</CardTitle>
          <CardDescription>
            General application preferences.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">

          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label className="text-sm font-medium">Idle timeout</Label>
              <div className="text-sm text-muted-foreground">
                How long before Focusd considers you away from your computer.
              </div>
            </div>
            <Select
              value={idleThreshold}
              onValueChange={(val) => updateSetting(SettingsKey.SettingsKeyIdleThreshold, val)}
            >
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="Select timeout" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="60">1 minute</SelectItem>
                <SelectItem value="120">2 minutes</SelectItem>
                <SelectItem value="300">5 minutes</SelectItem>
                <SelectItem value="600">10 minutes</SelectItem>
                <SelectItem value="900">15 minutes</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label className="text-sm font-medium">History retention</Label>
              <div className="text-sm text-muted-foreground">
                How long to keep your Focusd usage history.
              </div>
            </div>
            <Select
              value={historyRetention}
              onValueChange={(val) => updateSetting(SettingsKey.SettingsKeyHistoryRetention, val)}
            >
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="Select retention" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="7">7 days</SelectItem>
                <SelectItem value="14">14 days</SelectItem>
                <SelectItem value="30">30 days</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label className="text-sm font-medium">Daily distraction allowance</Label>
              <div className="text-sm text-muted-foreground">
                Maximum time allowed on distracting apps/sites per day.
              </div>
            </div>
            <Select
              value={distractionAllowance}
              onValueChange={(val) => updateSetting(SettingsKey.SettingsKeyDistractionAllowance, val)}
            >
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="Select allowance" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="0">Unlimited</SelectItem>
                <SelectItem value="900">15 minutes</SelectItem>
                <SelectItem value="1800">30 minutes</SelectItem>
                <SelectItem value="3600">1 hour</SelectItem>
                <SelectItem value="7200">2 hours</SelectItem>
              </SelectContent>
            </Select>
          </div>

        </CardContent>
      </Card>
    </div>
  );
}
