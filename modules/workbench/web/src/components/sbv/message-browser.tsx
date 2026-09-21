// Byline: Claude Code · Opus 5 · 2026-09-20 (three-panel Review message browser; detail follows selection)
//
// Layout shape ported from RAGFlow's chunk-review screen
// (web/src/pages/chunk/parsed-result/add-knowledge/components/knowledge-chunk/index.tsx,
// Apache-2.0, github.com/infiniflow/ragflow): a toolbar over a virtualized list
// inside a resizable panel group, with the detail columns driven by the current
// selection instead of a per-row detail action. No RAGFlow source is copied here;
// see modules/workbench/web/THIRD_PARTY_NOTICES.md.
"use client";

import { Loader2, MessageSquareText } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";

import { MessageBrowserGrid } from "@/components/sbv/message-browser-grid";
import { MessageBrowserToolbar } from "@/components/sbv/message-browser-toolbar";
import { MessageDetailPanel } from "@/components/sbv/message-detail-panel";
import { MessageSourcePanel } from "@/components/sbv/message-source-panel";
import {
  filterLoadedRows,
  useProfferPreviewMessages,
  usePreviewMessageRows,
} from "@/hooks/use-preview-messages";
import type { MatterMode, ProfferPackageProjection } from "@/lib/shared/types";

interface MessageBrowserProps {
  previewHandle: string;
  mode: MatterMode;
  packageProjection: ProfferPackageProjection | null;
}

const HANDLE_CLASS =
  "w-px bg-border transition-colors data-[resize-handle-state=drag]:bg-primary data-[resize-handle-state=hover]:bg-primary";

export function MessageBrowser({ previewHandle, mode, packageProjection }: MessageBrowserProps) {
  const query = useProfferPreviewMessages(previewHandle, mode);
  const { rows, participants } = usePreviewMessageRows(query.data?.pages);
  const [filter, setFilter] = useState("");
  const [attachmentsOnly, setAttachmentsOnly] = useState(false);
  const [selectedMessageId, setSelectedMessageId] = useState<string | null>(null);
  const [loadingAll, setLoadingAll] = useState(false);

  const visibleRows = useMemo(
    () => filterLoadedRows(rows, filter, attachmentsOnly),
    [attachmentsOnly, filter, rows],
  );

  const selectedIndex = useMemo(() => {
    if (!selectedMessageId) return visibleRows.length ? 0 : -1;
    const index = visibleRows.findIndex((row) => row.message.message_id === selectedMessageId);
    return index === -1 ? (visibleRows.length ? 0 : -1) : index;
  }, [selectedMessageId, visibleRows]);

  const selectedRow = selectedIndex >= 0 ? visibleRows[selectedIndex] ?? null : null;

  const selectIndex = useCallback(
    (index: number) => {
      const row = visibleRows[index];
      if (row) setSelectedMessageId(row.message.message_id);
    },
    [visibleRows],
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

  return (
    <section
      className="platform-panel flex h-[38rem] min-h-0 flex-col overflow-hidden"
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
        {query.isFetching && !loadingAll && (
          <span className="flex items-center gap-2 text-xs text-muted-foreground" role="status">
            <Loader2 className="size-3.5 animate-spin motion-reduce:animate-none" /> Loading rows…
          </span>
        )}
      </header>

      <PanelGroup direction="horizontal" className="min-h-0 flex-1" autoSaveId="proffer-message-browser">
        <Panel id="message-list" order={1} defaultSize={52} minSize={30}>
          <div className="flex h-full min-h-0 flex-col">
            <MessageBrowserToolbar
              query={filter}
              onQueryChange={setFilter}
              attachmentsOnly={attachmentsOnly}
              onAttachmentsOnlyChange={setAttachmentsOnly}
              loadedCount={rows.length}
              visibleCount={visibleRows.length}
              hasMore={Boolean(hasNextPage)}
              loadingAll={loadingAll}
              fetching={isFetchingNextPage}
              onLoadAll={() => void loadAll()}
            />
            <div className="min-h-0 flex-1">
              {query.isPending ? (
                <p className="p-4 text-sm text-muted-foreground" role="status">Loading message rows…</p>
              ) : visibleRows.length === 0 ? (
                <p className="p-4 text-sm text-muted-foreground">
                  {rows.length === 0
                    ? "No message rows are projected for this attempt."
                    : "No loaded row matches the current filter."}
                </p>
              ) : (
                <MessageBrowserGrid
                  rows={visibleRows}
                  selectedIndex={selectedIndex}
                  onSelectIndex={selectIndex}
                  onReachEnd={requestNextPage}
                />
              )}
            </div>
          </div>
        </Panel>

        <PanelResizeHandle className={HANDLE_CLASS} />

        <Panel id="message-detail" order={2} defaultSize={28} minSize={18}>
          <MessageDetailPanel row={selectedRow} participants={participants} />
        </Panel>

        <PanelResizeHandle className={HANDLE_CLASS} />

        <Panel id="message-source" order={3} defaultSize={20} minSize={16}>
          <MessageSourcePanel row={selectedRow} packageProjection={packageProjection} />
        </Panel>
      </PanelGroup>
    </section>
  );
}
