// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { createFileRoute } from "@tanstack/react-router";
import { ChatPanel } from "@/components/chat/chat-panel";

export const Route = createFileRoute("/chat")({
  component: () => (
    <div className="h-full rounded-[var(--radius-md)] border border-border bg-surface">
      <ChatPanel className="h-full" />
    </div>
  ),
});
