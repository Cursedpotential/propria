// Byline: Codex · GPT-6 · 2026-10-06

export const PREVIEW_KEYS = ["resource", "preview_handle", "attempt"] as const;

/** Build a Read URL while retaining unrelated query context and exact record IDs.
 * Inputs: current query and explicit replacements (null removes a parameter).
 * Output: an encoded /read href; effects: none. Pick for all Read navigation.
 */
export function readHref(params: URLSearchParams, changes: Record<string, string | null> = {}): string {
  const next = new URLSearchParams(params);
  for (const [key, value] of Object.entries(changes)) {
    if (value === null || value === "") next.delete(key);
    else next.set(key, value);
  }
  return `/read${next.size ? `?${next.toString()}` : ""}`;
}

/** Identify processing previews without interpreting or rewriting source identifiers.
 * Input: route query. Output: preview visibility; effects: none.
 * Pick for the Read entry point, including legacy preview redirects.
 */
export function isProcessingPreview(params: URLSearchParams): boolean {
  return params.get("view") === "review" || PREVIEW_KEYS.some((key) => Boolean(params.get(key)?.trim()));
}

/** Return to imported reading while preserving the selected source, thread and search.
 * Input: preview query. Output: /read href without preview selectors; effects: none.
 * Pick for the preview workspace's return link.
 */
export function importedReadingHref(params: URLSearchParams): string {
  return readHref(params, { view: null, resource: null, preview_handle: null, attempt: null });
}

/** Open a search hit without confusing its source filename with a source ID.
 * Inputs: current route, original thread ID and original message ID.
 * Output: a focused Read href; effects: none. Pick for cross-conversation hits.
 */
export function searchHitHref(params: URLSearchParams, threadId: string, messageId: string): string {
  return readHref(params, { source: null, thread: threadId, around: messageId });
}
