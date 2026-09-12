// Byline: Claude Code · Sonnet 5 · 2026-09-07
//
// Status chip. Owner rule (2026-09-07 addendum, "no bright ass colors"):
// status is communicated via a MUTED fill + 1px border + font-weight, never
// a loud/saturated color block. Every variant here maps to a desaturated,
// mid-luminance token pair from tokens.css — there is no "success green" /
// "danger red" full-saturation escape hatch anywhere in this component.
import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-1 rounded-[var(--radius-sm)] border px-1.5 py-0.5 text-xs font-medium leading-none",
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
