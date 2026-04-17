import { useState } from "react";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  IconPlayerPlay,
  IconLoader2,
  IconAlertTriangle,
  IconTerminal,
  IconTag,
} from "@tabler/icons-react";
import { TestClassifyCustomRules, GetSandboxExecutionLogs } from "../../bindings/github.com/focusd-so/focusd/internal/usage/service";
import type { CustomRulesClassificationResult, CustomRulesTracePayload } from "../../bindings/github.com/focusd-so/focusd/internal/usage/models";
import type { Event as TimelineEvent } from "../../bindings/github.com/focusd-so/focusd/internal/timeline/models";
import { parsePayload } from "@/lib/timeline";

const TIMEZONES = [
  {
    group: "Common", items: [
      { value: "local", label: "Local Time" },
      { value: "UTC", label: "UTC" },
    ]
  },
  {
    group: "Americas", items: [
      { value: "America/New_York", label: "US Eastern (New York)" },
      { value: "America/Chicago", label: "US Central (Chicago)" },
      { value: "America/Denver", label: "US Mountain (Denver)" },
      { value: "America/Los_Angeles", label: "US Pacific (Los Angeles)" },
      { value: "America/Toronto", label: "Canada (Toronto)" },
      { value: "America/Sao_Paulo", label: "Brazil (São Paulo)" },
    ]
  },
  {
    group: "Europe & Africa", items: [
      { value: "Europe/London", label: "UK (London)" },
      { value: "Europe/Paris", label: "France (Paris)" },
      { value: "Europe/Berlin", label: "Germany (Berlin)" },
      { value: "Europe/Moscow", label: "Russia (Moscow)" },
      { value: "Africa/Cairo", label: "Egypt (Cairo)" },
    ]
  },
  {
    group: "Asia & Oceania", items: [
      { value: "Asia/Dubai", label: "UAE (Dubai)" },
      { value: "Asia/Kolkata", label: "India (Kolkata)" },
      { value: "Asia/Shanghai", label: "China (Shanghai)" },
      { value: "Asia/Tokyo", label: "Japan (Tokyo)" },
      { value: "Asia/Seoul", label: "South Korea (Seoul)" },
      { value: "Australia/Sydney", label: "Australia (Sydney)" },
      { value: "Pacific/Auckland", label: "New Zealand (Auckland)" },
    ]
  },
] as const;

function formatDatetimeLocal(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

/**
 * Build an ISO 8601 string interpreting the naive datetime-local value in the given timezone.
 * For "local", uses the browser's local timezone. For IANA names, computes the UTC offset.
 */
function buildISOInTimezone(naiveDatetime: string, timezone: string): string {
  const [datePart, timePart] = naiveDatetime.split("T");
  const [y, m, d] = datePart.split("-").map(Number);
  const [h, min] = timePart.split(":").map(Number);

  if (timezone === "local") {
    return new Date(y, m - 1, d, h, min).toISOString();
  }

  // Create a reference date in UTC with these wall-clock values
  const refUtc = Date.UTC(y, m - 1, d, h, min);

  // Use Intl to find the UTC offset of the target timezone at this approximate time
  const formatter = new Intl.DateTimeFormat("en-US", {
    timeZone: timezone,
    year: "numeric", month: "2-digit", day: "2-digit",
    hour: "2-digit", minute: "2-digit", second: "2-digit",
    hour12: false,
  });

  // Format refUtc in the target timezone, then parse back to compute offset
  const parts = formatter.formatToParts(new Date(refUtc));
  const get = (type: string) => Number(parts.find((p) => p.type === type)?.value ?? 0);
  const tzDate = Date.UTC(get("year"), get("month") - 1, get("day"), get("hour"), get("minute"), get("second"));
  const offsetMs = tzDate - refUtc;

  const pad = (n: number) => String(Math.abs(n)).padStart(2, "0");
  const offsetMin = offsetMs / 60000;
  const sign = offsetMin >= 0 ? "+" : "-";
  const oh = pad(Math.floor(Math.abs(offsetMin) / 60));
  const om = pad(Math.abs(offsetMin) % 60);

  // Build RFC3339 string with explicit offset
  const iso = `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}T${String(h).padStart(2, "0")}:${String(min).padStart(2, "0")}:00${sign}${oh}:${om}`;
  return iso;
}

function classificationColor(classification: string): string {
  switch (classification) {
    case "productive":
      return "border-green-500/30 text-green-400 bg-green-500/10";
    case "distracting":
      return "border-red-500/30 text-red-400 bg-red-500/10";
    case "neutral":
      return "border-yellow-500/30 text-yellow-400 bg-yellow-500/10";
    case "system":
      return "border-blue-500/30 text-blue-400 bg-blue-500/10";
    default:
      return "border-muted-foreground/30 text-muted-foreground bg-muted/10";
  }
}

function tryParseJSON(str: string | null | undefined): string {
  if (!str) return "—";
  try {
    return JSON.stringify(JSON.parse(str), null, 2);
  } catch {
    return str;
  }
}

function formatSandboxLogs(logsStr: string | null | undefined): string {
  if (!logsStr) return "";
  try {
    const logs = JSON.parse(logsStr);
    if (Array.isArray(logs)) {
      return logs.join("\n");
    }
    return logsStr;
  } catch {
    return logsStr;
  }
}

export function TestRulesSheet({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [appName, setAppName] = useState("");
  const [url, setUrl] = useState("");
  const [datetime, setDatetime] = useState(formatDatetimeLocal(new Date()));
  const [timezone, setTimezone] = useState("local");
  const [isLoading, setIsLoading] = useState(false);
  const [result, setResult] = useState<CustomRulesClassificationResult | null>(null);
  const [lastLog, setLastLog] = useState<{ logs: string; output: string; context: string; error: string } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [hasRun, setHasRun] = useState(false);

  const handleTest = async () => {
    if (!appName.trim()) return;

    setIsLoading(true);
    setError(null);
    setResult(null);
    setLastLog(null);

    try {
      const urlParam = url.trim() || null;
      const nowTime = buildISOInTimezone(datetime, timezone);

      const response = await TestClassifyCustomRules(
        appName.trim(),
        urlParam,
        nowTime
      );

      setResult(response);
      setHasRun(true);

      // Always fetch the latest execution log to capture console output
      // (logs aren't included in the response when classification is null)
      try {
        const events = (await GetSandboxExecutionLogs("", "", 0, 1)) as (TimelineEvent | null)[];
        const event = events && events[0];
        if (event) {
          const payload = parsePayload<CustomRulesTracePayload>(event);
          if (payload) {
            setLastLog({
              context: payload.context ?? "",
              output: payload.resp ?? "",
              error: payload.error ?? "",
              logs:
                payload.logs && payload.logs.length > 0
                  ? JSON.stringify(payload.logs)
                  : "",
            });
          }
        }
      } catch {
        // Non-critical — just won't show logs
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setHasRun(true);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-lg p-0 flex flex-col">
        <SheetHeader className="px-4 pt-4 pb-3 border-b border-border/50 shrink-0">
          <SheetTitle className="text-sm">Test Custom Rules</SheetTitle>
          <SheetDescription className="text-xs">
            Simulate rule execution with custom inputs
          </SheetDescription>
        </SheetHeader>

        <ScrollArea className="flex-1 min-h-0">
          <div className="p-4 space-y-4">
            {/* Input Fields */}
            <div className="space-y-3">
              <div className="space-y-1.5">
                <label className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                  App Name <span className="text-red-400">*</span>
                </label>
                <Input
                  placeholder='e.g. "Slack", "Chrome"'
                  value={appName}
                  onChange={(e) => setAppName(e.target.value)}
                  className="h-9 text-sm bg-background/50"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                  URL <span className="text-[10px] font-normal text-muted-foreground/40">(optional)</span>
                </label>
                <Input
                  placeholder="e.g. https://youtube.com/watch?v=..."
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  className="h-9 text-sm bg-background/50 font-mono text-xs"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                  Simulated Time
                </label>
                <div className="flex gap-2">
                  <Input
                    type="datetime-local"
                    value={datetime}
                    onChange={(e) => setDatetime(e.target.value)}
                    className="h-9 text-sm bg-background/50 flex-1"
                  />
                  <Select value={timezone} onValueChange={setTimezone}>
                    <SelectTrigger size="sm" className="h-9 text-xs bg-background/50 min-w-[140px]">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent position="popper" className="max-h-[280px]">
                      {TIMEZONES.map((group) => (
                        <SelectGroup key={group.group}>
                          <SelectLabel>{group.group}</SelectLabel>
                          {group.items.map((tz) => (
                            <SelectItem key={tz.value} value={tz.value} className="text-xs">
                              {tz.label}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </div>

            {/* Test Button */}
            <Button
              onClick={handleTest}
              disabled={isLoading || !appName.trim()}
              className="w-full h-9 text-sm font-semibold bg-primary hover:bg-primary/90 transition-all"
            >
              {isLoading ? (
                <>
                  <IconLoader2 className="w-4 h-4 mr-2 animate-spin" />
                  Running…
                </>
              ) : (
                <>
                  <IconPlayerPlay className="w-4 h-4 mr-2" />
                  Run Test
                </>
              )}
            </Button>

            {/* Results */}
            {hasRun && (
              <>
                <Separator className="my-1" />

                {error ? (
                  <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-3 space-y-2">
                    <div className="flex items-center gap-2">
                      <IconAlertTriangle className="w-4 h-4 text-red-400 shrink-0" />
                      <span className="text-xs font-semibold text-red-400">
                        Error
                      </span>
                    </div>
                    <pre className="text-[11px] text-red-400/80 font-mono whitespace-pre-wrap break-all">
                      {error}
                    </pre>
                  </div>
                ) : result === null ? (
                  <div className="space-y-3">
                    <div className="rounded-lg border border-border/50 bg-muted/20 p-4 text-center">
                      <p className="text-xs text-muted-foreground/60">
                        No match — your rules returned <code className="px-1 py-0.5 bg-muted rounded text-[11px]">undefined</code>
                      </p>
                      <p className="text-[10px] text-muted-foreground/40 mt-1">
                        The default classification pipeline would handle this.
                      </p>
                    </div>

                    {/* Show console logs from execution log even on no-match */}
                    {lastLog?.logs && lastLog.logs.trim() !== "" && (
                      <div>
                        <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/50 mb-1 flex items-center gap-1">
                          <IconTerminal className="w-3 h-3" />
                          Console Logs
                        </span>
                        <pre className="text-[11px] text-yellow-400/80 bg-background/50 rounded-lg p-2.5 overflow-x-auto max-h-[100px] overflow-y-auto font-mono whitespace-pre-wrap break-all border border-border/30">
                          {formatSandboxLogs(lastLog.logs)}
                        </pre>
                      </div>
                    )}
                  </div>
                ) : (
                  <div className="space-y-3">
                    {/* Classification Badge */}
                    <div className="rounded-lg border border-border/50 bg-card/50 p-3 space-y-3">
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/50">
                          Classification
                        </span>
                        <Badge
                          variant="outline"
                          className={`px-2 py-0.5 text-[11px] font-bold rounded-full ${classificationColor(result.classification)}`}
                        >
                          {result.classification || "unknown"}
                        </Badge>
                      </div>

                      {result.classification_reason && (
                        <div>
                          <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/50 mb-1 block">
                            Reasoning
                          </span>
                          <p className="text-xs text-foreground/80 leading-relaxed">
                            {result.classification_reason}
                          </p>
                        </div>
                      )}

                      {result.tags && result.tags.length > 0 && (
                        <div>
                          <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/50 mb-1.5 flex items-center gap-1">
                            <IconTag className="w-3 h-3" />
                            Tags
                          </span>
                          <div className="flex flex-wrap gap-1.5">
                            {result.tags.map((tag) => (
                              <Badge
                                key={tag}
                                variant="outline"
                                className="px-2 py-0 text-[10px] font-medium rounded-full border-border/50 text-muted-foreground/70"
                              >
                                {tag}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>

                    {/* Sandbox Context */}
                    {result.sandbox_context && (
                      <div>
                        <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/50 mb-1 block">
                          Sandbox Context
                        </span>
                        <pre className="text-[11px] text-muted-foreground bg-background/50 rounded-lg p-2.5 overflow-x-auto max-h-[180px] overflow-y-auto font-mono whitespace-pre-wrap break-all border border-border/30">
                          {tryParseJSON(result.sandbox_context)}
                        </pre>
                      </div>
                    )}

                    {/* Sandbox Response */}
                    {result.sandbox_output && result.sandbox_output !== "no response" && (
                      <div>
                        <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/50 mb-1 block">
                          Sandbox Response
                        </span>
                        <pre className="text-[11px] text-green-400/80 bg-background/50 rounded-lg p-2.5 overflow-x-auto max-h-[120px] overflow-y-auto font-mono whitespace-pre-wrap break-all border border-border/30">
                          {tryParseJSON(result.sandbox_output)}
                        </pre>
                      </div>
                    )}

                    {/* Console Logs */}
                    {result.sandbox_logs && result.sandbox_logs.length > 0 && (
                      <div>
                        <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/50 mb-1 flex items-center gap-1">
                          <IconTerminal className="w-3 h-3" />
                          Console Logs
                        </span>
                        <pre className="text-[11px] text-yellow-400/80 bg-background/50 rounded-lg p-2.5 overflow-x-auto max-h-[100px] overflow-y-auto font-mono whitespace-pre-wrap break-all border border-border/30">
                          {formatSandboxLogs(JSON.stringify(result.sandbox_logs))}
                        </pre>
                      </div>
                    )}
                  </div>
                )}
              </>
            )}
          </div>
        </ScrollArea>
      </SheetContent>
    </Sheet>
  );
}
