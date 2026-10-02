import { TraceEvent } from '@/types';

const baseCities = [
  { name: 'Detroit, MI', lat: 42.3314, lng: -83.0458 },
  { name: 'Ann Arbor, MI', lat: 42.2808, lng: -83.7430 },
  { name: 'Grand Rapids, MI', lat: 42.9634, lng: -85.6681 },
  { name: 'Lansing, MI', lat: 42.7325, lng: -84.5555 },
  { name: 'Flint, MI', lat: 43.0125, lng: -83.6875 },
  { name: 'Traverse City, MI', lat: 44.7631, lng: -85.6206 },
  { name: 'Kalamazoo, MI', lat: 42.2917, lng: -85.5872 },
  { name: 'Toledo, OH', lat: 41.6528, lng: -83.5379 },
];

const rawTypes = [
  'google_maps_visit',
  'semantic_location_history',
  'timeline_path',
  'activity_segment',
  'place_visit',
];

const tagPool = ['home', 'work', 'overnight', 'travel', 'sensitive', 'medical', 'commercial', 'social', 'repeated'];

function randInt(min: number, max: number) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

function fmt(d: Date) {
  const pad = (n: number) => n.toString().padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

export function generateMockEvents(count = 320): TraceEvent[] {
  const events: TraceEvent[] = [];
  const now = new Date();
  const start = new Date(now);
  start.setDate(start.getDate() - 90);

  for (let i = 0; i < count; i++) {
    const t0 = new Date(start.getTime() + Math.random() * (now.getTime() - start.getTime()));
    const durationMin = randInt(15, 12 * 60);
    const t1 = new Date(t0.getTime() + durationMin * 60000);

    const city = baseCities[randInt(0, baseCities.length - 1)];
    const jitter = () => (Math.random() - 0.5) * 0.04;
    const lat = city.lat + jitter();
    const lng = city.lng + jitter();
    const latR4 = Math.round(lat * 10000) / 10000;
    const lngR4 = Math.round(lng * 10000) / 10000;

    const raw = rawTypes[randInt(0, rawTypes.length - 1)];
    let eventType: TraceEvent['event_type'] = 'visit';
    if (raw === 'timeline_path') eventType = 'timeline_path';
    else if (raw === 'activity_segment') eventType = 'activity';

    const hourStart = t0.getHours();
    const overnight = hourStart >= 20 || hourStart <= 5 ? 'overnight' : 'daytrip';

    const tags: string[] = [];
    const tagCount = randInt(1, 3);
    while (tags.length < tagCount) {
      const t = tagPool[randInt(0, tagPool.length - 1)];
      if (!tags.includes(t)) tags.push(t);
    }
    if (overnight === 'overnight' && !tags.includes('overnight')) tags.push('overnight');

    const prob = Math.round((0.5 + Math.random() * 0.5) * 100) / 100;

    events.push({
      event_id: crypto.randomUUID(),
      serial_display: `E${(i + 1).toString().padStart(4, '0')}`,
      raw_type: raw,
      event_type: eventType,
      start_eastern: fmt(t0),
      end_eastern: fmt(t1),
      start_utc: fmt(new Date(t0.getTime() + 4 * 3600000)),
      end_utc: fmt(new Date(t1.getTime() + 4 * 3600000)),
      lat,
      lng,
      lat_r4: latR4,
      lng_r4: lngR4,
      place_id: `place_${crypto.randomUUID().slice(0, 8)}`,
      tags,
      overnight_simple: Math.random() > 0.15 ? overnight : null,
      probability: prob,
    });
  }

  return events.sort((a, b) => new Date(a.start_utc).getTime() - new Date(b.start_utc).getTime());
}

export const MOCK_EVENTS = generateMockEvents();
