// Byline: Codex · GPT-6.1-Sol · 2026-10-05.
import type { CourtCase, MatterDetail, MatterMode } from "@/lib/shared/types";

/** Resolve input-only rollout aliases; omission is Live, unknown values fail.
 * Input: URL mode. Output: canonical policy. Effects: none.
 * Use for URL hydration, never as a case identity or authentication selector.
 */
export function parseOperatingMode(value: string | null): MatterMode {
  if (value === null || value === "LIVE" || value === "REAL") return "LIVE";
  if (value === "DEV" || value === "TEST") return "DEV";
  throw new Error(`Unknown operating mode: ${value}`);
}

/** Select only the engine-admitted court; a primary flag does not grant scope.
 * Input: admitted matter detail. Output: matching court or null. Effects: none.
 * Use for the case shell, never discover scope from primary/sole court heuristics.
 */
export function selectAdmittedCourtCase(matter: MatterDetail | null): CourtCase | null {
  return matter?.court_cases.find((courtCase) => courtCase.id === matter.admitted_court_case_id) ?? null;
}
