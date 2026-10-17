// Read-only synthetic smoke checks for the new library and advanced search slice.
// Uses the Workbench's TypeScript dev dependency; no case data or browser required.
import { readFileSync } from "node:fs";
import assert from "node:assert/strict";
import test from "node:test";
import ts from "typescript";

const modelSource = readFileSync(new URL("../src/components/library/library-facets.ts", import.meta.url), "utf8");
const modelJavaScript = ts.transpileModule(modelSource, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
}).outputText;
const model = await import("data:text/javascript;base64," + Buffer.from(modelJavaScript).toString("base64"));

function source(id, fileName, key, overrides = {}) {
  return {
    id, file_name: fileName, casevault_key: key,
    format: "SMS", owner: null, owner_name: null, device: null,
    status: "committed", ...overrides,
  };
}

test("category parser fails closed to all, not an invented source type", () => {
  assert.equal(model.libraryFormat("SMS"), "SMS");
  assert.equal(model.libraryFormat("Facebook"), "Facebook");
  assert.equal(model.libraryFormat("AI Chat Export"), "all");
  assert.equal(model.libraryFormat(null), "all");
});

test("source-owner facet uses recorded owner, not sender names or filename guesses", () => {
  const rows = [
    source("1", "person-name-sms.xml", "backup/a.xml"),
    source("2", "plain.xml", "backup/b.xml", { owner_name: "Source account A" }),
  ];
  assert.deepEqual(model.filterImportedLibrary(rows, {
    text: "", owner: "recorded", status: "all",
  }).map((row) => row.id), ["2"]);
  assert.deepEqual(model.filterImportedLibrary(rows, {
    text: "person-name", owner: "recorded", status: "all",
  }), []);
});

test("search and state facets operate on the loaded window with intact IDs", () => {
  const rows = [
    source("1", "one.xml", "phone/one.xml", { status: "failed" }),
    source("2", "two.xml", "phone/two.xml", { status: "committed" }),
    source("3", "three.xml", "other/three.xml", { status: "failed" }),
  ];
  assert.deepEqual(model.filterImportedLibrary(rows, {
    text: "phone/", owner: "all", status: "failed",
  }).map((row) => row.id), ["1"]);
});

test("same-name imports are candidates only and never silently deduplicated", () => {
  const rows = [
    source("a", "thread.xml", "phone1/thread.xml"),
    source("b", "thread.xml", "phone2/thread.xml"),
    source("c", "thread.xml", "phone1/thread.xml", { status: "failed" }),
    source("d", "different.xml", "phone1/different.xml"),
  ];
  assert.deepEqual(model.possibleSameNameImports(rows[0], rows).map((row) => row.id), ["b", "c"]);
  assert.equal(rows.length, 4);
});

test("UI reuses the existing Read reader and search rather than introducing a detached application", () => {
  const root = new URL("../src/components/", import.meta.url);
  const read = readFileSync(new URL("read/read-workspace.tsx", root), "utf8");
  const library = readFileSync(new URL("library/imported-library.tsx", root), "utf8");
  const search = readFileSync(new URL("read/context-search.tsx", root), "utf8");
  assert.match(read, /<ImportedLibrary /);
  assert.match(read, /<UnitWorkspace /);
  const unit = readFileSync(new URL("read/unit-workspace.tsx", root), "utf8");
  assert.match(unit, /<MessageBubble /);
  assert.match(unit, /unitMessages/);
  assert.doesNotMatch(read, /<ReadAnalysis /);
  assert.match(library, /importedApi\.sources/);
  assert.match(library, /Load more recorded imports/);
  assert.match(library, /Name match only/);
  assert.match(search, /"casebible"/);
  assert.match(search, /Advanced search · exact source filters/);
});
