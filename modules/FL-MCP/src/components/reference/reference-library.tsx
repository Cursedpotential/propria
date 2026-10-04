// Byline: OpenAI Codex · GPT-6 · 2026-10-04
import { ExternalLink } from "lucide-react";
import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { referenceLibraryQuery } from "@/lib/queries";
import { isUnavailable, type StoreResponse, type ReferenceLibraryPage } from "@/types/store";

const PAGE_SIZE = 25;

/**
 * Reads an optional string from the reference's open provenance record.
 * Input is an unknown field value; output is a non-empty string or null. It
 * only inspects the in-memory row and keeps loose shared-store shapes safe.
 * Byline: OpenAI Codex · GPT-6 · 2026-10-04
 */
function sourceText(source: Record<string, unknown> | null | undefined, key: string): string | null {
  const value = source?.[key];
  return typeof value === "string" && value.trim() ? value : null;
}

/**
 * Opens a recorded local source path in the desktop shell. Input is the stored
 * path; output is a promise resolving after the opener call. It invokes the
 * existing Tauri opener and reports failures in the library; use source URLs
 * for browser-accessible sources instead.
 * Byline: OpenAI Codex · GPT-6 · 2026-10-04
 */
async function openReferencePath(path: string): Promise<void> {
  const { openPath } = await import("@tauri-apps/plugin-opener");
  await openPath(path);
}

/**
 * Renders the shared read-only reference list and selected record body.
 * Inputs come from the canonical shared-store page API; output is a paged
 * native list/detail view. Side effects are debounced API reads and optional
 * source-path opening. Choose this view for reference definitions; ontology
 * match hits remain in ReferenceView below. Original rows are never changed.
 * Byline: OpenAI Codex · GPT-6 · 2026-10-04
 */
export function ReferenceLibrary() {
  const [searchInput, setSearchInput] = React.useState("");
  const [query, setQuery] = React.useState("");
  const [offset, setOffset] = React.useState(0);
  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [openError, setOpenError] = React.useState<string | null>(null);
  React.useEffect(() => {
    const timer = window.setTimeout(() => {
      setQuery(searchInput.trim());
      setOffset(0);
    }, 250);
    return () => window.clearTimeout(timer);
  }, [searchInput]);

  const result = useQuery(referenceLibraryQuery({ q: query, limit: PAGE_SIZE, offset }));
  const response: StoreResponse<ReferenceLibraryPage> | undefined = result.data;
  const page = response && !isUnavailable(response) ? response : null;
  const entries = page?.entries ?? [];
  const selected = entries.find((row) => row.id === selectedId) ?? entries[0] ?? null;
  const selectedSource = selected?.source && typeof selected.source === "object" ? selected.source : null;
  const sourcePath = sourceText(selectedSource, "path") ?? sourceText(selectedSource, "source_path");
  const sourceHash = sourceText(selectedSource, "sha256") ?? sourceText(selectedSource, "hash");
  const sourceUrl = sourceText(selectedSource, "url") ?? sourceText(selectedSource, "source_url");
  const r2Path = sourceText(selectedSource, "r2_path");
  let safeUrl: string | null = null;
  if (sourceUrl) {
    try {
      const parsed = new URL(sourceUrl);
      if (parsed.protocol === "https:" || parsed.protocol === "http:") safeUrl = parsed.href;
    } catch {
      safeUrl = null;
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle>Shared reference library</CardTitle>
          {page && <span className="text-xs tabular-nums text-text-tertiary">{page.total} records</span>}
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        <Input
          value={searchInput}
          maxLength={120}
          onChange={(event) => setSearchInput(event.target.value)}
          placeholder="Search definitions, categories, aliases, or source metadata"
          aria-label="Search reference library"
        />
        {result.isPending ? (
          <p role="status" className="text-sm text-text-tertiary">Loading shared references…</p>
        ) : result.isError ? (
          <div role="alert" className="rounded-[var(--radius-sm)] border border-critical-border bg-critical-fill p-3 text-sm text-critical-text">
            Could not load the shared reference library: {result.error.message}
            <Button className="ml-2" size="sm" onClick={() => void result.refetch()}>Retry</Button>
          </div>
        ) : response && isUnavailable(response) ? (
          <div role="alert" className="rounded-[var(--radius-sm)] border border-critical-border bg-critical-fill p-3 text-sm text-critical-text">
            Shared reference store unavailable: {response.reason}
          </div>
        ) : page ? (
          <>
            <div className="grid min-h-[360px] grid-cols-1 gap-3 lg:grid-cols-[minmax(240px,0.8fr)_minmax(0,1.5fr)]">
              <div className="min-h-0 space-y-1 overflow-y-auto rounded-[var(--radius-sm)] border border-border bg-bg p-1" aria-label="Reference records">
                {entries.length === 0 ? (
                  <p className="p-3 text-sm text-text-tertiary">{query ? "No matching references." : "No references are available."}</p>
                ) : entries.map((row) => (
                  <button
                    type="button"
                    key={row.id}
                    aria-current={row.id === selected?.id ? "true" : undefined}
                    onClick={() => { setSelectedId(row.id); setOpenError(null); }}
                    className={`block w-full rounded-[var(--radius-sm)] border px-3 py-2 text-left text-sm ${row.id === selected?.id ? "border-accent-border bg-accent-fill" : "border-transparent hover:bg-surface-hover"}`}
                  >
                    <span className="flex flex-wrap items-center gap-1.5">
                      {row.category && <Badge tone="accent">{row.category}</Badge>}
                      {row.kind && <Badge tone="neutral">{row.kind}</Badge>}
                    </span>
                    <span className="mt-1 block line-clamp-2 text-text-primary">{row.pattern || row.id}</span>
                    <span className="mt-0.5 block truncate font-mono text-[11px] text-text-tertiary">{row.id}</span>
                  </button>
                ))}
              </div>

              <section className="min-w-0 overflow-y-auto rounded-[var(--radius-sm)] border border-border bg-surface p-4" aria-label="Reference detail" aria-live="polite">
                {selected ? (
                  <div className="space-y-4">
                    <div>
                      <div className="flex flex-wrap gap-1.5">
                        {selected.category && <Badge tone="accent">{selected.category}</Badge>}
                        {selected.kind && <Badge tone="neutral">{selected.kind}</Badge>}
                      </div>
                      <h3 className="mt-2 break-words text-base font-semibold text-text-primary">{selected.pattern || selected.id}</h3>
                      <p className="mt-1 break-all font-mono text-xs text-text-tertiary">{selected.id}</p>
                    </div>
                    <div>
                      <h4 className="text-xs font-medium uppercase tracking-wide text-text-tertiary">Definition</h4>
                      <p className="mt-1 whitespace-pre-wrap text-sm leading-relaxed text-text-primary">{typeof selected.definition === "string" && selected.definition ? selected.definition : "No definition body is stored for this record."}</p>
                    </div>
                    {Array.isArray(selected.aliases) && selected.aliases.length > 0 && (
                      <div>
                        <h4 className="text-xs font-medium uppercase tracking-wide text-text-tertiary">Aliases</h4>
                        <p className="mt-1 text-sm text-text-secondary">{selected.aliases.join(" · ")}</p>
                      </div>
                    )}
                    <div className="space-y-2 border-t border-border pt-3">
                      <h4 className="text-xs font-medium uppercase tracking-wide text-text-tertiary">Source provenance</h4>
                      {sourcePath && <div className="break-all text-xs"><span className="text-text-tertiary">Path </span><code>{sourcePath}</code></div>}
                      {sourceHash && <div className="break-all text-xs"><span className="text-text-tertiary">SHA-256 </span><code>{sourceHash}</code></div>}
                      {safeUrl && <a className="inline-flex items-center gap-1 text-sm text-accent-text underline underline-offset-2" href={safeUrl} target="_blank" rel="noreferrer"><ExternalLink aria-hidden />Open source link</a>}
                      {r2Path && <div className="break-all text-xs"><span className="text-text-tertiary">R2 path </span><code>{r2Path}</code></div>}
                      {!sourcePath && !sourceHash && !safeUrl && !r2Path && <p className="text-sm text-text-tertiary">No source provenance recorded.</p>}
                      {sourcePath && <Button variant="outline" size="sm" onClick={() => {
                        setOpenError(null);
                        void openReferencePath(sourcePath).catch((error: unknown) => setOpenError(error instanceof Error ? error.message : "Could not open the source path."));
                      }}>Open local source</Button>}
                      {openError && <p role="alert" className="text-sm text-critical-text">Could not open source: {openError}</p>}
                    </div>
                  </div>
                ) : <p className="text-sm text-text-tertiary">Select a reference record to read its definition.</p>}
              </section>
            </div>
            <div className="flex items-center justify-between border-t border-border pt-2 text-xs text-text-tertiary">
              <span>{page.total === 0 ? "0 records" : `${offset + 1}–${Math.min(offset + entries.length, page.total)} of ${page.total}`}</span>
              <div className="flex gap-2">
                <Button variant="outline" size="sm" disabled={offset === 0 || result.isFetching} onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}>Previous</Button>
                <Button variant="outline" size="sm" disabled={page.next_offset === null || result.isFetching} onClick={() => page.next_offset !== null && setOffset(page.next_offset)}>Next</Button>
              </div>
            </div>
          </>
        ) : null}
      </CardContent>
    </Card>
  );
}
