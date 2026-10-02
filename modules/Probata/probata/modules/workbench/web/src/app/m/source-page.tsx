// Byline: Claude Code · Sonnet · 2026-10-02
import { useParams } from "@tanstack/react-router";

import { SourceThreadsView } from "@/components/mobile/imported-views";

export default function MobileSourcePage() {
  const { sourceId } = useParams({ strict: false }) as { sourceId?: string };
  return sourceId ? <SourceThreadsView sourceId={sourceId} /> : null;
}
