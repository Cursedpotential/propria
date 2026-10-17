import test from "node:test";
import assert from "node:assert/strict";
import worker, {testHooks} from "./index.js";

const TOKEN="test-trigger-secret";
const N8N="https://n8n.example.test/webhook/propria/knowledge/reconcile";
function env(){
  return {RECONCILER_TRIGGER_TOKEN:TOKEN,
    N8N_RECONCILER_WEBHOOK_URL:N8N,
    N8N_RECONCILER_TOKEN:"test-n8n-secret"};
}
function request(path,body,token=TOKEN){
  return new Request("https://edge.example.test"+path,{
    method:"POST",
    headers:{"authorization":"Bearer "+token,"content-type":"application/json"},
    body:JSON.stringify(body)
  });
}
function complete(overrides={}){
  return {
    schema:"propria-knowledge-workflow-result/v1",
    run_id:"n8n-knowledge-123",
    status:"review_pending",
    scanned:9,findings:1,
    coverage:[{store:"memsearch",state:"queried",reason:""}],
    review_queue:[{review_ref:"review-123",status:"queued",severity:"high",
      kind:"possible_topic_conflict",newly_created:true}],
    alerts:[{review_ref:"review-123",status:"sent"}],
    ...overrides
  };
}
async function withFetch(fake,action){
  const old=globalThis.fetch;
  globalThis.fetch=fake;
  try{return await action();}finally{globalThis.fetch=old;}
}
test("health is minimal; n8n owns complete workflow",async()=>{
  const result=await (await worker.fetch(new Request("https://edge.example.test/health"),env())).json();
  assert.equal(result.workflow_owner,"n8n");
  assert.equal(result.source_mutations,0);
});
test("trigger requires auth, Propria scope and reference-only payload",async()=>{
  await withFetch(async()=>{throw Error("should not call network");},async()=>{
    const payload={type:"memory.changed",source:"memsearch",record_ref:"abc",
      revision:"v2",scope:"propria"};
    assert.equal((await worker.fetch(request("/events",payload,"bad-token"),env())).status,401);
    assert.equal((await worker.fetch(request("/events",{...payload,content:"private"},TOKEN),env())).status,400);
    assert.equal((await worker.fetch(request("/runs",{scope:"custody-case"},TOKEN),env())).status,400);
  });
});
test("one n8n call returns verified persisted review identity, no separate notification call",async()=>{
  const requests=[];
  await withFetch(async(url,params)=>{
    requests.push({url:String(url),payload:JSON.parse(params.body)});
    return Response.json(complete());
  },async()=>{
    const result=await worker.fetch(request("/events",{
      type:"document.changed",source:"docstore",record_ref:"doc-abc",
      revision:"3",scope:"propria"
    }),env());
    const json=await result.json();
    assert.equal(result.status,200);
    assert.equal(json.status,"review_pending");
    assert.equal(json.review_queue[0].review_ref,"review-123");
    assert.equal(json.alerts[0].status,"sent");
    assert.equal(requests.length,1);
    assert.equal(requests[0].url,N8N);
    assert.equal(requests[0].payload.kind,"event");
    assert.equal(requests[0].payload.record_ref,"doc-abc");
    assert.ok(!JSON.stringify(requests[0]).includes("private"));
  });
});
test("invalid n8n receipts cannot claim completion",async()=>{
  await withFetch(async()=>Response.json({ok:true,run_id:"fake",status:"completed"}),async()=>{
    const r=await worker.fetch(request("/runs",{scope:"propria"}),env());
    assert.equal(r.status,503);
    assert.equal((await r.json()).status,"blocked");
  });
});
test("missing n8n config fails closed",async()=>{
  const e=env();delete e.N8N_RECONCILER_WEBHOOK_URL;
  const r=await worker.fetch(request("/runs",{scope:"propria"}),e);
  assert.equal(r.status,503);
  assert.equal((await r.json()).status,"blocked");
});
test("per-store unavailability is surfaced, not declared a full scan",async()=>{
  const verified=testHooks.verifiedN8nResult(complete({
    status:"partial",
    coverage:[{store:"memsearch",state:"queried"},{store:"engram",state:"unavailable",reason:"no adapter"}],
    alerts:[]
  }));
  assert.equal(verified.status,"partial");
  assert.equal(verified.coverage[1].state,"unavailable");
});
test("daily Cron invokes n8n exactly once with a reference-only scope",async()=>{
  const tasks=[];const calls=[];
  await withFetch(async(url,options)=>{
    calls.push(JSON.parse(options.body));
    return Response.json(complete({status:"completed",review_queue:[],alerts:[]}));
  },async()=>{
    await worker.scheduled({scheduledTime:Date.UTC(2026,9,17,12,0,0)},env(),
      {waitUntil(p){tasks.push(p);}});
    await Promise.all(tasks);
  });
  assert.equal(calls.length,1);
  assert.equal(calls[0].kind,"scheduled");
  assert.equal(calls[0].scope,"propria");
});
