import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdir,mkdtemp,writeFile} from 'node:fs/promises';
import {resolve,join} from 'node:path';
import {createLiveWork} from '../live-work.mjs';
import {liveWorkProvidersFromEnv,liveTodoFeed} from '../server.mjs';

const PRIVATE='PRIVATE_PERSON_SOURCE_TYPE_MEMO_TOKEN_CURSOR';
const CLOCK=Date.parse('2026-10-11T20:00:00Z');
const clusters=[{name:'original',baseUrl:'http://original.example:8233',namespace:'default',uiUrl:'http://original.example:8233/'},{name:'trial',baseUrl:'http://trial.example:8233',uiUrl:'http://trial.example:8233/'}];
/** Build an upstream-shaped visibility fixture with deliberately private extra metadata.
 * Inputs: ordinal/type/status. Outputs: native API row. Effects: none.
 * Choose for projection and native clock checks without real workflow data.
 */
function row(i=1,type='ResumeHashLedgerCopy',status='RUNNING') {
 return {execution:{workflowId:PRIVATE+i,runId:'synthetic-run-'+i},type:{name:type},status:'WORKFLOW_EXECUTION_STATUS_'+status,startTime:'2026-10-08T03:41:13.229523Z',closeTime:null,memo:{fields:{private:PRIVATE}},searchAttributes:{private:PRIVATE},input:PRIVATE};
}
/** Return a synthetic streamed API response for bounded-reader tests.
 * Inputs: JSON value and status. Outputs: Response. Effects: none.
 * Choose to exercise production response parsing without live requests.
 */
function json(value,status=200){return new Response(JSON.stringify(value),{status});}
/** Create a retained synthetic credential fixture for global mixed-provider tests.
 * Inputs: none. Outputs: synthetic token file path. Effects: creates private test files only.
 * Choose for existing authenticated providers; Temporal must never receive this token.
 */
async function tokenFile(){await mkdir(resolve('.preview-check'),{recursive:true});const dir=await mkdtemp(resolve('.preview-check/temporal-fixture-'));const file=join(dir,'token');await writeFile(file,'SYNTHETIC_TOKEN_012345678901234567890',{flag:'wx',mode:0o600});return file;}

test('Temporal native visibility shape is sanitized, linked to independent clusters and counted by actual status',async()=>{
 let calls=0;
 const result=await createLiveWork({providers:{temporal:{clusters}},now:()=>CLOCK,fetcher:async(url,options)=>{
  calls++;assert.equal(url.pathname,'/api/v1/namespaces/default/workflows');assert.equal(url.searchParams.get('query'),'ExecutionStatus="Running"');assert.equal(url.searchParams.get('pageSize'),'100');assert.equal(url.searchParams.has('nextPageToken'),false);assert.deepEqual(options.headers,{Accept:'application/json'});assert.equal(options.method,'GET');
  return url.hostname==='original.example'?json({executions:[row(),row(2,PRIVATE),{...row(3,'ProfferWorkflow','COMPLETED'),closeTime:'2026-10-11T19:00:00Z'}],nextPageToken:PRIVATE}):json({executions:[{...row(4,'AprilGooglePayloads'),startTime:'invalid-clock'},row(5,'PDFTakeoutBackups','PRIVATE_STATUS')]});
 }}).read();
 assert.equal(calls,2);assert.equal(result.counts.running,3);assert.equal(result.counts.completed,1);assert.equal(result.counts.unknown,1);assert.equal(result.providers[2].truncated,true);assert.equal(result.providers[3].count,2);
 assert.deepEqual(result.items.map(i=>i.title),['Storage preservation copy','Background workflow','Proffer workflow','Google export processing','Google export processing']);
 assert.equal(result.items[0].started_at,'2026-10-08T03:41:13.229Z');assert.equal(result.items[0].created_at,null);assert.equal(result.items[0].updated_at,null);assert.equal(result.items[2].finished_at,'2026-10-11T19:00:00.000Z');assert.equal(result.items[3].started_at,null);assert.equal(result.items[4].terminal,null);
 assert.equal(result.items[0].url,clusters[0].uiUrl);assert.equal(result.items[3].url,clusters[1].uiUrl);assert.match(result.items[0].id,/^[a-f0-9]{64}$/);assert.equal(result.items[0].observed_at,new Date(CLOCK).toISOString());
 assert.ok(!JSON.stringify(result).includes(PRIVATE));assert.ok(!JSON.stringify(result).includes('ResumeHashLedgerCopy'));assert.match(result.scope,/default namespaces only/);assert.match(result.scope,/also appear as a Proffer operation/);
 const todo=liveTodoFeed(result);assert.ok(todo.items.some(i=>i.area==='Temporal original'));assert.ok(todo.items.some(i=>i.area==='Temporal trial'));assert.deepEqual(todo.counts,result.counts);
});

test('Temporal empty objects are genuine zero while malformed, auth and cursor errors stay explicit',async()=>{
 for(const [body,status,state,code] of [[{},200,'ready',null],[{executions:[]},200,'ready',null],[{error:PRIVATE},200,'unavailable','invalid_response'],[{results:[]},200,'unavailable','invalid_response'],[[],200,'unavailable','invalid_response'],[{executions:{}},200,'unavailable','invalid_response'],[{executions:[{}]},200,'unavailable','invalid_response'],[{nextPageToken:{}},200,'unavailable','invalid_response'],[{nextPageToken:'x'.repeat(8193)},200,'unavailable','invalid_response'],[{},401,'auth_required','authentication_required'],[{},403,'auth_required','access_denied'],[{},500,'unavailable','http_error']]) {
  const result=await createLiveWork({providers:{temporal:{clusters:[clusters[0]]}},fetcher:async()=>json(body,status)}).read();const p=result.providers[2];assert.equal(p.state,state);assert.equal(p.code,code);assert.equal(p.count,0);assert.ok(!JSON.stringify(result).includes(PRIVATE));
 }
 let calls=0;const page=await createLiveWork({providers:{temporal:{clusters:[clusters[0]]}},fetcher:async()=>{calls++;return json({executions:Array.from({length:105},(_,i)=>row(i)),nextPageToken:PRIVATE});}}).read();assert.equal(calls,1);assert.equal(page.items.length,100);assert.equal(page.providers[2].truncated,true);
});

test('Temporal declared and streamed byte caps cancel without forwarding content',async()=>{
 for(const declared of [true,false]) {let canceled=false,pulls=0;const result=await createLiveWork({providers:{temporal:{clusters:[clusters[0]]}},fetcher:async()=>new Response(new ReadableStream({pull(c){pulls++;c.enqueue(new Uint8Array(256*1024));},cancel(){canceled=true;}}),{headers:declared?{'content-length':String(1024*1024+1)}:{}})}).read();assert.equal(result.providers[2].code,'response_too_large');assert.equal(result.items.length,0);assert.ok(canceled);assert.ok(pulls<=6);}
});

test('optional configuration preserves two-provider compatibility and rejects unsafe cluster scopes',async()=>{
 assert.equal(liveWorkProvidersFromEnv({}).temporal,undefined);
 assert.deepEqual(liveWorkProvidersFromEnv({LIVE_WORK_TEMPORAL_CLUSTERS_JSON:JSON.stringify(clusters)}).temporal,{clusters});
 for(const value of ['invalid','x'.repeat(16385),JSON.stringify([...clusters,clusters[0]])]) assert.deepEqual(liveWorkProvidersFromEnv({LIVE_WORK_TEMPORAL_CLUSTERS_JSON:value}).temporal,{clusters:[]});
 for(const value of [[],[clusters[0],clusters[0]],[{...clusters[0],name:PRIVATE}],[{...clusters[0],namespace:'temporal-system'}],[{...clusters[0],baseUrl:'http://user:secret@original.example'}],[{...clusters[0],uiUrl:'http://original.example/?workflow='+PRIVATE}]]) {const result=await createLiveWork({providers:{temporal:{clusters:value}},fetcher:async()=>{throw Error('must not fetch');}}).read();assert.equal(result.providers[2].state,'unconfigured');assert.ok(!JSON.stringify(result).includes(PRIVATE));}
 const old=await createLiveWork({fetcher:async()=>{throw Error('must not fetch');}}).read();assert.equal(old.providers.length,2);
});

test('all providers share four slots through full body reads with cached single-flight refreshes',async()=>{
 const token=await tokenFile();let clock=CLOCK,calls=0,active=0,maxActive=0;
 const config={proffer:{baseUrl:'http://starter.example',tokenFile:token},coolify:{baseUrl:'http://coolify.example/api/v1',tokenFile:token,apps:Array.from({length:8},(_,i)=>({uuid:'app'+i}))},temporal:{clusters}};
 const adapter=createLiveWork({providers:config,now:()=>clock,fetcher:async(url,options)=>{
  calls++;active++;maxActive=Math.max(active,maxActive);if(url.port==='8233')assert.equal(options.headers.Authorization,undefined);else assert.ok(options.headers.Authorization);
  const body=url.port==='8233'?{executions:[row()]}:url.hostname==='starter.example'?{items:[]}:{count:0,deployments:[]};
  let emitted=false;return new Response(new ReadableStream({async pull(controller){if(emitted)return;emitted=true;await new Promise(resolve=>setTimeout(resolve,8));controller.enqueue(new TextEncoder().encode(JSON.stringify(body)));controller.close();active--;}}));
 }});
 const snapshots=await Promise.all(Array.from({length:20},()=>adapter.read()));assert.equal(calls,11);assert.ok(maxActive<=4);assert.ok(maxActive>=2);assert.equal(active,0);assert.equal(snapshots[0].counts.running,2);
 snapshots[0].items[0].title=PRIVATE;assert.ok(!(await adapter.read()).items.some(i=>i.title===PRIVATE));clock+=19999;await adapter.read();assert.equal(calls,11);clock++;await Promise.all([adapter.read(),adapter.read()]);assert.equal(calls,22);assert.ok(maxActive<=4);
});
