// Byline: Claude Code · Sonnet 5 · 2026-09-14
// Xplorer is the only pane in this shell; Review & metadata and AI Chat are
// docked inside it (see ReviewDockPanel.tsx). This script only reflects the
// engine's live selection in the header context strip -- it does not relay
// to a second frame (there isn't one).
const BRIDGE_SOURCE = "xplorer-intake-bridge";
const engineFrame = document.getElementById("engine-frame");
const ctxPath = document.getElementById("ctx-path");
const ctxSelection = document.getElementById("ctx-selection");
const ctxStatus = document.getElementById("ctx-status");

window.addEventListener("message", (event) => {
  const { data } = event;
  if (!data || typeof data !== "object" || data.source !== BRIDGE_SOURCE)
    return;
  if (event.source !== engineFrame.contentWindow) return; // only trust the engine pane
  ctxStatus.textContent = "Explorer connected";
  ctxStatus.dataset.prStatus = "positive";
  if (data.type !== "selection") return;
  ctxPath.textContent = data.currentPath || "not reported";
  const count = Number.isFinite(data.count)
    ? data.count
    : (data.selection || []).length;
  ctxSelection.textContent = `${count} item${count === 1 ? "" : "s"}`;
});
