// Byline: Claude Code · Opus 5 · 2026-09-20 (server-side filter controls for the Review message browser)
//
// The search box and "has attachments" toggle here were originally a client-side
// filter over already-loaded rows. The backend now supports `q`/`has_attachments`/
// `from`/`to` as real server-side query parameters (see api-client.ts), so this
// toolbar debounces the typed query and reports server-reported match/total counts
// instead of counting only the rows this browser happened to have loaded.
"use client";

import { Loader2, Paperclip, Search } from "lucide-react";
import { useEffect, useState } from "react";

import { DateRangeFilter } from "@/components/sbv/date-range-filter";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

/** Debounce window before a typed query becomes a live server request. */
const SEARCH_DEBOUNCE_MS = 300;

interface MessageBrowserToolbarProps {
  query: string;
  onQueryChange: (value: string) => void;
  attachmentsOnly: boolean;
  onAttachmentsOnlyChange: (value: boolean) => void;
  dateFrom: string;
  dateTo: string;
  onDateFromChange: (value: string) => void;
  onDateToChange: (value: string) => void;
  loadedCount: number;
  /** Server-reported count of messages matching the active filter, or null if unreported. */
  totalMatches: number | null;
  /** Server-reported count of all messages in this preview, or null if unreported. */
  totalMessages: number | null;
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
  dateFrom,
  dateTo,
  onDateFromChange,
  onDateToChange,
  loadedCount,
  totalMatches,
  totalMessages,
  hasMore,
  loadingAll,
  fetching,
  onLoadAll,
}: MessageBrowserToolbarProps) {
  const [draftQuery, setDraftQuery] = useState(query);
  // Adjusting state during render (React's own pattern for "reset state when a prop
  // changes") instead of an effect, so an externally reset `query` (e.g. clearing
  // filters) re-syncs the input without a synchronous setState-in-effect render pass.
  const [syncedQuery, setSyncedQuery] = useState(query);
  if (query !== syncedQuery) {
    setSyncedQuery(query);
    setDraftQuery(query);
  }

  useEffect(() => {
    if (draftQuery === query) return;
    const timer = window.setTimeout(() => onQueryChange(draftQuery), SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draftQuery]);

  const matchLabel = totalMatches !== null && totalMessages !== null
    ? `${totalMatches.toLocaleString()} of ${totalMessages.toLocaleString()}`
    : totalMatches !== null
      ? `${totalMatches.toLocaleString()} matches`
      : `${loadedCount.toLocaleString()} loaded`;

  return (
    <div className="flex flex-wrap items-center gap-2 border-b px-3 py-2" role="toolbar" aria-label="Message list controls">
      <label className="relative min-w-52 flex-1">
        <span className="sr-only">Search messages</span>
        <Search className="pointer-events-none absolute left-2.5 top-2.5 size-4 text-muted-foreground" aria-hidden="true" />
        <Input
          className="pl-9"
          value={draftQuery}
          onChange={(event) => setDraftQuery(event.target.value)}
          placeholder="Search message bodies"
          maxLength={200}
          data-testid="message-browser-filter"
        />
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

      <DateRangeFilter from={dateFrom} to={dateTo} onFromChange={onDateFromChange} onToChange={onDateToChange} />

      <Badge variant="outline" className={cn("font-mono text-[11px]", hasMore && "border-dashed")} data-testid="message-browser-match-count">
        {matchLabel}
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
