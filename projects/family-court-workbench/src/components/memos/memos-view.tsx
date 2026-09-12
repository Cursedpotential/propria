// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { marked } from "marked";
import * as React from "react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { CaseMemoRecord } from "@/types/store";

const KIND_TONE: Record<string, "accent" | "warn" | "critical" | "good" | "neutral"> = {
  analysis: "accent",
  strategy: "good",
  weakness: "critical",
  direction: "warn",
};

export function MemosView({ memos }: { memos: CaseMemoRecord[] }) {
  const [selectedId, setSelectedId] = React.useState<string | null>(memos[0]?.id ?? null);
  const selected = memos.find((m) => m.id === selectedId) ?? null;

  // Latest per kind: memos not superseded by another memo of the same kind.
  const supersededIds = new Set(memos.map((m) => m.supersedes).filter(Boolean) as string[]);
  const latestByKind = memos.filter((m) => !supersededIds.has(m.id));

  if (memos.length === 0) {
    return <p className="text-sm text-text-tertiary">No memos on record yet.</p>;
  }

  return (
    <div className="grid h-full grid-cols-[280px_1fr] gap-3">
      <div className="space-y-2 overflow-y-auto">
        {latestByKind.map((memo) => (
          <button
            key={memo.id}
            onClick={() => setSelectedId(memo.id)}
            className={
              "block w-full rounded-[var(--radius-md)] border px-3 py-2 text-left text-sm " +
              (memo.id === selectedId ? "border-accent-border bg-accent-fill" : "border-border bg-surface hover:bg-surface-hover")
            }
          >
            <div className="flex items-center justify-between gap-2">
              <span className="truncate font-medium text-text-primary">{memo.title}</span>
              <Badge tone={KIND_TONE[memo.kind] ?? "neutral"}>{memo.kind}</Badge>
            </div>
            <div className="mt-0.5 font-mono text-xs tabular-nums text-text-tertiary">{memo.created_at}</div>
          </button>
        ))}
      </div>

      <Card className="min-h-0">
        <CardHeader>
          <CardTitle>{selected?.title ?? "Select a memo"}</CardTitle>
        </CardHeader>
        <CardContent className="prose prose-sm max-w-none overflow-y-auto text-text-primary">
          {selected ? (
            // Safety note: memo bodies are owner-authored markdown from the local case store, not user-supplied HTML from an untrusted source.
            <div dangerouslySetInnerHTML={{ __html: marked.parse(selected.text, { async: false }) as string }} />
          ) : (
            <p className="text-sm text-text-tertiary">No memo selected.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
