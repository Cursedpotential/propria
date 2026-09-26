import { CitationParse } from "@/components/CitationParse";

// Byline: Grok · grok-4.6 · 2026-08-18

export default function CitationCheckPage() {
  return (
    <>
      <h1 className="legal">Citation check</h1>
      <p>
        Check citation format, normalize the text, or compare a stored snapshot.
      </p>
      <CitationParse />
    </>
  );
}
