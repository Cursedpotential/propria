// Byline: Claude Code · Sonnet · 2026-10-02
// Small shared pieces for the mobile shell: formatters, status badge, page bar, empty/error/loading blocks.
import { ChevronLeft } from "lucide-react";

import { errorText } from "@/components/mobile/mobile-format";
import { AppLink } from "@/lib/router-compat";
import type { SourceStatus } from "@/lib/imported-client";
import { cn } from "@/lib/utils";

const STATUS_STYLE: Record<SourceStatus, { label: string; className: string }> = {
  committed: { label: "Committed", className: "bg-emerald-100 text-emerald-900 dark:bg-emerald-950 dark:text-emerald-200" },
  awaiting_review: { label: "Awaiting review", className: "bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-200" },
  parked: { label: "Parked", className: "bg-orange-100 text-orange-900 dark:bg-orange-950 dark:text-orange-200" },
  failed: { label: "Failed", className: "bg-red-100 text-red-900 dark:bg-red-950 dark:text-red-200" },
  running: { label: "Running", className: "bg-sky-100 text-sky-900 dark:bg-sky-950 dark:text-sky-200" },
  not_finished: { label: "Not finished", className: "bg-muted text-muted-foreground" },
};

export function StatusBadge({ status, className }: { status: SourceStatus; className?: string }) {
  const style = STATUS_STYLE[status];
  return (
    <span className={cn("inline-flex shrink-0 items-center rounded-full px-2.5 py-1 text-xs font-semibold", style.className, className)}>
      {style.label}
    </span>
  );
}

export function Chip({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <span className={cn("inline-flex items-center rounded-full bg-muted px-2.5 py-1 text-xs font-medium text-foreground/80", className)}>
      {children}
    </span>
  );
}

export function PageBar({ title, subtitle, back }: { title: string; subtitle?: string; back?: string }) {
  return (
    <div className="sticky top-[var(--m-header-h)] z-10 flex min-h-14 items-center gap-1 border-b border-border bg-background/95 px-2 backdrop-blur">
      {back ? (
        <AppLink href={back} aria-label="Back" className="flex size-12 shrink-0 items-center justify-center rounded-full active:bg-muted">
          <ChevronLeft className="size-6" />
        </AppLink>
      ) : null}
      <div className={cn("min-w-0 flex-1 py-2", back ? "" : "px-2")}>
        <h1 className="truncate text-base font-semibold leading-tight">{title}</h1>
        {subtitle ? <p className="truncate text-xs text-muted-foreground">{subtitle}</p> : null}
      </div>
    </div>
  );
}

export function Loading({ label = "Loading" }: { label?: string }) {
  return (
    <div className="space-y-3 p-4" role="status" aria-label={label}>
      {[0, 1, 2].map((index) => (
        <div key={index} className="h-24 animate-pulse rounded-lg bg-muted" />
      ))}
    </div>
  );
}

export function ErrorBox({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  return (
    <div className="m-4 rounded-lg border border-destructive/40 bg-destructive/10 p-4 text-sm" role="alert">
      <p className="font-semibold text-destructive">Could not load this</p>
      <p className="mt-1 text-foreground/80">{errorText(error)}</p>
      {onRetry ? (
        <button type="button" onClick={onRetry} className="mt-3 h-12 rounded-md border border-border bg-card px-5 text-sm font-semibold active:bg-muted">
          Try again
        </button>
      ) : null}
    </div>
  );
}

export function Empty({ children }: { children: React.ReactNode }) {
  return <p className="px-6 py-12 text-center text-sm text-muted-foreground">{children}</p>;
}

export function LoadMore({ onClick, loading, label = "Load more" }: { onClick: () => void; loading: boolean; label?: string }) {
  return (
    <div className="p-4">
      <button
        type="button"
        onClick={onClick}
        disabled={loading}
        className="h-12 w-full rounded-lg border border-border bg-card text-sm font-semibold active:bg-muted disabled:opacity-60"
      >
        {loading ? "Loading" : label}
      </button>
    </div>
  );
}
