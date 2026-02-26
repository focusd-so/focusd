import { useEffect, useState } from "react";
import { GetVersion } from "../../../bindings/github.com/focusd-so/focusd/internal/settings/service";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export function AboutSettings() {
  const [version, setVersion] = useState<string>("");

  useEffect(() => {
    GetVersion().then(setVersion).catch(console.error);
  }, []);

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>About Focusd</CardTitle>
          <CardDescription>
            Application information and version details.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <div className="text-sm font-medium">Version</div>
              <div className="text-sm text-muted-foreground">
                The current version of the application.
              </div>
            </div>
            <div className="font-mono text-sm bg-muted px-2 py-1 rounded">
              {version || "Loading..."}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
