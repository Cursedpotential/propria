// Byline: Claude Code · Opus 5.5 · 2026-09-25
// Numbers for the summary header, computed from the same rows the strip draws.

import type { StripRow } from "./rows";
import { schemeValue, type Scheme } from "./schemes";

export interface StripTotals {
  rows: number;
  blocks: number;
  weight: number;
  byCategory: [string, number][];
  /** Diverging schemes only: weight on each side of neutral, plus words the scheme does not know. */
  balance?: { negative: number; neutral: number; positive: number; unmapped: number };
  unmapped: string[];
}

export function summarizeRows(rows: readonly StripRow[], scheme?: Scheme): StripTotals {
  const byCategory = new Map<string, number>();
  let blocks = 0;
  for (const row of rows) {
    blocks += row.blocks.length;
    for (const block of row.blocks)
      for (const seg of block.segments) byCategory.set(seg.category, (byCategory.get(seg.category) ?? 0) + seg.weight);
  }
  const entries = [...byCategory.entries()].sort(([, a], [, b]) => b - a);
  const weight = entries.reduce((sum, [, w]) => sum + w, 0);
  const known = scheme && scheme.kind !== "categorical";
  const unmapped = known ? entries.filter(([c]) => schemeValue(scheme, c) === undefined).map(([c]) => c) : [];
  let balance: StripTotals["balance"];
  if (scheme?.kind === "diverging") {
    balance = { negative: 0, neutral: 0, positive: 0, unmapped: 0 };
    for (const [c, w] of entries) {
      const v = schemeValue(scheme, c);
      if (v === undefined) balance.unmapped += w;
      else if (v < 0) balance.negative += w;
      else if (v > 0) balance.positive += w;
      else balance.neutral += w;
    }
  }
  return { rows: rows.length, blocks, weight, byCategory: entries, balance, unmapped };
}
