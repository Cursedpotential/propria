// Byline: Codex · 2026-10-06.
import { Loader2, Search } from "lucide-react";
import { Button } from "@/components/ui/button";

/** Filter the browsed storage location by its recorded names and paths.
 * Inputs: controlled query and callbacks; output: a filename filter form.
 * Effects: calls the existing server-scoped listing filter. Full-text/vector/graph
 * retrieval uses the shared ContextSearch component beside the file browser.
 */
export function SourceSearch({ query, searching, onQueryChange, onSubmit, onClear, resultSummary }: {
  query: string; searching: boolean; onQueryChange: (value: string) => void;
  onSubmit: () => void; onClear: () => void; resultSummary: string | null;
}) {
  return <form className="flex flex-wrap items-center gap-2 border-b bg-card px-3 py-2"
    aria-label="Filter file names and paths" onSubmit={(event) => { event.preventDefault(); onSubmit(); }}>
    <label className="text-xs font-semibold" htmlFor="source-name-filter">File names and paths</label>
    <input id="source-name-filter" className="h-8 min-w-0 flex-1 border bg-background px-2 text-xs"
      value={query} placeholder="Find a file in this location" onChange={(event) => onQueryChange(event.target.value)} />
    <Button type="submit" size="sm" disabled={searching || !query.trim()}>
      {searching ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Search className="h-3.5 w-3.5" />} Find file
    </Button>
    {query || resultSummary ? <Button type="button" size="sm" variant="outline" onClick={onClear}>Clear</Button> : null}
    {resultSummary ? <span className="text-[11px] text-muted-foreground" role="status">{resultSummary}</span> : null}
  </form>;
}
