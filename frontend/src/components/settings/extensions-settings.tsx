import {
  Card,
  CardContent,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useState } from "react";
import {
  IconTerminal,
  IconCopy,
  IconCheck,
  IconRobot,
  IconBook,
  IconLock,
  IconCode
} from "@tabler/icons-react";
import { Browser, Clipboard } from "@wailsio/runtime";
import { useAccountStore } from "@/stores/account-store";
import { useQuery } from "@tanstack/react-query";
import { DeviceHandshakeResponse_AccountTier } from "../../../bindings/github.com/focusd-so/focusd/gen/api/v1/models";

const PORT = 50533;

const API_EXAMPLES = [
  {
    id: "whitelist",
    title: "Whitelist a site",
    method: "POST",
    path: "/whitelist",
    description: "Temporarily allow access to a specific domain while in focus mode. Useful for giving agents access to docs.",
    code: `curl -X POST http://localhost:50533/whitelist \\
  -H "Content-Type: application/json" \\
  -d '{
    "hostname": "example.com",
    "duration_seconds": 3600
  }'`
  },
  {
    id: "unwhitelist",
    title: "Remove from whitelist",
    method: "POST",
    path: "/unwhitelist",
    description: "Remove a previously whitelisted domain from the allowed list to re-enable blocking.",
    code: `curl -X POST http://localhost:50533/unwhitelist \\
  -H "Content-Type: application/json" \\
  -d '{
    "id": 1
  }'`
  },
  {
    id: "pause",
    title: "Pause Focus",
    method: "POST",
    path: "/pause",
    description: "Temporarily pause your current focus session for a specific duration.",
    code: `curl -X POST http://localhost:50533/pause \\
  -H "Content-Type: application/json" \\
  -d '{
    "duration_seconds": 300
  }'`
  },
  {
    id: "unpause",
    title: "Unpause Focus",
    method: "POST",
    path: "/unpause",
    description: "Resume your focus session immediately, canceling any active pause.",
    code: `curl -X POST http://localhost:50533/unpause`
  },
  {
    id: "status",
    title: "Get Status",
    method: "GET",
    path: "/status",
    description: "Retrieve the current status of Focusd, including active sessions, pauses, and whitelists.",
    code: `curl -X GET http://localhost:50533/status`
  }
];

export function ExtensionsSettings() {
  const [copied, setCopied] = useState<string | null>(null);
  const [selectedExampleId, setSelectedExampleId] = useState<string>(API_EXAMPLES[0].id);

  const { checkoutLink, fetchAccountTier } = useAccountStore();

  const { data: accountTier } = useQuery({
    queryKey: ['accountTier'],
    queryFn: () => fetchAccountTier(),
  });

  const isLocked = accountTier === DeviceHandshakeResponse_AccountTier.DeviceHandshakeResponse_ACCOUNT_TIER_FREE;

  const baseUrl = `http://localhost:${PORT}`;

  const copyToClipboard = (text: string, label: string) => {
    Clipboard.SetText(text);
    setCopied(label);
    setTimeout(() => setCopied(null), 2000);
  };

  const activeEx = API_EXAMPLES.find(ex => ex.id === selectedExampleId) || API_EXAMPLES[0];

  return (
    <div className="flex flex-col h-[calc(100vh-10rem)] max-w-5xl gap-4 pb-2">
      {/* Locked Banner */}
      {isLocked && (
        <div className="flex-none flex items-center justify-between p-3 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-200/90 text-[13px]">
          <div className="flex items-center gap-2.5">
            <div className="p-1 rounded-md bg-amber-500/20">
              <IconLock className="w-3.5 h-3.5" />
            </div>
            <span>Extensions are available on <strong>Plus</strong> or <strong>Pro</strong> plans. Upgrade to automate your workflow.</span>
          </div>
          <Button
            onClick={() => checkoutLink && Browser.OpenURL(checkoutLink)}
            size="sm"
            className="h-7 px-3 bg-amber-600 hover:bg-amber-500 text-white text-[11px] font-bold rounded-lg transition-all"
          >
            Upgrade Now
          </Button>
        </div>
      )}

      {/* Top Section: Endpoint & Info */}
      <div className="flex-none flex flex-col md:flex-row items-stretch gap-4">
        <Card className="flex-1 bg-muted/10 border-border/50 shadow-sm overflow-hidden">
          <CardContent className="p-4 flex items-center justify-between h-full">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-emerald-500/10 border border-emerald-500/20">
                <IconTerminal className="w-5 h-5 text-emerald-400" />
              </div>
              <div>
                <div className="text-[11px] font-semibold text-muted-foreground uppercase tracking-widest leading-none mb-1.5 flex items-center gap-2">
                  Local API Endpoint
                  {!isLocked && (
                    <Badge variant="outline" className="bg-emerald-500/10 text-emerald-400 border-emerald-500/20 px-1.5 py-0 font-bold uppercase tracking-wider text-[8px]">
                      Plus Feature
                    </Badge>
                  )}
                </div>
                <div className="font-mono text-sm text-foreground/90">{baseUrl}</div>
              </div>
            </div>
            <Button
              variant="outline"
              size="sm"
              className="gap-2 text-xs h-8 border-border/50 bg-background hover:bg-muted/50"
              onClick={() => copyToClipboard(baseUrl, "url")}
            >
              {copied === "url" ? <IconCheck className="w-3.5 h-3.5 text-emerald-400" /> : <IconCopy className="w-3.5 h-3.5" />}
              {copied === "url" ? "Copied" : "Copy URL"}
            </Button>
          </CardContent>
        </Card>

        <div className="hidden lg:flex flex-col justify-center px-5 py-4 w-[320px] rounded-xl border border-border/50 bg-muted/5 text-[12px] text-muted-foreground leading-snug relative overflow-hidden">
          <div className="absolute top-0 right-0 -translate-y-1/2 translate-x-1/3 w-24 h-24 bg-emerald-500/10 blur-[20px] rounded-full pointer-events-none" />
          <h3 className="font-semibold text-foreground flex items-center gap-1.5 mb-1.5 z-10">
            <IconRobot className="w-4 h-4 text-emerald-400" />
            Extensible Workflow
          </h3>
          <p className="z-10 text-foreground/70">
            Automate focus with local scripts or let coding agents manage blocking automatically while researching.
          </p>
        </div>
      </div>

      {/* Split View for Examples */}
      <Card className="flex-1 overflow-hidden border-border/50 shadow-sm flex flex-col min-h-0 transition-all">
        <div className="flex flex-col md:flex-row h-full divide-y md:divide-y-0 md:divide-x divide-border/50 min-h-0">

          {/* Sidebar List */}
          <div className="w-full md:w-[280px] shrink-0 bg-muted/5 flex flex-col h-full min-h-0">
            <div className="flex-none p-3 border-b border-border/50 bg-muted/10 flex items-center gap-2 text-foreground/80">
              <IconBook className="w-4 h-4 text-emerald-400" />
              <span className="text-xs font-semibold uppercase tracking-wider">Usage Examples</span>
            </div>
            <ScrollArea className="flex-1">
              <div className="p-2 space-y-0.5">
                {API_EXAMPLES.map(ex => (
                  <button
                    key={ex.id}
                    onClick={() => setSelectedExampleId(ex.id)}
                    className={`w-full text-left px-3 py-2.5 rounded-lg text-sm transition-all flex items-center justify-between group ${selectedExampleId === ex.id
                      ? 'bg-emerald-500/10 text-emerald-400 font-medium'
                      : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'
                      }`}
                  >
                    <div className="truncate pr-2">{ex.title}</div>
                    <Badge variant="outline" className={`text-[9px] py-0 px-1 font-mono uppercase bg-transparent ${selectedExampleId === ex.id
                      ? 'border-emerald-500/30 text-emerald-400'
                      : 'border-border/30 text-muted-foreground group-hover:border-border/60'
                      }`}>
                      {ex.method}
                    </Badge>
                  </button>
                ))}
              </div>
            </ScrollArea>
          </div>

          {/* Main Content Area */}
          <div className="flex-1 flex flex-col h-full min-w-0 bg-background relative">
            <div className="flex-none p-5 pb-4">
              <div className="flex items-center justify-between mb-2">
                <h3 className="text-lg font-semibold text-foreground tracking-tight">{activeEx.title}</h3>
                <Badge variant="outline" className="font-mono text-[10px] sm:text-xs bg-muted/20 border-border/50">
                  <span className={`mr-1.5 ${activeEx.method === 'GET' ? 'text-blue-400' : 'text-emerald-400'}`}>{activeEx.method}</span>
                  {activeEx.path}
                </Badge>
              </div>
              <p className="text-[13px] text-muted-foreground/90 leading-relaxed max-w-2xl">{activeEx.description}</p>
            </div>

            <div className="flex-1 p-5 pt-0 min-h-0 flex flex-col">
              <div className="relative group flex-1 rounded-xl overflow-hidden border border-border/50 bg-zinc-950 flex flex-col min-h-0">
                <div className="flex-none px-4 py-2.5 bg-zinc-900 border-b border-white/5 flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <IconCode className="w-4 h-4 text-zinc-500" />
                    <span className="text-xs font-mono text-zinc-400">cURL format</span>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 text-zinc-400 hover:text-white hover:bg-white/10"
                    onClick={() => copyToClipboard(activeEx.code, "curl-" + activeEx.id)}
                  >
                    {copied === "curl-" + activeEx.id ? <IconCheck className="w-4 h-4 text-emerald-400" /> : <IconCopy className="w-4 h-4" />}
                  </Button>
                </div>
                <div className="flex-1 overflow-auto bg-zinc-950 p-4">
                  <pre className="text-[13px] leading-relaxed font-mono text-zinc-300 w-full">
                    <code>{activeEx.code}</code>
                  </pre>
                </div>
              </div>
            </div>
          </div>

        </div>
      </Card>
    </div>
  );
}
