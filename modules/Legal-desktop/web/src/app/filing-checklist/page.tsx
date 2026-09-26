// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
import { legalApiBase } from "@/lib/api/client";

export default async function FilingPage() {
  let report: {
    ready: boolean;
    filed: boolean;
    blocking_count: number;
    checks: Array<{
      check_id: string;
      label: string;
      state: string;
      reason: string;
      human: boolean;
    }>;
  } | null = null;
  let error: string | null = null;
  try {
    const response = await fetch(`${legalApiBase()}/v1/filing-readiness`, { cache: "no-store" });
    if (!response.ok) throw new Error(`legal-api filing ${response.status}`);
    report = await response.json();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <h1 style={{ fontFamily: "Georgia, serif", fontWeight: 500 }}>
        Review filing readiness
      </h1>
      <p>
        Review each gate, blocking item, and owner verification requirement before release.
      </p>
      {error ? <p>{error}</p> : null}
      {report ? (
        <>
          <p>
            Ready: {String(report.ready)} · Filed: {String(report.filed)} ·
            Blocking: {report.blocking_count}
          </p>
          <ul style={{ listStyle: "none", padding: 0 }}>
            {report.checks.map((check) => (
              <li
                key={check.check_id}
                style={{ borderTop: "1px solid var(--border)", padding: "12px 0" }}
              >
                <strong>
                  [{check.state}] {check.label}
                </strong>
                <div style={{ color: "var(--text-muted)" }}>
                  {check.reason}
                  {check.human ? " · owner verification required" : ""}
                </div>
              </li>
            ))}
          </ul>
        </>
      ) : null}
    </>
  );
}
