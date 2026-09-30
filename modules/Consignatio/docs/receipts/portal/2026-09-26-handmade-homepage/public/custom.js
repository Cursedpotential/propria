/* Byline: Claude Code · Sonnet 5 · 2026-09-14 — native portal chart widgets.
   Owner rejected the whole-board iframe ("a little tiny window ... where I sit
   and scroll") and asked for widgets/graphs/Gantt charts built into Homepage
   that look native and are reactive. The stat widgets (lane rollups, surfaces,
   health, to-do feed) are real Homepage `customapi` widgets configured in
   services.yaml and need no JS. This file adds the two chart cards Homepage
   has no built-in widget type for: a health-latency trend line and a
   per-lane Gantt/timeline, both drawn with ApexCharts (vendored locally from
   progress-board, MIT-licensed v3.54.1 — see docs/URGENT-TODO.md for the
   licence note) directly into the real .service-card DOM so they use the
   same classes/tokens as every other tile and stay clickable through to the
   full page (the existing global `target: _blank` setting, unchanged here,
   already opens that full page in a whole browser tab rather than an iframe).
   Same file is used unmodified on homepage-public; BOARD_BASE below is always
   same-origin.
   2026-09-15 (Claude Code · Sonnet 5): CORRECTED — the previous BOARD_BASE
   special-cased "int.mitechconsult.com" to fetch the cross-origin absolute
   URL https://progress.int.mitechconsult.com, which sits behind its own
   Authentik forward-auth gate separate from homepage.int.mitechconsult.com;
   the browser's fetch() had no way to prove that second origin's session
   (progress-board's CORS reply never even applied, since Authentik's 302
   intercepts the request before it reaches progress-board), so both chart
   cards always fell back to "unavailable" on the public instance. Fix: use
   the SAME-ORIGIN "/progress" path on every hostname. That path already
   proxies to progress-board on BOTH instances without any code changes here
   — tailnet homepage.tilapia-skilift.ts.net has "tailscale serve" wired
   `/progress -> http://100.72.169.40:3020` (verified live via `tailscale
   serve status`), and homepage.int.mitechconsult.com has the Traefik file-
   provider router "homepage-progress-path" (PathPrefix `/progress`,
   /data/coolify/proxy/dynamic/propria-public-portal.yaml) gated by the SAME
   authentik-forwardauth middleware as the page itself. Same-origin means no
   CORS, no credentials mode, and no new route — the browser just reuses
   whatever session already loaded the page. */
(function () {
  "use strict";

  var BOARD_BASE = location.protocol + "//" + location.hostname + "/progress";

  var PALETTE = ["#f0b45a", "#82bdc0", "#6cc392", "#f27479", "#b8b1a5", "#9aa5e8"];
  var REFRESH_MS = 30000;
  var charts = {}; // dataName -> ApexCharts instance

  function loadApex() {
    if (window.ApexCharts) return Promise.resolve();
    if (window.__apexLoading) return window.__apexLoading;
    window.__apexLoading = new Promise(function (res, rej) {
      var s = document.createElement("script");
      s.src = BOARD_BASE + "/vendor/apexcharts.min.js";
      s.onload = function () { res(); };
      s.onerror = function () { rej(new Error("apexcharts failed to load")); };
      document.head.appendChild(s);
    });
    return window.__apexLoading;
  }

  function findMount(dataName) {
    var li = document.querySelector('li.service[data-name="' + dataName + '"]');
    if (!li) return null;
    var card = li.querySelector(".service-card");
    if (!card) return null;
    var mount = card.querySelector(".board-chart-mount");
    if (mount) return mount;
    var container = document.createElement("div");
    container.className = "relative flex flex-row w-full service-container";
    var block = document.createElement("div");
    block.className =
      "bg-theme-200/50 dark:bg-theme-900/20 rounded-sm m-1 flex-1 flex flex-col items-center justify-center text-center scheme-light service-block board-chart-block";
    mount = document.createElement("div");
    mount.className = "board-chart-mount";
    mount.style.width = "100%";
    mount.style.minHeight = "200px";
    block.appendChild(mount);
    container.appendChild(block);
    card.appendChild(container);
    return mount;
  }

  function showUnavailable(mount, label) {
    mount.innerHTML =
      '<p class="text-theme-500 dark:text-theme-300 text-xs font-light board-chart-note">' +
      label + " unavailable — open the full page for live data.</p>";
  }

  function apexBase(extra) {
    var base = {
      chart: {
        background: "transparent",
        toolbar: { show: false },
        animations: { enabled: true, speed: 300 },
        fontFamily: "inherit",
      },
      theme: { mode: "dark" },
      grid: { borderColor: "rgba(255,255,255,0.08)", strokeDashArray: 3 },
      tooltip: { theme: "dark" },
      colors: PALETTE,
      legend: { labels: { colors: "#c9c4ba" } },
      xaxis: { labels: { style: { colors: "#c9c4ba" } } },
      yaxis: { labels: { style: { colors: "#c9c4ba" } } },
    };
    for (var k in extra) base[k] = extra[k];
    return base;
  }

  function renderLatencyChart() {
    var mount = findMount("Latency trend");
    if (!mount) return;
    fetch(BOARD_BASE + "/api/health-history", { cache: "no-store" })
      .then(function (r) { if (!r.ok) throw new Error(r.status); return r.json(); })
      .then(function (data) {
        var samples = Array.isArray(data.samples) ? data.samples : [];
        if (!samples.length) { showUnavailable(mount, "Latency trend"); return; }
        var p50 = samples.map(function (s) { return [Date.parse(s.ts), s.p50_ms]; });
        var p95 = samples.map(function (s) { return [Date.parse(s.ts), s.p95_ms]; });
        var series = [
          { name: "p50 ms", data: p50 },
          { name: "p95 ms", data: p95 },
        ];
        if (charts.latency) {
          charts.latency.updateSeries(series);
          return;
        }
        mount.innerHTML = "";
        var opts = apexBase({
          series: series,
          chart: { type: "area", height: 210, background: "transparent", toolbar: { show: false } },
          stroke: { curve: "smooth", width: 2 },
          fill: { type: "gradient", gradient: { opacityFrom: 0.35, opacityTo: 0.02 } },
          xaxis: { type: "datetime", labels: { style: { colors: "#c9c4ba" }, datetimeUTC: false } },
          dataLabels: { enabled: false },
        });
        charts.latency = new window.ApexCharts(mount, opts);
        charts.latency.render();
      })
      .catch(function () { showUnavailable(mount, "Latency trend"); });
  }

  function renderTimelineChart() {
    var mount = findMount("Lane timeline");
    if (!mount) return;
    fetch(BOARD_BASE + "/api/timeline", { cache: "no-store" })
      .then(function (r) { if (!r.ok) throw new Error(r.status); return r.json(); })
      .then(function (data) {
        var raw = Array.isArray(data.series) ? data.series : [];
        if (!raw.length) { showUnavailable(mount, "Lane timeline"); return; }
        // One row per LANE (not per observation): every point in a series
        // shares its own lane name as the category, so ApexCharts draws all
        // of that lane's observations as parallel bars on one row instead of
        // one unreadable row per observation title.
        var series = raw.map(function (s) {
          return {
            name: s.name,
            data: (s.data || []).map(function (d) {
              return { x: s.name, y: d.y, title: d.x, status: d.status };
            }),
          };
        });
        if (charts.timeline) {
          charts.timeline.updateSeries(series);
          return;
        }
        mount.innerHTML = "";
        var opts = apexBase({
          series: series,
          chart: { type: "rangeBar", height: Math.max(220, Math.min(420, 70 + raw.length * 30)), toolbar: { show: false } },
          plotOptions: { bar: { horizontal: true, barHeight: "70%", rangeBarGroupRows: true } },
          xaxis: { type: "datetime", labels: { style: { colors: "#c9c4ba" }, datetimeUTC: false } },
          dataLabels: { enabled: false },
          tooltip: {
            custom: function (opt) {
              var pt = opt.w.config.series[opt.seriesIndex].data[opt.dataPointIndex];
              var start = new Date(pt.y[0]).toLocaleString();
              var end = new Date(pt.y[1]).toLocaleString();
              return (
                '<div style="background:#1f2023;color:#e6e2d8;padding:6px 10px;' +
                'font-size:11px;max-width:260px;border:1px solid rgba(255,255,255,0.12);">' +
                "<strong>" + (pt.title || "") + "</strong><br/>" +
                (pt.status || "") + "<br/>" + start + " – " + end + "</div>"
              );
            },
          },
        });
        charts.timeline = new window.ApexCharts(mount, opts);
        charts.timeline.render();
      })
      .catch(function () { showUnavailable(mount, "Lane timeline"); });
  }

  function tick() {
    loadApex()
      .then(function () {
        renderLatencyChart();
        renderTimelineChart();
      })
      .catch(function () {
        var l = findMount("Latency trend");
        var t = findMount("Lane timeline");
        if (l) showUnavailable(l, "Chart library");
        if (t) showUnavailable(t, "Chart library");
      });
  }

  // 2026-09-15 (Claude Code · Sonnet 5): owner rule 23:26 — dynamic-list
  // customapi widgets (Pipeline lanes, Surfaces needing attention, Slowest
  // endpoints, Open to-dos) render one run-on string per row and Homepage's
  // own dynamic-list mapping schema has no chip/multi-field layout (only a
  // single name+label pair per item — checked against gethomepage docs).
  // The API side (progress-board server.mjs) now sends a much shorter
  // top-N label; this is the DOM-side half: clip whatever label text still
  // doesn't fit to one line (CSS below) and put the FULL text Homepage
  // already rendered into a native `title` attribute, so hovering a row
  // shows the complete detail without another network request. Every
  // dynamic-list row in the app shares this exact wrapper shape
  // (`.flex.flex-row.text-right > .font-bold`), so this is one fix for
  // "any other widget with long strings", not just Pipeline lanes.
  function applyRowTitles() {
    var labels = document.querySelectorAll(
      ".flex.flex-row.text-right > .font-bold:not([data-title-applied])",
    );
    for (var i = 0; i < labels.length; i++) {
      var el = labels[i];
      var full = (el.textContent || "").trim();
      if (full) el.title = full;
      el.setAttribute("data-title-applied", "1");
    }
  }

  // Homepage hydrates client-side; the service tiles may not exist yet at
  // script-load time, so retry briefly before settling into the steady poll.
  var attempts = 0;
  var boot = setInterval(function () {
    attempts += 1;
    var ready =
      document.querySelector('li.service[data-name="Latency trend"] .service-card') ||
      document.querySelector('li.service[data-name="Lane timeline"] .service-card');
    if (ready || attempts > 10) {
      clearInterval(boot);
      tick();
      setInterval(tick, REFRESH_MS);
    }
  }, 500);

  // Row-title pass runs on its own short interval, independent of the chart
  // boot gate above, since dynamic-list widgets exist and refresh whether or
  // not the chart cards are present.
  setInterval(applyRowTitles, 2000);
})();

/* Byline: Claude Code · Sonnet 5 · 2026-09-15 — portal-editor live preview
   auto-refresh. Owner ask: saving services.yaml/settings.yaml/widgets.yaml/
   custom.css/custom.js in the editor should reload the side-by-side preview
   within ~2s, WITHOUT ever polling for a normal (non-framed) portal viewer.
   Gate: window.top !== window.self is only true when this page is loaded
   inside another page's frame — the editor's Simple Browser preview pane is
   the only intended case. dir is inferred from hostname since this same
   custom.js file is shared verbatim by both Homepage instances. */
(function () {
  if (window.top === window.self) return; // not framed: never poll
  var dir = location.hostname.indexOf(".int.mitechconsult.com") !== -1
    ? "homepage-public"
    : "homepage";
  var lastVersion = null;
  var POLL_MS = 2000;
  function poll() {
    fetch("/progress/api/portal-config-version?dir=" + dir, { cache: "no-store" })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (data) {
        if (!data || typeof data.version !== "number") return;
        if (lastVersion === null) {
          lastVersion = data.version;
          return;
        }
        if (data.version !== lastVersion) {
          lastVersion = data.version;
          location.reload();
        }
      })
      .catch(function () { /* transient fetch error: skip this tick */ });
  }
  poll();
  setInterval(poll, POLL_MS);
})();

/* Propria portal actions: source timestamps, authentication and repair requests. */
(function () {
  'use strict';
  const base = '/progress';
  let busy = false;
  let provider = 'claude';
  try { const saved=localStorage.getItem('propria-repair-provider'); if(['claude','codex','portkey'].includes(saved)) provider=saved; } catch {}
  const receipts = new Map();
  const card = name => document.querySelector('li.service[data-name="' + name + '"] .service-card');
  function element(tag, text, className) {
    const node = document.createElement(tag);
    if (text != null) node.textContent = text;
    if (className) node.className = className;
    return node;
  }
  function mount(name, id) {
    const parent = card(name);
    if (!parent) return null;
    let node = parent.querySelector('#' + id);
    if (!node) { node = element('div', null, 'portal-actions'); node.id = id; parent.append(node); }
    return node;
  }
  async function read(path) {
    const response = await fetch(base + path, {cache: 'no-store', credentials: 'same-origin'});
    if (!response.ok) throw new Error('Status unavailable (' + response.status + ')');
    return response.json();
  }
  function safeUrl(value) {
    try { const url = new URL(value); return ['https:', 'http:'].includes(url.protocol) && !url.username && !url.password ? url.href : null; } catch { return null; }
  }
  function renderLanes(data) {
    const node = mount('Pipeline lanes', 'portal-lane-times');
    if (!node) return;
    const list = element('div', null, 'portal-lanes');
    (data.lanes || []).forEach(lane => {
      const row = element('div', null, 'portal-lane');
      const heading=element('div',null,'portal-lane-heading');
      heading.append(element('strong', lane.lane),element('span',lane.total+' reports','portal-lane-total'));
      row.append(heading);
      const badges=element('div',null,'portal-lane-badges');
      (lane.status_breakdown||[]).forEach(entry=>{
        const raw=entry.status||'unknown';let label=raw.replaceAll('_',' '),tone='neutral';
        if(/^http 2/.test(raw)){label='Responding';tone='positive';}
        else if(/403/.test(raw)){label='Access denied';tone='neutral';}
        else if(/401|sign.in|auth required/.test(raw)){label='Sign in';tone='caution';}
        else if(/not connected|no deployed link/.test(raw)){label='Not configured';tone='neutral';}
        else if(raw==='done'){label='Done';tone='positive';}
        else if(/in_progress|running|in_flight/.test(raw)){label='In progress';tone='information';}
        else if(/blocked|fail|interrupted|unreachable/.test(raw)){label=raw==='interrupted'?'Interrupted':raw==='blocked'?'Blocked':'Needs attention';tone='caution';}
        else if(label==='preview available'){label='Preview ready';tone='information';}
        if(label.length>28)label='Other status';
        const badge=element('span',entry.count+' '+label,'portal-status-badge portal-status-'+tone);badge.title=raw;badges.append(badge);
      });
      row.append(badges);
      const date = new Date(lane.last_updated);
      const valid = lane.last_updated && Number.isFinite(date.getTime());
      const time = element('time', valid ? 'Updated '+date.toLocaleString([], {month:'short', day:'numeric', hour:'numeric', minute:'2-digit'}) : 'Update time not reported');
      if (valid) { time.dateTime = date.toISOString(); time.title = date.toLocaleString([], {timeZoneName:'short'}); }
      const footer=element('div',null,'portal-lane-footer');footer.append(time);
      const detail=element('a','View lane reports');detail.href='/progress/?lane='+encodeURIComponent(lane.lane);detail.className='portal-lane-link';footer.append(detail);
      row.append(footer);list.append(row);
    });
    node.replaceChildren(list);
    node.parentElement.classList.add('portal-has-actions');
  }
  function renderSurfaces(data) {
    const totals = mount('Live surfaces', 'portal-access-counts');
    if (totals) totals.textContent = (data.auth_required || 0) + ' sign-in required · ' + (data.unconfigured || 0) + ' not configured · '+(data.monitor_denied||0)+' monitor access denied';
    const node = mount('Surfaces needing attention', 'portal-surface-actions');
    if (!node) return;
    // Don't replace focused controls or an in-flight request when the poll returns.
    if (busy || node.contains(document.activeElement)) return;
    const fragment = document.createDocumentFragment();
    const providerLabel=element('label','Repair provider ', 'portal-provider-choice');
    const providerSelect=element('select'); providerSelect.setAttribute('aria-label','Repair provider');
    [['claude','Claude SDK'],['codex','Codex SDK'],['portkey','Portkey routing']].forEach(([value,label])=>{const option=element('option',label);option.value=value;option.selected=value===provider;providerSelect.append(option);});
    providerSelect.onchange=()=>{provider=providerSelect.value;try{localStorage.setItem('propria-repair-provider',provider);}catch{}};
    providerLabel.append(providerSelect);fragment.append(providerLabel,element('p','Uses your saved usage limits and fallback setting','portal-action-note'));
    (data.down_list || []).forEach(surface => {
      const row = element('div', null, 'portal-surface-action');
      row.append(element('strong', surface.name));
      const auth = surface.raw_state === 'auth required';
      row.append(element('span', surface.raw_state==='monitor access denied' ? 'Monitor access denied · your tailnet access may still work' : auth ? 'Sign-in required · service responded' : surface.raw_state === 'not connected' ? 'Not configured' : surface.summary, 'portal-action-state'));
      const controls = element('div', null, 'portal-action-controls');
      const href = safeUrl(surface.url);
      if ((auth || surface.raw_state==='monitor access denied') && href) {
        const link = element('a',auth?'Sign in':'Open app', 'portal-action-button'); link.href = href; link.target = '_blank'; link.rel = 'noopener noreferrer';
        link.title = 'Open the service login. Browser sign-in does not authenticate the server health probe.';
        controls.append(link);
      }
      const recheck = element('button', 'Recheck', 'portal-action-button'); recheck.type = 'button';
      recheck.onclick = async () => { recheck.disabled = true; recheck.textContent = 'Checking…'; try { const latest=await read('/api/surfaces-summary'); const current=(latest.down_list||[]).find(s=>s.name===surface.name); status.textContent=current ? (current.raw_state+' · checked '+new Date(current.checked_at).toLocaleTimeString()) : 'Service responding to the latest probe.'; } catch(error) {status.textContent=error.message;} finally { recheck.disabled = false; recheck.textContent = 'Recheck'; } };
      controls.append(recheck);
      const queue = element('button', 'Queue repair agent', 'portal-action-button'); queue.type='button';
      const status = element('div', receipts.get(surface.name)||'', 'portal-job-status'); status.setAttribute('role','status'); status.setAttribute('aria-live','polite');
      let requestId = null;
      queue.onclick = async () => {
        busy=true; queue.disabled=true; providerSelect.disabled=true; status.textContent='Submitting to '+provider+'…';
        requestId ||= crypto.randomUUID();
        try {
          const response=await fetch(base+'/api/repair-jobs',{method:'POST',credentials:'same-origin',headers:{'Content-Type':'application/json','Idempotency-Key':requestId},body:JSON.stringify({surface:surface.name,provider})});
          const result=await response.json();
          if(!response.ok) throw new Error(result.message || 'Queue unavailable ('+response.status+')');
          status.textContent=result.message || ('Job '+result.id+' · '+result.status);
          receipts.set(surface.name,status.textContent);
          queue.textContent='Request recorded';
        } catch(error) { status.textContent=error.message; queue.disabled=false; }
        finally { busy=false;providerSelect.disabled=false; }
      };
      controls.append(queue); row.append(controls,status); fragment.append(row);
    });
    if (!fragment.childNodes.length) fragment.append(element('p','No surfaces currently need attention.'));
    const note=element('p','Sign-in opens the service in a new tab. Recheck tests reachability; it cannot inspect your private browser session.','portal-action-note');
    fragment.append(note); node.replaceChildren(fragment);
    node.parentElement.classList.add('portal-has-actions');
  }
  async function refresh() {
    await Promise.allSettled([
      read('/api/lane-rollups').then(renderLanes),
      read('/api/surfaces-summary').then(renderSurfaces)
    ]);
  }
  let attempts=0;
  const boot=setInterval(()=>{ if(card('Pipeline lanes') || ++attempts>30){clearInterval(boot);refresh();setInterval(refresh,30000);} },500);
})();

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
