import { useCallback, useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import Editor, { type Monaco } from "@monaco-editor/react";
import type { editor } from "monaco-editor";
import { useSettingsStore } from "@/stores/settings-store";
import { useAccountStore } from "@/stores/account-store";
import { DeviceHandshakeResponse_AccountTier } from "../../bindings/github.com/focusd-so/focusd/gen/api/v1/models";
import { Browser } from "@wailsio/runtime";
import { Button } from "@/components/ui/button";
import { IconDeviceFloppy, IconFileText, IconTerminal, IconTestPipe, IconCrown } from "@tabler/icons-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { ExecutionLogsSheet } from "@/components/execution-logs";
import { TestRulesSheet } from "@/components/test-rules-sheet";

const TYPES_FILE_PATH = "file:///focusd-types.d.ts";
const SETTINGS_KEY = "custom_rules";
const DRAFT_STORAGE_KEY = "focusd_custom_rules_draft";

const typeDefinitions = `
/**
 * Represents the type of activity classification.
 */
type ClassificationType = "unknown" | "productive" | "distracting" | "neutral" | "system";

/**
 * Global constant for classification values.
 * Use these values when returning a ClassificationDecision.
 * @example
 * return {
 *   classification: Classification.Productive,
 *   classificationReasoning: "Work-related activity"
 * };
 */
declare const Classification: {
  readonly Unknown: "unknown";
  readonly Productive: "productive";
  readonly Distracting: "distracting";
  readonly Neutral: "neutral";
  readonly System: "system";
};

/**
 * Determines whether to block or allow the activity.
 */
type EnforcementActionType = "none" | "block" | "paused" | "allow";

/**
 * Global constant for termination mode values.
 * Use these values when returning a EnforcementDecision.
 * @example
 * return {
 *   enforcementAction: EnforcementAction.Block,
 *   enforcementReason: "Blocked during focus hours"
 * };
 */
declare const EnforcementAction: {
  readonly None: "none";
  readonly Block: "block";
  readonly Paused: "paused";
  readonly Allow: "allow";
};

/**
 * Decision returned from the enforcementDecision function.
 */
interface EnforcementDecision {
  /** The termination mode to apply. Use EnforcementAction constants. */
  enforcementAction: EnforcementActionType;
  /** Human-readable explanation for why this decision was made. */
  enforcementReason: string;
}

/**
 * Decision returned from the classify function.
 */
interface ClassificationDecision {
  /** The classification to apply. Use Classification constants. */
  classification: ClassificationType;
  /** Human-readable explanation for why this classification was chosen. */
  classificationReasoning: string;
}

/**
 * Provides context for the current rule execution including usage data.
 */
interface UsageContext {
  /** The display name of the application (e.g., 'Safari', 'Slack'). */
  readonly appName?: string;
  /** The application's bundle identifier (e.g., 'com.apple.Safari'). */
  readonly bundleID: string;
  /** The hostname if the activity is a website (e.g., 'www.github.com'). */
  readonly hostname: string;
  /** The registered domain extracted from the hostname (e.g., 'github.com'). */
  readonly domain: string;
  /** The full URL if available. */
  readonly url: string;
  /** The current classification of this usage (may be empty if not yet classified). */
  readonly classification: string;
  /** Minutes since this app/site was last blocked (-1 if never blocked). */
  readonly minutesSinceLastBlock: number;
  /** Total minutes of usage since this app/site was last blocked (-1 if never blocked). */
  readonly minutesUsedSinceLastBlock: number;
  /**
   * Returns total minutes this app/site was used in the last N minutes.
   * @param minutes - The time window to check (e.g., 60 for last hour, 30 for last 30 minutes)
   * @returns Total minutes of usage in the specified time window
   * @example
   * // Block if used more than 30 minutes in the last hour
   * if (context.minutesUsedInPeriod(60) > 30) {
   *   return { enforcementAction: EnforcementAction.Block, enforcementReason: 'Usage limit exceeded' };
   * }
   */
  minutesUsedInPeriod(minutes: number): number;
}

// ============ Timezone Constants ============

/**
 * Common IANA timezone constants for use with now() and dayOfWeek().
 * Type Timezone. to see autocomplete suggestions.
 * @example
 * const londonTime = now(Timezone.Europe_London);
 * const tokyoDay = dayOfWeek(Timezone.Asia_Tokyo);
 */
declare const Timezone: {
  // Americas
  readonly America_New_York: "America/New_York";
  readonly America_Chicago: "America/Chicago";
  readonly America_Denver: "America/Denver";
  readonly America_Los_Angeles: "America/Los_Angeles";
  readonly America_Anchorage: "America/Anchorage";
  readonly America_Toronto: "America/Toronto";
  readonly America_Vancouver: "America/Vancouver";
  readonly America_Mexico_City: "America/Mexico_City";
  readonly America_Sao_Paulo: "America/Sao_Paulo";
  readonly America_Buenos_Aires: "America/Buenos_Aires";
  readonly America_Bogota: "America/Bogota";
  readonly America_Santiago: "America/Santiago";
  // Europe
  readonly Europe_London: "Europe/London";
  readonly Europe_Paris: "Europe/Paris";
  readonly Europe_Berlin: "Europe/Berlin";
  readonly Europe_Madrid: "Europe/Madrid";
  readonly Europe_Rome: "Europe/Rome";
  readonly Europe_Amsterdam: "Europe/Amsterdam";
  readonly Europe_Zurich: "Europe/Zurich";
  readonly Europe_Brussels: "Europe/Brussels";
  readonly Europe_Stockholm: "Europe/Stockholm";
  readonly Europe_Oslo: "Europe/Oslo";
  readonly Europe_Helsinki: "Europe/Helsinki";
  readonly Europe_Warsaw: "Europe/Warsaw";
  readonly Europe_Prague: "Europe/Prague";
  readonly Europe_Vienna: "Europe/Vienna";
  readonly Europe_Athens: "Europe/Athens";
  readonly Europe_Bucharest: "Europe/Bucharest";
  readonly Europe_Istanbul: "Europe/Istanbul";
  readonly Europe_Moscow: "Europe/Moscow";
  readonly Europe_Dublin: "Europe/Dublin";
  readonly Europe_Lisbon: "Europe/Lisbon";
  // Asia
  readonly Asia_Dubai: "Asia/Dubai";
  readonly Asia_Riyadh: "Asia/Riyadh";
  readonly Asia_Tehran: "Asia/Tehran";
  readonly Asia_Kolkata: "Asia/Kolkata";
  readonly Asia_Dhaka: "Asia/Dhaka";
  readonly Asia_Bangkok: "Asia/Bangkok";
  readonly Asia_Singapore: "Asia/Singapore";
  readonly Asia_Hong_Kong: "Asia/Hong_Kong";
  readonly Asia_Shanghai: "Asia/Shanghai";
  readonly Asia_Tokyo: "Asia/Tokyo";
  readonly Asia_Seoul: "Asia/Seoul";
  readonly Asia_Taipei: "Asia/Taipei";
  readonly Asia_Jakarta: "Asia/Jakarta";
  readonly Asia_Manila: "Asia/Manila";
  readonly Asia_Karachi: "Asia/Karachi";
  readonly Asia_Jerusalem: "Asia/Jerusalem";
  readonly Asia_Yerevan: "Asia/Yerevan";
  readonly Asia_Tbilisi: "Asia/Tbilisi";
  readonly Asia_Baku: "Asia/Baku";
  // Africa
  readonly Africa_Cairo: "Africa/Cairo";
  readonly Africa_Lagos: "Africa/Lagos";
  readonly Africa_Johannesburg: "Africa/Johannesburg";
  readonly Africa_Nairobi: "Africa/Nairobi";
  readonly Africa_Casablanca: "Africa/Casablanca";
  // Oceania
  readonly Australia_Sydney: "Australia/Sydney";
  readonly Australia_Melbourne: "Australia/Melbourne";
  readonly Australia_Perth: "Australia/Perth";
  readonly Australia_Brisbane: "Australia/Brisbane";
  readonly Pacific_Auckland: "Pacific/Auckland";
  readonly Pacific_Honolulu: "Pacific/Honolulu";
  // UTC
  readonly UTC: "UTC";
};

// ============ Weekday Constants ============

/**
 * Weekday enum values returned by dayOfWeek().
 * Use Weekday.Monday, Weekday.Tuesday, etc. for comparisons.
 * @example
 * if (dayOfWeek() === Weekday.Friday) { ... }
 */
declare const Weekday: {
  readonly Sunday: "Sunday";
  readonly Monday: "Monday";
  readonly Tuesday: "Tuesday";
  readonly Wednesday: "Wednesday";
  readonly Thursday: "Thursday";
  readonly Friday: "Friday";
  readonly Saturday: "Saturday";
};

type WeekdayType = "Sunday" | "Monday" | "Tuesday" | "Wednesday" | "Thursday" | "Friday" | "Saturday";

/**
 * Boolean constants for the current day of the week (local timezone).
 * For timezone-specific checks, use dayOfWeek(Timezone.X) === Weekday.Monday.
 * @example
 * if (IsMonday) { ... }
 * if (IsWeekend) { ... }
 */
declare const IsMonday: boolean;
declare const IsTuesday: boolean;
declare const IsWednesday: boolean;
declare const IsThursday: boolean;
declare const IsFriday: boolean;
declare const IsSaturday: boolean;
declare const IsSunday: boolean;
declare const IsWeekday: boolean;
declare const IsWeekend: boolean;

// ============ Global Helper Functions ============

/**
 * Returns a Date object for the current time in the specified IANA timezone.
 * Use Timezone.* constants for autocomplete, or pass any valid IANA timezone string.
 * If no timezone is provided or the string is invalid, uses local time.
 * @param timezone - IANA timezone (e.g. Timezone.Europe_London, Timezone.Asia_Tokyo)
 * @returns A Date object representing the current time
 * @example
 * const currentTime = now();
 * const londonTime = now(Timezone.Europe_London);
 * if (now(Timezone.America_New_York).getHours() >= 22) {
 *   // After 10 PM in New York
 * }
 */
declare function now(timezone?: string): Date;

/**
 * Returns the day of the week for the current time in the specified IANA timezone.
 * @param timezone - IANA timezone (e.g. Timezone.Europe_London)
 * @returns The day name: 'Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', or 'Saturday'
 * @example
 * if (dayOfWeek(Timezone.Europe_London) === 'Saturday' || dayOfWeek(Timezone.Europe_London) === 'Sunday') {
 *   // Weekend in London
 * }
 */
declare function dayOfWeek(timezone?: string): WeekdayType;

/**
 * Console logging (output appears in application logs).
 */
declare const console: {
  log(...args: unknown[]): void;
  info(...args: unknown[]): void;
  warn(...args: unknown[]): void;
  error(...args: unknown[]): void;
  debug(...args: unknown[]): void;
};
`;

const starterRulesTS = `/**
 * Custom classification logic.
 * Return a ClassificationDecision to override the default, or undefined to keep the default.
 *
 * @example
 * // Classify all GitHub activity as productive
 * if (context.domain === 'github.com') {
 *   return {
 *     classification: Classification.Productive,
 *     classificationReasoning: 'GitHub is a development tool'
 *   };
 * }
 */
export function classify(context: UsageContext): ClassificationDecision | undefined {
  return undefined;
}

/**
 * Custom termination logic (blocking).
 * Return a EnforcementDecision to override the default, or undefined to keep the default.
 *
 * @example
 * // Block social media after 10 PM in London
 * if (context.domain === 'twitter.com' && now(Timezone.Europe_London).getHours() >= 22) {
 *   return {
 *     enforcementAction: EnforcementAction.Block,
 *     enforcementReason: 'Social media blocked after 10 PM'
 *   };
 * }
 */
export function enforcementDecision(context: UsageContext): EnforcementDecision | undefined {
  return undefined;
}
`;

export function CustomRules() {
  const {
    customRules,
    updateSetting,
  } = useSettingsStore();

  const { checkoutLink, fetchAccountTier } = useAccountStore();
  const { data: accountTier } = useQuery({
    queryKey: ['accountTier'],
    queryFn: () => fetchAccountTier(),
  });

  const isFreeTier = accountTier === DeviceHandshakeResponse_AccountTier.DeviceHandshakeResponse_ACCOUNT_TIER_FREE;

  // Track unsaved draft changes - null means no local changes (use store value)
  const [draft, setDraft] = useState<string | null>(null);
  const [logsOpen, setLogsOpen] = useState(false);
  const [testOpen, setTestOpen] = useState(false);
  // Track whether to show the draft restoration banner
  const [showDraftBanner, setShowDraftBanner] = useState(false);
  const monacoRef = useRef<Monaco | null>(null);
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null);

  // Derive the displayed value: local draft takes precedence, then store, then starter template
  const displayedRules = draft ?? (customRules || starterRulesTS);
  const hasUnsavedChanges = draft !== null && draft !== customRules;

  // Load draft from localStorage on mount
  useEffect(() => {
    const savedDraft = localStorage.getItem(DRAFT_STORAGE_KEY);
    if (savedDraft) {
      const savedValue = customRules || starterRulesTS;
      // Only show banner if the draft differs from the current saved value
      if (savedDraft !== savedValue) {
        setShowDraftBanner(true);
      } else {
        // Draft matches saved value, clean it up
        localStorage.removeItem(DRAFT_STORAGE_KEY);
      }
    }
  }, [customRules]);

  // Save draft to localStorage whenever it changes
  useEffect(() => {
    if (draft !== null) {
      const savedValue = customRules || starterRulesTS;
      if (draft !== savedValue) {
        localStorage.setItem(DRAFT_STORAGE_KEY, draft);
      } else {
        // Draft matches saved value, clean it up
        localStorage.removeItem(DRAFT_STORAGE_KEY);
      }
    }
  }, [draft, customRules]);

  const handleRestoreDraft = useCallback(() => {
    const savedDraft = localStorage.getItem(DRAFT_STORAGE_KEY);
    if (savedDraft) {
      setDraft(savedDraft);
      setShowDraftBanner(false);
      toast.info("Draft restored. Click Save to apply changes.");
    }
  }, []);

  const handleDiscardDraft = useCallback(() => {
    localStorage.removeItem(DRAFT_STORAGE_KEY);
    setShowDraftBanner(false);
    toast.info("Draft discarded.");
  }, []);

  const handleChange = useCallback((value: string | undefined) => {
    setDraft(value ?? "");
  }, []);

  const handleSave = useCallback(async () => {
    // Early return if no changes to save
    if (draft === null || draft === customRules) {
      return;
    }

    try {
      await updateSetting(SETTINGS_KEY, draft);
      setDraft(null); // Clear draft after successful save
      localStorage.removeItem(DRAFT_STORAGE_KEY); // Clear localStorage draft
      setShowDraftBanner(false);
      toast.success("Custom rules saved successfully");
    } catch (error) {
      toast.error("Failed to save custom rules");
      console.error(error);
    }
  }, [draft, customRules, updateSetting]);

  // Ref to hold the latest save function for keybinding
  const saveRef = useRef(handleSave);
  saveRef.current = handleSave;

  const handleEditorMount = useCallback(
    (editor: editor.IStandaloneCodeEditor, monaco: Monaco) => {
      editorRef.current = editor;

      // Add Cmd+S / Ctrl+S keybinding for save
      editor.addAction({
        id: "save-custom-rules",
        label: "Save Custom Rules",
        keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS],
        run: () => {
          saveRef.current();
        },
      });
    },
    []
  );

  const handleEditorWillMount = useCallback((monaco: Monaco) => {
    monacoRef.current = monaco;

    // Add extraLib for intellisense
    monaco.languages.typescript.typescriptDefaults.addExtraLib(
      typeDefinitions,
      TYPES_FILE_PATH
    );

    // Create a model for the types file so Go to Definition works
    const typesUri = monaco.Uri.parse(TYPES_FILE_PATH);
    if (!monaco.editor.getModel(typesUri)) {
      monaco.editor.createModel(typeDefinitions, "typescript", typesUri);
    }
  }, []);

  return (
    <div className="flex flex-col h-full w-full gap-4 pb-2">
      {isFreeTier && (
        <div className="flex-none flex items-center justify-between p-3 rounded-xl bg-violet-500/10 border border-violet-500/20 text-violet-200/90 text-[13px]">
          <div className="flex items-center gap-2.5">
            <div className="p-1 rounded-md bg-violet-500/20">
              <IconCrown className="w-3.5 h-3.5" />
            </div>
            <span>Custom Rules are available on <strong>Plus</strong> or <strong>Pro</strong> plans. Upgrade to execute advanced logic.</span>
          </div>
          <Button
            onClick={() => checkoutLink && Browser.OpenURL(checkoutLink)}
            size="sm"
            className="h-7 px-3 bg-violet-600 hover:bg-violet-500 text-white text-[11px] font-bold rounded-lg transition-all"
          >
            Get Plus
          </Button>
        </div>
      )}

      <div className="flex-1 flex flex-col min-h-0 border rounded-lg bg-card overflow-hidden">
        {/* Integrated Toolbar */}
        <div className="flex items-center justify-between px-3 py-2 bg-muted/30 border-b border-border/50">
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1.5 px-2 py-1 rounded bg-background/50 border border-border/50 shadow-sm">
              <IconFileText className="w-4 h-4 text-muted-foreground" />
              <span className="text-xs font-medium">rules.ts</span>
            </div>
            {!isFreeTier && (
              <div className="flex items-center gap-1 px-2 py-0.5 rounded-full bg-violet-500/10 border border-violet-500/20 shadow-sm">
                <IconCrown className="w-3 h-3 text-violet-400" />
                <span className="text-[10px] font-bold text-violet-400 uppercase tracking-tight">Plus Feature</span>
              </div>
            )}
            {hasUnsavedChanges && (
              <div className="flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-primary/10 border border-primary/20">
                <span className="relative flex h-1.5 w-1.5">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75"></span>
                  <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-primary"></span>
                </span>
                <span className="text-[10px] uppercase tracking-wider font-bold text-primary">Unsaved</span>
              </div>
            )}
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => setLogsOpen(true)}
              className="inline-flex items-center gap-1.5 h-8 px-2 text-xs font-medium text-muted-foreground/60 hover:text-foreground hover:underline underline-offset-4 transition-colors disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:text-muted-foreground/60 disabled:hover:no-underline"
            >
              <IconTerminal className="w-3.5 h-3.5" />
              <span className="sr-only sm:not-sr-only">Exec Logs</span>
            </button>

            <button
              onClick={() => setTestOpen(true)}
              className="inline-flex items-center gap-1.5 h-8 px-2 text-xs font-medium text-muted-foreground/60 hover:text-foreground hover:underline underline-offset-4 transition-colors disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:text-muted-foreground/60 disabled:hover:no-underline"
            >
              <IconTestPipe className="w-3.5 h-3.5" />
              <span className="sr-only sm:not-sr-only">Test</span>
            </button>

            <Button
              size="sm"
              onClick={handleSave}
              disabled={!hasUnsavedChanges}
              className={cn(
                "h-8 px-3 transition-all duration-200",
                hasUnsavedChanges
                  ? "bg-emerald-600 text-white shadow-lg shadow-emerald-500/20 hover:bg-emerald-500"
                  : "bg-muted text-muted-foreground opacity-50 cursor-not-allowed"
              )}
            >
              <IconDeviceFloppy className="w-4 h-4 sm:mr-1.5" />
              <span className="text-xs font-bold sm:inline hidden">Save Rules</span>
            </Button>
          </div>
        </div>

        {/* Draft restoration banner */}
        {showDraftBanner && (
          <div className="flex items-center justify-between gap-3 px-4 py-2 bg-primary/5 border-b border-primary/10">
            <div className="flex items-center gap-2">
              <IconFileText className="w-4 h-4 text-primary/70" />
              <span className="text-xs text-muted-foreground">
                Restorable draft found from a previous session.
              </span>
            </div>
            <div className="flex items-center gap-2">
              <Button
                size="sm"
                variant="ghost"
                onClick={handleDiscardDraft}
                className="h-7 px-2 text-xs text-muted-foreground hover:text-destructive transition-colors"
              >
                Discard
              </Button>
              <Button
                size="sm"
                onClick={handleRestoreDraft}
                className="h-7 px-3 text-xs bg-primary/20 text-primary hover:bg-primary/30 border-none"
              >
                Restore Draft
              </Button>
            </div>
          </div>
        )}

        <div className="flex-1 min-h-[400px] user-select-allow bg-[#1e1e1e] relative">
          <Editor
            value={displayedRules}
            height="100%"
            language="typescript"
            theme="vs-dark"
            beforeMount={handleEditorWillMount}
            onMount={handleEditorMount}
            onChange={handleChange}
            options={{
              lineNumbers: "on",
              folding: true,
              renderLineHighlight: "line",
              minimap: { enabled: false },
              tabSize: 2,
              fontSize: 13,
              fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Source Code Pro', monospace",
              scrollBeyondLastLine: false,
              padding: { top: 10, bottom: 10 },
              overviewRulerBorder: false,
              hideCursorInOverviewRuler: true,
              definitionLinkOpensInPeek: true,
              scrollbar: {
                vertical: 'visible',
                horizontal: 'visible',
                useShadows: false,
                verticalScrollbarSize: 10,
                horizontalScrollbarSize: 10,
              }
            }}
          />
        </div>

        <ExecutionLogsSheet open={logsOpen} onOpenChange={setLogsOpen} />
        <TestRulesSheet open={testOpen} onOpenChange={setTestOpen} />
      </div>
    </div>
  );
}
