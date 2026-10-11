// Byline: Codex · GPT-6.1-Sol · 2026-10-05.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { runInNewContext } from "node:vm";
import test from "node:test";

const require = createRequire(import.meta.url);
const ts = require("typescript");
const source = (path) => readFileSync(new URL(path, import.meta.url), "utf8");
const modeSource = source("../src/lib/operating-mode.ts");

function load(code, extra = {}) {
  const module = { exports: {} };
  const compiled = ts.transpileModule(code, { compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX } }).outputText;
  runInNewContext(compiled, { module, exports: module.exports, URLSearchParams, ...extra });
  return module.exports;
}

const { parseOperatingMode, selectAdmittedCourtCase } = load(modeSource);
test("shell court selection uses the admitted ID, never foreign primary flags", () => {
  const approved = { id: "approved", is_primary: false };
  const foreign = { id: "foreign", is_primary: true };
  const matter = { admitted_court_case_id: "approved", court_cases: [foreign, approved] };
  assert.equal(selectAdmittedCourtCase(matter), approved);
  approved.is_primary = true;
  foreign.is_primary = false;
  assert.equal(selectAdmittedCourtCase(matter), approved);
  assert.equal(selectAdmittedCourtCase({ ...matter, admitted_court_case_id: undefined }), null);
  assert.equal(selectAdmittedCourtCase({ ...matter, court_cases: [foreign] }), null);
});
test("omission is Live, aliases normalize only at input, unknown URL values reject", () => {
  for (const [raw, canonical] of [[null, "LIVE"], ["LIVE", "LIVE"], ["DEV", "DEV"], ["REAL", "LIVE"], ["TEST", "DEV"]]) {
    assert.equal(parseOperatingMode(raw), canonical);
  }
  for (const raw of ["", "unknown", "dev", "live"]) assert.throws(() => parseOperatingMode(raw), /Unknown operating mode/);
});

test("all Case write client methods carry selected canonical policy with actor-key boundary intact", async () => {
  const calls = [];
  const client = load(source("../src/lib/case-identity-client.ts"), {
    require: () => ({ ApiError: Error }),
    fetch: async (path, init) => {
      calls.push({ path, init });
      return { ok: true, json: async () => ({ ref: "receipt" }) };
    },
  });
  const operations = [
    (mode) => client.addIdentifier({}, "key", mode),
    (mode) => client.editIdentifier("alias", {}, "key", mode),
    (mode) => client.deleteIdentifier("alias", {}, "key", mode),
    (mode) => client.editCaseHeader(mode, {}, "key"),
    (mode) => client.editPerson("person", {}, "key", mode),
    (mode) => client.addPerson({}, "key", mode),
    (mode) => client.addPlaceholders({}, "key", mode),
    (mode) => client.mergePerson("person", {}, "key", mode),
    (mode) => client.triageIdentifier({}, "key", mode),
  ];
  for (const mode of ["DEV", "LIVE"]) {
    for (const invoke of operations) {
      await invoke(mode);
      const call = calls.at(-1);
      assert.equal(new URLSearchParams(call.path.split("?")[1]).get("mode"), mode);
      assert.equal(call.init.method, "POST");
      assert.equal(call.init.headers["Idempotency-Key"], "key");
    }
  }
  assert.equal(calls.length, 18);
});

test("mobile identity actions honor explicit URL Dev and reject unknown flags without a desktop provider", () => {
  const mobileWindow = { location: { search: "" } };
  const context = load(source("../src/lib/fixed-case-context.tsx"), {
    window: mobileWindow,
    require: (name) => name === "@/lib/operating-mode" ? { parseOperatingMode } :
      name === "react" ? { createContext: () => ({}), useContext: () => null } : {},
  });
  assert.equal(context.useOperatingMode(), "LIVE");
  mobileWindow.location.search = "?mode=DEV";
  assert.equal(context.useOperatingMode(), "DEV");
  mobileWindow.location.search = "?mode=TEST";
  assert.equal(context.useOperatingMode(), "DEV");
  mobileWindow.location.search = "?mode=unexpected";
  assert.throws(() => context.useOperatingMode(), /Unknown operating mode/);
});

test("policy changes remount open Case dialogs and header uses current policy, never cached view.mode", () => {
  const context = source("../src/lib/fixed-case-context.tsx");
  assert.match(context, /invalidMode \? <div role="alert">\{invalidMode\}<\/div> : children/);
  const screen = source("../src/components/case/case-identity-screen.tsx");
  assert.match(screen, /ModeScopedCaseIdentityScreen key=\{mode\}/);
  assert.match(screen, /HeaderDialog mode=\{mode\}/);
  assert.doesNotMatch(screen, /HeaderDialog mode=\{view.mode\}/);
  const dialogs = source("../src/components/case/identity-dialogs.tsx");
  assert.match(dialogs, /save\(body, key, mode\)/);
  const who = source("../src/components/identity/who-is-this.tsx");
  assert.match(who, /useOperatingMode\(\)/);
  assert.match(who, /newIdempotencyKey\("merge"\), operatingMode/);
});
