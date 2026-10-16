import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { getHumanContextSourceStatus } from "@/lib/api-client";
import { humanPreviewHandle } from "@/lib/human-context";
import { AppLink } from "@/lib/router-compat";
import { attemptReadHref } from "@/lib/workflow-links";
import type { MatterMode } from "@/lib/shared/types";

/** Track one submitted human source and open SBV only with its real registration identity.
 * Inputs: returned workflow ID, filename and display mode. Outputs: status and native reader link.
 * Effects: actor-bound status GETs. Choose for normal SMS Process, without legacy handler retry or fabricated handles.
 */
export function HumanProcessStatus({ workflowID, name, mode }: { workflowID: string; name: string; mode: MatterMode }) {
  const status = useQuery({
    queryKey: ["sources", "human-context", workflowID, mode],
    queryFn: async ({ signal }) => {
      const result = await getHumanContextSourceStatus(workflowID, signal);
      if (result.workflow_id !== workflowID) throw new Error("Processing status returned a different workflow.");
      return result;
    },
    retry: false, gcTime: 0,
    refetchInterval: (query) => query.state.error || ["complete", "partial", "partial_enrichment", "needs_parser", "failed"].includes(query.state.data?.status ?? "") ? false : 3000,
  });
  const handle = humanPreviewHandle(status.data?.source_version_ref);
  return <div className="flex flex-wrap items-center gap-2 text-xs" role="status">
    <span>{name}: {status.data?.status ?? "Submitted"}{status.data?.reason ? ` · ${status.data.reason}` : ""}</span>
    {handle && <Button asChild size="sm" variant="outline"><AppLink href={attemptReadHref(handle, mode)}>View processed messages</AppLink></Button>}
    {status.error && <><span role="alert" className="text-destructive">{status.error.message}</span>
      <Button size="sm" variant="outline" onClick={() => void status.refetch()}>Retry status</Button></>}
  </div>;
}
