// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
"use client";

import { useState } from "react";
import { legalApiBase } from "@/lib/api/client";

type Marker = {
  kind: string;
  label: string;
  matched: string;
  excerpt: string;
};

type Scan = {
  method: string;
  source: string;
  section_id: string | null;
  hypothesized_markers: Marker[];
  kinds_hit: string[];
  court_safe: boolean;
  legal_conclusion: boolean;
  disclaimer: string;
};

export function PrivilegeScanForm({
  drafts,
}: {
  drafts: Array<{ section_id: string; heading: string }>;
}) {
  const [text, setText] = useState("");
  const [sectionId, setSectionId] = useState("");
  const [scan, setScan] = useState<Scan | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const payload: { text?: string; section_id?: string } = {};
      if (text.trim()) payload.text = text;
      else if (sectionId) payload.section_id = sectionId;
      const response = await fetch(`${legalApiBase()}/v1/privilege:scan`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (!response.ok) throw new Error(`privilege scan ${response.status}`);
      setScan((await response.json()) as Scan);
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "scan failed");
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
        Draft section (optional)
        <select
          value={sectionId}
          onChange={(event) => setSectionId(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        >
          <option value="">Paste text instead</option>
          {drafts.map((draft) => (
            <option key={draft.section_id} value={draft.section_id}>
              {draft.heading}
            </option>
          ))}
        </select>
      </label>
      <label>
        Text to scan
        <textarea
          value={text}
          onChange={(event) => setText(event.target.value)}
          rows={6}
          placeholder="Paste the text to check for sensitive information."
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      {error ? <p style={{ color: "var(--status-warn)" }}>{error}</p> : null}
      <button type="submit" disabled={busy || (!text.trim() && !sectionId)}>
        {busy ? "Scanning…" : "Check text"}
      </button>
      {scan ? (
        <article style={{ borderTop: "1px solid var(--border)", padding: "16px 0" }}>
          <strong>
            {scan.method} · {scan.source}
          </strong>
          <p className="muted">Keyword scan · Review matches</p>
          {scan.hypothesized_markers.length === 0 ? (
            <p>No keyword matches found.</p>
          ) : (
            <ul>
              {scan.hypothesized_markers.map((marker, index) => (
                <li key={`${marker.kind}-${marker.matched}-${index}`}>
                  [{marker.label}] “{marker.matched}” — {marker.excerpt}
                </li>
              ))}
            </ul>
          )}
        </article>
      ) : null}
    </form>
  );
}
