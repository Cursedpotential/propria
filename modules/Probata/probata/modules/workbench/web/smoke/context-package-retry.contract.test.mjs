// Byline: Codex · GPT-6 · 2026-10-12.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import test from "node:test";
import { runInNewContext } from "node:vm";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import ts from "typescript";

const require = createRequire(import.meta.url);
/** Load actual package retry code with controlled HTTP and query state.
 * Inputs: module path and explicit dependencies. Output: real exports. Effects: in-memory only.
 * Pick to check request contracts and rendered business failure without starting extraction.
 */
function load(path, dependencies = {}, globals = {}) {
  const module = { exports: {} };
  const source = readFileSync(new URL(path, import.meta.url), "utf8").replaceAll("import.meta.env.VITE_API_URL", '""');
  const js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX, target: ts.ScriptTarget.ES2022 } }).outputText;
  runInNewContext(js, { module, exports: module.exports, URL, require: (id) => dependencies[id] ?? require(id), ...globals });
  return module.exports;
}

test("native model retry preserves the exact source version and reads business failure despite Temporal completion", async () => {
  const calls = [];
  let failStart = false;
  let clickNumber = 0;
  const receipt = { workflow_id: "context-source:retry:actual", run_id: "actual-run", source_version_ref: "actual-version", status_path: "/api/context/packages/actual/attempts/context-source%3Aretry%3Aactual" };
  const failure = { status: "failed", reason: "configured extraction failed", execution_status: "COMPLETED", source_version_ref: "actual-version" };
  const client = load("../src/lib/context-package-client.ts", { "@/lib/api-client": { ApiError: Error } }, { fetch: async (path, init) => { calls.push({ path, init }); if (failStart && init.method === "POST") throw new Error("Uncertain network failure"); return { ok: true, json: async () => init.method === "POST" ? receipt : failure }; } });
  const capability = { operation: "retry_extraction", available: true, method: "POST", path: "/api/context/packages/actual/retry", body: { source_version_ref: "actual-version", model_id: "configured" }, models: [{ id: "configured", label: "Configured native extraction model" }] };
  const buttons = [];
  const mutations = [];
  const invalidated = [];
  let queryOptions;
  let stateIndex = 0;
  const { PackageRetry } = load("../src/components/packages/package-retry.tsx", {
    react: { ...React, useState: () => [["", "Owner requested retry"][stateIndex++], () => {}], useRef: (value) => ({ current: value }), useEffect: (effect) => effect() },
    "@/components/ui/button": { Button: ({ children, ...props }) => { buttons.push({ children, ...props }); return React.createElement("button", props, children); } },
    "@/components/ui/input": { Input: (props) => React.createElement("input", props) },
    "@/lib/context-package-client": client,
    "@tanstack/react-query": {
      useQueryClient: () => ({ invalidateQueries: async (value) => invalidated.push(value) }),
      useMutation: (options) => ({ data: receipt, isPending: false, mutateAsync: (input) => { const promise = options.mutationFn(input).then(async (value) => { await options.onSuccess(value); return value; }); mutations.push(promise); return promise; } }),
      useQuery: (options) => { queryOptions = options; return { data: failure, isFetching: false, refetch: async () => {} }; },
    },
  }, { crypto: { randomUUID: () => `click-key-${++clickNumber}` } });
  const html = renderToStaticMarkup(React.createElement(PackageRetry, { handle: "actual", capabilities: [capability], disabled: false, onDirtyChange: () => {}, onPendingChange: () => {} }));
  assert.match(html, /Extraction business status: failed/);
  assert.match(html, /configured extraction failed/);
  assert.doesNotMatch(html, /Extraction business status: COMPLETED/);
  const button = buttons.find((entry) => entry.children === "Retry extraction");
  assert.equal(button.disabled, false);
  button.onClick();
  await Promise.all(mutations);
  const body = JSON.parse(calls[0].init.body);
  assert.deepEqual(body, { source_version_ref: "actual-version", model_id: "configured" });
  assert.doesNotMatch(html, /Extraction retry reason/, "the strict backend DTO accepts no owner reason field");
  assert.equal(calls[0].init.headers["Idempotency-Key"], "click-key-1");
  assert.equal(invalidated[0].queryKey.join("/"), "context-package/actual");
  assert.equal((await queryOptions.queryFn({ signal: undefined })).status, "failed");
  assert.equal(calls[1].path, receipt.status_path);
  button.onClick();
  await Promise.all(mutations);
  assert.equal(calls[2].init.headers["Idempotency-Key"], "click-key-2", "a new explicit attempt gets a new key after successful submission");
  failStart = true;
  button.onClick();
  await Promise.allSettled(mutations);
  failStart = false;
  button.onClick();
  await Promise.allSettled(mutations);
  assert.equal(calls[3].init.headers["Idempotency-Key"], calls[4].init.headers["Idempotency-Key"], "an unchanged failed submission reuses the uncertain request key");
  for (const status of ["failed", "partial_enrichment", "complete"]) assert.equal(queryOptions.refetchInterval({ state: { data: { status } } }), false);
  assert.equal(queryOptions.refetchInterval({ state: { data: { status: "enriching" } } }), 3000);
  assert.equal(queryOptions.refetchInterval({ state: { error: new Error("unavailable") } }), false);
  stateIndex = 0;
  const beforeDirtyRender = invalidated.length;
  renderToStaticMarkup(React.createElement(PackageRetry, { handle: "actual", capabilities: [capability], disabled: true, onDirtyChange: () => {}, onPendingChange: () => {} }));
  assert.equal(invalidated.length, beforeDirtyRender, "terminal retry refresh must wait while the item editor has an unsaved draft");
  stateIndex = 0;
  const buttonsBeforeUnavailable = buttons.length;
  const unavailable = renderToStaticMarkup(React.createElement(PackageRetry, { handle: "actual", capabilities: [], disabled: false, onDirtyChange: () => {}, onPendingChange: () => {} }));
  assert.match(unavailable, /Extraction business status: failed/);
  assert.match(unavailable, /configured extraction failed/);
  assert.equal(buttons.slice(buttonsBeforeUnavailable).some((entry) => entry.children === "Retry extraction"), false, "retain progress without offering a withdrawn capability");
  for (const path of ["https://outside.invalid/api/context/packages/p/attempts/a", "//outside.invalid/api/context/packages/p/attempts/a", "/api/context/packages/p/../../sources"]) assert.throws(() => client.getPackageRetryStatus(path), /Invalid/);
});
