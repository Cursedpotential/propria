"use client";

import { useState } from "react";
import { legalApiBase } from "@/lib/api/client";

// Byline: Grok · grok-4.6 · 2026-08-18

type Playbook = {
  playbook_id: string;
  title: string;
  steps: string[];
  note?: string;
};

export function PlaybookRun({ playbooks }: { playbooks: Playbook[] }) {
  const [busy, setBusy] = useState<string | null>(null);
  const [result, setResult] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function run(id: string) {
    setBusy(id);
    setError(null);
    try {
      const response = await fetch(
        `${legalApiBase()}/v1/automations/playbooks/${id}:run`,
        { method: "POST" },
      );
      const body = await response.json();
      if (!response.ok) throw new Error(body.detail ?? `run ${response.status}`);
      setResult(JSON.stringify(body, null, 2));
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "run failed");
    } finally {
      setBusy(null);
    }
  }

  return (
    <>
      {playbooks.map((item) => (
        <article key={item.playbook_id} className="surface-card">
          <h2>{item.title}</h2>
          <p className="muted">{item.note}</p>
          <ol>
            {item.steps.map((step) => (
              <li key={step}>{step}</li>
            ))}
          </ol>
          <button type="button" disabled={busy === item.playbook_id} onClick={() => void run(item.playbook_id)}>
            {busy === item.playbook_id ? "Running…" : "Run now"}
          </button>
        </article>
      ))}
      {error ? <p className="unsupported">{error}</p> : null}
      {result ? <pre className="draft-pre">{result}</pre> : null}
    </>
  );
}
