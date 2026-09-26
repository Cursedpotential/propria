export interface TraceEvent {
  event_id: string;
  serial_display: string;
  raw_type: string;
  event_type: 'visit' | 'activity' | 'timeline_path';
  start_eastern: string;
  end_eastern: string;
  start_utc: string;
  end_utc: string;
  lat: number;
  lng: number;
  lat_r4: number;
  lng_r4: number;
  place_id: string;
  tags: string[];
  overnight_simple: 'overnight' | 'daytrip' | null;
  probability: number;
}

export interface FilterState {
  query: string;
  dateFrom: string;
  dateTo: string;
  eventTypes: TraceEvent['event_type'][];
  tags: string[];
  overnight: TraceEvent['overnight_simple'][];
  minProbability: number;
}

export type MapMode = 'pins' | 'paths' | 'heatmap' | 'time';
export type WorkspaceMode = 'manual' | 'agent';

export type TabKey = 'explore' | 'analytics' | 'tables' | 'export' | 'config';

/** One windowed page of events, plus the honest match count for that window. */
export interface EventsPage {
  events: TraceEvent[];
  /** Rows returned in this response. */
  count: number;
  /** True match count for the current window, computed before LIMIT. */
  total_in_window: number;
  /** True when the window matched more rows than were returned. */
  truncated: boolean;
}

/**
 * Corpus extent for the current non-date filters — always displayed, so a 3-month
 * window is never mistaken for the whole record.
 */
export interface CorpusBounds {
  total: number;
  min_date: string | null;
  max_date: string | null;
}

export interface DataAdapter {
  listEvents(): Promise<TraceEvent[]>;
  searchEvents(filters: Partial<FilterState>): Promise<TraceEvent[]>;
  /** Windowed fetch: only the requested date range, with window/total metadata. */
  searchEventsPage(filters: Partial<FilterState>): Promise<EventsPage>;
  /** Corpus totals + first/last date, ignoring the date window. */
  getBounds(filters: Partial<FilterState>): Promise<CorpusBounds>;
  getEvent(id: string): Promise<TraceEvent | null>;
  askAgent(question: string, filters: Partial<FilterState>): Promise<{
    answer: string;
    receipts: string[];
  }>;
}

export interface KnownPlace {
  known_place_id?: number;
  place_id: string | null;
  lat_r4: number | null;
  lng_r4: number | null;
  label: string;
  tags: string[];
  is_concerning: boolean;
  is_watch: boolean;
  severity: number | null;
  note: string | null;
}

export interface LatLng { lat: number; lng: number; }
