// Byline: Codex · GPT-6.1-Sol · 2026-10-05.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import test from "node:test";
import { runInNewContext } from "node:vm";

const require = createRequire(import.meta.url);
const ts = require("typescript");
const read = (path) => readFileSync(new URL(path, import.meta.url), "utf8");
const digest = "ab".repeat(32);
const file = { size: 42, type: "application/xml", name: "original.xml" };

/** Load the real client against a deterministic XHR boundary, without a browser.
 * Inputs: response/event overrides. Output: client and captured requests.
 * Effects: transpiles only; does not transmit bytes. Pick for upload wire contracts.
 */
function fixture(overrides = {}) {
  const calls = [];
  class Xhr {
    constructor() {
      this.listeners = {};
      this.upload = { addEventListener: (event, listener) => { this.progress = listener; } };
      this.headers = {};
      calls.push(this);
    }
    addEventListener(event, listener) { this.listeners[event] = listener; }
    open(method, url) { this.method = method; this.url = url; }
    setRequestHeader(key, value) { this.headers[key] = value; }
    send(body) {
      this.body = body;
      this.progress({ lengthComputable: true, loaded: 21, total: 42 });
      this.status = overrides.status ?? 201;
      this.responseText = overrides.text ?? JSON.stringify(overrides.receipt ?? {
        acquisition_ref: `upload://${digest}`, sha256: digest, byte_length: 42, matter_mode: "LIVE",
      });
      this.listeners[overrides.event ?? "load"]();
    }
  }
  const module = { exports: {} };
  const code = read("../src/lib/api-client.ts").replace("import.meta.env.VITE_API_URL", '""');
  const compiled = ts.transpileModule(code, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText;
  runInNewContext(compiled, { module, exports: module.exports, XMLHttpRequest: Xhr, URLSearchParams,
    FormData: class { constructor() { assert.fail("Canonical acquisition must not wrap original bytes"); } } });
  return { client: module.exports, calls };
}

test("fresh upload sends the exact File with selected policy, credentials, MIME and progress", async () => {
  for (const mode of ["LIVE", "DEV"]) {
    const receipt = { acquisition_ref: `upload://${digest}`, sha256: digest, byte_length: 42, matter_mode: mode };
    const { client, calls } = fixture({ receipt });
    const progress = [];
    assert.equal((await client.uploadProfferSource(file, mode, (value) => progress.push(value))).acquisition_ref, receipt.acquisition_ref);
    const [call] = calls;
    assert.equal(call.method, "POST");
    assert.equal(call.url, `/api/proffer/upload?mode=${mode}`);
    assert.equal(call.withCredentials, true);
    assert.equal(call.headers["Content-Type"], "application/xml");
    assert.equal(call.body, file);
    assert.deepEqual(progress, [50]);
    assert.equal(calls.length, 1);
  }
  const { client, calls } = fixture();
  await client.uploadProfferSource({ ...file, type: "" }, "LIVE");
  assert.equal(calls[0].headers["Content-Type"], "application/octet-stream");
});

test("canonical receipt rejects scope, bytes, digest and retired locator contradictions", async () => {
  for (const change of [{ matter_mode: "DEV" }, { byte_length: 41 }, { sha256: "invalid" },
    { acquisition_ref: `r2://nexus/${digest}` }, { acquisition_ref: `upload://${"cd".repeat(32)}` }]) {
    const { client } = fixture({ receipt: { acquisition_ref: `upload://${digest}`, sha256: digest,
      byte_length: 42, matter_mode: "LIVE", ...change } });
    await assert.rejects(client.uploadProfferSource(file, "LIVE"), (error) => error.status === 502 && /exact bytes/.test(error.message));
  }
});

test("upload preserves bounded upstream errors and network/abort feedback", async () => {
  for (const status of [401, 409, 413, 503]) {
    const { client } = fixture({ status, text: JSON.stringify({ detail: `upstream-${status}` }) });
    await assert.rejects(client.uploadProfferSource(file, "DEV"), (error) => error.status === status && error.message === `upstream-${status}`);
  }
  for (const [event, message] of [["error", /Network error/], ["abort", /Upload aborted/]]) {
    const { client } = fixture({ event });
    await assert.rejects(client.uploadProfferSource(file, "LIVE"), (error) => error.status === 0 && message.test(error.message));
  }
  const { client } = fixture({ text: "not-json" });
  await assert.rejects(client.uploadProfferSource(file, "LIVE"), /Malformed response/);
});

test("deprecated upload refuses locally before XHR, and actual form shows canonical receipt metadata", async () => {
  const { client, calls } = fixture();
  await assert.rejects(client.uploadFile(file), (error) => error.status === 410 && /retired/.test(error.message));
  assert.equal(calls.length, 0);
  const form = read("../src/components/upload/upload-form.tsx");
  assert.match(form, /useOperatingMode\(\)/);
  assert.match(form, /uploadProfferSource\(item.file, mode,/);
  assert.doesNotMatch(form, /uploadFile\(|UploadResult|r2_key/);
  for (const field of ["name", "mime", "byte_length", "sha256", "acquisition_ref", "matter_mode"]) assert.match(form, new RegExp(`r\\.${field}`));
});
