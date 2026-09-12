// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { describe, expect, it } from "vitest";
import { isUnavailable } from "./store";

describe("isUnavailable", () => {
  it("detects an unavailable store response", () => {
    expect(isUnavailable({ available: false, reason: "native module failed to load" })).toBe(true);
  });
  it("does not misclassify real data", () => {
    expect(isUnavailable({ configured: true })).toBe(false);
  });
});
