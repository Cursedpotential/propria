import { useQuery } from "@tanstack/react-query";
import { MessageBrowser } from "@/components/sbv/message-browser";
import { Button } from "@/components/ui/button";
import { getHumanContextPreview } from "@/lib/api-client";
import { humanPreviewHandle } from "@/lib/human-context";
import type { MatterMode } from "@/lib/shared/types";

/** Bind a human source snapshot to the existing SBV browser without legacy preview gates.
 * Inputs: real human source handle and display mode. Outputs: source identity, actual stage status and existing message renderer.
 * Effects: abortable metadata reads and child page reads. Choose for native human bundles; no decision or promotion actions.
 */
export function HumanContextPreview({ handle, mode }: { handle: string; mode: MatterMode }) {
  const snapshot = useQuery({
    queryKey: ["human-context", "snapshot", handle, mode],
    queryFn: async ({ signal }) => {
      const result = await getHumanContextPreview(handle, signal);
      if (result.preview_handle !== handle || humanPreviewHandle(result.source_version_ref) !== handle) {
        throw new Error("The native preview returned a different source version.");
      }
      return result;
    },
    retry: false, gcTime: 0,
    refetchInterval: (query) => query.state.status === "error" || query.state.data?.stages.some((stage) => stage.stage === "verified" && ["complete", "partial"].includes(stage.status)) ? false : 5000,
  });
  return <section className="space-y-3" aria-label="Native human message preview">
    <header className="flex flex-wrap items-center justify-between gap-2">
      <div><h2 className="text-base font-semibold">{snapshot.data?.source_ref.split("/").at(-1) ?? "Messages"}</h2>
        <p className="text-xs text-muted-foreground">{snapshot.data?.manifest_ref ? "Processed source messages" : "Waiting for extracted messages…"}</p></div>
      <Button size="sm" variant="outline" disabled={snapshot.isFetching} onClick={() => void snapshot.refetch()}>Refresh processing status</Button>
    </header>
    {snapshot.error && <p role="alert" className="text-sm text-destructive">{snapshot.error.message}</p>}
    {snapshot.data && <details className="rounded border p-3 text-xs">
      <summary className="cursor-pointer">Source and processing details</summary>
      <dl className="mt-2 space-y-1 break-all"><dt>Source</dt><dd>{snapshot.data.source_ref}</dd>
        <dt>Source version</dt><dd>{snapshot.data.source_version_ref}</dd>
        <dt>Workflow</dt><dd>{snapshot.data.workflow_id}</dd>
        {snapshot.data.manifest_ref && <><dt>Manifest</dt><dd>{snapshot.data.manifest_ref}</dd></>}</dl>
      <ul className="mt-3 space-y-1">{snapshot.data.stages.map((stage, index) => <li key={`${stage.stage}:${index}`}>
        {stage.stage}: {stage.status} · {stage.records} records · {stage.pending} pending · {stage.published} published · {stage.verified} verified
      </li>)}</ul>
    </details>}
    {snapshot.data?.manifest_ref && <MessageBrowser key={handle} previewHandle={handle} mode={mode} packageProjection={null} nativeManifestRef={snapshot.data.manifest_ref} />}
  </section>;
}
