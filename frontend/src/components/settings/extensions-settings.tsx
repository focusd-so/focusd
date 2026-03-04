import {
  Card,
  CardContent,
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
  IconPlus,
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

  const isLocked = accountTier === DeviceHandshakeResponse_AccountTier.DeviceHandshakeResponse_ACCOUNT_TIER_FREE;

  const baseUrl = `http://localhost:${PORT}`;

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopied(label);
    setTimeout(() => setCopied(null), 2000);
  };

  return (
    <div className="space-y-4 max-w-5xl pb-6">
      {/* Locked Banner Replacement */}
      {isLocked && (
        <div className="flex items-center justify-between p-3 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-200/90 text-[13px] animate-in slide-in-from-top-2 duration-300">
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

      {/* Hero Section - More Compact */}
      <div className="relative overflow-hidden rounded-2xl border border-emerald-500/10 bg-gradient-to-br from-emerald-500/5 via-transparent to-transparent p-6 shadow-sm text-foreground">
        <div className="relative z-10 flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
          <div className="space-y-2 flex-1">
            <div className="flex items-center gap-2">
              <Badge variant="outline" className="bg-emerald-500/10 text-emerald-400 border-emerald-500/20 px-2 py-0.5 font-bold uppercase tracking-wider text-[9px]">
                Plus Feature
              </Badge>
            </div>
            <h1 className="text-2xl font-bold tracking-tight bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text">
              Extend Focusd with Local API
            </h1>
            <p className="text-muted-foreground text-[13px] leading-snug max-w-2xl">
              Automate your focus workflow and integrate Focusd with third-party tools, coding agents, and custom scripts. Our Local API provides the building blocks for a customized productivity environment.
            </p>
            <div className="flex flex-wrap gap-2.5 pt-1">
              {[
                { icon: IconRobot, label: "AI Agents" },
                { icon: IconTerminal, label: "Custom Scripts" },
                { icon: IconPlug, label: "Workflow Tools" },
              ].map((item, idx) => (
                <div key={idx} className="flex items-center gap-1.5 text-[11px] text-muted-foreground/80 bg-muted/20 px-2.5 py-1 rounded-full border border-border/30">
                  <item.icon className="w-3 h-3" />
                  <span>{item.label}</span>
                </div>
              ))}
            </div>
          </div>
          <div className="hidden lg:block shrink-0">
            <div className="p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/20 shadow-inner">
              <IconPlus className="w-8 h-8 text-emerald-400/80" />
            </div>
          </div>
        </div>

        {/* Background blobs for aesthetics - smaller */}
        <div className="absolute top-0 right-0 -translate-y-1/2 translate-x-1/2 w-48 h-48 bg-emerald-500/5 blur-[60px] rounded-full pointer-events-none" />
        <div className="absolute bottom-0 left-0 translate-y-1/2 -translate-x-1/2 w-32 h-32 bg-emerald-500/5 blur-[40px] rounded-full pointer-events-none" />
      </div>

      <div className={`grid grid-cols-1 lg:grid-cols-12 gap-4 transition-all duration-500 ${isLocked ? 'grayscale-[0.5] opacity-80 pointer-events-none select-none' : ''}`}>
        {/* Main Content */}
        <div className="lg:col-span-8 space-y-4">
          <Card className="border-border/50 shadow-sm overflow-hidden rounded-xl">
            <CardHeader className="bg-muted/20 py-3 px-4">
              <div className="flex items-center gap-2">
                <div className="p-1 rounded-md bg-background border border-border/50">
                  <IconTerminal className="w-3.5 h-3.5 text-emerald-400" />
                </div>
                <div>
                  <CardTitle className="text-sm font-semibold">Local API Endpoint</CardTitle>
                </div>
              </div>
            </CardHeader>
            <CardContent className="py-4 px-4 space-y-3">
              <div className="space-y-1.5">
                <div className="text-[10px] font-semibold text-muted-foreground uppercase tracking-widest">Base URL</div>
                <div className="flex items-center gap-2 group">
                  <div className="flex-1 font-mono text-[13px] bg-muted/40 px-3 py-2 rounded-lg border border-border/40 flex items-center justify-between group-hover:border-emerald-500/30 transition-colors">
                    <span className="text-foreground/80">{baseUrl}</span>
                    <button
                      disabled={isLocked}
                      onClick={() => !isLocked && copyToClipboard(baseUrl, "url")}
                      className="p-1 rounded hover:bg-emerald-500/10 text-muted-foreground hover:text-emerald-400 transition-all opacity-40 group-hover:opacity-100"
                    >
                      {copied === "url" ? <IconCheck className="w-3.5 h-3.5" /> : <IconCopy className="w-3.5 h-3.5" />}
                    </button>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card className="border-border/50 shadow-sm overflow-hidden rounded-xl">
            <CardHeader className="bg-muted/20 py-3 px-4">
              <div className="flex items-center gap-2">
                <div className="p-1 rounded-md bg-background border border-border/50">
                  <IconBook className="w-3.5 h-3.5 text-emerald-400" />
                </div>
                <div>
                  <CardTitle className="text-sm font-semibold">Usage Examples</CardTitle>
                </div>
              </div>
            </CardHeader>
            <CardContent className="py-4 px-4 space-y-4">
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <div className="text-[10px] font-semibold text-muted-foreground uppercase tracking-widest">Whitelist a site</div>
                  <Badge variant="outline" className="text-[9px] py-0 px-1 border-border/30 bg-muted/20 text-muted-foreground font-mono">POST /whitelist</Badge>
                </div>
                <div className="relative group">
                  <pre className="text-[12px] bg-zinc-950/80 text-zinc-300 p-3 rounded-lg overflow-x-auto font-mono border border-white/5 leading-relaxed">
                    {`curl -X POST ${baseUrl}/whitelist \\
  -H "Content-Type: application/json" \\
  -d '{"hostname":"x.com","duration_seconds":3600}'`}
                  </pre>
                  <Button
                    disabled={isLocked}
                    variant="ghost"
                    size="icon"
                    className="absolute top-2 right-2 h-7 w-7 text-zinc-500 hover:text-white hover:bg-white/10 transition-colors"
                    onClick={() => !isLocked &&
                      copyToClipboard(
                        `curl -X POST ${baseUrl}/whitelist -H "Content-Type: application/json" -d '{"hostname":"x.com","duration_seconds":3600}'`,
                        "curl-whitelist"
                      )
                    }
                  >
                    {copied === "curl-whitelist" ? <IconCheck className="w-3.5 h-3.5" /> : <IconCopy className="w-3.5 h-3.5" />}
                  </Button>
                </div>
              </div>

              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <div className="text-[10px] font-semibold text-muted-foreground uppercase tracking-widest">Remove from whitelist</div>
                  <Badge variant="outline" className="text-[9px] py-0 px-1 border-border/30 bg-muted/20 text-muted-foreground font-mono">POST /unwhitelist</Badge>
                </div>
                <div className="relative group">
                  <pre className="text-[12px] bg-zinc-950/80 text-zinc-300 p-3 rounded-lg overflow-x-auto font-mono border border-white/5 leading-relaxed">
                    {`curl -X POST ${baseUrl}/unwhitelist \\
  -H "Content-Type: application/json" \\
  -d '{"id":1}'`}
                  </pre>
                  <Button
                    disabled={isLocked}
                    variant="ghost"
                    size="icon"
                    className="absolute top-2 right-2 h-7 w-7 text-zinc-500 hover:text-white hover:bg-white/10 transition-colors"
                    onClick={() => !isLocked &&
                      copyToClipboard(
                        `curl -X POST ${baseUrl}/unwhitelist -H "Content-Type: application/json" -d '{"id":1}'`,
                        "curl-unwhitelist"
                      )
                    }
                  >
                    {copied === "curl-unwhitelist" ? <IconCheck className="w-3.5 h-3.5" /> : <IconCopy className="w-3.5 h-3.5" />}
                  </Button>
                </div>
              </div>

              <div className="pt-3 border-t border-border/30">
                <div className="flex flex-wrap gap-2 items-center">
                  <div className="text-[10px] text-muted-foreground uppercase tracking-widest font-semibold">Quick Reference:</div>
                  {["/pause", "/unpause", "/whitelist", "/unwhitelist"].map((path) => (
                    <code key={path} className="text-[10px] bg-muted/40 px-1.5 py-0.5 rounded border border-border/30 text-foreground/70 font-mono">
                      {path}
                    </code>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Sidebar Info */}
        <div className="lg:col-span-4 space-y-4">
          <Card className="border-emerald-500/20 bg-emerald-500/[0.02] shadow-sm overflow-hidden rounded-xl h-full">
            <CardHeader className="py-3 px-4 border-b border-emerald-500/10 bg-emerald-500/5">
              <div className="flex items-center gap-2 text-emerald-400 mb-0.5">
                <IconStar className="w-3.5 h-3.5 fill-emerald-400/20" />
                <span className="text-[10px] font-bold uppercase tracking-widest">How it works</span>
              </div>
              <CardTitle className="text-xs font-bold uppercase tracking-tight">Extensible Workflow</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3.5 p-4 text-[11px] text-muted-foreground leading-relaxed">
              <p>
                The Local API allows external tools to query and modify your focus state. This is perfect for the <span className="text-foreground/90 font-medium text-[10px] tracking-tight border border-border/30 rounded-md px-1.5 py-0.5 bg-muted/40">AGENT ERA</span>.
              </p>
              <div className="space-y-2.5">
                {[
                  { title: "Agents", desc: "can pause blocking while they research on your behalf." },
                  { title: "Scripts", desc: "can automate blocking based on your specific dev environment state." },
                  { title: "Dashboards", desc: "can pull your focus stats for custom displays." },
                ].map((item, idx) => (
                  <div key={idx} className="flex gap-2.5">
                    <div className="mt-1 h-1 w-1 rounded-full bg-emerald-500/50 shrink-0" />
                    <p><span className="text-foreground font-medium underline decoration-emerald-500/20 underline-offset-2">{item.title}</span> {item.desc}</p>
                  </div>
                ))}
              </div>
              <div className="pt-2">
                <Button
                  disabled={isLocked}
                  variant="outline"
                  className="w-full text-[10px] h-7 border-emerald-500/20 hover:bg-emerald-500/10 hover:text-emerald-400 group shadow-sm transition-all"
                >
                  API Documentation
                  <IconActivity className="w-3 h-3 ml-1.5 opacity-40 group-hover:opacity-100 transition-opacity" />
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
