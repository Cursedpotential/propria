/** Exercise generated input forms and submission semantics in a detached test DOM.
 * Inputs: synthetic tool schemas. Outputs: assertions. Effects: no browser, network,
 * authentication, or real tool execution; complements backend and hosted checks.
 */
const {JSDOM}=require('jsdom');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const schema={type:'object',properties:{count:{type:'integer'},enabled:{type:'boolean'},payload:{type:'object'},text:{type:'string'}},required:['count']};
const tool={name:'test-read',title:'Read test',description:'Test readable inputs.',inputSchema:schema,annotations:{readOnlyHint:true}};
const html=fs.readFileSync(path.join(__dirname,'static/index.html'),'utf8');
const dom=new JSDOM(html,{url:'https://runner.example/runner/',runScripts:'outside-only'});
const win=dom.window,$=id=>win.document.getElementById(id),calls=[];
win.fetch=async(url,options)=>{
  if(url==='api/session')return{ok:true,json:async()=>({name:'Test person',csrf:'test-csrf'})};
  if(url==='api/tools')return{ok:true,json:async()=>({tools:[tool,{...tool,name:'other-read',title:'Other read'}]})};
  if(url==='api/run'){calls.push(JSON.parse(options.body));return{ok:true,json:async()=>({tool:'test-read',elapsed_ms:3,result:{content:[{type:'text',text:'42'}]}})};}
  throw new Error('Unexpected request '+url);
};
win.eval(fs.readFileSync(path.join(__dirname,'static/app.js'),'utf8'));
const tick=()=>new Promise(resolve=>setImmediate(resolve));
const change=el=>el.dispatchEvent(new win.Event('change',{bubbles:true}));
async function main(){
  for(let i=0;i<8;i++)await tick();
  assert.equal($('tools').children.length,2,'catalog loaded');
  $('search').value='test-readable-no-match';$('search').dispatchEvent(new win.Event('input'));
  assert.match($('catalog-status').textContent,/0 of 2/);
  $('search').value='Read test';$('search').dispatchEvent(new win.Event('input'));
  $('tools').querySelector('button').click();
  $('field-0').value='0';
  const include=$('fields').querySelectorAll('.optional input');include[0].click();
  $('field-1').value='false';
  include[1].click();$('field-2').value='{"a":1}';
  $('json-mode').checked=true;change($('json-mode'));
  assert.deepEqual(JSON.parse($('raw').value),{count:0,enabled:false,payload:{a:1}},'typed values and omission preserved');
  $('json-mode').checked=false;change($('json-mode'));
  $('field-0').value='';
  $('json-mode').checked=true;change($('json-mode'));
  assert.equal($('json-mode').checked,false,'invalid conversion leaves form visible');
  assert.equal($('fields').hidden,false);
  $('field-0').value='7';
  $('refresh').click();for(let i=0;i<8;i++)await tick();
  assert.equal($('field-0').value,'7','refresh preserves prepared inputs');
  $('run-form').dispatchEvent(new win.Event('submit',{cancelable:true}));
  $('run-form').dispatchEvent(new win.Event('submit',{cancelable:true}));
  for(let i=0;i<8;i++)await tick();
  assert.equal(calls.length,1,'double submit sends one request');
  assert.deepEqual(calls[0].arguments,{count:7,enabled:false,payload:{a:1}});
  assert.match($('result').textContent,/42/);
  assert.equal($('download').disabled,false);
  $('json-mode').checked=true;change($('json-mode'));
  $('raw').value='{"count":7,"enabled":null}';
  $('json-mode').checked=false;change($('json-mode'));
  assert.equal($('json-mode').checked,true,'lossy conversion stays in JSON mode');
  assert.equal(JSON.parse($('raw').value).enabled,null);
  console.log('UI checks passed: search, schema fields, typed values, omission, conversion preservation, refresh, single submit, results.');
  dom.window.close();
}
main().catch(e=>{console.error(e);process.exitCode=1;dom.window.close();});
