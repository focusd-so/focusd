import { useMemo, useState } from "react";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { RULE_SNIPPETS } from "@/lib/rules/snippets";
import { StaticCodeBlock } from "@/components/ui/static-code-block";
import { toast } from "sonner";
import { IconSearch, IconCheck, IconCopy, IconBulb } from "@tabler/icons-react";

export function RulesReferenceSheet({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [search, setSearch] = useState("");
  const [copiedId, setCopiedId] = useState<string | null>(null);
  
  const examples = useMemo(() => {
    let filtered = RULE_SNIPPETS.filter((snippet) => snippet.id !== "import-runtime");
    if (search) {
      const q = search.toLowerCase();
      filtered = filtered.filter(s => 
        s.title.toLowerCase().includes(q) || 
        s.description.toLowerCase().includes(q) ||
        s.code.toLowerCase().includes(q)
      );
    }
    return filtered;
  }, [search]);

  const copySnippet = async (id: string, code: string) => {
    try {
      await navigator.clipboard.writeText(code);
      setCopiedId(id);
      setTimeout(() => setCopiedId(null), 2000);
      toast.success("Example copied to clipboard");
    } catch {
      toast.error("Could not copy example");
    }
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange} modal={false}>
      <SheetContent side="right" className="w-full sm:max-w-[450px] p-0 flex flex-col shadow-2xl border-l border-border/50 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80" modal={false}>
        <SheetHeader className="px-4 pt-4 pb-3 border-b border-border/30 shrink-0 bg-background/50">
          <SheetTitle className="text-sm">Docs & Examples</SheetTitle>
          <SheetDescription className="text-xs">
            Runtime API reference and ready-to-use rule snippets
          </SheetDescription>
        </SheetHeader>

        <Tabs defaultValue="examples" className="flex-1 min-h-0 flex flex-col w-full">
          <div className="px-4 py-3 border-b border-border/30 shrink-0 bg-background/50 space-y-3">
            <TabsList className="grid w-full grid-cols-2 h-8">
              <TabsTrigger value="examples" className="text-xs">Examples</TabsTrigger>
              <TabsTrigger value="docs" className="text-xs">Reference</TabsTrigger>
            </TabsList>
            
            <div className="relative">
              <IconSearch className="absolute left-2.5 top-2 h-3.5 w-3.5 text-muted-foreground" />
              <Input
                type="search"
                placeholder="Search snippets and docs..."
                className="h-8 pl-8 text-xs bg-muted/50 border-border/50 focus-visible:ring-1 focus-visible:ring-primary/30"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
          </div>

          <TabsContent value="examples" className="m-0 flex-1 min-h-0 bg-muted/10 w-full overflow-hidden data-[state=active]:flex flex-col">
            <ScrollArea className="flex-1 min-h-0 w-full">
              <div className="p-4 space-y-4">
                {examples.length === 0 && (
                  <div className="text-center py-8 text-muted-foreground text-xs">
                    No examples found matching "{search}"
                  </div>
                )}
                
                {examples.map((example) => {
                  return (
                    <div 
                      key={example.id} 
                      className="group flex flex-col rounded-lg border border-border/50 bg-card p-3 space-y-3 shadow-sm hover:shadow-md hover:border-border transition-all duration-200 w-full min-w-0 overflow-hidden"
                    >
                      <div>
                        <p className="text-sm font-semibold text-foreground">{example.title}</p>
                        <p className="text-[11px] text-muted-foreground mt-0.5">{example.description}</p>
                      </div>
                      
                      <div className="w-full min-w-0 overflow-hidden rounded-md border border-border/50">
                        <StaticCodeBlock code={example.code} />
                      </div>
                      
                      <div className="flex items-center gap-2 pt-1 opacity-100 sm:opacity-50 group-hover:opacity-100 transition-opacity">
                        <Button
                          size="sm"
                          variant="outline"
                          className="flex-1 h-7 text-[11px] text-muted-foreground hover:text-foreground transition-all"
                          onClick={() => copySnippet(example.id, example.code)}
                        >
                          {copiedId === example.id ? (
                            <><IconCheck className="w-3.5 h-3.5 mr-1.5 text-green-500" /> Copied</>
                          ) : (
                            <><IconCopy className="w-3.5 h-3.5 mr-1.5" /> Copy</>
                          )}
                        </Button>
                      </div>
                    </div>
                  );
                })}
              </div>
            </ScrollArea>
          </TabsContent>

          <TabsContent value="docs" className="m-0 flex-1 min-h-0 bg-muted/10 w-full overflow-hidden data-[state=active]:flex flex-col">
            <ScrollArea className="flex-1 min-h-0 w-full">
              <div className="p-4 space-y-6 text-sm w-full">
                
                {(!search || "core concepts classify enforce".includes(search.toLowerCase())) && (
                  <div className="space-y-3">
                    <h3 className="font-semibold text-foreground flex items-center gap-2">
                      <div className="h-5 w-1 rounded-full bg-yellow-500"></div>
                      Core Concepts
                    </h3>
                    
                    <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-md p-3 space-y-2">
                      <div className="flex gap-2">
                        <IconBulb className="w-4 h-4 text-yellow-600 dark:text-yellow-400 shrink-0 mt-0.5" />
                        <div className="space-y-2">
                          <p className="text-xs font-medium text-yellow-800 dark:text-yellow-300">
                            Classify vs. Enforce
                          </p>
                          <p className="text-[11px] leading-relaxed text-yellow-900/80 dark:text-yellow-200/80">
                            <strong>Classification</strong> (<code className="bg-yellow-500/20 px-1 rounded">productive</code>, <code className="bg-yellow-500/20 px-1 rounded">distracting</code>, <code className="bg-yellow-500/20 px-1 rounded">neutral</code>) feeds your charts and stats. It determines <em>what</em> the usage is.
                          </p>
                          <p className="text-[11px] leading-relaxed text-yellow-900/80 dark:text-yellow-200/80">
                            <strong>Enforcement</strong> (<code className="bg-yellow-500/20 px-1 rounded">block</code>, <code className="bg-yellow-500/20 px-1 rounded">allow</code>, <code className="bg-yellow-500/20 px-1 rounded">pause</code>) controls the actual access. It determines <em>what to do</em> about the usage.
                          </p>
                          <p className="text-[11px] leading-relaxed text-yellow-900/80 dark:text-yellow-200/80 mt-1 font-medium">
                            Don't use blocks when you just want to track something as distracting!
                          </p>
                        </div>
                      </div>
                    </div>
                  </div>
                )}
                
                {(!search || "import".includes(search.toLowerCase())) && (
                  <div className="space-y-2">
                    <h3 className="font-semibold text-foreground flex items-center gap-2">
                      <div className="h-5 w-1 rounded-full bg-blue-500"></div>
                      Importing SDK
                    </h3>
                    <p className="text-xs text-muted-foreground leading-relaxed">
                      All runtime functions and types must be imported from the <code className="bg-muted px-1 py-0.5 rounded">@focusd/runtime</code> module.
                    </p>
                    <div className="w-full min-w-0 overflow-hidden rounded-md border border-border/50">
                      <StaticCodeBlock 
                        code={`import {
  productive, distracting, neutral,
  block, allow, pause,
  Timezone, runtime
} from "@focusd/runtime";`} 
                      />
                    </div>
                  </div>
                )}

                {(!search || "classification actions".includes(search.toLowerCase())) && (
                  <div className="space-y-2">
                    <h3 className="font-semibold text-foreground flex items-center gap-2">
                      <div className="h-5 w-1 rounded-full bg-emerald-500"></div>
                      Classification Actions
                    </h3>
                    <p className="text-xs text-muted-foreground leading-relaxed">
                      Return these from your script to categorize the current activity.
                    </p>
                    <div className="grid grid-cols-1 gap-2 w-full">
                      <div className="bg-card border border-border/50 rounded p-2 text-xs font-mono break-all w-full overflow-hidden text-ellipsis whitespace-nowrap">
                        productive(reason: string, tags?: string[])
                      </div>
                      <div className="bg-card border border-border/50 rounded p-2 text-xs font-mono break-all w-full overflow-hidden text-ellipsis whitespace-nowrap">
                        distracting(reason: string, tags?: string[])
                      </div>
                      <div className="bg-card border border-border/50 rounded p-2 text-xs font-mono break-all w-full overflow-hidden text-ellipsis whitespace-nowrap">
                        neutral(reason: string, tags?: string[])
                      </div>
                    </div>
                  </div>
                )}

                {(!search || "enforcement actions".includes(search.toLowerCase())) && (
                  <div className="space-y-2">
                    <h3 className="font-semibold text-foreground flex items-center gap-2">
                      <div className="h-5 w-1 rounded-full bg-red-500"></div>
                      Enforcement Actions
                    </h3>
                    <p className="text-xs text-muted-foreground leading-relaxed">
                      Return these to actively control the user's access.
                    </p>
                    <div className="grid grid-cols-1 gap-2 w-full">
                      <div className="bg-card border border-border/50 rounded p-2 text-xs font-mono w-full overflow-hidden text-ellipsis whitespace-nowrap">
                        block(reason: string)
                      </div>
                      <div className="bg-card border border-border/50 rounded p-2 text-xs font-mono w-full overflow-hidden text-ellipsis whitespace-nowrap">
                        allow(reason: string)
                      </div>
                      <div className="bg-card border border-border/50 rounded p-2 text-xs font-mono w-full overflow-hidden text-ellipsis whitespace-nowrap">
                        pause(reason: string)
                      </div>
                    </div>
                  </div>
                )}

                {(!search || "context runtime identity app url".includes(search.toLowerCase())) && (
                  <div className="space-y-2 w-full min-w-0">
                    <h3 className="font-semibold text-foreground flex items-center gap-2">
                      <div className="h-5 w-1 rounded-full bg-purple-500"></div>
                      Activity Context
                    </h3>
                    <p className="text-xs text-muted-foreground leading-relaxed">
                      Properties available under <code className="bg-muted px-1 py-0.5 rounded text-purple-400">runtime.usage</code> describing what the user is currently looking at.
                    </p>
                    <ul className="space-y-2 text-xs font-mono w-full">
                      <li className="flex flex-col gap-1 border-b border-border/30 pb-2 w-full min-w-0">
                        <span className="text-foreground">.app</span>
                        <span className="text-muted-foreground font-sans w-full truncate">Name of the active application (e.g. "Google Chrome")</span>
                      </li>
                      <li className="flex flex-col gap-1 border-b border-border/30 pb-2 w-full min-w-0">
                        <span className="text-foreground">.title</span>
                        <span className="text-muted-foreground font-sans w-full truncate">Window title of the active app</span>
                      </li>
                      <li className="flex flex-col gap-1 border-b border-border/30 pb-2 w-full min-w-0">
                        <span className="text-foreground break-all">.url / .domain / .host / .path</span>
                        <span className="text-muted-foreground font-sans w-full truncate">Browser URL components (only for supported browsers)</span>
                      </li>
                    </ul>
                  </div>
                )}

                {(!search || "history stats today hour metrics".includes(search.toLowerCase())) && (
                  <div className="space-y-2 w-full min-w-0">
                    <h3 className="font-semibold text-foreground flex items-center gap-2">
                      <div className="h-5 w-1 rounded-full bg-amber-500"></div>
                      Usage History & Metrics
                    </h3>
                    <p className="text-xs text-muted-foreground leading-relaxed">
                      Metrics available under <code className="bg-muted px-1 py-0.5 rounded text-amber-400 break-all">runtime.today</code> and <code className="bg-muted px-1 py-0.5 rounded text-amber-400 break-all">runtime.hour</code>.
                    </p>
                    <ul className="space-y-2 text-xs font-mono w-full">
                      <li className="flex flex-col gap-1 border-b border-border/30 pb-2 w-full min-w-0">
                        <span className="text-foreground break-all">.focusScore</span>
                        <span className="text-muted-foreground font-sans w-full truncate">Current focus score (0-100)</span>
                      </li>
                      <li className="flex flex-col gap-1 border-b border-border/30 pb-2 w-full min-w-0">
                        <span className="text-foreground break-all">.productiveMinutes</span>
                        <span className="text-muted-foreground font-sans w-full truncate">Total minutes spent on productive tasks</span>
                      </li>
                      <li className="flex flex-col gap-1 border-b border-border/30 pb-2 w-full min-w-0">
                        <span className="text-foreground break-all">.distractingMinutes</span>
                        <span className="text-muted-foreground font-sans w-full truncate">Total minutes spent on distracting tasks</span>
                      </li>
                    </ul>
                  </div>
                )}
                
                {(!search || "current stats usage blocks".includes(search.toLowerCase())) && (
                  <div className="space-y-2 w-full min-w-0">
                    <h3 className="font-semibold text-foreground flex items-center gap-2">
                      <div className="h-5 w-1 rounded-full bg-cyan-500"></div>
                      Current App/Site Stats
                    </h3>
                    <p className="text-xs text-muted-foreground leading-relaxed">
                      Specific metrics for the exact app/domain currently active under <code className="bg-muted px-1 py-0.5 rounded text-cyan-400 break-all">runtime.usage.current</code>.
                    </p>
                    <ul className="space-y-2 text-xs font-mono w-full">
                      <li className="flex flex-col gap-1 border-b border-border/30 pb-2 w-full min-w-0">
                        <span className="text-foreground break-all">.usedToday</span>
                        <span className="text-muted-foreground font-sans w-full truncate">Minutes spent on this specific app/site today</span>
                      </li>
                      <li className="flex flex-col gap-1 border-b border-border/30 pb-2 w-full min-w-0">
                        <span className="text-foreground break-all">.blocks</span>
                        <span className="text-muted-foreground font-sans w-full truncate">Number of times this app/site was blocked today</span>
                      </li>
                      <li className="flex flex-col gap-1 border-b border-border/30 pb-2 w-full min-w-0">
                        <span className="text-foreground break-all">.sinceBlock</span>
                        <span className="text-muted-foreground font-sans w-full truncate">Minutes elapsed since the last block event</span>
                      </li>
                      <li className="flex flex-col gap-1 border-b border-border/30 pb-2 w-full min-w-0">
                        <span className="text-foreground break-all">.usedSinceBlock</span>
                        <span className="text-muted-foreground font-sans w-full truncate">Minutes spent actively using this app/site since it was last blocked</span>
                      </li>
                      <li className="flex flex-col gap-1 border-b border-border/30 pb-2 w-full min-w-0">
                        <span className="text-foreground break-all">.last(minutes)</span>
                        <span className="text-muted-foreground font-sans w-full truncate">Usage in the last N minutes</span>
                      </li>
                    </ul>
                  </div>
                )}
              </div>
            </ScrollArea>
          </TabsContent>
        </Tabs>
      </SheetContent>
    </Sheet>
  );
}
