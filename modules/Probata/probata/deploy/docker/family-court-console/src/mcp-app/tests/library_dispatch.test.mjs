// Byline: Codex, GPT-6, 2026-10-04. Credential-free dispatch failure and retained draft tests.
import assert from 'node:assert/strict';
import test from 'node:test';
const { queueLibraryValidation } = await import(process.env.LIBRARY_DISPATCH_MODULE || '../dist/library-validation-dispatch.js');

/** Build a synthetic proposal store capturing only dispatch metadata.
 * Inputs: retained proposal state. Outputs: fake store and write array. Effects: local test arrays only.
 * Choose for dispatch failure behavior; real atomic library SQL is tested separately on the VPS.
 */
function proposalStore(state) {
  const writes = [];
  return { writes, store: { db: { query: async (query, args) => {
    if (query.startsWith('SELECT')) return [state];
    writes.push(args.outcome); return [];
  } } } };
}
const proposal = 'library_proposal:aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa';
test('missing validation configuration preserves an explicit retryable queue failure', async () => {
  const saved = process.env.TOOLKIT_VALIDATION_START_URL;
  delete process.env.TOOLKIT_VALIDATION_START_URL;
  try {
    const fixture = proposalStore({ status: 'pending_validation', citations: [{}] });
    const result = await queueLibraryValidation(fixture.store, proposal);
    assert.equal(result.state, 'queue_failed'); assert.equal(result.retryable, true);
    assert.equal(result.code, 'VALIDATION_CONFIGURATION'); assert.deepEqual(fixture.writes, [result]);
  } finally { if (saved !== undefined) process.env.TOOLKIT_VALIDATION_START_URL = saved; }
});
test('uncited retained imports do not launch meaningless validation workflows', async () => {
  const fixture = proposalStore({ status: 'citation_required', citations: [] });
  assert.equal((await queueLibraryValidation(fixture.store, proposal)).state, 'citation_required');
  assert.equal(fixture.writes.length, 0);
});
test('missing proposals fail before dispatch and published proposals remain published', async () => {
  await assert.rejects(queueLibraryValidation(proposalStore(undefined).store, proposal), /missing/);
  const fixture = proposalStore({ status: 'published', citations: [{}] });
  assert.equal((await queueLibraryValidation(fixture.store, proposal)).state, 'already_published');
  assert.equal(fixture.writes.length, 0);
});
