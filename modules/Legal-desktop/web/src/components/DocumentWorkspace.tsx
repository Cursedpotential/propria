"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { legalApiBase } from "@/lib/api/client";
import styles from "./DocumentWorkspace.module.css";

type WorkDocument = {
  document_id: string; title: string; filename: string; format: string;
  current_revision: number; updated_at: string;
};
type OfficeSession = {
  document_id: string; revision: number; editor_url: string;
  access_token: string; access_token_ttl: number;
};
type OfficeTemplate = { template_id: string; title: string; description?: string; version: string };

async function readResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let message = `Request failed (${response.status}).`;
    try {
      const result = await response.json();
      if (typeof result.detail === "string") message = result.detail;
      else if (result.detail?.message) message = result.detail.message;
    } catch { /* Preserve readable fallback when the service returns HTML. */ }
    throw new Error(message);
  }
  return response.json();
}

export function DocumentWorkspace() {
  const [documents, setDocuments] = useState<WorkDocument[]>([]);
  const [templates, setTemplates] = useState<OfficeTemplate[]>([]);
  const [selected, setSelected] = useState<WorkDocument | null>(null);
  const [session, setSession] = useState<OfficeSession | null>(null);
  const [title, setTitle] = useState("");
  const [templateId, setTemplateId] = useState("");
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editorStatus, setEditorStatus] = useState("");
  const [history, setHistory] = useState<Array<{ revision: number; created_at?: string; filename?: string }>>([]);
  const frame = useRef<HTMLIFrameElement>(null);
  const editorForm = useRef<HTMLFormElement>(null);
  const fileInput = useRef<HTMLInputElement>(null);
  const selectionGeneration = useRef(0);
  const editorReady = useRef(false);
  const leaveAllowed = useRef(false);
  const saveWaiter = useRef<{ resolve: (saved: boolean) => void; timer: ReturnType<typeof setTimeout> } | null>(null);

  const refresh = useCallback(async () => {
    const rows = await readResponse<WorkDocument[]>(await fetch(`${legalApiBase()}/v1/work-documents`, { cache: "no-store" }));
    setDocuments(rows);
    return rows;
  }, []);

  useEffect(() => {
    let active = true;
    void refresh().catch((exc) => { if (active) setError(exc.message); }).finally(() => { if (active) setLoading(false); });
    void fetch(`${legalApiBase()}/v1/office-templates`).then(readResponse<OfficeTemplate[]>).then((rows) => {
      if (active) setTemplates(rows);
    }).catch(() => { /* Blank writing and import remain usable if templates are unavailable. */ });
    return () => { active = false; selectionGeneration.current += 1; };
  }, [refresh]);

  useEffect(() => {
    if (!session) return;
    editorReady.current = false;
    leaveAllowed.current = false;
    const origin = new URL(session.editor_url).origin;
    const listener = (event: MessageEvent) => {
      if (event.origin !== origin || event.source !== frame.current?.contentWindow) return;
      let message;
      try { message = typeof event.data === "string" ? JSON.parse(event.data) : event.data; } catch { return; }
      if (message?.MessageId === "App_LoadingStatus" && ["Initialized", "Document_Loaded"].includes(message.Values?.Status)) {
        frame.current?.contentWindow?.postMessage(JSON.stringify({ MessageId: "Host_PostmessageReady", SendTime: Date.now(), Values: {} }), origin);
        if (message.Values.Status === "Document_Loaded") {
          editorReady.current = true;
          setEditorStatus("Editor ready");
        }
      }
      if (message?.MessageId === "Action_Save_Resp") {
        const saved = message.Values?.success === true;
        if (saved) {
          setEditorStatus(`Last save confirmed at ${new Date().toLocaleTimeString()}`);
          const generation = selectionGeneration.current;
          void refresh().then((rows) => {
            if (generation === selectionGeneration.current) setSelected(rows.find(row => row.document_id === session.document_id) ?? null);
          }).catch(() => { if (generation === selectionGeneration.current) setEditorStatus("Saved · list refresh unavailable"); });
        } else setEditorStatus("Save failed — keep this document open and retry");
        if (saveWaiter.current) {
          clearTimeout(saveWaiter.current.timer);
          saveWaiter.current.resolve(saved);
          saveWaiter.current = null;
        }
      }
    };
    window.addEventListener("message", listener);
    const beforeUnload = (event: BeforeUnloadEvent) => { if (!leaveAllowed.current) { event.preventDefault(); event.returnValue = ""; } };
    window.addEventListener("beforeunload", beforeUnload);
    const beforeNavigate = (event: MouseEvent) => {
      const anchor = event.target instanceof Element ? event.target.closest("a[href]") as HTMLAnchorElement | null : null;
      if (!anchor || anchor.target === "_blank" || event.ctrlKey || event.metaKey || event.shiftKey || anchor.download) return;
      if (anchor.href.startsWith("javascript:") || anchor.getAttribute("href")?.startsWith("#")) return;
      event.preventDefault(); event.stopPropagation();
      void saveDocument().then(saved => {
        if (!saved) { setError("Save was not confirmed. Your document remains open; retry saving before leaving."); return; }
        leaveAllowed.current = true;
        window.location.assign(anchor.href);
      });
    };
    document.addEventListener("click", beforeNavigate, true);
    editorForm.current?.submit();
    return () => {
      window.removeEventListener("message", listener);
      window.removeEventListener("beforeunload", beforeUnload);
      document.removeEventListener("click", beforeNavigate, true);
      editorReady.current = false;
      if (saveWaiter.current) { clearTimeout(saveWaiter.current.timer); saveWaiter.current.resolve(false); saveWaiter.current = null; }
    };
  }, [session, refresh]);

  async function openDocument(document: WorkDocument) {
    if (session && !(await saveDocument())) {
      setError("Finish saving the open document before switching. Your editor is still open.");
      return;
    }
    const generation = ++selectionGeneration.current;
    setSelected(document); setSession(null); setHistory([]); setError(null); setEditorStatus("Opening editor…");
    try {
      const result = await readResponse<OfficeSession>(await fetch(`${legalApiBase()}/v1/work-documents/${document.document_id}/office-session`, { method: "POST" }));
      if (generation !== selectionGeneration.current) return;
      // An editor URL is provided only by the authenticated office service.
      const editorUrl = new URL(result.editor_url);
      if (!["http:", "https:"].includes(editorUrl.protocol)) throw new Error("The editor address is invalid.");
      setSession(result);
    } catch (exc) {
      if (generation === selectionGeneration.current) {
        setError(exc instanceof Error ? exc.message : "The editor could not open.");
        setEditorStatus("Editor unavailable");
      }
    }
  }

  async function createDocument() {
    setBusy(true); setError(null);
    try {
      const endpoint = templateId ? `/v1/office-templates/${encodeURIComponent(templateId)}/instantiate` : "/v1/work-documents";
      const row = await readResponse<WorkDocument>(await fetch(`${legalApiBase()}${endpoint}`, {
        method: "POST", headers: { "content-type": "application/json" },
        body: JSON.stringify({ title: title.trim() || "Untitled document" }),
      }));
      await refresh(); setTitle(""); await openDocument(row);
    } catch (exc) { setError(exc instanceof Error ? exc.message : "The document could not be created."); }
    finally { setBusy(false); }
  }

  async function importDocument(file: File) {
    setBusy(true); setError(null);
    try {
      const data = new FormData(); data.set("file", file);
      const row = await readResponse<WorkDocument>(await fetch(`${legalApiBase()}/v1/work-documents/import`, { method: "POST", body: data }));
      await refresh(); await openDocument(row);
    } catch (exc) { setError(exc instanceof Error ? exc.message : "The document could not be imported."); }
    finally { setBusy(false); if (fileInput.current) fileInput.current.value = ""; }
  }

  function saveDocument(): Promise<boolean> {
    if (!session) return Promise.resolve(true);
    if (!editorReady.current || saveWaiter.current) return Promise.resolve(false);
    setEditorStatus("Saving…");
    return new Promise(resolve => {
      const timer = setTimeout(() => {
        saveWaiter.current = null;
        setEditorStatus("Save not confirmed — keep this document open and retry");
        resolve(false);
      }, 20000);
      saveWaiter.current = { resolve, timer };
      frame.current?.contentWindow?.postMessage(JSON.stringify({
        MessageId: "Action_Save", SendTime: Date.now(), Values: { DontTerminateEdit: true, DontSaveIfUnmodified: false, Notify: true },
      }), new URL(session.editor_url).origin);
    });
  }

  async function showHistory() {
    if (!selected) return;
    const generation = selectionGeneration.current;
    try {
      const rows = await readResponse<typeof history>(await fetch(`${legalApiBase()}/v1/work-documents/${selected.document_id}/revisions`, { cache: "no-store" }));
      if (generation === selectionGeneration.current) setHistory(rows);
    } catch (exc) { setError(exc instanceof Error ? exc.message : "Version history is unavailable."); }
  }

  return <section className={styles.workspace} aria-label="Writing workspace">
    <div className={styles.start}>
      <form onSubmit={event => { event.preventDefault(); void createDocument(); }} className={styles.create}>
        <label>Document name<input value={title} onChange={event => setTitle(event.target.value)} placeholder="Untitled document" maxLength={200} /></label>
        <label>Starting point<select value={templateId} onChange={event => setTemplateId(event.target.value)}>
          <option value="">Blank document</option>
          {templates.map(template => <option key={template.template_id} value={template.template_id}>{template.title}</option>)}
        </select></label>
        <button disabled={busy} type="submit">{busy ? "Working…" : "Start writing"}</button>
      </form>
      <button type="button" disabled={busy} onClick={() => fileInput.current?.click()}>Open an existing document</button>
      <input ref={fileInput} type="file" accept=".docx,.odt" hidden onChange={event => { const file = event.target.files?.[0]; if (file) void importDocument(file); }} />
    </div>
    {error ? <p role="alert" className={styles.error}>{error}</p> : null}
    <div className={styles.body}>
      <div className={styles.library} aria-label="Saved documents">
        <h2>Your documents</h2>
        {loading ? <p>Loading documents…</p> : documents.length === 0 ? <p>Start a blank document or open a DOCX or ODT file.</p> : null}
        {documents.map(document => <button type="button" key={document.document_id} className={styles.document} aria-pressed={selected?.document_id === document.document_id} onClick={() => void openDocument(document)}>
          <strong>{document.title}</strong><span>{document.format.toUpperCase()} · Version {document.current_revision}</span>
        </button>)}
      </div>
      <div className={styles.editing}>
        {selected ? <>
          <div className={styles.toolbar}>
            <strong>{selected.title}</strong><span role="status">{editorStatus}</span>
            <button type="button" disabled={!session || !editorReady.current || editorStatus === "Saving…"} onClick={() => void saveDocument()}>Save</button>
            <a href={`${legalApiBase()}/v1/work-documents/${selected.document_id}/content`}>Download</a>
            <button type="button" onClick={() => void showHistory()}>Versions</button>
          </div>
          {history.length ? <div className={styles.history} aria-label="Document versions">
            {history.map(revision => <a key={revision.revision} href={`${legalApiBase()}/v1/work-documents/${selected.document_id}/content?revision=${revision.revision}`}>Version {revision.revision}</a>)}
          </div> : null}
          {session ? <>
            <form ref={editorForm} action={session.editor_url} method="post" target="advocatio-office-editor" hidden>
              <input type="hidden" name="access_token" value={session.access_token} />
              <input type="hidden" name="access_token_ttl" value={session.access_token_ttl} />
            </form>
            <iframe ref={frame} name="advocatio-office-editor" title={`Edit ${selected.title}`} className={styles.editor} allow="clipboard-read; clipboard-write; fullscreen" />
          </> : <div className={styles.empty}><p>{editorStatus}</p>{editorStatus === "Editor unavailable" ? <button type="button" onClick={() => void openDocument(selected)}>Try opening again</button> : null}</div>}
        </> : <div className={styles.empty}><p>Choose a document or start writing.</p><p>Your original files and saved versions stay available here.</p></div>}
      </div>
    </div>
  </section>;
}
