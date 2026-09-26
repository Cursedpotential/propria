import { NextRequest, NextResponse } from 'next/server';
import { pool } from '@/lib/db';

// Owner write-path for ref.known_place only (never raw/working).
// Mirrors ops/flag.py semantics: upsert by place_id OR lat_r4/lng_r4 key.
async function upsertKnownPlace(
  place_id: string | null,
  lat_r4: number | null,
  lng_r4: number | null,
  label: string,
  tagList: string[],
  concerning: boolean,
  watch: boolean,
  severity: number | null,
  note: string | null,
): Promise<number> {
  // We cannot use a stable unique constraint across both place_id and lat/lng keys,
  // so we perform a conditional upsert: look up existing row, then update or insert.
  if (place_id) {
    const existing = await pool.query(
      `SELECT known_place_id FROM ref.known_place WHERE place_id = $1 LIMIT 1`,
      [place_id],
    );
    if (existing.rows.length) {
      const id = existing.rows[0].known_place_id;
      await pool.query(
        `UPDATE ref.known_place
         SET label = $2, tags = $3, is_concerning = $4, is_watch = $5,
             severity = $6, note = $7, updated_at = NOW()
         WHERE known_place_id = $1`,
        [id, label, tagList, concerning, watch, severity, note],
      );
      return id;
    }
    const { rows } = await pool.query(
      `INSERT INTO ref.known_place
        (place_id, lat_r4, lng_r4, label, tags, is_concerning, is_watch,
         severity, note, created_by, created_at, updated_at)
       VALUES ($1, NULL, NULL, $2, $3, $4, $5, $6, $7, 'ui', NOW(), NOW())
       RETURNING known_place_id`,
      [place_id, label, tagList, concerning, watch, severity, note],
    );
    return rows[0].known_place_id;
  }
  // No place_id: key on rounded lat/lng (matches analysis events join).
  const existing = await pool.query(
    `SELECT known_place_id FROM ref.known_place
     WHERE place_id IS NULL AND lat_r4 = $1 AND lng_r4 = $2
     LIMIT 1`,
    [lat_r4, lng_r4],
  );
  if (existing.rows.length) {
    const id = existing.rows[0].known_place_id;
    await pool.query(
      `UPDATE ref.known_place
       SET label = $2, tags = $3, is_concerning = $4, is_watch = $5,
           severity = $6, note = $7, updated_at = NOW()
       WHERE known_place_id = $1`,
      [id, label, tagList, concerning, watch, severity, note],
    );
    return id;
  }
  const { rows } = await pool.query(
    `INSERT INTO ref.known_place
      (place_id, lat_r4, lng_r4, label, tags, is_concerning, is_watch,
       severity, note, created_by, created_at, updated_at)
     VALUES (NULL, $1, $2, $3, $4, $5, $6, $7, $8, 'ui', NOW(), NOW())
     RETURNING known_place_id`,
    [lat_r4, lng_r4, label, tagList, concerning, watch, severity, note],
  );
  return rows[0].known_place_id;
}

export async function POST(req: NextRequest) {
  let body: any;
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: 'Invalid JSON body' }, { status: 400 });
  }

  const {
    place_id,
    lat_r4,
    lng_r4,
    label,
    tags,
    is_concerning,
    is_watch,
    severity,
    note,
  } = body || {};

  if (!label || typeof label !== 'string') {
    return NextResponse.json({ error: 'label is required' }, { status: 400 });
  }
  if (!place_id && (lat_r4 === undefined || lng_r4 === undefined)) {
    return NextResponse.json(
      { error: 'Either place_id or both lat_r4 and lng_r4 are required' },
      { status: 400 },
    );
  }

  const tagList = Array.isArray(tags)
    ? tags.filter((t): t is string => typeof t === 'string' && t.trim().length > 0)
    : [];

  const safeBool = (v: unknown) => v === true || v === 'true';
  const safeSeverity =
    severity === undefined || severity === null
      ? null
      : Math.max(1, Math.min(5, Math.round(Number(severity)))) || null;

  try {
    const known_place_id = await upsertKnownPlace(
      (place_id as string | undefined) || null,
      lat_r4 === undefined ? null : Number(lat_r4),
      lng_r4 === undefined ? null : Number(lng_r4),
      label.trim(),
      tagList,
      safeBool(is_concerning),
      safeBool(is_watch),
      safeSeverity,
      note === undefined || note === null ? null : String(note).trim() || null,
    );

    return NextResponse.json({
      ok: true,
      known_place_id,
      place_id: place_id || null,
      lat_r4: lat_r4 === undefined ? null : Number(lat_r4),
      lng_r4: lng_r4 === undefined ? null : Number(lng_r4),
      label: label.trim(),
      tags: tagList,
      is_concerning: safeBool(is_concerning),
      is_watch: safeBool(is_watch),
      severity: safeSeverity,
      note,
    });
  } catch (err) {
    return NextResponse.json({ error: String(err) }, { status: 500 });
  }
}
