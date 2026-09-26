// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
import { AgentRunForm } from "@/components/AgentRunForm";
import { RoutingEditor } from "@/components/RoutingEditor";
import { legalApiBase } from "@/lib/api/client";

export default async function AgentPage() {
  let runs: Array<{
    run_id: string;
    role: string;
    intent: string;
    output: string;
    status: string;
    effective_model: string;
    court_safe: boolean;
  }> = [];
  let error: string | null = null;
  let routing: {
    framework: string;
    notes: string;
    chat: {
      backend: string;
      next_path: string;
      run_path: string;
      confidential_path: string;
      default_confidential_model: string;
    };
    agents: Record<string, { backend: string; model: string; surface: string }>;
    surfaces?: Array<{ path: string; label: string; group: string; help?: string }>;
  } | null = null;
  try {
    const response = await fetch(`${legalApiBase()}/v1/agent-runs`, { cache: "no-store" });
    if (!response.ok) throw new Error(`legal-api agent-runs ${response.status}`);
    runs = await response.json();
    const table = await fetch(`${legalApiBase()}/v1/routing`, { cache: "no-store" });
    if (table.ok) routing = await table.json();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <h1 style={{ fontFamily: "Georgia, serif", fontWeight: 500 }}>
        Agent run history
      </h1>
      <p>
        Review assistant requests, the model used, and each run’s result.
      </p>
      {error ? <p>{error}</p> : null}
      {routing ? <RoutingEditor table={routing} /> : null}
      <AgentRunForm />
      {runs.length === 0 && !error ? <p>No agent runs recorded.</p> : null}
      {runs.map((run) => (
        <article
          key={run.run_id}
          style={{ borderTop: "1px solid var(--border)", padding: "16px 0" }}
        >
          <strong>
            [{run.status}] {run.role} · {run.intent}
          </strong>
          <p>{run.output}</p>
          <p style={{ color: "var(--text-muted)" }}>
            Model: {run.effective_model}
          </p>
        </article>
      ))}
    </>
  );
}
