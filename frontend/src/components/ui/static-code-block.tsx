import { useEffect, useState } from "react";
import { loader } from "@monaco-editor/react";

export function StaticCodeBlock({ code, height = "auto" }: { code: string; height?: string }) {
  const [html, setHtml] = useState<string>('');

  useEffect(() => {
    let mounted = true;
    loader.init().then(monaco => {
      if (!mounted) return;
      monaco.editor.colorize(code, "typescript", { theme: "vs-dark" })
        .then((res: string) => {
          if (mounted) setHtml(res);
        });
    });
    return () => { mounted = false; };
  }, [code]);

  return (
    <div className="rounded-md border border-border/50 bg-[#1e1e1e] ring-offset-background focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-2 w-full text-left overflow-hidden flex flex-col min-w-0">
      <div className="overflow-x-auto overflow-y-hidden w-full">
        <div 
          className="p-3 text-[12px] font-mono leading-relaxed [&_div]:!whitespace-pre-wrap [&_div]:!break-all [&_span]:!whitespace-pre-wrap [&_span]:!break-all" 
          style={{ minHeight: height, color: '#d4d4d4' }}
          dangerouslySetInnerHTML={{ __html: html || code.replace(/</g, "&lt;").replace(/\n/g, "<br/>") }}
        />
      </div>
    </div>
  );
}
