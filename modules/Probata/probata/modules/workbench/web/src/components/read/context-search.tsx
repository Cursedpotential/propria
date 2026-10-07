// Byline: Codex · 2026-10-06.
import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { ErrorBox, Loading } from "@/components/mobile/mobile-ui";
import { searchContext } from "@/lib/api-client";
import { AppLink } from "@/lib/router-compat";
import { SearchRelationships } from "@/components/read/search-relationships";
import { readHref } from "@/components/read/read-location";
import type { RetrievalHit, RetrievalLeg, RetrievalMode } from "@/lib/retrieval-types";

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
  useEffect(() => {
    setDraft(initialQuery);
    setSubmitted((previous) => previous?.text === initialQuery ? previous : initialQuery
      ? { text: initialQuery, mode: previous?.mode ?? "hybrid", legs: previous?.legs ?? ["intake", "proffer"], id: crypto.randomUUID() }
      : null);
  }, [initialQuery]);
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
  const sources = hit.navigation?.sources ?? [];
  const locator = hit.citation.locator;
  const title = sources[0]?.name || hit.location?.name || (locator.source_path || locator.vault_key || "").split(/[\\/]/).pop() || "Indexed passage";
  return <li className="space-y-2 rounded border p-3">
    <p className="text-sm font-semibold">{title}</p>
    <p className="line-clamp-5 whitespace-pre-wrap break-words text-sm leading-relaxed">{hit.text}</p>
    <details className="text-sm">
      <summary className="cursor-pointer underline underline-offset-4">Read indexed passage</summary>
      <p className="mt-2 max-h-80 overflow-auto whitespace-pre-wrap break-words leading-relaxed">{hit.text}</p>
    </details>
    <div className="flex flex-wrap gap-3">{sources.filter((source) => source.href.startsWith("/read?")).map((source) =>
      <AppLink className="text-sm underline underline-offset-4" key={source.source_version_id}
        href={readHref(new URLSearchParams(source.href.slice("/read?".length)), { q: query })}>Read {source.name}</AppLink>)}</div>
    {hit.location?.href.startsWith("/sources?") ? <AppLink className="inline-block text-sm underline underline-offset-4"
      href={hit.location.href}>Locate original file</AppLink> : null}
    {hit.navigation?.status === "unavailable" ? <p className="text-xs text-muted-foreground">The source lookup could not complete.</p> : null}
    {hit.citation.source_id && hit.citation.document_id ? <SearchRelationships sourceId={hit.citation.source_id} documentId={hit.citation.document_id} /> : null}
    <details className="text-xs text-muted-foreground"><summary className="cursor-pointer">Source details</summary>
      <dl className="mt-2 space-y-1 break-all">
        {Object.entries(locator).map(([key, value]) => <div key={key}><dt className="font-semibold">{key.replaceAll("_", " ")}</dt><dd>{value}</dd></div>)}
        {hit.citation.source_version_ids.map((version) => <div key={version}><dt>Source version</dt><dd>{version}</dd></div>)}
        <div><dt>Search object</dt><dd>{hit.citation.collection} / {hit.citation.object_id}</dd></div>
      </dl>
    </details>
  </li>;
}
