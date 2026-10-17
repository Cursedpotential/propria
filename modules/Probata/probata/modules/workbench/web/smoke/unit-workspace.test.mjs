// Synthetic SSR checks for the real unit workspace. No browser or live data.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { runInNewContext } from "node:vm";
import test from "node:test";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import ts from "typescript";
const require = createRequire(import.meta.url);

function load(relative, overrides) {
  const input = readFileSync(new URL(relative, import.meta.url), "utf8");
  const code = ts.transpileModule(input, { compilerOptions: {
    target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX,
  } }).outputText;
  const module = { exports: {} };
  runInNewContext(code, { module, exports: module.exports, URLSearchParams,
    require: (id) => overrides[id] ?? require(id) });
  return module.exports;
}
const source = (part) => ({ id: `source-${part}`, format: "Facebook", file_name: `message_${part}.json`, owner: null });
const unit = {
  source_id: "source-7", root: "fixture/messages/inbox/thread_1", membership_basis: "registered_facebook_folder",
  members: Array.from({ length: 7 }, (_, i) => ({ source: source(i + 1), versions: [], messages: 3 })),
  extraction_targets: [], generation_signature: "signature", message_count: 21,
  generation_policy: "Latest per source", coverage: "Registered sources only", read_at: "2026-10-17T00:00:00Z",
};
const message = {
  id: "record-1", at: "2020-01-01T00:00:00Z", body: "Synthetic <script>not markup</script>",
  sender: { label: "Fixture participant" }, outgoing: false, attachments: 2,
  attribution: { direction: "unknown", record_ordinal: 4 }, source: { ...source(1), thread_id: "thread-1" },
};

function render(paramsText = "", queryResult = { data: unit }) {
  const reads = []; const bubbles = []; const queries = [];
  const locations = load("../src/components/read/read-location.ts", {});
  const components = load("../src/components/read/unit-workspace.tsx", {
    "@tanstack/react-query": {
      useQuery: () => queryResult,
      useInfiniteQuery: (options) => { queries.push(options); return { data: { pages: [{ items: [message], generation_signature: "signature" }] }, hasNextPage: true }; },
    },
    "@tanstack/react-router": { useRouter: () => ({ navigate() {} }) },
    "@/components/read/read-location": locations,
    "@/components/conversations/selectable-conversations": { SelectableConversations: () => React.createElement("div", null, "Existing bulk tools") },
    "@/components/conversations/conversation-actions": { ConversationActionSheet: () => { throw Error("No extraction should run during ordinary reading"); } },
    "@/components/imported/record-rows": { toMessageRow: (item) => ({ message: item }) },
    "@/components/mobile/mobile-ui": {
      Empty: ({ children }) => React.createElement("p", null, children),
      ErrorBox: ({ error }) => React.createElement("p", { role: "alert" }, error.message),
      Loading: () => React.createElement("p", null, "Loading"),
    },
    "@/components/mobile/mobile-format": { formatDate: (value) => value ?? "Unknown" },
    "@/components/read/read-context": { ReadContext: () => { throw Error("Unopened extraction results must not load"); } },
    "@/components/read/unit-inspector": { UnitInspector: ({ recordId }) => React.createElement("aside", { "data-record": recordId }, "Selection inspector") },
    "@/components/sbv/message-bubble": { MessageBubble: ({ row }) => { bubbles.push(row); return React.createElement("p", { "data-sbv": true }, row.message.body); } },
    "@/components/ui/button": { Button: ({ variant, size, children, ...props }) => React.createElement("button", props, children) },
    "@/components/ui/input": { Input: (props) => React.createElement("input", props) },
    "@/lib/imported-client": { importedApi: { unitMessages: (...args) => { reads.push(args); return Promise.resolve({ items: [], generation_signature: "signature" }); } } },
    "@/lib/fixed-case-context": { useOperatingMode: () => "LIVE" },
    "@/lib/router-compat": { AppLink: ({ children, ...props }) => React.createElement("a", props, children) },
  });
  const html = renderToStaticMarkup(React.createElement(components.UnitWorkspace, {
    sourceId: "source-7", threadId: null, params: new URLSearchParams(paramsText),
  }));
  return { html, bubbles, queries, reads };
}

test("selected unit uses all seven parts, SBV content and a selection inspector, not global search", () => {
  const { html, bubbles } = render("around=record-1");
  for (let i = 1; i <= 7; i++) assert.ok(html.includes(`message_${i}.json`));
  assert.match(html, /Search entire unit/);
  assert.match(html, /data-sbv="true"/);
  assert.match(html, /data-record="record-1"/);
  assert.match(html, /&lt;script&gt;not markup&lt;\/script&gt;/);
  assert.match(html, /2 attachment references/);
  assert.doesNotMatch(html, /Super Index|Case Bible \/ Intake|Search content/);
  assert.equal(bubbles[0].direction, "unknown");
});

test("search forwards exact unit/member scope and cursor to the server", async () => {
  const result = render("unit_q=50%25_done&unit_member=source-3");
  const signal = new AbortController().signal;
  await result.queries[0].queryFn({ pageParam: "next-page", signal });
  assert.equal(result.reads[0][0], "source-7");
  assert.equal(result.reads[0][1].memberId, "source-3");
  assert.equal(result.reads[0][1].q, "50%_done");
  assert.equal(result.reads[0][1].cursor, "next-page");
  assert.equal(result.reads[0][2], signal);
  assert.match(result.html, /Search selected member/);
  assert.equal(result.queries[0].queryKey.at(-1), "signature");
});

test("unavailable unit reports the read failure, not a zero-count success", () => {
  const { html } = render("", { isError: true, error: new Error("Unit read unavailable") });
  assert.match(html, /role="alert"/);
  assert.match(html, /Unit read unavailable/);
  assert.doesNotMatch(html, /0 messages|normalization verified/i);
});

test("selected-source entry exits discovery before global search is mounted", () => {
  const source = readFileSync(new URL("../src/components/read/read-workspace.tsx", import.meta.url), "utf8");
  assert.match(source, /if \(sourceId\) return <UnitWorkspace/);
  assert.match(source, /if \(threadId\) return <ThreadUnitEntry/);
  assert.ok(source.indexOf("if (sourceId) return <UnitWorkspace") < source.indexOf("<ReadSearch"));
});
