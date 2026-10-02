import { NextRequest, NextResponse } from 'next/server';
import { pool } from '@/lib/db';
import { buildEventFilter, KNOWN_PLACE_JOIN } from '@/lib/eventFilters';

// Corpus extent for the current filters, IGNORING the date window.
// The UI always shows these numbers ("20,160 events · 2017-09-20 → 2024-08-05") even
// when only a 3-month window is loaded, so a windowed view can never be mistaken for
// the whole record. Read-only.
//
// Byline: Claude Code · Opus 5 · 2026-07-25
export async function GET(req: NextRequest) {
  const { where, args } = buildEventFilter(req.nextUrl.searchParams, {
    includeDateWindow: false,
  });

  const sql = `
    SELECT count(*)::int AS total,
           to_char(min(e.start_eastern), 'YYYY-MM-DD') AS min_date,
           to_char(max(e.start_eastern), 'YYYY-MM-DD') AS max_date
    FROM analysis.latest_events e
    ${KNOWN_PLACE_JOIN}
    WHERE ${where}`;

  try {
    const { rows } = await pool.query(sql, args);
    const r = rows[0];
    return NextResponse.json({
      total: r.total ?? 0,
      min_date: r.min_date ?? null,
      max_date: r.max_date ?? null,
    });
  } catch (err) {
    return NextResponse.json(
      { error: String(err), total: 0, min_date: null, max_date: null },
      { status: 500 },
    );
  }
}
