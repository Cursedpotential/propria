// Byline: Grok · grok-4.6 · 2026-08-18
// Unsaved screen state. The assistant cannot see this unless we send it.

export type LiveField = { label: string; value: string };

export type LiveSurface = {
  path: string;
  selection: string;
  fields: LiveField[];
};

const MAX_FIELD = 4000;
const MAX_TOTAL = 12000;

function fieldLabel(el: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement): string {
  if (el.name) return el.name;
  if (el.id) return el.id;
  if (el.getAttribute("aria-label")) return el.getAttribute("aria-label") ?? "field";
  if ("placeholder" in el && el.placeholder) return el.placeholder;
  return el.tagName.toLowerCase();
}

export function captureLiveSurface(path: string): LiveSurface {
  const selection = (window.getSelection()?.toString() ?? "").trim().slice(0, MAX_FIELD);
  const fields: LiveField[] = [];
  let used = selection.length;
  const nodes = document.querySelectorAll("textarea, input, select");
  for (const node of nodes) {
    if (!(node instanceof HTMLInputElement || node instanceof HTMLTextAreaElement || node instanceof HTMLSelectElement)) {
      continue;
    }
    if (node.classList.contains("term-input")) continue;
    if (node instanceof HTMLInputElement) {
      const type = (node.type || "text").toLowerCase();
      if (type === "password" || type === "hidden" || type === "file" || type === "checkbox" || type === "radio") {
        continue;
      }
    }
    const value = node.value.trim();
    if (!value) continue;
    const clipped = value.slice(0, MAX_FIELD);
    used += clipped.length;
    if (used > MAX_TOTAL) break;
    fields.push({ label: fieldLabel(node), value: clipped });
  }
  return { path, selection, fields };
}

export function liveSurfaceKey(): string {
  return "lw-chat-live";
}
