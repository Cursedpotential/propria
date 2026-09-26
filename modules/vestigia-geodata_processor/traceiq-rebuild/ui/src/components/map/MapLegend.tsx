"use client";
import { useWorkspace } from '@/lib/WorkspaceContext';

const pinLegend = [
  { label: 'Visit', color: 'var(--signal)' },
  { label: 'Activity', color: 'var(--concerning)' },
  { label: 'Path', color: 'var(--faint)' },
  { label: 'Selected', color: '#3db8cc' },
];

const timeLegend = [
  { label: 'Night (20-06)', color: 'var(--overnight)' },
  { label: 'Day (06-20)', color: 'var(--concerning)' },
];

export function MapLegend() {
  const { mapMode } = useWorkspace();
  if (mapMode === 'heatmap') return null;
  const items = mapMode === 'time' ? timeLegend : pinLegend;

  return (
    <div className="absolute bottom-12 left-3 z-20 bg-surface/90 backdrop-blur border border-border rounded-[var(--r)] shadow-[var(--shadow)] px-3 py-2">
      <div className="text-[10px] uppercase tracking-wide font-mono text-muted mb-1.5">{mapMode === 'time' ? 'Time of day' : 'Pins'}</div>
      <div className="flex flex-col gap-1">
        {items.map(item => (
          <div key={item.label} className="flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full" style={{ background: item.color }} />
            <span className="text-[11px] text-ink">{item.label}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
