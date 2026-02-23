import { useCallback, useEffect, useRef, useState } from "react";
import Editor, { type Monaco } from "@monaco-editor/react";
import type { editor } from "monaco-editor";
import { useSettingsStore } from "@/stores/settings-store";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { IconDeviceFloppy, IconHistory, IconFileText } from "@tabler/icons-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

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
type TerminationModeType = "none" | "block" | "paused" | "allow";

/**
 * Global constant for termination mode values.
 * Use these values when returning a TerminationDecision.
 * @example
 * return {
 *   terminationMode: TerminationMode.Block,
 *   terminationReasoning: "Blocked during focus hours"
 * };
 */
declare const TerminationMode: {
  readonly None: "none";
  readonly Block: "block";
  readonly Paused: "paused";
  readonly Allow: "allow";
};

/**
 * Decision returned from the terminationMode function.
 */
interface TerminationDecision {
  /** The termination mode to apply. Use TerminationMode constants. */
  terminationMode: TerminationModeType;
  /** Human-readable explanation for why this decision was made. */
  terminationReasoning: string;
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
interface Context {
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
   * if (ctx.minutesUsedInPeriod(60) > 30) {
   *   return { terminationMode: TerminationMode.Block, terminationReasoning: 'Usage limit exceeded' };
   * }
   */
  minutesUsedInPeriod(minutes: number): number;
}

// ============ Global Helper Functions ============

/**
 * Returns a Date object for the current time, optionally shifted to a specific country's timezone.
 * If no country code is provided or the code is invalid, uses local time.
 * @param countryCode - Optional 2-letter ISO country code (e.g., 'US', 'JP', 'GB')
 * @returns A Date object representing the current time
 * @example
 * const currentTime = now();
 * const tokyoTime = now('JP');
 * if (now().getHours() >= 22) {
 *   // After 10 PM
 * }
 */
declare function now(countryCode?: string): Date;

/**
 * Returns the day of the week for the current time, optionally in a specific country's timezone.
 * @param countryCode - Optional 2-letter ISO country code (e.g., 'US', 'JP', 'GB')
 * @returns The day name: 'Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', or 'Saturday'
 * @example
 * if (dayOfWeek() === 'Saturday' || dayOfWeek() === 'Sunday') {
 *   // Weekend logic
 * }
 */
declare function dayOfWeek(countryCode?: string): "Sunday" | "Monday" | "Tuesday" | "Wednesday" | "Thursday" | "Friday" | "Saturday";

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
 * if (ctx.domain === 'github.com') {
 *   return {
 *     classification: Classification.Productive,
 *     classificationReasoning: 'GitHub is a development tool'
 *   };
 * }
 */
export function classify(ctx: Context): ClassificationDecision | undefined {
  return undefined;
}

/**
 * Custom termination logic (blocking).
 * Return a TerminationDecision to override the default, or undefined to keep the default.
 *
 * @example
 * // Block social media after 10 PM
 * if (ctx.domain === 'twitter.com' && now().getHours() >= 22) {
 *   return {
 *     terminationMode: TerminationMode.Block,
 *     terminationReasoning: 'Social media blocked after 10 PM'
 *   };
 * }
 */
export function terminationMode(ctx: Context): TerminationDecision | undefined {
  return undefined;
}
`;

function formatDate(timestamp: number): string {
  const date = new Date(timestamp * 1000);
  return date.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function CustomRules() {
  const {
    customRules,
    customRulesHistory,
    updateSetting,
    fetchCustomRulesHistory,
  } = useSettingsStore();

  // Track unsaved draft changes - null means no local changes (use store value)
  const [draft, setDraft] = useState<string | null>(null);
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

  const handleHistoryOpen = (open: boolean) => {
    if (open) {
      fetchCustomRulesHistory(10);
    }
  };

  const handleRestoreVersion = (value: string) => {
    setDraft(value);
    toast.info("Version restored. Click Save to apply changes.");
  };

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
    <div className="flex flex-col h-full w-full border rounded-lg bg-card overflow-hidden">
      {/* Integrated Toolbar */}
      <div className="flex items-center justify-between px-3 py-2 bg-muted/30 border-b border-border/50">
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1.5 px-2 py-1 rounded bg-background/50 border border-border/50 shadow-sm">
            <IconFileText className="w-4 h-4 text-muted-foreground" />
            <span className="text-xs font-medium">rules.ts</span>
          </div>
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
          <DropdownMenu onOpenChange={handleHistoryOpen}>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className="h-8 px-2 text-muted-foreground hover:text-foreground transition-colors"
              >
                <IconHistory className="w-4 h-4" />
                <span className="text-xs font-medium sr-only sm:not-sr-only ml-1.5">History</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-64">
              <DropdownMenuLabel className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Version History</DropdownMenuLabel>
              <DropdownMenuSeparator />
              {customRulesHistory.length === 0 ? (
                <DropdownMenuItem disabled className="text-xs">No history available</DropdownMenuItem>
              ) : (
                customRulesHistory.map((version, index) => (
                  <DropdownMenuItem
                    key={version.id}
                    onClick={() => handleRestoreVersion(version.value)}
                    className="flex flex-col items-start gap-0.5 py-2"
                  >
                    <div className="flex items-center justify-between w-full">
                      <span className="font-semibold text-sm">Version {version.version}</span>
                      <span className="text-[10px] text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
                        {index === 0 && version.value === customRules
                          ? "CURRENT"
                          : formatDate(version.created_at)}
                      </span>
                    </div>
                  </DropdownMenuItem>
                ))
              )}
            </DropdownMenuContent>
          </DropdownMenu>

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

      <div className="flex-1 min-h-[400px] user-select-allow bg-[#1e1e1e]">
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
    </div>
  );
}
