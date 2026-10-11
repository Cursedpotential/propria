// Byline: Codex · GPT-6 · 2026-10-11.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import test from "node:test";
import { runInNewContext } from "node:vm";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import ts from "typescript";

const require = createRequire(import.meta.url);

/** Load one reader with stubbed transport and navigation for bounded render checks.
 * Inputs: module path and explicit dependencies. Output: exports. Effects: reads and
 * in-memory compilation only. Use for reader contracts without accessing live data.
 */
function load(path, dependencies) {
  const module = { exports: {} };
  const code = ts.transpileModule(readFileSync(new URL(path, import.meta.url), "utf8"), {
    compilerOptions: { jsx: ts.JsxEmit.ReactJSX, module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  }).outputText;
  runInNewContext(code, { module, exports: module.exports, URLSearchParams,
    require: (name) => dependencies[name] ?? require(name) });
  return module.exports;
}

const Link = ({ href, children }) => React.createElement("a", { href }, children);

test("Sources can open the existing donor reader by exact original source without processing", () => {
  const source = "b2://salem-data/path with spaces/sms+original.xml";
  let supplied;
  const { default: ReadPage } = load("../src/app/read/page.tsx", {
    "@/components/review/ai-candidate-review": { AICandidateReview: () => null },
    "@/components/read/read-workspace": { ReadWorkspace: () => { throw new Error("wrong reader"); } },
    "@/components/read/read-location": { isProcessingPreview: () => { throw new Error("must not start a preview workflow"); } },
    "@/components/sbv/proffer-preview-client": { ProfferPreviewClient: () => { throw new Error("must not process"); } },
    "@/components/sbv/decoded-source-viewer": { DecodedSourceViewer: ({ sourceRef }) => { supplied = sourceRef; return React.createElement("p", null, "Readable messages"); } },
    "@/lib/router-compat": { AppLink: Link, useBrowserSearchParams: () => new URLSearchParams({ decoded_source_ref: source, root: "b2-salem-data", mode: "LIVE" }) },
  });
  const html = renderToStaticMarkup(React.createElement(ReadPage));
  assert.equal(supplied, source);
  assert.match(html, /sms\+original.xml/);
  assert.match(html, /Readable messages/);
  assert.match(html, /root=b2-salem-data/);
});

test("decoded reader keeps one message page and exposes explicit next navigation", () => {
  const options = [];
  const calls = [];
  const messages = [{ ordinal: 0, sender: "self", occurred_at: null, body: "Original message", attachments: [], missing_attachments: 0 }];
  const manifest = { records: 10000, threads: [{ file: "thread.ndjson", thread: "8101234567", participants: ["self", "8101234567"], records: 10000 }], media_objects: 0, rejected: 0 };
  const { DecodedSourceViewer } = load("../src/components/sbv/decoded-source-viewer.tsx", {
    "@tanstack/react-query": { useQuery: (value) => { options.push(value); return { data: value.queryKey[0] === "decoded-manifest" ? manifest : { total_records: 10000, messages, next_offset: 200 } }; } },
    "@/components/ui/input": { Input: (props) => React.createElement("input", props) },
    "@/components/ui/button": { Button: ({ size, variant, children, ...props }) => React.createElement("button", props, children) },
    "@/lib/decoded-source-client": { getDecodedManifest() {}, getDecodedThreadPage: (...args) => calls.push(args), getDecodedMediaUrl: () => null, DecodedSourceError: Error },
  });
  const html = renderToStaticMarkup(React.createElement(DecodedSourceViewer, { sourceRef: "b2://bucket/source.xml" }));
  assert.match(html, /Original message/);
  assert.match(html, /Next messages/);
  assert.match(html, /Previous messages/);
  const page = options.find((value) => value.queryKey[0] === "decoded-thread-page");
  assert.equal(page.gcTime, 0);
  const signal = new AbortController().signal;
  page.queryFn({ signal });
  assert.deepEqual(calls[0], ["b2://bucket/source.xml", "thread.ndjson", 0, signal]);
});
