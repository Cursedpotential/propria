"use client";
import { useState, useMemo } from 'react';
import { useWorkspace } from '@/lib/WorkspaceContext';
import { TraceEvent } from '@/types';
import { KnownPlaceEditor } from '@/components/editor/KnownPlaceEditor';
import { ArrowUpDown, MapPin, ExternalLink, Hash, AlertCircle, Edit3 } from 'lucide-react';

type SortKey = 'serial_display' | 'place_id' | 'start_eastern' | 'overnight_simple' | 'duration';

export function ResultsTable() {
  const { filteredEvents, selectedEvent, setSelectedEvent, filters, setFilteredEvents } = useWorkspace();
  const [sort, setSort] = useState<{ key: SortKey; dir: 'asc' | 'desc' }>({ key: 'start_eastern', dir: 'desc' });
  const [editingEvent, setEditingEvent] = useState<TraceEvent | null>(null);

  const sorted = useMemo(() => {
    const out = [...filteredEvents];
    out.sort((a, b) => {
      let va: any = a[sort.key as keyof TraceEvent];
      let vb: any = b[sort.key as keyof TraceEvent];
      if (sort.key === 'start_eastern') {
        va = new Date(va).getTime();
        vb = new Date(vb).getTime();
      } else if (sort.key === 'duration') {
        va = durationMinutes(a.start_eastern, a.end_eastern);
        vb = durationMinutes(b.start_eastern, b.end_eastern);
      }
      if (va < vb) return sort.dir === 'asc' ? -1 : 1;
      if (va > vb) return sort.dir === 'asc' ? 1 : -1;
      return 0;
    });
    return out;
  }, [filteredEvents, sort]);

  const Header = ({ k, children, className = '' }: { k: SortKey; children: React.ReactNode; className?: string }) => (
    <button
      onClick={() => setSort({ key: k, dir: sort.key === k && sort.dir === 'asc' ? 'desc' : 'asc' })}
      className={`flex items-center gap-1 text-left text-[10.5px] uppercase tracking-wide font-mono font-medium text-muted hover:text-ink ${className}`}
    >
      {children} <ArrowUpDown className="w-3 h-3" />
    </button>
  );

  return (
    <div className="flex flex-col h-full w-full min-w-[280px] max-w-[420px] bg-surface border-l border-border">
      <div className="px-4 py-3 border-b border-border flex items-center justify-between">
        <div className="text-sm font-medium text-ink flex items-center gap-2"><Hash className="w-4 h-4" /> Results</div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted">{filteredEvents.length} events</span>
          <button onClick={() => setSelectedEvent(selectedEvent ? null : filteredEvents[0] ?? null)} className="p-1.5 rounded-md border border-border text-muted hover:text-ink hover:bg-surface-2" title="Open in map"><MapPin className="w-3.5 h-3.5" /></button>
          <button onClick={() => setEditingEvent(editingEvent ? null : (selectedEvent ?? filteredEvents[0] ?? null))} className={`p-1.5 rounded-md border text-muted hover:text-ink hover:bg-surface-2 ${editingEvent ? 'bg-signal-soft border-signal text-signal' : 'border-border'}`} title="Edit known place"><Edit3 className="w-3.5 h-3.5" /></button>
          <button onClick={() => alert('Analytics view stub')} className="p-1.5 rounded-md border border-border text-muted hover:text-ink hover:bg-surface-2" title="Open in analytics"><ExternalLink className="w-3.5 h-3.5" /></button>
        </div>
      </div>
      <div className="flex-1 overflow-auto">
        <table className="w-full text-left border-collapse">
          <thead className="sticky top-0 bg-surface z-10">
            <tr className="border-b border-border">
              <th className="px-3 py-2"><Header k="serial_display">ID</Header></th>
              <th className="px-3 py-2"><Header k="place_id">Place</Header></th>
              <th className="px-3 py-2"><Header k="start_eastern">Start</Header></th>
              <th className="px-3 py-2"><Header k="overnight_simple">Ovn</Header></th>
              <th className="px-3 py-2 text-right"><Header k="duration" className="justify-end">Dur</Header></th>
            </tr>
          </thead>
          <tbody>
            {sorted.map(e => {
              const active = selectedEvent?.event_id === e.event_id;
              const dur = durationMinutes(e.start_eastern, e.end_eastern);
              return (
                <tr
                  key={e.event_id}
                  onClick={() => setSelectedEvent(e)}
                  className={`text-xs border-b border-border cursor-pointer transition ${active ? 'bg-signal-soft' : 'hover:bg-surface-2'}`}
                >
                  <td className="px-3 py-2 font-mono text-muted">{e.serial_display}</td>
                  <td className="px-3 py-2 max-w-[120px] truncate" title={e.place_id}>{e.place_id}</td>
                  <td className="px-3 py-2 font-mono text-muted">{e.start_eastern.slice(0, 10)}<br/><span className="text-[10px]">{e.start_eastern.slice(11, 16)}</span></td>
                  <td className="px-3 py-2">
                    {e.overnight_simple === 'overnight' ? <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-overnight/10 text-overnight border border-overnight/20">ON</span> : <span className="text-faint">—</span>}
                  </td>
                  <td className="px-3 py-2 text-right font-mono text-muted">{formatDuration(dur)}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
        {editingEvent && (
          <div className="border-t border-border p-3 bg-surface-2">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-ink">Edit known place</span>
              <button onClick={() => setEditingEvent(null)} className="text-[10px] text-muted hover:text-ink">Close</button>
            </div>
            <KnownPlaceEditor
              event={editingEvent}
              onSaved={(place) => {
                const next = filteredEvents.map(ev =>
                  (ev.place_id && ev.place_id === place.place_id) ||
                  (!ev.place_id && ev.lat_r4 === place.lat_r4 && ev.lng_r4 === place.lng_r4)
                    ? { ...ev, tags: place.tags }
                    : ev
                );
                setFilteredEvents(next);
                setSelectedEvent(next.find(ev => ev.event_id === selectedEvent?.event_id) ?? selectedEvent);
                setEditingEvent(null);
              }}
            />
          </div>
        )}
        {sorted.length === 0 && (
          <div className="flex flex-col items-center justify-center h-40 text-muted text-sm gap-2">
            <AlertCircle className="w-5 h-5" /> No events match filters.
          </div>
        )}
      </div>
    </div>
  );
}

function durationMinutes(a: string, b: string) {
  return Math.max(0, Math.round((new Date(b).getTime() - new Date(a).getTime()) / 60000));
}

function formatDuration(min: number) {
  if (min < 60) return `${min}m`;
  const h = Math.floor(min / 60);
  const m = min % 60;
  return m ? `${h}h ${m}m` : `${h}h`;
}
