// Byline: Codex · 2026-10-06.
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { ErrorBox, Loading } from "@/components/mobile/mobile-ui";
import { graphReference } from "@/lib/graph-reference";
import { Button } from "@/components/ui/button";
import { getDiscoveryNeighbors, resolveSearchRelationships } from "@/lib/api-client";

/** Resolve an indexed source to completed graph snapshots and browse its recorded relationships.
 * Inputs: exact source/document IDs; outputs: explicit snapshot choices and bounded related records.
 * Effects: read-only resolve/neighborhood requests. Use for filesystem relationships, not entity analysis.
 */
export function SearchRelationships({ sourceId, documentId }: { sourceId: string; documentId: string }) {
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState<string | null>(null);
  const resolution = useQuery({
    queryKey: ["search-relationships", sourceId, documentId],
    queryFn: ({ signal }) => resolveSearchRelationships(sourceId, documentId, signal),
    enabled: open, retry: false,
  });
  const matches = resolution.data?.matches ?? [];
  const match = matches.find((item) => item.record_id === selected) ?? (matches.length === 1 ? matches[0] : null);
  return <div className="space-y-2">
    <Button type="button" size="sm" variant="outline" aria-expanded={open} onClick={() => setOpen((value) => !value)}>
      {open ? "Hide relationships" : "Explore file relationships"}
    </Button>
    {open ? <div className="space-y-2 border-l pl-3">
      {resolution.isPending ? <Loading /> : null}
      {resolution.isError ? <ErrorBox error={resolution.error} onRetry={() => void resolution.refetch()} /> : null}
      {resolution.data?.overflow ? <p className="text-xs">Too many matching snapshots; select a specific source version before exploring.</p> : null}
      {resolution.isSuccess && !resolution.data.overflow && !matches.length ? <p className="text-xs">No completed graph projection matches this source.</p> : null}
      {matches.length > 1 ? <label className="block text-xs">Choose a recorded snapshot
        <select className="ml-2 max-w-full rounded border bg-background p-1" value={selected ?? ""} onChange={(event) => setSelected(event.target.value)}>
          <option value="">Select snapshot</option>
          {matches.map((item) => <option key={item.record_id} value={item.record_id}>{item.version_id || "Unversioned source"} · {item.snapshot_key}</option>)}
        </select>
      </label> : null}
      {match ? <Neighborhood key={match.record_id} recordId={match.record_id} /> : null}
    </div> : null}
  </div>;
}

/** Browse one graph node at a time using typed references returned by the server.
 * Input: an authoritative record ID; output: readable node metadata and up to 50 edges.
 * Effects: bounded graph reads and local navigation. No graph edits or evidence promotion occur.
 */
function Neighborhood({ recordId }: { recordId: string }) {
  const [trail, setTrail] = useState<string[]>([recordId]);
  const current = trail[trail.length - 1];
  const reference = graphReference(current);
  const table = reference?.table ?? "";
  const key = reference?.key ?? "";
  const query = useQuery({
    queryKey: ["search-neighborhood", table, key],
    queryFn: ({ signal }) => getDiscoveryNeighbors(table, key, signal), retry: false,
    enabled: Boolean(reference),
  });
  const root = query.data?.root as Record<string, unknown> | undefined;
  const edges = (Array.isArray(query.data?.edges) ? query.data.edges : []) as { id: string; in: string; out: string; relation: string }[];
  const title = [root?.name, root?.source_path, root?.path, root?.label].find((value) => typeof value === "string") as string | undefined;
  return <div className="space-y-2 rounded border p-2 text-xs">
    <div className="flex items-center gap-2"><strong>{title || table.replaceAll("_", " ")}</strong>
      {trail.length > 1 ? <Button size="sm" variant="ghost" onClick={() => setTrail((items) => items.slice(0, -1))}>Back</Button> : null}</div>
    {!reference ? <p>The returned graph reference cannot be opened.</p> : query.isPending ? <Loading /> : null}
    {query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : null}
    {query.isSuccess && !edges.length ? <p>No related records were returned.</p> : null}
    <ul className="space-y-1">{edges.map((edge) => {
      const incoming = graphReference(edge.in);
      const outgoing = graphReference(edge.out);
      const next = incoming?.identity === reference?.identity ? outgoing : outgoing?.identity === reference?.identity ? incoming : null;
      if (!next || typeof edge.relation !== "string") return null;
      return <li key={edge.id}><button type="button" className="text-left underline underline-offset-4"
        onClick={() => setTrail((items) => [...items, next.identity])}>{edge.relation.replaceAll("_", " ")} → {next.table.replaceAll("_", " ")}</button></li>;
    })}</ul>
    {edges.length === 50 ? <p>Showing the first 50 relationships for this record.</p> : null}
    <details><summary className="cursor-pointer">Record reference</summary><p className="break-all">{current}</p></details>
  </div>;
}
