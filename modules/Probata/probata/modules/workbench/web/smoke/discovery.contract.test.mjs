// Byline: Codex · GPT-6 · 2026-10-11. Browser-free discovery race and mutation contracts.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { runInNewContext } from "node:vm";
import test from "node:test";
import ts from "typescript";
import React from "react";

const require = createRequire(import.meta.url);
const scope = { mode: "LIVE", matter_id: "matter", court_case_id: "court" };
const dataset = { dataset_id: "recorded", input_backend: "postgresql", selection_revision: "pinned", generation_ids: ["generation"], scope };
const source = readFileSync(new URL("../src/components/review/discovery-review.tsx", import.meta.url), "utf8");
const compiled = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, jsx: ts.JsxEmit.ReactJSX, module: ts.ModuleKind.CommonJS } }).outputText;
class ApiError extends Error { get isConflict() { return true; } }

function harness({ states = ["postgresql", "recorded", "described behavior"], start = async () => ({}), save = async () => ({}), run = undefined, items = [] } = {}) {
  if (run && !run.request) run = { ...run, request: { hypotheses: [] } };
  const effects = [], updates = [], navigation = [], calls = [];
  let index = 0;
  const module = { exports: {} };
  runInNewContext(compiled, { module, exports: module.exports, URLSearchParams, AbortController, crypto: { randomUUID: () => "stable-uuid" }, console,
    require: (name) => {
      if (name === "react") return { ...React, useState: (initial) => { const slot = index++; return [slot < states.length ? states[slot] : initial, (value) => updates.push([slot, value])]; }, useRef: (current) => ({ current }), useEffect: (effect) => effects.push(effect()), useMemo: (effect) => effect() };
      if (name === "@tanstack/react-query") return { useQuery: (options) => options.queryKey[0] === "discovery-datasets" ? { data: { items: [dataset] }, isSuccess: true } : { data: run, refetch: async () => ({}) }, useInfiniteQuery: () => ({ data: { pages: [{ items }] }, refetch: async () => ({}) }), useQueryClient: () => ({ setQueryData: (...args) => calls.push(["cache", ...args]) }) };
      if (name === "@tanstack/react-router") return { useRouter: () => ({ navigate: async (value) => navigation.push(value) }) };
      if (name === "@/lib/router-compat") return { AppLink: "a", useBrowserSearchParams: () => new URLSearchParams(run ? "run=33333333-3333-4333-8333-333333333333" : "") };
      if (name === "@/lib/api-client") return { ApiError };
      if (name === "@/lib/discovery-client") return { discoveryScopeKey: (scope) => JSON.stringify(scope), startDiscovery: (...args) => { calls.push(["start", ...args]); return start(...args); }, saveDiscoveryFeedback: (...args) => { calls.push(["feedback", ...args]); return save(...args); } };
      if (name === "@/components/read/read-location") return { readHref: (_params, fields) => `/read?${new URLSearchParams(Object.entries(fields).filter(([, value]) => value !== null))}` };
      if (name === "@/components/ui/button") return { Button: "button" };
      if (name === "@/components/mobile/mobile-ui") return { ErrorBox: "aside", Loading: "aside", LoadMore: "button" };
      return require(name);
    },
  });
  return { component: module.exports, effects, updates, navigation, calls, resetStates: (next) => { index = 0; states = next; } };
}
function elements(node) {
  if (!node || typeof node !== "object") return [];
  if (Array.isArray(node)) return node.flatMap(elements);
  return [node, ...elements(node.props?.children)];
}
const submit = (tree) => elements(tree).find((item) => item.type === "form").props.onSubmit({ preventDefault() {} });

test("changed scope aborts pending discovery, suppresses stale navigation, and duplicate submits cannot dispatch", async () => {
  let resolve;
  const pending = new Promise((finish) => { resolve = finish; });
  const h = harness({ start: () => pending });
  const tree = h.component.DiscoveryReview({ scope });
  submit(tree); submit(tree);
  assert.equal(h.calls.filter((call) => call[0] === "start").length, 1);
  const [, body, key, signal] = h.calls[0];
  assert.equal(body.run_id, key);
  h.effects.forEach((cleanup) => cleanup?.());
  assert.equal(signal.aborted, true);
  resolve({ request: { run_id: key } });
  await new Promise((finish) => setImmediate(finish));
  assert.deepEqual(h.navigation, []);
});

test("DEV forms disable mutations and even direct submit cannot dispatch", () => {
  const h = harness();
  const tree = h.component.DiscoveryReview({ scope: { ...scope, mode: "DEV" } });
  assert.equal(elements(tree).find((item) => item.type === "button" && item.props.type === "submit").props.disabled, true);
  submit(tree);
  assert.equal(h.calls.length, 0);
});

test("feedback becomes saved only after receipt and uses the displayed run revision", async () => {
  let resolve;
  const pending = new Promise((finish) => { resolve = finish; });
  const row = { candidate_id: "candidate", episode_id: "episode", title: "Possible behavior", score: .6 };
  const h = harness({ run: { revision: 7, score_revision: 1 }, items: [row], save: () => pending });
  const tree = h.component.DiscoveryReview({ scope });
  const candidate = elements(tree).find((item) => item.props?.row?.candidate_id === "candidate");
  h.resetStates([false, "relevant", "checked", [], ""]);
  const review = candidate.type({ ...candidate.props, refresh: async () => ({}) });
  h.updates.length = 0;
  submit(review); submit(review);
  const calls = h.calls.filter((call) => call[0] === "feedback");
  assert.equal(calls.length, 1);
  assert.equal(calls[0][3].expected_revision, 7);
  assert.equal(calls[0][3].feedback_id, calls[0][4]);
  assert.equal(h.updates.some(([, value]) => value?.feedback_id), false);
  resolve({ feedback: { feedback_id: "stable-uuid", kind: "relevant", revision: 8 }, run: { revision: 8 } });
  await new Promise((finish) => setImmediate(finish));
  assert.equal(h.updates.some(([, value]) => value?.feedback_id === "stable-uuid"), true);
});

test("source navigation accepts genuine imported IDs and never links browser file/source URLs", () => {
  const h = harness();
  for (const source_ref of ["file:///private/source", "javascript:alert(1)", "https://untrusted.invalid/source", "b2://vault/source"]) {
    assert.equal(h.component.discoveryRecordHref({ record_id: "33333333-3333-4333-8333-333333333333", source_ref }, scope), null);
  }
  assert.equal(h.component.discoveryRecordHref({ record_id: "33333333-3333-4333-8333-333333333333", imported_thread_id: "https://untrusted.invalid" }, scope), null);
  assert.equal(h.component.discoveryRecordHref({ record_id: "33333333-3333-4333-8333-333333333333", imported_thread_id: "actual_thread" }, scope), null);
  const href = h.component.discoveryRecordHref({ record_id: "33333333-3333-4333-8333-333333333333", imported_thread_id: "actual_thread", imported_message_id: "44444444-4444-4444-8444-444444444444" }, scope);
  assert.match(href, /^\/read\?/);
  assert.match(href, /around=44444444-4444-4444-8444-444444444444/);
  assert.doesNotMatch(href, /33333333/);
});


function text(node) {
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(text).join(" ");
  return node?.props ? text(node.props.children) : "";
}

test("behavior corrections and partial matches submit only selected current definitions, including none", async () => {
  const definitions = [{ hypothesis_id: "description-1", text: "Pressure about parenting time" }, { hypothesis_id: "description-2", text: "Financial threats" }];
  for (const kind of ["wrong_behavior", "partial_match"]) {
    for (const labels of [["description-2", "unrecognized"], []]) {
      const row = { candidate_id: "candidate", episode_id: "episode", title: "Possible behavior", score: .6, labels: ["description-1"], rationale: "Found description-1 and description-2" };
      const h = harness({ run: { revision: 7, score_revision: 1, request: { hypotheses: definitions } }, items: [row], save: async () => ({ feedback: {}, run: {} }) });
      const tree = h.component.DiscoveryReview({ scope });
      const candidate = elements(tree).find((item) => item.props?.row?.candidate_id === "candidate");
      h.resetStates([false, kind, "checked", labels, ""]);
      const review = candidate.type({ ...candidate.props, refresh: async () => ({}) });
      assert.match(text(review), /Pressure about parenting time/);
      assert.match(text(review), /Financial threats/);
      assert.doesNotMatch(text(review), /description-1|description-2/);
      assert.equal(elements(review).filter((item) => item.type === "input" && item.props.type === "checkbox").length, 2);
      submit(review);
      const call = h.calls.find((call) => call[0] === "feedback");
      assert.deepEqual(Array.from(call[3].labels), labels.length ? ["description-2"] : []);
      await new Promise((finish) => setImmediate(finish));
    }
  }
});

test("bounded coverage and failure stages remain visible without error stack content", () => {
  const h = harness({ run: { revision: 7, score_revision: 1, input_record_count: 200, input_truncated: true, status: "failed", failure_stage: "scoring", error: "private stack trace" } });
  const tree = h.component.DiscoveryReview({ scope });
  assert.match(text(tree), /Searched\s+200\s+source records/);
  assert.match(text(tree), /Record limit reached; later records were not searched/);
  assert.match(text(tree), /Discovery scoring failed\s+at scoring/);
  assert.doesNotMatch(text(tree), /private stack trace/);
});
