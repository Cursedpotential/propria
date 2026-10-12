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
    "@/components/packages/category-landing": { CategoryLanding: () => { throw new Error("wrong landing"); } },
    "@/components/packages/context-package-workspace": { ContextPackageWorkspace: () => { throw new Error("wrong package"); } },
    "@/components/sbv/human-context-preview": { HumanContextPreview: () => null },
    "@/lib/human-context": { isHumanPreviewHandle: () => false },
    "@/lib/fixed-case-context": { useFixedCase: () => ({ mode: "LIVE" }) },
    "@/components/read/read-workspace": { ReadWorkspace: () => { throw new Error("wrong reader"); } },
    "@/components/read/read-location": { isProcessingPreview: () => { throw new Error("must not start a preview workflow"); } },
    "@/components/sbv/proffer-preview-client": { ProfferPreviewClient: () => { throw new Error("must not process"); } },
    "@/components/sbv/decoded-source-viewer": { DecodedSourceViewer: ({ sourceRef }) => { supplied = sourceRef; return React.createElement("p", null, "Readable messages"); } },
    "@/components/sbv/source-message-preview": { SourceMessagePreview: () => null },
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

test("raw reader replaces pages, bounds cursor history and passes cancellation without media downloads", () => {
  const states = [];
  let stateIndex = 0;
  let buttons = [];
  let options;
  const calls = [];
  let expired = false;
  const selection = { source_ref: "b2://bucket/huge.xml", root_id: "root", key: "huge.xml", etag: '"e"', byte_length: 941300000 };
  const { SourceMessagePreview } = load("../src/components/sbv/source-message-preview.tsx", {
    react: { ...React, useState: (initial) => { const index = stateIndex++; if (states[index] === undefined) states[index] = initial; return [states[index], (value) => { states[index] = value; }]; } },
    "@tanstack/react-query": { useQuery: (value) => { options = value; return { isError: expired, error: new Error("Preview expired"), refetch() {}, data: { messages: [{ id: "m", direction: "incoming", participants: ["participant"], date_ms: null, body: "Bounded text", body_truncated: true, attachments: [{ name: "huge-photo.jpg", encoded_bytes: 900000000 }] }], next_cursor: `cursor-${states[2] + states[1]}`, truncation_reason: "byte_limit" } }; } },
    "@/components/ui/button": { Button: ({ children, ...props }) => { buttons.push({ children, ...props }); return React.createElement("button", props, children); } },
    "@/lib/source-message-preview-client": { getSourceMessagePreview: (...args) => calls.push(args) },
  });
  const render = () => { stateIndex = 0; buttons = []; return renderToStaticMarkup(React.createElement(SourceMessagePreview, { selection })); };
  let html = render();
  assert.match(html, /Bounded text/);
  assert.match(html, /Message shortened/);
  assert.match(html, /huge-photo.jpg/);
  assert.doesNotMatch(html, /<img|<video|<audio|base64|900000000/);
  const signal = new AbortController().signal;
  options.queryFn({ signal });
  assert.deepEqual(calls[0], [selection, null, signal]);
  assert.equal(options.gcTime, 0);
  for (let page = 0; page < 70; page++) {
    buttons.find((button) => button.children === "Next messages").onClick();
    html = render();
  }
  assert.equal(states[0].length, 64);
  assert.match(html, /Page 71/);
  assert.equal((html.match(/Bounded text/g) ?? []).length, 1);
  buttons.find((button) => button.children === "Previous messages").onClick();
  assert.match(render(), /Page 70/);
  expired = true;
  assert.match(render(), /Preview expired/);
  buttons.find((button) => button.children === "First messages").onClick();
  expired = false;
  assert.match(render(), /Page 1/);
  assert.equal(states[0].length, 1);
  assert.equal(options.queryKey[3], null);
});

test("processed message navigation replaces pages, preserves server filters and stops scroll/load-all fetching", () => {
  const states = [];
  let stateIndex = 0;
  let buttons = [];
  let toolbar;
  let grid;
  let options;
  const calls = [];
  const Empty = ({ children }) => React.createElement("div", null, children);
  const { MessageBrowser } = load("../src/components/sbv/message-browser.tsx", {
    react: { ...React, useState: (initial) => { const index = stateIndex++; if (states[index] === undefined) states[index] = initial; return [states[index], (value) => { states[index] = value; }]; } },
    "@tanstack/react-query": { useQuery: (value) => { options = value; return { data: { messages: [{ message_id: value.queryKey[3] ?? "first", body: "Current page" }], participants: [], next_cursor: `next-${states[5].first + states[5].index}`, total_matches: 50000, total_messages: 100000 } }; } },
    "react-resizable-panels": { Panel: Empty, PanelGroup: Empty, PanelResizeHandle: () => null },
    "@/components/sbv/message-browser-grid": { MessageBrowserGrid: (props) => { grid = props; return React.createElement("p", null, props.rows[0]?.message.message_id); } },
    "@/components/sbv/message-browser-toolbar": { MessageBrowserToolbar: (props) => { toolbar = props; return null; } },
    "@/components/sbv/message-detail-panel": { MessageDetailPanel: () => null },
    "@/components/sbv/message-source-panel": { MessageSourcePanel: () => null },
    "@/components/sbv/media-only-grid": { MediaOnlyGrid: () => null },
    "@/components/sbv/message-thread-view": { MessageThreadView: () => null },
    "@/components/ui/button": { Button: ({ size, variant, children, ...props }) => { buttons.push({ children, ...props }); return React.createElement("button", props, children); } },
    "@/hooks/use-preview-messages": { MESSAGE_PAGE_SIZE: 200, usePreviewMessageRows: (pages) => ({ rows: pages?.[0].messages.map((message) => ({ message, missingPayloadCount: 0 })) ?? [], participants: new Map() }), usePreviewMessageTotals: (pages) => ({ totalMatches: pages?.[0].total_matches, totalMessages: pages?.[0].total_messages }) },
    "@/lib/utils": { cn: (...values) => values.filter(Boolean).join(" ") },
    "@/lib/api-client": { getProfferPreviewMessages: (...args) => calls.push(args) },
    "@/lib/human-context": { isHumanPreviewHandle: () => false, humanMessagePage: (page) => page },
  });
  const render = () => { stateIndex = 0; buttons = []; return renderToStaticMarkup(React.createElement(MessageBrowser, { previewHandle: "exact-attempt", mode: "LIVE", packageProjection: null })); };
  render();
  assert.equal(toolbar.paged, true);
  assert.equal(options.gcTime, 0);
  const firstKey = JSON.stringify(options.queryKey);
  grid.onReachEnd(); toolbar.onLoadAll();
  assert.equal(JSON.stringify(options.queryKey), firstKey);
  assert.equal(calls.length, 0);
  for (let index = 0; index < 70; index++) { buttons.find((button) => button.children === "Next messages").onClick(); render(); }
  assert.equal(states[5].cursors.length, 64);
  assert.equal(grid.rows.length, 1);
  buttons.find((button) => button.children === "Previous messages").onClick(); render();
  toolbar.onQueryChange("important words");
  assert.match(render(), /Page 1/);
  assert.equal(options.queryKey[3], undefined);
  const signal = new AbortController().signal;
  options.queryFn({ signal });
  assert.deepEqual(calls[0].slice(0, 5), ["exact-attempt", "LIVE", undefined, 200, signal]);
  assert.deepEqual(JSON.parse(JSON.stringify(calls[0][5])), { q: "important words" });
  toolbar.onQueryChange("");
  assert.match(render(), /Page 1/);
  assert.equal(options.queryKey[3], undefined, "returning to an earlier filter must not restore its old cursor");
});

/** Evaluate the actual async read callback in isolation from unrelated workflow controls.
 * Input: callback name and explicit transport/state dependencies. Output: callable.
 * Effects: in-memory compilation only; no live source or workflow is touched.
 */
function readerCallback(name, dependencies) {
  const text = readFileSync(new URL("../src/components/sbv/proffer-preview-client.tsx", import.meta.url), "utf8");
  const file = ts.createSourceFile("reader.tsx", text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  let callback;
  const visit = (node) => {
    if (ts.isVariableDeclaration(node) && node.name.getText(file) === name && ts.isCallExpression(node.initializer)) callback = node.initializer.arguments[0].getText(file);
    ts.forEachChild(node, visit);
  };
  visit(file);
  assert.ok(callback, `${name} callback exists`);
  const code = ts.transpileModule(`module.exports = ${callback}`, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS } }).outputText;
  const module = { exports: {} };
  runInNewContext(code, { module, AbortController, Map, Set, ...dependencies });
  return module.exports;
}

test("fallback message reader discards previous bodies and rejects a cancelled stale page", async () => {
  const state = {};
  const requests = [];
  const loadMessages = readerCallback("loadMessages", {
    previewHandle: "attempt", mode: "LIVE", generationRef: { current: 1 }, activeHandleRef: { current: "attempt" },
    requestedCursorsRef: { current: new Set() }, messageControllersRef: { current: new Map() },
    setMessagesLoading() {}, setMessageError() {}, setNextCursor() {},
    setParticipants: (participants) => { state.participants = participants; }, setMessages: (messages) => { state.messages = messages; },
    getProfferPreviewMessages: (...args) => new Promise((resolve) => requests.push({ args, resolve })),
  });
  const first = loadMessages();
  const second = loadMessages("cursor-2");
  assert.equal(requests[0].args[4].aborted, true);
  const page = (id) => ({ preview_handle: "attempt", participants: [{ participant_id: id }], messages: [{ message_id: id, ordinal: 1, source_locator_ref: `original:${id}`, body: id }], next_cursor: null });
  requests[1].resolve(page("second")); await second;
  requests[0].resolve(page("first")); await first;
  assert.equal(state.messages.length, 1);
  assert.equal(state.messages[0].message_id, "second");
  assert.equal(state.messages[0].source_locator_ref, "original:second");
  assert.equal(state.participants[0].participant_id, "second");
});

test("content reader retains one exact record/chunk page rather than accumulating payloads", async () => {
  let current;
  let number = 0;
  const coordinates = { current: {} };
  const loadContent = readerCallback("loadContent", {
    previewHandle: "attempt", mode: "LIVE", generationRef: { current: 1 }, activeHandleRef: { current: "attempt" },
    contentControllerRef: { current: null }, contentCoordinatesRef: coordinates,
    setContentLoading() {}, setContentError() {}, setContent: (page) => { current = page; },
    getProfferPreviewContent: async () => { number++; return { records: [{ record_id: `r${number}`, source_locator_ref: "original:exact" }], chunks: [{ chunk_ref: `c${number}` }] }; },
  });
  await loadContent(); await loadContent("records-2", "chunks-3");
  assert.equal(current.records.length, 1); assert.equal(current.records[0].record_id, "r2");
  assert.equal(current.records[0].source_locator_ref, "original:exact");
  assert.equal(current.chunks.length, 1); assert.equal(current.chunks[0].chunk_ref, "c2");
  assert.deepEqual(JSON.parse(JSON.stringify(coordinates.current)), { record: "records-2", chunk: "chunks-3" });
});
