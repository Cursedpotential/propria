import { NextRequest, NextResponse } from 'next/server';
import { pool } from '@/lib/db';
import { buildEventFilter, KNOWN_PLACE_JOIN, MAX_WINDOW_ROWS } from '@/lib/eventFilters';

// Real-data read path over analysis.latest_events (deduped, corroboration-safe).
// Reflects ADR-0005/0007: the app reads the deterministic derived layer, never raw.
//
// WINDOWED: returns only the requested date window, but always reports `total_in_window`
// (the true match count for this window, computed before LIMIT) and `truncated`, so the
// client can never present a partial timeline as complete. Corpus-wide totals come from
// /api/events/bounds.
export async function GET(req: NextRequest) {
  const p = req.nextUrl.searchParams;
  const { where, args } = buildEventFilter(p, { includeDateWindow: true });

  const requested = Number(p.get('limit') ?? MAX_WINDOW_ROWS);
  const limit = Math.min(
    Number.isFinite(requested) && requested > 0 ? requested : MAX_WINDOW_ROWS,
    MAX_WINDOW_ROWS,
  );
  const offsetRaw = Number(p.get('offset') ?? 0);
  const offset = Number.isFinite(offsetRaw) && offsetRaw > 0 ? Math.floor(offsetRaw) : 0;

  // count(*) OVER () is evaluated across the full filtered set before LIMIT/OFFSET,
  // so total_in_window is the honest match count even when we return fewer rows.
  const sql = `
    SELECT e.event_id::text, e.serial_display, e.raw_type, e.event_type,
           to_char(e.start_eastern, 'YYYY-MM-DD"T"HH24:MI:SS') AS start_eastern,
           to_char(e.end_eastern,   'YYYY-MM-DD"T"HH24:MI:SS') AS end_eastern,
           to_char(e.start_utc, 'YYYY-MM-DD"T"HH24:MI:SSOF') AS start_utc,
           to_char(e.end_utc,   'YYYY-MM-DD"T"HH24:MI:SSOF') AS end_utc,
           e.lat, e.lng, e.lat_r4, e.lng_r4, e.place_id,
           COALESCE(kp.tags, '{}') AS tags,
           e.overnight_simple, e.probability,
           count(*) OVER ()::int AS total_in_window
    FROM analysis.latest_events e
    ${KNOWN_PLACE_JOIN}
    WHERE ${where}
    ORDER BY e.start_utc
    LIMIT ${limit} OFFSET ${offset}`;

  try {
    const { rows } = await pool.query(sql, args);
    const totalInWindow = rows.length ? Number(rows[0].total_in_window) : 0;
    const events = rows.map(r => ({
      event_id: r.event_id,
      serial_display: r.serial_display,
      raw_type: r.raw_type,
      event_type: r.event_type,
      start_eastern: r.start_eastern,
      end_eastern: r.end_eastern,
      start_utc: r.start_utc,
      end_utc: r.end_utc,
      lat: Number(r.lat), lng: Number(r.lng),
      lat_r4: Number(r.lat_r4), lng_r4: Number(r.lng_r4),
      place_id: r.place_id ?? '',
      tags: r.tags ?? [],
      overnight_simple: r.overnight_simple ? 'overnight' : 'daytrip',
      probability: r.probability === null ? 0 : Number(r.probability),
    }));
    return NextResponse.json({
      events,
      count: events.length,
      total_in_window: totalInWindow,
      limit,
      offset,
      truncated: offset + events.length < totalInWindow,
    });
  } catch (err) {
    return NextResponse.json(
      { error: String(err), events: [], count: 0, total_in_window: 0, truncated: false },
      { status: 500 },
    );
  }
}
