import { useMemo, useState } from "react";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  IconSearch,
  IconChevronDown,
  IconChevronRight,
  IconAlertTriangle,
  IconTerminal,
} from "@tabler/icons-react";
import { useDeferredValue } from "react";
import { useSandboxLogs, useUsingDevFallbackData } from "@/hooks/queries/use-usage";
import { parsePayload, type CustomRulesTracePayload } from "@/lib/timeline";
import type { Event as TimelineEvent } from "../../bindings/github.com/focusd-so/focusd/internal/timeline/models";

function formatTimestamp(unixSeconds: number): string {
  const date = new Date(unixSeconds * 1000);
  return date.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function tryParseJSON(str: string | null | undefined): string {
  if (!str) return "—";
  try {
    return JSON.stringify(JSON.parse(str), null, 2);
  } catch {
    return str;
  }
}

function formatSandboxLogs(logs: string[] | null | undefined): string {
  if (!logs) return "";
  if (Array.isArray(logs)) return logs.join("\n");
  return String(logs);
}

function extractAppInfo(contextStr: string): string {
  try {
    const ctx = JSON.parse(contextStr) as unknown;
    const getString = (obj: Record<string, unknown>, key: string): string => {
      const value = obj[key];
      return typeof value === "string" ? value : "";
    };

    const root = typeof ctx === "object" && ctx !== null ? (ctx as Record<string, unknown>) : {};
    const usage =
      typeof root.usage === "object" && root.usage !== null
        ? (root.usage as Record<string, unknown>)
        : null;
    const meta =
      usage && typeof usage.meta === "object" && usage.meta !== null
        ? (usage.meta as Record<string, unknown>)
        : (usage ?? root);

    const appName = getString(meta, "appName") || getString(meta, "app");
    const host = getString(meta, "host") || getString(meta, "hostname");
    const domain = getString(meta, "domain");
    const title = getString(meta, "title");

    const parts: string[] = [];
    if (appName) parts.push(appName);
    if (host) parts.push(host);
    else if (domain) parts.push(domain);
    else if (title) parts.push(title);

    return parts.join(" · ") || "unknown";
  } catch {
    return contextStr.slice(0, 60);
  }
}

interface SandboxLogView {
  id: number;
  type: string;
  created_at: number;
  context: string;
  response: string;
  logs: string[];
  error: string;
}

function eventToLogView(event: TimelineEvent): SandboxLogView {
  const payload = parsePayload<CustomRulesTracePayload>(event) ?? {
    context: "",
    logs: [],
    resp: "",
    error: "",
  };
  // The timeline event currently uses a single trace type; future split between
  // classification / enforcement_action will land in the event "type" tag.
  return {
    id: event.id,
    type: event.type === "custom_rules_trace" ? "classification" : event.type,
    created_at: event.occurred_at,
    context: payload.context ?? "",
    response: payload.resp ?? "",
    logs: payload.logs ?? [],
    error: payload.error ?? "",
  };
}

function LogEntry({ log }: { log: SandboxLogView }) {
  const [expanded, setExpanded] = useState(false);
  const hasError = !!log.error;
  const appInfo = extractAppInfo(log.context);

  return (
    <div
      className={`rounded-lg border transition-colors ${hasError
        ? "border-red-500/20 bg-red-500/5"
        : "border-border/50 bg-card/50 hover:bg-card/80"
        }`}
    >
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-2 p-2.5 text-left"
      >
        {expanded ? (
          <IconChevronDown className="w-3.5 h-3.5 text-muted-foreground/50 shrink-0" />
        ) : (
          <IconChevronRight className="w-3.5 h-3.5 text-muted-foreground/50 shrink-0" />
        )}

        <div className="flex-1 min-w-0 flex flex-col gap-0.5">
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-foreground truncate">
              {appInfo}
            </span>
            <Badge
              variant="outline"
              className={`px-1.5 py-0 text-[9px] font-bold rounded-full shrink-0 ${log.type === "classification"
                ? "border-blue-500/30 text-blue-400"
                : "border-purple-500/30 text-purple-400"
                }`}
            >
              {log.type}
            </Badge>
            {hasError && (
              <IconAlertTriangle className="w-3 h-3 text-red-400 shrink-0" />
            )}
          </div>
          <div className="flex items-center gap-2">
            <span className="text-[10px] text-muted-foreground/50 font-mono">
              {formatTimestamp(log.created_at)}
            </span>
            {log.response && log.response !== "no response" && (
              <>
                <span className="text-[10px] text-muted-foreground/30">·</span>
                <span className="text-[10px] text-green-400/70 truncate max-w-[150px]">
                  {log.response}
                </span>
              </>
            )}
            {log.logs.length > 0 && !expanded && (
              <>
                <span className="text-[10px] text-muted-foreground/30">·</span>
                <span className="text-[10px] text-yellow-400/50 truncate max-w-[200px]">
                  {formatSandboxLogs(log.logs).replace(/\n/g, " ")}
                </span>
              </>
            )}
          </div>
        </div>
      </button>

      {expanded && (
        <div className="border-t border-border/30 p-3 space-y-3">
          <div>
            <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/50 mb-1 block">
              Context
            </span>
            <pre className="text-[11px] text-muted-foreground bg-background/50 rounded p-2 overflow-x-auto max-h-[200px] overflow-y-auto font-mono whitespace-pre-wrap break-all">
              {tryParseJSON(log.context)}
            </pre>
          </div>

          <div>
            <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/50 mb-1 block">
              Response
            </span>
            <pre className="text-[11px] text-green-400/80 bg-background/50 rounded p-2 overflow-x-auto max-h-[150px] overflow-y-auto font-mono whitespace-pre-wrap break-all">
              {tryParseJSON(log.response)}
            </pre>
          </div>

          {log.logs.length > 0 && (
            <div>
              <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/50 mb-1 flex items-center gap-1">
                <IconTerminal className="w-3 h-3" />
                Console Logs
              </span>
              <pre className="text-[11px] text-yellow-400/80 bg-background/50 rounded p-2 overflow-x-auto max-h-[100px] overflow-y-auto font-mono whitespace-pre-wrap break-all">
                {formatSandboxLogs(log.logs)}
              </pre>
            </div>
          )}

          {hasError && (
            <div>
              <span className="text-[10px] font-bold uppercase tracking-wider text-red-400/50 mb-1 block">
                Error
              </span>
              <pre className="text-[11px] text-red-400 bg-red-500/5 rounded p-2 overflow-x-auto font-mono whitespace-pre-wrap break-all">
                {log.error}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export function ExecutionLogsSheet({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search);
  const [typeFilter, setTypeFilter] = useState("");
  const usingFallback = useUsingDevFallbackData();

  const { data: events = [], isLoading } = useSandboxLogs(typeFilter, deferredSearch);

  const logs = useMemo<SandboxLogView[]>(
    () => events.map(eventToLogView),
    [events],
  );

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-lg p-0 flex flex-col">
        <SheetHeader className="px-4 pt-4 pb-3 border-b border-border/50 shrink-0">
          <SheetTitle className="text-sm">Execution Logs</SheetTitle>
          <SheetDescription className="text-xs">
            Custom rules sandbox execution history (last 7 days)
          </SheetDescription>
        </SheetHeader>

        {usingFallback && (
          <div className="px-4 pt-3">
            <div className="rounded-md border border-amber-500/20 bg-amber-500/10 px-3 py-2 text-[11px] text-amber-200/90">
              Showing sample data — backend endpoint pending timeline rewrite.
            </div>
          </div>
        )}

        <div className="px-4 py-3 flex items-center gap-2 border-b border-border/30 shrink-0">
          <div className="relative flex-1">
            <IconSearch className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground/50" />
            <Input
              placeholder="Search context or response..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="h-8 pl-8 text-xs bg-background/50"
            />
          </div>
          <div className="flex items-center gap-1">
            <Button
              variant={typeFilter === "" ? "secondary" : "ghost"}
              size="sm"
              onClick={() => setTypeFilter("")}
              className="h-7 px-2 text-[10px] font-medium"
            >
              All
            </Button>
            <Button
              variant={typeFilter === "classification" ? "secondary" : "ghost"}
              size="sm"
              onClick={() => setTypeFilter("classification")}
              className="h-7 px-2 text-[10px] font-medium"
            >
              Classify
            </Button>
            <Button
              variant={typeFilter === "enforcement_action" ? "secondary" : "ghost"}
              size="sm"
              onClick={() => setTypeFilter("enforcement_action")}
              className="h-7 px-2 text-[10px] font-medium"
            >
              Terminate
            </Button>
          </div>
        </div>

        <ScrollArea className="flex-1 min-h-0">
          <div className="p-3 space-y-2">
            {logs.length === 0 && !isLoading ? (
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground/50">
                <IconTerminal className="w-8 h-8 mb-2 opacity-50" />
                <p className="text-xs">No execution logs found</p>
                {search && (
                  <p className="text-[10px] mt-1">
                    Try adjusting your search query
                  </p>
                )}
              </div>
            ) : (
              logs.map((log) => <LogEntry key={log.id} log={log} />)
            )}
          </div>
        </ScrollArea>

        <div className="px-4 py-2 border-t border-border/30 shrink-0">
          <p className="text-[10px] text-muted-foreground/40 text-center">
            {logs.length} log{logs.length !== 1 ? "s" : ""} loaded · retained for
            7 days
          </p>
        </div>
      </SheetContent>
    </Sheet>
  );
}
