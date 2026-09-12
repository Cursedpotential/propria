// Byline: Claude Code · Sonnet 5 · 2026-09-07
//
// Minimal hand-rolled SSE reader for POST /api/chat. Not using EventSource
// because it only supports GET; the sidecar's chat endpoint is a POST with
// a JSON body (the prompt), so we parse the `text/event-stream` body of a
// `fetch()` response by hand.

import { resolveSidecarBaseUrl } from "./sidecar";

export interface ChatEvent {
  event: string;
  data: unknown;
}

export interface StreamChatOptions {
  prompt: string;
  sessionId?: string;
  signal?: AbortSignal;
  onEvent: (event: ChatEvent) => void;
}

/** Returns a rejected-with-401 style error object when the sidecar reports no auth. */
export class ChatAuthError extends Error {
  fix?: string;
  constructor(message: string, fix?: string) {
    super(message);
    this.name = "ChatAuthError";
    this.fix = fix;
  }
}

export async function streamChat({ prompt, sessionId, signal, onEvent }: StreamChatOptions): Promise<void> {
  const base = await resolveSidecarBaseUrl();
  const res = await fetch(`${base}/api/chat`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ prompt, sessionId }),
    signal,
  });

  if (res.status === 401) {
    const body = await res.json().catch(() => ({}));
    throw new ChatAuthError(body.fix ?? "Not authenticated with Claude.", body.fix);
  }
  if (!res.ok || !res.body) {
    throw new Error(`Chat request failed: ${res.status} ${res.statusText}`);
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  for (;;) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });

    let sepIndex: number;
    // SSE frames are separated by a blank line ("\n\n").
    while ((sepIndex = buffer.indexOf("\n\n")) !== -1) {
      const frame = buffer.slice(0, sepIndex);
      buffer = buffer.slice(sepIndex + 2);

      let eventName = "message";
      let dataLine = "";
      for (const line of frame.split("\n")) {
        if (line.startsWith("event:")) eventName = line.slice(6).trim();
        else if (line.startsWith("data:")) dataLine += line.slice(5).trim();
      }
      if (!dataLine) continue;
      try {
        onEvent({ event: eventName, data: JSON.parse(dataLine) });
      } catch {
        onEvent({ event: eventName, data: dataLine });
      }
    }
  }
}
