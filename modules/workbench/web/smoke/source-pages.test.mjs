// Byline: Codex · GPT-6 · 2026-09-23. Runtime continuation checks for Sources.
import assert from "node:assert/strict";
import test from "node:test";

import { mergeSourcePages, sourceContinuation } from "../src/components/sources/source-pages.ts";

function page(keys, token = null, truncated = false) {
  return {
    objects: keys.map((key) => ({ source_ref: `r2://bucket/${key}`, key })),
    prefixes: [{ prefix: "folder/", name: "folder" }],
    continuation_token: token,
    is_truncated: truncated,
  };
}

test("first page exposes the next token and the next page adds unique files", () => {
  const first = page(["a", "b"], "page-2", true);
  assert.equal(sourceContinuation([first], [undefined]).token, "page-2");
  const second = page(["b", "c"], "page-3", true);
  const merged = mergeSourcePages([first, second]);
  assert.deepEqual(merged.objects.map((item) => item.key), ["a", "b", "c"]);
  assert.deepEqual(merged.prefixes.map((item) => item.prefix), ["folder/"]);
  assert.equal(sourceContinuation([first, second], [undefined, "page-2"]).token, "page-3");
});

test("terminal page has no next request", () => {
  const state = sourceContinuation([page(["a"], "page-2", true), page(["c"])], [undefined, "page-2"]);
  assert.deepEqual(state, { complete: true });
});

test("failed next page leaves its token available for a deliberate retry", () => {
  const state = sourceContinuation([page(["a"], "page-2", true)], [undefined]);
  assert.equal(state.token, "page-2");
  assert.equal(state.complete, false);
});

test("repeated or missing continuation fails closed instead of looping", () => {
  const repeated = sourceContinuation(
    [page(["a"], "page-2", true), page(["a"], "page-2", true)],
    [undefined, "page-2"],
  );
  assert.equal(repeated.token, undefined);
  assert.match(repeated.issue, /repeated/);
  const missing = sourceContinuation([page(["a"], null, true)], [undefined]);
  assert.equal(missing.token, undefined);
  assert.match(missing.issue, /partial page/);
});
