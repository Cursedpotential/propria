// Byline: Codex · GPT-5 · 2026-09-12 (DEV/LIVE fail-closed UI isolation and deep-link hydration contract)
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

function source(path) {
  return readFileSync(new URL(path, import.meta.url), "utf8");
}

const context = source("../src/lib/fixed-case-context.tsx");
const intake = source("../src/components/intake/unified-intake.tsx");
const preview = source("../src/components/sbv/proffer-preview-client.tsx");
const client = source("../src/lib/api-client.ts");
const selector = source("../src/components/intake/matter-mode-selector.tsx");
const header = source("../src/components/layout/header.tsx");
const matterWorkspace = source("../src/components/matters/matter-workspace.tsx");
const types = source("../src/lib/shared/types.ts");

// AMENDED 2026-09-22 (Claude Code · Opus 5): the wire values are unchanged —
// DEV/LIVE on every request and every durable record — but the VISIBLE labels
// are now Test / Live (owner 2026-09-22 09:08: the non-test side is "Live"; he
// hates "LIVE"). The assertion below therefore pins the label map instead of
// the raw `{mode}` interpolation, and still pins the wire values.
test("DEV and LIVE are explicit global modes, labelled Dev / Live", () => {
  assert.match(types, /type MatterMode = "DEV" \| "LIVE"/);
  assert.match(selector, /const MODES: MatterMode\[\] = \["DEV", "LIVE"\]/);
  assert.match(selector, /MODE_LABEL: Record<MatterMode, string> = \{ DEV: "Dev", LIVE: "Live" \}/);
  assert.match(selector, /\{MODE_LABEL\[mode\]\} mode active/);
  assert.match(header, /<MatterModeSelector compact/);
});

test("mode switching clears matter scope and remounts every import or preview cache", () => {
  assert.match(context, /setMatter\(null\)/);
  assert.match(context, /setLoading\(true\)/);
  assert.match(intake, /<UnifiedIntakeMode key=\{mode\} mode=\{mode\}/);
  assert.match(preview, /<ModeScopedPreviewClient key=\{mode\} mode=\{mode\}/);
  assert.match(matterWorkspace, /<ModeScopedMatterWorkspace key=\{mode\} mode=\{mode\}/);
  assert.match(matterWorkspace, /getMatter\(matterId, mode\)/);
  assert.match(matterWorkspace, /listMatters\(50, 0, mode\)/);
  assert.match(preview, /key=\{`\$\{mode\}:\$\{previewHandle\}`\}/);
});

test("all import and preview boundaries carry and verify the active mode", () => {
  for (const phrase of [
    "Staged acquisition did not confirm DEV/LIVE mode",
    "source browser did not confirm",
    "source inspection did not confirm",
    "source-context receipt did not confirm",
    "started preview did not confirm",
    "preview did not confirm",
    "preview messages did not confirm",
    "decision response did not confirm",
  ]) assert.match(client, new RegExp(phrase));
  assert.match(client, /new URLSearchParams\(\{ mode \}\)/);
  assert.match(preview, /event\.matter_mode !== mode/);
  assert.match(intake, /event\.matter_mode !== mode/);
});

test("a URL preview handle is accepted only for its matching mode", () => {
  assert.match(preview, /parseOperatingMode\(query.get\("mode"\)\) !== mode\) return ""/);
  assert.match(preview, /url\.searchParams\.set\("mode", mode\)/);
});

test("a direct LIVE preview deep link hydrates mode before resolving its handle", () => {
  assert.match(context, /function initialMatterMode\(\): \{ mode: MatterMode; error: string \| null \}/);
  assert.match(context, /mode: parseOperatingMode\(requestedMode\)/);
  assert.match(context, /useState<MatterMode>\(initialSelection.mode\)/);
  assert.doesNotMatch(context, /useState<MatterMode>\("DEV"\)/);

  const handleInitializer = preview.indexOf("useState(() => initialHandle(mode))");
  const activeHandleInitializer = preview.indexOf("useRef(initialUrlHandle)");
  const locationCanonicalizer = preview.indexOf("window.history.replaceState");
  assert.ok(handleInitializer >= 0, "the URL handle must initialize synchronously for the validated mode");
  assert.ok(activeHandleInitializer > handleInitializer, "the active correlation ref must start with the URL handle");
  assert.ok(locationCanonicalizer > activeHandleInitializer, "URL canonicalization must run after initial correlation state exists");
  assert.doesNotMatch(preview, /setDraftHandle\(handle\);\s*activateHandle\(handle\)/);
});
