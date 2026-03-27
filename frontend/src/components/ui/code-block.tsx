import Editor, { type Monaco } from "@monaco-editor/react";
import { useCallback } from "react";
import { RUNTIME_TYPES_FILE_PATH, RUNTIME_TYPES_SOURCE } from "@/lib/rules/runtime-types";

export function CodeBlock({ code, height = "100px" }: { code: string; height?: string }) {
  const handleEditorWillMount = useCallback((monaco: Monaco) => {
    monaco.languages.typescript.typescriptDefaults.addExtraLib(RUNTIME_TYPES_SOURCE, RUNTIME_TYPES_FILE_PATH);

    const typesUri = monaco.Uri.parse(RUNTIME_TYPES_FILE_PATH);
    if (!monaco.editor.getModel(typesUri)) {
      monaco.editor.createModel(RUNTIME_TYPES_SOURCE, "typescript", typesUri);
    }
  }, []);

  return (
    <div className="rounded-md overflow-hidden border border-border/50 bg-[#1e1e1e] ring-offset-background focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-2">
      <Editor
        value={code}
        height={height}
        language="typescript"
        theme="vs-dark"
        beforeMount={handleEditorWillMount}
        options={{
          readOnly: true,
          minimap: { enabled: false },
          lineNumbers: "off",
          scrollBeyondLastLine: false,
          folding: false,
          wordWrap: "on",
          padding: { top: 8, bottom: 8 },
          fontSize: 12,
          fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Source Code Pro', monospace",
          renderLineHighlight: "none",
          hideCursorInOverviewRuler: true,
          overviewRulerBorder: false,
          scrollbar: {
            vertical: "hidden",
            horizontal: "hidden"
          }
        }}
      />
    </div>
  );
}
