// Byline: Claude Code · Opus 5 · 2026-09-20 (three-panel Review message browser; detail follows selection)
// Byline: Claude Code · Opus 5 · 2026-09-20 (Thread | Table mode switch; SBV thread view is the default reading mode)
// Byline: Claude Code · Opus 5.5 · 2026-09-26 (detail panel carries the run so it can show context review)
//
// Layout shape ported from RAGFlow's chunk-review screen
// (web/src/pages/chunk/parsed-result/add-knowledge/components/knowledge-chunk/index.tsx,
// Apache-2.0, github.com/infiniflow/ragflow): a toolbar over a virtualized list
// inside a resizable panel group, with the detail columns driven by the current
// selection instead of a per-row detail action. No RAGFlow source is copied here;
// see modules/workbench/web/THIRD_PARTY_NOTICES.md.
//
// Owner directive 2026-09-20: "message preview is where we were supposed to pull in
// the front end of the original sbv app." SBV's chat-bubble thread reading view
// (message-thread-view.tsx, a fuller port of modules/forks/sbv/frontend/src/
// components/MessageThread.jsx, MIT, lowcarbdev) was first wired here as the
// default reading mode. A same-day follow-up owner ruling (Control surfaces
// decisions, relayed mid-task) corrected that: the dense Glide **Table** mode is
// the default, and the straight SBV-ported bubble view is kept as an explicit,
// optional **Conversation** mode — "borrow from it," not replace the default with
// it. Both modes render off the exact same `useProfferPreviewMessages` page data
// and the exact same server-side search/date/attachment filters — switching modes
// never refetches or reinterprets the underlying rows. The dense Table mode's
// detail panel (message-source-panel.tsx) borrows SBV's media/vCard preview
// behavior directly, per that same ruling.
"use client";

import { Loader2, MessageSquareText } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";

import { MessageBrowserGrid } from "@/components/sbv/message-browser-grid";
import { MessageBrowserToolbar } from "@/components/sbv/message-browser-toolbar";
import { MessageDetailPanel } from "@/components/sbv/message-detail-panel";
import { MessageSourcePanel } from "@/components/sbv/message-source-panel";
import { MediaOnlyGrid } from "@/components/sbv/media-only-grid";
import { MessageThreadView } from "@/components/sbv/message-thread-view";
import { Button } from "@/components/ui/button";
import {
  MESSAGE_PAGE_SIZE,
  usePreviewMessageRows,
  usePreviewMessageTotals,
} from "@/hooks/use-preview-messages";
import type { MatterMode, ProfferPackageProjection } from "@/lib/shared/types";
import { cn } from "@/lib/utils";
import { getProfferPreviewMessages } from "@/lib/api-client";

interface MessageBrowserProps {
  previewHandle: string;
  mode: MatterMode;
  packageProjection: ProfferPackageProjection | null;
}

type ViewMode = "thread" | "table" | "media";

const HANDLE_CLASS =
  "w-px bg-border transition-colors data-[resize-handle-state=drag]:bg-primary data-[resize-handle-state=hover]:bg-primary";

/** Read one processed message page while retaining server-side search semantics.
 * Inputs: exact attempt, mode and package projection. Output: one bounded page
 * in table, donor conversation or media view. Effects: abortable read API calls;
 * no load-all, scroll-prefetch, extraction, indexing or source mutation.
 */
export function MessageBrowser({ previewHandle, mode, packageProjection }: MessageBrowserProps) {
  // Default is "table" (dense Glide rows) per the owner's same-day correction; the
  // SBV bubble view is opt-in via the mode switch below.
  const [viewMode, setViewMode] = useState<ViewMode>("table");
  const [filter, setFilter] = useState("");
  const [attachmentsOnly, setAttachmentsOnly] = useState(false);
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");

  const filters = useMemo(
    () => ({
      q: filter.trim() || undefined,
      hasAttachments: attachmentsOnly || undefined,
      from: dateFrom ? new Date(`${dateFrom}T00:00:00.000Z`).toISOString() : undefined,
      to: dateTo ? new Date(`${dateTo}T23:59:59.999Z`).toISOString() : undefined,
    }),
    [attachmentsOnly, dateFrom, dateTo, filter],
  );

  const filterKey = JSON.stringify([previewHandle, mode, filters]);
  const [paging, setPaging] = useState<{ filterKey: string; cursors: (string | undefined)[]; index: number; first: number }>({ filterKey: "", cursors: [undefined], index: 0, first: 1 });
  const activePaging = paging.filterKey === filterKey ? paging : { filterKey, cursors: [undefined], index: 0, first: 1 };
  const cursor = activePaging.cursors[activePaging.index];
  const query = useQuery({
    queryKey: ["proffer", "single-message-page", filterKey, cursor],
    queryFn: ({ signal }) => getProfferPreviewMessages(previewHandle, mode, cursor, MESSAGE_PAGE_SIZE, signal, filters),
    enabled: Boolean(previewHandle), gcTime: 0, retry: false, staleTime: Infinity,
  });
  const pages = useMemo(() => query.data ? [query.data] : undefined, [query.data]);
  const { rows, participants } = usePreviewMessageRows(pages);
  const missingPayloads = useMemo(() => rows.reduce((total, row) => total + row.missingPayloadCount, 0), [rows]);
  const { totalMatches, totalMessages } = usePreviewMessageTotals(pages);
  const [selectedMessageId, setSelectedMessageId] = useState<string | null>(null);

  const selectedIndex = useMemo(() => {
    if (!selectedMessageId) return rows.length ? 0 : -1;
    const index = rows.findIndex((row) => row.message.message_id === selectedMessageId);
    return index === -1 ? (rows.length ? 0 : -1) : index;
  }, [selectedMessageId, rows]);

  const selectedRow = selectedIndex >= 0 ? rows[selectedIndex] ?? null : null;

  const selectIndex = useCallback(
    (index: number) => {
      const row = rows[index];
      if (row) setSelectedMessageId(row.message.message_id);
    },
    [rows],
  );

  const hasNextPage = Boolean(query.data?.next_cursor);
  const requestNextPage = () => {
    if (!query.data?.next_cursor || query.isFetching) return;
    const cursors = [...activePaging.cursors.slice(0, activePaging.index + 1), query.data.next_cursor];
    const dropped = Math.max(0, cursors.length - 64);
    setPaging({ filterKey, cursors: cursors.slice(dropped), index: cursors.length - dropped - 1, first: activePaging.first + dropped });
  };

  /** Release page coordinates when server-side filters change; retain no old result rows. */
  function resetPages() {
    setPaging({ filterKey: "", cursors: [undefined], index: 0, first: 1 });
    setSelectedMessageId(null);
  }

  // Per-attempt reset is handled by the caller's `key`, which remounts this
  // component when the preview handle or DEV/LIVE mode changes.
  if (query.isError) {
    return (
      <div className="border border-destructive/40 bg-destructive/5 p-4 text-sm text-destructive" role="alert">
        Message rows are unavailable: {query.error instanceof Error ? query.error.message : "unknown error"}
        <div className="mt-2 flex gap-2">
          <Button size="sm" variant="outline" disabled={query.isFetching} onClick={() => void query.refetch()}>Retry messages</Button>
          <Button size="sm" variant="outline" onClick={resetPages}>First page</Button>
        </div>
      </div>
    );
  }

  // query.isError already returned above, so there is never a live error to
  // surface into MessageThreadView past this point.
  const queryError: string | null = null;

  return (
    <section
      className="platform-panel flex h-[calc(100vh-25rem)] min-h-[22rem] flex-col overflow-hidden"
      aria-label="Message browser"
      data-testid="message-browser"
    >
      <header className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
        <div>
          <p className="platform-kicker">Normalized message projection</p>
          <h2 className="mt-1 flex items-center gap-2 text-base font-semibold">
            <MessageSquareText className="size-4" aria-hidden="true" /> Messages
          </h2>
        </div>
        <div className="flex items-center gap-3">
          {query.isFetching && (
            <span className="flex items-center gap-2 text-xs text-muted-foreground" role="status">
              <Loader2 className="size-3.5 animate-spin motion-reduce:animate-none" /> Loading rows…
            </span>
          )}
          <div
            className="inline-flex rounded-md border p-0.5"
            role="group"
            aria-label="Message view mode"
            data-testid="message-browser-view-mode"
          >
            <Button
              type="button"
              size="sm"
              variant="ghost"
              aria-pressed={viewMode === "thread"}
              className={cn("h-7 px-3", viewMode === "thread" && "bg-accent text-accent-foreground")}
              onClick={() => setViewMode("thread")}
              data-testid="message-browser-mode-thread"
              title="Optional SBV-style bubble conversation view"
            >
              Conversation
            </Button>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              aria-pressed={viewMode === "table"}
              className={cn("h-7 px-3", viewMode === "table" && "bg-accent text-accent-foreground")}
              onClick={() => setViewMode("table")}
              data-testid="message-browser-mode-table"
            >
              Table
            </Button>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              aria-pressed={viewMode === "media"}
              className={cn("h-7 px-3", viewMode === "media" && "bg-accent text-accent-foreground")}
              onClick={() => setViewMode("media")}
              data-testid="message-browser-mode-media"
              title="Photos and videos in the loaded messages; a missing one keeps its slot"
            >
              Media
            </Button>
          </div>
          {missingPayloads > 0 && (
            <span
              className="rounded border border-destructive/60 px-2 py-0.5 text-[11px] text-destructive"
              data-testid="message-browser-missing-payloads"
              title="Attachments this backup names but carries no bytes for, in the messages loaded so far"
            >
              {missingPayloads} missing from this backup
            </span>
          )}
        </div>
      </header>

      <MessageBrowserToolbar
        query={filter}
        onQueryChange={(value) => { setFilter(value); resetPages(); }}
        attachmentsOnly={attachmentsOnly}
        onAttachmentsOnlyChange={(value) => { setAttachmentsOnly(value); resetPages(); }}
        dateFrom={dateFrom}
        dateTo={dateTo}
        onDateFromChange={(value) => { setDateFrom(value); resetPages(); }}
        onDateToChange={(value) => { setDateTo(value); resetPages(); }}
        loadedCount={rows.length}
        totalMatches={totalMatches}
        totalMessages={totalMessages}
        hasMore={Boolean(hasNextPage)}
        loadingAll={false}
        fetching={query.isFetching}
        onLoadAll={() => {}}
        paged
      />

      <nav className="flex items-center justify-between gap-2 border-b px-3 py-2" aria-label="Processed message pages">
        <Button size="sm" variant="outline" disabled={activePaging.index === 0 || query.isFetching} onClick={() => setPaging({ ...activePaging, index: activePaging.index - 1 })}>Previous messages</Button>
        <span className="text-xs" role="status">Page {activePaging.first + activePaging.index} · {rows.length} messages on this page</span>
        <Button size="sm" variant="outline" disabled={!hasNextPage || query.isFetching} onClick={requestNextPage}>Next messages</Button>
      </nav>

      {viewMode === "media" ? (
        <div className="min-h-0 flex-1 overflow-y-auto p-3">
          <MediaOnlyGrid rows={rows} previewHandle={previewHandle} mode={mode} />
        </div>
      ) : viewMode === "thread" ? (
        <MessageThreadView
          key={cursor ?? "first"}
          rows={rows}
          participants={participants}
          loading={query.isPending}
          error={queryError}
          previewHandle={previewHandle}
          mode={mode}
          hasMore={false}
          fetching={query.isFetching}
          onLoadMore={() => {}}
        />
      ) : (
        <PanelGroup direction="horizontal" className="min-h-0 flex-1" autoSaveId="proffer-message-browser">
          <Panel id="message-list" order={1} defaultSize={52} minSize={30}>
            <div className="min-h-0 h-full flex-1">
              {query.isPending ? (
                <p className="p-4 text-sm text-muted-foreground" role="status">Loading message rows…</p>
              ) : rows.length === 0 ? (
                <p className="p-4 text-sm text-muted-foreground">
                  No message rows are projected for this attempt with the current filters.
                </p>
              ) : (
                <MessageBrowserGrid
                  rows={rows}
                  selectedIndex={selectedIndex}
                  onSelectIndex={selectIndex}
                  onReachEnd={() => {}}
                />
              )}
            </div>
          </Panel>

          <PanelResizeHandle className={HANDLE_CLASS} />

          <Panel id="message-detail" order={2} defaultSize={28} minSize={18}>
            <MessageDetailPanel row={selectedRow} participants={participants} previewHandle={previewHandle} mode={mode} />
          </Panel>

          <PanelResizeHandle className={HANDLE_CLASS} />

          <Panel id="message-source" order={3} defaultSize={20} minSize={16}>
            <MessageSourcePanel row={selectedRow} packageProjection={packageProjection} previewHandle={previewHandle} mode={mode} />
          </Panel>
        </PanelGroup>
      )}
    </section>
  );
}
