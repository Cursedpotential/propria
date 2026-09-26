// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
"use client";

// Byline: Grok · grok-4.6 · 2026-08-18

import { useRouter } from "next/navigation";
import { useState } from "react";
import { legalApiBase } from "@/lib/api/client";

export function InvestigationForm() {
  const router = useRouter();
  const [kind, setKind] = useState("missing_proof");
  const [needed, setNeeded] = useState("");
  const [why, setWhy] = useState("");
  const [linkedIssue, setLinkedIssue] = useState("");
  const [factorLetter, setFactorLetter] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(`${legalApiBase()}/v1/investigations`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          kind,
          needed,
          why,
          linked_issue: linkedIssue,
          factor_letter: factorLetter || null,
        }),
      });
      if (!response.ok) throw new Error(`investigation ${response.status}`);
      setNeeded("");
      setWhy("");
      setLinkedIssue("");
      setFactorLetter("");
      router.refresh();
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "investigation failed");
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
        Request type
        <select
          value={kind}
          onChange={(event) => setKind(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        >
          <option value="missing_proof">Missing proof</option>
          <option value="contradiction">Contradiction</option>
        </select>
      </label>
      <label>
        Proof needed (required)
        <textarea
          value={needed}
          onChange={(event) => setNeeded(event.target.value)}
          required
          rows={3}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Why it is needed (required)
        <textarea
          value={why}
          onChange={(event) => setWhy(event.target.value)}
          required
          rows={2}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Linked issue or element
        <input
          value={linkedIssue}
          onChange={(event) => setLinkedIssue(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Factor letter (optional)
        <input
          value={factorLetter}
          onChange={(event) => setFactorLetter(event.target.value)}
          placeholder="j"
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      {error ? <p style={{ color: "var(--status-warn)" }}>{error}</p> : null}
      <button type="submit" disabled={busy || !needed.trim() || !why.trim()}>
        {busy ? "Saving…" : "Save investigation request"}
      </button>
    </form>
  );
}
