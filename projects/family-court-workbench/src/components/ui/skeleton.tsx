// Byline: Claude Code · Sonnet 5 · 2026-09-07
import type * as React from "react";
import { cn } from "@/lib/utils";

export function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("animate-pulse rounded-[var(--radius-sm)] bg-surface-hover", className)} {...props} />;
}
