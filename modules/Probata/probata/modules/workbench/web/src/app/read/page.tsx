// Byline: Codex · GPT-6 · 2026-10-06
import { AICandidateReview } from "@/components/review/ai-candidate-review";
import { ReadWorkspace } from "@/components/read/read-workspace";
import { importedReadingHref, isProcessingPreview } from "@/components/read/read-location";
import { ProfferPreviewClient } from "@/components/sbv/proffer-preview-client";
import { DecodedSourceViewer } from "@/components/sbv/decoded-source-viewer";
import { SourceMessagePreview } from "@/components/sbv/source-message-preview";
import { AppLink, useBrowserSearchParams } from "@/lib/router-compat";

/** Render imported reading or the existing processing preview selected by the URL.
 * Input: source/thread/around/q or preview identifiers from the browser route.
 * Output: the Read workspace; effects: existing child queries and explicit preview actions.
 * Pick as /read and the destination of legacy review/preview redirects.
 */
export default function ReadPage() {
  const params = useBrowserSearchParams();
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
  if (contextWorkflow) return <div className="space-y-4 p-4">
    <header className="flex items-center justify-between gap-3"><h1 className="text-xl font-semibold">Read</h1>
      <AppLink href="/sources" className="text-sm underline underline-offset-4">Back to Sources</AppLink></header>
    <AICandidateReview key={contextWorkflow} workflowId={contextWorkflow} />
  </div>;
  if (!isProcessingPreview(params)) return <ReadWorkspace />;
  return (
    <div className="space-y-4 p-4">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div><h1 className="text-xl font-semibold">Read</h1><p className="text-sm text-muted-foreground">Processing previews</p></div>
        <AppLink href={importedReadingHref(params)} className="text-sm underline underline-offset-4">Back to imported reading</AppLink>
      </header>
      <ProfferPreviewClient />
    </div>
  );
}
