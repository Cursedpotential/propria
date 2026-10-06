// Byline: Codex · GPT-6 · 2026-10-06
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import ts from "typescript";

// Transpile this pure helper only; no browser, API, service, or corpus access.
const source = readFileSync(new URL("./read-location.ts", import.meta.url), "utf8");
const compiled = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext } }).outputText;
const { readHref, importedReadingHref, isProcessingPreview, searchHitHref } = await import(`data:text/javascript;base64,${Buffer.from(compiled).toString("base64")}`);

/** Parse the query of a generated Read href for exact navigation assertions.
 * Input: local href. Output: query parameters; effects: none.
 * Pick in these tests to avoid relying on parameter order.
 */
function queryOf(href) {
  assert.equal(new URL(href, "https://test.invalid").pathname, "/read");
  return new URL(href, "https://test.invalid").searchParams;
}

test("source selection keeps search and mode but removes a stale thread and focus", () => {
  const original = new URLSearchParams("source=old&thread=old-thread&around=old-message&q=two+words&mode=LIVE");
  const next = queryOf(readHref(original, { source: "source/+?= Ω", thread: null, around: null }));
  assert.equal(next.get("source"), "source/+?= Ω");
  assert.equal(next.get("thread"), null);
  assert.equal(next.get("around"), null);
  assert.equal(next.get("q"), "two words");
  assert.equal(next.get("mode"), "LIVE");
  assert.equal(original.get("thread"), "old-thread", "navigation must not mutate current route state");
});

test("search results preserve original IDs and do not reuse an unrelated source ID", () => {
  const next = queryOf(searchHitHref(new URLSearchParams("source=unrelated&q=search&mode=LIVE"), "thread/+==", "a-message-id"));
  assert.equal(next.get("source"), null);
  assert.equal(next.get("thread"), "thread/+==");
  assert.equal(next.get("around"), "a-message-id");
  assert.equal(next.get("q"), "search");
  assert.equal(next.get("mode"), "LIVE");
});

test("all legacy preview selectors and the explicit preview-list route open previews", () => {
  for (const query of ["resource=one", "preview_handle=two", "attempt=three", "view=review", "view=other&attempt=three"]) {
    assert.equal(isProcessingPreview(new URLSearchParams(query)), true, query);
  }
  for (const query of ["", "source=file&thread=thread&around=message&q=words", "resource=", "attempt=+", "view=messages"]) {
    assert.equal(isProcessingPreview(new URLSearchParams(query)), false, query);
  }
});

test("preview return retains reading context and removes every preview alias", () => {
  const next = queryOf(importedReadingHref(new URLSearchParams("source=s&thread=t&around=m&q=words&mode=LIVE&view=review&resource=r&preview_handle=p&attempt=a")));
  assert.deepEqual(Object.fromEntries(next), { source: "s", thread: "t", around: "m", q: "words", mode: "LIVE" });
  assert.equal(isProcessingPreview(next), false);
});

test("clearing search or jumping to latest retains the independent selection", () => {
  const current = new URLSearchParams("source=s&thread=t&around=m&q=words");
  assert.deepEqual(Object.fromEntries(queryOf(readHref(current, { q: null }))), { source: "s", thread: "t", around: "m" });
  assert.deepEqual(Object.fromEntries(queryOf(readHref(current, { around: null }))), { source: "s", thread: "t", q: "words" });
  assert.equal(readHref(new URLSearchParams()), "/read");
});
