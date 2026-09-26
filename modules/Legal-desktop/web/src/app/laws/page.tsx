import { legalApiBase } from "@/lib/api/client";

// Byline: Grok · grok-4.6 · 2026-08-18

const QUICK = ["MCL 722.23", "MCL 722.27(1)(c)", "MCL 722.27a(3)"];

export default async function LawsPage() {
  let authorities: Array<{
    identifier: string;
    proposition: string;
    authority_level: string;
    pinpoint: string | null;
    snapshot_hash: string;
    is_citator_verified: boolean;
  }> = [];
  let error: string | null = null;
  try {
    const response = await fetch(`${legalApiBase()}/v1/authorities`, { cache: "no-store" });
    if (!response.ok) throw new Error(`legal-api authorities ${response.status}`);
    authorities = await response.json();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <h1 className="legal">Laws</h1>
      <p>
        Browse the Michigan statutes and rules saved for this matter, including source links and the points they support.
      </p>
      <p>
        {QUICK.map((id) => (
          <a key={id} className="surface-chip" href={`#${id.replaceAll(" ", "-")}`}>
            {id}
          </a>
        ))}
      </p>
      {error ? <p className="unsupported">{error}</p> : null}
      {authorities.map((authority) => (
        <article
          key={authority.identifier}
          id={authority.identifier.replaceAll(" ", "-")}
          style={{ borderTop: "1px solid var(--border)", padding: "16px 0" }}
        >
          <strong>
            [{authority.authority_level}] {authority.identifier}
          </strong>
          <p>{authority.proposition}</p>
          <p className="muted">
            {authority.pinpoint ?? "no pinpoint"} · {authority.snapshot_hash} ·
            citator={String(authority.is_citator_verified)}
          </p>
        </article>
      ))}
    </>
  );
}
