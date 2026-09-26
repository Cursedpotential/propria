// Byline: Codex · 2026-09-20. Read-only original-occurrence discovery and selection.
import { ChevronDown, ChevronRight, Download, FolderOpen, Loader2, Search } from "lucide-react";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { getDiscoveryCapabilities, listDiscoveryTree, searchDiscovery, listDiscoveryUnits, getDiscoveryUnitMembers } from "@/lib/api-client";
import type { DiscoveryCapabilities, DiscoveryItem, DiscoveryPage, DiscoveryUnits, DiscoveryUnitMembers } from "@/lib/discovery-types";

const MAX_SELECTION = 200;
const message = (error: unknown) => error instanceof Error ? error.message : "The catalog request failed.";

export function DiscoveryExplorer({ directStorage }: { directStorage: ReactNode }) {
  const [view, setView] = useState<"catalog" | "storage">("catalog");
  const [capabilities, setCapabilities] = useState<DiscoveryCapabilities | null>(null);
  const [capabilityError, setCapabilityError] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [mode, setMode] = useState<"filename_substring" | "filename_prefix" | "contents" | "hybrid">("filename_substring");
  const [parent, setParent] = useState("");
  const [applied, setApplied] = useState<{ query: string; parent: string; mode: "filename_substring" | "filename_prefix" | "contents" | "hybrid" } | null>(null);
  const [results, setResults] = useState<DiscoveryPage | null>(null);
  const [searching, setSearching] = useState(false);
  const [searchError, setSearchError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Map<string, DiscoveryItem>>(() => new Map());
  const request = useRef<AbortController | null>(null);
  const generation = useRef(0);

  useEffect(() => {
    const controller = new AbortController();
    getDiscoveryCapabilities(controller.signal).then((value) => {
      if (!controller.signal.aborted) {
        setCapabilities(value);
        if (!value.modes.filename_substring && value.modes.contents) setMode("contents");
        else if (!value.modes.filename_substring && value.modes.hybrid) setMode("hybrid");
      }
    }).catch((error) => {
      if (!controller.signal.aborted) setCapabilityError(message(error));
    });
    return () => { controller.abort(); request.current?.abort(); };
  }, []);

  function toggle(item: DiscoveryItem) {
    setSelected((current) => {
      const next = new Map(current);
      if (next.has(item.id)) next.delete(item.id);
      else if (next.size < MAX_SELECTION) next.set(item.id, item);
      return next;
    });
  }

  async function search(more = false) {
    if (searching || (more && !results?.next_cursor)) return;
    const criteria = more && applied ? applied : { query: query.trim(), parent: mode.startsWith("filename_") ? parent : "", mode };
    if (!criteria.query) return;
    request.current?.abort();
    const controller = new AbortController();
    request.current = controller;
    const current = ++generation.current;
    setSearching(true);
    setSearchError(null);
    if (!more) { setApplied(criteria); setResults(null); }
    try {
      const page = await searchDiscovery(criteria.query, criteria.parent, criteria.mode, more ? results?.next_cursor ?? undefined : undefined, controller.signal);
      if (controller.signal.aborted || current !== generation.current) return;
      setResults((previous) => ({ ...page, items: more && previous ? [...previous.items, ...page.items] : page.items }));
    } catch (error) {
      if (!controller.signal.aborted && current === generation.current) setSearchError(message(error));
    } finally {
      if (!controller.signal.aborted && current === generation.current) setSearching(false);
    }
  }

  function downloadSelection() {
    const manifest = { schema: "propria.original-occurrence-selection.v1", byline: "Probata Workbench · owner selection", created_at: new Date().toISOString(), backend: capabilities?.backend, ingestion_dispatched: false, eligibility: "Current storage resolution and per-source intake gates have not been verified.", occurrences: [...selected.values()] };
    const url = URL.createObjectURL(new Blob([JSON.stringify(manifest, null, 2)], { type: "application/json" }));
    const link = document.createElement("a");
    link.href = url;
    link.download = `original-occurrences-${new Date().toISOString().replaceAll(":", "-")}.json`;
    link.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
  const pending = applied && (query.trim() !== applied.query || mode !== applied.mode || (mode.startsWith("filename_") && parent !== applied.parent));

  return <div className="mx-auto max-w-[1180px] space-y-4">
    <div className="flex flex-wrap gap-2" role="tablist" aria-label="Source discovery method">
      <Button type="button" role="tab" aria-selected={view === "catalog"} variant={view === "catalog" ? "default" : "outline"} onClick={() => setView("catalog")}>Indexed catalog</Button>
      <Button type="button" role="tab" aria-selected={view === "storage"} variant={view === "storage" ? "default" : "outline"} onClick={() => setView("storage")}>Direct storage</Button>
    </div>
    <section hidden={view !== "catalog"} className="platform-panel overflow-hidden" aria-label="Indexed original occurrence catalog">
      <header className="border-b px-5 py-4">
        <h2 className="text-xl font-semibold">Discover original files</h2>
        <p className="mt-1 text-sm text-muted-foreground">Browse the pre-ingest catalog. Each row represents an original occurrence; identical names or bytes remain separate records.</p>
        <p className="mt-2 text-xs text-muted-foreground">Catalog paths describe recorded locations. Current storage availability and intake eligibility require separate verification.</p>
      </header>
      {!capabilities && !capabilityError && <p className="flex items-center gap-2 p-5 text-sm" role="status"><Loader2 className="h-4 w-4 animate-spin" /> Checking catalog availability — please wait.</p>}
      {capabilityError && <div className="space-y-2 p-5" role="alert"><p>Indexed catalog unavailable: {capabilityError}</p><Button type="button" variant="outline" onClick={() => setView("storage")}>Browse direct storage</Button></div>}
      {capabilities && <>
        <form className="space-y-3 border-b p-4" onSubmit={(event) => { event.preventDefault(); void search(); }}>
          <label className="grid gap-1 text-xs font-semibold">Search mode
            <select className="h-10 border bg-background px-3 font-normal" value={mode} onChange={(event) => setMode(event.target.value as typeof mode)}>
              <option value="filename_substring" disabled={!capabilities.modes.filename_substring}>Original names and paths</option>
              <option value="filename_prefix" disabled={!capabilities.modes.filename_prefix}>Original file names (prefix)</option>
              <option value="contents" disabled={!capabilities.modes.contents}>Indexed contents</option>
              <option value="hybrid" disabled={!capabilities.modes.hybrid}>Hybrid indexed search</option>
            </select>
          </label>
          <label className="grid gap-1 text-xs font-semibold">{mode.startsWith("filename_") ? "Search catalog names and paths" : "Search indexed content"}
            <input className="h-10 border bg-background px-3 font-normal" value={query} onChange={(event) => setQuery(event.target.value)} placeholder={mode.startsWith("filename_") ? mode === "filename_prefix" ? "Beginning of a file name" : "Any part of a file name or path" : "Words or concepts in indexed content"} />
          </label>
          <div className="flex flex-wrap items-center gap-3">
            <Button type="submit" disabled={searching || !query.trim() || !capabilities.modes[mode]}>{searching ? <Loader2 className="h-4 w-4 animate-spin" /> : <Search className="h-4 w-4" />}{searching ? "Searching…" : "Search"}</Button>
            <span className="text-xs text-muted-foreground">Scope: {mode.startsWith("filename_") ? parent || "all catalog folders" : "configured content index"}</span>
            {mode.startsWith("filename_") && parent && <Button type="button" variant="outline" size="sm" onClick={() => setParent("")}>All folders</Button>}
          </div>
          <p className="text-xs text-muted-foreground" role="status">{pending ? "Criteria changed. Press Search or Enter; existing results use the earlier query." : mode.startsWith("filename_") ? "Names and paths search queries recorded catalog occurrences. Content modes use the separately configured content index; coverage is not established by a successful query." : "Content results are indexed hits, not verified current source files. Folder and file-type filters are unavailable for this index connection."}</p>
        </form>
        <div className="border-b px-4 py-3 text-xs text-muted-foreground">
          <p>Index coverage: {capabilities.coverage}. ZIP-member content coverage and intake eligibility are not verified.</p>
          {capabilities.limitations?.length > 0 && <details className="mt-2"><summary>Connection limitations</summary><ul className="mt-1 list-disc pl-5">{capabilities.limitations.map((limitation) => <li key={limitation}>{limitation}</li>)}</ul></details>}
        </div>
        <div className="grid gap-0 lg:grid-cols-[minmax(18rem,0.8fr)_minmax(22rem,1.2fr)]">
          <aside className="min-w-0 border-b p-4 lg:border-b-0 lg:border-r" aria-label="Catalog folder tree">
            <h3 className="mb-3 text-sm font-semibold">Folders and original files</h3>
            {capabilities.tree ? <DirectoryBranch parent="" selected={selected} onToggle={toggle} onScope={(value) => { setParent(value); setMode("filename_substring"); }} /> : <p className="text-xs text-muted-foreground">The catalog folder connection is not configured.</p>}
          </aside>
          <div className="min-w-0 p-4" aria-busy={searching}>
            <h3 className="mb-3 text-sm font-semibold">Search results</h3>
            {searching && <p className="flex items-center gap-2 text-sm" role="status"><Loader2 className="h-4 w-4 animate-spin" /> Searching the catalog — please wait.</p>}
            {searchError && <p className="text-sm text-destructive" role="alert">{searchError}</p>}
            {!searching && results && <>
              <p className="mb-2 text-xs text-muted-foreground">{results.items.length} {applied?.mode.startsWith("filename_") ? "original occurrences" : "indexed hits"} returned for “{applied?.query}”{results.has_more ? "; more available" : results.complete ? "; end of results" : "; index coverage unknown"}.</p>
              {!results.items.length && <p className="text-sm">No matches returned for this query. This does not establish absence from unindexed sources.</p>}
              {results.freshness && <p className="mb-2 text-xs text-muted-foreground">Catalog snapshot: {results.freshness.catalog_snapshot}. A fresh query does not refresh source metadata.</p>}
              {results.notice && <p className="mb-2 text-xs text-muted-foreground">{results.notice}</p>}
              <ul className="space-y-1">{results.items.map((item) => <OccurrenceRow key={item.id} item={item} selected={selected.has(item.id)} limitReached={selected.size >= MAX_SELECTION} onToggle={toggle} />)}</ul>
              {results.has_more && results.next_cursor && <Button className="mt-3" variant="outline" type="button" disabled={Boolean(pending)} onClick={() => void search(true)}>More catalog results</Button>}
            </>}
            {!applied && <p className="text-sm text-muted-foreground">Enter a name or path above, or expand folders to browse recorded occurrences.</p>}
          </div>
        </div>
        {capabilities.atomic_unit_catalog && <AtomicUnits unitTypes={capabilities.unit_types ?? []} />}
        <footer className="space-y-3 border-t bg-card p-4">
          <strong className="block text-sm">{selected.size} original occurrences selected{selected.size >= MAX_SELECTION ? ` (limit ${MAX_SELECTION})` : ""}</strong>
          <p className="text-xs text-muted-foreground">Selection persists across folders and result pages. Bulk intake is unavailable: occurrence-to-current-storage resolution and a batch review queue are not connected. Selecting or exporting does not start ingestion.</p>
          <div className="flex gap-2"><Button type="button" variant="outline" disabled={!selected.size} onClick={downloadSelection}><Download className="h-4 w-4" /> Export occurrence selection</Button><Button type="button" variant="outline" disabled={!selected.size} onClick={() => setSelected(new Map())}>Clear selection</Button></div>
          {selected.size > 0 && <details><summary className="cursor-pointer text-xs">Review selected occurrences</summary><ul className="mt-2 space-y-1">{[...selected.values()].map((item) => <OccurrenceRow key={item.id} item={item} selected limitReached={false} onToggle={toggle} />)}</ul></details>}
        </footer>
      </>}
    </section>
    <div hidden={view !== "storage"}><p className="mb-3 text-xs text-muted-foreground">Direct storage lists current objects. Its bounded filename/path scan is separate from the indexed catalog.</p>{directStorage}</div>
  </div>;
}

function DirectoryBranch({ parent, selected, onToggle, onScope }: { parent: string; selected: Map<string, DiscoveryItem>; onToggle: (item: DiscoveryItem) => void; onScope: (parent: string) => void }) {
  const [page, setPage] = useState<DiscoveryPage | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set());
  const controller = useRef<AbortController | null>(null);
  useEffect(() => {
    const pending = new AbortController();
    controller.current = pending;
    listDiscoveryTree(parent, undefined, pending.signal).then((value) => { if (!pending.signal.aborted) setPage(value); }).catch((failure) => { if (!pending.signal.aborted) setError(message(failure)); }).finally(() => { if (!pending.signal.aborted) setLoading(false); });
    return () => pending.abort();
  }, [parent]);
  async function more() {
    if (loading || !page?.next_cursor) return;
    setLoading(true); setError(null);
    try {
      const next = await listDiscoveryTree(parent, page.next_cursor, controller.current?.signal);
      if (!controller.current?.signal.aborted) setPage({ ...next, items: [...page.items, ...next.items] });
    } catch (failure) { if (!controller.current?.signal.aborted) setError(message(failure)); }
    finally { if (!controller.current?.signal.aborted) setLoading(false); }
  }
  return <div aria-busy={loading}>
    {loading && <p className="flex items-center gap-1 py-2 text-xs" role="status"><Loader2 className="h-3 w-3 animate-spin" /> Loading folder…</p>}
    {error && <p className="py-2 text-xs text-destructive" role="alert">{error}</p>}
    <ul className="space-y-1">{page?.items.map((item) => item.kind === "directory" ? <li key={item.id}>
      <div className="flex items-center gap-1">
        <button type="button" className="flex min-w-0 flex-1 items-center gap-1 py-1 text-left text-xs" aria-expanded={expanded.has(item.id)} onClick={() => setExpanded((current) => { const next = new Set(current); if (next.has(item.id)) next.delete(item.id); else next.add(item.id); return next; })}>{expanded.has(item.id) ? <ChevronDown className="h-3 w-3 shrink-0" /> : <ChevronRight className="h-3 w-3 shrink-0" />}<FolderOpen className="h-3 w-3 shrink-0" /><span className="break-all">{item.name}</span></button>
        <button type="button" className="shrink-0 text-[10px] text-primary" onClick={() => onScope(item.rel)}>Search here</button>
      </div>
      {expanded.has(item.id) && <div className="ml-3 border-l pl-2"><DirectoryBranch parent={item.rel} selected={selected} onToggle={onToggle} onScope={onScope} /></div>}
    </li> : <OccurrenceRow key={item.id} item={item} selected={selected.has(item.id)} limitReached={selected.size >= MAX_SELECTION} onToggle={onToggle} />)}</ul>
    {!loading && page && !page.items.length && <p className="text-xs text-muted-foreground">No catalog entries in this folder.</p>}
    {page?.has_more && page.next_cursor && <Button type="button" size="sm" variant="outline" className="mt-2" disabled={loading} onClick={() => void more()}>More folder entries</Button>}
  </div>;
}

function OccurrenceRow({ item, selected, limitReached, onToggle }: { item: DiscoveryItem; selected: boolean; limitReached: boolean; onToggle: (item: DiscoveryItem) => void }) {
  if (item.kind === "content_hit") return <li className="rounded border p-2 text-xs"><strong className="block break-all">{item.name}</strong><span className="block break-all text-muted-foreground">{item.rel}</span><p className="mt-1 whitespace-pre-wrap break-words">{item.text}</p><span className="mt-1 block text-[10px] text-muted-foreground">Indexed hit {item.id} · Current source resolution unavailable; not selectable for intake.</span></li>;
  return <li className="rounded border p-2 text-xs"><label className="flex items-start gap-2"><input type="checkbox" className="mt-0.5" checked={selected} disabled={!selected && limitReached} onChange={() => onToggle(item)} /><span className="min-w-0"><strong className="block break-all font-medium">{item.name}</strong><span className="block break-all text-muted-foreground">{item.rel}</span><span className="block text-[10px] text-muted-foreground">Occurrence {item.id}{item.recorded_at ? ` · catalog recorded ${new Date(item.recorded_at).toLocaleString()}` : ""}</span></span></label></li>;
}

function AtomicUnits({ unitTypes }: { unitTypes: string[] }) {
  const [unitType, setUnitType] = useState("");
  const [appliedType, setAppliedType] = useState("");
  const [page, setPage] = useState<DiscoveryUnits | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const request = useRef<AbortController | null>(null);
  useEffect(() => () => request.current?.abort(), []);
  async function load(more = false) {
    if (loading || (more && (page?.next_after_id === null || page?.next_after_id === undefined))) return;
    request.current?.abort();
    const controller = new AbortController(); request.current = controller;
    const filter = more ? appliedType : unitType.trim();
    setLoading(true); setError(null);
    if (!more) { setPage(null); setAppliedType(filter); }
    try {
      const next = await listDiscoveryUnits(filter, more ? page?.next_after_id ?? -1 : -1, controller.signal);
      if (!controller.signal.aborted) setPage((previous) => ({ ...next, items: more && previous ? [...previous.items, ...next.items] : next.items }));
    } catch (failure) { if (!controller.signal.aborted) setError(message(failure)); }
    finally { if (!controller.signal.aborted) setLoading(false); }
  }
  return <section className="space-y-3 border-t p-4" aria-label="Historical processing units">
    <h3 className="text-sm font-semibold">Historical processing units</h3>
    <p className="text-xs text-muted-foreground">Browse recorded source groups and their members. These records describe prior cataloging; current storage links and intake readiness remain unverified.</p>
    <form className="flex flex-wrap items-end gap-2" onSubmit={(event) => { event.preventDefault(); void load(); }}>
      <label className="grid gap-1 text-xs">Recorded unit type<select className="h-9 border bg-background px-2" value={unitType} onChange={(event) => setUnitType(event.target.value)}><option value="">All unit types</option>{unitTypes.map((type) => <option key={type} value={type}>{type.replaceAll("_", " ")}</option>)}</select></label>
      <Button type="submit" variant="outline" disabled={loading}>{loading ? "Loading units…" : "Find units"}</Button>
    </form>
    {loading && <p className="text-xs" role="status">Loading historical units — please wait.</p>}
    {error && <p className="text-xs text-destructive" role="alert">{error}</p>}
    {page && <>
      <p className="text-xs text-muted-foreground">{page.notice} · {page.items.length} units returned for {appliedType || "all types"}.</p>
      <ul className="space-y-2">{page.items.map((unit) => <li key={unit.unit_id} className="border p-3"><strong className="block break-all text-xs">{unit.unit_type} · {unit.unit_root}</strong><span className="block text-xs text-muted-foreground">Unit {unit.unit_id} · {unit.member_count} members · {unit.members_without_sha1} members without a recorded SHA-1</span><UnitMembers unitId={unit.unit_id} /></li>)}</ul>
      {page.next_after_id !== null && <Button type="button" variant="outline" disabled={loading || unitType.trim() !== appliedType} onClick={() => void load(true)}>More units</Button>}
    </>}
  </section>;
}

function UnitMembers({ unitId }: { unitId: number }) {
  const [members, setMembers] = useState<DiscoveryUnitMembers | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const request = useRef<AbortController | null>(null);
  useEffect(() => () => request.current?.abort(), []);
  async function load(more = false) {
    if (loading || (members && !more) || (more && !members?.next_cursor)) return;
    const controller = new AbortController(); request.current = controller;
    setLoading(true); setError(null);
    try { const response = await getDiscoveryUnitMembers(unitId, controller.signal, more ? members?.next_cursor ?? undefined : undefined); if (!controller.signal.aborted) setMembers((previous) => ({ ...response, items: more && previous ? [...previous.items, ...response.items] : response.items })); }
    catch (failure) { if (!controller.signal.aborted) setError(message(failure)); }
    finally { if (!controller.signal.aborted) setLoading(false); }
  }
  return <details className="mt-2 text-xs" onToggle={(event) => { if (event.currentTarget.open) void load(); }}><summary className="cursor-pointer">Show recorded member keys</summary>
    {loading && <p role="status">Loading members — please wait.</p>}
    {error && <p role="alert">{error}</p>}
    {members && <><p className="my-2 text-muted-foreground">{members.notice}{members.has_more ? " More members are available." : ""}</p><ul className="max-h-64 space-y-1 overflow-auto">{members.items.map((member) => <li key={`${member.unit_id}:${member.key}`} className="break-all font-mono text-[10px]">{member.key}</li>)}</ul>{members.has_more && members.next_cursor && <Button type="button" size="sm" variant="outline" disabled={loading} onClick={() => void load(true)}>More member keys</Button>}</>}
  </details>;
}
