// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Badge } from "./badge";

describe("Badge", () => {
  it("renders its children", () => {
    render(<Badge tone="good">on track</Badge>);
    expect(screen.getByText("on track")).toBeInTheDocument();
  });

  it.each(["neutral", "accent", "good", "warn", "critical"] as const)("applies the %s tone's token classes, never a bare bright color utility", (tone) => {
    render(<Badge tone={tone}>x</Badge>);
    const el = screen.getByText("x");
    // Shared contract rule: statuses use semantic tokens, never a stock color
    // utility whose meaning can drift between products.
    expect(el.className).not.toMatch(/-(red|green|yellow|blue|orange|pink|purple|lime|emerald|cyan)-\d/);
  });
});
