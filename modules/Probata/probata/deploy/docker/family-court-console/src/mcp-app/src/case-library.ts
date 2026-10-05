// Byline: Codex, GPT-6, 2026-10-04. Shared personalized library revision contract.
import { randomUUID } from "node:crypto";
import { normalize, type StoreOk } from "./store.js";
import { prepareLibraryEditCapture } from "./library-sync-integration.js";

export interface LibraryCitation {
  source_id: string;
  source_version: string;
  pinpoint: string;
  claim: string;
}
export interface LibraryProposalInput {
  id: string;
  expected_version: string;
  patch: Record<string, unknown>;
  citations: LibraryCitation[];
  rationale: string;
}

/** Define additive shared library proposal, validation and revision tables.
 * Inputs: connected case store. Outputs: no result. Effects: idempotent additive schema definitions only.
 * Choose during toolkit migration; never stores case data in a client or removes existing fields.
 */
export async function migrateLibrary(store: Pick<StoreOk, "db">): Promise<void> {
  for (const table of ["library_proposal", "library_validation", "library_revision", "record_revision"])
    await store.db.query(`DEFINE TABLE IF NOT EXISTS ${table} SCHEMALESS;`);
}

/** Preserve full child identities and personal context in the private shared case store.
 * Inputs: connected case store. Outputs: no result. Effects: additive schema relaxation without deleting records or fields.
 * Choose during migration after the owner's explicit personal-context decision; initials remain available as optional display metadata.
 */
export async function migratePersonalCaseContext(store: Pick<StoreOk, "db">): Promise<void> {
  await store.db.query("DEFINE TABLE OVERWRITE child SCHEMALESS; DEFINE FIELD OVERWRITE initials ON child TYPE option<string>; DEFINE FIELD OVERWRITE age ON child TYPE option<number>;");
}

/** Save a personal case-document source without permitting an existing legal authority to be overwritten.
 * Inputs: source key and full case_document fields. Outputs: updated SDK record.
 * Effects: one atomic identity-kind check and case-source merge. Choose for personal intake documents; legal sources use proposals.
 */
export async function putPersonalCaseSource(store: StoreOk, key: string, data: Record<string, unknown>, expectedVersion?: string): Promise<Record<string, unknown>> {
  if (data.kind !== "case_document") throw new Error("Personal case sources require kind case_document");
  if (expectedVersion !== undefined) return putVersionedPersonalRecord(store, "source", key, data, expectedVersion);
  const results = await libraryTransaction(store, `BEGIN TRANSACTION;
    LET $r = (SELECT * FROM ONLY type::record('source', $key));
    IF $r != NONE AND $r.kind != 'case_document' { THROW 'Existing library sources require governed revision'; };
    UPSERT type::record('source', $key) MERGE $data RETURN AFTER;
    COMMIT TRANSACTION;`, { key, data });
  for (const value of [...results].reverse()) {
    const row = Array.isArray(value) ? value[0] : value;
    if (row && typeof row === "object" && "id" in row) return row as Record<string, unknown>;
  }
  throw new Error("Malformed personal source result");
}

/** Merge a private case edit only against its exact version, retaining both full revisions atomically.
 * Inputs: admitted personal table/key, changed fields and expected case_record version or absent.
 * Outputs: saved SDK record. Effects: record merge and retained before/after revision; conflicts write nothing.
 * Choose for shared phone/workdesk editing; source authorities and references still require validated library proposals.
 * Byline: Codex · GPT-6 · 2026-10-04.
 */
export async function putVersionedPersonalRecord(store: StoreOk, table: string, key: string, data: Record<string, unknown>, expectedVersion: string): Promise<Record<string, unknown>> {
  const personalTables = ["person", "child", "order", "hearing", "deadline", "event", "message", "exhibit", "factor", "source", "note", "court", "court_event", "filing", "draft", "memo", "evidence_log", "eval", "case_status"];
  if (!personalTables.includes(table) || !key || (expectedVersion !== "absent" && !/^sha256:[a-f0-9]{64}$/.test(expectedVersion))) throw new Error("Personal edit requires an admitted table and exact expected version");
  if ("id" in data || "embedding" in data || "library_file_id" in data) throw new Error("Record identity, embedding and file bindings are server-owned fields");
  if (table === "source" && data.kind !== "case_document") throw new Error("Personal case sources require kind case_document");
  const capture = await prepareLibraryEditCapture(store, table + ":" + key, "$sync_full", "$sync_version", "$sync_revision");
  const results = await libraryTransaction(store, `BEGIN TRANSACTION;
    LET $rid = type::record($tb, $key);
    LET $personal_prior = (SELECT * OMIT embedding FROM ONLY $rid);
    LET $version = IF $personal_prior = NONE { 'absent' } ELSE { 'sha256:' + crypto::sha256(<string> $personal_prior) };
    IF $version != $expected { THROW 'Personal record version conflict'; };
    IF $tb = 'source' AND $personal_prior != NONE AND $personal_prior.kind != 'case_document' { THROW 'Existing library sources require governed revision'; };
    UPSERT $rid MERGE $data;
    LET $personal_next = object::extend((SELECT * OMIT embedding FROM ONLY $rid), { id: $rid });
    CREATE type::record('record_revision', $revision) CONTENT { target: $rid, previous_version: $version,
      previous_record: $personal_prior, current_version: 'sha256:' + crypto::sha256(<string> $personal_next), current_record: $personal_next, created_at: time::now() };
    LET $sync_full = object::extend((SELECT * FROM ONLY $rid), { id: $rid });
    LET $sync_version = 'sha256:' + crypto::sha256(<string> object::remove($sync_full, ['embedding']));
    LET $sync_revision = type::record('record_revision', $revision);
    ${capture.sql}
    RETURN { personal_record: $personal_next };
    COMMIT TRANSACTION;`, { tb: table, key, data, expected: expectedVersion, revision: randomUUID(), ...capture.params });
  for (const value of [...results].reverse()) {
    const row = Array.isArray(value) ? value[0] : value;
    if (row && typeof row === "object" && "personal_record" in row) return (row as { personal_record: Record<string, unknown> }).personal_record;
  }
  throw new Error("Malformed versioned personal edit result");
}

/** Retain imported library material that does not yet carry claim-to-source citations.
 * Inputs: exact library target, full imported fields and original provenance. Outputs: retained proposal id/hash and citation-required state.
 * Effects: atomic draft retention only; no published source/reference changes. Choose for legacy mixed case imports so missing citations never discard personal content.
 */
export async function retainLibraryImport(store: StoreOk, id: string, patch: Record<string, unknown>): Promise<Record<string, unknown>> {
  const match = /^(source|reference):([^\s:]{1,200})$/.exec(id);
  if (!match || Buffer.byteLength(JSON.stringify(patch), "utf8") > 512000) throw new Error("Invalid or oversized imported library draft");
  const { id: _id, embedding: _embedding, ...fields } = patch;
  const key = randomUUID();
  const results = await libraryTransaction(store, `BEGIN TRANSACTION;
    LET $r = (SELECT * OMIT embedding FROM ONLY type::record($tb, $target));
    LET $version = IF $r = NONE { 'absent' } ELSE { 'sha256:' + crypto::sha256(<string> $r) };
    LET $base = IF $r = NONE { { id: type::record($tb, $target) } } ELSE { $r };
    LET $proposed = object::extend($base, $patch);
    LET $hash = 'sha256:' + crypto::sha256(<string> $proposed);
    CREATE type::record('library_proposal', $key) CONTENT { target: type::record($tb, $target),
      expected_version: $version, proposed_record: $proposed, proposed_hash: $hash, citations: [],
      rationale: 'Legacy import retained; add exact claim citations before validation',
      status: 'citation_required', created_at: time::now() };
    RETURN { proposal_id: 'library_proposal:' + $key, proposed_hash: $hash, status: 'citation_required', target: $tb + ':' + $target };
    COMMIT TRANSACTION;`, { tb: match[1], target: match[2], key, patch: fields });
  return resultObject(results);
}

/** Validate a library edit without removing or redacting personal fields.
 * Inputs: exact shared identity/version, partial patch and claim citations. Outputs: parsed table/key.
 * Effects: none. Choose before proposal creation; source identifiers and hashes do not themselves prove legal support.
 */
export function validateLibraryProposal(input: LibraryProposalInput): { table: string; key: string } {
  const match = /^(reference|source):([^\s:]{1,200})$/.exec(input.id);
  if (!match || !/^(?:absent|sha256:[a-f0-9]{64})$/.test(input.expected_version)) throw new Error("Invalid library identity or expected version");
  if (!input.patch || typeof input.patch !== "object" || Array.isArray(input.patch) || !Object.keys(input.patch).length
    || Object.keys(input.patch).some(k => ["id", "embedding", "library_file_id", "validation", "validation_status", "published_at"].includes(k)))
    throw new Error("Invalid library patch or reserved publication field");
  if (Buffer.byteLength(JSON.stringify(input.patch), "utf8") > 512000) throw new Error("Library patch exceeds 512KB budget");
  if (!Array.isArray(input.citations) || input.citations.length < 1 || input.citations.length > 64) throw new Error("One to 64 claim citations are required");
  for (const c of input.citations) {
    if (!/^source:[^\s:]{1,200}$/.test(c.source_id) || !/^(?:absent|sha256:[a-f0-9]{64})$/.test(c.source_version)
      || typeof c.pinpoint !== "string" || !c.pinpoint.trim() || c.pinpoint.length > 512
      || typeof c.claim !== "string" || !c.claim.trim() || c.claim.length > 8000) throw new Error("Invalid claim citation");
    if (c.source_version === "absent" && (input.expected_version !== "absent" || c.source_id !== input.id))
      throw new Error("Absent source versions are only valid for a new primary source citing its own verified capture");
  }
  if (typeof input.rationale !== "string" || !input.rationale.trim() || input.rationale.length > 4000) throw new Error("An edit rationale is required");
  return { table: match[1], key: match[2] };
}

/** Retain a complete proposed library revision in the shared case database.
 * Inputs: partial edit, current expected version, explicit claim-to-source citations and rationale.
 * Outputs: proposal identity/hash and pending-validation status. Effects: atomic proposal write; published record stays intact.
 * Choose for all phone, desktop, workdesk and agent library edits; full personal content remains in the proposed record.
 */
export async function libraryPropose(store: StoreOk, input: LibraryProposalInput): Promise<Record<string, unknown>> {
  const { table, key } = validateLibraryProposal(input);
  const proposal = randomUUID();
  const results = await libraryTransaction(store, `
    BEGIN TRANSACTION;
    LET $r = (SELECT * OMIT embedding FROM ONLY type::record($tb, $key));
    LET $version = IF $r = NONE { 'absent' } ELSE { 'sha256:' + crypto::sha256(<string> $r) };
    IF $version != $expected { THROW 'Library version conflict'; };
    LET $base = IF $r = NONE { { id: type::record($tb, $key) } } ELSE { $r };
    LET $proposed = object::extend($base, $patch);
    LET $hash = 'sha256:' + crypto::sha256(<string> $proposed);
    CREATE type::record('library_proposal', $proposal) CONTENT {
      target: type::record($tb, $key), expected_version: $version, proposed_record: $proposed,
      proposed_hash: $hash, citations: $citations, rationale: $rationale,
      status: 'pending_validation', created_at: time::now()
    };
    RETURN { proposal_id: 'library_proposal:' + $proposal, proposed_hash: $hash, status: 'pending_validation' };
    COMMIT TRANSACTION;
  `, { tb: table, key, expected: input.expected_version, patch: input.patch, proposal, citations: input.citations, rationale: input.rationale });
  return resultObject(results);
}

/** Publish one exact proposed revision after a server-side, source-version-bound validation receipt.
 * Inputs: shared proposal identity. Outputs: new shared record version and retained revision identity.
 * Effects: one transaction checks record/source versions, receipt and claim coverage, retains old full content, then replaces published content.
 * Choose after validation; generic client-provided flags or successful URL fetches cannot authorize publication.
 */
export async function libraryPublish(store: StoreOk, proposalId: string): Promise<Record<string, unknown>> {
  const match = /^library_proposal:([a-f0-9-]{36})$/.exec(proposalId);
  if (!match) throw new Error("Invalid library proposal identity");
  const preflight = await store.db.query("SELECT VALUE target FROM ONLY type::record('library_proposal', $key);", { key: match[1] });
  const target = preflight.at(-1);
  if (!target) throw new Error("Library proposal missing");
  const targetId = String(target);
  const capture = await prepareLibraryEditCapture(store, targetId, "$sync_full", "$sync_version", "$sync_revision");
  const results = await libraryTransaction(store, `
    BEGIN TRANSACTION;
    LET $p = (SELECT * FROM ONLY type::record('library_proposal', $key));
    IF $p = NONE { THROW 'Library proposal missing'; };
    IF <string> $p.target != $sync_publication_target { THROW 'Library proposal target changed'; };
    LET $status = $p.status;
    IF $status = 'published' { RETURN $p.publication; };
    LET $v = (SELECT * FROM ONLY type::record('library_validation', $key));
    IF $v = NONE { THROW 'Library validation pending'; };
    LET $valid = $v.status = 'VERIFIED_PRIMARY' AND $v.proposed_hash = $p.proposed_hash
      AND $v.proposal_id = $p.id AND $v.proposal_version = 'sha256:' + crypto::sha256(<string> object::remove($p, ['dispatch', 'dispatch_at']))
      AND type::is_string($v.validator_version) AND string::len($v.validator_version) > 0
      AND type::is_datetime($v.completed_at) AND $v.completed_at <= time::now()
      AND type::is_datetime($v.expires_at) AND $v.expires_at > time::now() AND $v.currency_status = 'cleared'
      AND $v.claims = $p.citations AND array::len($p.citations) > 0
      AND array::len($v.claim_checks) = array::len($p.citations);
    IF !$valid { THROW 'Library validation receipt missing, stale or mismatched'; };
    FOR $i IN 0..array::len($p.citations) {
      LET $claim = $p.citations[$i];
      LET $check = $v.claim_checks[$i];
      LET $complete = $check.status = 'VERIFIED_PRIMARY' AND $check.currency_status = 'cleared'
        AND $check.source_id = $claim.source_id AND $check.source_version = $claim.source_version
        AND $check.pinpoint = $claim.pinpoint AND $check.claim = $claim.claim
        AND type::is_string($check.snapshot_ref) AND string::len($check.snapshot_ref) > 0
        AND type::is_string($check.snapshot_version_id) AND string::len($check.snapshot_version_id) > 0
        AND type::is_string($check.snapshot_sha256) AND string::matches($check.snapshot_sha256, '^[a-f0-9]{64}$')
        AND type::is_string($check.quote_sha256) AND string::matches($check.quote_sha256, '^[a-f0-9]{64}$')
        AND type::is_string($check.primary_url) AND string::starts_with($check.primary_url, 'https://')
        AND type::is_datetime($check.evidence_time) AND $check.evidence_time <= time::now()
        AND type::is_string($check.check_version) AND string::len($check.check_version) > 0
        AND $check.currency_evidence != NONE AND $check.currency_evidence != NULL;
      IF !$complete { THROW 'Library claim validation incomplete or mismatched'; };
    };
    FOR $c IN $p.citations {
      LET $source = (SELECT * OMIT embedding FROM ONLY type::record('source', string::split($c.source_id, ':')[1]));
      LET $source_version = IF $source = NONE { 'absent' } ELSE { 'sha256:' + crypto::sha256(<string> $source) };
      LET $new_root = $source = NONE AND $c.source_id = record::tb($p.target) + ':' + <string> record::id($p.target) AND $p.expected_version = 'absent';
      IF $source = NONE AND !$new_root { THROW 'Library citation source missing'; };
      IF $source_version != $c.source_version { THROW 'Library citation source version changed'; };
    };
    LET $r = (SELECT * OMIT embedding FROM ONLY $p.target);
    LET $version = IF $r = NONE { 'absent' } ELSE { 'sha256:' + crypto::sha256(<string> $r) };
    IF $version != $p.expected_version { THROW 'Library version conflict'; };
    LET $proposed_hash = 'sha256:' + crypto::sha256(<string> $p.proposed_record);
    IF $proposed_hash != $p.proposed_hash { THROW 'Library proposal changed after validation'; };
    CREATE type::record('library_revision', $key) CONTENT {
      target: $p.target, previous_record: $r, previous_version: $version,
      proposal: $p.id, validation: $v.id, changed_at: time::now()
    };
    UPSERT $p.target CONTENT $p.proposed_record;
    LET $new = (SELECT * OMIT embedding FROM ONLY $p.target);
    LET $publication = { id: record::tb($p.target) + ':' + <string> record::id($p.target), version: 'sha256:' + crypto::sha256(<string> $new),
      revision_id: 'library_revision:' + $key, proposal_id: record::tb($p.id) + ':' + <string> record::id($p.id), status: 'published' };
    UPDATE $p.id SET status = 'published', publication = $publication, published_at = time::now();
    LET $sync_full = object::extend((SELECT * FROM ONLY $p.target), { id: $p.target });
    LET $sync_version = 'sha256:' + crypto::sha256(<string> object::remove($sync_full, ['embedding']));
    LET $sync_revision = type::record('library_revision', $key);
    ${capture.sql}
    RETURN $publication;
    COMMIT TRANSACTION;
  `, { key: match[1], sync_publication_target: targetId, ...capture.params });
  return resultObject(results);
}

/** Retain the actual failed condition when Surreal's transaction wrapper reports earlier statements as not executed.
 * Inputs: parameterized atomic query and bindings. Outputs: successful statement results or the actual bounded failure.
 * Effects: one database transaction. Choose over awaiting SDK collect(), which can mask a later THROW with NotExecuted.
 */
async function libraryTransaction(store: StoreOk, query: string, params: Record<string, unknown>): Promise<unknown[]> {
  const pending = store.db.query(query, params) as Promise<unknown[]> & {
    responses?: () => Promise<Array<{ success: boolean; result?: unknown; error?: { message?: string } }>>;
  };
  if (!pending.responses) return pending;
  const responses = await pending.responses();
  const failures = responses.filter(r => !r.success);
  if (failures.length) {
    const actual = failures.find(r => !/not executed|failed transaction/i.test(r.error?.message ?? "")) ?? failures[0];
    throw new Error((actual.error?.message ?? "Library transaction failed").slice(0, 300));
  }
  return responses.map(r => r.result);
}

/** Extract one bounded mutation result from the SDK transaction response.
 * Inputs: statement results. Outputs: proposal/publication envelope. Effects: none.
 * Choose instead of returning stored personal record bodies or transaction internals to logs.
 */
function resultObject(results: unknown[]): Record<string, unknown> {
  for (const raw of [...results].reverse()) {
    const value = normalize(raw);
    if (value && typeof value === "object" && !Array.isArray(value)
      && ("proposal_id" in value || "revision_id" in value)) return value as Record<string, unknown>;
  }
  throw new Error("Malformed library mutation result");
}
