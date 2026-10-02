// Byline: Claude Code · Opus 5 · 2026-09-20 (Glide theme from the live design tokens)
// Byline: Claude Code · Opus 5.5 · 2026-10-01 (moved out of message-browser-grid.tsx so the Case page grid shares it)
import type { Theme } from "@glideapps/glide-data-grid";
import { useMemo } from "react";

import { useTheme } from "@/components/layout/theme-provider";

function cssVariable(element: HTMLElement, name: string, fallback: string) {
  const value = getComputedStyle(element).getPropertyValue(name).trim();
  return value || fallback;
}

/** Used only when a custom property is missing; mirrors `src/app/globals.css`. */
const FALLBACK_PALETTE = {
  light: {
    primary: "#4051b9",
    accent: "#e9ecfb",
    foreground: "#1d2228",
    mutedForeground: "#687078",
    card: "#fffefb",
    muted: "#ebe8e0",
    border: "#d5d1c9",
  },
  dark: {
    primary: "#8591f0",
    accent: "#313a66",
    foreground: "#f0f1ef",
    mutedForeground: "#b1b8bd",
    card: "#242e36",
    muted: "#2c373f",
    border: "#43505a",
  },
} as const;

/**
 * Glide paints to a canvas, so it cannot inherit the surface tokens through CSS.
 * The palette is read back from the live custom properties whenever the resolved
 * light/dark theme changes, which keeps the grid inside the existing design system.
 */
export function useGridTheme(host: HTMLElement | null): Partial<Theme> | undefined {
  const { resolvedTheme } = useTheme();

  const fallback = FALLBACK_PALETTE[resolvedTheme === "dark" ? "dark" : "light"];

  return useMemo(() => {
    if (!host) return undefined;
    return {
      accentColor: cssVariable(host, "--primary", fallback.primary),
      accentLight: cssVariable(host, "--accent", fallback.accent),
      textDark: cssVariable(host, "--foreground", fallback.foreground),
      textMedium: cssVariable(host, "--muted-foreground", fallback.mutedForeground),
      textLight: cssVariable(host, "--muted-foreground", fallback.mutedForeground),
      textHeader: cssVariable(host, "--foreground", fallback.foreground),
      bgCell: cssVariable(host, "--card", fallback.card),
      bgCellMedium: cssVariable(host, "--muted", fallback.muted),
      bgHeader: cssVariable(host, "--muted", fallback.muted),
      bgHeaderHasFocus: cssVariable(host, "--accent", fallback.accent),
      bgHeaderHovered: cssVariable(host, "--accent", fallback.accent),
      borderColor: cssVariable(host, "--border", fallback.border),
      horizontalBorderColor: cssVariable(host, "--border", fallback.border),
      fontFamily: cssVariable(host, "--font-sans", "system-ui, sans-serif"),
      baseFontStyle: "13px",
      headerFontStyle: "600 12px",
    };
  }, [fallback, host]);
}
