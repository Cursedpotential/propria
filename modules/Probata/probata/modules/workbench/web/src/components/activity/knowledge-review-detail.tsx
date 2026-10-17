/** Exact, version-guarded knowledge decision control inside EXISTING Activity.
 * Input: selected canonical PostgreSQL review reference and signed-in Workbench owner.
 * Output: reviewed source/revision pins and decision receipt; never mutates memory sources.
 * Prefer this over a second review/approval dashboard or generic evidence promotion.
 */
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

type Choice = "approve_plan" | "reject" | "defer";
type Detail = {
  schema: "propria-knowledge-review-detail/v1";
  review_ref: string;
  kind: string;
  severity: string;
  status: string;
  scope: string;
  summary: string;
  source_refs: string[];
  source_revisions: { ref: string; revision: string }[];
  case_version: number;
  final_disposition: string | null;
};
async function loadDetail(ref: string, signal?: AbortSignal): Promise<Detail> {
  const response = await fetch("/api/knowledge-review/" + encodeURIComponent(ref), {
    method: "GET",
    credentials: "same-origin",
    headers: { Accept: "application/json" },
    signal
  });
  if (!response.ok) throw new Error("Review detail unavailable (HTTP " + response.status + ")");
  const data = await response.json();
  if (data.schema !== "propria-knowledge-review-detail/v1" || data.review_ref !== ref)
    throw new Error("Canonical review identity mismatch");
  return data as Detail;
}
export function KnowledgeReviewDetail({ reviewRef }: { reviewRef: string }) {
  const client = useQueryClient();
  const [choice, setChoice] = useState<Choice>("defer");
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState(false);
  const [feedback, setFeedback] = useState("");
  const [error, setError] = useState("");
  const query = useQuery({
    queryKey: ["knowledge-review-detail", reviewRef],
    queryFn: ({ signal }) => loadDetail(reviewRef, signal),
    staleTime: 5_000,
    retry: false
  });
  async function decide() {
    const current = query.data;
    if (!current || !["queued","deferred"].includes(current.status) || pending) return;
    if (reason.trim().length < 2) {setError("A reason is required.");return;}
    setPending(true);setError("");setFeedback("");
    try{
      const response = await fetch("/api/knowledge-review/" + encodeURIComponent(reviewRef) + "/decision",{
        method:"POST",credentials:"same-origin",
        headers:{"content-type":"application/json"},
        body:JSON.stringify({
          expected_case_version:current.case_version,
          request_id:crypto.randomUUID(),
          decision:choice,
          rationale:reason.trim()
        })
      });
      if(!response.ok)throw new Error(response.status===409?
        "This review changed; reload before deciding.":
        "Review decision not recorded (HTTP " + response.status + ").");
      const result=await response.json();
      if(result.review_ref!==reviewRef||result.decision!==choice||result.source_applied!==false)
        throw new Error("Decision response does not match the selected case");
      setFeedback(choice==="approve_plan"?
        "Proposed change approved. Its native source has NOT been updated yet.":
        choice==="defer"?"Review deferred; no source changed.":
        "Proposal rejected; original records remain intact.");
      setReason("");
      await Promise.all([
        client.invalidateQueries({queryKey:["knowledge-review-inbox"]}),
        client.invalidateQueries({queryKey:["knowledge-review-detail",reviewRef]})
      ]);
    }catch(e){setError(e instanceof Error?e.message:"Review decision failed");}
    finally{setPending(false);}
  }
  return (
    <div className="space-y-3 border-t pt-4" aria-label="Knowledge review decision">
      {query.isPending?<p role="status" className="text-xs text-muted-foreground">Loading source references…</p>:null}
      {query.isError?<p role="alert" className="text-xs text-destructive">{query.error.message}</p>:null}
      {query.data?<>
        <h3 className="text-sm font-semibold">{query.data.summary}</h3>
        <p className="break-all text-xs text-muted-foreground">
          {query.data.kind.replaceAll("_"," ")} · {query.data.severity} · {query.data.status} · version {query.data.case_version}
        </p>
        <p className="text-xs font-medium">Original source IDs and expected revisions</p>
        <ul className="max-h-44 space-y-1 overflow-auto rounded border p-2 text-xs">
          {query.data.source_revisions.map(item=><li key={item.ref+item.revision} className="break-all">
            <code>{item.ref}</code> · revision {item.revision}
          </li>)}
        </ul>
        {["queued","deferred"].includes(query.data.status)?<>
          <label htmlFor="knowledge-review-choice" className="block text-xs font-medium">Decision</label>
          <select id="knowledge-review-choice" value={choice} onChange={e=>setChoice(e.target.value as Choice)}
            className="w-full max-w-lg rounded border bg-background px-2 py-2 text-sm">
            <option value="defer">Defer for more investigation</option>
            <option value="reject">Reject the proposed action</option>
            <option value="approve_plan">Approve proposed action (native writer still required)</option>
          </select>
          <label htmlFor="knowledge-review-reason" className="block text-xs font-medium">Reason</label>
          <textarea id="knowledge-review-reason" value={reason} onChange={e=>setReason(e.target.value)}
            rows={2} maxLength={2000} className="w-full max-w-2xl rounded border bg-background px-3 py-2 text-sm"
            placeholder="Why is this decision correct? Include any necessary correction."/>
          <button type="button" disabled={pending||reason.trim().length<2}
            onClick={()=>void decide()} className="rounded border px-3 py-2 text-xs font-semibold disabled:opacity-50">
            {pending?"Recording…":"Record owner decision"}
          </button>
        </>:<p className="text-xs text-muted-foreground">
          Final disposition: {query.data.final_disposition||"recorded"}. Check the native writer receipt before claiming a memory was changed.
        </p>}
      </>:null}
      {feedback?<p role="status" className="text-xs text-emerald-700 dark:text-emerald-300">{feedback}</p>:null}
      {error?<p role="alert" className="text-xs text-destructive">{error}</p>:null}
    </div>
  );
}
