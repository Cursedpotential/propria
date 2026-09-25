// Byline: Claude Code · Opus 5.5 · 2026-09-25
// Summary header for a category strip, like the bout review page: title, a plain-language description, the counts, and
// (for a diverging scheme) how the weight splits between the negative side, neutral and the positive side.

import "./category-strip.css";

import type { StripRow } from "./rows";
import { summarizeRows } from "./summary";
import { schemeValue, type Scheme } from "./schemes";

export interface Fact {
  label: string;
  value: string | number;
}

export interface StripSummaryProps {
  rows: StripRow[];
  title: string;
  description?: string;
  scheme?: Scheme;
  /** Names for the counts, e.g. { row: "Days", block: "Bouts", weight: "Messages" }. */
  nouns?: { row?: string; block?: string; weight?: string };
  /** More counts to show after the computed ones (e.g. shifts, labelled). */
  facts?: Fact[];
}

const pct = (part: number, whole: number) => (whole ? `${Math.round((part / whole) * 100)}%` : "0%");

export function StripSummary({ rows, title, description, scheme, nouns = {}, facts = [] }: StripSummaryProps) {
  const t = summarizeRows(rows, scheme);
  const all: Fact[] = [
    { label: nouns.row ?? "Rows", value: t.rows.toLocaleString() },
    { label: nouns.block ?? "Blocks", value: t.blocks.toLocaleString() },
    { label: nouns.weight ?? "Total", value: t.weight.toLocaleString() },
    ...facts,
  ];
  const ramp = (scheme?.ramp ?? {}) as Record<number, string>;
  const b = t.balance;
  // Which of the data's categories fall on each side, for the hover text.
  const side = (on: (v: number) => boolean) =>
    t.byCategory.filter(([c]) => { const v = scheme ? schemeValue(scheme, c) : undefined; return v !== undefined && on(v); })
      .map(([c]) => c).join(", ");

  return (
    <header className="strip-summary">
      <div>
        <h2>{title}</h2>
        {description ? <p className="strip-summary-description">{description}</p> : null}
      </div>
      <div className="strip-summary-facts">
        {all.map((f) => (
          <span key={f.label}>
            {f.label} <b>{typeof f.value === "number" ? f.value.toLocaleString() : f.value}</b>
          </span>
        ))}
      </div>
      {b && scheme ? (
        <div className="strip-balance" aria-label={`${scheme.label} balance`}>
          <div className="strip-balance-bar">
            <i style={{ flexGrow: b.negative, background: ramp[-2] }} title={`${scheme.ends?.[0] ?? "Negative"} side ${pct(b.negative, t.weight)}`} />
            <i style={{ flexGrow: b.neutral, background: ramp[0] }} title={`Neutral ${pct(b.neutral, t.weight)}`} />
            <i style={{ flexGrow: b.positive, background: ramp[2] }} title={`${scheme.ends?.[1] ?? "Positive"} side ${pct(b.positive, t.weight)}`} />
            {b.unmapped ? <i style={{ flexGrow: b.unmapped, background: "var(--muted)" }} title={`Not on the scale ${pct(b.unmapped, t.weight)}`} /> : null}
          </div>
          <div className="strip-balance-labels">
            <span title={side((v) => v < 0)}>{scheme.ends?.[0] ?? "Negative"} side <b>{pct(b.negative, t.weight)}</b></span>
            <span title={side((v) => v === 0)}>Neutral <b>{pct(b.neutral, t.weight)}</b></span>
            <span title={side((v) => v > 0)}>{scheme.ends?.[1] ?? "Positive"} side <b>{pct(b.positive, t.weight)}</b></span>
            {b.unmapped ? <span>Not on the scale <b>{pct(b.unmapped, t.weight)}</b> ({t.unmapped.join(", ")})</span> : null}
          </div>
        </div>
      ) : null}
    </header>
  );
}
