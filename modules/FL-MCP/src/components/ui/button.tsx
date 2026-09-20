// Byline: Claude Code · Sonnet 5 · 2026-09-07
// Updated by: OpenAI Codex · GPT-5 · 2026-09-12 — 36px targets and brass focus.
import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-1.5 whitespace-nowrap rounded-[var(--radius-sm)] text-sm font-medium " +
    "transition-colors duration-100 disabled:pointer-events-none disabled:opacity-40 " +
    "focus-visible:outline focus-visible:outline-[3px] focus-visible:outline-offset-2 focus-visible:outline-focus " +
    "[&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        // Solid means deliberate action and therefore uses the shared seal.
        // Information remains teal and never impersonates a primary action.
        solid: "bg-accent text-text-on-accent hover:bg-accent-strong border border-transparent",
        outline: "border border-border-strong bg-transparent text-text-primary hover:bg-surface-hover",
        ghost: "border border-transparent bg-transparent text-text-secondary hover:bg-surface-hover hover:text-text-primary",
        destructive: "border border-critical-border bg-critical-fill text-critical-text hover:bg-critical/20",
      },
      size: {
        sm: "h-9 px-2.5 text-xs",
        md: "h-9 px-3",
        lg: "h-11 px-4",
        icon: "size-9 p-0",
      },
    },
    defaultVariants: { variant: "outline", size: "md" },
  },
);

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement>, VariantProps<typeof buttonVariants> {}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(({ className, variant, size, ...props }, ref) => (
  <button ref={ref} className={cn(buttonVariants({ variant, size }), className)} {...props} />
));
Button.displayName = "Button";
