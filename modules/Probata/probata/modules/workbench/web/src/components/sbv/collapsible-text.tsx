// Byline: Claude Code · Sonnet 5.5 · 2026-10-03
"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";

/** Characters shown before a message body folds behind "Show more" in a chat bubble. */
export const BUBBLE_TEXT_LIMIT = 600;

/** Characters shown before a message body folds in the detail pane (the reading place). */
export const DETAIL_TEXT_LIMIT = 2000;

/**
 * Cuts `text` to at most `limit` characters, preferring the last whitespace in the final
 * fifth so a word is not split. Returns the text unchanged when it already fits.
 */
export function foldText(text: string, limit: number): { head: string; folded: boolean } {
  if (text.length <= limit) return { head: text, folded: false };
  const floor = Math.floor(limit * 0.8);
  const slice = text.slice(0, limit);
  const lastSpace = Math.max(slice.lastIndexOf(" "), slice.lastIndexOf("\n"));
  return { head: lastSpace >= floor ? slice.slice(0, lastSpace) : slice, folded: true };
}

interface CollapsibleTextProps {
  text: string;
  /** Characters kept visible while folded. */
  limit: number;
  className?: string;
  /** Extra classes for the toggle button, e.g. a light-on-dark colour inside a sent bubble. */
  toggleClassName?: string;
}

/**
 * Shows a message body in full when it is short, and the first `limit` characters with a
 * "Show more" toggle when it is long (a pasted 21,000-character document would otherwise
 * fill the whole pane). The text is never altered, rendered as markdown or reformatted:
 * what is shown is exactly the stored body, expanded in place.
 *
 * Inputs: the body and the fold limit. Output: the text plus, when folded, one toggle
 * button. No side effects, no network.
 */
export function CollapsibleText({ text, limit, className, toggleClassName }: CollapsibleTextProps) {
  const [expanded, setExpanded] = useState(false);
  const { head, folded } = foldText(text, limit);
  if (!folded) return <div className={className}>{text}</div>;
  return (
    <div data-testid="collapsible-text" data-expanded={expanded}>
      <div className={className}>{expanded ? text : `${head}…`}</div>
      <Button
        type="button"
        variant="link"
        size="sm"
        className={`h-auto px-0 py-1 text-xs ${toggleClassName ?? ""}`}
        aria-expanded={expanded}
        onClick={() => setExpanded((value) => !value)}
        data-testid="collapsible-text-toggle"
      >
        {expanded ? "Show less" : `Show more (${(text.length - head.length).toLocaleString()} more characters)`}
      </Button>
    </div>
  );
}
