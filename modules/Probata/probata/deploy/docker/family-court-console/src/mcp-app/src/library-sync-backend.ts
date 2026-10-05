/** Own durable library bindings, immutable exports and fenced B2 reconciliation in the shared application store.
 * Inputs: trusted server mappings, parent transaction variables and authenticated worker evidence.
 * Outputs: frozen toolkit-library-sync/1 JSON envelopes or exact payload bytes; effects: parameterized Surreal transactions.
 * Choose alongside case-library.ts: publication/citation/currency authorization stays in the parent's edit transaction.
 * Byline: Codex · GPT-6.1 Sol · 2026-10-05.
 */
import { createHash, randomUUID } from "node:crypto";
import { posix } from "node:path";
import { DateTime, Decimal, Duration, RecordId, StringRecordId, Uuid } from "surrealdb";

export const LIBRARY_SYNC_CONTRACT = "toolkit-library-sync/1";
export const LIBRARY_SYNC_API_BASE = "/api/internal/library-sync";
export const LIBRARY_SYNC_LIMITS = Object.freeze({ metadata: 2 << 20, payload: 8 << 20, original: 20 << 20 });
const UUID = /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/;
const SHA = /^[a-f0-9]{64}$/;
const ROOT = "consignatio/casevault/KnowledgeBase/legal/";
// Server mapping admission mirrors case-library.ts personalTables plus governed reference records; it does not grant edit authorization.
const MAPPED_TABLES = new Set(['reference', 'person', 'child', 'order', 'hearing', 'deadline', 'event', 'message', 'exhibit', 'factor', 'source', 'note', 'court', 'court_event', 'filing', 'draft', 'memo', 'evidence_log', 'eval', 'case_status']);

export interface SyncScope { account_scope: string; bucket: string; legal_root: string }
export interface SyncPointer { version_id: string; sha256: string; size: number; content_type: string; observed_at: string }
export interface SyncObject { bucket: string; key: string; version_id: string; size: number; latest: boolean; hidden: boolean; uploaded_at: string; sha256?: string; content_type?: string; operation_id?: string; intent_id?: string }
export interface SyncBinding extends SyncScope { id: string; provider: "b2"; key: string; record_id: string; record_version: string; pointer_revision: string; format: string; accepted_pointer: SyncPointer | null; original_pointer: SyncPointer | null; record_export_key?: string; record_export_pointer: SyncPointer | null }
export interface SyncOperation { contract_version: string; operation_id: string; binding_id: string; record_id: string; record_version: string; revision_ref: string; bucket: string; key: string; payload_sha256: string; payload_size: number; content_type: string; codec_version: string; base_pointer_revision: string; base_b2_pointer: SyncPointer | null }
export interface SyncClaim { operation: SyncOperation; lease_id: string; fence: number; lease_expires_at: string; status: string; write_intent_id: string }
export interface SyncOutcome { status: string; binding_id?: string; proposal_id?: string; operation_id?: string; error_code?: string }
export interface SyncObservation { contract_version: string; observation_id: string; binding_id: string; object: SyncObject; raw_ref?: unknown; extraction_ref?: unknown; status: string; error_code?: string }
export interface SyncCompletion { intent_id: string; lease_id: string; fence: number; base_pointer_revision: string; written_pointer: SyncPointer | null; observed_versions: SyncObject[]; coverage: string; status: string; error_code?: string }
export interface SyncDatabase { query(sql: string, params?: Record<string, unknown>): PromiseLike<unknown[]> & { responses?: () => Promise<Array<{ success: boolean; result?: unknown; error?: { message?: string } }>> } }
export interface SyncStore { db: SyncDatabase }
export interface SyncCapture { operation_id: string; binding_id: string; snapshot_variable: string; record_version_variable: string; revision_ref_variable: string; expected_pointer_revision: string }

/** Hash complete bytes with raw SHA-256; inputs: bytes/text; outputs: hex; effects: none; use for payload identity only. */
function hash(value: string | Uint8Array): string { return createHash("sha256").update(value).digest("hex"); }
/** Validate bounded identifiers without normalization; inputs: string; outputs: boolean; effects: none; use for exact provider coordinates. */
function safeString(value: unknown, max = 2048): value is string { return typeof value === "string" && value.length > 0 && Buffer.byteLength(value, 'utf8') <= max && !/[\r\n\x00]/.test(value); }
/** Reject malformed route identities; inputs: UUID; outputs: same UUID; effects: none; choose before database lookup. */
function uuid(value: string): string { if (!UUID.test(value)) throw new Error("invalid_operation_id"); return value; }
/** Restrict all mappings and objects to the owner's permanent legal root; inputs: configured scope and exact key; outputs: none; effects: none. */
function admit(scope: SyncScope, bucket: string, key: string): void {
  if (!safeString(scope.account_scope, 200) || scope.bucket !== "salem-data" || scope.legal_root !== ROOT || bucket !== scope.bucket
    || !safeString(key, 1024) || key.startsWith("/") || key.includes("\\") || posix.normalize(key) !== key || key.includes("//") || key.endsWith("/")
    || !["case-law/", "benchbooks/", "reference-data/"].some(child => key.startsWith(ROOT + child) && key.length > (ROOT + child).length)) throw new Error("outside_b2_legal_scope");
}
/** Reproduce Go encoding/json's escaped string-array digest exactly; inputs: configured account/bucket/original key; outputs: stable binding ID; effects: none.
 * Choose for original identity, never a mutable latest-version identity.
 */
export function libraryFileBindingId(scope: SyncScope, originalKey: string): string {
  admit(scope, scope.bucket, originalKey);
  const json = JSON.stringify(["b2", scope.account_scope, scope.bucket, originalKey]).replace(/[<>&\u2028\u2029]/g, c => "\\u" + c.charCodeAt(0).toString(16).padStart(4, "0"));
  return "library_file:" + hash(json);
}
/** Derive the Go observation identity from exact binding and retained provider version; inputs: stable binding/version; outputs: SHA; effects: none.
 * Choose for idempotency and collision checks; neither latest flags nor content changes create a new identity for the same version.
 */
export function librarySyncObservationId(bindingId: string, versionId: string): string {
  bindingKey(bindingId); if (!safeString(versionId) || versionId === 'null') throw new Error('invalid_version');
  return hash(JSON.stringify([bindingId, versionId]).replace(/[<>&\u2028\u2029]/g, c => '\\u' + c.charCodeAt(0).toString(16).padStart(4, '0')));
}
/** Validate a stable mapping identity; inputs: ID; outputs: digest; effects: none; choose before binding lookup. */
function bindingKey(id: string): string { const key = id.replace(/^library_file:/, ""); if (!id.startsWith("library_file:") || !SHA.test(key)) throw new Error("invalid_binding_id"); return key; }
/** Validate a complete exact-version byte pointer; inputs: pointer; outputs: none; effects: none; choose over ETags/latest aliases. */
function pointer(p: SyncPointer): void {
  if (!p || !safeString(p.version_id) || p.version_id === "null" || !SHA.test(p.sha256) || !Number.isSafeInteger(p.size) || p.size < 0 || p.size > LIBRARY_SYNC_LIMITS.original || !safeString(p.content_type, 200) || !Number.isFinite(Date.parse(p.observed_at))) throw new Error("invalid_pinned_pointer");
}
/** Surface actual transaction failures instead of SDK NotExecuted wrappers; inputs: SQL and bindings; outputs: statement values; effects: one DB query.
 * Choose for every sync transaction; errors never include payloads or DB error text containing private values.
 */
async function query(store: SyncStore, sql: string, params: Record<string, unknown> = {}): Promise<unknown[]> {
  const pending = store.db.query(sql, params);
  if (!pending.responses) return await pending;
  const responses = await pending.responses();
  const failed = responses.filter(r => !r.success);
  if (failed.length) {
    const actual = failed.find(r => !/not executed|failed transaction/i.test(r.error?.message ?? "")) ?? failed[0];
    const code = /sync:[a-z_]+/.exec(actual.error?.message ?? "")?.[0] ?? "sync:transaction_failed";
    throw new Error(code);
  }
  return responses.map(r => r.result);
}
/** Extract a tagged SQL return without leaking bookkeeping; inputs: results; outputs: typed value; effects: none; choose for transaction results. */
function result<T>(rows: unknown[]): T {
  for (const value of [...rows].reverse()) if (value && typeof value === "object" && "sync_result" in value) return (value as { sync_result: T }).sync_result;
  throw new Error("sync:malformed_result");
}
/** Define additive private bookkeeping and physical-coordinate uniqueness; inputs: shared store; outputs: none; effects: idempotent DDL.
 * Choose during parent migration; generic record-authenticated MCP users receive no table permissions.
 */
export async function migrateLibrarySync(store: SyncStore): Promise<void> {
  for (const table of ["library_file", "library_sync_target", "library_sync_outbox", "library_sync_observation", "library_sync_conflict"])
    await query(store, `DEFINE TABLE IF NOT EXISTS ${table} SCHEMALESS PERMISSIONS NONE;`);
  await query(store, `DEFINE INDEX IF NOT EXISTS library_original_unique ON library_file FIELDS account_scope, bucket, key UNIQUE;
    DEFINE INDEX IF NOT EXISTS library_record_unique ON library_file FIELDS record_id UNIQUE;
    DEFINE INDEX IF NOT EXISTS library_destination_unique ON library_sync_target FIELDS account_scope, bucket, key UNIQUE;`);
  await query(store, `DEFINE FUNCTION IF NOT EXISTS fn::library_sync_native_types($value: any, $path: string) -> object {
    LET $native = IF $value = NONE { 'none' } ELSE IF type::is_int($value) { 'int' } ELSE IF type::is_float($value) { 'float' }
      ELSE IF type::is_decimal($value) { 'decimal' } ELSE IF type::is_datetime($value) { 'datetime' }
      ELSE IF type::is_record($value) { 'record' } ELSE IF type::is_duration($value) { 'duration' }
      ELSE IF type::is_uuid($value) { 'uuid' } ELSE IF type::is_bytes($value) { 'bytes' } ELSE { '' };
    IF $native != '' { RETURN object::from_entries([[$path, $native]]); };
    IF type::is_object($value) {
      RETURN object::keys($value).fold({}, |$out, $key| object::extend($out,
        fn::library_sync_native_types($value[$key], $path + '/' + string::replace(string::replace($key, '~', '~0'), '/', '~1'))));
    } ELSE IF type::is_array($value) {
      RETURN array::range(0, array::len($value)).fold({}, |$out, $i| object::extend($out, fn::library_sync_native_types($value[$i], $path + '/' + <string> $i)));
    };
    RETURN {};
  } COMMENT 'Preserve native scalar type annotations from caller-supplied values. Inputs: native value/JSON pointer; outputs: annotation object; effects: none, no database reads. Choose over guessing int/float from SDK numbers.' PERMISSIONS FULL;`);
}

/** Create a server-owned original/export mapping without changing an existing baseline; inputs: trusted mapping and optional exact original pointer.
 * Outputs: binding; effects: atomic mapping/physical-target reservation. Choose during governed acquisition, never from a client record patch.
 * A second writable row requires its own original key and export destination; read-only aliases remain parent-owned.
 */
export async function bindLibraryFile(store: SyncStore, scope: SyncScope, input: { key: string; record_id: string; format: string; record_export_key?: string; original_pointer?: SyncPointer | null }): Promise<SyncBinding> {
  const id = libraryFileBindingId(scope, input.key);
  if (!/^[a-z_]+:[^\s:]{1,200}$/.test(input.record_id) || !MAPPED_TABLES.has(input.record_id.split(':')[0])) throw new Error("invalid_mapped_record");
  if (!safeString(input.format, 80)) throw new Error("invalid_format");
  if (input.original_pointer) pointer(input.original_pointer);
  if (input.format === "markdown" && posix.extname(input.key) !== ".md") throw new Error("markdown_destination_required");
  if (input.format === "markdown" && input.record_export_key) throw new Error("markdown_companion_not_supported");
  if (input.record_export_key) { admit(scope, scope.bucket, input.record_export_key); if (posix.extname(input.record_export_key) !== ".json" || input.key === input.record_export_key) throw new Error("explicit_companion_required"); }
  if (input.format !== "markdown" && !input.record_export_key) throw new Error("explicit_companion_required");
  const destination = input.format === "markdown" ? input.key : input.record_export_key!;
  return result(await query(store, `BEGIN TRANSACTION;
    LET $prior = (SELECT * FROM ONLY type::record('library_file', $binding));
    IF $prior != NONE { THROW 'sync:binding_exists'; };
    LET $record = (SELECT * OMIT embedding FROM ONLY type::record($table, $record_key));
    LET $rv = IF $record = NONE { 'absent' } ELSE { 'sha256:' + crypto::sha256(<string> $record) };
    CREATE type::record('library_file', $binding) CONTENT {
      provider: 'b2', account_scope: $scope.account_scope, bucket: $scope.bucket, legal_root: $scope.legal_root, key: $key,
      record_id: $record_id, record_version: $rv, pointer_revision: $pointer_revision, format: $format,
      original_pointer: $original, accepted_pointer: $original, record_export_key: $export_key, record_export_pointer: NULL
    };
    FOR $physical_key IN array::distinct([$key, $destination]) {
      CREATE library_sync_target CONTENT { account_scope: $scope.account_scope, bucket: $scope.bucket, key: $physical_key,
        binding_id: $binding_id, active_operation: NULL, lease_id: '', fence: 0, lease_expires_at: time::now(), write_intent_id: '' };
    };
    RETURN { sync_result: { id: $binding_id, provider: 'b2', account_scope: $scope.account_scope, bucket: $scope.bucket,
      legal_root: $scope.legal_root, key: $key, record_id: $record_id, record_version: $rv, pointer_revision: $pointer_revision,
      format: $format, original_pointer: $original, accepted_pointer: $original, record_export_key: $export_key, record_export_pointer: NULL } };
    COMMIT TRANSACTION;`, { binding: bindingKey(id), binding_id: id, scope, key: input.key, record_id: input.record_id, table: input.record_id.split(":")[0], record_key: input.record_id.split(":")[1], format: input.format, export_key: input.record_export_key ?? "", destination, original: input.original_pointer ?? null, pointer_revision: randomUUID() }));
}

/** Build snapshot capture inside the parent's already authorized edit/publication transaction; inputs: trusted SQL variable names and exact base revision.
 * Outputs: SQL fragment and parameters; effects: none until appended before COMMIT, then full native snapshot/outbox capture and binding record-version advance.
 * Choose after parent citation/currency/version gates and revision creation; never call after commit or reread a live record to encode it.
 * Parent materializes id explicitly with object::extend(SELECT *, {id: $rid}) after same-transaction writes, includes private/unknown fields,
 * and supplies its non-embedding record version with that same id and a retained revision reference; native read-own-write projections can omit implicit id.
 */
export function buildLibrarySyncCapture(input: SyncCapture): { sql: string; params: Record<string, unknown> } {
  uuid(input.operation_id); bindingKey(input.binding_id);
  const vars = [input.snapshot_variable, input.record_version_variable, input.revision_ref_variable];
  if (!vars.every(v => /^\$[a-z][a-z0-9_]*$/.test(v)) || !safeString(input.expected_pointer_revision, 200)) throw new Error("invalid_capture_variables");
  const [snapshot, version, revision] = vars;
  return { params: { sync_operation_id: input.operation_id, sync_binding_key: bindingKey(input.binding_id), sync_binding_id: input.binding_id, sync_base_revision: input.expected_pointer_revision }, sql: `
    LET $sync_b = (SELECT * FROM ONLY type::record('library_file', $sync_binding_key));
    IF $sync_b = NONE OR $sync_b.pointer_revision != $sync_base_revision { THROW 'sync:capture_pointer_conflict'; };
    IF !type::is_object(${snapshot}) OR record::tb(${snapshot}.id) + ':' + <string> record::id(${snapshot}.id) != $sync_b.record_id { THROW 'sync:capture_identity_mismatch'; };
    LET $sync_rid = type::record(string::split($sync_b.record_id, ':')[0], string::split($sync_b.record_id, ':')[1]);
    LET $sync_current = object::extend((SELECT * FROM ONLY $sync_rid), { id: $sync_rid });
    IF $sync_current != ${snapshot} OR ${version} != 'sha256:' + crypto::sha256(<string> object::remove(${snapshot}, ['embedding'])) { THROW 'sync:capture_snapshot_mismatch'; };
    LET $sync_key = IF $sync_b.format = 'markdown' { $sync_b.key } ELSE { $sync_b.record_export_key };
    LET $sync_base = IF $sync_b.format = 'markdown' { $sync_b.accepted_pointer } ELSE { $sync_b.record_export_pointer };
    CREATE type::record('library_sync_outbox', $sync_operation_id) CONTENT {
      contract_version: '${LIBRARY_SYNC_CONTRACT}', operation_id: $sync_operation_id, binding_id: $sync_binding_id,
      record_id: $sync_b.record_id, record_version: ${version}, revision_ref: <string> ${revision},
      account_scope: $sync_b.account_scope, bucket: $sync_b.bucket, key: $sync_key,
      codec_version: IF $sync_b.format = 'markdown' { 'markdown-utf8/1' } ELSE { 'surreal-record-json/1' },
      content_type: IF $sync_b.format = 'markdown' { 'text/markdown' } ELSE { 'application/json' },
      base_pointer_revision: $sync_b.pointer_revision, base_b2_pointer: $sync_base,
      typed_snapshot: ${snapshot}, snapshot_hash: crypto::sha256(<string> ${snapshot}),
      status: 'pending_encoding', payload_sha256: '', payload_size: 0, payload_utf8: '',
      lease_id: '', fence: 0, attempt_id: '', write_intent_id: '', created_at: time::now()
    };
    UPDATE $sync_b.id SET record_version = ${version};
  ` };
}

/** Encode every snapshot value without normalizing native scalars or omitting unknown/private fields; inputs: SDK value and JSON pointer.
 * Outputs: JSON value plus scalar type annotations; effects: none; choose only for retained snapshots, rejecting unsupported native classes.
 */
function typedJSON(value: unknown, path: string, types: Record<string, string>, stack: Set<object>): unknown {
  if (value === undefined) { types[path] = "none"; return null; }
  if (value === null || typeof value === "string" || typeof value === "boolean") return value;
  if (typeof value === "number") { if (!Number.isFinite(value) || (Number.isInteger(value) && !Number.isSafeInteger(value))) throw new Error("unsupported_native_type"); if (Object.is(value, -0)) { types[path] = "float"; return "-0"; } return value; }
  if (typeof value === "bigint") { types[path] = "int"; return value.toString(); }
  if (value instanceof Date || value instanceof DateTime) { types[path] = "datetime"; return value instanceof Date ? value.toISOString() : value.toString(); }
  if (value instanceof RecordId || value instanceof StringRecordId) { types[path] = "record"; return value.toString(); }
  if (value instanceof Decimal) { types[path] = "decimal"; return value.toString(); }
  if (value instanceof Duration) { types[path] = "duration"; return value.toString(); }
  if (value instanceof Uuid) { types[path] = "uuid"; return value.toString(); }
  if (value instanceof Uint8Array) { types[path] = "bytes"; return Buffer.from(value).toString("base64"); }
  if (typeof value !== "object" || stack.has(value)) throw new Error("unsupported_native_type");
  stack.add(value);
  try {
    if (Array.isArray(value)) return value.map((v, i) => typedJSON(v, `${path}/${i}`, types, stack));
    if (Object.getPrototypeOf(value) !== Object.prototype && Object.getPrototypeOf(value) !== null) throw new Error("unsupported_native_type");
    const out: Record<string, unknown> = Object.create(null);
    for (const key of Object.keys(value).sort()) out[key] = typedJSON((value as Record<string, unknown>)[key], path + "/" + key.replace(/~/g, "~0").replace(/\//g, "~1"), types, stack);
    return out;
  } finally { stack.delete(value); }
}
/** Seal the saved native snapshot once with a snapshot-hash CAS before any claim; inputs: operation ID; outputs: pending/blocked outcome.
 * Effects: reads only outbox, atomically stores exact UTF-8 bytes/hash/size or retains a visible blocked snapshot. Choose before worker scheduling.
 */
export async function sealLibrarySyncOperation(store: SyncStore, operationId: string): Promise<SyncOutcome> {
  uuid(operationId);
  const saved = result<Record<string, any>>(await query(store, "RETURN { sync_result: (SELECT * FROM ONLY type::record('library_sync_outbox', $op)) };", { op: operationId }));
  if (!saved) throw new Error("sync:operation_missing");
  if (saved.status !== "pending_encoding") return { status: saved.status, operation_id: operationId, binding_id: saved.binding_id };
  let payload = "", code = "";
  try {
    if (saved.codec_version === "markdown-utf8/1") {
      if (posix.extname(saved.key) !== ".md" || saved.content_type !== "text/markdown" || typeof saved.typed_snapshot?.body !== "string") throw new Error("markdown_body_required");
      payload = saved.typed_snapshot.body;
    } else if (saved.codec_version === "surreal-record-json/1") {
      if (posix.extname(saved.key) !== ".json" || saved.content_type !== "application/json") throw new Error("explicit_companion_required");
      const types: Record<string, string> = Object.create(null);
      const record = typedJSON(saved.typed_snapshot, "/record", types, new Set());
      if (!record || Array.isArray(record) || typeof record !== "object") throw new Error("record_snapshot_required");
      let nativeTypes: Record<string, string>;
      try {
        nativeTypes = result<Record<string, string>>(await query(store, `LET $o = (SELECT * FROM ONLY type::record('library_sync_outbox', $op));
          IF $o = NONE OR $o.snapshot_hash != $snapshot_hash { THROW 'sync:snapshot_changed'; };
          RETURN { sync_result: fn::library_sync_native_types($o.typed_snapshot, '/record') };`, { op: operationId, snapshot_hash: saved.snapshot_hash }));
      } catch { throw new Error('native_annotation_failed'); }
      Object.assign(types, nativeTypes);
      payload = JSON.stringify({ codec_version: saved.codec_version, record_id: saved.record_id, record_version: saved.record_version, record, types });
    } else throw new Error("unsupported_codec");
    if (!payload || !Buffer.from(payload, "utf8").equals(Buffer.from(Buffer.from(payload, "utf8").toString("utf8"), "utf8")) || /[\uD800-\uDBFF](?![\uDC00-\uDFFF])|(?<![\uD800-\uDBFF])[\uDC00-\uDFFF]/u.test(payload)) throw new Error("invalid_utf8_payload");
    if (Buffer.byteLength(payload, "utf8") > LIBRARY_SYNC_LIMITS.payload) throw new Error("payload_budget_exceeded");
  } catch (error) { code = error instanceof Error && /^[a-z_]+$/.test(error.message) ? error.message : "unsupported_native_type"; }
  return result(await query(store, `BEGIN TRANSACTION;
    LET $o = (SELECT * FROM ONLY type::record('library_sync_outbox', $op));
    IF $o = NONE OR $o.snapshot_hash != $snapshot_hash OR crypto::sha256(<string> $o.typed_snapshot) != $snapshot_hash { THROW 'sync:snapshot_changed'; };
    IF $o.status = 'pending_encoding' {
      UPDATE $o.id SET status = $status, payload_utf8 = $payload, payload_sha256 = $payload_hash, payload_size = $payload_size,
        error_code = $code, sealed_at = time::now();
    };
    LET $sealed = (SELECT * FROM ONLY $o.id);
    RETURN { sync_result: { status: $sealed.status, operation_id: $op, binding_id: $o.binding_id, error_code: $sealed.error_code } };
    COMMIT TRANSACTION;`, { op: operationId, snapshot_hash: saved.snapshot_hash, status: code ? "blocked" : "pending", payload: code ? "" : payload, payload_hash: code ? "" : hash(payload), payload_size: code ? 0 : Buffer.byteLength(payload), code }));
}

const LOAD = `LET $o = (SELECT * FROM ONLY type::record('library_sync_outbox', $op));
  IF $o = NONE { THROW 'sync:operation_missing'; };
  LET $b = (SELECT * FROM ONLY type::record('library_file', string::split($o.binding_id, ':')[1]));
  IF $b = NONE OR $b.account_scope != $scope.account_scope OR $b.bucket != $scope.bucket OR $b.legal_root != $scope.legal_root
    OR $o.account_scope != $b.account_scope OR $o.bucket != $b.bucket OR $o.record_id != $b.record_id
    OR $o.key != (IF $b.format = 'markdown' { $b.key } ELSE { $b.record_export_key }) { THROW 'sync:mapping_mismatch'; };
  LET $t = (SELECT * FROM ONLY library_sync_target WHERE account_scope = $o.account_scope AND bucket = $o.bucket AND key = $o.key);
  IF $t = NONE OR $t.binding_id != $o.binding_id { THROW 'sync:physical_target_missing'; };`;
const LEASE = `IF $o.lease_id != $lease OR $o.fence != $fence OR $t.active_operation != $op OR $t.lease_id != $lease OR $t.fence != $fence
  OR $t.lease_expires_at <= time::now() { THROW 'sync:stale_lease'; };`;
const DESCRIPTOR = `{ contract_version: $o.contract_version, operation_id: $o.operation_id, binding_id: $o.binding_id, record_id: $o.record_id,
  record_version: $o.record_version, revision_ref: $o.revision_ref, bucket: $o.bucket, key: $o.key, payload_sha256: $o.payload_sha256,
  payload_size: $o.payload_size, content_type: $o.content_type, codec_version: $o.codec_version,
  base_pointer_revision: $o.base_pointer_revision, base_b2_pointer: $o.base_b2_pointer }`;

/** Create application-owned transaction methods and an authenticated bare-JSON worker dispatcher; inputs: store, permanent scope and parent auth callback.
 * Outputs: backend methods; effects: none until invoked. Choose for private routes, never register these methods as generic MCP write tools.
 * Worker authorization must validate the dedicated mounted token; this module never reads network credentials or grants end-user publication authority.
 */
export function createLibrarySyncBackend(store: SyncStore, scope: SyncScope, authorizeWorker: (request: Request) => boolean | Promise<boolean> = () => false) {
  admit(scope, scope.bucket, ROOT + "reference-data/scope-check.json");
  if (typeof authorizeWorker !== "function") throw new Error("worker_authorization_required");
  const backend = {
    /** Lease one sealed operation's physical target with monotonic fencing; inputs: operation/attempt; outputs: frozen Claim; effects: atomic lease.
     * Choose before payload/write; an expired prior intent remains attached and must be reconciled, including across newer operations.
     */
    async claim(input: { operation_id: string; attempt_id: string }): Promise<SyncClaim> {
      uuid(input.operation_id); if (!safeString(input.attempt_id, 200)) throw new Error("invalid_attempt");
      return result(await query(store, `BEGIN TRANSACTION; ${LOAD}
        IF $o.status = 'pending_encoding' OR $o.status = 'blocked' OR $o.status = 'conflicted' { THROW 'sync:operation_unclaimable'; };
        IF $o.status = 'synced' { RETURN { sync_result: { operation: ${DESCRIPTOR}, lease_id: $o.lease_id, fence: $o.fence,
          lease_expires_at: <string> $o.lease_expires_at, status: 'synced', write_intent_id: $o.write_intent_id } }; };
        IF $t.active_operation != NULL AND $t.active_operation != $op {
          IF $t.write_intent_id != '' OR $t.lease_expires_at > time::now() { THROW 'sync:physical_target_busy'; };
        };
        IF $t.active_operation = $op AND $t.lease_expires_at > time::now() {
          IF $o.attempt_id != $attempt { THROW 'sync:physical_target_busy'; };
        } ELSE {
          LET $next_fence = $t.fence + 1;
          UPDATE $t.id SET active_operation = $op, lease_id = $new_lease, fence = $next_fence, lease_expires_at = time::now() + 90s;
          UPDATE $o.id SET lease_id = $new_lease, fence = $next_fence, lease_expires_at = time::now() + 90s, attempt_id = $attempt, status = 'pending';
        };
        LET $o = (SELECT * FROM ONLY $o.id);
        RETURN { sync_result: { operation: ${DESCRIPTOR}, lease_id: $o.lease_id, fence: $o.fence,
          lease_expires_at: <string> $o.lease_expires_at, status: $o.status, write_intent_id: $o.write_intent_id } };
        COMMIT TRANSACTION;`, { op: input.operation_id, attempt: input.attempt_id, new_lease: randomUUID(), scope }));
    },
    /** Read only sealed immutable bytes under an active lease; inputs: operation/lease; outputs: Buffer; effects: metadata read.
     * Choose for the worker payload route; no live record lookup occurs and hash/size are checked before returning bytes.
     */
    async payload(operationId: string, lease: string): Promise<Buffer> {
      uuid(operationId); if (!safeString(lease, 200)) throw new Error("invalid_lease");
      const saved = result<{ payload_utf8: string; payload_sha256: string; payload_size: number }>(await query(store, `BEGIN TRANSACTION; ${LOAD}
        IF $o.lease_id != $lease OR $t.lease_id != $lease OR $t.active_operation != $op OR $t.lease_expires_at <= time::now()
          OR $o.status NOT IN ['pending', 'write_unknown', 'retry_wait'] { THROW 'sync:stale_lease'; };
        RETURN { sync_result: { payload_utf8: $o.payload_utf8, payload_sha256: $o.payload_sha256, payload_size: $o.payload_size } };
        COMMIT TRANSACTION;`, { op: operationId, lease, scope }));
      const bytes = Buffer.from(saved.payload_utf8, "utf8");
      if (bytes.length <= 0 || bytes.length > LIBRARY_SYNC_LIMITS.payload || bytes.length !== saved.payload_size || hash(bytes) !== saved.payload_sha256) throw new Error("payload_integrity_failed");
      return bytes;
    },
    /** Grant exactly one durable PUT permission, surviving retries and lease expiry; inputs: operation/lease/fence/attempt; outputs: Intent.
     * Effects: atomic intent creation; choose before PUT, repeated/lost replies always require reconciliation of the saved intent.
     */
    async beginWrite(operationId: string, input: { lease_id: string; fence: number; attempt_id: string }): Promise<{ may_write: boolean; intent_id: string }> {
      uuid(operationId); validateLease(input); if (!safeString(input.attempt_id, 200)) throw new Error("invalid_attempt");
      return result(await query(store, `BEGIN TRANSACTION; ${LOAD} ${LEASE}
        IF $o.attempt_id != $attempt OR $o.status != 'pending' { THROW 'sync:attempt_mismatch'; };
        IF $o.write_intent_id != '' { RETURN { sync_result: { may_write: false, intent_id: $o.write_intent_id } }; };
        IF $t.write_intent_id != '' { THROW 'sync:unreconciled_intent'; };
        LET $r = (SELECT * OMIT embedding FROM ONLY type::record(string::split($o.record_id, ':')[0], string::split($o.record_id, ':')[1]));
        LET $rv = IF $r = NONE { 'absent' } ELSE { 'sha256:' + crypto::sha256(<string> $r) };
        LET $base = IF $b.format = 'markdown' { $b.accepted_pointer } ELSE { $b.record_export_pointer };
        IF $rv != $o.record_version OR $b.record_version != $o.record_version OR $b.pointer_revision != $o.base_pointer_revision OR $base != $o.base_b2_pointer { THROW 'sync:write_base_conflict'; };
        UPDATE $o.id SET write_intent_id = $intent, write_attempt_id = $attempt, intent_at = time::now();
        UPDATE $t.id SET write_intent_id = $intent;
        RETURN { sync_result: { may_write: true, intent_id: $intent } };
        COMMIT TRANSACTION;`, { op: operationId, lease: input.lease_id, fence: input.fence, attempt: input.attempt_id, intent: randomUUID(), scope }));
    },
    /** Retain every external version/hide as a citation-required proposal or blocked observation; inputs: exact worker observation; outputs: Outcome.
     * Effects: atomic observation/proposal retention only; choose for incoming files, never clear currency, auto-publish or advance accepted pointers.
     * Incoming content hydration is not implemented: proposals retain evidence references with proposed_record NULL and citations required.
     * A bound personal record keeps its mapping; later hydration must use the existing personal revision/CAS path instead of legal publication.
     * Unbound reference identities are provisional metadata reservations, not a verified classification or a populated shared-library record.
     */
    async observe(input: SyncObservation): Promise<SyncOutcome> {
      if (input.contract_version !== LIBRARY_SYNC_CONTRACT || !SHA.test(input.observation_id)) throw new Error("invalid_observation");
      bindingKey(input.binding_id); admit(scope, input.object.bucket, input.object.key);
      const o = input.object;
      if (!safeString(o.version_id) || o.version_id === "null" || !Number.isSafeInteger(o.size) || o.size < 0 || typeof o.hidden !== "boolean" || typeof o.latest !== "boolean" || !Number.isFinite(Date.parse(o.uploaded_at))) throw new Error("invalid_observation_object");
      if (input.binding_id !== libraryFileBindingId(scope, o.key) || input.observation_id !== librarySyncObservationId(input.binding_id, o.version_id)) throw new Error('observation_identity_mismatch');
      const raw = input.raw_ref as { uri?: string; version_id?: string; sha256?: string; bytes?: number } | undefined;
      const plain = ['text/markdown', 'text/plain', 'application/json'].includes((o.content_type ?? '').split(';')[0]);
      const complete = !o.hidden && o.size <= LIBRARY_SYNC_LIMITS.original && !!o.sha256 && SHA.test(o.sha256) && safeString(o.content_type, 200)
        && raw && safeString(raw.uri, 4096) && raw.uri.startsWith('b2://') && safeString(raw.version_id) && raw.version_id !== 'null'
        && raw.sha256 === 'sha256:' + o.sha256 && raw.bytes === o.size && (plain || !!input.extraction_ref) && input.status === "citation_required";
      const status = o.hidden ? "hidden" : complete ? "citation_required" : "blocked";
      const code = o.size > LIBRARY_SYNC_LIMITS.original ? "original_budget_exceeded" : safeCode(input.error_code ?? (complete || o.hidden ? "" : "incomplete_observation"));
      return result(await query(store, `BEGIN TRANSACTION;
        LET $prior = (SELECT * FROM ONLY type::record('library_sync_observation', $observation_id));
        IF $prior != NONE {
          IF $prior.observation != $observation { THROW 'sync:observation_identity_conflict'; };
          RETURN { sync_result: $prior.outcome };
        };
        LET $physical = (SELECT * FROM ONLY library_sync_target WHERE account_scope = $scope.account_scope AND bucket = $object.bucket AND key = $object.key);
        IF $physical = NONE {
          LET $p = IF $status = 'citation_required' { { version_id: $object.version_id, sha256: $object.sha256, size: $object.size,
            content_type: $object.content_type, observed_at: $object.uploaded_at } } ELSE { NULL };
          CREATE type::record('library_file', $binding) CONTENT { provider: 'b2', account_scope: $scope.account_scope, bucket: $scope.bucket, legal_root: $scope.legal_root,
            key: $object.key, record_id: $new_record_id, record_version: 'absent', pointer_revision: $new_revision, format: $new_format,
            original_pointer: $p, accepted_pointer: NULL, record_export_key: $new_export_key, record_export_pointer: NULL };
          FOR $physical_key IN array::distinct([$object.key, $new_destination]) {
            CREATE library_sync_target CONTENT { account_scope: $scope.account_scope, bucket: $scope.bucket, key: $physical_key,
              binding_id: $binding_id, active_operation: NULL, lease_id: '', fence: 0, lease_expires_at: time::now(), write_intent_id: '' };
          };
        };
        LET $canonical_id = IF $physical = NONE { $binding_id } ELSE { $physical.binding_id };
        LET $b = (SELECT * FROM ONLY type::record('library_file', string::split($canonical_id, ':')[1]));
        IF $b = NONE OR $b.account_scope != $scope.account_scope OR $b.bucket != $object.bucket OR ($b.key != $object.key AND $b.record_export_key != $object.key) { THROW 'sync:observation_mapping_mismatch'; };
        LET $rid = type::record(string::split($b.record_id, ':')[0], string::split($b.record_id, ':')[1]);
        LET $r = (SELECT * OMIT embedding FROM ONLY $rid);
        LET $rv = IF $r = NONE { 'absent' } ELSE { 'sha256:' + crypto::sha256(<string> $r) };
        LET $outcome = { status: $status, binding_id: $canonical_id, proposal_id: IF $status = 'citation_required' { 'library_proposal:' + $proposal } ELSE { '' }, error_code: $code };
        IF $status = 'citation_required' {
          CREATE type::record('library_proposal', $proposal) CONTENT { target: $rid, expected_version: $rv,
            proposed_record: NULL, proposed_hash: '', citations: [], status: 'citation_required',
            external_observation: $observation, rationale: 'Exact external version retained; hydrate full proposal and add citations before validation', created_at: time::now() };
        };
        CREATE type::record('library_sync_observation', $observation_id) CONTENT { observation: $observation, object: $object, binding_id: $canonical_id,
          status: $status, outcome: $outcome, retained_at: time::now() };
        RETURN { sync_result: $outcome }; COMMIT TRANSACTION;`, { observation_id: input.observation_id, observation: input, object: o, binding: bindingKey(input.binding_id), binding_id: input.binding_id, scope, status, code, proposal: randomUUID(),
          new_record_id: 'reference:b2_' + bindingKey(input.binding_id), new_revision: randomUUID(), new_format: posix.extname(o.key) === '.md' ? 'markdown' : 'record',
          new_export_key: posix.extname(o.key) === '.md' ? '' : o.key + '.record.json', new_destination: posix.extname(o.key) === '.md' ? o.key : o.key + '.record.json' }));
    },
    /** Report only durable processed/blocked/hide observations as seen; inputs: SHA observation ID; outputs: seen boolean; effects: read.
     * Choose before provider reads; reserved or absent observations remain unseen.
     */
    async seen(id: string): Promise<{ seen: boolean }> {
      if (!SHA.test(id)) throw new Error("invalid_observation_id");
      return result(await query(store, `LET $r = (SELECT * FROM ONLY type::record('library_sync_observation', $id));
        RETURN { sync_result: { seen: $r != NONE AND $r.status IN ['citation_required', 'blocked', 'hidden'] } };`, { id }));
    },
    /** Independently reconcile saved operation, current record, mapping, byte evidence and intent before pointer CAS; inputs: authenticated Completion.
     * Outputs: synced/conflicted/write_unknown outcome; effects: atomic pointer update or retained conflict containing app revision and all pointers.
     * Choose after exact-version GET/hash plus complete provider history; no implicit latest adoption or S3 conditional-write guarantee is assumed.
     */
    async complete(operationId: string, input: SyncCompletion): Promise<SyncOutcome> {
      uuid(operationId); validateLease(input);
      if (!safeString(input.intent_id, 200) || !safeString(input.base_pointer_revision, 200) || !Array.isArray(input.observed_versions) || input.observed_versions.length > 1000 || !["synced", "conflicted", "write_unknown", "blocked", "retry_wait"].includes(input.status)) throw new Error("invalid_completion");
      if (input.written_pointer) pointer(input.written_pointer);
      for (const o of input.observed_versions) {
        admit(scope, o.bucket, o.key);
        if (!safeString(o.version_id) || o.version_id === "null" || !Number.isSafeInteger(o.size) || o.size < 0 || !Number.isFinite(Date.parse(o.uploaded_at)) || typeof o.latest !== "boolean" || typeof o.hidden !== "boolean") throw new Error("invalid_completion_observation");
      }
      return result(await query(store, `BEGIN TRANSACTION; ${LOAD}
        IF $o.status = 'synced' {
          IF $o.completion != $completion { THROW 'sync:completion_replay_mismatch'; };
          RETURN { sync_result: { status: 'synced', binding_id: $o.binding_id, operation_id: $op } };
        };
        ${LEASE}
        IF $o.write_intent_id = '' OR $o.write_intent_id != $completion.intent_id OR $t.write_intent_id != $completion.intent_id { THROW 'sync:intent_mismatch'; };
        LET $r = (SELECT * OMIT embedding FROM ONLY type::record(string::split($o.record_id, ':')[0], string::split($o.record_id, ':')[1]));
        LET $rv = IF $r = NONE { 'absent' } ELSE { 'sha256:' + crypto::sha256(<string> $r) };
        LET $base = IF $b.format = 'markdown' { $b.accepted_pointer } ELSE { $b.record_export_pointer };
        LET $own = $completion.observed_versions[WHERE !hidden AND operation_id = $op AND intent_id = $o.write_intent_id];
        LET $own_ok = array::len($own) = 1 AND $own[0].latest = true AND $own[0].bucket = $o.bucket AND $own[0].key = $o.key
          AND $own[0].sha256 = $o.payload_sha256 AND $own[0].size = $o.payload_size AND $own[0].content_type = $o.content_type
          AND $completion.written_pointer != NULL AND $completion.written_pointer.version_id = $own[0].version_id
          AND $completion.written_pointer.sha256 = $o.payload_sha256 AND $completion.written_pointer.size = $o.payload_size
          AND $completion.written_pointer.content_type = $o.content_type;
        LET $coords_ok = array::all($completion.observed_versions.map(|$v| $v.bucket = $o.bucket AND $v.key = $o.key));
        LET $versions = $completion.observed_versions.map(|$v| $v.version_id);
        LET $unique_ok = array::len(array::distinct($versions)) = array::len($versions);
        LET $latest = $completion.observed_versions[WHERE latest];
        LET $baseline = $completion.observed_versions[WHERE version_id = $o.base_b2_pointer.version_id];
        LET $base_ok = IF $o.base_b2_pointer = NULL { true } ELSE {
          array::len($baseline) = 1 AND !$baseline[0].hidden AND $baseline[0].sha256 = $o.base_b2_pointer.sha256
          AND $baseline[0].size = $o.base_b2_pointer.size AND $baseline[0].content_type = $o.base_b2_pointer.content_type
        };
        LET $competing = $completion.observed_versions[WHERE operation_id != $op OR intent_id != $o.write_intent_id];
        LET $history_ok = IF $o.base_b2_pointer = NULL { array::len($competing) = 0 } ELSE {
          array::len($competing) = 1 AND $competing[0].version_id = $o.base_b2_pointer.version_id AND !$competing[0].latest
        };
        LET $bytes_ok = crypto::sha256($o.payload_utf8) = $o.payload_sha256 AND bytes::len(<bytes> $o.payload_utf8) = $o.payload_size;
        LET $cas_ok = $rv = $o.record_version AND $b.record_version = $o.record_version AND $b.pointer_revision = $o.base_pointer_revision
          AND $completion.base_pointer_revision = $o.base_pointer_revision AND $base = $o.base_b2_pointer;
        LET $ok = $completion.status = 'synced' AND $completion.coverage = 'complete' AND $own_ok AND $coords_ok AND $unique_ok
          AND array::len($latest) = 1 AND $base_ok AND $history_ok AND $bytes_ok AND $cas_ok;
        LET $status = IF $ok { 'synced' } ELSE IF $completion.coverage != 'complete' OR !$bytes_ok { 'write_unknown' } ELSE { 'conflicted' };
        IF $ok {
          IF $b.format = 'markdown' { UPDATE $b.id SET accepted_pointer = $completion.written_pointer, pointer_revision = $next_revision; }
          ELSE { UPDATE $b.id SET record_export_pointer = $completion.written_pointer, pointer_revision = $next_revision; };
          UPDATE $t.id SET active_operation = NULL, write_intent_id = '';
        } ELSE {
          CREATE library_sync_conflict CONTENT { operation_id: $op, binding_id: $o.binding_id, revision_ref: $o.revision_ref,
            app_record_version: $o.record_version, current_record_version: $rv, typed_snapshot: $o.typed_snapshot,
            original_pointer: $b.original_pointer, accepted_pointer: $b.accepted_pointer, record_export_pointer: $b.record_export_pointer,
            base_pointer: $o.base_b2_pointer, completion: $completion, status: $status, retained_at: time::now() };
        };
        UPDATE $o.id SET status = $status, completion = $completion, written_pointer = $completion.written_pointer, completed_at = time::now();
        RETURN { sync_result: { status: $status, binding_id: $o.binding_id, operation_id: $op, error_code: IF $ok { '' } ELSE { 'reconciliation_not_proven' } } };
        COMMIT TRANSACTION;`, { op: operationId, lease: input.lease_id, fence: input.fence, completion: input, next_revision: randomUUID(), scope }));
    },
    /** Retain safe worker failure state without clearing an uncertain write intent; inputs: active lease, failure status/code; outputs: Outcome.
     * Effects: atomic outbox failure update; choose on bounded I/O failure, never release a target after an unproven PUT.
     */
    async failure(operationId: string, input: { lease_id: string; fence: number; status: string; error_code: string }): Promise<SyncOutcome> {
      uuid(operationId); validateLease(input); if (!["blocked", "write_unknown", "retry_wait", "conflicted"].includes(input.status)) throw new Error("invalid_failure_status");
      const code = safeCode(input.error_code);
      return result(await query(store, `BEGIN TRANSACTION; ${LOAD} ${LEASE}
        IF $o.status = 'synced' { THROW 'sync:operation_terminal'; };
        LET $status = IF $o.write_intent_id != '' AND $status IN ['blocked', 'retry_wait'] { 'write_unknown' } ELSE { $status };
        UPDATE $o.id SET status = $status, error_code = $code;
        RETURN { sync_result: { status: $status, operation_id: $op, binding_id: $o.binding_id, error_code: $code } };
        COMMIT TRANSACTION;`, { op: operationId, lease: input.lease_id, fence: input.fence, status: input.status, code, scope }));
    },
    /** Resolve only an exact retained original version through its server mapping; inputs: stable binding/version; outputs: frozen Binding.
     * Effects: metadata read; choose after parent end-user authorization, never accept arbitrary bucket/key or latest aliases.
     */
    async original(id: string, versionId: string): Promise<SyncBinding> {
      const key = bindingKey(id); if (!safeString(versionId) || versionId === "null") throw new Error("exact_original_version_required");
      return result(await query(store, `LET $b = (SELECT * FROM ONLY type::record('library_file', $binding));
        IF $b = NONE OR $b.account_scope != $scope.account_scope OR $b.bucket != $scope.bucket OR $b.legal_root != $scope.legal_root { THROW 'sync:binding_missing'; };
        LET $retained = (SELECT * FROM library_sync_observation WHERE binding_id = $binding_id AND object.key = $b.key AND object.version_id = $version AND object.hidden = false AND status = 'citation_required' AND object.sha256 != NONE);
        LET $p = IF $b.original_pointer != NULL AND $b.original_pointer.version_id = $version { $b.original_pointer } ELSE IF array::len($retained) = 1 AND $retained[0].object.size <= ${LIBRARY_SYNC_LIMITS.original} {
          { version_id: $version, sha256: $retained[0].object.sha256, size: $retained[0].object.size, content_type: $retained[0].object.content_type, observed_at: $retained[0].object.uploaded_at }
        } ELSE { NULL };
        IF $p = NULL { THROW 'sync:original_version_not_retained'; };
        RETURN { sync_result: { id: $binding_id, provider: $b.provider, account_scope: $b.account_scope, bucket: $b.bucket, legal_root: $b.legal_root,
          key: $b.key, record_id: $b.record_id, record_version: $b.record_version, pointer_revision: $b.pointer_revision, format: $b.format,
          accepted_pointer: $b.accepted_pointer, original_pointer: $p, record_export_key: $b.record_export_key, record_export_pointer: $b.record_export_pointer } };`, { binding: key, binding_id: id, version: versionId, scope }));
    },
    /** Dispatch the frozen private HTTP routes with dedicated parent-provided authentication; inputs: Request; outputs: bare JSON/raw bytes.
     * Effects: bounded request read and the selected transaction; choose in server.ts before ordinary MCP routing, retaining authentication outside this module.
     */
    async dispatch(request: Request): Promise<Response> {
      if (!(await authorizeWorker(request))) return jsonResponse({ error_code: "worker_unauthorized" }, 401);
      const url = new URL(request.url);
      if (!url.pathname.startsWith(LIBRARY_SYNC_API_BASE + "/")) return jsonResponse({ error_code: "route_missing" }, 404);
      const route = url.pathname.slice(LIBRARY_SYNC_API_BASE.length);
      try {
        if (request.method === "POST" && route === "/outbox/claim") return jsonResponse(await backend.claim(await metadata(request)));
        if (request.method === "POST" && route === "/observations") return jsonResponse(await backend.observe(await metadata(request)));
        const seen = /^\/observations\/([a-f0-9]{64})\/status$/.exec(route);
        if (request.method === "GET" && seen) return jsonResponse(await backend.seen(seen[1]));
        const outbox = /^\/outbox\/([a-f0-9-]{36})\/(payload|write-intent|complete|failure)$/.exec(route);
        if (outbox) {
          if (request.method === "GET" && outbox[2] === "payload") return new Response(new Uint8Array(await backend.payload(outbox[1], request.headers.get("X-Toolkit-Sync-Lease") ?? "")), { headers: { "Content-Type": "application/octet-stream", "Cache-Control": "no-store" } });
          if (request.method === "POST" && outbox[2] === "write-intent") return jsonResponse(await backend.beginWrite(outbox[1], await metadata(request)));
          if (request.method === "POST" && outbox[2] === "complete") return jsonResponse(await backend.complete(outbox[1], await metadata(request)));
          if (request.method === "POST" && outbox[2] === "failure") return jsonResponse(await backend.failure(outbox[1], await metadata(request)));
        }
        const original = /^\/bindings\/([^/]+)\/original$/.exec(route);
        if (request.method === "GET" && original) return jsonResponse(await backend.original(decodeURIComponent(original[1]), url.searchParams.get("version_id") ?? ""));
        return jsonResponse({ error_code: "route_missing" }, 404);
      } catch (error) {
        const code = error instanceof Error && /^(sync:)?[a-z_]+$/.test(error.message) ? error.message.replace(/^sync:/, "") : "invalid_request";
        return jsonResponse({ error_code: code }, code === "metadata_budget_exceeded" ? 413 : 409);
      }
    }
  };
  return backend;
}
/** Require nonempty lease identity and positive safe fencing; inputs: lease envelope; outputs: none; effects: none; choose before transaction binding. */
function validateLease(input: { lease_id: string; fence: number }): void { if (!input || !safeString(input.lease_id, 200) || !Number.isSafeInteger(input.fence) || input.fence <= 0) throw new Error("invalid_lease"); }
/** Restrict errors to non-personal symbolic codes; inputs: code; outputs: same code; effects: none; choose over raw worker exceptions. */
function safeCode(code: string): string { if (typeof code !== "string" || (code !== "" && !/^[A-Za-z][A-Za-z0-9_]{0,100}$/.test(code))) throw new Error("invalid_error_code"); return code; }
/** Bound request bytes while streaming rather than after materialization; inputs: Request; outputs: JSON object; effects: reads at most metadata budget plus one chunk.
 * Choose for worker metadata; overflow is visible and never silently truncated or accepted.
 */
async function metadata<T>(request: Request): Promise<T> {
  if (Number(request.headers.get("content-length") ?? 0) > LIBRARY_SYNC_LIMITS.metadata) throw new Error("metadata_budget_exceeded");
  const reader = request.body?.getReader(); if (!reader) throw new Error("request_body_required");
  let length = 0; const chunks: Uint8Array[] = [];
  try { for (;;) { const next = await reader.read(); if (next.done) break; length += next.value.length; if (length > LIBRARY_SYNC_LIMITS.metadata) throw new Error("metadata_budget_exceeded"); chunks.push(next.value); } }
  finally { reader.releaseLock(); }
  const value = JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(Buffer.concat(chunks)));
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("request_object_required");
  return value as T;
}
/** Emit bounded bare JSON without SDK-native values or payload logs; inputs: contract value/status; outputs: Response; effects: none; choose for Go routes. */
function jsonResponse(value: unknown, status = 200): Response {
  const body = JSON.stringify(value);
  if (Buffer.byteLength(body) > LIBRARY_SYNC_LIMITS.metadata) return new Response('{"error_code":"metadata_budget_exceeded"}', { status: 413, headers: { "Content-Type": "application/json" } });
  return new Response(body, { status, headers: { "Content-Type": "application/json", "Cache-Control": "no-store" } });
}

export interface LibrarySyncDispatchInput { method: 'GET' | 'POST'; path: string; query: URLSearchParams; body: Record<string, unknown>; leaseId?: string }
export interface LibrarySyncDispatchOutput { body: unknown | Uint8Array; contentType?: string }

/** Adapt separately testable transaction methods to the parent's dedicated authenticated HTTP handler.
 * Inputs: backend and decoded route/query/body/lease from library-sync-http.ts after authentication and body-budget checks.
 * Outputs: bare Go contract value or exact Uint8Array payload; effects: only the selected backend method.
 * Choose for server integration; parent owns the strong mounted-token check before invoking this privileged adapter.
 */
export function createLibrarySyncDispatcher(backend: ReturnType<typeof createLibrarySyncBackend>) {
  return async (input: LibrarySyncDispatchInput): Promise<LibrarySyncDispatchOutput> => {
    const { method, path, body } = input;
    if (method === 'POST' && path === '/outbox/claim') return { body: await backend.claim(body as unknown as { operation_id: string; attempt_id: string }) };
    if (method === 'POST' && path === '/observations') return { body: await backend.observe(body as unknown as SyncObservation) };
    const seen = /^\/observations\/([a-f0-9]{64})\/status$/.exec(path);
    if (method === 'GET' && seen) return { body: await backend.seen(seen[1]) };
    const op = /^\/outbox\/([a-f0-9-]{36})\/(payload|write-intent|complete|failure)$/.exec(path);
    if (op) {
      if (method === 'GET' && op[2] === 'payload') return { body: await backend.payload(op[1], input.leaseId ?? ''), contentType: 'application/octet-stream' };
      if (method === 'POST' && op[2] === 'write-intent') return { body: await backend.beginWrite(op[1], body as unknown as { lease_id: string; fence: number; attempt_id: string }) };
      if (method === 'POST' && op[2] === 'complete') return { body: await backend.complete(op[1], body as unknown as SyncCompletion) };
      if (method === 'POST' && op[2] === 'failure') return { body: await backend.failure(op[1], body as unknown as { lease_id: string; fence: number; status: string; error_code: string }) };
    }
    const original = /^\/bindings\/(library_file:[a-f0-9]{64})\/original$/.exec(path);
    if (method === 'GET' && original) return { body: await backend.original(original[1], input.query.get('version_id') ?? '') };
    throw new Error('sync:route_missing');
  };
}
