// Byline: Claude Code · Opus 5.5 · 2026-09-26
// One block of metadata fields with the owner's corrections beside them. The
// observed value is always shown as recorded; a correction is a new attributed
// revision (append-only), and withdrawing one is a revision too.
"use client";

import { History, Loader2, Pencil, Undo2 } from "lucide-react";
import { useState } from "react";

import { formatValue, formatWhen, parseCorrectedValue } from "@/components/metadata/metadata-format";
import { SmallFlag } from "@/components/metadata/small-flag";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { MetadataCorrection } from "@/lib/review-overlays-client";

export interface FieldRow {
  fieldKey: string;
  label: string;
  value: unknown;
}

export interface CorrectionSubmit {
  fieldKey: string;
  observed: unknown;
  supersedesRef: string;
  action: "correct" | "retract";
  correctedValue: unknown;
  reason: string;
}

export function MetadataFieldTable({
  rows,
  corrections,
  conflicts,
  correctable,
  pendingField,
  onSubmit,
}: {
  rows: FieldRow[];
  corrections: Map<string, MetadataCorrection[]>;
  conflicts?: Map<string, string>;
  correctable: boolean;
  pendingField: string | null;
  onSubmit: (input: CorrectionSubmit) => Promise<boolean>;
}) {
  const [editing, setEditing] = useState<string | null>(null);
  const [draftValue, setDraftValue] = useState("");
  const [draftReason, setDraftReason] = useState("");
  const [openHistory, setOpenHistory] = useState<Set<string>>(new Set());

  if (!rows.length) return <p className="px-3 py-2 text-xs text-muted-foreground">No fields recorded.</p>;

  const startEdit = (row: FieldRow, current: MetadataCorrection | undefined) => {
    setEditing(row.fieldKey);
    setDraftValue(formatValue(current?.action === "correct" ? current.corrected_value : row.value).replace(/^—$/, ""));
    setDraftReason("");
  };

  const submit = async (row: FieldRow, newest: MetadataCorrection | undefined, action: "correct" | "retract") => {
    const saved = await onSubmit({
      fieldKey: row.fieldKey,
      observed: row.value ?? null,
      supersedesRef: newest?.correction_ref ?? "",
      action,
      correctedValue: action === "correct" ? parseCorrectedValue(draftValue, row.value) : null,
      reason: draftReason.trim(),
    });
    if (saved) setEditing(null);
  };

  return (
    <dl className="divide-y border">
      {rows.map((row) => {
        const history = corrections.get(row.fieldKey) ?? [];
        const newest = history[0];
        const active = newest?.action === "correct" ? newest : undefined;
        const conflict = conflicts?.get(row.fieldKey);
        const isEditing = editing === row.fieldKey;
        const showHistory = openHistory.has(row.fieldKey);
        return (
          <div key={row.fieldKey} className="grid gap-1 px-3 py-1.5 text-xs md:grid-cols-[minmax(10rem,18rem)_minmax(0,1fr)_auto]" data-field-key={row.fieldKey}>
            <dt className="break-all font-mono text-[10px] text-muted-foreground">{row.label}</dt>
            <dd className="min-w-0 space-y-1">
              <p className="flex flex-wrap items-start gap-1.5">
                <span className={active ? "break-all text-muted-foreground" : "break-all"} title="Recorded value (never changed)">{formatValue(row.value)}</span>
                {conflict && <SmallFlag title={conflict}>sidecar disagrees</SmallFlag>}
              </p>
              {active && (
                <p className="flex flex-wrap items-center gap-1.5">
                  <SmallFlag tone="info" title={`${active.change_reason} · ${active.actor_username} · ${formatWhen(active.recorded_at)}`}>corrected · r{active.revision}</SmallFlag>
                  <span className="break-all font-medium">{formatValue(active.corrected_value)}</span>
                </p>
              )}
              {isEditing && (
                <div className="grid gap-1.5 border-l-2 border-primary pl-2 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]">
                  <Input aria-label={`Corrected value for ${row.label}`} className="h-7 text-xs" value={draftValue} onChange={(event) => setDraftValue(event.target.value)} />
                  <Input aria-label="Reason for the correction" className="h-7 text-xs" placeholder="Why (required)" value={draftReason} onChange={(event) => setDraftReason(event.target.value)} maxLength={4000} />
                  <span className="flex gap-1">
                    <Button size="sm" className="h-7" disabled={!draftReason.trim() || !draftValue.trim() || pendingField === row.fieldKey} onClick={() => void submit(row, newest, "correct")}>
                      {pendingField === row.fieldKey ? <Loader2 className="size-3.5 animate-spin" /> : null} Save
                    </Button>
                    {active && (
                      <Button size="sm" variant="outline" className="h-7" disabled={!draftReason.trim() || pendingField === row.fieldKey} onClick={() => void submit(row, newest, "retract")} title="Withdraw the correction; the recorded value stands again">
                        <Undo2 className="size-3.5" /> Withdraw
                      </Button>
                    )}
                    <Button size="sm" variant="ghost" className="h-7" onClick={() => setEditing(null)}>Cancel</Button>
                  </span>
                </div>
              )}
              {showHistory && history.length > 0 && (
                <ol className="space-y-0.5 border-l pl-2 text-[10px] text-muted-foreground" aria-label={`Correction history for ${row.label}`}>
                  {history.map((item) => (
                    <li key={item.correction_ref}>
                      r{item.revision} · {item.action === "retract" ? "withdrawn" : formatValue(item.corrected_value)} · {item.change_reason} · {item.actor_username} · {formatWhen(item.recorded_at)}
                    </li>
                  ))}
                </ol>
              )}
            </dd>
            <span className="flex items-start gap-0.5">
              {history.length > 0 && (
                <Button size="icon" variant="ghost" className="size-6" title="Correction history" aria-pressed={showHistory} onClick={() => setOpenHistory((previous) => {
                  const next = new Set(previous);
                  if (next.has(row.fieldKey)) next.delete(row.fieldKey);
                  else next.add(row.fieldKey);
                  return next;
                })}>
                  <History className="size-3.5" />
                </Button>
              )}
              {correctable && !isEditing && (
                <Button size="icon" variant="ghost" className="size-6" title="Correct this value (the recorded value is kept)" onClick={() => startEdit(row, newest)}>
                  <Pencil className="size-3.5" />
                </Button>
              )}
            </span>
          </div>
        );
      })}
    </dl>
  );
}
