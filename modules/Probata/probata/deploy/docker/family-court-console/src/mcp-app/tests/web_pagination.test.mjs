// Byline: Codex · GPT-6 · 2026-10-04
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { createServer as createNetServer } from "node:net";
import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
import test from "node:test";
import { transformSync } from "esbuild";

const BEARER_TOKEN = "test-pagination-bearer-5ac9";
const AUTH_HEADERS = { "x-authentik-uid": "pagination-test", "x-authentik-username": "pagination-test" };

// Byline: Codex · GPT-6 · 2026-10-04
/** Reserve an available loopback port for an isolated HTTP test process.
 * Inputs: none.
 * Outputs: a Promise resolving to the assigned TCP port.
 * Side effects: briefly opens and closes a loopback server socket.
 * Use this instead of fixed ports so parallel tests do not collide.
 */
async function findFreePort() {
  return new Promise((resolvePort, reject) => {
    const probe = createNetServer();
    probe.once("error", reject);
    probe.listen(0, "127.0.0.1", () => {
      const { port } = probe.address();
      probe.close(() => resolvePort(port));
    });
  });
}

// Byline: Codex · GPT-6 · 2026-10-04
/** Start the console with an ephemeral in-memory store and always stop the child process.
 * Inputs: an async callback that receives the authenticated test server's base URL.
 * Outputs: resolves after the callback and process cleanup; rejects on test assertion or setup failure.
 * Side effects: starts a local child process with `CUSTODY_CASE_DB=mem://` and synthetic proxy identity.
 * Use for route integration tests; pure host paging behavior is covered by the helper tests below.
 */
async function withHttpServer(fn) {
  const port = await findFreePort();
  const baseUrl = `http://127.0.0.1:${port}`;
  const child = spawn(process.execPath, [resolve("dist/server.js")], {
    env: {
      ...process.env,
      MCP_TRANSPORT: "http",
      MCP_HTTP_HOST: "127.0.0.1",
      MCP_HTTP_PORT: String(port),
      MCP_BEARER_TOKEN: BEARER_TOKEN,
      CUSTODY_CASE_DB: "mem://",
      TRUSTED_AUTH_PROXY_CIDRS: "127.0.0.1/32",
    },
    stdio: ["ignore", "pipe", "pipe"],
  });
  let stderr = "";
  child.stderr.on("data", (chunk) => { stderr += chunk.toString(); });
  const exitPromise = new Promise((done) => child.once("exit", done));
  try {
    const deadline = Date.now() + 20000;
    while (Date.now() < deadline) {
      try {
        if ((await fetch(`${baseUrl}/healthz`)).ok) break;
      } catch {}
      await new Promise((done) => setTimeout(done, 100));
    }
    assert.equal(child.exitCode, null, `HTTP server exited before readiness: ${stderr}`);
    await fn(baseUrl);
  } finally {
    child.kill();
    let exitTimer;
    const exited = await Promise.race([
      exitPromise.then(() => true),
      new Promise((done) => { exitTimer = setTimeout(() => done(false), 3000); }),
    ]).finally(() => clearTimeout(exitTimer));
    if (!exited && child.exitCode === null && child.signalCode === null) {
      child.kill("SIGKILL");
      await exitPromise;
    }
  }
}

// Byline: Codex · GPT-6 · 2026-10-04
/** Send one HTTP request with the test-only Authentik identity headers.
 * Inputs: server base URL, path, and optional fetch initialization.
 * Outputs: the raw Fetch Response for status and JSON assertions.
 * Side effects: makes an authenticated request to the ephemeral local test server.
 * Use for protected web routes; `/healthz` remains intentionally open.
 */
async function authenticatedFetch(baseUrl, path, init = {}) {
  return fetch(`${baseUrl}${path}`, { ...init, headers: { ...AUTH_HEADERS, ...(init.headers ?? {}) } });
}

// Byline: Codex · GPT-6 · 2026-10-04
/** Compile one named TypeScript helper from the browser host for direct, DOM-free testing.
 * Inputs: full host source, a declaration prefix, and the exact following declaration marker.
 * Outputs: the helper function evaluated from its TypeScript-transformed source.
 * Side effects: creates an isolated JavaScript function with no browser globals.
 * Use only for pure helper functions when the desktop test environment has no browser DOM.
 */
function loadHostHelper(source, declaration, endMarker) {
  const start = source.indexOf(declaration);
  const end = source.indexOf(endMarker, start);
  assert.ok(start >= 0 && end > start, `${declaration} helper is present in the host source`);
  const functionName = declaration.startsWith("export function ") ? declaration.slice("export function ".length).split(/[<( ]/, 1)[0] : declaration.slice("function ".length).split(/[<( ]/, 1)[0];
  const code = source.slice(start, end + 2).replace(/^function /, "export function ");
  const compiled = transformSync(code, { loader: "ts", format: "cjs" }).code;
  const loaded = { exports: {} };
  new Function("module", "exports", compiled)(loaded, loaded.exports);
  return loaded.exports[functionName];
}

// Byline: Codex · GPT-6 · 2026-10-04
/** Exercise real HTTP paging across the former 1,000-row cap and reject invalid requests.
 * Inputs: none; the test creates only ephemeral synthetic notes in its child process.
 * Outputs: passes when all rows arrive once in deterministic order and bad parameters return 400.
 * Side effects: seeds the isolated memory database and terminates it after the assertion.
 * Use this as the records-route regression test; it does not contact persistent case data.
 */
test("records paging crosses 1,000 rows in stable bounded pages and rejects invalid inputs", async () => {
  await withHttpServer(async (baseUrl) => {
    const batchSize = 80;
    for (let start = 0; start < 1001; start += batchSize) {
      const statements = [];
      for (let i = start; i < Math.min(start + batchSize, 1001); i += 1) {
        const id = `mobilepage${String(i).padStart(4, "0")}`;
        statements.push(`CREATE note:${id} SET title = "synthetic pagination row"`);
      }
      const seeded = await authenticatedFetch(baseUrl, "/api/tools/case_query", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ arguments: { surql: statements.join(";"), write: true } }),
      });
      assert.equal(seeded.status, 200);
      const result = await seeded.json();
      assert.notEqual(result.isError, true, JSON.stringify(result));
      assert.equal(result.structuredContent?.available, true, JSON.stringify(result));
    }

    const firstResponse = await authenticatedFetch(baseUrl, "/api/records?table=note&limit=1000");
    assert.equal(firstResponse.status, 200);
    const first = await firstResponse.json();
    assert.equal(first.records.length, 1000);
    assert.ok(first.total > 1000);
    assert.equal(first.nextCursor, "1000");
    assert.deepEqual(first.records.map((record) => record.id), [...first.records.map((record) => record.id)].sort());

    const secondResponse = await authenticatedFetch(baseUrl, `/api/records?table=note&limit=1000&cursor=${first.nextCursor}`);
    assert.equal(secondResponse.status, 200);
    const second = await secondResponse.json();
    assert.ok(second.records.length >= 1);
    assert.equal(second.total, first.total);
    assert.equal(second.nextCursor, null);
    const ids = [...first.records, ...second.records].map((record) => record.id);
    assert.equal(ids.length, first.total);
    assert.equal(new Set(ids).size, ids.length);
    assert.deepEqual(ids, [...ids].sort());

    for (const query of ["limit=0", "limit=1001", "limit=1.5", "limit=nope", "cursor=-1", "cursor=1e3", "cursor=2147483648"]) {
      const response = await authenticatedFetch(baseUrl, `/api/records?table=note&${query}`);
      assert.equal(response.status, 400, query);
      assert.match((await response.json()).error, /whole number/);
    }
  });
});

// Byline: Codex · GPT-6 · 2026-10-04
/** Exercise the production library page helper with more than 300 rows and a changed filter.
 * Inputs: none; file metadata is synthetic and remains in process memory.
 * Outputs: passes when later pages remain accessible and a new filter starts at page one.
 * Side effects: reads the host TypeScript source and evaluates only the pure helper in isolation.
 * Use this where browser automation is unavailable; the interactive host itself is not started.
 */
test("library pages include rows beyond 300, apply filters, and reset after a filter change", async () => {
  const hostSource = (await readFile(resolve("web/host.ts"), "utf8")).replace(/\r\n/g, "\n");
  const libraryPage = loadHostHelper(hostSource, "export function libraryPage<", "\n}\n\nfunction recordList");
  const files = Array.from({ length: 420 }, (_, index) => ({
    path: `${index < 315 ? "guides" : "forms"}/item-${String(index).padStart(3, "0")}.md`,
    size: index,
  }));

  const pageOne = libraryPage(files, "", 1);
  const pageFour = libraryPage(files, "", 4);
  assert.equal(pageOne.total, 420);
  assert.equal(pageOne.visible.length, 100);
  assert.equal(pageFour.visible.length, 100);
  assert.notEqual(pageOne.visible[0].path, pageFour.visible[0].path);

  const filteredAfterPageFour = libraryPage(files, "guides/", 4, "");
  assert.equal(filteredAfterPageFour.page, 1);
  assert.equal(filteredAfterPageFour.total, 315);
  assert.equal(filteredAfterPageFour.visible.length, 100);
  assert.ok(filteredAfterPageFour.visible.every((file) => file.path.startsWith("guides/")));
  assert.equal(libraryPage(files, "guides/", 4, "guides/").visible.length, 15);
});

// Byline: Codex · GPT-6 · 2026-10-04
/** Verify the browser page validator exposes empty, nonadvancing and duplicate pages as errors.
 * Inputs: none; all page responses and table state are synthetic values.
 * Outputs: passes when each inconsistent response throws a descriptive paging error.
 * Side effects: evaluates the pure production validator without starting a browser host.
 * Use this to check the error paths rendered by the explicit table “Show more” action.
 */
test("record page validation rejects empty pages with remaining rows and repeated cursors or IDs", async () => {
  const hostSource = (await readFile(resolve("web/host.ts"), "utf8")).replace(/\r\n/g, "\n");
  const validateRecordPage = loadHostHelper(hostSource, "function validateRecordPage(", "\n}\n\n// Byline: Codex · GPT-6 · 2026-10-04\n/** Fetch and append exactly one bounded page");
  const state = () => ({ table: "note", rows: [], total: 0, cursor: null, seenCursors: new Set(), seenIds: new Set() });

  assert.throws(
    () => validateRecordPage({ records: [], total: 1, nextCursor: "0" }, state(), null),
    /empty page while 1 records remain/,
  );
  assert.throws(
    () => validateRecordPage({ records: [{ id: "note:a", kind: "note", record: {} }], total: 2, nextCursor: "0" }, state(), null),
    /did not advance/,
  );
  const repeated = state();
  repeated.seenIds.add("note:a");
  assert.throws(
    () => validateRecordPage({ records: [{ id: "note:a", kind: "note", record: {} }], total: 1, nextCursor: null }, repeated, null),
    /repeated record note:a/,
  );
});
