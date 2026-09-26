// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
"use client";

import { useState } from "react";
import { legalApiBase } from "@/lib/api/client";

type Hit = {
  id: string;
  name: string;
  citation: string;
  court: string;
  date: string;
  url: string;
};

export function SourceSearch() {
  const [query, setQuery] = useState("");
  const [hits, setHits] = useState<Hit[]>([]);
  const [reason, setReason] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function search() {
    setBusy(true);
    setReason(null);
    try {
      const response = await fetch(
        `${legalApiBase()}/v1/sources/courtlistener/search?q=${encodeURIComponent(query)}`,
        { cache: "no-store" },
      );
      const body = (await response.json()) as { ok?: boolean; hits?: Hit[]; reason?: string };
      setHits(body.hits ?? []);
      if (!body.ok) setReason(body.reason ?? `search ${response.status}`);
    } catch (exc) {
      setReason(exc instanceof Error ? exc.message : "search failed");
      setHits([]);
    } finally {
      setBusy(false);
    }
  }

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        void search();
      }}
      style={{ display: "grid", gap: 8, margin: "16px 0 24px" }}
    >
      <label>
        CourtListener case search
        <input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Vodvarka Michigan custody"
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <button type="submit" disabled={busy || !query.trim()}>
        {busy ? "Searching…" : "Search CourtListener"}
      </button>
      {reason ? <p style={{ color: "var(--status-warn)" }}>{reason}</p> : null}
      {hits.map((hit) => (
        <article key={`${hit.id}-${hit.url}`} style={{ borderTop: "1px solid var(--border)", padding: "8px 0" }}>
          <strong>{hit.name || hit.citation || hit.id}</strong>
          <p className="dim">
            {hit.citation} · {hit.court} · {hit.date}
          </p>
          {hit.url ? (
            <p className="muted">
              <a href={hit.url.startsWith("http") ? hit.url : `https://www.courtlistener.com${hit.url}`}>
                {hit.url}
              </a>
            </p>
          ) : null}
        </article>
      ))}
    </form>
  );
}
