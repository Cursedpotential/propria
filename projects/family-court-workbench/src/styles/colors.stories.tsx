// Byline: Claude Code · Sonnet 5 · 2026-09-07
//
// Design-system color reference story. Exists specifically to document and
// visually enforce the owner's color rule (2026-09-07 13:18-13:20,
// addendum 13:20, verbatim intent):
//
//   "clean but feature filled, dark mode, NO neon, not super rounded and
//   comical either" — and then, explicitly: "no bright ass colors."
//
// Rule, in force everywhere in this app (see tokens.css for the source of
// truth): the accent and every semantic color (good/warn/critical) are
// DESATURATED and MID-LUMINANCE. No pure saturated hues, no high-chroma
// chips, no neon, no glow. Status is communicated through a muted fill +
// 1px border + font-weight, never a loud color alone. The exact values
// below are the owner-specified reference colors:
//   accent   #5b7a99  steel blue
//   good     #4f7d5c  muted sage
//   warn     #a9853a  muted ochre
//   critical #a4524a  muted brick
import type { Meta, StoryObj } from "@storybook/react-vite";

const TOKENS = [
  { name: "bg", var: "--bg" },
  { name: "surface", var: "--surface" },
  { name: "surface-raised", var: "--surface-raised" },
  { name: "border", var: "--border-strong" },
  { name: "text-primary", var: "--text-primary" },
  { name: "text-secondary", var: "--text-secondary" },
  { name: "accent (#5b7a99 steel blue)", var: "--accent" },
  { name: "good (#4f7d5c muted sage)", var: "--good" },
  { name: "warn (#a9853a muted ochre)", var: "--warn" },
  { name: "critical (#a4524a muted brick)", var: "--critical" },
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
    <div data-theme={theme} className="rounded-[var(--radius-md)] border border-border bg-bg p-4">
      <h3 className="mb-3 text-xs font-semibold uppercase tracking-wide text-text-tertiary">{theme}</h3>
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
        Owner rule: <strong>no bright ass colors.</strong> Every swatch below is desaturated and mid-luminance by
        construction — there is no saturated-hue escape hatch anywhere in this palette, in either theme.
      </p>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Palette theme="dark" />
        <Palette theme="light" />
      </div>
    </div>
  ),
};
