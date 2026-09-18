// Byline: Claude Code · Sonnet 5 · 2026-09-07
//
// Generic "form that asks Claude to run a plugin console tool" card. Per
// the owner spec, the Survival Guide and Court-Safe Language forms call the
// console tools THROUGH the chat/agent path (not a direct sidecar REST
// endpoint) — the prompt is crafted to name the tool explicitly so the
// model reliably reaches for it rather than answering from general
// knowledge.
import { Send } from "lucide-react";
import * as React from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useChat, type ChatTurn } from "@/hooks/use-chat";

export function ToolFormCard({
  title,
  description,
  placeholder,
  buildPrompt,
}: {
  title: string;
  description: string;
  placeholder: string;
  buildPrompt: (input: string) => string;
}) {
  const [input, setInput] = React.useState("");
  const { turns, sending, send } = useChat();
  const lastAssistant = [...turns].reverse().find((t) => t.role === "assistant");

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!input.trim()) return;
    void send(buildPrompt(input));
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <CardDescription>{description}</CardDescription>
        <form onSubmit={onSubmit} className="flex items-start gap-2">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder={placeholder}
            rows={3}
            className="h-20 flex-1 rounded-[var(--radius-sm)] border border-border-strong bg-bg-inset px-2.5 py-1.5 text-sm text-text-primary placeholder:text-text-tertiary"
          />
          <Button type="submit" variant="solid" disabled={sending || !input.trim()}>
            <Send className="size-3.5" /> Run
          </Button>
        </form>
        {lastAssistant && <ToolResultView turn={lastAssistant} />}
      </CardContent>
    </Card>
  );
}

function ToolResultView({ turn }: { turn: ChatTurn }) {
  return (
    <div className="rounded-[var(--radius-md)] border border-border bg-surface-raised p-3 text-sm">
      {turn.pending && turn.blocks.length === 0 && <span className="text-text-tertiary">Working…</span>}
      {turn.blocks.map((block, i) => {
        if (block.kind === "text") return <p key={i} className="whitespace-pre-wrap">{block.text}</p>;
        if (block.kind === "tool_use") {
          return (
            <div key={i} className="mb-1 font-mono text-xs text-text-tertiary">
              → {block.name}
            </div>
          );
        }
        return null;
      })}
      {turn.error && <p className="text-critical-text">{turn.error}</p>}
    </div>
  );
}
