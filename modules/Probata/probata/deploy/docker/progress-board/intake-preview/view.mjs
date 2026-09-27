// Byline: Claude Code · Sonnet 5 · 2026-09-14 | Workspace layout state only; no integration logic.
export function viewState(mode) {
  if (mode === "engine") return { mode, message: "Explorer only" };
  if (mode === "review") return { mode, message: "Review & metadata only" };
  return { mode: "split", message: "Explorer and Review docked together" };
}
