// Byline: Codex · GPT-5 · 2026-09-12
import assert from "node:assert/strict";
import { readdir, readFile } from "node:fs/promises";
import { test } from "node:test";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const vendor = path.join(root, "vendor", "propria-design-contract");

async function text(relativePath) {
  return readFile(path.join(root, relativePath), "utf8");
}

async function sourceFiles(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const nested = await Promise.all(
    entries.map(async (entry) => {
      const next = path.join(directory, entry.name);
      return entry.isDirectory() ? sourceFiles(next) : [next];
    }),
  );
  return nested.flat();
}

test("vendors the complete pinned Propria design contract", async () => {
  const required = [
    "CALLABILITY.md",
    "CHANGELOG.md",
    "README.md",
    "adapters/advocatio.css",
    "adapters/intake.css",
    "adapters/portal.css",
    "adapters/probata.css",
    "build.mjs",
    "examples/probata-two-surface-sample.html",
    "package.json",
    "theme.css",
    "tokens.css",
    "tokens.json",
    "verify.mjs",
  ];

  const files = (await sourceFiles(vendor))
    .map((file) => path.relative(vendor, file).replaceAll(path.sep, "/"))
    .sort();
  assert.deepEqual(files, required.sort());

  const receipt = await text("vendor/PROPRIA-DESIGN-CONTRACT.md");
  assert.match(receipt, /c6da1419e2baaea0c481047083f178d46ac14ca7/);
  assert.match(receipt, /9dcb135efedfb05b423e8e7e4b64529b09b556a9/);
});

test("loads tokens, theme, adapter, then product CSS and stays General-first", async () => {
  const layout = await text("src/app/layout.tsx");
  const imports = [
    "tokens.css",
    "theme.css",
    "adapters/advocatio.css",
    "./globals.css",
  ].map((needle) => layout.indexOf(needle));

  assert.ok(imports.every((index) => index >= 0), "all design layers must be imported");
  assert.deepEqual(imports, [...imports].sort((left, right) => left - right));
  assert.match(layout, /data-pr-theme="dark"/);
  assert.match(layout, /data-pr-experience="general"/);
  assert.match(layout, /className="pr-app"/);
});

test("keeps palette control in the shared token source", async () => {
  const css = await text("src/app/globals.css");
  const tsxFiles = (await sourceFiles(path.join(root, "src"))).filter((file) => file.endsWith(".tsx"));
  const tsx = (await Promise.all(tsxFiles.map((file) => readFile(file, "utf8")))).join("\n");

  assert.doesNotMatch(css, /#[0-9a-f]{3,8}\b/i);
  assert.doesNotMatch(css, /rgba?\(/i);
  assert.doesNotMatch(css, /!important/i);
  assert.doesNotMatch(tsx, /#[0-9a-f]{3,8}\b/i);
  assert.match(css, /--bg:\s*var\(--pr-canvas\)/);
  assert.match(css, /outline:\s*3px solid var\(--ring\)/);
  assert.match(css, /min-height:\s*var\(--pr-context-strip-min-height\)/);
});

test("renders workspace connectivity and confidentiality status", async () => {
  const shell = await text("src/components/TerminalShell.tsx");

  assert.match(shell, /Confidential setting: local only/);
  assert.match(shell, /Confidential mode:/);
  assert.match(shell, /confidentialEnforcement/);
  assert.match(shell, /Available.*workspace backend/s);
  assert.match(shell, /Unavailable.*workspace backend/s);
});
