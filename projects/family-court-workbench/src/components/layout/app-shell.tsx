// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { Outlet } from "@tanstack/react-router";
import { ChatDrawer } from "@/components/chat/chat-drawer";
import { Header } from "./header";
import { Nav } from "./nav";

export function AppShell() {
  return (
    <div className="flex h-screen w-screen overflow-hidden bg-bg text-text-primary">
      <Nav />
      <div className="flex min-w-0 flex-1 flex-col">
        <Header />
        <main className="min-h-0 flex-1 overflow-y-auto p-4">
          <Outlet />
        </main>
      </div>
      <ChatDrawer />
    </div>
  );
}
