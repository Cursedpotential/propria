// Byline: Claude Code · Sonnet · 2026-10-02
// The few pieces the mobile shell adds around the Workbench's own components: the page bar, the status
// badge (shadcn Badge), and the loading / error / empty / load-more blocks (shadcn Skeleton and Button).
import { ChevronLeft } from "lucide-react";

import { errorText } from "@/components/mobile/mobile-format";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import type { SourceStatus } from "@/lib/imported-client";
import { AppLink } from "@/lib/router-compat";
import { cn } from "@/lib/utils";

const STATUS: Record<SourceStatus, { label: string; className: string }> = {
  committed: { label: "Done", className: "border-transparent bg-emerald-100 text-emerald-900 dark:bg-emerald-950 dark:text-emerald-200" },
  awaiting_review: { label: "Awaiting review", className: "border-transparent bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-200" },
  parked: { label: "Parked", className: "border-transparent bg-orange-100 text-orange-900 dark:bg-orange-950 dark:text-orange-200" },
  failed: { label: "Failed", className: "border-transparent bg-red-100 text-red-900 dark:bg-red-950 dark:text-red-200" },
  running: { label: "Running", className: "border-transparent bg-sky-100 text-sky-900 dark:bg-sky-950 dark:text-sky-200" },
  not_finished: { label: "Not finished", className: "border-transparent bg-muted text-muted-foreground" },
  skipped: { label: "Skipped", className: "border-transparent bg-muted text-muted-foreground" },
};

export function StatusBadge({ status, className }: { status: SourceStatus; className?: string }) {
  const style = STATUS[status];
  return <Badge variant="outline" className={cn("px-2.5 py-1 text-xs font-semibold", style.className, className)}>{style.label}</Badge>;
}

export function PageBar({ title, subtitle, back }: { title: string; subtitle?: string; back?: string }) {
  return (
    <div className="sticky top-[var(--m-header-h)] z-10 flex min-h-14 items-center gap-1 border-b border-border bg-background/95 px-2 backdrop-blur">
      {back ? (
        <Button asChild variant="ghost" size="icon" className="size-12 shrink-0 rounded-full">
          <AppLink href={back} aria-label="Back"><ChevronLeft className="size-6" /></AppLink>
        </Button>
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
      {[0, 1, 2].map((index) => <Skeleton key={index} className="h-20 w-full" />)}
    </div>
  );
}

export function ErrorBox({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  return (
    <div className="m-4 rounded-lg border border-destructive/40 bg-destructive/10 p-4 text-sm" role="alert">
      <p className="font-semibold text-destructive">Could not load this</p>
      <p className="mt-1 text-foreground/80">{errorText(error)}</p>
      {onRetry ? <Button type="button" variant="outline" onClick={onRetry} className="mt-3 h-12">Try again</Button> : null}
    </div>
  );
}

export function Empty({ children }: { children: React.ReactNode }) {
  return <p className="px-6 py-12 text-center text-sm text-muted-foreground">{children}</p>;
}

export function LoadMore({ onClick, loading, label = "Load more" }: { onClick: () => void; loading: boolean; label?: string }) {
  return (
    <div className="p-4">
      <Button type="button" variant="outline" onClick={onClick} disabled={loading} className="h-12 w-full">{loading ? "Loading" : label}</Button>
    </div>
  );
}
