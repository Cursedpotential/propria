// Byline: Codex · GPT-6 · 2026-09-23. Bounded source-list continuation.
import type { ProfferSourceBrowserResponse } from "@/lib/shared/types";

export function mergeSourcePages(pages: readonly ProfferSourceBrowserResponse[]) {
  const objects = new Map<string, ProfferSourceBrowserResponse["objects"][number]>();
  const prefixes = new Map<string, ProfferSourceBrowserResponse["prefixes"][number]>();
  for (const page of pages) {
    for (const object of page.objects) objects.set(object.source_ref, object);
    for (const prefix of page.prefixes) prefixes.set(prefix.prefix, prefix);
  }
  return { objects: [...objects.values()], prefixes: [...prefixes.values()] };
}

export function sourceContinuation(
  pages: readonly ProfferSourceBrowserResponse[],
  requestedTokens: readonly unknown[],
): { token?: string; issue?: string; complete: boolean } {
  const last = pages.at(-1);
  if (!last) return { complete: false };
  if (!last.is_truncated) return { complete: true };
  const token = last.continuation_token;
  if (!token) return { complete: false, issue: "The source returned a partial page without a continuation. Refresh the listing." };
  if (requestedTokens.includes(token)) {
    return { complete: false, issue: "The source repeated a listing page. Refresh the listing before continuing." };
  }
  return { token, complete: false };
}
