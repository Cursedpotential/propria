// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { EvidenceView } from "@/components/evidence/evidence-view";
import { UnavailableNotice } from "@/components/ui/unavailable-notice";
import { evidenceQuery } from "@/lib/queries";
import { isUnavailable } from "@/types/store";

export const Route = createFileRoute("/evidence")({
  loader: ({ context }) => context.queryClient.ensureQueryData(evidenceQuery()),
  component: EvidencePage,
});

function EvidencePage() {
  const { data } = useSuspenseQuery(evidenceQuery());
  if (isUnavailable(data)) return <UnavailableNotice reason={data.reason} />;
  return <EvidenceView data={data} />;
}
