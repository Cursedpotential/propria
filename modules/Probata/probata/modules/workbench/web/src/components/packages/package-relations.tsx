// Byline: Codex · GPT-6 · 2026-10-12.
import { PackageSources } from "@/components/packages/package-content";
import { Button } from "@/components/ui/button";
import type { ContextPackageItem, PackageEdge } from "@/lib/context-package-client";

/** Resolve the recorded endpoints of an edge without guessing its relation or direction.
 * Inputs: exact edge and owning item. Output: literal endpoint labels. Effects: none.
 * Pick for both SVG and tabular relationships; missing endpoint fields remain explicit.
 */
function endpoints(edge: PackageEdge, item: ContextPackageItem) {
  return { from: String(edge.from_node_id ?? edge.from ?? edge.source ?? item.id), to: String(edge.to_node_id ?? edge.to ?? edge.target ?? edge.target_ref ?? "Endpoint not returned"), relation: String(edge.relation ?? edge.type ?? edge.kind ?? "Relation not returned") };
}

/** Draw actual edges and expose their complete fields, direction and source basis.
 * Inputs: loaded section items, optional selected item and target selection callback.
 * Output: bounded graph rows plus complete edge inspector. Effects: selection only.
 * Pick for source, event, identity, timeline and strategy graphs; no inferred edges.
 */
export function PackageRelations({ items, selected, onSelect }: { items: ContextPackageItem[]; selected?: ContextPackageItem | null; onSelect: (id: string) => void }) {
  const owners = selected ? [selected] : items;
  const edges = owners.flatMap((item) => item.edges.map((edge, index) => ({ item, edge, key: `${item.id}:${edge.id ?? index}`, ...endpoints(edge, item) })));
  const ids = new Set(items.map((item) => item.id));
  return <section className="space-y-3 p-4" aria-label="Recorded relationships">
    <header><h2 className="text-base font-semibold">Relationships · {edges.length}</h2><p className="mt-1 text-xs text-muted-foreground">Actual edges in {selected ? "the selected item" : "this loaded section"}. Each edge retains its own fields and source basis.</p></header>
    {!edges.length && <p className="text-sm text-muted-foreground">No edges returned for {selected ? "this item" : "these loaded items"}.</p>}
    {edges.length > 0 && <div className="max-h-[25rem] overflow-auto rounded border" role="group" tabIndex={0} aria-label="Relationship graph">
      <svg role="img" aria-label={`${edges.length} recorded directed relationships`} viewBox={`0 0 900 ${Math.max(90, edges.length * 84)}`} className="min-w-[700px] w-full text-foreground">
        <defs><marker id="package-edge-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse"><path d="M0 0 L10 5 L0 10 z" fill="currentColor" /></marker></defs>
        {edges.map((entry, index) => {
          const y = index * 84 + 42;
          return <g key={entry.key}><title>{`${entry.from} → ${entry.relation} → ${entry.to}; direction: ${String(entry.edge.direction ?? "as recorded")}`}</title>
            <rect x="12" y={y - 20} width="245" height="40" rx="4" className="fill-muted stroke-border" />
            <text x="22" y={y + 4} className="fill-foreground text-[12px]">{entry.from.length > 32 ? `${entry.from.slice(0, 29)}…` : entry.from}</text>
            <line x1="267" y1={y} x2="623" y2={y} stroke="currentColor" markerEnd={entry.edge.direction === "in" || entry.edge.direction === "incoming" ? undefined : "url(#package-edge-arrow)"} markerStart={entry.edge.direction === "in" || entry.edge.direction === "incoming" ? "url(#package-edge-arrow)" : undefined} />
            <text x="445" y={y - 9} textAnchor="middle" className="fill-foreground text-[12px]">{entry.relation}</text>
            <rect x="635" y={y - 20} width="253" height="40" rx="4" className="fill-accent stroke-border" />
            <text x="645" y={y + 4} className="fill-foreground text-[12px]">{entry.to.length > 32 ? `${entry.to.slice(0, 29)}…` : entry.to}</text>
          </g>;
        })}
      </svg>
    </div>}
    <ol className="space-y-3">{edges.map((entry) => <li key={entry.key} className="rounded border p-3 text-xs">
      <p className="flex flex-wrap items-center gap-2 break-all"><strong>{entry.from}</strong><span>→ {entry.relation} →</span>
        {ids.has(entry.to) ? <Button variant="link" size="sm" onClick={() => onSelect(entry.to)}>{entry.to}</Button> : <strong>{entry.to}</strong>}</p>
      <p className="mt-1 text-muted-foreground">Direction: {String(entry.edge.direction ?? "as recorded")}</p>
      <details className="mt-2"><summary className="cursor-pointer">All edge fields and provenance</summary><pre className="mt-2 whitespace-pre-wrap break-all font-mono text-[11px]">{JSON.stringify(entry.edge, null, 2)}</pre></details>
      {Array.isArray(entry.edge.source_refs ?? entry.edge.source_pins) && <div className="mt-3"><PackageSources sources={(entry.edge.source_refs ?? entry.edge.source_pins) as ContextPackageItem["source_refs"]} /></div>}
    </li>)}</ol>
  </section>;
}

/** Show each native clock independently without coercing uncertainty into a precise instant.
 * Inputs: current section items and selection callback. Output: source-linked clock cards.
 * Effects: selection only. Pick for timeline mode on the same paged package items.
 */
export function PackageTimeline({ items, onSelect }: { items: ContextPackageItem[]; onSelect: (id: string) => void }) {
  const clocks = ["occurred_at", "event_time", "message_sent_at", "sent_at", "when_stated", "known_from", "source_available_from", "recorded_at", "start_time", "end_time", "temporal_precision", "timezone", "uncertainty"];
  return <div className="p-4"><p className="mb-3 text-xs text-muted-foreground">Source order as returned. Native event, message, knowledge and recording clocks remain separate.</p><ol className="space-y-3" aria-label="Package timeline">{items.map((item) => {
    const native = item.fields.native && typeof item.fields.native === "object" && !Array.isArray(item.fields.native) ? item.fields.native as Record<string, unknown> : {};
    const fields = [...Object.entries(item.fields), ...Object.entries(native).map(([key, value]) => [`native.${key}`, value] as const)].filter(([key]) => clocks.includes(key.replace(/^native\./, "")));
    return <li key={item.id} className="border-l-2 border-primary pl-3">
    <Button variant="link" className="h-auto whitespace-normal px-0 text-left" onClick={() => onSelect(item.id)}>{item.label}</Button>
    <dl className="my-2 space-y-1 text-xs">{fields.map(([key, value]) => <div key={key}><dt className="inline font-medium">{key.replaceAll("_", " ")}: </dt><dd className="inline whitespace-pre-wrap break-all">{typeof value === "string" ? value : JSON.stringify(value)}</dd></div>)}
      {!fields.length && <div className="text-muted-foreground">No native clock returned for this item.</div>}</dl>
    <PackageSources sources={item.source_refs} />
  </li>; })}</ol></div>;
}
