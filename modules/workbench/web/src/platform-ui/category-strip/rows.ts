// Byline: Claude Code · Opus 5.5 · 2026-09-24
// Data side of the category strip: the row/block/segment shape, a grouper that builds it from flat records (any query
// result), category totals and color assignment.

export interface StripSegment {
  category: string;
  weight: number;
  title?: string;
}

export interface StripBlock {
  id: string;
  segments: StripSegment[];
  /** Block width; defaults to the sum of its segment weights. */
  weight?: number;
  title?: string;
}

export interface StripRow {
  id: string;
  label: string;
  blocks: StripBlock[];
  /** Right-hand number; defaults to the sum of segment weights. */
  total?: number;
}

/** Categorical palette (Tableau 10 + 2), cycled; categories past the palette get evenly spread hues. */
const PALETTE = [
  "#4e79a7", "#f28e2b", "#e15759", "#76b7b2", "#59a14f", "#edc948",
  "#b07aa1", "#ff9da7", "#9c755f", "#bab0ac", "#86bcb6", "#d37295",
];

export function categoryColors(categories: readonly string[], fixed: Record<string, string> = {}): Record<string, string> {
  const out: Record<string, string> = {};
  let next = 0;
  for (const c of categories) {
    if (fixed[c]) {
      out[c] = fixed[c];
      continue;
    }
    out[c] = next < PALETTE.length ? PALETTE[next] : `hsl(${Math.round((next * 137.508) % 360)} 55% 55%)`;
    next += 1;
  }
  return out;
}

/** Every category with its total weight: `order` first, then the rest largest first. */
export function categoryTotals(rows: readonly StripRow[], order: readonly string[] = []): [string, number][] {
  const totals = new Map<string, number>();
  for (const row of rows)
    for (const block of row.blocks)
      for (const seg of block.segments) totals.set(seg.category, (totals.get(seg.category) ?? 0) + seg.weight);
  const rank = (c: string) => (order.includes(c) ? order.indexOf(c) : Number.POSITIVE_INFINITY);
  return [...totals.entries()].sort(([a, x], [b, y]) => rank(a) - rank(b) || y - x || a.localeCompare(b));
}

export interface RecordKeys<T> {
  row: (r: T) => string;
  block: (r: T) => string;
  category: (r: T) => string;
  weight?: (r: T) => number;
  rowLabel?: (r: T) => string;
  blockTitle?: (r: T) => string;
  segmentTitle?: (r: T) => string;
}

/**
 * Build strip rows from flat records, one record per segment, in the order they arrive. Rows are sorted by id;
 * blocks and segments keep record order. Any query returning (row, block, category, weight) can feed the strip.
 */
export function rowsFromRecords<T>(records: readonly T[], keys: RecordKeys<T>): StripRow[] {
  const rows = new Map<string, StripRow>();
  const blocks = new Map<string, StripBlock>();
  for (const r of records) {
    const rowId = keys.row(r);
    let row = rows.get(rowId);
    if (!row) {
      row = { id: rowId, label: keys.rowLabel?.(r) ?? rowId, blocks: [] };
      rows.set(rowId, row);
    }
    const blockKey = `${rowId}\u0000${keys.block(r)}`;
    let block = blocks.get(blockKey);
    if (!block) {
      block = { id: keys.block(r), segments: [], title: keys.blockTitle?.(r) };
      blocks.set(blockKey, block);
      row.blocks.push(block);
    }
    block.segments.push({ category: keys.category(r), weight: keys.weight?.(r) ?? 1, title: keys.segmentTitle?.(r) });
  }
  return [...rows.values()].sort((a, b) => a.id.localeCompare(b.id));
}
