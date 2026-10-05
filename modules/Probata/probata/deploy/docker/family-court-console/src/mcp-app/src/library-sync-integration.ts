// Byline: Codex · GPT-6 · 2026-10-05. Atomic application-edit integration for Case Bible sync.
import { randomUUID } from "node:crypto";
import { buildLibrarySyncCapture, type SyncScope, type SyncStore } from "./library-sync-backend.js";

/** Resolve the explicitly configured shared Case Bible scope without inventing an account identity.
 * Inputs: server environment. Outputs: exact scope or undefined when disabled. Effects: none.
 * Choose at server composition and edit preflight; malformed enabled configuration fails visibly.
 */
export function configuredLibrarySyncScope(): SyncScope | undefined {
  const account = process.env.TOOLKIT_LIBRARY_SYNC_ACCOUNT_SCOPE?.trim();
  if (!account) return undefined;
  const bucket = process.env.TOOLKIT_LIBRARY_SYNC_B2_BUCKET?.trim() || "salem-data";
  const root = process.env.TOOLKIT_LIBRARY_SYNC_LEGAL_ROOT?.trim() || "consignatio/casevault/KnowledgeBase/legal/";
  if (!/^b2-account-sha256:[a-f0-9]{64}$/.test(account) || bucket !== "salem-data" || root !== "consignatio/casevault/KnowledgeBase/legal/")
    throw new Error("Invalid shared library sync scope");
  return { account_scope: account, bucket, legal_root: root };
}

/** Prepare a same-transaction native snapshot capture or a guard against a newly appearing binding.
 * Inputs: store, trusted record identity and SQL variables for the saved full snapshot/version/revision.
 * Outputs: parameterized SQL fragment. Effects: bounded binding lookup only; caller executes after authorized edit before COMMIT.
 * Choose for personal edits and validated library publication; preserves embedding and every private/unknown field in durable exports.
 */
export async function prepareLibraryEditCapture(store: SyncStore, recordId: string, snapshot: string, version: string, revision: string): Promise<{ sql: string; params: Record<string, unknown> }> {
  const scope = configuredLibrarySyncScope();
  if (!scope) return { sql: "", params: {} };
  if (!/^[a-z_]+:[^\s:]{1,200}$/.test(recordId)) throw new Error("Invalid sync capture record identity");
  const rows = await store.db.query("SELECT * FROM library_file WHERE record_id = $sync_record_id LIMIT 2;", { sync_record_id: recordId });
  const bindings = rows.at(-1);
  if (!Array.isArray(bindings) || bindings.length > 1) throw new Error("Ambiguous shared file binding");
  if (!bindings.length) return { params: { sync_record_id: recordId }, sql: `
    LET $sync_unexpected = (SELECT VALUE id FROM library_file WHERE record_id = $sync_record_id LIMIT 1);
    IF array::len($sync_unexpected) > 0 { THROW 'sync:binding_appeared_retry'; };
  ` };
  const binding = bindings[0] as Record<string, unknown>;
  if (binding.account_scope !== scope.account_scope || binding.bucket !== scope.bucket || binding.legal_root !== scope.legal_root || typeof binding.pointer_revision !== "string")
    throw new Error("Shared file binding scope mismatch");
  return buildLibrarySyncCapture({ operation_id: randomUUID(), binding_id: String(binding.id), snapshot_variable: snapshot,
    record_version_variable: version, revision_ref_variable: revision, expected_pointer_revision: binding.pointer_revision });
}
