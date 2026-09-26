// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
import { EvidenceRequestForm } from "@/components/EvidenceRequestForm";
import { legalApiBase } from "@/lib/api/client";

export default async function EvidenceRequestsPage() {
  let requests: Array<{
    request_id: string;
    kind: string;
    text: string;
    purpose: string;
    linked_issue: string;
    missing_proof: string;
    status: string;
    served_on: string | null;
    authority: string;
  }> = [];
  let error: string | null = null;
  try {
    const response = await fetch(`${legalApiBase()}/v1/discovery`, { cache: "no-store" });
    if (!response.ok) throw new Error(`legal-api discovery ${response.status}`);
    requests = await response.json();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <h1 style={{ fontFamily: "Georgia, serif", fontWeight: 500 }}>
        Discovery requests
      </h1>
      <p>
        Create and review requests for documents, admissions, answers, and subpoenas.
      </p>
      {error ? <p>{error}</p> : null}
      <EvidenceRequestForm />
      {requests.map((item) => (
        <article
          key={item.request_id}
          style={{ borderTop: "1px solid var(--border)", padding: "16px 0" }}
        >
          <strong>
            [{item.status}] {item.kind}
          </strong>
          <p>{item.text}</p>
          <p>
            <em>Purpose:</em> {item.purpose}
          </p>
          {item.served_on ? (
            <p>
              <em>Served:</em> {item.served_on}
            </p>
          ) : null}
        </article>
      ))}
    </>
  );
}
