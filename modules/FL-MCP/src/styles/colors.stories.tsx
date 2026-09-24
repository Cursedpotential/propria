// Byline: Claude Code · Sonnet 5 · 2026-09-07
//
// Updated by: OpenAI Codex · GPT-5 · 2026-09-12 — Probata graphite/indigo semantics.
// The vendored contract owns values; this story documents Family Court aliases.
import type { Meta, StoryObj } from "@storybook/react-vite";

const TOKENS = [
  { name: "bg", var: "--bg" },
  { name: "surface", var: "--surface" },
  { name: "surface-raised", var: "--surface-raised" },
  { name: "border", var: "--border-strong" },
  { name: "text-primary", var: "--text-primary" },
  { name: "text-secondary", var: "--text-secondary" },
  { name: "action / seal", var: "--accent" },
  { name: "verified positive", var: "--good" },
  { name: "pending / caution", var: "--warn" },
  { name: "failure / destructive", var: "--critical" },
] as const;

function Swatch({ name, cssVar }: { name: string; cssVar: string }) {
  return (
    <div className="flex items-center gap-3 text-sm">
      <div
        className="size-10 shrink-0 rounded-[var(--radius-sm)] border border-border-strong"
        style={{ background: `var(${cssVar})` }}
        aria-hidden
      />
      <div>
        <div className="text-text-primary">{name}</div>
        <code className="text-xs text-text-tertiary">{cssVar}</code>
      </div>
    </div>
  );
}

function Palette({ theme }: { theme: "dark" | "light" }) {
  return (
    <div data-theme={theme} data-pr-theme={theme} className="rounded-[var(--radius-md)] border border-border bg-bg p-4">
      <h3 className="mb-3 text-sm font-semibold text-text-secondary">{theme} theme</h3>
      <div className="grid grid-cols-2 gap-3">
        {TOKENS.map((t) => (
          <Swatch key={t.var} name={t.name} cssVar={t.var} />
        ))}
      </div>
    </div>
  );
}

const meta: Meta = {
  title: "Design System/Colors",
  parameters: { layout: "padded" },
};
export default meta;

export const Palettes: StoryObj = {
  render: () => (
    <div className="max-w-3xl space-y-4">
      <p className="text-sm text-text-secondary">
        Probata graphite/indigo uses indigo for deliberate action, indigo for focus, green only for verified completion,
        caution for pending or stale states, and teal only for neutral information. Every state remains text-labelled.
      </p>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Palette theme="dark" />
        <Palette theme="light" />
      </div>
    </div>
  ),
};
