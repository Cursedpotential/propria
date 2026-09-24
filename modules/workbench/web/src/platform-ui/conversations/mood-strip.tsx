// Byline: Claude Code · Opus 5.5 · 2026-09-24
// Mood strip: one row per day, each bout a block sized by its message count and split into its tone stretches.
// Rebuilt from the chart on the owner's bout review page (Probata scripts/jev_eval/bout_review_page.py), which the
// owner asked to keep as a reusable element (2026-09-24). Plain React and CSS: the chart is flex rows, no library.
// Nothing here names the people in a conversation; sender names come from the data.

import type { KeyboardEvent } from "react";

import "./conversations.css";

import { TONES, TONE_LABEL, senderColor, senderTotals, type BoutTone } from "./tone";

export interface MoodStripProps {
  bouts: BoutTone[];
  /** A line under the legend, e.g. how to read the strip. */
  description?: string;
  showLegend?: boolean;
  /** Show each sender (names from the data) with their message total. */
  showSenders?: boolean;
  selectedDay?: string | null;
  /** When set, each day row is a button that reports its day (`YYYY-MM-DD`). */
  onSelectDay?: (day: string) => void;
}

function boutTitle(bout: BoutTone): string {
  const who = Object.entries(bout.senders).map(([name, n]) => `${name} ${n}`).join(", ");
  const tones = bout.stretches.map(([tone, n, driver]) => `${TONE_LABEL[tone]} ${n} (${driver})`).join(" → ");
  return `${bout.start.slice(11)}–${bout.end.slice(11)} · ${bout.messages} messages` +
    (who ? ` (${who})` : "") + (tones ? ` · ${tones}` : " · not labelled") +
    (bout.shifts ? ` · ${bout.shifts} shift${bout.shifts > 1 ? "s" : ""}, ${bout.abrupt} abrupt` : "");
}

function groupByDay(bouts: BoutTone[]): [string, BoutTone[]][] {
  const byDay = new Map<string, BoutTone[]>();
  for (const b of bouts) {
    const day = b.start.slice(0, 10);
    const list = byDay.get(day);
    if (list) list.push(b);
    else byDay.set(day, [b]);
  }
  return [...byDay.entries()].sort(([a], [b]) => a.localeCompare(b));
}

function BoutBlock({ bout }: { bout: BoutTone }) {
  return (
    <span className="mood-bout" style={{ flexGrow: Math.max(1, bout.messages) }} title={boutTitle(bout)}>
      {bout.stretches.length ? (
        bout.stretches.map(([tone, n], i) => (
          <i key={i} className={`tone-${tone}`} style={{ flexGrow: Math.max(1, n) }} />
        ))
      ) : (
        <i className="mood-unlabelled" />
      )}
    </span>
  );
}

export function MoodStrip({
  bouts,
  description,
  showLegend = true,
  showSenders = true,
  selectedDay = null,
  onSelectDay,
}: MoodStripProps) {
  const days = groupByDay(bouts);
  const senders = senderTotals(bouts);
  const toneMessages = new Map<string, number>();
  for (const b of bouts) for (const [tone, n] of b.stretches) toneMessages.set(tone, (toneMessages.get(tone) ?? 0) + n);

  return (
    <section className="mood-strip">
      {showLegend ? (
        <div className="mood-strip-legend" aria-label="Tone colors">
          {TONES.map((tone) => (
            <span key={tone} className={`tone-${tone}`}>
              <i />
              {tone} · {(toneMessages.get(tone) ?? 0).toLocaleString()} msgs
            </span>
          ))}
        </div>
      ) : null}
      {showSenders && senders.length ? (
        <div className="mood-strip-senders" aria-label="Senders">
          {senders.map(([name, n], i) => (
            <span key={name}>
              <b style={{ color: senderColor(i) }}>{name}</b> · {n.toLocaleString()} msgs
            </span>
          ))}
        </div>
      ) : null}
      {description ? <p className="mood-strip-description">{description}</p> : null}
      <div className="mood-strip-rows">
        {days.map(([day, list]) => {
          const total = list.reduce((sum, b) => sum + b.messages, 0);
          const select = onSelectDay ? () => onSelectDay(day) : undefined;
          const onKeyDown = select
            ? (e: KeyboardEvent) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  select();
                }
              }
            : undefined;
          return (
            <div
              key={day}
              className="mood-day-row"
              role={select ? "button" : undefined}
              tabIndex={select ? 0 : undefined}
              aria-pressed={select ? selectedDay === day : undefined}
              onClick={select}
              onKeyDown={onKeyDown}
            >
              <span className="mood-day-date">{day}</span>
              <div className="mood-day-bar">
                {list.map((b) => (
                  <BoutBlock key={b.id} bout={b} />
                ))}
              </div>
              <span className="mood-day-count">{total}</span>
            </div>
          );
        })}
      </div>
    </section>
  );
}
