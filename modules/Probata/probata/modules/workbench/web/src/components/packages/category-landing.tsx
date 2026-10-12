// Byline: Codex · GPT-6 · 2026-10-12.
import { FolderOpen } from "lucide-react";
import { ErrorBox, LoadMore, Loading } from "@/components/mobile/mobile-ui";
import { Button } from "@/components/ui/button";
import { useContextCategories, useContextPackages } from "@/hooks/use-context-package";
import type { ContextCategory } from "@/lib/context-package-client";
import { AppLink, useBrowserSearchParams } from "@/lib/router-compat";

/** Show intact package links immediately beneath one actual Case Vault category header.
 * Inputs: server category and current mode. Output: source-specific package links and paging.
 * Effects: category-bound GETs. Pick on the landing; files never move or receive inferred categories.
 */
function CategoryPackages({ category, mode }: { category: ContextCategory; mode: string }) {
  const query = useContextPackages(category.id, category.domain ?? "");
  const packages = query.data?.pages.flatMap((page) => page.items) ?? [];
  return <section className="rounded-md border bg-card" aria-label={category.label}>
    <header className="flex flex-wrap items-start justify-between gap-2 border-b bg-muted/30 px-4 py-3">
      <div><p className="platform-kicker">{category.domain || "Domain unresolved"}</p>
        <h2 className="mt-1 flex items-center gap-2 text-base font-semibold"><FolderOpen className="size-4" />{category.label}</h2>
        {category.path && <p className="mt-1 break-all text-xs text-muted-foreground">{Array.isArray(category.path) ? category.path.join(" / ") : category.path}</p>}
      </div>
      <span className="text-xs text-muted-foreground">{category.count.toLocaleString()} packages</span>
      {(category.unresolved || !category.category) && <p className="w-full text-xs text-destructive">Placement unresolved. Open the package to inspect its recorded placement and correction capability.</p>}
    </header>
    {query.isPending && <Loading />}
    {query.isError && <ErrorBox error={query.error} onRetry={() => void query.refetch()} />}
    <ul className="divide-y">
      {packages.map((entry) => <li key={entry.handle}>
        <AppLink className="flex flex-wrap items-start justify-between gap-2 px-4 py-3 hover:bg-accent focus-visible:outline focus-visible:outline-ring" href={`/read?${new URLSearchParams({ package: entry.handle, mode })}`}>
          <div className="min-w-0"><strong className="block break-words text-sm">{entry.label || entry.source_ref}</strong>
            <span className="mt-1 block break-all text-xs text-muted-foreground">{entry.source_ref}</span>
            {entry.placement?.unresolved && <span className="mt-1 block text-xs text-destructive">Unresolved placement</span>}
          </div>
          <span className="rounded border px-2 py-0.5 text-xs">{entry.source_type}</span>
        </AppLink>
      </li>)}
    </ul>
    {!query.isPending && !query.isError && !packages.length && <p className="px-4 py-4 text-sm text-muted-foreground">No ingested packages are recorded under this category.</p>}
    {query.hasNextPage && <div className="px-4 pb-3"><LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} label={`More in ${category.label}`} /></div>}
  </section>;
}

/** Open Context review with category headers and their packages visible on the same screen.
 * Inputs: route mode only. Output: the actual Case Vault domain/category hierarchy.
 * Effects: paged category and package reads. Pick before any file selection; no flat fallback list.
 */
export function CategoryLanding() {
  const params = useBrowserSearchParams();
  const query = useContextCategories();
  const categories = query.data?.pages.flatMap((page) => page.items) ?? [];
  return <main className="mx-auto max-w-[100rem] space-y-5 p-4 lg:p-6" aria-label="Context review categories" data-testid="context-category-landing">
    <header className="flex flex-wrap items-center justify-between gap-3">
      <div><h1 className="text-xl font-semibold">Context review</h1><p className="mt-1 text-sm text-muted-foreground">Open a package under its Case Vault category to inspect and correct the extracted material.</p></div>
      <Button variant="outline" size="sm" disabled={query.isFetching} onClick={() => void query.refetch()}>Refresh categories</Button>
    </header>
    {query.isPending && <Loading />}
    {query.isError && <ErrorBox error={query.error} onRetry={() => void query.refetch()} />}
    {!query.isPending && !query.isError && !categories.length && <p className="rounded border bg-card p-5 text-sm">No ingested category placements are available. <AppLink href="/sources" className="underline">Open Sources</AppLink> to inspect source placement.</p>}
    {categories.map((category) => <CategoryPackages key={`${category.domain}:${category.id}`} category={category} mode={params.get("mode") ?? "LIVE"} />)}
    {query.hasNextPage && <LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} label="More categories" />}
  </main>;
}
