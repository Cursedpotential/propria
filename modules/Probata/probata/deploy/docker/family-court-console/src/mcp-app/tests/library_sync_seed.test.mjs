// Byline: Codex · GPT-6 · 2026-10-05. Pinned corpus binding admission, synthetic metadata only.
import test from 'node:test';
import assert from 'node:assert/strict';
import { admitLibrarySeed } from '../dist/library-sync-seed.js';

/** Construct the approved-size synthetic shape without actual source names or case content.
 * Inputs: none. Outputs: complete metadata fixture. Effects: none; choose for count/path/hash-pin admission boundaries.
 */
function fixture() {
  return { receipt_sha256: '244061ffa2d62568938de8b8ebd0792fa924a94c7aee3b5c4d868e3c47cf9b84', manifest_sha256: '3f5a219804149b1e3175ffe048887723c2de233789a50860b7cfbf4d781f1ce4', reference_map_sha256: '6801dcf9fc4d11d812b000180066f941b167471a1dd7e5a2bbdd3010d71232e3',
    files: Array.from({ length: 443 }, (_, i) => ({ key: 'consignatio/casevault/KnowledgeBase/legal/reference-data/synthetic-' + i + '.md',
      pointer: { version_id: 'exact-' + i, sha256: 'a'.repeat(64), size: 1, content_type: 'text/markdown', observed_at: '2026-10-05T00:00:00Z' },
      ...(i < 321 ? { reference_id: 'reference:synthetic-' + i } : {}) })),
    source_ids: Array.from({ length: 193 }, (_, i) => 'source:synthetic-' + i) };
}

test('complete pinned metadata admits without source bodies', () => { assert.doesNotThrow(() => admitLibrarySeed(fixture())); });
test('partial corpus, duplicate identities and changed evidence pins refuse', () => {
  for (const change of [x => x.files.pop(), x => x.source_ids.pop(), x => x.receipt_sha256 = 'b'.repeat(64),
    x => x.files[1].key = x.files[0].key, x => x.files[1].reference_id = x.files[0].reference_id,
    x => x.source_ids[1] = x.source_ids[0]]) { const value = fixture(); change(value); assert.throws(() => admitLibrarySeed(value)); }
});
test('R2 paths, arbitrary record tables and nonexact provider pins cannot enter the binding map', () => {
  for (const change of [x => x.files[0].key = 'r2/old.md', x => x.source_ids[0] = 'library_sync_outbox:malicious',
    x => x.files[0].reference_id = 'person:arbitrary', x => x.files[0].pointer.version_id = 'null',
    x => x.files[0].pointer.size = 20 * 1024 * 1024 + 1]) { const value = fixture(); change(value); assert.throws(() => admitLibrarySeed(value)); }
});
