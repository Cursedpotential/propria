import { useQuery } from "@tanstack/react-query";
import { AlertCircle, ArrowUpRight, Loader2 } from "lucide-react";

import { AppLink } from "@/lib/router-compat";
import { getProfferBatch } from "@/lib/api-client";
import type { MatterMode, ProfferBatchItem, ProfferBatchItemStatus } from "@/lib/shared/types";

import { attemptReviewHref, sourceFilename } from "./proffer-operations-state";

const BATCH_ID_PATTERN = /^[A-Za-z0-9_-]{32,128}$/;

function batchStatusLabel(status: ProfferBatchItemStatus) {
  if (status === "waiting_on_gate") return "Needs review";
  if (status === "done") return "Completed";
  if (status === "failed") return "Stopped";
  if (status === "skipped") return "Skipped";
  if (status === "queued") return "Queued";
  return "In progress";
}

// Byline: Codex · GPT-6 · 2026-10-06
/** Render one server-reported batch item and its exact attempt link when available.
 * Inputs: batch item and the mode that scopes its operation.
 * Output: filename, lifecycle label, reason, and optional Read link.
 * Side effects: none.
 * Use inside BatchActivityPanel; batch items without a valid handle remain informational.
 */
function BatchItemRow({ item, mode }: { item: ProfferBatchItem; mode: MatterMode }) {
  const title = sourceFilename(item.source_ref || item.key);
  const canOpen = /^[A-Za-z0-9_-]{32,128}$/.test(item.preview_handle);

  return (
    <li className="grid gap-2 border-t px-4 py-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
      <div className="min-w-0">
        <p className="truncate text-sm font-medium" title={item.source_ref || item.key}>{title}</p>
        <p className="mt-1 text-xs text-muted-foreground">
          {batchStatusLabel(item.status)}{item.reason ? ` · ${item.reason}` : ""}
        </p>
      </div>
      {canOpen && (
        <AppLink className="inline-flex min-h-9 items-center gap-1.5 text-xs font-semibold text-primary hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2" href={attemptReviewHref(item.preview_handle, mode)}>
          {item.status === "failed" ? "Open for a new import" : item.status === "waiting_on_gate" ? "Review this import" : "Open attempt"}
          <ArrowUpRight className="size-3.5" aria-hidden="true" />
        </AppLink>
      )}
    </li>
  );
}

// Byline: Codex · GPT-6 · 2026-10-06
/** Show persistent server-owned status for a batch named by the Activity URL.
 * Inputs: batch ID, fixed-case mode, and close callback.
 * Output: durable batch counts and item rows, or loading/error state.
 * Side effects: reads the mode-scoped batch endpoint and polls while it is active.
 * Use for /activity?batch= links from Sources; do not derive progress from local UI state.
 */
export function BatchActivityPanel({ batchId, mode, onClose }: { batchId: string; mode: MatterMode; onClose: () => void }) {
  const validBatchId = BATCH_ID_PATTERN.test(batchId);
  const batch = useQuery({
    queryKey: ["activity", "proffer-batch", mode, batchId],
    queryFn: ({ signal }) => getProfferBatch(batchId, mode, signal),
    enabled: validBatchId,
    retry: false,
    refetchInterval: (query) => query.state.data?.terminal ? false : 5_000,
  });

  return (
    <section className="border-y bg-card" aria-labelledby="activity-batch-heading" aria-live="polite">
      <header className="flex flex-wrap items-start justify-between gap-3 px-4 py-4 sm:px-5">
        <div>
          <h2 id="activity-batch-heading" className="text-base font-semibold">Folder import</h2>
          <p className="mt-1 text-xs text-muted-foreground">Current status is read from the durable batch record.</p>
        </div>
        <button type="button" onClick={onClose} className="min-h-9 px-3 text-xs font-medium text-muted-foreground hover:text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2" aria-label="Close batch status">Close</button>
      </header>

      {!validBatchId ? (
        <div className="flex items-center gap-2 border-t px-4 py-4 text-sm text-destructive" role="alert"><AlertCircle className="size-4" /> This batch link is invalid.</div>
      ) : batch.isPending ? (
        <div className="flex items-center gap-2 border-t px-4 py-5 text-sm text-muted-foreground" role="status"><Loader2 className="size-4 animate-spin motion-reduce:animate-none" /> Loading batch status…</div>
      ) : batch.error ? (
        <div className="flex items-start gap-2 border-t px-4 py-4 text-sm text-destructive" role="alert"><AlertCircle className="mt-0.5 size-4 shrink-0" /><span>{batch.error.message}</span></div>
      ) : batch.data ? (
        <>
          <dl className="grid grid-cols-2 gap-px border-t bg-border sm:grid-cols-4">
            {[
              ["Queued", batch.data.counts.queued],
              ["In progress", batch.data.counts.running],
              ["Needs review", batch.data.counts.waiting_on_gate],
              ["Completed", batch.data.counts.done],
              ["Stopped", batch.data.counts.failed],
              ["Skipped", batch.data.counts.skipped],
            ].map(([label, value]) => (
              <div key={label} className="bg-card px-4 py-3">
                <dt className="text-xs text-muted-foreground">{label}</dt>
                <dd className="mt-1 text-lg font-semibold tabular-nums">{value}</dd>
              </div>
            ))}
            <div className="col-span-2 bg-card px-4 py-3 sm:col-span-2">
              <dt className="text-xs text-muted-foreground">Batch</dt>
              <dd className="mt-1 text-sm font-medium">{batch.data.terminal ? "Finished" : "Active"}</dd>
            </div>
          </dl>
          {(batch.data.listing_truncated || batch.data.items_truncated) && (
            <p className="border-t px-4 py-2 text-xs text-muted-foreground">The batch response is truncated. The counts remain server-reported.</p>
          )}
          <ul aria-label="Files in this folder import">
            {batch.data.items.map((item) => <BatchItemRow key={item.key} item={item} mode={mode} />)}
            {batch.data.items.length === 0 && <li className="border-t px-4 py-5 text-sm text-muted-foreground">No file status rows are available yet.</li>}
          </ul>
          <details className="border-t px-4 py-3 text-xs text-muted-foreground">
            <summary className="cursor-pointer">Batch reference</summary>
            <code className="mt-2 block break-all">{batch.data.batch_id}</code>
          </details>
        </>
      ) : null}
    </section>
  );
}
