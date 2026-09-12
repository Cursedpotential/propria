// Byline: Claude Code · Sonnet 5 · 2026-09-07
// Updated by: OpenAI Codex · GPT-5 · 2026-09-12 — shared compact-status geometry.
//
// Status chip. Every tone maps to a labelled semantic state; color is
// reinforcement, never the only carrier of authority or outcome.
import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-1 rounded-full border px-2 py-1 text-xs font-semibold leading-none",
  {
    variants: {
      tone: {
        neutral: "border-border-strong bg-surface-raised text-text-secondary",
        accent: "border-accent-border bg-accent-fill text-accent-text",
        good: "border-good-border bg-good-fill text-good-text",
        warn: "border-warn-border bg-warn-fill text-warn-text",
        critical: "border-critical-border bg-critical-fill text-critical-text",
      },
    },
    defaultVariants: { tone: "neutral" },
  },
);

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement>, VariantProps<typeof badgeVariants> {}

export function Badge({ className, tone, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ tone }), className)} {...props} />;
}
