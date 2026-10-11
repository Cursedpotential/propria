import {open, lstat} from 'node:fs/promises';
import {constants} from 'node:fs';
import {isAbsolute} from 'node:path';
import {createHash} from 'node:crypto';

const MAX_BODY_BYTES = 1024 * 1024;
const MAX_TOKEN_BYTES = 4098;
const REQUEST_TIMEOUT_MS = 10000;
const CACHE_MS = 20000;
const MAX_OPERATIONS = 100;
const MAX_APPS = 8;
const MAX_DEPLOYMENTS_PER_APP = 10;
const MAX_CONCURRENT = 4;
const MAX_TEMPORAL_CLUSTERS = 2;
const TEMPORAL_NAMES = new Set(['original', 'trial']);
const WORKFLOW_LABELS = new Map([['ResumeHashLedgerCopy','Storage preservation copy'],['ZeroCandidateExhaustive20261007','Storage recovery scan'],['AprilGooglePayloads','Google export processing'],['PDFTakeoutBackups','Google export processing'],['ProfferWorkflow','Proffer workflow']]);
const WORKFLOW_STATES = new Map([['RUNNING','running'],['COMPLETED','completed'],['FAILED','failed'],['CANCELED','cancelled'],['TERMINATED','cancelled'],['TIMED_OUT','failed']]);
const ID_PATTERN = /^[A-Za-z0-9_-]{1,128}$/;
const OPERATION_ID_PATTERN = /^[A-Za-z0-9_-]{32,128}$/;
const OPERATION_STATES = new Set(['running', 'awaiting_repair_decision', 'awaiting_preview_decision', 'rerun_required', 'completed', 'failed', 'cancelled', 'unavailable']);
const DEPLOYMENT_STATES = new Map([['queued','queued'], ['in_progress','running'], ['finished','completed'], ['failed','failed'], ['cancelled','cancelled'], ['canceled','cancelled']]);
const WAITS = new Set(['repair_decision', 'preview_decision']);
// Only stagegraph constants can cross this projection; arbitrary upstream text is private.
const STAGES = new Set([
 'register_source_activity', 'retain_original_activity', 'assess_source_repair_activity', 'resolve_source_repair_activity',
 'capture_filesystem_metadata_activity', 'fingerprint_source_activity', 'inventory_container_activity', 'extract_embedded_metadata_activity',
 'select_parser_activity', 'execute_parser_activity', 'persist_raw_generation_activity', 'fingerprint_raw_records_activity',
 'fingerprint_raw_generation_activity', 'reconcile_record_accounting_activity', 'reconcile_byte_coverage_activity', 'verify_raw_coverage_against_source_activity',
 'normalize_generation_activity', 'persist_normalized_generation_activity', 'persist_lineage_activity', 'validate_raw_lineage_activity',
 'hash_normalized_records_activity', 'hash_normalized_generation_activity', 'verify_normalized_generation_activity', 'publish_preview_activity',
 'seal_generation_activity', 'publish_generation_activity', 'chunk_document_activity', 'derive_sms_threads_activity', 'publish_context_search_activity',
 'resolve_context_participants_activity', 'propose_first_party_context_activity', 'confirm_first_party_context_activity', 'commit_first_party_messages_activity',
 'commit_first_party_context_threads_activity', 'record_auto_approval_activity', 'commit_call_log_activity', 'match_message_occurrences_activity',
 'repair.find_other_version', 'repair.salvage_truncated_xml', 'repair.lenient_decode', 'repair_validate_plan_activity', 'repair_record_step_receipt_activity',
]);

/** Represent a provider failure with a constant public code and no upstream error text.
 * Inputs: safe state/code constants. Outputs: local error. Effects: none.
 * Choose instead of forwarding network, credential, parsing or provider errors to the portal.
 */
class ProviderFailure extends Error {
 constructor(state, code) { super(code); this.state = state; this.code = code; }
}

/** Hash provider-local identity before it reaches portal output.
 * Inputs: provider and bounded opaque upstream identity. Outputs: stable SHA-256 identity.
 * Effects: none. Choose for item/check keys; never expose source, request or matter identifiers.
 */
function opaqueId(provider, value) {
 return createHash('sha256').update(provider + '\0' + value).digest('hex');
}

/** Validate an administrator-configured HTTP base without embedded credentials or queries.
 * Inputs: server-side URL. Outputs: normalized URL or null. Effects: none.
 * Choose before attaching service credentials; request parameters never configure providers.
 */
function configuredUrl(value) {
 try {
  if (typeof value !== 'string' || value.length > 2048) return null;
  const url = new URL(value);
  return ['http:', 'https:'].includes(url.protocol) && !url.username && !url.password && !url.search && !url.hash ? url : null;
 } catch { return null; }
}

/** Retain only valid timestamps actually supplied by a provider.
 * Inputs: candidate timestamp. Outputs: normalized ISO timestamp or null. Effects: none.
 * Choose for native clocks; do not substitute observation time for missing start/update time.
 */
function timestamp(value) {
 return typeof value === 'string' && value.length <= 64 && /^\d{4}-\d\d-\d\dT.*(?:Z|[+-]\d\d:\d\d)$/.test(value) && Number.isFinite(Date.parse(value)) ? new Date(value).toISOString() : null;
}

/** Read a bounded regular credential file, rejecting symlinks and replacement races.
 * Inputs: absolute server-side token path and minimum token length. Outputs: token in memory.
 * Effects: local file read only; never logs or persists the token. Choose for provider Bearer auth.
 */
async function credential(path, minimum) {
 let file;
 try {
  if (typeof path !== 'string' || !isAbsolute(path)) throw Error();
  const before = await lstat(path);
  if (!before.isFile() || before.isSymbolicLink() || before.size < minimum || before.size > MAX_TOKEN_BYTES) throw Error();
  file = await open(path, constants.O_RDONLY | (constants.O_NOFOLLOW || 0));
  const after = await file.stat();
  if (!after.isFile() || before.dev !== after.dev || before.ino !== after.ino) throw Error();
  const bytes = Buffer.alloc(MAX_TOKEN_BYTES + 1);
  let length = 0;
  while (length < bytes.length) {
   const read = await file.read(bytes, length, bytes.length - length, null);
   if (!read.bytesRead) break;
   length += read.bytesRead;
  }
  if (length > MAX_TOKEN_BYTES) throw Error();
  const token = bytes.subarray(0, length).toString('utf8').replace(/[\r\n]+$/, '');
  if (token.length < minimum || token.length > 4096 || !/^[A-Za-z0-9._~+\/|=-]+$/.test(token)) throw Error();
  return token;
 } catch { throw new ProviderFailure('unavailable', 'credential_unavailable'); }
 finally { await file?.close().catch(() => {}); }
}

/** Fetch at most one MiB of JSON with a ten-second deadline and no redirect credential forwarding.
 * Inputs: configured URL, optional Bearer token and fetch implementation. Outputs: parsed bounded JSON.
 * Effects: GET only, optional provider authentication, cancels response streams; no logs or writes.
 * Choose for metadata endpoints, never source content or deployment logs.
 */
async function boundedJson(url, token, fetcher) {
 let response;
 try {
  response = await fetcher(url, {method:'GET', redirect:'manual', signal:AbortSignal.timeout(REQUEST_TIMEOUT_MS), headers:{Accept:'application/json', ...(token ? {Authorization:'Bearer ' + token} : {})}});
  if (response.status === 401 || response.status === 403) throw new ProviderFailure('auth_required', response.status === 403 ? 'access_denied' : 'authentication_required');
  if (!response.ok) throw new ProviderFailure('unavailable', 'http_error');
  const length = Number(response.headers.get('content-length') || 0);
  if (length > MAX_BODY_BYTES) throw new ProviderFailure('unavailable', 'response_too_large');
  if (!response.body?.getReader) throw new ProviderFailure('unavailable', 'invalid_response');
  const reader = response.body.getReader();
  const chunks = []; let bytes = 0;
  try {
   for (;;) {
    const {value, done} = await reader.read();
    if (done) break;
    bytes += value.byteLength;
    if (bytes > MAX_BODY_BYTES) throw new ProviderFailure('unavailable', 'response_too_large');
    chunks.push(Buffer.from(value));
   }
   try { return JSON.parse(Buffer.concat(chunks, bytes).toString('utf8')); }
   catch { throw new ProviderFailure('unavailable', 'invalid_response'); }
  } finally { await reader.cancel().catch(() => {}); reader.releaseLock(); }
 } catch (error) {
  if (error instanceof ProviderFailure) throw error;
  throw new ProviderFailure('unavailable', 'request_failed');
 } finally { if (response?.body && !response.body.locked) await response.body.cancel().catch(() => {}); }
}

/** Validate a configured Workbench Activity link without item identifiers or private queries.
 * Inputs: server-side Activity URL. Outputs: safe static URL or null. Effects: none.
 * Choose for Proffer navigation; upstream source/request links are never forwarded.
 */
function activityLink(value) {
 const url = configuredUrl(value);
 return url && /^\/activity\/?$/.test(url.pathname) ? url.href : null;
}

/** Validate an exact Coolify deployment link on the configured control origin.
 * Inputs: row, app configuration and API base. Outputs: exact safe deployment URL or null.
 * Effects: none. Choose configured verified prefixes when history omits deployment_url.
 */
function deploymentLink(row, app, base) {
 const pattern = /^\/project\/[A-Za-z0-9_-]+\/environment\/[A-Za-z0-9_-]+\/application\/[A-Za-z0-9_-]+\/deployment\/[A-Za-z0-9_-]+\/?$/;
 const candidate = configuredUrl(row.deployment_url);
 if (candidate && candidate.origin === base.origin && pattern.test(candidate.pathname) && candidate.pathname.includes('/application/' + app.uuid + '/deployment/') && candidate.pathname.replace(/\/$/,'').endsWith('/' + row.deployment_uuid)) return candidate.href;
 const prefix = configuredUrl(app.deploymentUrlPrefix);
 if (!prefix || prefix.origin !== base.origin || !prefix.pathname.endsWith('/application/' + app.uuid + '/deployment/')) return null;
 prefix.pathname += row.deployment_uuid;
 return pattern.test(prefix.pathname) ? prefix.href : null;
}

/** Project one durable operation through an explicit metadata allowlist.
 * Inputs: operation summary, configured Activity link and observation time. Outputs: safe item.
 * Effects: none. Choose instead of returning raw summaries containing source/person/matter fields.
 */
function operationItem(row, url, observedAt) {
 if (!row || !OPERATION_ID_PATTERN.test(row.preview_handle || '') || row.service !== 'proffer') throw new ProviderFailure('unavailable', 'invalid_response');
 const lifecycle = OPERATION_STATES.has(row.lifecycle) ? row.lifecycle : 'unknown';
 const state = lifecycle.startsWith('awaiting_') ? 'waiting' : lifecycle === 'unavailable' ? 'unknown' : lifecycle;
 return {id:opaqueId('proffer', row.preview_handle), provider:'proffer', title:'Proffer operation', lifecycle, state,
  stage:STAGES.has(row.current_stage) ? row.current_stage : null, wait:WAITS.has(row.wait) ? row.wait : null,
  terminal:typeof row.terminal === 'boolean' ? row.terminal : null, observed_at:observedAt,
  created_at:timestamp(row.created_at), started_at:timestamp(row.started_at), updated_at:timestamp(row.updated_at), finished_at:null, url};
}

/** Project one deployment without application names, commits, request IDs or build logs.
 * Inputs: deployment, configured app/API base and observation time. Outputs: safe item.
 * Effects: none. Choose for lifecycle counts; HTTP success alone never implies completion.
 */
function deploymentItem(row, app, base, observedAt) {
 if (!row || !ID_PATTERN.test(row.deployment_uuid || '')) throw new ProviderFailure('unavailable', 'invalid_response');
 const state = DEPLOYMENT_STATES.get(row.status) || 'unknown';
 return {id:opaqueId('coolify', app.uuid + '\0' + row.deployment_uuid), provider:'coolify', title:'Service deployment',
  lifecycle:DEPLOYMENT_STATES.has(row.status) ? row.status : 'unknown', state, stage:state === 'running' ? 'deployment' : null,
  wait:null, terminal:['completed','failed','cancelled'].includes(state) ? true : state === 'unknown' ? null : false,
  observed_at:observedAt, created_at:timestamp(row.created_at), started_at:timestamp(row.started_at), updated_at:timestamp(row.updated_at), finished_at:timestamp(row.finished_at), url:deploymentLink(row, app, base)};
}

/** Read one bounded operations page from the existing starter API.
 * Inputs: server-side provider configuration, shared bounded requester and observation time.
 * Outputs: safe provider status/items. Effects: credential-file read and authenticated GET only.
 * Choose for durable Proffer state; never starts, approves or cancels operations.
 */
async function readProffer(config, request, observedAt) {
 const base = configuredUrl(config?.baseUrl);
 if (!base || !config?.tokenFile) return {provider:'proffer', state:'unconfigured', code:'not_configured', observed_at:observedAt, truncated:false, items:[]};
 try {
  const token = await credential(config.tokenFile, 32);
  const url = new URL(base); url.pathname = url.pathname.replace(/\/$/,'') + '/reference-import/operations'; url.search = '?limit=' + MAX_OPERATIONS;
  const body = await request(url, token);
  if (!body || !Array.isArray(body.items)) throw new ProviderFailure('unavailable', 'invalid_response');
  const items = body.items.slice(0, MAX_OPERATIONS).map(row => operationItem(row, activityLink(config.activityUrl), observedAt));
  return {provider:'proffer', state:'ready', code:null, observed_at:observedAt, truncated:body.items.length > MAX_OPERATIONS || Boolean(body.next_cursor), items};
 } catch (error) { return {provider:'proffer', state:error.state || 'unavailable', code:error.code || 'request_failed', observed_at:observedAt, truncated:false, items:[]}; }
}

/** Read bounded deployment pages only for explicitly allowlisted applications.
 * Inputs: server-side provider configuration, shared bounded requester and observation time.
 * Outputs: safe items with per-app hashed availability checks. Effects: credential read and GETs.
 * Choose instead of walking the team-wide app or deployment inventory.
 */
async function readCoolify(config, request, observedAt) {
 const base = configuredUrl(config?.baseUrl);
 const apps = Array.isArray(config?.apps) ? config.apps : [];
 const pageSize = config?.pageSize ?? 1;
 if (!base || !config?.tokenFile || !apps.length || apps.length > MAX_APPS || !Number.isInteger(pageSize) || pageSize < 1 || pageSize > MAX_DEPLOYMENTS_PER_APP || apps.some(app=>!ID_PATTERN.test(app?.uuid || '')) || new Set(apps.map(app=>app.uuid)).size !== apps.length) return {provider:'coolify', state:'unconfigured', code:'not_configured', observed_at:observedAt, truncated:false, items:[], checks:[]};
 let token;
 try { token = await credential(config.tokenFile, 1); }
 catch (error) { return {provider:'coolify', state:error.state, code:error.code, observed_at:observedAt, truncated:false, items:[], checks:[]}; }
 const results = new Array(apps.length); let index = 0;
 await Promise.all(Array.from({length:Math.min(MAX_CONCURRENT - 1, apps.length)}, async()=>{
  while (index < apps.length) {
   const i = index++; const app = apps[i];
   try {
    const url = new URL(base); url.pathname = url.pathname.replace(/\/$/,'') + '/deployments/applications/' + app.uuid; url.search = '?skip=0&take=' + pageSize;
    const body = await request(url, token);
    const rows = Array.isArray(body) ? body : body?.deployments;
    if (!Array.isArray(rows)) throw new ProviderFailure('unavailable', 'invalid_response');
    results[i] = {id:opaqueId('coolify-app',app.uuid), state:'ready', code:null, truncated:rows.length > pageSize || (Number.isInteger(body?.count) ? body.count > rows.length : rows.length >= pageSize), items:rows.slice(0,pageSize).map(row=>deploymentItem(row,app,base,observedAt))};
   } catch (error) { results[i] = {id:opaqueId('coolify-app',app.uuid), state:error.state || 'unavailable', code:error.code || 'request_failed', truncated:false, items:[]}; }
  }
 }));
 const ready = results.filter(result=>result.state==='ready').length;
 const state = ready === results.length ? 'ready' : ready ? 'partial' : results.every(result=>result.state==='auth_required') ? 'auth_required' : 'unavailable';
 return {provider:'coolify', state, code:state==='ready'?null:'provider_checks_incomplete', observed_at:observedAt, truncated:results.some(result=>result.truncated), items:results.flatMap(result=>result.items), checks:results.map(({items,...result})=>({...result,count:items.length}))};
}

/** Share four request slots across all providers for the entire bounded response read.
 * Inputs: fetch implementation. Outputs: queued bounded JSON request function.
 * Effects: RAM queue only; choose instead of independent per-provider concurrency limits.
 */
function boundedRequests(fetcher) {
 let active = 0; const queue = [];
 /** Run one GET after a global slot is available and release it after stream cancellation.
  * Inputs: fixed provider URL and optional token. Outputs: bounded parsed JSON.
  * Effects: read-only network request; choose for every live-work provider request.
  */
 return async function request(url, token) {
  await new Promise(resolve => { queue.push(resolve); if (active < MAX_CONCURRENT) { active++; queue.shift()(); } });
  try { return await boundedJson(url, token, fetcher); }
  finally { if (queue.length) queue.shift()(); else active--; }
 };
}

/** Project Temporal visibility metadata without exposing handles, memo, search attributes or raw types.
 * Inputs: one execution row, validated static cluster configuration and observation time.
 * Outputs: hashed identity, allowlisted label and native lifecycle clocks only. Effects: none.
 * Choose for running-list observations, never workflow history, input or failure endpoints.
 */
function temporalItem(row, cluster, observedAt) {
 const workflowId = row?.execution?.workflowId, runId = row?.execution?.runId;
 if (typeof workflowId !== 'string' || !workflowId.length || workflowId.length > 2048 || typeof runId !== 'string' || !runId.length || runId.length > 128 || typeof row.status !== 'string') throw new ProviderFailure('unavailable','invalid_response');
 const native = row.status.startsWith('WORKFLOW_EXECUTION_STATUS_') ? row.status.slice(26) : '';
 const state = WORKFLOW_STATES.get(native) || 'unknown';
 return {id:opaqueId('temporal:' + cluster.name,workflowId + '\0' + runId),provider:'temporal:' + cluster.name,
  title:WORKFLOW_LABELS.get(row.type?.name) || 'Background workflow', lifecycle:state,state,stage:null,wait:null,
  terminal:state === 'unknown' ? null : state !== 'running',observed_at:observedAt,created_at:null,
  started_at:timestamp(row.startTime),updated_at:null,finished_at:timestamp(row.closeTime),url:configuredUrl(cluster.uiUrl).href};
}

/** Read one bounded default-namespace running page on an explicitly named Temporal cluster.
 * Inputs: validated static configuration, shared requester and observation clock. Outputs: safe provider status/items.
 * Effects: one unauthenticated tailnet GET, no token forwarding or follow-up pagination.
 * Choose for live active-work visibility; page cursors indicate truncation rather than triggering more requests.
 */
async function readTemporal(cluster, request, observedAt) {
 const provider = 'temporal:' + cluster.name;
 try {
  const url = new URL('/api/v1/namespaces/default/workflows',cluster.baseUrl);
  url.searchParams.set('query','ExecutionStatus="Running"'); url.searchParams.set('pageSize',String(MAX_OPERATIONS));
  const body = await request(url);
  if (!body || typeof body !== 'object' || Array.isArray(body) || Object.keys(body).some(key => !['executions','nextPageToken'].includes(key)) || (body.executions !== undefined && !Array.isArray(body.executions)) || (body.nextPageToken !== undefined && (typeof body.nextPageToken !== 'string' || body.nextPageToken.length > 8192))) throw new ProviderFailure('unavailable','invalid_response');
  const rows = body.executions || [];
  return {provider,state:'ready',code:null,observed_at:observedAt,truncated:rows.length > MAX_OPERATIONS || Boolean(body.nextPageToken),items:rows.slice(0,MAX_OPERATIONS).map(row=>temporalItem(row,cluster,observedAt))};
 } catch (error) { return {provider,state:error.state || 'unavailable',code:error.code || 'request_failed',observed_at:observedAt,truncated:false,items:[]}; }
}

/** Validate the optional two-cluster Temporal allowlist before making any visibility request.
 * Inputs: administrator configuration, shared requester and observation clock. Outputs: provider read promises.
 * Effects: configured read-only GETs only; invalid configuration remains explicitly unconfigured.
 * Choose at refresh composition; absent Temporal configuration preserves the existing two-provider contract.
 */
function temporalProviders(config, request, observedAt) {
 const clusters = config?.clusters;
 if (!Array.isArray(clusters) || !clusters.length || clusters.length > MAX_TEMPORAL_CLUSTERS || new Set(clusters.map(c=>c?.name)).size !== clusters.length || clusters.some(c=>!TEMPORAL_NAMES.has(c?.name) || (c.namespace !== undefined && c.namespace !== 'default') || !configuredUrl(c.baseUrl) || configuredUrl(c.baseUrl).pathname !== '/' || !configuredUrl(c.uiUrl))) return [Promise.resolve({provider:'temporal',state:'unconfigured',code:'not_configured',observed_at:observedAt,truncated:false,items:[]})];
 return clusters.map(cluster=>readTemporal(cluster,request,observedAt));
}

/** Create a cached, single-flight, read-only projection of bounded live work provider pages.
 * Inputs: providers {proffer:{baseUrl,tokenFile,activityUrl},coolify:{baseUrl,tokenFile,
 * apps:[{uuid,deploymentUrlPrefix?}],pageSize?:1..10},temporal?:{clusters:[{name:'original'|'trial',
 * baseUrl,namespace?:'default',uiUrl}]}}, optional fetcher and millisecond clock.
 * Starter base is its root URL; Coolify base includes /api/v1. Coolify defaults to one record
 * per app because its history endpoint includes logs even though this projection drops them.
 * Outputs: {read()} returning safe items, actual lifecycle counts, provider states and scope.
 * Effects: configured credential-file reads and bounded GETs, authenticated for Proffer/Coolify; 20-second RAM
 * cache, global four-request limit, no overlapping refresh, persistence, workflow starts or mutations.
 * Choose for live-work widgets instead of presenting a manually synced TODO document as live.
 */
export function createLiveWork({providers={}, fetcher=fetch, now=Date.now}={}) {
 let cache = null, cachedAt = 0, pending = null;
 const request = boundedRequests(fetcher);
 /** Read or refresh one sanitized live-work snapshot without overlapping provider collections.
  * Inputs: none. Outputs: cloned safe snapshot. Effects: bounded provider reads when stale.
  * Choose for portal routes; missing/error providers remain explicit rather than empty success.
  */
 async function read() {
  if (pending) return structuredClone(await pending);
  if (cache && now() - cachedAt < CACHE_MS) return structuredClone(cache);
  pending = (async()=>{
   const observedAt = new Date(now()).toISOString();
   const results = await Promise.all([readProffer(providers.proffer,request,observedAt),readCoolify(providers.coolify,request,observedAt),...(providers.temporal === undefined ? [] : temporalProviders(providers.temporal,request,observedAt))]);
   const items = results.flatMap(result=>result.items);
   const counts = {queued:0,running:0,waiting:0,rerun_required:0,completed:0,failed:0,cancelled:0,unknown:0,total:items.length};
   const lifecycle_counts = {};
   for (const item of items) { counts[item.state]++; const key=item.provider + ':' + item.lifecycle; lifecycle_counts[key]=(lifecycle_counts[key]||0)+1; }
   const states = results.map(result=>result.state);
   const snapshot = {observed_at:observedAt, state:states.every(state=>state==='ready')?'ready':states.every(state=>state==='unconfigured')?'unconfigured':states.some(state=>state==='ready'||state==='partial')?'partial':'unavailable', scope:providers.temporal === undefined ? 'Latest bounded provider pages only; counts describe returned items, not all historical or active work.' : 'Latest bounded provider observations only; Temporal covers default namespaces only. A workflow may also appear as a Proffer operation; counts are returned observations, not unique jobs or all work.', cache_seconds:20, items, counts, lifecycle_counts, providers:results.map(({items,...result})=>({...result,count:items.length}))};
   cache = snapshot; cachedAt = now(); return snapshot;
  })().finally(()=>{pending=null;});
  return structuredClone(await pending);
 }
 return {read};
}
