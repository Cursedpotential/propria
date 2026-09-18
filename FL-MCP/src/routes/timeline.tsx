// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { UnavailableNotice } from "@/components/ui/unavailable-notice";
import { TimelineList } from "@/components/timeline/timeline-list";
import { timelineQuery } from "@/lib/queries";
import { isUnavailable, TIMELINE_MODES, type TimelineMode } from "@/types/store";

// search-validation (TanStack Router best practice): every search param is
// schema-validated, never trusted as a bare string from the URL.
const timelineSearchSchema = z.object({
  mode: z.enum(TIMELINE_MODES).catch("merged"),
  knownBy: z.string().optional(),
});

export const Route = createFileRoute("/timeline")({
  validateSearch: timelineSearchSchema,
  loaderDeps: ({ search }) => ({ mode: search.mode, knownBy: search.knownBy }),
  loader: ({ context, deps }) => context.queryClient.ensureQueryData(timelineQuery(deps.mode, deps.knownBy)),
  component: TimelinePage,
});

function TimelinePage() {
  const { mode, knownBy } = Route.useSearch();
  const navigate = Route.useNavigate();
  const { data } = useSuspenseQuery(timelineQuery(mode, knownBy));

  if (isUnavailable(data)) return <UnavailableNotice reason={data.reason} />;

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <div className="flex overflow-hidden rounded-[var(--radius-sm)] border border-border-strong">
          {TIMELINE_MODES.map((m) => (
            <Button
              key={m}
              variant={m === mode ? "solid" : "ghost"}
              size="sm"
              className="rounded-none border-0"
              onClick={() => navigate({ search: (prev) => ({ ...prev, mode: m as TimelineMode }) })}
            >
              {m}
            </Button>
          ))}
        </div>
        <label className="flex items-center gap-2 text-xs text-text-tertiary">
          Known by
          <Input
            type="date"
            className="w-36"
            value={knownBy ?? ""}
            onChange={(e) => navigate({ search: (prev) => ({ ...prev, knownBy: e.target.value || undefined }) })}
          />
        </label>
      </div>
      <TimelineList entries={data.entries} />
    </div>
  );
}
