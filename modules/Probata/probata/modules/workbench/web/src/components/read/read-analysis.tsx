// Byline: Codex · GPT-6 · 2026-10-08.
import { useEffect, useState } from "react";
import { useInfiniteQuery, useMutation, useQuery } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { ErrorBox, Loading } from "@/components/mobile/mobile-ui";
import { Button } from "@/components/ui/button";
import { readHref } from "@/components/read/read-location";
import { analysisQuery, getAnalysisContent, getAnalysisProjections, getAnalysisStatus, startAnalysisQuery, type AnalysisContent, type AnalysisProjection, type AnalysisQuery } from "@/lib/analysis-client";

/** Show query-time analysis beside Read using actual completed graph snapshots.
 * Input: existing Read route context. Output: perspective/cutoff controls and cited findings.
 * Effects: catalog/status/content reads and explicit Temporal query starts only.
 * Pick for analytical reading, never source ingestion, approval or evidence promotion.
 */
export function ReadAnalysis({ params }: { params: URLSearchParams }) {
  const router = useRouter();
  const workflowId = params.get("analysis");
  const [selected, setSelected] = useState<string | null>(null);
  const [perspective, setPerspective] = useState<AnalysisQuery["perspective"]>("as_lived");
  const [cutoff, setCutoff] = useState("");
  const [message, setMessage] = useState("");
  const [loadedWorkflow, setLoadedWorkflow] = useState<string | null>(null);
  const catalog = useInfiniteQuery({
    queryKey: ["analysis-projections"], initialPageParam: "",
    queryFn: ({ pageParam, signal }) => getAnalysisProjections(pageParam || undefined, signal),
    getNextPageParam: (last) => last.next_cursor || undefined, retry: false,
  });
  const status = useQuery({
    queryKey: ["analysis-status", workflowId],
    queryFn: ({ signal }) => getAnalysisStatus(workflowId!, signal), enabled: Boolean(workflowId), retry: false,
    refetchInterval: (query) => query.state.data?.outcome === "running" ? 3000 : false,
  });
  const content = useQuery({
    queryKey: ["analysis-content", workflowId],
    queryFn: ({ signal }) => getAnalysisContent(workflowId!, signal),
    enabled: Boolean(workflowId && status.data?.outcome === "completed"), retry: false,
  });
  useEffect(() => {
    if (!content.data || !workflowId || loadedWorkflow === workflowId) return;
    setSelected(content.data.projection_generation_id);
    setPerspective(content.data.perspective);
    if (content.data.horizon) {
      const date = new Date(content.data.horizon);
      setCutoff(new Date(date.getTime() - date.getTimezoneOffset() * 60_000).toISOString().slice(0, 19));
    }
    setLoadedWorkflow(workflowId);
  }, [content.data, workflowId, loadedWorkflow]);
  const snapshots = catalog.data?.pages.flatMap((page) => page.items) ?? [];
  const result = status.data?.result;
  const choices: AnalysisProjection[] = result && !snapshots.some((item) => item.projection_generation_id === result.projection_generation_id) ? [result, ...snapshots] : snapshots;
  const projection = choices.find((item) => item.projection_generation_id === selected) ?? null;
  const start = useMutation({
    mutationFn: ({ query, key }: { query: AnalysisQuery; key: string }) => startAnalysisQuery(query, key),
    onSuccess: (started) => { setMessage(""); void router.navigate({ href: readHref(params, { analysis: started.workflow_id }) }); },
  });
  const submit = (cursor?: string, fromContent?: AnalysisContent) => {
    try {
      const pin = fromContent ?? projection;
      if (!pin) throw new Error("Choose a completed analysis snapshot.");
      const query = fromContent ? { ...analysisQuery(pin, fromContent.perspective, fromContent.horizon ?? "", cursor), limit: fromContent.limit } : analysisQuery(pin, perspective, cutoff, cursor);
      setMessage(""); start.mutate({ query, key: crypto.randomUUID() });
    } catch (error) { setMessage(error instanceof Error ? error.message : "Analysis could not start"); }
  };
  return <section className="space-y-3 rounded-lg border border-border bg-card p-4" aria-label="Analysis reading">
    <h2 className="text-sm font-semibold">Read analysis</h2>
    <form className="flex flex-wrap items-end gap-3" onSubmit={(event) => { event.preventDefault(); submit(); }}>
      <label className="min-w-48 flex-1 text-xs">Completed snapshot
        <select className="mt-1 block w-full rounded border bg-background p-2 text-sm" value={selected ?? ""} onChange={(event) => setSelected(event.target.value)}>
          <option value="">Choose a recorded snapshot</option>
          {choices.map((item) => <option key={item.projection_generation_id} value={item.projection_generation_id}>{item.source_pins?.[0]?.source_object_uri.split("/").pop() || "Approved context"} · {item.claim_count} findings{item.completed_at ? ` · ${new Date(item.completed_at).toLocaleString()}` : ""}</option>)}
        </select>
      </label>
      <label className="text-xs">Perspective
        <select className="mt-1 block rounded border bg-background p-2 text-sm" value={perspective} onChange={(event) => setPerspective(event.target.value as AnalysisQuery["perspective"])}>
          <option value="as_lived">As lived</option><option value="hindsight">Hindsight</option>
        </select>
      </label>
      {perspective === "as_lived" ? <label className="text-xs">Known by ({Intl.DateTimeFormat().resolvedOptions().timeZone})
        <input className="mt-1 block rounded border bg-background p-2 text-sm" type="datetime-local" step="1" required value={cutoff} onChange={(event) => setCutoff(event.target.value)} />
      </label> : null}
      <Button type="submit" disabled={!projection || start.isPending}>{start.isPending ? "Starting…" : "Read findings"}</Button>
    </form>
    {catalog.isPending ? <Loading /> : null}
    {catalog.isError ? <ErrorBox error={catalog.error} onRetry={() => void catalog.refetch()} /> : null}
    {catalog.isSuccess && !choices.length ? <p className="text-sm">No completed analysis snapshots are recorded yet.</p> : null}
    {catalog.hasNextPage ? <Button variant="outline" size="sm" disabled={catalog.isFetchingNextPage} onClick={() => void catalog.fetchNextPage()}>More snapshots</Button> : null}
    {message ? <p role="alert" className="text-sm text-destructive">{message}</p> : null}
    {start.isError ? <ErrorBox error={start.error} onRetry={() => start.variables && start.mutate(start.variables)} /> : null}
    {workflowId && status.isPending ? <Loading /> : null}
    {status.isError ? <ErrorBox error={status.error} onRetry={() => void status.refetch()} /> : null}
    {status.data?.outcome === "running" ? <p role="status" className="text-sm">Reading the selected graph snapshot…</p> : null}
    {status.data && !["running", "completed"].includes(status.data.outcome) ? <p role="alert" className="text-sm text-destructive">{status.data.error || `Query ${status.data.outcome}`}</p> : null}
    {content.isFetching ? <Loading /> : null}
    {content.isError ? <ErrorBox error={content.error} onRetry={() => void content.refetch()} /> : null}
    {content.data ? <div className="space-y-3" aria-live="polite">
      <p className="text-sm font-medium">{content.data.perspective === "as_lived" ? `As lived · known by ${new Date(content.data.horizon!).toLocaleString()}` : "Hindsight"} · {content.data.claims.length} findings on this page</p>
      {!content.data.claims.length ? <p className="text-sm">No findings match this snapshot and perspective.</p> : null}
      <ul className="max-h-[65vh] space-y-3 overflow-auto">{content.data.claims.map((claim) => <li key={claim.id} className="space-y-2 rounded border p-3">
        <p className="whitespace-pre-wrap break-words text-sm leading-relaxed">{claim.text}</p>
        <p className="text-xs text-muted-foreground">{claim.kind.replaceAll("_", " ")}{claim.predicate ? ` · ${claim.predicate}` : ""}{claim.occurred_at ? ` · occurred ${new Date(claim.occurred_at).toLocaleString()}` : ""} · source available {claim.source_available_from ? new Date(claim.source_available_from).toLocaleString() : "precise timestamp unknown"}</p>
        <details className="text-xs"><summary className="cursor-pointer underline underline-offset-4">Source citation</summary>
          <dl className="mt-2 space-y-1 break-all">
            <div><dt>Original source</dt><dd>{claim.source_object_uri}</dd></div>
            <div><dt>Source version</dt><dd>{claim.source_version_id}</dd></div>
            <div><dt>Original SHA-256</dt><dd>{claim.source_sha256}</dd></div>
            {claim.native_json_pointer ? <div><dt>Original JSON field</dt><dd>{claim.native_json_pointer}</dd></div> : null}
            {claim.native_span_start != null && claim.native_span_end != null ? <div><dt>Native source span ({claim.native_span_unit})</dt><dd>{claim.native_span_start}–{claim.native_span_end}{!claim.native_json_pointer ? " in the original document" : " in the original JSON field"}</dd></div> : null}
            {claim.native_span_sha256 ? <div><dt>Native span SHA-256</dt><dd>{claim.native_span_sha256}</dd></div> : null}
            <div><dt>Source record</dt><dd>{claim.record_id}</dd></div>
            <div><dt>Record SHA-256</dt><dd>{claim.record_sha256}</dd></div>
            <div><dt>Approved candidate</dt><dd>{claim.candidate_id} · {claim.candidate_sha256}</dd></div>
          </dl>
        </details>
      </li>)}</ul>
      {content.data.has_more ? <Button type="button" variant="outline" disabled={start.isPending} onClick={() => submit(content.data!.next_cursor, content.data)}>Next findings</Button> : null}
    </div> : null}
    {status.data ? <details className="text-xs text-muted-foreground"><summary className="cursor-pointer">Pinned query details</summary>
      <dl className="mt-2 space-y-1 break-all"><div><dt>Workflow</dt><dd>{workflowId}</dd></div>
        {result ? <><div><dt>Approval revision and digest</dt><dd>{result.approved_revision_id} · {result.approval_digest}</dd></div>
          <div><dt>Graph generation and hash</dt><dd>{result.projection_generation_id} · {result.projection_hash}</dd></div>
          <div><dt>Result version and SHA-256</dt><dd>{result.artifact.version_id} · {result.artifact.sha256}</dd></div></> : null}
      </dl>
    </details> : null}
  </section>;
}
