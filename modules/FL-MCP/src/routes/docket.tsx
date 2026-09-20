// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { DocketGrid } from "@/components/docket/docket-grid";
import { UnavailableNotice } from "@/components/ui/unavailable-notice";
import { docketQuery } from "@/lib/queries";
import { isUnavailable } from "@/types/store";

export const Route = createFileRoute("/docket")({
  loader: ({ context }) => context.queryClient.ensureQueryData(docketQuery()),
  component: DocketPage,
});

function DocketPage() {
  const { data } = useSuspenseQuery(docketQuery());

  if (isUnavailable(data)) {
    return <UnavailableNotice reason={data.reason} />;
  }

  return <DocketGrid entries={data.entries} />;
}
