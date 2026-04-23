import { IconTerminal } from "@tabler/icons-react";
import {
  formatSandboxLogs,
  hasSandboxData,
  hasSandboxResult,
  tryParseJSON,
} from "@/components/usage-item/formatters";
import type { UsageSandboxData } from "@/lib/timeline";

export function UsageItemSandboxPanel({
  classificationSandbox,
  enforcementSandbox,
}: {
  classificationSandbox?: UsageSandboxData;
  enforcementSandbox?: UsageSandboxData;
}) {
  const hasClassificationSandbox = hasSandboxData(classificationSandbox);
  const hasEnforcementSandbox = hasSandboxData(enforcementSandbox);

  if (!hasClassificationSandbox && !hasEnforcementSandbox) return null;

  return (
    <div className="w-full space-y-3">
      <div className="space-y-1">
        <h4 className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/45 flex items-center gap-1.5">
          <span className="h-1 w-1 rounded-full bg-muted-foreground/30" />
          Execution Trace
        </h4>
        <p className="text-[9px] text-muted-foreground/40 font-medium italic leading-tight">
          Rule engine debug output
        </p>
      </div>

      {hasClassificationSandbox && (
        <SandboxSection title="Classification Sandbox" sandbox={classificationSandbox} />
      )}

      {hasEnforcementSandbox && (
        <SandboxSection title="Enforcement Sandbox" sandbox={enforcementSandbox} />
      )}
    </div>
  );
}

function SandboxSection({
  title,
  sandbox,
}: {
  title: string;
  sandbox?: UsageSandboxData;
}) {
  const hasContext = hasSandboxResult(sandbox?.context);
  const hasResponse = hasSandboxResult(sandbox?.response);
  const hasLogs = hasSandboxResult(sandbox?.logs) && formatSandboxLogs(sandbox?.logs).trim() !== "";

  return (
    <div className="space-y-3">
      <span className="text-[9px] font-bold uppercase tracking-wider text-muted-foreground/30 block">
        {title}
      </span>

      {(hasContext || hasResponse) && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {hasContext && (
            <div>
              <span className="text-[9px] font-bold uppercase tracking-wider text-muted-foreground/30 mb-1 block">
                Context
              </span>
              <pre className="text-[10px] text-muted-foreground/70 bg-background/30 rounded p-1.5 overflow-x-auto max-h-[150px] overflow-y-auto font-mono whitespace-pre-wrap break-all border border-border/10">
                {tryParseJSON(sandbox?.context)}
              </pre>
            </div>
          )}
          {hasResponse && (
            <div>
              <span className="text-[9px] font-bold uppercase tracking-wider text-muted-foreground/30 mb-1 block">
                Response
              </span>
              <pre className="text-[10px] text-green-400/60 bg-background/30 rounded p-1.5 overflow-x-auto max-h-[150px] overflow-y-auto font-mono whitespace-pre-wrap break-all border border-border/10">
                {tryParseJSON(sandbox?.response)}
              </pre>
            </div>
          )}
        </div>
      )}

      {hasLogs && (
        <div>
          <span className="text-[9px] font-bold uppercase tracking-wider text-muted-foreground/30 mb-1 flex items-center gap-1">
            <IconTerminal className="w-3 h-3 opacity-70" />
            Console Logs
          </span>
          <pre className="text-[10px] text-yellow-400/60 bg-background/30 rounded p-1.5 overflow-x-auto max-h-[150px] overflow-y-auto font-mono whitespace-pre-wrap break-all border border-border/10">
            {formatSandboxLogs(sandbox?.logs)}
          </pre>
        </div>
      )}
    </div>
  );
}
