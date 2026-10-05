// Byline: Codex · GPT-6.1-Sol · 2026-10-05.
import type { MatterMode } from "@/lib/shared/types";

/** Resolve input-only rollout aliases; omission is Live, unknown values fail.
 * Input: URL mode. Output: canonical policy. Effects: none.
 * Use for URL hydration, never as a case identity or authentication selector.
 */
export function parseOperatingMode(value: string | null): MatterMode {
  if (value === null || value === "LIVE" || value === "REAL") return "LIVE";
  if (value === "DEV" || value === "TEST") return "DEV";
  throw new Error(`Unknown operating mode: ${value}`);
}
