// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { legalApiBase } from "@/lib/api/client";

export function DocketForm() {
  const router = useRouter();
  const [title, setTitle] = useState("");
  const [occursAt, setOccursAt] = useState("");
  const [kind, setKind] = useState("hearing");
  const [detail, setDetail] = useState("");
  const [location, setLocation] = useState("");
  const [confirmed, setConfirmed] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(`${legalApiBase()}/v1/docket-events`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          title,
          occurs_at: new Date(occursAt).toISOString(),
          kind,
          detail,
          location,
          source: "owner",
          confirmed,
        }),
      });
      if (!response.ok) throw new Error(`docket ${response.status}`);
      setTitle("");
      setOccursAt("");
      setDetail("");
      setLocation("");
      setConfirmed(false);
      router.refresh();
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "docket failed");
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
        Title
        <input
          value={title}
          onChange={(event) => setTitle(event.target.value)}
          required
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Date and time
        <input
          type="datetime-local"
          value={occursAt}
          onChange={(event) => setOccursAt(event.target.value)}
          required
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Kind
        <select
          value={kind}
          onChange={(event) => setKind(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        >
          <option value="hearing">hearing</option>
          <option value="deadline">deadline</option>
          <option value="filing">filing</option>
          <option value="order">order</option>
          <option value="foc">foc</option>
          <option value="conference">conference</option>
          <option value="service">service</option>
          <option value="other">other</option>
        </select>
      </label>
      <label>
        Location
        <input
          value={location}
          onChange={(event) => setLocation(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Detail
        <textarea
          value={detail}
          onChange={(event) => setDetail(event.target.value)}
          rows={2}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        <input
          type="checkbox"
          checked={confirmed}
          onChange={(event) => setConfirmed(event.target.checked)}
        />{" "}
        Confirmed by clerk
      </label>
      {error ? <p style={{ color: "var(--status-warn)" }}>{error}</p> : null}
      <button type="submit" disabled={busy || !title.trim() || !occursAt}>
        {busy ? "Saving…" : "Add docket event"}
      </button>
    </form>
  );
}
