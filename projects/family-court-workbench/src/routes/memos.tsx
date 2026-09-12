// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { MemosView } from "@/components/memos/memos-view";
import { UnavailableNotice } from "@/components/ui/unavailable-notice";
import { memosQuery } from "@/lib/queries";
import { isUnavailable } from "@/types/store";

export const Route = createFileRoute("/memos")({
  loader: ({ context }) => context.queryClient.ensureQueryData(memosQuery()),
  component: MemosPage,
});

function MemosPage() {
  const { data } = useSuspenseQuery(memosQuery());
  if (isUnavailable(data)) return <UnavailableNotice reason={data.reason} />;
  return <MemosView memos={data.memos} />;
}
