// Read-only advanced retrieval shared with Sources and imported reading.
// Mode, selected retrieval sources and exact Super Index scope survive direct links.
import { useRouter } from "@tanstack/react-router";
import { ContextSearch } from "@/components/read/context-search";
import { readHref } from "@/components/read/read-location";
import type { RetrievalLeg, RetrievalMode } from "@/lib/retrieval-types";

/**
 * Preserve advanced search state while navigating between original and imported readers.
 * Input: existing route parameters. Output: existing combined search with optional
 * pre-ranking Super Index filters. Effects: read-only queries and route navigation.
 * Choose instead of a separate message-only or vault-only search UI.
 */
export function ReadSearch({ params }: { params: URLSearchParams }) {
  const router = useRouter();
  const mode = params.get("search_mode");
  const initialMode: RetrievalMode =
    mode === "keyword" || mode === "vector" || mode === "hybrid" ? mode : "hybrid";
  const accepted: RetrievalLeg[] = ["intake", "proffer", "casebible"];
  const parsed = (params.get("search_legs") ?? "").split(",").filter(
    (candidate): candidate is RetrievalLeg => accepted.includes(candidate as RetrievalLeg),
  );
  const initialLegs = parsed.length ? [...new Set(parsed)] : ["intake", "proffer"] as RetrievalLeg[];

  return <ContextSearch
    initialQuery={params.get("q")?.trim() ?? ""}
    initialMode={initialMode}
    initialLegs={initialLegs}
    initialPathPrefix={params.get("search_path") ?? ""}
    initialSourceId={params.get("search_source_id") ?? ""}
    onSubmit={(query, selection) => {
      void router.navigate({ href: readHref(params, {
        q: query || null,
        search_mode: selection.mode === "hybrid" ? null : selection.mode,
        search_legs: selection.legs.join(","),
        search_path: selection.pathPrefix || null,
        search_source_id: selection.sourceId || null,
      }) });
    }} />;
}
