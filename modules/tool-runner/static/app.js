/** Build the runner from live MCP schemas; inputs are user edits, effects are explicit API calls. */
const $ = id => document.getElementById(id);
let tools = [], selected = null, csrf = '', fields = [], lastResult = null, running = false;
const groups = ['family-court','coolify-write','cloudflare-docs','context7','agno-docs','n8n-docs','atomic-tools','advocatio','memsearch','octopoda','docstore','n8n','mem','docs','surreal'];
const groupOf = tool => groups.find(g => tool.name.replaceAll('_','-').startsWith(g)) || tool.name.split(/[-_]/)[0];
const labelOf = tool => tool.title || tool.name.replaceAll(/[-_]/g,' ');

/** Fetch this origin's runner API; output parsed JSON, no credentials leave the server. */
async function api(path, data) {
  const response = await fetch(`api/${path}`, {method:data?'POST':'GET', headers:data?{'Content-Type':'application/json','X-CSRF-Token':csrf}:{}, body:data?JSON.stringify(data):undefined});
  const result = await response.json();
  if(!response.ok) { if(response.status===401) $('welcome-signin').hidden=false; throw new Error(result.error || `Request failed (${response.status})`); }
  return result;
}
function error(message) { $('error').hidden=!message; $('error').textContent=message; }

/** Render filtered tool choices as text nodes; catalog descriptions cannot inject markup. */
function renderList() {
  const query=$('search').value.toLowerCase().trim(), group=$('group').value;
  const matches=tools.filter(t=>(!group||groupOf(t)===group)&&`${t.name} ${t.title||''} ${t.description||''}`.toLowerCase().includes(query));
  $('tools').replaceChildren();
  $('catalog-status').textContent=`${matches.length} of ${tools.length} tools`;
  for(const tool of matches) {
    const button=document.createElement('button'); button.type='button'; button.className='tool'; button.disabled=running;
    button.setAttribute('aria-current',String(selected?.name===tool.name));
    const title=document.createElement('strong'), subtitle=document.createElement('small');
    title.textContent=labelOf(tool); subtitle.textContent=groupOf(tool);
    button.append(title,subtitle); button.onclick=()=>choose(tool); $('tools').append(button);
  }
  if(!matches.length) $('tools').textContent='No tools match. Try a different search.';
}

/** Create one schema field; complex unions, arrays and objects use typed JSON editors. */
function makeField(name,schema,required) {
  const wrap=document.createElement('div'); wrap.className='field';
  const heading=document.createElement('div'); heading.className='field-title';
  const label=document.createElement('label'); label.textContent=`${schema.title||name}${required?' *':''}`;
  let input, kind='string';
  if(schema.enum) { input=document.createElement('select'); kind='enum'; schema.enum.forEach((v,i)=>{const o=document.createElement('option');o.value=String(i);o.textContent=JSON.stringify(v);input.append(o);}); }
  else if(schema.type==='boolean') {input=document.createElement('select');kind='boolean';for(const v of ['false','true']){const o=document.createElement('option');o.value=v;o.textContent=v;input.append(o);}}
  else if(['number','integer'].includes(schema.type)){input=document.createElement('input');input.type='number';input.step=schema.type==='integer'?'1':'any';kind='number';if(schema.minimum!==undefined)input.min=schema.minimum;if(schema.maximum!==undefined)input.max=schema.maximum;}
  else if(schema.type==='string'&&!schema.anyOf&&!schema.oneOf&&!schema.$ref){input=document.createElement('input');input.type=schema.format==='password'?'password':'text';if(schema.minLength!==undefined)input.minLength=schema.minLength;if(schema.maxLength!==undefined)input.maxLength=schema.maxLength;}
  else {input=document.createElement('textarea');input.rows=4;input.spellcheck=false;kind='json';input.placeholder=schema.type==='array'?'[]':schema.type==='object'?'{}':'Enter a JSON value';}
  input.id=`field-${fields.length}`; label.htmlFor=input.id; input.required=required; heading.append(label);
  const include=document.createElement('input');include.type='checkbox';include.checked=required;
  if(!required){const inclusion=document.createElement('label');inclusion.className='optional';inclusion.append(include,document.createTextNode('Include'));heading.append(inclusion);input.disabled=true;include.onchange=()=>{input.disabled=!include.checked;};}
  if(schema.default!==undefined) {include.checked=true;input.disabled=false;input.value=kind==='enum'?String(schema.enum.findIndex(v=>JSON.stringify(v)===JSON.stringify(schema.default))):kind==='json'?JSON.stringify(schema.default,null,2):String(schema.default);}
  wrap.append(heading,input);
  if(schema.description){const help=document.createElement('p');help.className='help';help.textContent=schema.description;help.id=input.id+'-help';input.setAttribute('aria-describedby',help.id);wrap.append(help);}
  fields.push({name,schema,input,include,kind});return wrap;
}

/** Select a tool and generate fields without running it or persisting its arguments. */
function choose(tool) {
  if(running)return;
  selected=tool; fields=[]; lastResult=null; error(''); $('welcome').hidden=true; $('editor').hidden=false;
  $('service').textContent=groupOf(tool); $('title').textContent=labelOf(tool); $('tool-name').textContent=tool.name;
  $('description').textContent=tool.description||'No description supplied by this service.';
  const readOnly=tool.annotations?.readOnlyHint===true;
  $('effect').textContent=readOnly?'Read-only':'May make changes'; $('review').hidden=readOnly; $('confirm').checked=false;
  $('schema').textContent=JSON.stringify(tool.inputSchema,null,2); $('fields').replaceChildren();
  const schema=tool.inputSchema||{}, properties=schema.properties||{};
  for(const [name,field] of Object.entries(properties)) $('fields').append(makeField(name,field,(schema.required||[]).includes(name)));
  if(!Object.keys(properties).length) $('fields').textContent='This tool has no named inputs. Use JSON for additional or structured inputs.';
  $('json-mode').checked=false; $('raw').value='{}'; setMode(false);
  if(schema.oneOf||schema.anyOf||schema.allOf||schema.$ref||schema.type!=='object'){$('json-mode').checked=true;setMode(true);}
  $('result').textContent='';$('result-summary').textContent='The result will appear here after you run the tool.';$('download').disabled=true;$('run-status').textContent='';
  renderList();
}

/** Convert included field values to JSON, preserving false, zero, empty strings and null. */
function formValues() {
  const value=Object.create(null);
  for(const f of fields) if(f.include.checked) {
    const raw=f.input.value;
    if(f.kind==='json'){try{value[f.name]=JSON.parse(raw);}catch{throw new Error(`Enter valid JSON for ${f.name}.`);}}
    else if(f.kind==='number'){if(raw.trim()===''||!Number.isFinite(Number(raw)))throw new Error(`Enter a number for ${f.name}.`);value[f.name]=Number(raw);}
    else if(f.kind==='boolean')value[f.name]=raw==='true';
    else if(f.kind==='enum')value[f.name]=f.schema.enum[Number(raw)];
    else value[f.name]=raw;
  }
  return value;
}
function setMode(raw) {
  $('fields').hidden=raw;$('raw').hidden=!raw;$('raw-label').hidden=!raw;
  for(const f of fields)f.input.disabled=raw||!f.include.checked;
}
$('json-mode').onchange=()=>{
  const requestedRaw=$('json-mode').checked;
  try {
    if($('json-mode').checked){$('raw').value=JSON.stringify(formValues(),null,2);setMode(true);}
    else {const value=JSON.parse($('raw').value);if(!value||Array.isArray(value)||typeof value!=='object')throw new Error('Inputs must be an object.');
      if(Object.keys(value).some(k=>!fields.some(f=>f.name===k)))throw new Error('Additional fields are present. Keep JSON mode to preserve them.');
      for(const f of fields)if(Object.hasOwn(value,f.name)){
        const v=value[f.name];
        if((f.kind==='string'&&typeof v!=='string')||(f.kind==='number'&&typeof v!=='number')||(f.kind==='boolean'&&typeof v!=='boolean')||(f.kind==='enum'&&!f.schema.enum.some(e=>JSON.stringify(e)===JSON.stringify(v))))throw new Error(`Keep JSON mode to preserve the value of ${f.name}.`);
      }
      for(const f of fields){f.include.checked=Object.hasOwn(value,f.name)|| (selected.inputSchema.required||[]).includes(f.name);if(Object.hasOwn(value,f.name))f.input.value=f.kind==='enum'?String(f.schema.enum.findIndex(v=>JSON.stringify(v)===JSON.stringify(value[f.name]))):f.kind==='json'?JSON.stringify(value[f.name],null,2):String(value[f.name]);}setMode(false);}
    error('');
  }catch(e){$('json-mode').checked=!requestedRaw;setMode(!requestedRaw);error(e.message);}
};

/** Refresh tool definitions from the user's gateway; no tools execute on discovery. */
async function refresh(){
  $('refresh').disabled=true;error('');$('catalog-status').textContent='Loading tools…';
  try{const result=await api('tools');tools=result.tools.sort((a,b)=>a.name.localeCompare(b.name));const previous=$('group').value;
    $('group').replaceChildren(new Option('All services',''));for(const g of [...new Set(tools.map(groupOf))].sort())$('group').add(new Option(g,g));$('group').value=previous;
    if(selected){const updated=tools.find(t=>t.name===selected.name);if(updated){
      if(JSON.stringify(updated.inputSchema)!==JSON.stringify(selected.inputSchema)){
        error('This tool’s inputs changed. Your edits are preserved; review the updated definition before running.');
        $('schema').textContent=JSON.stringify(updated.inputSchema,null,2);
      }
      selected=updated;$('confirm').checked=false;$('review').hidden=updated.annotations?.readOnlyHint===true;
    }else{selected=null;$('editor').hidden=true;$('welcome').hidden=false;}}renderList();
  }catch(e){$('catalog-status').textContent='Tools could not be loaded.';error(e.message);}finally{$('refresh').disabled=false;}
}

/** Submit once after explicit review; show server errors and unknown outcomes without retry. */
$('run-form').onsubmit=async event=>{
  event.preventDefault();if(!selected||running)return;error('');
  try{
    const args=$('json-mode').checked?JSON.parse($('raw').value):formValues();
    if(selected.annotations?.readOnlyHint!==true&&!$('confirm').checked)throw new Error('Review the inputs and check the confirmation before running.');
    running=true;$('run').disabled=true;$('refresh').disabled=true;renderList();$('run-status').textContent='Running…';$('result-summary').textContent='Waiting for the service…';$('result').textContent='';$('download').disabled=true;lastResult=null;
    const data=await api('run',{name:selected.name,arguments:args,request_id:crypto.randomUUID(),confirmed:$('confirm').checked});lastResult=data;
    $('result').textContent=JSON.stringify(data.result,null,2);$('result-summary').textContent=`${data.result.isError?'Tool returned an error':'Completed'} · ${(data.elapsed_ms/1000).toFixed(2)} seconds`;$('download').disabled=false;$('run-status').textContent=data.result.isError?'Tool error':'Finished';
  }catch(e){error(e.message);$('run-status').textContent='Not completed';$('result-summary').textContent='No successful result received.';}
  finally{running=false;$('run').disabled=false;$('refresh').disabled=false;$('confirm').checked=false;renderList();}
};
$('download').onclick=()=>{if(!lastResult)return;const url=URL.createObjectURL(new Blob([JSON.stringify(lastResult,null,2)],{type:'application/json'}));const a=document.createElement('a');a.href=url;a.download=`propria-${lastResult.tool.replaceAll(/[^a-z0-9_-]/gi,'_')}-${Date.now()}.json`;a.click();setTimeout(()=>URL.revokeObjectURL(url),1000);};
$('search').oninput=renderList;$('group').onchange=renderList;$('refresh').onclick=refresh;
async function start(){try{const me=await api('session');csrf=me.csrf;const name=document.createElement('span');name.textContent=me.name;const logout=document.createElement('button');logout.textContent='Sign out';logout.onclick=async()=>{try{await api('logout',{});location.reload();}catch(e){error(e.message);}};$('account').replaceChildren(name,logout);$('welcome-signin').hidden=true;await refresh();}catch{$('catalog-status').textContent='Sign in to load your tools.';}}
start();
