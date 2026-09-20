// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { factorMapQuery, referenceQuery } from "@/lib/queries";
import { isMatchHits } from "@/types/store";

export function ReferenceView() {
  const [match, setMatch] = React.useState("");
  const factorMap = useQuery(factorMapQuery());
  const reference = useQuery(referenceQuery(match || undefined));

  return (
    <div className="space-y-3">
      <Card>
        <CardHeader>
          <CardTitle>MCL 722.23 factor map</CardTitle>
        </CardHeader>
        <CardContent>
          {factorMap.data && "entries" in factorMap.data ? (
            <ul className="grid grid-cols-1 gap-1.5 text-sm sm:grid-cols-2">
              {factorMap.data.entries.map((f) => (
                <li key={f.letter} className="flex items-center justify-between gap-2 rounded-[var(--radius-sm)] border border-border px-2 py-1">
                  <span>
                    <span className="mr-1.5 font-mono text-text-tertiary">({f.letter})</span>
                    {f.title}
                  </span>
                  <span className="flex shrink-0 gap-1">
                    <Badge tone="good">{f.support_count}</Badge>
                    <Badge tone="critical">{f.contradict_count}</Badge>
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-sm text-text-tertiary">Loading…</p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Behavior patterns / ontology</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <Input value={match} onChange={(e) => setMatch(e.target.value)} placeholder="Match text against reference ontology (case_reference match)…" />
          {reference.data && "entries" in reference.data ? (
            reference.data.entries.length === 0 ? (
              <p className="text-sm text-text-tertiary">{match ? "No matches." : "No reference rows loaded yet."}</p>
            ) : isMatchHits(reference.data.entries) ? (
              <ul className="space-y-1 text-sm">
                {reference.data.entries.map((hit, i) => (
                  <li key={`${hit.id}-${i}`} className="rounded-[var(--radius-sm)] border border-border px-2 py-1.5">
                    <div className="flex items-center gap-2">
                      {hit.category && <Badge tone="accent">{hit.category}</Badge>}
                      {hit.kind && <Badge tone="neutral">{hit.kind}</Badge>}
                      <Badge tone="neutral">{hit.via}</Badge>
                    </div>
                    <p className="mt-0.5 font-mono text-xs text-text-secondary">
                      …matched <strong className="text-text-primary">"{hit.matched}"</strong> at [{hit.span[0]}, {hit.span[1]}]
                    </p>
                  </li>
                ))}
              </ul>
            ) : (
              <ul className="space-y-1 text-sm">
                {reference.data.entries.map((row) => (
                  <li key={row.id} className="rounded-[var(--radius-sm)] border border-border px-2 py-1.5">
                    <div className="flex items-center gap-2">
                      {row.category && <Badge tone="accent">{row.category}</Badge>}
                      {row.kind && <Badge tone="neutral">{row.kind}</Badge>}
                    </div>
                    {row.pattern && <p className="mt-0.5 font-mono text-xs text-text-tertiary">/{row.pattern}/</p>}
                  </li>
                ))}
              </ul>
            )
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
