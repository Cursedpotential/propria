const DEFAULT_ADVOCATIO_WEB_ORIGIN = "https://legal.tilapia-skilift.ts.net";
const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/** Build a read-only navigation link for a native Probata registry entity. */
export function buildAdvocatioEntityUrl(personId: string, configuredOrigin?: string): string | null {
  if (!UUID_PATTERN.test(personId) || personId === "00000000-0000-0000-0000-000000000000") return null;

  let destination: URL;
  try {
    destination = new URL(configuredOrigin?.trim() || DEFAULT_ADVOCATIO_WEB_ORIGIN);
  } catch {
    return null;
  }

  // The override is an origin only. Keep the destination on HTTPS and reject
  // credentials or appended paths/queries that could redirect the return flow.
  if (
    destination.protocol !== "https:" ||
    destination.username.length > 0 ||
    destination.password.length > 0 ||
    destination.pathname !== "/" ||
    destination.search.length > 0 ||
    destination.hash.length > 0
  ) {
    return null;
  }

  const url = new URL("/claims", destination.origin);
  url.searchParams.set("probata_kind", "entity");
  url.searchParams.set("probata_id", personId);
  return url.toString();
}
