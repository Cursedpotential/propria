// Byline: Codex · 2026-09-12 — project board and independently refreshed widgets.
// Extended by: Claude Code · Sonnet · 2026-09-14 — KPI strip, status/lane-swimlane filters, and a
// second surfaces/observations tracker view. Same read-only /api/board + /api/health-report
// contract; no write paths added.
import { escapeHtml as e, timeTag, taskCard, observationCard, bucket, safeLink, age, boardTasks, filterRows, taskDistribution, healthChart, publicReachabilityList, storageCard, kpiCounts, laneTotalsWidget, healthSummary, renderBoardMarkup, trackerRows, filterTrackerRows, trackerTableRows, liveWorkCard, storageHealthCard } from './view.mjs';
import {installTaskActions} from './task-actions.mjs';
import './provider-settings.js';
let initialLane=new URLSearchParams(location.search).get('lane');
const updateActions=installTaskActions({getTasks:()=>current?boardTasks(current):[]});
const $=id=>document.getElementById(id);
let current=null,lastSuccess=null,fetching=false,health=null,healthFetching=false,liveWork=null,storageHealth=null,liveFetching=false;
const filters=()=>({lane:$('lane').value,owner:$('owner').value,search:$('search').value,status:$('status').value});
const trackerFilters=()=>({lane:$('tracker-lane').value,kind:$('tracker-kind').value,search:$('tracker-search').value});

/** Render task KPIs and separately scoped endpoint observations.
 * Inputs: documented tasks, cached health and live work. Outputs: separately scoped DOM cards.
 * Effects: updates the board DOM. Choose for summary rendering, not protected journey proof.
 */
function renderKpis(tasks) {
  const counts=kpiCounts(tasks);
  const hs=health?healthSummary(health.endpoints):null;
  const cards=[
    {label:'Backlog / ready',value:counts.backlog,cls:'neutral'},
    {label:'In progress',value:counts.inProgress,cls:'information'},
    {label:'Blocked',value:counts.blocked,cls:'caution'},
    {label:'Done',value:counts.done,cls:'positive'},
    {label:'Overdue',value:counts.overdue,cls:counts.overdue?'destructive':'neutral'},
    {label:'Due within 7 days',value:counts.dueSoon,cls:counts.dueSoon?'caution':'neutral'},
  ];
  $('kpis').innerHTML=cards.map(c=>`<div class="kpi-card kpi-${c.cls}"><strong>${c.value}</strong><span>${e(c.label)}</span></div>`).join('')
    +`<div class="kpi-card kpi-health"><strong>${hs?`${hs.healthy}/${hs.total}`:'—'}</strong><span>Endpoints responding${hs?'':' (probe pending)'}</span></div>`;
  $('kpi-freshness').textContent=current?`Documented backlog · read ${age(current.fetched_at)} · status labels are source reports, not live execution`:'Documented backlog read pending';
  renderLiveWork();
}

/** Render observed execution status even when the manual board is unavailable.
 * Inputs: cached live-work snapshot. Outputs: dedicated live widget. Effects: DOM updates only.
 * Choose independently of documented backlog rendering.
 */
function renderLiveWork(){
 let liveNode=$('live-work-summary');
 if(!liveNode){liveNode=document.createElement('section');liveNode.id='live-work-summary';$('kpis').after(liveNode);}
 liveNode.innerHTML=liveWorkCard(liveWork);
}

/** Render documented planning tasks and source reports with their reported statuses.
 * Inputs: current manual board snapshot and filters. Outputs: backlog cards and trackers.
 * Effects: DOM updates only. Choose for planning; live execution uses its independent widget.
 */
function render() {
 if(!current)return;
 const allTasks=boardTasks(current);
 renderKpis(allTasks);
 const groupByLane=$('group-by-lane').checked;
 const rows=filterRows(allTasks,filters());
 const observations=filterRows(current.observations,filters());
 $('counts').textContent=`${rows.length} of ${allTasks.length} documented backlog entries in view${current.truncated?' · bounded snapshot':''}`;
 $('board').innerHTML=renderBoardMarkup(rows,groupByLane);
 const other=rows.filter(task=>bucket(task.status)==='other');
 $('other-states').innerHTML=other.length&&!groupByLane?`<details class="other"><summary>Other source states (${other.length})</summary>${other.map(taskCard).join('')}</details>`:'';
 $('distribution').innerHTML='<p class="widget-note">'+timeTag(current.fetched_at,'Source read')+'</p>'+taskDistribution(rows);
 $('lane-totals').innerHTML=laneTotalsWidget(rows);
 for(const [target,kinds] of [['workers',['worker']],['catalog',['catalog']],['migration',['migration','file_migration']]]) {
  const subset=observations.filter(row=>kinds.includes(row.kind));
  $(target).innerHTML=(subset.length?subset.map(observationCard).join(''):'<p class="empty">No matching source report.</p>');
 }
 const surfaces=Array.isArray(current.surfaces)?current.surfaces:[];
 $('surface-count').textContent=surfaces.length;
 $('surfaces').innerHTML=surfaces.map(surface=>{
  const report=current.observations.find(row=>row.kind==='surface'&&row.lane===surface.lane&&row.title===surface.name);
  const item=report?{...surface,status:report.status||surface.status,updated_at:report.updated_at,checked_at:report.checked_at}:surface;
  const link=safeLink(item.url);
  return `<div class="surface">${link?`<a href="${e(link)}" target="_blank" rel="noopener noreferrer">${e(item.name)}<span aria-hidden="true">↗</span></a>`:`<span>${e(item.name)}</span>`}<small>${e(link?(item.status||'Status not reported'):'Link pending')}</small>${timeTag(item.updated_at,'Updated')}${timeTag(item.checked_at,'Checked')}</div>`;
 }).join('');
 renderTracker();
 updateActions();
}

function renderTracker() {
 if(!current)return;
 const all=trackerRows(current);
 const rows=filterTrackerRows(all,trackerFilters());
 setOptions('tracker-lane',all.map(row=>row.lane),'All lanes');
 setOptions('tracker-kind',all.map(row=>row.kind),'All kinds');
 $('tracker-counts').textContent=`${rows.length} of ${all.length} tracked items`;
 $('tracker-body').innerHTML=trackerTableRows(rows);
}

function setOptions(id, values, label) {
 const selected=$(id).value;
 const options=[...new Set(values.filter(Boolean))].sort();
 $(id).innerHTML=`<option value="">${label}</option>`+options.map(value=>`<option value="${e(value)}">${e(value)}</option>`).join('');
 $(id).value=options.includes(selected)?selected:'';
}
async function refreshBoard() {
 if(fetching)return;fetching=true;
 try {
  const response=await fetch('./api/board',{cache:'no-store',signal:AbortSignal.timeout(12000)});
  const result=await response.json();
  if(!Array.isArray(result.tasks)||!Array.isArray(result.observations))throw Error('Invalid report');
  if(response.ok&&result.status==='connected'){current=result;lastSuccess=result.fetched_at;}
  else if(!current)current=result;
  const tasks=boardTasks(current);
  setOptions('lane',[...tasks,...current.observations].map(row=>row.lane),'All lanes');
  if(initialLane){$('lane').value=initialLane;initialLane=null;}
  setOptions('owner',[...tasks,...current.observations].map(row=>row.owner||'Unassigned'),'All owners');
  $('connection').className='connection '+(response.ok&&result.status==='connected'?'connected':'stale');
  $('connection').textContent=response.ok&&result.status==='connected'?`Connected · read ${age(result.fetched_at)}`:`Unavailable${lastSuccess?' · last read '+age(lastSuccess):''}`;
  render();
 }catch{
  $('connection').className='connection stale';
  $('connection').textContent=`Connection failed${lastSuccess?' · last read '+age(lastSuccess):''}`;
 }finally{fetching=false;}
}
/** Render endpoint access and public reachability as separate observations.
 * Inputs: cached health report and authenticated storage snapshot. Outputs: separate health/storage DOM.
 * Effects: updates board widgets. Choose after a health refresh without changing browser status.
 */
function renderHealth(){
 if(!health)return;
 $('health').innerHTML=healthChart(health.endpoints)+publicReachabilityList(health.endpoints);
 const exceptions=health.endpoints.filter(row=>!['responding','TCP open'].includes(row.state)).slice(0,4);
 if(exceptions.length)$('health').innerHTML+='<ul class="attention-list">'+exceptions.map(row=>`<li><span>${e(row.service)}<br><small>${e(row.kind)}</small></span><span>${e(row.state)}</span></li>`).join('')+'</ul>';
 $('storage').innerHTML=storageHealthCard(storageHealth)+storageCard(health.storage);
 if(current)renderKpis(boardTasks(current));
}
/** Refresh endpoint and host-disk observations independently of provider connection checks.
 * Inputs: none. Outputs: cached health widget. Effects: same-origin GET and DOM updates.
 * Choose for health observations; storageHealthCard keeps its independent authenticated result.
 */
async function refreshHealth(){
 if(healthFetching)return;healthFetching=true;
 try{
  const response=await fetch('./api/health-report',{cache:'no-store',signal:AbortSignal.timeout(45000)});
  const result=await response.json();
  if(!response.ok||!Array.isArray(result.endpoints))throw Error('Invalid health report');
  health=result;
  $('health-connection').className='widget-note';
  $('health-connection').textContent=`Checked ${age(result.generated_at)}`;
  renderHealth();
 }catch{
  $('health-connection').className='widget-note stale';
  $('health-connection').textContent=health?`Stale · ${age(health.generated_at)}`:'Unavailable';
  if(health)renderHealth();
  else $('storage').innerHTML=storageHealthCard(storageHealth)+'<p class="empty">Host disk telemetry unavailable.</p>';
 }finally{healthFetching=false;}
}
/** Refresh independent live execution and authenticated storage snapshots without overlapping reads.
 * Inputs: none. Outputs: current metadata widgets. Effects: same-origin read-only GETs and DOM updates.
 * Choose alongside board refresh; manual planning status never substitutes for provider state.
 */
async function refreshLive(){
 if(liveFetching)return;liveFetching=true;
 try{
  const read=async path=>{try{const response=await fetch(path,{cache:'no-store',signal:AbortSignal.timeout(45000)});if(!response.ok)return null;return await response.json();}catch{return null;}};
  [liveWork,storageHealth]=await Promise.all([read('./api/live-work'),read('./api/storage-health')]);
  renderLiveWork();
  if(current)renderKpis(boardTasks(current));
  if(health)renderHealth();else $('storage').innerHTML=storageHealthCard(storageHealth);
 }finally{liveFetching=false;}
}
/** Refresh the independent board, health and live-provider widgets together.
 * Inputs: refresh action. Outputs: current available snapshots. Effects: bounded GETs and DOM updates.
 * Choose for manual refresh without conflating their freshness or availability.
 */
async function refreshAll(){ $('refresh').disabled=true;try{await Promise.all([refreshBoard(),refreshHealth(),refreshLive()]);}finally{$('refresh').disabled=false;} }
$('refresh').addEventListener('click',refreshAll);
for(const id of ['lane','owner','status'])$(id).addEventListener('change',render);
$('search').addEventListener('input',render);
$('group-by-lane').addEventListener('change',render);
for(const id of ['tracker-lane','tracker-kind'])$(id).addEventListener('change',renderTracker);
$('tracker-search').addEventListener('input',renderTracker);
for(const button of document.querySelectorAll('[data-widget]'))button.addEventListener('click',()=>{
 const expanded=$(button.dataset.widget).classList.toggle('is-expanded');
 button.setAttribute('aria-expanded',String(expanded));button.textContent=expanded?'Compact':'Expand';
});

// Tabs: Board / Trackers & widgets. Plain button-based tablist with left/right arrow support.
const tabs=[['tab-board','panel-board'],['tab-trackers','panel-trackers']];
function selectTab(tabId){
 for(const [tab,panel] of tabs){
  const active=tab===tabId;
  $(tab).classList.toggle('is-active',active);
  $(tab).setAttribute('aria-selected',String(active));
  $(tab).tabIndex=active?0:-1;
  $(panel).hidden=!active;
 }
}
for(const [tab] of tabs){
 $(tab).addEventListener('click',()=>selectTab(tab));
 $(tab).addEventListener('keydown',ev=>{
  if(ev.key!=='ArrowRight'&&ev.key!=='ArrowLeft')return;
  ev.preventDefault();
  const idx=tabs.findIndex(([t])=>t===tab);
  const next=tabs[(idx+(ev.key==='ArrowRight'?1:tabs.length-1))%tabs.length][0];
  selectTab(next);$(next).focus();
 });
}

if(window.matchMedia('(max-width:760px)').matches)document.querySelector('.surface-directory').open=false;
refreshAll();setInterval(refreshBoard,15000);setInterval(refreshHealth,30000);setInterval(refreshLive,30000);
