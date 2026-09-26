// Byline: Claude Code · Fable 5.1 · 2026-09-21
// Owner requirement 2026-09-20: a missing attachment payload is flagged where the
// message is reviewed — one small flag on the item, never a banner — and a missing
// photo keeps its slot in the Media view so a gap is visible as a gap.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const read = (path) => readFileSync(new URL(path, import.meta.url), "utf8");

test("the message browser offers grid, conversation and media, and counts missing payloads", () => {
  const browser = read("../src/components/sbv/message-browser.tsx");
  assert.match(browser, /type ViewMode = "thread" \| "table" \| "media"/);
  assert.match(browser, /useState<ViewMode>\("table"\)/, "dense rows stay the default view");
  for (const id of ["message-browser-mode-thread", "message-browser-mode-table", "message-browser-mode-media", "message-browser-missing-payloads"]) {
    assert.ok(browser.includes(`data-testid="${id}"`), id);
  }
  assert.match(browser, /<MediaOnlyGrid rows=\{rows\}/);
});

test("a missing payload is a flag on the attachment, a count on the row, and a kept slot in the media grid", () => {
  assert.match(read("../src/lib/shared/types.ts"), /payload_missing\?: boolean;/);
  assert.match(read("../src/hooks/use-preview-messages.ts"), /missingPayloadCount: message\.attachments\.filter\(\(attachment\) => attachment\.payload_missing\)\.length/);
  assert.match(read("../src/components/sbv/message-browser-grid.tsx"), /\$\{entry\.missingPayloadCount\} missing/);
  const preview = read("../src/components/sbv/attachment-preview.tsx");
  assert.ok(preview.includes('data-testid="attachment-payload-missing"'));
  assert.ok(preview.includes("Missing from this backup"));
  assert.match(read("../src/components/sbv/media-only-grid.tsx"), /\|\| attachment\.payload_missing\) items\.push\(attachment\)/);
});
