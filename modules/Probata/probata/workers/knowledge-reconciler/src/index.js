/**
 * Propria knowledge reconciliation: Cloudflare ingress to ONE complete n8n workflow.
 *
 * Inputs: daily Cron, authorized manual request, source-change event with canonical refs.
 * Outputs: compact n8n workflow receipt with verified persistent HITL review references.
 * Effects: exactly one invocation of the n8n workflow; no database, notification, rule
 *   or memory writes at the edge. Choose this ingress instead of chaining atomic jobs.
 *
 * The n8n workflow owns discovery, comparison, deduplication, idempotent review
 * persistence, notifications, reminders and an auditable completion receipt.
 */
const MAX_BODY_BYTES = 16384;
const MAX_REPLY_BYTES = 131072;
const SOURCES = new Set([
  "memsearch", "engram", "cnf", "docstore", "propria-search",
  "superindex", "casebible", "workbench", "ccc"
]);
const EVENTS = new Set([
  "memory.created", "memory.changed", "memory.retired",
  "document.created", "document.changed", "document.retired",
  "record.created", "record.changed", "record.retired"
]);
const STATUSES = new Set(["completed", "review_pending", "partial", "blocked"]);
function response(body, status=200) {
  return new Response(JSON.stringify(body), {status, headers: {
    "content-type": "application/json; charset=utf-8",
    "cache-control": "no-store",
    "x-content-type-options": "nosniff"
  }});
}
function value(x, max=512) {return typeof x==="string" ? x.trim().slice(0,max) : "";}
function eventInput(body) {
  if (!EVENTS.has(body.type) || !SOURCES.has(body.source))
    throw new Error("unknown event or source");
  if (Object.hasOwn(body,"content") || Object.hasOwn(body,"raw") || Object.hasOwn(body,"binary"))
    throw new Error("triggers must contain reference IDs, not source content");
  const ref=value(body.record_ref,1024), revision=value(body.revision,256), scope=value(body.scope,120);
  if (!ref || !revision || !(scope==="propria" || scope.startsWith("propria/")))
    throw new Error("record_ref, revision and a Propria scope are required");
  return {kind:"event",type:body.type,source:body.source,record_ref:ref,revision,scope};
}
function manualInput(body) {
  const scope=value(body.scope,120)||"propria", source=value(body.source,60);
  if (!(scope==="propria" || scope.startsWith("propria/")))
    throw new Error("unrecognized scope");
  if (source && !SOURCES.has(source))throw new Error("unrecognized source");
  if (Object.hasOwn(body,"content") || Object.hasOwn(body,"records"))
    throw new Error("supply scoped references, not full records");
  return {kind:"manual",scope,source:source||null};
}
async function validUser(request,env) {
  const expected=value(env.RECONCILER_TRIGGER_TOKEN,4096);
  if (!expected) return false;
  const auth=request.headers.get("authorization")||"";
  if (!auth.startsWith("Bearer "))return false;
  const a=new TextEncoder().encode(auth.slice(7)),b=new TextEncoder().encode(expected);
  let diff=a.length^b.length;
  for(let i=0;i<Math.max(a.length,b.length);i++) diff|=(a[i]||0)^(b[i]||0);
  return diff===0;
}
async function bodyInput(request) {
  if(Number(request.headers.get("content-length")||"0")>MAX_BODY_BYTES)
    throw new Error("input too large");
  const raw=await request.text();
  if(new TextEncoder().encode(raw).length>MAX_BODY_BYTES)
    throw new Error("input too large");
  let body;
  try{body=JSON.parse(raw);}catch{throw new Error("invalid JSON");}
  if(!body || typeof body!=="object" || Array.isArray(body))throw new Error("expected input object");
  return body;
}
function verifiedN8nResult(result){
  if(!result || result.schema!=="propria-knowledge-workflow-result/v1" ||
    typeof result.run_id!=="string" || !result.run_id ||
    !STATUSES.has(result.status) || !Array.isArray(result.review_queue) ||
    !Array.isArray(result.coverage) || !Array.isArray(result.alerts))
    throw new Error("n8n workflow response does not meet verified receipt contract");
  if(result.review_queue.length>100 || result.coverage.length>30 || result.alerts.length>100)
    throw new Error("unbounded workflow receipt");
  const queue=result.review_queue.map(row=>{
    if(typeof row?.review_ref!=="string" || !row.review_ref ||
      !["queued","deferred","decided"].includes(row.status))
      throw new Error("review without canonical identity");
    return {review_ref:value(row.review_ref),status:row.status,
      severity:value(row.severity,16),kind:value(row.kind,80),
      newly_created:row.newly_created===true};
  });
  return {schema:"propria-knowledge-workflow-result/v1",
    status:result.status,run_id:value(result.run_id,128),
    counts:{
      scanned:Number.isSafeInteger(result.scanned)?result.scanned:null,
      findings:Number.isSafeInteger(result.findings)?result.findings:null,
      new_reviews:queue.filter(x=>x.newly_created).length
    },
    review_queue:queue,
    coverage:result.coverage.map(row=>({
      store:value(row?.store,64),
      state:value(row?.state,32),
      reason:value(row?.reason,180)
    })),
    alerts:result.alerts.map(row=>({
      review_ref:value(row?.review_ref),status:value(row?.status,32)
    }))
  };
}
async function dispatch(env,input){
  const target=env.N8N_RECONCILER_WEBHOOK_URL;
  if(!target || !env.N8N_RECONCILER_TOKEN)
    return {status:"blocked",reason:"n8n reconciler webhook and credentials are not configured"};
  let endpoint;
  try{
    endpoint=new URL(target);
    if(endpoint.protocol!=="https:" || endpoint.username || endpoint.password ||
      endpoint.hash || endpoint.search)
      throw new Error("bad n8n endpoint");
  }catch{
    return {status:"blocked",reason:"n8n requires a fixed HTTPS endpoint"};
  }
  try{
    const r=await fetch(endpoint.toString(),{
      method:"POST",
      redirect:"error",
      headers:{"content-type":"application/json",
        "authorization":"Bearer "+env.N8N_RECONCILER_TOKEN},
      body:JSON.stringify({schema:"propria-knowledge-trigger/v1",...input}),
      signal:AbortSignal.timeout(55000)
    });
    if(!r.ok)throw new Error("n8n returned HTTP "+r.status);
    const body=await r.text();
    if(new TextEncoder().encode(body).length>MAX_REPLY_BYTES)
      throw new Error("n8n response too large");
    return verifiedN8nResult(JSON.parse(body));
  }catch(error){
    return {status:"blocked",reason:"n8n workflow did not complete: "+String(error.message).slice(0,120)};
  }
}
export default {
  async fetch(request,env) {
    const path=new URL(request.url).pathname;
    if(path==="/health"&&request.method==="GET")
      return response({ok:true,service:"propria-knowledge-trigger",
        n8n_configured:!!(env.N8N_RECONCILER_WEBHOOK_URL&&env.N8N_RECONCILER_TOKEN),
        workflow_owner:"n8n",source_mutations:0});
    if(!["/runs","/events"].includes(path))return response({error:"not found"},404);
    if(!(await validUser(request,env)))return response({error:"unauthorized"},401);
    if(request.method!=="POST")return response({error:"method not allowed"},405);
    let input;
    try{
      const body=await bodyInput(request);
      input=path==="/events"?eventInput(body):manualInput(body);
    }catch(e){return response({error:String(e.message).slice(0,140)},400);}
    const result=await dispatch(env,input);
    return response(result,result.status==="blocked"?503:200);
  },
  async scheduled(event,env,ctx){
    const task=dispatch(env,{kind:"scheduled",scope:"propria",
      scheduled_at:new Date(event.scheduledTime).toISOString()}).then(receipt=>{
      if(receipt.status==="blocked")console.error("Knowledge reconciling blocked");
      else console.log("Knowledge reconciliation",receipt.run_id,receipt.status);
    }).catch(()=>console.error("Knowledge reconciliation transport failure"));
    ctx.waitUntil(task);
  }
};
export const testHooks={eventInput,manualInput,verifiedN8nResult,dispatch};
