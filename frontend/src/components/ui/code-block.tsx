import Editor, { type Monaco } from "@monaco-editor/react";
import { useCallback } from "react";
import { RUNTIME_TYPES_FILE_PATH, fetchRuntimeTypes } from "@/lib/rules/runtime-types";

export function CodeBlock({ code, height = "100px" }: { code: string; height?: string }) {
  const handleEditorWillMount = useCallback(async (monaco: Monaco) => {
    const typesSource = await fetchRuntimeTypes();
    monaco.languages.typescript.typescriptDefaults.addExtraLib(typesSource, RUNTIME_TYPES_FILE_PATH);

    const typesUri = monaco.Uri.parse(RUNTIME_TYPES_FILE_PATH);
    if (!monaco.editor.getModel(typesUri)) {
      monaco.editor.createModel(typesSource, "typescript", typesUri);
    }
  }, []);

  return (
    <div className="rounded-md overflow-hidden border border-border/50 bg-[#1e1e1e] ring-offset-background focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-2">
      <Editor
        height={height}
        language="typescript"
        theme="vs-dark"
        value={code}
        beforeMount={handleEditorWillMount}
        options={{
          readOnly: true,
          minimap: { enabled: false },
          scrollBeyondLastLine: false,
          padding: { top: 12, bottom: 12 },
          lineNumbers: "off",
          folding: false,
          renderLineHighlight: "none",
          scrollbar: {
            vertical: "hidden",
            horizontal: "hidden",
            handleMouseWheel: false,
          },
          overviewRulerBorder: false,
          overviewRulerLanes: 0,
          hideCursorInOverviewRuler: true,
          matchBrackets: "never",
          renderValidationDecorations: "on",
          contextmenu: false,
        }}
      />
    </div>
  );
}
