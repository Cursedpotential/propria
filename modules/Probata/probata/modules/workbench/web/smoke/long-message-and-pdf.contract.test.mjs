// Byline: Claude Code · Sonnet 5.5 · 2026-10-03
// Owner report 2026-10-03: the Review preview of a run holding two 21k/7k-character pasted
// documents and a PDF-only message was unreadable. Long bodies fold behind "Show more"
// everywhere they are shown, the table line is capped, and a PDF the media route serves
// opens instead of reading "preview unavailable".
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const read = (path) => readFileSync(new URL(path, import.meta.url), "utf8");

test("long message bodies fold in the bubble and the detail pane, never rewritten", () => {
  const text = read("../src/components/sbv/collapsible-text.tsx");
  assert.match(text, /export function foldText/);
  assert.ok(text.includes('data-testid="collapsible-text-toggle"'));
  assert.match(text, /Show more/);
  assert.match(text, /Show less/);
  assert.doesNotMatch(text, /marked|dangerouslySetInnerHTML/, "the stored text is shown verbatim, never rendered as markdown");
  assert.match(read("../src/components/sbv/message-bubble.tsx"), /<CollapsibleText\s+text=\{message\.body\}\s+limit=\{BUBBLE_TEXT_LIMIT\}/);
  assert.match(read("../src/components/sbv/message-detail-panel.tsx"), /limit=\{DETAIL_TEXT_LIMIT\}/);
});

test("the table preview line is capped", () => {
  const hook = read("../src/hooks/use-preview-messages.ts");
  assert.match(hook, /export const BODY_LINE_LIMIT = 300/);
  assert.match(hook, /line\.slice\(0, BODY_LINE_LIMIT\)/);
});

test("an attachment-only message shows its attachment in the detail pane, and a served PDF opens", () => {
  assert.ok(read("../src/components/sbv/message-detail-panel.tsx").includes('data-testid="message-detail-attachments"'));
  const preview = read("../src/components/sbv/attachment-preview.tsx");
  assert.match(preview, /kind === "pdf" && url/);
  assert.ok(preview.includes('data-testid="attachment-preview-pdf-open"'));
});
