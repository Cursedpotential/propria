// Byline: OpenAI Codex · GPT-6 · 2026-10-04
import { ExternalLink } from "lucide-react";
import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { caseRecordQuery, referenceLibraryQuery } from "@/lib/queries";
import { isUnavailable, type CaseRecordDetail, type LibraryRecord, type ReferenceLibraryPage, type StoreResponse } from "@/types/store";

const PAGE_SIZE = 25;
const BODY_FIELDS = ["definition", "text", "body", "content", "data", "description", "why", "citation"] as const;

/**
 * Reads a non-empty string from a loose shared-store record.
 * Inputs are an open record and a field name; output is a string or null.
 * It only inspects local response data and supports source-level or nested
 * provenance; use this instead of assuming every table shares one schema.
 * Byline: OpenAI Codex · GPT-6 · 2026-10-04
 */
function readText(record: Record<string, unknown>, key: string): string | null {
  const value = record[key];
  return typeof value === "string" && value.trim() ? value : null;
}

/**
 * Finds the first non-empty provenance field across a record and nested blocks.
 * Inputs are loose record candidates and preferred field names; output is a
 * string or null. It performs no I/O and supports both direct source rows and
 * reference rows carrying a nested source object.
 * Byline: OpenAI Codex · GPT-6 · 2026-10-04
 */
function firstRecordText(records: Array<Record<string, unknown> | null>, keys: string[]): string | null {
  for (const record of records) {
    if (!record) continue;
    for (const key of keys) {
      const value = readText(record, key);
      if (value) return value;
    }
  }
  return null;
}

/**
 * Serializes a library row for page-local search.
 * Input is one loaded row; output is searchable text. It only reads the current
 * bounded page and does not imply that matches exist elsewhere in the table.
 * Byline: OpenAI Codex · GPT-6 · 2026-10-04
 */
function searchableRow(row: LibraryRecord): string {
  return JSON.stringify(row).toLowerCase();
}

/**
 * Safely formats a stored body value for display without interpreting markup.
 * Input is an arbitrary record field; output is readable text or JSON. It has
 * no side effects and keeps shared source content inert in the desktop UI.
 * Byline: OpenAI Codex · GPT-6 · 2026-10-04
 */
function formatStoredValue(value: unknown): string {
  if (typeof value === "string") return value;
  const json = JSON.stringify(value, null, 2);
  return json ?? String(value);
}

/**
 * Opens a recorded local source path in the desktop shell.
 * Input is the stored path; output resolves after the Tauri opener completes.
 * It invokes the existing opener and may reject for unavailable paths; use an
 * HTTP(S) provenance link when the owner supplied a remote source URL.
 * Byline: OpenAI Codex · GPT-6 · 2026-10-04
 */
async function openLibraryPath(path: string): Promise<void> {
  const { openPath } = await import("@tauri-apps/plugin-opener");
  await openPath(path);
}

/**
 * Validates a stored source link for safe HTTP navigation.
 * Input is an arbitrary URL string; output is a normalized HTTP(S) URL or null.
 * It has no side effects and prevents non-web schemes from becoming links.
 * Byline: OpenAI Codex · GPT-6 · 2026-10-04
 */
function safeSourceUrl(value: string | null): string | null {
  if (!value) return null;
  try {
    const parsed = new URL(value);
    return parsed.protocol === "https:" || parsed.protocol === "http:" ? parsed.href : null;
  } catch {
    return null;
  }
}

/**
 * Renders paged source and reference tables with the canonical exact record body.
 * Inputs are bounded shared-store pages and a caseRecord response; output is a
 * native, read-only list/detail panel. Side effects are paged GETs and optional
 * source opening. Search filters only the loaded page. Use this for full source
 * citations and reference text; the neighboring ontology matcher remains separate.
 * Byline: OpenAI Codex · GPT-6 · 2026-10-04
 */
export function ReferenceLibrary() {
  const [table, setTable] = React.useState<"reference" | "source">("reference");
  const [offset, setOffset] = React.useState(0);
  const [searchInput, setSearchInput] = React.useState("");
  const [search, setSearch] = React.useState("");
  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [openError, setOpenError] = React.useState<string | null>(null);

  React.useEffect(() => {
    const timer = window.setTimeout(() => setSearch(searchInput.trim().toLowerCase()), 200);
    return () => window.clearTimeout(timer);
  }, [searchInput]);

  const pageQuery = useQuery(referenceLibraryQuery({ table, limit: PAGE_SIZE, offset }));
  const response: StoreResponse<ReferenceLibraryPage> | undefined = pageQuery.data;
  const page = response && !isUnavailable(response) ? response : null;
  const pageRows = page?.entries ?? [];
  const visibleRows = search ? pageRows.filter((row) => searchableRow(row).includes(search)) : pageRows;
  const selected = visibleRows.find((row) => row.id === selectedId) ?? visibleRows[0] ?? null;
  const detailQuery = useQuery(caseRecordQuery(selected?.id ?? null));
  const detailResponse: StoreResponse<CaseRecordDetail> | undefined = detailQuery.data;
  const detail = detailResponse && !isUnavailable(detailResponse) ? detailResponse : null;
  const record = detail?.record ?? null;
  const nestedSource = record?.source && typeof record.source === "object" ? record.source as Record<string, unknown> : null;
  const nestedData = record?.data && typeof record.data === "object" ? record.data as Record<string, unknown> : null;
  const provenanceRecords = [nestedSource, nestedData, record];
  const sourcePath = firstRecordText(provenanceRecords, ["source_path", "path"]);
  const sourceHash = firstRecordText(provenanceRecords, ["sha256", "sha", "hash"]);
  const sourceUrl = safeSourceUrl(firstRecordText(provenanceRecords, ["source_url", "official_url", "url"]));
  const r2Path = firstRecordText(provenanceRecords, ["r2_path"]);
  const bodyFields = record ? BODY_FIELDS.filter((field) => record[field] !== undefined && record[field] !== null) : [];

  /**
   * Changes the active shared table and resets paging and detail selection.
   * Input is an allowlisted table; output is void. It updates local view state
   * only and prevents carrying an offset or selected id across table siblings.
   * Byline: OpenAI Codex · GPT-6 · 2026-10-04
   */
  const changeTable = (nextTable: "reference" | "source") => {
    setTable(nextTable);
    setOffset(0);
    setSelectedId(null);
    setOpenError(null);
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle>Shared source and reference library</CardTitle>
          {page && <span className="text-xs tabular-nums text-text-tertiary">{page.total} {table} records</span>}
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex flex-wrap gap-2" role="group" aria-label="Library table">
          <Button variant={table === "reference" ? "solid" : "outline"} onClick={() => changeTable("reference")}>References</Button>
          <Button variant={table === "source" ? "solid" : "outline"} onClick={() => changeTable("source")}>Sources</Button>
        </div>
        <div>
          <Input
            value={searchInput}
            maxLength={120}
            onChange={(event) => setSearchInput(event.target.value)}
            placeholder="Search the loaded page"
            aria-label="Search the loaded library page"
          />
          <p className="mt-1 text-xs text-text-tertiary">Search filters these {pageRows.length} loaded rows only; use paging to inspect other records.</p>
        </div>

        {pageQuery.isPending ? (
          <p role="status" className="text-sm text-text-tertiary">Loading shared {table} records…</p>
        ) : pageQuery.isError ? (
          <div role="alert" className="rounded-[var(--radius-sm)] border border-critical-border bg-critical-fill p-3 text-sm text-critical-text">
            Could not load shared {table} records: {pageQuery.error.message}
            <Button className="ml-2" size="sm" onClick={() => void pageQuery.refetch()}>Retry</Button>
          </div>
        ) : response && isUnavailable(response) ? (
          <div role="alert" className="rounded-[var(--radius-sm)] border border-critical-border bg-critical-fill p-3 text-sm text-critical-text">
            Shared store unavailable: {response.reason}
          </div>
        ) : page ? (
          <>
            <div className="grid min-h-[360px] grid-cols-1 gap-3 lg:grid-cols-[minmax(240px,0.8fr)_minmax(0,1.5fr)]">
              <div className="min-h-0 space-y-1 overflow-y-auto rounded-[var(--radius-sm)] border border-border bg-bg p-1" aria-label={`${table} records`}>
                {visibleRows.length === 0 ? (
                  <p className="p-3 text-sm text-text-tertiary">{search ? "No matches in this loaded page." : `No ${table} records are available.`}</p>
                ) : visibleRows.map((row) => {
                  const title = readText(row, "title") ?? readText(row, "name") ?? readText(row, "label") ?? readText(row, "key") ?? readText(row, "pattern") ?? row.id;
                  const kind = readText(row, "kind") ?? readText(row, "category");
                  return (
                    <button
                      type="button"
                      key={row.id}
                      aria-current={row.id === selected?.id ? "true" : undefined}
                      onClick={() => { setSelectedId(row.id); setOpenError(null); }}
                      className={`block w-full rounded-[var(--radius-sm)] border px-3 py-2 text-left text-sm ${row.id === selected?.id ? "border-accent-border bg-accent-fill" : "border-transparent hover:bg-surface-hover"}`}
                    >
                      {kind && <Badge tone="accent">{kind}</Badge>}
                      <span className="mt-1 block line-clamp-2 text-text-primary">{title}</span>
                      <span className="mt-0.5 block truncate font-mono text-[11px] text-text-tertiary">{row.id}</span>
                    </button>
                  );
                })}
              </div>

              <section className="min-w-0 overflow-y-auto rounded-[var(--radius-sm)] border border-border bg-surface p-4" aria-label="Exact record detail" aria-live="polite">
                {!selected ? (
                  <p className="text-sm text-text-tertiary">Select a {table} record to load its full canonical body.</p>
                ) : detailQuery.isPending ? (
                  <p role="status" className="text-sm text-text-tertiary">Loading exact record and version…</p>
                ) : detailQuery.isError ? (
                  <p role="alert" className="text-sm text-critical-text">Could not load exact record {selected.id}: {detailQuery.error.message}</p>
                ) : detailResponse && isUnavailable(detailResponse) ? (
                  <p role="alert" className="text-sm text-critical-text">Shared store unavailable: {detailResponse.reason}</p>
                ) : detail ? (
                  <div className="space-y-4">
                    <div>
                      <div className="flex flex-wrap items-center gap-2">
                        <Badge tone="accent">{detail.table}</Badge>
                        <span className="break-all font-mono text-xs text-text-tertiary">{detail.id}</span>
                      </div>
                      <p className="mt-2 break-all text-xs text-text-secondary">{detail.contract} · {detail.version}</p>
                    </div>
                    {bodyFields.length > 0 ? bodyFields.map((field) => (
                      <div key={field}>
                        <h3 className="text-xs font-medium uppercase tracking-wide text-text-tertiary">{field}</h3>
                        <pre className="mt-1 whitespace-pre-wrap break-words font-sans text-sm leading-relaxed text-text-primary">{formatStoredValue(record?.[field])}</pre>
                      </div>
                    )) : <p className="text-sm text-text-tertiary">No dedicated body field is stored; the complete record is available below.</p>}

                    <div className="space-y-2 border-t border-border pt-3">
                      <h3 className="text-xs font-medium uppercase tracking-wide text-text-tertiary">Source provenance</h3>
                      {sourcePath && <div className="break-all text-xs"><span className="text-text-tertiary">Source path </span><code>{sourcePath}</code></div>}
                      {sourceHash && <div className="break-all text-xs"><span className="text-text-tertiary">SHA-256 </span><code>{sourceHash}</code></div>}
                      {sourceUrl && <a className="inline-flex items-center gap-1 text-sm text-accent-text underline underline-offset-2" href={sourceUrl} target="_blank" rel="noreferrer"><ExternalLink aria-hidden />Open source link</a>}
                      {r2Path && <div className="break-all text-xs"><span className="text-text-tertiary">R2 path </span><code>{r2Path}</code></div>}
                      {!sourcePath && !sourceHash && !sourceUrl && !r2Path && <p className="text-sm text-text-tertiary">No source provenance fields are recorded.</p>}
                      {sourcePath && <Button variant="outline" size="sm" onClick={() => {
                        setOpenError(null);
                        void openLibraryPath(sourcePath).catch((error: unknown) => setOpenError(error instanceof Error ? error.message : "Could not open the source path."));
                      }}>Open local source</Button>}
                      {openError && <p role="alert" className="text-sm text-critical-text">Could not open source: {openError}</p>}
                    </div>
                    <details className="border-t border-border pt-3">
                      <summary className="cursor-pointer text-sm text-accent-text">Full stored record</summary>
                      <pre className="mt-2 overflow-x-auto whitespace-pre-wrap break-words font-mono text-xs text-text-secondary">{JSON.stringify(record, null, 2)}</pre>
                    </details>
                  </div>
                ) : null}
              </section>
            </div>
            <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border pt-2 text-xs text-text-tertiary">
              <span>{page.total === 0 ? "0 records" : `${Math.min(offset + 1, page.total)}–${Math.min(offset + pageRows.length, page.total)} of ${page.total}`}</span>
              <div className="flex gap-2">
                <Button variant="outline" size="sm" disabled={offset === 0 || pageQuery.isFetching} onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}>Previous</Button>
                <Button variant="outline" size="sm" disabled={page.next_offset === null || pageQuery.isFetching} onClick={() => page.next_offset !== null && setOffset(page.next_offset)}>Next</Button>
              </div>
            </div>
          </>
        ) : null}
      </CardContent>
    </Card>
  );
}
