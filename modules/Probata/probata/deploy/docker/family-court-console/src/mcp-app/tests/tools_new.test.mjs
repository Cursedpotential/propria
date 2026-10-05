// Byline: Claude Code · Fable 5.1 · 2026-09-07
// Byline: Claude Code · Sonnet 5 · 2026-09-07 — tool count 20 -> 27
// Byline: Claude Code · Sonnet 5 · 2026-09-08 — buildSurvivalGuide()/reviewCourtLanguage()/
// loadLexicon() are now store-first (content-store.ts) and therefore async; force-exit
// after close (see tests/core.test.mjs for why).
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import test, { after } from "node:test";
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";
import { buildSurvivalGuide, listSurvivalGuideEvents } from "../dist/survival-guide.js";
import { DOC_TYPES, loadLexicon, reviewCourtLanguage } from "../dist/court-language.js";
import { closeAllStoresForTests } from "../dist/store.js";

after(async () => {
  await closeAllStoresForTests();
  await new Promise((r) => setTimeout(r, 250));
  process.exit(0);
});

const VALID_PRESETS = new Set(["referee_objection", "appeal_of_right", "motion_response", "mail_service_addon"]);

test("survival_guide: every event id returns a context pack with non-empty sequence/do_not/deadlines and valid presets", async () => {
  const ids = listSurvivalGuideEvents();
  assert.equal(ids.length, 17, `expected 17 events, got ${ids.length}: ${ids.join(", ")}`);
  for (const id of ids) {
    const result = await buildSurvivalGuide({ event: id, format: "json" });
    const pack = result.context_pack;
    assert.equal(pack.id, id);
    assert.ok(Array.isArray(pack.sequence) && pack.sequence.length > 0, `${id} sequence empty`);
    assert.ok(Array.isArray(pack.do_not) && pack.do_not.length > 0, `${id} do_not empty`);
    assert.ok(Array.isArray(pack.deadlines) && pack.deadlines.length > 0, `${id} deadlines empty`);
    assert.ok(Array.isArray(pack.applicable_rules) && pack.applicable_rules.length >= 1, `${id} needs >=1 cited rule`);
    assert.ok(Array.isArray(pack.traps) && pack.traps.length >= 3, `${id} needs >=3 traps`);
    for (const deadline of pack.deadlines) {
      if (deadline.rule_preset !== null) {
        assert.ok(VALID_PRESETS.has(deadline.rule_preset), `${id} has an invalid rule_preset: ${deadline.rule_preset}`);
      }
    }
  }
});

test("survival_guide: full format includes the master template and current release state", async () => {
  const result = await buildSurvivalGuide({ event: "referee-hearing", format: "full" });
  assert.match(result.template, /Fixed sections/);
  assert.equal(typeof result.release_status.label, "string");
  assert.equal("release_warning" in result, false);
});

test("survival_guide: card format returns the card template", async () => {
  const result = await buildSurvivalGuide({ event: "referee-hearing", format: "card" });
  assert.match(result.template, /Card sections/);
});

test("survival_guide: include_case_facts merges case_facts without a configured file falling over", async () => {
  const result = await buildSurvivalGuide({ event: "motion", format: "json", include_case_facts: true });
  assert.ok("case_facts" in result);
});

test("survival_guide: unknown event throws a clear error", async () => {
  await assert.rejects(() => buildSurvivalGuide({ event: "not-a-real-event" }), /Unknown survival_guide event/);
});

test("court_language_review: an angry paragraph with a child-witness line flags >=5 categories including a stop, and returns a rewrite plan", async () => {
  const text =
    "He is a narcissist and an alcoholic who always shows up late and never calls, and it's obviously " +
    "outrageous. He probably did it on purpose because he's manipulative, and my son told me dad left " +
    "him home alone, which is basically abuse.";
  const result = await reviewCourtLanguage({ text, doc_type: "affidavit", mode: "review" });
  const categories = new Set(result.findings.map((f) => f.category));
  assert.ok(categories.size >= 5, `expected >=5 categories, got ${[...categories].join(", ")}`);
  assert.ok(result.stop_flags.includes("child_as_witness"), "expected child_as_witness in stop_flags");
  assert.ok(result.stop_flags.length >= 1, "expected at least one stop flag");
  assert.ok(Array.isArray(result.rewrite_plan) && result.rewrite_plan.length > 0);
  assert.ok(result.score < 50, `expected a low score for this text, got ${result.score}`);
});

test("court_language_review: a clean affidavit paragraph scores >= 90", async () => {
  const text =
    "1. On March 15, 2024, at 6:45 PM, I observed the other party arrive for the scheduled exchange. " +
    "2. A true and accurate copy of the exchange log is attached as Exhibit A.";
  const result = await reviewCourtLanguage({ text, doc_type: "affidavit", mode: "review" });
  assert.ok(result.score >= 90, `expected score >= 90, got ${result.score}: ${JSON.stringify(result.findings)}`);
  assert.deepEqual(result.stop_flags, []);
});

test("court_language_review: rewrite_plan mode still returns findings and a plan", async () => {
  const result = await reviewCourtLanguage({ text: "He is always late.", doc_type: "message_to_other_parent", mode: "rewrite_plan" });
  assert.equal(result.mode, "rewrite_plan");
  assert.ok(result.rewrite_plan.length > 0);
});

test("court_language_review: self-consistency — every lexicon example_after never trips its own lexicon", async () => {
  const lexicon = await loadLexicon();
  for (const entry of lexicon.entries) {
    const result = await reviewCourtLanguage({ text: entry.example_after, doc_type: "affidavit", mode: "review" });
    assert.deepEqual(
      result.findings.map((f) => f.category),
      [],
      `lexicon entry "${entry.category}"'s example_after re-triggers findings: ${entry.example_after}`,
    );
  }
});

test("court_language_review: self-consistency — every EXAMPLES.md 'After' text never trips the lexicon", async () => {
  const examplesPath = resolve("..", "skills/family-court-toolkit/references/court-language/EXAMPLES.md");
  const markdown = readFileSync(examplesPath, "utf8");
  const afters = [...markdown.matchAll(/^\*\*After[^*]*\*\*\s*"(.*)"\s*$/gm)].map((m) => m[1]);
  assert.ok(afters.length >= 6, `expected several worked "After" examples, found ${afters.length}`);
  for (const after of afters) {
    if (after.startsWith("[")) continue; // bracketed non-text instructions (e.g. "[REMOVED — ...]") are not prose to lint
    const result = await reviewCourtLanguage({ text: after, doc_type: "affidavit", mode: "review" });
    assert.deepEqual(
      result.findings.map((f) => f.category),
      [],
      `EXAMPLES.md "After" text re-triggers findings: ${after}`,
    );
  }
});

test("court_language_review: every doc_type has a non-empty safe_phrasebank loaded from EXAMPLES.md", async () => {
  for (const docType of DOC_TYPES) {
    const result = await reviewCourtLanguage({ text: "A brief, factual sentence.", doc_type: docType, mode: "review" });
    assert.ok(result.safe_phrasebank.length > 0, `${docType} has an empty safe_phrasebank`);
  }
});

test("MCP protocol: tools/list shows 28 tools including the two new ones, and both are callable", async () => {
  const transport = new StdioClientTransport({ command: process.execPath, args: [resolve("dist/server.js")] });
  const client = new Client({ name: "tools-new-test", version: "1.0.0" });
  try {
    await client.connect(transport);
    const tools = await client.listTools();
    const names = tools.tools.map((t) => t.name).sort();
    assert.equal(names.length, 28, names.join(", ")); // + case_record (Claude Code · Opus 5.5 · 2026-09-27)
    assert.ok(names.includes("survival_guide"));
    assert.ok(names.includes("court_language_review"));
    assert.ok(tools.tools.find((t) => t.name === "survival_guide").annotations?.readOnlyHint === true);
    assert.ok(tools.tools.find((t) => t.name === "court_language_review").annotations?.readOnlyHint === true);

    const guide = await client.callTool({ name: "survival_guide", arguments: { event: "referee-hearing", format: "card" } });
    assert.match(guide.structuredContent.template, /Card sections/);

    const review = await client.callTool({
      name: "court_language_review",
      arguments: { text: "He always lies and my son told me he was left alone.", doc_type: "incident_log", mode: "review" },
    });
    assert.ok(review.structuredContent.stop_flags.includes("child_as_witness"));
  } finally {
    await client.close();
  }
});
