// Byline: Claude Code · Opus 5.5 · 2026-10-02
// Family Law Toolkit tools, run from the desk (owner requirement R45). The desk's API
// lists the hosted console's tools over MCP and runs the one the owner fills in and
// clicks Run on. Sibling: ../page.tsx (toolkit records).
import Link from "next/link";
import { legalApiBase } from "@/lib/api/client";
import { ToolRunner, type ToolDefinition } from "@/components/ToolRunner";

type Connection = {
  id: string;
  name: string;
  transport: string;
  configured: boolean;
  reachable: boolean;
  tool_count: number;
  detail: string;
};

async function getJson<T>(path: string): Promise<T> {
  const response = await fetch(`${legalApiBase()}${path}`, { cache: "no-store" });
  if (!response.ok) throw new Error(`legal-api ${path.split("?")[0]} ${response.status}`);
  return response.json();
}

export default async function ToolkitToolsPage() {
  let connection: Connection | null = null;
  let tools: ToolDefinition[] = [];
  let error: string | null = null;
  try {
    [connection] = await getJson<Connection[]>("/v1/mcp/connections");
    if (connection?.configured && connection.reachable) {
      tools = (await getJson<{ items: ToolDefinition[] }>("/v1/mcp/tools")).items;
    }
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <p style={{ letterSpacing: "0.12em", textTransform: "uppercase", color: "var(--text-muted)" }}>Family Law Toolkit</p>
      <h1 style={{ fontFamily: "Georgia, serif", fontWeight: 500 }}>Toolkit tools</h1>
      <p>
        <Link href="/toolkit">← Toolkit records</Link>
      </p>
      {error ? <p>{error}</p> : null}
      {connection && !connection.configured ? <p>The toolkit console connection is not configured.</p> : null}
      {connection?.configured && !connection.reachable ? <p>Toolkit console unreachable. {connection.detail}</p> : null}
      {connection?.reachable ? (
        <p style={{ color: "var(--text-muted)" }} data-tool-count={tools.length}>
          {connection.name} · {connection.transport} · {tools.length} tools
        </p>
      ) : null}
      {tools.length > 0 ? <ToolRunner tools={tools} /> : null}
    </>
  );
}
