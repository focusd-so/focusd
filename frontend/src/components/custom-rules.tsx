import { useCallback, useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import Editor, { type Monaco } from "@monaco-editor/react";
import type { editor } from "monaco-editor";
import { useSettingsStore } from "@/stores/settings-store";
import { useAccountStore } from "@/stores/account-store";
import { DeviceHandshakeResponse_AccountTier } from "../../bindings/github.com/focusd-so/focusd/gen/api/v1/models";
import { Browser } from "@wailsio/runtime";
import { Button } from "@/components/ui/button";
import { IconBook, IconCrown, IconFileText, IconTerminal, IconTestPipe } from "@tabler/icons-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger, TooltipProvider } from "@/components/ui/tooltip";
import { ExecutionLogsSheet } from "@/components/execution-logs";
import { TestRulesSheet } from "@/components/test-rules-sheet";
import { RulesReferenceSheet } from "@/components/rules-reference-sheet";
import { RUNTIME_TYPES_FILE_PATH, fetchRuntimeTypes } from "@/lib/rules/runtime-types";

const SETTINGS_KEY = "custom_rules";
const DRAFT_STORAGE_KEY = "focusd_custom_rules_draft";

export function CustomRules() {
  const { customRules, updateSetting } = useSettingsStore();
  const { checkoutLink, fetchAccountTier } = useAccountStore();
  const { data: accountTier } = useQuery({
    queryKey: ["accountTier"],
    queryFn: () => fetchAccountTier(),
  });

  const isFreeTier = accountTier === DeviceHandshakeResponse_AccountTier.DeviceHandshakeResponse_ACCOUNT_TIER_FREE;
  const [draft, setDraft] = useState<string | null>(null);
  const [logsOpen, setLogsOpen] = useState(false);
  const [testOpen, setTestOpen] = useState(false);
  const [referenceOpen, setReferenceOpen] = useState(false);
  const [showDraftBanner, setShowDraftBanner] = useState(false);
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null);

  const displayedRules = draft ?? customRules;
  const hasUnsavedChanges = draft !== null && draft !== customRules;

  useEffect(() => {
    const savedDraft = localStorage.getItem(DRAFT_STORAGE_KEY);
    if (!savedDraft) {
      return;
    }

    const savedValue = customRules;
    if (savedDraft !== savedValue) {
      setShowDraftBanner(true);
      return;
    }

    localStorage.removeItem(DRAFT_STORAGE_KEY);
  }, [customRules]);

  useEffect(() => {
    if (draft === null) {
      return;
    }

    const savedValue = customRules;
    if (draft !== savedValue) {
      localStorage.setItem(DRAFT_STORAGE_KEY, draft);
      return;
    }

    localStorage.removeItem(DRAFT_STORAGE_KEY);
  }, [draft, customRules]);

  const handleRestoreDraft = useCallback(() => {
    const savedDraft = localStorage.getItem(DRAFT_STORAGE_KEY);
    if (!savedDraft) {
      return;
    }

    setDraft(savedDraft);
    setShowDraftBanner(false);
    toast.info("Draft restored. Press Cmd/Ctrl+S to apply changes.");
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
    if (draft === null || draft === customRules) {
      return;
    }

    try {
      await updateSetting(SETTINGS_KEY, draft);
      setDraft(null);
      localStorage.removeItem(DRAFT_STORAGE_KEY);
      setShowDraftBanner(false);
      toast.success("Custom rules saved successfully");
    } catch (error) {
      toast.error("Failed to save custom rules");
      console.error(error);
    }
  }, [draft, customRules, updateSetting]);

  const saveRef = useRef(handleSave);
  saveRef.current = handleSave;

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "s") {
        event.preventDefault();
        saveRef.current();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, []);

  const handleEditorMount = useCallback((instance: editor.IStandaloneCodeEditor, monaco: Monaco) => {
    editorRef.current = instance;

    instance.addAction({
      id: "save-custom-rules",
      label: "Save Custom Rules",
      keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS],
      run: () => {
        saveRef.current();
      },
    });

    instance.onMouseDown(() => {
      setReferenceOpen(false);
      setTestOpen(false);
      setLogsOpen(false);
    });
  }, []);

  const handleEditorWillMount = useCallback(async (monaco: Monaco) => {
    const typesSource = await fetchRuntimeTypes();
    monaco.languages.typescript.typescriptDefaults.addExtraLib(typesSource, RUNTIME_TYPES_FILE_PATH);

    const typesUri = monaco.Uri.parse(RUNTIME_TYPES_FILE_PATH);
    if (!monaco.editor.getModel(typesUri)) {
      monaco.editor.createModel(typesSource, "typescript", typesUri);
    }
  }, []);

  return (
    <div className="flex flex-col h-full w-full pb-2 relative">
      <div className="flex-1 flex flex-col min-h-0 border rounded-lg bg-card overflow-hidden">
        <div className="flex items-center justify-between px-3 py-2 bg-muted/30 border-b border-border/50 z-10">
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1.5 px-2 py-1 rounded bg-background/50 border border-border/50 shadow-sm">
              <IconFileText className="w-4 h-4 text-muted-foreground" />
              <span className="text-xs font-medium">rules.ts</span>
            </div>
            {isFreeTier ? (
              <TooltipProvider>
                <Tooltip delayDuration={0}>
                  <TooltipTrigger asChild>
                    <button 
                      onClick={() => checkoutLink && Browser.OpenURL(checkoutLink)}
                      className="flex items-center gap-1 px-2 py-0.5 rounded-full bg-violet-500/10 border border-violet-500/20 shadow-sm hover:bg-violet-500/20 hover:border-violet-500/30 transition-colors cursor-pointer"
                    >
                      <IconCrown className="w-3 h-3 text-violet-400" />
                      <span className="text-[10px] font-bold text-violet-400 uppercase tracking-tight">Plus Feature</span>
                    </button>
                  </TooltipTrigger>
                  <TooltipContent side="bottom" className="max-w-[280px] p-3 border border-border/50 shadow-xl bg-popover text-popover-foreground z-50" sideOffset={8}>
                    <p className="font-medium text-[13px] mb-1">What does this mean?</p>
                    <p className="text-muted-foreground leading-relaxed text-[13px]">
                      Custom rules will execute and you can view their logs, but they won't enforce blocks or warnings unless you upgrade to a Plus or Pro plan.
                    </p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            ) : (
              <div className="flex items-center gap-1 px-2 py-0.5 rounded-full bg-violet-500/10 border border-violet-500/20 shadow-sm">
                <IconCrown className="w-3 h-3 text-violet-400" />
                <span className="text-[10px] font-bold text-violet-400 uppercase tracking-tight">Plus</span>
              </div>
            )}
            {hasUnsavedChanges && (
              <div className="flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-primary/10 border border-primary/20">
                <span className="relative flex h-1.5 w-1.5">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75" />
                  <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-primary" />
                </span>
                <span className="text-[10px] uppercase tracking-wider font-bold text-primary">Unsaved</span>
              </div>
            )}
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => setLogsOpen(true)}
              className="inline-flex items-center gap-1.5 h-8 px-2 text-xs font-medium text-muted-foreground/60 hover:text-foreground hover:underline underline-offset-4 transition-colors"
            >
              <IconTerminal className="w-3.5 h-3.5" />
              <span className="sr-only sm:not-sr-only">Exec Logs</span>
            </button>

            <button
              onClick={() => setTestOpen(true)}
              className="inline-flex items-center gap-1.5 h-8 px-2 text-xs font-medium text-muted-foreground/60 hover:text-foreground hover:underline underline-offset-4 transition-colors"
            >
              <IconTestPipe className="w-3.5 h-3.5" />
              <span className="sr-only sm:not-sr-only">Test</span>
            </button>

            <button
              onClick={() => setReferenceOpen(true)}
              className={cn(
                "inline-flex items-center gap-1.5 h-8 px-2 text-xs font-medium transition-colors",
                referenceOpen 
                  ? "text-primary bg-primary/10 rounded" 
                  : "text-muted-foreground/60 hover:text-foreground hover:underline underline-offset-4"
              )}
            >
              <IconBook className="w-3.5 h-3.5" />
              <span className="sr-only sm:not-sr-only">Examples</span>
            </button>

          </div>
        </div>

        {showDraftBanner && (
          <div className="flex items-center justify-between gap-3 px-4 py-2 bg-primary/5 border-b border-primary/10">
            <div className="flex items-center gap-2">
              <IconFileText className="w-4 h-4 text-primary/70" />
              <span className="text-xs text-muted-foreground">Restorable draft found from a previous session.</span>
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

        <div className="flex-1 min-h-[400px] flex user-select-allow bg-[#1e1e1e] relative">
          <div className="flex-1 min-h-[320px] h-full">
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
                fontSize: 12,
                fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Source Code Pro', monospace",
                scrollBeyondLastLine: false,
                padding: { top: 10, bottom: 10 },
                overviewRulerBorder: false,
                hideCursorInOverviewRuler: true,
                definitionLinkOpensInPeek: true,
                scrollbar: {
                  vertical: "visible",
                  horizontal: "visible",
                  useShadows: false,
                  verticalScrollbarSize: 10,
                  horizontalScrollbarSize: 10,
                },
              }}
            />
          </div>
        </div>

        <ExecutionLogsSheet open={logsOpen} onOpenChange={setLogsOpen} />
        <TestRulesSheet open={testOpen} onOpenChange={setTestOpen} />
        
        <RulesReferenceSheet 
          open={referenceOpen} 
          onOpenChange={setReferenceOpen} 
        />
      </div>
    </div>
  );
}
