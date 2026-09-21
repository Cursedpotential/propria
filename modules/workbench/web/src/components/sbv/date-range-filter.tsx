// Ported from modules/forks/sbv/frontend/src/components/DateFilter.jsx
// MIT, Copyright (c) 2025 lowcarbdev
// Adapted: react-datepicker + date-fns dropped (not installed in this app, and no
// new npm dependency may be added here) in favor of native <input type="date">
// styled with this app's own design tokens. Behavior kept: a labeled From/To pair
// that clamps against each other and a Clear button that appears once either is set.
// Byline: Claude Code · Opus 5 · 2026-09-20
"use client";

import { CalendarDays, X } from "lucide-react";

import { Button } from "@/components/ui/button";

interface DateRangeFilterProps {
  from: string; // yyyy-mm-dd, "" when unset
  to: string; // yyyy-mm-dd, "" when unset
  onFromChange: (value: string) => void;
  onToChange: (value: string) => void;
}

export function DateRangeFilter({ from, to, onFromChange, onToChange }: DateRangeFilterProps) {
  const clear = () => {
    onFromChange("");
    onToChange("");
  };

  return (
    <div className="flex flex-wrap items-center gap-1.5 text-xs" data-testid="date-range-filter">
      <CalendarDays className="size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
      <label className="flex items-center gap-1" htmlFor="preview-date-from">
        <span className="font-medium text-muted-foreground">From</span>
        <input
          id="preview-date-from"
          type="date"
          value={from}
          max={to || undefined}
          onChange={(event) => onFromChange(event.target.value)}
          className="h-7 rounded-md border bg-background px-1.5 text-xs shadow-xs"
        />
      </label>
      <label className="flex items-center gap-1" htmlFor="preview-date-to">
        <span className="font-medium text-muted-foreground">To</span>
        <input
          id="preview-date-to"
          type="date"
          value={to}
          min={from || undefined}
          onChange={(event) => onToChange(event.target.value)}
          className="h-7 rounded-md border bg-background px-1.5 text-xs shadow-xs"
        />
      </label>
      {(from || to) && (
        <Button type="button" variant="outline" size="xs" onClick={clear} data-testid="date-range-filter-clear">
          <X className="size-3" /> Clear
        </Button>
      )}
    </div>
  );
}
