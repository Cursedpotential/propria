// Byline: Codex · 2026-10-06.
/** Parse a typed graph reference returned by Intake, preserving escaped string IDs.
 * Input: SDK-rendered reference; output: validated table/key/canonical identity or null.
 * No I/O occurs. Pick before neighborhood navigation; never splice raw IDs into queries.
 */
export function graphReference(value: unknown): { table: string; key: string; identity: string } | null {
  if (typeof value !== "string") return null;
  const match = /^([A-Za-z_][A-Za-z0-9_]{0,63}):(?:([A-Za-z0-9_-]{1,255})|⟨([0-9]{64})⟩)$/.exec(value);
  if (!match) return null;
  const key = match[2] ?? match[3];
  return { table: match[1], key, identity: `${match[1]}:${key}` };
}
