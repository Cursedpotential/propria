// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
import { ExhibitForm } from "@/components/ExhibitForm";
import { legalApiBase } from "@/lib/api/client";

export default async function ExhibitsPage() {
  let exhibits: Array<{
    item_id: string;
    package_id: string;
    assertion_id: string;
    span_locator: string;
    custody_locator: string;
    content_hash: string;
    review_state: string;
    exhibit_label: string;
    bates_number: string;
    foundation: string;
    custody_note: string;
    redaction_note: string;
    relevance: string;
    issues: string[];
    objection_notes: string;
    readiness: string;
    bytes_present: boolean;
  }> = [];
  let error: string | null = null;
  try {
    const response = await fetch(`${legalApiBase()}/v1/exhibits`, { cache: "no-store" });
    if (!response.ok) throw new Error(`legal-api exhibits ${response.status}`);
    exhibits = await response.json();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <h1 style={{ fontFamily: "Georgia, serif", fontWeight: 500 }}>
        Approved package items only
      </h1>
      <p>
        Review evidence linked to this case, add notes, and record where each item can be found.
      </p>
      {error ? <p>{error}</p> : null}
      <ExhibitForm
        candidates={exhibits.map((item) => ({
          item_id: item.item_id,
          span_locator: item.span_locator,
          exhibit_label: item.exhibit_label,
        }))}
      />
      {exhibits.length === 0 && !error ? (
        <p>No approved package items. Import a LegalSourcePackage first.</p>
      ) : null}
      {exhibits.map((item) => (
        <article
          key={item.item_id}
          style={{ borderTop: "1px solid var(--border)", padding: "16px 0" }}
        >
          <strong>
            [{item.readiness}] {item.exhibit_label || "unlabeled"}
          </strong>
          <p>
            {item.span_locator} · {item.custody_locator}
          </p>
          <p style={{ color: "var(--text-muted)" }}>
            Bates: {item.bates_number || "none"} · hash {item.content_hash} ·
            bytes={String(item.bytes_present)}
          </p>
          <p style={{ color: "var(--text-muted)" }}>
            {item.foundation ? `foundation: ${item.foundation} · ` : ""}
            {item.relevance ? `relevance: ${item.relevance}` : "no relevance note"}
          </p>
        </article>
      ))}
    </>
  );
}
