"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { legalApiBase } from "@/lib/api/client";

// Byline: Grok · grok-4.6 · 2026-08-18
// One dense sheet: click a cell to change it, type the last row to add.

export type SheetField = {
  key: string;
  label: string;
  kind?: "text" | "textarea" | "select";
  options?: Array<{ value: string; label: string }>;
  required?: boolean;
};

export type SheetRow = Record<string, string>;

export function WorkSheet({
  fields,
  rows,
  createPath,
  updatePath,
  createExtras,
  addLabel,
}: {
  fields: SheetField[];
  rows: SheetRow[];
  createPath: string;
  updatePath?: string;
  createExtras?: Record<string, string>;
  addLabel: string;
}) {
  const router = useRouter();
  const blank = useMemo(
    () => Object.fromEntries(fields.map((field) => [field.key, ""])) as SheetRow,
    [fields],
  );
  const [draft, setDraft] = useState<SheetRow>(blank);
  const [editing, setEditing] = useState<SheetRow | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function send(method: string, path: string, body: Record<string, string>) {
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(`${legalApiBase()}${path}`, {
        method,
        headers: { "content-type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!response.ok) throw new Error(`${response.status}`);
      router.refresh();
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "save failed");
    } finally {
      setBusy(false);
    }
  }

  function create() {
    const missing = fields.find((field) => field.required && !draft[field.key]?.trim());
    if (missing) {
      setError(`${missing.label} is required`);
      return;
    }
    void send("POST", createPath, { ...createExtras, ...draft }).then(() => setDraft(blank));
  }

  function pathFor(row: SheetRow) {
    if (!updatePath) return "";
    return updatePath.replace("{id}", row.id ?? "");
  }

  function saveEdit() {
    if (!editing || !updatePath) return;
    void send("PUT", pathFor(editing), editing).then(() => setEditing(null));
  }

  return (
    <div className="work-sheet">
      <table>
        <thead>
          <tr>
            {fields.map((field) => (
              <th key={field.key}>{field.label}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, index) => {
            const active = editing && updatePath && pathFor(editing) === pathFor(row);
            return (
              <tr
                key={row.id ?? String(index)}
                onClick={() => updatePath && setEditing({ ...row })}
              >
                {fields.map((field) => (
                  <td key={field.key}>
                    {active ? (
                      <Field
                        field={field}
                        value={editing[field.key] ?? ""}
                        onChange={(value) => setEditing({ ...editing, [field.key]: value })}
                        onEnter={saveEdit}
                      />
                    ) : (
                      editing && active ? null : (row[field.key] || "—")
                    )}
                  </td>
                ))}
              </tr>
            );
          })}
          <tr className="work-sheet-add">
            {fields.map((field) => (
              <td key={field.key}>
                <Field
                  field={field}
                  value={draft[field.key] ?? ""}
                  onChange={(value) => setDraft({ ...draft, [field.key]: value })}
                  onEnter={create}
                />
              </td>
            ))}
          </tr>
        </tbody>
      </table>
      <div className="work-sheet-bar">
        <button type="button" disabled={busy} onClick={create}>
          {busy ? "Saving…" : addLabel}
        </button>
        {editing ? (
          <button type="button" disabled={busy} onClick={saveEdit}>
            Save change
          </button>
        ) : null}
        {error ? <span className="unsupported">{error}</span> : null}
      </div>
    </div>
  );
}

function Field({
  field,
  value,
  onChange,
  onEnter,
}: {
  field: SheetField;
  value: string;
  onChange: (value: string) => void;
  onEnter: () => void;
}) {
  if (field.kind === "select" && field.options) {
    return (
      <select value={value} onChange={(event) => onChange(event.target.value)} aria-label={field.label}>
        <option value=""> </option>
        {field.options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    );
  }
  if (field.kind === "textarea") {
    return (
      <textarea
        value={value}
        aria-label={field.label}
        rows={2}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) onEnter();
        }}
      />
    );
  }
  return (
    <input
      value={value}
      aria-label={field.label}
      onChange={(event) => onChange(event.target.value)}
      onKeyDown={(event) => {
        if (event.key === "Enter") {
          event.preventDefault();
          onEnter();
        }
      }}
    />
  );
}
