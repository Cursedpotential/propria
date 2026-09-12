// Byline: Claude Code · Sonnet 5 · 2026-09-07
// Updated by: OpenAI Codex · GPT-5 · 2026-09-12 — host-aligned, readable Glide themes.
//
// Glide Data Grid over docket entries (filings/drafts/orders/upcoming
// court_events — see mcp-app/src/store.ts's `caseDocket`). Row click opens a
// detail drawer whose `source` link either opens the local file via Tauri's
// `opener` plugin or shows the R2 pointer as text (no network fetch from the
// UI — that stays server-side, per the sidecar boundary).
import "@glideapps/glide-data-grid/dist/index.css";

import { DataEditor, GridCellKind, type GridCell, type GridColumn, type Item, type Theme } from "@glideapps/glide-data-grid";
import * as React from "react";
import { useTheme } from "@/components/layout/theme-provider";
import { Badge } from "@/components/ui/badge";
import type { DocketEntry } from "@/types/store";
import { DocketDetailDrawer } from "./docket-detail-drawer";

const COLUMNS: GridColumn[] = [
  { title: "Title", id: "title", width: 320 },
  { title: "Table", id: "table", width: 110 },
  { title: "Status", id: "status", width: 130 },
  { title: "In force", id: "in_force", width: 90 },
  { title: "Date", id: "date", width: 120 },
];

/** `record.in_force` only means anything for the `order` table. */
function inForceLabel(entry: DocketEntry): string {
  if (entry.table !== "order") return "—";
  const val = entry.record.in_force;
  return val === undefined || val === null ? "—" : val ? "yes" : "no";
}

export function DocketGrid({ entries }: { entries: DocketEntry[] }) {
  const [selected, setSelected] = React.useState<DocketEntry | null>(null);
  const { theme } = useTheme();

  const gridTheme = React.useMemo<Partial<Theme>>(
    () =>
      theme === "dark"
        ? {
            accentColor: "#7697b8",
            accentFg: "#111820",
            accentLight: "rgba(118, 151, 184, 0.18)",
            textDark: "#f0f1ef",
            textMedium: "#b1b8bd",
            textLight: "#8b959c",
            textHeader: "#d9dfe2",
            textHeaderSelected: "#f0f1ef",
            bgCell: "#242e36",
            bgCellMedium: "#2c373f",
            bgHeader: "#202b33",
            bgHeaderHasFocus: "#314050",
            bgHeaderHovered: "#2c3944",
            bgSearchResult: "#313a66",
            borderColor: "#43505a",
            horizontalBorderColor: "#36434c",
            headerBottomBorderColor: "#62707a",
            linkColor: "#a9c0d6",
            fontFamily: '"CaskaydiaCove Nerd Font Propo", "CaskaydiaCove Nerd Font", "JetBrainsMono Nerd Font", "Segoe UI", sans-serif',
            baseFontStyle: "14px",
            headerFontStyle: "600 13px",
            lineHeight: 1.55,
            roundingRadius: 4,
          }
        : {
            accentColor: "#4051b9",
            accentFg: "#ffffff",
            accentLight: "#e9ecfb",
            textDark: "#1d2228",
            textMedium: "#687078",
            textLight: "#8a9096",
            textHeader: "#d9dfe2",
            textHeaderSelected: "#ffffff",
            bgCell: "#fffefb",
            bgCellMedium: "#ebe8e0",
            bgHeader: "#202b33",
            bgHeaderHasFocus: "#314050",
            bgHeaderHovered: "#2c3944",
            bgSearchResult: "#e9ecfb",
            borderColor: "#d5d1c9",
            horizontalBorderColor: "#e2ded6",
            headerBottomBorderColor: "#3d4952",
            linkColor: "#2f3d9c",
            fontFamily: '"CaskaydiaCove Nerd Font Propo", "CaskaydiaCove Nerd Font", "JetBrainsMono Nerd Font", "Segoe UI", sans-serif',
            baseFontStyle: "14px",
            headerFontStyle: "600 13px",
            lineHeight: 1.55,
            roundingRadius: 4,
          },
    [theme],
  );

  const getCellContent = React.useCallback(
    ([col, row]: Item): GridCell => {
      const entry = entries[row];
      const text = (() => {
        switch (col) {
          case 0:
            return entry.title;
          case 1:
            return entry.table;
          case 2:
            return entry.status ?? "—";
          case 3:
            return inForceLabel(entry);
          case 4:
            return entry.date ?? "—";
          default:
            return "";
        }
      })();
      return {
        kind: GridCellKind.Text,
        data: text,
        displayData: text,
        allowOverlay: false,
      };
    },
    [entries],
  );

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-2">
        <h1 className="text-sm font-semibold text-text-primary">Docket</h1>
        <Badge tone="neutral">{entries.length} entries</Badge>
      </div>
      <div className="overflow-hidden rounded-[var(--radius-md)] border border-border">
        <DataEditor
          columns={COLUMNS}
          theme={gridTheme}
          rows={entries.length}
          getCellContent={getCellContent}
          onCellClicked={([, row]) => setSelected(entries[row] ?? null)}
          smoothScrollX
          smoothScrollY
          rowHeight={32}
          headerHeight={32}
          width="100%"
          height={Math.min(560, 32 + entries.length * 32 + 4)}
        />
      </div>
      {selected && <DocketDetailDrawer entry={selected} onClose={() => setSelected(null)} />}
    </div>
  );
}
