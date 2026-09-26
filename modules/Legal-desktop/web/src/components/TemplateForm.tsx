// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { legalApiBase } from "@/lib/api/client";

export function TemplateForm({
  templates,
}: {
  templates: Array<{ template_id: string; title: string }>;
}) {
  const router = useRouter();
  const [templateId, setTemplateId] = useState(templates[0]?.template_id ?? "");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(`${legalApiBase()}/v1/templates:instantiate`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ template_id: templateId }),
      });
      if (!response.ok) throw new Error(`template ${response.status}`);
      router.push("/drafts");
      router.refresh();
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "instantiate failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        void submit();
      }}
      style={{ display: "grid", gap: 10, margin: "16px 0 32px" }}
    >
      <label>
        Choose a template
        <select
          value={templateId}
          onChange={(event) => setTemplateId(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        >
          {templates.map((template) => (
            <option key={template.template_id} value={template.template_id}>
              {template.title}
            </option>
          ))}
        </select>
      </label>
      {error ? <p style={{ color: "var(--status-warn)" }}>{error}</p> : null}
      <button type="submit" disabled={busy || !templateId}>
        {busy ? "Creating…" : "Create draft"}
      </button>
    </form>
  );
}
