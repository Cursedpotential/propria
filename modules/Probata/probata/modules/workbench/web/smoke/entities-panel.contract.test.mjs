// Byline: Claude Code · Opus 5.5 · 2026-09-25
// Byline: Claude Code · Opus 5.5 · 2026-09-27 (DF-27: Mark-as-event pinned to the detail panel only)
// Owner flow 2026-09-25 19:15: "Extract Entities" proposes; the owner corrects;
// running the workflow commits. Events follow the same pattern, and any message
// can be marked as an event worth recalling. Nothing is written by extraction.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const read = (path) => readFileSync(new URL(path, import.meta.url), "utf8");

test("the panel proposes on Extract and commits only a validated set through the workflow", () => {
  const panel = read("../src/components/entities/entities-panel.tsx");
  assert.ok(panel.includes('data-testid="extract-entities-button"'));
  assert.ok(panel.includes("Extract entities"));
  assert.ok(panel.includes('data-testid="run-entity-workflow-button"'));
  assert.match(panel, /disabled=\{!report\?\.ok \|\| writing \|\| committing\}/, "Run workflow needs a passing validation");
  assert.match(panel, /commit\.mutateAsync\(\{ digest: report\.digest, key: newIdempotencyKey\("commit"\) \}\)/);
  assert.match(panel, /validation\.basis === data && !writing/, "a validation is hidden once the proposals change");
  assert.match(panel, /<EntityCard/);
  assert.match(panel, /<EventCard/);
  assert.match(panel, /<ValidationChecklist report=\{report\} \/>/);
});

test("every owner act goes through the BFF with an idempotency key", () => {
  const client = read("../src/lib/entity-extraction-client.ts");
  for (const route of ["/api/entities/extract", "/api/entities/corrections", "/api/entities/validate", "/api/entities/commit", "/api/events/from-record"]) {
    assert.ok(client.includes(route), route);
  }
  assert.ok(client.includes('headers["Idempotency-Key"] = key'));
  const hooks = read("../src/hooks/use-entity-extraction.ts");
  assert.match(hooks, /useIsMutating\(\{ mutationKey: writesKey\(mode, previewHandle\) \}\)/);
});

test("any message can be marked as an event worth recalling, and flags stay small", () => {
  const mark = read("../src/components/entities/mark-event-button.tsx");
  assert.ok(mark.includes("Mark as event worth recalling"));
  assert.match(mark, /actions\.markEvent\(recordId, title\.trim\(\) \|\| undefined\)/);
  // DF-27 (2026-09-27): the button renders once, on the selected message's detail panel.
  assert.match(read("../src/components/sbv/message-detail-panel.tsx"), /<MarkEventButton key=\{message\.message_id\} previewHandle=\{previewHandle\} mode=\{mode\} recordId=\{message\.message_id\} compact \/>/);
  assert.doesNotMatch(read("../src/components/entities/record-peek.tsx"), /MarkEventButton/);
  const card = read("../src/components/entities/event-card.tsx");
  assert.ok(card.includes("worth recalling"));
  assert.ok(card.includes("visible from"), "an event shows the horizon its sources carry");
  for (const file of ["entities-panel.tsx", "entity-card.tsx", "event-card.tsx"]) {
    assert.doesNotMatch(read(`../src/components/entities/${file}`), /<Alert\b|<Banner\b/, `${file}: one small flag, never a banner`);
  }
});
