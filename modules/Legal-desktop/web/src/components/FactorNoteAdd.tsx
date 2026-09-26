"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { legalApiBase } from "@/lib/api/client";

// Byline: Grok · grok-4.6 · 2026-08-18
// One composer. Pick the factor, pick the bucket, type the note.

export function FactorNoteAdd({
  factors,
}: {
  factors: Array<{ letter: string; title: string }>;
}) {
  const router = useRouter();
  const [letter, setLetter] = useState(factors[0]?.letter ?? "a");
  const [side, setSide] = useState("petitioner");
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function add() {
    if (!text.trim() || !letter) return;
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(`${legalApiBase()}/v1/factors/${letter}/notes`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ side, text: text.trim() }),
      });
      if (!response.ok) throw new Error(`note ${response.status}`);
      setText("");
      router.refresh();
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "save failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="work-sheet-bar factor-composer">
      <select
        value={letter}
        aria-label="Factor"
        onChange={(event) => setLetter(event.target.value)}
      >
        {factors.map((factor) => (
          <option key={factor.letter} value={factor.letter}>
            ({factor.letter}) {factor.title}
          </option>
        ))}
      </select>
      <select
        value={side}
        aria-label="Note category"
        onChange={(event) => setSide(event.target.value)}
      >
        <option value="petitioner">Petitioner</option>
        <option value="respondent">Respondent</option>
        <option value="contradiction">Contradiction</option>
        <option value="missing">Missing proof</option>
      </select>
      <input
        value={text}
        aria-label="Factor note"
        placeholder="Type a note and press Enter to save"
        onChange={(event) => setText(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            void add();
          }
        }}
      />
      <button type="button" disabled={busy || !text.trim()} onClick={() => void add()}>
        {busy ? "Saving…" : "Add note"}
      </button>
      {error ? <span className="unsupported">{error}</span> : null}
    </div>
  );
}
