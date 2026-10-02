export type ProbataLink = { kind: "entity" | "event"; recordId: string };

export function parseProbataLink(search: string): ProbataLink | null {
  const params = new URLSearchParams(search);
  const kind = params.get("probata_kind");
  const recordId = params.get("probata_id");
  if ((kind !== "entity" && kind !== "event") || !recordId
    || !/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(recordId)
    || recordId === "00000000-0000-0000-0000-000000000000") return null;
  return { kind, recordId: recordId.toLowerCase() };
}
