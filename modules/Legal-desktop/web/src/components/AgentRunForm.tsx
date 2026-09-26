// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { legalApiBase } from "@/lib/api/client";

export function AgentRunForm() {
  const router = useRouter();
  const [intent, setIntent] = useState("");
  const [prompt, setPrompt] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(`${legalApiBase()}/v1/agent-runs`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          intent,
          prompt,
          requested_model: "unevaluated-manual",
        }),
      });
      if (!response.ok) throw new Error(`agent-run ${response.status}`);
      setIntent("");
      setPrompt("");
      router.refresh();
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "agent run failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        void submit();
      }}
      style={{ display: "grid", gap: 10, margin: "16px 0 32px" }}
    >
      <label>
        Intent
        <input
          value={intent}
          onChange={(event) => setIntent(event.target.value)}
          required
          placeholder="e.g. research Vodvarka lookback"
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Prompt / notes
        <textarea
          value={prompt}
          onChange={(event) => setPrompt(event.target.value)}
          required
          rows={3}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      {error ? <p style={{ color: "var(--status-warn)" }}>{error}</p> : null}
      <button type="submit" disabled={busy || !intent.trim()}>
        {busy ? "Recording…" : "Record routed run"}
      </button>
    </form>
  );
}
