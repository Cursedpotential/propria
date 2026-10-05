/**
 * Exercises the worker's actual receipt SQL against an isolated embedded memory database.
 * Inputs: repository.go path and SUR_REAL_NODE_MODULES installation directory; outputs: assertion count.
 * Effects: synthetic mem:// writes only, closed on completion; no credentials, live stores, deletion or publication.
 * Pick this integration probe after Go fixtures to verify database casts, atomic gates and substantive THROW errors.
 * Byline: Codex · GPT-6.1 · 2026-10-04.
 */
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';
import { createRequire } from 'node:module';
import { join } from 'node:path';

const modules = process.env.SUR_REAL_NODE_MODULES;
if (!modules || !process.argv[2]) throw new Error('explicit installed SDK path and repository.go required');
const requireSdk = createRequire(join(modules, '..', 'package.json'));
const { Surreal, RecordId, jsonify } = requireSdk('surrealdb');
const { createNodeEngines } = await import(pathToFileURL(join(modules, '@surrealdb/node/dist/surrealdb-node.mjs')));
const sql = /const commitSQL = `([\s\S]*?)`/.exec(readFileSync(process.argv[2], 'utf8'))?.[1];
const versionSQL = /const proposalVersionSQL = `([\s\S]*?)`/.exec(readFileSync(process.argv[2], 'utf8'))?.[1];
const currencySQL = process.argv[3] && /const currencyCommitSQL = `([\s\S]*?)`/.exec(readFileSync(process.argv[3], 'utf8'))?.[1];
assert.ok(sql, 'probe must load actual worker SQL');
assert.ok(versionSQL && currencySQL, 'probe must load actual version/currency SQL');
const db = new Surreal({ engines: createNodeEngines() });
let assertions = 1;
let fixtureNumber = 0;
const key = '11111111-2222-3333-4444-555555555555';

/** Runs all statements and selects a substantive error over transaction NotExecuted wrappers. */
async function query(text, bindings = {}) {
  const responses = await db.query(text, bindings).responses();
  const failures = responses.filter(r => !r.success).map(r => r.error);
  if (failures.length) throw failures.find(e => e.kind !== 'NotExecuted' && !/not executed|failed transaction|cancelled/i.test(e.message)) ?? failures[0];
  return responses.map(r => jsonify(r.result));
}

/** Creates synthetic personal fields and current source/currency versions in a fresh memory-only database. */
async function fixture(self = false) {
  await db.use({ namespace: 'worker_receipt_proof', database: `fixture_${++fixtureNumber}` });
  await query('DEFINE TABLE library_proposal SCHEMALESS; DEFINE TABLE library_validation SCHEMALESS; DEFINE TABLE library_currency SCHEMALESS; DEFINE TABLE source SCHEMALESS; DEFINE TABLE reference SCHEMALESS;');
  const target = new RecordId(self ? 'source' : 'reference', self ? 'new-primary' : 'personal');
  if (!self) await query('CREATE $target CONTENT $body; CREATE source:official CONTENT $source;', {
    target, body: { personal_note: 'Synthetic full person, child and address text stays preserved.' },
    source: { primary_url: 'https://www.courts.michigan.gov/fixture', current_version: 'fixture-release' },
  });
  const [targetVersion, sourceVersion] = self ? ['absent', 'absent'] : await query(`
    RETURN 'sha256:' + crypto::sha256(<string> (SELECT * OMIT embedding FROM ONLY $target));
    RETURN 'sha256:' + crypto::sha256(<string> (SELECT * OMIT embedding FROM ONLY source:official));`, { target });
  const citation = { source_id: self ? 'source:new-primary' : 'source:official', source_version: sourceVersion, pinpoint: 'MCR 3.215(E)(4)', claim: 'Fixture exact supported claim.' };
  const proposed = { personal_note: 'Synthetic complete correction with retained personal fields.', primary_url: 'https://www.courts.michigan.gov/fixture' };
  const [proposedHash] = await query("RETURN 'sha256:' + crypto::sha256(<string> $proposed);", { proposed });
  await query(`CREATE $id CONTENT $proposal; CREATE $currency CONTENT {signature:'fixture-independent-currency-signature'};`, {
    id: new RecordId('library_proposal', key), currency: new RecordId('library_currency', self ? 'new-primary' : 'official'),
    proposal: { target, expected_version: targetVersion, status: 'pending_validation', proposed_hash: proposedHash, proposed_record: proposed, citations: [citation] },
  });
  const [proposalVersion] = await query("RETURN 'sha256:' + crypto::sha256(<string> (SELECT * OMIT embedding FROM ONLY $id));", { id: new RecordId('library_proposal', key) });
  const completed = new Date(Date.now() - 1000).toISOString();
  const expires = new Date(Date.now() + 60000).toISOString();
  return { proposal_id: `library_proposal:${key}`, proposal_version: proposalVersion, proposed_hash: proposedHash,
    status: 'VERIFIED_PRIMARY', currency_status: 'cleared', validator_version: 'fixture-validator', completed_at: completed, expires_at: expires,
    claims: [citation], claim_checks: [{ ...citation, status: 'VERIFIED_PRIMARY', currency_status: 'cleared',
      snapshot_ref: 'b2://fixture/validation/pin?versionId=fixture-version', snapshot_version_id: 'fixture-version',
      snapshot_sha256: 'a'.repeat(64), quote_sha256: 'b'.repeat(64), primary_url: 'https://www.courts.michigan.gov/fixture',
      evidence_time: completed, check_version: 'fixture-check', currency_evidence: { signature: 'fixture-independent-currency-signature' } }],
    signature: 'fixture-worker-signature' };
}

/** Asserts a specific atomic rejection and confirms no validation row survived the rollback. */
async function rejects(receipt, expected) {
  await assert.rejects(query(sql, { key, receipt }), expected); assertions++;
  const [count] = await query('RETURN count(SELECT * FROM library_validation);');
  assert.equal(count, 0); assertions++;
}

try {
  await db.connect('mem://');
  let receipt = await fixture();
  const first = await query(sql, { key, receipt });
  assert.ok(first.some(r => r?.id === `library_validation:${key}` && r.signature === receipt.signature), JSON.stringify(first)); assertions++;
  const typedResponses = await query(`LET $r = (SELECT * FROM ONLY $id); RETURN {
    proposal: type::is_record($r.proposal_id), completed: type::is_datetime($r.completed_at),
    expires: type::is_datetime($r.expires_at), evidence: type::is_datetime($r.claim_checks[0].evidence_time)
  };`, { id: new RecordId('library_validation', key) });
  assert.deepEqual(typedResponses.at(-1), { proposal: true, completed: true, expires: true, evidence: true }); assertions++;
  await query(sql, { key, receipt });
  const [count] = await query('RETURN count(SELECT * FROM library_validation);');
  assert.equal(count, 1); assertions++;
  const [personal] = await query('RETURN (SELECT * FROM ONLY reference:personal).personal_note;');
  assert.equal(personal, 'Synthetic full person, child and address text stays preserved.'); assertions++;
  await assert.rejects(query(sql, { key, receipt: { ...receipt, signature: 'conflicting-signature' } }), /Library validation receipt conflict/); assertions++;
  const [preservedSignature] = await query('RETURN (SELECT * FROM ONLY $id).signature;', { id: new RecordId('library_validation', key) });
  assert.equal(preservedSignature, receipt.signature); assertions++;

  for (const [name, mutate, expected] of [
    ['proposal changed', async r => { r.proposed_hash = 'sha256:' + '0'.repeat(64); }, /Library proposal changed/],
    ['source changed', async () => { await query("UPDATE source:official SET current_version = 'changed';"); }, /Library source changed/],
    ['target changed', async () => { await query("UPDATE reference:personal SET personal_note = 'changed';"); }, /Library proposal stale/],
    ['status changed', async () => { await query("UPDATE type::record('library_proposal', $key) SET status = 'published';", { key }); }, /Library proposal changed|Library proposal stale/],
    ['currency changed', async () => { await query("UPDATE library_currency:official SET signature = 'changed';"); }, /Library currency review changed/],
    ['expired', async r => { r.expires_at = new Date(Date.now() - 1000).toISOString(); }, /Library validation expired/],
    ['future completion', async r => { r.completed_at = new Date(Date.now() + 60000).toISOString(); }, /Library validation expired/],
    ['incomplete', async r => { r.claim_checks = []; }, /Library validation evidence incomplete/],
  ]) {
    receipt = await fixture();
    await mutate(receipt);
    await rejects(receipt, expected);
    console.log(`PASS ${name}`);
  }
  receipt = await fixture();
  receipt.status = 'PARTIAL';
  receipt.currency_status = 'provisional';
  receipt.claim_checks[0].status = 'BLOCKED';
  receipt.claim_checks[0].currency_status = 'provisional';
  receipt.claim_checks[0].failure_code = 'CLAIM_UNPROCESSED';
  receipt.claim_checks[0].currency_evidence = null;
  await query(sql, { key, receipt });
  const [partial] = await query('RETURN (SELECT * FROM ONLY $id).status;', { id: new RecordId('library_validation', key) });
  assert.equal(partial, 'PARTIAL'); assertions++;
  const blockedSignature = receipt.signature;
  const retry = { ...receipt, attempt_id: 'new-run', previous_signature: blockedSignature, signature: 'new-signed-receipt' };
  await query(sql, { key, receipt: retry });
  const [replacement] = await query('RETURN (SELECT * FROM ONLY $id).signature;', { id: new RecordId('library_validation', key) });
  assert.equal(replacement, retry.signature); assertions++;
  await assert.rejects(query(sql, { key, receipt: { ...retry, attempt_id: 'racing-run', signature: 'racing-signature' } }), /Library validation receipt conflict/); assertions++;

  receipt = await fixture();
  const versionsBefore = (await query(versionSQL, { key })).at(-1);
  assert.equal(versionsBefore.immutable, receipt.proposal_version); assertions++;
  await query("UPDATE type::record('library_proposal', $key) SET dispatch = {state:'started'}, dispatch_at = time::now();", { key });
  const versionsAfter = (await query(versionSQL, { key })).at(-1);
  assert.equal(versionsAfter.immutable, versionsBefore.immutable); assertions++;
  assert.notEqual(versionsAfter.full, versionsBefore.full); assertions++;
  await query(sql, { key, receipt });
  assertions++;
  await query("UPDATE type::record('library_proposal', $key) SET rationale = 'substantive changed rationale';", { key });
  await assert.rejects(query(sql, { key, receipt }), /Library proposal changed/); assertions++;

  // Currency writes use their own actual transaction and never modify the source or proposal.
  receipt = await fixture();
  const checked = new Date(Date.now() - 1000).toISOString();
  const evidence = { source_id: 'source:official', source_version: receipt.claims[0].source_version,
    status: 'cleared', signature: 'fixture-authenticated-review', effective_at: new Date(Date.now()-60000).toISOString(),
    reviewed_through: checked, checked_at: checked, expires_at: new Date(Date.now()+60000).toISOString() };
  await query('DELETE library_currency:official;');
  await query(currencySQL, { evidence, proposal_id: receipt.proposal_id, proposal_version: receipt.proposal_version, proposed_hash: receipt.proposed_hash });
  const currencyTyped = (await query('LET $c = (SELECT * FROM ONLY library_currency:official); RETURN type::is_datetime($c.checked_at) AND type::is_datetime($c.reviewed_through) AND type::is_datetime($c.expires_at);')).at(-1);
  assert.equal(currencyTyped, true); assertions++;
  await query(currencySQL, { evidence, proposal_id: receipt.proposal_id, proposal_version: receipt.proposal_version, proposed_hash: receipt.proposed_hash }); assertions++;
  await assert.rejects(query(currencySQL, { evidence: { ...evidence, signature:'out-of-order-review' }, proposal_id: receipt.proposal_id, proposal_version: receipt.proposal_version, proposed_hash: receipt.proposed_hash }), /Library currency review changed/); assertions++;
  await query("UPDATE source:official SET current_version = 'changed-after-review';");
  await assert.rejects(query(currencySQL, { evidence, proposal_id: receipt.proposal_id, proposal_version: receipt.proposal_version, proposed_hash: receipt.proposed_hash }), /Library source changed/); assertions++;
  receipt = await fixture(true);
  await query(sql, { key, receipt });
  const [absent] = await query('RETURN count(SELECT * FROM source);');
  assert.equal(absent, 0, 'primary capture validation must not publish a new source'); assertions++;
  await query('DELETE library_currency:`new-primary`;');
  await query(currencySQL, { evidence: { ...evidence, source_id:'source:new-primary', source_version:'absent' }, proposal_id:receipt.proposal_id, proposal_version:receipt.proposal_version, proposed_hash:receipt.proposed_hash }); assertions++;
  console.log(JSON.stringify({ engine: 'embedded mem://', assertions, production_writes: 0 }));
} finally {
  await db.close();
}
