// Byline: Claude Code · Fable 5.1 · 2026-09-21
// Read-only desk over the Consignatio catalog. Lifecycle comes from the catalog's
// promotion record, never from a folder name. Bytes stay on B2.
import Link from "next/link";
import { legalApiBase } from "@/lib/api/client";

type Occurrence = Record<string, unknown>;
type CatalogObject = {
  id: string;
  bucket: string;
  key: string;
  size: number | null;
  hash: string | null;
  hash_algorithm: string;
  uploaded_at: string | null;
  lifecycle: "context" | "pending_promotion" | "promoted";
  promotion: Record<string, unknown> | null;
  artifact_url: string | null;
  metadata: unknown;
  occurrences: Occurrence[];
};
type Status = {
  configured: boolean;
  reachable: boolean;
  detail: string;
  generation: Record<string, unknown> | null;
  promotion_binding: "bound" | "unbound";
  retrieval_configured: boolean;
};

const VIEWS = [
  { id: "context", label: "Context" },
  { id: "pending", label: "Pending promotion" },
  { id: "promoted", label: "Promoted evidence" },
] as const;

const muted = { color: "var(--text-muted)" } as const;
const flag = {
  display: "inline-block",
  border: "1px solid var(--border)",
  borderRadius: 4,
  padding: "1px 8px",
  fontSize: 12,
  color: "var(--text-muted)",
} as const;

async function getJson<T>(path: string): Promise<T> {
  const response = await fetch(`${legalApiBase()}${path}`, { cache: "no-store" });
  if (!response.ok) throw new Error(`legal-api ${path.split("?")[0]} ${response.status}`);
  return response.json();
}

function href(params: Record<string, string>): string {
  const query = new URLSearchParams(Object.entries(params).filter(([, v]) => v)).toString();
  return `/evidence-catalog${query ? `?${query}` : ""}`;
}

export default async function EvidenceCatalogPage({
  searchParams,
}: {
  searchParams: Promise<{ view?: string; prefix?: string; q?: string; after?: string; id?: string }>;
}) {
  const params = await searchParams;
  const view = VIEWS.some((v) => v.id === params.view) ? params.view! : "context";
  const prefix = params.prefix ?? "";
  const q = params.q ?? "";

  let status: Status | null = null;
  let items: CatalogObject[] = [];
  let nextAfter: string | null = null;
  let records: Record<string, unknown>[] = [];
  let detail: CatalogObject | null = null;
  let link: { url: string; expires_in: number } | null = null;
  let error: string | null = null;
  try {
    status = await getJson<Status>("/v1/evidence-catalog/status");
    if (status.configured && status.reachable) {
      if (params.id) {
        detail = await getJson<CatalogObject>(
          `/v1/evidence-catalog/objects/${encodeURIComponent(params.id)}`,
        );
        if (status.retrieval_configured) {
          // Signed for this exact object version; short-lived, so it is made per page view.
          link = await getJson(`/v1/evidence-catalog/objects/${encodeURIComponent(params.id)}/link`);
        }
      } else if (view === "context") {
        const page = await getJson<{ items: CatalogObject[]; next_after: string | null }>(
          `/v1/evidence-catalog/objects?${new URLSearchParams({ prefix, q, after: params.after ?? "" })}`,
        );
        items = page.items;
        nextAfter = page.next_after;
      } else {
        records = await getJson(`/v1/evidence-catalog/promotions?state=${view}`);
      }
    }
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <p style={{ letterSpacing: "0.12em", textTransform: "uppercase", ...muted }}>
        Evidence catalog
      </p>
      <h1 style={{ fontFamily: "Georgia, serif", fontWeight: 500 }}>Consignatio catalog · read only</h1>
      <nav style={{ display: "flex", gap: 16, margin: "12px 0" }}>
        {VIEWS.map((v) => (
          <Link key={v.id} href={href({ view: v.id })} style={{ fontWeight: v.id === view ? 700 : 400 }}>
            {v.label}
          </Link>
        ))}
        {status?.promotion_binding === "unbound" ? (
          <span style={flag}>no promotion record in the catalog yet</span>
        ) : null}
        {status && !status.retrieval_configured ? <span style={flag}>B2 retrieval not configured</span> : null}
      </nav>
      {error ? <p>{error}</p> : null}
      {status && !status.configured ? <p>Catalog connection is not configured.</p> : null}
      {status?.configured && !status.reachable ? <p>Catalog unreachable. {status.detail}</p> : null}

      {detail ? (
        <article>
          <p>
            <Link href={href({ view, prefix, q })}>← back</Link>
          </p>
          <h2 style={{ wordBreak: "break-all" }}>{detail.key}</h2>
          <p>
            <span style={flag}>{detail.lifecycle.replace("_", " ")}</span> {detail.bucket} ·{" "}
            {detail.size ?? "?"} bytes · {detail.hash_algorithm} {detail.hash ?? "none"} · uploaded{" "}
            {detail.uploaded_at ?? "unknown"}
          </p>
          <p style={muted}>catalog id {detail.id}</p>
          {link ?? detail.artifact_url ? (
            <p>
              <a href={link?.url ?? detail.artifact_url ?? ""} target="_blank" rel="noreferrer">
                Open from B2
              </a>
              {link ? <span style={muted}> · link valid {Math.round(link.expires_in / 60)} min</span> : null}
            </p>
          ) : null}
          {detail.promotion ? (
            <>
              <h3>Promotion</h3>
              <pre style={{ whiteSpace: "pre-wrap" }}>{JSON.stringify(detail.promotion, null, 2)}</pre>
            </>
          ) : null}
          <h3>Source occurrences ({detail.occurrences.length})</h3>
          {detail.occurrences.map((occ, index) => (
            <div key={index} style={{ borderTop: "1px solid var(--border)", padding: "10px 0" }}>
              <strong>{String(occ.source ?? "")}</strong> · {String(occ.source_path ?? "")}
              <p style={muted}>
                {String(occ.availability ?? "")} · matched by {String(occ.match_basis ?? "")} ·{" "}
                {String(occ.native_hash_kind ?? "")} {String(occ.native_hash ?? "")} · modified{" "}
                {String(occ.recorded_modtime ?? "unknown")}
              </p>
            </div>
          ))}
          <h3>Object metadata</h3>
          <pre style={{ whiteSpace: "pre-wrap" }}>{JSON.stringify(detail.metadata, null, 2)}</pre>
        </article>
      ) : null}

      {!detail && view === "context" && status?.reachable ? (
        <>
          <form action="/evidence-catalog" style={{ display: "flex", gap: 8, margin: "8px 0 16px" }}>
            <input name="prefix" defaultValue={prefix} placeholder="folder under the vault, e.g. v1/Evidence/" />
            <input name="q" defaultValue={q} placeholder="name contains" />
            <button type="submit">Search</button>
          </form>
          {items.length === 0 ? <p>No catalog objects match.</p> : null}
          {items.map((item) => (
            <article key={item.id} style={{ borderTop: "1px solid var(--border)", padding: "10px 0" }}>
              <Link href={href({ view, prefix, q, id: item.id })} style={{ wordBreak: "break-all" }}>
                {item.key}
              </Link>
              <p style={muted}>
                <span style={flag}>{item.lifecycle.replace("_", " ")}</span> {item.size ?? "?"} bytes ·{" "}
                {item.hash_algorithm} {item.hash ?? "none"}
              </p>
            </article>
          ))}
          {nextAfter ? <Link href={href({ view, prefix, q, after: nextAfter })}>Next page →</Link> : null}
        </>
      ) : null}

      {!detail && view !== "context" && status?.reachable ? (
        <>
          {records.length === 0 ? (
            <p>{view === "promoted" ? "No completed promotions." : "No pending promotions."}</p>
          ) : null}
          {records.map((record, index) => (
            <pre key={index} style={{ borderTop: "1px solid var(--border)", whiteSpace: "pre-wrap" }}>
              {JSON.stringify(record, null, 2)}
            </pre>
          ))}
        </>
      ) : null}
    </>
  );
}
