// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
import { DraftEditor } from "@/components/DraftEditor";
import { TemplateForm } from "@/components/TemplateForm";
import { DocumentWorkspace } from "@/components/DocumentWorkspace";
import { fetchDrafts, legalApiBase } from "@/lib/api/client";

export default async function DraftPage() {
  let drafts: Awaited<ReturnType<typeof fetchDrafts>> = [];
  let error: string | null = null;
  try {
    drafts = await fetchDrafts();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  let templates: Array<{ template_id: string; title: string }> = [];
  let templateError: string | null = null;
  try {
    const response = await fetch(`${legalApiBase()}/v1/templates`, { cache: "no-store" });
    if (!response.ok) throw new Error("Templates are temporarily unavailable.");
    templates = await response.json();
  } catch {
    templateError = "Templates are temporarily unavailable. Try reloading this page.";
  }

  return (
    <>
      <h1 className="legal">Documents and writing</h1>
      <p>
        Write a document, open an existing file, or start from an editable template.
      </p>
      <DocumentWorkspace />
      <details style={{ marginTop: 24 }}>
      <summary>Earlier section drafts and outlines</summary>
      {error ? <p>{error}</p> : null}
      <details open={drafts.length === 0} className="surface-card">
        <summary>Start a draft from an outline</summary>
        {templateError ? <p role="status">{templateError}</p> : templates.length ? (
          <TemplateForm templates={templates} />
        ) : <p>No templates are available yet.</p>}
        <a href="/templates">Browse template outlines</a>
      </details>
      {drafts.map((draft) => (
        <article
          key={draft.section_id}
          style={{ borderTop: "1px solid var(--border)", padding: "16px 0" }}
        >
          <h2>{draft.heading}</h2>
          {draft.unsupported ? (
            <span className="pr-status" data-pr-status="caution">Citation support needed</span>
          ) : null}
          <DraftEditor
            sectionId={draft.section_id}
            heading={draft.heading}
            body={draft.body}
          />
          <p style={{ color: "var(--text-muted)" }}>
            Factor ({draft.factor_letter}) · {draft.citation_count} citations
            {draft.support ? ` · unsupported paragraphs: ${draft.support.unsupported_count}` : ""}
          </p>
          <details>
            <summary>Citation support</summary>
          {draft.support
            ? draft.support.paragraphs.map((paragraph) => (
                <p
                  key={paragraph.index}
                  style={{
                    color: paragraph.state === "unsupported" ? "var(--status-warn)" : "var(--text-muted)",
                    fontSize: 13,
                  }}
                >
                  [{paragraph.state}] {paragraph.text}
                </p>
              ))
            : <p>No support assessment is available yet.</p>}
          </details>
        </article>
      ))}
      </details>
    </>
  );
}
