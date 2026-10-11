import test from 'node:test';
import assert from 'node:assert/strict';
import {createStorageHealth,STORAGE_PROFILES} from '../storage-health.mjs';

const TOKEN='SYNTHETIC_OPENLIST_TOKEN';
const PRIVATE='PRIVATE_ENTRY_NAME_SIGN_URL_BODY_ERROR';
const CLOCK=Date.parse('2026-10-11T20:00:00Z');
const one=[STORAGE_PROFILES[0]];
const allowed=new Set(['profile','backend','state','checked_at','latency_ms','http_status','api_status','authenticated_list_success','returned_entry_count','total']);

/** Return a synthetic OpenList envelope with private surplus listing fields.
 * Inputs: optional total and API code. Outputs: streaming Response. Effects: none.
 * Choose to verify safe count projection without retaining names, signs or raw URLs.
 */
function response(total=5,code=200) {
 return new Response(JSON.stringify({code,message:PRIVATE,data:{total,content:total?[{name:PRIVATE,sign:PRIVATE,raw_url:'https://private.example/'+PRIVATE,body:PRIVATE}]:null}}),{headers:{'content-type':'application/json'}});
}

test('four fixed mounts receive one forced-fresh bounded read-only list with at most two concurrent requests',async()=>{
 let active=0,maxActive=0;const calls=[];const totals=[5,480,174,11];
 const result=await createStorageHealth({token:TOKEN,now:()=>CLOCK,fetcher:async(url,options)=>{
  calls.push({url:String(url),options});const index=calls.length-1;
  active++;maxActive=Math.max(maxActive,active);await new Promise(resolve=>setTimeout(resolve,3));active--;
  return response(totals[index]);
 }}).read();
 assert.equal(result.state,'ready');assert.equal(calls.length,4);assert.ok(maxActive<=2);
 for(let i=0;i<4;i++){
  assert.equal(calls[i].url,'https://files.tilapia-skilift.ts.net/api/fs/list');
  assert.equal(calls[i].options.method,'POST');assert.equal(calls[i].options.redirect,'manual');assert.ok(calls[i].options.signal instanceof AbortSignal);
  assert.deepEqual(calls[i].options.headers,{authorization:TOKEN,'content-type':'application/json'});
  assert.deepEqual(JSON.parse(calls[i].options.body),{path:STORAGE_PROFILES[i].path,password:'',page:1,per_page:1,refresh:true});
  const row=result.profiles[i];assert.equal(row.state,'connected');assert.equal(row.authenticated_list_success,true);assert.equal(row.returned_entry_count,1);assert.equal(row.total,totals[i]);assert.equal(row.http_status,200);assert.equal(row.api_status,200);
  assert.ok(Object.keys(row).every(key=>allowed.has(key)));
 }
 const serialized=JSON.stringify(result);for(const text of [TOKEN,PRIVATE,'/b2/','/r2/','raw_url','sign','password','https://','content'])assert.ok(!serialized.includes(text),text);
});

test('empty authenticated lists are valid connection evidence without invented entries',async()=>{
 const result=await createStorageHealth({token:TOKEN,profiles:one,fetcher:async()=>response(0)}).read();
 assert.equal(result.state,'ready');assert.equal(result.profiles[0].authenticated_list_success,true);assert.equal(result.profiles[0].returned_entry_count,0);assert.equal(result.profiles[0].total,0);
 const withoutTotal=await createStorageHealth({token:TOKEN,profiles:one,fetcher:async()=>new Response('{"code":200,"data":{"content":[]}}')}).read();
 assert.equal(withoutTotal.profiles[0].state,'connected');assert.ok(!('total' in withoutTotal.profiles[0]));
});

test('HTTP auth, API auth, HTTP failure, API failure and connection errors remain distinct',async()=>{
 for(const status of [401,403,500]){
  let canceled=false;
  const result=await createStorageHealth({token:TOKEN,profiles:one,fetcher:async()=>new Response(new ReadableStream({cancel(){canceled=true;}}),{status})}).read();
  assert.equal(result.state,'unavailable');assert.equal(result.profiles[0].state,status===500?'http_error':'auth_required');assert.equal(result.profiles[0].http_status,status);assert.equal(result.profiles[0].api_status,null);assert.equal(result.profiles[0].authenticated_list_success,false);assert.ok(canceled);
 }
 for(const code of [401,403,500]){
  const result=await createStorageHealth({token:TOKEN,profiles:one,fetcher:async()=>response(5,code)}).read();assert.equal(result.profiles[0].state,code===500?'api_error':'auth_required');assert.equal(result.profiles[0].http_status,200);assert.equal(result.profiles[0].api_status,code);assert.equal(result.profiles[0].authenticated_list_success,false);assert.ok(!JSON.stringify(result).includes(PRIVATE));
 }
 const failed=await createStorageHealth({token:TOKEN,profiles:one,fetcher:async()=>{throw Error(PRIVATE);}}).read();assert.equal(failed.profiles[0].state,'connection_error');assert.equal(failed.profiles[0].http_status,null);assert.ok(!JSON.stringify(failed).includes(PRIVATE));
 const mixed=await createStorageHealth({token:TOKEN,fetcher:async(_url,{body})=>JSON.parse(body).path==='/b2/salem-data'?response():response(5,500)}).read();assert.equal(mixed.state,'partial');assert.equal(mixed.profiles.filter(row=>row.authenticated_list_success).length,1);
});

test('malformed envelopes and oversized result shapes never become authenticated success',async()=>{
 const bodies=[PRIVATE,{code:'200',data:{}},{code:200},{code:200,data:{content:[{},{}],total:2}},{code:200,data:{content:[PRIVATE],total:1}},{code:200,data:{content:[{}],total:0}},{code:200,data:{content:[],total:-1}},{code:200,data:{content:[],total:1.5}}];
 for(const body of bodies){
  const result=await createStorageHealth({token:TOKEN,profiles:one,fetcher:async()=>new Response(typeof body==='string'?body:JSON.stringify(body))}).read();assert.equal(result.profiles[0].state,'invalid_response');assert.equal(result.profiles[0].authenticated_list_success,false);assert.equal(result.profiles[0].returned_entry_count,0);assert.ok(!JSON.stringify(result).includes(PRIVATE));
 }
});

test('256KiB streamed and declared caps cancel the response before unbounded allocation',async()=>{
 for(const declared of [true,false]){
  let canceled=false,pulls=0;
  const stream=new ReadableStream({pull(controller){pulls++;controller.enqueue(new Uint8Array(64*1024));},cancel(){canceled=true;}});
  const result=await createStorageHealth({token:TOKEN,profiles:one,fetcher:async()=>new Response(stream,{headers:declared?{'content-length':String(256*1024+1)}:{}})}).read();
  assert.equal(result.profiles[0].state,'response_too_large');assert.equal(result.profiles[0].authenticated_list_success,false);assert.ok(canceled);assert.ok(pulls<=6);
 }
});

test('invalid credentials, destinations and arbitrary or excessive profiles never trigger requests',async()=>{
 const invalid=[{token:''},{token:'bad\n'+TOKEN},{token:'A'.repeat(4097)},{token:TOKEN,baseUrl:'https://user:pass@files.example'},{token:TOKEN,profiles:[{profile:'B2 active vault',backend:'B2',path:'/private/arbitrary'}]},{token:TOKEN,profiles:[...STORAGE_PROFILES,STORAGE_PROFILES[0]]},{token:TOKEN,profiles:[one[0],one[0]]}];
 for(const options of invalid){const result=await createStorageHealth({...options,fetcher:async()=>{throw Error('must not fetch');}}).read();assert.equal(result.state,'unconfigured');assert.ok(result.profiles.every(row=>row.state==='unconfigured'));}
});

test('30-second cache and single-flight suppress overlapping refreshes and return detached safe copies',async()=>{
 let clock=CLOCK,calls=0,release;
 const adapter=createStorageHealth({token:TOKEN,profiles:one,now:()=>clock,fetcher:async()=>{calls++;await new Promise(resolve=>{release=resolve;});return response();}});
 const reads=Array.from({length:20},()=>adapter.read());while(!release)await new Promise(resolve=>setTimeout(resolve,1));assert.equal(calls,1);release();const results=await Promise.all(reads);
 results[0].profiles[0].profile=PRIVATE;assert.equal((await adapter.read()).profiles[0].profile,'B2 active vault');assert.equal(calls,1);
 clock+=29999;await adapter.read();assert.equal(calls,1);clock++;
 release=null;const next=[adapter.read(),adapter.read()];while(!release)await new Promise(resolve=>setTimeout(resolve,1));assert.equal(calls,2);clock+=40000;next.push(adapter.read());assert.equal(calls,2);release();await Promise.all(next);assert.equal(calls,2);
});

test('ten-second deadline aborts a stalled list and reports timeout rather than success',{timeout:15000},async()=>{
 const started=Date.now();const keepAlive=setTimeout(()=>{},14000);let aborted=false;
 const result=await createStorageHealth({token:TOKEN,profiles:one,fetcher:async(_url,{signal})=>new Promise((_resolve,reject)=>signal.addEventListener('abort',()=>{aborted=true;reject(Error(PRIVATE));},{once:true}))}).read();clearTimeout(keepAlive);
 assert.ok(aborted);assert.ok(Date.now()-started>=9000);assert.ok(Date.now()-started<14000);assert.equal(result.profiles[0].state,'timeout');assert.equal(result.profiles[0].authenticated_list_success,false);assert.ok(!JSON.stringify(result).includes(PRIVATE));
});
