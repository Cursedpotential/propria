"use client";

import { useEffect, useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { getSourcePinnedToolStatus, startSourcePinnedToolAction } from "@/lib/api-client";

const SOURCE_TOOLS = [
  { id: "repair.detect", label: "Inspect source format" },
  { id: "repair.preview", label: "Preview possible repair" },
  { id: "repair.pdf-inspect", label: "Inspect PDF integrity" },
  { id: "documents.extract-text", label: "Extract document text" },
] as const;

type Status = Awaited<ReturnType<typeof getSourcePinnedToolStatus>>;

/** Offer the first approved source-file tools through a real Temporal action.
 * Inputs are an immutable locator, reviewed SHA-256 and optional JSON options;
 * output is workflow/run status plus a content-store ref and audit chain head.
 * This is separate from direct MCP browsing because the browser never runs a
 * tool or names a host path.
 */
export function SourcePinnedAction({ initialSourceRef = "", initialSHA256 = "" }: { initialSourceRef?: string; initialSHA256?: string }) {
  const [toolID, setToolID] = useState<(typeof SOURCE_TOOLS)[number]["id"]>(SOURCE_TOOLS[0].id);
  const [sourceRef, setSourceRef] = useState(initialSourceRef);
  const [sha256, setSha256] = useState(initialSHA256);
  const [options, setOptions] = useState("{}");
  const [clickKey, setClickKey] = useState(() => crypto.randomUUID());
  const [started, setStarted] = useState<{ workflow_id: string; run_id: string } | null>(null);
  const [status, setStatus] = useState<Status | null>(null);
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  useEffect(() => {
    if (!started || status?.outcome === "completed" || status?.outcome === "failed") return;
    let active = true;
    const check = () => void getSourcePinnedToolStatus(started.workflow_id)
      .then((next) => { if (active) { setStatus(next); setError(""); } })
      .catch((cause) => { if (active) setError(cause instanceof Error ? cause.message : "Could not read workflow status"); });
    check();
    const timer = window.setInterval(check, 2500);
    return () => { active = false; window.clearInterval(timer); };
  }, [started, status?.outcome]);

  async function start(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    let args: Record<string, unknown>;
    try {
      const parsed: unknown = JSON.parse(options);
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) throw new Error("Options must be a JSON object");
      args = parsed as Record<string, unknown>;
      if (Object.keys(args).some((key) => key === "path" || key.startsWith("_"))) throw new Error("A host path or reserved option cannot be sent");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Options must be a JSON object");
      return;
    }
    setPending(true);
    try {
      const next = await startSourcePinnedToolAction({ tool_id: toolID, source_ref: sourceRef.trim(), source_sha256: sha256.trim(), args }, clickKey);
      setStarted(next);
      setStatus(null);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Could not start the source action");
    } finally { setPending(false); }
  }

  return <section id="source-pinned-action" className="border bg-card p-5" aria-label="Source-pinned tool action">
    <p className="platform-rule-title">One source · one tool</p>
    <h2 className="mt-1 text-lg font-semibold">Run a source-backed tool</h2>
    <p className="mt-1 text-sm text-muted-foreground">Choose an existing immutable source and enter its verified SHA-256. The tool reads a checked copy; the result stays in the content store.</p>
    <form onSubmit={start} className="mt-4 grid gap-3 md:grid-cols-2">
      <label className="text-sm">Tool<select className="mt-1 block h-10 w-full border bg-background px-3" value={toolID} onChange={(event) => { setToolID(event.target.value as typeof toolID); setClickKey(crypto.randomUUID()); }}>
        {SOURCE_TOOLS.map((tool) => <option key={tool.id} value={tool.id}>{tool.label}</option>)}
      </select></label>
      <label className="text-sm">Immutable source locator<Input className="mt-1" value={sourceRef} onChange={(event) => { setSourceRef(event.target.value); setClickKey(crypto.randomUUID()); }} placeholder="r2://… or b2://…" required /></label>
      <label className="text-sm">Verified SHA-256<Input className="mt-1 font-mono" value={sha256} onChange={(event) => { setSha256(event.target.value); setClickKey(crypto.randomUUID()); }} pattern="[0-9a-f]{64}" maxLength={64} required /></label>
      <label className="text-sm">Options (JSON)<Input className="mt-1 font-mono" value={options} onChange={(event) => { setOptions(event.target.value); setClickKey(crypto.randomUUID()); }} /></label>
      <div className="md:col-span-2"><Button type="submit" disabled={pending || !!started && status?.outcome !== "failed" && status?.outcome !== "completed"}>{pending ? "Starting…" : "Run read-only tool"}</Button></div>
    </form>
    {error && <p role="alert" className="mt-3 text-sm text-destructive">{error}</p>}
    {started && <div className="mt-4 border-t pt-3 text-sm" role="status">
      <p>Workflow: <code className="break-all">{started.workflow_id}</code></p>
      <p>Run: <code className="break-all">{started.run_id}</code></p>
      <p>State: {status?.outcome ?? "accepted"}</p>
      {status?.error && <p className="text-destructive">{status.error}</p>}
      {status?.result && <>
        <p>Result reference: <code className="break-all">{status.result.result_ref}</code></p>
        <p>Audit chain: <code className="break-all">{status.result.audit_chain_head}</code></p>
      </>}
      {(status?.outcome === "completed" || status?.outcome === "failed") && <Button type="button" variant="outline" className="mt-2" onClick={() => { setStarted(null); setStatus(null); setClickKey(crypto.randomUUID()); }}>New action</Button>}
    </div>}
  </section>;
}
