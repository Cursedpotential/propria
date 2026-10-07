// Byline: Codex · GPT-6 · 2026-10-06
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import test from "node:test";
import { runInNewContext } from "node:vm";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import ts from "typescript";

const require = createRequire(import.meta.url);

/** Load a component with isolated data/navigation boundaries for browser-free render checks.
 * Inputs: source path and dependency overrides. Output: module exports.
 * Effects: reads source and compiles in memory; no API calls or filesystem writes.
 * Pick for rendering tests on this desktop; live interactions remain a VPS gate.
 */
function load(relative, overrides) {
  const source = readFileSync(new URL(relative, import.meta.url), "utf8");
  const compiled = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, jsx: ts.JsxEmit.ReactJSX, module: ts.ModuleKind.CommonJS } }).outputText;
  const module = { exports: {} };
  runInNewContext(compiled, { module, exports: module.exports, URLSearchParams, require: (name) => overrides[name] ?? require(name) });
  return module.exports;
}

const locations = load("./read-location.ts", {});
const ui = {
  Loading: () => React.createElement("p", null, "Loading content"),
  Empty: ({ children }) => React.createElement("p", null, children),
  ErrorBox: ({ error }) => React.createElement("p", { role: "alert" }, error.message),
  LoadMore: ({ label }) => React.createElement("button", null, label),
};
const common = {
  "@/components/read/read-location": locations,
  "@/components/mobile/mobile-format": { formatDateTime: (date) => date ?? "Undated", formatDate: (date) => date ?? "Undated" },
  "@/components/mobile/mobile-ui": ui,
  "@/components/ui/button": { Button: ({ children, asChild, variant, ...props }) => asChild ? children : React.createElement("button", props, children) },
  "@/lib/router-compat": { AppLink: ({ href, children, ...props }) => React.createElement("a", { href, ...props }, children) },
};

/** Render ReadSearch with supplied query state and a recorded existing API boundary.
 * Input: query result. Output: HTML plus captured query options and API calls.
 * Effects: in-memory rendering only. Pick for search error/empty/navigation regression checks.
 */
function search(result) {
  let options;
  const calls = [];
  const { ReadSearch } = load("./read-search.tsx", {
    ...common,
    "@tanstack/react-router": { useRouter: () => ({ navigate() {} }) },
    "@tanstack/react-query": { useInfiniteQuery: (value) => { options = value; return result; } },
    "@/lib/imported-client": { importedApi: { search: (...args) => { calls.push(args); return Promise.resolve({}); } } },
  });
  const html = renderToStaticMarkup(React.createElement(ReadSearch, { params: new URLSearchParams("q=hearing&source=other&mode=LIVE") }));
  return { html, options, calls };
}

/** Render a cited conversation using isolated messages and exact-thread participant pages.
 * Inputs: message header, thread result and route context. Output: HTML, rows and API captures.
 * Effects: in-memory rendering only. Pick for title/content regressions without corpus access.
 */
function conversation(head, threadResult = {}, params = "source=wrong&thread=original-thread&around=original-record&q=words") {
  let options;
  let threadOptions;
  const calls = [];
  const threadCalls = [];
  const bubbleRows = [];
  const { ReadConversation } = load("./read-conversation.tsx", {
    ...common,
    "@tanstack/react-query": { useInfiniteQuery: (value) => {
      if (value.queryKey[0] === "m-threads") { threadOptions = value; return threadResult; }
      options = value;
      return { data: { pages: [head] }, hasNextPage: true };
    } },
    "@/lib/imported-client": { importedApi: {
      messages: (...args) => { calls.push(args); return Promise.resolve(head); },
      threads: (...args) => { threadCalls.push(args); return Promise.resolve({}); },
    } },
    "@/components/conversations/conversation-actions": { ConversationToolbar: () => React.createElement("div", null, "Existing tools") },
    "@/components/imported/record-rows": { toMessageRow: (message) => ({ message }) },
    "@/components/sbv/message-bubble": { MessageBubble: ({ row, highlighted }) => {
      bubbleRows.push(row);
      return React.createElement("p", { "data-highlighted": highlighted }, row.message.body);
    } },
    "@/components/read/read-context": { ReadContext: () => React.createElement("aside", null, "Existing context") },
  });
  const html = renderToStaticMarkup(React.createElement(ReadConversation, { threadId: "original-thread", around: "original-record", params: new URLSearchParams(params) }));
  return { html, options, calls, threadOptions, threadCalls, bubbleRows };
}

test("content search uses existing API pagination/abort and links hits by original IDs", async () => {
  const { html, options, calls } = search({ isSuccess: true, hasNextPage: true, data: { pages: [{ note: "Existing search coverage", items: [
    { id: "record-id", thread_id: "thread/+==", body: "Message <script>unsafe</script>", sender: "Sender", source: "export.json" },
    { id: "call-id", thread_id: null, body: "Call entry", kind: "call_log" },
  ] }] } });
  const signal = new AbortController().signal;
  await options.queryFn({ pageParam: 20, signal });
  assert.deepEqual(calls[0], ["hearing", 20, signal]);
  assert.equal(options.getNextPageParam({ next_offset: 40 }), 40);
  assert.equal(options.getNextPageParam({ next_offset: null }), undefined);
  assert.match(html, /thread=thread%2F%2B%3D%3D&amp;around=record-id/);
  assert.doesNotMatch(html, /source=other[^<]*>Read in conversation/);
  assert.match(html, /&lt;script&gt;unsafe&lt;\/script&gt;/);
  assert.match(html, /Call-log result; no conversation link/);
  assert.match(html, /Load more results/);
});

test("search reports pending, unavailable and successful empty states distinctly", () => {
  assert.match(search({ isPending: true }).html, /Loading content/);
  const failed = search({ isError: true, error: new Error("Search unavailable") }).html;
  assert.match(failed, /role="alert"[^>]*>Search unavailable/);
  assert.doesNotMatch(failed, /No imported content matches/);
  assert.match(search({ isSuccess: true, data: { pages: [{ items: [], note: "" }] } }).html, /No imported content matches/);
});

test("focused reading uses around only on the first page and canonical source for citations", async () => {
  const head = { conversation: "Conversation", source: { id: "canonical/+", file_name: "original.json", format: "Other" }, items: [
    { id: "original-record", at: null, body: "Original content", sender: { label: "Person" }, attachments: 2, outgoing: false },
  ], older_cursor: "older" };
  const { html, options, calls } = conversation(head);
  const signal = new AbortController().signal;
  await options.queryFn({ pageParam: null, signal });
  await options.queryFn({ pageParam: "older", signal });
  assert.equal(calls[0][0], "original-thread");
  assert.equal(calls[0][1].around, "original-record");
  assert.equal(calls[1][1].around, null);
  assert.equal(calls[1][1].cursor, "older");
  assert.equal(calls[1][2], signal);
  assert.match(html, /data-highlighted="true"/);
  assert.match(html, /source=canonical%2F%2B/);
  assert.match(html, /Original record ID/);
  assert.match(html, /content is not supplied by this messages API/);
  assert.match(html, /Jump to latest messages/);
  assert.match(html, /source=canonical%2F%2B&amp;thread=original-thread&amp;around=original-record&amp;q=words#read-conversations/);
  assert.doesNotMatch(html, /href="\/conversations/);
});

test("Read heading uses only exact-thread API participants and reuses the canonical source query", async () => {
  const head = { conversation: "810001_810002", source: { id: "canonical/+", file_name: "original.json", format: "SMS", owner_name: "Owner" }, items: [
    { id: "original-record", body: "Outgoing only", sender: { label: "Owner", mine: true }, attachments: 0, outgoing: true },
  ] };
  const participants = [{ label: "Owner", mine: true }, { label: "Resolved person", mine: false }];
  const { html, threadOptions, threadCalls } = conversation(head, { data: { pages: [{ items: [
    { id: "different-thread", participants: [{ label: "Wrong person", mine: false }] },
    { id: "original-thread", title: "810001_810002", participants },
  ] }] } });
  assert.match(html, /<h2[^>]*>Resolved person<\/h2>/);
  assert.doesNotMatch(html, /<h2[^>]*>[^<]*(810001|Owner|Wrong person)/);
  assert.match(html, /Original conversation key<\/dt><dd>810001_810002/);
  assert.equal(threadOptions.queryKey[0], "m-threads");
  assert.equal(threadOptions.queryKey[1], "canonical/+");
  assert.equal(threadOptions.enabled, true);
  const signal = new AbortController().signal;
  await threadOptions.queryFn({ pageParam: 25, signal });
  assert.deepEqual(threadCalls[0], ["canonical/+", 25, signal]);
  assert.equal(threadOptions.getNextPageParam({ next_offset: 50 }), 50);
});

test("Read title uses returned senders while thread participants are unavailable and never promotes the internal key", () => {
  const head = { conversation: "810001_810002", source: { id: "canonical", file_name: "original.json", format: "SMS", owner_name: "Owner" }, items: [
    { id: "original-record", body: "Incoming", sender: { label: "Known sender", mine: false }, attachments: 0, outgoing: false },
    { id: "second", body: "Outgoing", sender: { label: "Owner", mine: true }, attachments: 0, outgoing: true },
  ] };
  assert.match(conversation(head, { isError: true }).html, /<h2[^>]*>Known sender<\/h2>/);
  const outgoingOnly = { ...head, items: [head.items[1]] };
  assert.match(conversation(outgoingOnly).html, /<h2[^>]*>Conversation · original.json<\/h2>/);
  const unknown = { ...head, items: [{ ...head.items[0], sender: { label: "Unknown", mine: false } }] };
  assert.match(conversation(unknown).html, /<h2[^>]*>Conversation · original.json<\/h2>/);
  assert.doesNotMatch(conversation(unknown).html, /<h2[^>]*>[^<]*810001/);
});

test("participant headings retain thread-list group and Facebook policies", () => {
  const head = { conversation: "internal", source: { id: "source", file_name: "original.json", format: "SMS" }, items: [] };
  const participants = ["One", "Two", "One", "Three", "Four", " "].map((label) => ({ label, mine: false }));
  const threads = { data: { pages: [{ items: [{ id: "original-thread", participants }] }] } };
  assert.match(conversation(head, threads).html, /<h2[^>]*>One, Two, Three\.\.\.<\/h2>/);
  assert.match(conversation({ ...head, source: { ...head.source, format: "Facebook" } }, threads).html, /<h2[^>]*>One<\/h2>/);
});

test("empty original records keep their bodies and citation IDs with a compact explanation only when attachment-free", () => {
  const originals = [
    { id: "empty", body: "", attachments: 0 },
    { id: "whitespace", body: " \n\t", attachments: 0 },
    { id: "attachment-only", body: "", attachments: 2 },
    { id: "text", body: "Actual source text", attachments: 0 },
  ].map((message) => ({ ...message, at: "2026-10-06T00:00:00Z", sender: { label: "Known sender", mine: false }, outgoing: false }));
  const before = JSON.stringify(originals);
  const { html, bubbleRows } = conversation({ conversation: "internal", source: { id: "source", file_name: "original.json", format: "SMS" }, items: originals });
  assert.equal((html.match(/No text in this record/g) ?? []).length, 2);
  for (const message of originals) {
    assert.match(html, new RegExp(`id="read-message-${message.id}"`));
    assert.match(html, new RegExp(`around=${message.id}`));
  }
  assert.deepEqual(bubbleRows.map((row) => row.message.body), originals.map((message) => message.body));
  assert.equal(JSON.stringify(originals), before);
  assert.match(html, /Actual source text/);
  assert.match(html, /2 \(content is not supplied by this messages API\)/);
});

test("Read route switches preview aliases without remounting preview state on query changes", () => {
  for (const query of ["resource=one", "preview_handle=two", "attempt=three", "view=review"]) {
    const { default: ReadPage } = load("../../app/read/page.tsx", {
      ...common,
      "@/lib/router-compat": { ...common["@/lib/router-compat"], useBrowserSearchParams: () => new URLSearchParams(`${query}&thread=kept`) },
      "@/components/read/read-workspace": { ReadWorkspace: () => React.createElement("div", null, "Imported reading") },
      "@/components/sbv/proffer-preview-client": { ProfferPreviewClient: () => React.createElement("div", null, "Existing previews") },
    });
    const tree = ReadPage();
    assert.equal(tree.props.children[1].key, null, "URL changes must preserve pending parser/repair gate answers");
    const html = renderToStaticMarkup(tree);
    assert.match(html, /Existing previews/);
    assert.match(html, /href="\/read\?thread=kept"/);
  }
});

test("extracted context retains run versions and links supplied entity/event record citations", () => {
  const { ReadContext } = load("./read-context.tsx", {
    ...common,
    "@/components/conversations/extractions-view": { ExtractionsView: () => React.createElement("div", null, "Existing extraction findings") },
    "@/hooks/use-conversation-actions": { useThreadExtractions: () => ({ data: { extractors: [{
      id: "extractor-id", label: "Extractor label",
      runs: [{ id: "run-id", extractor: "Extractor", version: "original-version", status: "completed" }],
      entities: [{ id: "entity-id", run_id: "run-id", name: "Person", review_state: "proposed", confidence: 0.8, mentions: [
        { record_id: "entity-record", snippet: "Original mention" }, { record_id: null, snippet: null },
      ] }],
      events: [{ id: "event-id", run_id: "run-id", title: "Event", review_state: "proposed", confidence: 0.7, record_ids: ["event-record"], description: "Extracted description" }],
    }] } }) },
  });
  const html = renderToStaticMarkup(React.createElement(ReadContext, { threadId: "thread-id", params: new URLSearchParams("source=source-id&thread=thread-id&q=words") }));
  assert.match(html, /Existing extraction findings/);
  assert.match(html, /version original-version/);
  assert.match(html, /source=source-id&amp;thread=thread-id&amp;q=words&amp;around=entity-record/);
  assert.match(html, /around=event-record/);
  assert.match(html, /This mention has no source record ID/);
  assert.match(html, /Entity entity-id · run run-id/);
  assert.match(html, /Event event-id · run run-id/);
});
