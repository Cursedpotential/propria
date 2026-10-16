// Milestone 1: readable, category-first import browsing over the existing read-only API.
// This component never infers ownership, duplicate status, or derivative authority from a filename.
import { useInfiniteQuery } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { Archive, FileQuestion, FolderOpen, Layers, MessageSquareText, PhoneCall, Search } from "lucide-react";
import { useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ErrorBox, Loading } from "@/components/mobile/mobile-ui";
import { formatCount, statusLabel } from "@/components/mobile/mobile-format";
import { readHref } from "@/components/read/read-location";
import { importedApi, type SourceStatus } from "@/lib/imported-client";
import {
  libraryFormat, possibleSameNameImports,
  type LibraryFormat, type LibraryOwnerFilter,
} from "./library-facets";

const FORMATS = [
  { id: "all", label: "All imports", icon: Layers },
  { id: "SMS", label: "Texts", icon: MessageSquareText },
  { id: "Calls", label: "Calls", icon: PhoneCall },
  { id: "Facebook", label: "Social", icon: FolderOpen },
  { id: "Other", label: "Other", icon: FileQuestion },
] as const;

const STATUSES: Array<{ id: "all" | SourceStatus; label: string }> = [
  { id: "all", label: "Any status" },
  { id: "committed", label: "Committed" },
  { id: "awaiting_review", label: "Awaiting review" },
  { id: "running", label: "Running" },
  { id: "failed", label: "Failed" },
  { id: "parked", label: "Parked" },
  { id: "not_finished", label: "Not finished" },
  { id: "skipped", label: "Skipped" },
];

/**
 * Browse real imported sources by their recorded server format, not a filename guess.
 * Inputs: /read query context and optional selected source. Output: a source picker
 * with observed processing history and possible same-name import candidates.
 * Effects: read-only paged GETs and navigation; never rewrites canonical sources.
 * Choose for the imported-library section of Read/Review, including phone widths.
 */
export function ImportedLibrary({ params, selectedSourceId }: {
  params: URLSearchParams; selectedSourceId: string | null;
}) {
  const router = useRouter();
  const [format, setFormat] = useState<LibraryFormat>(() => libraryFormat(params.get("lib_format")));
  const [search, setSearch] = useState(params.get("lib_q") ?? "");
  const [appliedSearch, setAppliedSearch] = useState(params.get("lib_q") ?? "");
  const [status, setStatus] = useState<"all" | SourceStatus>(() => {
    const value = params.get("lib_status");
    return STATUSES.find((item) => item.id === value)?.id ?? "all";
  });
  const [showHistory, setShowHistory] = useState(false);
  const [ownerFilter, setOwnerFilter] = useState<LibraryOwnerFilter>(() => {
    const value = params.get("lib_owner");
    return value === "recorded" || value === "unrecorded" ? value : "all";
  });

  const query = useInfiniteQuery({
    queryKey: ["m-sources", "library", format, status, ownerFilter, appliedSearch],
    queryFn: ({ pageParam, signal }) => importedApi.sources(
      pageParam, format === "all" ? undefined : format, signal, {
        status: status === "all" ? undefined : status,
        owner: ownerFilter === "all" ? undefined : ownerFilter,
        q: appliedSearch || undefined,
      }
    ),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_offset ?? undefined,
  });
  const history = useInfiniteQuery({
    queryKey: ["m-source-history", selectedSourceId],
    queryFn: ({ pageParam, signal }) => importedApi.sourceHistory(selectedSourceId!, pageParam, signal),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_offset ?? undefined,
    enabled: Boolean(selectedSourceId) && showHistory,
  });
  const historyRows = history.data?.pages.flatMap((page) => page.items) ?? [];
  const rows = useMemo(() => query.data?.pages.flatMap((page) => page.items) ?? [], [query.data]);
  const shown = rows;
  const selected = rows.find((source) => source.id === selectedSourceId);
  const candidateNames = selected ? possibleSameNameImports(selected, rows) : [];
  const total = query.data?.pages[0]?.total;
  const hasDisplayFilter = Boolean(appliedSearch.trim()) || status !== "all" || ownerFilter !== "all";

  function preserve(filters: {
    lib_format?: LibraryFormat;
    lib_status?: "all" | SourceStatus;
    lib_owner?: LibraryOwnerFilter;
    lib_q?: string;
    source?: string | null;
    thread?: string | null;
    around?: string | null;
  }) {
    const changes: Record<string, string | null> = {};
    const next = { lib_format: format, lib_status: status, lib_owner: ownerFilter, lib_q: appliedSearch, ...filters };
    changes.lib_format = next.lib_format === "all" ? null : next.lib_format;
    changes.lib_status = next.lib_status === "all" ? null : next.lib_status;
    changes.lib_owner = next.lib_owner === "all" ? null : next.lib_owner;
    changes.lib_q = next.lib_q.trim() || null;
    if ("source" in filters) changes.source = filters.source ?? null;
    if ("thread" in filters) changes.thread = filters.thread ?? null;
    if ("around" in filters) changes.around = filters.around ?? null;
    void router.navigate({ href: readHref(params, changes), replace: true });
  }

  return (
    <section className="overflow-hidden rounded-lg border bg-card" aria-label="Imported library" data-testid="imported-library">
      <header className="border-b px-3 py-3">
        <div className="flex items-center gap-2">
          <Archive className="size-4 text-primary" aria-hidden="true" />
          <h2 className="text-sm font-semibold">Imported library</h2>
        </div>
        <p className="mt-1 text-xs text-muted-foreground">
          Browse recorded imports by source type. Processing attempts remain linked to the source.
        </p>
      </header>

      <div className="grid grid-cols-2 gap-1.5 border-b p-3 sm:grid-cols-3" aria-label="Imported source categories">
        {FORMATS.map((category) => {
          const Icon = category.icon;
          const chosen = format === category.id;
          return (
            <button key={category.id} type="button" aria-pressed={chosen}
              className={chosen
                ? "flex min-h-10 items-center gap-2 rounded-md border border-primary bg-primary/10 px-3 py-2 text-left text-xs font-semibold text-foreground"
                : "flex min-h-10 items-center gap-2 rounded-md border border-border bg-muted/30 px-3 py-2 text-left text-xs text-muted-foreground hover:bg-accent hover:text-foreground"}
              onClick={() => {
                setFormat(category.id);
                preserve({ lib_format: category.id });
              }}>
              <Icon className="size-4 shrink-0" aria-hidden="true" />
              <span>{category.label}</span>
            </button>
          );
        })}
      </div>

      <div className="space-y-2 border-b p-3">
        <form role="search" onSubmit={(event) => { event.preventDefault(); setAppliedSearch(search.trim()); preserve({ lib_q: search.trim() }); }}>
          <label className="mb-1 block text-xs font-medium" htmlFor="imported-library-filter">Find across imported sources</label>
          <div className="flex items-center gap-2">
            <Input id="imported-library-filter" type="search" value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Filename, path, owner, device…" className="h-9 min-w-0 flex-1 text-sm" />
            <Button type="submit" size="sm" variant="outline" aria-label="Keep import filter in the URL">
              <Search className="size-4" aria-hidden="true" />
            </Button>
          </div>
        </form>
        <div className="grid grid-cols-2 gap-2">
          <label className="space-y-1 text-xs text-muted-foreground">
            <span className="block">Processing state</span>
            <select className="h-9 w-full rounded border bg-background px-2 text-xs text-foreground"
              value={status} onChange={(event) => {
                const value = event.target.value as "all" | SourceStatus;
                setStatus(value);
                preserve({ lib_status: value });
              }}>
              {STATUSES.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}
            </select>
          </label>
          <label className="space-y-1 text-xs text-muted-foreground">
            <span className="block">Recorded source owner</span>
            <select className="h-9 w-full rounded border bg-background px-2 text-xs text-foreground"
              value={ownerFilter} onChange={(event) => {
                const value = event.target.value as LibraryOwnerFilter;
                setOwnerFilter(value);
                preserve({ lib_owner: value });
              }}>
              <option value="all">Any owner</option>
              <option value="recorded">Owner recorded</option>
              <option value="unrecorded">Owner not recorded</option>
            </select>
          </label>
        </div>
        <p className="text-[11px] leading-relaxed text-muted-foreground">
          {hasDisplayFilter ? "Filters apply across all recorded imports. " : ""}
          {format === "all" ? "All recorded formats" : "Recorded " + format + " sources"}
          {typeof total === "number" ? " · " + formatCount(total) + " total" : ""}
          {" · " + formatCount(rows.length) + " loaded"}
          {hasDisplayFilter ? " · " + formatCount(shown.length) + " loaded matches" : ""}
        </p>
      </div>

      {query.isPending ? <Loading /> : null}
      {query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : null}
      {query.data ? (
        <>
          <ul className="max-h-[23rem] divide-y overflow-y-auto" aria-label="Recorded imported sources">
            {shown.map((source, index) => {
              const active = source.id === selectedSourceId;
              const owner = source.owner_name || source.owner || "not recorded";
              return (
                <li key={source.id + ":" + index}>
                  <button type="button" aria-current={active ? "true" : undefined}
                    className={active ? "w-full border-l-2 border-primary bg-primary/5 px-3 py-2.5 text-left" : "w-full border-l-2 border-transparent px-3 py-2.5 text-left hover:bg-accent/60"}
                    onClick={() => preserve({ source: source.id, thread: null, around: null })}>
                    <span className="block break-all text-sm font-medium text-foreground">{source.file_name}</span>
                    <span className="mt-1 block break-all text-[11px] text-muted-foreground">{source.casevault_key || "No source path recorded"}</span>
                    <span className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-[11px] text-muted-foreground">
                      <span>{source.format}</span>
                      <span>{statusLabel(source.status)}</span>
                      <span>{formatCount(source.messages + source.calls)} communications</span>
                    </span>
                    <span className="mt-0.5 block truncate text-[11px] text-muted-foreground">Source owner: {owner}</span>
                  </button>
                </li>
              );
            })}
          </ul>
          {!shown.length ? (
            <p className="px-3 py-7 text-sm text-muted-foreground">
              No recorded imported sources match these filters.
            </p>
          ) : null}
          {query.hasNextPage ? (
            <div className="border-t p-3">
              <Button type="button" variant="outline" className="w-full"
                disabled={query.isFetchingNextPage}
                onClick={() => void query.fetchNextPage()}>
                {query.isFetchingNextPage ? "Loading…" : "Load more recorded imports"}
              </Button>
            </div>
          ) : null}
        </>
      ) : null}

      {selected ? (
        <div className="space-y-2 border-t bg-muted/20 p-3" aria-label="Selected source processing history">
          <h3 className="text-xs font-semibold">Recorded processing history</h3>
          <p className="text-xs text-muted-foreground">
            {formatCount(selected.files)} files · {formatCount(selected.raw)} raw records ·
            {" " + formatCount(selected.normalized)} extracted · {formatCount(selected.committed)} committed
          </p>
          {selected.failed_attempts > 0 ? (
            <p className="text-xs text-muted-foreground">{formatCount(selected.failed_attempts)} historical failed attempts recorded</p>
          ) : null}
          {selected.split ? (
            <p className="text-xs text-muted-foreground">
              Split package: {formatCount(selected.split.done)} done, {formatCount(selected.split.failed)} failed,
              {" " + formatCount(selected.split.in_progress)} in progress of {formatCount(selected.split.total)}
            </p>
          ) : null}
          <div className="space-y-2 rounded-md border bg-card p-2">
            <button type="button" className="flex w-full items-center justify-between text-left text-xs font-semibold text-foreground"
              aria-expanded={showHistory} onClick={() => setShowHistory((open) => !open)}>
              <span>View recorded source versions</span>
              <span className="text-muted-foreground">{showHistory ? "Hide" : "Open"}</span>
            </button>
            {showHistory ? <>
              <p className="text-[11px] text-muted-foreground">
                Exact export-key association, not proof that two files or generated versions are duplicates.
              </p>
              {history.isPending ? <Loading /> : null}
              {history.isError ? <ErrorBox error={history.error} onRetry={() => void history.refetch()} /> : null}
              {history.data ? <>
                <p className="text-[11px] text-muted-foreground">
                  {formatCount(historyRows.length)} of {formatCount(history.data.pages[0].total)} recorded versions
                </p>
                <ul className="max-h-72 divide-y overflow-y-auto" aria-label="Recorded source versions">
                  {historyRows.map((entry) => (
                    <li key={entry.source_version_id} className="space-y-1 py-2 text-xs">
                      <p className="break-all font-medium text-foreground">{entry.source_key}</p>
                      <p className="text-[11px] text-muted-foreground">
                        {entry.relation === "export_root" ? "Export root" : "Member of recorded export"}
                        {" · version " + entry.version_ordinal}
                        {" · " + (entry.published ? "published" : entry.approved ? "approved" : entry.rejected ? "rejected" : "not yet approved")}
                      </p>
                      <p className="text-[11px] text-muted-foreground">
                        {formatCount(entry.raw)} raw · {formatCount(entry.normalized)} extracted
                        {" · " + (entry.acquired_at || "acquisition time unknown")}
                      </p>
                      <p className="break-all font-mono text-[10px] text-muted-foreground">Source version: {entry.source_version_id}</p>
                    </li>
                  ))}
                </ul>
                {history.hasNextPage ? (
                  <Button type="button" size="sm" variant="outline" className="w-full"
                    disabled={history.isFetchingNextPage} onClick={() => void history.fetchNextPage()}>
                    {history.isFetchingNextPage ? "Loading…" : "Load more versions"}
                  </Button>
                ) : null}
              </> : null}
            </> : null}
          </div>
          {candidateNames.length ? (
            <details className="text-xs">
              <summary className="cursor-pointer font-medium">Same-name imports in loaded results ({candidateNames.length})</summary>
              <p className="mt-1 text-muted-foreground">Name match only; bytes and derivation are not verified.</p>
              <ul className="mt-2 space-y-2">
                {candidateNames.map((candidate) => (
                  <li key={candidate.id} className="break-all">
                    <button type="button" className="text-left text-primary underline underline-offset-4"
                      onClick={() => preserve({ source: candidate.id, thread: null, around: null })}>
                      {candidate.casevault_key || candidate.id}
                    </button>
                  </li>
                ))}
              </ul>
            </details>
          ) : null}
          <details className="text-xs text-muted-foreground">
            <summary className="cursor-pointer">Source identity and status detail</summary>
            <p className="mt-2 break-all">Import ID: {selected.id}</p>
            <p className="mt-1 break-all">Recorded location: {selected.casevault_key || "unavailable"}</p>
            <p className="mt-1">Device: {selected.device || "not recorded"}</p>
            <p className="mt-1">Source ownership is not the same as message authorship or first-/third-party classification.</p>
          </details>
        </div>
      ) : null}
    </section>
  );
}
