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

/** Compile a module in memory with isolated API/UI dependencies for selection checks.
 * Inputs: source path and dependency overrides. Output: module exports.
 * Effects: source reads only. Pick for desktop-safe tests without real workflow starts.
 */
function load(relative, overrides) {
  const source = readFileSync(new URL(relative, import.meta.url), "utf8");
  const compiled = ts.transpileModule(source, { compilerOptions: { jsx: ts.JsxEmit.ReactJSX, module: ts.ModuleKind.CommonJS } }).outputText;
  const module = { exports: {} };
  runInNewContext(compiled, { module, exports: module.exports, URLSearchParams, require: (name) => overrides[name] ?? require(name) });
  return module.exports;
}

test("Read reuses complete-conversation selection and forwards exact selected IDs to bulk actions", () => {
  let selected = new Set();
  let list;
  let actions;
  let selectAll;
  const hooks = load("../../hooks/use-conversation-actions.ts", {
    react: { ...React, useCallback: (callback) => callback, useState: () => [selected, (update) => { selected = typeof update === "function" ? update(selected) : update; }] },
    "@tanstack/react-query": {},
    "@/lib/conversation-actions-client": {},
  });
  const Button = ({ children, asChild, variant, ...props }) => {
    if (String(children).startsWith("Select all")) selectAll = props.onClick;
    return asChild ? children : React.createElement("button", props, children);
  };
  const selectable = load("../conversations/selectable-conversations.tsx", {
    "@/hooks/use-conversation-actions": hooks,
    "@/components/ui/button": { Button },
    "@/components/sbv/conversation-list": { ConversationList: (props) => {
      list = props;
      return React.createElement("ul", null, props.items.map((item) => React.createElement("li", { key: item.id }, React.createElement("a", { href: item.href }, item.contactName))));
    } },
    "@/components/conversations/conversation-actions": { ConversationSelectionBar: (props) => {
      actions = props;
      return props.threadIds.length ? React.createElement("div", null, "Extract / Send to Surreal") : null;
    } },
  });
  const threads = ["original-thread/+==", "second-thread_-"].map((id) => ({ id, title: id, participants: [], last_message: "Text", messages: 5, calls: 0, records: 5, last_at: null, party: "first_party" }));
  const params = new URLSearchParams("source=original-source&q=hearing&around=previous-record&mode=LIVE");
  const { ReadWorkspace } = load("./read-workspace.tsx", {
    "@tanstack/react-query": { useInfiniteQuery: ({ queryKey }) => ({ data: { pages: [{ items: queryKey[0] === "m-threads" ? threads : [], source: { file_name: "original.json" } }] } }) },
    "@tanstack/react-router": { useRouter: () => ({ navigate() {} }) },
    "@/components/conversations/selectable-conversations": selectable,
    "@/components/imported/imported-grid": { ImportedGrid: () => null },
    "@/components/mobile/mobile-format": { formatCount: String, statusLabel: String },
    "@/components/mobile/mobile-ui": { Empty: ({ children }) => React.createElement("p", null, children) },
    "@/components/read/read-conversation": { ReadConversation: () => null },
    "@/components/read/read-search": { ReadSearch: () => null },
    "@/components/read/read-location": load("./read-location.ts", {}),
    "@/components/ui/button": { Button },
    "@/lib/imported-client": { importedApi: {} },
    "@/lib/router-compat": { useBrowserSearchParams: () => params, AppLink: ({ href, children }) => React.createElement("a", { href }, children) },
  });
  const render = () => renderToStaticMarkup(React.createElement(ReadWorkspace));
  const initial = render();
  assert.deepEqual(Array.from(list.items, (item) => item.id), threads.map((thread) => thread.id));
  assert.equal(actions.threadIds.length, 0);
  for (const item of list.items) {
    const url = new URL(item.href, "https://test.invalid");
    assert.equal(url.pathname, "/read");
    assert.equal(url.searchParams.get("source"), "original-source");
    assert.equal(url.searchParams.get("q"), "hearing");
    assert.equal(url.searchParams.get("thread"), item.id);
    assert.equal(url.searchParams.get("around"), null);
    assert.equal(url.searchParams.get("mode"), "LIVE");
  }
  assert.match(initial, /<details[^>]*><summary[^>]*>Change source file/);
  assert.match(initial, /id="read-conversations"/);

  list.selection.onToggle(threads[0].id);
  assert.match(render(), /Extract \/ Send to Surreal/);
  assert.deepEqual(Array.from(actions.threadIds), [threads[0].id]);
  assert.equal(actions.placement, "desktop");

  selectAll();
  render();
  assert.deepEqual(Array.from(actions.threadIds), threads.map((thread) => thread.id));
  actions.onClear();
  render();
  assert.equal(actions.threadIds.length, 0);
  assert.equal(params.get("q"), "hearing", "selecting conversations must not rewrite parent search");
  assert.equal(params.get("source"), "original-source");
});
