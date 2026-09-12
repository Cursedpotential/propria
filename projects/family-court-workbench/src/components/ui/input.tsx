// Byline: Claude Code · Sonnet 5 · 2026-09-07
import * as React from "react";
import { cn } from "@/lib/utils";

export const Input = React.forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(({ className, ...props }, ref) => (
  <input
    ref={ref}
    className={cn(
      "h-8 w-full rounded-[var(--radius-sm)] border border-border-strong bg-bg-inset px-2.5 text-sm text-text-primary",
      "placeholder:text-text-tertiary focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent-border",
      className,
    )}
    {...props}
  />
));
Input.displayName = "Input";
