// Byline: Codex · GPT-6 · 2026-10-06
import { AICandidateReview } from "@/components/review/ai-candidate-review";
import { ReadWorkspace } from "@/components/read/read-workspace";
import { importedReadingHref, isProcessingPreview } from "@/components/read/read-location";
import { ProfferPreviewClient } from "@/components/sbv/proffer-preview-client";
import { AppLink, useBrowserSearchParams } from "@/lib/router-compat";

/** Render imported reading or the existing processing preview selected by the URL.
 * Input: source/thread/around/q or preview identifiers from the browser route.
 * Output: the Read workspace; effects: existing child queries and explicit preview actions.
 * Pick as /read and the destination of legacy review/preview redirects.
 */
export default function ReadPage() {
  const params = useBrowserSearchParams();
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
