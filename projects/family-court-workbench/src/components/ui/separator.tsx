// Byline: Claude Code · Sonnet 5 · 2026-09-07
import * as React from "react";
import { cn } from "@/lib/utils";

export function Separator({ orientation = "horizontal", className, ...props }: React.HTMLAttributes<HTMLDivElement> & { orientation?: "horizontal" | "vertical" }) {
  return (
    <div
      role="separator"
      aria-orientation={orientation}
      className={cn("shrink-0 bg-border", orientation === "horizontal" ? "h-px w-full" : "h-full w-px", className)}
      {...props}
    />
  );
}
