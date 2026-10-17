// Discovery and selected-unit inspection are distinct working states.
// Byline: ChatGPT, 2026-10-17.
import { useQuery } from "@tanstack/react-query";
import { ImportedLibrary } from "@/components/library/imported-library";
import { ErrorBox, Loading } from "@/components/mobile/mobile-ui";
import { UnitWorkspace } from "@/components/read/unit-workspace";
import { readHref } from "@/components/read/read-location";
import { ReadSearch } from "@/components/read/read-search";
import { Button } from "@/components/ui/button";
import { importedApi } from "@/lib/imported-client";
import { AppLink, useBrowserSearchParams } from "@/lib/router-compat";

/** Open selected sources in the unit workspace; reserve global search for discovery.
 * Input: existing source/thread/around URL context. Output: one working state.
 * Effects: child reads and explicit user navigation. No automatic data mutation.
 */
export function ReadWorkspace() {
  const params = useBrowserSearchParams();
  const sourceId = params.get("source");
  const threadId = params.get("thread");
  if (sourceId) return <UnitWorkspace key={`${sourceId}:${threadId ?? ""}`} sourceId={sourceId} threadId={threadId} params={params} />;
  if (threadId) return <ThreadUnitEntry key={threadId} threadId={threadId} params={params} />;
  return <div className="mx-auto max-w-[100rem] space-y-5 p-4 lg:p-6">
    <header className="flex flex-wrap items-center justify-between gap-3">
      <div><h1 className="text-xl font-semibold">Imported library</h1>
        <p className="mt-1 text-sm text-muted-foreground">Find a source, then open its unit to read messages, inspect stored records and review outputs.</p>
      </div>
      <Button asChild variant="outline"><AppLink href={readHref(params, { view: "review" })}>Processing previews</AppLink></Button>
    </header>
    <div className="grid items-start gap-5 xl:grid-cols-[27rem_minmax(0,1fr)]">
      <ImportedLibrary params={params} selectedSourceId={null} />
      <ReadSearch params={params} />
    </div>
  </div>;
}

/** Preserve thread-only deep links through the source ID returned by the existing API.
 * Inputs: native thread and route. Output: selected-unit reader. Effects: one
 * cached read; no filename inference or promotion is performed in the browser.
 */
function ThreadUnitEntry({ threadId, params }: { threadId: string; params: URLSearchParams }) {
  const header = useQuery({ queryKey: ["unit-thread-source", threadId],
    queryFn: ({ signal }) => importedApi.messages(threadId, {}, signal), retry: false,
  });
  if (header.isPending) return <Loading />;
  if (!header.data) return <ErrorBox error={header.error} onRetry={() => void header.refetch()} />;
  return <UnitWorkspace sourceId={header.data.source.id} threadId={threadId} params={params} />;
}
