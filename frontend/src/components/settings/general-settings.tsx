import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export function GeneralSettings() {
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Preferences</CardTitle>
          <CardDescription>
            General application preferences.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between opacity-50 cursor-not-allowed">
            <div className="space-y-0.5">
              <div className="text-sm font-medium">Launch at login</div>
              <div className="text-sm text-muted-foreground">
                Automatically start Focusd when you log in.
              </div>
            </div>
            <div className="text-xs font-medium uppercase tracking-wider text-muted-foreground bg-muted px-2 py-1 rounded">
              Coming Soon
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
