// Byline: Claude Code · Opus 5.5 · 2026-09-26
// Owner 2026-09-25 19:13 / 19:15: click a file in Review and see ALL of its metadata (embedded,
// sidecars, hashes) with corrections as attributed overlays; per message, who it is to / about,
// about the child, relevant, and a hindsight-only foreshadowing flag that never reaches the
// as-lived view. These pins read the source; they are not a browser or live proof.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const read = (path) => readFileSync(new URL(path, import.meta.url), "utf8");

const client = read("../src/lib/review-overlays-client.ts");
const screen = read("../src/components/metadata/file-metadata-screen.tsx");
const table = read("../src/components/metadata/metadata-field-table.tsx");
const review = read("../src/components/review/context-review.tsx");
const surface = read("../src/components/sbv/proffer-operator-preview.tsx");
const detail = read("../src/components/sbv/message-detail-panel.tsx");
const sourcePanel = read("../src/components/sbv/message-source-panel.tsx");

test("the Review file name and every attachment open the full metadata screen", () => {
  assert.match(surface, /data-testid="review-file-metadata"/);
  assert.match(surface, /<FileMetadataScreen open=\{metadataOpen\} onOpenChange=\{setMetadataOpen\} previewHandle=\{snapshot\.preview_handle\} mode=\{snapshot\.matter_mode\} \/>/);
  assert.match(sourcePanel, /All metadata/);
  assert.match(sourcePanel, /subjectSha256=\{metadataFor\?\.sha256\}/);
  assert.match(screen, /data-testid="file-metadata-screen"/);
  for (const section of ["This file", "Recorded metadata", "Sidecars", "Custody hashes"]) {
    assert.match(screen, new RegExp(`label="${section}"`), section);
  }
});

test("corrections are overlays: the recorded value stays, a correction is a new revision", () => {
  assert.match(client, /\/metadata\/corrections/);
  assert.match(table, /title="Recorded value \(never changed\)"/);
  assert.match(table, /supersedesRef: newest\?\.correction_ref \?\? ""/);
  assert.match(table, /Withdraw/);
  assert.match(screen, /source_value: input\.observed/);
  assert.doesNotMatch(`${screen}\n${table}`, /line-through/, "a recorded value is never struck out");
});

test("unavailable data is one small flag, never a banner", () => {
  for (const [name, text] of Object.entries({ screen, table, review })) {
    assert.doesNotMatch(text, /<Alert\b|<Banner\b|role="alert"/, `${name}: one small flag, never a banner`);
  }
  assert.match(screen, /embedded metadata not read/);
  assert.match(screen, /catalog not connected/);
  assert.match(review, /review unavailable/);
});

test("context review saves to / about / child / relevant; foreshadowing is a separate hindsight-only overlay", () => {
  assert.match(detail, /<ContextReviewPanel key=\{message\.message_id\} previewHandle=\{previewHandle\} mode=\{mode\} messageId=\{message\.message_id\} \/>/);
  assert.match(review, /getContextReview\(previewHandle, mode, messageId, "hindsight"\)/, "the owner surface asks for hindsight explicitly");
  assert.match(review, /postContextReview\(previewHandle, mode, messageId, \{\s+\.\.\.value,\s+supersedes_ref: current\?\.review_ref \?\? "",/);
  assert.match(review, /postForeshadowing\(previewHandle, mode, messageId, \{/);
  assert.match(review, /hindsight only/);
  for (const label of ["About the child", "Relevant", "Foreshadowing"]) assert.ok(review.includes(label), label);
  // The review body type carries no foreshadowing member; the flag has its own route.
  const reviewInput = client.slice(client.indexOf("export interface ContextReviewInput"), client.indexOf("export interface ForeshadowingInput"));
  assert.doesNotMatch(reviewInput, /foreshadowing/);
  assert.match(client, /\/context-review/);
  assert.match(client, /\/foreshadowing/);
  assert.match(client, /Present only on a hindsight read; never on an as-lived read/);
});
