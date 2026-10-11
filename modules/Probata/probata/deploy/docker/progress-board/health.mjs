// Byline: Codex · 2026-09-12 — live bounded, unauthenticated reachability projection.
import {readFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import {resolve} from 'node:path';
import net from 'node:net';
const root=fileURLToPath(new URL('.',import.meta.url));
const now=()=>new Date().toISOString();
/** Probe one configured endpoint without credentials or response-body reads.
 * Inputs: endpoint row and optional fetch implementation. Outputs: bounded status metadata.
 * Effects: one HTTP GET or TCP handshake, canceled after the existing timeout.
 * Choose this for reachability; it does not verify a protected user journey.
 */
export async function probeEndpoint(row,fetcher=fetch) {
 const started=Date.now();const base={service:row.service,kind:row.kind,endpoint:row.url || (row.host?`${row.host}:${row.port}`:null),checked_at:now(),http_status:null,latency_ms:null};
 if(!row.url&&!row.host)return {...base,state:'not connected',detail:row.note || 'Endpoint unknown'};
 if(row.host) {
   const state=await new Promise(resolve=>{const s=net.createConnection({host:row.host,port:row.port});let done=false;const finish=(x)=>{if(done)return;done=true;s.destroy();resolve(x);};s.setTimeout(2500,()=>finish('timeout'));s.on('connect',()=>finish('TCP open'));s.on('error',()=>finish('unreachable'));});
   return {...base,state,latency_ms:Date.now()-started,detail:'TCP handshake only; authentication, queries and storage health not proven'};
 }
 try {
  const u=new URL(row.url);if(!['http:','https:'].includes(u.protocol)||u.username||u.password)throw Error('Invalid endpoint');
  const r=await fetcher(u,{signal:AbortSignal.timeout(5000),redirect:'manual',headers:{'User-Agent':'Propria-Live-Health/1'}});
  const ct=r.headers.get('content-type')||'';await r.body?.cancel();
  const state=r.status===403?'monitor access denied':r.status===401?'auth required':r.status>=300&&r.status<400?'redirect':r.ok?(row.expect==='json'&&!ct.includes('json')?'unexpected content':'responding'):'HTTP error';
  return {...base,state,http_status:r.status,content_type:ct,latency_ms:Date.now()-started,detail:row.expect==='json'&&!ct.includes('json')&&r.ok?'Expected a JSON health API, received another content type. A SPA fallback is not API health.':'Unauthenticated endpoint response only; no workflow or credential check'};
 }catch{return {...base,state:'unreachable',latency_ms:Date.now()-started,detail:'Request failed or timed out; no credentials used'};}
}
/** Probe a bounded registry once per distinct request and retain each surface's own status.
 * Inputs: registry rows and optional fetch implementation. Outputs: at most 100 observations.
 * Effects: at most four concurrent credential-free probes; response bodies are canceled.
 * Choose this over single probes to share public health results without changing browser access.
 */
export async function probeHealthRows(rows,fetcher=fetch) {
 const bounded=rows.slice(0,100);const groups=new Map();
 bounded.forEach((row,i)=>{const key=JSON.stringify([row.url||null,row.host||null,row.port||null,row.expect||'any',row.url||row.host?null:i]);if(!groups.has(key))groups.set(key,[]);groups.get(key).push(i);});
 const work=[...groups.values()];let index=0;const results=new Array(bounded.length);
 await Promise.all(Array.from({length:Math.min(4,work.length)},async()=>{while(index<work.length){const group=work[index++];const result=await probeEndpoint(bounded[group[0]],fetcher);for(const i of group)results[i]={...result,service:bounded[i].service,kind:bounded[i].kind};}}));
 const publicByOrigin=new Map();
 bounded.forEach((row,i)=>{if(row.surface_origin)publicByOrigin.set(row.surface_origin,results[i]);});
 return results.map(row=>{
  if(row.kind!=='Browser surface'||!row.endpoint)return row;
  let origin;try{origin=new URL(row.endpoint).origin;}catch{return row;}
  const health=publicByOrigin.get(origin);if(!health)return row;
  return {...row,public_reachability:{endpoint:health.endpoint,state:health.state,http_status:health.http_status,checked_at:health.checked_at,label:health.state==='responding'?'Server reachable via public health check':'Public health check: '+health.state,scope:'Public endpoint only; protected browser access was not verified'}};
 });
}
/** Collect the configured health snapshot and optional storage telemetry.
 * Inputs: optional fetch implementation. Outputs: bounded endpoint and telemetry snapshot.
 * Effects: reads local registry/telemetry and performs credential-free bounded probes.
 * Choose this for a fresh snapshot; use cachedHealth to consolidate concurrent requests.
 */
export async function collectHealth(fetcher=fetch) {
 const [surfaces,extra]=await Promise.all(['surfaces.json','health-endpoints.json'].map(async f=>JSON.parse(await readFile(resolve(root,f),'utf8'))));
 const rows=[...surfaces.map(s=>({service:s.name,kind:'Browser surface',url:s.url,expect:'any',note:s.status})),...extra].slice(0,100);
 const results=await probeHealthRows(rows,fetcher);
 let storage={observed_at:null,status:'No storage telemetry supplied',filesystems:[],services:[]};
 try{storage=JSON.parse(await readFile(resolve(root,'storage-telemetry.json'),'utf8'));}catch{}
 return {generated_at:now(),poll_seconds:30,probe_host:'ovh-app VPS',scope:'Unauthenticated HTTP/TCP probes. Browser rendering, authenticated workflows and per-service storage sizes require separate proof.',endpoints:results,storage};
}
let cache,pending,at=0;
/** Share one recent health sample across concurrent board requests.
 * Inputs: none. Outputs: snapshot promise cached for 25 seconds after completion.
 * Effects: refreshes bounded probes and process-local cache when stale.
 * Choose this for HTTP routes instead of issuing duplicate fresh health collections.
 */
export async function cachedHealth(){if(cache&&Date.now()-at<25000)return cache;if(!pending)pending=collectHealth().then(x=>{cache=x;at=Date.now();return x;}).finally(()=>pending=null);return pending;}
