"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { legalApiBase } from "@/lib/api/client";
import { parseProbataLink } from "@/lib/probata-link";
import styles from "./ClaimsWorkspace.module.css";

type ClaimKind = "assertion" | "allegation" | "question" | "theory";
type Relationship = "supports" | "partial" | "contradicts" | "context";
type FollowupKind = "locate_document" | "investigate" | "research" | "discovery";
type ProbataOrigin = { system: "probata"; kind: "entity" | "event"; record_id: string; record_version: string };
type ProbataRecord = { origin: ProbataOrigin; title: string; record: Record<string, unknown> };
type SourceReference = {
  package_id: string; item_id: string; assertion_id: string; assertion_version: number;
  manifest_hash: string; content_hash: string; span_locator: string; custody_locator: string;
};
type EvidenceLink = {
  link_id: string; relationship: Relationship; source: SourceReference | null;
  context_reference: string; note: string; validation_status: string; active: boolean;
};
type Gap = { gap_id: string; description: string; status: "open" | "resolved" };
type Investigation = {
  state: "prepared" | "acknowledged"; request_id: string | null;
  remote_status: "received" | "running" | "completed" | "failed" | "cancelled" | null;
  remote_updated_at: string | null; last_error: string;
  request: { question: string; mode: string; matter_id: string; court_case_id: string };
  results: { summary: string; sources: { kind: string; record_id: string; record_version: string }[]; tool: string; run_id: string }[];
};
type Followup = {
  followup_id: string; kind: FollowupKind; description: string;
  gap_id: string | null; status: "open" | "done" | "cancelled";
  investigation?: Investigation | null;
};
type Claim = {
  claim_id: string; revision: number; text: string; kind: ClaimKind; claimant: string;
  response: string; links: EvidenceLink[]; gaps: Gap[]; followups: Followup[];
  evidence_status: string; updated_at: string;
  origin?: ProbataOrigin | null; origin_state?: "none" | "unchanged" | "changed" | "unavailable";
  origin_record?: { origin: ProbataOrigin; title: string; payload: Record<string, unknown> } | null;
};
type GapReport = {
  claim_id: string; claim_text: string; revision: number; evidence_status: string;
  gap_id?: string | null; description: string; followups: Followup[];
};
type ClaimFields = Pick<Claim, "text" | "kind" | "claimant" | "response">;
type History = { revision: number; actor: string; action: string; occurred_at: string; record: Claim };
const emptyFields: ClaimFields = { text: "", kind: "assertion", claimant: "", response: "" };
const kinds: Record<ClaimKind, string> = { assertion: "Factual statement", allegation: "Allegation", question: "Question", theory: "Working theory" };
const relationships: Record<Relationship, string> = { supports: "Supports", partial: "Partially supports", contradicts: "Contradicts", context: "Context" };
const followupKinds: Record<FollowupKind, string> = { locate_document: "Locate a document", investigate: "Plan an evidence investigation", research: "Research a legal question", discovery: "Prepare discovery" };
const evidenceLabels: Record<string, string> = {
  evidence_needed: "Evidence needed", evidence_linked: "Evidence linked", partially_supported: "Partially supported",
  conflicting_evidence: "Conflicting evidence", context_only: "Context only",
};
const investigationLabels: Record<string, string> = {
  received: "Received by Probata", running: "In progress", completed: "Completed",
  failed: "Needs attention", cancelled: "Cancelled in Probata",
};
function fieldsOf(claim: Claim): ClaimFields { return { text: claim.text, kind: claim.kind, claimant: claim.claimant, response: claim.response }; }
function statusLabel(status: string) { return evidenceLabels[status] ?? "Review needed"; }
function fieldLabel(value: string) { return value.replace(/_/g, " ").replace(/([a-z])([A-Z])/g, "$1 $2").replace(/^./, letter => letter.toUpperCase()); }

function RecordFields({ record, depth = 0 }: { record: unknown; depth?: number }) {
  if (record === null || record === undefined) return <span className={styles.muted}>Not recorded</span>;
  if (typeof record === "boolean") return <span>{record ? "Yes" : "No"}</span>;
  if (typeof record !== "object") return <span>{String(record)}</span>;
  if (depth >= 4) return <pre className={styles.quote}>{JSON.stringify(record, null, 2)}</pre>;
  return <dl className={styles.recordFields}>{Object.entries(record).map(([key, value]) => <div key={key}><dt>{Array.isArray(record) ? `Item ${Number(key) + 1}` : fieldLabel(key)}</dt><dd>{value !== null && typeof value === "object" ? <details><summary>{Array.isArray(value) ? `${value.length} items` : "Details"}</summary><RecordFields record={value} depth={depth + 1} /></details> : <RecordFields record={value} depth={depth + 1} />}</dd></div>)}</dl>;
}

class RequestError extends Error {
  constructor(message: string, readonly status: number) { super(message); }
}
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${legalApiBase()}${path}`, { cache: "no-store", ...init });
  if (!response.ok) {
    let message = `Unable to complete this request (${response.status}).`;
    try {
      const body = await response.json();
      if (typeof body.detail === "string") message = body.detail;
      else if (typeof body.detail?.message === "string") message = body.detail.message;
    } catch { /* Keep a readable message for proxy failures. */ }
    throw new RequestError(message, response.status);
  }
  return response.json();
}
function json(method: string, value: unknown): RequestInit {
  return { method, headers: { "content-type": "application/json" }, body: JSON.stringify(value) };
}

function SourceInspector({ source }: { source: SourceReference }) {
  return <details className={styles.inspector}><summary>Source details</summary><dl>
    <dt>Package</dt><dd>{source.package_id}</dd><dt>Item</dt><dd>{source.item_id}</dd>
    <dt>Assertion</dt><dd>{source.assertion_id}</dd><dt>Version</dt><dd>{source.assertion_version}</dd>
    <dt>Passage</dt><dd>{source.span_locator}</dd><dt>Custody location</dt><dd>{source.custody_locator}</dd>
    <dt>Content hash</dt><dd>{source.content_hash}</dd><dt>Manifest hash</dt><dd>{source.manifest_hash}</dd>
  </dl></details>;
}

export function ClaimsWorkspace() {
  const [claims, setClaims] = useState<Claim[]>([]);
  const [report, setReport] = useState<GapReport[]>([]);
  const [sources, setSources] = useState<SourceReference[]>([]);
  const [sourcesMessage, setSourcesMessage] = useState("");
  const [selected, setSelected] = useState<Claim | null>(null);
  const [incomingRecord, setIncomingRecord] = useState<ProbataRecord | null>(null);
  const [creating, setCreating] = useState(false);
  const [fields, setFields] = useState<ClaimFields>(emptyFields);
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState("");
  const [tab, setTab] = useState<"claims" | "gaps" | "probata">("claims");
  const [recordKind, setRecordKind] = useState<"event" | "entity">("event");
  const [probataRecords, setProbataRecords] = useState<ProbataRecord[]>([]);
  const [probataAvailable, setProbataAvailable] = useState<boolean | null>(null);
  const [probataLoading, setProbataLoading] = useState(false);
  const [probataError, setProbataError] = useState("");
  const [probataUpdated, setProbataUpdated] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [conflict, setConflict] = useState(false);
  const [notice, setNotice] = useState("");
  const [sourceIndex, setSourceIndex] = useState("");
  const [relationship, setRelationship] = useState<Relationship>("supports");
  const [contextReference, setContextReference] = useState("");
  const [linkNote, setLinkNote] = useState("");
  const [gapText, setGapText] = useState("");
  const [followupText, setFollowupText] = useState("");
  const [followupKind, setFollowupKind] = useState<FollowupKind>("locate_document");
  const [followupGap, setFollowupGap] = useState("");
  const [history, setHistory] = useState<History[] | null>(null);
  const generation = useRef(0);
  const listingGeneration = useRef(0);
  const probataGeneration = useRef(0);
  const dirty = JSON.stringify(fields) !== JSON.stringify(selected ? fieldsOf(selected) : emptyFields)
    || Boolean(contextReference.trim() || linkNote.trim() || sourceIndex || gapText.trim() || followupText.trim());
  const dirtyRef = useRef(dirty);
  dirtyRef.current = dirty;

  const refreshSources = useCallback(async () => {
    try {
      const result = await request<{ available: boolean; reason: string | null; items: SourceReference[] }>("/v1/claim-sources");
      setSources(result.items);
      setSourcesMessage(!result.available ? "No accepted evidence package is available yet. You can record claims, gaps and follow-up plans now." : result.items.length === 0 ? "No accepted passages are available to link yet. You can record the evidence you need below." : "");
    } catch { setSourcesMessage("Evidence sources are unavailable. Your claims and follow-up plans remain accessible."); }
  }, []);

  const refresh = useCallback(async () => {
    const current = ++listingGeneration.current;
    const [rows, gaps] = await Promise.all([request<Claim[]>("/v1/claims"), request<GapReport[]>("/v1/claim-gaps")]);
    if (current === listingGeneration.current) { setClaims(rows); setReport(gaps); }
    return rows;
  }, []);

  const refreshProbata = useCallback(async () => {
    const current = ++probataGeneration.current;
    setProbataLoading(true); setProbataError("");
    try {
      const params = new URLSearchParams({ kind: recordKind, q: query });
      const result = await request<{ available: boolean; reason: string | null; records: ProbataRecord[] }>(`/v1/probata/records?${params}`);
      if (current !== probataGeneration.current) return;
      setProbataAvailable(result.available); setProbataRecords(result.records);
      setProbataUpdated(new Date().toLocaleTimeString());
    } catch {
      if (current === probataGeneration.current) { setProbataError("Probata records could not be loaded. Your saved legal work remains available."); setProbataAvailable(false); }
    } finally { if (current === probataGeneration.current) setProbataLoading(false); }
  }, [recordKind, query]);

  useEffect(() => {
    if (tab !== "probata") return;
    const initial = window.setTimeout(() => void refreshProbata(), 250);
    const poll = window.setInterval(() => { if (document.visibilityState === "visible") void refreshProbata(); }, 30000);
    const focus = () => void refreshProbata();
    window.addEventListener("focus", focus);
    return () => { clearTimeout(initial); clearInterval(poll); window.removeEventListener("focus", focus); probataGeneration.current += 1; };
  }, [tab, refreshProbata]);

  useEffect(() => {
    let active = true;
    void Promise.all([request<Claim[]>("/v1/claims"), request<GapReport[]>("/v1/claim-gaps")])
      .then(([rows, gaps]) => { if (active) { setClaims(rows); setReport(gaps); } })
      .catch((exc) => { if (active) setError(exc.message); })
      .finally(() => { if (active) setLoading(false); });
    void request<{ available: boolean; reason: string | null; items: SourceReference[] }>("/v1/claim-sources")
      .then((result) => { if (active) { setSources(result.items); setSourcesMessage(!result.available ? "No accepted evidence package is available yet. You can record claims, gaps and follow-up plans now." : result.items.length === 0 ? "No accepted passages are available to link yet. You can record the evidence you need below." : ""); } })
      .catch(() => { if (active) setSourcesMessage("Evidence sources are unavailable. Your claims and follow-up plans remain accessible."); });
    return () => { active = false; generation.current += 1; listingGeneration.current += 1; };
  }, []);

  useEffect(() => {
    const link = parseProbataLink(window.location.search);
    if (!link) {
      if (new URLSearchParams(window.location.search).has("probata_id")) setError("The Probata link has an invalid record identity.");
      return;
    }
    let active = true;
    const current = generation.current;
    setBusy(true);
    const params = new URLSearchParams({ kind: link.kind, record_id: link.recordId });
    void (async () => {
      try {
        const saved = await request<Claim | null>(`/v1/claims/by-origin?${params}`);
        if (!active || current !== generation.current || dirtyRef.current) return;
        if (saved) {
          setSelected(saved); setFields(fieldsOf(saved)); setCreating(false); setTab("claims");
          setNotice("Opened the saved legal response for this Probata record.");
          return;
        }
        const listing = await request<{ available: boolean; records: ProbataRecord[] }>(`/v1/probata/records?${params}`);
        if (!active || current !== generation.current || dirtyRef.current) return;
        const source = listing.records.find(row => row.origin.kind === link.kind && row.origin.record_id === link.recordId);
        if (!listing.available || !source) throw new Error("This Probata record is unavailable. No legal response was created.");
        setIncomingRecord(source);
      } catch (exc) {
        if (active && current === generation.current) setError(exc instanceof Error ? exc.message : "Unable to open the linked Probata record.");
      } finally { if (active && current === generation.current) setBusy(false); }
    })();
    return () => { active = false; };
  }, []);

  useEffect(() => {
    const beforeUnload = (event: BeforeUnloadEvent) => { if (dirtyRef.current) { event.preventDefault(); event.returnValue = ""; } };
    const beforeNavigate = (event: MouseEvent) => {
      const anchor = event.target instanceof Element ? event.target.closest("a[href]") as HTMLAnchorElement | null : null;
      if (!dirtyRef.current || !anchor || anchor.target === "_blank" || anchor.download || event.ctrlKey || event.metaKey || event.shiftKey || anchor.getAttribute("href")?.startsWith("#")) return;
      if (!window.confirm("Leave this page and discard unsaved changes?")) { event.preventDefault(); event.stopPropagation(); }
    };
    window.addEventListener("beforeunload", beforeUnload);
    document.addEventListener("click", beforeNavigate, true);
    return () => { window.removeEventListener("beforeunload", beforeUnload); document.removeEventListener("click", beforeNavigate, true); };
  }, []);

  function resetForms() {
    setSourceIndex(""); setContextReference(""); setLinkNote(""); setGapText(""); setFollowupText(""); setFollowupGap("");
    setHistory(null); setError(""); setConflict(false); setNotice("");
  }
  async function selectClaim(id: string) {
    if (busy || (dirty && !window.confirm("Discard unsaved changes and open this claim?"))) return;
    const current = ++generation.current;
    setBusy(true); setError("");
    try {
      const claim = await request<Claim>(`/v1/claims/${encodeURIComponent(id)}`);
      if (generation.current !== current) return;
      resetForms(); setIncomingRecord(null); setSelected(claim); setFields(fieldsOf(claim)); setCreating(false); setTab("claims");
    } catch (exc) { if (generation.current === current) setError(exc instanceof Error ? exc.message : "Unable to open this claim."); }
    finally { if (generation.current === current) setBusy(false); }
  }
  function newClaim() {
    if (busy || (dirty && !window.confirm("Discard unsaved changes and start a new claim?"))) return;
    generation.current += 1; resetForms(); setIncomingRecord(null); setSelected(null); setFields(emptyFields); setCreating(true); setTab("claims");
  }
  function showError(exc: unknown) {
    setConflict(exc instanceof RequestError && exc.status === 409);
    setError(exc instanceof Error ? exc.message : "Unable to save. Your unsaved changes are still here.");
  }
  async function saveClaim() {
    if (busy) return;
    const current = generation.current;
    listingGeneration.current += 1;
    setBusy(true); setError(""); setConflict(false);
    try {
      const claim = await request<Claim>(selected ? `/v1/claims/${selected.claim_id}` : "/v1/claims",
        json(selected ? "PATCH" : "POST", { ...fields, ...(selected ? { expected_revision: selected.revision } : {}) }));
      if (current !== generation.current) return;
      setSelected(claim); setFields(fieldsOf(claim)); setCreating(false); setHistory(null); setNotice("Claim saved.");
      setClaims((rows) => [claim, ...rows.filter(row => row.claim_id !== claim.claim_id)]);
      try { await refresh(); } catch { setNotice("Claim saved. The list could not refresh; retry when the connection returns."); }
    } catch (exc) { if (current === generation.current) showError(exc); }
    finally { if (current === generation.current) setBusy(false); }
  }
  async function mutate(path: string, method: string, body: Record<string, unknown>, message: string, after?: () => void) {
    if (!selected || busy) return;
    const current = generation.current;
    listingGeneration.current += 1;
    setBusy(true); setError(""); setConflict(false);
    try {
      const claim = await request<Claim>(`/v1/claims/${selected.claim_id}${path}`, json(method, { ...body, expected_revision: selected.revision }));
      if (current !== generation.current) return;
      setSelected(claim); setFields(fieldsOf(claim)); setHistory(null); setNotice(message); after?.();
      setClaims(rows => rows.map(row => row.claim_id === claim.claim_id ? claim : row));
      try { await refresh(); } catch { setNotice(`${message} The list could not refresh; retry when the connection returns.`); }
    } catch (exc) { if (current === generation.current) showError(exc); }
    finally { if (current === generation.current) setBusy(false); }
  }
  async function loadHistory() {
    if (!selected || busy) return;
    const current = generation.current;
    setBusy(true); setError("");
    try {
      const result = await request<History[]>(`/v1/claims/${selected.claim_id}/history`);
      if (current === generation.current) setHistory(result);
    } catch (exc) { if (current === generation.current) showError(exc); }
    finally { if (current === generation.current) setBusy(false); }
  }
  async function investigationAction(followupId: string, refreshStatus = false) {
    if (!selected || !canMutate) return;
    const current = generation.current;
    const claimId = selected.claim_id;
    listingGeneration.current += 1;
    setBusy(true); setError(""); setConflict(false); setNotice("");
    try {
      const action = refreshStatus ? "refresh-status" : "dispatch";
      const claim = await request<Claim>(`/v1/claims/${claimId}/followups/${followupId}/${action}`,
        json("POST", { expected_revision: selected.revision }));
      if (current !== generation.current) return;
      setSelected(claim); setFields(fieldsOf(claim)); setHistory(null);
      setClaims(rows => rows.map(row => row.claim_id === claimId ? claim : row));
      const result = claim.followups.find(row => row.followup_id === followupId)?.investigation;
      if (result?.last_error) setNotice(result.last_error);
      else setNotice(refreshStatus ? "Investigation status refreshed." : result?.request_id ? "Probata received the investigation request." : "Request prepared. Retry sending when the connection returns.");
      try { await refresh(); } catch { /* The returned request state is already saved. */ }
    } catch (exc) {
      if (current !== generation.current) return;
      showError(exc);
      // A lost HTTP acknowledgement can still leave a durable prepared request.
      // Recover that state without clearing unsaved secondary forms.
      try {
        const latest = await request<Claim>(`/v1/claims/${claimId}`);
        if (current === generation.current) {
          setSelected(latest); setFields(fieldsOf(latest));
          setClaims(rows => rows.map(row => row.claim_id === claimId ? latest : row));
        }
      } catch { /* Keep the current response and readable error until retry. */ }
    } finally { if (current === generation.current) setBusy(false); }
  }
  async function openProbata(record: ProbataRecord) {
    if (busy || (dirty && !window.confirm("Discard unsaved changes and open the legal response for this Probata record?"))) return;
    const current = ++generation.current;
    listingGeneration.current += 1;
    setBusy(true); setError("");
    try {
      const claim = await request<Claim>("/v1/claims/from-probata", json("POST", { origin: record.origin }));
      if (current !== generation.current) return;
      resetForms(); setSelected(claim); setFields(fieldsOf(claim)); setCreating(false); setTab("claims");
      setIncomingRecord(null);
      setNotice("Legal response opened. The Probata record stays linked to its source.");
      setClaims(rows => [claim, ...rows.filter(row => row.claim_id !== claim.claim_id)]);
      try { await refresh(); } catch { setNotice("Legal response opened. The list could not refresh."); }
    } catch (exc) { if (current === generation.current) showError(exc); }
    finally { if (current === generation.current) setBusy(false); }
  }
  async function refreshOrigin() {
    if (!selected || busy) return;
    const current = generation.current;
    setBusy(true); setError("");
    try {
      const claim = await request<Claim>(`/v1/claims/${selected.claim_id}`);
      if (current === generation.current) setSelected(previous => previous ? { ...previous, origin_record: claim.origin_record, origin_state: claim.origin_state } : null);
    } catch (exc) { if (current === generation.current) showError(exc); }
    finally { if (current === generation.current) setBusy(false); }
  }
  useEffect(() => {
    if (!selected?.origin || tab !== "claims" || busy) return;
    const id = selected.claim_id;
    const current = generation.current;
    let active = true;
    const update = async () => {
      if (document.visibilityState !== "visible") return;
      try {
        const claim = await request<Claim>(`/v1/claims/${id}`);
        if (active && current === generation.current) setSelected(previous =>
          previous?.claim_id === id ? { ...previous, origin_record: claim.origin_record, origin_state: claim.origin_state } : previous);
      } catch { /* Keep the last source view and all unsaved legal fields. */ }
    };
    const poll = window.setInterval(() => void update(), 30000);
    const focus = () => void update();
    window.addEventListener("focus", focus);
    return () => { active = false; clearInterval(poll); window.removeEventListener("focus", focus); };
  }, [selected?.claim_id, Boolean(selected?.origin), tab, busy]);
  const fieldsDirty = JSON.stringify(fields) !== JSON.stringify(selected ? fieldsOf(selected) : emptyFields);
  const canMutate = Boolean(selected && !busy && !fieldsDirty && !conflict);
  const pickedSource = sourceIndex === "" ? null : sources[Number(sourceIndex)] ?? null;
  const visible = claims.filter(claim => (!filter || claim.evidence_status === filter) && `${claim.text} ${claim.claimant} ${claim.response} ${claim.origin_record?.title ?? ""}`.toLowerCase().includes(query.toLowerCase()));
  const gapsVisible = report.filter(gap => (!filter || gap.evidence_status === filter) && `${gap.claim_text} ${gap.description}`.toLowerCase().includes(query.toLowerCase()));

  return <div className={styles.workspace}>
    <header className={styles.heading}><div><h1>Claims and evidence</h1><p className={styles.intro}>Connect statements and allegations to exact sources, track contradictions, and plan what to find next.</p></div><button className={styles.primary} onClick={newClaim} disabled={busy}>Add claim</button></header>
    <div className={styles.metrics}><span>{claims.length} claims</span><span>{claims.filter(claim => ["evidence_needed", "context_only"].includes(claim.evidence_status)).length} needing evidence</span><span>{claims.filter(claim => claim.evidence_status === "conflicting_evidence").length} with conflicting evidence</span></div>
    <div className={styles.tabs} aria-label="Workspace views"><button aria-pressed={tab === "claims"} onClick={() => setTab("claims")}>Claims</button><button aria-pressed={tab === "gaps"} onClick={() => setTab("gaps")}>Gaps and follow-up ({report.length})</button><button aria-pressed={tab === "probata"} onClick={() => setTab("probata")}>Probata records</button><button disabled={busy || loading || probataLoading} onClick={() => { setError(""); if (tab === "probata") void refreshProbata(); else void refresh().catch(showError); }}>Refresh list</button></div>
    <div className={styles.toolbar}><label>{tab === "probata" ? "Search Probata records" : "Search claims"}<input type="search" value={query} onChange={event => setQuery(event.target.value)} placeholder={tab === "probata" ? "Event or person" : "Statement, person or response"} /></label>{tab === "probata" ? <label>Record category<select value={recordKind} onChange={event => { setProbataRecords([]); setRecordKind(event.target.value as "entity" | "event"); }}><option value="event">Events</option><option value="entity">People and entities</option></select></label> : <label>Evidence status<select value={filter} onChange={event => setFilter(event.target.value)}><option value="">All statuses</option>{Object.entries(evidenceLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>}</div>
    {error && <div className={`${styles.notice} ${styles.error}`} role="alert"><p>{conflict ? "This claim changed in another session. Your unsaved entries are still here." : error}</p>{conflict && selected && <button onClick={() => void selectClaim(selected.claim_id)} disabled={busy}>Reload latest claim</button>}</div>}
    {notice && <p className={styles.notice} role="status">{notice}</p>}
    {incomingRecord && <section className={styles.item} aria-label="Record opened from Probata"><div className={styles.itemHeader}><h2>{incomingRecord.title || `Probata ${incomingRecord.origin.kind}`}</h2><span className={styles.badge}>Shared {incomingRecord.origin.kind}</span></div><p>This record has no legal response yet.</p><details className={styles.inspector}><summary>Source record</summary><p className={styles.locator}>{incomingRecord.origin.record_id}</p><RecordFields record={incomingRecord.record} /></details><button disabled={busy} onClick={() => void openProbata(incomingRecord)}>Add legal response</button></section>}
    {loading ? <p role="status">Loading claims…</p> : tab === "probata" ? <section aria-label="Probata shared records">
      <p className={styles.muted}>Read events and entities from Probata and add a linked legal response. {probataUpdated ? `Last refreshed ${probataUpdated}.` : ""}</p>
      {probataLoading && <p role="status">Refreshing Probata records…</p>}
      {probataError && <p className={styles.notice} role="alert">{probataError}</p>}
      {!probataError && probataAvailable === false && <p className={styles.empty}>Probata records are currently unavailable. Refresh to try again.</p>}
      {!probataLoading && probataAvailable && probataRecords.length === 0 && <p className={styles.empty}>No {recordKind === "event" ? "events" : "people or entities"} match this view.</p>}
      {probataRecords.map(record => <article className={styles.item} key={`${record.origin.kind}-${record.origin.record_id}`}><div className={styles.itemHeader}><h2>{record.title || (record.origin.kind === "event" ? "Event" : "Entity")}</h2><span className={styles.badge}>Shared {record.origin.kind}</span></div><details className={styles.inspector}><summary>Source record</summary><p className={styles.locator}>{record.origin.record_id} · version {record.origin.record_version}</p><RecordFields record={record.record} /></details><div className={styles.actions}><button disabled={busy} onClick={() => void openProbata(record)}>Add legal response</button></div></article>)}
    </section> : tab === "gaps" ? <section aria-label="Gaps report">
      {gapsVisible.length === 0 ? <p className={styles.empty}>{report.length ? "No gaps match these filters." : "No evidence gaps are recorded. Add a claim to begin tracking its sources and follow-up."}</p> : gapsVisible.map((gap, index) => <article className={styles.gap} key={`${gap.claim_id}-${gap.gap_id ?? index}`}><div className={styles.itemHeader}><h2>{gap.claim_text}</h2><span className={styles.badge}>{statusLabel(gap.evidence_status)}</span></div><p>{gap.description}</p>{gap.followups.length > 0 ? <ul className={styles.gapReasons}>{gap.followups.map(followup => <li key={followup.followup_id}>{followupKinds[followup.kind]}: {followup.description} · {followup.status === "open" ? "Planned" : followup.status === "done" ? "Done" : "Cancelled"}</li>)}</ul> : <p className={styles.muted}>No follow-up planned.</p>}<button disabled={busy} onClick={() => void selectClaim(gap.claim_id)}>Open claim and plan follow-up</button></article>)}
    </section> : <div className={styles.body}>
      <aside className={styles.list} aria-label="Claim list">{visible.length === 0 ? <p className={styles.empty}>{claims.length ? "No claims match these filters." : "Add a statement, allegation, question or working theory. Evidence can be linked later."}</p> : visible.map(claim => <button key={claim.claim_id} className={styles.claimButton} aria-pressed={selected?.claim_id === claim.claim_id} onClick={() => void selectClaim(claim.claim_id)} disabled={busy}><strong>{claim.origin_record?.title || claim.text}</strong><small>{claim.origin ? `Probata ${claim.origin.kind} · ` : ""}{kinds[claim.kind]}{claim.claimant ? ` · ${claim.claimant}` : ""}</small><span className={`${styles.badge} ${claim.evidence_status === "conflicting_evidence" ? styles.warning : ""}`}>{statusLabel(claim.evidence_status)}</span></button>)}</aside>
      <section className={styles.panel} aria-label="Claim details">{!selected && !creating ? <div className={styles.empty}><h2>Build the case one claim at a time</h2><p>Choose a claim to review its response, evidence and next steps, or add the first claim.</p><button className={styles.primary} onClick={newClaim}>Add your first claim</button></div> : <>
        {selected?.origin && <section className={styles.item} aria-label="Linked Probata record"><div className={styles.itemHeader}><h2>{selected.origin_record?.title || `Probata ${selected.origin.kind}`}</h2><span className={`${styles.badge} ${selected.origin_state === "changed" ? styles.warning : ""}`}>{selected.origin_state === "changed" ? "Changed since linked" : selected.origin_state === "unavailable" ? "Source unavailable" : "Linked source"}</span></div><details className={styles.inspector}><summary>Current source record</summary><p className={styles.locator}>{selected.origin.record_id} · linked version {selected.origin.record_version}</p>{selected.origin_record && <><p className={styles.locator}>Current version {selected.origin_record.origin.record_version}</p><RecordFields record={selected.origin_record.payload} /></>}</details><button disabled={busy} onClick={() => void refreshOrigin()}>Refresh source</button></section>}
        <form className={styles.form} onSubmit={event => { event.preventDefault(); void saveClaim(); }}>
          <div className={styles.itemHeader}><h2>{creating ? "New claim" : "Claim and response"}</h2>{selected && <span className={styles.badge}>{statusLabel(selected.evidence_status)}</span>}</div>
          <div className={styles.fields}><label>Record type<select value={fields.kind} disabled={busy} onChange={event => setFields({ ...fields, kind: event.target.value as ClaimKind })}>{Object.entries(kinds).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label><label>Who made this statement?<input value={fields.claimant} maxLength={250} disabled={busy} onChange={event => setFields({ ...fields, claimant: event.target.value })} placeholder="Person or source" /></label></div>
          <label>Statement or allegation<textarea required rows={4} maxLength={20000} value={fields.text} disabled={busy} onChange={event => setFields({ ...fields, text: event.target.value })} /></label>
          <label>Your response<textarea rows={3} maxLength={20000} value={fields.response} disabled={busy} onChange={event => setFields({ ...fields, response: event.target.value })} /></label>
          <div className={styles.actions}><button className={styles.primary} type="submit" disabled={busy || !fields.text.trim() || conflict}>{busy ? "Saving…" : creating ? "Save claim" : "Save changes"}</button>{selected && <span className={styles.muted}>{fieldsDirty ? "Unsaved changes" : `Saved · version ${selected.revision}`}</span>}</div>
        </form>
        {selected && <>
          <section className={styles.section} aria-label="Linked evidence"><h2>Evidence and context</h2><p className={styles.muted}>The status reflects the links recorded for this claim.</p>
            {selected.links.filter(link => link.active).length === 0 && <p>No sources linked yet.</p>}
            {selected.links.filter(link => link.active).map(link => <article key={link.link_id} className={styles.item}><div className={styles.itemHeader}><strong>{relationships[link.relationship]}</strong><span className={styles.badge}>{link.validation_status === "accepted" ? "Accepted source" : link.validation_status === "context" ? "Context reference" : "Source unavailable"}</span></div><p className={styles.locator}>{link.source?.span_locator || link.context_reference}</p>{link.note && <p>{link.note}</p>}{link.source && <SourceInspector source={link.source} />}<div className={styles.actions}><button disabled={!canMutate} onClick={() => { if (window.confirm("Remove this link from the active claim? It will remain in the history.")) void mutate(`/links/${link.link_id}`, "PATCH", { active: false }, "Evidence link removed from the active claim."); }}>Remove link</button></div></article>)}
            {selected.links.some(link => !link.active) && <details className={styles.inspector}><summary>Removed links ({selected.links.filter(link => !link.active).length})</summary>{selected.links.filter(link => !link.active).map(link => <article className={styles.item} key={link.link_id}><strong>{relationships[link.relationship]}</strong><p className={styles.locator}>{link.source?.span_locator || link.context_reference}</p>{link.note && <p>{link.note}</p>}{link.source && <SourceInspector source={link.source} />}<button disabled={!canMutate} onClick={() => void mutate(`/links/${link.link_id}`, "PATCH", { active: true }, "Evidence link restored.")}>Restore link</button></article>)}</details>}
            <details className={styles.section} open><summary>Link a source or context reference</summary>
              {sourcesMessage && <p className={styles.muted}>{sourcesMessage}</p>}
              <button disabled={busy} onClick={() => { setSourceIndex(""); void refreshSources(); }}>Refresh evidence sources</button>
              <form className={styles.form} onSubmit={event => { event.preventDefault(); void mutate("/links", "POST", { relationship, source: pickedSource, context_reference: contextReference, note: linkNote }, "Source link saved.", () => { setSourceIndex(""); setContextReference(""); setLinkNote(""); }); }}>
                <div className={styles.fields}><label>Relationship to this claim<select value={relationship} disabled={!canMutate} onChange={event => { setRelationship(event.target.value as Relationship); setSourceIndex(""); setContextReference(""); }} >{Object.entries(relationships).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label><label>Accepted evidence source<select value={sourceIndex} disabled={!canMutate || sources.length === 0} onChange={event => setSourceIndex(event.target.value)}><option value="">Choose an exact passage or item</option>{sources.map((source, index) => <option key={`${source.package_id}-${source.item_id}-${source.assertion_id}`} value={index}>{source.span_locator} · {source.item_id}</option>)}</select></label></div>
                {pickedSource && <SourceInspector source={pickedSource} />}
                {relationship === "context" && !pickedSource && <label>Context location<input value={contextReference} disabled={!canMutate} maxLength={2000} onChange={event => setContextReference(event.target.value)} placeholder="Document, URL, message or note location" /></label>}
                <label>What does this source show?<textarea rows={2} value={linkNote} maxLength={5000} disabled={!canMutate} onChange={event => setLinkNote(event.target.value)} /></label>
                {fieldsDirty && <p className={styles.muted}>Save your claim changes before adding evidence or next steps.</p>}
                <button type="submit" disabled={!canMutate || (!pickedSource && !(relationship === "context" && contextReference.trim()))}>Save source link</button>
              </form>
            </details>
          </section>
          <section className={styles.section} aria-label="Evidence gaps"><h2>What is missing?</h2>{selected.gaps.length === 0 && <p className={styles.muted}>Record the specific proof or answer you still need.</p>}{selected.gaps.map(gap => <article className={styles.item} key={gap.gap_id}><div className={styles.itemHeader}><strong>{gap.description}</strong><span className={styles.badge}>{gap.status === "open" ? "Open" : "Resolved"}</span></div><button disabled={!canMutate} onClick={() => void mutate(`/gaps/${gap.gap_id}`, "PATCH", { status: gap.status === "open" ? "resolved" : "open" }, gap.status === "open" ? "Gap marked resolved." : "Gap reopened.")}>{gap.status === "open" ? "Mark resolved" : "Reopen gap"}</button></article>)}
            <form className={styles.form} onSubmit={event => { event.preventDefault(); void mutate("/gaps", "POST", { description: gapText }, "Evidence gap saved.", () => setGapText("")); }}><label>Missing evidence or unanswered question<textarea rows={2} value={gapText} maxLength={5000} disabled={!canMutate} onChange={event => setGapText(event.target.value)} /></label><button disabled={!canMutate || !gapText.trim()} type="submit">Add gap</button></form>
          </section>
          <section className={styles.section} aria-label="Follow-up plans"><h2>Follow-up</h2>{selected.followups.length === 0 && <p className={styles.muted}>Record what to find next. Send an evidence investigation to Probata when it is ready.</p>}{selected.followups.map(followup => <article className={styles.item} key={followup.followup_id}>
            <div className={styles.itemHeader}><strong>{followupKinds[followup.kind]}</strong><span className={styles.badge}>{followup.status === "open" ? "Planned" : followup.status === "done" ? "Done" : "Cancelled"}</span></div><p>{followup.description}</p>
            {followup.investigation && <div aria-label="Investigation request status">
              <span className={styles.badge}>{followup.investigation.remote_status ? investigationLabels[followup.investigation.remote_status] : "Ready to retry"}</span>
              {followup.investigation.remote_status === "received" && <p className={styles.muted}>Waiting for execution.</p>}
              {followup.investigation.last_error && <p className={styles.muted}>{followup.investigation.last_error}</p>}
              <details className={styles.inspector}><summary>Request details</summary><dl>
                <dt>Request</dt><dd>{followup.investigation.request_id || "Prepared; awaiting acknowledgement"}</dd>
                <dt>Question sent</dt><dd>{followup.investigation.request.question}</dd>
                <dt>Case mode</dt><dd>{followup.investigation.request.mode === "REAL" ? "Current case" : "Test case"}</dd>
                <dt>Probata matter</dt><dd>{followup.investigation.request.matter_id}</dd>
                <dt>Court case</dt><dd>{followup.investigation.request.court_case_id}</dd>
                {followup.investigation.remote_updated_at && <><dt>Status updated</dt><dd>{new Date(followup.investigation.remote_updated_at).toLocaleString()}</dd></>}
              </dl></details>
              {followup.investigation.results?.map((result, index) => <div className={styles.item} key={index}><h3>Finding {index + 1}</h3><p>{result.summary}</p><details className={styles.inspector}><summary>Linked sources and method</summary><RecordFields record={{ sources: result.sources, tool: result.tool, run: result.run_id }} /></details></div>)}
            </div>}
            <div className={styles.actions}>
              {followup.kind === "investigate" && followup.status === "open" && !followup.investigation?.request_id && <button className={styles.primary} disabled={!canMutate} onClick={() => void investigationAction(followup.followup_id)}>{followup.investigation ? "Retry send" : "Send to Probata"}</button>}
              {followup.investigation?.request_id && <button disabled={!canMutate} onClick={() => void investigationAction(followup.followup_id, true)}>Refresh request status</button>}
              {followup.status === "open" ? <><button disabled={!canMutate || followup.investigation?.state === "prepared"} onClick={() => void mutate(`/followups/${followup.followup_id}`, "PATCH", { status: "done" }, "Follow-up marked done.")}>Mark done</button><button disabled={!canMutate || followup.investigation?.state === "prepared"} onClick={() => void mutate(`/followups/${followup.followup_id}`, "PATCH", { status: "cancelled" }, "Follow-up cancelled.")}>Cancel plan</button></> : <button disabled={!canMutate} onClick={() => void mutate(`/followups/${followup.followup_id}`, "PATCH", { status: "open" }, "Follow-up reopened.")}>Reopen plan</button>}
            </div>
          </article>)}
            <form className={styles.form} onSubmit={event => { event.preventDefault(); void mutate("/followups", "POST", { kind: followupKind, description: followupText, gap_id: followupGap || null }, "Follow-up plan saved.", () => { setFollowupText(""); setFollowupGap(""); }); }}><div className={styles.fields}><label>Next action<select disabled={!canMutate} value={followupKind} onChange={event => setFollowupKind(event.target.value as FollowupKind)}>{Object.entries(followupKinds).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label><label>Related gap<select disabled={!canMutate} value={followupGap} onChange={event => setFollowupGap(event.target.value)}><option value="">General follow-up for this claim</option>{selected.gaps.filter(gap => gap.status === "open").map(gap => <option key={gap.gap_id} value={gap.gap_id}>{gap.description}</option>)}</select></label></div><label>Follow-up details<textarea rows={2} disabled={!canMutate} value={followupText} maxLength={5000} onChange={event => setFollowupText(event.target.value)} /></label><button disabled={!canMutate || !followupText.trim()} type="submit">Save follow-up plan</button></form>
          </section>
          <section className={styles.section}><button disabled={busy} onClick={() => void loadHistory()}>View saved history</button>{history && <div>{history.map(version => <details className={styles.item} key={version.revision}><summary>Version {version.revision} · {new Date(version.occurred_at).toLocaleString()}</summary><p className={styles.quote}>{version.record.text}</p>{version.record.response && <p>Response: {version.record.response}</p>}<p className={styles.muted}>{version.record.links.filter(link => link.active).length} active links · {version.record.gaps.length} gaps · {version.record.followups.length} follow-up plans</p></details>)}</div>}</section>
        </>}
      </>}</section>
    </div>}
  </div>;
}
