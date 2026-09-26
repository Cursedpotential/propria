// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
import { legalApiBase } from "@/lib/api/client";

async function fetchStrategy() {
  const response = await fetch(`${legalApiBase()}/v1/strategy`, { cache: "no-store" });
  if (!response.ok) {
    throw new Error(`legal-api strategy ${response.status}`);
  }
  return response.json() as Promise<
    Array<{
      note_id: string;
      kind: string;
      title: string;
      body: string;
      disclosure: string;
      court_safe: boolean;
    }>
  >;
}

export default async function StrategyPage() {
  let notes: Awaited<ReturnType<typeof fetchStrategy>> = [];
  let error: string | null = null;
  try {
    notes = await fetchStrategy();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <h1 style={{ fontFamily: "Georgia, serif", fontWeight: 500 }}>
        Working notes and strategy
      </h1>
      <p>
        Review private strategy notes and their disclosure status.
      </p>
      {error ? <p>{error}</p> : null}
      {notes.length === 0 && !error ? <p>No strategy notes yet.</p> : null}
      {notes.map((note) => (
        <article
          key={note.note_id}
          style={{ borderTop: "1px solid var(--border)", padding: "16px 0" }}
        >
          <strong>
            [{note.kind}] {note.title}
          </strong>
          <p>{note.body}</p>
          <p style={{ color: "var(--text-muted)" }}>
            <span className="pr-status" data-pr-status="information">Private note</span>
          </p>
        </article>
      ))}
    </>
  );
}
