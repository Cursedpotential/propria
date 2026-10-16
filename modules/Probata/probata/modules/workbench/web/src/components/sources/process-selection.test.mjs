// Byline: Claude Code · Opus 5.5 · 2026-10-16
// The starter's 409 already_imported refusal is shown as "already imported -- view it?"
// (owner spec 2026-10-15), not as a processing failure.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import ts from "typescript";

function load() {
  const source = readFileSync(new URL("./process-selection.ts", import.meta.url), "utf8");
  const compiled = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS } }).outputText;
  class ApiError extends Error { constructor(message, status, body) { super(message); this.status = status; this.body = body; } }
  const module = { exports: {} };
  new Function("module", "exports", "require", compiled)(module, module.exports, (name) => {
    if (name === "@/lib/api-client") return { ApiError };
    throw new Error(`unexpected import ${name}`);
  });
  return { ...module.exports, ApiError };
}

test("a 409 already_imported refusal becomes a view-it notice, not a failure", async () => {
  const { processSelection, ApiError } = load();
  const refusal = new ApiError("This content has already been imported.", 409, { detail: "This content has already been imported.", already_imported: { status: "imported", source_key: "b2://bucket/message_3.json", imported_at: "2026-10-03T02:32:00Z" } });
  const result = await processSelection([{ request: { request_id: "r" } }], async () => { throw refusal; }, () => {}, "message_3.json");
  assert.equal(result.error, null);
  assert.deepEqual(result.alreadyImported, { status: "imported", sourceKey: "b2://bucket/message_3.json", importedAt: "2026-10-03T02:32:00Z", name: "message_3.json" });
});

test("other failures stay failures", async () => {
  const { processSelection, ApiError } = load();
  const result = await processSelection([{ request: {} }], async () => { throw new ApiError("upstream down", 503, {}); }, () => {}, "x");
  assert.equal(result.error, "upstream down");
  assert.equal(result.alreadyImported, undefined);
  const conflict = await processSelection([{ request: {} }], async () => { throw new ApiError("other conflict", 409, { detail: "other" }); }, () => {}, "x");
  assert.equal(conflict.error, "other conflict");
});

test("the view link is the Read source id the Imported API issues (unpadded URL-safe base64)", () => {
  const { readSourceHref } = load();
  const key = "b2://salem-data/consignatio/casevault/SourceCorpus/messaging/facebook-messenger/meta-2025-08-18/katrinakinzel_1614217831996100/message_3.json";
  assert.equal(readSourceHref(key), "/read?source=YjI6Ly9zYWxlbS1kYXRhL2NvbnNpZ25hdGlvL2Nhc2V2YXVsdC9Tb3VyY2VDb3JwdXMvbWVzc2FnaW5nL2ZhY2Vib29rLW1lc3Nlbmdlci9tZXRhLTIwMjUtMDgtMTgva2F0cmluYWtpbnplbF8xNjE0MjE3ODMxOTk2MTAwL21lc3NhZ2VfMy5qc29u");
  const id = new URLSearchParams(readSourceHref("upload://" + "ab".repeat(32)).split("?")[1]).get("source");
  assert.ok(id && !id.includes("=") && !id.includes("+") && !id.includes("/"), id);
});
