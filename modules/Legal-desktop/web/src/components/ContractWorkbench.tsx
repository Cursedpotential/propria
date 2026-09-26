"use client";

import { useMemo, useState } from "react";

// Byline: Grok · grok-4.6 · 2026-08-18
// Review and compare working documents.

type Draft = {
  section_id: string;
  heading: string;
  body: string;
};

type Tab = "analyze" | "compare" | "negotiate";

export function ContractWorkbench({ drafts }: { drafts: Draft[] }) {
  const [tab, setTab] = useState<Tab>("analyze");
  const [leftId, setLeftId] = useState(drafts[0]?.section_id ?? "");
  const [rightId, setRightId] = useState(drafts[1]?.section_id ?? drafts[0]?.section_id ?? "");
  const [note, setNote] = useState("");

  const left = useMemo(
    () => drafts.find((item) => item.section_id === leftId) ?? drafts[0],
    [drafts, leftId],
  );
  const right = useMemo(
    () => drafts.find((item) => item.section_id === rightId) ?? drafts[1] ?? drafts[0],
    [drafts, rightId],
  );

  if (!drafts.length) {
    return <p className="muted">No drafts available to review yet. Create a draft in Motion writer.</p>;
  }

  return (
    <div>
      <div className="tab-row" role="tablist">
        {(
          [
            ["analyze", "Read"],
            ["compare", "Compare"],
            ["negotiate", "Review notes"],
          ] as const
        ).map(([id, label]) => (
          <button
            key={id}
            type="button"
            role="tab"
            aria-selected={tab === id}
            className={tab === id ? "tab-btn active" : "tab-btn"}
            onClick={() => setTab(id)}
          >
            {label}
          </button>
        ))}
      </div>
      <label>
        This draft
        <select value={leftId} onChange={(event) => setLeftId(event.target.value)}>
          {drafts.map((item) => (
            <option key={item.section_id} value={item.section_id}>
              {item.heading}
            </option>
          ))}
        </select>
      </label>
      {tab === "compare" ? (
        <label>
          Other draft
          <select value={rightId} onChange={(event) => setRightId(event.target.value)}>
            {drafts.map((item) => (
              <option key={item.section_id} value={item.section_id}>
                {item.heading}
              </option>
            ))}
          </select>
        </label>
      ) : null}
      {tab === "analyze" && left ? (
        <article className="surface-card">
          <h2>{left.heading}</h2>
          <pre className="draft-pre">{left.body || "(empty)"}</pre>
        </article>
      ) : null}
      {tab === "compare" && left && right ? (
        <div className="compare-grid">
          <article className="surface-card">
            <h2>{left.heading}</h2>
            <pre className="draft-pre">{left.body || "(empty)"}</pre>
          </article>
          <article className="surface-card">
            <h2>{right.heading}</h2>
            <pre className="draft-pre">{right.body || "(empty)"}</pre>
          </article>
        </div>
      ) : null}
      {tab === "negotiate" && left ? (
        <article className="surface-card">
          <h2>{left.heading}</h2>
          <p className="muted">
            Notes on wording, proposed changes, or questions about this document.
          </p>
          <textarea
            rows={6}
            value={note}
            onChange={(event) => setNote(event.target.value)}
            placeholder="Add your review notes"
          />
          <pre className="draft-pre">{left.body || "(empty)"}</pre>
        </article>
      ) : null}
    </div>
  );
}
