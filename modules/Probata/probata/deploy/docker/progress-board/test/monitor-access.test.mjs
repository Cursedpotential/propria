import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import vm from 'node:vm';
import {probeEndpoint,probeHealthRows} from '../health.mjs';
import {createServer,summarizeHealthSample,recordHealthHistory,classifySurfaceState} from '../server.mjs';
import {healthCounts,healthChart,healthSummary,publicReachabilityList} from '../public/view.mjs';

const workbench='https://workbench.tilapia-skilift.ts.net';
const family='https://family-court.tilapia-skilift.ts.net';
const sample={generated_at:'2026-10-11T19:00:00.000Z',endpoints:['responding','TCP open','auth required','monitor access denied','HTTP error','unreachable','timeout','not connected','redirect','unexpected content'].map(state=>({state,latency_ms:5}))};

test('summary and bounded history separate access restrictions and missing configuration from errors',()=>{
 const counts=summarizeHealthSample(sample);
 assert.deepEqual([counts.ok,counts.warn,counts.fail,counts.auth_required,counts.monitor_denied,counts.unconfigured,counts.total],[2,2,3,1,1,1,10]);
 assert.equal(counts.ok+counts.warn+counts.fail+counts.auth_required+counts.monitor_denied+counts.unconfigured,counts.total);
 const history=[];recordHealthHistory(sample,history,2);recordHealthHistory(sample,history,2);recordHealthHistory(sample,history,2);
 assert.equal(history.length,2);assert.deepEqual(history[0],counts);
 assert.equal(classifySurfaceState('monitor access denied'),'monitor_denied');
 assert.equal(classifySurfaceState('auth required'),'auth_required');
 assert.equal(classifySurfaceState('HTTP error'),'down');
 assert.equal(classifySurfaceState('not connected'),'unconfigured');
});

test('HTTP status classification remains truthful, sends only existing UA and cancels all bodies',async()=>{
 let canceled=0;
 for(const [status,state] of [[200,'responding'],[401,'auth required'],[403,'monitor access denied'],[500,'HTTP error'],[302,'redirect']]){
  const result=await probeEndpoint({service:'synthetic',url:workbench+'/health',expect:'json'},async(_url,options)=>{
   assert.equal(options.redirect,'manual');assert.ok(options.signal instanceof AbortSignal);
   assert.deepEqual(options.headers,{'User-Agent':'Propria-Live-Health/1'});
   return {status,ok:status===200,headers:new Headers({'content-type':'application/json'}),body:{cancel:async()=>{canceled++;}}};
  });assert.equal(result.state,state);assert.equal(result.http_status,status);
 }
 assert.equal(canceled,5);
 assert.equal((await probeEndpoint({url:workbench},async()=>{throw Error('synthetic network failure');})).state,'unreachable');
 assert.equal((await probeEndpoint({service:'missing'})).state,'not connected');
});

test('one public health probe serves multiple protected surfaces without changing their denied state or URLs',async()=>{
 const rows=[
  {service:'Probata workbench',kind:'Browser surface',url:workbench+'/'},
  {service:'Probata intake',kind:'Browser surface',url:workbench+'/intake'},
  {service:'Schemas A',kind:'Browser surface',url:workbench+'/schemas'},
  {service:'Schemas B',kind:'Browser surface',url:workbench+'/schemas'},
  {service:'Family Law Toolkit',kind:'Browser surface',url:family+'/'},
  {service:'Workbench health',kind:'API',url:workbench+'/health',expect:'json',surface_origin:workbench},
  {service:'Duplicate health registry entry',kind:'API',url:workbench+'/health',expect:'json'},
  {service:'Family health',kind:'API',url:family+'/api/health',expect:'json',surface_origin:family},
 ];
 let active=0,maxActive=0,canceled=0;const calls=new Map();
 const results=await probeHealthRows(rows,async url=>{
  active++;maxActive=Math.max(maxActive,active);calls.set(url.href,(calls.get(url.href)||0)+1);
  await new Promise(resolve=>setTimeout(resolve,2));active--;
  const status=url.pathname.endsWith('/health')?200:403;
  return {status,ok:status===200,headers:new Headers({'content-type':'application/json'}),body:{cancel:async()=>{canceled++;}}};
 });
 assert.ok(maxActive<=4);assert.equal(calls.get(workbench+'/health'),1);assert.equal(calls.get(workbench+'/schemas'),1);assert.equal(calls.size,6);assert.equal(canceled,6);
 for(let i=0;i<5;i++){assert.equal(results[i].state,'monitor access denied');assert.equal(results[i].http_status,403);assert.equal(results[i].endpoint,rows[i].url);assert.equal(results[i].public_reachability.state,'responding');assert.equal(results[i].public_reachability.http_status,200);}
 const counts=summarizeHealthSample({endpoints:results});assert.equal(counts.monitor_denied,5);assert.equal(counts.ok,3);assert.equal(counts.fail,0);
 const markup=publicReachabilityList(results);assert.match(markup,/Monitor restricted \(HTTP 403\)/);assert.match(markup,/Server reachable via public health check/);assert.match(markup,/Protected browser access not verified/);assert.match(markup,/href="https:\/\/workbench.tilapia-skilift.ts.net\/intake"/);
 assert.equal(healthCounts(results).monitor_denied,5);assert.equal(healthSummary(results).healthy,3);
 assert.match(healthChart(results),/Monitor restricted/);
 const capped=await probeHealthRows(Array.from({length:110},(_,i)=>({service:String(i)})));assert.equal(capped.length,100);
});

test('a failed public check is not displayed as server reachable and does not change browser access',async()=>{
 const result=await probeHealthRows([{kind:'Browser surface',url:workbench+'/'},{kind:'API',url:workbench+'/health',surface_origin:workbench}],async url=>({status:url.pathname==='/health'?500:403,ok:false,headers:new Headers(),body:{cancel:async()=>{}}}));
 assert.equal(result[0].state,'monitor access denied');assert.equal(result[0].public_reachability.state,'HTTP error');
 assert.doesNotMatch(publicReachabilityList(result),/Server reachable/);
});

test('summary and surfaces HTTP APIs preserve compatible fields and paired protected status',async()=>{
 const health={...sample,endpoints:[{service:'Probata intake',kind:'Browser surface',endpoint:workbench+'/intake',state:'monitor access denied',http_status:403,public_reachability:{state:'responding',http_status:200,endpoint:workbench+'/health'}},...sample.endpoints]};
 const server=createServer({config:{},healthReader:async()=>health});
 await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
 try{
  const base=`http://127.0.0.1:${server.address().port}`;
  const summary=await (await fetch(base+'/api/health-summary')).json();
  assert.equal(summary.fail,3);assert.equal(summary.monitor_denied,2);assert.equal(summary.auth_required,1);assert.equal(summary.unconfigured,1);assert.equal(summary.total,11);
  assert.equal(summary.ok+summary.warn+summary.fail+summary.monitor_denied+summary.auth_required+summary.unconfigured,summary.total);
  const surfaces=await (await fetch(base+'/api/surfaces-summary')).json();
  assert.equal(surfaces.up,0);assert.equal(surfaces.down,0);assert.equal(surfaces.monitor_denied,1);
  assert.equal(surfaces.down_list[0].url,workbench+'/intake');assert.equal(surfaces.down_list[0].state,'monitor_denied');assert.equal(surfaces.down_list[0].public_reachability.http_status,200);
  assert.match(surfaces.down_list[0].summary,/Monitor restricted/);assert.match(surfaces.down_list[0].summary,/protected access not verified/);
  const detailed=await (await fetch(base+'/api/health-report')).json();assert.equal(detailed.endpoints[0].http_status,403);
 }finally{await new Promise(resolve=>server.close(resolve));}
});

test('full report renderer separates denied, unconfigured and actual failures without greenwashing',async()=>{
 const html=await readFile(new URL('../outputs/live-service-storage-health.html',import.meta.url),'utf8');
 for(const match of html.matchAll(/<script>([\s\S]*?)<\/script>/g))new vm.Script(match[1]);
 const render=html.slice(html.indexOf('function render(){'),html.indexOf('async function refresh(){'));
 const nodes=new Map();const $=id=>{if(!nodes.has(id))nodes.set(id,{value:'',innerHTML:'',textContent:''});return nodes.get(id);};
 const context={current:{...sample,endpoints:[...sample.endpoints,{service:'Protected',kind:'Browser surface',state:'monitor access denied',http_status:403,latency_ms:5,public_reachability:{label:'Server reachable via public health check'}}]},$,e:x=>String(x??''),time:x=>String(x??''),bytes:x=>String(x??''),Date};
 vm.runInNewContext(render+';render();',context);
 assert.match($('stats').innerHTML,/<strong>2<\/strong><span>Monitor restricted/);
 assert.match($('stats').innerHTML,/<strong>3<\/strong><span>Request \/ HTTP errors/);
 assert.match($('stats').innerHTML,/<strong>1<\/strong><span>Unconfigured/);
 assert.match($('endpoints').innerHTML,/Monitor restricted · 403/);
 assert.match($('endpoints').innerHTML,/Server reachable via public health check/);
 assert.match($('endpoints').innerHTML,/Protected browser access not verified/);
 $('filter').value='restricted';vm.runInNewContext(render+';render();',context);
 assert.match($('endpoints').innerHTML,/Monitor restricted/);assert.doesNotMatch($('endpoints').innerHTML,/HTTP error|not connected/);
 $('filter').value='attention';vm.runInNewContext(render+';render();',context);
 assert.match($('endpoints').innerHTML,/HTTP error/);assert.doesNotMatch($('endpoints').innerHTML,/Monitor restricted|not connected/);
});
