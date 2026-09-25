// Byline: Claude Code · Opus 5.5 · 2026-09-24
// Same real data, two different categories: the strip takes whatever category the records carry and colors it.
// Data: 2024 bout tone results for the texts from her phone (645 bouts, bout-tone-v1, no message text), from
// raw_duck.msg_bouts_20260924 ⋈ msg_bout_labels_20260924 (Probata scripts/jev_eval/mood_strip_fixture.sql).
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import herPhone2024 from "./__fixtures__/her-phone-2024-tone.json";
import { CategoryStrip } from "./category-strip";
import { rowsFromRecords } from "./rows";

interface Bout {
  id: string;
  start: string;
  end: string;
  stretches: [string, number, string][];
}

// One flat record per tone stretch: the shape any query can return.
const STRETCHES = (herPhone2024 as unknown as Bout[]).flatMap((b) =>
  b.stretches.map(([tone, n, driver]) => ({ day: b.start.slice(0, 10), bout: b.id, time: `${b.start.slice(11)}–${b.end.slice(11)}`, tone, n, driver })),
);

type Stretch = (typeof STRETCHES)[number];
const byTone = (records: Stretch[]) =>
  rowsFromRecords(records, { row: (r) => r.day, block: (r) => r.bout, category: (r) => r.tone, weight: (r) => r.n, blockTitle: (r) => r.time });
const byDriver = (records: Stretch[]) =>
  rowsFromRecords(records, { row: (r) => r.day, block: (r) => r.bout, category: (r) => r.driver, weight: (r) => r.n, blockTitle: (r) => r.time });

// Tones carry meaning, so they keep the bout review page's colors; any other category gets palette colors.
const TONE_COLORS = {
  affectionate: "#c2477a", friendly: "#3d8b5a", neutral: "#9aa1ab", tense: "#c98a1c",
  hostile: "#c0392b", distressed: "#7a4bb5", conciliatory: "#1f8a8a",
};
const TONE_ORDER = Object.keys(TONE_COLORS);

function Selectable(args: React.ComponentProps<typeof CategoryStrip>) {
  const [row, setRow] = useState<string | null>(null);
  return (
    <div className="grid max-w-[1100px] gap-3 p-4">
      <CategoryStrip {...args} selectedRowId={row} onSelectRow={setRow} />
      <p className="text-sm text-muted-foreground">{row ? `Picked ${row}` : "Click a row."}</p>
    </div>
  );
}

const meta = {
  title: "Platform/Blocks/Category strip",
  component: CategoryStrip,
  render: (args) => <Selectable {...args} />,
  args: { rows: [], unit: "msgs" },
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof CategoryStrip>;

export default meta;
type Story = StoryObj<typeof meta>;

/** Rows = days, blocks = bouts, categories = tone (colors pinned because tones carry meaning). */
export const Tones: Story = {
  args: {
    rows: byTone(STRETCHES),
    colors: TONE_COLORS,
    order: TONE_ORDER,
    description: "Each row is a day; each block is a bout, split into its tone stretches.",
  },
};

/** Same records, category = who drove each stretch; colors are assigned automatically. */
export const WhoDrove: Story = {
  args: {
    rows: byDriver(STRETCHES),
    description: "Each row is a day; each block is a bout, split by who drove each stretch.",
  },
};

/** One month of tones. */
export const OneMonth: Story = {
  args: {
    rows: byTone(STRETCHES.filter((r) => r.day.startsWith("2024-10"))),
    colors: TONE_COLORS,
    order: TONE_ORDER,
  },
};
