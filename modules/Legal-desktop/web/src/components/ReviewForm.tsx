// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { legalApiBase } from "@/lib/api/client";

export function ReviewForm({
  sections,
}: {
  sections: Array<{ section_id: string; heading: string }>;
}) {
  const router = useRouter();
  const [sectionId, setSectionId] = useState(sections[0]?.section_id ?? "");
  const [verdict, setVerdict] = useState("approve");
  const [rationale, setRationale] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  if (sections.length === 0) {
    return <p>No draft sections are available. Write a draft first.</p>;
  }

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(`${legalApiBase()}/v1/reviews`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          section_id: sectionId,
          verdict,
          rationale,
          reviewer: "owner",
        }),
      });
      if (!response.ok) {
        const body = await response.json().catch(() => ({}));
        throw new Error(body.detail ?? `review ${response.status}`);
      }
      setRationale("");
      router.refresh();
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "review failed");
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
        Draft section to review
        <select
          value={sectionId}
          onChange={(event) => setSectionId(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        >
          {sections.map((section) => (
            <option key={section.section_id} value={section.section_id}>
              {section.heading}
            </option>
          ))}
        </select>
      </label>
      <label>
        Verdict
        <select
          value={verdict}
          onChange={(event) => setVerdict(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        >
          <option value="approve">Approve</option>
          <option value="reject">Reject</option>
          <option value="request_changes">Request changes</option>
        </select>
      </label>
      <label>
        Reason for this decision (required)
        <textarea
          value={rationale}
          onChange={(event) => setRationale(event.target.value)}
          required
          rows={3}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      {error ? <p style={{ color: "var(--status-warn)" }}>{error}</p> : null}
      <button type="submit" disabled={busy || !rationale.trim()}>
        {busy ? "Saving…" : "Save review decision"}
      </button>
    </form>
  );
}
