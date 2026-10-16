// Byline: Codex · GPT-6 · 2026-10-11.
import { useRef, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { ErrorBox, Loading } from "@/components/mobile/mobile-ui";
import { Button } from "@/components/ui/button";
import { readHref } from "@/components/read/read-location";
import type { AnalysisProjection } from "@/lib/analysis-client";
import { getWalkContent, getWalkStatus, getWalkStep, startWalk, walkRequest, type WalkRequest } from "@/lib/analysis-walk-client";

/** Compare successive knowledge dates and separate hindsight using the actual walk workflow.
 * Inputs: selected recorded snapshot and Read route. Output: cited one-step view and comparison.
 * Effects: explicit walk start, bounded status/result reads. Never source import or evidence promotion.
 */
export function ReadAnalysisWalk({ projection, params }: { projection: AnalysisProjection | null; params: URLSearchParams }) {
  const router = useRouter();
  const id = params.get("walk");
  const [dates, setDates] = useState([""]);
  const [index, setIndex] = useState(0);
  const [message, setMessage] = useState("");
  const click = useRef<{ payload: string; key: string } | null>(null);
  const start = useMutation({
    mutationFn: ({ body, key }: { body: WalkRequest; key: string }) => startWalk(body, key),
    onSuccess: (value) => { click.current = null; setIndex(0); void router.navigate({ href: readHref(params, { walk: value.workflow_id }) }); },
  });
  const status = useQuery({ queryKey: ["analysis-walk-status", id], enabled: Boolean(id), retry: false,
    queryFn: ({ signal }) => getWalkStatus(id!, signal),
    refetchInterval: (query) => query.state.data?.outcome === "running" ? 3000 : false });
  const content = useQuery({ queryKey: ["analysis-walk-content", id], retry: false,
    enabled: Boolean(id && status.data?.outcome === "completed"), queryFn: ({ signal }) => getWalkContent(id!, signal) });
  const selected = useQuery({ queryKey: ["analysis-walk-step", id, index], retry: false, gcTime: 0,
    enabled: Boolean(id && content.data && index < content.data.checkpoints.length),
    queryFn: ({ signal }) => getWalkStep(id!, index, signal) });
  const checkpoint = selected.data?.checkpoint;
  return <section className="space-y-3 border-t pt-3" aria-label="Analysis across knowledge dates">
    <h3 className="text-sm font-semibold">Compare knowledge over time</h3>
    <form className="space-y-2" onSubmit={(event) => {
      event.preventDefault();
      try {
        if (!projection) throw new Error("Choose a completed snapshot above.");
        const body = walkRequest(projection, dates);
        const payload = JSON.stringify(body);
        if (click.current?.payload !== payload) click.current = { payload, key: crypto.randomUUID() };
        setMessage(""); start.mutate({ body, key: click.current.key });
      } catch (error) { setMessage(error instanceof Error ? error.message : "Walk could not start"); }
    }}>
      <p className="text-xs text-muted-foreground">Choose dates in order ({Intl.DateTimeFormat().resolvedOptions().timeZone}). Hindsight is compared separately after the last date.</p>
      <div className="flex flex-wrap gap-2">{dates.map((date, position) => <label key={position} className="text-xs">Known by {position + 1}
        <input className="mt-1 block rounded border bg-background p-2 text-sm" type="datetime-local" step="1" required value={date}
          onChange={(event) => setDates((old) => old.map((value, i) => i === position ? event.target.value : value))} />
      </label>)}</div>
      <div className="flex flex-wrap gap-2">
        <Button type="button" variant="outline" size="sm" disabled={dates.length >= 20} onClick={() => setDates((old) => [...old, ""])}>Add date</Button>
        {dates.length > 1 ? <Button type="button" variant="outline" size="sm" onClick={() => setDates((old) => old.slice(0, -1))}>Remove last date</Button> : null}
        <Button type="submit" disabled={!projection || start.isPending}>{start.isPending ? "Starting…" : "Compare dates"}</Button>
      </div>
    </form>
    {message ? <p role="alert" className="text-sm text-destructive">{message}</p> : null}
    {start.isError ? <ErrorBox error={start.error} onRetry={() => start.variables && start.mutate(start.variables)} /> : null}
    {status.isFetching && !status.data ? <Loading /> : null}
    {status.isError ? <ErrorBox error={status.error} onRetry={() => void status.refetch()} /> : null}
    {status.data?.outcome === "running" ? <p role="status" className="text-sm">Comparing knowledge dates · step {status.data.step + 1}…</p> : null}
    {status.data && status.data.outcome !== "running" && status.data.outcome !== "completed" ? <p role="alert" className="text-sm text-destructive">{status.data.error || `Walk ${status.data.outcome}`}</p> : null}
    {content.isFetching ? <Loading /> : null}
    {content.isError ? <ErrorBox error={content.error} onRetry={() => void content.refetch()} /> : null}
    {content.data ? <div className="space-y-3">
      <p className="text-sm">{content.data.partial ? "Partial retrieval · " : ""}{content.data.coverage}</p>
      <label className="text-xs">Read a step
        <select className="mt-1 block rounded border bg-background p-2 text-sm" value={index} onChange={(event) => setIndex(Number(event.target.value))}>
          {content.data.cutoffs.map((date, i) => <option key={i} value={i}>As lived · {new Date(date).toLocaleString()}</option>)}
          <option value={content.data.cutoffs.length}>Hindsight · separate perspective</option>
        </select>
      </label>
      <p className="text-xs">Hindsight includes {content.data.comparison.hindsight_only_claim_ids.length} additional claims; {content.data.comparison.unknown_availability_claim_ids.length} have unknown availability dates.</p>
      <details className="text-sm"><summary className="cursor-pointer">What differs in hindsight</summary>
        {content.data.comparison.hindsight_findings_not_identical_to_last_as_lived.length === 0 ? <p className="mt-2">No changed findings were recorded.</p> :
          <ul className="mt-2 space-y-2">{content.data.comparison.hindsight_findings_not_identical_to_last_as_lived.map((finding, i) => <li key={i} className="rounded border p-2">
            <p className="text-xs text-muted-foreground">{finding.kind.replaceAll("_", " ")}</p>
            <p className="whitespace-pre-wrap">{finding.text}</p>
            {finding.citations.map((cite, j) => <details key={j} className="mt-1 break-all text-xs"><summary className="cursor-pointer">Source citation {j + 1}</summary>
              <blockquote className="whitespace-pre-wrap">{cite.quote}</blockquote>
              <p>{cite.source_pin.source_object_uri}</p><p>{cite.source_pin.source_version_id} · {cite.source_pin.source_sha256}</p>
              {cite.source_pin.native_span_start !== null && cite.source_pin.native_span_start !== undefined ? <p>{cite.source_pin.native_json_pointer || "Original document"} · {cite.source_pin.native_span_start}–{cite.source_pin.native_span_end} · {cite.source_pin.native_span_sha256}</p> : null}
            </details>)}
          </li>)}</ul>}
      </details>
      {selected.isFetching ? <Loading /> : null}
      {selected.isError ? <ErrorBox error={selected.error} onRetry={() => void selected.refetch()} /> : null}
      {checkpoint ? <div className="space-y-3" aria-live="polite">
        <h4 className="text-sm font-medium">{checkpoint.perspective === "hindsight" ? "Hindsight" : `As lived · ${new Date(checkpoint.horizon!).toLocaleString()}`}</h4>
        {!checkpoint.findings.length ? <p className="text-sm">No supported findings were returned for this step.</p> : null}
        <ul className="max-h-[65vh] space-y-3 overflow-auto">{checkpoint.findings.map((finding, i) => <li key={i} className="space-y-2 rounded border p-3">
          <p className="text-xs text-muted-foreground">{finding.kind.replaceAll("_", " ")}</p>
          <p className="whitespace-pre-wrap break-words text-sm">{finding.text}</p>
          {finding.citations.map((cite, j) => <details key={j} className="text-xs">
            <summary className="cursor-pointer underline underline-offset-4">Source citation {j + 1}</summary>
            <blockquote className="my-2 whitespace-pre-wrap border-l-2 pl-2">{cite.quote}</blockquote>
            <dl className="space-y-1 break-all">
              <div><dt>Original source</dt><dd>{cite.source_pin.source_object_uri}</dd></div>
              <div><dt>Source version</dt><dd>{cite.source_pin.source_version_id}</dd></div>
              <div><dt>Original SHA-256</dt><dd>{cite.source_pin.source_sha256}</dd></div>
              <div><dt>Known from</dt><dd>{cite.source_pin.source_available_from ? new Date(cite.source_pin.source_available_from).toLocaleString() : "Unknown"}</dd></div>
              {cite.source_pin.native_json_pointer ? <div><dt>Source field</dt><dd>{cite.source_pin.native_json_pointer}</dd></div> : null}
              {cite.source_pin.native_span_start !== null && cite.source_pin.native_span_start !== undefined ? <div><dt>Native span ({cite.source_pin.native_span_unit})</dt><dd>{cite.source_pin.native_span_start}–{cite.source_pin.native_span_end} · {cite.source_pin.native_span_sha256}</dd></div> : null}
              <div><dt>Record and hash</dt><dd>{cite.source_pin.record_id} · {cite.source_pin.record_sha256}</dd></div>
              <div><dt>Candidate and hash</dt><dd>{cite.source_pin.candidate_id} · {cite.source_pin.candidate_sha256}</dd></div>
            </dl>
          </details>)}
        </li>)}</ul>
        <details className="text-xs"><summary className="cursor-pointer">Coverage and validation details</summary>
          <p className="mt-2">{content.data.validation}</p><p>{content.data.family_independence}</p>
          <ul>{checkpoint.limitations.map((value, i) => <li key={i}>{value}</li>)}</ul>
          <p>{checkpoint.prompt_version} · {checkpoint.prompt_sha256}</p>
          <p>Checkpoint version {selected.data?.artifact.version_id} · {selected.data?.artifact.sha256}</p>
        </details>
      </div> : null}
    </div> : null}
  </section>;
}
