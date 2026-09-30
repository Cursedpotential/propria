// Byline: Claude Code · Opus 5.5 · 2026-09-28
// Cancel this run (D05-C06). Owner 2026-09-27 22:05, of two parked TEST runs: "sounds like
// there's an issue to fix" — the Workbench had no way to stop or discard a run. The engine
// cancels through Temporal and keeps who and why in the run's own history; nothing is deleted,
// and the run stays listed as "cancelled".
"use client";

import { useQueryClient } from "@tanstack/react-query";
import { Loader2, XCircle } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { ApiError, cancelProfferRun } from "@/lib/api-client";
import type { ProfferOperatorSnapshot } from "@/lib/shared/types";

export function CancelRunSection({ snapshot }: { snapshot: ProfferOperatorSnapshot }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [requested, setRequested] = useState(false);

  if (snapshot.terminal) {
    return snapshot.lifecycle === "cancelled" ? (
      <section className="border-t pt-3 text-xs text-muted-foreground" data-testid="review-actions-cancel">
        Cancelled{snapshot.reason ? ` — ${snapshot.reason.replace(/^cancelled( by)?\s*/, "")}` : ""}
      </section>
    ) : null;
  }

  async function confirm() {
    setPending(true);
    setError(null);
    try {
      await cancelProfferRun(snapshot.preview_handle, snapshot.matter_mode, reason.trim());
      setRequested(true);
      setOpen(false);
      await queryClient.invalidateQueries();
    } catch (failure) {
      setError(failure instanceof ApiError || failure instanceof Error ? failure.message : "Cancel failed");
    } finally {
      setPending(false);
    }
  }

  return (
    <section aria-labelledby="review-cancel-heading" className="space-y-1.5 border-t pt-3" data-testid="review-actions-cancel">
      <h3 id="review-cancel-heading" className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Cancel</h3>
      {requested ? (
        <p className="text-xs" role="status">Cancel requested. The run stops and is listed as cancelled.</p>
      ) : open ? (
        <div className="space-y-1.5">
          <label className="block text-xs" htmlFor="review-cancel-reason">Why cancel this run?</label>
          <input
            id="review-cancel-reason"
            className="h-8 w-full rounded-md border border-input bg-transparent px-2 text-xs"
            value={reason}
            maxLength={4000}
            onChange={(event) => setReason(event.target.value)}
            placeholder="Required"
          />
          <div className="flex gap-2">
            <Button type="button" size="sm" variant="destructive" disabled={pending || !reason.trim()} onClick={() => void confirm()}>
              {pending ? <Loader2 className="size-3.5 animate-spin motion-reduce:animate-none" /> : <XCircle className="size-3.5" />} Cancel run
            </Button>
            <Button type="button" size="sm" variant="ghost" disabled={pending} onClick={() => setOpen(false)}>Keep it</Button>
          </div>
        </div>
      ) : (
        <Button type="button" size="sm" variant="outline" className="w-full" onClick={() => setOpen(true)} data-testid="review-cancel-open">
          <XCircle className="size-3.5" /> Cancel this run
        </Button>
      )}
      {error && <p className="text-xs text-destructive" role="alert">{error}</p>}
    </section>
  );
}
