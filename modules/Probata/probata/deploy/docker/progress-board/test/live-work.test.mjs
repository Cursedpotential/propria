import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdir, mkdtemp, writeFile, symlink} from 'node:fs/promises';
import {resolve, join} from 'node:path';
import {createLiveWork} from '../live-work.mjs';

const SECRET = 'SYNTHETIC_SERVICE_TOKEN_0123456789abcdef';
const PRIVATE = 'PRIVATE_FIXTURE_PERSON_SOURCE_MATTER_REQUEST_BODY_ERROR';
const CLOCK = Date.parse('2026-10-11T20:00:00Z');
const CREATED = '2026-10-10T12:00:00.000Z';
const BASE = 'http://control.example/api/v1';
const prefix = 'http://control.example/project/p/environment/e/application/app1/deployment/';

/** Retain isolated synthetic credential fixtures under the test workspace.
 * Inputs: optional bytes. Outputs: private fixture path. Effects: creates new test files only.
 * Choose for adapter tests; fixtures contain no production credentials and are never deleted.
 */
async function tokenFile(bytes=SECRET) {
 const root = resolve('.preview-check'); await mkdir(root,{recursive:true});
 const dir = await mkdtemp(join(root,'live-work-fixture-'));
 const path = join(dir,'synthetic-token'); await writeFile(path,bytes,{mode:0o600,flag:'wx'}); return path;
}

/** Build a synthetic operation with deliberately private surplus fields.
 * Inputs: index and lifecycle. Outputs: upstream-shaped fixture. Effects: none.
 * Choose to verify explicit projection rather than accidental raw-row forwarding.
 */
function operation(index, lifecycle='running') {
 return {preview_handle:'synthetic_opaque_handle_01234567890123456789_' + index, service:'proffer', created_at:CREATED,
  lifecycle, current_stage:'execute_parser_activity', wait:'preview_decision', terminal:['completed','failed','cancelled','rerun_required'].includes(lifecycle),
  source_ref:PRIVATE, matter_id:PRIVATE, request_id:PRIVATE, reason:PRIVATE, body:PRIVATE, person:PRIVATE};
}

/** Build server-side provider configuration with synthetic credentials and safe static links.
 * Inputs: credential path and optional app configurations. Outputs: provider config. Effects: none.
 * Choose for isolated GET-only tests; no environment or production credentials are consulted.
 */
function providers(token, apps=[{uuid:'app1',deploymentUrlPrefix:prefix}]) {
 return {proffer:{baseUrl:'http://starter.example',tokenFile:token,activityUrl:'https://workbench.example/activity'},
  coolify:{baseUrl:BASE,tokenFile:token,apps,pageSize:10}};
}

/** Return a JSON HTTP response for a synthetic provider request.
 * Inputs: object and optional status. Outputs: standard streaming Response. Effects: none.
 * Choose to exercise the same streamed reader as production fetch responses.
 */
function json(body,status=200) { return new Response(JSON.stringify(body),{status,headers:{'content-type':'application/json'}}); }

test('unconfigured providers never masquerade as a successful empty feed',async()=>{
 const adapter=createLiveWork({fetcher:async()=>{throw Error('must not fetch');},now:()=>CLOCK});
 const result=await adapter.read();assert.equal(result.state,'unconfigured');assert.equal(result.items.length,0);
 assert.deepEqual(result.providers.map(p=>p.state),['unconfigured','unconfigured']);assert.equal(result.observed_at,new Date(CLOCK).toISOString());
});

test('durable lifecycle projection redacts private fields and preserves unknown clocks/stages',async()=>{
 const token=await tokenFile();const config=providers(token);
 const states=['running','awaiting_preview_decision','awaiting_repair_decision','completed','failed','cancelled','unavailable','unexpected_private_status','rerun_required'];
 const rows=states.map((state,i)=>operation(i,state));rows[0].current_stage=PRIVATE;rows[0].wait=PRIVATE;
 rows[0].started_at='not-a-clock';rows[0].updated_at='2026-10-10T12:00:00';
 let calls=0;
 const result=await createLiveWork({providers:{proffer:config.proffer},now:()=>CLOCK,fetcher:async(url,options)=>{
  calls++;assert.equal(url.pathname,'/reference-import/operations');assert.equal(url.search,'?limit=100');
  assert.deepEqual(options.headers,{Accept:'application/json',Authorization:'Bearer '+SECRET});assert.equal(options.method,'GET');assert.equal(options.redirect,'manual');assert.ok(options.signal instanceof AbortSignal);
  return json({items:rows,next_cursor:'PRIVATE_CURSOR'});
 }}).read();
 assert.equal(calls,1);assert.equal(result.state,'partial');assert.equal(result.providers[0].state,'ready');assert.equal(result.providers[0].truncated,true);
 assert.equal(result.counts.running,1);assert.equal(result.counts.waiting,2);assert.equal(result.counts.completed,1);assert.equal(result.counts.failed,1);assert.equal(result.counts.cancelled,1);assert.equal(result.counts.unknown,2);assert.equal(result.counts.rerun_required,1);
 assert.equal(result.items[0].created_at,CREATED);assert.equal(result.items[0].started_at,null);assert.equal(result.items[0].updated_at,null);assert.equal(result.items[0].stage,null);assert.equal(result.items[0].wait,null);
 assert.equal(result.items[0].url,'https://workbench.example/activity');assert.match(result.items[0].id,/^[a-f0-9]{64}$/);
 const serialized=JSON.stringify(result);for(const forbidden of [PRIVATE,SECRET,'PRIVATE_CURSOR',rows[0].preview_handle,'unexpected_private_status','source_ref','matter_id','request_id','reason','body','person'])assert.ok(!serialized.includes(forbidden),forbidden);
 assert.match(result.scope,/Latest bounded provider pages only/);
});

test('Coolify wrapper maps actual statuses and only verified same-origin exact deployment links',async()=>{
 const token=await tokenFile();const config=providers(token);
 const statuses=['queued','in_progress','finished','failed','cancelled','unrecognized-private-status'];
 const rows=statuses.map((status,i)=>({deployment_uuid:'deployment'+i,status,created_at:CREATED,finished_at:i===2?'2026-10-11T01:00:00Z':null,logs:PRIVATE,commit_message:PRIVATE,application_name:PRIVATE,request_id:PRIVATE}));
 rows[0].deployment_url=prefix+'deployment0';rows[1].deployment_url='https://evil.example/'+PRIVATE;
 const result=await createLiveWork({providers:{coolify:config.coolify},now:()=>CLOCK,fetcher:async url=>{
  assert.equal(url.origin,'http://control.example');assert.equal(url.pathname,'/api/v1/deployments/applications/app1');assert.equal(url.search,'?skip=0&take=10');
  return json({count:20,deployments:rows});
 }}).read();
 assert.deepEqual(result.items.map(item=>item.state),['queued','running','completed','failed','cancelled','unknown']);
 assert.equal(result.providers[1].truncated,true);assert.equal(result.items[0].url,prefix+'deployment0');assert.equal(result.items[1].url,prefix+'deployment1');assert.equal(result.items[2].finished_at,'2026-10-11T01:00:00.000Z');assert.equal(result.items[2].updated_at,null);
 assert.ok(!JSON.stringify(result).includes(PRIVATE));assert.ok(!JSON.stringify(result).includes('unrecognized-private-status'));
 const noLink=await createLiveWork({providers:{coolify:{...config.coolify,apps:[{uuid:'app1'}]}},fetcher:async()=>json([{...rows[0],deployment_url:'https://evil.example/deployment0'}])}).read();
 assert.equal(noLink.items[0].url,null);
});

test('401/403, failures and malformed HTTP200 are explicit provider failures with no leaked errors',async()=>{
 const token=await tokenFile();const config=providers(token);
 for(const [status,state,code] of [[401,'auth_required','authentication_required'],[403,'auth_required','access_denied'],[500,'unavailable','http_error']]){
  let canceled=false;
  const result=await createLiveWork({providers:{proffer:config.proffer},fetcher:async()=>new Response(new ReadableStream({cancel(){canceled=true;}}),{status})}).read();
  assert.equal(result.providers[0].state,state);assert.equal(result.providers[0].code,code);assert.equal(result.items.length,0);assert.ok(canceled);
 }
 for(const fetcher of [async()=>{throw Error(PRIVATE);},async()=>json({error:PRIVATE}),async()=>new Response(PRIVATE),async()=>json({items:[{service:'proffer',preview_handle:'bad/id'}]})]){
  const result=await createLiveWork({providers:{proffer:config.proffer},fetcher}).read();assert.equal(result.providers[0].state,'unavailable');assert.equal(result.items.length,0);assert.ok(!JSON.stringify(result).includes(PRIVATE));
 }
 const mixed=await createLiveWork({providers:{coolify:{...config.coolify,apps:[{uuid:'app1'},{uuid:'app2'}]}},fetcher:async url=>url.pathname.endsWith('app1')?json({count:1,deployments:[{deployment_uuid:'d1',status:'finished'}]}):json({},403)}).read();
 assert.equal(mixed.providers[1].state,'partial');assert.equal(mixed.providers[1].checks[1].state,'auth_required');assert.equal(mixed.counts.completed,1);assert.equal(mixed.providers[1].count,1);
});

test('streamed and declared response byte caps cancel input without parsing or forwarding payload',async()=>{
 const token=await tokenFile();const config=providers(token);
 for(const declared of [true,false]){
  let canceled=false;let pulls=0;
  const stream=new ReadableStream({pull(controller){pulls++;controller.enqueue(new Uint8Array(256*1024));},cancel(){canceled=true;}});
  const result=await createLiveWork({providers:{proffer:config.proffer},fetcher:async()=>new Response(stream,{headers:declared?{'content-length':String(1024*1024+1)}:{}})}).read();
  assert.equal(result.providers[0].state,'unavailable');assert.equal(result.providers[0].code,'response_too_large');assert.equal(result.items.length,0);assert.ok(canceled);assert.ok(pulls<=6);
 }
});

test('operations, per-app pages, allowlist and request concurrency remain bounded',async()=>{
 const token=await tokenFile();const config=providers(token,Array.from({length:8},(_,i)=>({uuid:'app'+i})));
 let active=0,maxActive=0,calls=0;
 const result=await createLiveWork({providers:config,fetcher:async url=>{
  calls++;active++;maxActive=Math.max(maxActive,active);await new Promise(resolve=>setTimeout(resolve,3));active--;
  return url.hostname==='starter.example'?json({items:Array.from({length:110},(_,i)=>operation(i))}):json({count:30,deployments:Array.from({length:12},(_,i)=>({deployment_uuid:'d'+i,status:'in_progress'}))});
 }}).read();
 assert.equal(calls,9);assert.ok(maxActive<=4);assert.equal(result.items.length,180);assert.equal(result.counts.total,180);assert.equal(result.providers[0].truncated,true);assert.equal(result.providers[1].count,80);
 const invalid=await createLiveWork({providers:{coolify:{...config.coolify,apps:Array.from({length:9},(_,i)=>({uuid:'app'+i}))}},fetcher:async()=>{throw Error('must not fetch');}}).read();assert.equal(invalid.providers[1].state,'unconfigured');
 let defaultTake=null;await createLiveWork({providers:{coolify:{...config.coolify,apps:[{uuid:'app1'}],pageSize:undefined}},fetcher:async url=>{defaultTake=url.searchParams.get('take');return json({count:0,deployments:[]});}}).read();assert.equal(defaultTake,'1');
});

test('20-second cache and single-flight prevent overlapping refreshes and protect cached output',async()=>{
 const token=await tokenFile();const config=providers(token);let clock=CLOCK,calls=0,release;
 const adapter=createLiveWork({providers:{proffer:config.proffer},now:()=>clock,fetcher:async()=>{calls++;await new Promise(resolve=>{release=resolve;});return json({items:[operation(1)]});}});
 const reads=Array.from({length:25},()=>adapter.read());
 while(!release)await new Promise(resolve=>setTimeout(resolve,1));assert.equal(calls,1);release();const first=await Promise.all(reads);
 first[0].items[0].title=PRIVATE;assert.equal((await adapter.read()).items[0].title,'Proffer operation');assert.equal(calls,1);
 clock+=19999;await adapter.read();assert.equal(calls,1);clock++;
 release=null;const next=[adapter.read(),adapter.read()];while(!release)await new Promise(resolve=>setTimeout(resolve,1));assert.equal(calls,2);
 clock+=40000;next.push(adapter.read());assert.equal(calls,2);release();await Promise.all(next);assert.equal(calls,2);
});

test('credentials are bounded regular files and unsafe Activity/deployment links remain absent',async()=>{
 for(const value of ['short',Buffer.alloc(4099,65),'A'.repeat(32)+'\n'+PRIVATE]){
  const token=await tokenFile(value);const result=await createLiveWork({providers:{proffer:providers(token).proffer},fetcher:async()=>{throw Error('must not fetch');}}).read();assert.equal(result.providers[0].code,'credential_unavailable');
 }
 const token=await tokenFile();const link=token+'-symlink';await symlink(token,link);
 const rejected=await createLiveWork({providers:{proffer:providers(link).proffer},fetcher:async()=>{throw Error('must not fetch');}}).read();assert.equal(rejected.providers[0].code,'credential_unavailable');
 for(const activityUrl of ['https://workbench.example/activity?matter='+PRIVATE,'https://user:password@workbench.example/activity','https://workbench.example/intake']){
  const result=await createLiveWork({providers:{proffer:{...providers(token).proffer,activityUrl}},fetcher:async()=>json({items:[operation(1)]})}).read();assert.equal(result.items[0].url,null);assert.ok(!JSON.stringify(result).includes(PRIVATE));
 }
});

test('request deadline aborts a stalled provider without overlapping or leaking its error',{timeout:15000},async()=>{
 const token=await tokenFile();const started=Date.now();let aborted=false;
 const keepAlive=setTimeout(()=>{},14000);
 const result=await createLiveWork({providers:{proffer:providers(token).proffer},fetcher:async(_url,{signal})=>new Promise((_resolve,reject)=>signal.addEventListener('abort',()=>{aborted=true;reject(Error(PRIVATE));},{once:true}))}).read();
 clearTimeout(keepAlive);
 assert.ok(aborted);assert.ok(Date.now()-started>=9000);assert.ok(Date.now()-started<14000);assert.equal(result.providers[0].state,'unavailable');assert.equal(result.providers[0].code,'request_failed');assert.ok(!JSON.stringify(result).includes(PRIVATE));
});
