// Byline: Claude Code · Opus 5.5 · 2026-09-27
// Family Law Toolkit records, read through the shared legal-record contract
// (propria.legal-record.v1). The toolkit's case store owns them; this page never
// copies them, and shows the same id and version the toolkit shows.
// Sibling: ../evidence-catalog/page.tsx.
import Link from "next/link";
import { legalApiBase } from "@/lib/api/client";

type Status = { configured: boolean; reachable: boolean; detail: string; contract: string };
type Listing = { table: string; items: { id: string; title: string }[] };
type ToolkitRecord = {
  contract: string;
  owner: string;
  id: string;
  table: string;
  version: string;
  record: Record<string, unknown>;
};

const TABLES = [
  { id: "source", label: "Legal sources" },
  { id: "reference", label: "Cheat sheets and references" },
  { id: "filing", label: "Filed documents" },
  { id: "draft", label: "Drafts" },
  { id: "exhibit", label: "Exhibits" },
  { id: "note", label: "Notes" },
] as const;

const muted = { color: "var(--text-muted)" } as const;
const mono = { fontFamily: "ui-monospace, monospace", fontSize: 12, wordBreak: "break-all" } as const;

async function getJson<T>(path: string): Promise<T> {
  const response = await fetch(`${legalApiBase()}${path}`, { cache: "no-store" });
  if (!response.ok) throw new Error(`legal-api ${path.split("?")[0]} ${response.status}`);
  return response.json();
}

function href(params: Record<string, string>): string {
  const query = new URLSearchParams(Object.entries(params).filter(([, v]) => v)).toString();
  return `/toolkit${query ? `?${query}` : ""}`;
}

function bodyText(record: Record<string, unknown>): string | null {
  for (const key of ["body", "text", "definition"]) {
    const value = record[key];
    if (typeof value === "string" && value.trim()) return value;
  }
  return null;
}

export default async function ToolkitPage({
  searchParams,
}: {
  searchParams: Promise<{ table?: string; id?: string }>;
}) {
  const params = await searchParams;
  const table = TABLES.some((t) => t.id === params.table) ? params.table! : "source";

  let status: Status | null = null;
  let listing: Listing | null = null;
  let detail: ToolkitRecord | null = null;
  let error: string | null = null;
  try {
    status = await getJson<Status>("/v1/toolkit/status");
    if (status.configured && status.reachable) {
      if (params.id) detail = await getJson<ToolkitRecord>(`/v1/toolkit/records/${encodeURIComponent(params.id)}`);
      else listing = await getJson<Listing>(`/v1/toolkit/records?table=${table}`);
    }
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  const body = detail ? bodyText(detail.record) : null;

  return (
    <>
      <p style={{ letterSpacing: "0.12em", textTransform: "uppercase", ...muted }}>Family Law Toolkit</p>
      <h1 style={{ fontFamily: "Georgia, serif", fontWeight: 500 }}>Toolkit records</h1>
      <nav style={{ display: "flex", flexWrap: "wrap", gap: 16, margin: "12px 0" }}>
        {TABLES.map((t) => (
          <Link key={t.id} href={href({ table: t.id })} style={{ fontWeight: t.id === table && !detail ? 700 : 400 }}>
            {t.label}
          </Link>
        ))}
      </nav>
      {error ? <p>{error}</p> : null}
      {status && !status.configured ? <p>Toolkit store connection is not configured.</p> : null}
      {status?.configured && !status.reachable ? <p>Toolkit store unreachable. {status.detail}</p> : null}

      {detail ? (
        <article>
          <p>
            <Link href={href({ table: detail.table })}>← back</Link>
          </p>
          <h2 style={{ wordBreak: "break-word" }}>
            {String(detail.record.title ?? detail.record.citation ?? detail.record.key ?? detail.id)}
          </h2>
          <p style={mono}>id {detail.id}</p>
          <p style={mono}>version {detail.version}</p>
          <p style={muted}>
            {detail.contract} · owned by {detail.owner}
          </p>
          {body ? <pre style={{ whiteSpace: "pre-wrap" }}>{body}</pre> : null}
          <h3>Record</h3>
          <pre style={{ whiteSpace: "pre-wrap", ...mono }}>{JSON.stringify(detail.record, null, 2)}</pre>
        </article>
      ) : null}

      {listing ? (
        <>
          {listing.items.length === 0 ? <p>No {table} records in the toolkit store.</p> : null}
          {listing.items.map((item) => (
            <article key={item.id} style={{ borderTop: "1px solid var(--border)", padding: "8px 0" }}>
              <Link href={href({ table, id: item.id })}>{item.title || item.id}</Link>
              <p style={{ ...muted, ...mono, margin: 0 }}>{item.id}</p>
            </article>
          ))}
        </>
      ) : null}
    </>
  );
}
