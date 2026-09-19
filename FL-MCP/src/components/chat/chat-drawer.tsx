// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { MessageSquare, X } from "lucide-react";
import * as React from "react";
import { Button } from "@/components/ui/button";
import { ChatPanel } from "./chat-panel";

/** Persistent right-side drawer available from every route (owner spec: "/chat ... persistent across routes as a right-side drawer too"). */
export function ChatDrawer() {
  const [open, setOpen] = React.useState(false);

  return (
    <>
      <Button
        variant="solid"
        size="icon"
        className="fixed bottom-4 right-4 z-40 shadow-none"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        aria-controls="chat-drawer-panel"
        aria-label="Toggle Claude chat"
      >
        <MessageSquare className="size-4" />
      </Button>

      {open && (
        <div
          id="chat-drawer-panel"
          role="complementary"
          aria-label="Claude chat"
          className="fixed bottom-16 right-4 z-40 h-[32rem] w-96 overflow-hidden rounded-[var(--radius-md)] border border-border bg-surface"
        >
          <div className="flex h-8 items-center justify-between border-b border-border px-2">
            <span className="text-xs font-medium text-text-secondary">Claude</span>
            <Button variant="ghost" size="icon" className="size-6" onClick={() => setOpen(false)} aria-label="Close chat">
              <X className="size-3.5" />
            </Button>
          </div>
          <ChatPanel className="h-[calc(100%-2rem)]" />
        </div>
      )}
    </>
  );
}
