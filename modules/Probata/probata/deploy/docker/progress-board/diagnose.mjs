// Byline: Codex · 2026-09-12 — bounded connection diagnosis; never print credentials.
import {configFromEnv,QUERY} from './server.mjs';
const c = configFromEnv();
console.log(JSON.stringify({endpoint:c.endpoint,namespace:c.namespace,database:c.database,hasUser:!!c.user,hasPassword:!!c.pass}));
try {
 const r=await fetch(c.endpoint,{method:'POST',headers:{Accept:'application/json','Content-Type':'text/plain','Surreal-NS':c.namespace,'Surreal-DB':c.database,'Surreal-Auth-NS':c.namespace,'Surreal-Auth-DB':c.database,Authorization:`Basic ${Buffer.from(`${c.user}:${c.pass}`).toString('base64')}`},body:'RETURN 1;',signal:AbortSignal.timeout(10000),redirect:'manual'});
 console.log(JSON.stringify({status:r.status,location:r.headers.get('location'),body:(await r.text()).slice(0,700)}));
 const q=await fetch(c.endpoint,{method:'POST',headers:{Accept:'application/json','Content-Type':'text/plain','Surreal-NS':c.namespace,'Surreal-DB':c.database,'Surreal-Auth-NS':c.namespace,'Surreal-Auth-DB':c.database,Authorization:`Basic ${Buffer.from(`${c.user}:${c.pass}`).toString('base64')}`},body:QUERY,signal:AbortSignal.timeout(10000)});
 const data=await q.json();console.log(JSON.stringify({status:q.status,queries:Array.isArray(data)?data.map(x=>({status:x.status,count:Array.isArray(x.result)?x.result.length:null,error:x.status==='ERR'?x.result:null})):data}));
}catch(e){console.log(JSON.stringify({error:e.message,cause:e.cause?.code}));}
