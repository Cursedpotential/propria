# Dark Mode Token System

Design dark mode as its own deliberate system, not an inverted light theme.

## Elevation Scale
Define 4-5 grey/navy steps from deepest background to highest surface, each slightly lighter than the one below (no drop shadows for elevation): bg-base, bg-surface, bg-elevated, bg-overlay.
Never use pure black (#000) for bg-base or pure white for text.

## Accent Colors
Desaturate light-mode accent/brand colors by roughly 20 points before using in dark mode.

## Status Color Mapping
Red: alerts/overdue. Amber: warnings. Green: healthy/complete. Blue: neutral info.
Status color must always be paired with an icon or text label — never rely on color alone.

## Contrast Requirements (WCAG 2.2)
Normal text: minimum 4.5:1. Large text/headings: minimum 3:1. Verify with scripts/contrast_check.py.

## Token Implementation
Build as design tokens (CSS custom properties), named semantically (--color-status-danger, --bg-surface-1).
