// Byline: Grok · grok-4.6 · 2026-08-18
import { EventList, type AuditEvent } from "@/components/EventList";
import { fetchMatter, legalApiBase } from "@/lib/api/client";

const HOME_GROUPS: Array<{ title: string; paths: string[] }> = [
  { title: "Research", paths: ["/case-search", "/laws", "/citation-check"] },
  { title: "Documents and drafting", paths: ["/documents", "/drafts", "/agreements"] },
  { title: "Privilege", paths: ["/confidentiality-check"] },
  { title: "Operations", paths: ["/analysis-queue", "/playbooks", "/scheduled-jobs", "/notices", "/activity-log", "/external-sources"] },
];

export default async function MatterHomePage() {
  let matter = null;
  let error: string | null = null;
  let recent: AuditEvent[] = [];
  try {
    matter = await fetchMatter();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }
  try {
    const response = await fetch(`${legalApiBase()}/v1/audit?limit=8`, { cache: "no-store" });
    if (response.ok) recent = await response.json();
  } catch {
    recent = [];
  }

  const surfaces = matter?.next_surfaces ?? [];

  return (
    <>
      <div className="section-eyebrow">Case dashboard</div>
      <h1 className="legal">
        {matter?.matter.display_name ?? "Genesee County custody matter"}
      </h1>
      {error ? (
        <p className="unsupported">
          legal-api is not running. Start it with uv run uvicorn on port 8010.
        </p>
      ) : null}
      {matter ? (
        <>
          <p className="dim">
            {matter.court_case.display_name}
            {matter.court_case.court ? ` · ${matter.court_case.court}` : ""}
          </p>
          <p className="muted">
            Judge: {matter.judge_confirmed ? "confirmed by the clerk" : "not confirmed by the clerk"} ·
            Family-court office: {matter.foc_confirmed ? "confirmed" : "not confirmed"}
          </p>
          <p>
            <a className="quick-action" href="/assistant" title="Ask the assistant about the current workspace context.">
              Ask the assistant
            </a>
          </p>
          <div className="metric-row">
            <div className="metric-tile">
              <span className="metric-label">Deadlines</span>
              <span className="metric-value">{matter.upcoming_event_count}</span>
            </div>
            <div className="metric-tile">
              <span className="metric-label">Documents</span>
              <span className="metric-value">{matter.draft_count}</span>
            </div>
            <div className="metric-tile">
              <span className="metric-label">Exhibits</span>
              <span className="metric-value">{matter.exhibit_count}</span>
            </div>
            <div className="metric-tile">
              <span className="metric-label">Activity log</span>
              <span className="metric-value">{matter.audit_count ?? 0}</span>
            </div>
          </div>
          {HOME_GROUPS.map((group) => (
            <div key={group.title} className="surface-card">
              <div className="section-eyebrow">{group.title}</div>
              <div className="qa-grid">
                {surfaces
                  .filter((surface) => group.paths.includes(surface.path))
                  .map((surface) => (
                    <a
                      key={surface.path}
                      className="quick-action"
                      href={surface.path}
                      title={surface.help}
                    >
                      {surface.label}
                    </a>
                  ))}
              </div>
            </div>
          ))}
          <div className="surface-card">
            <div className="section-eyebrow">Recent activity</div>
            <EventList events={recent} />
          </div>
          <div className="surface-card">
            <div className="section-eyebrow">Questions the judge decides</div>
            <h2>{matter.issue.title}</h2>
            <p className="muted">{matter.issue.governing_authority}</p>
            <p>
              <a href="/questions">Questions the judge decides</a>
            </p>
          </div>
        </>
      ) : null}
    </>
  );
}
