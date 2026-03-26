import { useMemo } from "react";
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
import { RULE_SNIPPETS } from "@/lib/rules/snippets";
import { CodeBlock } from "@/components/ui/code-block";
import { toast } from "sonner";

export function RulesReferenceSheet({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const examples = useMemo(
    () => RULE_SNIPPETS.filter((snippet) => snippet.id !== "import-runtime"),
    []
  );

  const copySnippet = async (code: string) => {
    try {
      await navigator.clipboard.writeText(code);
      toast.success("Example copied");
    } catch {
      toast.error("Could not copy example");
    }
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-lg p-0 flex flex-col">
        <SheetHeader className="px-4 pt-4 pb-3 border-b border-border/50 shrink-0">
          <SheetTitle className="text-sm">Docs & Examples</SheetTitle>
          <SheetDescription className="text-xs">
            Runtime API reference and ready-to-use rule snippets
          </SheetDescription>
        </SheetHeader>

        <Tabs defaultValue="docs" className="flex-1 min-h-0 flex flex-col">
          <div className="px-4 py-3 border-b border-border/30 shrink-0">
            <TabsList className="grid w-full grid-cols-2 h-8">
              <TabsTrigger value="docs" className="text-xs">Docs</TabsTrigger>
              <TabsTrigger value="examples" className="text-xs">Examples</TabsTrigger>
            </TabsList>
          </div>

          <TabsContent value="docs" className="m-0 flex-1 min-h-0">
            <ScrollArea className="h-full">
              <div className="p-4 space-y-4 text-xs">
                <div className="space-y-1">
                  <p className="font-semibold text-foreground">Import</p>
                  <CodeBlock 
                    height="120px"
                    code={`import {
  productive, distracting, neutral,
  block, allow, pause,
  Timezone, runtime,
  type Classify, type Enforce,
} from "@focusd/runtime";`} 
                  />
                </div>

                <div className="space-y-1">
                  <p className="font-semibold text-foreground">Function Signature</p>
                  <CodeBlock 
                    height="50px"
                    code={`export function classify(): Classify | undefined;
export function enforcement(): Enforce | undefined;`} 
                  />
                </div>

                <div className="space-y-1 mt-2">
                  <p className="font-semibold text-foreground">Identity</p>
                  <ul className="space-y-1 text-muted-foreground font-mono text-[11px]">
                    <li>runtime.usage.app / title / domain / host / path / url</li>
                    <li>runtime.usage.classification</li>
                  </ul>
                </div>

                <div className="space-y-1">
                  <p className="font-semibold text-foreground">Day & Hour (Runtime)</p>
                  <ul className="space-y-1 text-muted-foreground font-mono text-[11px]">
                    <li>runtime.today.focusScore / runtime.today.productiveMinutes / runtime.today.distractingMinutes</li>
                    <li>runtime.hour.focusScore / runtime.hour.productiveMinutes / runtime.hour.distractingMinutes</li>
                  </ul>
                </div>

                <div className="space-y-1">
                  <p className="font-semibold text-foreground">Current App/Site</p>
                  <ul className="space-y-1 text-muted-foreground font-mono text-[11px]">
                    <li>runtime.usage.current.usedToday</li>
                    <li>runtime.usage.current.blocks</li>
                    <li>runtime.usage.current.sinceBlock</li>
                    <li>runtime.usage.current.usedSinceBlock</li>
                    <li>runtime.usage.current.last(60)</li>
                  </ul>
                </div>

                <div className="space-y-1">
                  <p className="font-semibold text-foreground">Migration</p>
                  <ul className="space-y-1 text-muted-foreground font-mono text-[11px]">
                    <li>Old style classify(usage) / enforcement(usage) is no longer supported.</li>
                  </ul>
                </div>

                <div className="space-y-1">
                  <p className="font-semibold text-foreground">Time (Runtime)</p>
                  <ul className="space-y-1 text-muted-foreground font-mono text-[11px]">
                    <li>runtime.time.now(Timezone.UTC)</li>
                    <li>runtime.time.day(Timezone.Europe_London)</li>
                  </ul>
                </div>

                <div className="space-y-1">
                  <p className="font-semibold text-foreground">Helpers</p>
                  <ul className="space-y-1 text-muted-foreground font-mono text-[11px]">
                    <li>productive(reason, tags?)</li>
                    <li>distracting(reason, tags?)</li>
                    <li>neutral(reason, tags?)</li>
                    <li>block(reason)</li>
                    <li>allow(reason)</li>
                    <li>pause(reason)</li>
                  </ul>
                </div>
              </div>
            </ScrollArea>
          </TabsContent>

          <TabsContent value="examples" className="m-0 flex-1 min-h-0">
            <ScrollArea className="h-full">
              <div className="p-4 space-y-3">
                {examples.map((example) => {
                  const lineCount = example.code.split('\n').length;
                  const estimatedHeight = Math.max(50, lineCount * 22) + 20 + "px"; // 22px per line + padding

                  return (
                    <div key={example.id} className="rounded-lg border border-border/50 bg-card/50 p-3 space-y-2">
                      <div>
                        <p className="text-xs font-semibold text-foreground">{example.title}</p>
                        <p className="text-[11px] text-muted-foreground">{example.description}</p>
                      </div>
                      <CodeBlock code={example.code} height={estimatedHeight} />
                      <Button
                        size="sm"
                        variant="outline"
                        className="h-7 text-[11px] mt-1"
                        onClick={() => copySnippet(example.code)}
                      >
                        Copy Snippet
                      </Button>
                    </div>
                  );
                })}
              </div>
            </ScrollArea>
          </TabsContent>
        </Tabs>
      </SheetContent>
    </Sheet>
  );
}
