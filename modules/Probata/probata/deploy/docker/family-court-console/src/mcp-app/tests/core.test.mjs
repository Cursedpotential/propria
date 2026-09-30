// Byline: OpenAI Codex / GPT-5.6, 2026-08-13
// Byline: Claude Code · Sonnet 5 · 2026-09-08 — auditSources()/searchRecords() are now
// store-first (content-store.ts) and so open a mem:// connection under CUSTODY_CASE_DB=mem://;
// force-exit after close, matching store.test.mjs's proven pattern (the embedded engine's
// native handle otherwise keeps the process alive past every test finishing).
import assert from "node:assert/strict";
import test, { after } from "node:test";
import { auditSources, buildChronology, calculateDirectionalDeadline, routeIssue, searchRecords } from "../dist/core.js";
import { closeAllStoresForTests } from "../dist/store.js";

after(async () => {
  await closeAllStoresForTests();
  await new Promise((resolve) => setTimeout(resolve, 250));
  process.exit(0);
});

test("forward dates move a weekend forward", () => {
  const value = calculateDirectionalDeadline({ anchorDate: "2026-08-07", days: 1, direction: "after" });
  assert.equal(value.rawDate, "2026-08-08");
  assert.equal(value.adjustedDate, "2026-08-10");
  assert.equal(value.adjustment, "moved forward");
});

test("backward dates move a weekend backward, never later", () => {
  const value = calculateDirectionalDeadline({ anchorDate: "2026-08-09", days: 1, direction: "before" });
  assert.equal(value.rawDate, "2026-08-08");
  assert.equal(value.adjustedDate, "2026-08-07");
  assert.equal(value.adjustment, "moved backward");
});

test("a listed holiday adjusts in the counting direction", () => {
  const value = calculateDirectionalDeadline({ anchorDate: "2026-08-31", days: 7, direction: "after", holidays: ["2026-09-07"] });
  assert.equal(value.rawDate, "2026-09-07");
  assert.equal(value.adjustedDate, "2026-09-08");
});

test("invalid calendar dates are rejected", () => {
  assert.throws(() => calculateDirectionalDeadline({ anchorDate: "2026-02-30", days: 1, direction: "after" }), /valid calendar date/);
});

test("route flags immediate danger and unsafe legal lanes", () => {
  const value = routeIssue("There is immediate danger, a PPO, and an out of state order.");
  assert.equal(value.status, "STOP_AND_VERIFY");
  assert.deepEqual(value.flags, ["IMMEDIATE_DANGER", "DV_PPO_CPS", "UCCJEA_INTERSTATE"]);
  assert.match(value.nextActions[0], /911/);
});

test("source audit keeps provisional currency visible", async () => {
  const value = await auditSources(["MCL-722"]);
  assert.equal(value.sources[0].status, "PROVISIONAL_CURRENCY_NOT_CLEARED");
});

test("source search searches curated records only", async () => {
  const value = await searchRecords("PPO motion form");
  assert.equal(value[0].id, "CC-379");
});

test("chronology sorts events without inventing a source", () => {
  const value = buildChronology([
    { date: "2026-05-02", title: "Second" },
    { date: "2026-05-01", title: "First", source: "Order" },
  ]);
  assert.deepEqual(value.events.map((event) => event.title), ["First", "Second"]);
  assert.equal(value.events[1].source, "Unspecified");
});
