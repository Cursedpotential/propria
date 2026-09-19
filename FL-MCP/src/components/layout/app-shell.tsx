// Byline: Claude Code · Sonnet 5 · 2026-09-07
// Updated by: OpenAI Codex · GPT-5 · 2026-09-12 — shared shell and bounded context strip.
import { Outlet } from "@tanstack/react-router";
import { ChatDrawer } from "@/components/chat/chat-drawer";
import { ContextStrip } from "./context-strip";
import { Header } from "./header";
import { Nav } from "./nav";

export function AppShell() {
  return (
    <div className="pr-app flex h-screen w-screen overflow-hidden bg-bg text-text-primary max-[720px]:flex-col" data-pr-experience="general">
      <Nav />
      <div className="flex min-w-0 flex-1 flex-col">
        <Header />
        <ContextStrip />
        <main className="min-h-0 flex-1 overflow-y-auto p-4">
          <Outlet />
        </main>
      </div>
      <ChatDrawer />
    </div>
  );
}
