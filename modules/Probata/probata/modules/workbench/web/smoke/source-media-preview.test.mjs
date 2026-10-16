import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { runInNewContext } from "node:vm";
import test from "node:test";
import { renderToStaticMarkup } from "react-dom/server";
import ts from "typescript";

const require = createRequire(import.meta.url);
const source = readFileSync(new URL("../src/components/sources/source-media-preview.tsx", import.meta.url), "utf8");

/** Run media event callbacks against the real reader component with deterministic hook state.
 * Input: media kind. Output: component tree and rerender. Effects: memory only;
 * choose for error/retry identity checks; actual decoding still requires a VPS browser.
 */
function reader(kind) {
  let cursor = 0;
  const state = [];
  const module = { exports: {} };
  runInNewContext(ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX } }).outputText, {
    module, exports: module.exports, require: (name) => name === "react" ? {
      useState(initial) { const index = cursor++; if (!(index in state)) state[index] = initial; return [state[index], (value) => { state[index] = typeof value === "function" ? value(state[index]) : value; }]; },
    } : name === "@/components/ui/button" ? { Button: "button" } : require(name),
  });
  const inspection = { preview_kind: kind, name: `fixture.${kind}`, content_type: `${kind}/fixture`, provider_version_id: "version-a", preview_url: "/api/proffer/source-content?root_id=fixture-root&key=fixture&version_id=version-a&etag=opaque&expected_byte_length=123" };
  const render = () => { cursor = 0; return module.exports.SourceMediaPreview({ inspection }); };
  return { render, inspection };
}

for (const kind of ["image", "audio", "video"]) test(`${kind} exposes loading, decoding error and deliberate retry of the identical original version`, () => {
  const fixture = reader(kind);
  let tree = fixture.render();
  assert.match(renderToStaticMarkup(tree), /Loading original media/);
  let media = tree.props.children.find((child) => ["img", "audio", "video"].includes(child?.type));
  const initialKey = media.key;
  assert.equal(media.props.src, fixture.inspection.preview_url);
  media.props.onError();
  tree = fixture.render();
  assert.match(renderToStaticMarkup(tree), /could not load or decode/);
  assert.match(renderToStaticMarkup(tree), /does not repair or replace the original/);
  tree.props.children.find((child) => child?.type === "button").props.onClick();
  tree = fixture.render();
  media = tree.props.children.find((child) => ["img", "audio", "video"].includes(child?.type));
  assert.notEqual(media.key, initialKey);
  assert.notEqual(media.props.src, fixture.inspection.preview_url, "retry must request a new resource URL rather than reuse a cached browser failure");
  const original = new URL(fixture.inspection.preview_url, "http://fixture.local");
  const retried = new URL(media.props.src, "http://fixture.local");
  assert.equal(retried.pathname, original.pathname);
  assert.equal(retried.searchParams.get("_preview_retry"), "1");
  retried.searchParams.delete("_preview_retry");
  assert.deepEqual([...retried.searchParams], [...original.searchParams], "all source identity pins must remain unchanged");
  (media.props.onLoad ?? media.props.onLoadedMetadata)();
  assert.doesNotMatch(renderToStaticMarkup(fixture.render()), /Loading original media|could not load or decode/);
});
