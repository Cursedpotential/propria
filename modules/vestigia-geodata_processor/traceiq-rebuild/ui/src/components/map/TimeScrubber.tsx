"use client";
import { useWorkspace } from '@/lib/WorkspaceContext';

/**
 * Window selector. The track spans the ENTIRE corpus (from bounds), not just the loaded
 * window — otherwise you could never scrub outside what happens to be loaded, which is
 * exactly how the old 2,000-row cap made 2019-2024 unreachable.
 *
 * Byline: Claude Code · Opus 5 · 2026-07-25
 */
export function TimeScrubber() {
  const { filters, setFilters, bounds, windowMeta, loading } = useWorkspace();

  const min = bounds.min_date;
  const max = bounds.max_date;
  if (!min || !max) return null;

  const from = filters.dateFrom || min;
  const to = filters.dateTo || max;
  const span = daysBetween(min, max);

  // Keep the handles from crossing each other.
  const setFrom = (days: number) => {
    const next = addDays(min, days);
    setFilters({ ...filters, dateFrom: next > to ? to : next });
  };
  const setTo = (days: number) => {
    const next = addDays(min, days);
    setFilters({ ...filters, dateTo: next < from ? from : next });
  };

  const windowDays = daysBetween(from, to);
  const shown = windowMeta.totalInWindow.toLocaleString();

  return (
    <div className="absolute bottom-3 left-3 right-3 z-20 bg-surface/95 backdrop-blur border border-border rounded-[var(--r)] shadow-[var(--shadow)] px-4 py-3">
      <div className="flex items-center justify-between mb-2">
        <span className="text-[10px] uppercase tracking-wide font-mono text-muted">Window</span>
        <span className="text-[10px] font-mono text-faint">
          {loading ? 'loading…' : `${shown} events · ${formatSpan(windowDays)}`}
        </span>
      </div>

      <div className="flex items-center gap-3">
        <span className="text-[11px] font-mono text-ink whitespace-nowrap tabular-nums">{from}</span>

        {/* Dual overlaid ranges: the inputs ignore pointer events so BOTH thumbs stay
            grabbable even when they overlap; only the thumbs accept the pointer. */}
        <div className="flex-1 relative h-8">
          <div className="absolute left-0 right-0 top-1/2 -translate-y-1/2 h-1.5 rounded-full bg-surface-2 border border-border" />
          <div
            className="absolute top-1/2 -translate-y-1/2 h-1.5 rounded-full bg-signal/40"
            style={{
              left: `${(daysBetween(min, from) / Math.max(1, span)) * 100}%`,
              right: `${100 - (daysBetween(min, to) / Math.max(1, span)) * 100}%`,
            }}
          />
          <input
            type="range" aria-label="Window start"
            min={0} max={span} value={daysBetween(min, from)}
            onChange={e => setFrom(Number(e.target.value))}
            className="absolute w-full top-1/2 -translate-y-1/2 h-8 appearance-none bg-transparent pointer-events-none accent-signal [&::-webkit-slider-thumb]:pointer-events-auto [&::-moz-range-thumb]:pointer-events-auto"
          />
          <input
            type="range" aria-label="Window end"
            min={0} max={span} value={daysBetween(min, to)}
            onChange={e => setTo(Number(e.target.value))}
            className="absolute w-full top-1/2 -translate-y-1/2 h-8 appearance-none bg-transparent pointer-events-none accent-signal [&::-webkit-slider-thumb]:pointer-events-auto [&::-moz-range-thumb]:pointer-events-auto"
          />
        </div>

        <span className="text-[11px] font-mono text-ink whitespace-nowrap tabular-nums">{to}</span>
      </div>

      <div className="flex items-center justify-between mt-1.5 text-[10px] font-mono text-faint">
        <span>{min}</span>
        <span>full record</span>
        <span>{max}</span>
      </div>
    </div>
  );
}

function formatSpan(days: number) {
  if (days < 62) return `${days} d`;
  const months = Math.round(days / 30.44);
  if (months < 24) return `${months} mo`;
  return `${(days / 365.25).toFixed(1)} yr`;
}

function daysBetween(a: string, b: string) {
  const d1 = new Date(`${a}T00:00:00`).getTime();
  const d2 = new Date(`${b}T00:00:00`).getTime();
  return Math.max(0, Math.round((d2 - d1) / 86400000));
}

function addDays(base: string, n: number) {
  const d = new Date(`${base}T00:00:00`);
  d.setDate(d.getDate() + n);
  return d.toISOString().slice(0, 10);
}
