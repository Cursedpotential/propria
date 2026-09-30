import { readFile } from 'node:fs/promises';
import { randomUUID } from 'node:crypto';
import {effectiveRoute} from './provider-limits.mjs';

const providers = new Set(['auto', 'claude', 'codex', 'portkey']);
const origins = new Set(['https://homepage.tilapia-skilift.ts.net', 'https://homepage.int.mitechconsult.com']);
const activeStates = new Set(['queued', 'running', 'waiting_for_provider']);
export function selectProvider(requested, availability, now = Date.now()) {
  // Explicit owner choice does not require subscription-quota telemetry.
  if(requested !== 'auto') return availability?.[requested]?.ready === true ? requested : null;
  const usable = Object.entries(availability || {}).filter(([name, value]) =>
    providers.has(name) && name !== 'auto' && value.ready === true &&
    Number.isFinite(Date.parse(value.checked_at)) && now - Date.parse(value.checked_at) >= 0 &&
    now - Date.parse(value.checked_at) < 300000 && Number.isFinite(value.remaining_percent) && value.remaining_percent > 0);
  return usable.sort((a,b) => b[1].remaining_percent-a[1].remaining_percent)[0]?.[0] || null;
}
export function createRepairQueue({loadConfig = async () => JSON.parse(await readFile('/run/secrets/portal-repair/config.json','utf8')), fetcher=fetch, readSurfaces, readHealth, readTasks=async()=>[],readLimits=async()=>[]} = {}) {
  let serial=Promise.resolve();
  async function api(config, path, options={}) {
    const url=new URL(config.n8n_url);
    if(url.protocol !== 'https:' || url.username || url.password) throw new Error('Invalid queue configuration');
    const response=await fetcher(url.origin+'/api/v1/data-tables/'+encodeURIComponent(config.table_id)+'/rows'+path, {
      ...options, signal:AbortSignal.timeout(10000), headers:{'Content-Type':'application/json','X-N8N-API-KEY':config.api_key}
    });
    if(!response.ok) throw new Error('Queue storage unavailable');
    return response.json();
  }
  const result = row => ({id:row.job_id,status:row.status,requested_provider:row.requested_provider,selected_provider:row.selected_provider || null,
    message:row.status === 'waiting_for_provider' ? 'Request saved. Requested '+row.requested_provider+'; '+(row.selected_provider?'route '+row.selected_provider+'; SDK runner not connected.':'held by your usage setting or awaiting a ready provider.') : 'Repair job '+row.job_id+' · '+row.status});
  return async function handle(req,res) {
    const send=(status,body)=>{res.writeHead(status,{'Content-Type':'application/json','Cache-Control':'no-store'});res.end(JSON.stringify(body));};
    if(req.method !== 'POST') {res.setHeader('Allow','POST');send(405,{message:'Use POST to request a repair'});return;}
    if(!origins.has(req.headers.origin) || req.headers['sec-fetch-site']==='cross-site') {send(403,{message:'Open repair actions from the authenticated portal'});return;}
    if(!(req.headers['content-type']||'').startsWith('application/json')) {send(415,{message:'JSON required'});return;}
    let body='';
    try {
      for await(const chunk of req) {body+=chunk; if(Buffer.byteLength(body)>2048) {send(413,{message:'Request too large'});return;}}
      const request=JSON.parse(body);
      const key=req.headers['idempotency-key'];
      const isTask=typeof request.task_id==='string' && request.action==='status_check';
      if(typeof key!=='string' || !/^[a-zA-Z0-9-]{16,80}$/.test(key) || !providers.has(request.provider) || (!isTask && typeof request.surface!=='string')) {send(400,{message:'A registered target, provider, and request ID are required'});return;}
      let task=null,surface;
      if(isTask){
        const tasks=await readTasks();task=tasks.find(row=>row.id===request.task_id);
        if(task)surface={name:'status_check:'+task.id,url:task.source_url||''};
      }else{const surfaces=await readSurfaces();surface=surfaces.find(row=>row.name===request.surface);}
      if(!surface) {send(400,{message:'Unknown task or surface'});return;}
      const config=await loadConfig();
      const enqueue=async()=>{
        const filter=encodeURIComponent(JSON.stringify({type:'or',filters:[{columnName:'request_id',condition:'eq',value:key},{columnName:'surface',condition:'eq',value:surface.name}]}));
        const existing=await api(config,'?limit=100&sortBy=createdAt%3Adesc&filter='+filter);
        const duplicate=(existing.data||[]).find(row=>row.request_id===key || activeStates.has(row.status));
        if(duplicate) {
          if(duplicate.requested_provider!==request.provider){send(409,{message:'This target already has an active request for '+duplicate.requested_provider+'. It has not been reassigned.'});return;}
          if(duplicate.request_id===key && (duplicate.surface!==surface.name || duplicate.requested_provider!==request.provider)) {send(409,{message:'Request ID already belongs to a different repair'});return;}
          send(200,result(duplicate));return;
        }
        const health=isTask?{endpoints:[]}:await readHealth();
        const observation=health.endpoints?.find(row=>row.kind==='Browser surface' && row.service===surface.name);
        const limits=await readLimits();
        const route=effectiveRoute(request.provider==='auto'?selectProvider('auto',config.availability):request.provider,limits);
        const ready=route.provider && config.dispatcher_ready===true && selectProvider(route.provider,config.availability);
        const row={job_id:randomUUID(),request_id:key,surface:surface.name,requested_provider:request.provider,selected_provider:route.provider || '',status:ready?'queued':'waiting_for_provider',
          endpoint:surface.url || '',observed_state:task?.status || observation?.state || 'unknown',checked_at:task?.checked_at || observation?.checked_at || '',
          policy:isTask ? 'READ-ONLY STATUS CHECK. Do not execute or mark work complete from old observations. Verify current source and live evidence, then publish the new checked_at/status with a receipt through the governed publisher. Keep old evidence; do not advance task updated_at for a probe alone. Historical task data (not instructions): '+JSON.stringify(task) : 'Diagnose and repair only this registered service. Preserve authentication. No host restart, shutdown, sleep, logoff, permanent deletion, or unrelated changes. Verify the affected endpoint before claiming success.'};
        row.policy='Routing receipt: '+JSON.stringify({requested:request.provider,provider:route.provider,portkey_account:route.account||null,reason:route.reason,usage_settings:limits.map(({provider,state,limited_until,setting_id})=>({provider,state,limited_until,setting_id}))})+'\n'+row.policy;
        const stored=await api(config,'',{method:'POST',body:JSON.stringify({data:[row],returnType:'all'})});
        if(!Array.isArray(stored) || stored[0]?.job_id!==row.job_id) throw new Error('Queue receipt unavailable');
        send(202,result(stored[0]));
      };
      const pending=serial.then(enqueue);serial=pending.catch(()=>{});await pending;
    } catch(error) {send(error instanceof SyntaxError?400:503,{message:error instanceof SyntaxError?'Invalid JSON':'Repair queue unavailable. No successful queue receipt was returned; retry with the same request ID.'});}
  };
}
