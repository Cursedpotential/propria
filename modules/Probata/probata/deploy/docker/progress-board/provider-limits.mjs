import {readFile} from 'node:fs/promises';
import {randomUUID} from 'node:crypto';
const names=new Set(['claude','codex','portkey']);
const allowedOrigins=new Set(['https://homepage.tilapia-skilift.ts.net','https://homepage.int.mitechconsult.com']);
export function validateLimit(value,now=Date.now()) {
  if(!names.has(value.provider)||!['available','low','exhausted'].includes(value.state))throw Error('Choose a provider and usage state');
  if(value.state!=='available' && (!Number.isFinite(Date.parse(value.limited_until))||Date.parse(value.limited_until)<=now))throw Error('Choose a future reset date and time');
  if(value.fallback_provider && !names.has(value.fallback_provider))throw Error('Unknown fallback provider');
  if(value.fallback_provider===value.provider && value.provider!=='portkey')throw Error('Choose a different fallback provider');
  if(value.fallback_account && (value.fallback_provider!=='portkey'||!/^[a-zA-Z0-9][a-zA-Z0-9_.:/ -]{0,99}$/.test(value.fallback_account)||/^(sk-|pk-)/i.test(value.fallback_account)))throw Error('Use a Portkey route/account name, never an API key');
  if(value.fallback_provider==='portkey'&&!value.fallback_account)throw Error('Enter the named Portkey account/config route');
  return {provider:value.provider,state:value.state,limited_until:value.state==='available'?'':new Date(value.limited_until).toISOString(),fallback_provider:value.state==='available'?'':value.fallback_provider||'',fallback_account:value.state==='available'?'':value.fallback_account||'',setting_id:randomUUID()};
}
export function effectiveRoute(requested,limits,now=Date.now()) {
  let current=requested,account='',seen=new Set();
  for(let i=0;i<4;i++){
    if(seen.has(current))return {provider:null,reason:'Fallback routing forms a loop'};seen.add(current);
    const limit=limits.find(item=>item.provider===current);
    if(!limit||limit.state==='available'||Date.parse(limit.limited_until)<=now)return {provider:current,account,reason:seen.size>1?'Manual usage override':'Requested provider'};
    if(!limit.fallback_provider)return {provider:null,reason:current+' marked '+limit.state+' until '+limit.limited_until};
    if(limit.fallback_provider==='portkey'&&limit.fallback_account)return {provider:'portkey',account:limit.fallback_account,reason:'Manual Portkey account override'};
    current=limit.fallback_provider;
  }
  return {provider:null,reason:'Invalid routing configuration'};
}
export function createProviderLimits({loadConfig=async()=>JSON.parse(await readFile('/run/secrets/portal-repair/config.json','utf8')),fetcher=fetch}={}){
  const request=async(method,body)=>{
    const config=await loadConfig();if(!config.settings_table_id)throw Error('Usage settings are not configured');
    const response=await fetcher(config.n8n_url+'/api/v1/data-tables/'+encodeURIComponent(config.settings_table_id)+'/rows'+(method==='GET'?'?limit=100&sortBy=createdAt%3Adesc':''),{method,headers:{'X-N8N-API-KEY':config.api_key,'Content-Type':'application/json'},signal:AbortSignal.timeout(10000),...(body?{body:JSON.stringify({data:[body],returnType:'all'})}:{})});
    if(!response.ok)throw Error('Usage settings storage unavailable');return response.json();
  };
  async function read(){const response=await request('GET');const seen=new Set();return(response.data||[]).filter(row=>{if(seen.has(row.provider))return false;seen.add(row.provider);return true;});}
  async function handle(req,res){
    const send=(status,body)=>{res.writeHead(status,{'Content-Type':'application/json','Cache-Control':'no-store'});res.end(JSON.stringify(body));};
    try{
      if(req.method==='GET'){send(200,{limits:await read()});return;}
      if(req.method!=='POST'){send(405,{message:'Use GET or POST'});return;}
      if(!allowedOrigins.has(req.headers.origin)||req.headers['sec-fetch-site']==='cross-site'){send(403,{message:'Use the portal to change usage settings'});return;}
      if(!(req.headers['content-type']||'').startsWith('application/json')){send(415,{message:'JSON required'});return;}
      let raw='';for await(const chunk of req){raw+=chunk;if(Buffer.byteLength(raw)>2048){send(413,{message:'Request too large'});return;}}
      let setting;try{setting=validateLimit(JSON.parse(raw));}catch(error){send(400,{message:error.message});return;}
      const stored=await request('POST',setting);if(stored[0]?.setting_id!==setting.setting_id)throw Error('Setting receipt unavailable');
      send(201,{setting:stored[0],message:setting.state==='available'?'Usage override cleared.':setting.provider+' marked '+setting.state+' until '+setting.limited_until+'.'});
    }catch{send(503,{message:'Usage settings unavailable; no successful receipt returned.'});}
  }
  return {read,handle};
}
