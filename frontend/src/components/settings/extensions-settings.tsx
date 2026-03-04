import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useState } from "react";
import {
  IconTerminal,
  IconCopy,
  IconCheck,
  IconRobot,
  IconPlug,
  IconBook,
  IconActivity,
  IconLock,
  IconStar
} from "@tabler/icons-react";
import { Browser } from "@wailsio/runtime";
import { useAccountStore } from "@/stores/account-store";
import { useQuery } from "@tanstack/react-query";
import { DeviceHandshakeResponse_AccountTier } from "../../../bindings/github.com/focusd-so/focusd/gen/api/v1/models";

const PORT = 50533;

export function ExtensionsSettings() {
  const [copied, setCopied] = useState<string | null>(null);
  const { checkoutLink, fetchAccountTier } = useAccountStore();

  const { data: accountTier } = useQuery({
    queryKey: ['accountTier'],
    queryFn: () => fetchAccountTier(),
  });

  const isFreeTier = accountTier === DeviceHandshakeResponse_AccountTier.DeviceHandshakeResponse_ACCOUNT_TIER_FREE;
  const baseUrl = `http://localhost:${PORT}`;

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopied(label);
    setTimeout(() => setCopied(null), 2000);
  };

  return (
    <div className="space-y-8 max-w-4xl pb-10">
      {/* Hero Section */}
      <div className="relative overflow-hidden rounded-3xl border border-emerald-500/10 bg-gradient-to-br from-emerald-500/5 via-transparent to-transparent p-8 shadow-sm">
        <div className="relative z-10 flex flex-col md:flex-row items-start md:items-center justify-between gap-6">
          <div className="space-y-3 flex-1">
            <div className="flex items-center gap-2">
              <Badge variant="outline" className="bg-emerald-500/10 text-emerald-400 border-emerald-500/20 px-2 py-0.5 font-bold uppercase tracking-wider text-[10px]">
                Plus Feature
              </Badge>
            </div>
            <h1 className="text-3xl font-bold tracking-tight bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text">
              Control Focusd with AI Agents
            </h1>
            <p className="text-muted-foreground text-sm leading-relaxed max-w-2xl">
              Focusd provides a local API that allows coding agents like <span className="text-foreground font-medium">Claude Code</span>, <span className="text-foreground font-medium">Cursor</span>, or <span className="text-foreground font-medium">Antigravity</span> to automatically manage your focus state.
            </p>
            <div className="flex flex-wrap gap-4 pt-2">
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground bg-muted/50 px-3 py-1.5 rounded-full border border-border/50">
                <IconRobot className="w-3.5 h-3.5" />
                <span>Claude Code</span>
              </div>
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground bg-muted/50 px-3 py-1.5 rounded-full border border-border/50">
                <IconActivity className="w-3.5 h-3.5" />
                <span>Cursor</span>
              </div>
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground bg-muted/50 px-3 py-1.5 rounded-full border border-border/50">
                <IconPlug className="w-3.5 h-3.5" />
                <span>Antigravity/OpenCode</span>
              </div>
            </div>
          </div>
          <div className="hidden lg:block">
            <div className="p-4 rounded-2xl bg-emerald-500/10 border border-emerald-500/20 shadow-inner">
              <IconPlug className="w-12 h-12 text-emerald-400" />
            </div>
          </div>
        </div>

        {/* Background blobs for aesthetics */}
        <div className="absolute top-0 right-0 -translate-y-1/2 translate-x-1/2 w-64 h-64 bg-emerald-500/5 blur-[80px] rounded-full pointer-events-none" />
        <div className="absolute bottom-0 left-0 translate-y-1/2 -translate-x-1/2 w-48 h-48 bg-emerald-500/5 blur-[60px] rounded-full pointer-events-none" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 relative">
        {/* Main Content */}
        <div className="lg:col-span-8 space-y-6">
          <Card className="border-border/50 shadow-sm overflow-hidden">
            <CardHeader className="bg-muted/30 pb-4">
              <div className="flex items-center gap-2">
                <div className="p-1.5 rounded-md bg-background border border-border/50">
                  <IconTerminal className="w-4 h-4 text-emerald-400" />
                </div>
                <div>
                  <CardTitle className="text-base">Local API Endpoint</CardTitle>
                  <CardDescription className="text-xs">Access your Focusd instance programmatically</CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent className="pt-6 space-y-4">
              <div className="space-y-2">
                <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Base URL</div>
                <div className="flex items-center gap-2 group">
                  <div className="flex-1 font-mono text-sm bg-muted/50 px-4 py-3 rounded-lg border border-border/50 flex items-center justify-between group-hover:border-emerald-500/30 transition-colors">
                    <span className="text-foreground/80">{baseUrl}</span>
                    <button
                      onClick={() => copyToClipboard(baseUrl, "url")}
                      className="p-1 rounded hover:bg-emerald-500/10 text-muted-foreground hover:text-emerald-400 transition-all opacity-40 group-hover:opacity-100"
                    >
                      {copied === "url" ? <IconCheck className="w-4 h-4" /> : <IconCopy className="w-4 h-4" />}
                    </button>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card className="border-border/50 shadow-sm overflow-hidden">
            <CardHeader className="bg-muted/30 pb-4">
              <div className="flex items-center gap-2">
                <div className="p-1.5 rounded-md bg-background border border-border/50">
                  <IconBook className="w-4 h-4 text-emerald-400" />
                </div>
                <div>
                  <CardTitle className="text-base">Usage Examples</CardTitle>
                  <CardDescription className="text-xs">Common commands to control your session</CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent className="pt-6 space-y-6">
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Whitelist a site</div>
                  <Badge variant="outline" className="text-[10px] py-0">POST /whitelist</Badge>
                </div>
                <div className="relative group">
                  <pre className="text-[13px] bg-zinc-950 text-zinc-300 p-4 rounded-xl overflow-x-auto font-mono border border-white/5 leading-relaxed">
                    {`curl -X POST ${baseUrl}/whitelist \\
  -H "Content-Type: application/json" \\
  -d '{"hostname":"x.com","duration_seconds":3600}'`}
                  </pre>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="absolute top-2 right-2 h-8 w-8 text-zinc-500 hover:text-white hover:bg-white/10 transition-colors"
                    onClick={() =>
                      copyToClipboard(
                        `curl -X POST ${baseUrl}/whitelist -H "Content-Type: application/json" -d '{"hostname":"x.com","duration_seconds":3600}'`,
                        "curl-whitelist"
                      )
                    }
                  >
                    {copied === "curl-whitelist" ? <IconCheck className="w-4 h-4" /> : <IconCopy className="w-4 h-4" />}
                  </Button>
                </div>
              </div>

              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Remove from whitelist</div>
                  <Badge variant="outline" className="text-[10px] py-0">POST /unwhitelist</Badge>
                </div>
                <div className="relative group">
                  <pre className="text-[13px] bg-zinc-950 text-zinc-300 p-4 rounded-xl overflow-x-auto font-mono border border-white/5 leading-relaxed">
                    {`curl -X POST ${baseUrl}/unwhitelist \\
  -H "Content-Type: application/json" \\
  -d '{"id":1}'`}
                  </pre>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="absolute top-2 right-2 h-8 w-8 text-zinc-500 hover:text-white hover:bg-white/10 transition-colors"
                    onClick={() =>
                      copyToClipboard(
                        `curl -X POST ${baseUrl}/unwhitelist -H "Content-Type: application/json" -d '{"id":1}'`,
                        "curl-unwhitelist"
                      )
                    }
                  >
                    {copied === "curl-unwhitelist" ? <IconCheck className="w-4 h-4" /> : <IconCopy className="w-4 h-4" />}
                  </Button>
                </div>
              </div>

              <div className="pt-4 border-t border-border/50">
                <div className="flex flex-wrap gap-2">
                  <div className="text-xs text-muted-foreground mr-2 self-center">Quick Reference:</div>
                  {["/pause", "/unpause", "/whitelist", "/unwhitelist"].map((path) => (
                    <code key={path} className="text-[10px] bg-muted px-2 py-0.5 rounded border border-border/50 text-foreground/70 font-mono">
                      {path}
                    </code>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Sidebar Info */}
        <div className="lg:col-span-4 space-y-6">
          <Card className="border-emerald-500/20 bg-emerald-500/[0.02] shadow-sm">
            <CardHeader className="pb-3">
              <div className="flex items-center gap-2 text-emerald-400 mb-1">
                <IconStar className="w-4 h-4 fill-emerald-400/20" />
                <span className="text-xs font-bold uppercase tracking-widest">How it works</span>
              </div>
              <CardTitle className="text-sm">Auto-Pilot Mode</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4 text-xs text-muted-foreground leading-relaxed">
              <p>
                When you run <span className="text-foreground">Claude Code</span> or other agents, they can call the <code className="text-foreground">/pause</code> endpoint to instantly stop Focusd from blocking your research sites.
              </p>
              <p>
                Once the task is finished, the agent calls <code className="text-foreground">/unpause</code> to resume your productivity protections.
              </p>
              <div className="pt-2">
                <Button variant="outline" className="w-full text-[10px] h-7 border-emerald-500/30 hover:bg-emerald-500/10 hover:text-emerald-400 group">
                  View Documentation
                  <IconActivity className="w-3 h-3 ml-1.5 opacity-40 group-hover:opacity-100 transition-opacity" />
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Paywall Overlay */}
        {isFreeTier && (
          <div className="absolute inset-0 z-20 flex items-center justify-center bg-background/40 backdrop-blur-[2px] rounded-xl">
            <div className="max-w-sm w-full mx-4 space-y-5 rounded-2xl border border-border/50 bg-card p-8 shadow-2xl text-center">
              <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-emerald-500/10">
                <IconStar className="h-7 w-7 text-emerald-400 fill-emerald-400/10" />
              </div>
              <div className="space-y-2">
                <h3 className="text-xl font-bold tracking-tight">Integrations are a Plus feature</h3>
                <p className="text-sm text-muted-foreground leading-relaxed">
                  Connect Focusd with your favorite coding agents and automate your focus flow.
                </p>
              </div>
              <Button
                onClick={() => checkoutLink && Browser.OpenURL(checkoutLink)}
                className="w-full bg-emerald-600 hover:bg-emerald-500 text-white shadow-lg shadow-emerald-900/20 h-10 font-bold"
              >
                <IconLock className="mr-2 h-4 w-4" />
                Upgrade to Unlock
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
