// Byline: Claude Code · Opus 5.5 · 2026-09-27; factors and tools link 2026-10-02
// Updated by: OpenAI Codex · GPT-6 · 2026-10-04 — expose all backend-supported toolkit tables.
// Updated by: OpenAI Codex · GPT-6-Luna · 2026-10-05 — show scoped B2 originals outside record bodies.
// Family Law Toolkit records, read through the shared legal-record contract
// (propria.legal-record.v1). The toolkit's case store owns them; this page never
// copies them, and shows the same id and version the toolkit shows.
// Sibling: ../evidence-catalog/page.tsx.
import Link from "next/link";
import { legalApiBase } from "@/lib/api/client";
import { LibraryProposalEditor } from "@/components/LibraryProposalEditor";
import { SharedCaseRecordEditor, type PersonalTable } from "@/components/SharedCaseRecordEditor";

type Status = { configured: boolean; reachable: boolean; detail: string; contract: string };
type Listing = { table: string; items: { id: string; title: string }[] };
type ToolkitRecord = {
  contract: string;
  owner: string;
  id: string;
  table: string;
  version: string;
  record: Record<string, unknown>;
  original_links?: unknown;
};

type OriginalLink = {
  binding_id: string;
  version_id: string;
  sha256: string;
  bytes: number;
  content_type: "application/pdf";
  href: string;
};

const ORIGINAL_LINK_HOST = "family-court.tilapia-skilift.ts.net";
const MAX_ORIGINAL_BYTES = 20 * 1024 * 1024;
const MAX_VERSION_ID_BYTES = 2048;

/** Accept only exact hosted PDF links and rebuild their URL from validated identifiers.
 * Input is the separate API envelope field; output is bounded safe links for rendering.
 * It performs no I/O and prevents record content or arbitrary server URLs from becoming navigation.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-05.
 */
function safeOriginalLinks(value: unknown): OriginalLink[] {
  if (!Array.isArray(value)) return [];
  return value.slice(0, 8).flatMap((item: unknown) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) return [];
    const candidate = item as Record<string, unknown>;
    const { binding_id: bindingId, version_id: versionId, sha256, bytes, content_type: contentType, href } = candidate;
    if (typeof bindingId !== "string" || !/^library_file:[a-f0-9]{64}$/.test(bindingId) ||
      typeof versionId !== "string" || !versionId || versionId === "null" || /[\r\n\0]/.test(versionId) ||
      new TextEncoder().encode(versionId).byteLength > MAX_VERSION_ID_BYTES ||
      typeof sha256 !== "string" || !/^[a-f0-9]{64}$/.test(sha256) ||
      typeof bytes !== "number" || !Number.isSafeInteger(bytes) || bytes <= 0 || bytes > MAX_ORIGINAL_BYTES ||
      contentType !== "application/pdf" || typeof href !== "string") return [];
    try {
      const supplied = new URL(href);
      const keys = [...supplied.searchParams.keys()].sort();
      if (supplied.protocol !== "https:" || supplied.hostname !== ORIGINAL_LINK_HOST || supplied.port ||
        supplied.username || supplied.password || supplied.pathname !== "/api/library/original" ||
        supplied.hash || keys.join(",") !== "binding_id,version_id" ||
        supplied.searchParams.getAll("binding_id").length !== 1 ||
        supplied.searchParams.getAll("version_id").length !== 1 ||
        supplied.searchParams.get("binding_id") !== bindingId ||
        supplied.searchParams.get("version_id") !== versionId) return [];
    } catch {
      return [];
    }
    const query = new URLSearchParams({ binding_id: bindingId, version_id: versionId });
    return [{ binding_id: bindingId, version_id: versionId, sha256, bytes, content_type: "application/pdf", href: "https://" + ORIGINAL_LINK_HOST + "/api/library/original?" + query }];
  });
}

const TABLES = [
  { id: "source", label: "Legal sources" },
  { id: "reference", label: "Cheat sheets and references" },
  { id: "filing", label: "Filed documents" },
  { id: "draft", label: "Drafts" },
  { id: "order", label: "Orders" },
  { id: "exhibit", label: "Exhibits" },
  { id: "note", label: "Notes" },
  { id: "memo", label: "Memos" },
  { id: "factor", label: "Best-interest factors" },
  { id: "person", label: "People" },
  { id: "child", label: "Children" },
  { id: "court", label: "Court" },
  { id: "hearing", label: "Hearings" },
  { id: "deadline", label: "Dates and deadlines" },
  { id: "event", label: "Events" },
  { id: "court_event", label: "Court events" },
  { id: "message", label: "Messages" },
  { id: "evidence_log", label: "Evidence log" },
  { id: "eval", label: "Evaluations" },
  { id: "case_status", label: "Case status" },
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
  searchParams: Promise<{ table?: string; id?: string; new?: string; kind?: string }>;
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
      else if (params.new !== "1") listing = await getJson<Listing>(`/v1/toolkit/records?table=${table}`);
    }
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  const body = detail ? bodyText(detail.record) : null;
  const personalSource = detail?.table === "source" && detail.record.kind === "case_document";
  const creatingPersonal = table !== "reference" && (table !== "source" || params.kind === "case_document");

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
      {!detail ? (
        <p><Link href={href({ table, new: "1" })}>{table === "source" || table === "reference" ? "Propose a new" : "Create a personal"} {table}</Link>
          {table === "source" ? <> · <Link href={href({ table, new: "1", kind: "case_document" })}>Add a personal case document</Link></> : null}
        </p>
      ) : null}
      <p>
        <Link href="/toolkit/tools">Run toolkit tools →</Link>
      </p>
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
          {safeOriginalLinks(detail.original_links).map((link, index) => (
            <p key={link.binding_id + ":" + link.version_id}>
              <a href={link.href} target="_blank" rel="noopener noreferrer">Open Case Bible original{index ? " (" + (index + 1) + ")" : ""}</a>
            </p>
          ))}
          {body ? <pre style={{ whiteSpace: "pre-wrap" }}>{body}</pre> : null}
          <h3>Record</h3>
          <pre style={{ whiteSpace: "pre-wrap", ...mono }}>{JSON.stringify(detail.record, null, 2)}</pre>
          {(detail.table === "source" && !personalSource) || detail.table === "reference" ? (
            <LibraryProposalEditor key={detail.id} table={detail.table as "source" | "reference"} initialId={detail.id} expectedVersion={detail.version} initialPatch={detail.record} />
          ) : (
            <SharedCaseRecordEditor key={detail.id} table={detail.table as PersonalTable} initialId={detail.id} expectedVersion={detail.version} initialRecord={detail.record} />
          )}
        </article>
      ) : null}

      {params.new === "1" ? creatingPersonal ? (
        <SharedCaseRecordEditor key={`${table}:new-personal`} table={table as PersonalTable} />
      ) : (
        <LibraryProposalEditor key={`${table}:new-library`} table={table as "source" | "reference"} />
      ) : null}

      {listing && listing.table === "factor" ? (
        <section aria-label="Best-interest factors">
          <h2 style={{ fontFamily: "Georgia, serif", fontWeight: 500 }}>Best-interest factors (MCL 722.23)</h2>
          <p style={muted}>The {listing.items.length} factors the court weighs, as the toolkit store holds them.</p>
        </section>
      ) : null}

      {listing ? (
        <>
          {listing.items.length === 0 ? <p>No {table} records in the toolkit store.</p> : null}
          {listing.items.map((item) => (
            <article key={item.id} style={{ borderTop: "1px solid var(--border)", padding: "8px 0" }}>
              <Link href={href({ table, id: item.id })}>
                {table === "factor" ? `(${item.id.split(":")[1]}) ` : ""}
                {item.title || item.id}
              </Link>
              <p style={{ ...muted, ...mono, margin: 0 }}>{item.id}</p>
            </article>
          ))}
        </>
      ) : null}
    </>
  );
}
