// Byline: Codex · GPT-6 · 2026-10-06.

/** Resolve retired entry points without dropping source, attempt or search identity.
 * Inputs: pathname, encoded query and fragment. Output: same-origin canonical href.
 * Effects: none. Use at router boundaries; API/source locators are never rewritten.
 */
export function canonicalWorkflowHref(pathname: string, search = "", hash = ""): string {
  const params = new URLSearchParams(search);
  let target = pathname;
  if (pathname === "/review" || pathname === "/evidence/preview") {
    target = "/read";
    params.set("view", "review");
  } else if (pathname === "/conversations") {
    target = "/read";
  } else if (pathname === "/intake") {
    target = params.has("preview_handle") || params.has("operation_status") || params.has("operation_source") || params.has("batch")
      ? "/activity" : "/sources";
  } else if (pathname === "/") {
    target = params.has("resource") || params.has("attempt") || params.has("preview_handle") ? "/read" : "/sources";
  }
  const query = params.toString();
  return `${target}${query ? `?${query}` : ""}${hash ? `#${hash.replace(/^#/, "")}` : ""}`;
}

/** Construct a reading link to an exact processing attempt.
 * Inputs: opaque handle and operating policy. Output: encoded Read href. Effects: none.
 * Use for durable operation results; never substitute a run ID for a preview handle.
 */
export function attemptReadHref(handle: string, mode: "LIVE" | "DEV"): string {
  return `/read?${new URLSearchParams({ resource: handle, mode })}`;
}

/** Select an attempt while retaining the reading context to return to.
 * Inputs: current encoded query, exact handle and mode. Output: canonical Read href.
 * Effects: none. Use when a parser/repair creates a new attempt in the same reader.
 */
export function previewSelectionHref(search: string, handle: string, mode: "LIVE" | "DEV"): string {
  const params = new URLSearchParams(search);
  params.delete("preview_handle");
  params.delete("attempt");
  params.set("view", "review");
  params.set("mode", mode);
  if (handle) params.set("resource", handle);
  else params.delete("resource");
  return `/read?${params}`;
}
