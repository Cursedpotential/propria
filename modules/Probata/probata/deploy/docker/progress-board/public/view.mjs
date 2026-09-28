export const escapeHtml = value => String(value ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);
export const safeLink = value => { try { const u = new URL(value); return ['http:', 'https:'].includes(u.protocol) && !u.username && !u.password ? u.href : null; } catch { return null; } };
export function age(value, now = Date.now()) { const parsed = Date.parse(value); if (!Number.isFinite(parsed)) return 'unknown'; const seconds = Math.round((now - parsed) / 1000); if (seconds < -30) return 'future timestamp'; if (seconds < 60) return 'just now'; if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`; if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`; return `${Math.floor(seconds / 86400)}d ago`; }
export function timeTag(value, label) { const parsed = Date.parse(value); const valid = Number.isFinite(parsed); const exact = valid ? new Intl.DateTimeFormat('en-US', { timeZone: 'America/New_York', dateStyle: 'medium', timeStyle: 'long' }).format(parsed) : 'Unknown'; return `<span class="timestamp" title="${escapeHtml(exact)}">${escapeHtml(label)} <time${valid ? ` datetime="${escapeHtml(value)}"` : ''}>${escapeHtml(age(value))}</time></span>`; }
// Byline: Codex · 2026-09-12 — honest source-state projection and accessible snapshot charts.
// Extended by: Claude Code · Sonnet · 2026-09-14 — project-management board: KPI strip, priority/due,
// lane swimlanes, status filter, tracker table and health summary. Original honesty rules preserved:
// no fabricated completion, no color-only status, every count states its source and denominator.
export const bucket = status => status === 'done' ? 'done' : status === 'blocked' ? 'blocked' : ['in_progress', 'in_flight', 'running'].includes(status) ? 'flight' : ['open', 'upcoming', 'planned'].includes(status) ? 'upcoming' : 'other';
export const BUCKET_LABELS = { upcoming: 'Backlog / ready', flight: 'In progress', blocked: 'Blocked', done: 'Done', other: 'Other reported states' };
export const observationTask = row => ({ ...row, provenance: 'Lane observation', source_doc: row.id, updated_at: row.updated_at, status: row.status || 'unknown' });
export const columns = [['upcoming','Backlog / ready'],['flight','In progress'],['blocked','Blocked'],['done','Done']];
export function boardTasks(snapshot) {
  const reported = (snapshot.observations || []).filter(row => row.kind === 'task').map(observationTask);
  const next = reported.flatMap(task => (Array.isArray(task.upcoming) ? task.upcoming : []).map((title,index) => ({ ...task, id: `${task.id}:upcoming:${index}`, title, detail: `Reported next step for: ${task.title}`, status:'open', provenance:'Reported upcoming step', updated_at:null, checked_at:null })));
  return [...(snapshot.tasks || []), ...reported, ...next];
}
export function filterRows(rows, {lane='',owner='',search='',status=''} = {}) {
  const needle=search.toLowerCase();
  return rows.filter(row => (!lane || row.lane === lane) && (!owner || (row.owner || 'Unassigned') === owner) && (!status || bucket(row.status) === status) && (!needle || [row.title,row.detail,row.owner,row.id,row.source_doc,row.lane].join(' ').toLowerCase().includes(needle)));
}
export function taskDistribution(rows) {
  const groups=[...columns,['other','Other states']].map(([key,label])=>({key,label,count:rows.filter(row=>bucket(row.status)===key).length}));
  if(!rows.length)return '<p class="empty">No matching tasks reported.</p>';
  return groups.map(({key,label,count})=>`<div class="chart-row ${key}"><span>${label}</span><meter min="0" max="${rows.length}" value="${count}" aria-label="${label}: ${count} of ${rows.length} reported tasks"></meter><strong>${count}</strong></div>`).join('');
}
export function healthCounts(rows) {
  const counts={reachable:0,gated:0,attention:0,unknown:0};
  for(const row of rows) { if(['responding','TCP open'].includes(row.state))counts.reachable++;else if(['auth required','redirect'].includes(row.state))counts.gated++;else if(['unreachable','timeout','HTTP error','unexpected content'].includes(row.state))counts.attention++;else counts.unknown++; }
  return counts;
}
export function healthChart(rows) {
  if(!rows.length)return '<p class="empty">No endpoint observations are available.</p>';
  const counts=healthCounts(rows);let offset=0;
  const rings=Object.entries(counts).map(([key,count])=>{const size=count/rows.length*100;const circle=key==='unknown'?'':`<circle class="ring-${key}" cx="50" cy="50" r="36" pathLength="100" stroke-dasharray="${size} ${100-size}" stroke-dashoffset="${-offset}" transform="rotate(-90 50 50)"></circle>`;offset+=size;return circle;}).join('');
  const labels={reachable:'Responding / TCP',gated:'Auth / redirect',attention:'Needs attention',unknown:'Not connected'};
  return `<div class="health-plot"><svg viewBox="0 0 100 100" role="img" aria-label="Endpoint snapshot: ${counts.reachable} responding or TCP open, ${counts.gated} require authentication or redirect, ${counts.attention} need attention, ${counts.unknown} not connected"><circle class="ring-base" cx="50" cy="50" r="36"></circle>${rings}</svg><div class="health-legend">${Object.entries(counts).map(([key,count])=>`<div><span class="legend-key ${key}">${labels[key]}</span><strong>${count}</strong></div>`).join('')}</div></div><p class="widget-note">${rows.length} endpoints</p>`;
}
export function bytes(value) { if(typeof value!=='number'||!Number.isFinite(value)||value<0)return 'Unknown';if(value===0)return '0 B';const index=Math.min(4,Math.floor(Math.log(value)/Math.log(1024)));return `${(value/1024**index).toLocaleString('en-US',{maximumFractionDigits:1})} ${['B','KiB','MiB','GiB','TiB'][index]}`; }
export function storageCard(storage, now=Date.now()) {
  const rows=Array.isArray(storage?.filesystems)?storage.filesystems:[];
  const services=Array.isArray(storage?.services)?storage.services:[];
  const parsed=Date.parse(storage?.observed_at);const stale=!Number.isFinite(parsed)||now-parsed>120000;
  const heading=`<p class="widget-note${stale?' storage-warning':''}">${escapeHtml(storage?.status || 'Storage telemetry unavailable')} ${timeTag(storage?.observed_at,'Observed')}</p>`;
  const files=rows.map(row=>{const capacity=row.capacity_bytes,used=row.used_bytes;const valid=typeof capacity==='number'&&capacity>0&&Number.isFinite(capacity)&&typeof used==='number'&&Number.isFinite(used)&&used>=0&&used<=capacity;const percent=valid?used/capacity*100:null;return `<article class="storage-item"><h4>${escapeHtml(row.host || 'Unknown host')} / ${escapeHtml(row.mount || 'Unknown mount')}</h4>${valid?`<div class="chart-row ${percent>=90?'blocked':'done'}"><meter min="0" max="100" value="${percent}" aria-label="${escapeHtml(row.mount || 'Filesystem')} ${percent.toFixed(1)} percent used"></meter><strong>${percent.toFixed(1)}%</strong></div>`:'<p class="empty">Usage ratio unknown</p>'}<div class="capacity-values"><span>Capacity <strong>${bytes(capacity)}</strong></span><span>Used <strong>${bytes(used)}</strong></span><span>Free <strong>${bytes(row.free_bytes)}</strong></span></div>${timeTag(row.observed_at || storage.observed_at,'Measured')}<p class="widget-note">${escapeHtml(row.scope || 'Host filesystem; not a per-service allocation')}</p></article>`;}).join('');
  const serviceRows=services.map(row=>`<article class="storage-item"><h4>${escapeHtml(row.service || 'Unknown service')}</h4><p class="widget-note">${escapeHtml(row.volume || row.mount || 'Volume not reported')} · ${escapeHtml(row.scope || 'Service usage')}</p><div class="capacity-values"><span>Used <strong>${bytes(row.used_bytes)}</strong></span><span>Capacity <strong>${bytes(row.capacity_bytes)}</strong></span></div>${timeTag(row.observed_at || storage.observed_at,'Measured')}</article>`).join('');
  return heading+(files||'<p class="empty">No filesystem capacity measurement.</p>')+(serviceRows||'<p class="empty">No per-service storage reported.</p>');
}
function priorityBadge(priority) {
  if (!Number.isInteger(priority)) return '';
  const cls = priority <= 1 ? 'chip-destructive' : priority === 2 ? 'chip-caution' : 'chip-information';
  return `<span class="chip ${cls}" title="Source-reported priority value">P${escapeHtml(String(priority))}</span>`;
}
export function dueState(due, statusBucketKey, now = Date.now()) {
  if (!due) return null;
  const parsed = Date.parse(due);
  if (!Number.isFinite(parsed)) return null;
  if (statusBucketKey === 'done') return 'done';
  const soon = now + 7 * 86400000;
  if (parsed < now) return 'overdue';
  if (parsed <= soon) return 'due-soon';
  return 'scheduled';
}
function dueBadge(task) {
  if (!task.due) return '';
  const state = dueState(task.due, bucket(task.status));
  const cls = state === 'overdue' ? 'chip-destructive' : state === 'due-soon' ? 'chip-caution' : 'chip-information';
  const label = state === 'overdue' ? 'Overdue' : state === 'due-soon' ? 'Due soon' : 'Due';
  return timeTag(task.due, label).replace('class="timestamp"', `class="timestamp chip ${cls}"`);
}
export function taskCard(task) {
  const source = safeLink(task.source_url);
  const b = bucket(task.status);
  return `<article class="task ${b}" data-task-id="${escapeHtml(task.id)}"><div class="task-top"><span class="lane-label">${escapeHtml(task.lane || 'Unassigned lane')}</span><span class="chip chip-${b === 'done' ? 'positive' : b === 'blocked' ? 'caution' : b === 'flight' ? 'information' : 'neutral'}">${escapeHtml(String(task.status || 'unknown').replaceAll('_', ' '))}</span></div><h4>${escapeHtml(task.title || 'Untitled task')}</h4>${task.detail ? `<p class="task-detail">${escapeHtml(task.detail)}</p>` : ''}<div class="task-meta-row">${priorityBadge(task.priority)}${dueBadge(task)}</div><div class="owner">${escapeHtml(task.owner || 'Unassigned')}</div>${timeTag(task.updated_at, 'Task updated')}${timeTag(task.checked_at, 'Checked')}<details><summary>Source &amp; report details</summary>${task.detail ? `<p>${escapeHtml(task.detail)}</p>` : ''}<span class="receipt">${escapeHtml(task.id || 'Unknown record')}</span><span class="receipt">${escapeHtml(task.provenance || 'Task register')}</span>${timeTag(task.observed_at, 'Observed')}${task.source_doc ? `<div class="receipt">Source ${escapeHtml(task.source_doc)}</div>` : '<span class="receipt">Source document not reported</span>'}${source ? `<a class="receipt" href="${escapeHtml(source)}" target="_blank" rel="noopener noreferrer">Open source receipt</a>` : ''}</details></article>`;
}
export function observationCard(row) { const link = safeLink(row.source_url); return `<article class="observation"><div class="task-top"><span>${escapeHtml(row.lane)}</span><span class="chip ${statusChipClass(row.status)}">${escapeHtml(row.status || 'Unknown')}</span></div><h3>${escapeHtml(row.title)}</h3>${row.owner ? `<div class="owner">${escapeHtml(row.owner)}</div>` : ''}<p>${escapeHtml(row.detail)}</p>${timeTag(row.observed_at, 'Report observed')}${timeTag(row.updated_at, 'Work updated')}${row.kind === 'worker' ? timeTag(row.heartbeat_at, 'Heartbeat') : ''}${row.checked_at ? timeTag(row.checked_at, 'Checked') : ''}${row.upcoming?.length ? `<details><summary>Upcoming (${row.upcoming.length})</summary><ul>${row.upcoming.map(x => `<li>${escapeHtml(x)}</li>`).join('')}</ul></details>` : ''}${link ? `<a class="receipt" href="${escapeHtml(link)}" target="_blank" rel="noopener noreferrer">Open source receipt</a>` : '<span class="receipt">No linked receipt</span>'}</article>`; }

/* ---- Project-management additions (Claude Code · Sonnet · 2026-09-14) ---- */

// Heuristic chip classification of a free-text source status string. Never the only signal:
// callers must also render the literal text, per the shared contract's "status never relies on
// color alone" rule.
export function statusChipClass(text) {
  const t = String(text || '').toLowerCase();
  if (/error|unreachable|fail|interrupted/.test(t)) return 'chip-destructive';
  if (/auth required|redirect|pending|not connected|sign-in|awaiting|setup|undeployed|disconnected|schema ready|blocked/.test(t)) return 'chip-caution';
  if (/responding|^http 2|200 ·|done|verified|connected|tcp open/.test(t)) return 'chip-positive';
  return 'chip-information';
}

// KPI strip counts. `tasks` is the same array rendered on the board (boardTasks() + synthetic
// upcoming-step rows), so the strip and the kanban always agree on the same denominator.
export function kpiCounts(tasks, now = Date.now()) {
  const counts = { total: tasks.length, backlog: 0, inProgress: 0, blocked: 0, done: 0, overdue: 0, dueSoon: 0 };
  for (const task of tasks) {
    const b = bucket(task.status);
    if (b === 'upcoming' || b === 'other') counts.backlog++;
    else if (b === 'flight') counts.inProgress++;
    else if (b === 'blocked') counts.blocked++;
    else if (b === 'done') counts.done++;
    const state = dueState(task.due, b, now);
    if (state === 'overdue') counts.overdue++;
    else if (state === 'due-soon') counts.dueSoon++;
  }
  return counts;
}

export function laneTotals(tasks) {
  const map = new Map();
  for (const task of tasks) { const lane = task.lane || 'Unassigned'; map.set(lane, (map.get(lane) || 0) + 1); }
  return [...map.entries()].sort((a, b) => b[1] - a[1]);
}

export function laneTotalsWidget(tasks) {
  const totals = laneTotals(tasks);
  if (!totals.length) return '<p class="empty">No lane totals available from the current filtered set.</p>';
  const max = totals[0][1] || 1;
  return totals.slice(0, 12).map(([lane, count]) => `<div class="chart-row lane"><span>${escapeHtml(lane)}</span><meter min="0" max="${max}" value="${count}" aria-label="${escapeHtml(lane)}: ${count} tasks"></meter><strong>${count}</strong></div>`).join('');
}

export function healthSummary(endpoints) {
  const counts = healthCounts(endpoints || []);
  return { healthy: counts.reachable, gated: counts.gated, attention: counts.attention, unknown: counts.unknown, total: (endpoints || []).length };
}

export function kanbanColumnsHtml(rows) {
  return columns.map(([key, label]) => {
    const group = rows.filter(task => bucket(task.status) === key);
    return `<section class="column ${key}" aria-label="${label} tasks"><h3>${label}<span>${group.length}</span></h3>${group.length ? group.map(taskCard).join('') : '<p class="empty">No matching tasks reported.</p>'}</section>`;
  }).join('');
}

// Board body markup for either a single flat kanban or one swimlane per lane. Both paths reuse
// kanbanColumnsHtml so column semantics (labels, "done" caveat) never drift between the two modes.
export function renderBoardMarkup(rows, groupByLane) {
  if (!groupByLane) return `<div class="kanban">${kanbanColumnsHtml(rows)}</div>`;
  if (!rows.length) return '<p class="empty">No matching tasks reported.</p>';
  const lanes = [...new Set(rows.map(row => row.lane || 'Unassigned'))].sort();
  return lanes.map(lane => {
    const laneRows = rows.filter(row => (row.lane || 'Unassigned') === lane);
    return `<section class="lane-swimlane"><h3 class="lane-swimlane-title">${escapeHtml(lane)}<span>${laneRows.length}</span></h3><div class="kanban">${kanbanColumnsHtml(laneRows)}</div></section>`;
  }).join('');
}

// Unified tracker rows for the second view: known surfaces (merged with their matching
// kind='surface' observation, same merge rule the sidebar already used) plus every non-task
// observation kind (worker, catalog, migration, and any future kind) so nothing silently drops.
export function trackerRows(snapshot) {
  const surfaces = Array.isArray(snapshot.surfaces) ? snapshot.surfaces : [];
  const observations = Array.isArray(snapshot.observations) ? snapshot.observations : [];
  const surfaceRows = surfaces.map(surface => {
    const report = observations.find(row => row.kind === 'surface' && row.lane === surface.lane && row.title === surface.name);
    return {
      kind: 'surface', lane: surface.lane, name: surface.name, owner: report?.owner || null,
      status: report?.status || surface.status || 'Not checked', url: surface.url,
      observed_at: report?.observed_at || null, updated_at: report?.updated_at || null, checked_at: report?.checked_at || surface.checked_at || null,
    };
  });
  const otherRows = observations.filter(row => row.kind !== 'task' && row.kind !== 'surface').map(row => ({
    kind: row.kind, lane: row.lane, name: row.title, owner: row.owner, status: row.status || 'unknown',
    url: row.source_url, observed_at: row.observed_at, updated_at: row.updated_at, checked_at: row.checked_at,
  }));
  return [...surfaceRows, ...otherRows];
}

export function filterTrackerRows(rows, { lane = '', kind = '', search = '' } = {}) {
  const needle = search.toLowerCase();
  return rows.filter(row => (!lane || row.lane === lane) && (!kind || row.kind === kind) && (!needle || [row.name, row.lane, row.owner, row.status].join(' ').toLowerCase().includes(needle)));
}

export function trackerTableRows(rows) {
  if (!rows.length) return '<tr><td colspan="6" class="empty">No matching surfaces or observations reported.</td></tr>';
  return rows.map(row => {
    const link = safeLink(row.url);
    const freshest = row.checked_at || row.updated_at || row.observed_at;
    return `<tr><td>${link ? `<a href="${escapeHtml(link)}" target="_blank" rel="noopener noreferrer">${escapeHtml(row.name)}<span aria-hidden="true"> ↗</span></a>` : escapeHtml(row.name || 'Untitled')}</td><td>${escapeHtml(row.lane || 'Unassigned')}</td><td><span class="chip chip-neutral">${escapeHtml(row.kind || 'unknown')}</span></td><td><span class="chip ${statusChipClass(row.status)}">${escapeHtml(row.status || 'Unknown')}</span></td><td>${escapeHtml(row.owner || 'Unassigned')}</td><td>${freshest ? timeTag(freshest, 'Last checked') : '<span class="empty">Unknown</span>'}</td></tr>`;
  }).join('');
}
