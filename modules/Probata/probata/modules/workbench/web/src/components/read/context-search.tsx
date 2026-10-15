// Byline: Codex · 2026-10-06.
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { ErrorBox, Loading } from "@/components/mobile/mobile-ui";
import { searchContext } from "@/lib/api-client";
import { AppLink } from "@/lib/router-compat";
import { SearchRelationships } from "@/components/read/search-relationships";
import { readHref } from "@/components/read/read-location";
import type { RetrievalHit, RetrievalLeg, RetrievalMode } from "@/lib/retrieval-types";
import { nativeSearchTargets } from "@/lib/search-read-target";
import { attemptReadHref } from "@/lib/workflow-links";
import { useFixedCase } from "@/lib/fixed-case-context";

const families: { id: RetrievalLeg; label: string }[] = [
  { id: "intake", label: "Case Bible / Intake" }, { id: "proffer", label: "Ingested material" },
];

/** Search before and after ingestion while retaining original citations and explicit source failures.
 * Inputs: optional initial query and URL callback; output: query controls and cited results.
 * Effects: bounded read-only requests and source navigation, never ingestion or evidence writes.
 * Use on both Sources and Read instead of duplicating retrieval logic in page components.
 */
export function ContextSearch({ initialQuery = "", onSubmit }: { initialQuery?: string; onSubmit?: (query: string) => void }) {
  const [draft, setDraft] = useState(initialQuery);
  const [mode, setMode] = useState<RetrievalMode>("hybrid");
  const [legs, setLegs] = useState<RetrievalLeg[]>(["intake", "proffer"]);
  const [submitted, setSubmitted] = useState<{ text: string; mode: RetrievalMode; legs: RetrievalLeg[]; id: string } | null>(
    () => initialQuery ? { text: initialQuery, mode: "hybrid", legs: ["intake", "proffer"], id: crypto.randomUUID() } : null,
  );
  const [previousInitialQuery, setPreviousInitialQuery] = useState(initialQuery);
  if (previousInitialQuery !== initialQuery) {
    setPreviousInitialQuery(initialQuery);
    setDraft(initialQuery);
    setSubmitted((previous) => previous?.text === initialQuery ? previous : initialQuery
      ? { text: initialQuery, mode: previous?.mode ?? "hybrid", legs: previous?.legs ?? ["intake", "proffer"], id: crypto.randomUUID() }
      : null);
  }
  const query = useQuery({
    queryKey: ["context-retrieval", submitted],
    queryFn: ({ signal }) => searchContext(submitted!.text, submitted!.mode, submitted!.legs, submitted!.id, signal),
    enabled: Boolean(submitted), retry: false,
  });
  return <section className="space-y-3 rounded-md border bg-card p-3" aria-label="Search content across sources">
    <form className="space-y-3" role="search" onSubmit={(event) => {
      event.preventDefault();
      if (!draft.trim() || !legs.length) return;
      setSubmitted({ text: draft.trim(), mode, legs: [...legs], id: crypto.randomUUID() });
      onSubmit?.(draft.trim());
    }}>
      <label className="block text-sm font-semibold" htmlFor="combined-search-query">Search content</label>
      <div className="flex gap-2">
        <input id="combined-search-query" type="search" className="min-w-0 flex-1 rounded border bg-background px-3 py-2 text-sm"
          value={draft} onChange={(event) => setDraft(event.target.value)} maxLength={2000} placeholder="Find a passage, topic, or phrase" />
        <Button disabled={!draft.trim() || !legs.length || query.isFetching} type="submit">{query.isFetching ? "Searching…" : "Search"}</Button>
        {submitted ? <Button type="button" variant="outline" onClick={() => { setSubmitted(null); setDraft(""); onSubmit?.(""); }}>Clear</Button> : null}
      </div>
      <div className="flex flex-wrap items-center gap-4 text-xs">
        <label>Match <select className="ml-2 rounded border bg-background px-2 py-1" value={mode} onChange={(event) => setMode(event.target.value as RetrievalMode)}>
          <option value="hybrid">Words and meaning</option><option value="keyword">Full text</option><option value="vector">Meaning</option>
        </select></label>
        {families.map((family) => <label className="flex items-center gap-1.5" key={family.id}>
          <input type="checkbox" checked={legs.includes(family.id)} onChange={(event) => setLegs((current) => event.target.checked ? [...current, family.id] : current.filter((value) => value !== family.id))} />
          {family.label}
        </label>)}
      </div>
    </form>
    {submitted && query.isPending ? <Loading /> : null}
    {submitted && query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : null}
    {submitted && query.data ? <div className="space-y-3" aria-live="polite">
      <p className="text-xs text-muted-foreground">{query.data.items.length} ranked results for “{submitted.text}”</p>
      {query.data.legs.filter((leg) => leg.status === "failed").map((leg) => <p className="text-sm text-amber-700 dark:text-amber-300" role="status" key={leg.name}>
        {families.find((family) => family.id === leg.name)?.label ?? leg.name}: {leg.error === "timeout" ? "search timed out" : leg.error === "unsupported_mode" ? "this matching mode is not available" : "search could not complete"}.
      </p>)}
      {!query.data.items.length && query.data.status !== "failed" ? <p className="text-sm">No matches in the indexes that answered.</p> : null}
      <ul className="max-h-[65vh] space-y-3 overflow-auto">{query.data.items.map((hit) => <SearchHit key={JSON.stringify(hit.citation)} hit={hit} query={submitted.text} />)}</ul>
    </div> : null}
  </section>;
}

/** Render an excerpt and recorded provenance without treating an index locator as a verified source URL.
 * Input: cited hit; output: safe source links supplied by the BFF and expandable coordinates.
 * Effects: navigation only. Pick for all combined-search result rows.
 */
function SearchHit({ hit, query }: { hit: RetrievalHit; query: string }) {
  const { mode } = useFixedCase();
  const sources = hit.navigation?.sources ?? [];
  const locator = hit.citation.locator;
  const target = nativeSearchTargets({
    corpus: hit.corpus ?? hit.citation.corpus ?? locator.corpus,
    source_kind: hit.source_kind ?? hit.citation.source_kind ?? locator.source_kind,
    source_format: hit.source_format ?? hit.citation.source_format ?? locator.source_format ?? locator.declared_format,
    record_kind: hit.record_kind ?? hit.citation.record_kind ?? locator.record_kind ?? locator.record_type,
    origin_system: hit.origin_system ?? hit.citation.origin_system ?? locator.origin_system,
    preview_handle: hit.preview_handle ?? hit.citation.preview_handle ?? locator.preview_handle,
    source_version_ref: locator.source_version_ref,
    source_version_id: locator.source_version_id,
    source_version_ids: hit.citation.source_version_ids,
  });
  const links = target.native
    ? target.handles.map((handle) => ({ id: handle, name: "source messages", href: attemptReadHref(handle, mode) }))
    : sources.filter((source) => source.href.startsWith("/read?")).flatMap((source) => {
      const params = new URLSearchParams(source.href.slice("/read?".length));
      const native = nativeSearchTargets({ preview_handle: params.get("resource") ?? params.get("preview_handle") ?? params.get("attempt"), source_version_id: source.source_version_id });
      if (native.native) return native.handles.map((handle) => ({ id: handle, name: source.name, href: attemptReadHref(handle, mode) }));
      return [{ id: source.source_version_id, name: source.name, href: readHref(params, { q: query }) }];
    });
  const title = sources[0]?.name || hit.location?.name || (locator.source_path || locator.vault_key || "").split(/[\\/]/).pop() || "Indexed passage";
  const kind = locator.source_kind || locator.record_type || locator.content_type || "type not supplied by index";
  const origin = hit.origin_system || hit.citation.origin_system || locator.origin_system || hit.legs.join(", ") || "origin not supplied";
  const excerpt = searchExcerpt(hit.text);
  return <li className="space-y-2 rounded border p-3">
    <p className="text-sm font-semibold">{title}</p>
    <p className="text-xs text-muted-foreground">{kind} · Indexed match from {origin} · {hit.citation.source_version_ids.length} recorded source version(s)</p>
    <p className="line-clamp-5 whitespace-pre-wrap break-words text-sm leading-relaxed">{excerpt}</p>
    <details className="text-sm">
      <summary className="cursor-pointer underline underline-offset-4">Details: complete indexed passage</summary>
      <p className="mt-2 max-h-80 overflow-auto whitespace-pre-wrap break-words leading-relaxed">{hit.text}</p>
    </details>
    <div className="flex flex-wrap gap-3">{links.map((source) =>
      <AppLink className="text-sm underline underline-offset-4" key={source.id}
        href={source.href}>Read {source.name}</AppLink>)}</div>
    {target.native && !links.length ? <p className="text-xs text-muted-foreground">This indexed passage has no verified human-source reader reference. Its excerpt and source coordinates remain available.</p> : null}
    {hit.location?.href.startsWith("/sources?") ? <AppLink className="inline-block text-sm underline underline-offset-4"
      href={hit.location.href}>Locate original file</AppLink> : null}
    {hit.navigation?.status === "unavailable" ? <p className="text-xs text-muted-foreground">The source lookup could not complete.</p> : null}
    {!target.native && hit.citation.source_id && hit.citation.document_id ? <SearchRelationships sourceId={hit.citation.source_id} documentId={hit.citation.document_id} /> : null}
    <details className="text-xs text-muted-foreground"><summary className="cursor-pointer">Source details</summary>
      <dl className="mt-2 space-y-1 break-all">
        {Object.entries(locator).map(([key, value]) => <div key={key}><dt className="font-semibold">{key.replaceAll("_", " ")}</dt><dd>{value}</dd></div>)}
        {hit.citation.source_version_ids.map((version) => <div key={version}><dt>Source version</dt><dd>{version}</dd></div>)}
        <div><dt>Search object</dt><dd>{hit.citation.collection} / {hit.citation.object_id}</dd></div>
      </dl>
    </details>
  </li>;
}

/** Render bounded text from a structured indexed passage without changing retrieval ranking.
 * Input: raw index text. Output: readable excerpt or a labeled structured fragment.
 * Effects: none; choose for result summaries while retaining complete raw text in Details.
 */
function searchExcerpt(text: string): string {
  try {
    const value: unknown = JSON.parse(text);
    if (typeof value === "string") return value.slice(0, 800);
    if (value && typeof value === "object" && !Array.isArray(value)) {
      const record = value as Record<string, unknown>;
      for (const key of ["text", "content", "message", "body", "summary"]) {
        if (typeof record[key] === "string") return String(record[key]).slice(0, 800);
      }
      return `Structured indexed fragment. Fields: ${Object.keys(record).slice(0, 12).join(", ")}. Open Details to inspect the complete passage.`;
    }
    if (Array.isArray(value)) return `Structured indexed array (${value.length} entries). Open Details to inspect the complete passage.`;
  } catch { /* Non-JSON text remains a literal excerpt. */ }
  return text.slice(0, 800);
}
