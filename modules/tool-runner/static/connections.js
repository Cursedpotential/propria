/** Human domain marketplace: select connections for ONE package, independently of the full tool runner. */
async function api(path,data){let headers={};if(data){const me=await fetch('api/session').then(r=>r.json());headers={'Content-Type':'application/json','X-CSRF-Token':me.csrf};}const r=await fetch('api/'+path,{method:data?'POST':'GET',headers,body:data?JSON.stringify(data):undefined});const value=await r.json();if(!r.ok)throw new Error(value.error||'Sign in to manage domain packages.');return value;}
const main=document.querySelector('main');
const tabs=document.createElement('nav');tabs.className='surface-tabs';tabs.setAttribute('aria-label','Runner and connection marketplace');
const runnerButton=document.createElement('button');runnerButton.type='button';runnerButton.textContent='All tools';
const connectionsButton=document.createElement('button');connectionsButton.type='button';connectionsButton.textContent='Connections & skills';
const panel=document.createElement('section');panel.className='connection-marketplace';panel.hidden=true;panel.setAttribute('aria-label','Domain plugin marketplace');
tabs.append(runnerButton,connectionsButton);main.before(tabs);main.after(panel);
let choices=new Set();let cards=[];
function message(text){const element=panel.querySelector('[role=status]');if(element)element.textContent=text;}
async function copy(text){try{await navigator.clipboard.writeText(text);message('Copied: '+text);}catch{message('Copy this address: '+text);}}
function showRunner(){panel.hidden=true;main.hidden=false;runnerButton.setAttribute('aria-pressed','true');connectionsButton.setAttribute('aria-pressed','false');}
/** Render every available domain at once; selection changes only the exported package. */
async function showConnections(){
  main.hidden=true;panel.hidden=false;runnerButton.setAttribute('aria-pressed','false');connectionsButton.setAttribute('aria-pressed','true');panel.replaceChildren();
  const heading=document.createElement('h1');heading.textContent='Choose your domain connections';
  const intro=document.createElement('p');intro.textContent='One Propria package. Independent MCP servers, skills and resources. These choices do not remove tools from your human runner or stop any services.';
  const status=document.createElement('p');status.setAttribute('role','status');status.textContent='Loading available connections…';
  panel.append(heading,intro,status);
  try{
    const data=await api('domains');cards=data.domains;choices=new Set(data.selected);
    const shortcut=document.createElement('div');shortcut.className='connection-actions';
    const one=document.createElement('button');one.type='button';one.textContent='Copy single-entry MCP URL';one.onclick=()=>copy(data.single_connection);
    const selectAll=document.createElement('button');selectAll.type='button';selectAll.textContent='Select all domains';selectAll.onclick=()=>{choices=new Set(cards.map(x=>x.id));panel.querySelectorAll('input[data-domain]').forEach(x=>{x.checked=true;});};
    const download=document.createElement('button');download.type='button';download.className='primary';download.textContent='Export one plugin package';
    download.onclick=async()=>{try{await api('domains',{domains:[...choices]});const a=document.createElement('a');a.href='api/package';a.download='propria-domain-plugin.zip';a.click();message('Exported selected domain connections together. The host may still require its OAuth sign-in and installation approval.');}catch(e){message(e.message);}};
    shortcut.append(one,selectAll,download);panel.append(shortcut);
    const grid=document.createElement('div');grid.className='domain-grid';
    for(const item of cards){
      const card=document.createElement('article');card.className='domain-card';
      const label=document.createElement('label');label.className='domain-choice';const checkbox=document.createElement('input');checkbox.type='checkbox';checkbox.dataset.domain=item.id;checkbox.checked=choices.has(item.id);checkbox.onchange=()=>checkbox.checked?choices.add(item.id):choices.delete(item.id);
      const name=document.createElement('strong');name.textContent=item.name;label.append(checkbox,name);
      const count=document.createElement('p');count.textContent=item.native_tools+' native tools / processes · MCP + skill + resources';
      const state=document.createElement('p');state.className='muted';state.textContent='Catalog state: '+item.state+' (not an invocation test)';
      const url=document.createElement('code');url.textContent=item.mcp_url;
      const button=document.createElement('button');button.type='button';button.textContent='Copy domain URL';button.onclick=()=>copy(item.mcp_url);
      card.append(label,count,state,url,button);grid.append(card);
    }
    panel.append(grid);status.textContent='Select the domains to include. The All tools surface remains unfiltered.';
  }catch(e){status.textContent=e.message;}
}
runnerButton.onclick=showRunner;connectionsButton.onclick=showConnections;
showRunner();
if(new URLSearchParams(location.search).get('view')==='connections')showConnections();
