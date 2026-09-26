// Viewing standard for this workspace: imperial distances, Eastern time.
// The DB stores distances in METERS (working.event.distance_m_haversine,
// path_distance_m_haversine, working.waypoint.step_distance_m) and carries both UTC and
// pre-computed Eastern timestamps. Display always converts: meters -> mi/ft, and times
// come from the *_eastern columns, never from raw UTC.
//
// Byline: Claude Code · Opus 5 · 2026-07-25

const METERS_PER_MILE = 1609.344;
const METERS_PER_FOOT = 0.3048;

/** Meters -> imperial string. Short distances read better in feet. */
export function formatDistance(meters: number | null | undefined): string {
  if (meters === null || meters === undefined || Number.isNaN(Number(meters))) return '—';
  const m = Number(meters);
  if (m < 0) return '—';
  const miles = m / METERS_PER_MILE;
  if (miles < 0.1) return `${Math.round(m / METERS_PER_FOOT).toLocaleString()} ft`;
  if (miles < 10) return `${miles.toFixed(1)} mi`;
  return `${Math.round(miles).toLocaleString()} mi`;
}

/** Meters -> miles as a number, for charts/aggregates. */
export function metersToMiles(meters: number): number {
  return meters / METERS_PER_MILE;
}

/** Meters-per-second -> mph. */
export function formatSpeed(metersPerSecond: number | null | undefined): string {
  if (metersPerSecond === null || metersPerSecond === undefined) return '—';
  return `${(Number(metersPerSecond) * 2.236936).toFixed(1)} mph`;
}

/** Minutes -> compact duration ("3h 15m", "17m"). */
export function formatDuration(minutes: number | null | undefined): string {
  if (minutes === null || minutes === undefined || Number.isNaN(Number(minutes))) return '—';
  const total = Math.max(0, Math.round(Number(minutes)));
  const h = Math.floor(total / 60);
  const m = total % 60;
  return h ? `${h}h ${m}m` : `${m}m`;
}

/**
 * Eastern-time display. Values from the API's *_eastern columns are already wall-clock
 * Eastern with no offset, so they are formatted as-is — never re-zoned, which would
 * double-shift them.
 */
export function formatEastern(easternIso: string | null | undefined, withTime = true): string {
  if (!easternIso) return '—';
  const [date, time] = easternIso.split('T');
  if (!withTime) return date;
  return time ? `${date} ${time.slice(0, 5)}` : date;
}

export const TZ_LABEL = 'ET';
