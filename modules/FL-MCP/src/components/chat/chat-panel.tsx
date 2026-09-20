// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { CornerDownLeft, Square, Wrench } from "lucide-react";
import * as React from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useChat, type ChatTurn } from "@/hooks/use-chat";

export function ChatPanel({ className }: { className?: string }) {
  const { turns, sending, send, cancel, authFix } = useChat();
  const [draft, setDraft] = React.useState("");
  const scrollRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight });
  }, [turns]);

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    const prompt = draft;
    setDraft("");
    void send(prompt);
  }

  return (
    <div className={"flex h-full flex-col " + (className ?? "")}>
      <div ref={scrollRef} className="flex-1 space-y-3 overflow-y-auto px-3 py-3">
        {turns.length === 0 && (
          <p className="text-sm text-text-tertiary">
            Ask about the case docket, timeline, or reference material. This pane calls the Agent SDK with access only to the
            family-court-console MCP tools and Read.
          </p>
        )}
        {turns.map((turn) => (
          <TurnView key={turn.id} turn={turn} />
        ))}
        {authFix && (
          <div className="rounded-[var(--radius-md)] border border-warn-border bg-warn-fill px-3 py-2 text-xs text-warn-text">
            Not authenticated: {authFix}
          </div>
        )}
      </div>

      <form onSubmit={onSubmit} className="flex items-center gap-2 border-t border-border p-2">
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder="Ask the console…"
          className="h-8 flex-1 rounded-[var(--radius-sm)] border border-border-strong bg-bg-inset px-2.5 text-sm text-text-primary placeholder:text-text-tertiary"
        />
        {sending ? (
          <Button type="button" variant="outline" size="icon" onClick={cancel} aria-label="Stop">
            <Square className="size-3.5" />
          </Button>
        ) : (
          <Button type="submit" variant="solid" size="icon" aria-label="Send" disabled={!draft.trim()}>
            <CornerDownLeft className="size-3.5" />
          </Button>
        )}
      </form>
    </div>
  );
}

function TurnView({ turn }: { turn: ChatTurn }) {
  const isUser = turn.role === "user";
  return (
    <div className={"flex flex-col gap-1 " + (isUser ? "items-end" : "items-start")}>
      <div
        className={
          "max-w-[92%] rounded-[var(--radius-md)] border px-3 py-2 text-sm " +
          (isUser ? "border-accent-border bg-accent-fill text-text-primary" : "border-border bg-surface-raised text-text-primary")
        }
      >
        {turn.blocks.map((block, i) => {
          if (block.kind === "text") return <p key={i} className="whitespace-pre-wrap">{block.text}</p>;
          if (block.kind === "tool_use") {
            return (
              <div key={i} className="mt-1 flex items-center gap-1.5 rounded-[var(--radius-sm)] border border-border-strong bg-bg-inset px-2 py-1 font-mono text-xs text-text-secondary">
                <Wrench className="size-3 shrink-0" aria-hidden />
                {block.name}
              </div>
            );
          }
          return (
            <details key={i} className="mt-1 rounded-[var(--radius-sm)] border border-border-strong bg-bg-inset px-2 py-1 text-xs text-text-tertiary">
              <summary className="cursor-pointer select-none font-mono">tool_result</summary>
              <pre className="mt-1 max-h-40 overflow-auto whitespace-pre-wrap font-mono">{JSON.stringify(block.content, null, 2)}</pre>
            </details>
          );
        })}
        {turn.pending && turn.blocks.length === 0 && <span className="text-text-tertiary">…</span>}
        {turn.error && <Badge tone="critical" className="mt-1">{turn.error}</Badge>}
      </div>
    </div>
  );
}
