// Byline: Claude Code · Sonnet 5 · 2026-09-07
// Updated by: OpenAI Codex · GPT-5 · 2026-09-12 — resolved Probata graphite/indigo Glide themes.
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

function contractToken(name: string, fallback: string): string {
  if (typeof document === "undefined") return fallback;
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback;
}

export function DocketGrid({ entries }: { entries: DocketEntry[] }) {
  const [selected, setSelected] = React.useState<DocketEntry | null>(null);
  const { theme } = useTheme();

  const gridTheme = React.useMemo<Partial<Theme>>(
    () => ({
      accentColor: contractToken("--pr-action", theme === "dark" ? "#8591F0" : "#4051B9"),
      accentFg: contractToken("--pr-action-text", theme === "dark" ? "#111820" : "#ffffff"),
      accentLight: contractToken("--pr-action-soft", theme === "dark" ? "#313A66" : "#E9ECFB"),
      textDark: contractToken("--pr-ink", theme === "dark" ? "#F7F8F7" : "#1D2228"),
      textMedium: contractToken("--pr-ink-muted", theme === "dark" ? "#B1B8BD" : "#687078"),
      textLight: contractToken("--pr-ink-muted", theme === "dark" ? "#B1B8BD" : "#687078"),
      textHeader: contractToken("--pr-shell-text", theme === "dark" ? "#F7F8F7" : "#F7F8F7"),
      textHeaderSelected: contractToken("--pr-shell-text", theme === "dark" ? "#F7F8F7" : "#F7F8F7"),
      bgCell: contractToken("--pr-surface", theme === "dark" ? "#242E36" : "#FFFEFB"),
      bgCellMedium: contractToken("--pr-surface-muted", theme === "dark" ? "#2C373F" : "#EBE8E0"),
      bgHeader: contractToken("--pr-shell-surface", theme === "dark" ? "#202B33" : "#202B33"),
      bgHeaderHasFocus: contractToken("--pr-action-soft", theme === "dark" ? "#313A66" : "#E9ECFB"),
      bgHeaderHovered: contractToken("--pr-border-strong", theme === "dark" ? "#62707A" : "#B8B6B0"),
      bgSearchResult: contractToken("--pr-action-soft", theme === "dark" ? "#313A66" : "#E9ECFB"),
      borderColor: contractToken("--pr-border", theme === "dark" ? "#43505A" : "#D5D1C9"),
      horizontalBorderColor: contractToken("--pr-border", theme === "dark" ? "#43505A" : "#D5D1C9"),
      headerBottomBorderColor: contractToken("--pr-border-strong", theme === "dark" ? "#62707A" : "#B8B6B0"),
      linkColor: contractToken("--pr-information", theme === "dark" ? "#82bdc0" : "#376f72"),
      fontFamily: contractToken("--pr-font-ui", '"Instrument Sans", "Segoe UI", sans-serif'),
      baseFontStyle: "14px",
      headerFontStyle: "600 13px",
      lineHeight: 1.55,
      roundingRadius: 4,
    }),
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
