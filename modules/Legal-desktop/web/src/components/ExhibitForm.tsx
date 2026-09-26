// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { legalApiBase } from "@/lib/api/client";

export function ExhibitForm({
  candidates,
}: {
  candidates: Array<{ item_id: string; span_locator: string; exhibit_label: string }>;
}) {
  const router = useRouter();
  const [itemId, setItemId] = useState(candidates[0]?.item_id ?? "");
  const [exhibitLabel, setExhibitLabel] = useState("");
  const [batesNumber, setBatesNumber] = useState("");
  const [foundation, setFoundation] = useState("");
  const [relevance, setRelevance] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  if (candidates.length === 0) {
    return <p>No approved source items are available. Import a LegalSourcePackage first.</p>;
  }

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(`${legalApiBase()}/v1/exhibits`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          item_id: itemId,
          exhibit_label: exhibitLabel,
          bates_number: batesNumber,
          foundation,
          relevance,
        }),
      });
      if (!response.ok) {
        const body = await response.json().catch(() => ({}));
        throw new Error(body.detail ?? `exhibit ${response.status}`);
      }
      setExhibitLabel("");
      setBatesNumber("");
      setFoundation("");
      setRelevance("");
      router.refresh();
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "exhibit annotate failed");
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
        Approved package item
        <select
          value={itemId}
          onChange={(event) => setItemId(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        >
          {candidates.map((item) => (
            <option key={item.item_id} value={item.item_id}>
              {item.exhibit_label || item.span_locator}
            </option>
          ))}
        </select>
      </label>
      <label>
        Exhibit label
        <input
          value={exhibitLabel}
          onChange={(event) => setExhibitLabel(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Bates number
        <input
          value={batesNumber}
          onChange={(event) => setBatesNumber(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Foundation
        <input
          value={foundation}
          onChange={(event) => setFoundation(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Relevance
        <input
          value={relevance}
          onChange={(event) => setRelevance(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      {error ? <p style={{ color: "var(--status-warn)" }}>{error}</p> : null}
      <button type="submit" disabled={busy}>
        {busy ? "Saving…" : "Save exhibit annotation"}
      </button>
      <button
        type="button"
        disabled={busy || !itemId}
        onClick={() => {
          void (async () => {
            setBusy(true);
            setError(null);
            try {
              const response = await fetch(`${legalApiBase()}/v1/exhibits/${itemId}:bates`, {
                method: "POST",
              });
              if (!response.ok) {
                const body = await response.json().catch(() => ({}));
                throw new Error(body.detail ?? `bates ${response.status}`);
              }
              router.refresh();
            } catch (exc) {
              setError(exc instanceof Error ? exc.message : "bates assign failed");
            } finally {
              setBusy(false);
            }
          })();
        }}
      >
        Assign next Bates
      </button>
    </form>
  );
}
