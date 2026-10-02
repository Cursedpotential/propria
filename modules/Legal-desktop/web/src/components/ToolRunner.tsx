"use client";

// Byline: Claude Code · Opus 5.5 · 2026-10-02
// Family Law Toolkit tool browser: pick a tool, fill the form generated from its input
// schema (or edit the arguments as JSON), see the exact call, click Run. A tool that can
// change the toolkit store needs the "reviewed" box ticked; the Run click authorizes the
// displayed call (MCP-CLIENT-AND-PORTS.md, "Manual call"). Results distinguish a tool
// error from a failed call. Sibling: FactorNoteAdd.tsx (same-origin BFF POST).
import { useMemo, useState } from "react";
import { legalApiBase } from "@/lib/api/client";

type JsonSchema = {
  type?: string | string[];
  enum?: unknown[];
  description?: string;
  maxLength?: number;
  minLength?: number;
  minimum?: number;
  maximum?: number;
  properties?: Record<string, JsonSchema>;
  required?: string[];
  items?: JsonSchema;
  anyOf?: JsonSchema[];
};

export type ToolDefinition = {
  connection_id: string;
  name: string;
  title: string;
  description: string;
  input_schema: JsonSchema;
  annotations: Record<string, unknown>;
  schema_hash: string;
  writes: boolean;
};

type Invocation = {
  id: string;
  tool: string;
  writes: boolean;
  state: "ok" | "tool_error";
  text: string[];
  structured: unknown;
  content: Record<string, unknown>[];
  truncated: boolean;
  completed_at: string;
};

const muted = { color: "var(--text-muted)" } as const;
const mono = { fontFamily: "ui-monospace, monospace", fontSize: 12 } as const;
const box = { border: "1px solid var(--border)", padding: 12, margin: "12px 0" } as const;

function kindOf(schema: JsonSchema): "enum" | "string" | "number" | "boolean" | "json" {
  if (schema.enum?.length) return "enum";
  const type = Array.isArray(schema.type) ? schema.type.find((t) => t !== "null") : schema.type;
  if (type === "string") return "string";
  if (type === "number" || type === "integer") return "number";
  if (type === "boolean") return "boolean";
  return "json";
}

function buildArguments(schema: JsonSchema, fields: Record<string, string>): Record<string, unknown> {
  const args: Record<string, unknown> = {};
  for (const [name, prop] of Object.entries(schema.properties ?? {})) {
    const raw = fields[name];
    if (raw === undefined || raw === "") continue;
    const kind = kindOf(prop);
    if (kind === "number") args[name] = Number(raw);
    else if (kind === "boolean") args[name] = raw === "true";
    else if (kind === "json") {
      try {
        args[name] = JSON.parse(raw);
      } catch {
        throw new Error(`${name}: not valid JSON`);
      }
    } else args[name] = raw;
  }
  return args;
}

function pretty(text: string): string {
  try {
    return JSON.stringify(JSON.parse(text), null, 2);
  } catch {
    return text;
  }
}

export function ToolRunner({ tools }: { tools: ToolDefinition[] }) {
  const [filter, setFilter] = useState("");
  const [selected, setSelected] = useState<string>(tools[0]?.name ?? "");
  const [fields, setFields] = useState<Record<string, string>>({});
  const [jsonMode, setJsonMode] = useState(false);
  const [jsonText, setJsonText] = useState("{}");
  const [reviewed, setReviewed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<Invocation | null>(null);

  const tool = tools.find((t) => t.name === selected) ?? null;
  const visible = tools.filter((t) =>
    `${t.name} ${t.title} ${t.description}`.toLowerCase().includes(filter.trim().toLowerCase()),
  );

  const built = useMemo((): { args: Record<string, unknown> | null; problem: string | null } => {
    if (!tool) return { args: null, problem: null };
    try {
      if (jsonMode) {
        const parsed = JSON.parse(jsonText);
        if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) throw new Error("arguments must be a JSON object");
        return { args: parsed as Record<string, unknown>, problem: null };
      }
      return { args: buildArguments(tool.input_schema, fields), problem: null };
    } catch (exc) {
      return { args: null, problem: exc instanceof Error ? exc.message : "arguments invalid" };
    }
  }, [tool, fields, jsonMode, jsonText]);

  function choose(name: string) {
    setSelected(name);
    setFields({});
    setJsonText("{}");
    setJsonMode(false);
    setReviewed(false);
    setResult(null);
    setError(null);
  }

  function toggleJson() {
    if (!jsonMode && built.args) setJsonText(JSON.stringify(built.args, null, 2));
    setJsonMode(!jsonMode);
    setReviewed(false);
  }

  async function run() {
    if (!tool || !built.args) return;
    setBusy(true);
    setError(null);
    setResult(null);
    try {
      const response = await fetch(`${legalApiBase()}/v1/mcp/invocations`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          connection_id: tool.connection_id,
          tool: tool.name,
          arguments: built.args,
          confirm_write: tool.writes && reviewed,
        }),
      });
      const body = await response.json().catch(() => null);
      if (!response.ok) throw new Error(`call refused (${response.status}): ${body?.detail ?? "no detail"}`);
      setResult(body as Invocation);
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "call failed");
    } finally {
      setBusy(false);
    }
  }

  const properties = Object.entries(tool?.input_schema.properties ?? {});
  const required = new Set(tool?.input_schema.required ?? []);
  const call = tool && built.args ? JSON.stringify({ tool: tool.name, arguments: built.args }, null, 2) : null;

  return (
    <div style={{ display: "flex", flexWrap: "wrap", gap: 24, alignItems: "flex-start" }}>
      <nav aria-label="Toolkit tools" style={{ flex: "1 1 220px", maxWidth: 320 }}>
        <input
          type="search"
          placeholder="Filter tools"
          value={filter}
          onChange={(event) => setFilter(event.target.value)}
          style={{ width: "100%", marginBottom: 8 }}
        />
        {visible.map((t) => (
          <div key={t.name} style={{ padding: "4px 0" }}>
            <button
              type="button"
              onClick={() => choose(t.name)}
              style={{ fontWeight: t.name === selected ? 700 : 400, textAlign: "left", background: "none", border: 0, padding: 0, color: "inherit", cursor: "pointer" }}
            >
              {t.title || t.name}
            </button>
            <div style={{ ...muted, ...mono }}>
              {t.name}
              {t.writes ? " · changes the store" : " · read-only"}
            </div>
          </div>
        ))}
      </nav>

      {tool ? (
        <section style={{ flex: "3 1 420px", minWidth: 0 }} data-tool={tool.name}>
          <h2 style={{ marginTop: 0 }}>{tool.title || tool.name}</h2>
          <p style={{ ...muted, ...mono }}>
            {tool.connection_id} / {tool.name} · schema {tool.schema_hash.slice(0, 19)}
          </p>
          <p style={{ whiteSpace: "pre-wrap" }}>{tool.description}</p>
          {tool.writes ? (
            <p style={{ ...box, borderColor: "var(--danger, #b33a3a)" }}>
              This tool can change the Family Law Toolkit store. Review the call below before running it.
            </p>
          ) : null}

          <p>
            <button type="button" onClick={toggleJson}>
              {jsonMode ? "Use the form" : "Edit arguments as JSON"}
            </button>
          </p>

          {jsonMode ? (
            <textarea
              aria-label="Arguments (JSON)"
              value={jsonText}
              onChange={(event) => {
                setJsonText(event.target.value);
                setReviewed(false);
              }}
              rows={10}
              style={{ width: "100%", ...mono }}
            />
          ) : properties.length === 0 ? (
            <p style={muted}>This tool takes no arguments.</p>
          ) : (
            properties.map(([name, prop]) => {
              const kind = kindOf(prop);
              const label = `${name}${required.has(name) ? " *" : ""}`;
              const value = fields[name] ?? "";
              const set = (next: string) => {
                setFields({ ...fields, [name]: next });
                setReviewed(false);
              };
              return (
                <label key={name} style={{ display: "block", margin: "8px 0" }}>
                  <span style={{ display: "block", fontWeight: 600 }}>{label}</span>
                  {prop.description ? <span style={{ display: "block", ...muted }}>{prop.description}</span> : null}
                  {kind === "enum" ? (
                    <select value={value} onChange={(event) => set(event.target.value)}>
                      <option value="">(not set)</option>
                      {prop.enum!.map((option) => (
                        <option key={String(option)} value={String(option)}>
                          {String(option)}
                        </option>
                      ))}
                    </select>
                  ) : kind === "boolean" ? (
                    <select value={value} onChange={(event) => set(event.target.value)}>
                      <option value="">(not set)</option>
                      <option value="true">true</option>
                      <option value="false">false</option>
                    </select>
                  ) : kind === "number" ? (
                    <input type="number" value={value} min={prop.minimum} max={prop.maximum} onChange={(event) => set(event.target.value)} />
                  ) : kind === "string" && (prop.maxLength ?? 0) <= 500 ? (
                    <input type="text" value={value} maxLength={prop.maxLength} onChange={(event) => set(event.target.value)} style={{ width: "100%" }} />
                  ) : (
                    <textarea
                      value={value}
                      rows={kind === "json" ? 4 : 6}
                      placeholder={kind === "json" ? "JSON value" : undefined}
                      onChange={(event) => set(event.target.value)}
                      style={{ width: "100%", ...(kind === "json" ? mono : {}) }}
                    />
                  )}
                </label>
              );
            })
          )}

          <h3>Call</h3>
          {built.problem ? <p>{built.problem}</p> : null}
          {call ? <pre style={{ whiteSpace: "pre-wrap", ...mono, ...box }}>{call}</pre> : null}
          {tool.writes ? (
            <label style={{ display: "block", margin: "8px 0" }}>
              <input type="checkbox" checked={reviewed} onChange={(event) => setReviewed(event.target.checked)} /> I reviewed this
              call; run it against the toolkit store.
            </label>
          ) : null}
          <button type="button" onClick={run} disabled={busy || !built.args || (tool.writes && !reviewed)}>
            {busy ? "Running…" : tool.writes ? "Run this write" : "Run"}
          </button>
          {error ? <p role="alert">{error}</p> : null}

          {result ? (
            <article style={box} aria-label="Result">
              <p style={{ ...muted, ...mono }}>
                {result.state === "ok" ? "Tool ran" : "Tool reported an error"} · {result.id} · {result.completed_at}
                {result.truncated ? " · output truncated" : ""}
              </p>
              {result.text.length === 0 && result.structured == null ? <p>The tool returned no content.</p> : null}
              {result.text.map((text, index) => (
                <pre key={index} style={{ whiteSpace: "pre-wrap", ...mono }}>
                  {pretty(text)}
                </pre>
              ))}
              {result.structured != null ? (
                <details>
                  <summary>Structured result</summary>
                  <pre style={{ whiteSpace: "pre-wrap", ...mono }}>{JSON.stringify(result.structured, null, 2)}</pre>
                </details>
              ) : null}
              <details>
                <summary>Raw content</summary>
                <pre style={{ whiteSpace: "pre-wrap", ...mono }}>{JSON.stringify(result.content, null, 2)}</pre>
              </details>
            </article>
          ) : null}
        </section>
      ) : (
        <p>No tool selected.</p>
      )}
    </div>
  );
}
