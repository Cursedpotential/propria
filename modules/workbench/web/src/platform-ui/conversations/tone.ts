// Byline: Claude Code · Opus 5.5 · 2026-09-24
// Tone vocabulary shared by the conversation components (mood strip, conversation grid, summary charts).
// The seven labels are the ones the bout tone pass (Probata scripts/jev_eval/bouts_tone_opus.py) emits; colors are
// data colors, defined as CSS custom properties in conversations.css so light and dark themes both work.

export const TONES = [
  "affectionate",
  "friendly",
  "neutral",
  "tense",
  "hostile",
  "distressed",
  "conciliatory",
] as const;

export type Tone = (typeof TONES)[number];

export const TONE_LABEL: Record<Tone, string> = {
  affectionate: "Affectionate",
  friendly: "Friendly",
  neutral: "Neutral",
  tense: "Tense",
  hostile: "Hostile",
  distressed: "Distressed",
  conciliatory: "Conciliatory",
};

/** A stretch of consecutive messages in one tone: [tone, message count, who drove it (a sender name or "both")]. */
export type ToneStretch = [tone: Tone, messages: number, driver: string];

/** One bout (messages within a day with no silence over 30 minutes) and its tone reading. */
export interface BoutTone {
  id: string;
  /** Local start and end, `YYYY-MM-DDTHH:mm`. */
  start: string;
  end: string;
  messages: number;
  /** Messages per sender, keyed by the sender's display name as the data carries it. */
  senders: Record<string, number>;
  stretches: ToneStretch[];
  shifts: number;
  abrupt: number;
}

export function isTone(value: string): value is Tone {
  return (TONES as readonly string[]).includes(value);
}

/** The tone covering the most messages across the given stretches (ties go to the earlier tone in TONES). */
export function dominantTone(stretches: readonly ToneStretch[]): Tone {
  const counts = new Map<Tone, number>();
  for (const [tone, n] of stretches) counts.set(tone, (counts.get(tone) ?? 0) + n);
  let best: Tone = "neutral";
  let bestCount = -1;
  for (const tone of TONES) {
    const n = counts.get(tone) ?? 0;
    if (n > bestCount) {
      best = tone;
      bestCount = n;
    }
  }
  return best;
}
