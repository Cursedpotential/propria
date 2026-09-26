// Shared SQL filter builder for the events read-path.
// Both /api/events (windowed page) and /api/events/bounds (corpus extent) build their
// WHERE clause from here, so the "total" reported by bounds can never drift from the
// rows returned by the page — that honesty is the whole point of the windowed design.
//
// Byline: Claude Code · Opus 5 · 2026-07-25

export interface BuiltFilter {
  where: string;
  args: unknown[];
}

export interface FilterOpts {
  /** Include the dateFrom/dateTo window. False for bounds (we want the full extent). */
  includeDateWindow: boolean;
}

export function buildEventFilter(
  p: URLSearchParams,
  { includeDateWindow }: FilterOpts,
): BuiltFilter {
  const where: string[] = ['e.lat IS NOT NULL'];
  const args: unknown[] = [];
  const add = (clause: string, ...vals: unknown[]) => {
    // Replace each $? with the next positional placeholder, so a clause may bind
    // the same value more than once (the query filter does).
    let out = clause;
    for (const v of vals) {
      args.push(v);
      out = out.replace('$?', `$${args.length}`);
    }
    where.push(out);
  };

  if (includeDateWindow) {
    // start_eastern is `timestamp without time zone`. Cast the bound to ::date and use a
    // half-open upper bound instead of string-concatenating 'T23:59:59' — that concat
    // produced a `timestamp <= text` comparison, which Postgres rejects outright.
    if (p.get('dateFrom')) add('e.start_eastern >= $?::date', p.get('dateFrom'));
    if (p.get('dateTo')) add("e.start_eastern < ($?::date + INTERVAL '1 day')", p.get('dateTo'));
  }

  const types = p.getAll('eventType');
  if (types.length) add('e.event_type = ANY($?)', types);
  if (p.get('overnight') === 'true') where.push('e.overnight_simple');
  if (p.get('query')) {
    const q = p.get('query');
    add(
      "(e.serial_display ILIKE '%' || $? || '%' OR e.place_id ILIKE '%' || $? || '%')",
      q,
      q,
    );
  }
  const minProb = p.get('minProbability');
  if (minProb) add('e.probability >= $?', Number(minProb));
  const tags = p.getAll('tag');
  if (tags.length) add('kp.tags && $?', tags);

  return { where: where.join(' AND '), args };
}

/** LEFT JOIN used by both routes (tag filtering needs it even when not selecting tags). */
export const KNOWN_PLACE_JOIN = `
  LEFT JOIN ref.known_place kp
    ON (kp.place_id = e.place_id)
    OR (kp.place_id IS NULL AND kp.lat_r4 = e.lat_r4 AND kp.lng_r4 = e.lng_r4)`;

/**
 * Hard ceiling on rows returned in one window. The full corpus is ~20,160 events
 * (~7.9 MB JSON, ~1.9 s), so this sits just above "everything" — a window may be
 * widened to the entire record, but never silently return more than this without
 * the response saying so via `truncated`.
 */
export const MAX_WINDOW_ROWS = 25000;

/** Default window span when the client hasn't chosen one yet. */
export const DEFAULT_WINDOW_MONTHS = 3;
