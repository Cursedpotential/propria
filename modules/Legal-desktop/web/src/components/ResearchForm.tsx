// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { legalApiBase } from "@/lib/api/client";

export function ResearchForm() {
  const router = useRouter();
  const [question, setQuestion] = useState("");
  const [plan, setPlan] = useState("");
  const [uncertainty, setUncertainty] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(`${legalApiBase()}/v1/research`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          question,
          plan,
          uncertainty,
          jurisdiction: "US-MI",
        }),
      });
      if (!response.ok) throw new Error(`research ${response.status}`);
      setQuestion("");
      setPlan("");
      setUncertainty("");
      router.refresh();
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "research failed");
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
        Research question (required)
        <input
          value={question}
          onChange={(event) => setQuestion(event.target.value)}
          required
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Research plan (required)
        <textarea
          value={plan}
          onChange={(event) => setPlan(event.target.value)}
          required
          rows={3}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Open uncertainty (optional)
        <textarea
          value={uncertainty}
          onChange={(event) => setUncertainty(event.target.value)}
          rows={2}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      {error ? <p style={{ color: "var(--status-warn)" }}>{error}</p> : null}
      <button type="submit" disabled={busy || !question.trim() || !plan.trim()}>
        {busy ? "Saving…" : "Save research question"}
      </button>
    </form>
  );
}
