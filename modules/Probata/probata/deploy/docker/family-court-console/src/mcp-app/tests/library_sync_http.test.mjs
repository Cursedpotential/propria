// Byline: Codex · GPT-6 · 2026-10-05. Private service admission and bounded transport proof.
import assert from "node:assert/strict";
import test from "node:test";
import { Readable } from "node:stream";
import { handleLibrarySyncHttpRequest, librarySyncAuthorized, parseLibrarySyncHttpOperation, readLibrarySyncMetadata } from "../dist/library-sync-http.js";

const token = "a".repeat(64);
const prefix = "http://console/api/internal/library-sync";
const operationId = "00000000-0000-0000-0000-000000000001";

/** Build an in-memory HTTP response probe without opening a socket.
 * Inputs: none. Outputs: status/header/body recorder. Effects: no network or database access.
 * Choose to prove auth-before-dispatch and exact response bytes with the real handler.
 */
function response() {
  return { status: 0, headers: {}, body: null,
    writeHead(status, headers) { this.status = status; this.headers = headers; },
    end(body) { this.body = body; },
  };
}

test("only the dedicated complete bearer is accepted", () => {
  assert.equal(librarySyncAuthorized("Bearer " + token, token), true);
  for (const header of [undefined, "Bearer " + token.slice(1), "Basic " + token, "Bearer ordinary-mcp-token"]) {
    assert.equal(librarySyncAuthorized(header, token), false);
  }
  assert.equal(librarySyncAuthorized("Bearer " + token, undefined), false);
  assert.equal(librarySyncAuthorized("Bearer short", "short"), false);
});

test("authentication occurs before body reads or backend dispatch", async () => {
  let dispatches = 0;
  const request = { method: "POST", headers: { authorization: "Bearer another-service" },
    async *[Symbol.asyncIterator]() { throw new Error("unauthorized body was read"); },
  };
  const res = response();
  assert.equal(await handleLibrarySyncHttpRequest(request, res, new URL(prefix + "/outbox/claim"),
    async () => { dispatches++; }, token), true);
  assert.equal(res.status, 401);
  assert.equal(dispatches, 0);
});

test("route admission rejects arbitrary operations, duplicate versions and missing leases", () => {
  assert.throws(() => parseLibrarySyncHttpOperation("POST", new URL(prefix + "/query")), { status: 404 });
  assert.throws(() => parseLibrarySyncHttpOperation("GET", new URL(prefix + "/outbox/claim")), { status: 405 });
  const original = prefix + "/bindings/library_file%3A" + "f".repeat(64) + "/original";
  assert.equal(parseLibrarySyncHttpOperation("GET", new URL(original + "?version_id=retained-1")).path,
    "/bindings/library_file:" + "f".repeat(64) + "/original");
  for (const query of ["?version_id=a&version_id=b", "?version_id=null", "?version_id=a&key=other", "?version_id=%0A"]) {
    assert.throws(() => parseLibrarySyncHttpOperation("GET", new URL(original + query)), { status: 400 });
  }
  assert.throws(() => parseLibrarySyncHttpOperation("GET", new URL(prefix + "/outbox/" + operationId + "/payload")), { status: 400 });
});

test("metadata budgets refuse complete over-limit input rather than truncate it", async () => {
  await assert.rejects(readLibrarySyncMetadata(Readable.from([Buffer.alloc(2 * 1024 * 1024 + 1)])), { status: 413 });
  for (const body of ["[]", "null", "invalid"]) {
    await assert.rejects(readLibrarySyncMetadata(Readable.from([body])), { status: 400 });
  }
  assert.deepEqual(await readLibrarySyncMetadata(Readable.from(['{"operation_id":"', operationId, '"}'])), { operation_id: operationId });
});

test("payload responses preserve exact bytes and private headers", async () => {
  const bytes = Buffer.from([0, 1, 2, 255]);
  const request = Readable.from([]);
  request.method = "GET";
  request.headers = { authorization: "Bearer " + token, "x-toolkit-sync-lease": "current-lease" };
  const res = response();
  await handleLibrarySyncHttpRequest(request, res, new URL(prefix + "/outbox/" + operationId + "/payload"), async op => {
    assert.equal(op.leaseId, "current-lease");
    return { body: bytes };
  }, token);
  assert.equal(res.status, 200);
  assert.deepEqual(res.body, bytes);
  assert.equal(res.headers["content-length"], bytes.length);
  assert.equal(res.headers["cache-control"], "private, no-store");
});

test("private backend failures cannot leak exception text or credential values", async () => {
  const request = Readable.from(['{"operation_id":"' + operationId + '"}']);
  request.method = "POST";
  request.headers = { authorization: "Bearer " + token, "content-type": "application/json" };
  const res = response();
  await handleLibrarySyncHttpRequest(request, res, new URL(prefix + "/outbox/claim"), async () => {
    throw Object.assign(new Error("credential=" + token), { status: 409 });
  }, token);
  assert.equal(res.status, 409);
  assert.deepEqual(JSON.parse(res.body), { error: "sync_operation_refused" });
  assert.equal(String(res.body).includes(token), false);
});
