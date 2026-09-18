// Byline: Claude Code · Sonnet 5 · 2026-09-07
import type { Meta, StoryObj } from "@storybook/react-vite";
import { Badge } from "./badge";

const meta: Meta<typeof Badge> = {
  title: "UI/Badge",
  component: Badge,
  args: { children: "on track" },
};
export default meta;

type Story = StoryObj<typeof Badge>;

export const Neutral: Story = { args: { tone: "neutral" } };
export const Accent: Story = { args: { tone: "accent", children: "queued" } };
export const Good: Story = { args: { tone: "good", children: "in force" } };
export const Warn: Story = { args: { tone: "warn", children: "deadline soon" } };
export const Critical: Story = { args: { tone: "critical", children: "overdue" } };

/**
 * Owner rule (2026-09-07 addendum): "no bright ass colors" — every tone here
 * is a MUTED fill + 1px border, never a saturated color block. Rendered in
 * both themes so a reviewer can eyeball that neither theme sneaks in a
 * bright/neon variant.
 */
export const AllTonesDarkAndLight: Story = {
  render: () => (
    <div className="flex flex-col gap-3">
      {(["dark", "light"] as const).map((theme) => (
        <div key={theme} data-theme={theme} data-pr-theme={theme} className="flex items-center gap-2 rounded-md bg-bg p-4">
          <Badge tone="neutral">neutral</Badge>
          <Badge tone="accent">accent</Badge>
          <Badge tone="good">good</Badge>
          <Badge tone="warn">warn</Badge>
          <Badge tone="critical">critical</Badge>
        </div>
      ))}
    </div>
  ),
};
