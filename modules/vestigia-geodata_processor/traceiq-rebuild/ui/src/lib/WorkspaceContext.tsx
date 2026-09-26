"use client";
import { createContext, useContext, useState, useEffect, useCallback, useMemo, ReactNode } from 'react';
import {
  CorpusBounds, FilterState, MapMode, TabKey, TraceEvent, WorkspaceMode,
} from '@/types';
import { useTheme } from './useTheme';
import { getAdapter } from '@/mock/adapter';

/**
 * Default loaded span. The corpus is ~20,160 events over ~7 years; loading all of it is
 * ~7.9 MB, so we load a 3-month window by default and let the scrubber widen it. The
 * corpus total and first/last date are ALWAYS shown (from bounds), so a narrow window is
 * never mistaken for the whole record.
 */
export const DEFAULT_WINDOW_MONTHS = 3;

interface WindowMeta {
  /** Rows currently loaded. */
  count: number;
  /** True match count for the loaded window. */
  totalInWindow: number;
  /** Window matched more than was returned (hit the row ceiling). */
  truncated: boolean;
}

interface WorkspaceCtx {
  theme: "light" | "dark";
  toggleTheme: () => void;
  mode: WorkspaceMode;
  setMode: (m: WorkspaceMode) => void;
  tab: TabKey;
  setTab: (t: TabKey) => void;
  filters: FilterState;
  setFilters: (f: FilterState) => void;
  /** Reset filters back to the default 3-month window (never to "load everything"). */
  resetFilters: () => void;
  /** Widen the window to the entire corpus, deliberately. */
  loadEntireCorpus: () => void;
  mapMode: MapMode;
  setMapMode: (m: MapMode) => void;
  selectedEvent: TraceEvent | null;
  setSelectedEvent: (e: TraceEvent | null) => void;
  filteredEvents: TraceEvent[];
  setFilteredEvents: (e: TraceEvent[]) => void;
  /** Corpus extent for the current non-date filters — independent of the window. */
  bounds: CorpusBounds;
  windowMeta: WindowMeta;
  loading: boolean;
  error: string | null;
}

const WorkspaceContext = createContext<WorkspaceCtx | null>(null);

const baseFilters: FilterState = {
  query: '',
  dateFrom: '',
  dateTo: '',
  eventTypes: [],
  tags: [],
  overnight: [],
  minProbability: 0,
};

export function shiftMonths(date: string, months: number): string {
  const d = new Date(`${date}T00:00:00`);
  d.setMonth(d.getMonth() + months);
  return d.toISOString().slice(0, 10);
}

export function WorkspaceProvider({ children }: { children: ReactNode }) {
  const [mode, setMode] = useState<WorkspaceMode>('manual');
  const [tab, setTab] = useState<TabKey>('explore');
  const [filters, setFilters] = useState<FilterState>(baseFilters);
  const [mapMode, setMapMode] = useState<MapMode>('pins');
  const [selectedEvent, setSelectedEvent] = useState<TraceEvent | null>(null);
  const [filteredEvents, setFilteredEvents] = useState<TraceEvent[]>([]);
  const [bounds, setBounds] = useState<CorpusBounds>({ total: 0, min_date: null, max_date: null });
  const [windowMeta, setWindowMeta] = useState<WindowMeta>({ count: 0, totalInWindow: 0, truncated: false });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [windowReady, setWindowReady] = useState(false);
  const { theme, toggle } = useTheme();

  // Non-date filters decide the corpus extent; the date window does not.
  const nonDateKey = useMemo(() => JSON.stringify({
    query: filters.query,
    eventTypes: filters.eventTypes,
    tags: filters.tags,
    overnight: filters.overnight,
    minProbability: filters.minProbability,
  }), [filters.query, filters.eventTypes, filters.tags, filters.overnight, filters.minProbability]);

  // 1) Bounds: corpus total + first/last date. Also seeds the default window on first load.
  useEffect(() => {
    let cancelled = false;
    const t = setTimeout(() => {
      getAdapter().getBounds(filters)
        .then(b => {
          if (cancelled) return;
          setBounds(b);
          setError(null);
          // Seed the default window once, anchored at the END of the record.
          setWindowReady(prev => {
            if (prev) return prev;
            if (b.max_date) {
              const from = shiftMonths(b.max_date, -DEFAULT_WINDOW_MONTHS);
              const min = b.min_date ?? from;
              setFilters(f => ({
                ...f,
                dateFrom: from < min ? min : from,
                dateTo: b.max_date as string,
              }));
            }
            return true;
          });
        })
        .catch(e => { if (!cancelled) setError(String(e.message ?? e)); });
    }, 150);
    return () => { cancelled = true; clearTimeout(t); };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [nonDateKey]);

  // 2) The window itself. Held back until the default window is seeded, so we never
  //    open the app by pulling all ~20k events.
  useEffect(() => {
    if (!windowReady) return;
    let cancelled = false;
    setLoading(true);
    const t = setTimeout(() => {
      getAdapter().searchEventsPage(filters)
        .then(page => {
          if (cancelled) return;
          setFilteredEvents(page.events);
          setWindowMeta({
            count: page.count,
            totalInWindow: page.total_in_window,
            truncated: page.truncated,
          });
          setError(null);
        })
        .catch(e => { if (!cancelled) { setError(String(e.message ?? e)); setFilteredEvents([]); } })
        .finally(() => { if (!cancelled) setLoading(false); });
    }, 150);
    return () => { cancelled = true; clearTimeout(t); };
  }, [filters, windowReady]);

  const resetFilters = useCallback(() => {
    const max = bounds.max_date;
    const from = max ? shiftMonths(max, -DEFAULT_WINDOW_MONTHS) : '';
    setFilters({
      ...baseFilters,
      dateFrom: bounds.min_date && from && from < bounds.min_date ? bounds.min_date : from,
      dateTo: max ?? '',
    });
  }, [bounds.max_date, bounds.min_date]);

  const loadEntireCorpus = useCallback(() => {
    setFilters(f => ({ ...f, dateFrom: bounds.min_date ?? '', dateTo: bounds.max_date ?? '' }));
  }, [bounds.min_date, bounds.max_date]);

  return (
    <WorkspaceContext.Provider value={{
      mode, setMode,
      tab, setTab,
      filters, setFilters,
      resetFilters, loadEntireCorpus,
      mapMode, setMapMode,
      selectedEvent, setSelectedEvent,
      filteredEvents, setFilteredEvents,
      bounds, windowMeta, loading, error,
      theme, toggleTheme: toggle,
    }}>
      {children}
    </WorkspaceContext.Provider>
  );
}

export function useWorkspace() {
  const ctx = useContext(WorkspaceContext);
  if (!ctx) throw new Error('useWorkspace must be inside WorkspaceProvider');
  return ctx;
}
