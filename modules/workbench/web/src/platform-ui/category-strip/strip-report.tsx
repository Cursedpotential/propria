// Byline: Claude Code · Opus 5.5 · 2026-09-25
// Template: summary header on top, category strip below, one scheme for both. Any rows, any scheme.

import { CategoryStrip, type CategoryStripProps } from "./category-strip";
import { StripSummary, type StripSummaryProps } from "./strip-summary";

export type StripReportProps = CategoryStripProps & Pick<StripSummaryProps, "title" | "nouns" | "facts"> & {
  /** Text under the title; `description` stays the line above the rows. */
  summary?: string;
};

export function StripReport({ title, summary, nouns, facts, ...strip }: StripReportProps) {
  return (
    <div className="grid gap-4">
      <StripSummary rows={strip.rows} scheme={strip.scheme} title={title} description={summary} nouns={nouns} facts={facts} />
      <CategoryStrip {...strip} />
    </div>
  );
}
