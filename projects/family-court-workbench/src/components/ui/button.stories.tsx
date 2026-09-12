// Byline: Claude Code · Sonnet 5 · 2026-09-07
import type { Meta, StoryObj } from "@storybook/react-vite";
import { Button } from "./button";

const meta: Meta<typeof Button> = {
  title: "UI/Button",
  component: Button,
  args: { children: "Continue" },
};
export default meta;

type Story = StoryObj<typeof Button>;

export const Solid: Story = { args: { variant: "solid" } };
export const Outline: Story = { args: { variant: "outline" } };
export const Ghost: Story = { args: { variant: "ghost" } };
export const Destructive: Story = { args: { variant: "destructive", children: "Delete order" } };
export const Disabled: Story = { args: { variant: "solid", disabled: true } };

export const AllVariantsDarkAndLight: Story = {
  render: () => (
    <div className="flex flex-col gap-4">
      {(["dark", "light"] as const).map((theme) => (
        <div key={theme} data-theme={theme} className="flex items-center gap-2 rounded-md bg-bg p-4">
          <Button variant="solid">Solid</Button>
          <Button variant="outline">Outline</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="destructive">Destructive</Button>
        </div>
      ))}
    </div>
  ),
};
