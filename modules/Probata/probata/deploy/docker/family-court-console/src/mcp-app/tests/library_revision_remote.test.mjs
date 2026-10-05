// Byline: Codex, GPT-6, 2026-10-04. Synthetic VPS-only memory database proof; no production connection.
import assert from "node:assert/strict";
import { Surreal } from "surrealdb";
import { createNodeEngines } from "@surrealdb/node";

/** Prove the shared library contract against an isolated server-side memory database.
 * Inputs: bundled library module URL in LIBRARY_PROOF_MODULE. Outputs: bounded assertion counts.
 * Effects: synthetic in-memory records only, closed in finally. Choose on the VPS; no real case or source credentials are used.
 */
async function proveLibrary() {
  const lib = await import(process.env.LIBRARY_PROOF_MODULE);
  const db = new Surreal({ engines: createNodeEngines() });
  let assertions = 0;
  try {
    await db.connect("mem://");
    await db.use({ namespace: "library_proof", database: "isolated" });
    const store = { available: true, db, dim: 1 };
    await db.query("DEFINE TABLE source SCHEMALESS; DEFINE TABLE reference SCHEMALESS;");
    await lib.migrateLibrary(store);
    await db.query("CREATE source:primary CONTENT { title: 'Synthetic primary authority', url: 'https://www.courts.michigan.gov/synthetic', body: 'Test authority' }; CREATE reference:guide CONTENT { title: 'Synthetic guide', body: 'Original', personal_context: 'Synthetic private case context' };");
    const version = async (table, key) => (await db.query("LET $r = (SELECT * OMIT embedding FROM ONLY type::record($tb, $key)); RETURN 'sha256:' + crypto::sha256(<string> $r);", { tb: table, key })).at(-1);
    const sourceVersion = await version("source", "primary");
    const citations = [{ source_id: "source:primary", source_version: sourceVersion, pinpoint: "Synthetic section", claim: "Synthetic supported claim" }];
    const input = { id: "reference:guide", expected_version: await version("reference", "guide"), patch: { body: "Proposed" }, citations, rationale: "Synthetic revision" };
    const proposal = await lib.libraryPropose(store, input);
    assert.equal(proposal.status, "pending_validation"); assertions++;
    assert.equal((await db.query("SELECT VALUE body FROM ONLY reference:guide;")).at(-1), "Original"); assertions++;
    const key = proposal.proposal_id.split(":")[1];
    const proposed = (await db.query("SELECT VALUE proposed_record FROM ONLY type::record('library_proposal', $key);", { key })).at(-1);
    assert.equal(proposed.personal_context, "Synthetic private case context"); assertions++;
    await assert.rejects(lib.libraryPublish(store, proposal.proposal_id), /validation pending/); assertions++;
    await assert.rejects(lib.libraryPropose(store, { ...input, expected_version: "sha256:" + "0".repeat(64) }), /version conflict/); assertions++;
    const claimChecks = citations.map(c => ({ ...c, status: "VERIFIED_PRIMARY", currency_status: "cleared", snapshot_ref: "b2://synthetic/primary?versionId=proof-v1", snapshot_version_id: "proof-v1", snapshot_sha256: "a".repeat(64), quote_sha256: "b".repeat(64), primary_url: "https://www.courts.michigan.gov/synthetic", evidence_time: new Date(), check_version: "synthetic-v1", currency_evidence: { release: "synthetic" } }));
    const receipt = async (patch = {}) => db.query("UPSERT type::record('library_validation', $key) CONTENT $receipt;", { key, receipt: { proposal_id: proposedProposalId(db, key), proposal_version: await version('library_proposal', key), proposed_hash: proposal.proposed_hash, status: "VERIFIED_PRIMARY", currency_status: "cleared", validator_version: "synthetic-proof-v1", completed_at: new Date(), expires_at: new Date(Date.now() + 3600000), claims: citations, claim_checks: claimChecks, ...patch } });
    await receipt({ proposed_hash: "sha256:" + "f".repeat(64) });
    await assert.rejects(lib.libraryPublish(store, proposal.proposal_id), /mismatched/); assertions++;
    await receipt({ expires_at: new Date(0) });
    await assert.rejects(lib.libraryPublish(store, proposal.proposal_id), /stale/); assertions++;
    await receipt({ currency_status: "PROVISIONAL_CURRENCY_NOT_CLEARED" });
    await assert.rejects(lib.libraryPublish(store, proposal.proposal_id), /stale/); assertions++;
    await receipt({ claim_checks: [] });
    await assert.rejects(lib.libraryPublish(store, proposal.proposal_id), /mismatched/); assertions++;
    await receipt({ claim_checks: [{ ...claimChecks[0], quote_sha256: "" }] });
    await assert.rejects(lib.libraryPublish(store, proposal.proposal_id), /incomplete/); assertions++;
    await receipt({ validator_version: "" });
    await assert.rejects(lib.libraryPublish(store, proposal.proposal_id), /mismatched/); assertions++;
    await receipt();
    const published = await lib.libraryPublish(store, proposal.proposal_id);
    assert.equal(published.status, "published"); assertions++;
    const updated = (await db.query("SELECT * FROM ONLY reference:guide;")).at(-1);
    assert.equal(updated.body, "Proposed"); assert.equal(updated.personal_context, "Synthetic private case context"); assertions += 2;
    const retained = (await db.query("SELECT VALUE previous_record FROM ONLY type::record('library_revision', $key);", { key })).at(-1);
    assert.equal(retained.body, "Original"); assert.equal(retained.personal_context, "Synthetic private case context"); assertions += 2;
    assert.deepEqual(await lib.libraryPublish(store, proposal.proposal_id), published); assertions++;
    const pending = await lib.libraryPropose(store, { ...input, expected_version: await version("reference", "guide") });
    await db.query("UPDATE source:primary SET title = 'Changed primary';");
    const pendingKey = pending.proposal_id.split(":")[1];
    await db.query("CREATE type::record('library_validation', $key) CONTENT $receipt;", { key: pendingKey, receipt: { proposal_id: proposedProposalId(db, pendingKey), proposal_version: await version('library_proposal', pendingKey), proposed_hash: pending.proposed_hash, status: "VERIFIED_PRIMARY", currency_status: "cleared", validator_version: "synthetic-proof-v1", completed_at: new Date(), expires_at: new Date(Date.now() + 3600000), claims: citations, claim_checks: claimChecks } });
    await assert.rejects(lib.libraryPublish(store, pending.proposal_id), /source version changed/); assertions++;
    const created = await lib.libraryPropose(store, { ...input, id: "reference:new", expected_version: "absent" });
    assert.equal(created.status, "pending_validation"); assertions++;
    assert.equal((await db.query("SELECT * FROM ONLY reference:new;")).at(-1), undefined); assertions++;
    await db.query("DEFINE TABLE child SCHEMAFULL; DEFINE FIELD initials ON child TYPE string; CREATE child:existing CONTENT { initials: 'S.C.' };");
    await lib.migratePersonalCaseContext(store);
    await db.query("UPDATE child:existing SET name = 'Synthetic Child Fullname', notes = 'Private context'; CREATE child:named CONTENT { name: 'Another Synthetic Child' };");
    assert.equal((await db.query("SELECT VALUE name FROM ONLY child:existing;")).at(-1), 'Synthetic Child Fullname'); assertions++;
    assert.equal((await db.query("SELECT VALUE initials FROM ONLY child:existing;")).at(-1), 'S.C.'); assertions++;
    assert.equal((await db.query("SELECT VALUE name FROM ONLY child:named;")).at(-1), 'Another Synthetic Child'); assertions++;
    const personalSource = await lib.putPersonalCaseSource(store, 'personal', { kind: 'case_document', title: 'Synthetic private document' });
    assert.equal(personalSource.title, 'Synthetic private document'); assertions++;
    await assert.rejects(lib.putPersonalCaseSource(store, 'primary', { kind: 'case_document', title: 'Attempted authority overwrite' }), /governed revision/); assertions++;
    const imported = await lib.retainLibraryImport(store, 'reference:imported', { body: 'Synthetic personal context', source: { path: 'synthetic/original' } });
    assert.equal(imported.status, 'citation_required'); assertions++;
    assert.equal((await db.query("SELECT * FROM ONLY reference:imported;")).at(-1), undefined); assertions++;
    const importKey = imported.proposal_id.split(':')[1];
    assert.equal((await db.query("SELECT VALUE proposed_record.body FROM ONLY type::record('library_proposal', $key);", { key: importKey })).at(-1), 'Synthetic personal context'); assertions++;
    await db.query("DEFINE TABLE note SCHEMALESS; CREATE note:private CONTENT { body: 'Original private context', party_name: 'Synthetic Full Name', extra: { unknown: 'Retained' } };");
    const personalVersion = await version('note', 'private');
    const personalEdited = await lib.putVersionedPersonalRecord(store, 'note', 'private', { body: 'Revised private context' }, personalVersion);
    assert.equal(personalEdited.party_name, 'Synthetic Full Name'); assertions++;
    assert.equal(String(personalEdited.id), 'note:private'); assertions++;
    assert.deepEqual(personalEdited.extra, { unknown: 'Retained' }); assertions++;
    await assert.rejects(lib.putVersionedPersonalRecord(store, 'note', 'private', { body: 'Stale overwrite' }, personalVersion), /version conflict/); assertions++;
    const history = (await db.query("SELECT previous_record, current_record, current_version FROM record_revision WHERE target = note:private;")).at(-1);
    assert.equal(history.length, 1); assertions++;
    assert.equal(history[0].previous_record.body, 'Original private context'); assertions++;
    assert.equal(history[0].previous_record.party_name, 'Synthetic Full Name'); assertions++;
    assert.equal(history[0].current_record.body, 'Revised private context'); assertions++;
    assert.equal(history[0].current_version, await version('note', 'private')); assertions++;
    await assert.rejects(lib.putVersionedPersonalRecord(store, 'source', 'primary', { kind: 'case_document' }, await version('source', 'primary')), /governed revision/); assertions++;
    assert.equal((await db.query("SELECT count() AS n FROM record_revision GROUP ALL;")).at(-1)[0].n, 1); assertions++;
    console.log(JSON.stringify({ passed: assertions, production_writes: 0, personal_fields_preserved: true }));
  } finally { await db.close(); }
}

/** Construct a synthetic receipt's typed proposal reference.
 * Inputs: test key. Outputs: SDK RecordId. Effects: none. Choose instead of a plain string that cannot equal a stored record reference.
 */
function proposedProposalId(_db, key) { return new SurrealRecordId("library_proposal", key); }
const { RecordId: SurrealRecordId } = await import("surrealdb");
if (process.env.LIBRARY_PROOF_MODULE) {
  await proveLibrary();
  // The native test engine holds a handle after close; finish only after every assertion and finally completed.
  process.exit(0);
} else {
  console.log("Skipped VPS-only library SQL proof; set LIBRARY_PROOF_MODULE explicitly on the server.");
}
