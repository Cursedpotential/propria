// Byline: Codex · GPT-6 · 2026-10-06
import type { ReactElement } from "react";
import { AICandidateReview } from "@/components/review/ai-candidate-review";
import { CategoryLanding } from "@/components/packages/category-landing";
import { ContextPackageWorkspace } from "@/components/packages/context-package-workspace";
import { HumanContextPreview } from "@/components/sbv/human-context-preview";
import { ReadWorkspace } from "@/components/read/read-workspace";
import { importedReadingHref, isProcessingPreview } from "@/components/read/read-location";
import { ProfferPreviewClient } from "@/components/sbv/proffer-preview-client";
import { DecodedSourceViewer } from "@/components/sbv/decoded-source-viewer";
import { SourceMessagePreview } from "@/components/sbv/source-message-preview";
import { AppLink, useBrowserSearchParams } from "@/lib/router-compat";
import { isHumanPreviewHandle } from "@/lib/human-context";
import { useFixedCase } from "@/lib/fixed-case-context";

/** Open category-organized context review while preserving existing source and preview links.
 * Inputs: native package/resource, source/thread/search or preview URL identifiers.
 * Output: category landing or a complete package, with existing human/SMS viewers retained.
 * Effects: bounded child reads and explicit review writes. Pick as /read and legacy redirects.
 */
export default function ReadPage(): ReactElement {
  const params = useBrowserSearchParams();
  const { mode } = useFixedCase();
  const selectedPackage = params.get("package");
  const nativeResource = params.get("resource") ?? params.get("preview_handle") ?? "";
  if (selectedPackage) return <ContextPackageWorkspace key={selectedPackage} handle={selectedPackage}
    original={isHumanPreviewHandle(selectedPackage) ? <HumanContextPreview key={selectedPackage} handle={selectedPackage} mode={mode} /> : undefined} />;
  if (isHumanPreviewHandle(nativeResource)) return <div className="space-y-3 p-4">
    <AppLink className="inline-block text-sm underline" href={`/read?${new URLSearchParams({ package: nativeResource, mode })}`}>Open full extracted package</AppLink>
    <HumanContextPreview key={nativeResource} handle={nativeResource} mode={mode} />
  </div>;
  const rawSource = params.get("raw_message_source");
  if (rawSource) {
    const bytes = Number(params.get("bytes"));
    const root = params.get("root") ?? "";
    const key = params.get("key") ?? "";
    const etag = params.get("etag") ?? "";
    return <div className="space-y-4 p-4">
      <header className="flex items-center justify-between gap-3"><div><h1 className="text-xl font-semibold">Messages</h1>
        <p className="text-sm text-muted-foreground">{key.split("/").at(-1)}</p></div>
        <AppLink href={`/sources?${new URLSearchParams({ root, file: key.split("/").at(-1) ?? "", mode: params.get("mode") ?? "LIVE" })}`} className="text-sm underline">Back to Sources</AppLink></header>
      {root && key && etag && params.has("bytes") && Number.isSafeInteger(bytes) && bytes >= 0 ?
        <SourceMessagePreview key={`${rawSource}:${etag}`} selection={{ source_ref: rawSource, root_id: root, key, etag, byte_length: bytes }} /> :
        <p role="alert">Reopen this file from Sources to load its current details.</p>}
    </div>;
  }
  const decodedSource = params.get("decoded_source_ref");
  if (decodedSource) return <div className="space-y-4 p-4">
    <header className="flex flex-wrap items-center justify-between gap-3">
      <div><h1 className="text-xl font-semibold">Messages</h1>
        <p className="mt-1 break-all text-sm text-muted-foreground">{decodedSource.split("/").at(-1)}</p></div>
      <AppLink href={`/sources?${new URLSearchParams({ root: params.get("root") ?? "", file: decodedSource.split("/").at(-1) ?? "", mode: params.get("mode") ?? "LIVE" })}`} className="text-sm underline underline-offset-4">Back to Sources</AppLink>
    </header>
    <DecodedSourceViewer key={decodedSource} sourceRef={decodedSource} />
  </div>;
  const contextWorkflow = params.get("context_workflow");
  if (contextWorkflow) return <div className="space-y-3 p-4">
    <AppLink className="inline-block text-sm underline" href={`/read?${new URLSearchParams({ package: contextWorkflow, mode })}`}>Open full extracted package</AppLink>
    <AICandidateReview key={contextWorkflow} workflowId={contextWorkflow} />
  </div>;
  if (!isProcessingPreview(params)) return params.has("source") || params.has("thread") || params.has("q") ? <ReadWorkspace /> : <CategoryLanding />;
  return (
    <div className="space-y-4 p-4">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div><h1 className="text-xl font-semibold">Context review</h1><p className="text-sm text-muted-foreground">Processing previews</p></div>
        <AppLink href={importedReadingHref(params)} className="text-sm underline underline-offset-4">Back to imported reading</AppLink>
      </header>
      {nativeResource && <AppLink className="inline-block text-sm underline" href={`/read?${new URLSearchParams({ package: nativeResource, mode })}`}>Open full extracted package</AppLink>}
      <ProfferPreviewClient />
    </div>
  );
}
