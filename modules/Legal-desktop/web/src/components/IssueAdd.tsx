"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { legalApiBase } from "@/lib/api/client";

// Byline: Grok · grok-4.6 · 2026-08-18

export function IssueAdd({ parentId }: { parentId: string }) {
  const router = useRouter();
  const [title, setTitle] = useState("");
  const [authority, setAuthority] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function add() {
    if (!title.trim()) return;
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(`${legalApiBase()}/v1/issues`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          title: title.trim(),
          governing_authority: authority.trim(),
          parent_id: parentId || null,
        }),
      });
      if (!response.ok) throw new Error(`issue ${response.status}`);
      setTitle("");
      setAuthority("");
      router.refresh();
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "save failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="work-sheet-bar">
      <input
        value={title}
        aria-label="New issue"
        placeholder="Add an issue, Enter to save"
        onChange={(event) => setTitle(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            void add();
          }
        }}
      />
      <input
        value={authority}
        aria-label="Governing authority"
        placeholder="Statute or case (optional)"
        onChange={(event) => setAuthority(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            void add();
          }
        }}
      />
      <button type="button" disabled={busy || !title.trim()} onClick={() => void add()}>
        {busy ? "Saving…" : "Add issue"}
      </button>
      {error ? <span className="unsupported">{error}</span> : null}
    </div>
  );
}
