/** Offline contract tests for the ONE complete n8n reconciliation workflow.
 * Inputs: importable JSON export and synthetic memory documents.
 * Outputs: verified graph integrity and deterministic candidate/receipt generation.
 * Effects: none; no n8n process, database, memory or alert is contacted.
 */
import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";

const workflow=JSON.parse(readFileSync(new URL("./wf-knowledge-reconcile.json",import.meta.url),"utf-8"));
const nodes=new Map(workflow.nodes.map(n=>[n.name,n]));
function run(name,json,results={},id="synthetic-123"){
 const code=nodes.get(name).parameters.jsCode;
 const fn=new Function("$input","$","$execution",code);
 const input={first:()=>({json})};
 const select=name=>({first:()=>({json:results[name]})});
 return fn(input,select,{id})[0].json;
}

test("one workflow validates all nodes, edges, and JavaScript syntax",()=>{
 assert.equal(workflow.active,false);
 assert.ok(workflow.nodes.length>=9);
 assert.equal(nodes.get("Webhook - Reconcile").parameters.authentication,"headerAuth");
 assert.equal(workflow.settings.saveDataSuccessExecution,"none");
 assert.equal(workflow.settings.saveDataErrorExecution,"none");
 for(const [name,links] of Object.entries(workflow.connections)){
   assert.ok(nodes.has(name));
   for(const lane of links.main)for(const edge of lane)assert.ok(nodes.has(edge.node));
 }
 for(const node of workflow.nodes.filter(n=>n.type==="n8n-nodes-base.code")){
   assert.doesNotThrow(()=>new Function("$input","$","$execution",node.parameters.jsCode));
 }
});

test("the record event carries only authoritative references",()=>{
 const trigger=run("Validate Trigger",{body:{
   schema:"propria-knowledge-trigger/v1",kind:"event",
   scope:"propria",source:"memsearch",type:"memory.changed",
   record_ref:"note-a",revision:"2"
 }});
 assert.equal(trigger.kind,"event");
 assert.equal(trigger.run_id,"n8n-knowledge-synthetic-123");
 assert.equal(trigger.record_ref,"note-a");
 assert.throws(()=>run("Validate Trigger",{body:{
   schema:"propria-knowledge-trigger/v1",kind:"manual",scope:"propria",
   content:"leaked evidence"
 }}));
});

test("conflicting active records produce candidate not an automatic correction",()=>{
 const trigger=run("Validate Trigger",{body:{
   schema:"propria-knowledge-trigger/v1",kind:"manual",scope:"propria"
 }});
 const snapshot={schema:"propria-knowledge-snapshot/v1",
   coverage:[{store:"memsearch",state:"queried"},{store:"engram",state:"queried"}],
   records:[
     {store:"memsearch",id:"note-a",revision:"1",scope:"propria",
       subject_key:"storage",text:"Keep CCC local",state:"active"},
     {store:"engram",id:"conv-b",revision:"4",scope:"propria",
       subject_key:"storage",text:"Move CCC indexes to a hosted server",state:"active"}
   ]};
 const review=run("Build Review Candidates",snapshot,{"Validate Trigger":trigger});
 assert.equal(review.findings,1);
 assert.equal(review.review_candidates[0].kind,"possible_topic_conflict");
 assert.equal(review.review_candidates[0].automatic,false);
 assert.equal(review.review_candidates[0].proposed_action,"review");
 assert.deepEqual(review.review_candidates[0].source_refs,["engram:conv-b","memsearch:note-a"]);
 const persisted={schema:"propria-knowledge-review-write/v1",run_id:trigger.run_id,
   review_queue:[{review_ref:"persisted-pg-case-1",status:"queued",severity:"high",
     kind:"possible_topic_conflict",newly_created:true}]};
 const output=run("Verify Persisted Review Receipt",persisted,
   {"Build Review Candidates":review});
 assert.equal(output.status,"review_pending");
 assert.equal(output.alert_candidates[0].review_ref,"persisted-pg-case-1");
 assert.equal(output.review_queue[0].newly_created,true);
});

test("exact same claims are low severity and do not trigger high-alert branch",()=>{
 const trigger=run("Validate Trigger",{body:{
   schema:"propria-knowledge-trigger/v1",kind:"manual",scope:"propria"
 }});
 const review=run("Build Review Candidates",{
   schema:"propria-knowledge-snapshot/v1",
   coverage:[{store:"memsearch",state:"queried"}],
   records:[
     {store:"memsearch",id:"a",revision:"1",scope:"propria",subject_key:"subject-A",text:"Important fact"},
     {store:"memsearch",id:"b",revision:"1",scope:"propria",subject_key:"subject-A",text:"Important fact"}
   ]
 },{"Validate Trigger":trigger});
 assert.equal(review.review_candidates[0].kind,"duplicate_active_claim");
 assert.equal(review.review_candidates[0].severity,"low");
});

test("old review fingerprints do not result in new alerts",()=>{
 const trigger=run("Validate Trigger",{body:{
   schema:"propria-knowledge-trigger/v1",kind:"scheduled",scope:"propria"
 }});
 const review={schema:"propria-knowledge-review-proposals/v1",run_id:trigger.run_id,
   scanned:5,findings:1,coverage:[{store:"memsearch",state:"queried"}]};
 const output=run("Verify Persisted Review Receipt",{
   schema:"propria-knowledge-review-write/v1",run_id:trigger.run_id,
   review_queue:[{review_ref:"already-queued",status:"queued",severity:"critical",
     kind:"possible_topic_conflict",newly_created:false}]
 },{"Build Review Candidates":review});
 assert.equal(output.alert_candidates.length,0);
});
