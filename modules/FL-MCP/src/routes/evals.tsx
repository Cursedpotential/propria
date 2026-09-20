// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { EvalsView } from "@/components/evals/evals-view";
import { UnavailableNotice } from "@/components/ui/unavailable-notice";
import { evalsQuery } from "@/lib/queries";
import { isUnavailable } from "@/types/store";

export const Route = createFileRoute("/evals")({
  loader: ({ context }) => context.queryClient.ensureQueryData(evalsQuery()),
  component: EvalsPage,
});

function EvalsPage() {
  const { data } = useSuspenseQuery(evalsQuery());
  if (isUnavailable(data)) return <UnavailableNotice reason={data.reason} />;
  return <EvalsView evals={data.evals} />;
}
