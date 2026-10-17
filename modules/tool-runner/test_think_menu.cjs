/** Test direct Think menu choices, descriptions and independent controls without contacting any real agent.
 * Inputs: the exact packaged Think catalog and a detached DOM. Outputs: assertions.
 * Side effects: no network, no real tool execution, no cross-conversation state changes.
 */
const {JSDOM}=require('jsdom');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const entries=JSON.parse(fs.readFileSync(path.join(__dirname,'think-library/index.json'),'utf8'));
const schema={type:'object',properties:{depth:{type:'string',enum:['adaptive','micro','focused','full'],default:'adaptive'},scope:{type:'string',enum:['task','conversation'],default:'conversation'},output:{type:'string',enum:['silent','normal','report'],default:'silent'}},additionalProperties:false};
const tools=entries.filter(e=>e.selectable!==false).map(e=>({name:'think--'+e.id,title:e.title,description:e.description,inputSchema:schema,annotations:{readOnlyHint:true},_domain:'think',_domains:['think'],_kind:e.kind}));
tools.push({name:'n8n-list',title:'n8n list',description:'An unrelated automation tool.',inputSchema:{type:'object',properties:{}},annotations:{readOnlyHint:true},_domain:'n8n',_domains:['n8n']});
const html=fs.readFileSync(path.join(__dirname,'static/index.html'),'utf8');
const dom=new JSDOM(html,{url:'https://runner.example/runner/?domain=think&kind=process',runScripts:'outside-only'});
const win=dom.window,$=id=>win.document.getElementById(id);let copied='',runs=0;
Object.defineProperty(win.navigator,'clipboard',{value:{writeText:async text=>{copied=text;}}});
win.fetch=async(url,options)=>{
 if(url==='api/session')return{ok:true,json:async()=>({name:'Test person',csrf:'csrf'})};
 if(url==='api/tools')return{ok:true,json:async()=>({tools,coverage:[]})};
 if(url==='api/run'){runs++;return{ok:true,json:async()=>({result:{kind:'think-activation',host_action_required:true},elapsed_ms:1})};}
 throw Error('Unexpected request: '+url);
};
win.eval(fs.readFileSync(path.join(__dirname,'static/app.js'),'utf8'));
const tick=()=>new Promise(r=>setImmediate(r));
async function main(){
 for(let i=0;i<8;i++)await tick();
 assert.equal($('group').value,'think');assert.equal($('think-kind').value,'process');
 assert.equal($('tools').querySelectorAll('button').length,84,'all individual processes directly visible');
 assert.equal(runs,0,'listing never executes a process');
 const first=$('tools').querySelector('[data-tool="think--thinking-first-principles"]');assert.ok(first);
 assert.match(first.querySelector('.tool-purpose').textContent,/fundamental truths/i,'description visible before click');
 first.click();
 assert.match($('description').textContent,/fundamental truths/i);
 assert.equal($('think-selection').hidden,false);assert.equal($('run').textContent,'Load selected process');
 const fields=[...$('fields').querySelectorAll('select')];assert.equal(fields.length,3,'independent depth scope output');
 fields[0].value='1';fields[1].value='1';fields[2].value='0';
 $('copy-invocation').click();for(let i=0;i<3;i++)await tick();
 assert.match(copied,/thinking-first-principles/);assert.match(copied,/depth: micro/);assert.match(copied,/Scope: conversation/);assert.match(copied,/Output: silent/);
 assert.equal(runs,0,'copying selection does not falsely activate a different conversation');
 $('think-kind').value='workflow';$('think-kind').dispatchEvent(new win.Event('change'));
 assert.ok($('tools').querySelector('[data-tool="think--reflex"]'));
 assert.ok($('tools').querySelector('[data-tool="think--workflow-debug"]'));
 $('think-kind').value='router';$('think-kind').dispatchEvent(new win.Event('change'));
 assert.equal($('tools').querySelectorAll('button').length,1);
 $('group').value='';$('group').dispatchEvent(new win.Event('change'));
 assert.equal($('tools').querySelectorAll('button').length,tools.length,'type filter never hides unrelated native tools in All domains');
 assert.equal($('think-filter').hidden,true);
 console.log('Think UI passed: 84 named processes, visible descriptions, workflow/router choices, independent controls and direct copy.');dom.window.close();
}
main().catch(e=>{console.error(e);process.exitCode=1;dom.window.close();});
