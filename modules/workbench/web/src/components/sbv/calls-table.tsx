// Byline: Claude Code · Opus 5 · 2026-09-20 (calls projection table for the Review surface)
"use client";

import {
  CompactSelection,
  DataEditor,
  GridCellKind,
  type GridCell,
  type GridColumn,
  type GridSelection,
  type Item,
  type Theme,
} from "@glideapps/glide-data-grid";
import { useCallback, useMemo, useState } from "react";

import { useTheme } from "@/components/layout/theme-provider";
import type { ProfferGenericRecord } from "@/lib/shared/types";

import "@glideapps/glide-data-grid/dist/index.css";

const COLUMNS: GridColumn[] = [
  { title: "When", id: "when", width: 190 },
  { title: "Direction", id: "direction", width: 120 },
  { title: "Number", id: "number", width: 160 },
  { title: "Duration", id: "duration", width: 100 },
  { title: "Disposition", id: "disposition", width: 140, grow: 1 },
];

function cssVariable(element: HTMLElement, name: string, fallback: string) {
  const value = getComputedStyle(element).getPropertyValue(name).trim();
  return value || fallback;
}

/** Used only when a custom property is missing; mirrors `src/app/globals.css`. */
const FALLBACK_PALETTE = {
  light: {
    primary: "#4051b9",
    accent: "#e9ecfb",
    foreground: "#1d2228",
    mutedForeground: "#687078",
    card: "#fffefb",
    muted: "#ebe8e0",
    border: "#d5d1c9",
  },
  dark: {
    primary: "#8591f0",
    accent: "#313a66",
    foreground: "#f0f1ef",
    mutedForeground: "#b1b8bd",
    card: "#242e36",
    muted: "#2c373f",
    border: "#43505a",
  },
} as const;

/**
 * Glide paints to a canvas, so it cannot inherit the surface tokens through CSS.
 * The palette is read back from the live custom properties whenever the resolved
 * light/dark theme changes, which keeps the grid inside the existing design system.
 * (Same pattern as `message-browser-grid.tsx` — kept in sync deliberately.)
 */
function useGridTheme(host: HTMLElement | null): Partial<Theme> | undefined {
  const { resolvedTheme } = useTheme();

  const fallback = FALLBACK_PALETTE[resolvedTheme === "dark" ? "dark" : "light"];

  return useMemo(() => {
    if (!host) return undefined;
    return {
      accentColor: cssVariable(host, "--primary", fallback.primary),
      accentLight: cssVariable(host, "--accent", fallback.accent),
      textDark: cssVariable(host, "--foreground", fallback.foreground),
      textMedium: cssVariable(host, "--muted-foreground", fallback.mutedForeground),
      textLight: cssVariable(host, "--muted-foreground", fallback.mutedForeground),
      textHeader: cssVariable(host, "--foreground", fallback.foreground),
      bgCell: cssVariable(host, "--card", fallback.card),
      bgCellMedium: cssVariable(host, "--muted", fallback.muted),
      bgHeader: cssVariable(host, "--muted", fallback.muted),
      bgHeaderHasFocus: cssVariable(host, "--accent", fallback.accent),
      bgHeaderHovered: cssVariable(host, "--accent", fallback.accent),
      borderColor: cssVariable(host, "--border", fallback.border),
      horizontalBorderColor: cssVariable(host, "--border", fallback.border),
      fontFamily: cssVariable(host, "--font-sans", "system-ui, sans-serif"),
      baseFontStyle: "13px",
      headerFontStyle: "600 12px",
    };
  }, [fallback, host]);
}

export type CallDirection = "incoming" | "outgoing" | "unknown";

/** One row of the Calls table, parsed defensively from a `call` record's payload. */
export interface CallRow {
  recordId: string;
  ordinal: number;
  whenLabel: string;
  direction: CallDirection;
  missed: boolean;
  number: string;
  durationLabel: string;
  disposition: string;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function formatWhen(occurredAt: unknown): string {
  if (typeof occurredAt !== "string" || !occurredAt) return "—";
  const parsed = new Date(occurredAt);
  if (Number.isNaN(parsed.getTime())) return occurredAt;
  return parsed.toLocaleString();
}

function formatDuration(durationSeconds: unknown): string {
  if (typeof durationSeconds !== "number" || !Number.isFinite(durationSeconds) || durationSeconds < 0) {
    return "—";
  }
  const total = Math.round(durationSeconds);
  const minutes = Math.floor(total / 60);
  const seconds = total % 60;
  if (minutes <= 0) return `${seconds}s`;
  return `${minutes}m ${String(seconds).padStart(2, "0")}s`;
}

function normalizeDirection(direction: unknown): CallDirection {
  return direction === "incoming" || direction === "outgoing" ? direction : "unknown";
}

/**
 * Parses one `ProfferGenericRecord` with `record_type === "call"` into a display
 * row. Every payload field is read defensively (`unknown` in, never a blind cast)
 * so a malformed or partial record renders blank cells instead of throwing.
 * Returns `null` for a non-call record.
 */
export function parseCallRecord(record: ProfferGenericRecord): CallRow | null {
  if (record.record_type !== "call") return null;

  const payload = isRecord(record.payload) ? record.payload : {};
  const content = isRecord(payload.content) ? payload.content : {};
  const participants = Array.isArray(payload.participants) ? payload.participants : [];
  const firstParticipant = participants.find(isRecord);
  const identifier = firstParticipant && typeof firstParticipant.identifier === "string"
    ? firstParticipant.identifier
    : "";

  const occurredAt = typeof payload.occurred_at === "string" ? payload.occurred_at : record.occurred_at;
  const missed = content.missed === true;
  const disposition = typeof content.disposition === "string" && content.disposition ? content.disposition : "—";

  return {
    recordId: record.record_id,
    ordinal: record.ordinal,
    whenLabel: formatWhen(occurredAt),
    direction: normalizeDirection(content.direction),
    missed,
    number: identifier || "—",
    durationLabel: formatDuration(content.duration_seconds),
    disposition,
  };
}

/** Extracts and parses every call row out of a page of content records, in order. */
export function parseCallRecords(records: ProfferGenericRecord[]): CallRow[] {
  const rows: CallRow[] = [];
  for (const record of records) {
    const row = parseCallRecord(record);
    if (row) rows.push(row);
  }
  return rows;
}

function directionLabel(row: CallRow): string {
  const base = row.direction === "incoming" ? "Incoming" : row.direction === "outgoing" ? "Outgoing" : "Unknown";
  return row.missed ? `${base} (missed)` : base;
}

interface CallsTableProps {
  rows: CallRow[];
}

export function CallsTable({ rows }: CallsTableProps) {
  const [host, setHost] = useState<HTMLDivElement | null>(null);
  const theme = useGridTheme(host);
  const [selectedIndex, setSelectedIndex] = useState(-1);

  const getCellContent = useCallback(
    ([column, row]: Item): GridCell => {
      const entry = rows[row];
      const text = (() => {
        if (!entry) return "";
        if (column === 0) return entry.whenLabel;
        if (column === 1) return directionLabel(entry);
        if (column === 2) return entry.number;
        if (column === 3) return entry.durationLabel;
        return entry.disposition;
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

  const handleSelectionChange = useCallback((next: GridSelection) => {
    const cell = next.current?.cell;
    if (!cell) return;
    setSelectedIndex(cell[1]);
  }, []);

  if (!rows.length) {
    return (
      <div className="border p-6 text-center text-sm text-muted-foreground" data-testid="calls-table-empty">
        No call records are projected for this attempt.
      </div>
    );
  }

  return (
    <div ref={setHost} className="h-[28rem] min-h-0 w-full" data-testid="calls-table">
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
        />
      )}
    </div>
  );
}
