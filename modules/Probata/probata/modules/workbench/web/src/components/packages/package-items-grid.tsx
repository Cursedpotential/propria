// Byline: Codex · GPT-6 · 2026-10-12.
// Adapted from sbv/message-browser-grid.tsx: same Glide selection and token theme.
import { CompactSelection, DataEditor, GridCellKind, type GridCell, type GridColumn, type GridSelection, type Item } from "@glideapps/glide-data-grid";
import { useCallback, useMemo, useState } from "react";
import { useGridTheme } from "@/components/sbv/grid-theme";
import type { ContextPackageItem } from "@/lib/context-package-client";
import "@glideapps/glide-data-grid/dist/index.css";

const COLUMNS: GridColumn[] = [
  { id: "kind", title: "Kind", width: 120 }, { id: "label", title: "Extracted item", width: 300, grow: 1 },
  { id: "state", title: "Review", width: 115 }, { id: "sources", title: "Sources", width: 75 }, { id: "edges", title: "Edges", width: 70 },
];

/** Render the current section's items with persistent selection driving the real editor.
 * Inputs: bounded rows, selected ID and selection callback. Output: token-themed Glide table.
 * Effects: selection only. Pick for all package kinds; complete content stays in the field panel.
 */
export function PackageItemsGrid({ rows, selectedId, onSelect }: { rows: ContextPackageItem[]; selectedId: string | null; onSelect: (id: string) => void }) {
  const [host, setHost] = useState<HTMLDivElement | null>(null);
  const theme = useGridTheme(host);
  const index = rows.findIndex((row) => row.id === selectedId);
  const selection = useMemo<GridSelection>(() => ({
    columns: CompactSelection.empty(), rows: CompactSelection.empty(),
    current: index >= 0 ? { cell: [0, index] as Item, range: { x: 0, y: index, width: COLUMNS.length, height: 1 }, rangeStack: [] } : undefined,
  }), [index]);
  const getCellContent = useCallback(([column, row]: Item): GridCell => {
    const item = rows[row];
    const value = item ? [item.kind, item.label, item.review_state, String(item.source_refs.length), String(item.edges.length)][column] ?? "" : "";
    return { kind: GridCellKind.Text, data: value, displayData: value, allowOverlay: false, readonly: true };
  }, [rows]);
  return <div ref={setHost} className="h-full min-h-[12rem] w-full" data-testid="package-items-grid">
    {theme && <DataEditor width="100%" height="100%" columns={COLUMNS} rows={rows.length} rowHeight={34} headerHeight={34} theme={theme}
      getCellContent={getCellContent} gridSelection={selection} rowMarkers="number" smoothScrollY rangeSelect="cell" keybindings={{ search: false }}
      onGridSelectionChange={(next) => { const row = next.current?.cell[1]; if (row !== undefined && rows[row]) onSelect(rows[row].id); }} />}
  </div>;
}
