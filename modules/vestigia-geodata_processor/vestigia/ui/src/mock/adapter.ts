import { DataAdapter, FilterState, TraceEvent } from '@/types';
import { MOCK_EVENTS } from './events';

function matches(event: TraceEvent, filters: Partial<FilterState>): boolean {
  if (filters.query) {
    const q = filters.query.toLowerCase();
    const hay = [event.serial_display, event.raw_type, event.event_type, event.place_id, event.tags.join(' ')].join(' ').toLowerCase();
    if (!hay.includes(q)) return false;
  }
  if (filters.dateFrom && event.start_eastern < filters.dateFrom) return false;
  if (filters.dateTo && event.end_eastern > filters.dateTo + 'T23:59:59') return false;
  if (filters.eventTypes?.length && !filters.eventTypes.includes(event.event_type)) return false;
  if (filters.tags?.length && !filters.tags.some(t => event.tags.includes(t))) return false;
  if (filters.overnight?.length && !filters.overnight.includes(event.overnight_simple)) return false;
  if (typeof filters.minProbability === 'number' && event.probability < filters.minProbability) return false;
  return true;
}

export const mockAdapter: DataAdapter = {
  async listEvents() {
    return MOCK_EVENTS;
  },
  async searchEvents(filters) {
    return MOCK_EVENTS.filter(e => matches(e, filters));
  },
  async searchEventsPage(filters) {
    const events = MOCK_EVENTS.filter(e => matches(e, filters));
    // The mock corpus is small enough that a window is never truncated.
    return { events, count: events.length, total_in_window: events.length, truncated: false };
  },
  async getBounds(filters) {
    // Bounds ignore the date window, matching the live adapter's contract.
    const { dateFrom: _f, dateTo: _t, ...rest } = filters;
    const all = MOCK_EVENTS.filter(e => matches(e, rest));
    const dates = all.map(e => e.start_eastern.slice(0, 10)).sort();
    return {
      total: all.length,
      min_date: dates[0] ?? null,
      max_date: dates[dates.length - 1] ?? null,
    };
  },
  async getEvent(id) {
    return MOCK_EVENTS.find(e => e.event_id === id) ?? null;
  },
  async askAgent(question, filters) {
    const relevant = MOCK_EVENTS.filter(e => matches(e, filters)).slice(0, 10);
    const answer = `I found ${relevant.length} events matching your query. Top pattern: ${relevant[0]?.event_type ?? 'none'} near ${relevant[0]?.place_id ?? 'unknown'}.`;
    const receipts = relevant.map(e => e.event_id).slice(0, 6);
    return { answer, receipts };
  },
};

import { pgAdapter } from '@/lib/pgAdapter';

// Live by default (real traceiq data via /api/events). Set NEXT_PUBLIC_DATA_MODE=mock
// to fall back to the in-memory mock corpus.
export function getAdapter(): DataAdapter {
  return process.env.NEXT_PUBLIC_DATA_MODE === 'mock' ? mockAdapter : pgAdapter;
}
