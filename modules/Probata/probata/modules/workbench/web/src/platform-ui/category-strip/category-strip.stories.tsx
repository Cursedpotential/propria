// Byline: Claude Code · Opus 5.5 · 2026-09-24; schemes and summary header 2026-09-25
// Same real data, different categories and schemes: the strip takes whatever category the records carry and colors it
// by the scheme it is given. Data: 2024 bout tone results for the texts from her phone (645 bouts, bout-tone-v1, no
// message text), from raw_duck.msg_bouts_20260924 ⋈ msg_bout_labels_20260924 (Probata scripts/jev_eval/mood_strip_fixture.sql).
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import herPhone2024 from "./__fixtures__/her-phone-2024-tone.json";
import { rowsFromRecords } from "./rows";
import { RAMPS, SCHEMES, type Scheme } from "./schemes";
import { StripReport } from "./strip-report";

interface Bout {
  id: string;
  start: string;
  end: string;
  shifts: number;
  abrupt: number;
  stretches: [string, number, string][];
}

const BOUTS = herPhone2024 as unknown as Bout[];
// One flat record per tone stretch: the shape any query can return.
const STRETCHES = BOUTS.flatMap((b) =>
  b.stretches.map(([tone, n, driver]) => ({
    day: b.start.slice(0, 10), bout: b.id, time: `${b.start.slice(11)}–${b.end.slice(11)}`, tone, n, driver,
  })),
);
type Stretch = (typeof STRETCHES)[number];
const rowsBy = (records: Stretch[], category: (r: Stretch) => string) =>
  rowsFromRecords(records, { row: (r) => r.day, block: (r) => r.bout, category, weight: (r) => r.n, blockTitle: (r) => r.time });

const SHIFT_FACTS = [
  { label: "Tone shifts", value: BOUTS.reduce((n, b) => n + b.shifts, 0) },
  { label: "Abrupt", value: BOUTS.reduce((n, b) => n + b.abrupt, 0) },
];
const NOUNS = { row: "Days", block: "Bouts", weight: "Messages" };

function Selectable(args: React.ComponentProps<typeof StripReport>) {
  const [row, setRow] = useState<string | null>(null);
  return (
    <div className="grid max-w-[1100px] gap-3 p-4">
      <StripReport {...args} selectedRowId={row} onSelectRow={setRow} />
      <p className="text-sm text-muted-foreground">{row ? `Picked ${row}` : "Click a row."}</p>
    </div>
  );
}

const meta = {
  title: "Platform/Blocks/Category strip",
  component: StripReport,
  render: (args) => <Selectable {...args} />,
  args: { rows: [], title: "", unit: "msgs", nouns: NOUNS },
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof StripReport>;

export default meta;
type Story = StoryObj<typeof meta>;

/** Tone on the diverging scale (hostile … affectionate), with the summary header and the balance of the two sides. */
export const Tones: Story = {
  args: {
    rows: rowsBy(STRETCHES, (r) => r.tone),
    scheme: SCHEMES.tone,
    title: "Tone by day",
    summary: "Texts from her phone, June to December 2024, cut into bouts. Each bout's tone stretches are colored from hostile (red) through neutral (grey) to affectionate (blue).",
    description: "Each row is a day; each block is a bout, split into its tone stretches. Click a row to pick that day.",
    facts: SHIFT_FACTS,
  },
};

/** Same records, category = who drove each stretch; no order, so colors come from the categorical palette. */
export const WhoDrove: Story = {
  args: {
    rows: rowsBy(STRETCHES, (r) => r.driver),
    scheme: SCHEMES.categorical,
    title: "Who drove each stretch",
    summary: "The same bouts, split by who set the tone of each stretch.",
  },
};

/** The bout review page's own tone colors, pinned over the tone scheme with `colors`. */
export const ArtifactColors: Story = {
  args: {
    rows: rowsBy(STRETCHES.filter((r) => r.day.startsWith("2024-10")), (r) => r.tone),
    scheme: SCHEMES.tone,
    colors: {
      affectionate: "#c2477a", friendly: "#3d8b5a", neutral: "#9aa1ab", tense: "#c98a1c",
      hostile: "#c0392b", distressed: "#7a4bb5", conciliatory: "#1f8a8a",
    },
    title: "October 2024, original colors",
    facts: SHIFT_FACTS,
  },
};

function SchemeRow({ scheme }: { scheme: Scheme }) {
  const ramp = scheme.ramp as Record<number, string>;
  const steps = Object.keys(ramp).map(Number).sort((a, b) => a - b);
  const wordsAt = (v: number) => Object.entries(scheme.values).filter(([, x]) => x === v).map(([w]) => w);
  return (
    <section className="grid gap-2 rounded-md border bg-card p-4">
      <div className="flex flex-wrap items-baseline gap-x-3">
        <b>{scheme.label}</b>
        <span className="text-sm text-muted-foreground">
          {scheme.kind}
          {scheme.ends ? ` · ${scheme.ends[0]} ↔ ${scheme.ends[1]}` : ""}
        </span>
      </div>
      {steps.length ? (
        <div className="grid gap-1" style={{ gridTemplateColumns: `repeat(${steps.length}, minmax(0, 1fr))` }}>
          {steps.map((v) => (
            <div key={v} className="grid gap-1 text-xs text-muted-foreground">
              <span className="h-5 rounded" style={{ background: ramp[v] }} />
              <span>{wordsAt(v).join(", ") || "—"}</span>
            </div>
          ))}
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">No order: each category takes the next palette color, in order of appearance.</p>
      )}
    </section>
  );
}

/** Every scheme with the words it knows, for picking one. Words a scheme does not know get a palette color and a dashed legend swatch. */
export const Schemes: Story = {
  render: () => (
    <div className="grid max-w-[1100px] gap-3 p-4">
      {Object.values(SCHEMES).map((s) => (
        <SchemeRow key={s.id} scheme={s} />
      ))}
      <p className="text-sm text-muted-foreground">
        Ramps: red–blue {Object.keys(RAMPS.redBlue).length} steps (sentiment, tone), orange–purple (conflict ↔ cooperation),
        blues and oranges (low → high), status (yes / no / unknown).
      </p>
    </div>
  ),
};
