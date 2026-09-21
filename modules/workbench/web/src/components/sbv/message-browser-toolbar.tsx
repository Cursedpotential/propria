// Byline: Claude Code · Opus 5 · 2026-09-20 (loaded-row filter controls for the Review message browser)
"use client";

import { Info, Loader2, Paperclip, Search } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

interface MessageBrowserToolbarProps {
  query: string;
  onQueryChange: (value: string) => void;
  attachmentsOnly: boolean;
  onAttachmentsOnlyChange: (value: boolean) => void;
  loadedCount: number;
  visibleCount: number;
  hasMore: boolean;
  loadingAll: boolean;
  fetching: boolean;
  onLoadAll: () => void;
}

export function MessageBrowserToolbar({
  query,
  onQueryChange,
  attachmentsOnly,
  onAttachmentsOnlyChange,
  loadedCount,
  visibleCount,
  hasMore,
  loadingAll,
  fetching,
  onLoadAll,
}: MessageBrowserToolbarProps) {
  return (
    <div className="flex flex-wrap items-center gap-2 border-b px-3 py-2" role="toolbar" aria-label="Message list controls">
      <label className="relative min-w-52 flex-1">
        <span className="sr-only">Filter loaded message rows</span>
        <Search className="pointer-events-none absolute left-2.5 top-2.5 size-4 text-muted-foreground" aria-hidden="true" />
        <Input
          className="pl-9 pr-8"
          value={query}
          onChange={(event) => onQueryChange(event.target.value)}
          placeholder="Filter loaded rows"
          data-testid="message-browser-filter"
        />
        <span
          className="absolute right-2.5 top-2.5 text-muted-foreground"
          title="Filters the rows already loaded in this browser. The Review API has no server-side message search yet, so rows that have not been loaded are not searched."
        >
          <Info className="size-4" aria-label="Filter scope" />
        </span>
      </label>

      <Button
        type="button"
        variant={attachmentsOnly ? "default" : "outline"}
        size="sm"
        aria-pressed={attachmentsOnly}
        onClick={() => onAttachmentsOnlyChange(!attachmentsOnly)}
        data-testid="message-browser-attachments-toggle"
      >
        <Paperclip className="size-3.5" /> Has attachments
      </Button>

      <Badge variant="outline" className={cn("font-mono text-[11px]", hasMore && "border-dashed")}>
        {visibleCount === loadedCount
          ? `${loadedCount.toLocaleString()} loaded`
          : `${visibleCount.toLocaleString()} of ${loadedCount.toLocaleString()} loaded`}
        {hasMore ? " · more pages available" : " · thread complete"}
      </Badge>

      {hasMore && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={loadingAll || fetching}
          onClick={onLoadAll}
          data-testid="message-browser-load-all"
        >
          {loadingAll ? <Loader2 className="size-3.5 animate-spin motion-reduce:animate-none" /> : null} Load all
        </Button>
      )}
    </div>
  );
}
