import { PlaybookRun } from "@/components/PlaybookRun";
import { legalApiBase } from "@/lib/api/client";

// Byline: Grok · grok-4.6 · 2026-08-18

export default async function ScheduledJobsPage() {
  let jobs: Array<{ id: string; next_run_time: string | null; trigger: string }> = [];
  let playbooks: Array<{
    playbook_id: string;
    title: string;
    steps: string[];
    note?: string;
  }> = [];
  let error: string | null = null;
  try {
    const [jobRes, bookRes] = await Promise.all([
      fetch(`${legalApiBase()}/v1/automations/analysis-queue`, { cache: "no-store" }),
      fetch(`${legalApiBase()}/v1/automations/playbooks`, { cache: "no-store" }),
    ]);
    if (!jobRes.ok) throw new Error(`legal-api jobs ${jobRes.status}`);
    if (!bookRes.ok) throw new Error(`legal-api playbooks ${bookRes.status}`);
    jobs = await jobRes.json();
    playbooks = await bookRes.json();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <h1 className="legal">Scheduled jobs</h1>
      <p>Review scheduled jobs and run recurring playbooks.</p>
      {error ? <p className="unsupported">{error}</p> : null}
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
      <PlaybookRun playbooks={playbooks} />
    </>
  );
}
