// Byline: Claude Code · Opus 5.5 · 2026-09-28
//
// Family Law Toolkit host page (bundled by build.mjs to dist/web/host.js). Every read and
// write calls the console's own MCP tools through /api/tools/:name (web.ts), so this page has
// no data logic of its own. Tools that declare a ui:// widget render in a sandboxed iframe
// driven by the MCP Apps AppBridge, exactly as an MCP App host would.

import { AppBridge, PostMessageTransport } from "@modelcontextprotocol/ext-apps/app-bridge";

type Json = Record<string, unknown>;
interface Tool { name: string; title?: string; description?: string; inputSchema?: Json; annotations?: Json; _meta?: { ui?: { resourceUri?: string } } }
interface ToolResult { structuredContent?: Json; content?: Array<{ type: string; text?: string }>; isError?: boolean }

const $ = <T extends HTMLElement>(sel: string) => document.querySelector(sel) as T;
const main = $("#main");
let tools: Tool[] = [];

// ---------------------------------------------------------------------------
// HTTP + tools
// ---------------------------------------------------------------------------

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, { ...init, headers: { "content-type": "application/json", ...(init?.headers ?? {}) } });
  const text = await res.text();
  let body: unknown = null;
  try { body = text ? JSON.parse(text) : null; } catch { body = text; }
  if (!res.ok) throw new Error((body as { error?: string })?.error ?? `HTTP ${res.status}`);
  return body as T;
}

async function call(name: string, args: Json = {}): Promise<ToolResult> {
  return api<ToolResult>(`/api/tools/${name}`, { method: "POST", body: JSON.stringify({ arguments: args }) });
}

/** A tool's structured result, or an Error carrying the tool's own message. */
async function data(name: string, args: Json = {}): Promise<Json> {
  const r = await call(name, args);
  if (r.isError) throw new Error(r.content?.find((c) => c.type === "text")?.text ?? `${name} failed`);
  const sc = r.structuredContent ?? JSON.parse(r.content?.find((c) => c.type === "text")?.text ?? "{}");
  if ((sc as Json).available === false) throw new Error(String((sc as Json).reason ?? "case store unavailable"));
  return sc as Json;
}

// ---------------------------------------------------------------------------
// DOM helpers
// ---------------------------------------------------------------------------

function el<K extends keyof HTMLElementTagNameMap>(tag: K, attrs: Json = {}, ...kids: Array<Node | string | null | undefined>): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (v === undefined || v === null || v === false) continue;
    if (k.startsWith("on") && typeof v === "function") node.addEventListener(k.slice(2), v as EventListener);
    else if (k === "text") node.textContent = String(v);
    else node.setAttribute(k, v === true ? "" : String(v));
  }
  for (const kid of kids) if (kid !== null && kid !== undefined) node.append(kid);
  return node;
}

function errBox(err: unknown): HTMLElement {
  return el("div", { class: "err", role: "alert", text: err instanceof Error ? err.message : String(err) });
}

function okBox(text: string): HTMLElement {
  return el("div", { class: "okmsg", role: "status", text });
}

function page(title: string, lede?: string): HTMLElement {
  main.replaceChildren(el("h1", { text: title }), ...(lede ? [el("p", { class: "lede", text: lede })] : []));
  main.focus({ preventScroll: true });
  return main;
}

function kv(record: Json): HTMLElement {
  const dl = el("dl", { class: "kv" });
  for (const [k, v] of Object.entries(record)) {
    if (v === null || v === undefined || v === "" || k === "body" || k === "file_b64") continue;
    dl.append(el("dt", { text: k }), el("dd", { text: typeof v === "object" ? JSON.stringify(v, null, 1) : String(v) }));
  }
  return dl;
}

function titleOf(row: Json): string {
  for (const key of ["title", "citation", "label", "key", "description", "text"]) {
    const v = row[key];
    if (typeof v === "string" && v.trim()) return v.trim().slice(0, 160);
  }
  return String(row.id ?? "record");
}

function recordRow(row: Json, meta: string, onOpen: () => void): HTMLElement {
  return el("button", { class: "row", type: "button", onclick: onOpen },
    el("span", {}, el("span", { class: "t", text: titleOf(row) }), el("br"), el("span", { class: "m", text: meta })),
    el("span", { class: "tag", text: String(row.kind ?? row.status ?? "") }));
}

function filterInput(placeholder: string, onInput: (q: string) => void): HTMLElement {
  const input = el("input", { type: "search", placeholder, "aria-label": placeholder, class: "filter" }) as HTMLInputElement;
  input.addEventListener("input", () => onInput(input.value.trim().toLowerCase()));
  return input;
}

// Small markdown renderer for the toolkit's own content (headings, lists, tables, code, quotes,
// emphasis, links). Input is escaped first, so content can never inject markup.
function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}
function inline(s: string): string {
  return esc(s)
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/(^|[^*])\*([^*]+)\*/g, "$1<em>$2</em>")
    .replace(/\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>');
}
function markdown(src: string): HTMLElement {
  const out: string[] = [];
  const lines = src.replace(/\r\n/g, "\n").split("\n");
  let i = 0;
  while (i < lines.length) {
    const line = lines[i];
    if (/^```/.test(line)) {
      const buf: string[] = [];
      for (i++; i < lines.length && !/^```/.test(lines[i]); i++) buf.push(lines[i]);
      out.push(`<pre><code>${esc(buf.join("\n"))}</code></pre>`); i++; continue;
    }
    const h = /^(#{1,6})\s+(.*)$/.exec(line);
    if (h) { const n = Math.min(h[1].length + 1, 6); out.push(`<h${n}>${inline(h[2])}</h${n}>`); i++; continue; }
    if (/^\s*\|.*\|\s*$/.test(line)) {
      const rows: string[][] = [];
      for (; i < lines.length && /^\s*\|.*\|\s*$/.test(lines[i]); i++) {
        if (/^\s*\|[\s:|-]+\|\s*$/.test(lines[i])) continue;
        rows.push(lines[i].trim().slice(1, -1).split("|").map((c) => c.trim()));
      }
      out.push(`<table>${rows.map((r, ri) => `<tr>${r.map((c) => ri === 0 ? `<th>${inline(c)}</th>` : `<td>${inline(c)}</td>`).join("")}</tr>`).join("")}</table>`);
      continue;
    }
    if (/^\s*([-*+]|\d+\.)\s+/.test(line)) {
      const ordered = /^\s*\d+\./.test(line);
      const items: string[] = [];
      for (; i < lines.length && /^\s*([-*+]|\d+\.)\s+/.test(lines[i]); i++) items.push(`<li>${inline(lines[i].replace(/^\s*([-*+]|\d+\.)\s+/, ""))}</li>`);
      out.push(ordered ? `<ol>${items.join("")}</ol>` : `<ul>${items.join("")}</ul>`);
      continue;
    }
    if (/^>\s?/.test(line)) {
      const buf: string[] = [];
      for (; i < lines.length && /^>\s?/.test(lines[i]); i++) buf.push(lines[i].replace(/^>\s?/, ""));
      out.push(`<blockquote>${inline(buf.join(" "))}</blockquote>`); continue;
    }
    if (!line.trim()) { i++; continue; }
    const buf: string[] = [];
    for (; i < lines.length && lines[i].trim() && !/^(#{1,6}\s|```|\s*([-*+]|\d+\.)\s|>|\s*\|)/.test(lines[i]); i++) buf.push(lines[i]);
    if (!buf.length) { buf.push(line); i++; }
    out.push(`<p>${inline(buf.join(" "))}</p>`);
  }
  const div = el("div", { class: "md" });
  div.innerHTML = out.join("\n");
  return div;
}

// ---------------------------------------------------------------------------
// Sheet (record detail, file reader, forms)
// ---------------------------------------------------------------------------

const sheet = $("#sheet") as HTMLDialogElement;
const sheetBody = $("#sheetBody");

function openSheet(title: string, ...kids: Array<Node | null>): void {
  sheetBody.replaceChildren(
    el("div", { class: "sheethead" }, el("h2", { text: title }), el("button", { type: "button", class: "secondary", onclick: () => sheet.close() }, "Close")),
    ...kids.filter((k): k is Node => k !== null),
  );
  if (!sheet.open) sheet.showModal();
}

async function openRecord(ref: string, onChanged?: () => void): Promise<void> {
  openSheet("Loading…");
  try {
    const rec = await api<Json>(`/api/records/${encodeURIComponent(ref)}`);
    const record = (rec.record ?? {}) as Json;
    const body = typeof record.body === "string" ? record.body : typeof record.text === "string" ? record.text : null;
    const fileLink = typeof record.file_b64 === "string" && record.file_b64
      ? el("a", { class: "btn secondary", download: String(record.file_name ?? "document"), href: `data:${String(record.file_mime ?? "application/octet-stream")};base64,${record.file_b64}` }, `Download ${String(record.file_name ?? "file")}`)
      : null;
    const url = typeof record.url === "string" && /^https?:\/\//.test(record.url) ? el("a", { class: "btn secondary", href: record.url, target: "_blank", rel: "noopener noreferrer" }, "Open official source") : null;
    openSheet(titleOf(record),
      el("p", { class: "idline", text: `id ${String(rec.id)}` }),
      el("p", { class: "idline", text: `version ${String(rec.version)}` }),
      el("div", { class: "actions" }, url, fileLink, el("button", { type: "button", onclick: () => correctForm(String(rec.id), String(rec.table), record, onChanged) }, "Correct this record")),
      body ? markdown(body) : null,
      el("h3", { text: "Fields" }),
      kv(record));
  } catch (err) {
    openSheet("Record", errBox(err));
  }
}

function correctForm(ref: string, table: string, record: Json, onChanged?: () => void): void {
  const editable = Object.entries(record).filter(([k, v]) => (typeof v === "string" || typeof v === "number" || typeof v === "boolean") && !["file_b64", "sha256", "loaded_at", "corrected_at"].includes(k));
  const form = el("form", {});
  const inputs = new Map<string, HTMLInputElement | HTMLTextAreaElement>();
  for (const [k, v] of editable) {
    const long = typeof v === "string" && (v.length > 120 || v.includes("\n"));
    const input = long ? el("textarea", { name: k }) : el("input", { name: k, value: String(v) });
    if (long) (input as HTMLTextAreaElement).value = String(v);
    inputs.set(k, input);
    form.append(el("label", {}, k, input));
  }
  const why = el("textarea", { name: "why", placeholder: "What was wrong and where the correct value comes from" }) as HTMLTextAreaElement;
  form.append(el("label", {}, "Reason for the correction", why), el("button", { type: "submit" }, "Save correction"));
  const status = el("div");
  form.addEventListener("submit", async (ev) => {
    ev.preventDefault();
    const changed: Json = {};
    const previous: Json = {};
    for (const [k, v] of editable) {
      const now = inputs.get(k)!.value;
      const next = typeof v === "number" ? Number(now) : typeof v === "boolean" ? now === "true" : now;
      if (next !== v) { changed[k] = next; previous[k] = v; }
    }
    if (!Object.keys(changed).length) { status.replaceChildren(errBox("Nothing changed.")); return; }
    try {
      const at = new Date().toISOString();
      await data("case_put", { table, id: ref.slice(ref.indexOf(":") + 1), data: { ...changed, corrected_at: at } });
      const noteId = `correction-${at.replace(/[^0-9]/g, "")}`;
      await data("case_put", {
        table: "note", id: noteId,
        data: { kind: "correction", title: `Correction to ${ref}`, text: why.value || `Corrected ${Object.keys(changed).join(", ")}`, target: ref, previous, corrected: changed, created_at: at, added_via: "toolkit-web" },
        relations: [{ edge: "about", from: `note:${noteId}`, to: ref }],
      });
      onChanged?.();
      await openRecord(ref, onChanged);
      sheetBody.prepend(okBox("Correction saved. The record has a new version; the correction note links to it."));
    } catch (err) {
      status.replaceChildren(errBox(err));
    }
  });
  openSheet(`Correct ${ref}`, form, status);
}

// ---------------------------------------------------------------------------
// Widgets (MCP Apps AppBridge)
// ---------------------------------------------------------------------------

async function renderWidget(host: HTMLElement, tool: Tool, args: Json): Promise<void> {
  const name = /ui:\/\/family-court\/([a-z]+)\.html/.exec(tool._meta?.ui?.resourceUri ?? "")?.[1];
  const result = await call(tool.name, args);
  if (!name) { host.replaceChildren(resultView(result)); return; }
  const frame = el("iframe", { class: "widget", title: tool.title ?? tool.name, sandbox: "allow-scripts allow-forms allow-popups allow-popups-to-escape-sandbox", src: `/api/widgets/${name}` }) as HTMLIFrameElement;
  const note = el("div");
  host.replaceChildren(frame, note);
  await new Promise<void>((resolve) => frame.addEventListener("load", () => resolve(), { once: true }));
  const bridge = new AppBridge(null, { name: "Family Law Toolkit", version: "1.0.0" }, { openLinks: {}, message: { text: {} } }, {
    hostContext: { theme: "dark", displayMode: "inline", platform: "web" },
  });
  bridge.onsizechange = ({ height }) => { if (typeof height === "number" && height > 0) frame.style.height = `${Math.ceil(height) + 4}px`; };
  bridge.onopenlink = async ({ url }) => { window.open(url, "_blank", "noopener,noreferrer"); return {}; };
  bridge.onmessage = async ({ content }) => {
    const text = (content as Array<{ type: string; text?: string }>).filter((c) => c.type === "text").map((c) => c.text).join("\n");
    note.replaceChildren(el("div", { class: "card" }, el("p", { text: text }), el("button", { type: "button", onclick: () => noteForm(text) }, "Save as a note")));
    return {};
  };
  bridge.oninitialized = () => {
    void bridge.sendToolInput({ arguments: args });
    void bridge.sendToolResult(result as never);
  };
  await bridge.connect(new PostMessageTransport(frame.contentWindow!, frame.contentWindow!));
}

function resultView(result: ToolResult): HTMLElement {
  if (result.isError) return errBox(result.content?.find((c) => c.type === "text")?.text ?? "Tool error");
  const text = result.content?.find((c) => c.type === "text")?.text ?? "";
  if (result.structuredContent) return el("pre", { class: "json", text: JSON.stringify(result.structuredContent, null, 2) });
  return markdown(text);
}

// Generic form from a tool's JSON Schema: strings, numbers, booleans and enums get their own
// controls; arrays of strings take one value per line; anything else is edited as JSON.
function toolForm(tool: Tool, onRun: (args: Json) => Promise<void>, preset: Json = {}): HTMLElement {
  const schema = (tool.inputSchema ?? {}) as { properties?: Record<string, Json>; required?: string[] };
  const props = schema.properties ?? {};
  const required = new Set(schema.required ?? []);
  const form = el("form", {});
  const readers: Array<() => [string, unknown] | null> = [];
  for (const [key, prop] of Object.entries(props)) {
    const label = `${key}${required.has(key) ? "" : " (optional)"}`;
    const type = prop.type as string | undefined;
    const enumValues = prop.enum as string[] | undefined;
    const initial = preset[key];
    if (enumValues) {
      const select = el("select", { name: key }, required.has(key) ? null : el("option", { value: "" }, "—"), ...enumValues.map((v) => el("option", { value: v, selected: initial === v }, v))) as HTMLSelectElement;
      form.append(el("label", {}, label, select));
      readers.push(() => (select.value ? [key, select.value] : null));
    } else if (type === "boolean") {
      const select = el("select", { name: key }, el("option", { value: "" }, "—"), el("option", { value: "true" }, "true"), el("option", { value: "false" }, "false")) as HTMLSelectElement;
      form.append(el("label", {}, label, select));
      readers.push(() => (select.value ? [key, select.value === "true"] : null));
    } else if (type === "number" || type === "integer") {
      const input = el("input", { name: key, type: "number", value: initial as string }) as HTMLInputElement;
      form.append(el("label", {}, label, input));
      readers.push(() => (input.value !== "" ? [key, Number(input.value)] : null));
    } else if (type === "string") {
      const long = /text|description|body|goal|surql|query/.test(key);
      const input = long ? el("textarea", { name: key }) : el("input", { name: key, value: initial as string });
      if (long && initial) (input as HTMLTextAreaElement).value = String(initial);
      form.append(el("label", {}, label, input));
      readers.push(() => ((input as HTMLInputElement).value !== "" ? [key, (input as HTMLInputElement).value] : null));
    } else if (type === "array" && (prop.items as Json | undefined)?.type === "string") {
      const input = el("textarea", { name: key, placeholder: "one per line" }) as HTMLTextAreaElement;
      form.append(el("label", {}, label, input));
      readers.push(() => { const v = input.value.split("\n").map((s) => s.trim()).filter(Boolean); return v.length ? [key, v] : null; });
    } else {
      const input = el("textarea", { name: key, placeholder: "JSON" }) as HTMLTextAreaElement;
      if (initial !== undefined) input.value = JSON.stringify(initial, null, 2);
      form.append(el("label", {}, `${label} — JSON`, input));
      readers.push(() => (input.value.trim() ? [key, JSON.parse(input.value)] : null));
    }
  }
  const status = el("div");
  form.append(el("button", { type: "submit" }, "Run"), status);
  form.addEventListener("submit", async (ev) => {
    ev.preventDefault();
    try {
      const args: Json = {};
      for (const read of readers) { const pair = read(); if (pair) args[pair[0]] = pair[1]; }
      status.replaceChildren(el("p", { class: "lede", text: "Running…" }));
      await onRun(args);
      status.replaceChildren();
    } catch (err) {
      status.replaceChildren(errBox(err));
    }
  });
  return form;
}

// ---------------------------------------------------------------------------
// Views
// ---------------------------------------------------------------------------

const RECORD_PAGE_SIZE = 100;

interface RecordPage {
  records: Array<{ id: string; kind: string; record: Json }>;
  total: number;
  nextCursor: string | null;
}

interface RecordTableState {
  table: string;
  rows: Json[];
  total: number;
  cursor: string | null;
  seenCursors: Set<string>;
  seenIds: Set<string>;
}

// Byline: Codex · GPT-6 · 2026-10-04
/** Validate one response page and identify repeated, empty, or nonadvancing continuations.
 * Inputs: one API page, its table's loaded state, and the cursor used for this request.
 * Outputs: normalized record rows, or a descriptive error for malformed pages, duplicate IDs, or omitted remaining rows.
 * Side effects: none; callers decide whether to show the error while retaining the current page.
 * Use this before appending table rows; library pages use `libraryPage` because they have no remote cursor.
 */
function validateRecordPage(page: RecordPage, state: RecordTableState, requestedCursor: string | null): Json[] {
  if (!Number.isSafeInteger(page.total) || page.total < 0 || !Array.isArray(page.records) || !(page.nextCursor === null || typeof page.nextCursor === "string")) {
    throw new Error(`Records paging for ${state.table} returned an invalid page.`);
  }
  if (page.records.length === 0 && page.total > state.rows.length) {
    throw new Error(`Records paging for ${state.table} returned an empty page while ${page.total - state.rows.length} records remain.`);
  }
  if (page.nextCursor !== null) {
    if (!/^(0|[1-9]\d*)$/.test(page.nextCursor)) throw new Error(`Records paging for ${state.table} returned an invalid cursor.`);
    const currentOffset = requestedCursor === null ? 0 : Number(requestedCursor);
    const nextOffset = Number(page.nextCursor);
    if (nextOffset <= currentOffset || nextOffset !== currentOffset + page.records.length || state.seenCursors.has(page.nextCursor)) {
      throw new Error(`Records paging for ${state.table} did not advance past cursor ${requestedCursor ?? "the first page"}.`);
    }
    if (state.rows.length + page.records.length >= page.total) {
      throw new Error(`Records paging for ${state.table} returned a cursor after all ${page.total} records.`);
    }
  } else if (state.rows.length + page.records.length < page.total) {
    throw new Error(`Records paging for ${state.table} stopped before all ${page.total} records were available.`);
  }
  const rows = page.records.map((record) => ({ ...(record.record as Json), id: record.id, kind: record.kind }));
  const pageIds = new Set<string>();
  for (const row of rows) {
    const id = String(row.id);
    if (state.seenIds.has(id) || pageIds.has(id)) throw new Error(`Records paging for ${state.table} repeated record ${id}.`);
    pageIds.add(id);
  }
  return rows;
}

// Byline: Codex · GPT-6 · 2026-10-04
/** Fetch and append exactly one bounded page for a table.
 * Inputs: mutable per-table paging state containing the next offset cursor and seen row/cursor sets.
 * Outputs: resolves after rows, total, and next cursor are updated; rejects with a visible error on paging inconsistency.
 * Side effects: issues one authenticated `/api/records` request and mutates only the supplied view state.
 * Use for explicit “Show more” actions; do not loop over the cursor to fetch a whole table automatically.
 */
async function loadRecordPage(state: RecordTableState): Promise<void> {
  const requestedCursor = state.cursor;
  if (requestedCursor !== null && state.seenCursors.has(requestedCursor)) {
    throw new Error(`Records paging for ${state.table} repeated cursor ${requestedCursor}.`);
  }
  const params = new URLSearchParams({ table: state.table, limit: String(RECORD_PAGE_SIZE) });
  if (requestedCursor !== null) params.set("cursor", requestedCursor);
  const page = await api<RecordPage>(`/api/records?${params.toString()}`);
  const rows = validateRecordPage(page, state, requestedCursor);
  if (requestedCursor !== null) state.seenCursors.add(requestedCursor);
  state.rows.push(...rows);
  state.total = page.total;
  state.cursor = page.nextCursor;
}

// Byline: Codex · GPT-6 · 2026-10-04
/** Render table rows with explicit bounded requests, local filtering, and visible page errors.
 * Inputs: host element, one or more table names, row metadata, refresh/empty behavior, and optional loaded-row filters.
 * Outputs: appends a searchable list, loaded/total status, and an explicit continuation button.
 * Side effects: loads one page per table initially and one additional page per table only after a click.
 * Use for case-store table browsing; `recordList` remains for tool-produced bounded collections such as the docket.
 */
async function pagedRecordList(
  host: HTMLElement,
  tables: string[],
  meta: (r: Json) => string,
  refresh: () => void,
  emptyText: string,
  filterRows: (rows: Json[]) => Json[] = (rows) => rows,
  onRowsChanged?: (rows: Json[]) => void,
): Promise<void> {
  const states = tables.map((table): RecordTableState => ({ table, rows: [], total: 0, cursor: null, seenCursors: new Set(), seenIds: new Set() }));
  await Promise.all(states.map((state) => loadRecordPage(state)));
  const list = el("div", { class: "list" });
  const count = el("p", { class: "m", role: "status" });
  const error = el("div");
  const more = el("button", { class: "btn secondary", type: "button", text: "Show more" }) as HTMLButtonElement;
  let query = "";
  const draw = () => {
    const loaded = states.flatMap((state) => state.rows);
    onRowsChanged?.(loaded);
    const rows = filterRows(loaded);
    const shown = rows.filter((r) => !query || JSON.stringify(r).toLowerCase().includes(query));
    const total = states.reduce((sum, state) => sum + state.total, 0);
    count.textContent = `Showing ${shown.length} matching records from ${rows.length} loaded; ${total} total.`;
    list.replaceChildren(...(shown.length ? shown.map((r) => recordRow(r, meta(r), () => void openRecord(String(r.id), refresh))) : [el("p", { class: "empty", text: rows.length ? "No matches in the loaded records." : emptyText })]));
    more.hidden = !states.some((state) => state.cursor !== null);
  };
  more.addEventListener("click", async () => {
    more.disabled = true;
    error.replaceChildren();
    try {
      await Promise.all(states.filter((state) => state.cursor !== null).map((state) => loadRecordPage(state)));
      draw();
    } catch (err) {
      error.replaceChildren(errBox(err));
    } finally {
      more.disabled = false;
    }
  });
  host.append(filterInput(`Filter ${states.reduce((sum, state) => sum + state.total, 0)} records`, (q) => { query = q; draw(); }), count, list, more, error);
  draw();
}

// Byline: Codex · GPT-6 · 2026-10-04
/** Return one bounded filtered library page and reset its position when its query changes.
 * Inputs: the listed file metadata, normalized query, requested page, previous query, and optional page size.
 * Outputs: matching files, the current visible slice, its normalized page number, and the matching total.
 * Side effects: none.
 * Use for the local toolkit file library; case-store rows require the server's records endpoint instead.
 */
export function libraryPage<T extends { path: string }>(files: T[], query: string, page: number, previousQuery = query, pageSize = 100): { matches: T[]; visible: T[]; page: number; total: number } {
  const safeSize = Number.isSafeInteger(pageSize) && pageSize > 0 ? pageSize : 100;
  const matches = files.filter((file) => !query || file.path.toLowerCase().includes(query));
  const lastPage = Math.max(1, Math.ceil(matches.length / safeSize));
  const requestedPage = query === previousQuery ? page : 1;
  const safePage = Number.isSafeInteger(requestedPage) ? Math.min(Math.max(requestedPage, 1), lastPage) : 1;
  return { matches, visible: matches.slice((safePage - 1) * safeSize, safePage * safeSize), page: safePage, total: matches.length };
}

function recordList(host: HTMLElement, rows: Json[], meta: (r: Json) => string, refresh: () => void, emptyText: string): void {
  const list = el("div", { class: "list" });
  const draw = (q: string) => {
    const shown = rows.filter((r) => !q || JSON.stringify(r).toLowerCase().includes(q));
    list.replaceChildren(...(shown.length ? shown.map((r) => recordRow(r, meta(r), () => void openRecord(String(r.id), refresh))) : [el("p", { class: "empty", text: rows.length ? "No matches." : emptyText })]));
  };
  host.append(filterInput(`Filter ${rows.length} records`, draw), list);
  draw("");
}

async function viewCase(): Promise<void> {
  const host = page("Case", "The case store the toolkit and the workdesk share.");
  try {
    const [summary, status] = await Promise.all([data("case_summary"), data("case_status", { action: "get" })]);
    const s = (summary.summary ?? summary) as Json;
    const counts = (s.counts ?? {}) as Record<string, number>;
    host.append(el("h2", { text: "Records" }), el("div", { class: "grid", "data-testid": "counts" },
      ...Object.entries(counts).map(([k, n]) => el("div", { class: "stat" }, el("b", { text: String(n) }), el("span", { text: k })))));
    host.append(el("h2", { text: "Court and status" }), el("div", { class: "card" }, kv({
      county: s.county, court: s.court, judge: s.judge, referee: s.referee, next_hearing: s.next_hearing,
      parties: Array.isArray(s.parties) ? (s.parties as string[]).join(", ") : undefined,
      ...(((status.status ?? status) as Json) ?? {}),
    })));
    const docket = await data("case_docket");
    const entries = (docket.entries ?? docket.docket ?? []) as Json[];
    host.append(el("h2", { text: `Docket (${entries.length})` }));
    recordList(host, entries.map((e) => ({ ...(e.record as Json), id: e.id, kind: e.table, title: e.title })), (r) => String(r.date ?? r.id), () => void viewCase(), "Nothing on the docket yet. Add documents under Documents.");
  } catch (err) {
    host.append(errBox(err));
  }
}

const DOC_TABLES = ["filing", "draft", "exhibit", "order"];

async function viewDocuments(): Promise<void> {
  const host = page("Documents", "Filed papers, drafts, exhibits and orders in the case store.");
  host.append(el("div", { class: "actions" }, el("button", { type: "button", onclick: () => documentForm() }, "Add a document")));
  try {
    await pagedRecordList(host, DOC_TABLES, (r) => `${String(r.id)} · ${String(r.date ?? r.entered ?? r.occurred_at ?? "")}`, () => void viewDocuments(), "No case documents yet. Use Add a document.");
  } catch (err) {
    host.append(errBox(err));
  }
}

function documentForm(): void {
  const form = el("form", {});
  const table = el("select", { name: "table" }, ...DOC_TABLES.map((t) => el("option", { value: t }, t))) as HTMLSelectElement;
  const title = el("input", { name: "title", required: true }) as HTMLInputElement;
  const date = el("input", { name: "date", type: "date" }) as HTMLInputElement;
  const status = el("input", { name: "status", placeholder: "e.g. filed, served, draft" }) as HTMLInputElement;
  const text = el("textarea", { name: "text", placeholder: "Text of the document, or notes about it" }) as HTMLTextAreaElement;
  const file = el("input", { name: "file", type: "file" }) as HTMLInputElement;
  form.append(el("label", {}, "Kind", table), el("label", {}, "Title", title), el("label", {}, "Date", date), el("label", {}, "Status", status), el("label", {}, "Text", text), el("label", {}, "File (up to 8 MB)", file), el("button", { type: "submit" }, "Save document"));
  const out = el("div");
  form.addEventListener("submit", async (ev) => {
    ev.preventDefault();
    try {
      const at = new Date().toISOString();
      const record: Json = { title: title.value, status: status.value || undefined, text: text.value || undefined, added_via: "toolkit-web", added_at: at };
      if (date.value) {
        const iso = new Date(`${date.value}T12:00:00Z`).toISOString();
        if (table.value === "exhibit") record.occurred_at = iso;
        else if (table.value === "order") record.entered = date.value;
        else record.date = iso;
      }
      const f = file.files?.[0];
      if (f) {
        if (f.size > 8 * 1024 * 1024) throw new Error("File is larger than 8 MB.");
        const bytes = new Uint8Array(await f.arrayBuffer());
        const digest = Array.from(new Uint8Array(await crypto.subtle.digest("SHA-256", bytes)), (b) => b.toString(16).padStart(2, "0")).join("");
        let bin = "";
        for (let i = 0; i < bytes.length; i += 0x8000) bin += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
        Object.assign(record, { file_name: f.name, file_mime: f.type || "application/octet-stream", file_size: f.size, file_sha256: digest, file_b64: btoa(bin) });
      }
      for (const k of Object.keys(record)) if (record[k] === undefined) delete record[k];
      const put = await data("case_put", { table: table.value, data: record });
      sheet.close();
      await viewDocuments();
      main.prepend(okBox(`Saved ${String(put.table)}:${String(put.id)}.`));
    } catch (err) {
      out.replaceChildren(errBox(err));
    }
  });
  openSheet("Add a document", form, out);
}

async function viewNotes(): Promise<void> {
  const host = page("Notes", "Your notes, memos and corrections.");
  host.append(el("div", { class: "actions" }, el("button", { type: "button", onclick: () => noteForm("") }, "Add a note")));
  try {
    await pagedRecordList(host, ["note", "memo"], (r) => `${String(r.id)} · ${String(r.created_at ?? "")}`, () => void viewNotes(), "No notes yet. Use Add a note.");
  } catch (err) {
    host.append(errBox(err));
  }
}

function noteForm(prefill: string): void {
  const form = el("form", {});
  const title = el("input", { name: "title" }) as HTMLInputElement;
  const kind = el("select", { name: "kind" }, ...["note", "task", "question", "observation"].map((k) => el("option", { value: k }, k))) as HTMLSelectElement;
  const text = el("textarea", { name: "text", required: true }) as HTMLTextAreaElement;
  text.value = prefill;
  form.append(el("label", {}, "Title", title), el("label", {}, "Kind", kind), el("label", {}, "Note", text), el("button", { type: "submit" }, "Save note"));
  const out = el("div");
  form.addEventListener("submit", async (ev) => {
    ev.preventDefault();
    try {
      const put = await data("case_put", { table: "note", data: { title: title.value || text.value.slice(0, 80), kind: kind.value, text: text.value, created_at: new Date().toISOString(), added_via: "toolkit-web" } });
      sheet.close();
      if (location.hash === "#notes") await viewNotes();
      main.prepend(okBox(`Saved note:${String(put.id)}.`));
    } catch (err) {
      out.replaceChildren(errBox(err));
    }
  });
  openSheet("Add a note", form, out);
}

async function viewReferences(): Promise<void> {
  const host = page("Cheat sheets", "Cheat sheets, templates, guides and reference material in the case store.");
  try {
    const tabs = el("div", { class: "tabs", role: "group", "aria-label": "Kinds" });
    const body = el("div");
    let selectedKind = "cheat_sheet";
    let kinds: string[] = [];
    const show = (kind: string) => {
      selectedKind = kind;
      for (const b of tabs.querySelectorAll("button")) b.setAttribute("aria-pressed", String(b.dataset.kind === kind));
      body.querySelector<HTMLInputElement>("input.filter")?.dispatchEvent(new Event("input"));
    };
    host.append(tabs, body);
    await pagedRecordList(body, ["reference"], (r) => String(r.id), () => void viewReferences(), "No reference records.",
      (rows) => rows.filter((r) => selectedKind === "all" || r.kind === selectedKind),
      (rows) => {
        const nextKinds = [...new Set(rows.map((r) => String(r.kind)))].sort();
        if (nextKinds.join("\0") === kinds.join("\0") && tabs.childElementCount > 0) return;
        kinds = nextKinds;
        if (selectedKind !== "all" && !kinds.includes(selectedKind)) selectedKind = "all";
        tabs.replaceChildren(...["all", ...kinds].map((kind) => el("button", { type: "button", "data-kind": kind, "aria-pressed": String(selectedKind === kind), onclick: () => show(kind) }, kind)));
      });
  } catch (err) {
    host.append(errBox(err));
  }
}

async function viewLaw(): Promise<void> {
  const host = page("Law & sources", "Verified legal sources: statutes, court rules, forms and guidance.");
  const auditHost = el("div");
  const audit = tools.find((t) => t.name === "audit_sources");
  host.append(el("div", { class: "actions" }, audit ? el("button", { type: "button", onclick: () => void renderWidget(auditHost, audit, {}).catch((e) => auditHost.replaceChildren(errBox(e))) }, "Audit sources") : null), auditHost);
  try {
    await pagedRecordList(host, ["source"], (r) => `${String(r.authority_class ?? "")} · verified ${String(r.last_verified ?? r.date_accessed ?? "?")}`, () => void viewLaw(), "No sources.");
  } catch (err) {
    host.append(errBox(err));
  }
}

async function viewSearch(): Promise<void> {
  const host = page("Search", "Search the case store and the reviewed guide records.");
  const form = el("form", {});
  const q = el("input", { name: "q", type: "search", required: true, placeholder: "What are you looking for?" }) as HTMLInputElement;
  const results = el("div");
  form.append(el("label", {}, "Search", q), el("button", { type: "submit" }, "Search"));
  form.addEventListener("submit", async (ev) => {
    ev.preventDefault();
    results.replaceChildren(el("p", { class: "lede", text: "Searching…" }));
    try {
      const hits = (await data("case_search", { query: q.value, k: 30 })).hits as Json[] ?? [];
      const guide = el("div");
      results.replaceChildren(el("h2", { text: `Case store (${hits.length})` }),
        el("div", { class: "list" }, ...(hits.length ? hits.map((h) => recordRow({ ...h, title: h.snippet ?? h.title ?? h.id }, String(h.id), () => void openRecord(String(h.id)))) : [el("p", { class: "empty", text: "No case-store matches." })])),
        el("h2", { text: "Guide records" }), guide);
      const tool = tools.find((t) => t.name === "search_guide");
      if (tool) await renderWidget(guide, tool, { query: q.value });
    } catch (err) {
      results.replaceChildren(errBox(err));
    }
  });
  host.append(form, results);
}

const GUIDE_TOOLS = ["open_dashboard", "route_issue", "calculate_planning_date", "get_packet_plan", "get_checklist", "build_chronology", "survival_guide", "court_language_review"];
const GUIDE_PRESETS: Record<string, Json> = {
  build_chronology: { events: [{ date: "2026-01-01", title: "What happened", source: "Where it is recorded" }] },
};

async function viewGuide(): Promise<void> {
  const host = page("Guide tools", "Route an issue, plan dates, build packets and checklists.");
  const tabs = el("div", { class: "tabs", role: "group", "aria-label": "Guide tools" });
  const body = el("div");
  const show = (name: string) => {
    const tool = tools.find((t) => t.name === name);
    if (!tool) return;
    for (const b of tabs.querySelectorAll("button")) b.setAttribute("aria-pressed", String(b.dataset.tool === name));
    const out = el("div");
    body.replaceChildren(el("p", { class: "lede", text: tool.description ?? "" }), toolForm(tool, (args) => renderWidget(out, tool, args), GUIDE_PRESETS[name]), out);
    if (!Object.keys((tool.inputSchema as { properties?: Json })?.properties ?? {}).length) void renderWidget(out, tool, {}).catch((e) => out.replaceChildren(errBox(e)));
  };
  for (const name of GUIDE_TOOLS) {
    const tool = tools.find((t) => t.name === name);
    if (tool) tabs.append(el("button", { type: "button", "data-tool": name, onclick: () => show(name) }, tool.title ?? name));
  }
  host.append(tabs, body);
  show("open_dashboard");
}

// Byline: Codex, 2026-10-04.
/** Display the shared reference and source library used by the other legal surfaces.
 * Inputs: none; table pages resolve through the authenticated shared-record API.
 * Outputs: a searchable, bounded list opening exact record versions and bodies.
 * Side effects: reads shared records only; failures remain visible without falling back to packaged files.
 * Use for the working library; viewPackagedFiles is explicit access to the shipped source snapshot.
 */
async function viewLibrary(): Promise<void> {
  const host = page("Library", "Guides, checklists, templates and legal sources from the shared library.");
  host.append(el("div", { class: "actions" },
    el("button", { type: "button", class: "secondary", onclick: () => void viewPackagedFiles() }, "Packaged files")));
  try {
    await pagedRecordList(host, ["reference", "source"],
      (record) => String(record.source_path ?? record.citation ?? record.id),
      () => void viewLibrary(), "No shared library records.");
  } catch (err) {
    host.append(errBox(err));
  }
}

// Byline: Codex, 2026-10-04.
/** Browse the files shipped with this console as a separately identified package snapshot.
 * Inputs: none; outputs: bounded file pages with original-byte downloads.
 * Side effects: reads packaged metadata and files only, without changing the shared library.
 * Use for packaged originals and PDFs; viewLibrary is the current shared working information.
 */
async function viewPackagedFiles(): Promise<void> {
  const host = page("Packaged files", "Files shipped with this console build. Open Library for the shared working records.");
  host.append(el("div", { class: "actions" },
    el("button", { type: "button", class: "secondary", onclick: () => void viewLibrary() }, "Back to Library")));
  try {
    const { files } = await api<{ files: Array<{ path: string; size: number }> }>("/api/library");
    const list = el("div", { class: "list" });
    const count = el("p", { class: "m", role: "status" });
    const more = el("button", { class: "btn secondary", type: "button", text: "Show more" }) as HTMLButtonElement;
    let currentPage = 1;
    let currentQuery = "";
    const draw = (q: string) => {
      const page = libraryPage(files, q, currentPage, currentQuery);
      currentPage = page.page;
      currentQuery = q;
      const start = (page.page - 1) * 100;
      const shown = page.visible;
      count.textContent = `Showing ${Math.min(start + shown.length, page.total)} of ${page.total} matching files${q ? ` (${files.length} total)` : ""}.`;
      more.hidden = start + shown.length >= page.total;
      list.replaceChildren(...shown.map((f) => el("button", { class: "row", type: "button", onclick: () => void openFile(f.path) },
        el("span", {}, el("span", { class: "t", text: f.path.split("/").pop() ?? f.path }), el("br"), el("span", { class: "m", text: f.path })),
        el("span", { class: "tag", text: `${Math.max(1, Math.round(f.size / 1024))} KB` }))));
    };
    more.addEventListener("click", () => { currentPage += 1; draw((host.querySelector("input.filter") as HTMLInputElement).value.trim().toLowerCase()); });
    host.append(filterInput(`Filter ${files.length} files`, (q) => { currentPage = 1; draw(q); }), count, list, more);
    draw("");
  } catch (err) {
    host.append(errBox(err));
  }
}

async function openFile(path: string): Promise<void> {
  openSheet("Loading…");
  try {
    const f = await api<{ path: string; version: string; text: string | null }>(`/api/library/file?path=${encodeURIComponent(path)}`);
    const raw = el("a", { class: "btn secondary", href: `/api/library/file?path=${encodeURIComponent(path)}&raw=1`, target: "_blank", rel: "noopener" }, "Open original");
    openSheet(path.split("/").pop() ?? path, el("p", { class: "idline", text: `${path} · ${f.version}` }), el("div", { class: "actions" }, raw),
      f.text === null ? null : path.endsWith(".md") ? markdown(f.text) : el("pre", { class: "json", text: f.text }));
  } catch (err) {
    openSheet(path, errBox(err));
  }
}

async function viewTools(): Promise<void> {
  const host = page("All tools", `Every operation the toolkit exposes (${tools.length}). Reads and writes run against the live case store.`);
  const list = el("div", { class: "list" });
  for (const tool of [...tools].sort((a, b) => a.name.localeCompare(b.name))) {
    const out = el("div");
    list.append(el("details", { class: "card" },
      el("summary", {}, el("strong", { text: tool.name }), " — ", tool.title ?? ""),
      el("p", { class: "lede", text: tool.description ?? "" }),
      toolForm(tool, (args) => renderWidget(out, tool, args)), out));
  }
  host.append(list);
}

const VIEWS: Record<string, () => Promise<void>> = {
  case: viewCase, documents: viewDocuments, notes: viewNotes, cheatsheets: viewReferences, law: viewLaw,
  search: viewSearch, guide: viewGuide, library: viewLibrary, tools: viewTools,
};

async function route(): Promise<void> {
  const view = location.hash.slice(1) in VIEWS ? location.hash.slice(1) : "case";
  for (const a of document.querySelectorAll<HTMLAnchorElement>("#nav a")) {
    if (a.dataset.view === view) a.setAttribute("aria-current", "page");
    else a.removeAttribute("aria-current");
  }
  $("#nav").classList.remove("open");
  $("#navToggle").setAttribute("aria-expanded", "false");
  await VIEWS[view]();
}

async function start(): Promise<void> {
  $("#navToggle").addEventListener("click", () => {
    const open = $("#nav").classList.toggle("open");
    $("#navToggle").setAttribute("aria-expanded", String(open));
  });
  window.addEventListener("hashchange", () => void route());
  try {
    const [who, list] = await Promise.all([api<{ principal: string; via: string }>("/api/whoami"), api<{ tools: Tool[] }>("/api/tools")]);
    tools = list.tools;
    const status = await data("case_status", { action: "get" }).catch(() => ({} as Json));
    const posture = ((status.status ?? status) as Json).posture;
    $("#posture").textContent = [typeof posture === "string" ? posture : "Case store connected", who.principal].join(" · ");
  } catch (err) {
    $("#posture").textContent = "Not connected";
    main.replaceChildren(errBox(err));
    return;
  }
  await route();
}

void start();
