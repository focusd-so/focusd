import { ShieldCheck } from "lucide-react";
import { Browser } from "@wailsio/runtime";

interface PrivacyNoteProps {
  className?: string;
  style?: React.CSSProperties;
}

export function PrivacyNote({ className = "", style }: PrivacyNoteProps) {
  return (
    <div
      className={`w-full flex items-center gap-3 rounded-xl px-4 py-3 ${className}`}
      style={{
        background: "rgba(255, 255, 255, 0.02)",
        border: "1px solid rgba(255,255,255,0.06)",
        ...style,
      }}
    >
      <ShieldCheck className="w-4 h-4 text-indigo-400 shrink-0" />
      <p className="text-xs text-zinc-400 leading-snug text-left">
        <span className="text-zinc-300 font-medium">Privacy First</span> — All
        data stays on your Mac. When needed, it's processed through our
        encrypted server to AI models with zero retention — never stored,
        shared, or used for training. Focusd is source available on{" "}
        <button
          onClick={() => Browser.OpenURL("https://github.com/focusd-so/focusd")}
          className="text-indigo-400 hover:text-indigo-300 underline cursor-pointer"
        >
          GitHub
        </button>
        .
      </p>
    </div>
  );
}
