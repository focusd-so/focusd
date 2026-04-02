import { useState, useMemo } from "react";
import { useInfiniteQuery } from "@tanstack/react-query";
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
import { GetSandboxExecutionLogs } from "../../bindings/github.com/focusd-so/focusd/internal/usage/service";
import type { SandboxExecutionLog } from "../../bindings/github.com/focusd-so/focusd/internal/usage/models";
import { useDeferredValue } from "react";

const PAGE_SIZE = 30;

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
        : root;

    const appName = getString(meta, "appName");
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

function LogEntry({ log }: { log: SandboxExecutionLog }) {
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
            {log.logs && log.logs !== "null" && !expanded && (
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
          {/* Context */}
          <div>
            <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/50 mb-1 block">
              Context
            </span>
            <pre className="text-[11px] text-muted-foreground bg-background/50 rounded p-2 overflow-x-auto max-h-[200px] overflow-y-auto font-mono whitespace-pre-wrap break-all">
              {tryParseJSON(log.context)}
            </pre>
          </div>

          {/* Response */}
          <div>
            <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/50 mb-1 block">
              Response
            </span>
            <pre className="text-[11px] text-green-400/80 bg-background/50 rounded p-2 overflow-x-auto max-h-[150px] overflow-y-auto font-mono whitespace-pre-wrap break-all">
              {tryParseJSON(log.response)}
            </pre>
          </div>

          {/* Console Logs */}
          {log.logs && log.logs.trim() !== "null" && (
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

          {/* Error */}
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

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
  } = useInfiniteQuery({
    queryKey: ["sandbox-logs", typeFilter, deferredSearch],
    initialPageParam: 0,
    queryFn: ({ pageParam }) =>
      GetSandboxExecutionLogs(typeFilter, deferredSearch, pageParam as number, PAGE_SIZE),
    getNextPageParam: (lastPage: SandboxExecutionLog[], allPages) => {
      return lastPage.length === PAGE_SIZE ? allPages.length : undefined;
    },
    enabled: open,
  });

  const logs = useMemo(() => {
    return data?.pages.flat() || [];
  }, [data]);

  const loadMore = () => {
    if (hasNextPage && !isFetchingNextPage) {
      fetchNextPage();
    }
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-lg p-0 flex flex-col">
        <SheetHeader className="px-4 pt-4 pb-3 border-b border-border/50 shrink-0">
          <SheetTitle className="text-sm">Execution Logs</SheetTitle>
          <SheetDescription className="text-xs">
            Custom rules sandbox execution history (last 7 days)
          </SheetDescription>
        </SheetHeader>

        {/* Search & Filters */}
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

        {/* Log List */}
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
              <>
                {logs.map((log) => (
                  <LogEntry key={log.id} log={log} />
                ))}
                {hasNextPage && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={loadMore}
                    disabled={isFetchingNextPage}
                    className="w-full h-8 text-xs text-muted-foreground/50 hover:text-muted-foreground"
                  >
                    {isFetchingNextPage ? "Loading..." : "Load more"}
                  </Button>
                )}
              </>
            )}
          </div>
        </ScrollArea>

        {/* Footer */}
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
