/* Shared usage overrides: owner-reported capacity, reset time and fallback route. */
(function(){
  let mounted=false;
  function element(tag,text){const node=document.createElement(tag);if(text!=null)node.textContent=text;return node;}
  function select(label,choices){const wrapper=element('label',label+' '),input=element('select');input.setAttribute('aria-label',label);choices.forEach(([value,text])=>{const o=element('option',text);o.value=value;input.append(o);});wrapper.append(input);return {wrapper,input};}
  async function mount(){
    if(mounted)return;
    const target=document.querySelector('#portal-surface-actions')||document.querySelector('#panel-board .toolbar');if(!target)return;
    mounted=true;
    const panel=element('details');panel.className='portal-usage-settings';panel.append(element('summary','Usage limits & rerouting'));
    const summary=element('p','Loading saved overrides…');summary.setAttribute('role','status');panel.append(summary);
    const provider=select('Provider',[['claude','Claude'],['codex','Codex'],['portkey','Portkey']]);
    const state=select('Usage',[['low','Low'],['exhausted','Used up'],['available','Clear override']]);
    const untilLabel=element('label','Limit until (your local time) '),until=element('input');until.type='datetime-local';until.required=true;until.setAttribute('aria-label','Usage reset date and time');untilLabel.append(until);
    const fallback=select('Until then',[['','Hold jobs'],['claude','Use Claude'],['codex','Use Codex'],['portkey','Use a Portkey provider/account']]);
    const accountLabel=element('label','Portkey account/config name '),account=element('input');account.type='text';account.maxLength=100;account.placeholder='Named route, not a secret key';account.setAttribute('aria-label','Portkey account or config name');accountLabel.append(account);accountLabel.hidden=true;
    fallback.input.onchange=()=>{accountLabel.hidden=fallback.input.value!=='portkey';};
    state.input.onchange=()=>{until.required=state.input.value!=='available';untilLabel.hidden=state.input.value==='available';};
    const save=element('button','Save usage override');save.type='button';
    panel.append(provider.wrapper,state.wrapper,untilLabel,fallback.wrapper,accountLabel,save);
    async function read(){try{const response=await fetch('/progress/api/provider-limits',{cache:'no-store'});if(!response.ok)throw Error();const data=await response.json();const active=data.limits.filter(row=>row.state!=='available'&&Date.parse(row.limited_until)>Date.now());summary.textContent=active.length?active.map(row=>row.provider+': '+row.state+' until '+new Date(row.limited_until).toLocaleString()+' → '+(row.fallback_provider?(row.fallback_provider+(row.fallback_account?' / '+row.fallback_account:'')):'hold jobs')).join(' · '):'No active usage overrides.';}catch{summary.textContent='Saved usage settings unavailable.';}}
    save.onclick=async()=>{if(state.input.value!=='available'&&(!until.value||!until.reportValidity()))return;save.disabled=true;try{const response=await fetch('/progress/api/provider-limits',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({provider:provider.input.value,state:state.input.value,limited_until:until.value?new Date(until.value).toISOString():null,fallback_provider:fallback.input.value,fallback_account:account.value.trim()})});const data=await response.json();if(!response.ok)throw Error(data.message);await read();}catch(error){summary.textContent=error.message;}finally{save.disabled=false;}};
    // Keep the panel outside the periodically replaced list of attention rows.
    target.parentElement.insertBefore(panel,target);
    await read();
  }
  const boot=setInterval(()=>{mount();if(mounted)clearInterval(boot);},500);
})();
