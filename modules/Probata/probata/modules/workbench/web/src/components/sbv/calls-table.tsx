// Ported from modules/forks/sbv/frontend/src/components/Calls.jsx
// MIT, Copyright (c) 2025 lowcarbdev
// Supersedes this file's prior Glide Data Grid implementation (Claude Code · Opus 5 ·
// 2026-09-20) with SBV's own card-list layout: an icon + name/number on the left,
// a direction/disposition badge and timestamp on the right, and a duration line
// underneath for answered calls. Adapted: Bootstrap classes and inline hex colors
// replaced with this app's shadcn tokens; SBV fetched its own `${API_BASE}/calls`
// page with an IntersectionObserver-driven infinite scroll — this component is fed
// already-parsed rows by its caller (`content.records` via `parseCallRecords`,
// wired to the existing "Load more records" button in proffer-operator-preview.tsx),
// so no fetching or intersection-observer bookkeeping lives here. The
// `parseCallRecord`/`parseCallRecords` pure parsing helpers are this app's own
// (not from SBV, whose Calls.jsx reads a different call-record shape) and are kept
// unchanged, exported, and unit-testable.
// Byline: Claude Code · Opus 5 · 2026-09-20
"use client";

import { Phone, PhoneCall, PhoneIncoming, PhoneMissed, PhoneOff, PhoneOutgoing } from "lucide-react";

import { NumberIdentity } from "@/components/identity/number-status";
import { Badge } from "@/components/ui/badge";
import type { ProfferGenericRecord } from "@/lib/shared/types";

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

export type CallDirection = "incoming" | "outgoing" | "unknown";

function normalizeDirection(direction: unknown): CallDirection {
  return direction === "incoming" || direction === "outgoing" ? direction : "unknown";
}

/** One row of the Calls view, parsed defensively from a `call` record's payload. */
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

function directionMeta(row: CallRow) {
  if (row.missed) return { label: "Missed", Icon: PhoneMissed, tone: "text-destructive" };
  if (row.direction === "incoming") return { label: "Incoming", Icon: PhoneIncoming, tone: "text-emerald-600 dark:text-emerald-400" };
  if (row.direction === "outgoing") return { label: "Outgoing", Icon: PhoneOutgoing, tone: "text-primary" };
  return { label: "Call", Icon: Phone, tone: "text-muted-foreground" };
}

interface CallsTableProps {
  rows: CallRow[];
}

/** SBV-styled call history list: one card per call, icon + number + direction badge + duration. */
export function CallsTable({ rows }: CallsTableProps) {
  if (!rows.length) {
    return (
      <div className="border p-6 text-center text-sm text-muted-foreground" data-testid="calls-table-empty">
        No call records are projected for this attempt.
      </div>
    );
  }

  return (
    <div className="space-y-2" data-testid="calls-table">
      {rows.map((row) => {
        const { label, Icon, tone } = directionMeta(row);
        return (
          <div
            key={row.recordId}
            className="flex items-center justify-between gap-3 rounded-md border bg-card px-3 py-2.5 shadow-sm"
            data-testid="calls-table-row"
          >
            <div className="flex min-w-0 items-center gap-3">
              <div className={`shrink-0 rounded-full bg-muted p-2 ${tone}`}>
                {row.missed ? <PhoneOff className="size-4" /> : row.direction === "unknown" ? <PhoneCall className="size-4" /> : <Icon className="size-4" />}
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{row.number}</p>
                <p className="text-xs text-muted-foreground">Duration: {row.durationLabel}</p>
                <NumberIdentity value={row.number} context="a call" />
              </div>
            </div>
            <div className="shrink-0 text-right">
              <Badge variant={row.missed ? "destructive" : "outline"} className="text-[11px]">
                {label}{row.disposition !== "—" ? ` · ${row.disposition}` : ""}
              </Badge>
              <p className="mt-1 text-xs text-muted-foreground">{row.whenLabel}</p>
            </div>
          </div>
        );
      })}
    </div>
  );
}
