// Byline: Claude Code · Sonnet 5 · 2026-09-07
// Updated by: OpenAI Codex · GPT-5 · 2026-09-12 — shared target and focus contract.
import * as React from "react";
import { cn } from "@/lib/utils";

export const Input = React.forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(({ className, ...props }, ref) => (
  <input
    ref={ref}
    className={cn(
      "h-9 w-full rounded-[var(--radius-sm)] border border-border-strong bg-bg-inset px-2.5 text-sm text-text-primary",
      "placeholder:text-text-tertiary focus-visible:outline focus-visible:outline-[3px] focus-visible:outline-offset-2 focus-visible:outline-focus",
      className,
    )}
    {...props}
  />
));
Input.displayName = "Input";
