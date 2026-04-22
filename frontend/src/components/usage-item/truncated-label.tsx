import React, { useState } from "react";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

function extractTextContent(node: React.ReactNode): string {
  if (typeof node === "string" || typeof node === "number") {
    return String(node);
  }
  if (node === null || node === undefined) {
    return "";
  }
  if (Array.isArray(node)) {
    return node.map(extractTextContent).join("");
  }
  if (typeof node === "object" && "props" in node) {
    const element = node as React.ReactElement<{ children?: React.ReactNode }>;
    return extractTextContent(element.props?.children);
  }
  return "";
}

export function TruncatedLabel({
  children,
  className = "",
}: React.PropsWithChildren<{ className?: string }>) {
  const [isTruncated, setIsTruncated] = useState(false);
  const textRef = React.useRef<HTMLSpanElement | null>(null);

  const textContent = extractTextContent(children);

  React.useEffect(() => {
    const checkTruncation = () => {
      if (!textRef.current) return;
      setIsTruncated(textRef.current.scrollWidth > textRef.current.clientWidth);
    };

    checkTruncation();
    const timeoutId = setTimeout(checkTruncation, 0);
    window.addEventListener("resize", checkTruncation);

    return () => {
      clearTimeout(timeoutId);
      window.removeEventListener("resize", checkTruncation);
    };
  }, [children]);

  const content = (
    <span
      ref={textRef}
      className={`truncate inline-block align-middle ${className}`}
    >
      {children}
    </span>
  );

  if (!isTruncated) return content;

  return (
    <TooltipProvider>
      <Tooltip delayDuration={300}>
        <TooltipTrigger asChild>{content}</TooltipTrigger>
        <TooltipContent side="top" className="max-w-[400px] break-words">
          <p>{textContent}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
