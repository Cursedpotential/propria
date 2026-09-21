// Byline: Claude Code · Opus 5 · 2026-09-20 (three-panel Review message browser; detail follows selection)
// Byline: Claude Code · Opus 5 · 2026-09-20 (Thread | Table mode switch; SBV thread view is the default reading mode)
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
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";

import { MessageBrowserGrid } from "@/components/sbv/message-browser-grid";
import { MessageBrowserToolbar } from "@/components/sbv/message-browser-toolbar";
import { MessageDetailPanel } from "@/components/sbv/message-detail-panel";
import { MessageSourcePanel } from "@/components/sbv/message-source-panel";
import { MessageThreadView } from "@/components/sbv/message-thread-view";
import { Button } from "@/components/ui/button";
import {
  useProfferPreviewMessages,
  usePreviewMessageRows,
  usePreviewMessageTotals,
} from "@/hooks/use-preview-messages";
import type { MatterMode, ProfferPackageProjection } from "@/lib/shared/types";
import { cn } from "@/lib/utils";

interface MessageBrowserProps {
  previewHandle: string;
  mode: MatterMode;
  packageProjection: ProfferPackageProjection | null;
}

type ViewMode = "thread" | "table";

const HANDLE_CLASS =
  "w-px bg-border transition-colors data-[resize-handle-state=drag]:bg-primary data-[resize-handle-state=hover]:bg-primary";

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

  const query = useProfferPreviewMessages(previewHandle, mode, filters);
  const { rows, participants } = usePreviewMessageRows(query.data?.pages);
  const { totalMatches, totalMessages } = usePreviewMessageTotals(query.data?.pages);
  const [selectedMessageId, setSelectedMessageId] = useState<string | null>(null);
  const [loadingAll, setLoadingAll] = useState(false);

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

  const { fetchNextPage, hasNextPage, isFetchingNextPage } = query;

  const requestNextPage = useCallback(() => {
    if (hasNextPage && !isFetchingNextPage) void fetchNextPage();
  }, [fetchNextPage, hasNextPage, isFetchingNextPage]);

  const loadAll = useCallback(async () => {
    setLoadingAll(true);
    try {
      let result = await fetchNextPage();
      while (result.hasNextPage) result = await fetchNextPage();
    } finally {
      setLoadingAll(false);
    }
  }, [fetchNextPage]);

  // Per-attempt reset is handled by the caller's `key`, which remounts this
  // component when the preview handle or TEST/REAL mode changes.
  if (query.isError) {
    return (
      <div className="border border-destructive/40 bg-destructive/5 p-4 text-sm text-destructive" role="alert">
        Message rows are unavailable: {query.error instanceof Error ? query.error.message : "unknown error"}
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
          {query.isFetching && !loadingAll && (
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
          </div>
        </div>
      </header>

      <MessageBrowserToolbar
        query={filter}
        onQueryChange={setFilter}
        attachmentsOnly={attachmentsOnly}
        onAttachmentsOnlyChange={setAttachmentsOnly}
        dateFrom={dateFrom}
        dateTo={dateTo}
        onDateFromChange={setDateFrom}
        onDateToChange={setDateTo}
        loadedCount={rows.length}
        totalMatches={totalMatches}
        totalMessages={totalMessages}
        hasMore={Boolean(hasNextPage)}
        loadingAll={loadingAll}
        fetching={isFetchingNextPage}
        onLoadAll={() => void loadAll()}
      />

      {viewMode === "thread" ? (
        <MessageThreadView
          rows={rows}
          participants={participants}
          loading={query.isPending}
          error={queryError}
          previewHandle={previewHandle}
          mode={mode}
          hasMore={Boolean(hasNextPage)}
          fetching={isFetchingNextPage}
          onLoadMore={requestNextPage}
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
                  onReachEnd={requestNextPage}
                />
              )}
            </div>
          </Panel>

          <PanelResizeHandle className={HANDLE_CLASS} />

          <Panel id="message-detail" order={2} defaultSize={28} minSize={18}>
            <MessageDetailPanel row={selectedRow} participants={participants} />
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
