// Byline: Claude Code · Sonnet · 2026-10-02
import { useParams } from "@tanstack/react-router";

import { ThreadView } from "@/components/mobile/imported-views";

export default function MobileThreadPage() {
  const { threadId } = useParams({ strict: false }) as { threadId?: string };
  return threadId ? <ThreadView threadId={threadId} /> : null;
}
