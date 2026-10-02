import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import ts from "typescript";

const helperSource = await readFile(new URL("../src/lib/advocatio-link.ts", import.meta.url), "utf8");
const componentSource = await readFile(new URL("../src/components/case/case-identity-screen.tsx", import.meta.url), "utf8");
const { outputText } = ts.transpileModule(helperSource, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
});
const helperUrl = `data:text/javascript;base64,${Buffer.from(outputText).toString("base64")}`;
const { buildAdvocatioEntityUrl } = await import(helperUrl);

test("keeps the native Probata person UUID in the Advocatio return URL", () => {
  const id = "9e4a826f-3f7d-44b2-955f-35d071fc0997";
  const destination = new URL(buildAdvocatioEntityUrl(id));

  assert.equal(destination.origin, "https://legal.tilapia-skilift.ts.net");
  assert.equal(destination.pathname, "/claims");
  assert.equal(destination.searchParams.get("probata_kind"), "entity");
  assert.equal(destination.searchParams.get("probata_id"), id);
});

test("rejects malformed identities and unsafe configured destinations", () => {
  const id = "9e4a826f-3f7d-44b2-955f-35d071fc0997";

  assert.equal(buildAdvocatioEntityUrl("catalog:case-bible-42"), null);
  assert.equal(buildAdvocatioEntityUrl("../claims?redirect=attacker"), null);
  assert.equal(buildAdvocatioEntityUrl("00000000-0000-0000-0000-000000000000"), null);
  assert.equal(buildAdvocatioEntityUrl(id, "javascript:alert(1)"), null);
  assert.equal(buildAdvocatioEntityUrl(id, "http://legal.example"), null);
  assert.equal(buildAdvocatioEntityUrl(id, "https://user:pass@legal.example"), null);
  assert.equal(buildAdvocatioEntityUrl(id, "https://legal.example/redirect?to=elsewhere"), null);
});

test("renders an external read-only anchor from the PersonCard's native person ID", () => {
  const personCard = componentSource.slice(componentSource.indexOf("function PersonCard("), componentSource.indexOf("function UnknownList("));

  assert.match(personCard, /buildAdvocatioEntityUrl\(person\.id, import\.meta\.env\.VITE_ADVOCATIO_WEB_ORIGIN\)/);
  assert.match(personCard, /<a[\s\S]*?href=\{legalResponseUrl\}[\s\S]*?target="_blank"[\s\S]*?rel="noopener noreferrer"[\s\S]*?>\s*Open legal response/);
  assert.doesNotMatch(personCard, /item\.identifier|catalog\.entity_id/);
});
