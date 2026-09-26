// Ported from modules/forks/sbv/frontend/src/components/MessageThread.jsx
// MIT, Copyright (c) 2025 lowcarbdev
// Supersedes the thin `platform-message-viewer.tsx` port (Codex · GPT-5.6 ·
// 2026-08-29, "Native Workbench port of SBV's useful viewing behavior") with the
// fuller SBV thread-reading behavior: chat-bubble layout, incoming/outgoing
// alignment, group-conversation sender labels, and a "Photos" toggle that swaps
// the body for a media grid — all driven by this app's own `use-preview-messages`
// hook and server-side filters instead of SBV's own backend axios calls.
//
// Owner ruling 2026-09-20 (Control surfaces decisions, relayed mid-task): this
// straight SBV-ported bubble view is an OPTIONAL "Conversation" mode, not the
// default — the dense Glide table (message-browser.tsx "Table" mode) is default
// and itself borrows SBV's media/vCard preview via message-source-panel.tsx. Also
// per that ruling: the thread reads OLDEST FIRST and pages forward (loads the next
// page when scrolling near the bottom, not via an explicit-only button), with
// Home/End jumping to the oldest/newest loaded message and scroll position
// preserved when a new page is appended (rows are only ever appended at the end,
// never reordered, so the already-rendered scroll offset never moves).
//
// Left out deliberately (see the port's task brief): the scroll-to-messageId deep
// link, print/PDF export, and SBV's older/newer bidirectional *offset*-windowed
// pagination — this app's Review API is a forward-only cursor over one flat
// message projection, not SBV's offset-windowed per-conversation history,  so
// "near the bottom" always means "the next chronological page," never "jump to an
// arbitrary window." Actual media bytes are never fetched or faked (see
// attachment-preview.tsx); date-separator dividers were not present in the donor
// MessageThread.jsx render and were not invented here.
// Byline: Claude Code · Opus 5 · 2026-09-20
"use client";

import { ImageIcon, Loader2, MessageSquareText } from "lucide-react";
import { type KeyboardEvent, useEffect, useMemo, useRef, useState } from "react";

import { MediaOnlyGrid } from "@/components/sbv/media-only-grid";
import { MessageBubble } from "@/components/sbv/message-bubble";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { PreviewMessageRow } from "@/hooks/use-preview-messages";
import type { MatterMode, ProfferPreviewParticipant } from "@/lib/shared/types";

interface MessageThreadViewProps {
  rows: PreviewMessageRow[];
  participants: Map<string, ProfferPreviewParticipant>;
  loading: boolean;
  error: string | null;
  previewHandle: string;
  mode: MatterMode;
  hasMore: boolean;
  fetching: boolean;
  onLoadMore: () => void;
}

export function MessageThreadView({
  rows,
  loading,
  error,
  previewHandle,
  mode,
  hasMore,
  fetching,
  onLoadMore,
}: MessageThreadViewProps) {
  const [showMediaOnly, setShowMediaOnly] = useState(false);
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const sentinelRef = useRef<HTMLDivElement | null>(null);

  // Mirrors SBV's isGroupConversation check (first item's addresses.length > 1),
  // adapted to this app's participant_ids: more than sender + one other party.
  const isGroupConversation = useMemo(
    () => rows.some((row) => new Set(row.message.participant_ids).size > 2),
    [rows],
  );

  // Oldest-first, forward-only paging: load the next page automatically once the
  // bottom sentinel scrolls into view. Rows are only ever appended, so the reader's
  // current scroll offset is never disturbed by a page landing.
  useEffect(() => {
    const root = scrollRef.current;
    const sentinel = sentinelRef.current;
    if (!root || !sentinel || showMediaOnly) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting && hasMore && !fetching) onLoadMore();
      },
      { root, rootMargin: "200px" },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [fetching, hasMore, onLoadMore, showMediaOnly]);

  function handleKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    const root = scrollRef.current;
    if (!root) return;
    if (event.key === "Home") {
      event.preventDefault();
      root.scrollTo({ top: 0, behavior: "smooth" });
    } else if (event.key === "End") {
      event.preventDefault();
      root.scrollTo({ top: root.scrollHeight, behavior: "smooth" });
    }
  }

  return (
    <section className="flex h-[38rem] min-h-0 flex-col overflow-hidden" aria-label="Conversation view" data-testid="message-thread-view">
      <header className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
        <div>
          <p className="platform-kicker">Normalized message projection · optional conversation view</p>
          <h2 className="mt-1 flex items-center gap-2 text-base font-semibold">
            <MessageSquareText className="size-4" aria-hidden="true" /> Conversation
          </h2>
        </div>
        <div className="flex items-center gap-2">
          {fetching && (
            <span className="flex items-center gap-2 text-xs text-muted-foreground" role="status">
              <Loader2 className="size-3.5 animate-spin motion-reduce:animate-none" /> Loading…
            </span>
          )}
          <Button
            type="button"
            variant={showMediaOnly ? "default" : "outline"}
            size="sm"
            aria-pressed={showMediaOnly}
            onClick={() => setShowMediaOnly((value) => !value)}
            data-testid="message-thread-photos-toggle"
          >
            <ImageIcon className="size-3.5" /> {showMediaOnly ? "Show all" : "Photos"}
          </Button>
          <Badge variant="outline">{rows.length.toLocaleString()} messages</Badge>
        </div>
      </header>

      <div
        ref={scrollRef}
        className="min-h-0 flex-1 overflow-y-auto bg-muted/25 p-4 outline-none"
        tabIndex={0}
        onKeyDown={handleKeyDown}
        aria-label="Oldest first; Home jumps to the oldest loaded message, End jumps to the newest loaded message"
      >
        {loading && rows.length === 0 && (
          <p className="text-sm text-muted-foreground" role="status">Loading messages…</p>
        )}
        {error && (
          <div className="rounded border border-destructive/40 bg-destructive/5 p-3 text-sm text-destructive" role="alert">
            {error}
          </div>
        )}
        {!loading && !error && rows.length === 0 && (
          <div className="mx-auto max-w-md py-16 text-center">
            <MessageSquareText className="mx-auto size-8 text-muted-foreground" aria-hidden="true" />
            <p className="mt-3 text-sm font-medium">No message records match the current filters.</p>
          </div>
        )}
        {!loading && !error && rows.length > 0 && (
          showMediaOnly ? (
            <MediaOnlyGrid rows={rows} previewHandle={previewHandle} mode={mode} />
          ) : (
            <>
              <ol className="space-y-1.5" aria-live="polite">
                {rows.map((row) => (
                  <li key={row.message.message_id}>
                    <MessageBubble row={row} previewHandle={previewHandle} mode={mode} showSenderLabel={isGroupConversation} />
                  </li>
                ))}
              </ol>
              <div ref={sentinelRef} aria-hidden="true" className="h-px" />
              {hasMore && (
                <div className="pt-3 text-center">
                  <button
                    type="button"
                    className="text-xs font-medium underline-offset-4 hover:underline disabled:opacity-50"
                    disabled={fetching}
                    onClick={onLoadMore}
                    data-testid="message-thread-load-more"
                  >
                    {fetching ? "Loading…" : "↓ Load more messages"}
                  </button>
                </div>
              )}
            </>
          )
        )}
      </div>

      <footer className="border-t px-4 py-2 text-[11px] text-muted-foreground">
        Attempt {previewHandle} · read-only platform projection · PostgreSQL remains canonical
      </footer>
    </section>
  );
}
