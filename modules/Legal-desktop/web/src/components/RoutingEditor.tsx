// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { legalApiBase } from "@/lib/api/client";

type AgentBinding = { backend: string; model: string; surface: string };
type SurfaceRow = { path: string; label: string; group: string; help?: string; advanced?: boolean };

export function RoutingEditor({
  table,
}: {
  table: {
    framework: string;
    notes: string;
    chat: {
      backend: string;
      next_path: string;
      run_path: string;
      confidential_path: string;
      default_confidential_model: string;
    };
    agents: Record<string, AgentBinding>;
    surfaces?: SurfaceRow[];
  };
}) {
  const router = useRouter();
  const [framework, setFramework] = useState(table.framework);
  const [chatBackend, setChatBackend] = useState(table.chat.backend);
  const [runPath, setRunPath] = useState(table.chat.run_path);
  const [confidentialPath, setConfidentialPath] = useState(table.chat.confidential_path);
  const [confidentialModel, setConfidentialModel] = useState(
    table.chat.default_confidential_model,
  );
  const [agents, setAgents] = useState(table.agents);
  const [surfaces, setSurfaces] = useState<SurfaceRow[]>(table.surfaces ?? []);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const roles = useMemo(() => Object.keys(agents), [agents]);

  async function save() {
    setBusy(true);
    setError(null);
    try {
      const current = await fetch(`${legalApiBase()}/v1/routing`, { cache: "no-store" });
      if (!current.ok) throw new Error(`routing ${current.status}`);
      const body = await current.json();
      body.framework = framework;
      body.chat.backend = chatBackend;
      body.chat.run_path = runPath;
      body.chat.confidential_path = confidentialPath;
      body.chat.default_confidential_model = confidentialModel;
      body.agents = agents;
      if (surfaces.length) body.surfaces = surfaces;
      const response = await fetch(`${legalApiBase()}/v1/routing`, {
        method: "PUT",
        headers: { "content-type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!response.ok) throw new Error(`save ${response.status}`);
      router.refresh();
    } catch (exc) {
      setError(exc instanceof Error ? exc.message : "routing save failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        void save();
      }}
      style={{ display: "grid", gap: 10, margin: "16px 0 32px" }}
    >
      <p className="section-eyebrow">Routing table</p>
      <p className="dim">
        Set the framework, chat backend, model paths, and agent models used by this workspace.
        Changes save to the workspace routing overlay.
      </p>
      <label>
        Framework label
        <input
          value={framework}
          onChange={(event) => setFramework(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Chat backend
        <input
          value={chatBackend}
          onChange={(event) => setChatBackend(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Agent-run path
        <input
          value={runPath}
          onChange={(event) => setRunPath(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Confidential invoke path
        <input
          value={confidentialPath}
          onChange={(event) => setConfidentialPath(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      <label>
        Default confidential model
        <input
          value={confidentialModel}
          onChange={(event) => setConfidentialModel(event.target.value)}
          style={{ display: "block", width: "100%", marginTop: 4 }}
        />
      </label>
      {surfaces.map((item, index) => (
        <label key={item.path}>
          {item.label} path
          <input
            value={item.path}
            onChange={(event) => {
              const next = [...surfaces];
              next[index] = { ...item, path: event.target.value };
              setSurfaces(next);
            }}
            style={{ display: "block", width: "100%", marginTop: 4 }}
          />
        </label>
      ))}
      {roles.map((role) => (
        <label key={role}>
          {role} model
          <input
            value={agents[role]?.model ?? ""}
            onChange={(event) =>
              setAgents({
                ...agents,
                [role]: { ...agents[role], model: event.target.value },
              })
            }
            style={{ display: "block", width: "100%", marginTop: 4 }}
          />
        </label>
      ))}
      {error ? <p style={{ color: "var(--status-warn)" }}>{error}</p> : null}
      <button type="submit" disabled={busy}>
        {busy ? "Saving…" : "Save routing overlay"}
      </button>
    </form>
  );
}
