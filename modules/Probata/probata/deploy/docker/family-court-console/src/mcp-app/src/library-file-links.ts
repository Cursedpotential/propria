// Byline: Codex · GPT-6 · 2026-10-05. Read-only shared original links outside record version payloads.
import type { StoreOk } from "./store.js";

export interface LibraryOriginalLink {
  binding_id: string; version_id: string; sha256: string; bytes: number; content_type: "application/pdf"; href: string;
}

/** Read server-owned exact original pointers associated with one shared record.
 * Inputs: connected shared store and exact record ID. Outputs: bounded hosted PDF links and byte/version identities.
 * Effects: one scoped metadata query; no source bytes, edits or changes to the record's version hash.
 * Choose for all human/MCP case_record envelopes; only the authenticated hosted original proxy holds the service credential.
 */
export async function libraryOriginalLinks(store: Pick<StoreOk, "db">, recordId: string): Promise<LibraryOriginalLink[]> {
  const scope = process.env.TOOLKIT_LIBRARY_SYNC_ACCOUNT_SCOPE;
  if (!scope) return [];
  const result = await store.db.query<unknown[]>(`LET $aliases = (SELECT VALUE binding_id FROM library_file_alias WHERE record_id = $record_id LIMIT 8);
    SELECT <string> id AS binding_id, original_pointer FROM library_file
    WHERE (record_id = $record_id OR <string> id IN $aliases) AND provider = 'b2' AND account_scope = $scope AND bucket = 'salem-data'
      AND legal_root = 'consignatio/casevault/KnowledgeBase/legal/' LIMIT 8;`, { record_id: recordId, scope });
  const rows = result.at(-1);
  if (!Array.isArray(rows)) throw new Error("Malformed original-file mapping result");
  const links: LibraryOriginalLink[] = [];
  for (const row of rows) {
    if (!row || typeof row !== "object") continue;
    const value = row as { binding_id?: unknown; original_pointer?: Record<string, unknown> };
    const p = value.original_pointer;
    if (typeof value.binding_id !== "string" || !/^library_file:[a-f0-9]{64}$/.test(value.binding_id) || !p
      || p.content_type !== "application/pdf" || typeof p.version_id !== "string" || !p.version_id || p.version_id === "null"
      || Buffer.byteLength(p.version_id) > 2048 || /[\r\n\0]/.test(p.version_id)
      || typeof p.sha256 !== "string" || !/^[a-f0-9]{64}$/.test(p.sha256)
      || typeof p.size !== "number" || !Number.isSafeInteger(p.size) || p.size <= 0 || p.size > 20 * 1024 * 1024) continue;
    const query = new URLSearchParams({ binding_id: value.binding_id, version_id: p.version_id });
    links.push({ binding_id: value.binding_id, version_id: p.version_id, sha256: p.sha256,
      bytes: p.size, content_type: "application/pdf", href: "https://family-court.tilapia-skilift.ts.net/api/library/original?" + query });
  }
  return links;
}
