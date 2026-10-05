// Byline: OpenAI Codex · GPT-6 · 2026-10-04 — guard the Toolkit UI against backend table drift.
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { test } from "node:test";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

/** Checks the backend read allowlist against toolkit DATA_TABLES and page routes.
 * Inputs: source files read from the worktree. Outputs: assertion result. Effects: read-only smoke verification.
 * Choose to catch table drift without requiring the parent page to own every personal tab here.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04.
 */
test("Toolkit records exposes every backend-supported table", async () => {
  const [page, service] = await Promise.all([
    readFile(path.join(root, "src/app/toolkit/page.tsx"), "utf8"),
    readFile(path.resolve(root, "../api/legal_workspace/services/family_court_toolkit.py"), "utf8"),
  ]);
  const toolkitStore = await readFile(path.resolve(root, "../../Probata/probata/deploy/docker/family-court-console/src/mcp-app/src/store.ts"), "utf8");
  const pageTables = page.match(/const TABLES = \[([\s\S]*?)\] as const;/)?.[1];
  const backendTables = service.match(/TOOLKIT_TABLES = \(([^)]*)\)/)?.[1];
  const sourceTables = toolkitStore.match(/export const DATA_TABLES = \[([\s\S]*?)\] as const;/)?.[1];

  assert.ok(pageTables, "page table navigation is present");
  assert.ok(backendTables, "backend table allowlist is present");
  assert.ok(sourceTables, "toolkit DATA_TABLES source is present");
  const pageIds = [...pageTables.matchAll(/\bid:\s*"([a-z_]+)"/g)].map((match) => match[1]).sort();
  const backendIds = [...backendTables.matchAll(/"([a-z_]+)"/g)].map((match) => match[1]).sort();
  const sourceIds = [...sourceTables.matchAll(/"([a-z_]+)"/g)].map((match) => match[1]).sort();
  const protectedTables = new Set(["library_validation", "library_proposal", "library_revision"]);
  assert.deepEqual(backendIds, sourceIds.filter((table) => !protectedTables.has(table)));
  assert.deepEqual(pageIds, backendIds, "page navigation exposes every shared table without a parallel catalog");
});
