// Byline: Claude Code · Opus 5.5 · 2026-09-25
// Assignable color schemes for categorical data (owner 2026-09-25: predictable data such as sentiment or conflict vs
// cooperation, "where there's a clear positive negative flow", plus other predictable associations).
//
// Four kinds:
//   diverging   ordered negative -> neutral -> positive (sentiment, tone, conflict/cooperation). Values -3..+3.
//   sequential  ordered low -> high with no good or bad side (confidence, intensity, severity). Values 0..n.
//   status      a small closed set (yes / no / unknown).
//   categorical no order at all (who, source, format): palette colors in order of appearance.
// Palettes are ColorBrewer (RdBu, PuOr, Blues, Oranges; colorblind-safe) with a visible grey for neutral, and Tableau 10
// for categorical. Presets map category words (case-insensitive) onto a scale; a word a preset does not know falls back
// to a categorical color and is reported as unmapped, never forced onto the scale.

export type SchemeKind = "diverging" | "sequential" | "status" | "categorical";

export interface Scheme {
  id: string;
  label: string;
  kind: SchemeKind;
  /** Category word -> position on the scale (diverging -3..+3, sequential 0..n, status 0..n). */
  values: Record<string, number>;
  /** Scale position -> color. */
  ramp: Record<number, string>;
  /** Names of the two sides for the balance bar ("Negative" / "Positive"); a side covers every word on it, not only the end. */
  ends?: [string, string];
}

/** Diverging ramps: most negative (-3) to most positive (+3), grey in the middle so "neutral" stays visible. */
export const RAMPS = {
  redBlue: { [-3]: "#b2182b", [-2]: "#d6604d", [-1]: "#f4a582", 0: "#a8adb3", 1: "#92c5de", 2: "#4393c3", 3: "#2166ac" },
  orangePurple: { [-3]: "#b35806", [-2]: "#e08214", [-1]: "#fdb863", 0: "#a8adb3", 1: "#b2abd2", 2: "#8073ac", 3: "#542788" },
  blues: { 0: "#c6dbef", 1: "#9ecae1", 2: "#6baed6", 3: "#3182bd", 4: "#08519c" },
  oranges: { 0: "#fdd0a2", 1: "#fdae6b", 2: "#fd8d3c", 3: "#e6550d", 4: "#a63603" },
  status: { 0: "#3b4a6b", 1: "#c9ced6", 2: "#eceef1" },
} as const satisfies Record<string, Record<number, string>>;

/** Categorical palette (Tableau 10 + 2), for data with no order. */
export const CATEGORICAL = [
  "#4e79a7", "#f28e2b", "#e15759", "#76b7b2", "#59a14f", "#edc948",
  "#b07aa1", "#ff9da7", "#9c755f", "#bab0ac", "#86bcb6", "#d37295",
];

const words = (entries: [string[], number][]) =>
  Object.fromEntries(entries.flatMap(([ws, v]) => ws.map((w) => [w.toLowerCase(), v])));

export const SCHEMES = {
  sentiment: {
    id: "sentiment", label: "Sentiment", kind: "diverging", ramp: RAMPS.redBlue, ends: ["Negative", "Positive"],
    values: words([[["very negative"], -3], [["negative"], -2], [["somewhat negative", "mixed negative"], -1],
      [["neutral", "mixed"], 0], [["somewhat positive", "mixed positive"], 1], [["positive"], 2], [["very positive"], 3]]),
  },
  tone: {
    id: "tone", label: "Tone", kind: "diverging", ramp: RAMPS.redBlue, ends: ["Negative", "Positive"],
    values: words([[["hostile", "aggressive", "contemptuous"], -3], [["tense", "irritated", "defensive"], -2],
      [["distressed", "upset", "anxious", "sad"], -1], [["neutral", "logistical", "flat"], 0],
      [["conciliatory", "apologetic", "calming"], 1], [["friendly", "warm", "cordial"], 2],
      [["affectionate", "loving", "tender"], 3]]),
  },
  conflict: {
    id: "conflict", label: "Conflict ↔ cooperation", kind: "diverging", ramp: RAMPS.orangePurple,
    ends: ["Conflict", "Cooperation"],
    values: words([[["escalating", "attack", "threat"], -3], [["conflict", "argument", "blame"], -2],
      [["tension", "disagreement", "friction"], -1], [["neutral", "no conflict", "none"], 0],
      [["de-escalating", "repair", "apology"], 1], [["cooperation", "agreement", "compromise"], 2],
      [["support", "joint planning", "co-parenting"], 3]]),
  },
  confidence: {
    id: "confidence", label: "Confidence", kind: "sequential", ramp: RAMPS.blues,
    values: words([[["none", "unknown"], 0], [["low", "weak"], 1], [["medium", "moderate"], 2], [["high", "strong"], 3],
      [["certain", "confirmed"], 4]]),
  },
  severity: {
    id: "severity", label: "Severity", kind: "sequential", ramp: RAMPS.oranges,
    values: words([[["none"], 0], [["low", "minor"], 1], [["moderate", "medium"], 2], [["high", "serious"], 3],
      [["critical", "severe"], 4]]),
  },
  yesNo: {
    id: "yesNo", label: "Yes / no", kind: "status", ramp: RAMPS.status,
    values: words([[["yes", "true", "present", "flagged"], 0], [["no", "false", "absent", "clear"], 1],
      [["unknown", "unsure", "n/a"], 2]]),
  },
  categorical: { id: "categorical", label: "Categories (no order)", kind: "categorical", ramp: {}, values: {} },
} as const satisfies Record<string, Scheme>;

export type SchemeId = keyof typeof SCHEMES;

/** Position of a category on a scheme's scale, or undefined when the scheme does not know the word. */
export function schemeValue(scheme: Scheme, category: string): number | undefined {
  return scheme.values[category.toLowerCase()];
}

/**
 * Color for every category. Mapped words take the scheme's ramp color; unmapped words (and every word under a
 * categorical scheme) take the next categorical color; `fixed` overrides both.
 */
export function schemeColors(categories: readonly string[], scheme: Scheme = SCHEMES.categorical,
                             fixed: Record<string, string> = {}): Record<string, string> {
  const out: Record<string, string> = {};
  let next = 0;
  for (const c of categories) {
    const v = scheme.kind === "categorical" ? undefined : schemeValue(scheme, c);
    const ramp = scheme.ramp as Record<number, string>;
    if (fixed[c]) out[c] = fixed[c];
    else if (v !== undefined && ramp[v]) out[c] = ramp[v];
    else {
      out[c] = next < CATEGORICAL.length ? CATEGORICAL[next] : `hsl(${Math.round((next * 137.508) % 360)} 55% 55%)`;
      next += 1;
    }
  }
  return out;
}

/** Legend order: along the scale (negative -> positive, low -> high); unmapped categories after, as given. */
export function schemeOrder(categories: readonly string[], scheme: Scheme): string[] {
  if (scheme.kind === "categorical") return [...categories];
  const mapped = categories.filter((c) => schemeValue(scheme, c) !== undefined);
  const rest = categories.filter((c) => schemeValue(scheme, c) === undefined);
  return [...mapped.sort((a, b) => (schemeValue(scheme, a) ?? 0) - (schemeValue(scheme, b) ?? 0)), ...rest];
}
