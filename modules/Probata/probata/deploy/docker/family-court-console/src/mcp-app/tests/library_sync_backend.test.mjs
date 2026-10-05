// Byline: Codex · GPT-6.1 Sol · 2026-10-05. Synthetic fixtures only; native proof runs on VPS in an isolated retained database.
import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createHash, randomUUID } from 'node:crypto';
import { mkdirSync } from 'node:fs';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { join, dirname } from 'node:path';
import { DateTime, Decimal, Duration, RecordId, Uuid } from 'surrealdb';

const here = dirname(fileURLToPath(import.meta.url));
let modulePath = process.env.LIBRARY_SYNC_PROOF_MODULE;
if (!modulePath) {
  const { build } = await import('esbuild');
  const retained = join(here, '../to_be_deleted', 'library-sync-tests-' + randomUUID());
  mkdirSync(retained, { recursive: true });
  modulePath = join(retained, 'library-sync-backend.mjs');
  await build({ entryPoints: [join(here, '../src/library-sync-backend.ts')], outfile: modulePath, platform: 'node', format: 'esm', bundle: true, packages: 'external' });
}
const lib = await import(pathToFileURL(modulePath).href);
const scope = { account_scope: 'synthetic-account', bucket: 'salem-data', legal_root: 'consignatio/casevault/KnowledgeBase/legal/' };
const key = scope.legal_root + 'reference-data/synthetic.md';
const op = '11111111-1111-4111-8111-111111111111';
const binding = lib.libraryFileBindingId(scope, key);

/** Provide queued database responses without interpreting or simulating SQL; inputs: response fixtures; outputs: store and captured calls.
 * Effects: records bounded test calls only; choose for transport/encoding tests, never to claim native transaction guarantees.
 */
function scripted(...values) {
  const calls = [];
  const store = { db: { query(sql, params) { calls.push({ sql, params }); if (!values.length) throw new Error('unexpected database call'); return Promise.resolve(values.shift()); } } };
  return { store, calls };
}
/** Compute an independent fixture digest; inputs: full text; outputs: raw hash; effects: none; choose for byte-integrity expectations. */
function digest(value) { return createHash('sha256').update(value).digest('hex'); }
/** Build a saved immutable outbox fixture; inputs: codec/snapshot overrides; outputs: fixture; effects: none; choose for sealing boundaries. */
function snapshot(overrides = {}) {
  return { operation_id: op, binding_id: binding, record_id: 'reference:synthetic', record_version: 'sha256:' + 'a'.repeat(64), snapshot_hash: 'b'.repeat(64), status: 'pending_encoding', codec_version: 'markdown-utf8/1', key, content_type: 'text/markdown', typed_snapshot: { id: new RecordId('reference', 'synthetic'), body: '# Exact body\r\n\nPrivate synthetic context 🧾\n' }, ...overrides };
}

test('stable binding matches independently specified Go encoding/json vector including HTML and line separator escaping', () => {
  const weird = scope.legal_root + 'case-law/a<&>\u2028.md';
  const goBytes = '["b2","synthetic-account","salem-data","consignatio/casevault/KnowledgeBase/legal/case-law/a\\u003c\\u0026\\u003e\\u2028.md"]';
  assert.equal(lib.libraryFileBindingId(scope, weird), 'library_file:' + digest(goBytes));
  assert.notEqual(lib.libraryFileBindingId(scope, weird), lib.libraryFileBindingId(scope, weird.replace('/a<', '/A<')));
});
test('scope refuses R2, other buckets, ambiguous paths and unapproved child roots', () => {
  for (const invalid of ['../bad.md', scope.legal_root + 'other/a.md', key + '/', key.replace('reference-data', 'reference-data/..'), key.replace('/synthetic', '//synthetic')]) assert.throws(() => lib.libraryFileBindingId(scope, invalid));
  assert.throws(() => lib.libraryFileBindingId({ ...scope, bucket: 'r2-copy' }, key));
});
test('capture fragment is parameterized and cannot become a postcommit transaction or accept injected SQL variables', () => {
  const captured = lib.buildLibrarySyncCapture({ operation_id: op, binding_id: binding, snapshot_variable: '$full_native', record_version_variable: '$version', revision_ref_variable: '$revision', expected_pointer_revision: 'r1' });
  assert.equal(captured.params.sync_base_revision, 'r1');
  assert.ok(captured.sql.includes('typed_snapshot: $full_native'));
  assert.ok(!/BEGIN TRANSACTION|COMMIT TRANSACTION/.test(captured.sql));
  assert.throws(() => lib.buildLibrarySyncCapture({ operation_id: op, binding_id: binding, snapshot_variable: '$x; RETURN true', record_version_variable: '$version', revision_ref_variable: '$revision', expected_pointer_revision: 'r1' }));
});

test('binding export guards reject ignored Markdown companions, missing companions, shared original keys and invalid physical destinations before DB access', async () => {
  const s = scripted();
  for (const input of [
    { key, format: 'markdown', record_export_key: key + '.json' },
    { key, format: 'record' },
    { key: key + '.json', format: 'record', record_export_key: key + '.json' },
    { key, format: 'record', record_export_key: key + '.md' },
    { key, format: 'record', record_export_key: 'outside.json' }
  ]) await assert.rejects(lib.bindLibraryFile(s.store, scope, { record_id: 'source:synthetic', ...input }));
  await assert.rejects(lib.bindLibraryFile(s.store, scope, { key, format: 'markdown', record_id: 'library_sync_outbox:client_selected' }), /invalid_mapped_record/);
  assert.equal(s.calls.length, 0);
});
test('Markdown sealing preserves actual UTF-8 body and CRLF and reads only the saved outbox', async () => {
  const saved = snapshot();
  const s = scripted([{ sync_result: saved }], [{ sync_result: { status: 'pending', operation_id: op } }]);
  await lib.sealLibrarySyncOperation(s.store, op);
  const write = s.calls[1].params;
  assert.equal(write.payload, saved.typed_snapshot.body);
  assert.equal(write.payload_hash, digest(saved.typed_snapshot.body));
  assert.equal(write.payload_size, Buffer.byteLength(saved.typed_snapshot.body));
  assert.equal(write.snapshot_hash, saved.snapshot_hash);
  assert.ok(!s.calls[0].sql.includes("type::record('reference'"));
});
test('record companion preserves unknown/private JSON and native scalar JSON-pointer annotations', async () => {
  const saved = snapshot({ codec_version: 'surreal-record-json/1', key: key + '.record.json', content_type: 'application/json', typed_snapshot: { id: new RecordId('reference', 'synthetic'), private_unknown: { 'a/b~c': new Decimal('123.456'), full_name: 'Synthetic Private Name' }, time: new DateTime('2026-10-05T00:00:00.123456789Z'), uuid: new Uuid(op), duration: new Duration('3s'), huge: 9223372036854775807n, bytes: Uint8Array.from([0, 255]), none: undefined, array: [new RecordId('source', 'private')] } });
  const s = scripted([{ sync_result: saved }], [{ sync_result: {} }], [{ sync_result: { status: 'pending' } }]);
  await lib.sealLibrarySyncOperation(s.store, op);
  const exported = JSON.parse(s.calls[2].params.payload);
  assert.equal(exported.record.private_unknown.full_name, 'Synthetic Private Name');
  assert.equal(exported.record.huge, '9223372036854775807');
  assert.equal(exported.types['/record/private_unknown/a~1b~0c'], 'decimal');
  assert.equal(exported.types['/record/array/0'], 'record');
  assert.equal(exported.types['/record/none'], 'none');
  assert.equal(exported.types['/record/time'], 'datetime');
  assert.equal(exported.record.time, '2026-10-05T00:00:00.123456789Z');
  assert.notEqual(s.calls[2].params.payload_hash, saved.record_version.replace('sha256:', ''));
});
test('unsupported native values retain snapshot and seal a visible blocked state', async () => {
  const saved = snapshot({ codec_version: 'surreal-record-json/1', key: key + '.json', content_type: 'application/json', typed_snapshot: { id: new RecordId('reference', 'synthetic'), unsupported: new Map([['private', 'retained']]) } });
  const s = scripted([{ sync_result: saved }], [{ sync_result: { status: 'blocked' } }]);
  await lib.sealLibrarySyncOperation(s.store, op);
  assert.equal(s.calls[1].params.status, 'blocked');
  assert.equal(s.calls[1].params.code, 'unsupported_native_type');
  assert.equal(s.calls[1].params.payload, '');
  assert.ok(!s.calls[1].sql.includes('typed_snapshot ='));
});
test('payload overflow and invalid Unicode block without truncating captured content', async () => {
  for (const body of ['x'.repeat(lib.LIBRARY_SYNC_LIMITS.payload + 1), '\uD800']) {
    const s = scripted([{ sync_result: snapshot({ typed_snapshot: { body } }) }], [{ sync_result: { status: 'blocked' } }]);
    await lib.sealLibrarySyncOperation(s.store, op);
    assert.equal(s.calls[1].params.status, 'blocked');
    assert.equal(s.calls[1].params.payload_size, 0);
  }
});

test('native annotation failure retains a visible sealing error without persisting partial payload or private exception text', async () => {
  const saved = snapshot({ codec_version: 'surreal-record-json/1', key: key + '.json', content_type: 'application/json' });
  const calls = [];
  const store = { db: { query(sql, params) {
    calls.push({ sql, params });
    if (calls.length === 1) return Promise.resolve([{ sync_result: saved }]);
    if (calls.length === 2) return Promise.reject(new Error('Synthetic private diagnostic that must not become an error code'));
    return Promise.resolve([{ sync_result: { status: 'blocked', error_code: params.code } }]);
  } } };
  const outcome = await lib.sealLibrarySyncOperation(store, op);
  assert.equal(outcome.status, 'blocked');
  assert.equal(outcome.error_code, 'native_annotation_failed');
  assert.equal(calls[2].params.payload, '');
  assert.equal(calls[2].params.payload_size, 0);
  assert.ok(!calls[2].sql.includes('typed_snapshot ='));
});
test('already sealed operations are never reencoded or overwritten', async () => {
  const s = scripted([{ sync_result: snapshot({ status: 'pending', payload_utf8: 'saved bytes' }) }]);
  assert.equal((await lib.sealLibrarySyncOperation(s.store, op)).status, 'pending');
  assert.equal(s.calls.length, 1);
});
test('worker auth denies before database access, and metadata overflow is not parsed', async () => {
  const s = scripted();
  const backend = lib.createLibrarySyncBackend(s.store, scope);
  const response = await backend.dispatch(new Request('https://private/api/internal/library-sync/outbox/claim', { method: 'POST', body: '{}' }));
  assert.equal(response.status, 401); assert.equal(s.calls.length, 0);
  const authenticated = lib.createLibrarySyncBackend(s.store, scope, () => true);
  const oversized = await authenticated.dispatch(new Request('https://private/api/internal/library-sync/outbox/claim', { method: 'POST', headers: { 'content-length': String(lib.LIBRARY_SYNC_LIMITS.metadata + 1) }, body: '{}' }));
  assert.equal(oversized.status, 413); assert.equal(s.calls.length, 0);
});
test('payload transport checks raw bytes against both stored hash and size', async () => {
  for (const changed of [{ payload_sha256: '0'.repeat(64) }, { payload_size: 1 }]) {
    const s = scripted([{ sync_result: { payload_utf8: '🧾 exact bytes', payload_sha256: digest('🧾 exact bytes'), payload_size: Buffer.byteLength('🧾 exact bytes'), ...changed } }]);
    await assert.rejects(lib.createLibrarySyncBackend(s.store, scope).payload(op, 'lease'), /integrity/);
  }
});
test('parent dispatcher adapter uses decoded route and returns exact raw bytes with no JSON wrapper', async () => {
  const bytes = Buffer.from('🧾 exact bytes'); const calls = [];
  const dispatch = lib.createLibrarySyncDispatcher({ payload: async (...args) => { calls.push(args); return bytes; } });
  const result = await dispatch({ method: 'GET', path: '/outbox/' + op + '/payload', query: new URLSearchParams(), body: {}, leaseId: 'lease' });
  assert.strictEqual(result.body, bytes); assert.equal(result.contentType, 'application/octet-stream'); assert.deepEqual(calls, [[op, 'lease']]);
  await assert.rejects(dispatch({ method: 'GET', path: '/other', query: new URLSearchParams(), body: {} }), /route_missing/);
});
test('native transaction errors surface a bounded symbolic failure instead of private DB text or NotExecuted', async () => {
  const db = { query() { return { responses: async () => [{ success: false, error: { message: 'not executed' } }, { success: false, error: { message: 'sync:physical_target_busy private unknown payload' } }] }; } };
  await assert.rejects(lib.createLibrarySyncBackend({ db }, scope).claim({ operation_id: op, attempt_id: 'attempt' }), /^Error: sync:physical_target_busy$/);
});

/** Run adversarial sync transactions on the VPS's isolated native engine and retain all fixtures on disk.
 * Inputs: LIBRARY_SYNC_NATIVE_PROOF=1 and explicit non-production fixture directory; outputs: native node:test cases.
 * Effects: synthetic records only, unique retained namespace/database, connection closed without removing namespace/files.
 * Choose to prove real SQL/CAS/race behavior; never connect to fct/case or read production credentials.
 */
async function nativeProof(t) {
  const { Surreal } = await import('surrealdb');
  const { createNodeEngines } = await import('@surrealdb/node');
  assert.equal(process.platform, 'linux', 'native proof must run on VPS');
  const retained = process.env.LIBRARY_SYNC_FIXTURES;
  assert.ok(retained?.includes('/to_be_deleted/toolkit-sync-backend-20261005/'));
  mkdirSync(retained, { recursive: true });
  const db = new Surreal({ engines: createNodeEngines() });
  const nativeKeepalive = setInterval(() => {}, 1000);
  const store = { db: {
    /** Preserve native responses while exposing synthetic fixture errors for diagnosis.
     * Inputs: proof SQL/parameters; outputs: SDK query; effects: logs only non-personal fixture failure messages; use only in this isolated proof.
     */
    query(sql, params) {
      const pending = db.query(sql, params);
      const responses = pending.responses.bind(pending);
      pending.responses = async () => {
        const values = await responses();
        for (const failure of values.filter(x => !x.success && !/not executed|failed transaction/i.test(x.error?.message ?? ''))) console.log('Synthetic native diagnostic: ' + failure.error?.message);
        return values;
      };
      return pending;
    }
  } };
  try {
    await db.connect('surrealkv://' + join(retained, 'case.db'));
    console.log('Native isolated retained database connected');
    await db.use({ namespace: 'toolkit_sync_backend_20261005_' + Date.now(), database: 'isolated_synthetic' });
    await db.query('DEFINE TABLE reference SCHEMALESS; DEFINE TABLE source SCHEMALESS; DEFINE TABLE record_revision SCHEMALESS; DEFINE TABLE library_proposal SCHEMALESS;');
    await lib.migrateLibrarySync(store);
    const backend = lib.createLibrarySyncBackend(store, scope);

    /** Create an independent server mapping fixture; inputs: suffix/format; outputs: binding; effects: synthetic record+mapping only. */
    async function mapped(suffix, format = 'markdown') {
      const originalKey = scope.legal_root + 'reference-data/' + suffix + (format === 'markdown' ? '.md' : '.pdf');
      await db.query("CREATE type::record('reference', $key) CONTENT { body: 'original', personal_unknown: { full_name: 'Synthetic Private Name' }, native_time: d'2026-10-05T00:00:00.123456789Z', native_decimal: 12.345dec, native_int: 1, native_float: <float> 1, embedding: [1, 2] };", { key: suffix });
      return lib.bindLibraryFile(store, scope, { key: originalKey, record_id: 'reference:' + suffix, format, ...(format === 'markdown' ? {} : { record_export_key: originalKey + '.record.json' }) });
    }
    /** Capture a native edit and outbox inside one actual transaction; inputs: binding/body/base; outputs: operation ID; effects: synthetic edit+revision+outbox. */
    async function capture(b, body = 'edited 🧾\r\n', base = b.pointer_revision) {
      const operationId = randomUUID();
      const part = lib.buildLibrarySyncCapture({ operation_id: operationId, binding_id: b.id, snapshot_variable: '$full', record_version_variable: '$rv', revision_ref_variable: '$revision', expected_pointer_revision: base });
      const pending = db.query(`BEGIN TRANSACTION;
        LET $rid = type::record($record_table, $record_key);
        UPDATE $rid SET body = $body;
        LET $versioned = object::extend((SELECT * OMIT embedding FROM ONLY $rid), { id: $rid });
        LET $rv = 'sha256:' + crypto::sha256(<string> $versioned);
        LET $revision = type::record('record_revision', $operation);
        CREATE $revision CONTENT { current_record: (SELECT * FROM ONLY $rid), current_version: $rv };
        LET $full = object::extend((SELECT * FROM ONLY $rid), { id: $rid });
        ${part.sql}
        COMMIT TRANSACTION;`, { ...part.params, record_table: b.record_id.split(':')[0], record_key: b.record_id.split(':')[1], body, operation: operationId });
      const responses = await pending.responses();
      const failed = responses.filter(x => !x.success);
      if (failed.length) {
        const identity = (await db.query("LET $r=(SELECT * FROM ONLY type::record('reference',$key)); LET $b=(SELECT * FROM ONLY type::record('library_file',$binding)); RETURN { record_cast:<string> $r.id, mapped_id:$b.record_id, record_type:type::is_object($r) };", { key: b.record_id.split(':')[1], binding: b.id.split(':')[1] })).at(-1);
        throw new Error(((failed.find(x => !/not executed|failed transaction/i.test(x.error?.message ?? '')) ?? failed[0]).error?.message ?? 'Native capture failed') + ' synthetic identity ' + JSON.stringify(identity));
      }
      await lib.sealLibrarySyncOperation(store, operationId);
      return operationId;
    }
    /** Read exact fixture bookkeeping; inputs: table/key; outputs: raw native row; effects: read; choose for proof assertions only. */
    async function row(table, recordKey) { return (await db.query('SELECT * FROM ONLY type::record($table, $key);', { table, key: recordKey })).at(-1); }
    /** Expire synthetic leases without waiting; inputs: operation; outputs: none; effects: isolated lease fixture mutation; choose for recovery tests. */
    async function expire(operationId) {
      await db.query("UPDATE type::record('library_sync_outbox', $op) SET lease_expires_at = time::now() - 1s; UPDATE library_sync_target SET lease_expires_at = time::now() - 1s WHERE active_operation = $op;", { op: operationId });
    }
    /** Produce full independent exact-byte worker evidence; inputs: claim/intent/version; outputs: Completion; effects: none; choose for reconciliation fixture. */
    function evidence(claim, intent, version = 'provider-retained-v1') {
      const p = { version_id: version, sha256: claim.operation.payload_sha256, size: claim.operation.payload_size, content_type: claim.operation.content_type, observed_at: '2026-10-05T12:00:01Z' };
      return { intent_id: intent.intent_id, lease_id: claim.lease_id, fence: claim.fence, base_pointer_revision: claim.operation.base_pointer_revision, written_pointer: p, observed_versions: [{ bucket: claim.operation.bucket, key: claim.operation.key, version_id: version, sha256: p.sha256, size: p.size, content_type: p.content_type, hidden: false, latest: true, uploaded_at: '2026-10-05T12:00:00Z', operation_id: claim.operation.operation_id, intent_id: intent.intent_id }], coverage: 'complete', status: 'synced' };
    }

    await t.test('same-transaction capture rollback and full snapshot survive a later live-record mutation', async () => {
      const b = await mapped('capture');
      await assert.rejects(capture(b, 'must rollback', 'stale-base'));
      assert.equal((await row('reference', 'capture')).body, 'original');
      const id = await capture(b);
      const saved = await row('library_sync_outbox', id);
      assert.equal(saved.typed_snapshot.personal_unknown.full_name, 'Synthetic Private Name');
      assert.deepEqual(saved.typed_snapshot.embedding, [1, 2]);
      assert.equal(String(saved.typed_snapshot.id), 'reference:capture');
      assert.ok(saved.typed_snapshot.native_time instanceof DateTime);
      await db.query("UPDATE reference:capture SET body = 'later live mutation';");
      assert.equal((await row('library_sync_outbox', id)).payload_utf8, 'edited 🧾\r\n');
      const claim = await backend.claim({ operation_id: id, attempt_id: 'first' });
      await assert.rejects(backend.beginWrite(id, { lease_id: claim.lease_id, fence: claim.fence, attempt_id: 'first' }), /write_base_conflict/);
    });
    await t.test('different bindings cannot share a physical destination even when original and companion roles differ', async () => {
      const b = await mapped('unique', 'pdf');
      await db.query("CREATE reference:other CONTENT { body: 'other' };");
      await assert.rejects(lib.bindLibraryFile(store, scope, { key: b.record_export_key, record_id: 'reference:other', format: 'json', record_export_key: b.record_export_key + '.json' }));
      assert.equal((await db.query('SELECT count() AS n FROM library_file WHERE record_id = $id GROUP ALL;', { id: 'reference:other' })).at(-1)[0]?.n ?? 0, 0);
    });
    await t.test('source alias cannot bind the canonical original but its separate JSON binding can capture and complete without touching that original', async () => {
      const primary = await mapped('alias-primary', 'pdf');
      await db.query("CREATE source:alias_edit CONTENT { kind: 'case_document', body: 'personal source', private_unknown: 'retained' };");
      await assert.rejects(lib.bindLibraryFile(store, scope, { key: primary.key, record_id: 'source:alias_edit', format: 'record', record_export_key: primary.key + '.source.json' }), /binding_exists/);
      const ownKey = scope.legal_root + 'reference-data/alias-source.record.json';
      const separate = await lib.bindLibraryFile(store, scope, { key: ownKey, record_id: 'source:alias_edit', format: 'record', record_export_key: ownKey + '.export.json' });
      const id = await capture(separate, 'personal edit retained');
      const claim = await backend.claim({ operation_id: id, attempt_id: 'source-edit' });
      assert.equal(claim.operation.record_id, 'source:alias_edit');
      assert.equal(claim.operation.key, separate.record_export_key);
      const intent = await backend.beginWrite(id, { lease_id: claim.lease_id, fence: claim.fence, attempt_id: 'source-edit' });
      assert.equal((await backend.complete(id, evidence(claim, intent))).status, 'synced');
      const untouched = await row('library_file', primary.id.split(':')[1]);
      assert.equal(untouched.pointer_revision, primary.pointer_revision);
      assert.equal(untouched.record_export_pointer, null);
      assert.equal((await row('source', 'alias_edit')).private_unknown, 'retained');
    });
    await t.test('all existing personal tables admit distinct server-owned JSON bindings and native capture/write CAS without granting client mapping', async () => {
      for (const table of ['person', 'child', 'order', 'hearing', 'deadline', 'event', 'message', 'exhibit', 'factor', 'source', 'note', 'court', 'court_event', 'filing', 'draft', 'memo', 'evidence_log', 'eval', 'case_status']) {
        await db.query("CREATE type::record($table, 'mapped_personal') CONTENT { kind: 'case_document', body: 'synthetic full private context', private_unknown: { retained: true }, embedding: [3, 4] };", { table });
        const originalKey = scope.legal_root + 'reference-data/personal-' + table + '.json';
        const b = await lib.bindLibraryFile(store, scope, { key: originalKey, record_id: table + ':mapped_personal', format: 'record', record_export_key: originalKey + '.record.json' });
        const id = await capture(b);
        const claim = await backend.claim({ operation_id: id, attempt_id: 'personal-table' });
        const intent = await backend.beginWrite(id, { lease_id: claim.lease_id, fence: claim.fence, attempt_id: 'personal-table' });
        assert.equal((await backend.complete(id, evidence(claim, intent))).status, 'synced', table);
        const saved = await row('library_sync_outbox', id);
        assert.deepEqual(saved.typed_snapshot.embedding, [3, 4], table);
        assert.equal(JSON.parse(saved.payload_utf8).record.private_unknown.retained, true, table);
      }
    });
    await t.test('claim requires seal and operation immutable fields never rebase on claim', async () => {
      const b = await mapped('unsealed'); const id = await capture(b);
      const saved = await row('library_sync_outbox', id);
      await db.query("UPDATE type::record('library_sync_outbox', $op) SET status = 'pending_encoding';", { op: id });
      await assert.rejects(backend.claim({ operation_id: id, attempt_id: 'a' }), /unclaimable/);
      await db.query("UPDATE type::record('library_sync_outbox', $op) SET status = 'pending';", { op: id });
      const claim = await backend.claim({ operation_id: id, attempt_id: 'a' });
      assert.equal(claim.operation.base_pointer_revision, saved.base_pointer_revision);
      assert.equal(claim.operation.base_b2_pointer, null);
      assert.equal(claim.operation.record_version, saved.record_version);
    });
    await t.test('concurrent write intent requests yield one durable PUT permission', async () => {
      const b = await mapped('intent'); const id = await capture(b);
      const c = await backend.claim({ operation_id: id, attempt_id: 'a' });
      const request = { lease_id: c.lease_id, fence: c.fence, attempt_id: 'a' };
      const attempts = await Promise.allSettled([backend.beginWrite(id, request), backend.beginWrite(id, request)]);
      assert.equal(attempts.filter(x => x.status === 'fulfilled' && x.value.may_write).length, 1);
      const retry = await backend.beginWrite(id, request);
      assert.equal(retry.may_write, false); assert.ok(retry.intent_id);
    });
    await t.test('expired intent refresh keeps old intent, rejects stale fence, and blocks other operations on the same target', async () => {
      const b = await mapped('recover'); const first = await capture(b); const c = await backend.claim({ operation_id: first, attempt_id: 'a' });
      const intent = await backend.beginWrite(first, { lease_id: c.lease_id, fence: c.fence, attempt_id: 'a' });
      await expire(first);
      const second = await capture(b, 'newer revision');
      await assert.rejects(backend.claim({ operation_id: second, attempt_id: 'b' }), /physical_target_busy/);
      const refreshed = await backend.claim({ operation_id: first, attempt_id: 'reconcile' });
      assert.ok(refreshed.fence > c.fence); assert.equal(refreshed.write_intent_id, intent.intent_id);
      assert.equal((await backend.beginWrite(first, { lease_id: refreshed.lease_id, fence: refreshed.fence, attempt_id: 'reconcile' })).may_write, false);
      await assert.rejects(backend.complete(first, evidence(c, intent)), /stale_lease/);
      assert.equal((await backend.complete(first, evidence(refreshed, intent))).status, 'conflicted');
      assert.equal((await row('reference', 'recover')).body, 'newer revision');
    });
    await t.test('complete CAS accepts exact bytes once and keeps original separate from accepted pointer', async () => {
      const b = await mapped('complete'); const id = await capture(b); const c = await backend.claim({ operation_id: id, attempt_id: 'a' });
      const intent = await backend.beginWrite(id, { lease_id: c.lease_id, fence: c.fence, attempt_id: 'a' });
      const completion = evidence(c, intent);
      assert.equal((await backend.complete(id, completion)).status, 'synced');
      assert.equal((await backend.complete(id, completion)).status, 'synced');
      const saved = await row('library_file', b.id.split(':')[1]);
      assert.equal(saved.original_pointer, null); assert.equal(saved.record_export_pointer, null);
      assert.equal(saved.accepted_pointer.version_id, completion.written_pointer.version_id);
      assert.notEqual(saved.pointer_revision, b.pointer_revision);
    });
    await t.test('PDF metadata companion preserves full native values without updating PDF pointers', async () => {
      const b = await mapped('pdf', 'pdf'); const id = await capture(b); const c = await backend.claim({ operation_id: id, attempt_id: 'a' });
      const exported = JSON.parse((await backend.payload(id, c.lease_id)).toString());
      assert.equal(exported.record.personal_unknown.full_name, 'Synthetic Private Name'); assert.equal(exported.types['/record/id'], 'record');
      assert.equal(exported.types['/record/native_decimal'], 'decimal'); assert.equal(exported.types['/record/native_time'], 'datetime');
      assert.equal(exported.types['/record/native_int'], 'int'); assert.equal(exported.types['/record/native_float'], 'float');
      assert.deepEqual(exported.record.embedding, [1, 2]);
      const intent = await backend.beginWrite(id, { lease_id: c.lease_id, fence: c.fence, attempt_id: 'a' });
      assert.equal((await backend.complete(id, evidence(c, intent))).status, 'synced');
      const saved = await row('library_file', b.id.split(':')[1]);
      assert.equal(saved.original_pointer, null); assert.equal(saved.accepted_pointer, null); assert.ok(saved.record_export_pointer);
    });
    for (const flaw of ['coverage', 'hash', 'size', 'intent', 'hide', 'duplicate', 'competing', 'base', 'live_record']) await t.test('completion retains conflict for ' + flaw, async () => {
      const b = await mapped('flaw_' + flaw); const id = await capture(b); const c = await backend.claim({ operation_id: id, attempt_id: 'a' });
      const intent = await backend.beginWrite(id, { lease_id: c.lease_id, fence: c.fence, attempt_id: 'a' });
      const e = evidence(c, intent);
      if (flaw === 'coverage') e.coverage = 'partial';
      if (flaw === 'hash') e.observed_versions[0].sha256 = '0'.repeat(64);
      if (flaw === 'size') e.observed_versions[0].size++;
      if (flaw === 'intent') e.observed_versions[0].intent_id = 'other';
      if (flaw === 'hide') e.observed_versions[0].hidden = true;
      if (flaw === 'duplicate') e.observed_versions.push({ ...e.observed_versions[0] });
      if (flaw === 'competing') e.observed_versions.push({ ...e.observed_versions[0], latest: false, operation_id: 'external', intent_id: 'external', version_id: 'external-retained-version' });
      if (flaw === 'base') e.base_pointer_revision = 'stale';
      if (flaw === 'live_record') await db.query("UPDATE type::record('reference', $key) SET body = 'new app revision retained';", { key: 'flaw_' + flaw });
      const out = await backend.complete(id, e);
      assert.equal(out.status, flaw === 'coverage' ? 'write_unknown' : 'conflicted');
      assert.equal((await row('library_file', b.id.split(':')[1])).accepted_pointer, null);
      const conflicts = (await db.query('SELECT * FROM library_sync_conflict WHERE operation_id = $id;', { id })).at(-1);
      assert.equal(conflicts.length, 1); assert.equal(conflicts[0].completion.observed_versions.length, e.observed_versions.length);
      assert.equal(conflicts[0].typed_snapshot.personal_unknown.full_name, 'Synthetic Private Name');
    });
    await t.test('same-base R2 cannot write after R1 CAS succeeds and cannot silently adopt R1 baseline', async () => {
      const b = await mapped('r1_r2'); const r1 = await capture(b); const r2 = await capture(b);
      const c1 = await backend.claim({ operation_id: r1, attempt_id: 'a' }); const i1 = await backend.beginWrite(r1, { lease_id: c1.lease_id, fence: c1.fence, attempt_id: 'a' });
      assert.equal((await backend.complete(r1, evidence(c1, i1))).status, 'synced');
      const c2 = await backend.claim({ operation_id: r2, attempt_id: 'b' });
      assert.equal(c2.operation.base_pointer_revision, b.pointer_revision);
      await assert.rejects(backend.beginWrite(r2, { lease_id: c2.lease_id, fence: c2.fence, attempt_id: 'b' }), /write_base_conflict/);
    });
    await t.test('observations retain citations-required references and hides, reject identity collision, never publish or alter pointers', async () => {
      const b = await mapped('observe');
      const observation = { contract_version: lib.LIBRARY_SYNC_CONTRACT, observation_id: lib.librarySyncObservationId(b.id, 'external-v1'), binding_id: b.id, object: { bucket: scope.bucket, key: b.key, version_id: 'external-v1', size: 10, latest: true, hidden: false, uploaded_at: '2026-10-05T12:00:00Z', sha256: 'd'.repeat(64), content_type: 'text/markdown' }, raw_ref: { uri: 'b2://salem-data/synthetic-retained-original', version_id: 'derivative-v1', sha256: 'sha256:' + 'd'.repeat(64), bytes: 10 }, status: 'citation_required' };
      const out = await backend.observe(observation);
      assert.equal(out.status, 'citation_required'); assert.deepEqual(await backend.observe(observation), out);
      assert.equal((await backend.seen(observation.observation_id)).seen, true);
      await assert.rejects(backend.observe({ ...observation, object: { ...observation.object, sha256: 'e'.repeat(64) } }), /identity_conflict/);
      const hidden = await backend.observe({ ...observation, observation_id: lib.librarySyncObservationId(b.id, 'hide-v2'), object: { ...observation.object, version_id: 'hide-v2', hidden: true }, status: 'blocked' });
      assert.equal(hidden.status, 'hidden'); assert.equal((await row('reference', 'observe')).body, 'original');
      assert.equal((await row('library_file', b.id.split(':')[1])).accepted_pointer, null);
      const original = await backend.original(b.id, 'external-v1'); assert.equal(original.original_pointer.version_id, 'external-v1'); assert.equal(original.record_export_pointer, null);
      await assert.rejects(backend.original(b.id, 'latest'), /not_retained/);
    });
    await t.test('failure retains uppercase worker code and never clears an uncertain intent', async () => {
      const b = await mapped('failure'); const id = await capture(b); const c = await backend.claim({ operation_id: id, attempt_id: 'a' });
      const intent = await backend.beginWrite(id, { lease_id: c.lease_id, fence: c.fence, attempt_id: 'a' });
      const out = await backend.failure(id, { lease_id: c.lease_id, fence: c.fence, status: 'retry_wait', error_code: 'PUT_REPLY_LOST' });
      assert.equal(out.status, 'write_unknown'); assert.equal(out.error_code, 'PUT_REPLY_LOST');
      assert.equal((await row('library_sync_outbox', id)).write_intent_id, intent.intent_id);
    });
    await t.test('native unsupported scalar blocks visibly and retains the complete saved snapshot', async () => {
      const b = await mapped('unsupported_native', 'pdf');
      await db.query("UPDATE reference:unsupported_native SET unsupported_geometry = { type: 'Point', coordinates: [1.1, 2.2] };");
      const id = await capture(b);
      const saved = await row('library_sync_outbox', id);
      assert.equal(saved.status, 'blocked'); assert.equal(saved.error_code, 'unsupported_native_type');
      assert.ok(saved.typed_snapshot.unsupported_geometry); assert.equal(saved.typed_snapshot.personal_unknown.full_name, 'Synthetic Private Name');
      await assert.rejects(backend.claim({ operation_id: id, attempt_id: 'a' }), /unclaimable/);
    });
    await t.test('native oversized export retains all source bytes and blocks without truncating', async () => {
      const b = await mapped('payload_overflow');
      const original = 'x'.repeat(lib.LIBRARY_SYNC_LIMITS.payload + 1);
      const id = await capture(b, original);
      const saved = await row('library_sync_outbox', id);
      assert.equal(saved.status, 'blocked'); assert.equal(saved.error_code, 'payload_budget_exceeded');
      assert.equal(saved.typed_snapshot.body.length, original.length); assert.equal(saved.payload_utf8, '');
    });
    await t.test('native escaped record identifiers remain mapped to their exact server-owned key', async () => {
      const b = await mapped('dotted-id.md'); const id = await capture(b);
      const c = await backend.claim({ operation_id: id, attempt_id: 'a' });
      assert.equal(c.operation.record_id, 'reference:dotted-id.md');
      const intent = await backend.beginWrite(id, { lease_id: c.lease_id, fence: c.fence, attempt_id: 'a' });
      assert.equal((await backend.complete(id, evidence(c, intent))).status, 'synced');
    });
    console.log('Native proof retained in isolated synthetic namespace at ' + retained);
  } finally {
    const closing = db.close();
    let closeTimeout;
    await Promise.race([closing, new Promise(resolve => { closeTimeout = setTimeout(resolve, 2000); })]);
    clearTimeout(closeTimeout); clearInterval(nativeKeepalive);
  }
}
test('VPS native Surreal transaction proof', { skip: process.env.LIBRARY_SYNC_NATIVE_PROOF !== '1', timeout: 120000 }, nativeProof);
