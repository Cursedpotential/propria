// Byline: Claude Code · Sonnet 5 · 2026-09-07
// Regression evals for the 2026-09-07 routeIssue/auditSources/calculateDirectionalDeadline patch.
// Byline: Claude Code · Sonnet 5 · 2026-09-08 — auditSources() is now store-first
// (content-store.ts); force-exit after close (see tests/core.test.mjs for why).
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import test, { after } from "node:test";
import { auditSources, calculateDirectionalDeadline, routeIssue } from "../dist/core.js";
import { closeAllStoresForTests } from "../dist/store.js";

after(async () => {
  await closeAllStoresForTests();
  await new Promise((resolve) => setTimeout(resolve, 250));
  process.exit(0);
});

function readEvalPrompt(relativePath) {
  const metadata = JSON.parse(readFileSync(resolve(relativePath), "utf8"));
  assert.equal(typeof metadata.prompt, "string", `${relativePath} has no string prompt field`);
  return metadata.prompt;
}

test("(a) documented regression: Ohio relocation + 10-day hearing stops with UCCJEA_INTERSTATE and DEADLINE_PROXIMITY", () => {
  const value = routeIssue("moved the child to Ohio without notice, hearing in 10 days");
  assert.equal(value.status, "STOP_AND_VERIFY");
  assert.ok(value.flags.includes("UCCJEA_INTERSTATE"), `expected UCCJEA_INTERSTATE, got ${JSON.stringify(value.flags)}`);
  assert.ok(value.flags.includes("DEADLINE_PROXIMITY"), `expected DEADLINE_PROXIMITY, got ${JSON.stringify(value.flags)}`);
});

test("(b) buried-Toledo-Ohio relocation clause fires UCCJEA_INTERSTATE even in an otherwise-clean full-packet request", () => {
  const prompt = readEvalPrompt("../tests/evals/iteration-2/eval-1-buried-uccjea-stop/eval_metadata.json");
  const value = routeIssue(prompt);
  assert.ok(value.flags.includes("UCCJEA_INTERSTATE"), `expected UCCJEA_INTERSTATE, got ${JSON.stringify(value.flags)}`);
  assert.equal(value.status, "STOP_AND_VERIFY");
});

test("(c) secret-recording-on-tape scenario fires RECORDING_RISK", () => {
  const prompt = readEvalPrompt("../tests/evals/iteration-1/eval-2-recording-route-out/eval_metadata.json");
  const value = routeIssue(prompt);
  assert.ok(value.flags.includes("RECORDING_RISK"), `expected RECORDING_RISK, got ${JSON.stringify(value.flags)}`);
  assert.equal(value.status, "STOP_AND_VERIFY");
});

test("(d) a long, keyword-free intake recommends unclassified review instead of guided intake", () => {
  const text =
    "I have been trying to keep good paperwork for my case but I still feel behind. There are receipts, " +
    "notes, and messages I want to organize before I meet with anyone about this. I keep a journal of what " +
    "happens and want to make sure the timeline stays clear and easy to follow for whoever reviews it eventually.";
  assert.ok(text.length >= 240, `fixture must be at least 240 chars, got ${text.length}`);
  const value = routeIssue(text);
  assert.equal(value.status, "UNCLASSIFIED_REVIEW_RECOMMENDED");
  assert.deepEqual(value.flags, []);
  assert.ok(value.nextActions.some((action) => /21 days/.test(action)));
});

test("(e) a routine parenting-time modification with a distant hearing does not stop", () => {
  const value = routeIssue("I want to modify parenting time; the referee hearing is next month");
  assert.notEqual(value.status, "STOP_AND_VERIFY");
});

test("(f) audit_sources reports more than 100 total records when the ledger is present", async () => {
  const value = await auditSources();
  assert.ok(value.source_stats.ledgerLoaded, "expected the ledger to load");
  assert.ok(value.source_stats.total > 100, `expected total > 100, got ${value.source_stats.total}`);
});

test("(g) referee_objection preset from 2026-09-05 lands on 2026-09-28 (raw Saturday 2026-09-26 rolls over the weekend)", () => {
  const value = calculateDirectionalDeadline({ anchorDate: "2026-09-05", rule: "referee_objection" });
  assert.equal(value.rawDate, "2026-09-26");
  assert.equal(value.adjustedDate, "2026-09-28");
  assert.equal(value.cite, "MCR 3.215(E)(4)");
  assert.equal(value.status, "PROVISIONAL_VERIFY");
  assert.match(value.warning, /verify against the current rule text before reliance/);
});
