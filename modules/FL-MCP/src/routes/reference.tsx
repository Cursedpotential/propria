// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { createFileRoute } from "@tanstack/react-router";
import { ReferenceView } from "@/components/reference/reference-view";
import { factorMapQuery, referenceLibraryQuery, referenceQuery } from "@/lib/queries";

export const Route = createFileRoute("/reference")({
  loader: ({ context }) => Promise.all([
    context.queryClient.ensureQueryData(factorMapQuery()).catch(() => undefined),
    context.queryClient.ensureQueryData(referenceQuery()).catch(() => undefined),
    context.queryClient.ensureQueryData(referenceLibraryQuery()).catch(() => undefined),
  ]),
  component: ReferenceView,
});
