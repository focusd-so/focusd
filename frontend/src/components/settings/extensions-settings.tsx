import { useQuery } from "@tanstack/react-query";
import { GetAPIKey } from "../../../bindings/github.com/focusd-so/focusd/internal/settings/service";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { useState } from "react";

const PORT = 50533;

export function ExtensionsSettings() {
  const { data: apiKey } = useQuery({
    queryKey: ["api-key"],
    queryFn: GetAPIKey,
  });

  const [keyVisible, setKeyVisible] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);

  const baseUrl = `http://localhost:${PORT}`;

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopied(label);
    setTimeout(() => setCopied(null), 2000);
  };

  const maskedKey = apiKey ? "•".repeat(apiKey.length) : "";

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Local API</CardTitle>
          <CardDescription>
            Use this API to integrate third-party tools like Claude Code or
            Cursor with Focusd. All requests require the{" "}
            <code className="text-xs bg-muted px-1 py-0.5 rounded">
              Authorization
            </code>{" "}
            header.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Base URL */}
          <div className="space-y-1.5">
            <div className="text-sm font-medium">Base URL</div>
            <div className="flex items-center gap-2">
              <code className="flex-1 text-sm bg-muted px-3 py-2 rounded font-mono">
                {baseUrl}
              </code>
              <Button
                variant="outline"
                size="sm"
                onClick={() => copyToClipboard(baseUrl, "url")}
              >
                {copied === "url" ? "Copied!" : "Copy"}
              </Button>
            </div>
          </div>

          {/* API Key */}
          <div className="space-y-1.5">
            <div className="text-sm font-medium">API Key</div>
            <div className="flex items-center gap-2">
              <code className="flex-1 text-sm bg-muted px-3 py-2 rounded font-mono select-all">
                {keyVisible ? apiKey : maskedKey}
              </code>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setKeyVisible(!keyVisible)}
              >
                {keyVisible ? "Hide" : "Show"}
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => apiKey && copyToClipboard(apiKey, "key")}
              >
                {copied === "key" ? "Copied!" : "Copy"}
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Usage Example</CardTitle>
          <CardDescription>
            Use these endpoints to control Focusd from your development tools.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <div className="text-sm font-medium">
              Whitelist a site (e.g. allow x.com for 1 hour)
            </div>
            <pre className="text-xs bg-muted p-3 rounded overflow-x-auto font-mono whitespace-pre-wrap break-all">
              {`curl -X POST ${baseUrl}/whitelist \\
  -H "Authorization: Bearer ${apiKey || "<your-api-key>"}" \\
  -H "Content-Type: application/json" \\
  -d '{"hostname":"x.com","duration_seconds":3600}'`}
            </pre>
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                copyToClipboard(
                  `curl -X POST ${baseUrl}/whitelist -H "Authorization: Bearer ${apiKey || "<your-api-key>"}" -H "Content-Type: application/json" -d '{"hostname":"x.com","duration_seconds":3600}'`,
                  "curl-whitelist"
                )
              }
            >
              {copied === "curl-whitelist" ? "Copied!" : "Copy command"}
            </Button>
          </div>

          <div className="space-y-2">
            <div className="text-sm font-medium">
              Remove a whitelist entry
            </div>
            <pre className="text-xs bg-muted p-3 rounded overflow-x-auto font-mono whitespace-pre-wrap break-all">
              {`curl -X POST ${baseUrl}/unwhitelist \\
  -H "Authorization: Bearer ${apiKey || "<your-api-key>"}" \\
  -H "Content-Type: application/json" \\
  -d '{"id":1}'`}
            </pre>
          </div>

          <div className="space-y-1">
            <div className="text-xs text-muted-foreground">
              Available endpoints:{" "}
              <code className="bg-muted px-1 rounded">/pause</code>{" "}
              <code className="bg-muted px-1 rounded">/unpause</code>{" "}
              <code className="bg-muted px-1 rounded">/whitelist</code>{" "}
              <code className="bg-muted px-1 rounded">/unwhitelist</code>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
