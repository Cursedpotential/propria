// Byline: Codex · GPT-6 · 2026-10-05. Governed linkage of verified permanent files to existing private records.
import { createHash } from "node:crypto";
import { bindLibraryFile, libraryFileBindingId, type SyncPointer, type SyncScope, type SyncStore } from "./library-sync-backend.js";

const ROOT = "consignatio/casevault/KnowledgeBase/legal/";
const PINS = { receipt: "244061ffa2d62568938de8b8ebd0792fa924a94c7aee3b5c4d868e3c47cf9b84", manifest: "3f5a219804149b1e3175ffe048887723c2de233789a50860b7cfbf4d781f1ce4", map: "6801dcf9fc4d11d812b000180066f941b167471a1dd7e5a2bbdd3010d71232e3" };
interface SeedFile { key: string; pointer: SyncPointer; reference_id?: string; source_id?: string }
export interface LibrarySeedInput { receipt_sha256: string; manifest_sha256: string; reference_map_sha256: string; files: SeedFile[]; source_ids: string[] }

/** Admit the complete pinned placement map before creating any file bindings or original aliases.
 * Inputs: metadata from the tracked catalog Activity. Outputs: same admitted batch. Effects: none.
 * Choose only for the approved 443-file corpus; files and private record content never enter source control.
 */
export function admitLibrarySeed(input: LibrarySeedInput): void {
  if (!input || input.receipt_sha256 !== PINS.receipt || input.manifest_sha256 !== PINS.manifest || input.reference_map_sha256 !== PINS.map
    || !Array.isArray(input.files) || input.files.length !== 443 || !Array.isArray(input.source_ids) || input.source_ids.length !== 193)
    throw new Error("Unapproved library binding batch");
  const keys = new Set<string>(); const refs = new Set<string>();
  for (const file of input.files) {
    libraryFileBindingId({ account_scope: "seed-admission", bucket: "salem-data", legal_root: ROOT }, file.key);
    if (keys.has(file.key)) throw new Error("Duplicate placement key"); keys.add(file.key);
    const p = file.pointer;
    if (!p || !/^[a-f0-9]{64}$/.test(p.sha256) || !p.version_id || p.version_id === "null" || p.version_id.length > 2048 || /[\r\n\0]/.test(p.version_id)
      || typeof p.content_type !== "string" || !p.content_type || p.content_type.length > 200 || /[\r\n\0]/.test(p.content_type) || !Number.isSafeInteger(p.size) || p.size < 0
      || p.size > 20 * 1024 * 1024 || !Number.isFinite(Date.parse(p.observed_at))) throw new Error("Invalid placement pointer");
    if (file.reference_id) { if (!/^reference:[^\s:]{1,200}$/.test(file.reference_id) || refs.has(file.reference_id)) throw new Error("Ambiguous reference binding"); refs.add(file.reference_id); }
    if (file.source_id && !/^source:[^\s:]{1,200}$/.test(file.source_id)) throw new Error("Invalid source alias");
  }
  if (refs.size !== 321 || new Set(input.source_ids).size !== 193 || input.source_ids.some(id => !/^source:[^\s:]{1,200}$/.test(id))) throw new Error("Incomplete shared record map");
}

/** Link verified originals without replacing record bodies and reserve distinct structured-source export files.
 * Inputs: shared store/scope and complete approved metadata. Outputs: bounded binding/alias counts.
 * Effects: additive private mappings only; each retry verifies exact existing identities and never resets live pointers.
 * Choose after catalog/placement readback, before enabling periodic sync; legal publication remains separately validated.
 */
export async function seedLibraryBindings(store: SyncStore, scope: SyncScope, input: LibrarySeedInput): Promise<Record<string, unknown>> {
  admitLibrarySeed(input);
  const referenceIds = input.files.flatMap(file => file.reference_id ? [file.reference_id] : []);
  const sourceRows = (await store.db.query("SELECT VALUE <string> id FROM source WHERE <string> id IN $ids LIMIT 194;", { ids: input.source_ids })).at(-1);
  const referenceRows = (await store.db.query("SELECT VALUE <string> id FROM reference WHERE <string> id IN $ids LIMIT 322;", { ids: referenceIds })).at(-1);
  if (!Array.isArray(sourceRows) || sourceRows.length !== 193 || !Array.isArray(referenceRows) || referenceRows.length !== 321)
    throw new Error("Shared corpus records are missing; no bindings written");
  await store.db.query("DEFINE TABLE IF NOT EXISTS library_file_alias SCHEMALESS PERMISSIONS NONE; DEFINE INDEX IF NOT EXISTS library_file_alias_unique ON library_file_alias FIELDS record_id, binding_id UNIQUE;");
  let originals = 0; let aliases = 0; let sources = 0;
  const ensure = async (key: string, recordId: string, format: string, exportKey?: string, original?: SyncPointer) => {
    const id = libraryFileBindingId(scope, key);
    const rows = await store.db.query("SELECT * FROM ONLY type::record('library_file', $key);", { key: id.slice("library_file:".length) });
    const prior = rows.at(-1) as Record<string, unknown> | undefined;
    if (prior) {
      if (prior.record_id !== recordId || prior.format !== format || (prior.record_export_key || "") !== (exportKey || "")
        || prior.account_scope !== scope.account_scope || prior.bucket !== scope.bucket || prior.key !== key) throw new Error("Existing library binding conflicts");
    } else await bindLibraryFile(store, scope, { key, record_id: recordId, format, record_export_key: exportKey, original_pointer: original });
    return id;
  };
  for (const file of input.files) {
    if (!file.reference_id && !file.source_id) continue;
    const referenceId = file.reference_id ?? "reference:original-" + createHash("sha256").update(file.key).digest("hex");
    const markdown = file.key.endsWith(".md");
    const companion = markdown ? undefined : ROOT + "reference-data/family-court-records/reference/" + createHash("sha256").update(referenceId).digest("hex") + ".json";
    const id = await ensure(file.key, referenceId, markdown ? "markdown" : "record", companion, file.pointer); originals++;
    if (file.source_id) {
      const aliasKey = createHash("sha256").update(JSON.stringify([file.source_id, id])).digest("hex");
      await store.db.query("UPSERT type::record('library_file_alias', $alias) CONTENT { record_id: $record_id, binding_id: $binding_id, provenance: $provenance };",
        { alias: aliasKey, record_id: file.source_id, binding_id: id, provenance: { reference_map_sha256: PINS.map, receipt_sha256: PINS.receipt } });
      aliases++;
    }
  }
  for (const recordId of input.source_ids) {
    const key = ROOT + "reference-data/family-court-records/source/" + createHash("sha256").update(recordId).digest("hex") + ".json";
    await ensure(key, recordId, "record-json", key); sources++;
  }
  return { status: "linked", original_bindings: originals, source_export_bindings: sources, original_aliases: aliases, receipt_sha256: PINS.receipt, reference_map_sha256: PINS.map };
}
