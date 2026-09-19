// Byline: Claude Code · Sonnet 5 · 2026-09-07
import * as React from "react";
import { ChatAuthError, streamChat } from "@/lib/chat-client";

export type ChatBlock =
  | { kind: "text"; text: string }
  | { kind: "tool_use"; id: string; name: string; input: unknown }
  | { kind: "tool_result"; tool_use_id: string; content: unknown };

export interface ChatTurn {
  id: string;
  role: "user" | "assistant";
  blocks: ChatBlock[];
  pending?: boolean;
  error?: string;
}

function newId() {
  return typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : String(Math.random());
}

export function useChat() {
  const [turns, setTurns] = React.useState<ChatTurn[]>([]);
  const [sending, setSending] = React.useState(false);
  const [authFix, setAuthFix] = React.useState<string | null>(null);
  const sessionIdRef = React.useRef<string | undefined>(undefined);
  const abortRef = React.useRef<AbortController | null>(null);

  const send = React.useCallback(async (prompt: string) => {
    if (!prompt.trim() || sending) return;
    setAuthFix(null);
    const userTurn: ChatTurn = { id: newId(), role: "user", blocks: [{ kind: "text", text: prompt }] };
    const assistantTurn: ChatTurn = { id: newId(), role: "assistant", blocks: [], pending: true };
    setTurns((prev) => [...prev, userTurn, assistantTurn]);
    setSending(true);

    const controller = new AbortController();
    abortRef.current = controller;

    function appendBlock(block: ChatBlock) {
      setTurns((prev) => prev.map((t) => (t.id === assistantTurn.id ? { ...t, blocks: [...t.blocks, block] } : t)));
    }

    try {
      await streamChat({
        prompt,
        sessionId: sessionIdRef.current,
        signal: controller.signal,
        onEvent: ({ event, data }) => {
          switch (event) {
            case "session_start":
              sessionIdRef.current = (data as { session_id?: string }).session_id;
              break;
            case "text":
              appendBlock({ kind: "text", text: (data as { text: string }).text });
              break;
            case "tool_use":
              appendBlock({ kind: "tool_use", id: (data as { id: string }).id, name: (data as { name: string }).name, input: (data as { input: unknown }).input });
              break;
            case "tool_result":
              appendBlock({ kind: "tool_result", tool_use_id: (data as { tool_use_id: string }).tool_use_id, content: (data as { content: unknown }).content });
              break;
            case "error":
            case "model_error":
              setTurns((prev) => prev.map((t) => (t.id === assistantTurn.id ? { ...t, error: JSON.stringify(data) } : t)));
              break;
            default:
              break;
          }
        },
      });
    } catch (err) {
      if (err instanceof ChatAuthError) {
        setAuthFix(err.fix ?? err.message);
      } else if (!(err instanceof DOMException && err.name === "AbortError")) {
        setTurns((prev) => prev.map((t) => (t.id === assistantTurn.id ? { ...t, error: err instanceof Error ? err.message : String(err) } : t)));
      }
    } finally {
      setTurns((prev) => prev.map((t) => (t.id === assistantTurn.id ? { ...t, pending: false } : t)));
      setSending(false);
      abortRef.current = null;
    }
  }, [sending]);

  const cancel = React.useCallback(() => {
    abortRef.current?.abort();
  }, []);

  return { turns, sending, send, cancel, authFix };
}
