import { CorpusBounds, DataAdapter, EventsPage, FilterState, TraceEvent } from '@/types';

/** Filters that constrain the date window (excluded when asking for corpus bounds). */
function toParams(filters: Partial<FilterState>, includeDateWindow = true): string {
  const p = new URLSearchParams();
  if (filters.query) p.set('query', filters.query);
  if (includeDateWindow) {
    if (filters.dateFrom) p.set('dateFrom', filters.dateFrom);
    if (filters.dateTo) p.set('dateTo', filters.dateTo);
  }
  (filters.eventTypes ?? []).forEach(t => p.append('eventType', t));
  if (filters.overnight?.length === 1 && filters.overnight[0] === 'overnight') p.set('overnight', 'true');
  if ((filters.minProbability ?? 0) > 0) p.set('minProbability', String(filters.minProbability));
  (filters.tags ?? []).forEach(t => p.append('tag', t));
  return p.toString();
}

async function fetchPage(filters: Partial<FilterState>): Promise<EventsPage> {
  const res = await fetch(`/api/events?${toParams(filters)}`);
  const data = await res.json();
  if (!res.ok) throw new Error(data?.error ? `events API: ${data.error}` : `events API ${res.status}`);
  return {
    events: (data.events ?? []) as TraceEvent[],
    count: data.count ?? 0,
    total_in_window: data.total_in_window ?? 0,
    truncated: Boolean(data.truncated),
  };
}

// Live adapter: her real corpus over analysis.latest_events. Same interface as the mock,
// so the whole UI is unchanged — only the data source swaps (ADR-0001 adapter seam).
export const pgAdapter: DataAdapter = {
  async listEvents() {
    return (await fetchPage({})).events;
  },
  async searchEvents(filters) {
    return (await fetchPage(filters)).events;
  },
  async searchEventsPage(filters) {
    return fetchPage(filters);
  },
  async getBounds(filters): Promise<CorpusBounds> {
    const res = await fetch(`/api/events/bounds?${toParams(filters, false)}`);
    const data = await res.json();
    if (!res.ok) throw new Error(data?.error ? `bounds API: ${data.error}` : `bounds API ${res.status}`);
    return {
      total: data.total ?? 0,
      min_date: data.min_date ?? null,
      max_date: data.max_date ?? null,
    };
  },
  async getEvent(id) {
    const all = (await fetchPage({})).events;
    return all.find(e => e.event_id === id) ?? null;
  },
  async askAgent(question, filters) {
    // Agent backend (Claude SDK + AgentOS) is a later phase. Until then, deterministic
    // retrieval over the real data with real receipts — no fabricated narration.
    const page = await fetchPage(filters);
    const relevant = page.events;
    const top = relevant.slice(0, 10);
    const scope = page.truncated
      ? `${page.total_in_window} real events match this window (${relevant.length} loaded)`
      : `${relevant.length} real events match`;
    const answer = top.length
      ? `${scope}. First: ${top[0].event_type} ${top[0].serial_display} at ${top[0].place_id || 'unnamed place'} (${top[0].start_eastern}).`
      : 'No events match the current filters.';
    return { answer, receipts: top.map(e => e.event_id).slice(0, 6) };
  },
};
