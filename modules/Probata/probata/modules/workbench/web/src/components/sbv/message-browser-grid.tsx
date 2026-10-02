// Byline: Claude Code · Opus 5 · 2026-09-20 (Glide Data Grid message list with cursor-driven paging)
// Byline: Claude Code · Opus 5.5 · 2026-10-01 (useGridTheme moved to grid-theme.ts, shared with the Case page events grid)
"use client";

import {
  CompactSelection,
  DataEditor,
  GridCellKind,
  type GridCell,
  type GridColumn,
  type GridSelection,
  type Item,
} from "@glideapps/glide-data-grid";
import { useCallback, useMemo, useState } from "react";

import { useGridTheme } from "@/components/sbv/grid-theme";
import type { PreviewMessageRow } from "@/hooks/use-preview-messages";

import "@glideapps/glide-data-grid/dist/index.css";

/** Rows remaining below the viewport before the next cursor page is requested. */
const PREFETCH_ROW_MARGIN = 40;

const COLUMNS: GridColumn[] = [
  { title: "Time", id: "time", width: 190 },
  { title: "Sender", id: "sender", width: 160 },
  { title: "Direction", id: "direction", width: 96 },
  { title: "Message", id: "body", width: 620, grow: 1 },
  { title: "Files", id: "attachments", width: 110 },
];

interface MessageBrowserGridProps {
  rows: PreviewMessageRow[];
  selectedIndex: number;
  onSelectIndex: (index: number) => void;
  onReachEnd: () => void;
}

export function MessageBrowserGrid({
  rows,
  selectedIndex,
  onSelectIndex,
  onReachEnd,
}: MessageBrowserGridProps) {
  const [host, setHost] = useState<HTMLDivElement | null>(null);
  const theme = useGridTheme(host);

  const getCellContent = useCallback(
    ([column, row]: Item): GridCell => {
      const entry = rows[row];
      const text = (() => {
        if (!entry) return "";
        if (column === 0) return entry.timeLabel;
        if (column === 1) return entry.senderName;
        if (column === 2) return entry.direction === "out" ? "out" : "in";
        if (column === 3) return entry.bodyLine;
        if (!entry.attachmentCount) return "";
        return entry.missingPayloadCount
          ? `${entry.attachmentCount} · ${entry.missingPayloadCount} missing`
          : String(entry.attachmentCount);
      })();
      return {
        kind: GridCellKind.Text,
        data: text,
        displayData: text,
        allowOverlay: false,
        readonly: true,
      };
    },
    [rows],
  );

  const selection = useMemo<GridSelection>(
    () => ({
      columns: CompactSelection.empty(),
      rows: CompactSelection.empty(),
      current:
        selectedIndex >= 0 && selectedIndex < rows.length
          ? {
              cell: [0, selectedIndex] as Item,
              range: { x: 0, y: selectedIndex, width: COLUMNS.length, height: 1 },
              rangeStack: [],
            }
          : undefined,
    }),
    [rows.length, selectedIndex],
  );

  const handleSelectionChange = useCallback(
    (next: GridSelection) => {
      const cell = next.current?.cell;
      if (!cell) return;
      onSelectIndex(cell[1]);
    },
    [onSelectIndex],
  );

  return (
    <div ref={setHost} className="h-full min-h-0 w-full" data-testid="message-browser-grid">
      {theme && (
        <DataEditor
          width="100%"
          height="100%"
          columns={COLUMNS}
          rows={rows.length}
          rowHeight={30}
          headerHeight={32}
          theme={theme}
          getCellContent={getCellContent}
          gridSelection={selection}
          onGridSelectionChange={handleSelectionChange}
          rangeSelect="cell"
          rowMarkers="number"
          smoothScrollY
          keybindings={{ search: false }}
          onVisibleRegionChanged={(range) => {
            if (range.y + range.height + PREFETCH_ROW_MARGIN >= rows.length) onReachEnd();
          }}
        />
      )}
    </div>
  );
}
