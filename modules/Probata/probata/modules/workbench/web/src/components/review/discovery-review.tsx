// Byline: Codex · GPT-6 · 2026-10-11.
import { useInfiniteQuery, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { useEffect, useMemo, useRef, useState } from "react";

import { ErrorBox, Loading, LoadMore } from "@/components/mobile/mobile-ui";
import { readHref } from "@/components/read/read-location";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api-client";
import { AppLink, useBrowserSearchParams } from "@/lib/router-compat";
import { discoveryCandidates, discoveryContext, discoveryDatasets, discoveryRun, discoveryScopeKey, saveDiscoveryFeedback, startDiscovery,
  type DiscoveryBackend, type DiscoveryCandidate, type DiscoveryFeedback, type DiscoveryFeedbackKind, type DiscoveryReceipt, type DiscoveryRecord, type DiscoveryRequest, type DiscoveryScope } from "@/lib/discovery-client";

const choices: { value: DiscoveryFeedbackKind; label: string }[] = [
  { value: "relevant", label: "Relevant" }, { value: "irrelevant", label: "Irrelevant" },
  { value: "wrong_person", label: "Wrong person" }, { value: "wrong_behavior", label: "Wrong behavior" },
  { value: "reported_speech", label: "Reported speech" }, { value: "missing_context", label: "Missing context" },
  { value: "partial_match", label: "Partial match" }, { value: "skip", label: "Skip" }, { value: "free_text", label: "Other correction" },
];
const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const field = "mt-1 block w-full rounded border bg-background p-2 text-sm";

/** Link only genuine Imported identities to the existing reader; never turn source URLs into links.
 * Inputs: recorded source and operating mode. Output: internal reader href or null.
 * Effects: none; preserve B2/file locators as displayed citations instead of browser URLs.
 */
export function discoveryRecordHref(record: DiscoveryRecord, scope: DiscoveryScope): string | null {
  if (!record.imported_thread_id || !/^[A-Za-z0-9_-]+$/.test(record.imported_thread_id) || !record.imported_message_id || !uuid.test(record.imported_message_id)) return null;
  return readHref(new URLSearchParams({ mode: scope.mode }), { source: record.imported_source_id ?? null, thread: record.imported_thread_id, around: record.imported_message_id });
}

/** Display chronologically ordered source records and their recorded locator proof.
 * Inputs: bounded records/scope. Output: source-linked context. Effects: explicit internal links only.
 * Pick for verifying proposals without interpreting a speaker as a conduct actor.
 */
function SourceContext({ records, scope }: { records: DiscoveryRecord[]; scope: DiscoveryScope }) {
  const ordered = [...records].sort((a, b) => (a.occurred_at ?? "9999").localeCompare(b.occurred_at ?? "9999") || a.ordinal - b.ordinal);
  return <ol className="space-y-3">{ordered.map((record) => <li key={record.record_id} className="rounded border p-3">
    <p className="text-xs text-muted-foreground">{record.occurred_at ? new Date(record.occurred_at).toLocaleString() : "Time not recorded"} · {record.platform || record.source_kind}
      {record.participants?.length ? ` · ${record.participants.map((person) => `${person.display_name || person.identifier} (${person.role})`).join(", ")}` : ""}</p>
    <p className="my-2 whitespace-pre-wrap break-words text-sm">{record.text}</p>
    {record.context_uncertainty?.map((note, index) => <p key={index} className="text-xs text-muted-foreground">{note}</p>)}
    {discoveryRecordHref(record, scope) ? <AppLink href={discoveryRecordHref(record, scope)!} className="text-xs underline">Open original message</AppLink> : null}
    <details className="break-all text-xs"><summary className="cursor-pointer">Source citation</summary>
      <dl className="mt-2 grid gap-1"><dt>Source version</dt><dd>{record.source_version_id}</dd>
        {record.source_ref ? <><dt>Original source</dt><dd>{record.source_ref}</dd></> : null}
        {record.source_sha256 ? <><dt>Original SHA-256</dt><dd>{record.source_sha256}</dd></> : null}
        {record.source_provider_version ? <><dt>Provider version</dt><dd>{record.source_provider_version}</dd></> : null}
        {record.source_object_id ? <><dt>Original object</dt><dd>{record.source_object_id}</dd></> : null}
        <dt>Record / generation</dt><dd>{record.record_id} / {record.generation_id}</dd>
        {record.record_revision ? <><dt>Record revision</dt><dd>{record.record_revision}</dd></> : null}
        {record.input_revision ? <><dt>Input revision</dt><dd>{record.input_revision}</dd></> : null}
        <dt>Source available</dt><dd>{record.source_available_from ?? "Not recorded"}</dd>
        {record.native_source_available_from ? <><dt>Native source available</dt><dd>{record.native_source_available_from}</dd></> : null}
        {record.group_knowledge_available_from ? <><dt>Group context available</dt><dd>{record.group_knowledge_available_from}</dd></> : null}
        {record.group_revision ? <><dt>Group revision</dt><dd>{record.group_revision}</dd></> : null}</dl>
      {record.locators?.map((locator) => <div key={locator.locator_id} className="mt-2 rounded border p-2">
        <p>Locator {locator.locator_id} · [{locator.range_start}, {locator.range_end}) {locator.coordinate_system}</p>
        {locator.exact_slice_sha256 ? <p>Exact slice SHA-256: {locator.exact_slice_sha256}</p> : null}
        {locator.verification_activity_receipt_id ? <p>Verification receipt: {locator.verification_activity_receipt_id}</p> : null}
        <pre className="whitespace-pre-wrap">{JSON.stringify(locator.projection, null, 2)}</pre>
      </div>)}</details>
  </li>)}</ol>;
}

/** Review one proposed episode and save feedback only after the backend acknowledgement.
 * Inputs: candidate/current run revision and admitted scope. Output: citations and saved receipt.
 * Effects: abortable context GET and explicit revision-checked feedback POST; no publication.
 */
function CandidateReview({ row, runId, revision, scope, scopeKey, refresh, definitions }: {
  row: DiscoveryCandidate; runId: string; revision: number; scope: DiscoveryScope; scopeKey: string; refresh: () => Promise<unknown>;
  definitions: DiscoveryRequest["hypotheses"];
}) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [kind, setKind] = useState<DiscoveryFeedbackKind>("relevant");
  const [note, setNote] = useState("");
  const [labels, setLabels] = useState<string[]>(row.labels ?? []);
  const [people, setPeople] = useState("");
  const [saved, setSaved] = useState<DiscoveryReceipt["feedback"] | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const active = useRef(true);
  const pending = useRef(false);
  const abort = useRef<AbortController | null>(null);
  const retry = useRef<{ identity: string; id: string; body: DiscoveryFeedback } | null>(null);
  useEffect(() => { active.current = true; return () => { active.current = false; abort.current?.abort(); }; }, []);
  const context = useQuery({ queryKey: ["discovery-context", scopeKey, runId, row.candidate_id, revision],
    queryFn: ({ signal }) => discoveryContext(runId, row.candidate_id, scope, signal), enabled: open, retry: false });

  /** Persist a click once and retain its UUID for uncertain-delivery retries.
   * Input: form fields and displayed revision. Output: acknowledged receipt. Effects:
   * feedback request and readback; stale/unmounted scope never receives the response.
   */
  async function submit() {
    if (pending.current || scope.mode !== "LIVE" || (kind === "free_text" && !note.trim())) return;
    const correction = { candidate_id: row.candidate_id, expected_revision: revision, kind,
      labels: kind === "wrong_behavior" || kind === "partial_match" ? definitions.filter((definition) => labels.includes(definition.hypothesis_id)).map((definition) => definition.hypothesis_id) : [],
      attributed_actor_ids: kind === "wrong_person" ? people.split(",").map((value) => value.trim()).filter(Boolean) : [], note: note.trim() };
    const identity = JSON.stringify({ ...correction, expected_revision: undefined });
    if (retry.current?.identity !== identity) {
      const id = crypto.randomUUID();
      retry.current = { identity, id, body: { ...correction, feedback_id: id } };
    }
    const { id, body } = retry.current;
    pending.current = true; setBusy(true); setError(null);
    abort.current = new AbortController();
    try {
      const receipt = await saveDiscoveryFeedback(runId, scope, body, id, abort.current.signal);
      if (!active.current) return;
      setSaved(receipt.feedback); retry.current = null;
      queryClient.setQueryData(["discovery-run", scopeKey, runId], receipt.run);
      await refresh();
    } catch (caught) {
      if (active.current && !abort.current.signal.aborted) {
        if (caught instanceof ApiError && caught.isConflict) retry.current = null;
        setError(caught instanceof Error ? caught : new Error("Feedback could not be confirmed"));
      }
    } finally {
      pending.current = false;
      if (active.current) setBusy(false);
    }
  }
  const definitionText = (id: string) => definitions.find((definition) => definition.hypothesis_id === id)?.text ?? "Behavior definition unavailable";
  const ids = definitions.map((definition) => definition.hypothesis_id)
    .sort((left, right) => right.length - left.length).map((id) => id.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));
  const rationale = ids.length ? (row.rationale ?? "").replace(new RegExp(`\\b(${ids.join("|")})\\b`, "g"), (id) => definitionText(id)) : row.rationale;
  return <article className="space-y-3 rounded-lg border bg-card p-4">
    <header><h3 className="font-semibold">{row.title}</h3><p className="mt-1 text-sm">{row.summary}</p>
      <p className="mt-1 text-xs text-muted-foreground">{row.labels?.map(definitionText).join(", ")} · Rank score {row.score.toFixed(2)}{row.review_state ? ` · ${row.review_state}` : ""}</p></header>
    <p className="text-sm">{rationale}</p>
    <p className="text-xs">Speakers: {row.speaker_ids?.join(", ") || "Not recorded"}. Attributed conduct: {row.attributed_actor_ids?.join(", ") || "Not established"}. {row.attribution_basis}</p>
    {row.uncertainty?.length ? <ul className="list-disc pl-5 text-xs text-muted-foreground">{row.uncertainty.map((text, index) => <li key={index}>{text}</li>)}</ul> : null}
    {row.citations?.map((cite, index) => <blockquote key={index} className="border-l-2 pl-3 text-sm"><p>{cite.quote}</p><p className="break-all text-xs text-muted-foreground">Record {cite.record_id} · locator {cite.locator_id}</p></blockquote>)}
    <Button variant="outline" size="sm" onClick={() => setOpen(!open)}>{open ? "Hide source context" : "Check chronological source context"}</Button>
    {open && context.isPending ? <Loading label="Loading source context" /> : null}
    {open && context.isError ? <ErrorBox error={context.error} onRetry={() => void context.refetch()} /> : null}
    {open && context.data ? <SourceContext records={context.data.records ?? []} scope={scope} /> : null}
    <form className="space-y-2 border-t pt-3" onSubmit={(event) => { event.preventDefault(); void submit(); }}>
      <label className="block text-xs">Feedback<select className={field} disabled={busy || scope.mode !== "LIVE"} value={kind} onChange={(event) => setKind(event.target.value as DiscoveryFeedbackKind)}>{choices.map((choice) => <option key={choice.value} value={choice.value}>{choice.label}</option>)}</select></label>
      {kind === "wrong_behavior" || kind === "partial_match" ? <fieldset className="space-y-2 text-xs" disabled={busy || scope.mode !== "LIVE"}>
        <legend className="mb-1">Which behaviors apply?</legend>
        {definitions.map((definition) => <label key={definition.hypothesis_id} className="flex items-start gap-2"><input type="checkbox" checked={labels.includes(definition.hypothesis_id)} onChange={(event) => setLabels(event.target.checked ? [...labels, definition.hypothesis_id] : labels.filter((id) => id !== definition.hypothesis_id))} />{definition.text}</label>)}
        <p className="text-muted-foreground">Clear all behaviors if none apply.</p>
      </fieldset> : null}
      {kind === "wrong_person" ? <label className="block text-xs">Attributed person identifiers (comma separated)<input className={field} value={people} onChange={(event) => setPeople(event.target.value)} disabled={busy} maxLength={1000} /></label> : null}
      <label className="block text-xs">Correction or missing context<textarea className={field} value={note} onChange={(event) => setNote(event.target.value)} disabled={busy || scope.mode !== "LIVE"} maxLength={4000} rows={2} /></label>
      <Button type="submit" size="sm" disabled={busy || scope.mode !== "LIVE" || (kind === "free_text" && !note.trim())}>{busy ? "Saving…" : "Save feedback"}</Button>
    </form>
    {error ? <><ErrorBox error={error} />{error instanceof ApiError && error.isConflict ? <Button variant="outline" onClick={() => void refresh()}>Refresh changed run before reviewing again</Button> : null}</> : null}
    {saved ? <p role="status" className="break-all text-xs">Saved {choices.find((choice) => choice.value === saved.kind)?.label ?? saved.kind} · revision {saved.revision} · receipt {saved.feedback_id}</p> : null}
  </article>;
}

/** Configure and review a bounded optional discovery run for one admitted case.
 * Inputs: scope and optional persisted run URL. Output: datasets, grouped candidates and context.
 * Effects: abortable reads and explicit run/feedback writes; source workflows remain independent.
 */
export function DiscoveryReview({ scope }: { scope: DiscoveryScope }) {
  const scopeKey = discoveryScopeKey(scope);
  const router = useRouter();
  const params = useBrowserSearchParams();
  const runId = params.get("run");
  const validRunId = runId && uuid.test(runId) ? runId : null;
  const [backend, setBackend] = useState<DiscoveryBackend>("postgresql");
  const [selected, setSelected] = useState("");
  const [definitions, setDefinitions] = useState("");
  const [generations, setGenerations] = useState<string[] | null>(null);
  const [perspective, setPerspective] = useState<"hindsight" | "as_lived">("hindsight");
  const [cutoff, setCutoff] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const pending = useRef(false);
  const active = useRef(true);
  const abort = useRef<AbortController | null>(null);
  const retry = useRef<{ identity: string; id: string } | null>(null);
  useEffect(() => { active.current = true; return () => { active.current = false; abort.current?.abort(); }; }, []);
  const datasets = useQuery({ queryKey: ["discovery-datasets", scopeKey], queryFn: ({ signal }) => discoveryDatasets(scope, signal), retry: false });
  const choices = datasets.data?.items.filter((dataset) => dataset.input_backend === backend) ?? [];
  const dataset = choices.find((item) => item.dataset_id === selected);
  const selectedGenerations = generations ?? dataset?.generation_ids ?? [];
  const run = useQuery({ queryKey: ["discovery-run", scopeKey, validRunId], queryFn: ({ signal }) => discoveryRun(validRunId!, scope, signal), enabled: Boolean(validRunId), retry: false,
    refetchInterval: (query) => ["pending", "scoring", "rescore_pending", "running"].includes(query.state.data?.status ?? "") ? 3000 : false });
  const queue = useInfiniteQuery({ queryKey: ["discovery-candidates", scopeKey, validRunId, run.data?.revision], initialPageParam: "",
    queryFn: ({ pageParam, signal }) => discoveryCandidates(validRunId!, scope, pageParam, signal), enabled: Boolean(validRunId && run.data && run.data.score_revision > 0),
    getNextPageParam: (page) => page.next_cursor || undefined, retry: false });
  const groups = useMemo(() => {
    const result = new Map<string, DiscoveryCandidate[]>();
    for (const row of queue.data?.pages.flatMap((page) => page.items) ?? []) result.set(row.episode_id, [...(result.get(row.episode_id) ?? []), row]);
    return [...result.entries()];
  }, [queue.data]);

  /** Start from selected dataset pins and run-local descriptions with a stable retry ID.
   * Input: configured form. Output: persisted run URL. Effects: clicked POST and navigation;
   * changing case/mode aborts the request and prevents stale navigation.
   */
  async function start() {
    if (pending.current || scope.mode !== "LIVE" || !dataset || !selectedGenerations.length || !definitions.trim()) return;
    const hypotheses = definitions.split("\n").map((text) => text.trim()).filter(Boolean).map((text, index) => ({ hypothesis_id: `description-${index + 1}`, text }));
    const config = { scope, input_backend: backend, dataset_id: dataset.dataset_id, selection_revision: dataset.selection_revision,
      generation_ids: selectedGenerations, hypotheses, filters: { perspective, ...(perspective === "as_lived" ? { cutoff: new Date(cutoff).toISOString() } : {}) }, max_records: 200, max_candidates: 25 };
    const identity = JSON.stringify(config);
    if (retry.current?.identity !== identity) retry.current = { identity, id: crypto.randomUUID() };
    const id = retry.current.id;
    pending.current = true; setBusy(true); setError(null); abort.current = new AbortController();
    try {
      const started = await startDiscovery({ ...config, run_id: id }, id, abort.current.signal);
      if (!active.current) return;
      const query = new URLSearchParams({ mode: scope.mode, run: started.request.run_id });
      await router.navigate({ href: `/discovery?${query}` }); retry.current = null;
    } catch (caught) { if (active.current && !abort.current.signal.aborted) setError(caught instanceof Error ? caught : new Error("Discovery could not start")); }
    finally { pending.current = false; if (active.current) setBusy(false); }
  }
  const refresh = async () => { await run.refetch(); await queue.refetch(); };
  return <main className="mx-auto max-w-6xl space-y-5 p-4 lg:p-6">
    <header><h1 className="text-xl font-semibold">Discovery</h1><p className="mt-1 text-sm text-muted-foreground">Find possible behavior across source-linked episodes, check the context, and teach this discovery scope through your corrections.</p></header>
    {scope.mode === "DEV" ? <p role="status" className="text-sm">Dev review is read-only. Saving requires an isolated development workspace.</p> : null}
    <form className="space-y-3 rounded-lg border bg-card p-4" onSubmit={(event) => { event.preventDefault(); void start(); }}>
      <div className="grid gap-3 sm:grid-cols-2"><label className="text-xs">Input backend<select className={field} disabled={busy} value={backend} onChange={(event) => { setBackend(event.target.value as DiscoveryBackend); setSelected(""); setGenerations(null); }}><option value="postgresql">PostgreSQL</option><option value="surrealdb">SurrealDB</option></select></label>
        <label className="text-xs">Dataset and source generations<select className={field} value={selected} disabled={busy || datasets.isPending} onChange={(event) => { setSelected(event.target.value); setGenerations(null); }}><option value="">Choose recorded inputs</option>{choices.map((item) => <option key={item.dataset_id} value={item.dataset_id}>{item.label} · {item.generation_ids.length} generations</option>)}</select></label></div>
      {dataset ? <details className="break-all text-xs"><summary className="cursor-pointer">Selected source pins</summary><p>{dataset.selection_revision}</p><fieldset className="mt-2 space-y-1"><legend>Source generations to include</legend>{dataset.generation_ids.map((id) => <label key={id} className="flex items-start gap-2"><input type="checkbox" disabled={busy} checked={selectedGenerations.includes(id)} onChange={(event) => setGenerations(event.target.checked ? [...selectedGenerations, id] : selectedGenerations.filter((entry) => entry !== id))} />{id}</label>)}</fieldset></details> : null}
      <label className="block text-xs">Behaviors to look for (one description per line)<textarea className={field} rows={3} maxLength={16000} value={definitions} disabled={busy} onChange={(event) => setDefinitions(event.target.value)} placeholder="Describe the behavior you want to find" /></label>
      <div className="grid gap-3 sm:grid-cols-2"><label className="text-xs">Perspective<select className={field} value={perspective} disabled={busy} onChange={(event) => setPerspective(event.target.value as typeof perspective)}><option value="hindsight">Hindsight</option><option value="as_lived">As lived</option></select></label>
        {perspective === "as_lived" ? <label className="text-xs">Known by (your local time)<input className={field} type="datetime-local" required value={cutoff} disabled={busy} onChange={(event) => setCutoff(event.target.value)} /></label> : null}</div>
      <Button type="submit" disabled={busy || scope.mode !== "LIVE" || !dataset || !selectedGenerations.length || !definitions.trim() || (perspective === "as_lived" && !cutoff)}>{busy ? "Starting…" : "Discover"}</Button>
    </form>
    {datasets.isPending ? <Loading label="Loading discovery inputs" /> : null}
    {datasets.isError ? <ErrorBox error={datasets.error} onRetry={() => void datasets.refetch()} /> : null}
    {datasets.isSuccess && !choices.length ? <p className="text-sm">No available datasets for this backend.</p> : null}
    {error ? <ErrorBox error={error} /> : null}
    {runId && !validRunId ? <p role="alert">The discovery run ID is invalid.</p> : null}
    {validRunId && run.isPending ? <Loading label="Loading discovery run" /> : null}
    {run.isError ? <ErrorBox error={run.error} onRetry={() => void run.refetch()} /> : null}
    {run.data ? <section className="space-y-3"><header className="flex flex-wrap justify-between gap-2"><div><h2 className="font-semibold">Review episodes</h2><p role="status" className="text-xs">{run.data.status} · revision {run.data.revision}{run.data.dispatch_pending ? " · Saved; discovery scoring is pending" : ""}</p></div><Button variant="outline" size="sm" onClick={() => void refresh()}>Refresh</Button></header>
      {run.data.input_record_count !== undefined ? <p className="text-xs">Searched {run.data.input_record_count} source records.</p> : null}
      {run.data.input_truncated ? <p role="status" className="text-sm">Record limit reached; later records were not searched.</p> : null}
      {run.data.status === "failed" ? <p role="status" className="text-sm">Discovery scoring failed{run.data.failure_stage ? ` at ${run.data.failure_stage}` : ""}. Saved feedback remains recorded.</p> : null}
      {queue.isPending && run.data.score_revision > 0 ? <Loading /> : null}
      {queue.isError ? <ErrorBox error={queue.error} onRetry={() => void queue.refetch()} /> : null}
      {queue.isSuccess && !groups.length ? <p className="text-sm">No candidate episodes in this bounded run.</p> : null}
      {groups.map(([episode, rows]) => <section key={episode} className="space-y-3" aria-label="Candidate episode"><h3 className="text-xs text-muted-foreground">Episode · {rows.length} proposals</h3>{rows.map((row) => <CandidateReview key={`${validRunId}:${row.candidate_id}`} row={row} runId={validRunId!} revision={run.data!.revision} scope={scope} scopeKey={scopeKey} refresh={refresh} definitions={run.data!.request.hypotheses} />)}</section>)}
      {queue.hasNextPage ? <LoadMore onClick={() => void queue.fetchNextPage()} loading={queue.isFetchingNextPage} label="More episode proposals" /> : null}
    </section> : null}
  </main>;
}
