// Byline: Codex · GPT-6 · 2026-10-05. Application capture preflight and durable dispatch boundaries.
import assert from 'node:assert/strict';
import test from 'node:test';
import { createHash } from 'node:crypto';
import { Readable } from 'node:stream';
import { configuredLibrarySyncScope, prepareLibraryEditCapture } from '../dist/library-sync-integration.js';
import { drainLibrarySyncOutbox } from '../dist/library-sync-dispatch.js';
import { handleLibrarySyncHttpRequest } from '../dist/library-sync-http.js';

const account = 'b2-account-sha256:' + 'a'.repeat(64);
const operation = '11111111-1111-4111-8111-111111111111';

/** Temporarily configure one synthetic account without persisting settings.
 * Inputs: test callback. Outputs: callback result. Effects: restored process environment only; choose for sequential configuration tests.
 */
async function configured(callback) {
  const previous = process.env.TOOLKIT_LIBRARY_SYNC_ACCOUNT_SCOPE;
  process.env.TOOLKIT_LIBRARY_SYNC_ACCOUNT_SCOPE = account;
  try { return await callback(); }
  finally { if (previous === undefined) delete process.env.TOOLKIT_LIBRARY_SYNC_ACCOUNT_SCOPE; else process.env.TOOLKIT_LIBRARY_SYNC_ACCOUNT_SCOPE = previous; }
}

test('unbound personal records capture a distinct complete JSON export in the same transaction', async () => configured(async () => {
  const store = { db: { query: async () => [[]] } };
  const capture = await prepareLibraryEditCapture(store, 'child:synthetic', '$full', '$version', '$revision');
  assert.equal(capture.params.sync_record_id, 'child:synthetic');
  assert.match(capture.sql, /binding_appeared_retry/);
  assert.match(capture.sql, /CREATE type::record\('library_file'/);
  assert.match(capture.sql, /typed_snapshot: \$full/);
  assert.match(capture.params.sync_new_key, /family-court-records\/child\/[a-f0-9]{64}\.json$/);
  assert.doesNotMatch(capture.sql, /COMMIT TRANSACTION/);
}));

test('bound edits capture native full snapshots and refuse a mismatched account', async () => configured(async () => {
  const binding = { id: 'library_file:' + 'b'.repeat(64), account_scope: account, bucket: 'salem-data', legal_root: 'consignatio/casevault/KnowledgeBase/legal/', pointer_revision: 'pointer-1' };
  const store = { db: { query: async () => [[binding]] } };
  const capture = await prepareLibraryEditCapture(store, 'child:synthetic', '$full', '$version', '$revision');
  assert.match(capture.sql, /typed_snapshot: \$full/);
  assert.equal(capture.params.sync_base_revision, 'pointer-1');
  assert.doesNotMatch(capture.sql, /COMMIT TRANSACTION/);
  binding.account_scope = 'other-account';
  await assert.rejects(prepareLibraryEditCapture(store, 'child:synthetic', '$full', '$version', '$revision'), /scope mismatch/);
}));

test('configured scope refuses an invented account identity', async () => configured(async () => {
  assert.equal(configuredLibrarySyncScope().account_scope, account);
  process.env.TOOLKIT_LIBRARY_SYNC_ACCOUNT_SCOPE = 'guess';
  assert.throws(configuredLibrarySyncScope, /Invalid/);
}));

test('outbox recovery skips blocked snapshots and never reads current record bodies', async () => {
  const calls = [];
  const store = { db: { query: async (sql, params) => { calls.push({ sql, params }); return [[{ operation_id: operation, status: 'pending_encoding' }]]; } } };
  const result = await drainLibrarySyncOutbox(store, async id => { assert.equal(id, operation); return { status: 'blocked' }; });
  assert.deepEqual(result, { attempted: 0, queued: 0 });
  assert.equal(calls.length, 1);
  assert.doesNotMatch(calls[0].sql, /SELECT \*/);
});

test('incoming raw content arrives byte-exact with bound payload and source hashes', async () => {
  const bytes = Buffer.from('# Unabridged synthetic private context\r\n', 'utf8');
  const sha = createHash('sha256').update(bytes).digest('hex');
  const req = Readable.from([bytes.subarray(0, 5), bytes.subarray(5)]);
  req.method = 'POST';
  req.headers = { authorization: 'Bearer ' + 'x'.repeat(64), 'content-type': 'text/markdown', 'x-toolkit-sync-payload-sha256': sha, 'x-toolkit-sync-source-sha256': sha };
  const res = { writeHead(status) { this.status = status; }, end(body) { this.body = body; } };
  let called = false;
  await handleLibrarySyncHttpRequest(req, res, new URL('http://console/api/internal/library-sync/observations/' + 'a'.repeat(64) + '/payload'), async input => {
    called = true; assert.deepEqual(input.body.incoming_bytes, bytes); assert.equal(input.body.source_sha256, sha);
    return { body: { status: 'retained' } };
  }, 'x'.repeat(64));
  assert.equal(called, true); assert.equal(res.status, 200);
});

test('mismatched incoming bytes refuse before a privileged backend call', async () => {
  const req = Readable.from(['changed']); req.method = 'POST';
  req.headers = { authorization: 'Bearer ' + 'x'.repeat(64), 'content-type': 'text/markdown', 'x-toolkit-sync-payload-sha256': 'a'.repeat(64), 'x-toolkit-sync-source-sha256': 'b'.repeat(64) };
  const res = { writeHead(status) { this.status = status; }, end() {} };
  await handleLibrarySyncHttpRequest(req, res, new URL('http://console/api/internal/library-sync/observations/' + 'a'.repeat(64) + '/payload'), async () => { throw new Error('must not dispatch'); }, 'x'.repeat(64));
  assert.equal(res.status, 422);
});
