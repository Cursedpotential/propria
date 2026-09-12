// Byline: Claude Code · Sonnet 5 · 2026-09-07
import type { Meta, StoryObj } from "@storybook/react-vite";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./card";

const meta: Meta = { title: "UI/Card" };
export default meta;

export const Basic: StoryObj = {
  render: () => (
    <Card className="max-w-sm">
      <CardHeader>
        <CardTitle>Controlling orders</CardTitle>
        <CardDescription>3 on record</CardDescription>
      </CardHeader>
      <CardContent className="text-sm text-text-secondary">Body content goes here.</CardContent>
    </Card>
  ),
};

export const DarkAndLight: StoryObj = {
  render: () => (
    <div className="flex gap-4">
      {(["dark", "light"] as const).map((theme) => (
        <div key={theme} data-theme={theme} data-pr-theme={theme} className="bg-bg p-4">
          <Card className="w-64">
            <CardHeader>
              <CardTitle>{theme}</CardTitle>
            </CardHeader>
            <CardContent className="text-sm text-text-secondary">Sample card content.</CardContent>
          </Card>
        </div>
      ))}
    </div>
  ),
};
