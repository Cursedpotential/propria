// Byline: Codex · GPT-6 · 2026-10-12.
// Resizable selection/editor composition follows sbv/message-browser.tsx's RAGFlow-inspired donor.
import { useBlocker, useRouter } from "@tanstack/react-router";
import { useMemo, useState, type ReactNode } from "react";
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";
import { MetadataFieldTable } from "@/components/metadata/metadata-field-table";
import { ErrorBox, Loading } from "@/components/mobile/mobile-ui";
import { PackageItemEditor } from "@/components/packages/package-item-editor";
import { PackageItemsGrid } from "@/components/packages/package-items-grid";
import { PackageRelations, PackageTimeline } from "@/components/packages/package-relations";
import { PackageRetry } from "@/components/packages/package-retry";
import { readHref } from "@/components/read/read-location";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useContextPackage, useContextPackageReview } from "@/hooks/use-context-package";
import type { ContextPackageItem } from "@/lib/context-package-client";
import { reviewedPackageItem } from "@/lib/context-package-review";
import { AppLink, useBrowserSearchParams } from "@/lib/router-compat";
import { cn } from "@/lib/utils";

/** Compose complete extracted-package review with dynamic source sections and real edits.
 * Inputs: native package handle and optional existing original-source renderer.
 * Output: category breadcrumb, section inventory, Glide/timeline/relations and selected editor.
 * Effects: bounded section GETs, explicit mutations and guarded route navigation.
 * Pick for package review across SMS/AI/document/screenshot/location sources; original
 * viewers and legacy URLs remain reachable without fabricating normalized record IDs.
 */
export function ContextPackageWorkspace({ handle, original }: { handle: string; original?: ReactNode }) {
  const params = useBrowserSearchParams();
  const router = useRouter();
  const sectionId = params.get("section") ?? "";
  const query = useContextPackage(handle, sectionId);
  const review = useContextPackageReview(handle);
  const [dirty, setDirty] = useState(false);
  const [retryDirty, setRetryDirty] = useState(false);
  const [retryPending, setRetryPending] = useState(false);
  const [view, setView] = useState<"table" | "timeline" | "relations">("table");
  const [lastReceipt, setLastReceipt] = useState<Record<string, unknown> | null>(null);
  const blocker = useBlocker({ shouldBlockFn: () => dirty || retryDirty || retryPending || review.isPending, enableBeforeUnload: dirty || retryDirty || retryPending || review.isPending, withResolver: true });
  const snapshot = query.data?.pages.at(-1);
  const items = useMemo(() => {
    const byId = new Map<string, ContextPackageItem>();
    for (const page of query.data?.pages ?? []) for (const item of page.items) byId.set(item.id, item);
    return [...byId.values()].map(reviewedPackageItem);
  }, [query.data]);
  const selectedId = params.get("item") ?? items[0]?.id ?? null;
  const selected = items.find((item) => item.id === selectedId) ?? null;
  const activeSection = snapshot?.sections.find((section) => section.id === sectionId) ?? (!sectionId ? snapshot?.sections.find((section) => section.available) : undefined);
  const originalSelected = params.get("display") === "original" && Boolean(original);

  /** Navigate through the existing route while retaining source/mode compatibility parameters.
   * Input: explicit route replacements. Output: router navigation. Effects: guarded URL change.
   * Pick for every section/item/view selection so dirty drafts cannot disappear silently.
   */
  const navigate = (changes: Record<string, string | null>) => { void router.navigate({ href: readHref(params, changes) }); };
  const select = (id: string) => navigate({ item: id, display: null });

  return <main className="flex min-h-[40rem] flex-col gap-3 p-3 lg:p-4" aria-label="Extracted package review" data-testid="context-package-workspace">
    <header className="flex flex-wrap items-start justify-between gap-2">
      <div className="min-w-0"><AppLink className="text-xs underline underline-offset-4" href={`/read?${new URLSearchParams({ mode: params.get("mode") ?? "LIVE" })}`}>Context review categories</AppLink>
        <p className="mt-2 text-xs text-muted-foreground">{snapshot?.placement?.domain ?? snapshot?.domain ?? "Domain unresolved"} / {snapshot?.placement?.category ?? snapshot?.category ?? "Category unresolved"}</p>
        <h1 className="mt-1 break-all text-lg font-semibold">{snapshot?.source_ref.split("/").at(-1) ?? "Extracted package"}</h1>
        <p className="mt-1 text-xs text-muted-foreground">{snapshot?.source_type} · {items.length} loaded items{activeSection ? ` in ${activeSection.label}` : ""}{dirty ? " · Unsaved change" : ""}</p>
      </div>
      <div className="flex gap-2"><Button size="sm" variant="outline" disabled={query.isFetching || dirty || retryDirty || retryPending || review.isPending} onClick={() => void query.refetch()}>Refresh package</Button>
        {original && <Button size="sm" variant="outline" aria-pressed={originalSelected} onClick={() => navigate({ display: originalSelected ? null : "original" })}>{originalSelected ? "Extracted package" : "Original viewer"}</Button>}
      </div>
    </header>
    {query.isError && <ErrorBox error={query.error} onRetry={() => void query.refetch()} />}
    {query.isPending && !originalSelected && <Loading label="Loading extracted package" />}
    {originalSelected ? original : snapshot && <>
      <details className="rounded border bg-card px-3 py-2 text-xs"><summary className="cursor-pointer">Package metadata, placement and source identity</summary>
        <dl className="my-3 grid gap-1 break-all"><dt className="font-medium">Package</dt><dd>{snapshot.package_ref}</dd><dt className="font-medium">Source</dt><dd>{snapshot.source_ref}</dd><dt className="font-medium">Source version</dt><dd>{snapshot.source_version_ref}</dd><dt className="font-medium">Revision</dt><dd>{snapshot.revision}</dd></dl>
        <MetadataFieldTable rows={Object.entries(snapshot.metadata).map(([fieldKey, value]) => ({ fieldKey, label: fieldKey, value }))} corrections={new Map()} correctable={false} pendingField={null} onSubmit={async () => false} />
        <pre className="mt-3 whitespace-pre-wrap break-all">{JSON.stringify({ placement: snapshot.placement, domain: snapshot.domain, category: snapshot.category, path: snapshot.path, classification_basis: snapshot.classification_basis }, null, 2)}</pre>
      </details>
      <PackageRetry handle={handle} capabilities={snapshot.capabilities} disabled={dirty || review.isPending} onDirtyChange={setRetryDirty} onPendingChange={setRetryPending} />
      <div className="flex min-h-[33rem] flex-1 flex-col overflow-hidden rounded border bg-card lg:h-[calc(100dvh-16rem)] lg:flex-row">
        <nav className="flex shrink-0 gap-1 overflow-auto border-b p-2 lg:w-48 lg:flex-col lg:border-b-0 lg:border-r" aria-label="Package sections">
          {snapshot.sections.map((section) => <Button key={section.id} variant="ghost" className={cn("h-auto min-h-9 justify-between gap-2 whitespace-normal text-left text-xs", section.id === activeSection?.id && "bg-accent text-accent-foreground")}
            aria-current={section.id === activeSection?.id ? "page" : undefined} title={section.reason ?? undefined} onClick={() => navigate({ section: section.id, item: null, display: null })}>
            <span>{section.label}{!section.available && <span className="mt-1 block font-normal text-muted-foreground">{section.reason || "Not available"}</span>}</span><span>{section.count ?? "?"}</span>
          </Button>)}
        </nav>
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b px-3 py-2">
            <div role="group" aria-label="Package item view" className="flex gap-1">{(["table", "timeline", "relations"] as const).map((mode) => <Button key={mode} size="sm" variant={view === mode ? "secondary" : "ghost"} aria-pressed={view === mode} onClick={() => setView(mode)}>{mode === "table" ? "Items" : mode === "timeline" ? "Timeline" : "Relationships / graph"}</Button>)}</div>
            <span className="text-xs text-muted-foreground">{activeSection?.count ?? items.length} recorded · {items.length} loaded</span>
          </div>
          {!activeSection?.available && activeSection && <p className="border-b px-4 py-3 text-sm">{activeSection.reason ?? "This section is not currently available."}</p>}
          {Boolean(query.data?.pageParams[0]) && <div className="flex items-center justify-between gap-2 border-b px-3 py-2 text-xs"><span>Earlier pages were unloaded to keep this review bounded.</span><Button size="sm" variant="outline" disabled={query.isFetching || dirty || retryDirty || retryPending || review.isPending} onClick={() => void query.restart()}>First section page</Button></div>}
          <PanelGroup direction="horizontal" autoSaveId="context-package-review" className="min-h-[28rem] flex-1">
            <Panel defaultSize={52} minSize={30}><div className="flex h-full min-h-0 flex-col">
              <div className="min-h-0 flex-1 overflow-auto">
                {!items.length ? <p className="p-4 text-sm text-muted-foreground">No items returned in this section.</p> : view === "table" ? <PackageItemsGrid rows={items} selectedId={selectedId} onSelect={select} /> : view === "timeline" ? <PackageTimeline items={items} onSelect={select} /> : <PackageRelations items={items} onSelect={select} />}
              </div>
              {query.hasNextPage && <Button className="m-3" size="sm" variant="outline" disabled={query.isFetchingNextPage || dirty} onClick={() => void query.fetchNextPage()}>{query.isFetchingNextPage ? "Loading…" : "Load next section page"}</Button>}
            </div></Panel>
            <PanelResizeHandle className="w-px bg-border data-[resize-handle-state=drag]:bg-primary" />
            <Panel defaultSize={48} minSize={25}><div className="h-full overflow-auto">
              {selected ? <PackageItemEditor key={`${selected.id}:${selected.revision}`} item={selected} externalBusy={retryDirty || retryPending} onDirtyChange={setDirty} onSelect={select} onSave={async (input, key) => { const receipt = await review.mutateAsync({ input, key }); setLastReceipt(receipt); return receipt; }} /> : <p className="p-4 text-sm text-muted-foreground">{params.has("item") ? "The linked item is not in the loaded section page. Load the next page or select its section." : "Select an item to inspect its complete fields, source locators and review actions."}</p>}
            </div></Panel>
          </PanelGroup>
        </div>
      </div>
      {lastReceipt && <details className="rounded border p-3 text-xs"><summary className="cursor-pointer">Latest saved review receipt</summary><pre className="mt-2 whitespace-pre-wrap break-all">{JSON.stringify(lastReceipt, null, 2)}</pre></details>}
    </>}
    {!originalSelected && query.isError && original && <Button variant="outline" onClick={() => navigate({ display: "original" })}>Open existing original viewer</Button>}
    <Dialog open={blocker.status === "blocked"} onOpenChange={(open) => { if (!open) blocker.reset?.(); }}>
      <DialogContent><DialogHeader><DialogTitle>{review.isPending || retryPending ? "Review action is in progress" : "Keep your unsaved review?"}</DialogTitle><DialogDescription>{review.isPending || retryPending ? "Wait for the action response before leaving this package." : "Stay to finish the change, or discard the draft and continue. Recorded sources have not changed."}</DialogDescription></DialogHeader>
        <DialogFooter><Button variant="outline" onClick={() => blocker.reset?.()}>Stay here</Button><Button disabled={review.isPending || retryPending} onClick={() => { setDirty(false); setRetryDirty(false); blocker.proceed?.(); }}>Discard draft and continue</Button></DialogFooter>
      </DialogContent>
    </Dialog>
  </main>;
}
