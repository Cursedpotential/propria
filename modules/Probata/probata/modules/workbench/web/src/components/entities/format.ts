// Byline: Claude Code · Opus 5.5 · 2026-09-25 (shared time label for the entities panel)

/** A stored UTC time in the viewer's local time; "no time" when absent. */
export function formatWhen(value: string | null | undefined) {
  if (!value) return "no time";
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString();
}
