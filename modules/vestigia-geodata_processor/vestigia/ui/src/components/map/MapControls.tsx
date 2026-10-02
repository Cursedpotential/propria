"use client";
import { useWorkspace } from '@/lib/WorkspaceContext';
import { MapMode } from '@/types';
import { MapPin, Route, Flame, Clock, ExternalLink } from 'lucide-react';

const mapModes: { key: MapMode; label: string; icon: React.ElementType }[] = [
  { key: 'pins', label: 'Pins', icon: MapPin },
  { key: 'paths', label: 'Paths', icon: Route },
  { key: 'heatmap', label: 'Heat', icon: Flame },
  { key: 'time', label: 'Time', icon: Clock },
];

export function MapControls() {
  const { mapMode, setMapMode, filters, bounds, windowMeta, loading } = useWorkspace();
  const dateFrom = filters.dateFrom || bounds.min_date || '—';
  const dateTo = filters.dateTo || bounds.max_date || '—';

  return (
    <>
      <div className="absolute top-3 left-3 z-20 flex flex-col gap-2">
        <div className="bg-surface/90 backdrop-blur border border-border rounded-[var(--r)] shadow-[var(--shadow)] p-1 flex flex-col gap-1">
          {mapModes.map(m => {
            const Icon = m.icon;
            return (
              <button
                key={m.key}
                onClick={() => setMapMode(m.key)}
                title={m.label}
                className={`flex items-center justify-center w-8 h-8 rounded-[var(--r-sm)] transition ${mapMode === m.key ? 'bg-signal text-white' : 'text-muted hover:text-ink hover:bg-surface-2'}`}
              >
                <Icon className="w-4 h-4" />
              </button>
            );
          })}
        </div>
        <button
          onClick={() => alert('Kepler.gl pop-out is a later-phase stub.')}
          className="bg-surface/90 backdrop-blur border border-border rounded-[var(--r)] shadow-[var(--shadow)] px-2.5 py-1.5 text-xs font-medium text-muted hover:text-ink flex items-center gap-1.5"
        >
          <ExternalLink className="w-3.5 h-3.5" /> Kepler
        </button>
      </div>

      {/* Always shows the corpus total and full extent, so a windowed view can never
          read as the complete record. */}
      <div className="absolute top-3 right-14 z-20 bg-surface/90 backdrop-blur border border-border rounded-[var(--r)] shadow-[var(--shadow)] px-3 py-2">
        <div className="text-[10px] uppercase tracking-wide font-mono text-muted mb-0.5">
          Events in window
        </div>
        <div className="text-sm font-semibold text-ink tabular-nums">
          {loading ? '…' : windowMeta.totalInWindow.toLocaleString()}
          <span className="text-muted font-normal"> of {bounds.total.toLocaleString()}</span>
        </div>
        <div className="text-[10px] font-mono text-faint mt-0.5">{dateFrom} → {dateTo}</div>
        <div className="text-[10px] font-mono text-faint">
          full record {bounds.min_date ?? '—'} → {bounds.max_date ?? '—'}
        </div>
        {windowMeta.truncated && (
          <div className="text-[10px] font-mono text-amber-500 mt-0.5">
            window truncated at {windowMeta.count.toLocaleString()} rows
          </div>
        )}
      </div>
    </>
  );
}
