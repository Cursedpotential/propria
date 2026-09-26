// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { legalApiBase } from "@/lib/api/client";

export function DraftEditor({
  sectionId,
  heading,
  body,
}: {
  sectionId: string;
  heading: string;
  body: string;
}) {
  const router = useRouter();
  const [title, setTitle] = useState(heading);
  const [text, setText] = useState(body);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function save() {
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(`${legalApiBase()}/v1/drafts/${sectionId}`, {
        method: "PUT",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ heading: title, body: text }),
      });
      if (!response.ok) throw new Error(`draft ${response.status}`);
      router.refresh();
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "save failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        void save();
      }}
      style={{ display: "grid", gap: 8, marginTop: 12 }}
    >
      <label>Section heading
      <input
        value={title}
        onChange={(event) => setTitle(event.target.value)}
        style={{ width: "100%" }}
      />
      </label>
      <label>Draft text
      <textarea
        value={text}
        onChange={(event) => setText(event.target.value)}
        rows={10}
        style={{ width: "100%", fontFamily: "Georgia, serif" }}
      />
      </label>
      {error ? <p style={{ color: "var(--status-warn)" }}>{error}</p> : null}
      <button type="submit" disabled={busy || !title.trim()}>
        {busy ? "Saving…" : "Save draft"}
      </button>
    </form>
  );
}
