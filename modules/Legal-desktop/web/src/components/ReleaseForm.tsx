// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { legalApiBase } from "@/lib/api/client";

export function ReleaseForm({
  sections,
}: {
  sections: Array<{ section_id: string; heading: string }>;
}) {
  const router = useRouter();
  const [selected, setSelected] = useState<string[]>(
    sections.map((section) => section.section_id),
  );
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  if (sections.length === 0) {
    return <p>Approve a draft in Your review before creating a release copy.</p>;
  }

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(`${legalApiBase()}/v1/releases`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ section_ids: selected }),
      });
      if (!response.ok) {
        throw new Error(`release ${response.status}`);
      }
      const body = (await response.json()) as { blocked: boolean; blockers: string[] };
      if (body.blocked) {
        throw new Error(body.blockers.join("; ") || "release blocked");
      }
      router.refresh();
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "release failed");
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
      {sections.map((section) => (
        <label key={section.section_id}>
          <input
            type="checkbox"
            checked={selected.includes(section.section_id)}
            onChange={(event) => {
              setSelected((current) =>
                event.target.checked
                  ? [...current, section.section_id]
                  : current.filter((id) => id !== section.section_id),
              );
            }}
          />{" "}
          {section.heading}
        </label>
      ))}
      {error ? <p style={{ color: "var(--status-warn)" }}>{error}</p> : null}
      <button type="submit" disabled={busy || selected.length === 0}>
        {busy ? "Building…" : "Build release copy"}
      </button>
    </form>
  );
}
