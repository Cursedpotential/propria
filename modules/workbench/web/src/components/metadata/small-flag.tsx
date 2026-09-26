// Byline: Claude Code · Opus 5.5 · 2026-09-26
// One small flag on the item it concerns — the Workbench rule for unavailable or
// disputed data (never a banner or a caveat paragraph). Same look as the Sources
// metadata panel's flag.
import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export function SmallFlag({ children, title, tone = "attention", className }: {
  children: ReactNode;
  title?: string;
  tone?: "attention" | "info" | "hindsight";
  className?: string;
}) {
  const tones = {
    attention: "border-[#c58214] bg-[#fff4dd] text-[#684b18] dark:bg-[#43351f] dark:text-[#ffe0a6]",
    info: "border-[#4051b9] bg-[#e9ecfb] text-[#2b3785] dark:bg-[#313a66] dark:text-[#c9d0fb]",
    hindsight: "border-[#7a4bb5] bg-[#f1e9fb] text-[#4d2a7a] dark:bg-[#3a2a52] dark:text-[#dcc6fb]",
  } as const;
  return (
    <span
      title={title}
      className={cn("inline-flex shrink-0 items-center border px-1 text-[9px] font-semibold uppercase leading-4", tones[tone], className)}
    >
      {children}
    </span>
  );
}
