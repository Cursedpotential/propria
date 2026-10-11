// Byline: Codex · 2026-10-06. SDK record references used by live graph navigation.
import assert from "node:assert/strict";
import test from "node:test";
import { graphReference } from "../src/lib/graph-reference.ts";
test("SDK escaped numeric string keys retain the same graph identity", () => {
  const key = "7".repeat(64);
  assert.deepEqual(graphReference(`occurrence:⟨${key}⟩`), graphReference(`occurrence:${key}`));
  assert.equal(graphReference(`occurrence:⟨${key}⟩`).key, key);
});
test("malformed records cannot become neighborhood queries", () => {
  for (const value of [null, {}, "occurrence:a/b", "occurrence:⟨a⟩", "x:y;SELECT * FROM memory", "https://host/path"])
    assert.equal(graphReference(value), null);
});
