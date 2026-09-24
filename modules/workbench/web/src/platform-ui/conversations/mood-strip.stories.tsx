// Byline: Claude Code · Opus 5.5 · 2026-09-24
// Stories use the real 2024 tone results for the texts from Katrina's phone (645 bouts, bout-tone-v1), exported from
// the catalog (raw_duck.msg_bouts_20260924 ⋈ msg_bout_labels_20260924) without message text; per-sender counts are
// a `senders` map so the component never assumes who is talking.
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import herPhone2024 from "./__fixtures__/her-phone-2024-tone.json";
import { MoodStrip } from "./mood-strip";
import { TONE_LABEL, type BoutTone } from "./tone";

// JSON arrays type as (string | number)[]; the export matches BoutTone (checked when the fixture was written).
const BOUTS = herPhone2024 as unknown as BoutTone[];

function WithSelection(args: React.ComponentProps<typeof MoodStrip>) {
  const [day, setDay] = useState<string | null>(null);
  const bouts = args.bouts.filter((b) => b.start.startsWith(day ?? "-"));
  return (
    <div className="grid max-w-[1100px] gap-3 p-4">
      <MoodStrip {...args} selectedDay={day} onSelectDay={setDay} />
      {day ? (
        <ul className="grid gap-1 text-sm">
          {bouts.map((b) => (
            <li key={b.id}>
              <span className="font-mono text-muted-foreground">{b.start.slice(11)}–{b.end.slice(11)}</span> · {b.messages}{" "}
              messages · {Object.entries(b.senders).map(([name, n]) => `${name} ${n}`).join(" · ")} ·{" "}
              {b.stretches.map(([tone, n, driver]) => `${TONE_LABEL[tone]} ${n} (${driver})`).join(" → ")}
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-sm text-muted-foreground">Click a row to pick that day.</p>
      )}
    </div>
  );
}

const meta = {
  title: "Platform/Conversations/Mood strip",
  component: MoodStrip,
  render: (args) => <WithSelection {...args} />,
  args: {
    bouts: BOUTS,
    description: "Each row is a day; each block is a bout, split into its tone stretches. Click a row to open that day.",
  },
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof MoodStrip>;

export default meta;
type Story = StoryObj<typeof meta>;

/** All of 2024 on her phone, the same chart as the bout review page. */
export const WholePeriod: Story = {};

/** One month (October 2024). */
export const OneMonth: Story = {
  args: { bouts: BOUTS.filter((b) => b.start.startsWith("2024-10")) },
};
