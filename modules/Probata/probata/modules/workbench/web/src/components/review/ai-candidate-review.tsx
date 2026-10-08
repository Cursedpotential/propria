// Byline: Codex · GPT-6 · 2026-10-08.
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useRef, useState } from "react";

import { ErrorBox, LoadMore, Loading } from "@/components/mobile/mobile-ui";
import { Button } from "@/components/ui/button";
import { getAIContextSourceStatus } from "@/lib/api-client";
import { decideAICandidate, getAICandidates, type AICandidateRow, type AIDecision, type AIDecisionReceipt } from "@/lib/ai-candidates-client";
import { getAnalysisProjections } from "@/lib/analysis-client";

const choices: { value: AIDecision; label: string }[] = [
  { value: "approved", label: "Approve" },
  { value: "rejected", label: "Reject" },
  { value: "needs_info", label: "Needs information" },
];

/** Show native AI proposals and record only the owner's explicit item decisions.
 * Input: original context workflow ID from Sources. Output: cited proposals and
 * actual decision receipts. Effects: bounded status/list reads and clicked writes.
 * Pick on Read for AI review; never treat it as normalized-message review.
 */
export function AICandidateReview({ workflowId }: { workflowId: string }) {
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [receipts, setReceipts] = useState<Record<string, AIDecisionReceipt>>({});
  const keys = useRef(new Map<string, string>());
  const status = useQuery({
    queryKey: ["ai-context-review-status", workflowId],
    queryFn: () => getAIContextSourceStatus(workflowId),
    retry: false,
    refetchInterval: (query) => query.state.data && !query.state.data.review_source
      && !["complete", "partial", "needs_parser", "failed"].includes(String(query.state.data.status)) ? 3000 : false,
  });
  const pages = useInfiniteQuery({
    queryKey: ["ai-context-candidates", workflowId],
    queryFn: ({ pageParam, signal }) => getAICandidates(workflowId, pageParam, signal),
    initialPageParam: "",
    getNextPageParam: (page) => page.next_cursor || undefined,
    enabled: Boolean(status.data?.review_source),
    retry: false,
  });
  const approved = Object.values(receipts).filter((receipt) => receipt.decision === "approved");
  const projections = useQuery({
    queryKey: ["ai-review-publication", workflowId, ...approved.map((receipt) => receipt.decision_id)],
    queryFn: ({ signal }) => getAnalysisProjections(undefined, signal),
    enabled: approved.length > 0,
    retry: false,
    refetchInterval: 5000,
  });

  /** Reuse a click key after uncertain delivery; never approve on load or refresh.
   * Inputs: displayed proposal and chosen state. Output: saved receipt. Effects:
   * one decision request followed by readback; originals remain unchanged.
   */
  async function decide(row: AICandidateRow, choice: AIDecision) {
    const identity = `${row.candidate_id}:${row.content_sha256}:${choice}`;
    if (!keys.current.has(identity)) keys.current.set(identity, crypto.randomUUID());
    setBusy(row.candidate_id);
    setError(null);
    try {
      const receipt = await decideAICandidate(workflowId, row, choice, keys.current.get(identity)!);
      setReceipts((prior) => ({ ...prior, [row.candidate_id]: receipt }));
      for (const saved of keys.current.keys()) {
        if (saved.startsWith(`${row.candidate_id}:`) && (saved !== identity || receipt.projection?.status !== "pending")) keys.current.delete(saved);
      }
      await pages.refetch();
    } catch (caught) {
      setError(caught instanceof Error ? caught : new Error("The decision could not be confirmed."));
    } finally {
      setBusy(null);
    }
  }

  const rows = pages.data?.pages.flatMap((page) => page.candidates) ?? [];
  return <section className="space-y-4 rounded-lg border border-border bg-card p-4" aria-label="AI extraction review">
    <header><h2 className="text-lg font-semibold">Review extracted context</h2>
      <p className="mt-1 text-sm text-muted-foreground">Check each proposal against its source quote, then choose what to keep.</p></header>
    {status.isPending ? <Loading /> : null}
    {status.isError ? <ErrorBox error={status.error} onRetry={() => void status.refetch()} /> : null}
    {status.data && !status.data.review_source ? <p role="status" className="text-sm">
      Extraction status: {String(status.data.status)}. {status.data.reason ? String(status.data.reason) : "Staged review results are not available yet."}
      <Button className="ml-2" variant="outline" size="sm" onClick={() => void status.refetch()}>Refresh</Button>
    </p> : null}
    {status.data?.review_source && pages.isPending ? <Loading /> : null}
    {pages.isError ? <ErrorBox error={pages.error} onRetry={() => void pages.refetch()} /> : null}
    {error ? <ErrorBox error={error} /> : null}
    {pages.data && !rows.length ? <p className="text-sm">This source has no staged proposals to review.</p> : null}
    {rows.map((row) => {
      const receipt = receipts[row.candidate_id];
      const published = receipt?.decision === "approved" && projections.data?.items.some((snapshot) =>
        snapshot.approved_revision_id === receipt.decision_id && snapshot.approval_digest === receipt.request_digest
        && snapshot.source_pins?.some((pin) => pin.source_version_id === row.candidate.source_version_id
          && pin.source_object_sha256 === row.candidate.source_sha256 && pin.source_object_uri === row.candidate.source_ref));
      return <article key={row.candidate_id} className="space-y-3 rounded-md border border-border p-3">
      <div className="flex flex-wrap justify-between gap-2"><h3 className="text-base font-medium">{row.candidate.name || row.candidate.statement || row.reported_kind}</h3>
        <span className="text-xs text-muted-foreground">{row.reported_kind} · {receipts[row.candidate_id]?.decision ?? row.review_state}</span></div>
      {row.candidate.evidence_quote ? <blockquote className="whitespace-pre-wrap border-l-2 border-border pl-3 text-sm">{row.candidate.evidence_quote}</blockquote> : null}
      <p className="text-xs text-muted-foreground">{row.candidate.occurred_at || "Precise timestamp unknown"}{row.candidate.predicate ? ` · ${row.candidate.predicate}` : ""}</p>
      <details className="text-xs"><summary className="cursor-pointer">Original source and locator</summary>
        <dl className="mt-2 grid gap-1 break-all"><dt>Original</dt><dd>{row.candidate.source_ref}</dd>
          <dt>Source version</dt><dd>{row.candidate.source_version_id} · {row.candidate.version_id || "No provider version recorded"}</dd>
          <dt>Original SHA-256</dt><dd>{row.candidate.source_sha256}</dd>
          {row.candidate.native_json_pointer ? <><dt>Native JSON pointer</dt><dd>{row.candidate.native_json_pointer}</dd></> : null}
          <dt>Native span</dt><dd>{row.candidate.source_span.start}–{row.candidate.source_span.end} {row.candidate.span_unit} · {row.candidate.source_span.sha256}</dd>
        </dl></details>
      <div className="flex flex-wrap gap-2">{choices.map((choice) => <Button key={choice.value} size="sm" variant={choice.value === "approved" ? "default" : "outline"}
        disabled={busy !== null || (receipts[row.candidate_id]?.decision ?? row.review_state) === choice.value}
        onClick={() => void decide(row, choice.value)}>{busy === row.candidate_id ? "Saving…" : choice.label}</Button>)}</div>
      {(receipts[row.candidate_id] || row.decision_id) ? <p role="status" className="break-all text-xs">Decision recorded: {receipts[row.candidate_id]?.decision_id ?? row.decision_id}.</p> : null}
      {receipts[row.candidate_id]?.decision === "approved" ? <div role="status" className="text-xs text-muted-foreground">
        Graph publication: {published ? "completed and verified" : receipt?.projection?.status === "enqueued" ? "queued" : receipt?.projection?.status || "not confirmed"}.
        {projections.error && !published ? " Publication status could not be read; your decision remains recorded." : ""}
        {receipts[row.candidate_id].projection?.error ? ` ${receipts[row.candidate_id].projection?.error}` : ""}
        {!published && receipt?.projection?.status === "pending" && receipt.projection.retryable ?
          <Button className="ml-2" variant="outline" size="sm" disabled={busy !== null} onClick={() => void decide(row, "approved")}>Retry graph publication</Button> : null}
      </div> : row.review_state === "approved" ? <p className="text-xs text-muted-foreground">Approval is recorded. Graph publication has not been confirmed by this review readback.</p> : null}
    </article>; })}
    {pages.hasNextPage ? <LoadMore onClick={() => void pages.fetchNextPage()} loading={pages.isFetchingNextPage} label="Load more proposals" /> : null}
  </section>;
}
