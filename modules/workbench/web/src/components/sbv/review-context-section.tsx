// Byline: Claude Code · Opus 5.5 · 2026-09-25
// Review Actions — Context: the run's current operator context (who, what, when, how
// acquired, notes), editable in place. A save is a new append-only revision that
// supersedes exactly the revision shown; nothing is overwritten.
"use client";

import { Loader2, PencilLine, RotateCcw } from "lucide-react";
import { useState } from "react";

import { SourceAssertionsFields } from "@/components/intake/source-assertions-fields";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { createProfferSourceContext } from "@/lib/api-client";
import type {
  ProfferHumanSourceAssertions,
  ProfferOperatorSnapshot,
  ProfferRunSourceContext,
  ProfferSourceContextReceipt,
} from "@/lib/shared/types";
import {
  assertionSummary,
  assertionsForForm,
  assertionsForSubmit,
  EMPTY_ASSERTIONS,
  observationFromRegistration,
} from "@/lib/source-assertions";

function when(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString(undefined, { month: "short", day: "numeric", year: "numeric", hour: "numeric", minute: "2-digit" });
}

export function ReviewContextSection({
  snapshot,
  runContext,
  loading,
  error,
  refreshing,
  rerunPending,
  onSaved,
  onRerunWithContext,
}: {
  snapshot: ProfferOperatorSnapshot;
  runContext: ProfferRunSourceContext | null;
  loading: boolean;
  error: string | null;
  refreshing: boolean;
  rerunPending: boolean;
  onSaved: () => void;
  onRerunWithContext: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState<ProfferHumanSourceAssertions>(EMPTY_ASSERTIONS);
  const [reason, setReason] = useState("");
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saved, setSaved] = useState<ProfferSourceContextReceipt | null>(null);
  const [showAll, setShowAll] = useState(false);

  const current = runContext?.current ?? null;
  const observation = current?.observed_source ?? observationFromRegistration(snapshot.source_ref, runContext?.registration);
  const summary = assertionSummary(current?.assertions);
  const visibleSummary = showAll ? summary : summary.slice(0, 5);

  function update<K extends keyof ProfferHumanSourceAssertions>(key: K, value: ProfferHumanSourceAssertions[K]) {
    setForm((previous) => ({ ...previous, [key]: value }));
  }

  function startEditing() {
    setForm(assertionsForForm(current?.assertions));
    setReason(current ? "Context corrected in Review" : "Context added in Review");
    setSaveError(null);
    setOpen(true);
  }

  async function save() {
    if (!runContext || !observation || !reason.trim()) return;
    setSaving(true);
    setSaveError(null);
    try {
      const receipt = await createProfferSourceContext({
        request_id: runContext.request_id,
        matter_id: snapshot.matter_id,
        court_case_id: snapshot.court_case_id,
        source_ref: snapshot.source_ref,
        observed_source: observation,
        supersedes_ref: current?.source_context_ref ?? null,
        assertions: assertionsForSubmit(form, current?.assertions),
        change_reason: reason.trim(),
        matter_mode: snapshot.matter_mode,
      });
      setSaved(receipt);
      setOpen(false);
      onSaved();
    } catch (requestError) {
      setSaveError(requestError instanceof Error ? requestError.message : "The context could not be saved");
    } finally {
      setSaving(false);
    }
  }

  const blockedReason = error
    ? null
    : runContext && !observation
      ? "This run never kept its original file, so there is nothing to attach context to. Re-run it first."
      : null;

  return (
    <section aria-labelledby="review-context-heading" className="space-y-1.5" data-testid="review-actions-context">
      <header className="flex items-center justify-between gap-2">
        <h3 id="review-context-heading" className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Context &amp; metadata</h3>
        <Button type="button" variant="outline" size="sm" className="h-7 px-2 text-xs" disabled={loading || Boolean(error) || !observation} onClick={startEditing}>
          <PencilLine className="size-3.5" /> {current ? "Edit" : "Add context"}
        </Button>
      </header>
      {loading ? (
        <p className="flex items-center gap-2 text-xs text-muted-foreground"><Loader2 className="size-3.5 animate-spin motion-reduce:animate-none" /> Reading this run&apos;s context…</p>
      ) : error ? (
        <p className="truncate text-xs text-muted-foreground" title={error}>Context read-back unavailable: {error}</p>
      ) : current ? (
        <>
          <dl className="space-y-0.5 text-xs">
            {visibleSummary.map(([label, value]) => (
              <div key={label} className="grid grid-cols-[6.5rem_minmax(0,1fr)] gap-2">
                <dt className="text-muted-foreground">{label}</dt>
                <dd className="line-clamp-2 break-words" title={value}>{value}</dd>
              </div>
            ))}
          </dl>
          {summary.length > 5 && <button type="button" className="text-[11px] text-muted-foreground underline-offset-2 hover:underline" onClick={() => setShowAll((value) => !value)}>{showAll ? "Show less" : `Show all ${summary.length}`}</button>}
          <p className="text-[11px] text-muted-foreground">Revision {current.revision} · {current.actor_username || "operator"} · {when(current.recorded_at)}</p>
        </>
      ) : (
        <p className="text-xs text-muted-foreground">No context recorded for this run yet.</p>
      )}
      {blockedReason && <p className="text-[11px] text-muted-foreground">{blockedReason}</p>}
      {saved && (
        <div className="flex flex-wrap items-center gap-2 border-l-2 border-l-[#2f9d67] pl-2 text-xs">
          <span className="font-semibold text-[#17794b] dark:text-[#72d9a1]">Saved as revision {saved.revision}</span>
          <Button type="button" size="sm" variant="outline" className="h-7 px-2 text-xs" disabled={refreshing || rerunPending} onClick={onRerunWithContext}>
            <RotateCcw className="size-3.5" /> Re-run with this context
          </Button>
        </div>
      )}

      <Dialog open={open} onOpenChange={(next) => !saving && setOpen(next)}>
        <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{current ? `Edit context (revision ${current.revision})` : "Add context"}</DialogTitle>
            <DialogDescription>
              Your assertions about this source. Saving records a new revision with your identity; earlier revisions stay as they were.
            </DialogDescription>
          </DialogHeader>
          <SourceAssertionsFields value={form} onChange={update} className="mt-0" />
          <label className="grid gap-1.5 text-xs font-semibold">Why this change
            <input className="h-9 border bg-background px-3 font-normal" value={reason} maxLength={4000} onChange={(event) => setReason(event.target.value)} />
          </label>
          {saveError && <p className="text-xs text-destructive" role="alert">{saveError}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" disabled={saving} onClick={() => setOpen(false)}>Cancel</Button>
            <Button type="button" disabled={saving || !reason.trim()} onClick={() => void save()}>
              {saving && <Loader2 className="size-4 animate-spin motion-reduce:animate-none" />} Save revision
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
}
