import { legalApiBase } from "@/lib/api/client";

// Byline: Grok · grok-4.6 · 2026-08-18

export default async function AnalysisQueuePage() {
  let jobs: Array<{ id: string; next_run_time: string | null; trigger: string }> = [];
  let error: string | null = null;
  try {
    const response = await fetch(`${legalApiBase()}/v1/automations/analysis-queue`, { cache: "no-store" });
    if (!response.ok) throw new Error(`legal-api jobs ${response.status}`);
    jobs = await response.json();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <h1 className="legal">Analysis queue</h1>
      <p>Review queued jobs and their next run times. Controls are on Scheduled jobs.</p>
      {error ? <p className="unsupported">{error}</p> : null}
      {jobs.length === 0 && !error ? <p className="muted">No jobs queued.</p> : null}
      <table className="work-sheet">
        <thead>
          <tr>
            <th>Job</th>
            <th>Next run</th>
            <th>Schedule</th>
          </tr>
        </thead>
        <tbody>
          {jobs.map((job) => (
            <tr key={job.id}>
              <td>{job.id}</td>
              <td>{job.next_run_time ?? "—"}</td>
              <td>{job.trigger}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p>
        <a href="/scheduled-jobs">Scheduled jobs</a>
      </p>
    </>
  );
}
