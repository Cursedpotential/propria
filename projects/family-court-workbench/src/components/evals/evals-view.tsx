// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { marked } from "marked";
import * as React from "react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { EvalRecord } from "@/types/store";

function verdictTone(verdict: string | null): "good" | "critical" | "warn" | "neutral" {
  const v = (verdict ?? "").toLowerCase();
  if (["pass", "verified", "ok", "good"].includes(v)) return "good";
  if (["fail", "failed", "bad", "rejected"].includes(v)) return "critical";
  if (v) return "warn";
  return "neutral";
}

export function EvalsView({ evals }: { evals: EvalRecord[] }) {
  const [selectedId, setSelectedId] = React.useState<string | null>(evals[0]?.id ?? null);
  const selected = evals.find((e) => e.id === selectedId) ?? null;

  if (evals.length === 0) {
    return <p className="text-sm text-text-tertiary">No evals on record yet.</p>;
  }

  return (
    <div className="grid h-full grid-cols-[280px_1fr] gap-3">
      <div className="space-y-2 overflow-y-auto">
        {evals.map((ev) => (
          <button
            key={ev.id}
            onClick={() => setSelectedId(ev.id)}
            className={
              "block w-full rounded-[var(--radius-md)] border px-3 py-2 text-left text-sm " +
              (ev.id === selectedId ? "border-accent-border bg-accent-fill" : "border-border bg-surface hover:bg-surface-hover")
            }
          >
            <div className="flex items-center justify-between gap-2">
              <span className="truncate font-medium text-text-primary">{ev.title}</span>
              {ev.verdict && <Badge tone={verdictTone(ev.verdict)}>{ev.verdict}</Badge>}
            </div>
            <div className="mt-0.5 flex items-center gap-2 text-xs text-text-tertiary">
              <span className="font-mono tabular-nums">{ev.created_at}</span>
              {ev.score !== null && <span className="tabular-nums">score {ev.score}</span>}
            </div>
          </button>
        ))}
      </div>
      <Card className="min-h-0">
        <CardHeader>
          <CardTitle>{selected?.title ?? "Select an eval"}</CardTitle>
        </CardHeader>
        <CardContent className="prose prose-sm max-w-none overflow-y-auto text-text-primary">
          {selected ? (
            selected.text ? (
              // Safety note: eval bodies are owner-authored markdown/text from the local case store.
              <div dangerouslySetInnerHTML={{ __html: marked.parse(selected.text, { async: false }) as string }} />
            ) : (
              <p className="text-sm text-text-tertiary">No detail text recorded for this eval.</p>
            )
          ) : (
            <p className="text-sm text-text-tertiary">No eval selected.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
