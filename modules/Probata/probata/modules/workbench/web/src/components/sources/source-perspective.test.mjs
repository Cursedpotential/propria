// Byline: Claude Code · Opus 5.5 · 2026-10-16
// Owner rule 2026-10-15: Katrina's number -> hers; otherwise mine; screenshots decided per item.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import ts from "typescript";

function load() {
  const source = readFileSync(new URL("./source-perspective.ts", import.meta.url), "utf8");
  const compiled = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS } }).outputText;
  const module = { exports: {} };
  new Function("module", "exports", "require", compiled)(module, module.exports, () => ({}));
  return module.exports;
}

const phone = (value, status = "confirmed") => ({ kind: "phone", status, normalized: value, raw_value: value });
const matt = { id: "matt", display_name: "Matthew S. Salem", role_in_case: "user", verification_state: "confirmed", identifiers: [phone("8103535467")] };
const katrina = { id: "katrina", display_name: "Katrina Kinzel", role_in_case: "co_parent", verification_state: "confirmed", identifiers: [phone("+18102689630")] };
const placeholder = { id: "ph", display_name: "Unknown 313", role_in_case: "unknown", verification_state: "proposed", identifiers: [phone("3135550101")] };
const people = [matt, katrina, placeholder];
const vault = "b2://salem-data/consignatio/casevault/SourceCorpus/messaging";

test("a backup from Katrina's device number is hers", () => {
  const { ruleSourcePerson } = load();
  assert.equal(ruleSourcePerson(people, `${vault}/sms-backup-restore/8102689630/sms-2024-11-24.xml?versionId=4_z1`).id, "katrina");
});

test("my Facebook export stays mine although the thread folder is named after her", () => {
  const { ruleSourcePerson } = load();
  assert.equal(ruleSourcePerson(people, `${vault}/facebook-messenger/meta-2025-08-18/katrinakinzel_1614217831996100/message_3.json`).id, "matt");
});

test("an unknown or placeholder device number is mine", () => {
  const { ruleSourcePerson } = load();
  assert.equal(ruleSourcePerson(people, `${vault}/sms-backup-restore/2485550199/sms-1.xml`).id, "matt");
  assert.equal(ruleSourcePerson(people, `${vault}/sms-backup-restore/3135550101/sms-1.xml`).id, "matt");
});

test("screenshots and photos get no default", () => {
  const { ruleSourcePerson, defaultSourcePerspective } = load();
  assert.equal(ruleSourcePerson(people, `${vault}/screenshots/Screenshot_20250601.png?versionId=1`), null);
  assert.equal(defaultSourcePerspective(people, [`${vault}/screenshots/IMG_1.JPG`]), null);
});

test("a mixed selection has no default; an empty one has none", () => {
  const { defaultSourcePerspective } = load();
  assert.equal(defaultSourcePerspective(people, [`${vault}/sms-backup-restore/8102689630/a.xml`, `${vault}/facebook-messenger/x/message_1.json`]), null);
  assert.equal(defaultSourcePerspective(people, [`${vault}/sms-backup-restore/8102689630/a.xml`, `${vault}/sms-backup-restore/8102689630/b.xml`]).id, "katrina");
  assert.equal(defaultSourcePerspective(people, []), null);
});

test("message JSON exports now require a perspective", () => {
  const { requiresMessagePerspective } = load();
  assert.equal(requiresMessagePerspective("message_3.json", "message_export_json"), true);
  assert.equal(requiresMessagePerspective("notes.txt", "delimited_text"), false);
});
