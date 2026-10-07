// Byline: Codex · 2026-10-06.
import { useRouter } from "@tanstack/react-router";
import { ContextSearch } from "@/components/read/context-search";
import { readHref } from "@/components/read/read-location";

/** Search across Intake and ingested content while preserving Read URL context.
 * Input: current route parameters; output: shared cited-search controls/results.
 * Effects: read-only search and navigation; no ingestion, projection or promotion.
 * Choose this shared surface instead of a separate message-only search engine.
 */
export function ReadSearch({ params }: { params: URLSearchParams }) {
  const router = useRouter();
  return <ContextSearch initialQuery={params.get("q")?.trim() ?? ""}
    onSubmit={(query) => { void router.navigate({ href: readHref(params, { q: query || null }) }); }} />;
}
