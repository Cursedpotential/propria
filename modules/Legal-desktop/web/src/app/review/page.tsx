// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
import { ReviewForm } from "@/components/ReviewForm";
import { legalApiBase } from "@/lib/api/client";

export default async function ReviewPage() {
  let reviews: Array<{
    review_id: string;
    section_id: string;
    verdict: string;
    rationale: string;
    reviewer: string;
    content_hash: string;
  }> = [];
  let drafts: Array<{ section_id: string; heading: string }> = [];
  let error: string | null = null;
  try {
    const [reviewResponse, draftResponse] = await Promise.all([
      fetch(`${legalApiBase()}/v1/reviews`, { cache: "no-store" }),
      fetch(`${legalApiBase()}/v1/drafts`, { cache: "no-store" }),
    ]);
    if (!reviewResponse.ok) throw new Error(`legal-api reviews ${reviewResponse.status}`);
    if (!draftResponse.ok) throw new Error(`legal-api drafts ${draftResponse.status}`);
    reviews = await reviewResponse.json();
    drafts = await draftResponse.json();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <h1 style={{ fontFamily: "Georgia, serif", fontWeight: 500 }}>
        Review draft sections
      </h1>
      <p>
        Choose a draft section, record your decision, and explain any requested changes. Editing an approved section requires a new review.
      </p>
      {error ? <p>{error}</p> : null}
      <ReviewForm sections={drafts} />
      {reviews.length === 0 && !error ? <p>No review decisions yet.</p> : null}
      {reviews.map((review) => (
        <article
          key={review.review_id}
          style={{ borderTop: "1px solid var(--border)", padding: "16px 0" }}
        >
          <strong>
            [{review.verdict}] {review.reviewer}
          </strong>
          <p>{review.rationale}</p>
          <p style={{ color: "var(--text-muted)" }}>{review.content_hash}</p>
        </article>
      ))}
    </>
  );
}
