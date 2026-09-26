// Byline: Grok · grok-4.6 · 2026-08-18
import { FactorNoteAdd } from "@/components/FactorNoteAdd";
import { fetchFactors, type FactorRow } from "@/lib/api/client";

function LinkedNotes({ factor }: { factor: FactorRow }) {
  const blocks: Array<{ label: string; items: string[] }> = [
    { label: "Petitioner", items: factor.petitioner.for_parent },
    { label: "Respondent", items: factor.respondent.for_parent },
    { label: "Contradictions", items: factor.contradictions },
    { label: "Missing proof", items: factor.missing_proof },
  ].filter((block) => block.items.length > 0);
  if (blocks.length === 0) return null;
  return (
    <div className="factor-notes">
      {blocks.map((block) => (
        <div key={block.label}>
          <div className="section-eyebrow">{block.label}</div>
          <ul>
            {block.items.map((note) => (
              <li key={note}>{note}</li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  );
}

export default async function FactorPage() {
  let factors: FactorRow[] = [];
  let error: string | null = null;
  try {
    factors = await fetchFactors();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <h1 className="legal">The 12 things the judge must weigh</h1>
      {error ? <p className="unsupported">{error}</p> : null}
      {factors.length > 0 ? <FactorNoteAdd factors={factors} /> : null}
      <ol className="factor-list">
        {factors.map((factor) => (
          <li key={factor.letter} className="factor-row">
            <strong>
              ({factor.letter}) {factor.title}
            </strong>
            <LinkedNotes factor={factor} />
          </li>
        ))}
      </ol>
    </>
  );
}
