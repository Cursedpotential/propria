// Byline: OpenAI Codex · GPT-5 · 2026-09-12
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { ThemeProvider, useTheme } from "./theme-provider";

function ThemeProbe() {
  const { theme, toggle } = useTheme();
  return <button onClick={toggle}>{theme}</button>;
}

afterEach(() => {
  cleanup();
  window.localStorage.clear();
});

describe("ThemeProvider", () => {
  it("keeps legacy and shared theme selectors synchronized", () => {
    window.localStorage.setItem("fct-console-theme", "dark");
    render(
      <ThemeProvider>
        <ThemeProbe />
      </ThemeProvider>,
    );

    expect(document.documentElement).toHaveAttribute("data-theme", "dark");
    expect(document.documentElement).toHaveAttribute("data-pr-theme", "dark");
    expect(document.documentElement).toHaveClass("theme-dark");

    fireEvent.click(screen.getByRole("button", { name: "dark" }));

    expect(document.documentElement).toHaveAttribute("data-theme", "light");
    expect(document.documentElement).toHaveAttribute("data-pr-theme", "light");
    expect(document.documentElement).toHaveClass("theme-light");
    expect(document.documentElement).not.toHaveClass("theme-dark");
  });
});
