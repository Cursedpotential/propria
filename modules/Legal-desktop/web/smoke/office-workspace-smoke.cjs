const {chromium}=require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const crypto=require('crypto');const fs=require('fs');
const base='http://127.0.0.1:3015',api='http://127.0.0.1:18010';
function headers(method,target,body){const stamp=String(Math.floor(Date.now()/1000)),nonce=crypto.randomBytes(16).toString('hex');return {'x-legal-bff-timestamp':stamp,'x-legal-bff-nonce':nonce,'x-legal-bff-signature':crypto.createHmac('sha256','synthetic-office-smoke-signature-key-only').update([stamp,nonce,method,target,crypto.createHash('sha256').update(body).digest('hex')].join('\n')).digest('hex')}}
(async()=>{const b=await chromium.launch({headless:true,executablePath:process.env.CHROMIUM_PATH || undefined});try{
 const p=await b.newPage({viewport:{width:1440,height:1100}});const errors=[];p.on('pageerror',e=>errors.push(e.message));
 await p.route('**/api/legal/**',async route=>{const req=route.request(),u=new URL(req.url()),target=u.pathname.replace('/api/legal','')+u.search,body=req.postDataBuffer()||Buffer.alloc(0);const h=headers(req.method(),target,body);if(req.headers()['content-type'])h['content-type']=req.headers()['content-type'];const r=await p.request.fetch(api+target,{method:req.method(),headers:h,data:body.length?body:undefined});await route.fulfill({response:r});});
 await p.goto(base+'/drafts');await p.getByRole('heading',{name:'Documents and writing',exact:true}).waitFor();
 const name='Synthetic browser draft '+Date.now();await p.getByLabel('Document name',{exact:true}).fill(name);await p.getByRole('button',{name:'Start writing',exact:true}).click();
 await p.getByText('Interactive office service is not configured',{exact:true}).waitFor();
 const docButton=p.getByRole('button',{name:new RegExp(name)});if(await docButton.count()!==1)throw Error('Document not created');
 await p.getByRole('button',{name:'Versions',exact:true}).click();await p.getByRole('link',{name:'Version 1',exact:true}).waitFor();
 const download=p.getByRole('link',{name:'Download',exact:true});const href=await download.getAttribute('href');const response=await p.evaluate(async h=>{const r=await fetch(h);return {status:r.status,length:(await r.arrayBuffer()).byteLength}},href);if(response.status!==200||response.length<1000)throw Error('Download failed');
 await p.reload();await p.getByRole('button',{name:new RegExp(name)}).waitFor();
 const starting=p.getByLabel('Starting point');await starting.selectOption({index:1});const templateName='Synthetic template '+Date.now();await p.getByLabel('Document name',{exact:true}).fill(templateName);await p.getByRole('button',{name:'Start writing',exact:true}).click();await p.getByRole('button',{name:new RegExp(templateName)}).waitFor();
 await p.screenshot({path:(process.env.OFFICE_SMOKE_OUTPUT || __dirname)+'/office-workspace-desktop.png',fullPage:true});
 await p.setViewportSize({width:390,height:844});await p.screenshot({path:(process.env.OFFICE_SMOKE_OUTPUT || __dirname)+'/office-workspace-mobile.png',fullPage:true});
 const result={blank_created:true,template_created:true,reload_persistence:true,original_download:response,history:true,editor_unavailable_explicit:true,errors,scope:'real local API and browser; Collabora runtime unavailable; synthetic work products only'};fs.writeFileSync((process.env.OFFICE_SMOKE_OUTPUT || __dirname)+'/office-workspace-proof.json',JSON.stringify(result,null,2));console.log(JSON.stringify(result));await p.unrouteAll({behavior:'ignoreErrors'});if(errors.length)throw Error('Page errors');
}finally{await b.close()}})().catch(e=>{console.error(e.message);process.exit(1)});



