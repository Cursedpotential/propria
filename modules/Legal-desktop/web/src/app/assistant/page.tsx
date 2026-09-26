"use client";

import { Suspense, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { ConfidentialFlag, readConfidential } from "@/components/ConfidentialFlag";
import { type LiveSurface, liveSurfaceKey } from "@/lib/liveSurface";
import { navHelp, navLabel, surfaceCatalog, surfaceForPath } from "@/lib/surfaces";

// Byline: Grok · grok-4.6 · 2026-08-18
// Cat 3: F1 from any surface. Live form text is sent; saved snapshot comes from legal-api.

function readStoredFrom(): string {
  if (typeof window === "undefined") return "/";
  try {
    return window.sessionStorage.getItem("lw-chat-from") || "/";
  } catch {
    return "/";
  }
}

function readStoredLive(): LiveSurface | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.sessionStorage.getItem(liveSurfaceKey());
    return raw ? (JSON.parse(raw) as LiveSurface) : null;
  } catch {
    return null;
  }
}

function ChatInner() {
  const params = useSearchParams();
  const [from, setFrom] = useState(params.get("from") || readStoredFrom());
  const [live, setLive] = useState<LiveSurface | null>(readStoredLive);
  const [prompt, setPrompt] = useState("");
  const [log, setLog] = useState<Array<{ role: string; content: string }>>([]);
  const [busy, setBusy] = useState(false);
  const surface = useMemo(
    () => surfaceForPath(from.split("?")[0] || "/", surfaceCatalog()),
    [from],
  );
  const unsavedCount = (live?.fields.length ?? 0) + (live?.selection ? 1 : 0);

  useEffect(() => {
    function onMessage(event: MessageEvent) {
      if (event.origin !== window.location.origin) return;
      const data = event.data as { source?: string; from?: string; live?: LiveSurface } | null;
      if (!data || data.source !== "lw-chat-context") return;
      if (data.from) setFrom(data.from);
      if (data.live) setLive(data.live);
    }
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, []);

  async function send() {
    if (!prompt.trim()) return;
    const next = [...log, { role: "user", content: prompt }];
    const confidential = readConfidential();
    setLog(next);
    setPrompt("");
    setBusy(true);
    try {
      const response = await fetch("/api/chat", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          messages: next,
          confidential,
          surface: {
            path: from,
            label: navLabel(surface),
            help: navHelp(surface),
          },
          live,
        }),
      });
      const body = (await response.json().catch(() => null)) as { content?: string; error?: string } | null;
      if (!response.ok) throw new Error(body?.error ?? `Assistant request failed (${response.status}).`);
      if (!body) throw new Error("The assistant returned an unreadable response. Try again.");
      setLog([...next, { role: "assistant", content: body.content ?? body.error ?? "no output" }]);
    } catch (exc) {
      setLog([...next, { role: "assistant", content: exc instanceof Error ? exc.message : "The assistant is unavailable. Try again." }]);
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <h1>Ask the assistant</h1>
      <p className="surface-chip" title={navHelp(surface)}>
        Looking at: <strong>{navLabel(surface)}</strong>
      </p>
      {unsavedCount > 0 ? (
        <p className="surface-chip live">
          Also sending {unsavedCount} unsaved field{unsavedCount === 1 ? "" : "s"} from this screen.
        </p>
      ) : (
        <p className="muted">
          Nothing unsaved on this screen. Only what is already stored will be attached.
        </p>
      )}
      <p className="muted">Ask about the selected workspace context. F1 from any page opens this beside that context.</p>
      <ConfidentialFlag />
      {log.map((item, index) => (
        <article key={`${item.role}-${index}`}>
          <strong>{item.role === "user" ? "You" : "Assistant"}</strong>
          <p style={{ whiteSpace: "pre-wrap" }}>{item.content}</p>
        </article>
      ))}
      <form
        onSubmit={(event) => {
          event.preventDefault();
          void send();
        }}
        style={{ display: "grid", gap: 8, marginTop: 16 }}
      >
        <textarea
          value={prompt}
          onChange={(event) => setPrompt(event.target.value)}
          rows={3}
          placeholder={`Ask about ${navLabel(surface).toLowerCase()}`}
        />
        <button type="submit" disabled={busy || !prompt.trim()}>
          {busy ? "Working…" : "Send"}
        </button>
      </form>
    </>
  );
}

export default function ChatPage() {
  return (
    <Suspense fallback={<p className="muted">Opening assistant…</p>}>
      <ChatInner />
    </Suspense>
  );
}
