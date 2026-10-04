// Byline: OpenAI Codex · GPT-6 · 2026-10-04 — guard the Toolkit UI against backend table drift.
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { test } from "node:test";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

test("Toolkit records exposes every backend-supported table", async () => {
  const [page, service] = await Promise.all([
    readFile(path.join(root, "src/app/toolkit/page.tsx"), "utf8"),
    readFile(path.resolve(root, "../api/legal_workspace/services/family_court_toolkit.py"), "utf8"),
  ]);
  const pageTables = page.match(/const TABLES = \[([\s\S]*?)\] as const;/)?.[1];
  const backendTables = service.match(/TOOLKIT_TABLES = \(([^)]*)\)/)?.[1];

  assert.ok(pageTables, "page table navigation is present");
  assert.ok(backendTables, "backend table allowlist is present");
  const pageIds = [...pageTables.matchAll(/\bid:\s*"([a-z_]+)"/g)].map((match) => match[1]).sort();
  const backendIds = [...backendTables.matchAll(/"([a-z_]+)"/g)].map((match) => match[1]).sort();
  assert.deepEqual(pageIds, backendIds);
});
