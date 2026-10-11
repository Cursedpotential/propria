const MAX_BODY_BYTES = 256 * 1024;
const REQUEST_TIMEOUT_MS = 10000;
const CACHE_MS = 30000;
const MAX_CONCURRENT = 2;
const DEFAULT_BASE_URL = 'https://files.tilapia-skilift.ts.net';

// Technical mount configuration stays server-side; only profile/backend labels are projected.
export const STORAGE_PROFILES = Object.freeze([
 Object.freeze({profile:'B2 active vault',backend:'B2',path:'/b2/salem-data'}),
 Object.freeze({profile:'R2 original storage',backend:'R2',path:'/r2/casebible-raw'}),
 Object.freeze({profile:'R2 sorted storage',backend:'R2',path:'/r2/casebible-sorted'}),
 Object.freeze({profile:'R2 quarantine storage',backend:'R2',path:'/r2/casebible-quarantine'}),
]);

/** Validate the configured OpenList root without embedded credentials or private query data.
 * Inputs: server-side base URL. Outputs: list endpoint URL or null. Effects: none.
 * Choose before authenticated listing; browser requests never select upstream destinations.
 */
function listingUrl(value) {
 try {
  if (typeof value !== 'string' || value.length > 2048) return null;
  const url = new URL(value);
  if (!['http:','https:'].includes(url.protocol) || url.username || url.password || url.search || url.hash) return null;
  url.pathname = url.pathname.replace(/\/$/,'') + '/api/fs/list';
  return url;
 } catch { return null; }
}

/** Restrict configured checks to at most the four existing source-controlled mount profiles.
 * Inputs: optional subset of fixed profile configurations. Outputs: copied safe configs or null.
 * Effects: none. Choose to prevent arbitrary path listing or unbounded inventory traversal.
 */
function fixedProfiles(profiles) {
 if (!Array.isArray(profiles) || !profiles.length || profiles.length > STORAGE_PROFILES.length) return null;
 const seen = new Set(); const out = [];
 for (const row of profiles) {
  const fixed = STORAGE_PROFILES.find(profile=>profile.profile===row?.profile && profile.backend===row?.backend && profile.path===row?.path);
  if (!fixed || seen.has(fixed.profile)) return null;
  seen.add(fixed.profile); out.push({...fixed});
 }
 return out;
}

/** Read at most 256 KiB of a list response and cancel its stream on every outcome.
 * Inputs: fetch Response. Outputs: parsed bounded JSON or a constant failure state.
 * Effects: response-stream reads/cancellation only. Choose for tiny health lists, never downloads.
 */
async function listingBody(response) {
 if (Number(response.headers.get('content-length') || 0) > MAX_BODY_BYTES) return {failure:'response_too_large'};
 if (!response.body?.getReader) return {failure:'invalid_response'};
 const reader = response.body.getReader(); const chunks = []; let bytes = 0;
 try {
  for (;;) {
   const {value, done} = await reader.read(); if (done) break;
   bytes += value.byteLength;
   if (bytes > MAX_BODY_BYTES) return {failure:'response_too_large'};
   chunks.push(Buffer.from(value));
  }
  try { return {value:JSON.parse(Buffer.concat(chunks,bytes).toString('utf8'))}; }
  catch { return {failure:'invalid_response'}; }
 } finally { await reader.cancel().catch(()=>{}); reader.releaseLock(); }
}

/** Check one existing mount with a forced-fresh single-entry authenticated list.
 * Inputs: fixed profile, configured endpoint/token, fetch implementation and millisecond clock.
 * Outputs: status/count metadata only. Effects: read-only POST list with ten-second deadline.
 * Choose instead of the browser bridge list() that may walk twenty thousand entries.
 */
async function checkProfile(profile, url, token, fetcher, now) {
 const started = now(); const signal = AbortSignal.timeout(REQUEST_TIMEOUT_MS); let response;
 const result = {profile:profile.profile,backend:profile.backend,state:'connection_error',checked_at:new Date(started).toISOString(),latency_ms:null,http_status:null,api_status:null,authenticated_list_success:false,returned_entry_count:0};
 try {
  response = await fetcher(url,{method:'POST',redirect:'manual',signal,headers:{authorization:token,'content-type':'application/json'},body:JSON.stringify({path:profile.path,password:'',page:1,per_page:1,refresh:true})});
  result.http_status = response.status;
  if (response.status===401 || response.status===403) { result.state='auth_required'; return result; }
  if (!response.ok) { result.state='http_error'; return result; }
  const body = await listingBody(response);
  if (body.failure) { result.state=body.failure; return result; }
  const value = body.value;
  if (!value || typeof value!=='object' || !Number.isInteger(value.code)) { result.state='invalid_response'; return result; }
  result.api_status = value.code;
  if (value.code===401 || value.code===403) { result.state='auth_required'; return result; }
  if (value.code!==200) { result.state='api_error'; return result; }
  const data = value.data;
  const content = data?.content===null && data?.total===0 ? [] : data?.content;
  if (!data || typeof data!=='object' || !Array.isArray(content) || content.length>1 || content.some(entry=>!entry || typeof entry!=='object' || Array.isArray(entry)) || (data.total!==undefined && (!Number.isSafeInteger(data.total) || data.total<content.length))) { result.state='invalid_response'; return result; }
  result.state='connected';result.authenticated_list_success=true;result.returned_entry_count=content.length;
  if (data.total!==undefined) result.total=data.total;
  return result;
 } catch { result.state=signal.aborted?'timeout':'connection_error'; return result; }
 finally {
  result.latency_ms=Math.max(0,now()-started);
  if (response?.body && !response.body.locked) await response.body.cancel().catch(()=>{});
 }
}

/** Create a cached projection of authenticated reachability for four existing OpenList mounts.
 * Inputs: optional root baseUrl, server-only raw OPENLIST_TOKEN, fixed profiles/subset, fetcher,
 * and millisecond clock. Outputs: {read()} returning checked_at/state and safe profile results.
 * Effects: at most four page-one/per-page-one read-only listing POSTs, two concurrent, each
 * capped at 256 KiB and ten seconds; thirty-second RAM cache and single-flight refresh.
 * Choose for actual B2/R2 connection checks; this does not enable R2 processing or migrate data.
 */
export function createStorageHealth({baseUrl=DEFAULT_BASE_URL,token=process.env.OPENLIST_TOKEN,profiles=STORAGE_PROFILES,fetcher=fetch,now=Date.now}={}) {
 const configured = fixedProfiles(profiles); const url = listingUrl(baseUrl);
 const validToken = typeof token==='string' && token.length>0 && Buffer.byteLength(token)<=4096 && /^[\x21-\x7e]+$/.test(token);
 let cache = null, cachedAt = 0, pending = null;
 /** Read a cached storage check or share one bounded refresh with concurrent callers.
  * Inputs: none. Outputs: cloned metadata-only snapshot, including explicit failures.
  * Effects: configured list probes when stale; no admin inventory, downloads or provider writes.
  * Choose for portal routes rather than guessing connection success from a browser health check.
  */
 async function read() {
  if (pending) return structuredClone(await pending);
  if (cache && now()-cachedAt<CACHE_MS) return structuredClone(cache);
  pending=(async()=>{
   const checkedAt=new Date(now()).toISOString();
   if (!configured || !url || !validToken) {
    const rows=(configured || STORAGE_PROFILES).map(profile=>({profile:profile.profile,backend:profile.backend,state:'unconfigured',checked_at:checkedAt,latency_ms:null,http_status:null,api_status:null,authenticated_list_success:false,returned_entry_count:0}));
    const snapshot={checked_at:checkedAt,state:'unconfigured',profiles:rows};cache=snapshot;cachedAt=now();return snapshot;
   }
   let index=0;const rows=new Array(configured.length);
   await Promise.all(Array.from({length:Math.min(MAX_CONCURRENT,configured.length)},async()=>{while(index<configured.length){const i=index++;rows[i]=await checkProfile(configured[i],url,token,fetcher,now);}}));
   const connected=rows.filter(row=>row.authenticated_list_success).length;
   const snapshot={checked_at:checkedAt,state:connected===rows.length?'ready':connected?'partial':'unavailable',profiles:rows};
   cache=snapshot;cachedAt=now();return snapshot;
  })().finally(()=>{pending=null;});
  return structuredClone(await pending);
 }
 return {read};
}
