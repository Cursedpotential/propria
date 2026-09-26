// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
import { legalApiBase } from "@/lib/api/client";

export default async function RedTeamPage() {
  let runs: Array<{
    run_id: string;
    lens: string;
    target_type: string;
    prompt_or_notes: string;
    court_safe: boolean;
    findings: Array<{ claim: string; severity: string }>;
  }> = [];
  let error: string | null = null;
  try {
    const response = await fetch(`${legalApiBase()}/v1/redteam`, { cache: "no-store" });
    if (!response.ok) throw new Error(`legal-api redteam ${response.status}`);
    runs = await response.json();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <h1 style={{ fontFamily: "Georgia, serif", fontWeight: 500 }}>
        Draft challenge results
      </h1>
      <p>
        Review stored runs using opposing-counsel, judge, FOC/referee, adverse-authority, and missing-proof lenses.
      </p>
      {error ? <p>{error}</p> : null}
      {runs.length === 0 && !error ? <p>No red-team runs stored yet.</p> : null}
      {runs.map((run) => (
        <article key={run.run_id} style={{ borderTop: "1px solid var(--border)", padding: "16px 0" }}>
          <strong>
            {run.lens} → {run.target_type}
          </strong>
          <p>{run.prompt_or_notes}</p>
          <ul>
            {run.findings.map((finding) => (
              <li key={finding.claim}>
                [{finding.severity}] {finding.claim}
              </li>
            ))}
          </ul>
          <p style={{ color: "var(--text-muted)" }}>
            Private critique
          </p>
        </article>
      ))}
    </>
  );
}
