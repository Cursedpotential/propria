import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp,writeFile,utimes,stat,readFile} from 'node:fs/promises';
import vm from 'node:vm';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {createServer,liveWorkProvidersFromEnv,liveTodoFeed,readTodoFeed} from '../server.mjs';
import {liveWorkCard,storageHealthCard,taskCard} from '../public/view.mjs';

/** Run a synthetic board server without reaching real providers or changing services.
 * Inputs: server options and assertion callback. Outputs: assertion results.
 * Effects: temporary loopback listener only; choose for HTTP route contract checks.
 */
async function withServer(options,check) {
 const server=createServer({config:{},...options});
 await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
 try {await check('http://127.0.0.1:'+server.address().port);} finally {await new Promise(resolve=>server.close(resolve));}
}

test('live routes authenticate upstream, share adapter caches, project real states and redact private fields',async()=>{
 const scratch=await mkdtemp(join(tmpdir(),'portal-live-integration-'));
 const profferToken='synthetic-proffer-token-0000000000000000',coolifyToken='synthetic-coolify-token';
 const profferFile=join(scratch,'proffer-token'),coolifyFile=join(scratch,'coolify-token');
 await writeFile(profferFile,profferToken);await writeFile(coolifyFile,coolifyToken);
 const counts={proffer:0,coolify:0};
 const providers={proffer:{baseUrl:'http://starter.invalid:8091',tokenFile:profferFile,activityUrl:'https://workbench.invalid/activity'},coolify:{baseUrl:'https://control.invalid/api/v1',tokenFile:coolifyFile,apps:[{uuid:'synthetic-app'}]}};
 const fetcher=async(url,options)=>{
  assert.equal(options.method,'GET');assert.equal(options.redirect,'manual');assert.ok(options.signal instanceof AbortSignal);
  if(url.hostname==='starter.invalid') {
   counts.proffer++;assert.equal(options.headers.Authorization,'Bearer '+profferToken);assert.equal(url.search,'?limit=100');
   return Response.json({items:[{service:'proffer',preview_handle:'a'.repeat(48),lifecycle:'awaiting_preview_decision',current_stage:'publish_preview_activity',wait:'preview_decision',terminal:false,started_at:'2026-10-11T17:00:00Z',source_ref:'PRIVATE-SOURCE',matter_id:'PRIVATE-MATTER',reason:'PRIVATE-ERROR'},{service:'proffer',preview_handle:'b'.repeat(48),lifecycle:'completed',terminal:true},{service:'proffer',preview_handle:'c'.repeat(48),lifecycle:'failed',terminal:true}],next_cursor:'private-cursor'});
  }
  counts.coolify++;assert.equal(options.headers.Authorization,'Bearer '+coolifyToken);assert.equal(url.pathname,'/api/v1/deployments/applications/synthetic-app');assert.equal(url.search,'?skip=0&take=1');
  return new Response('private denial body',{status:403});
 };
 await withServer({fetcher,liveWorkProviders:providers},async base=>{
  const responses=await Promise.all([fetch(base+'/api/live-work'),fetch(base+'/progress/api/todo-feed')]);
  assert.ok(responses.every(response=>response.status===200));
  const [live,todo]=await Promise.all(responses.map(response=>response.json()));
  assert.equal(live.state,'partial');assert.equal(live.counts.waiting,1);assert.equal(live.counts.completed,1);assert.equal(live.counts.failed,1);
  assert.equal(live.providers.find(row=>row.provider==='coolify').state,'auth_required');
  assert.equal(todo.status,'partial');assert.equal(todo.item_count,3);assert.equal(todo.items[0].area,'Proffer');assert.match(todo.items[0].summary,/waiting.*preview decision/);
  assert.equal(todo.items[0].started_at,'2026-10-11T17:00:00.000Z');assert.equal(todo.items[1].started_at,null);assert.equal(todo.generated_at,live.observed_at);
  assert.deepEqual(counts,{proffer:1,coolify:1});
  for(const value of [live,todo])assert.doesNotMatch(JSON.stringify(value),/PRIVATE-|private denial|private-cursor|synthetic-proffer-token|synthetic-coolify-token|proffer-token|coolify-token|starter.invalid/);
  const forbidden=await fetch(base+'/api/live-work',{method:'POST'});assert.equal(forbidden.status,405);assert.equal(forbidden.headers.get('allow'),'GET');assert.deepEqual(counts,{proffer:1,coolify:1});
 });
});

test('storage route uses bounded fixed probes and returns metadata-only connection results',async()=>{
 const previous=process.env.OPENLIST_TOKEN;process.env.OPENLIST_TOKEN='synthetic-openlist-token';
 let calls=0;
 try {
  await withServer({fetcher:async(url,options)=>{calls++;assert.equal(options.headers.authorization,'synthetic-openlist-token');assert.equal(options.method,'POST');const body=JSON.parse(options.body);assert.equal(body.page,1);assert.equal(body.per_page,1);assert.equal(body.refresh,true);return Response.json({code:200,data:{content:[{name:'PRIVATE-FILENAME',sign:'PRIVATE-SIGN'}],total:11}});}},async base=>{
   const [a,b]=await Promise.all([fetch(base+'/api/storage-health'),fetch(base+'/progress/api/storage-health')]);
   const value=await a.json();assert.equal(value.state,'ready');assert.equal(value.profiles.length,4);assert.ok(value.profiles.every(row=>row.authenticated_list_success&&row.returned_entry_count===1));assert.equal(calls,4);
   assert.deepEqual(await b.json(),value);assert.doesNotMatch(JSON.stringify(value),/PRIVATE-|synthetic-openlist-token|\/b2\/|\/r2\/|https?:/);
  });
 } finally {if(previous===undefined)delete process.env.OPENLIST_TOKEN;else process.env.OPENLIST_TOKEN=previous;}
});

test('unconfigured providers remain explicit and ToDo never falls back to manual tasks',async()=>{
 const scratch=await mkdtemp(join(tmpdir(),'portal-backlog-integration-')),file=join(scratch,'backlog.md');
 await writeFile(file,'## 2026-01-01\n- [ ] Synthetic documented task\n');
 await utimes(file,new Date('2026-01-02T03:04:05Z'),new Date('2026-01-02T03:04:05Z'));
 const mtime=(await stat(file)).mtime.toISOString();
 await withServer({todoPath:file,liveWorkProviders:{}},async base=>{
  const todo=await(await fetch(base+'/api/todo-feed')).json();assert.equal(todo.status,'unconfigured');assert.equal(todo.item_count,0);assert.ok(todo.providers.every(row=>row.state==='unconfigured'));assert.doesNotMatch(JSON.stringify(todo),/Synthetic documented task/);
  const backlog=await(await fetch(base+'/api/manual-backlog')).json();assert.equal(backlog.scope,'manual backlog');assert.equal(backlog.generated_at,mtime);assert.equal(backlog.source_updated_at,mtime);assert.equal(backlog.items[0].title,'Synthetic documented task');assert.equal(backlog.item_count,1);
 });
 const missing=await readTodoFeed(join(scratch,'absent.md'));assert.equal(missing.generated_at,null);assert.equal(missing.source_updated_at,null);assert.equal(missing.scope,'manual backlog');
 await writeFile(join(scratch,'large.md'),'x'.repeat(1024*1024+1));assert.equal((await readTodoFeed(join(scratch,'large.md'))).status,'unavailable');
});

test('configuration is server-only and malformed/oversized app allowlists stay unconfigured',()=>{
 const config=liveWorkProvidersFromEnv({PROFFER_SERVICE_TOKEN_FILE:'/run/secrets/proffer',LIVE_WORK_COOLIFY_API_URL:'https://control.invalid/api/v1',LIVE_WORK_COOLIFY_TOKEN_FILE:'/run/secrets/control',LIVE_WORK_COOLIFY_APPS_JSON:'[{"uuid":"fixed-app"}]'});
 assert.equal(config.proffer.baseUrl,'http://100.91.190.107:8091');assert.equal(config.proffer.tokenFile,'/run/secrets/proffer');assert.equal(config.coolify.pageSize,1);assert.equal(config.coolify.apps[0].uuid,'fixed-app');
 for(const value of ['invalid','x'.repeat(16385),JSON.stringify(Array(9).fill({uuid:'app'}))])assert.deepEqual(liveWorkProvidersFromEnv({LIVE_WORK_COOLIFY_APPS_JSON:value}).coolify.apps,[]);
});

test('renderers keep observed work, documented backlog, storage auth and network failures separate',()=>{
 const snapshot={observed_at:'2026-10-11T17:00:00Z',state:'partial',scope:'Bounded provider pages',counts:{running:1,waiting:1,failed:1,total:3},providers:[{provider:'proffer',state:'ready',count:3,truncated:true},{provider:'coolify',state:'auth_required',count:0}],items:[{title:'Proffer operation',provider:'proffer',state:'waiting',wait:'preview_decision',url:'https://workbench.invalid/activity'}]};
 assert.match(liveWorkCard(snapshot),/Live execution status/);assert.match(liveWorkCard(snapshot),/coolify: auth required/);assert.match(liveWorkCard(snapshot),/1 running · 1 waiting · 1 failed/);assert.match(liveWorkCard(snapshot),/bounded page/);
 const projected=liveTodoFeed({...snapshot,items:[{...snapshot.items[0],state:'unknown',stage:null}]});assert.equal(projected.items[0].state,'unknown');assert.match(projected.items[0].summary,/unknown/);
 const storage=storageHealthCard({profiles:[{profile:'B2 active vault',backend:'B2',state:'auth_required',authenticated_list_success:false,http_status:403,api_status:403},{profile:'R2 original storage',backend:'R2',state:'connection_error',authenticated_list_success:false,http_status:null,api_status:null}]});
 assert.match(storage,/auth required/);assert.match(storage,/connection error/);assert.doesNotMatch(storage,/Authenticated listing succeeded/);
 assert.match(taskCard({id:'synthetic',title:'Task',status:'running'}),/Documented backlog/);
});

test('live widget renders independently when no manual board snapshot is available',async()=>{
 const app=await readFile(new URL('../public/app.js',import.meta.url),'utf8');
 const helper=app.slice(app.indexOf('function renderLiveWork(){'),app.indexOf('\nfunction render()'));
 let node=null;const kpis={after:value=>{node=value;}};
 const context={$:id=>id==='kpis'?kpis:node,document:{createElement:()=>({})},liveWork:null,liveWorkCard};
 vm.runInNewContext(helper+';renderLiveWork();',context);
 assert.equal(node.id,'live-work-summary');assert.match(node.innerHTML,/Live execution status unavailable/);
 assert.match(app,/renderLiveWork\(\);\s+if\(current\)renderKpis/);
});

test('ToDo projection puts active work above newer history and sorts real clocks with stable ties',()=>{
 const item=(id,state,clocks={})=>({id,provider:'proffer',title:'Proffer operation',state,lifecycle:state,stage:null,wait:null,...clocks});
 const snapshot={observed_at:'2026-10-11T21:00:00.000Z',state:'ready',scope:'bounded provider pages',counts:{running:4,queued:1,waiting:1,rerun_required:1,failed:1,unknown:1,completed:1,cancelled:1,total:11},providers:[{provider:'proffer',state:'ready',count:11}],items:[
  item('completed','completed',{updated_at:'2026-10-11T20:00:00Z'}),
  item('cancelled','cancelled',{updated_at:'2026-10-11T20:00:00Z'}),
  item('failed','failed',{updated_at:'2026-10-11T20:00:00Z'}),
  item('waiting','waiting',{created_at:'2026-01-01T00:00:00Z'}),
  item('running-old','running',{updated_at:'2026-01-01T00:00:00Z'}),
  item('running-new-a','running',{started_at:'2026-01-03T00:00:00Z'}),
  item('running-new-b','running',{updated_at:'2026-01-02T00:00:00Z',started_at:'2026-01-03T00:00:00Z'}),
  item('running-unknown-clock','running',{updated_at:'invalid'}),
  item('unknown','unknown'),item('rerun','rerun_required'),item('queued','queued'),
 ]};
 const original=structuredClone(snapshot),projected=liveTodoFeed(snapshot);
 assert.deepEqual(projected.items.map(row=>row.id),['running-new-a','running-new-b','running-old','running-unknown-clock','queued','waiting','rerun','failed','unknown','completed','cancelled']);
 assert.deepEqual(projected.items.slice(0,8).map(row=>row.state),['running','running','running','running','queued','waiting','rerun_required','failed']);
 assert.deepEqual(snapshot,original);assert.deepEqual(projected.counts,original.counts);assert.deepEqual(projected.providers,original.providers);assert.equal(projected.observed_at,original.observed_at);assert.equal(projected.generated_at,original.observed_at);
 for(const row of projected.items){const native=original.items.find(item=>item.id===row.id);assert.equal(row.updated_at,native.updated_at);assert.equal(row.started_at,native.started_at);assert.equal(row.created_at,native.created_at);}
});
