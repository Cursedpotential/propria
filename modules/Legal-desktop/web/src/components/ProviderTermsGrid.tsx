import type { ProviderGrid } from "@/lib/api/client";
import { ConfidentialFlag } from "@/components/ConfidentialFlag";

// Byline: Grok · grok-4.6 · 2026-08-18

export function ProviderTermsGrid({
  grid,
  error,
}: {
  grid: ProviderGrid | null;
  error: string | null;
}) {
  return (
    <section className="surface-card">
      <h2>Provider privacy settings</h2>
      <p className="muted">Compare recorded data-retention and training settings for AI providers, and see which are eligible when Confidential Mode is on.</p>
      <div className="legal-context-states" aria-label="Provider settings status">
        <ConfidentialFlag />
        {grid ? <>
          <span className="pr-status" data-pr-status="information">Saved provider terms</span>
          <span className="pr-status" data-pr-status="information">PACER {grid.pacer ? "on" : "off"}</span>
        </> : null}
      </div>
      {!grid && !error ? <p className="muted">Provider settings unavailable.</p> : null}
      {error ? <p>{error}</p> : null}
      {grid ? (
        <table className="provider-grid">
          <thead>
            <tr>
              <th>Provider</th>
              <th>Role</th>
              <th>Uses data for training</th>
              <th>Data retention</th>
              <th>Confidential mode</th>
              <th>Use</th>
            </tr>
          </thead>
          <tbody>
            {grid.rows.map((row) => (
              <tr key={row.id}>
                <td>
                  <strong>{row.display_name}</strong>
                  <div className="muted">{row.id}</div>
                </td>
                <td>{row.role}</td>
                <td>{row.train ? "Assumed yes" : "No"}</td>
                <td>{row.retain.replaceAll("_", " ")}</td>
                <td className={row.confidential_eligible ? "eligible" : "blocked"}>
                  <span className="pr-status" data-pr-status={row.confidential_eligible ? "information" : "caution"}>
                    {row.confidential_eligible ? "Eligible" : "Blocked"}
                  </span>
                </td>
                <td>{row.use}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
    </section>
  );
}
