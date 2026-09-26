module.exports=[37702,(a,b,c)=>{b.exports=a.x("worker_threads",()=>require("worker_threads"))},83844,a=>{"use strict";let b,c,d,e,f,g,h,i;var j,k,l,m,n,o,p,q,r=a.i(97582),s=a.i(2107);function t(){let[a,b]=(0,s.useState)("light");return(0,s.useEffect)(()=>{let a=window.matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light";b(a),document.documentElement.setAttribute("data-theme",a)},[]),{theme:a,toggle:()=>{let c="dark"===a?"light":"dark";b(c),localStorage.setItem("traceiq-theme",c),document.documentElement.setAttribute("data-theme",c)}}}let u=(0,s.createContext)(null),v={query:"",dateFrom:"",dateTo:"",eventTypes:[],tags:[],overnight:[],minProbability:0};function w({children:a}){let[b,c]=(0,s.useState)("manual"),[d,e]=(0,s.useState)("explore"),[f,g]=(0,s.useState)(v),[h,i]=(0,s.useState)("pins"),[j,k]=(0,s.useState)(null),[l,m]=(0,s.useState)([]),{theme:n,toggle:o}=t();return(0,r.jsx)(u.Provider,{value:{mode:b,setMode:c,tab:d,setTab:e,filters:f,setFilters:g,mapMode:h,setMapMode:i,selectedEvent:j,setSelectedEvent:k,filteredEvents:l,setFilteredEvents:m,theme:n,toggleTheme:o},children:a})}function x(){let a=(0,s.useContext)(u);if(!a)throw Error("useWorkspace must be inside WorkspaceProvider");return a}let y=a=>{let b=a.replace(/^([A-Z])|[\s-_]+(\w)/g,(a,b,c)=>c?c.toUpperCase():b.toLowerCase());return b.charAt(0).toUpperCase()+b.slice(1)},z=(...a)=>a.filter((a,b,c)=>!!a&&""!==a.trim()&&c.indexOf(a)===b).join(" ").trim();var A={xmlns:"http://www.w3.org/2000/svg",width:24,height:24,viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:2,strokeLinecap:"round",strokeLinejoin:"round"};let B=(0,s.forwardRef)(({color:a="currentColor",size:b=24,strokeWidth:c=2,absoluteStrokeWidth:d,className:e="",children:f,iconNode:g,...h},i)=>(0,s.createElement)("svg",{ref:i,...A,width:b,height:b,stroke:a,strokeWidth:d?24*Number(c)/Number(b):c,className:z("lucide",e),...!f&&!(a=>{for(let b in a)if(b.startsWith("aria-")||"role"===b||"title"===b)return!0})(h)&&{"aria-hidden":"true"},...h},[...g.map(([a,b])=>(0,s.createElement)(a,b)),...Array.isArray(f)?f:[f]])),C=(a,b)=>{let c=(0,s.forwardRef)(({className:c,...d},e)=>(0,s.createElement)(B,{ref:e,iconNode:b,className:z(`lucide-${y(a).replace(/([a-z0-9])([A-Z])/g,"$1-$2").toLowerCase()}`,`lucide-${a}`,c),...d}));return c.displayName=y(a),c},D=C("activity",[["path",{d:"M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2",key:"169zse"}]]),E=C("chart-column",[["path",{d:"M3 3v16a2 2 0 0 0 2 2h16",key:"c24i48"}],["path",{d:"M18 17V9",key:"2bz60n"}],["path",{d:"M13 17V5",key:"1frdt8"}],["path",{d:"M8 17v-3",key:"17ska0"}]]),F=C("file-spreadsheet",[["path",{d:"M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z",key:"1rqfz7"}],["path",{d:"M14 2v4a2 2 0 0 0 2 2h4",key:"tnqrlb"}],["path",{d:"M8 13h2",key:"yr2amv"}],["path",{d:"M14 13h2",key:"un5t4a"}],["path",{d:"M8 17h2",key:"2yhykz"}],["path",{d:"M14 17h2",key:"10kma7"}]]),G=C("download",[["path",{d:"M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4",key:"ih7n3h"}],["polyline",{points:"7 10 12 15 17 10",key:"2ggqvy"}],["line",{x1:"12",x2:"12",y1:"15",y2:"3",key:"1vk2je"}]]),H=C("settings",[["path",{d:"M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z",key:"1qme2f"}],["circle",{cx:"12",cy:"12",r:"3",key:"1v7zrd"}]]),I=C("moon",[["path",{d:"M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z",key:"a7tn18"}]]),J=C("sun",[["circle",{cx:"12",cy:"12",r:"4",key:"4exip2"}],["path",{d:"M12 2v2",key:"tus03m"}],["path",{d:"M12 20v2",key:"1lh1kg"}],["path",{d:"m4.93 4.93 1.41 1.41",key:"149t6j"}],["path",{d:"m17.66 17.66 1.41 1.41",key:"ptbguv"}],["path",{d:"M2 12h2",key:"1t8f8n"}],["path",{d:"M20 12h2",key:"1q8mjw"}],["path",{d:"m6.34 17.66-1.41 1.41",key:"1m8zz5"}],["path",{d:"m19.07 4.93-1.41 1.41",key:"1shlcs"}]]),K=C("search",[["path",{d:"m21 21-4.34-4.34",key:"14j7rj"}],["circle",{cx:"11",cy:"11",r:"8",key:"4ej97u"}]]),L=[{key:"explore",label:"Explore",icon:K},{key:"analytics",label:"Analytics",icon:E},{key:"tables",label:"Tables",icon:F},{key:"export",label:"Export",icon:G},{key:"config",label:"Config",icon:H}];function M(){let{tab:a,setTab:b,mode:c,setMode:d}=x(),{theme:e,toggle:f}=t();return(0,r.jsxs)("header",{className:"flex items-center justify-between h-14 px-4 border-b border-border bg-surface shrink-0",children:[(0,r.jsxs)("div",{className:"flex items-center gap-4",children:[(0,r.jsxs)("div",{className:"flex items-center gap-2",children:[(0,r.jsx)(D,{className:"w-5 h-5 text-signal"}),(0,r.jsx)("span",{className:"font-semibold tracking-tight",children:"TraceIQ"})]}),(0,r.jsx)("span",{className:"text-xs px-2 py-0.5 rounded-full bg-surface-2 text-muted border border-border",children:"Subject K"})]}),(0,r.jsx)("nav",{className:"hidden md:flex items-center gap-1",children:L.map(c=>{let d=c.icon,e=a===c.key;return(0,r.jsxs)("button",{onClick:()=>b(c.key),className:`flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg transition-colors ${e?"bg-signal-soft text-signal":"text-muted hover:text-ink hover:bg-surface-2"}`,children:[(0,r.jsx)(d,{className:"w-4 h-4"}),c.label]},c.key)})}),(0,r.jsxs)("div",{className:"flex items-center gap-3",children:[(0,r.jsxs)("div",{className:"flex items-center bg-surface-2 rounded-lg p-0.5 border border-border",children:[(0,r.jsx)("button",{onClick:()=>d("manual"),className:`px-3 py-1 text-xs font-medium rounded-md transition ${"manual"===c?"bg-surface text-ink shadow-sm":"text-muted hover:text-ink"}`,children:"Manual"}),(0,r.jsx)("button",{onClick:()=>d("agent"),className:`px-3 py-1 text-xs font-medium rounded-md transition ${"agent"===c?"bg-signal text-white shadow-sm":"text-muted hover:text-ink"}`,children:"Agent"})]}),(0,r.jsx)("button",{onClick:f,className:"p-2 rounded-lg border border-border bg-surface text-muted hover:text-ink","aria-label":"Toggle theme",children:"dark"===e?(0,r.jsx)(J,{className:"w-4 h-4"}):(0,r.jsx)(I,{className:"w-4 h-4"})})]})]})}let N=[{name:"Detroit, MI",lat:42.3314,lng:-83.0458},{name:"Ann Arbor, MI",lat:42.2808,lng:-83.743},{name:"Grand Rapids, MI",lat:42.9634,lng:-85.6681},{name:"Lansing, MI",lat:42.7325,lng:-84.5555},{name:"Flint, MI",lat:43.0125,lng:-83.6875},{name:"Traverse City, MI",lat:44.7631,lng:-85.6206},{name:"Kalamazoo, MI",lat:42.2917,lng:-85.5872},{name:"Toledo, OH",lat:41.6528,lng:-83.5379}],O=["google_maps_visit","semantic_location_history","timeline_path","activity_segment","place_visit"],P=["home","work","overnight","travel","sensitive","medical","commercial","social","repeated"];function Q(a,b){return Math.floor(Math.random()*(b-a+1))+a}function R(a){let b=a=>a.toString().padStart(2,"0");return`${a.getFullYear()}-${b(a.getMonth()+1)}-${b(a.getDate())}T${b(a.getHours())}:${b(a.getMinutes())}:${b(a.getSeconds())}`}let S=function(a=320){let b=[],c=new Date,d=new Date(c);d.setDate(d.getDate()-90);for(let e=0;e<a;e++){let a=new Date(d.getTime()+Math.random()*(c.getTime()-d.getTime())),f=Q(15,720),g=new Date(a.getTime()+6e4*f),h=N[Q(0,N.length-1)],i=()=>(Math.random()-.5)*.04,j=h.lat+i(),k=h.lng+i(),l=Math.round(1e4*j)/1e4,m=Math.round(1e4*k)/1e4,n=O[Q(0,O.length-1)],o="visit";"timeline_path"===n?o="timeline_path":"activity_segment"===n&&(o="activity");let p=a.getHours(),q=p>=20||p<=5?"overnight":"daytrip",r=[],s=Q(1,3);for(;r.length<s;){let a=P[Q(0,P.length-1)];r.includes(a)||r.push(a)}"overnight"!==q||r.includes("overnight")||r.push("overnight");let t=Math.round((.5+.5*Math.random())*100)/100;b.push({event_id:crypto.randomUUID(),serial_display:`E${(e+1).toString().padStart(4,"0")}`,raw_type:n,event_type:o,start_eastern:R(a),end_eastern:R(g),start_utc:R(new Date(a.getTime()+144e5)),end_utc:R(new Date(g.getTime()+144e5)),lat:j,lng:k,lat_r4:l,lng_r4:m,place_id:`place_${crypto.randomUUID().slice(0,8)}`,tags:r,overnight_simple:Math.random()>.15?q:null,probability:t})}return b.sort((a,b)=>new Date(a.start_utc).getTime()-new Date(b.start_utc).getTime())}();function T(a,b){if(b.query){let c=b.query.toLowerCase();if(![a.serial_display,a.raw_type,a.event_type,a.place_id,a.tags.join(" ")].join(" ").toLowerCase().includes(c))return!1}return!(b.dateFrom&&a.start_eastern<b.dateFrom||b.dateTo&&a.end_eastern>b.dateTo+"T23:59:59"||b.eventTypes?.length&&!b.eventTypes.includes(a.event_type)||b.tags?.length&&!b.tags.some(b=>a.tags.includes(b))||b.overnight?.length&&!b.overnight.includes(a.overnight_simple))&&("number"!=typeof b.minProbability||!(a.probability<b.minProbability))}let U={listEvents:async()=>S,searchEvents:async a=>S.filter(b=>T(b,a)),getEvent:async a=>S.find(b=>b.event_id===a)??null,async askAgent(a,b){let c=S.filter(a=>T(a,b)).slice(0,10);return{answer:`I found ${c.length} events matching your query. Top pattern: ${c[0]?.event_type??"none"} near ${c[0]?.place_id??"unknown"}.`,receipts:c.map(a=>a.event_id).slice(0,6)}}},V=C("calendar",[["path",{d:"M8 2v4",key:"1cmpym"}],["path",{d:"M16 2v4",key:"4m81vk"}],["rect",{width:"18",height:"18",x:"3",y:"4",rx:"2",key:"1hopcy"}],["path",{d:"M3 10h18",key:"8toen8"}]]),W=C("tag",[["path",{d:"M12.586 2.586A2 2 0 0 0 11.172 2H4a2 2 0 0 0-2 2v7.172a2 2 0 0 0 .586 1.414l8.704 8.704a2.426 2.426 0 0 0 3.42 0l6.58-6.58a2.426 2.426 0 0 0 0-3.42z",key:"vktsd0"}],["circle",{cx:"7.5",cy:"7.5",r:".5",fill:"currentColor",key:"kqv944"}]]),X=C("funnel",[["path",{d:"M10 20a1 1 0 0 0 .553.895l2 1A1 1 0 0 0 14 21v-7a2 2 0 0 1 .517-1.341L21.74 4.67A1 1 0 0 0 21 3H3a1 1 0 0 0-.742 1.67l7.225 7.989A2 2 0 0 1 10 14z",key:"sc7q7i"}]]),Y=C("map-pin",[["path",{d:"M20 10c0 4.993-5.539 10.193-7.399 11.799a1 1 0 0 1-1.202 0C9.539 20.193 4 14.993 4 10a8 8 0 0 1 16 0",key:"1r0f0z"}],["circle",{cx:"12",cy:"10",r:"3",key:"ilqhr7"}]]),Z=C("route",[["circle",{cx:"6",cy:"19",r:"3",key:"1kj8tv"}],["path",{d:"M9 19h8.5a3.5 3.5 0 0 0 0-7h-11a3.5 3.5 0 0 1 0-7H15",key:"1d8sl"}],["circle",{cx:"18",cy:"5",r:"3",key:"gq8acd"}]]),$=C("flame",[["path",{d:"M8.5 14.5A2.5 2.5 0 0 0 11 12c0-1.38-.5-2-1-3-1.072-2.143-.224-4.054 2-6 .5 2.5 2 4.9 4 6.5 2 1.6 3 3.5 3 5.5a7 7 0 1 1-14 0c0-1.153.433-2.294 1-3a2.5 2.5 0 0 0 2.5 2.5z",key:"96xj49"}]]),_=C("clock",[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["polyline",{points:"12 6 12 12 16 14",key:"68esgv"}]]),aa=C("x",[["path",{d:"M18 6 6 18",key:"1bl5f8"}],["path",{d:"m6 6 12 12",key:"d8bk6v"}]]),ab=["home","work","overnight","travel","sensitive","medical","commercial","social","repeated"],ac=[{key:"visit",label:"Visit"},{key:"activity",label:"Activity"},{key:"timeline_path",label:"Path"}],ad=[{key:"pins",label:"Pins",icon:Y},{key:"paths",label:"Paths",icon:Z},{key:"heatmap",label:"Heat",icon:$},{key:"time",label:"Time",icon:_}];function ae(){let{filters:a,setFilters:b,mapMode:c,setMapMode:d,setFilteredEvents:e}=x(),[f,g]=(0,s.useState)(a.query);(0,s.useEffect)(()=>{let b=setTimeout(()=>{U.searchEvents(a).then(e)},150);return()=>clearTimeout(b)},[a,e]);let h=(0,s.useMemo)(()=>+!!a.eventTypes.length+ +!!a.tags.length+ +!!a.overnight.length+(a.dateFrom||a.dateTo?1:0)+ +(a.minProbability>0),[a]);return(0,r.jsxs)("div",{className:"flex flex-col h-full w-full min-w-[280px] max-w-[340px] bg-surface border-r border-border",children:[(0,r.jsxs)("div",{className:"flex items-center justify-between px-4 py-3 border-b border-border",children:[(0,r.jsxs)("div",{className:"flex items-center gap-2 text-sm font-medium text-ink",children:[(0,r.jsx)(X,{className:"w-4 h-4"})," Filters"]}),h>0&&(0,r.jsxs)("button",{onClick:()=>b({query:"",dateFrom:"",dateTo:"",eventTypes:[],tags:[],overnight:[],minProbability:0}),className:"text-xs text-signal hover:underline flex items-center gap-1",children:[(0,r.jsx)(aa,{className:"w-3 h-3"})," Clear ",h]})]}),(0,r.jsxs)("div",{className:"flex-1 overflow-y-auto p-4 space-y-5",children:[(0,r.jsxs)("div",{children:[(0,r.jsxs)("label",{className:"flex items-center gap-1.5 text-xs font-medium text-muted mb-2",children:[(0,r.jsx)(K,{className:"w-3 h-3"})," Search"]}),(0,r.jsx)("input",{value:f,onChange:c=>{g(c.target.value),b({...a,query:c.target.value})},placeholder:"Event ID, type, tag...",className:"w-full px-3 py-2 text-sm bg-surface-2 border border-border rounded-lg focus:outline-none focus:border-signal"})]}),(0,r.jsxs)("div",{children:[(0,r.jsxs)("label",{className:"flex items-center gap-1.5 text-xs font-medium text-muted mb-2",children:[(0,r.jsx)(V,{className:"w-3 h-3"})," Date range"]}),(0,r.jsxs)("div",{className:"flex items-center gap-2",children:[(0,r.jsx)("input",{type:"date",value:a.dateFrom,onChange:c=>b({...a,dateFrom:c.target.value}),className:"w-full px-2 py-1.5 text-xs bg-surface-2 border border-border rounded-md focus:outline-none focus:border-signal"}),(0,r.jsx)("span",{className:"text-muted",children:"→"}),(0,r.jsx)("input",{type:"date",value:a.dateTo,onChange:c=>b({...a,dateTo:c.target.value}),className:"w-full px-2 py-1.5 text-xs bg-surface-2 border border-border rounded-md focus:outline-none focus:border-signal"})]})]}),(0,r.jsxs)("div",{children:[(0,r.jsx)("label",{className:"text-xs font-medium text-muted mb-2 block",children:"Event type"}),(0,r.jsx)("div",{className:"flex flex-wrap gap-2",children:ac.map(c=>(0,r.jsx)("button",{onClick:()=>{var d;let e;return d=c.key,e=a.eventTypes.includes(d)?a.eventTypes.filter(a=>a!==d):[...a.eventTypes,d],void b({...a,eventTypes:e})},className:`px-2.5 py-1 text-xs rounded-full border transition ${a.eventTypes.includes(c.key)?"bg-signal-soft border-signal text-signal":"bg-surface-2 border-border text-muted hover:text-ink"}`,children:c.label},c.key))})]}),(0,r.jsxs)("div",{children:[(0,r.jsxs)("label",{className:"flex items-center gap-1.5 text-xs font-medium text-muted mb-2",children:[(0,r.jsx)(W,{className:"w-3 h-3"})," Tags"]}),(0,r.jsx)("div",{className:"flex flex-wrap gap-1.5",children:ab.map(c=>(0,r.jsx)("button",{onClick:()=>{let d;return d=a.tags.includes(c)?a.tags.filter(a=>a!==c):[...a.tags,c],void b({...a,tags:d})},className:`px-2 py-0.5 text-xs rounded-full border transition ${a.tags.includes(c)?"bg-signal-soft border-signal text-signal":"bg-surface-2 border-border text-muted hover:text-ink"}`,children:c},c))})]}),(0,r.jsxs)("div",{children:[(0,r.jsx)("label",{className:"text-xs font-medium text-muted mb-2 block",children:"Overnight"}),(0,r.jsx)("div",{className:"flex gap-2",children:["overnight","daytrip"].map(c=>(0,r.jsx)("button",{onClick:()=>{let d=a.overnight.includes(c)?a.overnight.filter(a=>a!==c):[...a.overnight,c];b({...a,overnight:d})},className:`px-2.5 py-1 text-xs rounded-full border capitalize transition ${a.overnight.includes(c)?"bg-signal-soft border-signal text-signal":"bg-surface-2 border-border text-muted hover:text-ink"}`,children:c},c))})]}),(0,r.jsxs)("div",{children:[(0,r.jsxs)("label",{className:"text-xs font-medium text-muted mb-2 block",children:["Min probability: ",a.minProbability.toFixed(2)]}),(0,r.jsx)("input",{type:"range",min:0,max:1,step:.05,value:a.minProbability,onChange:c=>b({...a,minProbability:Number(c.target.value)}),className:"w-full accent-signal"})]}),(0,r.jsxs)("div",{children:[(0,r.jsx)("label",{className:"text-xs font-medium text-muted mb-2 block",children:"Map mode"}),(0,r.jsx)("div",{className:"grid grid-cols-2 gap-2",children:ad.map(a=>{let b=a.icon;return(0,r.jsxs)("button",{onClick:()=>d(a.key),className:`flex items-center justify-center gap-1.5 px-2 py-1.5 text-xs rounded-lg border transition ${c===a.key?"bg-signal-soft border-signal text-signal":"bg-surface-2 border-border text-muted hover:text-ink"}`,children:[(0,r.jsx)(b,{className:"w-3.5 h-3.5"}),a.label]},a.key)})})]})]})]})}let af=s.createContext(null);function ag(a,b){if(a===b)return!0;if(!a||!b)return!1;if(Array.isArray(a)){if(!Array.isArray(b)||a.length!==b.length)return!1;for(let c=0;c<a.length;c++)if(!ag(a[c],b[c]))return!1;return!0}if(Array.isArray(b))return!1;if("object"==typeof a&&"object"==typeof b){let c=Object.keys(a),d=Object.keys(b);if(c.length!==d.length)return!1;for(let d of c)if(!b.hasOwnProperty(d)||!ag(a[d],b[d]))return!1;return!0}return!1}function ah(a,b){if(!a.getProjection)return;let c=a.getProjection();ag(c,b.getProjection())||b.setProjection(c)}function ai(a){return{longitude:a.center.lng,latitude:a.center.lat,zoom:a.zoom,pitch:a.pitch,bearing:a.bearing,padding:a.padding}}function aj(a,b){let c=b.viewState||b,d=!1;if("longitude"in c&&"latitude"in c){let b=a.center;a.center=new b.constructor(c.longitude,c.latitude),d=d||b!==a.center}if("zoom"in c){let b=a.zoom;a.zoom=c.zoom,d=d||b!==a.zoom}if("bearing"in c){let b=a.bearing;a.bearing=c.bearing,d=d||b!==a.bearing}if("pitch"in c){let b=a.pitch;a.pitch=c.pitch,d=d||b!==a.pitch}return c.padding&&!a.isPaddingEqual(c.padding)&&(d=!0,a.padding=c.padding),d}let ak=["type","source","source-layer","minzoom","maxzoom","filter","layout"];function al(a){if(!a)return null;if("string"==typeof a||("toJS"in a&&(a=a.toJS()),!a.layers))return a;let b={};for(let c of a.layers)b[c.id]=c;let c=a.layers.map(a=>{let c=null;"interactive"in a&&(c=Object.assign({},a),delete c.interactive);let d=b[a.ref];if(d)for(let b of(c=c||Object.assign({},a),delete c.ref,ak))b in d&&(c[b]=d[b]);return c||a});return{...a,layers:c}}let am={version:8,sources:{},layers:[]},an={mousedown:"onMouseDown",mouseup:"onMouseUp",mouseover:"onMouseOver",mousemove:"onMouseMove",click:"onClick",dblclick:"onDblClick",mouseenter:"onMouseEnter",mouseleave:"onMouseLeave",mouseout:"onMouseOut",contextmenu:"onContextMenu",touchstart:"onTouchStart",touchend:"onTouchEnd",touchmove:"onTouchMove",touchcancel:"onTouchCancel"},ao={movestart:"onMoveStart",move:"onMove",moveend:"onMoveEnd",dragstart:"onDragStart",drag:"onDrag",dragend:"onDragEnd",zoomstart:"onZoomStart",zoom:"onZoom",zoomend:"onZoomEnd",rotatestart:"onRotateStart",rotate:"onRotate",rotateend:"onRotateEnd",pitchstart:"onPitchStart",pitch:"onPitch",pitchend:"onPitchEnd"},ap={wheel:"onWheel",boxzoomstart:"onBoxZoomStart",boxzoomend:"onBoxZoomEnd",boxzoomcancel:"onBoxZoomCancel",resize:"onResize",load:"onLoad",render:"onRender",idle:"onIdle",remove:"onRemove",data:"onData",styledata:"onStyleData",sourcedata:"onSourceData",error:"onError"},aq=["minZoom","maxZoom","minPitch","maxPitch","maxBounds","projection","renderWorldCopies"],ar=["scrollZoom","boxZoom","dragRotate","dragPan","keyboard","doubleClickZoom","touchZoomRotate","touchPitch"];class as{constructor(a,b,c){this._map=null,this._internalUpdate=!1,this._inRender=!1,this._hoveredFeatures=null,this._deferredEvents={move:!1,zoom:!1,pitch:!1,rotate:!1},this._onEvent=a=>{let b=this.props[ap[a.type]];b?b(a):"error"===a.type&&console.error(a.error)},this._onPointerEvent=a=>{("mousemove"===a.type||"mouseout"===a.type)&&this._updateHover(a);let b=this.props[an[a.type]];b&&(this.props.interactiveLayerIds&&"mouseover"!==a.type&&"mouseout"!==a.type&&(a.features=this._hoveredFeatures||this._queryRenderedFeatures(a.point)),b(a),delete a.features)},this._onCameraEvent=a=>{if(!this._internalUpdate){let b=this.props[ao[a.type]];b&&b(a)}a.type in this._deferredEvents&&(this._deferredEvents[a.type]=!1)},this._MapClass=a,this.props=b,this._initialize(c)}get map(){return this._map}get transform(){return this._renderTransform}setProps(a){let b=this.props;this.props=a;let c=this._updateSettings(a,b);c&&this._createShadowTransform(this._map);let d=this._updateSize(a),e=this._updateViewState(a,!0);this._updateStyle(a,b),this._updateStyleComponents(a,b),this._updateHandlers(a,b),(c||d||e&&!this._map.isMoving())&&this.redraw()}static reuse(a,b){let c=as.savedMaps.pop();if(!c)return null;let d=c.map,e=d.getContainer();for(b.className=e.className;e.childNodes.length>0;)b.appendChild(e.childNodes[0]);d._container=b;let f=d._resizeObserver;f&&(f.disconnect(),f.observe(b)),c.setProps({...a,styleDiffing:!1}),d.resize();let{initialViewState:g}=a;return g&&(g.bounds?d.fitBounds(g.bounds,{...g.fitBoundsOptions,duration:0}):c._updateViewState(g,!1)),d.isStyleLoaded()?d.fire("load"):d.once("styledata",()=>d.fire("load")),d._update(),c}_initialize(a){let{props:b}=this,{mapStyle:c=am}=b,d={...b,...b.initialViewState,accessToken:b.mapboxAccessToken||function(){let a=null;if("u">typeof location){let b=/access_token=([^&\/]*)/.exec(location.search);a=b&&b[1]}try{a=a||process.env.MapboxAccessToken}catch(a){}try{a=a||process.env.REACT_APP_MAPBOX_ACCESS_TOKEN}catch(a){}return a}()||null,container:a,style:al(c)},e=d.initialViewState||d.viewState||d;if(Object.assign(d,{center:[e.longitude||0,e.latitude||0],zoom:e.zoom||0,pitch:e.pitch||0,bearing:e.bearing||0}),b.gl){let a=HTMLCanvasElement.prototype.getContext;HTMLCanvasElement.prototype.getContext=()=>(HTMLCanvasElement.prototype.getContext=a,b.gl)}let f=new this._MapClass(d);e.padding&&f.setPadding(e.padding),b.cursor&&(f.getCanvas().style.cursor=b.cursor),this._createShadowTransform(f);let g=f._render;f._render=a=>{this._inRender=!0,g.call(f,a),this._inRender=!1};let h=f._renderTaskQueue.run;f._renderTaskQueue.run=a=>{h.call(f._renderTaskQueue,a),this._onBeforeRepaint()},f.on("render",()=>this._onAfterRepaint());let i=f.fire;for(let a in f.fire=this._fireEvent.bind(this,i),f.on("resize",()=>{this._renderTransform.resize(f.transform.width,f.transform.height)}),f.on("styledata",()=>{this._updateStyleComponents(this.props,{}),ah(f.transform,this._renderTransform)}),f.on("sourcedata",()=>this._updateStyleComponents(this.props,{})),an)f.on(a,this._onPointerEvent);for(let a in ao)f.on(a,this._onCameraEvent);for(let a in ap)f.on(a,this._onEvent);this._map=f}recycle(){let a=this.map.getContainer().querySelector("[mapboxgl-children]");null==a||a.remove(),as.savedMaps.push(this)}destroy(){this._map.remove()}redraw(){let a=this._map;!this._inRender&&a.style&&(a._frame&&(a._frame.cancel(),a._frame=null),a._render())}_createShadowTransform(a){var b;let c,d=((c=(b=a.transform).clone()).pixelsToGLUnits=b.pixelsToGLUnits,c);a.painter.transform=d,this._renderTransform=d}_updateSize(a){let{viewState:b}=a;if(b){let a=this._map;if(b.width!==a.transform.width||b.height!==a.transform.height)return a.resize(),!0}return!1}_updateViewState(a,b){if(this._internalUpdate)return!1;let c=this._map,d=this._renderTransform,{zoom:e,pitch:f,bearing:g}=d,h=c.isMoving();h&&(d.cameraElevationReference="sea");let i=aj(d,{...ai(c.transform),...a});if(h&&(d.cameraElevationReference="ground"),i&&b){let a=this._deferredEvents;a.move=!0,a.zoom||(a.zoom=e!==d.zoom),a.rotate||(a.rotate=g!==d.bearing),a.pitch||(a.pitch=f!==d.pitch)}return h||aj(c.transform,a),i}_updateSettings(a,b){let c=this._map,d=!1;for(let e of aq)if(e in a&&!ag(a[e],b[e])){d=!0;let b=c[`set${e[0].toUpperCase()}${e.slice(1)}`];null==b||b.call(c,a[e])}return d}_updateStyle(a,b){if(a.cursor!==b.cursor&&(this._map.getCanvas().style.cursor=a.cursor||""),a.mapStyle!==b.mapStyle){let{mapStyle:b=am,styleDiffing:c=!0}=a,d={diff:c};return"localIdeographFontFamily"in a&&(d.localIdeographFontFamily=a.localIdeographFontFamily),this._map.setStyle(al(b),d),!0}return!1}_updateStyleComponents(a,b){let c=this._map,d=!1;return c.isStyleLoaded()&&("light"in a&&c.setLight&&!ag(a.light,b.light)&&(d=!0,c.setLight(a.light)),"fog"in a&&c.setFog&&!ag(a.fog,b.fog)&&(d=!0,c.setFog(a.fog)),"terrain"in a&&c.setTerrain&&!ag(a.terrain,b.terrain)&&(!a.terrain||c.getSource(a.terrain.source))&&(d=!0,c.setTerrain(a.terrain))),d}_updateHandlers(a,b){var c,d;let e=this._map,f=!1;for(let g of ar){let h=null==(c=a[g])||c;ag(h,null==(d=b[g])||d)||(f=!0,h?e[g].enable(h):e[g].disable())}return f}_queryRenderedFeatures(a){let b=this._map,c=b.transform,{interactiveLayerIds:d=[]}=this.props;try{return b.transform=this._renderTransform,b.queryRenderedFeatures(a,{layers:d.filter(b.getLayer.bind(b))})}catch(a){return[]}finally{b.transform=c}}_updateHover(a){var b;let{props:c}=this;if(c.interactiveLayerIds&&(c.onMouseMove||c.onMouseEnter||c.onMouseLeave)){let c=a.type,d=(null==(b=this._hoveredFeatures)?void 0:b.length)>0,e=this._queryRenderedFeatures(a.point),f=e.length>0;!f&&d&&(a.type="mouseleave",this._onPointerEvent(a)),this._hoveredFeatures=e,f&&!d&&(a.type="mouseenter",this._onPointerEvent(a)),a.type=c}else this._hoveredFeatures=null}_fireEvent(a,b,c){let d=this._map,e=d.transform,f="string"==typeof b?b:b.type;return("move"===f&&this._updateViewState(this.props,!1),f in ao&&("object"==typeof b&&(b.viewState=ai(e)),this._map.isMoving()))?(d.transform=this._renderTransform,a.call(d,b,c),d.transform=e,d):(a.call(d,b,c),d)}_onBeforeRepaint(){let a=this._map;for(let b in this._internalUpdate=!0,this._deferredEvents)this._deferredEvents[b]&&a.fire(b);this._internalUpdate=!1;let b=this._map.transform;a.transform=this._renderTransform,this._onAfterRepaint=()=>{ah(this._renderTransform,b),a.transform=b}}}as.savedMaps=[];let at=["setMaxBounds","setMinZoom","setMaxZoom","setMinPitch","setMaxPitch","setRenderWorldCopies","setProjection","setStyle","addSource","removeSource","addLayer","removeLayer","setLayerZoomRange","setFilter","setPaintProperty","setLayoutProperty","setLight","setTerrain","setFog","remove"],au="u">typeof document?s.useLayoutEffect:s.useEffect,av=["baseApiUrl","maxParallelImageRequests","workerClass","workerCount","workerUrl"],aw=s.createContext(null);var ax=a.i(54633);let ay=/box|flex|grid|column|lineHeight|fontWeight|opacity|order|tabSize|zIndex/;function az(a,b){if(!a||!b)return;let c=a.style;for(let a in b){let d=b[a];Number.isFinite(d)&&!ay.test(a)?c[a]=`${d}px`:c[a]=d}}function aA(a){return new Set(a?a.trim().split(/\s+/):[])}(0,s.memo)((0,s.forwardRef)(function(a,b){var c;let d,e,f,g,{map:h,mapLib:i}=(0,s.useContext)(aw),j=(0,s.useRef)({props:a});j.current.props=a;let k=(0,s.useMemo)(()=>{let b=!1;s.Children.forEach(a.children,a=>{a&&(b=!0)});let c={...a,element:b?document.createElement("div"):null},d=new i.Marker(c);return d.setLngLat([a.longitude,a.latitude]),d.getElement().addEventListener("click",a=>{var b,c;null==(c=(b=j.current.props).onClick)||c.call(b,{type:"click",target:d,originalEvent:a})}),d.on("dragstart",a=>{var b,c;a.lngLat=k.getLngLat(),null==(c=(b=j.current.props).onDragStart)||c.call(b,a)}),d.on("drag",a=>{var b,c;a.lngLat=k.getLngLat(),null==(c=(b=j.current.props).onDrag)||c.call(b,a)}),d.on("dragend",a=>{var b,c;a.lngLat=k.getLngLat(),null==(c=(b=j.current.props).onDragEnd)||c.call(b,a)}),d},[]);(0,s.useEffect)(()=>(k.addTo(h.getMap()),()=>{k.remove()}),[]);let{longitude:l,latitude:m,offset:n,style:o,draggable:p=!1,popup:q=null,rotation:r=0,rotationAlignment:t="auto",pitchAlignment:u="auto"}=a;return(0,s.useEffect)(()=>{az(k.getElement(),o)},[o]),(0,s.useImperativeHandle)(b,()=>k,[]),(k.getLngLat().lng!==l||k.getLngLat().lat!==m)&&k.setLngLat([l,m]),n&&(c=k.getOffset(),d=Array.isArray(c)?c[0]:c?c.x:0,e=Array.isArray(c)?c[1]:c?c.y:0,f=Array.isArray(n)?n[0]:n?n.x:0,g=Array.isArray(n)?n[1]:n?n.y:0,d!==f||e!==g)&&k.setOffset(n),k.isDraggable()!==p&&k.setDraggable(p),k.getRotation()!==r&&k.setRotation(r),k.getRotationAlignment()!==t&&k.setRotationAlignment(t),k.getPitchAlignment()!==u&&k.setPitchAlignment(u),k.getPopup()!==q&&k.setPopup(q),(0,ax.createPortal)(a.children,k.getElement())})),(0,s.memo)((0,s.forwardRef)(function(a,b){let{map:c,mapLib:d}=(0,s.useContext)(aw),e=(0,s.useMemo)(()=>document.createElement("div"),[]),f=(0,s.useRef)({props:a});f.current.props=a;let g=(0,s.useMemo)(()=>{let b={...a},c=new d.Popup(b);return c.setLngLat([a.longitude,a.latitude]),c.once("open",a=>{var b,c;null==(c=(b=f.current.props).onOpen)||c.call(b,a)}),c},[]);if((0,s.useEffect)(()=>{let a=a=>{var b,c;null==(c=(b=f.current.props).onClose)||c.call(b,a)};return g.on("close",a),g.setDOMContent(e).addTo(c.getMap()),()=>{g.off("close",a),g.isOpen()&&g.remove()}},[]),(0,s.useEffect)(()=>{az(g.getElement(),a.style)},[a.style]),(0,s.useImperativeHandle)(b,()=>g,[]),g.isOpen()&&((g.getLngLat().lng!==a.longitude||g.getLngLat().lat!==a.latitude)&&g.setLngLat([a.longitude,a.latitude]),a.offset&&!ag(g.options.offset,a.offset)&&g.setOffset(a.offset),(g.options.anchor!==a.anchor||g.options.maxWidth!==a.maxWidth)&&(g.options.anchor=a.anchor,g.setMaxWidth(a.maxWidth)),g.options.className!==a.className)){let b=aA(g.options.className),c=aA(a.className);for(let a of b)c.has(a)||g.removeClassName(a);for(let a of c)b.has(a)||g.addClassName(a);g.options.className=a.className}return(0,ax.createPortal)(a.children,e)}));let aB=function(a,b,c,d){let e=(0,s.useContext)(aw),f=(0,s.useMemo)(()=>a(e),[]);return(0,s.useEffect)(()=>{let a=d||c||b,g="function"==typeof b&&"function"==typeof c?b:null,h="function"==typeof c?c:"function"==typeof b?b:null,{map:i}=e;return!i.hasControl(f)&&(i.addControl(f,null==a?void 0:a.position),g&&g(e)),()=>{h&&h(e),i.hasControl(f)&&i.removeControl(f)}},[]),f};(0,s.memo)(function(a){let b=aB(({mapLib:b})=>new b.AttributionControl(a),{position:a.position});return(0,s.useEffect)(()=>{az(b._container,a.style)},[a.style]),null}),(0,s.memo)(function(a){let b=aB(({mapLib:b})=>new b.FullscreenControl({container:a.containerId&&document.getElementById(a.containerId)}),{position:a.position});return(0,s.useEffect)(()=>{az(b._controlContainer,a.style)},[a.style]),null}),(0,s.memo)((0,s.forwardRef)(function(a,b){let c=(0,s.useRef)({props:a}),d=aB(({mapLib:b})=>{let d=new b.GeolocateControl(a),e=d._setupUI;return d._setupUI=a=>{d._container.hasChildNodes()||e(a)},d.on("geolocate",a=>{var b,d;null==(d=(b=c.current.props).onGeolocate)||d.call(b,a)}),d.on("error",a=>{var b,d;null==(d=(b=c.current.props).onError)||d.call(b,a)}),d.on("outofmaxbounds",a=>{var b,d;null==(d=(b=c.current.props).onOutOfMaxBounds)||d.call(b,a)}),d.on("trackuserlocationstart",a=>{var b,d;null==(d=(b=c.current.props).onTrackUserLocationStart)||d.call(b,a)}),d.on("trackuserlocationend",a=>{var b,d;null==(d=(b=c.current.props).onTrackUserLocationEnd)||d.call(b,a)}),d},{position:a.position});return c.current.props=a,(0,s.useImperativeHandle)(b,()=>d,[]),(0,s.useEffect)(()=>{az(d._container,a.style)},[a.style]),null}));let aC=(0,s.memo)(function(a){let b=aB(({mapLib:b})=>new b.NavigationControl(a),{position:a.position});return(0,s.useEffect)(()=>{az(b._container,a.style)},[a.style]),null});(0,s.memo)(function(a){let b=aB(({mapLib:b})=>new b.ScaleControl(a),{position:a.position}),c=(0,s.useRef)(a),d=c.current;c.current=a;let{style:e}=a;return void 0!==a.maxWidth&&a.maxWidth!==d.maxWidth&&(b.options.maxWidth=a.maxWidth),void 0!==a.unit&&a.unit!==d.unit&&b.setUnit(a.unit),(0,s.useEffect)(()=>{az(b._container,e)},[e]),null});let aD=a.A(85702),aE=s.forwardRef(function(a,b){return function(a,b,c){let d=(0,s.useContext)(af),[e,f]=(0,s.useState)(null),g=(0,s.useRef)(),{current:h}=(0,s.useRef)({mapLib:null,map:null});(0,s.useEffect)(()=>{let b,e=a.mapLib,i=!0;return Promise.resolve(e||c).then(c=>{if(!i)return;if(!c)throw Error("Invalid mapLib");let e="Map"in c?c:c.default;if(!e.Map)throw Error("Invalid mapLib");if(!function(a,b){for(let c of av)c in b&&(a[c]=b[c]);let{RTLTextPlugin:c="https://api.mapbox.com/mapbox-gl-js/plugins/mapbox-gl-rtl-text/v0.2.3/mapbox-gl-rtl-text.js"}=b;c&&a.getRTLTextPluginStatus&&"unavailable"===a.getRTLTextPluginStatus()&&a.setRTLTextPlugin(c,a=>{a&&console.error(a)},!0)}(e,a),!e.supported||e.supported(a))a.reuseMaps&&(b=as.reuse(a,g.current)),b||(b=new as(e.Map,a,g.current)),h.map=function(a){if(!a)return null;let b=a.map,c={getMap:()=>b,getCenter:()=>a.transform.center,getZoom:()=>a.transform.zoom,getBearing:()=>a.transform.bearing,getPitch:()=>a.transform.pitch,getPadding:()=>a.transform.padding,getBounds:()=>a.transform.getBounds(),project:c=>{let d=b.transform;b.transform=a.transform;let e=b.project(c);return b.transform=d,e},unproject:c=>{let d=b.transform;b.transform=a.transform;let e=b.unproject(c);return b.transform=d,e},queryTerrainElevation:(c,d)=>{let e=b.transform;b.transform=a.transform;let f=b.queryTerrainElevation(c,d);return b.transform=e,f},queryRenderedFeatures:(c,d)=>{let e=b.transform;b.transform=a.transform;let f=b.queryRenderedFeatures(c,d);return b.transform=e,f}};for(let a of function(a){let b=new Set,c=a;for(;c;){for(let d of Object.getOwnPropertyNames(c))"_"!==d[0]&&"function"==typeof a[d]&&"fire"!==d&&"setEventedParent"!==d&&b.add(d);c=Object.getPrototypeOf(c)}return Array.from(b)}(b))a in c||at.includes(a)||(c[a]=b[a].bind(b));return c}(b),h.mapLib=e,f(b),null==d||d.onMapMount(h.map,a.id);else throw Error("Map is not supported by this browser")}).catch(b=>{let{onError:c}=a;c?c({type:"error",target:null,originalEvent:null,error:b}):console.error(b)}),()=>{i=!1,b&&(null==d||d.onMapUnmount(a.id),a.reuseMaps?b.recycle():b.destroy())}},[]),au(()=>{e&&e.setProps(a)}),(0,s.useImperativeHandle)(b,()=>h.map,[e]);let i=(0,s.useMemo)(()=>({position:"relative",width:"100%",height:"100%",...a.style}),[a.style]);return s.createElement("div",{id:a.id,ref:g,style:i},e&&s.createElement(aw.Provider,{value:h},s.createElement("div",{"mapboxgl-children":"",style:{height:"100%"}},a.children)))}(a,b,aD)}),aF=1,aG=1;class aH{time=0;channels=new Map;animations=new Map;playing=!1;lastEngineTime=-1;addChannel(a){let{delay:b=0,duration:c=1/0,rate:d=1,repeat:e=1}=a,f=aF++,g={time:0,delay:b,duration:c,rate:d,repeat:e};return this._setChannelTime(g,this.time),this.channels.set(f,g),f}removeChannel(a){for(let[b,c]of(this.channels.delete(a),this.animations))c.channel===a&&this.detachAnimation(b)}isFinished(a){let b=this.channels.get(a);return void 0!==b&&this.time>=b.delay+b.duration*b.repeat}getTime(a){if(void 0===a)return this.time;let b=this.channels.get(a);return void 0===b?-1:b.time}setTime(a){for(let b of(this.time=Math.max(0,a),this.channels.values()))this._setChannelTime(b,this.time);for(let a of this.animations.values()){let{animation:b,channel:c}=a;b.setTime(this.getTime(c))}}play(){this.playing=!0}pause(){this.playing=!1,this.lastEngineTime=-1}reset(){this.setTime(0)}attachAnimation(a,b){let c=aG++;return this.animations.set(c,{animation:a,channel:b}),a.setTime(this.getTime(b)),c}detachAnimation(a){this.animations.delete(a)}update(a){this.playing&&(-1===this.lastEngineTime&&(this.lastEngineTime=a),this.setTime(this.time+(a-this.lastEngineTime)),this.lastEngineTime=a)}_setChannelTime(a,b){let c=b-a.delay;c>=a.duration*a.repeat?a.time=a.duration*a.rate:(a.time=Math.max(0,c)%a.duration,a.time*=a.rate)}}let aI={number:{type:"number",validate:(a,b)=>Number.isFinite(a)&&"object"==typeof b&&(void 0===b.max||a<=b.max)&&(void 0===b.min||a>=b.min)},array:{type:"array",validate:(a,b)=>Array.isArray(a)||ArrayBuffer.isView(a)}};function aJ(a){return Array.isArray(a)||ArrayBuffer.isView(a)?"array":typeof a}let aK={vertex:`\
#ifdef MODULE_LOGDEPTH
  logdepth_adjustPosition(gl_Position);
#endif
`,fragment:`\
#ifdef MODULE_MATERIAL
  fragColor = material_filterColor(fragColor);
#endif

#ifdef MODULE_LIGHTING
  fragColor = lighting_filterColor(fragColor);
#endif

#ifdef MODULE_FOG
  fragColor = fog_filterColor(fragColor);
#endif

#ifdef MODULE_PICKING
  fragColor = picking_filterHighlightColor(fragColor);
  fragColor = picking_filterPickingColor(fragColor);
#endif

#ifdef MODULE_LOGDEPTH
  logdepth_setFragDepth();
#endif
`},aL=/void\s+main\s*\([^)]*\)\s*\{\n?/,aM=/}\n?[^{}]*$/,aN=[],aO="__LUMA_INJECT_DECLARATIONS__";function aP(a,b,c,d=!1){let e="vertex"===b;for(let b in c){let d=c[b];d.sort((a,b)=>a.order-b.order),aN.length=d.length;for(let a=0,b=d.length;a<b;++a)aN[a]=d[a].injection;let f=`${aN.join("\n")}
`;switch(b){case"vs:#decl":e&&(a=a.replace(aO,f));break;case"vs:#main-start":e&&(a=a.replace(aL,a=>a+f));break;case"vs:#main-end":e&&(a=a.replace(aM,a=>f+a));break;case"fs:#decl":e||(a=a.replace(aO,f));break;case"fs:#main-start":e||(a=a.replace(aL,a=>a+f));break;case"fs:#main-end":e||(a=a.replace(aM,a=>f+a));break;default:a=a.replace(b,a=>a+f)}}return a=a.replace(aO,""),d&&(a=a.replace(/\}\s*$/,a=>a+aK[b])),a}function aQ(a){a.map(a=>(function(a){var b;if(a.instance)return;aQ(a.dependencies||[]);let{propTypes:c={},deprecations:d=[],inject:e={}}=a,f={normalizedInjections:function(a){let b={vertex:{},fragment:{}};for(let c in a){let d=a[c];"string"==typeof d&&(d={order:0,injection:d}),b[function(a){let b=a.slice(0,2);switch(b){case"vs":return"vertex";case"fs":return"fragment";default:throw Error(b)}}(c)][c]=d}return b}(e),parsedDeprecations:((b=d).forEach(a=>{"function"===a.type?a.regex=RegExp(`\\b${a.old}\\(`):a.regex=RegExp(`${a.type} ${a.old};`)}),b)};c&&(f.propValidators=function(a){let b={};for(let[c,d]of Object.entries(a))b[c]=function(a){let b=aJ(a);if("object"!==b)return{value:a,...aI[b],type:b};if("object"==typeof a)return a?void 0!==a.type?{...a,...aI[a.type],type:a.type}:void 0===a.value?{type:"object",value:a}:(b=aJ(a.value),{...a,...aI[b],type:b}):{type:"object",value:null};throw Error("props")}(d);return b}(c)),a.instance=f;let g={};c&&(g=Object.entries(c).reduce((a,[b,c])=>{let d=c?.value;return d&&(a[b]=d),a},{})),a.defaultUniforms={...a.defaultUniforms,...g}})(a))}function aR(a,b,c){a.deprecations?.forEach(a=>{a.regex?.test(b)&&(a.deprecated?c.deprecated(a.old,a.new)():c.removed(a.old,a.new)())})}function aS(a){aQ(a);let b={},c={};!function a(b){let{modules:c,level:d,moduleMap:e,moduleDepth:f}=b;if(d>=5)throw Error("Possible loop in shader dependency graph");for(let a of c)e[a.name]=a,(void 0===f[a.name]||f[a.name]<d)&&(f[a.name]=d);for(let b of c)b.dependencies&&a({modules:b.dependencies,level:d+1,moduleMap:e,moduleDepth:f})}({modules:a,level:0,moduleMap:b,moduleDepth:c});let d=Object.keys(c).sort((a,b)=>c[b]-c[a]).map(a=>b[a]);return aQ(d),d}let aT=[[/^(#version[ \t]+(100|300[ \t]+es))?[ \t]*\n/,"#version 300 es\n"],[/\btexture(2D|2DProj|Cube)Lod(EXT)?\(/g,"textureLod("],[/\btexture(2D|2DProj|Cube)(EXT)?\(/g,"texture("]],aU=[...aT,[aX("attribute"),"in $1"],[aX("varying"),"out $1"]],aV=[...aT,[aX("varying"),"in $1"]];function aW(a,b){for(let[c,d]of b)a=a.replace(c,d);return a}function aX(a){return RegExp(`\\b${a}[ \\t]+(\\w+[ \\t]+\\w+(\\[\\w+\\])?;)`,"g")}function aY(a,b){if(!a){let a=Error(b||"shadertools: assertion failed.");throw Error.captureStackTrace?.(a,aY),a}}let aZ=/^(?:uniform\s+)?(?:(?:lowp|mediump|highp)\s+)?[A-Za-z0-9_]+(?:<[^>]+>)?\s+([A-Za-z0-9_]+)(?:\s*\[[^\]]+\])?\s*;/,a$=/((?:layout\s*\([^)]*\)\s*)*)uniform\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{([\s\S]*?)\}\s*([A-Za-z_][A-Za-z0-9_]*)?\s*;/g;function a_(a){return`${a.name}Uniforms`}function a0(a){let b=[];for(let c of a.replace(/\/\*[\s\S]*?\*\//g,"").replace(/\/\/.*$/gm,"").matchAll(a$)){let a=c[1]?.trim()||null;b.push({blockName:c[2],body:c[3],instanceName:c[4]||null,layoutQualifier:a,hasLayoutQualifier:!!a,isStd140:!!(a&&/\blayout\s*\([^)]*\bstd140\b[^)]*\)/.exec(a))})}return b}function a1(a,b=8){if(a.length<=b)return a.join(", ");let c=a.length-b;return`${a.slice(0,b).join(", ")}, ... (${c} more)`}function a2(a,b){let c="";for(let d in a){let e=a[d];if(c+=`void ${e.signature} {
`,e.header&&(c+=`  ${e.header}`),b[d]){let a=b[d];for(let b of(a.sort((a,b)=>a.order-b.order),a))c+=`  ${b.injection}
`}e.footer&&(c+=`  ${e.footer}`),c+="}\n"}return c}function a3(a){let b={vertex:{},fragment:{}};for(let c of a){let a,d;"string"!=typeof c?d=(a=c).hook:(a={},d=c);let[e,f]=(d=d.trim()).split(":"),g=d.replace(/\(.+/,""),h=Object.assign(a,{signature:f});switch(e){case"vs":b.vertex[g]=h;break;case"fs":b.fragment[g]=h;break;default:throw Error(e)}}return b}let a4="(?:var<\\s*(uniform|storage(?:\\s*,\\s*[A-Za-z_][A-Za-z0-9_]*)?)\\s*>|var)\\s+([A-Za-z_][A-Za-z0-9_]*)",a5=[RegExp(`@binding\\(\\s*(auto|\\d+)\\s*\\)\\s*@group\\(\\s*(\\d+)\\s*\\)\\s*${a4}`,"g"),RegExp(`@group\\(\\s*(\\d+)\\s*\\)\\s*@binding\\(\\s*(auto|\\d+)\\s*\\)\\s*${a4}`,"g")],a6=[RegExp(`@binding\\(\\s*(auto|\\d+)\\s*\\)\\s*@group\\(\\s*(\\d+)\\s*\\)\\s*${a4}`,"g"),RegExp(`@group\\(\\s*(\\d+)\\s*\\)\\s*@binding\\(\\s*(auto|\\d+)\\s*\\)\\s*${a4}`,"g")],a7=[RegExp(`@binding\\(\\s*(\\d+)\\s*\\)\\s*@group\\(\\s*(\\d+)\\s*\\)\\s*${a4}`,"g"),RegExp(`@group\\(\\s*(\\d+)\\s*\\)\\s*@binding\\(\\s*(\\d+)\\s*\\)\\s*${a4}`,"g")],a8=[RegExp(`@binding\\(\\s*(auto)\\s*\\)\\s*@group\\(\\s*(\\d+)\\s*\\)\\s*${a4}`,"g"),RegExp(`@group\\(\\s*(\\d+)\\s*\\)\\s*@binding\\(\\s*(auto)\\s*\\)\\s*${a4}`,"g"),RegExp(`@binding\\(\\s*(auto)\\s*\\)\\s*@group\\(\\s*(\\d+)\\s*\\)(?:[\\s\\n\\r]*@[A-Za-z_][^\\n\\r]*)*[\\s\\n\\r]*${a4}`,"g"),RegExp(`@group\\(\\s*(\\d+)\\s*\\)\\s*@binding\\(\\s*(auto)\\s*\\)(?:[\\s\\n\\r]*@[A-Za-z_][^\\n\\r]*)*[\\s\\n\\r]*${a4}`,"g")];function a9(a){let b=a.split(""),c=0,d=0,e=!1,f=!1,g=!1;for(;c<a.length;){let h=a[c],i=a[c+1];if(f){g?g=!1:"\\"===h?g=!0:'"'===h&&(f=!1),c++;continue}if(e){"\n"===h||"\r"===h?e=!1:b[c]=" ",c++;continue}if(d>0){if("/"===h&&"*"===i){b[c]=" ",b[c+1]=" ",d++,c+=2;continue}if("*"===h&&"/"===i){b[c]=" ",b[c+1]=" ",d--,c+=2;continue}"\n"!==h&&"\r"!==h&&(b[c]=" "),c++;continue}if('"'===h){f=!0,c++;continue}if("/"===h&&"/"===i){b[c]=" ",b[c+1]=" ",e=!0,c+=2;continue}if("/"===h&&"*"===i){b[c]=" ",b[c+1]=" ",d=1,c+=2;continue}c++}return b.join("")}function ba(a,b){let c=a9(a),d=[];for(let e of b){let f;for(e.lastIndex=0,f=e.exec(c);f;){let g=e===b[0],h=f.index,i=f[0].length;d.push({match:a.slice(h,h+i),index:h,length:i,bindingToken:f[g?1:2],groupToken:f[g?2:1],accessDeclaration:f[3]?.trim(),name:f[4]}),f=e.exec(c)}}return d.sort((a,b)=>a.index-b.index)}function bb(a,b,c){let d=ba(a,b);if(!d.length)return a;let e="",f=0;for(let b of d)e+=a.slice(f,b.index),e+=c(b),f=b.index+b.length;return e+a.slice(f)}function bc(a){return/@binding\(\s*auto\s*\)/.test(a9(a))}let bd=[RegExp(`@binding\\(\\s*(\\d+)\\s*\\)\\s*@group\\(\\s*(\\d+)\\s*\\)\\s*${a4}\\s*:\\s*([^;]+);`,"g"),RegExp(`@group\\(\\s*(\\d+)\\s*\\)\\s*@binding\\(\\s*(\\d+)\\s*\\)\\s*${a4}\\s*:\\s*([^;]+);`,"g")];function be(a,b=[]){let c=a9(a),d=new Map;for(let a of b)d.set(bf(a.name,a.group,a.location),a.moduleName);let e=[];for(let a of bd){let b;for(a.lastIndex=0,b=a.exec(c);b;){let f=a===bd[0],g=Number(b[f?1:2]),h=Number(b[f?2:1]),i=b[3]?.trim(),j=b[4],k=b[5].trim(),l=d.get(bf(j,h,g));e.push(function(a){var b;let c={name:a.name,group:a.group,binding:a.binding,owner:a.owner,kind:"unknown",moduleName:a.moduleName,resourceType:a.resourceType};if(a.accessDeclaration){let b=a.accessDeclaration.split(",").map(a=>a.trim());if("uniform"===b[0])return{...c,kind:"uniform",access:"uniform"};if("storage"===b[0]){let a=b[1]||"read_write";return{...c,kind:"read"===a?"read-only-storage":"storage",access:a}}}return"sampler"===a.resourceType||"sampler_comparison"===a.resourceType?{...c,kind:"sampler",samplerKind:"sampler_comparison"===a.resourceType?"comparison":"filtering"}:a.resourceType.startsWith("texture_storage_")?{...c,kind:"storage-texture",access:function(a){let b=/,\s*([A-Za-z_][A-Za-z0-9_]*)\s*>$/.exec(a);return b?.[1]}(a.resourceType),viewDimension:bg(a.resourceType)}:a.resourceType.startsWith("texture_")?{...c,kind:"texture",viewDimension:bg(a.resourceType),sampleType:(b=a.resourceType).startsWith("texture_depth_")?"depth":b.includes("<i32>")?"sint":b.includes("<u32>")?"uint":b.includes("<f32>")?"float":void 0,multisampled:a.resourceType.startsWith("texture_multisampled_")}:c}({name:j,group:h,binding:g,owner:l?"module":"application",moduleName:l,accessDeclaration:i,resourceType:k})),b=a.exec(c)}}return e.sort((a,b)=>a.group!==b.group?a.group-b.group:a.binding!==b.binding?a.binding-b.binding:a.name.localeCompare(b.name))}function bf(a,b,c){return`${b}:${c}:${a}`}function bg(a){return a.includes("cube_array")?"cube-array":a.includes("2d_array")?"2d-array":a.includes("cube")?"cube":a.includes("3d")?"3d":a.includes("2d")?"2d":a.includes("1d")?"1d":void 0}let bh=`

${aO}
`,bi=`\
precision highp float;
`;function bj(a,b){let{source:c,stage:d,language:e="glsl",modules:f,defines:g={},hookFunctions:h=[],inject:i={},prologue:j=!0,log:k}=b;aY("string"==typeof c,"shader source must be a string");let l="glsl"===e?({name:function(a,b="unnamed"){let c=/#define[^\S\r\n]*SHADER_NAME[^\S\r\n]*([A-Za-z0-9_-]+)\s*/.exec(a);return c?c[1]:b}(c,void 0),language:"glsl",version:function(a){let b=100,c=a.match(/[^\s]+/g);if(c&&c.length>=2&&"#version"===c[0]){let a=parseInt(c[1],10);Number.isFinite(a)&&(b=a)}if(100!==b&&300!==b)throw Error(`Invalid GLSL version ${b}`);return b}(c)}).version:-1,m=a.shaderLanguageVersion,n=100===l?"#version 100":"#version 300 es",o=c.split("\n").slice(1).join("\n"),p={};f.forEach(a=>{Object.assign(p,a.defines)}),Object.assign(p,g);let q="";switch(e){case"wgsl":break;case"glsl":q=j?`\
${n}

// ----- PROLOGUE -------------------------
#define SHADER_TYPE_${d.toUpperCase()}

${function(a){switch(a?.gpu.toLowerCase()){case"apple":return`\
#define APPLE_GPU
// Apple optimizes away the calculation necessary for emulated fp64
#define LUMA_FP64_CODE_ELIMINATION_WORKAROUND 1
#define LUMA_FP32_TAN_PRECISION_WORKAROUND 1
// Intel GPU doesn't have full 32 bits precision in same cases, causes overflow
#define LUMA_FP64_HIGH_BITS_OVERFLOW_WORKAROUND 1
`;case"nvidia":return`\
#define NVIDIA_GPU
// Nvidia optimizes away the calculation necessary for emulated fp64
#define LUMA_FP64_CODE_ELIMINATION_WORKAROUND 1
`;case"intel":return`\
#define INTEL_GPU
// Intel optimizes away the calculation necessary for emulated fp64
#define LUMA_FP64_CODE_ELIMINATION_WORKAROUND 1
// Intel's built-in 'tan' function doesn't have acceptable precision
#define LUMA_FP32_TAN_PRECISION_WORKAROUND 1
// Intel GPU doesn't have full 32 bits precision in same cases, causes overflow
#define LUMA_FP64_HIGH_BITS_OVERFLOW_WORKAROUND 1
`;case"amd":return`\
#define AMD_GPU
`;default:return`\
#define DEFAULT_GPU
// Prevent driver from optimizing away the calculation necessary for emulated fp64
#define LUMA_FP64_CODE_ELIMINATION_WORKAROUND 1
// Headless Chrome's software shader 'tan' function doesn't have acceptable precision
#define LUMA_FP32_TAN_PRECISION_WORKAROUND 1
// If the GPU doesn't have full 32 bits precision, will causes overflow
#define LUMA_FP64_HIGH_BITS_OVERFLOW_WORKAROUND 1
`}}(a)}
${"fragment"===d?bi:""}

// ----- APPLICATION DEFINES -------------------------

${function(a={}){let b="";for(let c in a){let d=a[c];(d||Number.isFinite(d))&&(b+=`#define ${c.toUpperCase()} ${a[c]}
`)}return b}(p)}

`:`${n}
`}let r=a3(h),s={},t={},u={};for(let a in i){let b="string"==typeof i[a]?{injection:i[a],order:0}:i[a],c=/^(v|f)s:(#)?([\w-]+)$/.exec(a);if(c){let d=c[2],e=c[3];d?"decl"===e?t[a]=[b]:u[a]=[b]:s[a]=[b]}else u[a]=[b]}for(let a of f){k&&aR(a,o,k),q+=bl(a,d,k);let b=a.instance?.normalizedInjections[d]||{};for(let a in b){let c=/^(v|f)s:#([\w-]+)$/.exec(a);if(c){let d="decl"===c[2]?t:u;d[a]=d[a]||[],d[a].push(b[a])}else s[a]=s[a]||[],s[a].push(b[a])}}return q+="// ----- MAIN SHADER SOURCE -------------------------",q+=bh,q=aP(q,d,t)+a2(r[d],s)+o,q=aP(q,d,u),"glsl"===e&&l!==m&&(q=function(a,b){if(300!==Number(a.match(/^#version[ \t]+(\d+)/m)?.[1]||100))throw Error("luma.gl v9 only supports GLSL 3.00 shader sources");switch(b){case"vertex":return aW(a,aU);case"fragment":return aW(a,aV);default:throw Error(b)}}(q,d)),"glsl"===e&&function(a,b,c){let d=a0(a).filter(a=>!a.isStd140),e=new Set;for(let a of d){if(e.has(a.blockName))continue;e.add(a.blockName);let d=a.hasLayoutQualifier?`declares ${a.layoutQualifier.replace(/\s+/g," ").trim()} instead of layout(std140)`:"does not declare layout(std140)",f=`${b} shader uniform block ${a.blockName} ${d}. luma.gl host-side shader block packing assumes explicit layout(std140) for GLSL uniform blocks. Add \`layout(std140)\` to the block declaration.`;c?.warn?.(f,a)()}}(q,d,k),q.trim()}function bk(a){return function(b){let c={};for(let d of a){let a=d.getUniforms?.(b,c);Object.assign(c,a)}return c}}function bl(a,b,c){let d;switch(b){case"vertex":d=a.vs||"";break;case"fragment":d=a.fs||"";break;case"wgsl":d=a.source||"";break;default:aY(!1)}if(!a.name)throw Error("Shader module must have a name");!function(a,b,c={}){let d=function(a,b){let c,d=Object.keys(a.uniformTypes||{});if(!d.length)return null;let e=(c="wgsl"===b?a.source:"vertex"===b?a.vs:a.fs)?function(a,b,c){var d,e;let f,g="wgsl"===b?function(a,b){let c=RegExp(`\\bstruct\\s+${b}\\b`,"m").exec(a);if(!c)return null;let d=a.indexOf("{",c.index);if(d<0)return null;let e=0;for(let b=d;b<a.length;b++){let c=a[b];if("{"===c){e++;continue}if("}"===c&&0==--e)return a.slice(d+1,b)}return null}(a,c):(d=a,e=c,f=a0(d).find(a=>a.blockName===e),f?.body||null);if(!g)return null;let h=[];for(let a of g.split("\n")){let c=a.replace(/\/\/.*$/,"").trim();if(!c||c.startsWith("#"))continue;let d="wgsl"===b?c.match(/^([A-Za-z0-9_]+)\s*:/):c.match(aZ);d&&h.push(d[1])}return h}(c,"wgsl"===b?"wgsl":"glsl",a_(a)):null;return e?{moduleName:a.name,uniformBlockName:a_(a),stage:b,expectedUniformNames:d,actualUniformNames:e,matches:function(a,b){if(a.length!==b.length)return!1;for(let c=0;c<a.length;c++)if(a[c]!==b[c])return!1;return!0}(d,e)}:null}(a,b);if(!d||d.matches)return;let e=function(a){let{expectedUniformNames:b,actualUniformNames:c}=a,d=b.filter(a=>!c.includes(a)),e=c.filter(a=>!b.includes(a)),f=[`Expected ${b.length} fields, found ${c.length}.`],g=function(a,b){let c=Math.min(a.length,b.length);for(let d=0;d<c;d++)if(a[d]!==b[d])return`First mismatch at field ${d+1}: expected ${a[d]}, found ${b[d]}.`;return a.length>b.length?`Shader block ends after field ${b.length}; expected next field ${a[b.length]}.`:b.length>a.length?`Shader block has extra field ${b.length}: ${b[a.length]}.`:null}(b,c);return g&&f.push(g),d.length&&f.push(`Missing from shader block (${d.length}): ${a1(d)}.`),e.length&&f.push(`Unexpected in shader block (${e.length}): ${a1(e)}.`),b.length<=12&&c.length<=12&&(d.length||e.length)&&(f.push(`Expected: ${b.join(", ")}.`),f.push(`Actual: ${c.join(", ")}.`)),`${a.moduleName}: ${a.stage} shader uniform block ${a.uniformBlockName} does not match module.uniformTypes. ${f.join(" ")}`}(d);c.log?.error?.(e,d)(),!1!==c.throwOnError&&aY(!1,e)}(a,b,{log:c});let e=a.name.toUpperCase().replace(/[^0-9a-z]/gi,"_"),f=`\
// ----- MODULE ${a.name} ---------------

`;return"wgsl"!==b&&(f+=`#define MODULE_${e}
`),f+=`${d}
`}function bm(a,b,c){if(0===a&&b>=100)throw Error(`Application binding "${c}" in group 0 uses reserved binding ${b}. Application-owned explicit group-0 bindings must stay below 100.`)}function bn(a,b,c,d){if(0===b&&c<100)throw Error(`Module "${a}" binding "${d}" in group 0 uses reserved application binding ${c}. Module-owned explicit group-0 bindings must be 100 or higher.`)}function bo(a,b,c,d){let e=a.get(b)||new Set;if(e.has(c))throw Error(`Duplicate WGSL binding assignment for ${d}: group ${b}, binding ${c}.`);e.add(c),a.set(b,e)}function bp(a,b,c){let d=b.get(a)||new Set,e=c??(0===a?100:d.size>0?Math.max(...d)+1:0);for(;d.has(e);)e++;return e}function bq(a,b,c){return`${a}:${b}:${c}`}let br="([a-zA-Z_][a-zA-Z0-9_]*)",bs=RegExp(`^\\s*\\#\\s*ifdef\\s*${br}\\s*$`),bt=RegExp(`^\\s*\\#\\s*ifndef\\s*${br}\\s*(?:\\/\\/.*)?$`),bu=/^\s*\#\s*else\s*(?:\/\/.*)?$/,bv=/^\s*\#\s*endif\s*$/,bw=RegExp(`^\\s*\\#\\s*ifdef\\s*${br}\\s*(?:\\/\\/.*)?$`),bx=/^\s*\#\s*endif\s*(?:\/\/.*)?$/;class by{static defaultShaderAssembler;_hookFunctions=[];_defaultModules=[];_wgslBindingRegistry=new Map;static getDefaultShaderAssembler(){return by.defaultShaderAssembler=by.defaultShaderAssembler||new by,by.defaultShaderAssembler}addDefaultModule(a){this._defaultModules.find(b=>b.name===("string"==typeof a?a:a.name))||this._defaultModules.push(a)}removeDefaultModule(a){let b="string"==typeof a?a:a.name;this._defaultModules=this._defaultModules.filter(a=>a.name!==b)}addShaderHook(a,b){b&&(a=Object.assign(b,{hook:a})),this._hookFunctions.push(a)}assembleWGSLShader(a){let b=this._getModuleList(a.modules),c=this._hookFunctions,{source:d,getUniforms:e,bindingAssignments:f}=function(a){let b=aS(a.modules||[]),{source:c,bindingAssignments:d}=function(a,b){let{source:c,stage:d,modules:e,hookFunctions:f=[],inject:g={},log:h}=b;aY("string"==typeof c,"shader source must be a string");let i="",j=a3(f),k={},l={},m={};for(let a in g){let b="string"==typeof g[a]?{injection:g[a],order:0}:g[a],c=/^(v|f)s:(#)?([\w-]+)$/.exec(a);if(c){let d=c[2],e=c[3];d?"decl"===e?l[a]=[b]:m[a]=[b]:k[a]=[b]}else m[a]=[b]}let n=function(a){let b=ba(a,a6),c=new Map;for(let a of b){if("auto"===a.bindingToken)continue;let b=Number(a.bindingToken),d=Number(a.groupToken);bm(d,b,a.name),bo(c,d,b,`application binding "${a.name}"`)}let d={sawSupportedBindingDeclaration:b.length>0},e=bb(a,a6,a=>(function(a,b,c){let{match:d,bindingToken:e,groupToken:f,name:g}=a,h=Number(f);if("auto"===e){let a=function(a,b){let c=b.get(a)||new Set,d=0;for(;c.has(d);)d++;return d}(h,b);return bm(h,a,g),bo(b,h,a,`application binding "${g}"`),d.replace(/@binding\(\s*auto\s*\)/,`@binding(${a})`)}return c.sawSupportedBindingDeclaration=!0,d})(a,c,d));if(bc(a)&&!d.sawSupportedBindingDeclaration)throw Error('Unsupported @binding(auto) declaration form in application WGSL. Use adjacent "@group(N)" and "@binding(auto)" decorators followed by a bindable "var" declaration.');return{source:e}}(c),o=function(a){let b=new Map;for(let c of ba(a,a7)){let a=Number(c.bindingToken),d=Number(c.groupToken);bm(d,a,c.name),bo(b,d,a,`application binding "${c.name}"`)}return b}(n.source),p=function(a,b,c){let d=new Map;if(!b)return d;for(let e of a)for(let a of function(a){let b=[];for(let c of ba(a.source||"",a5))b.push({name:c.name,group:Number(c.groupToken)});return b}(e)){let f=bq(a.group,e.name,a.name),g=b.get(f);if(void 0!==g){let b=d.get(a.group)||new Map,e=b.get(g);if(e&&e!==f)throw Error(`Duplicate WGSL binding reservation for modules "${e}" and "${f}": group ${a.group}, binding ${g}.`);bo(c,a.group,g,`registered module binding "${f}"`),b.set(g,f),d.set(a.group,b)}}return d}(e,b._bindingRegistry,o),q=[];for(let a of e){h&&aR(a,c,h);let e=function(a,b,c){let d=[],e={sawSupportedBindingDeclaration:ba(a,a5).length>0,nextHintedBindingLocation:"number"==typeof b.firstBindingSlot?b.firstBindingSlot:null},f=bb(a,a5,a=>(function(a,b){let{module:c,context:d,bindingAssignments:e,relocationState:f}=b,{match:g,bindingToken:h,groupToken:i,name:j}=a,k=Number(i);if("auto"===h){let a=bq(k,c.name,j),b=d.bindingRegistry?.get(a),h=void 0!==b?b:null===f.nextHintedBindingLocation?bp(k,d.usedBindingsByGroup):bp(k,d.usedBindingsByGroup,f.nextHintedBindingLocation);return(bn(c.name,k,h,j),void 0!==b&&function(a,b,c,d){let e=a.get(b);if(!e)return!1;let f=e.get(c);if(!f)return!1;if(f!==d)throw Error(`Registered module binding "${d}" collided with "${f}": group ${b}, binding ${c}.`);return!0}(d.reservedBindingKeysByGroup,k,h,a))?e.push({moduleName:c.name,name:j,group:k,location:h}):(bo(d.usedBindingsByGroup,k,h,`module "${c.name}" binding "${j}"`),d.bindingRegistry?.set(a,h),e.push({moduleName:c.name,name:j,group:k,location:h}),null!==f.nextHintedBindingLocation&&void 0===b&&(f.nextHintedBindingLocation=h+1)),g.replace(/@binding\(\s*auto\s*\)/,`@binding(${h})`)}let l=Number(h);return bn(c.name,k,l,j),bo(d.usedBindingsByGroup,k,l,`module "${c.name}" binding "${j}"`),e.push({moduleName:c.name,name:j,group:k,location:l}),g})(a,{module:b,context:c,bindingAssignments:d,relocationState:e}));if(bc(a)&&!e.sawSupportedBindingDeclaration)throw Error(`Unsupported @binding(auto) declaration form in module "${b.name}". Use adjacent "@group(N)" and "@binding(auto)" decorators followed by a bindable "var" declaration.`);return{source:f,bindingAssignments:d}}(bl(a,"wgsl",h),a,{usedBindingsByGroup:o,bindingRegistry:b._bindingRegistry,reservedBindingKeysByGroup:p});q.push(...e.bindingAssignments),i+=e.source;let f=a.injections?.[d]||{};for(let a in f){let b=/^(v|f)s:#([\w-]+)$/.exec(a);if(b){let c="decl"===b[2]?l:m;c[a]=c[a]||[],c[a].push(f[a])}else k[a]=k[a]||[],k[a].push(f[a])}}return i+=bh,i=aP(i,d,l)+a2(j[d],k)+function(a){if(0===a.length)return"";let b="// ----- MODULE WGSL BINDING ASSIGNMENTS ---------------\n";for(let c of a)b+=`// ${c.moduleName}.${c.name} -> @group(${c.group}) @binding(${c.location})
`;return b+"\n"}(q)+n.source,function(a){var b,c;let d,e=ba(a,a5==a5||a5===a6?a8:a5).find(a=>"auto"===a.bindingToken);if(!e)return;let f=function(a,b){let c,d,e=/^\/\/ ----- MODULE ([^\n]+) ---------------$/gm;for(d=e.exec(a);d&&d.index<=b;)c=d[1],d=e.exec(a);return c}(a,e.index);if(f)throw Error(`Unresolved @binding(auto) for module "${f}" binding "${e.name}" remained in assembled WGSL source.`);if(b=a,c=e.index,!((d=b.indexOf(bh))>=0)||c>d)throw Error(`Unresolved @binding(auto) for application binding "${e.name}" remained in assembled WGSL source.`);throw Error(`Unresolved @binding(auto) remained in assembled WGSL source near "${e.match.replace(/\s+/g," ").trim()}".`)}(i=aP(i,d,m)),{source:i,bindingAssignments:q}}(a.platformInfo,{...a,source:a.source,stage:"vertex",modules:b});return{source:c,getUniforms:bk(b),bindingAssignments:d,bindingTable:be(c,d)}}({...a,source:a.source,_bindingRegistry:this._wgslBindingRegistry,modules:b,hookFunctions:c}),g={...b.reduce((a,b)=>(Object.assign(a,b.defines),a),{}),...a.defines},h="wgsl"===a.platformInfo.shaderLanguage?function(a,b){let c=a.split("\n"),d=[],e=[],f=!0;for(let a of c){let c=a.match(bw)||a.match(bs),g=a.match(bt),h=a.match(bu),i=a.match(bx)||a.match(bv);if(c||g){let a=(c||g)?.[1],d=!!b?.defines?.[a],h=c?d:!d,i=f&&h;e.push({parentActive:f,branchTaken:h,active:i}),f=i}else if(h){let a=e[e.length-1];if(!a)throw Error("Encountered #else without matching #ifdef or #ifndef");a.active=a.parentActive&&!a.branchTaken,a.branchTaken=!0,f=a.active}else i?(e.pop(),f=!e.length||e[e.length-1].active):f&&d.push(a)}if(e.length>0)throw Error("Unterminated conditional block in shader source");return d.join("\n")}(d,{defines:g}):d;return{source:h,getUniforms:e,modules:b,bindingAssignments:f,bindingTable:be(h,f)}}assembleGLSLShaderPair(a){let b=this._getModuleList(a.modules),c=this._hookFunctions;return{...function(a){let{vs:b,fs:c}=a,d=aS(a.modules||[]);return{vs:bj(a.platformInfo,{...a,source:b,stage:"vertex",modules:d}),fs:bj(a.platformInfo,{...a,source:c,stage:"fragment",modules:d}),getUniforms:bk(d)}}({...a,vs:a.vs,fs:a.fs,modules:b,hookFunctions:c}),modules:b}}_getModuleList(a=[]){let b=Array(this._defaultModules.length+a.length),c={},d=0;for(let a=0,e=this._defaultModules.length;a<e;++a){let e=this._defaultModules[a],f=e.name;b[d++]=e,c[f]=!0}for(let e=0,f=a.length;e<f;++e){let f=a[e],g=f.name;c[g]||(b[d++]=f,c[g]=!0)}return b.length=d,aQ(b),b}}let bz=`\
layout(std140) uniform floatColorsUniforms {
  float useByteColors;
} floatColors;

vec3 floatColors_normalize(vec3 inputColor) {
  return floatColors.useByteColors > 0.5 ? inputColor / 255.0 : inputColor;
}

vec4 floatColors_normalize(vec4 inputColor) {
  return floatColors.useByteColors > 0.5 ? inputColor / 255.0 : inputColor;
}

vec4 floatColors_premultiplyAlpha(vec4 inputColor) {
  return vec4(inputColor.rgb * inputColor.a, inputColor.a);
}

vec4 floatColors_unpremultiplyAlpha(vec4 inputColor) {
  return inputColor.a > 0.0 ? vec4(inputColor.rgb / inputColor.a, inputColor.a) : vec4(0.0);
}

vec4 floatColors_premultiply_alpha(vec4 inputColor) {
  return floatColors_premultiplyAlpha(inputColor);
}

vec4 floatColors_unpremultiply_alpha(vec4 inputColor) {
  return floatColors_unpremultiplyAlpha(inputColor);
}
`,bA={name:"floatColors",props:{},uniforms:{},vs:bz,fs:bz,source:`\
struct floatColorsUniforms {
  useByteColors: f32
};

@group(0) @binding(auto) var<uniform> floatColors : floatColorsUniforms;

fn floatColors_normalize(inputColor: vec3<f32>) -> vec3<f32> {
  return select(inputColor, inputColor / 255.0, floatColors.useByteColors > 0.5);
}

fn floatColors_normalize4(inputColor: vec4<f32>) -> vec4<f32> {
  return select(inputColor, inputColor / 255.0, floatColors.useByteColors > 0.5);
}

fn floatColors_premultiplyAlpha(inputColor: vec4<f32>) -> vec4<f32> {
  return vec4<f32>(inputColor.rgb * inputColor.a, inputColor.a);
}

fn floatColors_unpremultiplyAlpha(inputColor: vec4<f32>) -> vec4<f32> {
  return select(
    vec4<f32>(0.0),
    vec4<f32>(inputColor.rgb / inputColor.a, inputColor.a),
    inputColor.a > 0.0
  );
}

fn floatColors_premultiply_alpha(inputColor: vec4<f32>) -> vec4<f32> {
  return floatColors_premultiplyAlpha(inputColor);
}

fn floatColors_unpremultiply_alpha(inputColor: vec4<f32>) -> vec4<f32> {
  return floatColors_unpremultiplyAlpha(inputColor);
}
`,uniformTypes:{useByteColors:"f32"},defaultUniforms:{useByteColors:!0}};var bB=a.i(83390);let bC=`\
precision highp int;

// #if (defined(SHADER_TYPE_FRAGMENT) && defined(LIGHTING_FRAGMENT)) || (defined(SHADER_TYPE_VERTEX) && defined(LIGHTING_VERTEX))
struct AmbientLight {
  vec3 color;
};

struct PointLight {
  vec3 color;
  vec3 position;
  vec3 attenuation; // 2nd order x:Constant-y:Linear-z:Exponential
};

struct SpotLight {
  vec3 color;
  vec3 position;
  vec3 direction;
  vec3 attenuation;
  vec2 coneCos;
};

struct DirectionalLight {
  vec3 color;
  vec3 direction;
};

struct UniformLight {
  vec3 color;
  vec3 position;
  vec3 direction;
  vec3 attenuation;
  vec2 coneCos;
};

layout(std140) uniform lightingUniforms {
  int enabled;
  int directionalLightCount;
  int pointLightCount;
  int spotLightCount;
  vec3 ambientColor;
  UniformLight lights[5];
} lighting;

PointLight lighting_getPointLight(int index) {
  UniformLight light = lighting.lights[index];
  return PointLight(light.color, light.position, light.attenuation);
}

SpotLight lighting_getSpotLight(int index) {
  UniformLight light = lighting.lights[lighting.pointLightCount + index];
  return SpotLight(light.color, light.position, light.direction, light.attenuation, light.coneCos);
}

DirectionalLight lighting_getDirectionalLight(int index) {
  UniformLight light =
    lighting.lights[lighting.pointLightCount + lighting.spotLightCount + index];
  return DirectionalLight(light.color, light.direction);
}

float getPointLightAttenuation(PointLight pointLight, float distance) {
  return pointLight.attenuation.x
       + pointLight.attenuation.y * distance
       + pointLight.attenuation.z * distance * distance;
}

float getSpotLightAttenuation(SpotLight spotLight, vec3 positionWorldspace) {
  vec3 light_direction = normalize(positionWorldspace - spotLight.position);
  float coneFactor = smoothstep(
    spotLight.coneCos.y,
    spotLight.coneCos.x,
    dot(normalize(spotLight.direction), light_direction)
  );
  float distanceAttenuation = getPointLightAttenuation(
    PointLight(spotLight.color, spotLight.position, spotLight.attenuation),
    distance(spotLight.position, positionWorldspace)
  );
  return distanceAttenuation / max(coneFactor, 0.0001);
}

// #endif
`,bD=`\
// #if (defined(SHADER_TYPE_FRAGMENT) && defined(LIGHTING_FRAGMENT)) || (defined(SHADER_TYPE_VERTEX) && defined(LIGHTING_VERTEX))
const MAX_LIGHTS: i32 = 5;

struct AmbientLight {
  color: vec3<f32>,
};

struct PointLight {
  color: vec3<f32>,
  position: vec3<f32>,
  attenuation: vec3<f32>, // 2nd order x:Constant-y:Linear-z:Exponential
};

struct SpotLight {
  color: vec3<f32>,
  position: vec3<f32>,
  direction: vec3<f32>,
  attenuation: vec3<f32>,
  coneCos: vec2<f32>,
};

struct DirectionalLight {
  color: vec3<f32>,
  direction: vec3<f32>,
};

struct UniformLight {
  color: vec3<f32>,
  position: vec3<f32>,
  direction: vec3<f32>,
  attenuation: vec3<f32>,
  coneCos: vec2<f32>,
};

struct lightingUniforms {
  enabled: i32,
  directionalLightCount: i32,
  pointLightCount: i32,
  spotLightCount: i32,
  ambientColor: vec3<f32>,
  lights: array<UniformLight, 5>,
};

@group(2) @binding(auto) var<uniform> lighting : lightingUniforms;

fn lighting_getPointLight(index: i32) -> PointLight {
  let light = lighting.lights[index];
  return PointLight(light.color, light.position, light.attenuation);
}

fn lighting_getSpotLight(index: i32) -> SpotLight {
  let light = lighting.lights[lighting.pointLightCount + index];
  return SpotLight(light.color, light.position, light.direction, light.attenuation, light.coneCos);
}

fn lighting_getDirectionalLight(index: i32) -> DirectionalLight {
  let light = lighting.lights[lighting.pointLightCount + lighting.spotLightCount + index];
  return DirectionalLight(light.color, light.direction);
}

fn getPointLightAttenuation(pointLight: PointLight, distance: f32) -> f32 {
  return pointLight.attenuation.x
       + pointLight.attenuation.y * distance
       + pointLight.attenuation.z * distance * distance;
}

fn getSpotLightAttenuation(spotLight: SpotLight, positionWorldspace: vec3<f32>) -> f32 {
  let lightDirection = normalize(positionWorldspace - spotLight.position);
  let coneFactor = smoothstep(
    spotLight.coneCos.y,
    spotLight.coneCos.x,
    dot(normalize(spotLight.direction), lightDirection)
  );
  let distanceAttenuation = getPointLightAttenuation(
    PointLight(spotLight.color, spotLight.position, spotLight.attenuation),
    distance(spotLight.position, positionWorldspace)
  );
  return distanceAttenuation / max(coneFactor, 0.0001);
}
`;function bE(a,b=!0){return a??b}function bF(a=[0,0,0],b=!0){return b?a.map(a=>a/255):[...a]}let bG={props:{},uniforms:{},name:"lighting",defines:{},uniformTypes:{enabled:"i32",directionalLightCount:"i32",pointLightCount:"i32",spotLightCount:"i32",ambientColor:"vec3<f32>",lights:[{color:"vec3<f32>",position:"vec3<f32>",direction:"vec3<f32>",attenuation:"vec3<f32>",coneCos:"vec2<f32>"},5]},defaultUniforms:bI(),bindingLayout:[{name:"lighting",group:2}],firstBindingSlot:0,source:bD,vs:bC,fs:bC,getUniforms:function(a,b={}){if(!(a=a?{...a}:a))return bI();a.lights&&(a={...a,...function(a){let b={pointLights:[],spotLights:[],directionalLights:[]};for(let c of a||[])switch(c.type){case"ambient":b.ambientLight=c;break;case"directional":b.directionalLights?.push(c);break;case"point":b.pointLights?.push(c);break;case"spot":b.spotLights?.push(c)}return b}(a.lights),lights:void 0});let{useByteColors:c,ambientLight:d,pointLights:e,spotLights:f,directionalLights:g}=a||{};if(!(d||e&&e.length>0||f&&f.length>0||g&&g.length>0))return{...bI(),enabled:0};let h={...bI(),...function({useByteColors:a,ambientLight:b,pointLights:c=[],spotLights:d=[],directionalLights:e=[]}){let f=bJ(),g=0,h=0,i=0,j=0;for(let b of c){if(g>=5)break;f[g]={...f[g],color:bH(b,a),position:b.position,attenuation:b.attenuation||[1,0,0]},g++,h++}for(let b of d){var k;if(g>=5)break;f[g]={...f[g],color:bH(b,a),position:b.position,direction:b.direction,attenuation:b.attenuation||[1,0,0],coneCos:[Math.cos((k=b).innerConeAngle??0),Math.cos(k.outerConeAngle??Math.PI/4)]},g++,i++}for(let b of e){if(g>=5)break;f[g]={...f[g],color:bH(b,a),direction:b.direction},g++,j++}return c.length+d.length+e.length>5&&bB.log.warn("MAX_LIGHTS exceeded, truncating to 5")(),{ambientColor:bH(b,a),directionalLightCount:j,pointLightCount:h,spotLightCount:i,lights:f}}({useByteColors:c,ambientLight:d,pointLights:e,spotLights:f,directionalLights:g})};return void 0!==a.enabled&&(h.enabled=+!!a.enabled),h}};function bH(a={},b){let{color:c=[0,0,0],intensity:d=1}=a;return bF(c,bE(b,!0)).map(a=>a*d)}function bI(){return{enabled:1,directionalLightCount:0,pointLightCount:0,spotLightCount:0,ambientColor:[.1,.1,.1],lights:bJ()}}function bJ(){return Array.from({length:5},()=>({color:[1,1,1],position:[1,1,2],direction:[1,1,1],attenuation:[1,0,0],coneCos:[1,0]}))}let bK=`\
layout(std140) uniform phongMaterialUniforms {
  uniform bool unlit;
  uniform float ambient;
  uniform float diffuse;
  uniform float shininess;
  uniform vec3  specularColor;
} material;
`,bL=`\
layout(std140) uniform phongMaterialUniforms {
  uniform bool unlit;
  uniform float ambient;
  uniform float diffuse;
  uniform float shininess;
  uniform vec3  specularColor;
} material;

vec3 lighting_getLightColor(vec3 surfaceColor, vec3 light_direction, vec3 view_direction, vec3 normal_worldspace, vec3 color) {
  vec3 halfway_direction = normalize(light_direction + view_direction);
  float lambertian = dot(light_direction, normal_worldspace);
  float specular = 0.0;
  if (lambertian > 0.0) {
    float specular_angle = max(dot(normal_worldspace, halfway_direction), 0.0);
    specular = pow(specular_angle, material.shininess);
  }
  lambertian = max(lambertian, 0.0);
  return (lambertian * material.diffuse * surfaceColor + specular * floatColors_normalize(material.specularColor)) * color;
}

vec3 lighting_getLightColor(vec3 surfaceColor, vec3 cameraPosition, vec3 position_worldspace, vec3 normal_worldspace) {
  vec3 lightColor = surfaceColor;

  if (material.unlit) {
    return surfaceColor;
  }

  if (lighting.enabled == 0) {
    return lightColor;
  }

  vec3 view_direction = normalize(cameraPosition - position_worldspace);
  lightColor = material.ambient * surfaceColor * lighting.ambientColor;

  for (int i = 0; i < lighting.pointLightCount; i++) {
    PointLight pointLight = lighting_getPointLight(i);
    vec3 light_position_worldspace = pointLight.position;
    vec3 light_direction = normalize(light_position_worldspace - position_worldspace);
    float light_attenuation = getPointLightAttenuation(pointLight, distance(light_position_worldspace, position_worldspace));
    lightColor += lighting_getLightColor(surfaceColor, light_direction, view_direction, normal_worldspace, pointLight.color / light_attenuation);
  }

  for (int i = 0; i < lighting.spotLightCount; i++) {
    SpotLight spotLight = lighting_getSpotLight(i);
    vec3 light_position_worldspace = spotLight.position;
    vec3 light_direction = normalize(light_position_worldspace - position_worldspace);
    float light_attenuation = getSpotLightAttenuation(spotLight, position_worldspace);
    lightColor += lighting_getLightColor(surfaceColor, light_direction, view_direction, normal_worldspace, spotLight.color / light_attenuation);
  }

  for (int i = 0; i < lighting.directionalLightCount; i++) {
    DirectionalLight directionalLight = lighting_getDirectionalLight(i);
    lightColor += lighting_getLightColor(surfaceColor, -directionalLight.direction, view_direction, normal_worldspace, directionalLight.color);
  }
  
  return lightColor;
}
`,bM=`\
struct phongMaterialUniforms {
  unlit: u32,
  ambient: f32,
  diffuse: f32,
  shininess: f32,
  specularColor: vec3<f32>,
};

@group(3) @binding(auto) var<uniform> phongMaterial : phongMaterialUniforms;

fn lighting_getLightColor(surfaceColor: vec3<f32>, light_direction: vec3<f32>, view_direction: vec3<f32>, normal_worldspace: vec3<f32>, color: vec3<f32>) -> vec3<f32> {
  let halfway_direction: vec3<f32> = normalize(light_direction + view_direction);
  var lambertian: f32 = dot(light_direction, normal_worldspace);
  var specular: f32 = 0.0;
  if (lambertian > 0.0) {
    let specular_angle = max(dot(normal_worldspace, halfway_direction), 0.0);
    specular = pow(specular_angle, phongMaterial.shininess);
  }
  lambertian = max(lambertian, 0.0);
  return (
    lambertian * phongMaterial.diffuse * surfaceColor +
    specular * floatColors_normalize(phongMaterial.specularColor)
  ) * color;
}

fn lighting_getLightColor2(surfaceColor: vec3<f32>, cameraPosition: vec3<f32>, position_worldspace: vec3<f32>, normal_worldspace: vec3<f32>) -> vec3<f32> {
  var lightColor: vec3<f32> = surfaceColor;

  if (phongMaterial.unlit != 0u) {
    return surfaceColor;
  }

  if (lighting.enabled == 0) {
    return lightColor;
  }

  let view_direction: vec3<f32> = normalize(cameraPosition - position_worldspace);
  lightColor = phongMaterial.ambient * surfaceColor * lighting.ambientColor;

  for (var i: i32 = 0; i < lighting.pointLightCount; i++) {
    let pointLight: PointLight = lighting_getPointLight(i);
    let light_position_worldspace: vec3<f32> = pointLight.position;
    let light_direction: vec3<f32> = normalize(light_position_worldspace - position_worldspace);
    let light_attenuation = getPointLightAttenuation(
      pointLight,
      distance(light_position_worldspace, position_worldspace)
    );
    lightColor += lighting_getLightColor(
      surfaceColor,
      light_direction,
      view_direction,
      normal_worldspace,
      pointLight.color / light_attenuation
    );
  }

  for (var i: i32 = 0; i < lighting.spotLightCount; i++) {
    let spotLight: SpotLight = lighting_getSpotLight(i);
    let light_position_worldspace: vec3<f32> = spotLight.position;
    let light_direction: vec3<f32> = normalize(light_position_worldspace - position_worldspace);
    let light_attenuation = getSpotLightAttenuation(spotLight, position_worldspace);
    lightColor += lighting_getLightColor(
      surfaceColor,
      light_direction,
      view_direction,
      normal_worldspace,
      spotLight.color / light_attenuation
    );
  }

  for (var i: i32 = 0; i < lighting.directionalLightCount; i++) {
    let directionalLight: DirectionalLight = lighting_getDirectionalLight(i);
    lightColor += lighting_getLightColor(surfaceColor, -directionalLight.direction, view_direction, normal_worldspace, directionalLight.color);
  }  
  
  return lightColor;
}

fn lighting_getSpecularLightColor(cameraPosition: vec3<f32>, position_worldspace: vec3<f32>, normal_worldspace: vec3<f32>) -> vec3<f32>{
  var lightColor = vec3<f32>(0, 0, 0);
  let surfaceColor = vec3<f32>(0, 0, 0);

  if (lighting.enabled != 0) {
    let view_direction = normalize(cameraPosition - position_worldspace);

    for (var i: i32 = 0; i < lighting.pointLightCount; i++) {
      let pointLight: PointLight = lighting_getPointLight(i);
      let light_position_worldspace: vec3<f32> = pointLight.position;
      let light_direction: vec3<f32> = normalize(light_position_worldspace - position_worldspace);
      let light_attenuation = getPointLightAttenuation(
        pointLight,
        distance(light_position_worldspace, position_worldspace)
      );
      lightColor += lighting_getLightColor(
        surfaceColor,
        light_direction,
        view_direction,
        normal_worldspace,
        pointLight.color / light_attenuation
      );
    }

    for (var i: i32 = 0; i < lighting.spotLightCount; i++) {
      let spotLight: SpotLight = lighting_getSpotLight(i);
      let light_position_worldspace: vec3<f32> = spotLight.position;
      let light_direction: vec3<f32> = normalize(light_position_worldspace - position_worldspace);
      let light_attenuation = getSpotLightAttenuation(spotLight, position_worldspace);
      lightColor += lighting_getLightColor(
        surfaceColor,
        light_direction,
        view_direction,
        normal_worldspace,
        spotLight.color / light_attenuation
      );
    }

    for (var i: i32 = 0; i < lighting.directionalLightCount; i++) {
        let directionalLight: DirectionalLight = lighting_getDirectionalLight(i);
        lightColor += lighting_getLightColor(surfaceColor, -directionalLight.direction, view_direction, normal_worldspace, directionalLight.color);
    }
  }
  return lightColor;
}
`,bN={props:{},name:"gouraudMaterial",bindingLayout:[{name:"gouraudMaterial",group:3}],vs:bL.replace("phongMaterial","gouraudMaterial"),fs:bK.replace("phongMaterial","gouraudMaterial"),source:bM.replaceAll("phongMaterial","gouraudMaterial"),defines:{LIGHTING_VERTEX:!0},dependencies:[bG,bA],uniformTypes:{unlit:"i32",ambient:"f32",diffuse:"f32",shininess:"f32",specularColor:"vec3<f32>"},defaultUniforms:{unlit:!1,ambient:.35,diffuse:.6,shininess:32,specularColor:[38.25,38.25,38.25]},getUniforms:a=>({...bN.defaultUniforms,...a})},bO={name:"phongMaterial",firstBindingSlot:0,bindingLayout:[{name:"phongMaterial",group:3}],dependencies:[bG,bA],source:bM,vs:bK,fs:bL,defines:{LIGHTING_FRAGMENT:!0},uniformTypes:{unlit:"i32",ambient:"f32",diffuse:"f32",shininess:"f32",specularColor:"vec3<f32>"},defaultUniforms:{unlit:!1,ambient:.35,diffuse:.6,shininess:32,specularColor:[38.25,38.25,38.25]},getUniforms:a=>({...bO.defaultUniforms,...a})},bP=`\
layout(std140) uniform layerUniforms {
  uniform float opacity;
} layer;
`,bQ={name:"layer",source:`\
struct LayerUniforms {
  opacity: f32,
};

@group(0) @binding(auto)
var<uniform> layer: LayerUniforms;
`,vs:bP,fs:bP,getUniforms:a=>({opacity:Math.pow(a.opacity,1/2.2)}),uniformTypes:{opacity:"f32"}},bR={name:"color",dependencies:[],source:`

@must_use
fn deckgl_premultiplied_alpha(fragColor: vec4<f32>) -> vec4<f32> {
    return vec4(fragColor.rgb * fragColor.a, fragColor.a); 
};
`,getUniforms:a=>({})},bS="#define SMOOTH_EDGE_RADIUS 0.5",bT={name:"geometry",source:`\
const SMOOTH_EDGE_RADIUS: f32 = 0.5;

struct VertexGeometry {
  position: vec4<f32>,
  worldPosition: vec3<f32>,
  worldPositionAlt: vec3<f32>,
  normal: vec3<f32>,
  uv: vec2<f32>,
  pickingColor: vec3<f32>,
};

var<private> geometry_: VertexGeometry = VertexGeometry(
  vec4<f32>(0.0, 0.0, 1.0, 0.0),
  vec3<f32>(0.0, 0.0, 0.0),
  vec3<f32>(0.0, 0.0, 0.0),
  vec3<f32>(0.0, 0.0, 0.0),
  vec2<f32>(0.0, 0.0),
  vec3<f32>(0.0, 0.0, 0.0)
);

struct FragmentGeometry {
  uv: vec2<f32>,
};

var<private> fragmentGeometry: FragmentGeometry;

fn smoothedge(edge: f32, x: f32) -> f32 {
  return smoothstep(edge - SMOOTH_EDGE_RADIUS, edge + SMOOTH_EDGE_RADIUS, x);
}
`,vs:`\
${bS}

struct VertexGeometry {
  vec4 position;
  vec3 worldPosition;
  vec3 worldPositionAlt;
  vec3 normal;
  vec2 uv;
  vec3 pickingColor;
} geometry = VertexGeometry(
  vec4(0.0, 0.0, 1.0, 0.0),
  vec3(0.0),
  vec3(0.0),
  vec3(0.0),
  vec2(0.0),
  vec3(0.0)
);
`,fs:`\
${bS}

struct FragmentGeometry {
  vec2 uv;
} geometry;

float smoothedge(float edge, float x) {
  return smoothstep(edge - SMOOTH_EDGE_RADIUS, edge + SMOOTH_EDGE_RADIUS, x);
}
`},bU=`\
#ifdef LUMA_FP32_TAN_PRECISION_WORKAROUND

// All these functions are for substituting tan() function from Intel GPU only
const float TWO_PI = 6.2831854820251465;
const float PI_2 = 1.5707963705062866;
const float PI_16 = 0.1963495463132858;

const float SIN_TABLE_0 = 0.19509032368659973;
const float SIN_TABLE_1 = 0.3826834261417389;
const float SIN_TABLE_2 = 0.5555702447891235;
const float SIN_TABLE_3 = 0.7071067690849304;

const float COS_TABLE_0 = 0.9807852506637573;
const float COS_TABLE_1 = 0.9238795042037964;
const float COS_TABLE_2 = 0.8314695954322815;
const float COS_TABLE_3 = 0.7071067690849304;

const float INVERSE_FACTORIAL_3 = 1.666666716337204e-01; // 1/3!
const float INVERSE_FACTORIAL_5 = 8.333333767950535e-03; // 1/5!
const float INVERSE_FACTORIAL_7 = 1.9841270113829523e-04; // 1/7!
const float INVERSE_FACTORIAL_9 = 2.75573188446287533e-06; // 1/9!

float sin_taylor_fp32(float a) {
  float r, s, t, x;

  if (a == 0.0) {
    return 0.0;
  }

  x = -a * a;
  s = a;
  r = a;

  r = r * x;
  t = r * INVERSE_FACTORIAL_3;
  s = s + t;

  r = r * x;
  t = r * INVERSE_FACTORIAL_5;
  s = s + t;

  r = r * x;
  t = r * INVERSE_FACTORIAL_7;
  s = s + t;

  r = r * x;
  t = r * INVERSE_FACTORIAL_9;
  s = s + t;

  return s;
}

void sincos_taylor_fp32(float a, out float sin_t, out float cos_t) {
  if (a == 0.0) {
    sin_t = 0.0;
    cos_t = 1.0;
  }
  sin_t = sin_taylor_fp32(a);
  cos_t = sqrt(1.0 - sin_t * sin_t);
}

float tan_taylor_fp32(float a) {
    float sin_a;
    float cos_a;

    if (a == 0.0) {
        return 0.0;
    }

    // 2pi range reduction
    float z = floor(a / TWO_PI);
    float r = a - TWO_PI * z;

    float t;
    float q = floor(r / PI_2 + 0.5);
    int j = int(q);

    if (j < -2 || j > 2) {
        return 1.0 / 0.0;
    }

    t = r - PI_2 * q;

    q = floor(t / PI_16 + 0.5);
    int k = int(q);
    int abs_k = int(abs(float(k)));

    if (abs_k > 4) {
        return 1.0 / 0.0;
    } else {
        t = t - PI_16 * q;
    }

    float u = 0.0;
    float v = 0.0;

    float sin_t, cos_t;
    float s, c;
    sincos_taylor_fp32(t, sin_t, cos_t);

    if (k == 0) {
        s = sin_t;
        c = cos_t;
    } else {
        if (abs(float(abs_k) - 1.0) < 0.5) {
            u = COS_TABLE_0;
            v = SIN_TABLE_0;
        } else if (abs(float(abs_k) - 2.0) < 0.5) {
            u = COS_TABLE_1;
            v = SIN_TABLE_1;
        } else if (abs(float(abs_k) - 3.0) < 0.5) {
            u = COS_TABLE_2;
            v = SIN_TABLE_2;
        } else if (abs(float(abs_k) - 4.0) < 0.5) {
            u = COS_TABLE_3;
            v = SIN_TABLE_3;
        }
        if (k > 0) {
            s = u * sin_t + v * cos_t;
            c = u * cos_t - v * sin_t;
        } else {
            s = u * sin_t - v * cos_t;
            c = u * cos_t + v * sin_t;
        }
    }

    if (j == 0) {
        sin_a = s;
        cos_a = c;
    } else if (j == 1) {
        sin_a = c;
        cos_a = -s;
    } else if (j == -1) {
        sin_a = -c;
        cos_a = s;
    } else {
        sin_a = -s;
        cos_a = -c;
    }
    return sin_a / cos_a;
}
#endif

float tan_fp32(float a) {
#ifdef LUMA_FP32_TAN_PRECISION_WORKAROUND
  return tan_taylor_fp32(a);
#else
  return tan(a);
#endif
}
`,bV="u">typeof Float32Array?Float32Array:Array,bW=Math.random;function bX(a){return a>=0?Math.round(a):a%.5==0?Math.floor(a):Math.round(a)}function bY(a){return a[0]=1,a[1]=0,a[2]=0,a[3]=0,a[4]=0,a[5]=1,a[6]=0,a[7]=0,a[8]=0,a[9]=0,a[10]=1,a[11]=0,a[12]=0,a[13]=0,a[14]=0,a[15]=1,a}function bZ(a,b){if(a===b){let c=b[1],d=b[2],e=b[3],f=b[6],g=b[7],h=b[11];a[1]=b[4],a[2]=b[8],a[3]=b[12],a[4]=c,a[6]=b[9],a[7]=b[13],a[8]=d,a[9]=f,a[11]=b[14],a[12]=e,a[13]=g,a[14]=h}else a[0]=b[0],a[1]=b[4],a[2]=b[8],a[3]=b[12],a[4]=b[1],a[5]=b[5],a[6]=b[9],a[7]=b[13],a[8]=b[2],a[9]=b[6],a[10]=b[10],a[11]=b[14],a[12]=b[3],a[13]=b[7],a[14]=b[11],a[15]=b[15];return a}function b$(a,b){let c=b[0],d=b[1],e=b[2],f=b[3],g=b[4],h=b[5],i=b[6],j=b[7],k=b[8],l=b[9],m=b[10],n=b[11],o=b[12],p=b[13],q=b[14],r=b[15],s=c*h-d*g,t=c*i-e*g,u=c*j-f*g,v=d*i-e*h,w=d*j-f*h,x=e*j-f*i,y=k*p-l*o,z=k*q-m*o,A=k*r-n*o,B=l*q-m*p,C=l*r-n*p,D=m*r-n*q,E=s*D-t*C+u*B+v*A-w*z+x*y;return E?(E=1/E,a[0]=(h*D-i*C+j*B)*E,a[1]=(e*C-d*D-f*B)*E,a[2]=(p*x-q*w+r*v)*E,a[3]=(m*w-l*x-n*v)*E,a[4]=(i*A-g*D-j*z)*E,a[5]=(c*D-e*A+f*z)*E,a[6]=(q*u-o*x-r*t)*E,a[7]=(k*x-m*u+n*t)*E,a[8]=(g*C-h*A+j*y)*E,a[9]=(d*A-c*C-f*y)*E,a[10]=(o*w-p*u+r*s)*E,a[11]=(l*u-k*w-n*s)*E,a[12]=(h*z-g*B-i*y)*E,a[13]=(c*B-d*z+e*y)*E,a[14]=(p*t-o*v-q*s)*E,a[15]=(k*v-l*t+m*s)*E,a):null}function b_(a){let b=a[0],c=a[1],d=a[2],e=a[3],f=a[4],g=a[5],h=a[6],i=a[7],j=a[8],k=a[9],l=a[10],m=a[11],n=a[12],o=a[13],p=a[14],q=a[15],r=b*g-c*f,s=b*h-d*f,t=c*h-d*g,u=j*o-k*n,v=j*p-l*n,w=k*p-l*o;return i*(b*w-c*v+d*u)-e*(f*w-g*v+h*u)+q*(j*t-k*s+l*r)-m*(n*t-o*s+p*r)}function b0(a,b,c){let d=b[0],e=b[1],f=b[2],g=b[3],h=b[4],i=b[5],j=b[6],k=b[7],l=b[8],m=b[9],n=b[10],o=b[11],p=b[12],q=b[13],r=b[14],s=b[15],t=c[0],u=c[1],v=c[2],w=c[3];return a[0]=t*d+u*h+v*l+w*p,a[1]=t*e+u*i+v*m+w*q,a[2]=t*f+u*j+v*n+w*r,a[3]=t*g+u*k+v*o+w*s,t=c[4],u=c[5],v=c[6],w=c[7],a[4]=t*d+u*h+v*l+w*p,a[5]=t*e+u*i+v*m+w*q,a[6]=t*f+u*j+v*n+w*r,a[7]=t*g+u*k+v*o+w*s,t=c[8],u=c[9],v=c[10],w=c[11],a[8]=t*d+u*h+v*l+w*p,a[9]=t*e+u*i+v*m+w*q,a[10]=t*f+u*j+v*n+w*r,a[11]=t*g+u*k+v*o+w*s,t=c[12],u=c[13],v=c[14],w=c[15],a[12]=t*d+u*h+v*l+w*p,a[13]=t*e+u*i+v*m+w*q,a[14]=t*f+u*j+v*n+w*r,a[15]=t*g+u*k+v*o+w*s,a}function b1(a,b,c){let d,e,f,g,h,i,j,k,l,m,n,o,p=c[0],q=c[1],r=c[2];return b===a?(a[12]=b[0]*p+b[4]*q+b[8]*r+b[12],a[13]=b[1]*p+b[5]*q+b[9]*r+b[13],a[14]=b[2]*p+b[6]*q+b[10]*r+b[14],a[15]=b[3]*p+b[7]*q+b[11]*r+b[15]):(d=b[0],e=b[1],f=b[2],g=b[3],h=b[4],i=b[5],j=b[6],k=b[7],l=b[8],m=b[9],n=b[10],o=b[11],a[0]=d,a[1]=e,a[2]=f,a[3]=g,a[4]=h,a[5]=i,a[6]=j,a[7]=k,a[8]=l,a[9]=m,a[10]=n,a[11]=o,a[12]=d*p+h*q+l*r+b[12],a[13]=e*p+i*q+m*r+b[13],a[14]=f*p+j*q+n*r+b[14],a[15]=g*p+k*q+o*r+b[15]),a}function b2(a,b,c){let d=c[0],e=c[1],f=c[2];return a[0]=b[0]*d,a[1]=b[1]*d,a[2]=b[2]*d,a[3]=b[3]*d,a[4]=b[4]*e,a[5]=b[5]*e,a[6]=b[6]*e,a[7]=b[7]*e,a[8]=b[8]*f,a[9]=b[9]*f,a[10]=b[10]*f,a[11]=b[11]*f,a[12]=b[12],a[13]=b[13],a[14]=b[14],a[15]=b[15],a}function b3(a,b,c,d){let e,f,g,h,i,j,k,l,m,n,o,p,q,r,s,t,u,v,w,x,y,z,A,B,C=d[0],D=d[1],E=d[2],F=Math.sqrt(C*C+D*D+E*E);return F<1e-6?null:(C*=F=1/F,D*=F,E*=F,f=Math.sin(c),g=1-(e=Math.cos(c)),h=b[0],i=b[1],j=b[2],k=b[3],l=b[4],m=b[5],n=b[6],o=b[7],p=b[8],q=b[9],r=b[10],s=b[11],t=C*C*g+e,u=D*C*g+E*f,v=E*C*g-D*f,w=C*D*g-E*f,x=D*D*g+e,y=E*D*g+C*f,z=C*E*g+D*f,A=D*E*g-C*f,B=E*E*g+e,a[0]=h*t+l*u+p*v,a[1]=i*t+m*u+q*v,a[2]=j*t+n*u+r*v,a[3]=k*t+o*u+s*v,a[4]=h*w+l*x+p*y,a[5]=i*w+m*x+q*y,a[6]=j*w+n*x+r*y,a[7]=k*w+o*x+s*y,a[8]=h*z+l*A+p*B,a[9]=i*z+m*A+q*B,a[10]=j*z+n*A+r*B,a[11]=k*z+o*A+s*B,b!==a&&(a[12]=b[12],a[13]=b[13],a[14]=b[14],a[15]=b[15]),a)}function b4(a,b,c){let d=Math.sin(c),e=Math.cos(c),f=b[4],g=b[5],h=b[6],i=b[7],j=b[8],k=b[9],l=b[10],m=b[11];return b!==a&&(a[0]=b[0],a[1]=b[1],a[2]=b[2],a[3]=b[3],a[12]=b[12],a[13]=b[13],a[14]=b[14],a[15]=b[15]),a[4]=f*e+j*d,a[5]=g*e+k*d,a[6]=h*e+l*d,a[7]=i*e+m*d,a[8]=j*e-f*d,a[9]=k*e-g*d,a[10]=l*e-h*d,a[11]=m*e-i*d,a}function b5(a,b,c){let d=Math.sin(c),e=Math.cos(c),f=b[0],g=b[1],h=b[2],i=b[3],j=b[8],k=b[9],l=b[10],m=b[11];return b!==a&&(a[4]=b[4],a[5]=b[5],a[6]=b[6],a[7]=b[7],a[12]=b[12],a[13]=b[13],a[14]=b[14],a[15]=b[15]),a[0]=f*e-j*d,a[1]=g*e-k*d,a[2]=h*e-l*d,a[3]=i*e-m*d,a[8]=f*d+j*e,a[9]=g*d+k*e,a[10]=h*d+l*e,a[11]=i*d+m*e,a}function b6(a,b,c){let d=Math.sin(c),e=Math.cos(c),f=b[0],g=b[1],h=b[2],i=b[3],j=b[4],k=b[5],l=b[6],m=b[7];return b!==a&&(a[8]=b[8],a[9]=b[9],a[10]=b[10],a[11]=b[11],a[12]=b[12],a[13]=b[13],a[14]=b[14],a[15]=b[15]),a[0]=f*e+j*d,a[1]=g*e+k*d,a[2]=h*e+l*d,a[3]=i*e+m*d,a[4]=j*e-f*d,a[5]=k*e-g*d,a[6]=l*e-h*d,a[7]=m*e-i*d,a}function b7(a,b,c){let d=b[0],e=b[1],f=b[2],g=b[3],h=d+d,i=e+e,j=f+f,k=d*h,l=d*i,m=d*j,n=e*i,o=e*j,p=f*j,q=g*h,r=g*i,s=g*j;return a[0]=1-(n+p),a[1]=l+s,a[2]=m-r,a[3]=0,a[4]=l-s,a[5]=1-(k+p),a[6]=o+q,a[7]=0,a[8]=m+r,a[9]=o-q,a[10]=1-(k+n),a[11]=0,a[12]=c[0],a[13]=c[1],a[14]=c[2],a[15]=1,a}function b8(a,b){let c=b[0],d=b[1],e=b[2],f=b[4],g=b[5],h=b[6],i=b[8],j=b[9],k=b[10];return a[0]=Math.sqrt(c*c+d*d+e*e),a[1]=Math.sqrt(f*f+g*g+h*h),a[2]=Math.sqrt(i*i+j*j+k*k),a}function b9(a,b){let c=b[0],d=b[1],e=b[2],f=b[3],g=c+c,h=d+d,i=e+e,j=c*g,k=d*g,l=d*h,m=e*g,n=e*h,o=e*i,p=f*g,q=f*h,r=f*i;return a[0]=1-l-o,a[1]=k+r,a[2]=m-q,a[3]=0,a[4]=k-r,a[5]=1-j-o,a[6]=n+p,a[7]=0,a[8]=m+q,a[9]=n-p,a[10]=1-j-l,a[11]=0,a[12]=0,a[13]=0,a[14]=0,a[15]=1,a}function ca(a,b,c,d,e,f,g){let h=1/(c-b),i=1/(e-d),j=1/(f-g);return a[0]=2*f*h,a[1]=0,a[2]=0,a[3]=0,a[4]=0,a[5]=2*f*i,a[6]=0,a[7]=0,a[8]=(c+b)*h,a[9]=(e+d)*i,a[10]=(g+f)*j,a[11]=-1,a[12]=0,a[13]=0,a[14]=g*f*2*j,a[15]=0,a}function cb(a,b,c,d,e){let f=1/Math.tan(b/2);if(a[0]=f/c,a[1]=0,a[2]=0,a[3]=0,a[4]=0,a[5]=f,a[6]=0,a[7]=0,a[8]=0,a[9]=0,a[11]=-1,a[12]=0,a[13]=0,a[15]=0,null!=e&&e!==1/0){let b=1/(d-e);a[10]=(e+d)*b,a[14]=2*e*d*b}else a[10]=-1,a[14]=-2*d;return a}function cc(a,b,c,d,e,f,g){let h=1/(b-c),i=1/(d-e),j=1/(f-g);return a[0]=-2*h,a[1]=0,a[2]=0,a[3]=0,a[4]=0,a[5]=-2*i,a[6]=0,a[7]=0,a[8]=0,a[9]=0,a[10]=2*j,a[11]=0,a[12]=(b+c)*h,a[13]=(e+d)*i,a[14]=(g+f)*j,a[15]=1,a}function cd(a,b,c,d){let e,f,g,h,i,j,k,l,m,n,o=b[0],p=b[1],q=b[2],r=d[0],s=d[1],t=d[2],u=c[0],v=c[1],w=c[2];return 1e-6>Math.abs(o-u)&&1e-6>Math.abs(p-v)&&1e-6>Math.abs(q-w)?bY(a):(e=1/Math.sqrt((l=o-u)*l+(m=p-v)*m+(n=q-w)*n),l*=e,m*=e,n*=e,(e=Math.sqrt((f=s*n-t*m)*f+(g=t*l-r*n)*g+(h=r*m-s*l)*h))?(f*=e=1/e,g*=e,h*=e):(f=0,g=0,h=0),(e=Math.sqrt((i=m*h-n*g)*i+(j=n*f-l*h)*j+(k=l*g-m*f)*k))?(i*=e=1/e,j*=e,k*=e):(i=0,j=0,k=0),a[0]=f,a[1]=i,a[2]=l,a[3]=0,a[4]=g,a[5]=j,a[6]=m,a[7]=0,a[8]=h,a[9]=k,a[10]=n,a[11]=0,a[12]=-(f*o+g*p+h*q),a[13]=-(i*o+j*p+k*q),a[14]=-(l*o+m*p+n*q),a[15]=1,a)}function ce(a,b,c){return a[0]=b[0]-c[0],a[1]=b[1]-c[1],a[2]=b[2]-c[2],a[3]=b[3]-c[3],a[4]=b[4]-c[4],a[5]=b[5]-c[5],a[6]=b[6]-c[6],a[7]=b[7]-c[7],a[8]=b[8]-c[8],a[9]=b[9]-c[9],a[10]=b[10]-c[10],a[11]=b[11]-c[11],a[12]=b[12]-c[12],a[13]=b[13]-c[13],a[14]=b[14]-c[14],a[15]=b[15]-c[15],a}a.s(["add",0,function(a,b,c){return a[0]=b[0]+c[0],a[1]=b[1]+c[1],a[2]=b[2]+c[2],a[3]=b[3]+c[3],a[4]=b[4]+c[4],a[5]=b[5]+c[5],a[6]=b[6]+c[6],a[7]=b[7]+c[7],a[8]=b[8]+c[8],a[9]=b[9]+c[9],a[10]=b[10]+c[10],a[11]=b[11]+c[11],a[12]=b[12]+c[12],a[13]=b[13]+c[13],a[14]=b[14]+c[14],a[15]=b[15]+c[15],a},"adjoint",0,function(a,b){let c=b[0],d=b[1],e=b[2],f=b[3],g=b[4],h=b[5],i=b[6],j=b[7],k=b[8],l=b[9],m=b[10],n=b[11],o=b[12],p=b[13],q=b[14],r=b[15],s=c*h-d*g,t=c*i-e*g,u=c*j-f*g,v=d*i-e*h,w=d*j-f*h,x=e*j-f*i,y=k*p-l*o,z=k*q-m*o,A=k*r-n*o,B=l*q-m*p,C=l*r-n*p,D=m*r-n*q;return a[0]=h*D-i*C+j*B,a[1]=e*C-d*D-f*B,a[2]=p*x-q*w+r*v,a[3]=m*w-l*x-n*v,a[4]=i*A-g*D-j*z,a[5]=c*D-e*A+f*z,a[6]=q*u-o*x-r*t,a[7]=k*x-m*u+n*t,a[8]=g*C-h*A+j*y,a[9]=d*A-c*C-f*y,a[10]=o*w-p*u+r*s,a[11]=l*u-k*w-n*s,a[12]=h*z-g*B-i*y,a[13]=c*B-d*z+e*y,a[14]=p*t-o*v-q*s,a[15]=k*v-l*t+m*s,a},"clone",0,function(a){let b=new bV(16);return b[0]=a[0],b[1]=a[1],b[2]=a[2],b[3]=a[3],b[4]=a[4],b[5]=a[5],b[6]=a[6],b[7]=a[7],b[8]=a[8],b[9]=a[9],b[10]=a[10],b[11]=a[11],b[12]=a[12],b[13]=a[13],b[14]=a[14],b[15]=a[15],b},"copy",0,function(a,b){return a[0]=b[0],a[1]=b[1],a[2]=b[2],a[3]=b[3],a[4]=b[4],a[5]=b[5],a[6]=b[6],a[7]=b[7],a[8]=b[8],a[9]=b[9],a[10]=b[10],a[11]=b[11],a[12]=b[12],a[13]=b[13],a[14]=b[14],a[15]=b[15],a},"create",0,function(){let a=new bV(16);return bV!=Float32Array&&(a[1]=0,a[2]=0,a[3]=0,a[4]=0,a[6]=0,a[7]=0,a[8]=0,a[9]=0,a[11]=0,a[12]=0,a[13]=0,a[14]=0),a[0]=1,a[5]=1,a[10]=1,a[15]=1,a},"decompose",0,function(a,b,c,d){b[0]=d[12],b[1]=d[13],b[2]=d[14];let e=d[0],f=d[1],g=d[2],h=d[4],i=d[5],j=d[6],k=d[8],l=d[9],m=d[10];c[0]=Math.sqrt(e*e+f*f+g*g),c[1]=Math.sqrt(h*h+i*i+j*j),c[2]=Math.sqrt(k*k+l*l+m*m);let n=1/c[0],o=1/c[1],p=1/c[2],q=e*n,r=f*o,s=g*p,t=h*n,u=i*o,v=j*p,w=k*n,x=l*o,y=m*p,z=q+u+y,A=0;return z>0?(A=2*Math.sqrt(z+1),a[3]=.25*A,a[0]=(v-x)/A,a[1]=(w-s)/A,a[2]=(r-t)/A):q>u&&q>y?(A=2*Math.sqrt(1+q-u-y),a[3]=(v-x)/A,a[0]=.25*A,a[1]=(r+t)/A,a[2]=(w+s)/A):u>y?(A=2*Math.sqrt(1+u-q-y),a[3]=(w-s)/A,a[0]=(r+t)/A,a[1]=.25*A,a[2]=(v+x)/A):(A=2*Math.sqrt(1+y-q-u),a[3]=(r-t)/A,a[0]=(w+s)/A,a[1]=(v+x)/A,a[2]=.25*A),a},"determinant",0,b_,"equals",0,function(a,b){let c=a[0],d=a[1],e=a[2],f=a[3],g=a[4],h=a[5],i=a[6],j=a[7],k=a[8],l=a[9],m=a[10],n=a[11],o=a[12],p=a[13],q=a[14],r=a[15],s=b[0],t=b[1],u=b[2],v=b[3],w=b[4],x=b[5],y=b[6],z=b[7],A=b[8],B=b[9],C=b[10],D=b[11],E=b[12],F=b[13],G=b[14],H=b[15];return Math.abs(c-s)<=1e-6*Math.max(1,Math.abs(c),Math.abs(s))&&Math.abs(d-t)<=1e-6*Math.max(1,Math.abs(d),Math.abs(t))&&Math.abs(e-u)<=1e-6*Math.max(1,Math.abs(e),Math.abs(u))&&Math.abs(f-v)<=1e-6*Math.max(1,Math.abs(f),Math.abs(v))&&Math.abs(g-w)<=1e-6*Math.max(1,Math.abs(g),Math.abs(w))&&Math.abs(h-x)<=1e-6*Math.max(1,Math.abs(h),Math.abs(x))&&Math.abs(i-y)<=1e-6*Math.max(1,Math.abs(i),Math.abs(y))&&Math.abs(j-z)<=1e-6*Math.max(1,Math.abs(j),Math.abs(z))&&Math.abs(k-A)<=1e-6*Math.max(1,Math.abs(k),Math.abs(A))&&Math.abs(l-B)<=1e-6*Math.max(1,Math.abs(l),Math.abs(B))&&Math.abs(m-C)<=1e-6*Math.max(1,Math.abs(m),Math.abs(C))&&Math.abs(n-D)<=1e-6*Math.max(1,Math.abs(n),Math.abs(D))&&Math.abs(o-E)<=1e-6*Math.max(1,Math.abs(o),Math.abs(E))&&Math.abs(p-F)<=1e-6*Math.max(1,Math.abs(p),Math.abs(F))&&Math.abs(q-G)<=1e-6*Math.max(1,Math.abs(q),Math.abs(G))&&Math.abs(r-H)<=1e-6*Math.max(1,Math.abs(r),Math.abs(H))},"exactEquals",0,function(a,b){return a[0]===b[0]&&a[1]===b[1]&&a[2]===b[2]&&a[3]===b[3]&&a[4]===b[4]&&a[5]===b[5]&&a[6]===b[6]&&a[7]===b[7]&&a[8]===b[8]&&a[9]===b[9]&&a[10]===b[10]&&a[11]===b[11]&&a[12]===b[12]&&a[13]===b[13]&&a[14]===b[14]&&a[15]===b[15]},"frob",0,function(a){return Math.sqrt(a[0]*a[0]+a[1]*a[1]+a[2]*a[2]+a[3]*a[3]+a[4]*a[4]+a[5]*a[5]+a[6]*a[6]+a[7]*a[7]+a[8]*a[8]+a[9]*a[9]+a[10]*a[10]+a[11]*a[11]+a[12]*a[12]+a[13]*a[13]+a[14]*a[14]+a[15]*a[15])},"fromQuat",0,b9,"fromQuat2",0,function(a,b){let c=new bV(3),d=-b[0],e=-b[1],f=-b[2],g=b[3],h=b[4],i=b[5],j=b[6],k=b[7],l=d*d+e*e+f*f+g*g;return l>0?(c[0]=(h*g+k*d+i*f-j*e)*2/l,c[1]=(i*g+k*e+j*d-h*f)*2/l,c[2]=(j*g+k*f+h*e-i*d)*2/l):(c[0]=(h*g+k*d+i*f-j*e)*2,c[1]=(i*g+k*e+j*d-h*f)*2,c[2]=(j*g+k*f+h*e-i*d)*2),b7(a,b,c),a},"fromRotation",0,function(a,b,c){let d,e,f,g=c[0],h=c[1],i=c[2],j=Math.sqrt(g*g+h*h+i*i);return j<1e-6?null:(g*=j=1/j,h*=j,i*=j,e=Math.sin(b),f=1-(d=Math.cos(b)),a[0]=g*g*f+d,a[1]=h*g*f+i*e,a[2]=i*g*f-h*e,a[3]=0,a[4]=g*h*f-i*e,a[5]=h*h*f+d,a[6]=i*h*f+g*e,a[7]=0,a[8]=g*i*f+h*e,a[9]=h*i*f-g*e,a[10]=i*i*f+d,a[11]=0,a[12]=0,a[13]=0,a[14]=0,a[15]=1,a)},"fromRotationTranslation",0,b7,"fromRotationTranslationScale",0,function(a,b,c,d){let e=b[0],f=b[1],g=b[2],h=b[3],i=e+e,j=f+f,k=g+g,l=e*i,m=e*j,n=e*k,o=f*j,p=f*k,q=g*k,r=h*i,s=h*j,t=h*k,u=d[0],v=d[1],w=d[2];return a[0]=(1-(o+q))*u,a[1]=(m+t)*u,a[2]=(n-s)*u,a[3]=0,a[4]=(m-t)*v,a[5]=(1-(l+q))*v,a[6]=(p+r)*v,a[7]=0,a[8]=(n+s)*w,a[9]=(p-r)*w,a[10]=(1-(l+o))*w,a[11]=0,a[12]=c[0],a[13]=c[1],a[14]=c[2],a[15]=1,a},"fromRotationTranslationScaleOrigin",0,function(a,b,c,d,e){let f=b[0],g=b[1],h=b[2],i=b[3],j=f+f,k=g+g,l=h+h,m=f*j,n=f*k,o=f*l,p=g*k,q=g*l,r=h*l,s=i*j,t=i*k,u=i*l,v=d[0],w=d[1],x=d[2],y=e[0],z=e[1],A=e[2],B=(1-(p+r))*v,C=(n+u)*v,D=(o-t)*v,E=(n-u)*w,F=(1-(m+r))*w,G=(q+s)*w,H=(o+t)*x,I=(q-s)*x,J=(1-(m+p))*x;return a[0]=B,a[1]=C,a[2]=D,a[3]=0,a[4]=E,a[5]=F,a[6]=G,a[7]=0,a[8]=H,a[9]=I,a[10]=J,a[11]=0,a[12]=c[0]+y-(B*y+E*z+H*A),a[13]=c[1]+z-(C*y+F*z+I*A),a[14]=c[2]+A-(D*y+G*z+J*A),a[15]=1,a},"fromScaling",0,function(a,b){return a[0]=b[0],a[1]=0,a[2]=0,a[3]=0,a[4]=0,a[5]=b[1],a[6]=0,a[7]=0,a[8]=0,a[9]=0,a[10]=b[2],a[11]=0,a[12]=0,a[13]=0,a[14]=0,a[15]=1,a},"fromTranslation",0,function(a,b){return a[0]=1,a[1]=0,a[2]=0,a[3]=0,a[4]=0,a[5]=1,a[6]=0,a[7]=0,a[8]=0,a[9]=0,a[10]=1,a[11]=0,a[12]=b[0],a[13]=b[1],a[14]=b[2],a[15]=1,a},"fromValues",0,function(a,b,c,d,e,f,g,h,i,j,k,l,m,n,o,p){let q=new bV(16);return q[0]=a,q[1]=b,q[2]=c,q[3]=d,q[4]=e,q[5]=f,q[6]=g,q[7]=h,q[8]=i,q[9]=j,q[10]=k,q[11]=l,q[12]=m,q[13]=n,q[14]=o,q[15]=p,q},"fromXRotation",0,function(a,b){let c=Math.sin(b),d=Math.cos(b);return a[0]=1,a[1]=0,a[2]=0,a[3]=0,a[4]=0,a[5]=d,a[6]=c,a[7]=0,a[8]=0,a[9]=-c,a[10]=d,a[11]=0,a[12]=0,a[13]=0,a[14]=0,a[15]=1,a},"fromYRotation",0,function(a,b){let c=Math.sin(b),d=Math.cos(b);return a[0]=d,a[1]=0,a[2]=-c,a[3]=0,a[4]=0,a[5]=1,a[6]=0,a[7]=0,a[8]=c,a[9]=0,a[10]=d,a[11]=0,a[12]=0,a[13]=0,a[14]=0,a[15]=1,a},"fromZRotation",0,function(a,b){let c=Math.sin(b),d=Math.cos(b);return a[0]=d,a[1]=c,a[2]=0,a[3]=0,a[4]=-c,a[5]=d,a[6]=0,a[7]=0,a[8]=0,a[9]=0,a[10]=1,a[11]=0,a[12]=0,a[13]=0,a[14]=0,a[15]=1,a},"frustum",0,ca,"getRotation",0,function(a,b){let c=new bV(3);b8(c,b);let d=1/c[0],e=1/c[1],f=1/c[2],g=b[0]*d,h=b[1]*e,i=b[2]*f,j=b[4]*d,k=b[5]*e,l=b[6]*f,m=b[8]*d,n=b[9]*e,o=b[10]*f,p=g+k+o,q=0;return p>0?(q=2*Math.sqrt(p+1),a[3]=.25*q,a[0]=(l-n)/q,a[1]=(m-i)/q,a[2]=(h-j)/q):g>k&&g>o?(q=2*Math.sqrt(1+g-k-o),a[3]=(l-n)/q,a[0]=.25*q,a[1]=(h+j)/q,a[2]=(m+i)/q):k>o?(q=2*Math.sqrt(1+k-g-o),a[3]=(m-i)/q,a[0]=(h+j)/q,a[1]=.25*q,a[2]=(l+n)/q):(q=2*Math.sqrt(1+o-g-k),a[3]=(h-j)/q,a[0]=(m+i)/q,a[1]=(l+n)/q,a[2]=.25*q),a},"getScaling",0,b8,"getTranslation",0,function(a,b){return a[0]=b[12],a[1]=b[13],a[2]=b[14],a},"identity",0,bY,"invert",0,b$,"lookAt",0,cd,"mul",0,b0,"multiply",0,b0,"multiplyScalar",0,function(a,b,c){return a[0]=b[0]*c,a[1]=b[1]*c,a[2]=b[2]*c,a[3]=b[3]*c,a[4]=b[4]*c,a[5]=b[5]*c,a[6]=b[6]*c,a[7]=b[7]*c,a[8]=b[8]*c,a[9]=b[9]*c,a[10]=b[10]*c,a[11]=b[11]*c,a[12]=b[12]*c,a[13]=b[13]*c,a[14]=b[14]*c,a[15]=b[15]*c,a},"multiplyScalarAndAdd",0,function(a,b,c,d){return a[0]=b[0]+c[0]*d,a[1]=b[1]+c[1]*d,a[2]=b[2]+c[2]*d,a[3]=b[3]+c[3]*d,a[4]=b[4]+c[4]*d,a[5]=b[5]+c[5]*d,a[6]=b[6]+c[6]*d,a[7]=b[7]+c[7]*d,a[8]=b[8]+c[8]*d,a[9]=b[9]+c[9]*d,a[10]=b[10]+c[10]*d,a[11]=b[11]+c[11]*d,a[12]=b[12]+c[12]*d,a[13]=b[13]+c[13]*d,a[14]=b[14]+c[14]*d,a[15]=b[15]+c[15]*d,a},"ortho",0,cc,"orthoNO",0,cc,"orthoZO",0,function(a,b,c,d,e,f,g){let h=1/(b-c),i=1/(d-e),j=1/(f-g);return a[0]=-2*h,a[1]=0,a[2]=0,a[3]=0,a[4]=0,a[5]=-2*i,a[6]=0,a[7]=0,a[8]=0,a[9]=0,a[10]=j,a[11]=0,a[12]=(b+c)*h,a[13]=(e+d)*i,a[14]=f*j,a[15]=1,a},"perspective",0,cb,"perspectiveFromFieldOfView",0,function(a,b,c,d){let e=Math.tan(b.upDegrees*Math.PI/180),f=Math.tan(b.downDegrees*Math.PI/180),g=Math.tan(b.leftDegrees*Math.PI/180),h=Math.tan(b.rightDegrees*Math.PI/180),i=2/(g+h),j=2/(e+f);return a[0]=i,a[1]=0,a[2]=0,a[3]=0,a[4]=0,a[5]=j,a[6]=0,a[7]=0,a[8]=-((g-h)*i*.5),a[9]=(e-f)*j*.5,a[10]=d/(c-d),a[11]=-1,a[12]=0,a[13]=0,a[14]=d*c/(c-d),a[15]=0,a},"perspectiveNO",0,cb,"perspectiveZO",0,function(a,b,c,d,e){let f=1/Math.tan(b/2);if(a[0]=f/c,a[1]=0,a[2]=0,a[3]=0,a[4]=0,a[5]=f,a[6]=0,a[7]=0,a[8]=0,a[9]=0,a[11]=-1,a[12]=0,a[13]=0,a[15]=0,null!=e&&e!==1/0){let b=1/(d-e);a[10]=e*b,a[14]=e*d*b}else a[10]=-1,a[14]=-d;return a},"rotate",0,b3,"rotateX",0,b4,"rotateY",0,b5,"rotateZ",0,b6,"scale",0,b2,"set",0,function(a,b,c,d,e,f,g,h,i,j,k,l,m,n,o,p,q){return a[0]=b,a[1]=c,a[2]=d,a[3]=e,a[4]=f,a[5]=g,a[6]=h,a[7]=i,a[8]=j,a[9]=k,a[10]=l,a[11]=m,a[12]=n,a[13]=o,a[14]=p,a[15]=q,a},"str",0,function(a){return`mat4(${a[0]}, ${a[1]}, ${a[2]}, ${a[3]}, ${a[4]}, ${a[5]}, ${a[6]}, ${a[7]}, ${a[8]}, ${a[9]}, ${a[10]}, ${a[11]}, ${a[12]}, ${a[13]}, ${a[14]}, ${a[15]})`},"sub",0,ce,"subtract",0,ce,"targetTo",0,function(a,b,c,d){let e=b[0],f=b[1],g=b[2],h=d[0],i=d[1],j=d[2],k=e-c[0],l=f-c[1],m=g-c[2],n=k*k+l*l+m*m;n>0&&(k*=n=1/Math.sqrt(n),l*=n,m*=n);let o=i*m-j*l,p=j*k-h*m,q=h*l-i*k;return(n=o*o+p*p+q*q)>0&&(o*=n=1/Math.sqrt(n),p*=n,q*=n),a[0]=o,a[1]=p,a[2]=q,a[3]=0,a[4]=l*q-m*p,a[5]=m*o-k*q,a[6]=k*p-l*o,a[7]=0,a[8]=k,a[9]=l,a[10]=m,a[11]=0,a[12]=e,a[13]=f,a[14]=g,a[15]=1,a},"translate",0,b1,"transpose",0,bZ],60464);var cf=a.i(60464),cf=cf;function cg(){let a=new bV(4);return bV!=Float32Array&&(a[0]=0,a[1]=0,a[2]=0,a[3]=0),a}function ch(a,b,c){return a[0]=b[0]-c[0],a[1]=b[1]-c[1],a[2]=b[2]-c[2],a[3]=b[3]-c[3],a}function ci(a,b,c){return a[0]=b[0]*c[0],a[1]=b[1]*c[1],a[2]=b[2]*c[2],a[3]=b[3]*c[3],a}function cj(a,b,c){return a[0]=b[0]/c[0],a[1]=b[1]/c[1],a[2]=b[2]/c[2],a[3]=b[3]/c[3],a}function ck(a,b){let c=b[0]-a[0],d=b[1]-a[1],e=b[2]-a[2],f=b[3]-a[3];return Math.sqrt(c*c+d*d+e*e+f*f)}function cl(a,b){let c=b[0]-a[0],d=b[1]-a[1],e=b[2]-a[2],f=b[3]-a[3];return c*c+d*d+e*e+f*f}function cm(a){let b=a[0],c=a[1],d=a[2],e=a[3];return Math.sqrt(b*b+c*c+d*d+e*e)}function cn(a){let b=a[0],c=a[1],d=a[2],e=a[3];return b*b+c*c+d*d+e*e}function co(a,b,c){let d=b[0],e=b[1],f=b[2],g=b[3];return a[0]=c[0]*d+c[4]*e+c[8]*f+c[12]*g,a[1]=c[1]*d+c[5]*e+c[9]*f+c[13]*g,a[2]=c[2]*d+c[6]*e+c[10]*f+c[14]*g,a[3]=c[3]*d+c[7]*e+c[11]*f+c[15]*g,a}let cp=(f=cg(),function(a,b,c,d,e,g){let h,i;for(b||(b=4),c||(c=0),i=d?Math.min(d*b+c,a.length):a.length,h=c;h<i;h+=b)f[0]=a[h],f[1]=a[h+1],f[2]=a[h+2],f[3]=a[h+3],e(f,f,g),a[h]=f[0],a[h+1]=f[1],a[h+2]=f[2],a[h+3]=f[3];return a});a.s(["add",0,function(a,b,c){return a[0]=b[0]+c[0],a[1]=b[1]+c[1],a[2]=b[2]+c[2],a[3]=b[3]+c[3],a},"ceil",0,function(a,b){return a[0]=Math.ceil(b[0]),a[1]=Math.ceil(b[1]),a[2]=Math.ceil(b[2]),a[3]=Math.ceil(b[3]),a},"clone",0,function(a){let b=new bV(4);return b[0]=a[0],b[1]=a[1],b[2]=a[2],b[3]=a[3],b},"copy",0,function(a,b){return a[0]=b[0],a[1]=b[1],a[2]=b[2],a[3]=b[3],a},"create",0,cg,"cross",0,function(a,b,c,d){let e=c[0]*d[1]-c[1]*d[0],f=c[0]*d[2]-c[2]*d[0],g=c[0]*d[3]-c[3]*d[0],h=c[1]*d[2]-c[2]*d[1],i=c[1]*d[3]-c[3]*d[1],j=c[2]*d[3]-c[3]*d[2],k=b[0],l=b[1],m=b[2],n=b[3];return a[0]=l*j-m*i+n*h,a[1]=-(k*j)+m*g-n*f,a[2]=k*i-l*g+n*e,a[3]=-(k*h)+l*f-m*e,a},"dist",0,ck,"distance",0,ck,"div",0,cj,"divide",0,cj,"dot",0,function(a,b){return a[0]*b[0]+a[1]*b[1]+a[2]*b[2]+a[3]*b[3]},"equals",0,function(a,b){let c=a[0],d=a[1],e=a[2],f=a[3],g=b[0],h=b[1],i=b[2],j=b[3];return Math.abs(c-g)<=1e-6*Math.max(1,Math.abs(c),Math.abs(g))&&Math.abs(d-h)<=1e-6*Math.max(1,Math.abs(d),Math.abs(h))&&Math.abs(e-i)<=1e-6*Math.max(1,Math.abs(e),Math.abs(i))&&Math.abs(f-j)<=1e-6*Math.max(1,Math.abs(f),Math.abs(j))},"exactEquals",0,function(a,b){return a[0]===b[0]&&a[1]===b[1]&&a[2]===b[2]&&a[3]===b[3]},"floor",0,function(a,b){return a[0]=Math.floor(b[0]),a[1]=Math.floor(b[1]),a[2]=Math.floor(b[2]),a[3]=Math.floor(b[3]),a},"forEach",0,cp,"fromValues",0,function(a,b,c,d){let e=new bV(4);return e[0]=a,e[1]=b,e[2]=c,e[3]=d,e},"inverse",0,function(a,b){return a[0]=1/b[0],a[1]=1/b[1],a[2]=1/b[2],a[3]=1/b[3],a},"len",0,cm,"length",0,cm,"lerp",0,function(a,b,c,d){let e=b[0],f=b[1],g=b[2],h=b[3];return a[0]=e+d*(c[0]-e),a[1]=f+d*(c[1]-f),a[2]=g+d*(c[2]-g),a[3]=h+d*(c[3]-h),a},"max",0,function(a,b,c){return a[0]=Math.max(b[0],c[0]),a[1]=Math.max(b[1],c[1]),a[2]=Math.max(b[2],c[2]),a[3]=Math.max(b[3],c[3]),a},"min",0,function(a,b,c){return a[0]=Math.min(b[0],c[0]),a[1]=Math.min(b[1],c[1]),a[2]=Math.min(b[2],c[2]),a[3]=Math.min(b[3],c[3]),a},"mul",0,ci,"multiply",0,ci,"negate",0,function(a,b){return a[0]=-b[0],a[1]=-b[1],a[2]=-b[2],a[3]=-b[3],a},"normalize",0,function(a,b){let c=b[0],d=b[1],e=b[2],f=b[3],g=c*c+d*d+e*e+f*f;return g>0&&(g=1/Math.sqrt(g)),a[0]=c*g,a[1]=d*g,a[2]=e*g,a[3]=f*g,a},"random",0,function(a,b){let c,d,e,f,g,h;b=void 0===b?1:b;do g=(c=2*bW()-1)*c+(d=2*bW()-1)*d;while(g>=1)do h=(e=2*bW()-1)*e+(f=2*bW()-1)*f;while(h>=1)let i=Math.sqrt((1-g)/h);return a[0]=b*c,a[1]=b*d,a[2]=b*e*i,a[3]=b*f*i,a},"round",0,function(a,b){return a[0]=bX(b[0]),a[1]=bX(b[1]),a[2]=bX(b[2]),a[3]=bX(b[3]),a},"scale",0,function(a,b,c){return a[0]=b[0]*c,a[1]=b[1]*c,a[2]=b[2]*c,a[3]=b[3]*c,a},"scaleAndAdd",0,function(a,b,c,d){return a[0]=b[0]+c[0]*d,a[1]=b[1]+c[1]*d,a[2]=b[2]+c[2]*d,a[3]=b[3]+c[3]*d,a},"set",0,function(a,b,c,d,e){return a[0]=b,a[1]=c,a[2]=d,a[3]=e,a},"sqrDist",0,cl,"sqrLen",0,cn,"squaredDistance",0,cl,"squaredLength",0,cn,"str",0,function(a){return`vec4(${a[0]}, ${a[1]}, ${a[2]}, ${a[3]})`},"sub",0,ch,"subtract",0,ch,"transformMat4",0,co,"transformQuat",0,function(a,b,c){let d=b[0],e=b[1],f=b[2],g=c[0],h=c[1],i=c[2],j=c[3],k=j*d+h*f-i*e,l=j*e+i*d-g*f,m=j*f+g*e-h*d,n=-g*d-h*e-i*f;return a[0]=k*j+-(n*g)+-(l*i)- -(m*h),a[1]=l*j+-(n*h)+-(m*g)- -(k*i),a[2]=m*j+-(n*i)+-(k*h)- -(l*g),a[3]=b[3],a},"zero",0,function(a){return a[0]=0,a[1]=0,a[2]=0,a[3]=0,a}],99295);var cq=a.i(99295),cq=cq,cr=a.i(26037);let cs=new cr.Log({id:"deck"});(j=n||(n={}))[j.Start=1]="Start",j[j.Move=2]="Move",j[j.End=4]="End",j[j.Cancel=8]="Cancel",(k=o||(o={}))[k.None=0]="None",k[k.Left=1]="Left",k[k.Right=2]="Right",k[k.Up=4]="Up",k[k.Down=8]="Down",k[k.Horizontal=3]="Horizontal",k[k.Vertical=12]="Vertical",k[k.All=15]="All",(l=p||(p={}))[l.Possible=1]="Possible",l[l.Began=2]="Began",l[l.Changed=4]="Changed",l[l.Ended=8]="Ended",l[l.Recognized=8]="Recognized",l[l.Cancelled=16]="Cancelled",l[l.Failed=32]="Failed";let ct="manipulation",cu="none",cv="pan-x",cw="pan-y";class cx{constructor(a,b){this.actions="",this.manager=a,this.set(b)}set(a){"compute"===a&&(a=this.compute()),this.manager.element&&(this.manager.element.style.touchAction=a,this.actions=a)}update(){this.set(this.manager.options.touchAction)}compute(){let a=[];for(let b of this.manager.recognizers)b.options.enable&&(a=a.concat(b.getTouchAction()));var b=a.join(" ");if(b.includes(cu))return cu;let c=b.includes(cv),d=b.includes(cw);return c&&d?cu:c||d?c?cv:cw:b.includes(ct)?ct:"auto"}}function cy(a){return a.trim().split(/\s+/g)}function cz(a,b,c){if(a)for(let d of cy(b))a.addEventListener(d,c,!1)}function cA(a,b,c){if(a)for(let d of cy(b))a.removeEventListener(d,c,!1)}function cB(a){return(a.ownerDocument||a).defaultView}function cC(a){let b=a.length;if(1===b)return{x:Math.round(a[0].clientX),y:Math.round(a[0].clientY)};let c=0,d=0,e=0;for(;e<b;)c+=a[e].clientX,d+=a[e].clientY,e++;return{x:Math.round(c/b),y:Math.round(d/b)}}function cD(a){let b=[],c=0;for(;c<a.pointers.length;)b[c]={clientX:Math.round(a.pointers[c].clientX),clientY:Math.round(a.pointers[c].clientY)},c++;return{timeStamp:Date.now(),pointers:b,center:cC(b),deltaX:a.deltaX,deltaY:a.deltaY}}function cE(a,b){let c=b.x-a.x,d=b.y-a.y;return Math.sqrt(c*c+d*d)}function cF(a,b){let c=b.clientX-a.clientX,d=b.clientY-a.clientY;return Math.sqrt(c*c+d*d)}function cG(a,b){let c=b.clientX-a.clientX;return 180*Math.atan2(b.clientY-a.clientY,c)/Math.PI}function cH(a,b){return a===b?o.None:Math.abs(a)>=Math.abs(b)?a<0?o.Left:o.Right:b<0?o.Up:o.Down}function cI(a,b,c){return{x:b/a||0,y:c/a||0}}class cJ{constructor(a){this.evEl="",this.evWin="",this.evTarget="",this.domHandler=a=>{this.manager.options.enable&&this.handler(a)},this.manager=a,this.element=a.element,this.target=a.options.inputTarget||a.element}callback(a,b){var c;let d,e,f,g,h;c=this.manager,d=b.pointers.length,e=b.changedPointers.length,f=a&n.Start&&d-e==0,g=a&(n.End|n.Cancel)&&d-e==0,b.isFirst=!!f,b.isFinal=!!g,f&&(c.session={}),b.eventType=a,h=function(a,b){var c,d;let e,f,g,h,i,{session:j}=a,{pointers:k}=b,{length:l}=k;j.firstInput||(j.firstInput=cD(b)),l>1&&!j.firstMultiple?j.firstMultiple=cD(b):1===l&&(j.firstMultiple=!1);let{firstInput:m,firstMultiple:o}=j,p=o?o.center:m.center,q=b.center=cC(k);b.timeStamp=Date.now(),b.deltaTime=b.timeStamp-m.timeStamp,e=q.x-p.x,b.angle=180*Math.atan2(q.y-p.y,e)/Math.PI,b.distance=cE(p,q);let{deltaX:r,deltaY:s}=(f=b.center,g=j.offsetDelta,h=j.prevDelta,i=j.prevInput,(b.eventType===n.Start||i?.eventType===n.End)&&(h=j.prevDelta={x:i?.deltaX||0,y:i?.deltaY||0},g=j.offsetDelta={x:f.x,y:f.y}),{deltaX:h.x+(f.x-g.x),deltaY:h.y+(f.y-g.y)});b.deltaX=r,b.deltaY=s,b.offsetDirection=cH(b.deltaX,b.deltaY);let t=cI(b.deltaTime,b.deltaX,b.deltaY);b.overallVelocityX=t.x,b.overallVelocityY=t.y,b.overallVelocity=Math.abs(t.x)>Math.abs(t.y)?t.x:t.y,b.scale=o?(c=o.pointers,cF(k[0],k[1])/cF(c[0],c[1])):1,b.rotation=o?(d=o.pointers,cG(k[1],k[0])-cG(d[1],d[0])):0,b.maxPointers=j.prevInput?b.pointers.length>j.prevInput.maxPointers?b.pointers.length:j.prevInput.maxPointers:b.pointers.length;let u=a.element;return function(a,b){let c=a;for(;c;){if(c===b)return!0;c=c.parentNode}return!1}(b.srcEvent.target,u)&&(u=b.srcEvent.target),b.target=u,!function(a,b){let c,d,e,f,g=a.lastInterval||b,h=b.timeStamp-g.timeStamp;if(b.eventType!==n.Cancel&&(h>25||void 0===g.velocity)){let i=b.deltaX-g.deltaX,j=b.deltaY-g.deltaY,k=cI(h,i,j);d=k.x,e=k.y,c=Math.abs(k.x)>Math.abs(k.y)?k.x:k.y,f=cH(i,j),a.lastInterval=b}else c=g.velocity,d=g.velocityX,e=g.velocityY,f=g.direction;b.velocity=c,b.velocityX=d,b.velocityY=e,b.direction=f}(j,b),b}(c,b),c.emit("hammer.input",h),c.recognize(h),c.session.prevInput=h}init(){cz(this.element,this.evEl,this.domHandler),cz(this.target,this.evTarget,this.domHandler),cz(cB(this.element),this.evWin,this.domHandler)}destroy(){cA(this.element,this.evEl,this.domHandler),cA(this.target,this.evTarget,this.domHandler),cA(cB(this.element),this.evWin,this.domHandler)}}let cK={pointerdown:n.Start,pointermove:n.Move,pointerup:n.End,pointercancel:n.Cancel,pointerout:n.Cancel};class cL extends cJ{constructor(a){super(a),this.evEl="pointerdown",this.evWin="pointermove pointerup pointercancel",this.store=this.manager.session.pointerEvents=[],this.init()}handler(a){let{store:b}=this,c=!1,d=cK[a.type],e=a.pointerType,f="touch"===e,g=b.findIndex(b=>b.pointerId===a.pointerId);d&n.Start&&(a.buttons||f)?g<0&&(b.push(a),g=b.length-1):d&(n.End|n.Cancel)&&(c=!0),!(g<0)&&(b[g]=a,this.callback(d,{pointers:b,changedPointers:[a],eventType:d,pointerType:e,srcEvent:a}),c&&b.splice(g,1))}}let cM=["","webkit","Moz","MS","ms","o"],cN={touchAction:"compute",enable:!0,inputTarget:null,cssProps:{userSelect:"none",userDrag:"none",touchCallout:"none",tapHighlightColor:"rgba(0,0,0,0)"}};class cO{constructor(a,b){this.options={...cN,...b,cssProps:{...cN.cssProps,...b.cssProps},inputTarget:b.inputTarget||a},this.handlers={},this.session={},this.recognizers=[],this.oldCssProps={},this.element=a,this.input=new cL(this),this.touchAction=new cx(this,this.options.touchAction),this.toggleCssProps(!0)}set(a){return Object.assign(this.options,a),a.touchAction&&this.touchAction.update(),a.inputTarget&&(this.input.destroy(),this.input.target=a.inputTarget,this.input.init()),this}stop(a){this.session.stopped=a?2:1}recognize(a){let b,{session:c}=this;if(c.stopped)return;this.session.prevented&&a.srcEvent.preventDefault();let{recognizers:d}=this,{curRecognizer:e}=c;(!e||e&&e.state&p.Recognized)&&(e=c.curRecognizer=null);let f=0;for(;f<d.length;)b=d[f],2!==c.stopped&&(!e||b===e||b.canRecognizeWith(e))?b.recognize(a):b.reset(),!e&&b.state&(p.Began|p.Changed|p.Ended)&&(e=c.curRecognizer=b),f++}get(a){let{recognizers:b}=this;for(let c=0;c<b.length;c++)if(b[c].options.event===a)return b[c];return null}add(a){if(Array.isArray(a)){for(let b of a)this.add(b);return this}let b=this.get(a.options.event);return b&&this.remove(b),this.recognizers.push(a),a.manager=this,this.touchAction.update(),a}remove(a){if(Array.isArray(a)){for(let b of a)this.remove(b);return this}let b="string"==typeof a?this.get(a):a;if(b){let{recognizers:a}=this,c=a.indexOf(b);-1!==c&&(a.splice(c,1),this.touchAction.update())}return this}on(a,b){if(!a||!b)return;let{handlers:c}=this;for(let d of cy(a))c[d]=c[d]||[],c[d].push(b)}off(a,b){if(!a)return;let{handlers:c}=this;for(let d of cy(a))b?c[d]&&c[d].splice(c[d].indexOf(b),1):delete c[d]}emit(a,b){let c=this.handlers[a]&&this.handlers[a].slice();if(!c||!c.length)return;b.type=a,b.preventDefault=function(){b.srcEvent.preventDefault()};let d=0;for(;d<c.length;)c[d](b),d++}destroy(){this.toggleCssProps(!1),this.handlers={},this.session={},this.input.destroy(),this.element=null}toggleCssProps(a){let{element:b}=this;if(b){for(let[c,d]of Object.entries(this.options.cssProps)){let e=function(a,b){let c=b[0].toUpperCase()+b.slice(1);for(let d of cM){let e=d?d+c:b;if(e in a)return e}}(b.style,c);a?(this.oldCssProps[e]=b.style[e],b.style[e]=d):b.style[e]=this.oldCssProps[e]||""}a||(this.oldCssProps={})}}}let cP=1;function cQ(a){return a&p.Cancelled?"cancel":a&p.Ended?"end":a&p.Changed?"move":a&p.Began?"start":""}class cR{constructor(a){this.options=a,this.id=cP++,this.state=p.Possible,this.simultaneous={},this.requireFail=[]}set(a){return Object.assign(this.options,a),this.manager.touchAction.update(),this}recognizeWith(a){let b;if(Array.isArray(a)){for(let b of a)this.recognizeWith(b);return this}if("string"==typeof a){if(!(b=this.manager.get(a)))throw Error(`Cannot find recognizer ${a}`)}else b=a;let{simultaneous:c}=this;return c[b.id]||(c[b.id]=b,b.recognizeWith(this)),this}dropRecognizeWith(a){let b;if(Array.isArray(a)){for(let b of a)this.dropRecognizeWith(b);return this}return(b="string"==typeof a?this.manager.get(a):a)&&delete this.simultaneous[b.id],this}requireFailure(a){let b;if(Array.isArray(a)){for(let b of a)this.requireFailure(b);return this}if("string"==typeof a){if(!(b=this.manager.get(a)))throw Error(`Cannot find recognizer ${a}`)}else b=a;let{requireFail:c}=this;return -1===c.indexOf(b)&&(c.push(b),b.requireFailure(this)),this}dropRequireFailure(a){let b;if(Array.isArray(a)){for(let b of a)this.dropRequireFailure(b);return this}if(b="string"==typeof a?this.manager.get(a):a){let a=this.requireFail.indexOf(b);a>-1&&this.requireFail.splice(a,1)}return this}hasRequireFailures(){return!!this.requireFail.find(a=>a.options.enable)}canRecognizeWith(a){return!!this.simultaneous[a.id]}emit(a){if(!a)return;let{state:b}=this;b<p.Ended&&this.manager.emit(this.options.event+cQ(b),a),this.manager.emit(this.options.event,a),a.additionalEvent&&this.manager.emit(a.additionalEvent,a),b>=p.Ended&&this.manager.emit(this.options.event+cQ(b),a)}tryEmit(a){this.canEmit()?this.emit(a):this.state=p.Failed}canEmit(){let a=0;for(;a<this.requireFail.length;){if(!(this.requireFail[a].state&(p.Failed|p.Possible)))return!1;a++}return!0}recognize(a){let b={...a};if(!this.options.enable){this.reset(),this.state=p.Failed;return}this.state&(p.Recognized|p.Cancelled|p.Failed)&&(this.state=p.Possible),this.state=this.process(b),this.state&(p.Began|p.Changed|p.Ended|p.Cancelled)&&this.tryEmit(b)}getEventNames(){return[this.options.event]}reset(){}}class cS extends cR{attrTest(a){let b=this.options.pointers;return 0===b||a.pointers.length===b}process(a){let{state:b}=this,{eventType:c}=a,d=b&(p.Began|p.Changed),e=this.attrTest(a);return d&&(c&n.Cancel||!e)?b|p.Cancelled:d||e?c&n.End?b|p.Ended:b&p.Began?b|p.Changed:p.Began:p.Failed}}class cT extends cR{constructor(a={}){super({enable:!0,event:"tap",pointers:1,taps:1,interval:300,time:250,threshold:9,posThreshold:10,...a}),this.pTime=null,this.pCenter=null,this._timer=null,this._input=null,this.count=0}getTouchAction(){return[ct]}process(a){let{options:b}=this,c=a.pointers.length===b.pointers,d=a.distance<b.threshold,e=a.deltaTime<b.time;if(this.reset(),a.eventType&n.Start&&0===this.count)return this.failTimeout();if(d&&e&&c){if(a.eventType!==n.End)return this.failTimeout();let c=!this.pTime||a.timeStamp-this.pTime<b.interval,d=!this.pCenter||cE(this.pCenter,a.center)<b.posThreshold;if(this.pTime=a.timeStamp,this.pCenter=a.center,d&&c?this.count+=1:this.count=1,this._input=a,0==this.count%b.taps)return this.hasRequireFailures()?(this._timer=setTimeout(()=>{this.state=p.Recognized,this.tryEmit(this._input)},b.interval),p.Began):p.Recognized}return p.Failed}failTimeout(){return this._timer=setTimeout(()=>{this.state=p.Failed},this.options.interval),p.Failed}reset(){clearTimeout(this._timer)}emit(a){this.state===p.Recognized&&(a.tapCount=this.count,this.manager.emit(this.options.event,a))}}let cU=["","start","move","end","cancel","up","down","left","right"];class cV extends cS{constructor(a={}){super({enable:!0,pointers:1,event:"pan",threshold:10,direction:o.All,...a}),this.pX=null,this.pY=null}getTouchAction(){let{options:{direction:a}}=this,b=[];return a&o.Horizontal&&b.push(cw),a&o.Vertical&&b.push(cv),b}getEventNames(){return cU.map(a=>this.options.event+a)}directionTest(a){let{options:b}=this,c=!0,{distance:d}=a,{direction:e}=a,f=a.deltaX,g=a.deltaY;return e&b.direction||(b.direction&o.Horizontal?(e=0===f?o.None:f<0?o.Left:o.Right,c=f!==this.pX,d=Math.abs(a.deltaX)):(e=0===g?o.None:g<0?o.Up:o.Down,c=g!==this.pY,d=Math.abs(a.deltaY))),a.direction=e,c&&d>b.threshold&&!!(e&b.direction)}attrTest(a){return super.attrTest(a)&&(!!(this.state&p.Began)||!(this.state&p.Began)&&this.directionTest(a))}emit(a){this.pX=a.deltaX,this.pY=a.deltaY;let b=o[a.direction].toLowerCase();b&&(a.additionalEvent=this.options.event+b),super.emit(a)}}let cW=["","start","move","end","cancel","in","out"];class cX{constructor(a,b,c){this.element=a,this.callback=b,this.options=c}}let cY="u">typeof navigator&&navigator.userAgent?navigator.userAgent.toLowerCase():"";a.g,a.g;let cZ=-1!==cY.indexOf("firefox");class c$ extends cX{constructor(a,b,c){super(a,b,{enable:!0,...c}),this.handleEvent=a=>{if(!this.options.enable)return;let b=a.deltaY;globalThis.WheelEvent&&(cZ&&a.deltaMode===globalThis.WheelEvent.DOM_DELTA_PIXEL&&(b/=globalThis.devicePixelRatio),a.deltaMode===globalThis.WheelEvent.DOM_DELTA_LINE&&(b*=40)),0!==b&&b%4.000244140625==0&&(b=Math.floor(b/4.000244140625)),a.shiftKey&&b&&(b*=.25),this.callback({type:"wheel",center:{x:a.clientX,y:a.clientY},delta:-b,srcEvent:a,pointerType:"mouse",target:a.target})},a.addEventListener("wheel",this.handleEvent,{passive:!1})}destroy(){this.element.removeEventListener("wheel",this.handleEvent)}enableEventType(a,b){"wheel"===a&&(this.options.enable=b)}}let c_=["mousedown","mousemove","mouseup","mouseover","mouseout","mouseleave"];class c0 extends cX{constructor(a,b,c){super(a,b,{enable:!0,...c}),this.handleEvent=a=>{this.handleOverEvent(a),this.handleOutEvent(a),this.handleEnterEvent(a),this.handleLeaveEvent(a),this.handleMoveEvent(a)},this.pressed=!1;const{enable:d}=this.options;this.enableMoveEvent=d,this.enableLeaveEvent=d,this.enableEnterEvent=d,this.enableOutEvent=d,this.enableOverEvent=d,c_.forEach(b=>a.addEventListener(b,this.handleEvent))}destroy(){c_.forEach(a=>this.element.removeEventListener(a,this.handleEvent))}enableEventType(a,b){switch(a){case"pointermove":this.enableMoveEvent=b;break;case"pointerover":this.enableOverEvent=b;break;case"pointerout":this.enableOutEvent=b;break;case"pointerenter":this.enableEnterEvent=b;break;case"pointerleave":this.enableLeaveEvent=b}}handleOverEvent(a){this.enableOverEvent&&"mouseover"===a.type&&this._emit("pointerover",a)}handleOutEvent(a){this.enableOutEvent&&"mouseout"===a.type&&this._emit("pointerout",a)}handleEnterEvent(a){this.enableEnterEvent&&"mouseenter"===a.type&&this._emit("pointerenter",a)}handleLeaveEvent(a){this.enableLeaveEvent&&"mouseleave"===a.type&&this._emit("pointerleave",a)}handleMoveEvent(a){if(this.enableMoveEvent)switch(a.type){case"mousedown":a.button>=0&&(this.pressed=!0);break;case"mousemove":0===a.buttons&&(this.pressed=!1),this.pressed||this._emit("pointermove",a);break;case"mouseup":this.pressed=!1}}_emit(a,b){this.callback({type:a,center:{x:b.clientX,y:b.clientY},srcEvent:b,pointerType:"mouse",target:b.target})}}let c1=["keydown","keyup"];class c2 extends cX{constructor(a,b,c){super(a,b,{enable:!0,tabIndex:0,...c}),this.handleEvent=a=>{let b=a.target||a.srcElement;("INPUT"!==b.tagName||"text"!==b.type)&&"TEXTAREA"!==b.tagName&&(this.enableDownEvent&&"keydown"===a.type&&this.callback({type:"keydown",srcEvent:a,key:a.key,target:a.target}),this.enableUpEvent&&"keyup"===a.type&&this.callback({type:"keyup",srcEvent:a,key:a.key,target:a.target}))},this.enableDownEvent=this.options.enable,this.enableUpEvent=this.options.enable,a.tabIndex=this.options.tabIndex,a.style.outline="none",c1.forEach(b=>a.addEventListener(b,this.handleEvent))}destroy(){c1.forEach(a=>this.element.removeEventListener(a,this.handleEvent))}enableEventType(a,b){"keydown"===a&&(this.enableDownEvent=b),"keyup"===a&&(this.enableUpEvent=b)}}class c3 extends cX{constructor(a,b,c){super(a,b,c),this.handleEvent=a=>{this.options.enable&&this.callback({type:"contextmenu",center:{x:a.clientX,y:a.clientY},srcEvent:a,pointerType:"mouse",target:a.target})},a.addEventListener("contextmenu",this.handleEvent)}destroy(){this.element.removeEventListener("contextmenu",this.handleEvent)}enableEventType(a,b){"contextmenu"===a&&(this.options.enable=b)}}let c4={pointerdown:1,pointermove:2,pointerup:4,mousedown:1,mousemove:2,mouseup:4},c5={srcElement:"root",priority:0};class c6{constructor(a,b){this.handleEvent=a=>{if(this.isEmpty())return;let b=this._normalizeEvent(a),c=a.srcEvent.target;for(;c&&c!==b.rootElement;){if(this._emit(b,c),b.handled)return;c=c.parentNode}this._emit(b,"root")},this.eventManager=a,this.recognizerName=b,this.handlers=[],this.handlersByElement=new Map,this._active=!1}isEmpty(){return!this._active}add(a,b,c,d=!1,e=!1){let{handlers:f,handlersByElement:g}=this,h={...c5,...c},i=g.get(h.srcElement);i||(i=[],g.set(h.srcElement,i));let j={type:a,handler:b,srcElement:h.srcElement,priority:h.priority};d&&(j.once=!0),e&&(j.passive=!0),f.push(j),this._active=this._active||!j.passive;let k=i.length-1;for(;k>=0&&!(i[k].priority>=j.priority);)k--;i.splice(k+1,0,j)}remove(a,b){let{handlers:c,handlersByElement:d}=this;for(let e=c.length-1;e>=0;e--){let f=c[e];if(f.type===a&&f.handler===b){c.splice(e,1);let a=d.get(f.srcElement);a.splice(a.indexOf(f),1),0===a.length&&d.delete(f.srcElement)}}this._active=c.some(a=>!a.passive)}_emit(a,b){let c=this.handlersByElement.get(b);if(c){let b=!1,d=()=>{a.handled=!0},e=()=>{a.handled=!0,b=!0},f=[];for(let g=0;g<c.length;g++){let{type:h,handler:i,once:j}=c[g];if(i({...a,type:h,stopPropagation:d,stopImmediatePropagation:e}),j&&f.push(c[g]),b)break}for(let a=0;a<f.length;a++){let{type:b,handler:c}=f[a];this.remove(b,c)}}}_normalizeEvent(a){let b=this.eventManager.getElement();return{...a,...function(a){let b=c4[a.srcEvent.type];if(!b)return null;let{buttons:c,button:d}=a.srcEvent,e=!1,f=!1,g=!1;return 2===b?(e=!!(1&c),f=!!(4&c),g=!!(2&c)):(e=0===d,f=1===d,g=2===d),{leftButton:e,middleButton:f,rightButton:g}}(a),...function(a,b){let c=a.center;if(!c)return null;let d=b.getBoundingClientRect(),e=d.width/b.offsetWidth||1,f=d.height/b.offsetHeight||1,g={x:(c.x-d.left-b.clientLeft)/e,y:(c.y-d.top-b.clientTop)/f};return{center:c,offsetCenter:g}}(a,b),preventDefault:()=>{a.srcEvent.preventDefault()},stopImmediatePropagation:null,stopPropagation:null,handled:!1,rootElement:b}}}class c7{constructor(a=null,b={}){if(this._onBasicInput=a=>{this.manager.emit(a.srcEvent.type,a)},this._onOtherEvent=a=>{this.manager.emit(a.type,a)},this.options={recognizers:[],events:{},touchAction:"compute",tabIndex:0,cssProps:{},...b},this.events=new Map,this.element=a,!a)return;for(const b of(this.manager=new cO(a,this.options),this.options.recognizers)){const{recognizer:a,recognizeWith:c,requireFailure:d}=function(a){let b;if("recognizer"in a)return a;let c=Array.isArray(a)?[...a]:[a];return{recognizer:b="function"==typeof c[0]?new(c.shift())(c.shift()||{}):c.shift(),recognizeWith:"string"==typeof c[0]?[c[0]]:c[0],requireFailure:"string"==typeof c[1]?[c[1]]:c[1]}}(b);this.manager.add(a),c&&a.recognizeWith(c),d&&a.requireFailure(d)}this.manager.on("hammer.input",this._onBasicInput),this.wheelInput=new c$(a,this._onOtherEvent,{enable:!1}),this.moveInput=new c0(a,this._onOtherEvent,{enable:!1}),this.keyInput=new c2(a,this._onOtherEvent,{enable:!1,tabIndex:b.tabIndex}),this.contextmenuInput=new c3(a,this._onOtherEvent,{enable:!1}),this.on(this.options.events)}getElement(){return this.element}destroy(){this.element&&(this.wheelInput.destroy(),this.moveInput.destroy(),this.keyInput.destroy(),this.contextmenuInput.destroy(),this.manager.destroy())}on(a,b,c){this._addEventHandler(a,b,c,!1)}once(a,b,c){this._addEventHandler(a,b,c,!0)}watch(a,b,c){this._addEventHandler(a,b,c,!1,!0)}off(a,b){this._removeEventHandler(a,b)}_toggleRecognizer(a,b){let{manager:c}=this;if(!c)return;let d=c.get(a);d&&(d.set({enable:b}),c.touchAction.update()),this.wheelInput?.enableEventType(a,b),this.moveInput?.enableEventType(a,b),this.keyInput?.enableEventType(a,b),this.contextmenuInput?.enableEventType(a,b)}_addEventHandler(a,b,c,d,e){if("string"!=typeof a){for(let[f,g]of(c=b,Object.entries(a)))this._addEventHandler(f,g,c,d,e);return}let{manager:f,events:g}=this;if(!f)return;let h=g.get(a);!h&&(h=new c6(this,this._getRecognizerName(a)||a),g.set(a,h),f&&f.on(a,h.handleEvent)),h.add(a,b,c,d,e),h.isEmpty()||this._toggleRecognizer(h.recognizerName,!0)}_removeEventHandler(a,b){if("string"!=typeof a){for(let[b,c]of Object.entries(a))this._removeEventHandler(b,c);return}let{events:c}=this,d=c.get(a);if(d&&(d.remove(a,b),d.isEmpty())){let{recognizerName:a}=d,b=!1;for(let d of c.values())if(d.recognizerName===a&&!d.isEmpty()){b=!0;break}b||this._toggleRecognizer(a,!1)}}_getRecognizerName(a){return this.manager.recognizers.find(b=>b.getEventNames().includes(a))?.options.event}}let c8={DEFAULT:"default",LNGLAT:"lnglat",METER_OFFSETS:"meter-offsets",LNGLAT_OFFSETS:"lnglat-offsets",CARTESIAN:"cartesian"};Object.defineProperty(c8,"IDENTITY",{get:()=>(cs.deprecated("COORDINATE_SYSTEM.IDENTITY","COORDINATE_SYSTEM.CARTESIAN")(),c8.CARTESIAN)});let c9={WEB_MERCATOR:1,GLOBE:2,WEB_MERCATOR_AUTO_OFFSET:4,IDENTITY:0},da={common:0,meters:1,pixels:2},db={click:"onClick",dblclick:"onClick",panstart:"onDragStart",panmove:"onDrag",panend:"onDragEnd"},dc={multipan:[cV,{threshold:10,direction:o.Vertical,pointers:2}],pinch:[class extends cS{constructor(a={}){super({enable:!0,event:"pinch",threshold:0,pointers:2,...a})}getTouchAction(){return[cu]}getEventNames(){return cW.map(a=>this.options.event+a)}attrTest(a){return super.attrTest(a)&&(Math.abs(a.scale-1)>this.options.threshold||!!(this.state&p.Began))}emit(a){if(1!==a.scale){let b=a.scale<1?"in":"out";a.additionalEvent=this.options.event+b}super.emit(a)}},{},null,["multipan"]],pan:[cV,{threshold:1},["pinch"],["multipan"]],dblclick:[cT,{event:"dblclick",taps:2}],click:[cT,{event:"click"},null,["dblclick"]]};function dd(a){let b,c={};return d=>{for(let e in d)if(!function(a,b){if(a===b)return!0;if(Array.isArray(a)){let c=a.length;if(!b||b.length!==c)return!1;for(let d=0;d<c;d++)if(a[d]!==b[d])return!1;return!0}return!1}(d[e],c[e])){b=a(d),c=d;break}return b}}let de=[0,0,0,0],df=[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,0],dg=[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1],dh=[0,0,0],di=[0,0,0],dj={default:-1,cartesian:0,lnglat:1,"meter-offsets":2,"lnglat-offsets":3};function dk(a){let b=dj[a];if(void 0===b)throw Error(`Invalid coordinateSystem: ${a}`);return b}let dl=dd(function({viewport:a,devicePixelRatio:b,coordinateSystem:c,coordinateOrigin:d}){let{projectionCenter:e,viewProjectionMatrix:f,originCommon:g,cameraPosCommon:h,shaderCoordinateOrigin:i,geospatialOrigin:j}=function(a,b,c){let{viewMatrixUncentered:d,projectionMatrix:e}=a,{viewMatrix:f,viewProjectionMatrix:g}=a,h=de,i=de,j=a.cameraPosition,{geospatialOrigin:k,shaderCoordinateOrigin:l,offsetMode:m}=dm(a,b,c);return m&&(i=a.projectPosition(k||l),j=[j[0]-i[0],j[1]-i[1],j[2]-i[2]],i[3]=1,h=cq.transformMat4([],i,g),f=d||f,g=cf.multiply([],e,f),g=cf.multiply([],g,df)),{viewMatrix:f,viewProjectionMatrix:g,projectionCenter:h,originCommon:i,cameraPosCommon:j,shaderCoordinateOrigin:l,geospatialOrigin:k}}(a,c,d),k=a.getDistanceScales(),l=[a.width*b,a.height*b],m=cq.transformMat4([],[0,0,-a.focalDistance,1],a.projectionMatrix)[3]||1,n={coordinateSystem:dk(c),projectionMode:a.projectionMode,coordinateOrigin:i,commonOrigin:g.slice(0,3),center:e,pseudoMeters:!!a._pseudoMeters,viewportSize:l,devicePixelRatio:b,focalDistance:m,commonUnitsPerMeter:k.unitsPerMeter,commonUnitsPerWorldUnit:k.unitsPerMeter,commonUnitsPerWorldUnit2:dh,scale:a.scale,wrapLongitude:!1,viewProjectionMatrix:f,modelMatrix:dg,cameraPosition:h};if(j){let b=a.getDistanceScales(j);switch(c){case"meter-offsets":n.commonUnitsPerWorldUnit=b.unitsPerMeter,n.commonUnitsPerWorldUnit2=b.unitsPerMeter2;break;case"lnglat":case"lnglat-offsets":a._pseudoMeters||(n.commonUnitsPerMeter=b.unitsPerMeter),n.commonUnitsPerWorldUnit=b.unitsPerDegree,n.commonUnitsPerWorldUnit2=b.unitsPerDegree2;break;case"cartesian":n.commonUnitsPerWorldUnit=[1,1,b.unitsPerMeter[2]],n.commonUnitsPerWorldUnit2=[0,0,b.unitsPerMeter2[2]]}}return n});function dm(a,b,c=di){let d;c.length<3&&(c=[c[0],c[1],0]);let e=c,f=!0;switch(d="lnglat-offsets"===b||"meter-offsets"===b?c:a.isGeospatial?[Math.fround(a.longitude),Math.fround(a.latitude),0]:null,a.projectionMode){case c9.WEB_MERCATOR:("lnglat"===b||"cartesian"===b)&&(d=[0,0,0],f=!1);break;case c9.WEB_MERCATOR_AUTO_OFFSET:"lnglat"===b?e=d:"cartesian"===b&&(e=[Math.fround(a.center[0]),Math.fround(a.center[1]),0],d=a.unprojectPosition(e),e[0]-=c[0],e[1]-=c[1],e[2]-=c[2]);break;case c9.IDENTITY:(e=a.position.map(Math.fround))[2]=e[2]||0;break;case c9.GLOBE:f=!1,d=null;break;default:f=!1}return{geospatialOrigin:d,shaderCoordinateOrigin:e,offsetMode:f}}let dn=["default","lnglat","meter-offsets","lnglat-offsets","cartesian"].map(a=>`const COORDINATE_SYSTEM_${a.toUpperCase().replaceAll("-","_")}: i32 = ${dk(a)};`).join(""),dp=Object.keys(c9).map(a=>`const PROJECTION_MODE_${a}: i32 = ${c9[a]};`).join(""),dq=Object.keys(da).map(a=>`const UNIT_${a.toUpperCase()}: i32 = ${da[a]};`).join(""),dr=`\
${dn}
${dp}
${dq}

const TILE_SIZE: f32 = 512.0;
const PI: f32 = 3.1415926536;
const WORLD_SCALE: f32 = TILE_SIZE / (PI * 2.0);
const ZERO_64_LOW: vec3<f32> = vec3<f32>(0.0, 0.0, 0.0);
const EARTH_RADIUS: f32 = 6370972.0; // meters
const GLOBE_RADIUS: f32 = 256.0;

// -----------------------------------------------------------------------------
// Uniform block (converted from GLSL uniform block)
// -----------------------------------------------------------------------------
struct ProjectUniforms {
  wrapLongitude: i32,
  coordinateSystem: i32,
  commonUnitsPerMeter: vec3<f32>,
  projectionMode: i32,
  scale: f32,
  commonUnitsPerWorldUnit: vec3<f32>,
  commonUnitsPerWorldUnit2: vec3<f32>,
  center: vec4<f32>,
  modelMatrix: mat4x4<f32>,
  viewProjectionMatrix: mat4x4<f32>,
  viewportSize: vec2<f32>,
  devicePixelRatio: f32,
  focalDistance: f32,
  cameraPosition: vec3<f32>,
  coordinateOrigin: vec3<f32>,
  commonOrigin: vec3<f32>,
  pseudoMeters: i32,
};

@group(0) @binding(auto)
var<uniform> project: ProjectUniforms;

// -----------------------------------------------------------------------------
// Geometry data shared across the project helpers.
// The active layer shader is responsible for populating this private module
// state before calling the project functions below.
// -----------------------------------------------------------------------------

// Structure to carry additional geometry data used by deck.gl filters.
struct Geometry {
  worldPosition: vec3<f32>,
  worldPositionAlt: vec3<f32>,
  position: vec4<f32>,
  normal: vec3<f32>,
  uv: vec2<f32>,
  pickingColor: vec3<f32>,
};

var<private> geometry: Geometry;
`,ds=`\
${dr}

// -----------------------------------------------------------------------------
// Functions
// -----------------------------------------------------------------------------

// Returns an adjustment factor for commonUnitsPerMeter
fn _project_size_at_latitude(lat: f32) -> f32 {
  let y = clamp(lat, -89.9, 89.9);
  return 1.0 / cos(radians(y));
}

// Overloaded version: scales a value in meters at a given latitude.
fn _project_size_at_latitude_m(meters: f32, lat: f32) -> f32 {
  return meters * project.commonUnitsPerMeter.z * _project_size_at_latitude(lat);
}

// Computes a non-linear scale factor based on geometry.
// (Note: This function relies on "geometry" being provided.)
fn project_size() -> f32 {
  if (project.projectionMode == PROJECTION_MODE_WEB_MERCATOR &&
      project.coordinateSystem == COORDINATE_SYSTEM_LNGLAT &&
      project.pseudoMeters == 0) {
    if (geometry.position.w == 0.0) {
      return _project_size_at_latitude(geometry.worldPosition.y);
    }
    let y: f32 = geometry.position.y / TILE_SIZE * 2.0 - 1.0;
    let y2 = y * y;
    let y4 = y2 * y2;
    let y6 = y4 * y2;
    return 1.0 + 4.9348 * y2 + 4.0587 * y4 + 1.5642 * y6;
  }
  return 1.0;
}

// Overloads to scale offsets (meters to world units)
fn project_size_float(meters: f32) -> f32 {
  return meters * project.commonUnitsPerMeter.z * project_size();
}

fn project_size_vec2(meters: vec2<f32>) -> vec2<f32> {
  return meters * project.commonUnitsPerMeter.xy * project_size();
}

fn project_size_vec3(meters: vec3<f32>) -> vec3<f32> {
  return meters * project.commonUnitsPerMeter * project_size();
}

fn project_size_vec4(meters: vec4<f32>) -> vec4<f32> {
  return vec4<f32>(meters.xyz * project.commonUnitsPerMeter, meters.w);
}

// Returns a rotation matrix aligning the z‑axis with the given up vector.
fn project_get_orientation_matrix(up: vec3<f32>) -> mat3x3<f32> {
  let uz = normalize(up);
  let ux = select(
    vec3<f32>(1.0, 0.0, 0.0),
    normalize(vec3<f32>(uz.y, -uz.x, 0.0)),
    abs(uz.z) == 1.0
  );
  let uy = cross(uz, ux);
  return mat3x3<f32>(ux, uy, uz);
}

// Since WGSL does not support "out" parameters, we return a struct.
struct RotationResult {
  needsRotation: bool,
  transform: mat3x3<f32>,
};

fn project_needs_rotation(commonPosition: vec3<f32>) -> RotationResult {
  if (project.projectionMode == PROJECTION_MODE_GLOBE) {
    return RotationResult(true, project_get_orientation_matrix(commonPosition));
  } else {
    return RotationResult(false, mat3x3<f32>());  // identity alternative if needed
  };
}

// Projects a normal vector from the current coordinate system to world space.
fn project_normal(vector: vec3<f32>) -> vec3<f32> {
  let normal_modelspace = project.modelMatrix * vec4<f32>(vector, 0.0);
  var n = normalize(normal_modelspace.xyz * project.commonUnitsPerMeter);
  let rotResult = project_needs_rotation(geometry.position.xyz);
  if (rotResult.needsRotation) {
    n = rotResult.transform * n;
  }
  return n;
}

// Applies a scale offset based on y-offset (dy)
fn project_offset_(offset: vec4<f32>) -> vec4<f32> {
  let dy: f32 = offset.y;
  let commonUnitsPerWorldUnit = project.commonUnitsPerWorldUnit + project.commonUnitsPerWorldUnit2 * dy;
  return vec4<f32>(offset.xyz * commonUnitsPerWorldUnit, offset.w);
}

// Projects lng/lat coordinates to a unit tile [0,1]
fn project_mercator_(lnglat: vec2<f32>) -> vec2<f32> {
  var x = lnglat.x;
  if (project.wrapLongitude != 0) {
    x = ((x + 180.0) % 360.0) - 180.0;
  }
  let y = clamp(lnglat.y, -89.9, 89.9);
  return vec2<f32>(
    radians(x) + PI,
    PI + log(tan(PI * 0.25 + radians(y) * 0.5))
  ) * WORLD_SCALE;
}

// Projects lng/lat/z coordinates for a globe projection.
fn project_globe_(lnglatz: vec3<f32>) -> vec3<f32> {
  let lambda = radians(lnglatz.x);
  let phi = radians(lnglatz.y);
  let cosPhi = cos(phi);
  let D = (lnglatz.z / EARTH_RADIUS + 1.0) * GLOBE_RADIUS;
  return vec3<f32>(
    sin(lambda) * cosPhi,
    -cos(lambda) * cosPhi,
    sin(phi)
  ) * D;
}

// Projects positions (with an optional 64-bit low part) from the input
// coordinate system to the common space.
fn project_position_vec4_f64(position: vec4<f32>, position64Low: vec3<f32>) -> vec4<f32> {
  var position_world = project.modelMatrix * position;

  // Work around for a Mac+NVIDIA bug:
  if (project.projectionMode == PROJECTION_MODE_WEB_MERCATOR) {
    if (project.coordinateSystem == COORDINATE_SYSTEM_LNGLAT) {
      return vec4<f32>(
        project_mercator_(position_world.xy),
        _project_size_at_latitude_m(position_world.z, position_world.y),
        position_world.w
      );
    }
    if (project.coordinateSystem == COORDINATE_SYSTEM_CARTESIAN) {
      position_world = vec4f(position_world.xyz + project.coordinateOrigin, position_world.w);
    }
  }
  if (project.projectionMode == PROJECTION_MODE_GLOBE) {
    if (project.coordinateSystem == COORDINATE_SYSTEM_LNGLAT) {
      return vec4<f32>(
        project_globe_(position_world.xyz),
        position_world.w
      );
    }
  }
  if (project.projectionMode == PROJECTION_MODE_WEB_MERCATOR_AUTO_OFFSET) {
    if (project.coordinateSystem == COORDINATE_SYSTEM_LNGLAT) {
      if (abs(position_world.y - project.coordinateOrigin.y) > 0.25) {
        return vec4<f32>(
          project_mercator_(position_world.xy) - project.commonOrigin.xy,
          project_size_float(position_world.z),
          position_world.w
        );
      }
    }
  }
  if (project.projectionMode == PROJECTION_MODE_IDENTITY ||
      (project.projectionMode == PROJECTION_MODE_WEB_MERCATOR_AUTO_OFFSET &&
       (project.coordinateSystem == COORDINATE_SYSTEM_LNGLAT ||
        project.coordinateSystem == COORDINATE_SYSTEM_CARTESIAN))) {
    position_world = vec4f(position_world.xyz - project.coordinateOrigin, position_world.w);
  }

  return project_offset_(position_world) +
         project_offset_(project.modelMatrix * vec4<f32>(position64Low, 0.0));
}

// Overloaded versions for different input types.
fn project_position_vec4_f32(position: vec4<f32>) -> vec4<f32> {
  return project_position_vec4_f64(position, ZERO_64_LOW);
}

fn project_position_vec3_f64(position: vec3<f32>, position64Low: vec3<f32>) -> vec3<f32> {
  let projected_position = project_position_vec4_f64(vec4<f32>(position, 1.0), position64Low);
  return projected_position.xyz;
}

fn project_position_vec3_f32(position: vec3<f32>) -> vec3<f32> {
  let projected_position = project_position_vec4_f64(vec4<f32>(position, 1.0), ZERO_64_LOW);
  return projected_position.xyz;
}

fn project_position_vec2_f32(position: vec2<f32>) -> vec2<f32> {
  let projected_position = project_position_vec4_f64(vec4<f32>(position, 0.0, 1.0), ZERO_64_LOW);
  return projected_position.xy;
}

// Transforms a common space position to clip space.
fn project_common_position_to_clipspace_with_projection(position: vec4<f32>, viewProjectionMatrix: mat4x4<f32>, center: vec4<f32>) -> vec4<f32> {
  return viewProjectionMatrix * position + center;
}

// Uses the project viewProjectionMatrix and center.
fn project_common_position_to_clipspace(position: vec4<f32>) -> vec4<f32> {
  return project_common_position_to_clipspace_with_projection(position, project.viewProjectionMatrix, project.center);
}

// Returns a clip space offset corresponding to a given number of screen pixels.
fn project_pixel_size_to_clipspace(pixels: vec2<f32>) -> vec2<f32> {
  let offset = pixels / project.viewportSize * project.devicePixelRatio * 2.0;
  return offset * project.focalDistance;
}

fn project_meter_size_to_pixel(meters: f32) -> f32 {
  return project_size_float(meters) * project.scale;
}

fn project_unit_size_to_pixel(size: f32, unit: i32) -> f32 {
  if (unit == UNIT_METERS) {
    return project_meter_size_to_pixel(size);
  } else if (unit == UNIT_COMMON) {
    return size * project.scale;
  }
  // UNIT_PIXELS: no scaling applied.
  return size;
}

fn project_pixel_size_float(pixels: f32) -> f32 {
  return pixels / project.scale;
}

fn project_pixel_size_vec2(pixels: vec2<f32>) -> vec2<f32> {
  return pixels / project.scale;
}
`,dt=["default","lnglat","meter-offsets","lnglat-offsets","cartesian"].map(a=>`const int COORDINATE_SYSTEM_${a.toUpperCase().replaceAll("-","_")} = ${dk(a)};`).join(""),du=Object.keys(c9).map(a=>`const int PROJECTION_MODE_${a} = ${c9[a]};`).join(""),dv=Object.keys(da).map(a=>`const int UNIT_${a.toUpperCase()} = ${da[a]};`).join(""),dw={},dx={name:"project",dependencies:[{name:"fp32",vs:bU},bT],source:ds,vs:`\
${dt}
${du}
${dv}
layout(std140) uniform projectUniforms {
bool wrapLongitude;
int coordinateSystem;
vec3 commonUnitsPerMeter;
int projectionMode;
float scale;
vec3 commonUnitsPerWorldUnit;
vec3 commonUnitsPerWorldUnit2;
vec4 center;
mat4 modelMatrix;
mat4 viewProjectionMatrix;
vec2 viewportSize;
float devicePixelRatio;
float focalDistance;
vec3 cameraPosition;
vec3 coordinateOrigin;
vec3 commonOrigin;
bool pseudoMeters;
} project;
const float TILE_SIZE = 512.0;
const float PI = 3.1415926536;
const float WORLD_SCALE = TILE_SIZE / (PI * 2.0);
const vec3 ZERO_64_LOW = vec3(0.0);
const float EARTH_RADIUS = 6370972.0;
const float GLOBE_RADIUS = 256.0;
float project_size_at_latitude(float lat) {
float y = clamp(lat, -89.9, 89.9);
return 1.0 / cos(radians(y));
}
float project_size() {
if (project.projectionMode == PROJECTION_MODE_WEB_MERCATOR &&
project.coordinateSystem == COORDINATE_SYSTEM_LNGLAT &&
project.pseudoMeters == false) {
if (geometry.position.w == 0.0) {
return project_size_at_latitude(geometry.worldPosition.y);
}
float y = geometry.position.y / TILE_SIZE * 2.0 - 1.0;
float y2 = y * y;
float y4 = y2 * y2;
float y6 = y4 * y2;
return 1.0 + 4.9348 * y2 + 4.0587 * y4 + 1.5642 * y6;
}
return 1.0;
}
float project_size_at_latitude(float meters, float lat) {
return meters * project.commonUnitsPerMeter.z * project_size_at_latitude(lat);
}
float project_size(float meters) {
return meters * project.commonUnitsPerMeter.z * project_size();
}
vec2 project_size(vec2 meters) {
return meters * project.commonUnitsPerMeter.xy * project_size();
}
vec3 project_size(vec3 meters) {
return meters * project.commonUnitsPerMeter * project_size();
}
vec4 project_size(vec4 meters) {
return vec4(meters.xyz * project.commonUnitsPerMeter, meters.w);
}
mat3 project_get_orientation_matrix(vec3 up) {
vec3 uz = normalize(up);
vec3 ux = abs(uz.z) == 1.0 ? vec3(1.0, 0.0, 0.0) : normalize(vec3(uz.y, -uz.x, 0));
vec3 uy = cross(uz, ux);
return mat3(ux, uy, uz);
}
bool project_needs_rotation(vec3 commonPosition, out mat3 transform) {
if (project.projectionMode == PROJECTION_MODE_GLOBE) {
transform = project_get_orientation_matrix(commonPosition);
return true;
}
return false;
}
vec3 project_normal(vec3 vector) {
vec4 normal_modelspace = project.modelMatrix * vec4(vector, 0.0);
vec3 n = normalize(normal_modelspace.xyz * project.commonUnitsPerMeter);
mat3 rotation;
if (project_needs_rotation(geometry.position.xyz, rotation)) {
n = rotation * n;
}
return n;
}
vec4 project_offset_(vec4 offset) {
float dy = offset.y;
vec3 commonUnitsPerWorldUnit = project.commonUnitsPerWorldUnit + project.commonUnitsPerWorldUnit2 * dy;
return vec4(offset.xyz * commonUnitsPerWorldUnit, offset.w);
}
vec2 project_mercator_(vec2 lnglat) {
float x = lnglat.x;
if (project.wrapLongitude) {
x = mod(x + 180., 360.0) - 180.;
}
float y = clamp(lnglat.y, -89.9, 89.9);
return vec2(
radians(x) + PI,
PI + log(tan_fp32(PI * 0.25 + radians(y) * 0.5))
) * WORLD_SCALE;
}
vec3 project_globe_(vec3 lnglatz) {
float lambda = radians(lnglatz.x);
float phi = radians(lnglatz.y);
float cosPhi = cos(phi);
float D = (lnglatz.z / EARTH_RADIUS + 1.0) * GLOBE_RADIUS;
return vec3(
sin(lambda) * cosPhi,
-cos(lambda) * cosPhi,
sin(phi)
) * D;
}
vec4 project_position(vec4 position, vec3 position64Low) {
vec4 position_world = project.modelMatrix * position;
if (project.projectionMode == PROJECTION_MODE_WEB_MERCATOR) {
if (project.coordinateSystem == COORDINATE_SYSTEM_LNGLAT) {
return vec4(
project_mercator_(position_world.xy),
project_size_at_latitude(position_world.z, position_world.y),
position_world.w
);
}
if (project.coordinateSystem == COORDINATE_SYSTEM_CARTESIAN) {
position_world.xyz += project.coordinateOrigin;
}
}
if (project.projectionMode == PROJECTION_MODE_GLOBE) {
if (project.coordinateSystem == COORDINATE_SYSTEM_LNGLAT) {
return vec4(
project_globe_(position_world.xyz),
position_world.w
);
}
}
if (project.projectionMode == PROJECTION_MODE_WEB_MERCATOR_AUTO_OFFSET) {
if (project.coordinateSystem == COORDINATE_SYSTEM_LNGLAT) {
if (abs(position_world.y - project.coordinateOrigin.y) > 0.25) {
return vec4(
project_mercator_(position_world.xy) - project.commonOrigin.xy,
project_size(position_world.z),
position_world.w
);
}
}
}
if (project.projectionMode == PROJECTION_MODE_IDENTITY ||
(project.projectionMode == PROJECTION_MODE_WEB_MERCATOR_AUTO_OFFSET &&
(project.coordinateSystem == COORDINATE_SYSTEM_LNGLAT ||
project.coordinateSystem == COORDINATE_SYSTEM_CARTESIAN))) {
position_world.xyz -= project.coordinateOrigin;
}
return project_offset_(position_world) + project_offset_(project.modelMatrix * vec4(position64Low, 0.0));
}
vec4 project_position(vec4 position) {
return project_position(position, ZERO_64_LOW);
}
vec3 project_position(vec3 position, vec3 position64Low) {
vec4 projected_position = project_position(vec4(position, 1.0), position64Low);
return projected_position.xyz;
}
vec3 project_position(vec3 position) {
vec4 projected_position = project_position(vec4(position, 1.0), ZERO_64_LOW);
return projected_position.xyz;
}
vec2 project_position(vec2 position) {
vec4 projected_position = project_position(vec4(position, 0.0, 1.0), ZERO_64_LOW);
return projected_position.xy;
}
vec4 project_common_position_to_clipspace(vec4 position, mat4 viewProjectionMatrix, vec4 center) {
return viewProjectionMatrix * position + center;
}
vec4 project_common_position_to_clipspace(vec4 position) {
return project_common_position_to_clipspace(position, project.viewProjectionMatrix, project.center);
}
vec2 project_pixel_size_to_clipspace(vec2 pixels) {
vec2 offset = pixels / project.viewportSize * project.devicePixelRatio * 2.0;
return offset * project.focalDistance;
}
float project_size_to_pixel(float meters) {
return project_size(meters) * project.scale;
}
vec2 project_size_to_pixel(vec2 meters) {
return project_size(meters) * project.scale;
}
float project_size_to_pixel(float size, int unit) {
if (unit == UNIT_METERS) return project_size_to_pixel(size);
if (unit == UNIT_COMMON) return size * project.scale;
return size;
}
float project_pixel_size(float pixels) {
return pixels / project.scale;
}
vec2 project_pixel_size(vec2 pixels) {
return pixels / project.scale;
}
`,getUniforms:function(a=dw){return"viewport"in a?function({viewport:a,devicePixelRatio:b=1,modelMatrix:c=null,coordinateSystem:d="default",coordinateOrigin:e=di,autoWrapLongitude:f=!1}){"default"===d&&(d=a.isGeospatial?"lnglat":"cartesian");let g=dl({viewport:a,devicePixelRatio:b,coordinateSystem:d,coordinateOrigin:e});return g.wrapLongitude=f,g.modelMatrix=c||dg,g}(a):{}},uniformTypes:{wrapLongitude:"f32",coordinateSystem:"i32",commonUnitsPerMeter:"vec3<f32>",projectionMode:"i32",scale:"f32",commonUnitsPerWorldUnit:"vec3<f32>",commonUnitsPerWorldUnit2:"vec3<f32>",center:"vec4<f32>",modelMatrix:"mat4x4<f32>",viewProjectionMatrix:"mat4x4<f32>",viewportSize:"vec2<f32>",devicePixelRatio:"f32",focalDistance:"f32",cameraPosition:"vec3<f32>",coordinateOrigin:"vec3<f32>",commonOrigin:"vec3<f32>",pseudoMeters:"f32"}},dy={name:"project32",dependencies:[dx],source:`\
// Define a structure to hold both the clip-space position and the common position.
struct ProjectResult {
  clipPosition: vec4<f32>,
  commonPosition: vec4<f32>,
};

// This function mimics the GLSL version with the 'out' parameter by returning both values.
fn project_position_to_clipspace_and_commonspace(
    position: vec3<f32>,
    position64Low: vec3<f32>,
    offset: vec3<f32>
) -> ProjectResult {
  // Compute the projected position.
  let projectedPosition: vec3<f32> = project_position_vec3_f64(position, position64Low);

  // Start with the provided offset.
  var finalOffset: vec3<f32> = offset;

  // Get whether a rotation is needed and the rotation matrix.
  let rotationResult = project_needs_rotation(projectedPosition);

  // If rotation is needed, update the offset.
  if (rotationResult.needsRotation) {
    finalOffset = rotationResult.transform * offset;
  }

  // Compute the common position.
  let commonPosition: vec4<f32> = vec4<f32>(projectedPosition + finalOffset, 1.0);

  // Convert to clip-space.
  let clipPosition: vec4<f32> = project_common_position_to_clipspace(commonPosition);

  return ProjectResult(clipPosition, commonPosition);
}

// A convenience overload that returns only the clip-space position.
fn project_position_to_clipspace(
    position: vec3<f32>,
    position64Low: vec3<f32>,
    offset: vec3<f32>
) -> vec4<f32> {
  return project_position_to_clipspace_and_commonspace(position, position64Low, offset).clipPosition;
}
`,vs:`\
vec4 project_position_to_clipspace(
  vec3 position, vec3 position64Low, vec3 offset, out vec4 commonPosition
) {
  vec3 projectedPosition = project_position(position, position64Low);
  mat3 rotation;
  if (project_needs_rotation(projectedPosition, rotation)) {
    // offset is specified as ENU
    // when in globe projection, rotate offset so that the ground alighs with the surface of the globe
    offset = rotation * offset;
  }
  commonPosition = vec4(projectedPosition + offset, 1.0);
  return project_common_position_to_clipspace(commonPosition);
}

vec4 project_position_to_clipspace(
  vec3 position, vec3 position64Low, vec3 offset
) {
  vec4 commonPosition;
  return project_position_to_clipspace(position, position64Low, offset, commonPosition);
}
`};globalThis.mathgl=globalThis.mathgl||{config:{EPSILON:1e-12,debug:!1,precision:4,printTypes:!1,printDegrees:!1,printRowMajor:!0,_cartographicRadians:!1}};let dz=globalThis.mathgl.config;function dA(a){return Array.isArray(a)||ArrayBuffer.isView(a)&&!(a instanceof DataView)}function dB(a,b,c){return function(a,b,c){if(dA(a)){c=c||(a.clone?a.clone():Array(a.length));for(let d=0;d<c.length&&d<a.length;++d){let e="number"==typeof a?a:a[d];c[d]=b(e,d,c)}return c}return b(a)}(a,a=>Math.max(b,Math.min(c,a)))}function dC(a,b,c){return dA(a)?a.map((a,d)=>dC(a,b[d],c)):c*b+(1-c)*a}function dD(a,b,c){let d=dz.EPSILON;c&&(dz.EPSILON=c);try{if(a===b)return!0;if(dA(a)&&dA(b)){if(a.length!==b.length)return!1;for(let c=0;c<a.length;++c)if(!dD(a[c],b[c]))return!1;return!0}if(a&&a.equals)return a.equals(b);if(b&&b.equals)return b.equals(a);if("number"==typeof a&&"number"==typeof b)return Math.abs(a-b)<=dz.EPSILON*Math.max(1,Math.abs(a),Math.abs(b));return!1}finally{dz.EPSILON=d}}class dE extends Array{clone(){return new this.constructor().copy(this)}fromArray(a,b=0){for(let c=0;c<this.ELEMENTS;++c)this[c]=a[c+b];return this.check()}toArray(a=[],b=0){for(let c=0;c<this.ELEMENTS;++c)a[b+c]=this[c];return a}toObject(a){return a}from(a){return Array.isArray(a)?this.copy(a):this.fromObject(a)}to(a){return a===this?this:dA(a)?this.toArray(a):this.toObject(a)}toTarget(a){return a?this.to(a):this}toFloat32Array(){return new Float32Array(this)}toString(){return this.formatString(dz)}formatString(a){let b="";for(let c=0;c<this.ELEMENTS;++c)b+=(c>0?", ":"")+function(a,{precision:b=dz.precision}={}){return a=Math.round(a/dz.EPSILON)*dz.EPSILON,`${parseFloat(a.toPrecision(b))}`}(this[c],a);return`${a.printTypes?this.constructor.name:""}[${b}]`}equals(a){if(!a||this.length!==a.length)return!1;for(let b=0;b<this.ELEMENTS;++b)if(!dD(this[b],a[b]))return!1;return!0}exactEquals(a){if(!a||this.length!==a.length)return!1;for(let b=0;b<this.ELEMENTS;++b)if(this[b]!==a[b])return!1;return!0}negate(){for(let a=0;a<this.ELEMENTS;++a)this[a]=-this[a];return this.check()}lerp(a,b,c){if(void 0===c)return this.lerp(this,a,b);for(let d=0;d<this.ELEMENTS;++d){let e=a[d],f="number"==typeof b?b:b[d];this[d]=e+c*(f-e)}return this.check()}min(a){for(let b=0;b<this.ELEMENTS;++b)this[b]=Math.min(a[b],this[b]);return this.check()}max(a){for(let b=0;b<this.ELEMENTS;++b)this[b]=Math.max(a[b],this[b]);return this.check()}clamp(a,b){for(let c=0;c<this.ELEMENTS;++c)this[c]=Math.min(Math.max(this[c],a[c]),b[c]);return this.check()}add(...a){for(let b of a)for(let a=0;a<this.ELEMENTS;++a)this[a]+=b[a];return this.check()}subtract(...a){for(let b of a)for(let a=0;a<this.ELEMENTS;++a)this[a]-=b[a];return this.check()}scale(a){if("number"==typeof a)for(let b=0;b<this.ELEMENTS;++b)this[b]*=a;else for(let b=0;b<this.ELEMENTS&&b<a.length;++b)this[b]*=a[b];return this.check()}multiplyByScalar(a){for(let b=0;b<this.ELEMENTS;++b)this[b]*=a;return this.check()}check(){if(dz.debug&&!this.validate())throw Error(`math.gl: ${this.constructor.name} some fields set to invalid numbers'`);return this}validate(){let a=this.length===this.ELEMENTS;for(let b=0;b<this.ELEMENTS;++b)a=a&&Number.isFinite(this[b]);return a}sub(a){return this.subtract(a)}setScalar(a){for(let b=0;b<this.ELEMENTS;++b)this[b]=a;return this.check()}addScalar(a){for(let b=0;b<this.ELEMENTS;++b)this[b]+=a;return this.check()}subScalar(a){return this.addScalar(-a)}multiplyScalar(a){for(let b=0;b<this.ELEMENTS;++b)this[b]*=a;return this.check()}divideScalar(a){return this.multiplyByScalar(1/a)}clampScalar(a,b){for(let c=0;c<this.ELEMENTS;++c)this[c]=Math.min(Math.max(this[c],a),b);return this.check()}get elements(){return this}}function dF(a){if(!Number.isFinite(a))throw Error(`Invalid number ${JSON.stringify(a)}`);return a}function dG(a,b,c=""){if(dz.debug&&!function(a,b){if(a.length!==b)return!1;for(let b=0;b<a.length;++b)if(!Number.isFinite(a[b]))return!1;return!0}(a,b))throw Error(`math.gl: ${c} some fields set to invalid numbers'`);return a}function dH(a,b){if(!a)throw Error(`math.gl assertion ${b}`)}class dI extends dE{get x(){return this[0]}set x(a){this[0]=dF(a)}get y(){return this[1]}set y(a){this[1]=dF(a)}len(){return Math.sqrt(this.lengthSquared())}magnitude(){return this.len()}lengthSquared(){let a=0;for(let b=0;b<this.ELEMENTS;++b)a+=this[b]*this[b];return a}magnitudeSquared(){return this.lengthSquared()}distance(a){return Math.sqrt(this.distanceSquared(a))}distanceSquared(a){let b=0;for(let c=0;c<this.ELEMENTS;++c){let d=this[c]-a[c];b+=d*d}return dF(b)}dot(a){let b=0;for(let c=0;c<this.ELEMENTS;++c)b+=this[c]*a[c];return dF(b)}normalize(){let a=this.magnitude();if(0!==a)for(let b=0;b<this.ELEMENTS;++b)this[b]/=a;return this.check()}multiply(...a){for(let b of a)for(let a=0;a<this.ELEMENTS;++a)this[a]*=b[a];return this.check()}divide(...a){for(let b of a)for(let a=0;a<this.ELEMENTS;++a)this[a]/=b[a];return this.check()}lengthSq(){return this.lengthSquared()}distanceTo(a){return this.distance(a)}distanceToSquared(a){return this.distanceSquared(a)}getComponent(a){return dH(a>=0&&a<this.ELEMENTS,"index is out of range"),dF(this[a])}setComponent(a,b){return dH(a>=0&&a<this.ELEMENTS,"index is out of range"),this[a]=b,this.check()}addVectors(a,b){return this.copy(a).add(b)}subVectors(a,b){return this.copy(a).subtract(b)}multiplyVectors(a,b){return this.copy(a).multiply(b)}addScaledVector(a,b){return this.add(new this.constructor(a).multiplyScalar(b))}}function dJ(){let a=new bV(3);return bV!=Float32Array&&(a[0]=0,a[1]=0,a[2]=0),a}function dK(a){let b=a[0],c=a[1],d=a[2];return Math.sqrt(b*b+c*c+d*d)}function dL(a,b,c){return a[0]=b[0]-c[0],a[1]=b[1]-c[1],a[2]=b[2]-c[2],a}function dM(a,b,c){return a[0]=b[0]*c[0],a[1]=b[1]*c[1],a[2]=b[2]*c[2],a}function dN(a,b,c){return a[0]=b[0]/c[0],a[1]=b[1]/c[1],a[2]=b[2]/c[2],a}function dO(a,b){let c=b[0]-a[0],d=b[1]-a[1],e=b[2]-a[2];return Math.sqrt(c*c+d*d+e*e)}function dP(a,b){let c=b[0]-a[0],d=b[1]-a[1],e=b[2]-a[2];return c*c+d*d+e*e}function dQ(a){let b=a[0],c=a[1],d=a[2];return b*b+c*c+d*d}function dR(a,b){return a[0]*b[0]+a[1]*b[1]+a[2]*b[2]}function dS(a,b,c){let d=b[0],e=b[1],f=b[2],g=c[0],h=c[1],i=c[2];return a[0]=e*i-f*h,a[1]=f*g-d*i,a[2]=d*h-e*g,a}function dT(a,b,c){let d=b[0],e=b[1],f=b[2],g=c[3]*d+c[7]*e+c[11]*f+c[15];return g=g||1,a[0]=(c[0]*d+c[4]*e+c[8]*f+c[12])/g,a[1]=(c[1]*d+c[5]*e+c[9]*f+c[13])/g,a[2]=(c[2]*d+c[6]*e+c[10]*f+c[14])/g,a}function dU(a,b,c){let d=b[0],e=b[1],f=b[2];return a[0]=d*c[0]+e*c[3]+f*c[6],a[1]=d*c[1]+e*c[4]+f*c[7],a[2]=d*c[2]+e*c[5]+f*c[8],a}function dV(a,b,c){let d=c[0],e=c[1],f=c[2],g=c[3],h=b[0],i=b[1],j=b[2],k=e*j-f*i,l=f*h-d*j,m=d*i-e*h,n=e*m-f*l,o=f*k-d*m,p=d*l-e*k,q=2*g;return k*=q,l*=q,m*=q,n*=2,o*=2,p*=2,a[0]=h+k+n,a[1]=i+l+o,a[2]=j+m+p,a}function dW(a,b,c,d){let e=[],f=[];return e[0]=b[0]-c[0],e[1]=b[1]-c[1],e[2]=b[2]-c[2],f[0]=e[0],f[1]=e[1]*Math.cos(d)-e[2]*Math.sin(d),f[2]=e[1]*Math.sin(d)+e[2]*Math.cos(d),a[0]=f[0]+c[0],a[1]=f[1]+c[1],a[2]=f[2]+c[2],a}function dX(a,b,c,d){let e=[],f=[];return e[0]=b[0]-c[0],e[1]=b[1]-c[1],e[2]=b[2]-c[2],f[0]=e[2]*Math.sin(d)+e[0]*Math.cos(d),f[1]=e[1],f[2]=e[2]*Math.cos(d)-e[0]*Math.sin(d),a[0]=f[0]+c[0],a[1]=f[1]+c[1],a[2]=f[2]+c[2],a}function dY(a,b,c,d){let e=[],f=[];return e[0]=b[0]-c[0],e[1]=b[1]-c[1],e[2]=b[2]-c[2],f[0]=e[0]*Math.cos(d)-e[1]*Math.sin(d),f[1]=e[0]*Math.sin(d)+e[1]*Math.cos(d),f[2]=e[2],a[0]=f[0]+c[0],a[1]=f[1]+c[1],a[2]=f[2]+c[2],a}function dZ(a,b){let c=a[0],d=a[1],e=a[2],f=b[0],g=b[1],h=b[2],i=Math.sqrt((c*c+d*d+e*e)*(f*f+g*g+h*h));return Math.acos(Math.min(Math.max(i&&dR(a,b)/i,-1),1))}let d$=(g=dJ(),function(a,b,c,d,e,f){let h,i;for(b||(b=3),c||(c=0),i=d?Math.min(d*b+c,a.length):a.length,h=c;h<i;h+=b)g[0]=a[h],g[1]=a[h+1],g[2]=a[h+2],e(g,g,f),a[h]=g[0],a[h+1]=g[1],a[h+2]=g[2];return a});function d_(a,b,c){let d=b[0],e=b[1],f=b[2],g=c[3]*d+c[7]*e+c[11]*f||1;return a[0]=(c[0]*d+c[4]*e+c[8]*f)/g,a[1]=(c[1]*d+c[5]*e+c[9]*f)/g,a[2]=(c[2]*d+c[6]*e+c[10]*f)/g,a}a.s(["add",0,function(a,b,c){return a[0]=b[0]+c[0],a[1]=b[1]+c[1],a[2]=b[2]+c[2],a},"angle",0,dZ,"bezier",0,function(a,b,c,d,e,f){let g=1-f,h=g*g,i=f*f,j=h*g,k=3*f*h,l=3*i*g,m=i*f;return a[0]=b[0]*j+c[0]*k+d[0]*l+e[0]*m,a[1]=b[1]*j+c[1]*k+d[1]*l+e[1]*m,a[2]=b[2]*j+c[2]*k+d[2]*l+e[2]*m,a},"ceil",0,function(a,b){return a[0]=Math.ceil(b[0]),a[1]=Math.ceil(b[1]),a[2]=Math.ceil(b[2]),a},"clone",0,function(a){let b=new bV(3);return b[0]=a[0],b[1]=a[1],b[2]=a[2],b},"copy",0,function(a,b){return a[0]=b[0],a[1]=b[1],a[2]=b[2],a},"create",0,dJ,"cross",0,dS,"dist",0,dO,"distance",0,dO,"div",0,dN,"divide",0,dN,"dot",0,dR,"equals",0,function(a,b){let c=a[0],d=a[1],e=a[2],f=b[0],g=b[1],h=b[2];return Math.abs(c-f)<=1e-6*Math.max(1,Math.abs(c),Math.abs(f))&&Math.abs(d-g)<=1e-6*Math.max(1,Math.abs(d),Math.abs(g))&&Math.abs(e-h)<=1e-6*Math.max(1,Math.abs(e),Math.abs(h))},"exactEquals",0,function(a,b){return a[0]===b[0]&&a[1]===b[1]&&a[2]===b[2]},"floor",0,function(a,b){return a[0]=Math.floor(b[0]),a[1]=Math.floor(b[1]),a[2]=Math.floor(b[2]),a},"forEach",0,d$,"fromValues",0,function(a,b,c){let d=new bV(3);return d[0]=a,d[1]=b,d[2]=c,d},"hermite",0,function(a,b,c,d,e,f){let g=f*f,h=g*(2*f-3)+1,i=g*(f-2)+f,j=g*(f-1),k=g*(3-2*f);return a[0]=b[0]*h+c[0]*i+d[0]*j+e[0]*k,a[1]=b[1]*h+c[1]*i+d[1]*j+e[1]*k,a[2]=b[2]*h+c[2]*i+d[2]*j+e[2]*k,a},"inverse",0,function(a,b){return a[0]=1/b[0],a[1]=1/b[1],a[2]=1/b[2],a},"len",0,dK,"length",0,dK,"lerp",0,function(a,b,c,d){let e=b[0],f=b[1],g=b[2];return a[0]=e+d*(c[0]-e),a[1]=f+d*(c[1]-f),a[2]=g+d*(c[2]-g),a},"max",0,function(a,b,c){return a[0]=Math.max(b[0],c[0]),a[1]=Math.max(b[1],c[1]),a[2]=Math.max(b[2],c[2]),a},"min",0,function(a,b,c){return a[0]=Math.min(b[0],c[0]),a[1]=Math.min(b[1],c[1]),a[2]=Math.min(b[2],c[2]),a},"mul",0,dM,"multiply",0,dM,"negate",0,function(a,b){return a[0]=-b[0],a[1]=-b[1],a[2]=-b[2],a},"normalize",0,function(a,b){let c=b[0],d=b[1],e=b[2],f=c*c+d*d+e*e;return f>0&&(f=1/Math.sqrt(f)),a[0]=b[0]*f,a[1]=b[1]*f,a[2]=b[2]*f,a},"random",0,function(a,b){b=void 0===b?1:b;let c=2*bW()*Math.PI,d=2*bW()-1,e=Math.sqrt(1-d*d)*b;return a[0]=Math.cos(c)*e,a[1]=Math.sin(c)*e,a[2]=d*b,a},"rotateX",0,dW,"rotateY",0,dX,"rotateZ",0,dY,"round",0,function(a,b){return a[0]=bX(b[0]),a[1]=bX(b[1]),a[2]=bX(b[2]),a},"scale",0,function(a,b,c){return a[0]=b[0]*c,a[1]=b[1]*c,a[2]=b[2]*c,a},"scaleAndAdd",0,function(a,b,c,d){return a[0]=b[0]+c[0]*d,a[1]=b[1]+c[1]*d,a[2]=b[2]+c[2]*d,a},"set",0,function(a,b,c,d){return a[0]=b,a[1]=c,a[2]=d,a},"slerp",0,function(a,b,c,d){let e=Math.acos(Math.min(Math.max(dR(b,c),-1),1)),f=Math.sin(e),g=Math.sin((1-d)*e)/f,h=Math.sin(d*e)/f;return a[0]=g*b[0]+h*c[0],a[1]=g*b[1]+h*c[1],a[2]=g*b[2]+h*c[2],a},"sqrDist",0,dP,"sqrLen",0,dQ,"squaredDistance",0,dP,"squaredLength",0,dQ,"str",0,function(a){return`vec3(${a[0]}, ${a[1]}, ${a[2]})`},"sub",0,dL,"subtract",0,dL,"transformMat3",0,dU,"transformMat4",0,dT,"transformQuat",0,dV,"zero",0,function(a){return a[0]=0,a[1]=0,a[2]=0,a}],33649);let d0=[0,0,0];class d1 extends dI{static get ZERO(){return b||Object.freeze(b=new d1(0,0,0)),b}constructor(a=0,b=0,c=0){super(-0,-0,-0),1==arguments.length&&dA(a)?this.copy(a):(dz.debug&&(dF(a),dF(b),dF(c)),this[0]=a,this[1]=b,this[2]=c)}set(a,b,c){return this[0]=a,this[1]=b,this[2]=c,this.check()}copy(a){return this[0]=a[0],this[1]=a[1],this[2]=a[2],this.check()}fromObject(a){return dz.debug&&(dF(a.x),dF(a.y),dF(a.z)),this[0]=a.x,this[1]=a.y,this[2]=a.z,this.check()}toObject(a){return a.x=this[0],a.y=this[1],a.z=this[2],a}get ELEMENTS(){return 3}get z(){return this[2]}set z(a){this[2]=dF(a)}angle(a){return dZ(this,a)}cross(a){return dS(this,this,a),this.check()}rotateX({radians:a,origin:b=d0}){return dW(this,this,b,a),this.check()}rotateY({radians:a,origin:b=d0}){return dX(this,this,b,a),this.check()}rotateZ({radians:a,origin:b=d0}){return dY(this,this,b,a),this.check()}transform(a){return this.transformAsPoint(a)}transformAsPoint(a){return dT(this,this,a),this.check()}transformAsVector(a){return d_(this,this,a),this.check()}transformByMatrix3(a){return dU(this,this,a),this.check()}transformByMatrix2(a){let b,c;return b=this[0],c=this[1],this[0]=a[0]*b+a[2]*c,this[1]=a[1]*b+a[3]*c,this[2]=this[2],this.check()}transformByQuaternion(a){return dV(this,this,a),this.check()}}class d2 extends dE{toString(){let a="[";if(dz.printRowMajor){a+="row-major:";for(let b=0;b<this.RANK;++b)for(let c=0;c<this.RANK;++c)a+=` ${this[c*this.RANK+b]}`}else{a+="column-major:";for(let b=0;b<this.ELEMENTS;++b)a+=` ${this[b]}`}return a+"]"}getElementIndex(a,b){return b*this.RANK+a}getElement(a,b){return this[b*this.RANK+a]}setElement(a,b,c){return this[b*this.RANK+a]=dF(c),this}getColumn(a,b=Array(this.RANK).fill(-0)){let c=a*this.RANK;for(let a=0;a<this.RANK;++a)b[a]=this[c+a];return b}setColumn(a,b){let c=a*this.RANK;for(let a=0;a<this.RANK;++a)this[c+a]=b[a];return this}}function d3(){let a=new bV(2);return bV!=Float32Array&&(a[0]=0,a[1]=0),a}function d4(a,b,c){return a[0]=b[0]-c[0],a[1]=b[1]-c[1],a}function d5(a,b,c){return a[0]=b[0]*c[0],a[1]=b[1]*c[1],a}function d6(a,b,c){return a[0]=b[0]/c[0],a[1]=b[1]/c[1],a}function d7(a,b){let c=b[0]-a[0],d=b[1]-a[1];return Math.sqrt(c*c+d*d)}function d8(a,b){let c=b[0]-a[0],d=b[1]-a[1];return c*c+d*d}function d9(a){let b=a[0],c=a[1];return Math.sqrt(b*b+c*c)}function ea(a){let b=a[0],c=a[1];return b*b+c*c}function eb(a,b,c){let d=b[0],e=b[1];return a[0]=c[0]*d+c[4]*e+c[12],a[1]=c[1]*d+c[5]*e+c[13],a}let ec=(h=d3(),function(a,b,c,d,e,f){let g,i;for(b||(b=2),c||(c=0),i=d?Math.min(d*b+c,a.length):a.length,g=c;g<i;g+=b)h[0]=a[g],h[1]=a[g+1],e(h,h,f),a[g]=h[0],a[g+1]=h[1];return a});a.s(["add",0,function(a,b,c){return a[0]=b[0]+c[0],a[1]=b[1]+c[1],a},"angle",0,function(a,b){let c=a[0],d=a[1],e=b[0],f=b[1],g=Math.sqrt((c*c+d*d)*(e*e+f*f));return Math.acos(Math.min(Math.max(g&&(c*e+d*f)/g,-1),1))},"ceil",0,function(a,b){return a[0]=Math.ceil(b[0]),a[1]=Math.ceil(b[1]),a},"clone",0,function(a){let b=new bV(2);return b[0]=a[0],b[1]=a[1],b},"copy",0,function(a,b){return a[0]=b[0],a[1]=b[1],a},"create",0,d3,"cross",0,function(a,b,c){let d=b[0]*c[1]-b[1]*c[0];return a[0]=a[1]=0,a[2]=d,a},"dist",0,d7,"distance",0,d7,"div",0,d6,"divide",0,d6,"dot",0,function(a,b){return a[0]*b[0]+a[1]*b[1]},"equals",0,function(a,b){let c=a[0],d=a[1],e=b[0],f=b[1];return Math.abs(c-e)<=1e-6*Math.max(1,Math.abs(c),Math.abs(e))&&Math.abs(d-f)<=1e-6*Math.max(1,Math.abs(d),Math.abs(f))},"exactEquals",0,function(a,b){return a[0]===b[0]&&a[1]===b[1]},"floor",0,function(a,b){return a[0]=Math.floor(b[0]),a[1]=Math.floor(b[1]),a},"forEach",0,ec,"fromValues",0,function(a,b){let c=new bV(2);return c[0]=a,c[1]=b,c},"inverse",0,function(a,b){return a[0]=1/b[0],a[1]=1/b[1],a},"len",0,d9,"length",0,d9,"lerp",0,function(a,b,c,d){let e=b[0],f=b[1];return a[0]=e+d*(c[0]-e),a[1]=f+d*(c[1]-f),a},"max",0,function(a,b,c){return a[0]=Math.max(b[0],c[0]),a[1]=Math.max(b[1],c[1]),a},"min",0,function(a,b,c){return a[0]=Math.min(b[0],c[0]),a[1]=Math.min(b[1],c[1]),a},"mul",0,d5,"multiply",0,d5,"negate",0,function(a,b){return a[0]=-b[0],a[1]=-b[1],a},"normalize",0,function(a,b){let c=b[0],d=b[1],e=c*c+d*d;return e>0&&(e=1/Math.sqrt(e)),a[0]=b[0]*e,a[1]=b[1]*e,a},"random",0,function(a,b){b=void 0===b?1:b;let c=2*bW()*Math.PI;return a[0]=Math.cos(c)*b,a[1]=Math.sin(c)*b,a},"rotate",0,function(a,b,c,d){let e=b[0]-c[0],f=b[1]-c[1],g=Math.sin(d),h=Math.cos(d);return a[0]=e*h-f*g+c[0],a[1]=e*g+f*h+c[1],a},"round",0,function(a,b){return a[0]=bX(b[0]),a[1]=bX(b[1]),a},"scale",0,function(a,b,c){return a[0]=b[0]*c,a[1]=b[1]*c,a},"scaleAndAdd",0,function(a,b,c,d){return a[0]=b[0]+c[0]*d,a[1]=b[1]+c[1]*d,a},"set",0,function(a,b,c){return a[0]=b,a[1]=c,a},"sqrDist",0,d8,"sqrLen",0,ea,"squaredDistance",0,d8,"squaredLength",0,ea,"str",0,function(a){return`vec2(${a[0]}, ${a[1]})`},"sub",0,d4,"subtract",0,d4,"transformMat2",0,function(a,b,c){let d=b[0],e=b[1];return a[0]=c[0]*d+c[2]*e,a[1]=c[1]*d+c[3]*e,a},"transformMat2d",0,function(a,b,c){let d=b[0],e=b[1];return a[0]=c[0]*d+c[2]*e+c[4],a[1]=c[1]*d+c[3]*e+c[5],a},"transformMat3",0,function(a,b,c){let d=b[0],e=b[1];return a[0]=c[0]*d+c[3]*e+c[6],a[1]=c[1]*d+c[4]*e+c[7],a},"transformMat4",0,eb,"zero",0,function(a){return a[0]=0,a[1]=0,a}],15486),(m=q||(q={}))[m.COL0ROW0=0]="COL0ROW0",m[m.COL0ROW1=1]="COL0ROW1",m[m.COL0ROW2=2]="COL0ROW2",m[m.COL0ROW3=3]="COL0ROW3",m[m.COL1ROW0=4]="COL1ROW0",m[m.COL1ROW1=5]="COL1ROW1",m[m.COL1ROW2=6]="COL1ROW2",m[m.COL1ROW3=7]="COL1ROW3",m[m.COL2ROW0=8]="COL2ROW0",m[m.COL2ROW1=9]="COL2ROW1",m[m.COL2ROW2=10]="COL2ROW2",m[m.COL2ROW3=11]="COL2ROW3",m[m.COL3ROW0=12]="COL3ROW0",m[m.COL3ROW1=13]="COL3ROW1",m[m.COL3ROW2=14]="COL3ROW2",m[m.COL3ROW3=15]="COL3ROW3";let ed=45*Math.PI/180,ee=Object.freeze([1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1]);class ef extends d2{static get IDENTITY(){return d||Object.freeze(d=new ef),d}static get ZERO(){return c||Object.freeze(c=new ef([0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0])),c}get ELEMENTS(){return 16}get RANK(){return 4}get INDICES(){return q}constructor(a){super(-0,-0,-0,-0,-0,-0,-0,-0,-0,-0,-0,-0,-0,-0,-0,-0),1==arguments.length&&Array.isArray(a)?this.copy(a):this.identity()}copy(a){return this[0]=a[0],this[1]=a[1],this[2]=a[2],this[3]=a[3],this[4]=a[4],this[5]=a[5],this[6]=a[6],this[7]=a[7],this[8]=a[8],this[9]=a[9],this[10]=a[10],this[11]=a[11],this[12]=a[12],this[13]=a[13],this[14]=a[14],this[15]=a[15],this.check()}set(a,b,c,d,e,f,g,h,i,j,k,l,m,n,o,p){return this[0]=a,this[1]=b,this[2]=c,this[3]=d,this[4]=e,this[5]=f,this[6]=g,this[7]=h,this[8]=i,this[9]=j,this[10]=k,this[11]=l,this[12]=m,this[13]=n,this[14]=o,this[15]=p,this.check()}setRowMajor(a,b,c,d,e,f,g,h,i,j,k,l,m,n,o,p){return this[0]=a,this[1]=e,this[2]=i,this[3]=m,this[4]=b,this[5]=f,this[6]=j,this[7]=n,this[8]=c,this[9]=g,this[10]=k,this[11]=o,this[12]=d,this[13]=h,this[14]=l,this[15]=p,this.check()}toRowMajor(a){return a[0]=this[0],a[1]=this[4],a[2]=this[8],a[3]=this[12],a[4]=this[1],a[5]=this[5],a[6]=this[9],a[7]=this[13],a[8]=this[2],a[9]=this[6],a[10]=this[10],a[11]=this[14],a[12]=this[3],a[13]=this[7],a[14]=this[11],a[15]=this[15],a}identity(){return this.copy(ee)}fromObject(a){return this.check()}fromQuaternion(a){return b9(this,a),this.check()}frustum(a){var b,c,d,e,f,g;let{left:h,right:i,bottom:j,top:k,near:l=.1,far:m=500}=a;return m===1/0?(b=this,c=h,d=i,e=j,f=k,g=l,b[0]=2*g/(d-c),b[1]=0,b[2]=0,b[3]=0,b[4]=0,b[5]=2*g/(f-e),b[6]=0,b[7]=0,b[8]=(d+c)/(d-c),b[9]=(f+e)/(f-e),b[10]=-1,b[11]=-1,b[12]=0,b[13]=0,b[14]=-2*g,b[15]=0):ca(this,h,i,j,k,l,m),this.check()}lookAt(a){let{eye:b,center:c=[0,0,0],up:d=[0,1,0]}=a;return cd(this,b,c,d),this.check()}ortho(a){let{left:b,right:c,bottom:d,top:e,near:f=.1,far:g=500}=a;return cc(this,b,c,d,e,f,g),this.check()}orthographic(a){let{fovy:b=ed,aspect:c=1,focalDistance:d=1,near:e=.1,far:f=500}=a;eg(b);let g=d*Math.tan(b/2),h=g*c;return this.ortho({left:-h,right:h,bottom:-g,top:g,near:e,far:f})}perspective(a){let{fovy:b=45*Math.PI/180,aspect:c=1,near:d=.1,far:e=500}=a;return eg(b),cb(this,b,c,d,e),this.check()}determinant(){return b_(this)}getScale(a=[-0,-0,-0]){return a[0]=Math.sqrt(this[0]*this[0]+this[1]*this[1]+this[2]*this[2]),a[1]=Math.sqrt(this[4]*this[4]+this[5]*this[5]+this[6]*this[6]),a[2]=Math.sqrt(this[8]*this[8]+this[9]*this[9]+this[10]*this[10]),a}getTranslation(a=[-0,-0,-0]){return a[0]=this[12],a[1]=this[13],a[2]=this[14],a}getRotation(a,b){a=a||[-0,-0,-0,-0,-0,-0,-0,-0,-0,-0,-0,-0,-0,-0,-0,-0],b=b||[-0,-0,-0];let c=this.getScale(b),d=1/c[0],e=1/c[1],f=1/c[2];return a[0]=this[0]*d,a[1]=this[1]*e,a[2]=this[2]*f,a[3]=0,a[4]=this[4]*d,a[5]=this[5]*e,a[6]=this[6]*f,a[7]=0,a[8]=this[8]*d,a[9]=this[9]*e,a[10]=this[10]*f,a[11]=0,a[12]=0,a[13]=0,a[14]=0,a[15]=1,a}getRotationMatrix3(a,b){a=a||[-0,-0,-0,-0,-0,-0,-0,-0,-0],b=b||[-0,-0,-0];let c=this.getScale(b),d=1/c[0],e=1/c[1],f=1/c[2];return a[0]=this[0]*d,a[1]=this[1]*e,a[2]=this[2]*f,a[3]=this[4]*d,a[4]=this[5]*e,a[5]=this[6]*f,a[6]=this[8]*d,a[7]=this[9]*e,a[8]=this[10]*f,a}transpose(){return bZ(this,this),this.check()}invert(){return b$(this,this),this.check()}multiplyLeft(a){return b0(this,a,this),this.check()}multiplyRight(a){return b0(this,this,a),this.check()}rotateX(a){return b4(this,this,a),this.check()}rotateY(a){return b5(this,this,a),this.check()}rotateZ(a){return b6(this,this,a),this.check()}rotateXYZ(a){return this.rotateX(a[0]).rotateY(a[1]).rotateZ(a[2])}rotateAxis(a,b){return b3(this,this,a,b),this.check()}scale(a){return b2(this,this,Array.isArray(a)?a:[a,a,a]),this.check()}translate(a){return b1(this,this,a),this.check()}transform(a,b){return 4===a.length?(dG(b=co(b||[-0,-0,-0,-0],a,this),4),b):this.transformAsPoint(a,b)}transformAsPoint(a,b){let c,{length:d}=a;switch(d){case 2:c=eb(b||[-0,-0],a,this);break;case 3:c=dT(b||[-0,-0,-0],a,this);break;default:throw Error("Illegal vector")}return dG(c,a.length),c}transformAsVector(a,b){let c;switch(a.length){case 2:var d;let e,f,g;d=b||[-0,-0],e=a[0],f=a[1],g=this[3]*e+this[7]*f||1,d[0]=(this[0]*e+this[4]*f)/g,d[1]=(this[1]*e+this[5]*f)/g,c=d;break;case 3:c=d_(b||[-0,-0,-0],a,this);break;default:throw Error("Illegal vector")}return dG(c,a.length),c}transformPoint(a,b){return this.transformAsPoint(a,b)}transformVector(a,b){return this.transformAsPoint(a,b)}transformDirection(a,b){return this.transformAsVector(a,b)}makeRotationX(a){return this.identity().rotateX(a)}makeTranslation(a,b,c){return this.identity().translate([a,b,c])}}function eg(a){if(a>2*Math.PI)throw Error("expected radians")}var cq=cq;function eh(a,b){let c=cq.transformMat4([],b,a);return cq.scale(c,c,1/c[3]),c}function ei(a,b,c){return a<b?b:a>c?c:a}let ej=Math.log2||function(a){return Math.log(a)*Math.LOG2E};var cf=cf,ek=a.i(15486),ek=ek,el=a.i(33649),el=el;function em(a,b){if(!a)throw Error(b||"@math.gl/web-mercator: assertion failed.")}let en=Math.PI,eo=en/4,ep=en/180,eq=180/en;function er(a){let[b,c]=a;em(Number.isFinite(b)),em(Number.isFinite(c)&&c>=-90&&c<=90,"invalid latitude");let d=512*(en+Math.log(Math.tan(eo+c*ep*.5)))/(2*en);return[512*(b*ep+en)/(2*en),d]}function es(a){let[b,c]=a,d=2*(Math.atan(Math.exp(c/512*(2*en)-en))-eo);return[(b/512*(2*en)-en)*eq,d*eq]}function et(a){return 512/4003e4/Math.cos(a*ep)}function eu(a){let{latitude:b,longitude:c,highPrecision:d=!1}=a;em(Number.isFinite(b)&&Number.isFinite(c));let e=Math.cos(b*ep),f=512/360/e,g=512/4003e4/e,h={unitsPerMeter:[g,g,g],metersPerUnit:[1/g,1/g,1/g],unitsPerDegree:[512/360,f,g],degreesPerUnit:[1/(512/360),1/f,1/g]};if(d){let a=ep*Math.tan(b*ep)/e,c=512/4003e4*a,d=c/f*g;h.unitsPerDegree2=[0,512/360*a/2,c],h.unitsPerMeter2=[d,0,d]}return h}function ev(a,b){let[c,d,e]=a,[f,g,h]=b,{unitsPerMeter:i,unitsPerMeter2:j}=eu({longitude:c,latitude:d,highPrecision:!0}),k=er(a);k[0]+=f*(i[0]+j[0]*g),k[1]+=g*(i[1]+j[1]*g);let l=es(k);return Number.isFinite(e)||Number.isFinite(h)?[l[0],l[1],(e||0)+(h||0)]:l}function ew(a){return 2*Math.atan(.5/a)*eq}function ex(a){return .5/Math.tan(.5*a*ep)}function ey(a,b){let[c,d,e=0]=a;return em(Number.isFinite(c)&&Number.isFinite(d)&&Number.isFinite(e)),eh(b,[c,d,e,1])}function ez(a,b,c=0){let[d,e,f]=a;if(em(Number.isFinite(d)&&Number.isFinite(e),"invalid pixel coordinate"),Number.isFinite(f))return eh(b,[d,e,f,1]);let g=eh(b,[d,e,0,1]),h=eh(b,[d,e,1,1]),i=g[2],j=h[2];return ek.lerp([],g,h,i===j?0:((c||0)-i)/(j-i))}let eA=`
layout(std140) uniform shadowUniforms {
  bool drawShadowMap;
  bool useShadowMap;
  vec4 color;
  highp int lightId;
  float lightCount;
  mat4 viewProjectionMatrix0;
  mat4 viewProjectionMatrix1;
  vec4 projectCenter0;
  vec4 projectCenter1;
} shadow;
`,eB=`
const int max_lights = 2;

out vec3 shadow_vPosition[max_lights];

vec4 shadow_setVertexPosition(vec4 position_commonspace) {
  mat4 viewProjectionMatrices[max_lights];
  viewProjectionMatrices[0] = shadow.viewProjectionMatrix0;
  viewProjectionMatrices[1] = shadow.viewProjectionMatrix1;
  vec4 projectCenters[max_lights];
  projectCenters[0] = shadow.projectCenter0;
  projectCenters[1] = shadow.projectCenter1;

  if (shadow.drawShadowMap) {
    return project_common_position_to_clipspace(position_commonspace, viewProjectionMatrices[shadow.lightId], projectCenters[shadow.lightId]);
  }
  if (shadow.useShadowMap) {
    for (int i = 0; i < max_lights; i++) {
      if(i < int(shadow.lightCount)) {
        vec4 shadowMap_position = project_common_position_to_clipspace(position_commonspace, viewProjectionMatrices[i], projectCenters[i]);
        shadow_vPosition[i] = (shadowMap_position.xyz / shadowMap_position.w + 1.0) / 2.0;
      }
    }
  }
  return gl_Position;
}
`,eC=`
${eA}
${eB}
`,eD=`
const int max_lights = 2;
uniform sampler2D shadow_uShadowMap0;
uniform sampler2D shadow_uShadowMap1;

in vec3 shadow_vPosition[max_lights];

const vec4 bitPackShift = vec4(1.0, 255.0, 65025.0, 16581375.0);
const vec4 bitUnpackShift = 1.0 / bitPackShift;
const vec4 bitMask = vec4(1.0 / 255.0, 1.0 / 255.0, 1.0 / 255.0,  0.0);

float shadow_getShadowWeight(vec3 position, sampler2D shadowMap) {
  vec4 rgbaDepth = texture(shadowMap, position.xy);

  float z = dot(rgbaDepth, bitUnpackShift);
  return smoothstep(0.001, 0.01, position.z - z);
}

vec4 shadow_filterShadowColor(vec4 color) {
  if (shadow.drawShadowMap) {
    vec4 rgbaDepth = fract(gl_FragCoord.z * bitPackShift);
    rgbaDepth -= rgbaDepth.gbaa * bitMask;
    return rgbaDepth;
  }
  if (shadow.useShadowMap) {
    float shadowAlpha = 0.0;
    shadowAlpha += shadow_getShadowWeight(shadow_vPosition[0], shadow_uShadowMap0);
    if(shadow.lightCount > 1.0) {
      shadowAlpha += shadow_getShadowWeight(shadow_vPosition[1], shadow_uShadowMap1);
    }
    shadowAlpha *= shadow.color.a / shadow.lightCount;
    float blendedAlpha = shadowAlpha + color.a * (1.0 - shadowAlpha);

    return vec4(
      mix(color.rgb, shadow.color.rgb, shadowAlpha / blendedAlpha),
      blendedAlpha
    );
  }
  return color;
}
`,eE=`
${eA}
${eD}
`,eF=dd(function({viewport:a,center:b}){return new ef(a.viewProjectionMatrix).invert().transform(b)}),eG=dd(function({viewport:a,shadowMatrices:b}){let c=[],d=a.pixelUnprojectionMatrix,e=a.isGeospatial?void 0:1,f=[[0,0,e],[a.width,0,e],[0,a.height,e],[a.width,a.height,e],[0,0,-1],[a.width,0,-1],[0,a.height,-1],[a.width,a.height,-1]].map(a=>(function(a,b){let[c,d,e]=a,f=ez([c,d,e],b);return Number.isFinite(e)?f:[f[0],f[1],0]})(a,d));for(let d of b){let b=d.clone().translate(new d1(a.center).negate()),e=f.map(a=>b.transform(a)),g=new ef().ortho({left:Math.min(...e.map(a=>a[0])),right:Math.max(...e.map(a=>a[0])),bottom:Math.min(...e.map(a=>a[1])),top:Math.max(...e.map(a=>a[1])),near:Math.min(...e.map(a=>-a[2])),far:Math.max(...e.map(a=>-a[2]))});c.push(g.multiplyRight(d))}return c}),eH=[0,0,0,1],eI=[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,0],eJ={name:"shadow",dependencies:[dx],vs:eC,fs:eE,inject:{"vs:DECKGL_FILTER_GL_POSITION":`
    position = shadow_setVertexPosition(geometry.position);
    `,"fs:DECKGL_FILTER_COLOR":`
    color = shadow_filterShadowColor(color);
    `},getUniforms:function(a){let{shadowEnabled:b=!0,project:c}=a;if(!b||!c||!a.shadowMatrices||!a.shadowMatrices.length)return{drawShadowMap:!1,useShadowMap:!1,shadow_uShadowMap0:a.dummyShadowMap,shadow_uShadowMap1:a.dummyShadowMap};let d=dx.getUniforms(c),e=eF({viewport:c.viewport,center:d.center}),f=[],g=eG({shadowMatrices:a.shadowMatrices,viewport:c.viewport}).slice();for(let b=0;b<a.shadowMatrices.length;b++){let a=g[b],h=a.clone().translate(new d1(c.viewport.center).negate());d.coordinateSystem===dk("lnglat")&&d.projectionMode===c9.WEB_MERCATOR?(g[b]=h,f[b]=e):(g[b]=a.clone().multiplyRight(eI),f[b]=h.transform(e))}let h={drawShadowMap:!!a.drawToShadowMap,useShadowMap:!!a.shadowMaps&&a.shadowMaps.length>0,color:a.shadowColor||eH,lightId:a.shadowLightId||0,lightCount:a.shadowMatrices.length,shadow_uShadowMap0:a.dummyShadowMap,shadow_uShadowMap1:a.dummyShadowMap};for(let a=0;a<g.length;a++)h[`viewProjectionMatrix${a}`]=g[a],h[`projectCenter${a}`]=f[a];for(let b=0;b<2;b++)h[`shadow_uShadowMap${b}`]=a.shadowMaps&&a.shadowMaps[b]||a.dummyShadowMap;return h},uniformTypes:{drawShadowMap:"f32",useShadowMap:"f32",color:"vec4<f32>",lightId:"i32",lightCount:"f32",viewProjectionMatrix0:"mat4x4<f32>",viewProjectionMatrix1:"mat4x4<f32>",projectCenter0:"vec4<f32>",projectCenter1:"vec4<f32>"}},eK={props:{},uniforms:{},name:"picking",uniformTypes:{isActive:"f32",isAttribute:"f32",isHighlightActive:"f32",useByteColors:"f32",highlightedObjectColor:"vec3<f32>",highlightColor:"vec4<f32>"},defaultUniforms:{isActive:!1,isAttribute:!1,isHighlightActive:!1,useByteColors:!0,highlightedObjectColor:[0,0,0],highlightColor:[0,1,1,1]},vs:`\
layout(std140) uniform pickingUniforms {
  float isActive;
  float isAttribute;
  float isHighlightActive;
  float useByteColors;
  vec3 highlightedObjectColor;
  vec4 highlightColor;
} picking;

out vec4 picking_vRGBcolor_Avalid;

// Normalize unsigned byte color to 0-1 range
vec3 picking_normalizeColor(vec3 color) {
  return picking.useByteColors > 0.5 ? color / 255.0 : color;
}

// Normalize unsigned byte color to 0-1 range
vec4 picking_normalizeColor(vec4 color) {
  return picking.useByteColors > 0.5 ? color / 255.0 : color;
}

bool picking_isColorZero(vec3 color) {
  return dot(color, vec3(1.0)) < 0.00001;
}

bool picking_isColorValid(vec3 color) {
  return dot(color, vec3(1.0)) > 0.00001;
}

// Check if this vertex is highlighted 
bool isVertexHighlighted(vec3 vertexColor) {
  vec3 highlightedObjectColor = picking_normalizeColor(picking.highlightedObjectColor);
  return
    bool(picking.isHighlightActive) && picking_isColorZero(abs(vertexColor - highlightedObjectColor));
}

// Set the current picking color
void picking_setPickingColor(vec3 pickingColor) {
  pickingColor = picking_normalizeColor(pickingColor);

  if (bool(picking.isActive)) {
    // Use alpha as the validity flag. If pickingColor is [0, 0, 0] fragment is non-pickable
    picking_vRGBcolor_Avalid.a = float(picking_isColorValid(pickingColor));

    if (!bool(picking.isAttribute)) {
      // Stores the picking color so that the fragment shader can render it during picking
      picking_vRGBcolor_Avalid.rgb = pickingColor;
    }
  } else {
    // Do the comparison with selected item color in vertex shader as it should mean fewer compares
    picking_vRGBcolor_Avalid.a = float(isVertexHighlighted(pickingColor));
  }
}

void picking_setPickingAttribute(float value) {
  if (bool(picking.isAttribute)) {
    picking_vRGBcolor_Avalid.r = value;
  }
}

void picking_setPickingAttribute(vec2 value) {
  if (bool(picking.isAttribute)) {
    picking_vRGBcolor_Avalid.rg = value;
  }
}

void picking_setPickingAttribute(vec3 value) {
  if (bool(picking.isAttribute)) {
    picking_vRGBcolor_Avalid.rgb = value;
  }
}
`,fs:`\
layout(std140) uniform pickingUniforms {
  float isActive;
  float isAttribute;
  float isHighlightActive;
  float useByteColors;
  vec3 highlightedObjectColor;
  vec4 highlightColor;
} picking;

in vec4 picking_vRGBcolor_Avalid;

/*
 * Returns highlight color if this item is selected.
 */
vec4 picking_filterHighlightColor(vec4 color) {
  // If we are still picking, we don't highlight
  if (picking.isActive > 0.5) {
    return color;
  }

  bool selected = bool(picking_vRGBcolor_Avalid.a);

  if (selected) {
    // Blend in highlight color based on its alpha value
    float highLightAlpha = picking.highlightColor.a;
    float blendedAlpha = highLightAlpha + color.a * (1.0 - highLightAlpha);
    float highLightRatio = highLightAlpha / blendedAlpha;

    vec3 blendedRGB = mix(color.rgb, picking.highlightColor.rgb, highLightRatio);
    return vec4(blendedRGB, blendedAlpha);
  } else {
    return color;
  }
}

/*
 * Returns picking color if picking enabled else unmodified argument.
 */
vec4 picking_filterPickingColor(vec4 color) {
  if (bool(picking.isActive)) {
    if (picking_vRGBcolor_Avalid.a == 0.0) {
      discard;
    }
    return picking_vRGBcolor_Avalid;
  }
  return color;
}

/*
 * Returns picking color if picking is enabled if not
 * highlight color if this item is selected, otherwise unmodified argument.
 */
vec4 picking_filterColor(vec4 color) {
  vec4 highlightColor = picking_filterHighlightColor(color);
  return picking_filterPickingColor(highlightColor);
}
`,getUniforms:function(a={},b){let c={},d=bE(a.useByteColors,!0);return void 0===a.highlightedObjectColor||(null===a.highlightedObjectColor?c.isHighlightActive=!1:(c.isHighlightActive=!0,c.highlightedObjectColor=a.highlightedObjectColor.slice(0,3))),a.highlightColor&&(c.highlightColor=function(a,b=!0){let c=bF(a.slice(0,3),b),d=Number.isFinite(a[3]),e=d?a[3]:1;return[c[0],c[1],c[2],b&&d?e/255:e]}(a.highlightColor,d)),void 0!==a.isActive&&(c.isActive=!!a.isActive,c.isAttribute=!!a.isAttribute),void 0!==a.useByteColors&&(c.useByteColors=!!a.useByteColors),c}},eL=`\
struct pickingUniforms {
  isActive: f32,
  isAttribute: f32,
  isHighlightActive: f32,
  useByteColors: f32,
  highlightedObjectColor: vec3<f32>,
  highlightColor: vec4<f32>,
};

@group(0) @binding(auto) var<uniform> picking: pickingUniforms;

fn picking_normalizeColor(color: vec3<f32>) -> vec3<f32> {
  return select(color, color / 255.0, picking.useByteColors > 0.5);
}

fn picking_normalizeColor4(color: vec4<f32>) -> vec4<f32> {
  return select(color, color / 255.0, picking.useByteColors > 0.5);
}

fn picking_isColorZero(color: vec3<f32>) -> bool {
  return dot(color, vec3<f32>(1.0)) < 0.00001;
}

fn picking_isColorValid(color: vec3<f32>) -> bool {
  return dot(color, vec3<f32>(1.0)) > 0.00001;
}
`,eM={...eK,source:eL,defaultUniforms:{...eK.defaultUniforms,useByteColors:!0},inject:{"vs:DECKGL_FILTER_GL_POSITION":`
    // for picking depth values
    picking_setPickingAttribute(position.z / position.w);
  `,"vs:DECKGL_FILTER_COLOR":`
  picking_setPickingColor(geometry.pickingColor);
  `,"fs:DECKGL_FILTER_COLOR":{order:99,injection:`
  // use highlight color if this fragment belongs to the selected object.
  color = picking_filterHighlightColor(color);

  // use picking color if rendering to picking FBO.
  color = picking_filterPickingColor(color);
    `}}},eN=[bT],eO=["vs:DECKGL_FILTER_SIZE(inout vec3 size, VertexGeometry geometry)","vs:DECKGL_FILTER_GL_POSITION(inout vec4 position, VertexGeometry geometry)","vs:DECKGL_FILTER_COLOR(inout vec4 color, VertexGeometry geometry)","fs:DECKGL_FILTER_COLOR(inout vec4 color, FragmentGeometry geometry)"],eP=[],eQ=Symbol.for("component"),eR=Symbol.for("propTypes"),eS=Symbol.for("deprecatedProps"),eT=Symbol.for("asyncPropDefaults"),eU=Symbol.for("asyncPropOriginal"),eV=Symbol.for("asyncPropResolved"),eW={};function eX(a){eW=a}function eY(a,b,c,d){cs.level>0&&eW[a]&&eW[a].call(null,b,c,d)}function eZ(a,b=()=>!0){return Array.isArray(a)?function a(b,c,d){let e=-1;for(;++e<b.length;){let f=b[e];Array.isArray(f)?a(f,c,d):c(f)&&d.push(f)}return d}(a,b,[]):b(a)?[a]:[]}var e$=a.i(88088);let e_=a=>null!==a&&"object"==typeof a,e0=a=>e_(a)&&a.constructor===({}).constructor,e1=a=>"u">typeof SharedArrayBuffer&&a instanceof SharedArrayBuffer,e2=a=>e_(a)&&"number"==typeof a.byteLength&&"function"==typeof a.slice,e3=a=>"u">typeof Response&&a instanceof Response||e_(a)&&"function"==typeof a.arrayBuffer&&"function"==typeof a.text&&"function"==typeof a.json,e4=a=>"u">typeof Blob&&a instanceof Blob,e5=a=>{let b,c;return b=a,"u">typeof ReadableStream&&b instanceof ReadableStream||e_(b)&&"function"==typeof b.tee&&"function"==typeof b.cancel&&"function"==typeof b.getReader||e_(c=a)&&"function"==typeof c.read&&"function"==typeof c.pipe&&"boolean"==typeof c.readable};function e6(a,b){if(!a)throw Error(b||"loader assertion failed.")}function e7(a){return!!a&&(Array.isArray(a)&&(a=a[0]),Array.isArray(a?.extensions))}function e8(a){let b;return e6(a,"null loader"),e6(e7(a),"invalid loader"),Array.isArray(a)&&(b=a[1],a={...a=a[0],options:{...a.options,...b}}),(a?.parseTextSync||a?.parseText)&&(a.text=!0),a.text||(a.binary=!0),a}let e9={};class fa extends Error{constructor(a,b){super(a),this.reason=b.reason,this.url=b.url,this.response=b.response}reason;url;response}let fb=/^data:([-\w.]+\/[-\w.+]+)(;|,)/,fc=/^([-\w.]+\/[-\w.+]+)/;function fd(a,b){return a.toLowerCase()===b.toLowerCase()}function fe(a){let b=fb.exec(a);return b?b[1]:""}let ff=/\?.*/;function fg(a){return a.replace(ff,"")}function fh(a){return e3(a)?a.url:e4(a)?("name"in a?a.name:"")||"":"string"==typeof a?a:""}function fi(a){if(e3(a)){let b,c=a.headers.get("content-type")||"",d=fg(a.url);return((b=fc.exec(c))?b[1]:c)||fe(d)}return e4(a)?a.type||"":"string"==typeof a?fe(a):""}async function fj(a){var b;if(e3(a))return a;let c={},d=e3(b=a)?b.headers["content-length"]||-1:e4(b)?b.size:"string"==typeof b?b.length:b instanceof ArrayBuffer||ArrayBuffer.isView(b)?b.byteLength:-1;d>=0&&(c["content-length"]=String(d));let e=fh(a),f=fi(a);f&&(c["content-type"]=f);let g=await fm(a);g&&(c["x-first-bytes"]=g),"string"==typeof a&&(a=new TextEncoder().encode(a));let h=new Response(a,{headers:c});return Object.defineProperty(h,"url",{value:e}),h}async function fk(a){if(!a.ok)throw await fl(a)}async function fl(a){let b=function(a){if(a.length<50)return a;let b=a.slice(a.length-15),c=a.substr(0,32);return`${c}...${b}`}(a.url),c=`Failed to fetch resource (${a.status}) ${a.statusText}: ${b}`;c=c.length>100?`${c.slice(0,100)}...`:c;let d={reason:a.statusText,url:a.url,response:a};try{let b=a.headers.get("Content-Type");d.reason=!a.bodyUsed&&b?.includes("application/json")?await a.json():await a.text()}catch(a){}return new fa(c,d)}async function fm(a){if("string"==typeof a)return`data:,${a.slice(0,5)}`;if(a instanceof Blob){let b=a.slice(0,5);return await new Promise(a=>{let c=new FileReader;c.onload=b=>a(b?.target?.result),c.readAsDataURL(b)})}if(a instanceof ArrayBuffer){let b=function(a){let b="",c=new Uint8Array(a);for(let a=0;a<c.byteLength;a++)b+=String.fromCharCode(c[a]);return btoa(b)}(a.slice(0,5));return`data:base64,${b}`}return null}async function fn(a,b){if("string"==typeof a){var c;let d=function(a){for(let b in e9)if(a.startsWith(b)){let c=e9[b];a=a.replace(b,c)}return a.startsWith("http://")||a.startsWith("https://")||(a=`${a}`),a}(a);return!((c=d).startsWith("http:")||c.startsWith("https:"))&&!d.startsWith("data:")&&globalThis.loaders?.fetchNode?globalThis.loaders?.fetchNode(d,b):await fetch(d,b)}return await fj(a)}a.s(["dirname",0,function(a){let b=a?a.lastIndexOf("/"):-1;return b>=0?a.substr(0,b):""},"filename",0,function(a){let b=a?a.lastIndexOf("/"):-1;return b>=0?a.substr(b+1):a},"join",0,function(...a){return(a=a.map((b,c)=>(c&&(b=b.replace(RegExp("^/"),"")),c!==a.length-1&&(b=b.replace(RegExp("/$"),"")),b))).join("/")},"resolve",0,function(...a){let b,c=[];for(let b=0;b<a.length;b++)c[b]=a[b];let d="",e=!1;for(let a=c.length-1;a>=-1&&!e;a--){let f;a>=0?f=c[a]:(void 0===b&&(b=function(){if("u">typeof process&&void 0!==process.cwd)return process.cwd();let a=window.location?.pathname;return a?.slice(0,a.lastIndexOf("/")+1)||""}()),f=b),0!==f.length&&(d=`${f}/${d}`,e=47===f.charCodeAt(0))}return(d=function(a,b){let c,d="",e=-1,f=0,g=!1;for(let h=0;h<=a.length;++h){if(h<a.length)c=a.charCodeAt(h);else if(47===c)break;else c=47;if(47===c){if(e===h-1||1===f);else if(e!==h-1&&2===f){if(d.length<2||!g||46!==d.charCodeAt(d.length-1)||46!==d.charCodeAt(d.length-2)){if(d.length>2){let a=d.length-1,b=a;for(;b>=0&&47!==d.charCodeAt(b);--b);if(b!==a){d=-1===b?"":d.slice(0,b),e=h,f=0,g=!1;continue}}else if(2===d.length||1===d.length){d="",e=h,f=0,g=!1;continue}}b&&(d.length>0?d+="/..":d="..",g=!0)}else{let b=a.slice(e+1,h);d.length>0?d+=`/${b}`:d=b,g=!1}e=h,f=0}else 46===c&&-1!==f?++f:f=-1}return d}(d,!e),e)?`/${d}`:d.length>0?d:"."}],79709);var fo=a.i(79709),fo=fo;let fp=new cr.Log({id:"loaders.gl"});class fq{log(){return()=>{}}info(){return()=>{}}warn(){return()=>{}}error(){return()=>{}}}let fr={self:"u">typeof self&&self,window:!1,global:a.g,document:"u">typeof document&&document};fr.self||fr.window||fr.global,fr.window||fr.self||fr.global,fr.global||fr.self||fr.window,fr.document;let fs="object"!=typeof process||"[object process]"!==String(process),ft="u">typeof process&&process.version&&/v([0-9]*)/.exec(process.version);ft&&parseFloat(ft[1]);let fu={core:{baseUrl:void 0,fetch:null,mimeType:void 0,fallbackMimeType:void 0,ignoreRegisteredLoaders:void 0,nothrow:!1,log:new class{console;constructor(){this.console=console}log(...a){return this.console.log.bind(this.console,...a)}info(...a){return this.console.info.bind(this.console,...a)}warn(...a){return this.console.warn.bind(this.console,...a)}error(...a){return this.console.error.bind(this.console,...a)}},useLocalLibraries:!1,CDN:"https://unpkg.com/@loaders.gl",worker:!0,maxConcurrency:3,maxMobileConcurrency:1,reuseWorkers:fs,_nodeWorkers:!1,_workerType:"",limit:0,_limitMB:0,batchSize:"auto",batchDebounceMs:0,metadata:!1,transforms:[]}},fv={baseUri:"core.baseUrl",fetch:"core.fetch",mimeType:"core.mimeType",fallbackMimeType:"core.fallbackMimeType",ignoreRegisteredLoaders:"core.ignoreRegisteredLoaders",nothrow:"core.nothrow",log:"core.log",useLocalLibraries:"core.useLocalLibraries",CDN:"core.CDN",worker:"core.worker",maxConcurrency:"core.maxConcurrency",maxMobileConcurrency:"core.maxMobileConcurrency",reuseWorkers:"core.reuseWorkers",_nodeWorkers:"core.nodeWorkers",_workerType:"core._workerType",_worker:"core._workerType",limit:"core.limit",_limitMB:"core._limitMB",batchSize:"core.batchSize",batchDebounceMs:"core.batchDebounceMs",metadata:"core.metadata",transforms:"core.transforms",throws:"nothrow",dataType:"(no longer used)",uri:"core.baseUrl",method:"core.fetch.method",headers:"core.fetch.headers",body:"core.fetch.body",mode:"core.fetch.mode",credentials:"core.fetch.credentials",cache:"core.fetch.cache",redirect:"core.fetch.redirect",referrer:"core.fetch.referrer",referrerPolicy:"core.fetch.referrerPolicy",integrity:"core.fetch.integrity",keepalive:"core.fetch.keepalive",signal:"core.fetch.signal"},fw=["baseUrl","fetch","mimeType","fallbackMimeType","ignoreRegisteredLoaders","nothrow","log","useLocalLibraries","CDN","worker","maxConcurrency","maxMobileConcurrency","reuseWorkers","_nodeWorkers","_workerType","limit","_limitMB","batchSize","batchDebounceMs","metadata","transforms"];function fx(){globalThis.loaders=globalThis.loaders||{};let{loaders:a}=globalThis;return a._state||(a._state={}),a._state}function fy(){let a=fx();return a.globalOptions=a.globalOptions||{...fu,core:{...fu.core}},fz(a.globalOptions)}function fz(a){var b;let c,d=(c={...b=a},b.core&&(c.core={...b.core}),c);for(let a of(fC(d),fw))d.core&&void 0!==d.core[a]&&delete d[a];return d.core&&void 0!==d.core._workerType&&delete d._worker,d}function fA(a,b,c,d,e){let f=b||"Top level",g=b?`${b}.`:"";for(let h in a){let i=!b&&e_(a[h]),j="baseUri"===h&&!b,k="workerUrl"===h&&b;if(!(h in c)&&!j&&!k){if(h in d)fp.level>0&&fp.warn(`${f} loader option '${g}${h}' no longer supported, use '${d[h]}'`)();else if(!i&&fp.level>0){let a=function(a,b){let c=a.toLowerCase(),d="";for(let e of b)for(let b in e.options){if(a===b)return`Did you mean '${e.id}.${b}'?`;let f=b.toLowerCase();(c.startsWith(f)||f.startsWith(c))&&(d=d||`Did you mean '${e.id}.${b}'?`)}return d}(h,e);fp.warn(`${f} loader option '${g}${h}' not recognized. ${a}`)()}}}}function fB(a,b){for(let c in b)c in b&&(e0(b[c])&&e0(a[c])?a[c]={...a[c],...b[c]}:a[c]=b[c])}function fC(a){for(let b of(void 0!==a.baseUri&&(a.core||={},void 0===a.core.baseUrl&&(a.core.baseUrl=a.baseUri)),fw))if(void 0!==a[b]){let c=a.core=a.core||{};void 0===c[b]&&(c[b]=a[b])}let b=a._worker;void 0!==b&&(a.core||={},void 0===a.core._workerType&&(a.core._workerType=b))}function fD(a,b){let c=fy(),d=a||c,e=d.fetch??d.core?.fetch;return"function"==typeof e?e:e_(e)?a=>fn(a,e):b?.fetch?b?.fetch:fn}let fE={self:"u">typeof self&&self,window:!1,global:a.g,document:"u">typeof document&&document};fE.self||fE.window||fE.global,fE.window||fE.self||fE.global,fE.global||fE.self||fE.window,fE.document;let fF="object"!=typeof process||"[object process]"!==String(process),fG="u">typeof process&&process.version&&/v([0-9]*)/.exec(process.version);fG&&parseFloat(fG[1]);var fH=a.i(37702);fH.parentPort;let fI=fH.Worker;function fJ(a,b){if(!a)throw Error(b||"loaders.gl assertion failed.")}let fK=new Map;function fL(a){let b=new Blob([a],{type:"application/javascript"});return URL.createObjectURL(b)}function fM(a){return!!a&&!!(a instanceof ArrayBuffer||"u">typeof MessagePort&&a instanceof MessagePort||"u">typeof ImageBitmap&&a instanceof ImageBitmap||"u">typeof OffscreenCanvas&&a instanceof OffscreenCanvas)}let fN=()=>{};class fO{name;source;url;terminated=!1;worker;onMessage;onError;_loadableURL="";static isSupported(){return"u">typeof Worker&&fF||void 0!==fI&&!fF}constructor(a){const{name:b,source:c,url:d}=a;fJ(c||d),this.name=b,this.source=c,this.url=d,this.onMessage=fN,this.onError=a=>console.log(a),this.worker=fF?this._createBrowserWorker():this._createNodeWorker()}destroy(){this.onMessage=fN,this.onError=fN,this.worker.terminate(),this.terminated=!0}get isRunning(){return!!this.onMessage}postMessage(a,b){b=b||function a(b,c=!0,d){let e=d||new Set;if(b){if(fM(b))e.add(b);else if(fM(b.buffer))e.add(b.buffer);else if(ArrayBuffer.isView(b));else if(c&&"object"==typeof b)for(let d in b)a(b[d],c,e)}return void 0===d?Array.from(e):[]}(a),this.worker.postMessage(a,b)}_getErrorFromErrorEvent(a){let b="Failed to load ";return b+=`worker ${this.name} from ${this.url}. `,a.message&&(b+=`${a.message} in `),a.lineno&&(b+=`:${a.lineno}:${a.colno}`),Error(b)}_createBrowserWorker(){var a,b,c;let d;this._loadableURL=(fJ((a={source:this.source,url:this.url}).source&&!a.url||!a.source&&a.url),(d=fK.get(a.source||a.url))||(a.url&&(d=(b=a.url).startsWith("http")?fL((c=b,`\
try {
  importScripts('${c}');
} catch (error) {
  console.error(error);
  throw error;
}`)):b,fK.set(a.url,d)),a.source&&(d=fL(a.source),fK.set(a.source,d))),fJ(d),d);let e=new Worker(this._loadableURL,{name:this.name});return e.onmessage=a=>{a.data?this.onMessage(a.data):this.onError(Error("No data received"))},e.onerror=a=>{this.onError(this._getErrorFromErrorEvent(a)),this.terminated=!0},e.onmessageerror=a=>console.error(a),e}_createNodeWorker(){let a;if(this.url)a=new fI(this.url.includes(":/")||this.url.startsWith("/")?this.url:`./${this.url}`,{eval:!1,type:this.url.endsWith(".ts")||this.url.endsWith(".mjs")?"module":"commonjs"});else if(this.source)a=new fI(this.source,{eval:!0});else throw Error("no worker");return a.on("message",a=>{this.onMessage(a)}),a.on("error",a=>{this.onError(a)}),a.on("exit",a=>{}),a}}class fP{name;workerThread;isRunning=!0;result;_resolve=()=>{};_reject=()=>{};constructor(a,b){this.name=a,this.workerThread=b,this.result=new Promise((a,b)=>{this._resolve=a,this._reject=b})}postMessage(a,b){this.workerThread.postMessage({source:"loaders.gl",type:a,payload:b})}done(a){fJ(this.isRunning),this.isRunning=!1,this._resolve(a)}error(a){fJ(this.isRunning),this.isRunning=!1,this._reject(a)}}class fQ{name="unnamed";source;url;maxConcurrency=1;maxMobileConcurrency=1;onDebug=()=>{};reuseWorkers=!0;props={};jobQueue=[];idleQueue=[];count=0;isDestroyed=!1;static isSupported(){return fO.isSupported()}constructor(a){this.source=a.source,this.url=a.url,this.setProps(a)}destroy(){this.idleQueue.forEach(a=>a.destroy()),this.isDestroyed=!0}setProps(a){this.props={...this.props,...a},void 0!==a.name&&(this.name=a.name),void 0!==a.maxConcurrency&&(this.maxConcurrency=a.maxConcurrency),void 0!==a.maxMobileConcurrency&&(this.maxMobileConcurrency=a.maxMobileConcurrency),void 0!==a.reuseWorkers&&(this.reuseWorkers=a.reuseWorkers),void 0!==a.onDebug&&(this.onDebug=a.onDebug)}async startJob(a,b=(a,b,c)=>a.done(c),c=(a,b)=>a.error(b)){let d=new Promise(d=>(this.jobQueue.push({name:a,onMessage:b,onError:c,onStart:d}),this));return this._startQueuedJob(),await d}async _startQueuedJob(){if(!this.jobQueue.length)return;let a=this._getAvailableWorker();if(!a)return;let b=this.jobQueue.shift();if(b){this.onDebug({message:"Starting job",name:b.name,workerThread:a,backlog:this.jobQueue.length});let c=new fP(b.name,a);a.onMessage=a=>b.onMessage(c,a.type,a.payload),a.onError=a=>b.onError(c,a),b.onStart(c);try{await c.result}catch(a){console.error(`Worker exception: ${a}`)}finally{this.returnWorkerToQueue(a)}}}returnWorkerToQueue(a){!fF||this.isDestroyed||!this.reuseWorkers||this.count>this._getMaxConcurrency()?(a.destroy(),this.count--):this.idleQueue.push(a),this.isDestroyed||this._startQueuedJob()}_getAvailableWorker(){return this.idleQueue.length>0?this.idleQueue.shift()||null:this.count<this._getMaxConcurrency()?(this.count++,new fO({name:`${this.name.toLowerCase()} (#${this.count} of ${this.maxConcurrency})`,source:this.source,url:this.url})):null}_getMaxConcurrency(){return this.maxConcurrency}}let fR={maxConcurrency:3,maxMobileConcurrency:1,reuseWorkers:!0,onDebug:()=>{}};class fS{props;workerPools=new Map;static _workerFarm;static isSupported(){return fO.isSupported()}static getWorkerFarm(a={}){return fS._workerFarm=fS._workerFarm||new fS({}),fS._workerFarm.setProps(a),fS._workerFarm}constructor(a){this.props={...fR},this.setProps(a),this.workerPools=new Map}destroy(){for(let a of this.workerPools.values())a.destroy();this.workerPools=new Map}setProps(a){for(let b of(this.props={...this.props,...a},this.workerPools.values()))b.setProps(this._getWorkerPoolProps())}getWorkerPool(a){let{name:b,source:c,url:d}=a,e=this.workerPools.get(b);return e||((e=new fQ({name:b,source:c,url:d})).setProps(this._getWorkerPoolProps()),this.workerPools.set(b,e)),e}_getWorkerPoolProps(){return{maxConcurrency:this.props.maxConcurrency,maxMobileConcurrency:this.props.maxMobileConcurrency,reuseWorkers:this.props.reuseWorkers,onDebug:this.props.onDebug}}}async function fT(a,b,c,d,e){let f=a.id,g=function(a,b={}){let c=b[a.id]||{},d=fF?`${a.id}-worker.js`:`${a.id}-worker-node.js`,e=c.workerUrl;if(e||"compression"!==a.id||(e=b.workerUrl),"test"===(b._workerType||b?.core?._workerType)&&(e=fF?`modules/${a.module}/dist/${d}`:`modules/${a.module}/src/workers/${a.id}-worker-node.ts`),!e){let b=a.version;"latest"===b&&(b="latest");let c=b?`@${b}`:"";e=`https://unpkg.com/@loaders.gl/${a.module}${c}/dist/${d}`}return fJ(e),e}(a,c),h=fS.getWorkerFarm(c?.core).getWorkerPool({name:f,url:g});c=JSON.parse(JSON.stringify(c)),d=JSON.parse(JSON.stringify(d||{}));let i=await h.startJob("process-on-worker",fU.bind(null,e));i.postMessage("process",{input:b,options:c,context:d});let j=await i.result;return await j.result}async function fU(a,b,c,d){switch(c){case"done":b.done(d);break;case"error":b.error(Error(d.error));break;case"process":let{id:e,input:f,options:g}=d;try{let c=await a(f,g);b.postMessage("done",{id:e,result:c})}catch(c){let a=c instanceof Error?c.message:"unknown error";b.postMessage("error",{id:e,error:a})}break;default:console.warn(`parse-with-worker unknown message ${c}`)}}let fV=(globalThis._loadersgl_?.version||(globalThis._loadersgl_=globalThis._loadersgl_||{},globalThis._loadersgl_.version="4.4.3"),globalThis._loadersgl_.version);async function fW(a){let b=[];for await(let c of a)b.push(function(a){if(a instanceof ArrayBuffer)return a;if(ArrayBuffer.isView(a)){let{buffer:b,byteOffset:c,byteLength:d}=a;return fX(b,c,d)}return fX(a)}(c));return function(...a){var b=a;let c=b.map(a=>a instanceof ArrayBuffer?new Uint8Array(a):a),d=new Uint8Array(c.reduce((a,b)=>a+b.byteLength,0)),e=0;for(let a of c)d.set(a,e),e+=a.byteLength;return d.buffer}(...b)}function fX(a,b=0,c=a.byteLength-b){let d=new Uint8Array(a,b,c),e=new Uint8Array(d.length);return e.set(d),e.buffer}function fY(a){return a&&"object"==typeof a&&a.isBuffer}function fZ(a){if(fY(a))return Buffer.isBuffer(a)?new Uint8Array(a.buffer,a.byteOffset,a.length).slice().buffer:a;if(a instanceof ArrayBuffer)return a;if(e1(a))return f_(a);if(ArrayBuffer.isView(a)){let b=a.buffer;return 0===a.byteOffset&&a.byteLength===a.buffer.byteLength?b:b.slice(a.byteOffset,a.byteOffset+a.byteLength)}if("string"==typeof a)return new TextEncoder().encode(a).buffer;if(a&&"object"==typeof a&&a._toArrayBuffer)return a._toArrayBuffer();throw Error("toArrayBuffer")}function f$(a){if(a instanceof ArrayBuffer)return a;if(e1(a))return f_(a);let{buffer:b,byteOffset:c,byteLength:d}=a;return b instanceof ArrayBuffer&&0===c&&d===b.byteLength?b:f_(b,c,d)}function f_(a,b=0,c=a.byteLength-b){let d=new Uint8Array(a,b,c),e=new Uint8Array(d.length);return e.set(d),e.buffer}async function*f0(a,b){let c=b?.chunkSize||1048576,d=0;for(;d<a.size;){let b=d+c,e=await a.slice(d,b).arrayBuffer();d=b,yield e}}function f1(a,b){return fs?f2(a,b):f3(a,b)}async function*f2(a,b){let c,d=a.getReader();try{for(;;){let a=c||d.read();b?._streamReadAhead&&(c=d.read());let{done:e,value:f}=await a;if(e)return;yield fZ(f)}}catch(a){d.releaseLock()}}async function*f3(a,b){for await(let b of a)yield fZ(b)}let f4="Cannot convert supplied data type";async function f5(a,b,c){let d,e;if("string"==typeof a||e2(a)){var f,g=a;if(b.text&&"string"==typeof g)return g;if(fY(g)&&(g=g.buffer),e2(g)){let a=ArrayBuffer.isView(f=g)?f:new Uint8Array(f);return b.text&&!b.binary?new TextDecoder("utf8").decode(a):fZ(a)}throw Error(f4)}if(e4(a)&&(a=await fj(a)),e3(a))return await fk(a),b.binary?await a.arrayBuffer():await a.text();if(e5(a)&&(a=function(a,b){if("string"==typeof a)return function*(a,b){let c=b?.chunkSize||262144,d=0,e=new TextEncoder;for(;d<a.length;){let b=Math.min(a.length-d,c),f=a.slice(d,d+b);d+=b,yield f$(e.encode(f))}}(a,b);if(a instanceof ArrayBuffer)return function*(a,b={}){let{chunkSize:c=262144}=b,d=0;for(;d<a.byteLength;){let b=Math.min(a.byteLength-d,c),e=new ArrayBuffer(b),f=new Uint8Array(a,d,b);new Uint8Array(e).set(f),d+=b,yield e}}(a,b);if(e4(a))return f0(a,b);if(e5(a))return f1(a,b);if(e3(a)){let c=a.body;if(!c)throw Error("Readable stream not available on Response");return f1(c,b)}throw Error("makeIterator")}(a,c)),(d=a)&&"function"==typeof d[Symbol.iterator]||(e=a)&&"function"==typeof e[Symbol.asyncIterator])return fW(a);throw Error(f4)}var fo=fo,fo=fo;let f6="4.4.3",f7=f6[0]>="0"&&f6[0]<="9"?`v${f6}`:"",f8=(i=new cr.Log({id:"loaders.gl"}),globalThis.loaders||={},globalThis.loaders.log=i,globalThis.loaders.version=f7,globalThis.probe||={},globalThis.probe.loaders=i,i),f9=()=>{let a=fx();return a.loaderRegistry=a.loaderRegistry||[],a.loaderRegistry},ga=/\.([^.]+)$/;async function gb(a,b=[],c,d){if(!ge(a))return null;let e=fz(c||{});if(e.core||={},a instanceof Response&&gc(a)){let c=gd(await a.clone().text(),b,{...e,core:{...e.core,nothrow:!0}},d);if(c)return c}let f=gd(a,b,{...e,core:{...e.core,nothrow:!0}},d);if(f)return f;if(e4(a)&&(f=gd(a=await a.slice(0,10).arrayBuffer(),b,e,d)),!f&&a instanceof Response&&gc(a)&&(f=gd(await a.clone().text(),b,e,d)),!f&&!e.core.nothrow)throw Error(gf(a));return f}function gc(a){let b=fi(a);return!!(b&&(b.startsWith("text/")||"application/json"===b||b.endsWith("+json")))}function gd(a,b=[],c,d){var e,f,g,h,i,j;let k,l,m,n,o,p,q;if(!ge(a))return null;let r=fz(c||{});if(r.core||={},b&&!Array.isArray(b))return e8(b);let s=[];b&&(s=s.concat(b)),r.core.ignoreRegisteredLoaders||s.push(...f9()),function(a){for(let b of a)e8(b)}(s);let t=(e=a,f=s,g=r,h=d,k=fh(e),l=fi(e),m=fg(k)||h?.url,n=null,o="",g?.core?.mimeType&&(n=gg(f,g?.core?.mimeType),o=`match forced by supplied MIME type ${g?.core?.mimeType}`),n=n||(i=f,(q=(p=(j=m)&&ga.exec(j))&&p[1])?function(a,b){for(let c of(b=b.toLowerCase(),a))for(let a of c.extensions)if(a.toLowerCase()===b)return c;return null}(i,q):null),o=o||(n?`matched url ${m}`:""),n=n||gg(f,l),o=o||(n?`matched MIME type ${l}`:""),n=n||function(a,b){if(!b)return null;for(let c of a)if("string"==typeof b){if(function(a,b){return b.testText?b.testText(a):(Array.isArray(b.tests)?b.tests:[b.tests]).some(b=>a.startsWith(b))}(b,c))return c}else if(ArrayBuffer.isView(b)){if(gh(b.buffer,b.byteOffset,c))return c}else if(b instanceof ArrayBuffer&&gh(b,0,c))return c;return null}(f,e),o=o||(n?`matched initial data ${gi(e)}`:""),g?.core?.fallbackMimeType&&(n=n||gg(f,g?.core?.fallbackMimeType),o=o||(n?`matched fallback MIME type ${l}`:"")),o&&f8.log(1,`selectLoader selected ${n?.name}: ${o}.`),n);if(!t&&!r.core.nothrow)throw Error(gf(a));return t}function ge(a){return!(a instanceof Response)||204!==a.status}function gf(a){let b=fh(a),c=fi(a),d="No valid loader found (";d+=(b?`${fo.filename(b)}, `:"no url provided, ")+`MIME type: ${c?`"${c}"`:"not provided"}, `;let e=a?gi(a):"";return d+((e?` first bytes: "${e}"`:"first bytes: not available")+")")}function gg(a,b){for(let c of a)if(c.mimeTypes?.some(a=>fd(b,a))||fd(b,`application/x.${c.id}`))return c;return null}function gh(a,b,c){return(Array.isArray(c.tests)?c.tests:[c.tests]).some(c=>(function(a,b,c){if(e2(c))return function(a,b,c){if(c=c||a.byteLength,a.byteLength<c||b.byteLength<c)return!1;let d=new Uint8Array(a),e=new Uint8Array(b);for(let a=0;a<d.length;++a)if(d[a]!==e[a])return!1;return!0}(c,a,c.byteLength);switch(typeof c){case"function":return c(f$(a));case"string":let d=gj(a,b,c.length);return c===d;default:return!1}})(a,b,c))}function gi(a,b=5){return"string"==typeof a?a.slice(0,b):ArrayBuffer.isView(a)?gj(a.buffer,a.byteOffset,b):a instanceof ArrayBuffer?gj(a,0,b):""}function gj(a,b,c){if(a.byteLength<b+c)return"";let d=new DataView(a),e="";for(let a=0;a<c;a++)e+=String.fromCharCode(d.getUint8(b+a));return e}async function gk(a,b,c,d){var e,f,g,h,i,j,k;let l,m;!b||Array.isArray(b)||e7(b)||(d=void 0,c=b,b=void 0),a=await a,c=c||{};let n=fh(a),o=function(a,b){let c;if(a&&!Array.isArray(a))return a;if(a&&(c=Array.isArray(a)?a:[a]),b&&b.loaders){let a=Array.isArray(b.loaders)?b.loaders:[b.loaders];c=c?[...c,...a]:a}return c&&c.length?c:void 0}(b,d),p=await gb(a,o,c);if(!p)return null;let q=(function(a,b){for(let c of(fA(a,null,fu,fv,b),b)){let d=a&&a[c.id]||{},e=c.options&&c.options[c.id]||{},f=c.deprecatedOptions&&c.deprecatedOptions[c.id]||{};fA(d,c.id,e,f,b)}}(e=c,f=Array.isArray(f=(f=o)||[])?f:[f]),fz((g=p,h=e,i=n,m={...l=g.options||{}},l.core&&(m.core={...l.core}),fC(m),m.core?.log===null&&(m.core={...m.core,log:new fq}),fB(m,fz(fy())),fB(m,fz(h)),j=m,(k=i)&&j.core?.baseUrl===void 0&&(j.core||={},j.core.baseUrl=fo.dirname(fg(k))),function(a){let b=a.core;if(b)for(let c of fw)void 0!==b[c]&&(a[c]=b[c])}(m),m)));return d=function(a,b,c){if(c)return c;let d={fetch:fD(b,a),...a};if(d.url){let a,b=fg(d.url);d.baseUrl=b,a=d.url.match(ff),d.queryString=a&&a[0],d.filename=fo.filename(b),d.baseUrl=fo.dirname(b)}return Array.isArray(d.loaders)||(d.loaders=null),d}({url:n,_parse:gk,loaders:o},q,d||null),await gl(p,a,q,d)}async function gl(a,b,c,d){if(!function(a,b=fV){fJ(a,"no worker provided");a.version}(a),c=function a(b,c,d=0){if(d>3)return c;let e={...b};for(let[b,f]of Object.entries(c))f&&"object"==typeof f&&!Array.isArray(f)?e[b]=a(e[b]||{},c[b],d+1):e[b]=c[b];return e}(a.options||{},c),e3(b)){let{ok:a,redirected:c,status:e,statusText:f,type:g,url:h}=b;d.response={headers:Object.fromEntries(b.headers.entries()),ok:a,redirected:c,status:e,statusText:f,type:g,url:h}}if(b=await f5(b,a,c),a.parseTextSync&&"string"==typeof b)return a.parseTextSync(b,c,d);if(function(a,b){if(!fS.isSupported())return!1;let c=b?._nodeWorkers??b?.core?._nodeWorkers;if(!fF&&!c)return!1;let d=b?.worker??b?.core?.worker;return!!(a.worker&&d)}(a,c))return await fT(a,b,c,d,gk);if(a.parseText&&"string"==typeof b)return await a.parseText(b,c,d);if(a.parse)return await a.parse(b,c,d);throw fJ(!a.parseSync),Error(`${a.id} loader - no parser found and worker is disabled`)}async function gm(a,b,c,d){let e,f;Array.isArray(b)||e7(b)?(e=b,f=c):(e=[],f=b);let g=fD(f),h=a;if("string"==typeof a&&(h=await g(a)),e4(a)&&(h=await g(a)),"string"==typeof a){let b=fz(f||{});b.core?.baseUrl||(f={...f,core:{...f?.core,baseUrl:a}})}return Array.isArray(e),await gk(h,e,f)}class gn{constructor(a,b,c){this._loadCount=0,this._subscribers=new Set,this.id=a,this.context=c,this.setData(b)}subscribe(a){this._subscribers.add(a)}unsubscribe(a){this._subscribers.delete(a)}inUse(){return this._subscribers.size>0}delete(){}getData(){return this.isLoaded?this._error?Promise.reject(this._error):this._content:this._loader.then(()=>this.getData())}setData(a,b){if(a===this._data&&!b)return;this._data=a;let c=++this._loadCount,d=a;for(let b of("string"==typeof a&&(d=gm(a)),d instanceof Promise?(this.isLoaded=!1,this._loader=d.then(a=>{this._loadCount===c&&(this.isLoaded=!0,this._error=void 0,this._content=a)}).catch(a=>{this._loadCount===c&&(this.isLoaded=!0,this._error=a||!0)})):(this.isLoaded=!0,this._error=void 0,this._content=a),this._subscribers))b.onChange(this.getData())}}class go{constructor(a){this.protocol=a.protocol||"resource://",this._context={device:a.device,gl:a.device?.gl,resourceManager:this},this._resources={},this._consumers={},this._pruneRequest=null}contains(a){return!!a.startsWith(this.protocol)||a in this._resources}add({resourceId:a,data:b,forceUpdate:c=!1,persistent:d=!0}){let e=this._resources[a];e?e.setData(b,c):(e=new gn(a,b,this._context),this._resources[a]=e),e.persistent=d}remove(a){let b=this._resources[a];b&&(b.delete(),delete this._resources[a])}unsubscribe({consumerId:a}){let b=this._consumers[a];if(b){for(let a in b){let c=b[a],d=this._resources[c.resourceId];d&&d.unsubscribe(c)}delete this._consumers[a],this.prune()}}subscribe({resourceId:a,onChange:b,consumerId:c,requestId:d="default"}){let{_resources:e,protocol:f}=this;a.startsWith(f)&&(e[a=a.replace(f,"")]||this.add({resourceId:a,data:null,persistent:!1}));let g=e[a];if(this._track(c,d,g,b),g)return g.getData()}prune(){this._pruneRequest||(this._pruneRequest=setTimeout(()=>this._prune(),0))}finalize(){for(let a in this._resources)this._resources[a].delete()}_track(a,b,c,d){let e=this._consumers,f=e[a]=e[a]||{},g=f[b],h=g&&g.resourceId&&this._resources[g.resourceId];h&&(h.unsubscribe(g),this.prune()),c&&(g?(g.onChange=d,g.resourceId=c.id):g={onChange:d,resourceId:c.id},f[b]=g,c.subscribe(g))}_prune(){for(let a of(this._pruneRequest=null,Object.keys(this._resources))){let b=this._resources[a];b.persistent||b.inUse()||(b.delete(),delete this._resources[a])}}}let gp=new class{constructor(a={}){this._pool=[],this.opts={overAlloc:2,poolSize:100},this.setOptions(a)}setOptions(a){Object.assign(this.opts,a)}allocate(a,b,{size:c=1,type:d,padding:e=0,copy:f=!1,initialize:g=!1,maxCount:h}){let i=d||a&&a.constructor||Float32Array,j=b*c+e;if(ArrayBuffer.isView(a)){if(j<=a.length)return a;if(j*a.BYTES_PER_ELEMENT<=a.buffer.byteLength)return new i(a.buffer,0,j)}let k=1/0;h&&(k=h*c+e);let l=this._allocate(i,j,g,k);return a&&f?l.set(a):g||l.fill(0,0,4),this._release(a),l}release(a){this._release(a)}_allocate(a,b,c,d){let e=Math.max(Math.ceil(b*this.opts.overAlloc),1);e>d&&(e=d);let f=this._pool,g=a.BYTES_PER_ELEMENT*e,h=f.findIndex(a=>a.byteLength>=g);if(h>=0){let b=new a(f.splice(h,1)[0],0,e);return c&&b.fill(0),b}return new a(e)}_release(a){if(!ArrayBuffer.isView(a))return;let b=this._pool,{buffer:c}=a,{byteLength:d}=c,e=b.findIndex(a=>a.byteLength>=d);e<0?b.push(c):(e>0||b.length<this.opts.poolSize)&&b.splice(e,0,c),b.length>this.opts.poolSize&&b.shift()}};function gq(){return[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1]}function gr(a,b){let c=a%b;return c<0?b+c:c}let gs=new d1;function gt(a,b,c,d){gs.set(a,b,c);let e=gs.len();return{distance:d/e,normal:new d1(-a/e,-b/e,-c/e)}}function gu(a,b){let{size:c=1,startIndex:d=0}=b,f=void 0!==b.endIndex?b.endIndex:a.length,g=(f-d)/c;e=gp.allocate(e,g,{type:Float32Array,size:2*c});let h=d,i=0;for(;h<f;){for(let b=0;b<c;b++){let d=a[h++];e[i+b]=d,e[i+b+c]=d-Math.fround(d)}i+=2*c}return e.subarray(0,g*c*2)}function gv(a){let b=null,c=!1;for(let d of a)d&&(b?(c||(b=[[b[0][0],b[0][1]],[b[1][0],b[1][1]]],c=!0),b[0][0]=Math.min(b[0][0],d[0][0]),b[0][1]=Math.min(b[0][1],d[0][1]),b[1][0]=Math.max(b[1][0],d[1][0]),b[1][1]=Math.max(b[1][1],d[1][1])):b=d);return b}var cf=cf;let gw=Math.PI/180,gx=gq(),gy=[0,0,0],gz={unitsPerMeter:[1,1,1],metersPerUnit:[1,1,1]};class gA{constructor(a={}){this._frustumPlanes={},this.id=a.id||this.constructor.displayName||"viewport",this.x=a.x||0,this.y=a.y||0,this.width=a.width||1,this.height=a.height||1,this.zoom=a.zoom||0,this.padding=a.padding,this.distanceScales=a.distanceScales||gz,this.focalDistance=a.focalDistance||1,this.position=a.position||gy,this.modelMatrix=a.modelMatrix||null;const{longitude:b,latitude:c}=a;this.isGeospatial=Number.isFinite(c)&&Number.isFinite(b),this._initProps(a),this._initMatrices(a),this.equals=this.equals.bind(this),this.project=this.project.bind(this),this.unproject=this.unproject.bind(this),this.projectPosition=this.projectPosition.bind(this),this.unprojectPosition=this.unprojectPosition.bind(this),this.projectFlat=this.projectFlat.bind(this),this.unprojectFlat=this.unprojectFlat.bind(this)}get subViewports(){return null}get metersPerPixel(){return this.distanceScales.metersPerUnit[2]/this.scale}get projectionMode(){return this.isGeospatial?this.zoom<12?c9.WEB_MERCATOR:c9.WEB_MERCATOR_AUTO_OFFSET:c9.IDENTITY}equals(a){return a instanceof gA&&(this===a||a.width===this.width&&a.height===this.height&&a.scale===this.scale&&dD(a.projectionMatrix,this.projectionMatrix)&&dD(a.viewMatrix,this.viewMatrix))}project(a,{topLeft:b=!0}={}){let c=ey(this.projectPosition(a),this.pixelProjectionMatrix),[d,e]=c,f=b?e:this.height-e;return 2===a.length?[d,f]:[d,f,c[2]]}unproject(a,{topLeft:b=!0,targetZ:c}={}){let[d,e,f]=a,g=b?e:this.height-e,h=c&&c*this.distanceScales.unitsPerMeter[2],i=ez([d,g,f],this.pixelUnprojectionMatrix,h),[j,k,l]=this.unprojectPosition(i);return Number.isFinite(f)?[j,k,l]:Number.isFinite(c)?[j,k,c]:[j,k]}projectPosition(a){let[b,c]=this.projectFlat(a);return[b,c,(a[2]||0)*this.distanceScales.unitsPerMeter[2]]}unprojectPosition(a){let[b,c]=this.unprojectFlat(a);return[b,c,(a[2]||0)*this.distanceScales.metersPerUnit[2]]}projectFlat(a){if(this.isGeospatial){let b=er(a);return b[1]=dB(b[1],-318,830),b}return a}unprojectFlat(a){return this.isGeospatial?es(a):a}getBounds(a={}){let b={targetZ:a.z||0},c=this.unproject([0,0],b),d=this.unproject([this.width,0],b),e=this.unproject([0,this.height],b),f=this.unproject([this.width,this.height],b);return[Math.min(c[0],d[0],e[0],f[0]),Math.min(c[1],d[1],e[1],f[1]),Math.max(c[0],d[0],e[0],f[0]),Math.max(c[1],d[1],e[1],f[1])]}getDistanceScales(a){return a&&this.isGeospatial?eu({longitude:a[0],latitude:a[1],highPrecision:!0}):this.distanceScales}containsPixel({x:a,y:b,width:c=1,height:d=1}){return a<this.x+this.width&&this.x<a+c&&b<this.y+this.height&&this.y<b+d}getFrustumPlanes(){var a;return this._frustumPlanes.near?this._frustumPlanes:(Object.assign(this._frustumPlanes,{left:gt((a=this.viewProjectionMatrix)[3]+a[0],a[7]+a[4],a[11]+a[8],a[15]+a[12]),right:gt(a[3]-a[0],a[7]-a[4],a[11]-a[8],a[15]-a[12]),bottom:gt(a[3]+a[1],a[7]+a[5],a[11]+a[9],a[15]+a[13]),top:gt(a[3]-a[1],a[7]-a[5],a[11]-a[9],a[15]-a[13]),near:gt(a[3]+a[2],a[7]+a[6],a[11]+a[10],a[15]+a[14]),far:gt(a[3]-a[2],a[7]-a[6],a[11]-a[10],a[15]-a[14])}),this._frustumPlanes)}panByPosition(a,b,c){return null}_initProps(a){let b=a.longitude,c=a.latitude;this.isGeospatial&&(Number.isFinite(a.zoom)||(this.zoom=function(a){let{latitude:b}=a;return em(Number.isFinite(b)),ej(4003e4*Math.cos(b*ep))-9}({latitude:c})+Math.log2(this.focalDistance)),this.distanceScales=a.distanceScales||eu({latitude:c,longitude:b}));let d=Math.pow(2,this.zoom);this.scale=d;let{position:e,modelMatrix:f}=a,g=gy;if(e&&(g=f?new ef(f).transformAsVector(e,[]):e),this.isGeospatial){let a=this.projectPosition([b,c,0]);this.center=new d1(g).scale(this.distanceScales.unitsPerMeter).add(a)}else this.center=this.projectPosition(g)}_initMatrices(a){var b;let{viewMatrix:c=gx,projectionMatrix:d=null,orthographic:e=!1,fovyRadians:f,fovy:g=75,near:h=.1,far:i=1e3,padding:j=null,focalDistance:k=1}=a;this.viewMatrixUncentered=c,this.viewMatrix=new ef().multiplyRight(c).translate(new d1(this.center).negate()),this.projectionMatrix=d||function({width:a,height:b,orthographic:c,fovyRadians:d,focalDistance:e,padding:f,near:g,far:h}){let i=a/b,j=c?new ef().orthographic({fovy:d,aspect:i,focalDistance:e,near:g,far:h}):new ef().perspective({fovy:d,aspect:i,near:g,far:h});if(f){let{left:c=0,right:d=0,top:e=0,bottom:g=0}=f,h=dB((c+a-d)/2,0,a)-a/2,i=dB((e+b-g)/2,0,b)-b/2;j[8]-=2*h/a,j[9]+=2*i/b}return j}({width:this.width,height:this.height,orthographic:e,fovyRadians:f||g*gw,focalDistance:k,padding:j,near:h,far:i});let l=gq();cf.multiply(l,l,this.projectionMatrix),cf.multiply(l,l,this.viewMatrix),this.viewProjectionMatrix=l,this.viewMatrixInverse=cf.invert([],this.viewMatrix)||this.viewMatrix,this.cameraPosition=[(b=this.viewMatrixInverse)[12],b[13],b[14]];let m=gq(),n=gq();cf.scale(m,m,[this.width/2,-this.height/2,1]),cf.translate(m,m,[1,-1,0]),cf.multiply(n,m,this.viewProjectionMatrix),this.pixelProjectionMatrix=n,this.pixelUnprojectionMatrix=cf.invert(gq(),this.pixelProjectionMatrix),this.pixelUnprojectionMatrix||cs.warn("Pixel project matrix not invertible")()}}gA.displayName="Viewport";let gB=gA;class gC{constructor(a,b){this._lastRenderedLayers=[],this._needsRedraw=!1,this._needsUpdate=!1,this._nextLayers=null,this._debug=!1,this._defaultShaderModulesChanged=!1,this.activateViewport=a=>{eY("layerManager.activateViewport",this,a),a&&(this.context.viewport=a)};const{deck:c,stats:d,viewport:e,timeline:f}=b||{};this.layers=[],this.resourceManager=new go({device:a,protocol:"deck://"}),this.context={mousePosition:null,userData:{},layerManager:this,device:a,gl:a?.gl,deck:c,shaderAssembler:function(a){let b=by.getDefaultShaderAssembler();for(let a of eN)b.addDefaultModule(a);for(let c of(b._hookFunctions.length=0,"glsl"===a?eO:eP))b.addShaderHook(c);return b}(a?.info?.shadingLanguage||"glsl"),defaultShaderModules:[bQ],renderPass:void 0,stats:d||new e$.Stats({id:"deck.gl"}),viewport:e||new gB({id:"DEFAULT-INITIAL-VIEWPORT"}),timeline:f||new aH,resourceManager:this.resourceManager,onError:void 0},Object.seal(this)}finalize(){for(let a of(this.resourceManager.finalize(),this.layers))this._finalizeLayer(a)}needsRedraw(a={clearRedrawFlags:!1}){let b=this._needsRedraw;for(let c of(a.clearRedrawFlags&&(this._needsRedraw=!1),this.layers)){let d=c.getNeedsRedraw(a);b=b||d}return b}needsUpdate(){return this._nextLayers&&this._nextLayers!==this._lastRenderedLayers?"layers changed":this._defaultShaderModulesChanged?"shader modules changed":this._needsUpdate}setNeedsRedraw(a){this._needsRedraw=this._needsRedraw||a}setNeedsUpdate(a){this._needsUpdate=this._needsUpdate||a}getLayers({layerIds:a}={}){return a?this.layers.filter(b=>a.find(a=>0===b.id.indexOf(a))):this.layers}setProps(a){"debug"in a&&(this._debug=a.debug),"userData"in a&&(this.context.userData=a.userData),"layers"in a&&(this._nextLayers=a.layers),"onError"in a&&(this.context.onError=a.onError)}setLayers(a,b){eY("layerManager.setLayers",this,b,a),this._lastRenderedLayers=a;let c=eZ(a,Boolean);for(let a of c)a.context=this.context;this._updateLayers(this.layers,c)}updateLayers(){let a=this.needsUpdate();a&&(this.setNeedsRedraw(`updating layers: ${a}`),this.setLayers(this._nextLayers||this._lastRenderedLayers,a)),this._nextLayers=null}addDefaultShaderModule(a){let{defaultShaderModules:b}=this.context;b.find(b=>b.name===a.name)||(b.push(a),this._defaultShaderModulesChanged=!0)}removeDefaultShaderModule(a){let{defaultShaderModules:b}=this.context,c=b.findIndex(b=>b.name===a.name);c>=0&&(b.splice(c,1),this._defaultShaderModulesChanged=!0)}_handleError(a,b,c){c.raiseError(b,`${a} of ${c}`)}_updateLayers(a,b){let c={};for(let b of a)c[b.id]?cs.warn(`Multiple old layers with same id ${b.id}`)():c[b.id]=b;if(this._defaultShaderModulesChanged){for(let b of a)b.setNeedsUpdate(),b.setChangeFlags({extensionsChanged:!0});this._defaultShaderModulesChanged=!1}let d=[];this._updateSublayersRecursively(b,c,d),this._finalizeOldLayers(c);let e=!1;for(let a of d)if(a.hasUniformTransition()){e=`Uniform transition in ${a}`;break}this._needsUpdate=e,this.layers=d}_updateSublayersRecursively(a,b,c){for(let d of a){d.context=this.context;let a=b[d.id];null===a&&cs.warn(`Multiple new layers with same id ${d.id}`)(),b[d.id]=null;let e=null;try{this._debug&&a!==d&&d.validateProps(),a?(this._transferLayerState(a,d),this._updateLayer(d)):this._initializeLayer(d),c.push(d),e=d.isComposite?d.getSubLayers():null}catch(a){this._handleError("matching",a,d)}e&&this._updateSublayersRecursively(e,b,c)}}_finalizeOldLayers(a){for(let b in a){let c=a[b];c&&this._finalizeLayer(c)}}_initializeLayer(a){try{a._initialize(),a.lifecycle="Initialized"}catch(b){this._handleError("initialization",b,a)}}_transferLayerState(a,b){b._transferState(a),b.lifecycle="Matched. State transferred from previous layer",b!==a&&(a.lifecycle="Discarded. Awaiting garbage collection")}_updateLayer(a){try{a._update()}catch(b){this._handleError("update",b,a)}}_finalizeLayer(a){this._needsRedraw=this._needsRedraw||`finalized ${a}`,a.lifecycle="No longer matched. Awaiting garbage collection";try{a._finalize(),a.lifecycle="Finalized! Awaiting garbage collection"}catch(b){this._handleError("finalization",b,a)}}}function gD(a,b,c){if(a===b)return!0;if(!c||!a||!b)return!1;if(Array.isArray(a)){if(!Array.isArray(b)||a.length!==b.length)return!1;for(let d=0;d<a.length;d++)if(!gD(a[d],b[d],c-1))return!1;return!0}if(Array.isArray(b))return!1;if("object"==typeof a&&"object"==typeof b){let d=Object.keys(a),e=Object.keys(b);if(d.length!==e.length)return!1;for(let e of d)if(!b.hasOwnProperty(e)||!gD(a[e],b[e],c-1))return!1;return!0}return!1}class gE{constructor(a){this.views=[],this.width=100,this.height=100,this.viewState={},this.controllers={},this.timeline=a.timeline,this._viewports=[],this._viewportMap={},this._isUpdating=!1,this._needsRedraw="First render",this._needsUpdate="Initialize",this._eventManager=a.eventManager,this._eventCallbacks={onViewStateChange:a.onViewStateChange,onInteractionStateChange:a.onInteractionStateChange},this._pickPosition=a.pickPosition,Object.seal(this),this.setProps(a)}finalize(){for(let a in this.controllers){let b=this.controllers[a];b&&b.finalize()}this.controllers={}}needsRedraw(a={clearRedrawFlags:!1}){let b=this._needsRedraw;return a.clearRedrawFlags&&(this._needsRedraw=!1),b}setNeedsUpdate(a){this._needsUpdate=this._needsUpdate||a,this._needsRedraw=this._needsRedraw||a}updateViewStates(){for(let a in this.controllers){let b=this.controllers[a];b&&b.updateTransition()}}getViewports(a){return a?this._viewports.filter(b=>b.containsPixel(a)):this._viewports}getViews(){let a={};return this.views.forEach(b=>{a[b.id]=b}),a}getView(a){return this.views.find(b=>b.id===a)}getViewState(a){let b="string"==typeof a?this.getView(a):a,c=b&&this.viewState[b.getViewStateId()]||this.viewState;return b?b.filterViewState(c):c}getViewport(a){return this._viewportMap[a]}unproject(a,b){let c=this.getViewports(),d={x:a[0],y:a[1]};for(let e=c.length-1;e>=0;--e){let f=c[e];if(f.containsPixel(d)){let c=a.slice();return c[0]-=f.x,c[1]-=f.y,f.unproject(c,b)}}return null}setProps(a){a.views&&this._setViews(a.views),a.viewState&&this._setViewState(a.viewState),("width"in a||"height"in a)&&this._setSize(a.width,a.height),"pickPosition"in a&&(this._pickPosition=a.pickPosition),this._isUpdating||this._update()}_update(){this._isUpdating=!0,this._needsUpdate&&(this._needsUpdate=!1,this._rebuildViewports()),this._needsUpdate&&(this._needsUpdate=!1,this._rebuildViewports()),this._isUpdating=!1}_setSize(a,b){(a!==this.width||b!==this.height)&&(this.width=a,this.height=b,this.setNeedsUpdate("Size changed"))}_setViews(a){a=eZ(a,Boolean),this._diffViews(a,this.views)&&this.setNeedsUpdate("views changed"),this.views=a}_setViewState(a){a?(gD(a,this.viewState,3)||this.setNeedsUpdate("viewState changed"),this.viewState=a):cs.warn("missing `viewState` or `initialViewState`")()}_createController(a,b){return new b.type({timeline:this.timeline,eventManager:this._eventManager,onViewStateChange:this._eventCallbacks.onViewStateChange,onStateChange:this._eventCallbacks.onInteractionStateChange,makeViewport:b=>this.getView(a.id)?.makeViewport({viewState:b,width:this.width,height:this.height}),pickPosition:this._pickPosition})}_updateController(a,b,c,d){let e=a.controller;if(e&&c){let f={...b,...e,id:a.id,x:c.x,y:c.y,width:c.width,height:c.height};return d&&d.constructor===e.type||(d=this._createController(a,f)),d&&d.setProps(f),d}return null}_rebuildViewports(){let{views:a}=this,b=this.controllers;this._viewports=[],this.controllers={};let c=!1;for(let d=a.length;d--;){let e=a[d],f=this.getViewState(e),g=e.makeViewport({viewState:f,width:this.width,height:this.height}),h=b[e.id],i=!!e.controller;i&&!h&&(c=!0),(c||!i)&&h&&(h.finalize(),h=null),this.controllers[e.id]=this._updateController(e,f,g,h),g&&this._viewports.unshift(g)}for(let a in b){let c=b[a];c&&!this.controllers[a]&&c.finalize()}this._buildViewportMap()}_buildViewportMap(){this._viewportMap={},this._viewports.forEach(a=>{a.id&&(this._viewportMap[a.id]=this._viewportMap[a.id]||a)})}_diffViews(a,b){return a.length!==b.length||a.some((c,d)=>!a[d].equals(b[d]))}}let gF=/^(?:\d+\.?\d*|\.\d+)$/;function gG(a){switch(typeof a){case"number":if(!Number.isFinite(a))throw Error(`Could not parse position string ${a}`);return{type:"literal",value:a};case"string":try{let b=function(a){let b=[],c=0;for(;c<a.length;){let d=a[c];if(/\s/.test(d)){c++;continue}if("+"===d||"-"===d||"("===d||")"===d||"%"===d){b.push({type:"symbol",value:d}),c++;continue}if(gJ(d)||"."===d){let e=c,f="."===d;for(c++;c<a.length;){let b=a[c];if(gJ(b)){c++;continue}if("."===b&&!f){f=!0,c++;continue}break}let g=a.slice(e,c);if(!gF.test(g))throw Error("Invalid number token");b.push({type:"number",value:parseFloat(g)});continue}if(gK(d)){let d=c;for(;c<a.length&&gK(a[c]);)c++;let e=a.slice(d,c).toLowerCase();b.push({type:"word",value:e});continue}throw Error("Invalid token in position string")}return b}(a);return new gI(b).parseExpression()}catch(c){let b=c instanceof Error?c.message:String(c);throw Error(`Could not parse position string ${a}: ${b}`)}default:throw Error(`Could not parse position string ${a}`)}}function gH(a,b){return function a(b,c){switch(b.type){case"literal":return b.value;case"percentage":return Math.round(b.value*c);case"binary":let d=a(b.left,c),e=a(b.right,c);return"+"===b.operator?d+e:d-e;default:throw Error("Unknown layout expression type")}}(a,b)}class gI{constructor(a){this.index=0,this.tokens=a}parseExpression(){let a=this.parseBinaryExpression();if(this.index<this.tokens.length)throw Error("Unexpected token at end of expression");return a}parseBinaryExpression(){var a;let b=this.parseFactor(),c=this.peek();for(;(a=c)&&"symbol"===a.type&&("+"===a.value||"-"===a.value);){this.index++;let a=this.parseFactor();b={type:"binary",operator:c.value,left:b,right:a},c=this.peek()}return b}parseFactor(){let a=this.peek();if(!a)throw Error("Unexpected end of expression");if("symbol"===a.type&&"+"===a.value)return this.index++,this.parseFactor();if("symbol"===a.type&&"-"===a.value)return this.index++,{type:"binary",operator:"-",left:{type:"literal",value:0},right:this.parseFactor()};if("symbol"===a.type&&"("===a.value){this.index++;let a=this.parseBinaryExpression();if(!this.consumeSymbol(")"))throw Error("Missing closing parenthesis");return a}if("word"===a.type&&"calc"===a.value){if(this.index++,!this.consumeSymbol("("))throw Error("Missing opening parenthesis after calc");let a=this.parseBinaryExpression();if(!this.consumeSymbol(")"))throw Error("Missing closing parenthesis");return a}if("number"===a.type){this.index++;let b=a.value,c=this.peek();return c&&"symbol"===c.type&&"%"===c.value?(this.index++,{type:"percentage",value:b/100}):(c&&"word"===c.type&&"px"===c.value&&this.index++,{type:"literal",value:b})}throw Error("Unexpected token in expression")}consumeSymbol(a){let b=this.peek();return!!b&&"symbol"===b.type&&b.value===a&&(this.index++,!0)}peek(){return this.tokens[this.index]||null}}function gJ(a){return a>="0"&&a<="9"}function gK(a){return a>="a"&&a<="z"||a>="A"&&a<="Z"}class gL{constructor(a){const{id:b,x:c=0,y:d=0,width:e="100%",height:f="100%",padding:g=null}=a;this.id=b||this.constructor.displayName||"view",this.props={...a,id:this.id},this._x=gG(c),this._y=gG(d),this._width=gG(e),this._height=gG(f),this._padding=g&&{left:gG(g.left||0),right:gG(g.right||0),top:gG(g.top||0),bottom:gG(g.bottom||0)},this.equals=this.equals.bind(this),Object.seal(this)}equals(a){return this===a||this.constructor===a.constructor&&gD(this.props,a.props,2)}clone(a){return new this.constructor({...this.props,...a})}makeViewport({width:a,height:b,viewState:c}){c=this.filterViewState(c);let d=this.getDimensions({width:a,height:b});return d.height&&d.width?new(this.getViewportType(c))({...c,...this.props,...d}):null}getViewStateId(){let{viewState:a}=this.props;return"string"==typeof a?a:a?.id||this.id}filterViewState(a){if(this.props.viewState&&"object"==typeof this.props.viewState){if(!this.props.viewState.id)return this.props.viewState;var b=this.props.viewState;let c={...a};for(let a in b)"id"!==a&&(Array.isArray(c[a])&&Array.isArray(b[a])?c[a]=function(a,b){a=a.slice();for(let c=0;c<b.length;c++){let d=b[c];Number.isFinite(d)&&(a[c]=d)}return a}(c[a],b[a]):c[a]=b[a]);return c}return a}getDimensions({width:a,height:b}){let c={x:gH(this._x,a),y:gH(this._y,b),width:gH(this._width,a),height:gH(this._height,b)};return this._padding&&(c.padding={left:gH(this._padding.left,a),top:gH(this._padding.top,b),right:gH(this._padding.right,a),bottom:gH(this._padding.bottom,b)}),c}get controller(){let a=this.props.controller;return a?!0===a?{type:this.ControllerType}:"function"==typeof a?{type:a}:{type:this.ControllerType,...a}:null}}var ek=ek;let gM=Math.PI/180;function gN(a,b,c){let{pixelUnprojectionMatrix:d}=a,e=eh(d,[b,0,1,1]),f=eh(d,[b,a.height,1,1]),g=(c*a.distanceScales.unitsPerMeter[2]-e[2])/(f[2]-e[2]),h=es(ek.lerp([],e,f,g));return h.push(c),h}var ek=ek;class gO extends gB{constructor(a={}){let b;const{latitude:c=0,longitude:d=0,zoom:e=0,pitch:f=0,bearing:g=0,nearZMultiplier:h=.1,farZMultiplier:i=1.01,nearZ:j,farZ:k,orthographic:l=!1,projectionMatrix:m,repeat:n=!1,worldOffset:o=0,position:p,padding:q,legacyMeterSizes:r=!1}=a;let{width:s,height:t,altitude:u=1.5}=a;const v=Math.pow(2,e);s=s||1,t=t||1;let w=null;if(m)b=ew(u=m[5]/2);else{let d;if(a.fovy?u=ex(b=a.fovy):b=ew(u),q){const{top:a=0,bottom:b=0}=q;d=[0,dB((a+t-b)/2,0,t)-t/2]}w=function(a){let{width:b,height:c,altitude:d,pitch:e=0,offset:f,center:g,scale:h,nearZMultiplier:i=1,farZMultiplier:j=1}=a,{fovy:k=ew(1.5)}=a;void 0!==d&&(k=ew(d));let l=k*ep,m=e*ep,n=ex(k),o=n;g&&(o+=g[2]*h/Math.cos(m)/c);let p=l*(.5+(f?f[1]:0)/c),q=Math.sin(p)*o/Math.sin(ei(Math.PI/2-m-p,.01,Math.PI-.01));return{fov:l,aspect:b/c,focalDistance:n,near:i,far:Math.min((Math.sin(m)*q+o)*j,10*o)}}({width:s,height:t,scale:v,center:p&&[0,0,p[2]*et(c)],offset:d,pitch:f,fovy:b,nearZMultiplier:h,farZMultiplier:i}),Number.isFinite(j)&&(w.near=j),Number.isFinite(k)&&(w.far=k)}let x=function(a){let{height:b,pitch:c,bearing:d,altitude:e,scale:f,center:g}=a,h=[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1];cf.translate(h,h,[0,0,-e]),cf.rotateX(h,h,-c*ep),cf.rotateZ(h,h,d*ep);let i=f/b;return cf.scale(h,h,[i,i,i]),g&&cf.translate(h,h,el.negate([],g)),h}({height:t,pitch:f,bearing:g,scale:v,altitude:u});o&&(x=new ef().translate([512*o,0,0]).multiplyLeft(x)),super({...a,width:s,height:t,viewMatrix:x,longitude:d,latitude:c,zoom:e,...w,fovy:b,focalDistance:u}),this.latitude=c,this.longitude=d,this.zoom=e,this.pitch=f,this.bearing=g,this.altitude=u,this.fovy=b,this.orthographic=l,this._subViewports=n?[]:null,this._pseudoMeters=r,Object.freeze(this)}get subViewports(){if(this._subViewports&&!this._subViewports.length){let a=this.getBounds(),b=Math.floor((a[0]+180)/360),c=Math.ceil((a[2]-180)/360);for(let a=b;a<=c;a++){let b=a?new gO({...this,worldOffset:a}):this;this._subViewports.push(b)}}return this._subViewports}projectPosition(a){if(this._pseudoMeters)return super.projectPosition(a);let[b,c]=this.projectFlat(a);return[b,c,(a[2]||0)*et(a[1])]}unprojectPosition(a){if(this._pseudoMeters)return super.unprojectPosition(a);let[b,c]=this.unprojectFlat(a),d=(a[2]||0)/et(c);return[b,c,d]}addMetersToLngLat(a,b){return ev(a,b)}panByPosition(a,b,c){let d=ez(b,this.pixelUnprojectionMatrix),e=this.projectFlat(a),f=ek.add([],e,ek.negate([],d)),g=ek.add([],this.center,f),[h,i]=this.unprojectFlat(g);return{longitude:h,latitude:i}}panByPosition3D(a,b){let c=a[2]||0,d=ek.sub([],a,this.unproject(b,{targetZ:c}));return{longitude:this.longitude+d[0],latitude:this.latitude+d[1]}}getBounds(a={}){let b=function(a,b=0){let c,d,{width:e,height:f,unproject:g}=a,h={targetZ:b},i=g([0,f],h),j=g([e,f],h);return(a.fovy?.5*a.fovy*gM:Math.atan(.5/a.altitude))>(90-a.pitch)*gM-.01?(c=gN(a,0,b),d=gN(a,e,b)):(c=g([0,0],h),d=g([e,0],h)),[i,j,d,c]}(this,a.z||0);return[Math.min(b[0][0],b[1][0],b[2][0],b[3][0]),Math.min(b[0][1],b[1][1],b[2][1],b[3][1]),Math.max(b[0][0],b[1][0],b[2][0],b[3][0]),Math.max(b[0][1],b[1][1],b[2][1],b[3][1])]}fitBounds(a,b={}){let{width:c,height:d}=this,{longitude:e,latitude:f,zoom:g}=function(a){let{width:b,height:c,bounds:d,minExtent:e=0,maxZoom:f=24,offset:g=[0,0]}=a,[[h,i],[j,k]]=d,l=function(a=0){return"number"==typeof a?{top:a,bottom:a,left:a,right:a}:(em(Number.isFinite(a.top)&&Number.isFinite(a.bottom)&&Number.isFinite(a.left)&&Number.isFinite(a.right)),a)}(a.padding),m=er([h,ei(k,-85.051129,85.051129)]),n=er([j,ei(i,-85.051129,85.051129)]),o=[Math.max(Math.abs(n[0]-m[0]),e),Math.max(Math.abs(n[1]-m[1]),e)],p=[b-l.left-l.right-2*Math.abs(g[0]),c-l.top-l.bottom-2*Math.abs(g[1])];em(p[0]>0&&p[1]>0);let q=p[0]/o[0],r=p[1]/o[1],s=(l.right-l.left)/2/q,t=(l.top-l.bottom)/2/r,u=es([(n[0]+m[0])/2+s,(n[1]+m[1])/2+t]),v=Math.min(f,ej(Math.abs(Math.min(q,r))));return em(Number.isFinite(v)),{longitude:u[0],latitude:u[1],zoom:v}}({width:c,height:d,bounds:a,...b});return new gO({width:c,height:d,longitude:e,latitude:f,zoom:g})}}gO.displayName="WebMercatorViewport";class gP{constructor(a){this._inProgress=!1,this._handle=null,this.time=0,this.settings={duration:0},this._timeline=a}get inProgress(){return this._inProgress}start(a){this.cancel(),this.settings=a,this._inProgress=!0,this.settings.onStart?.(this)}end(){this._inProgress&&(this._timeline.removeChannel(this._handle),this._handle=null,this._inProgress=!1,this.settings.onEnd?.(this))}cancel(){this._inProgress&&(this.settings.onInterrupt?.(this),this._timeline.removeChannel(this._handle),this._handle=null,this._inProgress=!1)}update(){if(!this._inProgress)return!1;if(null===this._handle){let{_timeline:a,settings:b}=this;this._handle=a.addChannel({delay:a.getTime(),duration:b.duration})}return this.time=this._timeline.getTime(this._handle),this._onUpdate(),this.settings.onUpdate?.(this),this._timeline.isFinished(this._handle)&&this.end(),!0}_onUpdate(){}}let gQ=()=>{},gR=a=>a;class gS{constructor(a){this._onTransitionUpdate=a=>{let{time:b,settings:{interpolator:c,startProps:d,endProps:e,duration:f,easing:g}}=a,h=g(b/f),i=c.interpolateProps(d,e,h);this.propsInTransition=this.getControllerState({...this.props,...i}).getViewportProps(),this.onViewStateChange({viewState:this.propsInTransition,oldViewState:this.props})},this.getControllerState=a.getControllerState,this.propsInTransition=null,this.transition=new gP(a.timeline),this.onViewStateChange=a.onViewStateChange||gQ,this.onStateChange=a.onStateChange||gQ}finalize(){this.transition.cancel()}getViewportInTransition(){return this.propsInTransition}processViewStateChange(a){let b=!1,c=this.props;if(this.props=a,!c||this._shouldIgnoreViewportChange(c,a))return!1;if(this._isTransitionEnabled(a)){let d=c;if(this.transition.inProgress){let{interruption:a,endProps:b}=this.transition.settings;d={...c,...2===a?b:this.propsInTransition||c}}this._triggerTransition(d,a),b=!0}else this.transition.cancel();return b}updateTransition(){this.transition.update()}_isTransitionEnabled(a){let{transitionDuration:b,transitionInterpolator:c}=a;return(b>0||"auto"===b)&&!!c}_isUpdateDueToCurrentTransition(a){return!!this.transition.inProgress&&!!this.propsInTransition&&this.transition.settings.interpolator.arePropsEqual(a,this.propsInTransition)}_shouldIgnoreViewportChange(a,b){return this.transition.inProgress?3===this.transition.settings.interruption||this._isUpdateDueToCurrentTransition(b):!this._isTransitionEnabled(b)||b.transitionInterpolator.arePropsEqual(a,b)}_triggerTransition(a,b){let c=this.getControllerState(a),d=this.getControllerState(b).shortestPathFrom(c),e=b.transitionInterpolator,f=e.getDuration?e.getDuration(a,b):b.transitionDuration;if(0===f)return;let g=e.initializeProps(a,d);this.propsInTransition={};let h={duration:f,easing:b.transitionEasing||gR,interpolator:e,interruption:b.transitionInterruption||1,startProps:g.start,endProps:g.end,onStart:b.onTransitionStart,onUpdate:this._onTransitionUpdate,onInterrupt:this._onTransitionEnd(b.onTransitionInterrupt),onEnd:this._onTransitionEnd(b.onTransitionEnd)};this.transition.start(h),this.onStateChange({inTransition:!0}),this.updateTransition()}_onTransitionEnd(a){return b=>{this.propsInTransition=null,this.onStateChange({inTransition:!1,isZooming:!1,isPanning:!1,isRotating:!1}),a?.(b)}}}function gT(a,b){if(!a)throw Error(b||"deck.gl: assertion failed.")}class gU{constructor(a){const{compare:b,extract:c,required:d}=a;this._propsToCompare=b,this._propsToExtract=c||b,this._requiredProps=d}arePropsEqual(a,b){for(let c of this._propsToCompare)if(!(c in a)||!(c in b)||!dD(a[c],b[c]))return!1;return!0}initializeProps(a,b){let c={},d={};for(let e of this._propsToExtract)(e in a||e in b)&&(c[e]=a[e],d[e]=b[e]);return this._checkRequiredProps(c),this._checkRequiredProps(d),{start:c,end:d}}getDuration(a,b){return b.transitionDuration}_checkRequiredProps(a){this._requiredProps&&this._requiredProps.forEach(b=>{let c=a[b];gT(Number.isFinite(c)||Array.isArray(c),`${b} is required for transition`)})}}var el=el,cq=cq;let gV=Math.PI/180,gW=180/Math.PI;class gX extends gB{constructor(a={}){const{longitude:b=0,zoom:c=0,nearZMultiplier:d=.5,farZMultiplier:e=1,resolution:f=10}=a;let{latitude:g=0,height:h,altitude:i=1.5,fovy:j}=a;g=Math.max(Math.min(g,85.051129),-85.051129),h=h||1,j?i=ex(j):j=ew(i);const k=Math.pow(2,c-gY(g)),l=a.nearZ??d,m=a.farZ??(i+512*k/h)*e,n=new ef().lookAt({eye:[0,-i,0],up:[0,0,1]});n.rotateX(g*gV),n.rotateZ(-b*gV),n.scale(k/h),super({...a,height:h,viewMatrix:n,longitude:b,latitude:g,zoom:c,distanceScales:function(){let a=Math.PI/180*256;return{unitsPerMeter:[4018225162502676e-20,4018225162502676e-20,4018225162502676e-20],unitsPerMeter2:[0,0,0],metersPerUnit:[24886.609375,24886.609375,24886.609375],unitsPerDegree:[a,a,4018225162502676e-20],unitsPerDegree2:[0,0,0],degreesPerUnit:[1/a,1/a,24886.609375]}}(),fovy:j,focalDistance:i,near:l,far:m}),this.scale=k,this.latitude=g,this.longitude=b,this.fovy=j,this.resolution=f}get projectionMode(){return c9.GLOBE}getDistanceScales(){return this.distanceScales}getBounds(a={}){let b={targetZ:a.z||0},c=this.unproject([0,this.height/2],b),d=this.unproject([this.width/2,0],b),e=this.unproject([this.width,this.height/2],b),f=this.unproject([this.width/2,this.height],b);return e[0]<this.longitude&&(e[0]+=360),c[0]>this.longitude&&(c[0]-=360),[Math.min(c[0],e[0],d[0],f[0]),Math.min(c[1],e[1],d[1],f[1]),Math.max(c[0],e[0],d[0],f[0]),Math.max(c[1],e[1],d[1],f[1])]}unproject(a,{topLeft:b=!0,targetZ:c}={}){let d,[e,f,g]=a,h=b?f:this.height-f,{pixelUnprojectionMatrix:i}=this;if(Number.isFinite(g))d=gZ(i,[e,h,g,1]);else{let a=gZ(i,[e,h,-1,1]),b=gZ(i,[e,h,1,1]),f=((c||0)/6370972+1)*256,g=el.sqrLen(el.sub([],a,b)),j=el.sqrLen(a),k=el.sqrLen(b),l=(4*j*k-(g-j-k)**2)/16*4/g,m=(Math.sqrt(j-l)-Math.sqrt(Math.max(0,f*f-l)))/Math.sqrt(g);d=el.lerp([],a,b,m)}let[j,k,l]=this.unprojectPosition(d);return Number.isFinite(g)?[j,k,l]:Number.isFinite(c)?[j,k,c]:[j,k]}projectPosition(a){let[b,c,d=0]=a,e=b*gV,f=c*gV,g=Math.cos(f),h=(d/6370972+1)*256;return[Math.sin(e)*g*h,-Math.cos(e)*g*h,Math.sin(f)*h]}unprojectPosition(a){let[b,c,d]=a,e=el.len(a);return[Math.atan2(b,-c)*gW,Math.asin(d/e)*gW,(e/256-1)*6370972]}projectFlat(a){return a}unprojectFlat(a){return a}panByPosition([a,b,c],d,e){let f=.25/Math.pow(2,this.zoom-gY(this.latitude)),g=a+f*(e[0]-d[0]),h=b-f*(e[1]-d[1]),i={longitude:g,latitude:h=Math.max(Math.min(h,85.051129),-85.051129),zoom:c-gY(b)};return i.zoom+=gY(i.latitude),i}}function gY(a){return Math.log2(Math.PI*Math.cos(a*Math.PI/180))}function gZ(a,b){let c=cq.transformMat4([],b,a);return cq.scale(c,c,1/c[3]),c}gX.displayName="GlobeViewport";let g$=["longitude","latitude","zoom","bearing","pitch"],g_=["longitude","latitude","zoom"];class g0 extends gU{constructor(a={}){const b=Array.isArray(a)?a:a.transitionProps,c=Array.isArray(a)?{}:a;c.transitionProps=Array.isArray(b)?{compare:b,required:b}:b||{compare:g$,required:g_},super(c.transitionProps),this.opts=c}initializeProps(a,b){let c=super.initializeProps(a,b),{makeViewport:d,around:e}=this.opts;if(d&&e)if(d(a)instanceof gX)cs.warn("around not supported in GlobeView")();else{let f=d(a),g=d(b),h=f.unproject(e);c.start.around=e,Object.assign(c.end,{around:g.project(h),aroundPosition:h,width:b.width,height:b.height})}return c}interpolateProps(a,b,c){let d={};for(let e of this._propsToExtract)d[e]=dC(a[e]||0,b[e]||0,c);if(b.aroundPosition&&this.opts.makeViewport){let e=this.opts.makeViewport({...b,...d});Object.assign(d,e.panByPosition(b.aroundPosition,dC(a.around,b.around,c)))}return d}}let g1={transitionDuration:0},g2=a=>1-(1-a)*(1-a),g3=["wheel"],g4=["panstart","panmove","panend"],g5=["pinchstart","pinchmove","pinchend"],g6=["multipanstart","multipanmove","multipanend"],g7=["dblclick"],g8=["keydown"],g9={};class ha{constructor(a){this.state={},this._events={},this._interactionState={isDragging:!1},this._customEvents=[],this._eventStartBlocked=null,this._panMove=!1,this.invertPan=!1,this.dragMode="rotate",this.inertia=0,this.scrollZoom=!0,this.dragPan=!0,this.dragRotate=!0,this.doubleClickZoom=!0,this.touchZoom=!0,this.touchRotate=!1,this.keyboard=!0,this.transitionManager=new gS({...a,getControllerState:a=>new this.ControllerState(a),onViewStateChange:this._onTransition.bind(this),onStateChange:this._setInteractionState.bind(this)}),this.handleEvent=this.handleEvent.bind(this),this.eventManager=a.eventManager,this.onViewStateChange=a.onViewStateChange||(()=>{}),this.onStateChange=a.onStateChange||(()=>{}),this.makeViewport=a.makeViewport,this.pickPosition=a.pickPosition}set events(a){this.toggleEvents(this._customEvents,!1),this.toggleEvents(a,!0),this._customEvents=a,this.props&&this.setProps(this.props)}finalize(){for(let a in this._events)this._events[a]&&this.eventManager?.off(a,this.handleEvent);this.transitionManager.finalize()}handleEvent(a){this._controllerState=void 0;let b=this._eventStartBlocked;switch(a.type){case"panstart":return!b&&this._onPanStart(a);case"panmove":return this._onPan(a);case"panend":return this._onPanEnd(a);case"pinchstart":return!b&&this._onPinchStart(a);case"pinchmove":return this._onPinch(a);case"pinchend":return this._onPinchEnd(a);case"multipanstart":return!b&&this._onMultiPanStart(a);case"multipanmove":return this._onMultiPan(a);case"multipanend":return this._onMultiPanEnd(a);case"dblclick":return this._onDoubleClick(a);case"wheel":return this._onWheel(a);case"keydown":return this._onKeyDown(a);default:return!1}}get controllerState(){return this._controllerState=this._controllerState||new this.ControllerState({makeViewport:this.makeViewport,...this.props,...this.state}),this._controllerState}getCenter(a){let{x:b,y:c}=this.props,{offsetCenter:d}=a;return[d.x-b,d.y-c]}isPointInBounds(a,b){let{width:c,height:d}=this.props;if(b&&b.handled)return!1;let e=a[0]>=0&&a[0]<=c&&a[1]>=0&&a[1]<=d;return e&&b&&b.stopPropagation(),e}isFunctionKeyPressed(a){let{srcEvent:b}=a;return!!(b.metaKey||b.altKey||b.ctrlKey||b.shiftKey)}isDragging(){return this._interactionState.isDragging||!1}blockEvents(a){let b=setTimeout(()=>{this._eventStartBlocked===b&&(this._eventStartBlocked=null)},a);this._eventStartBlocked=b}setProps(a){a.dragMode&&(this.dragMode=a.dragMode);let b=this.props;this.props=a,"transitionInterpolator"in a||(a.transitionInterpolator=this._getTransitionProps().transitionInterpolator),this.transitionManager.processViewStateChange(a);let{inertia:c}=a;this.inertia=Number.isFinite(c)?c:300*(!0===c);let{scrollZoom:d=!0,dragPan:e=!0,dragRotate:f=!0,doubleClickZoom:g=!0,touchZoom:h=!0,touchRotate:i=!1,keyboard:j=!0}=a,k=!!this.onViewStateChange;if(this.toggleEvents(g3,k&&d),this.toggleEvents(g4,k),this.toggleEvents(g5,k&&(h||i)),this.toggleEvents(g6,k&&i),this.toggleEvents(g7,k&&g),this.toggleEvents(g8,k&&j),this.scrollZoom=d,this.dragPan=e,this.dragRotate=f,this.doubleClickZoom=g,this.touchZoom=h,this.touchRotate=i,this.keyboard=j,(!b||b.height!==a.height||b.width!==a.width||b.maxBounds!==a.maxBounds)&&a.maxBounds){let b=new this.ControllerState({...a,makeViewport:this.makeViewport}),c=b.getViewportProps();Object.keys(c).some(b=>!gD(c[b],a[b],1))&&this.updateViewport(b)}}updateTransition(){this.transitionManager.updateTransition()}toggleEvents(a,b){this.eventManager&&a.forEach(a=>{this._events[a]!==b&&(this._events[a]=b,b?this.eventManager.on(a,this.handleEvent):this.eventManager.off(a,this.handleEvent))})}updateViewport(a,b=null,c={}){let d={...a.getViewportProps(),...b},e=this.controllerState!==a;if(this.state=a.getState(),this._setInteractionState(c),e){let a=this.controllerState&&this.controllerState.getViewportProps();this.onViewStateChange&&this.onViewStateChange({viewState:d,interactionState:this._interactionState,oldViewState:a,viewId:this.props.id})}}_onTransition(a){this.onViewStateChange({...a,interactionState:this._interactionState,viewId:this.props.id})}_setInteractionState(a){Object.assign(this._interactionState,a),this.onStateChange(this._interactionState)}_onPanStart(a){let b=this.getCenter(a);if(!this.isPointInBounds(b,a))return!1;let c=this.isFunctionKeyPressed(a)||a.rightButton||!1;(this.invertPan||"pan"===this.dragMode)&&(c=!c);let d=this.controllerState[c?"panStart":"rotateStart"]({pos:b});return this._panMove=c,this.updateViewport(d,g1,{isDragging:!0}),!0}_onPan(a){return!!this.isDragging()&&(this._panMove?this._onPanMove(a):this._onPanRotate(a))}_onPanEnd(a){return!!this.isDragging()&&(this._panMove?this._onPanMoveEnd(a):this._onPanRotateEnd(a))}_onPanMove(a){if(!this.dragPan)return!1;let b=this.getCenter(a),c=this.controllerState.pan({pos:b});return this.updateViewport(c,g1,{isDragging:!0,isPanning:!0}),!0}_onPanMoveEnd(a){let{inertia:b}=this;if(this.dragPan&&b&&a.velocity){let c=this.getCenter(a),d=[c[0]+a.velocityX*b/2,c[1]+a.velocityY*b/2],e=this.controllerState.pan({pos:d}).panEnd();this.updateViewport(e,{...this._getTransitionProps(),transitionDuration:b,transitionEasing:g2},{isDragging:!1,isPanning:!0})}else{let a=this.controllerState.panEnd();this.updateViewport(a,null,{isDragging:!1,isPanning:!1})}return!0}_onPanRotate(a){if(!this.dragRotate)return!1;let b=this.getCenter(a),c=this.controllerState.rotate({pos:b});return this.updateViewport(c,g1,{isDragging:!0,isRotating:!0}),!0}_onPanRotateEnd(a){let{inertia:b}=this;if(this.dragRotate&&b&&a.velocity){let c=this.getCenter(a),d=[c[0]+a.velocityX*b/2,c[1]+a.velocityY*b/2],e=this.controllerState.rotate({pos:d}).rotateEnd();this.updateViewport(e,{...this._getTransitionProps(),transitionDuration:b,transitionEasing:g2},{isDragging:!1,isRotating:!0})}else{let a=this.controllerState.rotateEnd();this.updateViewport(a,null,{isDragging:!1,isRotating:!1})}return!0}_onWheel(a){if(!this.scrollZoom)return!1;let b=this.getCenter(a);if(!this.isPointInBounds(b,a))return!1;a.srcEvent.preventDefault();let{speed:c=.01,smooth:d=!1}=!0===this.scrollZoom?{}:this.scrollZoom,{delta:e}=a,f=2/(1+Math.exp(-Math.abs(e*c)));e<0&&0!==f&&(f=1/f);let g=d?{...this._getTransitionProps({around:b}),transitionDuration:250}:g1,h=this.controllerState.zoom({pos:b,scale:f});return this.updateViewport(h,g,{isZooming:!0,isPanning:!0}),d||this._setInteractionState({isZooming:!1,isPanning:!1}),!0}_onMultiPanStart(a){let b=this.getCenter(a);if(!this.isPointInBounds(b,a))return!1;let c=this.controllerState.rotateStart({pos:b});return this.updateViewport(c,g1,{isDragging:!0}),!0}_onMultiPan(a){if(!this.touchRotate||!this.isDragging())return!1;let b=this.getCenter(a);b[0]-=a.deltaX;let c=this.controllerState.rotate({pos:b});return this.updateViewport(c,g1,{isDragging:!0,isRotating:!0}),!0}_onMultiPanEnd(a){if(!this.isDragging())return!1;let{inertia:b}=this;if(this.touchRotate&&b&&a.velocityY){let c=this.getCenter(a),d=[c[0],c[1]+=a.velocityY*b/2],e=this.controllerState.rotate({pos:d});this.updateViewport(e,{...this._getTransitionProps(),transitionDuration:b,transitionEasing:g2},{isDragging:!1,isRotating:!0}),this.blockEvents(b)}else{let a=this.controllerState.rotateEnd();this.updateViewport(a,null,{isDragging:!1,isRotating:!1})}return!0}_onPinchStart(a){let b=this.getCenter(a);if(!this.isPointInBounds(b,a))return!1;let c=this.controllerState.zoomStart({pos:b}).rotateStart({pos:b});return g9._startPinchRotation=a.rotation,g9._lastPinchEvent=a,this.updateViewport(c,g1,{isDragging:!0}),!0}_onPinch(a){if(!this.touchZoom&&!this.touchRotate||!this.isDragging())return!1;let b=this.controllerState;if(this.touchZoom){let{scale:c}=a,d=this.getCenter(a);b=b.zoom({pos:d,scale:c})}if(this.touchRotate){let{rotation:c}=a;b=b.rotate({deltaAngleX:g9._startPinchRotation-c})}return this.updateViewport(b,g1,{isDragging:!0,isPanning:this.touchZoom,isZooming:this.touchZoom,isRotating:this.touchRotate}),g9._lastPinchEvent=a,!0}_onPinchEnd(a){if(!this.isDragging())return!1;let{inertia:b}=this,{_lastPinchEvent:c}=g9;if(this.touchZoom&&b&&c&&a.scale!==c.scale){let d=this.getCenter(a),e=this.controllerState.rotateEnd(),f=Math.log2(a.scale),g=(f-Math.log2(c.scale))/(a.deltaTime-c.deltaTime),h=Math.pow(2,f+g*b/2);e=e.zoom({pos:d,scale:h}).zoomEnd(),this.updateViewport(e,{...this._getTransitionProps({around:d}),transitionDuration:b,transitionEasing:g2},{isDragging:!1,isPanning:this.touchZoom,isZooming:this.touchZoom,isRotating:!1}),this.blockEvents(b)}else{let a=this.controllerState.zoomEnd().rotateEnd();this.updateViewport(a,null,{isDragging:!1,isPanning:!1,isZooming:!1,isRotating:!1})}return g9._startPinchRotation=null,g9._lastPinchEvent=null,!0}_onDoubleClick(a){if(!this.doubleClickZoom)return!1;let b=this.getCenter(a);if(!this.isPointInBounds(b,a))return!1;let c=this.isFunctionKeyPressed(a),d=this.controllerState.zoom({pos:b,scale:c?.5:2});return this.updateViewport(d,this._getTransitionProps({around:b}),{isZooming:!0,isPanning:!0}),this.blockEvents(100),!0}_onKeyDown(a){let b;if(!this.keyboard)return!1;let c=this.isFunctionKeyPressed(a),{zoomSpeed:d,moveSpeed:e,rotateSpeedX:f,rotateSpeedY:g}=!0===this.keyboard?{}:this.keyboard,{controllerState:h}=this,i={};switch(a.srcEvent.code){case"Minus":b=c?h.zoomOut(d).zoomOut(d):h.zoomOut(d),i.isZooming=!0;break;case"Equal":b=c?h.zoomIn(d).zoomIn(d):h.zoomIn(d),i.isZooming=!0;break;case"ArrowLeft":c?(b=h.rotateLeft(f),i.isRotating=!0):(b=h.moveLeft(e),i.isPanning=!0);break;case"ArrowRight":c?(b=h.rotateRight(f),i.isRotating=!0):(b=h.moveRight(e),i.isPanning=!0);break;case"ArrowUp":c?(b=h.rotateUp(g),i.isRotating=!0):(b=h.moveUp(e),i.isPanning=!0);break;case"ArrowDown":c?(b=h.rotateDown(g),i.isRotating=!0):(b=h.moveDown(e),i.isPanning=!0);break;default:return!1}return this.updateViewport(b,this._getTransitionProps(),i),!0}_getTransitionProps(a){let{transition:b}=this;return b&&b.transitionInterpolator?a?{...b,transitionInterpolator:new g0({...a,...b.transitionInterpolator.opts,makeViewport:this.controllerState.makeViewport})}:b:g1}}class hb{constructor(a,b,c){this.makeViewport=c,this._viewportProps=this.applyConstraints(a),this._state=b}getViewportProps(){return this._viewportProps}getState(){return this._state}}let hc=[[-1/0,-90],[1/0,90]];function hd([a,b]){if(Math.abs(b)>90&&(b=90*Math.sign(b)),Number.isFinite(a)){let[c,d]=er([a,b]);return[c,dB(d,0,512)]}let[,c]=er([0,b]);return[a,dB(c,0,512)]}class he extends hb{constructor(a){const{width:b,height:c,latitude:d,longitude:e,zoom:f,bearing:g=0,pitch:h=0,altitude:i=1.5,position:j=[0,0,0],maxZoom:k=20,minZoom:l=0,maxPitch:m=60,minPitch:n=0,startPanLngLat:o,startZoomLngLat:p,startRotatePos:q,startRotateLngLat:r,startBearing:s,startPitch:t,startZoom:u,normalize:v=!0}=a;gT(Number.isFinite(e)),gT(Number.isFinite(d)),gT(Number.isFinite(f));const w=a.maxBounds||(v?hc:null);super({width:b,height:c,latitude:d,longitude:e,zoom:f,bearing:g,pitch:h,altitude:i,maxZoom:k,minZoom:l,maxPitch:m,minPitch:n,normalize:v,position:j,maxBounds:w},{startPanLngLat:o,startZoomLngLat:p,startRotatePos:q,startRotateLngLat:r,startBearing:s,startPitch:t,startZoom:u},a.makeViewport),this.getAltitude=a.getAltitude}panStart({pos:a}){return this._getUpdatedState({startPanLngLat:this._unproject(a)})}pan({pos:a,startPos:b}){let c=this.getState().startPanLngLat||this._unproject(b);if(!c)return this;let d=this.makeViewport(this.getViewportProps()).panByPosition(c,a);return this._getUpdatedState(d)}panEnd(){return this._getUpdatedState({startPanLngLat:null})}rotateStart({pos:a}){let b=this.getAltitude?.(a);return this._getUpdatedState({startRotatePos:a,startRotateLngLat:void 0!==b?this._unproject3D(a,b):void 0,startBearing:this.getViewportProps().bearing,startPitch:this.getViewportProps().pitch})}rotate({pos:a,deltaAngleX:b=0,deltaAngleY:c=0}){let d,{startRotatePos:e,startRotateLngLat:f,startBearing:g,startPitch:h}=this.getState();if(!e||void 0===g||void 0===h)return this;if(d=a?this._getNewRotation(a,e,h,g):{bearing:g+b,pitch:h+c},f){let a=this.makeViewport({...this.getViewportProps(),...d}),b="panByPosition3D"in a?"panByPosition3D":"panByPosition";return this._getUpdatedState({...d,...a[b](f,e)})}return this._getUpdatedState(d)}rotateEnd(){return this._getUpdatedState({startRotatePos:null,startRotateLngLat:null,startBearing:null,startPitch:null})}zoomStart({pos:a}){return this._getUpdatedState({startZoomLngLat:this._unproject(a),startZoom:this.getViewportProps().zoom})}zoom({pos:a,startPos:b,scale:c}){let{startZoom:d,startZoomLngLat:e}=this.getState();if(e||(d=this.getViewportProps().zoom,e=this._unproject(b)||this._unproject(a)),!e)return this;let f=this._constrainZoom(d+Math.log2(c)),g=this.makeViewport({...this.getViewportProps(),zoom:f});return this._getUpdatedState({zoom:f,...g.panByPosition(e,a)})}zoomEnd(){return this._getUpdatedState({startZoomLngLat:null,startZoom:null})}zoomIn(a=2){return this._zoomFromCenter(a)}zoomOut(a=2){return this._zoomFromCenter(1/a)}moveLeft(a=100){return this._panFromCenter([a,0])}moveRight(a=100){return this._panFromCenter([-a,0])}moveUp(a=100){return this._panFromCenter([0,a])}moveDown(a=100){return this._panFromCenter([0,-a])}rotateLeft(a=15){return this._getUpdatedState({bearing:this.getViewportProps().bearing-a})}rotateRight(a=15){return this._getUpdatedState({bearing:this.getViewportProps().bearing+a})}rotateUp(a=10){return this._getUpdatedState({pitch:this.getViewportProps().pitch+a})}rotateDown(a=10){return this._getUpdatedState({pitch:this.getViewportProps().pitch-a})}shortestPathFrom(a){let b=a.getViewportProps(),c={...this.getViewportProps()},{bearing:d,longitude:e}=c;return Math.abs(d-b.bearing)>180&&(c.bearing=d<0?d+360:d-360),Math.abs(e-b.longitude)>180&&(c.longitude=e<0?e+360:e-360),c}applyConstraints(a){let{maxPitch:b,minPitch:c,pitch:d,longitude:e,bearing:f,normalize:g,maxBounds:h}=a;if(g&&((e<-180||e>180)&&(a.longitude=gr(e+180,360)-180),(f<-180||f>180)&&(a.bearing=gr(f+180,360)-180)),a.pitch=dB(d,c,b),a.zoom=this._constrainZoom(a.zoom,a),h){let b=hd(h[0]),c=hd(h[1]),d=2**a.zoom,e=a.width/2/d,f=a.height/2/d,[g,i]=es([b[0]+e,b[1]+f]),[j,k]=es([c[0]-e,c[1]-f]);a.longitude=dB(a.longitude,g,j),a.latitude=dB(a.latitude,i,k)}return a}_constrainZoom(a,b){b||(b=this.getViewportProps());let{maxZoom:c,maxBounds:d}=b,e=null!==d&&b.width>0&&b.height>0,{minZoom:f}=b;if(e){let a=hd(d[0]),e=hd(d[1]),g=e[0]-a[0],h=e[1]-a[1];Number.isFinite(g)&&g>0&&(f=Math.max(f,Math.log2(b.width/g))),Number.isFinite(h)&&h>0&&(f=Math.max(f,Math.log2(b.height/h))),f>c&&(f=c)}return dB(a,f,c)}_zoomFromCenter(a){let{width:b,height:c}=this.getViewportProps();return this.zoom({pos:[b/2,c/2],scale:a})}_panFromCenter(a){let{width:b,height:c}=this.getViewportProps();return this.pan({startPos:[b/2,c/2],pos:[b/2+a[0],c/2+a[1]]})}_getUpdatedState(a){return new this.constructor({makeViewport:this.makeViewport,...this.getViewportProps(),...this.getState(),...a})}_unproject(a){let b=this.makeViewport(this.getViewportProps());return a&&b.unproject(a)}_unproject3D(a,b){return this.makeViewport(this.getViewportProps()).unproject(a,{targetZ:b})}_getNewRotation(a,b,c,d){let e=a[0]-b[0],f=a[1]-b[1],g=a[1],h=b[1],{width:i,height:j}=this.getViewportProps(),k=0;f>0?Math.abs(j-h)>5&&(k=f/(h-j)*1.2):f<0&&h>5&&(k=1-g/h),k=dB(k,-1,1);let{minPitch:l,maxPitch:m}=this.getViewportProps(),n=c;return k>0?n=c+k*(m-c):k<0&&(n=c-k*(l-c)),{pitch:n,bearing:d+e/i*180}}}class hf extends ha{constructor(){super(...arguments),this.ControllerState=he,this.transition={transitionDuration:300,transitionInterpolator:new g0({transitionProps:{compare:["longitude","latitude","zoom","bearing","pitch","position"],required:["longitude","latitude","zoom"]}})},this.dragMode="pan",this.rotationPivot="center",this._getAltitude=a=>{if("2d"===this.rotationPivot)return 0;if("3d"===this.rotationPivot&&this.pickPosition){let{x:b,y:c}=this.props,d=this.pickPosition(b+a[0],c+a[1]);if(d&&d.coordinate&&d.coordinate.length>=3)return d.coordinate[2]}}}setProps(a){"rotationPivot"in a&&(this.rotationPivot=a.rotationPivot||"center"),a.getAltitude=this._getAltitude,a.position=a.position||[0,0,0],a.maxBounds=a.maxBounds||(!1===a.normalize?null:hc),super.setProps(a)}updateViewport(a,b=null,c={}){let d=a.getState();c.isDragging&&d.startRotateLngLat?c={...c,rotationPivotPosition:d.startRotateLngLat}:!1===c.isDragging&&(c={...c,rotationPivotPosition:void 0}),super.updateViewport(a,b,c)}}class hg extends gL{constructor(a={}){super(a)}getViewportType(){return gO}get ControllerType(){return hf}}hg.displayName="MapView";let hh=[255,255,255],hi=0;class hj{constructor(a={}){this.type="ambient";const{color:b=hh}=a,{intensity:c=1}=a;this.id=a.id||`ambient-${hi++}`,this.color=b,this.intensity=c}}let hk=[255,255,255],hl=[0,0,-1],hm=0;class hn{constructor(a={}){this.type="directional";const{color:b=hk}=a,{intensity:c=1}=a,{direction:d=hl}=a,{_shadow:e=!1}=a;this.id=a.id||`directional-${hm++}`,this.color=b,this.intensity=c,this.type="directional",this.direction=new d1(d).normalize().toArray(),this.shadow=e}getProjectedLight(a){return this}}class ho{constructor(a,b={id:"pass"}){const{id:c}=b;this.id=c,this.device=a,this.props={...b}}setProps(a){Object.assign(this.props,a)}render(a){}cleanup(){}}let hp={depthWriteEnabled:!0,depthCompare:"less-equal",blendColorOperation:"add",blendColorSrcFactor:"src-alpha",blendColorDstFactor:"one",blendAlphaOperation:"add",blendAlphaSrcFactor:"one-minus-dst-alpha",blendAlphaDstFactor:"one"};class hq extends ho{constructor(){super(...arguments),this._lastRenderIndex=-1}render(a){this._render(a)}_render(a){let b=this.device.canvasContext,c=a.target??b.getCurrentFramebuffer(),[d,e]=b.getDrawingBufferSize(),f=a.clearCanvas??!0,g=a.clearColor??(!!f&&[0,0,0,0]),h=a.colorMask??15,i={viewport:[0,0,d,e]};a.colorMask&&(i.colorMask=h),a.scissorRect&&(i.scissorRect=a.scissorRect);let j=this.device.beginRenderPass({framebuffer:c,parameters:i,clearColor:g,clearDepth:!!f&&1,clearStencil:!!f&&0});try{return this._drawLayers(j,a)}finally{j.end(),this.device.submit()}}_drawLayers(a,b){let{target:c,shaderModuleProps:d,viewports:e,views:f,onViewportActive:g,clearStack:h=!0}=b;b.pass=b.pass||"unknown",h&&(this._lastRenderIndex=-1);let i=[];for(let h of e){let e=f&&f[h.id];g?.(h);let j=this._getDrawLayerParams(h,b);for(let f of h.subViewports||[h]){let g=this._drawLayersInViewport(a,{target:c,shaderModuleProps:d,viewport:f,view:e,pass:b.pass,layers:b.layers,isPicking:b.isPicking},j);i.push(g)}}return i}_getDrawLayerParams(a,{layers:b,pass:c,isPicking:d=!1,layerFilter:e,cullRect:f,views:g,effects:h,shaderModuleProps:i},j=!1){let k=[],l=function a(b=0,c={}){let d={},e=(f,g)=>{let h,i=f.props._offset,j=f.id,k=f.parent&&f.parent.id;if(!k||k in c||e(f.parent,!1),k in d){let b=d[k]=d[k]||a(c[k],c);h=b(f,g),d[j]=b}else Number.isFinite(i)?(h=i+(c[k]||0),d[j]=null):h=b;return g&&h>=b&&(b=h+1),c[j]=h,h};return e}(this._lastRenderIndex+1),m={layer:b[0],viewport:a,isPicking:d,renderPass:c,cullRect:f},n={};for(let d=0;d<b.length;d++){let f=b[d],o=this._shouldDrawLayer(f,m,e,n),p={shouldDrawLayer:o};o&&!j&&(p.shouldDrawLayer=!0,p.layerRenderIndex=l(f,o),p.shaderModuleProps=this._getShaderModuleProps(f,h,c,i),p.layerParameters={..."webgpu"===f.context.device.type?hp:null,...f.context.deck?.props.parameters,...g?.[a.id]?.props.parameters,...this.getLayerParameters(f,d,a)}),k[d]=p}return k}_drawLayersInViewport(a,{layers:b,shaderModuleProps:c,pass:d,target:e,viewport:f,view:g,isPicking:h},i){let j=function(a,{shaderModuleProps:b,target:c,viewport:d}){let e=b?.project?.devicePixelRatio??a.canvasContext.cssToDeviceRatio(),[,f]=a.canvasContext.getDrawingBufferSize(),g=c?c.height:f;return[d.x*e,g-(d.y+d.height)*e,d.width*e,d.height*e]}(this.device,{shaderModuleProps:c,target:e,viewport:f});if(g){let{clear:a,clearColor:b,clearDepth:c,clearStencil:d}=g.props;if(a){let a=[0,0,0,0],f=1,g=0;Array.isArray(b)&&!h?a=[...b.slice(0,3),b[3]||255].map(a=>a/255):!1===b&&(a=!1),void 0!==c&&(f=c),void 0!==d&&(g=d),this.device.beginRenderPass({framebuffer:e,parameters:{viewport:j,scissorRect:j},clearColor:a,clearDepth:f,clearStencil:g}).end()}}let k={totalCount:b.length,visibleCount:0,compositeCount:0,pickableCount:0};a.setParameters({viewport:j});for(let c=0;c<b.length;c++){let e=b[c],g=i[c],{shouldDrawLayer:h}=g;if(h&&e.props.pickable&&k.pickableCount++,e.isComposite&&k.compositeCount++,e.isDrawable&&g.shouldDrawLayer){let{layerRenderIndex:b,shaderModuleProps:c,layerParameters:h}=g;k.visibleCount++,this._lastRenderIndex=Math.max(this._lastRenderIndex,b),c.project&&(c.project.viewport=f),e.context.renderPass=a;try{e._drawLayer({renderPass:a,shaderModuleProps:c,uniforms:{layerIndex:b},parameters:h})}catch(a){e.raiseError(a,`drawing ${e} to ${d}`)}}}return k}shouldDrawLayer(a){return!0}getShaderModuleProps(a,b,c){return null}getLayerParameters(a,b,c){return a.props.parameters}_shouldDrawLayer(a,b,c,d){if(!(a.props.visible&&this.shouldDrawLayer(a)))return!1;b.layer=a;let e=a.parent;for(;e;){if(!e.props.visible||!e.filterSubLayer(b))return!1;b.layer=e,e=e.parent}if(c){let a=b.layer.id;if(a in d||(d[a]=c(b)),!d[a])return!1}return a.activateViewport(b.viewport),!0}_getShaderModuleProps(a,b,c,d){let e=this.device.canvasContext.cssToDeviceRatio(),f=a.internalState?.propsInTransition||a.props,g={layer:f,picking:{isActive:!1},project:{viewport:a.context.viewport,devicePixelRatio:e,modelMatrix:f.modelMatrix,coordinateSystem:f.coordinateSystem,coordinateOrigin:f.coordinateOrigin,autoWrapLongitude:a.wrapLongitude}};if(b)for(let c of b)hr(g,c.getShaderModuleProps?.(a,g));for(let b of a.context.defaultShaderModules)b.name in g||(g[b.name]={});return hr(g,this.getShaderModuleProps(a,b,g),d)}}function hr(a,...b){for(let c of b)if(c)for(let b in c)a[b]?Object.assign(a[b],c[b]):a[b]=c[b];return a}class hs extends hq{constructor(a,b){super(a,b);const c=a.createTexture({format:"rgba8unorm",width:1,height:1,sampler:{minFilter:"linear",magFilter:"linear",addressModeU:"clamp-to-edge",addressModeV:"clamp-to-edge"}}),d=a.createTexture({format:"depth16unorm",width:1,height:1});this.fbo=a.createFramebuffer({id:"shadowmap",width:1,height:1,colorAttachments:[c],depthStencilAttachment:d})}delete(){this.fbo&&(this.fbo.destroy(),this.fbo=null)}getShadowMap(){return this.fbo.colorAttachments[0].texture}render(a){let b=this.fbo,c=this.device.canvasContext.cssToDeviceRatio(),d=a.viewports[0],e=d.width*c,f=d.height*c;(e!==b.width||f!==b.height)&&b.resize({width:e,height:f}),super.render({...a,clearColor:[1,1,1,1],target:b,pass:"shadow"})}getLayerParameters(a,b,c){return{...a.props.parameters,blend:!1,depthWriteEnabled:!0,depthCompare:"less-equal"}}shouldDrawLayer(a){return!1!==a.props.shadowEnabled}getShaderModuleProps(a,b,c){return{shadow:{project:c.project,drawToShadowMap:!0}}}}let ht={color:[255,255,255],intensity:1},hu=[{color:[255,255,255],intensity:1,direction:[-1,3,-1]},{color:[255,255,255],intensity:.9,direction:[1,-8,-2.5]}],hv=[0,0,0,200/255];class hw{constructor(a={}){this.id="lighting-effect",this.shadowColor=hv,this.shadow=!1,this.directionalLights=[],this.pointLights=[],this.shadowPasses=[],this.dummyShadowMap=null,this.setProps(a)}setup(a){this.context=a;let{device:b,deck:c}=a;this.shadow&&!this.dummyShadowMap&&(this._createShadowPasses(b),c._addDefaultShaderModule(eJ),this.dummyShadowMap=b.createTexture({width:1,height:1}))}setProps(a){for(let b in this.ambientLight=void 0,this.directionalLights=[],this.pointLights=[],a){let c=a[b];switch(c.type){case"ambient":this.ambientLight=c;break;case"directional":this.directionalLights.push(c);break;case"point":this.pointLights.push(c)}}this._applyDefaultLights(),this.shadow=this.directionalLights.some(a=>a.shadow),this.context&&this.setup(this.context),this.props=a}preRender({layers:a,layerFilter:b,viewports:c,onViewportActive:d,views:e}){if(this.shadow){this.shadowMatrices=this._calculateMatrices();for(let f=0;f<this.shadowPasses.length;f++)this.shadowPasses[f].render({layers:a,layerFilter:b,viewports:c,onViewportActive:d,views:e,shaderModuleProps:{shadow:{shadowLightId:f,dummyShadowMap:this.dummyShadowMap,shadowMatrices:this.shadowMatrices}}})}}getShaderModuleProps(a,b){let c=this.shadow?{project:b.project,shadowMaps:this.shadowPasses.map(a=>a.getShadowMap()),dummyShadowMap:this.dummyShadowMap,shadowColor:this.shadowColor,shadowMatrices:this.shadowMatrices}:{},d={enabled:!0,lights:this._getLights(a)},e=a.props.material;return{shadow:c,lighting:d,phongMaterial:e,gouraudMaterial:e}}cleanup(a){for(let a of this.shadowPasses)a.delete();this.shadowPasses.length=0,this.dummyShadowMap&&(this.dummyShadowMap.destroy(),this.dummyShadowMap=null,a.deck._removeDefaultShaderModule(eJ))}_calculateMatrices(){let a=[];for(let b of this.directionalLights){let c=new ef().lookAt({eye:new d1(b.direction).negate()});a.push(c)}return a}_createShadowPasses(a){for(let b=0;b<this.directionalLights.length;b++){let c=new hs(a);this.shadowPasses[b]=c}}_applyDefaultLights(){let{ambientLight:a,pointLights:b,directionalLights:c}=this;a||0!==b.length||0!==c.length||(this.ambientLight=new hj(ht),this.directionalLights.push(new hn(hu[0]),new hn(hu[1])))}_getLights(a){let b=[];for(let c of(this.ambientLight&&b.push(this.ambientLight),this.pointLights))b.push(c.getProjectedLight({layer:a}));for(let c of this.directionalLights)b.push(c.getProjectedLight({layer:a}));return b}}let hx=new hw;class hy{constructor(a){this._resolvedEffects=[],this._defaultEffects=[],this.effects=[],this._context=a,this._needsRedraw="Initial render",this._setEffects([])}addDefaultEffect(a){let b=this._defaultEffects;if(!b.find(b=>b.id===a.id)){let c=b.findIndex(b=>(b.order??1/0)-(a.order??1/0)>0);c<0?b.push(a):b.splice(c,0,a),a.setup(this._context),this._setEffects(this.effects)}}setProps(a){"effects"in a&&!gD(a.effects,this.effects,1)&&this._setEffects(a.effects)}needsRedraw(a={clearRedrawFlags:!1}){let b=this._needsRedraw;return a.clearRedrawFlags&&(this._needsRedraw=!1),b}getEffects(){return this._resolvedEffects}_setEffects(a){let b={};for(let a of this.effects)b[a.id]=a;let c=[];for(let d of a){let a=b[d.id],e=d;a&&a!==d?a.setProps?(a.setProps(d.props),e=a):a.cleanup(this._context):a||d.setup(this._context),c.push(e),delete b[d.id]}for(let a in b)b[a].cleanup(this._context);this.effects=c,this._resolvedEffects=c.concat(this._defaultEffects),a.some(a=>a instanceof hw)||this._resolvedEffects.push(hx),this._needsRedraw="effects changed"}finalize(){for(let a of this._resolvedEffects)a.cleanup(this._context);this.effects.length=0,this._resolvedEffects.length=0,this._defaultEffects.length=0}}class hz extends hq{shouldDrawLayer(a){let{operation:b}=a.props;return b.includes("draw")||b.includes("terrain")}render(a){return this._render(a)}}let hA={blendColorOperation:"add",blendColorSrcFactor:"one",blendColorDstFactor:"zero",blendAlphaOperation:"add",blendAlphaSrcFactor:"constant",blendAlphaDstFactor:"zero"};class hB extends hq{constructor(){super(...arguments),this._colorEncoderState=null}render(a){return"pickingFBO"in a?this._drawPickingBuffer(a):{decodePickingColor:null,stats:super._render(a)}}_drawPickingBuffer({layers:a,layerFilter:b,views:c,viewports:d,onViewportActive:e,pickingFBO:f,deviceRect:{x:g,y:h,width:i,height:j},cullRect:k,effects:l,pass:m="picking",pickZ:n,shaderModuleProps:o,clearColor:p}){this.pickZ=n;let q=this._resetColorEncoder(n),r=super._render({target:f,layers:a,layerFilter:b,views:c,viewports:d,onViewportActive:e,cullRect:k,effects:l?.filter(a=>a.useInPicking),pass:m,isPicking:!0,shaderModuleProps:o,clearColor:p??[0,0,0,0],colorMask:15,scissorRect:[g,h,i,j]});return this._colorEncoderState=null,{decodePickingColor:q&&hD.bind(null,q),stats:r}}shouldDrawLayer(a){let{pickable:b,operation:c}=a.props;return b&&c.includes("draw")||c.includes("terrain")||c.includes("mask")}getShaderModuleProps(a,b,c){return{picking:{isActive:1,isAttribute:this.pickZ},lighting:{enabled:!1}}}getLayerParameters(a,b,c){let d={...a.props.parameters},{pickable:e,operation:f}=a.props;return this._colorEncoderState?e&&f.includes("draw")?(Object.assign(d,hA),d.blend=!0,"webgpu"===this.device.type?d.blendConstant=hC(this._colorEncoderState,a,c):d.blendColor=hC(this._colorEncoderState,a,c),f.includes("terrain")&&a.state?._hasPickingCover&&(d.blendAlphaSrcFactor="one")):f.includes("terrain")&&(d.blend=!1):d.blend=!1,d}_resetColorEncoder(a){return this._colorEncoderState=a?null:{byLayer:new Map,byAlpha:[]},this._colorEncoderState}}function hC(a,b,c){let d,{byLayer:e,byAlpha:f}=a,g=e.get(b);return g?(g.viewports.push(c),d=g.a):(d=e.size+1)<=255?(g={a:d,layer:b,viewports:[c]},e.set(b,g),f[d]=g):(cs.warn("Too many pickable layers, only picking the first 255")(),d=0),[0,0,0,d/255]}function hD(a,b){let c=a.byAlpha[b[3]];return c&&{pickedLayer:c.layer,pickedViewports:c.viewports,pickedObjectIndex:c.layer.decodePickingColor(b)}}class hE{constructor(a,b={}){this.device=a,this.stats=b.stats,this.layerFilter=null,this.drawPickingColors=!1,this.drawLayersPass=new hz(a),this.pickLayersPass=new hB(a),this.renderCount=0,this._needsRedraw="Initial render",this.renderBuffers=[],this.lastPostProcessEffect=null}setProps(a){this.layerFilter!==a.layerFilter&&(this.layerFilter=a.layerFilter,this._needsRedraw="layerFilter changed"),this.drawPickingColors!==a.drawPickingColors&&(this.drawPickingColors=a.drawPickingColors,this._needsRedraw="drawPickingColors changed")}renderLayers(a){if(!a.viewports.length)return;let b=this.drawPickingColors?this.pickLayersPass:this.drawLayersPass,c={layerFilter:this.layerFilter,isPicking:this.drawPickingColors,...a};c.effects&&this._preRender(c.effects,c);let d=this.lastPostProcessEffect?this.renderBuffers[0]:c.target;this.lastPostProcessEffect&&(c.clearColor=[0,0,0,0],c.clearCanvas=!0);let e=b.render({...c,target:d}),f="stats"in e?e.stats:e;c.effects&&(this.lastPostProcessEffect&&(c.clearCanvas=void 0===a.clearCanvas||a.clearCanvas),this._postRender(c.effects,c)),this.renderCount++,eY("deckRenderer.renderLayers",this,f,a),this._updateStats(f)}needsRedraw(a={clearRedrawFlags:!1}){let b=this._needsRedraw;return a.clearRedrawFlags&&(this._needsRedraw=!1),b}finalize(){let{renderBuffers:a}=this;for(let b of a)b.delete();a.length=0}_updateStats(a){if(!this.stats)return;let b=0;for(let{visibleCount:c}of a)b+=c;this.stats.get("Layers rendered").addCount(b)}_preRender(a,b){for(let c of(this.lastPostProcessEffect=null,b.preRenderStats=b.preRenderStats||{},a))b.preRenderStats[c.id]=c.preRender(b),c.postRender&&(this.lastPostProcessEffect=c.id);this.lastPostProcessEffect&&this._resizeRenderBuffers()}_resizeRenderBuffers(){let{renderBuffers:a}=this,b=this.device.canvasContext.getDrawingBufferSize(),[c,d]=b;for(let e of(0===a.length&&[0,1].map(b=>{let e=this.device.createTexture({sampler:{minFilter:"linear",magFilter:"linear"},width:c,height:d});a.push(this.device.createFramebuffer({id:`deck-renderbuffer-${b}`,colorAttachments:[e]}))}),a))e.resize(b)}_postRender(a,b){let{renderBuffers:c}=this,d={...b,inputBuffer:c[0],swapBuffer:c[1]};for(let e of a)if(e.postRender){d.target=e.id===this.lastPostProcessEffect?b.target:void 0;let a=e.postRender(d);d.inputBuffer=a,d.swapBuffer=a===c[0]?c[1]:c[0]}}}var hF=a.i(42566),hG=a.i(83893);let hH={pickedColor:null,pickedObjectIndex:-1};function hI({pickedColors:a,decodePickingColor:b,deviceX:c,deviceY:d,deviceRadius:e,deviceRect:f}){let{x:g,y:h,width:i,height:j}=f,k=e*e,l=-1,m=0;for(let b=0;b<j;b++){let e=b+h-d,f=e*e;if(f>k)m+=4*i;else for(let b=0;b<i;b++){if(a[m+3]-1>=0){let a=b+g-c,d=a*a+f;d<=k&&(k=d,l=m)}m+=4}}if(l>=0){let c=a.slice(l,l+4),d=b(c);if(d){let a=Math.floor(l/4/i),b=l/4-a*i;return{...d,pickedColor:c,pickedX:g+b,pickedY:h+a}}cs.error("Picked non-existent layer. Is picking buffer corrupt?")()}return hH}function hJ({pickedColors:a,decodePickingColor:b}){let c=new Map;if(a){for(let d=0;d<a.length;d+=4)if(a[d+3]-1>=0){let e=a.slice(d,d+4),f=e.join(",");if(!c.has(f)){let a=b(e);a?c.set(f,{...a,color:e}):cs.error("Picked non-existent layer. Is picking buffer corrupt?")()}}}return Array.from(c.values())}function hK({pickInfo:a,viewports:b,pixelRatio:c,x:d,y:e,z:f}){let g,h=b[0];if(b.length>1&&(h=function(a,b){for(let c=a.length-1;c>=0;c--){let d=a[c];if(d.containsPixel(b))return d}return a[0]}(a?.pickedViewports||b,{x:d,y:e})),h){let a=[d-h.x,e-h.y];void 0!==f&&(a[2]=f),g=h.unproject(a)}return{color:null,layer:null,viewport:h,index:-1,picked:!1,x:d,y:e,pixel:[d,e],coordinate:g,devicePixel:a&&"pickedX"in a?[a.pickedX,a.pickedY]:void 0,pixelRatio:c}}function hL(a){let{pickInfo:b,lastPickedInfo:c,mode:d,layers:e}=a,{pickedColor:f,pickedLayer:g,pickedObjectIndex:h}=b,i=g?[g]:[];if("hover"===d){let a=c.index,b=c.layerId,d=g?g.props.id:null;if(d!==b||h!==a){if(d!==b){let a=e.find(a=>a.props.id===b);a&&i.unshift(a)}c.layerId=d,c.index=h,c.info=null}}let j=hK(a),k=new Map;return k.set(null,j),i.forEach(a=>{let b={...j};a===g&&(b.color=f,b.index=h,b.picked=!0);let e=(b=hM({layer:a,info:b,mode:d})).layer;a===g&&"hover"===d&&(c.info=b),k.set(e.id,b),"hover"===d&&e.updateAutoHighlight(b)}),k}function hM({layer:a,info:b,mode:c}){for(;a&&b;){let d=b.layer||null;b.sourceLayer=d,b.layer=a,b=a.getPickingInfo({info:b,mode:c,sourceLayer:d}),a=a.parent}return b}class hN{constructor(a,b={}){this._pickable=!0,this.device=a,this.stats=b.stats,this.pickLayersPass=new hB(a),this.lastPickedInfo={index:-1,layerId:null,info:null}}setProps(a){"layerFilter"in a&&(this.layerFilter=a.layerFilter),"_pickable"in a&&(this._pickable=a._pickable)}finalize(){this.pickingFBO&&this.pickingFBO.destroy(),this.depthFBO&&this.depthFBO.destroy()}pickObjectAsync(a){return this._pickClosestObjectAsync(a)}pickObjectsAsync(a){return this._pickVisibleObjectsAsync(a)}pickObject(a){return this._pickClosestObject(a)}pickObjects(a){return this._pickVisibleObjects(a)}getLastPickedObject({x:a,y:b,layers:c,viewports:d},e=this.lastPickedInfo.info){let f=e&&e.layer&&e.layer.id,g=e&&e.viewport&&e.viewport.id,h=f?c.find(a=>a.id===f):null,i=g&&d.find(a=>a.id===g)||d[0],j=i&&i.unproject([a-i.x,b-i.y]);return{...e,x:a,y:b,viewport:i,coordinate:j,layer:h}}_resizeBuffer(a=this.device.getDefaultCanvasContext()){if(!this.pickingFBO){let a=this.device.createTexture({format:"rgba8unorm",width:1,height:1,usage:hG.Texture.RENDER_ATTACHMENT|hG.Texture.COPY_SRC});if(this.pickingFBO=this.device.createFramebuffer({colorAttachments:[a],depthStencilAttachment:"depth16unorm"}),this.device.isTextureFormatRenderable("rgba32float")){let a=this.device.createTexture({format:"rgba32float",width:1,height:1,usage:hG.Texture.RENDER_ATTACHMENT|hG.Texture.COPY_SRC}),b=this.device.createFramebuffer({colorAttachments:[a],depthStencilAttachment:"depth16unorm"});this.depthFBO=b}}let[b,c]=a.getDrawingBufferSize();this.pickingFBO?.resize({width:b,height:c}),this.depthFBO?.resize({width:b,height:c})}_getPickable(a){if(!1===this._pickable)return null;let b=a.filter(a=>this.pickLayersPass.shouldDrawLayer(a)&&!a.isComposite);return b.length?b:null}async _pickClosestObjectAsync({layers:a,views:b,viewports:c,x:d,y:e,radius:f=0,depth:g=1,mode:h="query",unproject3D:i,canvasContext:j=this.device.getDefaultCanvasContext(),onViewportActive:k,effects:l}){let m,n=j.cssToDeviceRatio(),o=this._getPickable(a);if(!o||0===c.length)return{result:[],emptyInfo:hK({viewports:c,x:d,y:e,pixelRatio:n})};this._resizeBuffer(j);let p=j.cssToDevicePixels([d,e],!0),q=[p.x+Math.floor(p.width/2),p.y+Math.floor(p.height/2)],r=Math.round(f*n),{width:s,height:t}=this.pickingFBO,u=this._getPickingRect({deviceX:q[0],deviceY:q[1],deviceRadius:r,deviceWidth:s,deviceHeight:t}),v={x:d-f,y:e-f,width:2*f+1,height:2*f+1},w=[],x=new Set;for(let a=0;a<g;a++){let f,j;f=u?hI({...await this._drawAndSampleAsync({layers:o,views:b,viewports:c,onViewportActive:k,deviceRect:u,cullRect:v,effects:l,pass:`picking:${h}`}),deviceX:q[0],deviceY:q[1],deviceRadius:r,deviceRect:u}):{pickedColor:null,pickedObjectIndex:-1};let p=this._getDepthLayers(f,o,i);if(p.length>0){let{pickedColors:a}=await this._drawAndSampleAsync({layers:p,views:b,viewports:c,onViewportActive:k,deviceRect:{x:f.pickedX??q[0],y:f.pickedY??q[1],width:1,height:1},cullRect:v,effects:l,pass:`picking:${h}:z`},!0);a[3]&&(j=a[0])}for(let b of(f.pickedLayer&&a+1<g&&(x.add(f.pickedLayer),f.pickedLayer.disablePickingIndex(f.pickedObjectIndex)),(m=hL({pickInfo:f,lastPickedInfo:this.lastPickedInfo,mode:h,layers:o,viewports:c,x:d,y:e,z:j,pixelRatio:n})).values()))b.layer&&w.push(b);if(!f.pickedColor)break}for(let a of x)a.restorePickingColors();return{result:w,emptyInfo:m.get(null)}}_pickClosestObject({layers:a,views:b,viewports:c,x:d,y:e,radius:f=0,depth:g=1,mode:h="query",unproject3D:i,canvasContext:j=this.device.getDefaultCanvasContext(),onViewportActive:k,effects:l}){let m,n=j.cssToDeviceRatio(),o=this._getPickable(a);if(!o||0===c.length)return{result:[],emptyInfo:hK({viewports:c,x:d,y:e,pixelRatio:n})};this._resizeBuffer(j);let p=j.cssToDevicePixels([d,e],!0),q=[p.x+Math.floor(p.width/2),p.y+Math.floor(p.height/2)],r=Math.round(f*n),{width:s,height:t}=this.pickingFBO,u=this._getPickingRect({deviceX:q[0],deviceY:q[1],deviceRadius:r,deviceWidth:s,deviceHeight:t}),v={x:d-f,y:e-f,width:2*f+1,height:2*f+1},w=[],x=new Set;for(let a=0;a<g;a++){let f,j;f=u?hI({...this._drawAndSample({layers:o,views:b,viewports:c,onViewportActive:k,deviceRect:u,cullRect:v,effects:l,pass:`picking:${h}`}),deviceX:q[0],deviceY:q[1],deviceRadius:r,deviceRect:u}):{pickedColor:null,pickedObjectIndex:-1};let p=this._getDepthLayers(f,o,i);if(p.length>0){let{pickedColors:a}=this._drawAndSample({layers:p,views:b,viewports:c,onViewportActive:k,deviceRect:{x:f.pickedX??q[0],y:f.pickedY??q[1],width:1,height:1},cullRect:v,effects:l,pass:`picking:${h}:z`},!0);a[3]&&(j=a[0])}for(let b of(f.pickedLayer&&a+1<g&&(x.add(f.pickedLayer),f.pickedLayer.disablePickingIndex(f.pickedObjectIndex)),(m=hL({pickInfo:f,lastPickedInfo:this.lastPickedInfo,mode:h,layers:o,viewports:c,x:d,y:e,z:j,pixelRatio:n})).values()))b.layer&&w.push(b);if(!f.pickedColor)break}for(let a of x)a.restorePickingColors();return{result:w,emptyInfo:m.get(null)}}async _pickVisibleObjectsAsync({layers:a,views:b,viewports:c,x:d,y:e,width:f=1,height:g=1,mode:h="query",maxObjects:i=null,canvasContext:j=this.device.getDefaultCanvasContext(),onViewportActive:k,effects:l}){let m=this._getPickable(a);if(!m||0===c.length)return[];this._resizeBuffer(j);let n=j.cssToDeviceRatio(),o=j.cssToDevicePixels([d,e],!0),p=o.x,q=o.y+o.height,r=j.cssToDevicePixels([d+f,e+g],!0),s=r.x+r.width,t=r.y,u=await this._drawAndSampleAsync({layers:m,views:b,viewports:c,onViewportActive:k,deviceRect:{x:p,y:t,width:s-p,height:q-t},cullRect:{x:d,y:e,width:f,height:g},effects:l,pass:`picking:${h}`}),v=hJ(u),w=new Map,x=[],y=Number.isFinite(i);for(let a=0;a<v.length&&(!y||!(x.length>=i));a++){let b=v[a],c={color:b.pickedColor,layer:null,index:b.pickedObjectIndex,picked:!0,x:d,y:e,pixelRatio:n},f=(c=hM({layer:b.pickedLayer,info:c,mode:h})).layer.id;w.has(f)||w.set(f,new Set);let g=w.get(f),i=c.object??c.index;g.has(i)||(g.add(i),x.push(c))}return x}_pickVisibleObjects({layers:a,views:b,viewports:c,x:d,y:e,width:f=1,height:g=1,mode:h="query",maxObjects:i=null,canvasContext:j=this.device.getDefaultCanvasContext(),onViewportActive:k,effects:l}){let m=this._getPickable(a);if(!m||0===c.length)return[];this._resizeBuffer(j);let n=j.cssToDeviceRatio(),o=j.cssToDevicePixels([d,e],!0),p=o.x,q=o.y+o.height,r=j.cssToDevicePixels([d+f,e+g],!0),s=r.x+r.width,t=r.y,u=this._drawAndSample({layers:m,views:b,viewports:c,onViewportActive:k,deviceRect:{x:p,y:t,width:s-p,height:q-t},cullRect:{x:d,y:e,width:f,height:g},effects:l,pass:`picking:${h}`}),v=hJ(u),w=new Map,x=[],y=Number.isFinite(i);for(let a=0;a<v.length&&(!y||!(x.length>=i));a++){let b=v[a],c={color:b.pickedColor,layer:null,index:b.pickedObjectIndex,picked:!0,x:d,y:e,pixelRatio:n},f=(c=hM({layer:b.pickedLayer,info:c,mode:h})).layer.id;w.has(f)||w.set(f,new Set);let g=w.get(f),i=c.object??c.index;g.has(i)||(g.add(i),x.push(c))}return x}async _drawAndSampleAsync({layers:a,views:b,viewports:c,onViewportActive:d,deviceRect:e,cullRect:f,effects:g,pass:h},i=!1){let j=i?this.depthFBO:this.pickingFBO,k={layers:a,layerFilter:this.layerFilter,views:b,viewports:c,onViewportActive:d,pickingFBO:j,deviceRect:e,cullRect:f,effects:g,pass:h,pickZ:i,preRenderStats:{},isPicking:!0};for(let a of g)a.useInPicking&&(k.preRenderStats[a.id]=a.preRender(k));let{decodePickingColor:l,stats:m}=this.pickLayersPass.render(k);this._updateStats(m);let{x:n,y:o,width:p,height:q}=e,r=j.colorAttachments[0]?.texture;if(!r)throw Error("Picking framebuffer color attachment is missing");let s=await this._readTextureDataAsync(r,{x:n,y:o,width:p,height:q},i?Float32Array:Uint8Array);if(!i){let a=!1;for(let b=3;b<s.length;b+=4)if(0!==s[b]){a=!0;break}!a&&s.length>0&&cs.warn("Async pick readback returned only zero alpha values",{deviceRect:e,bytes:Array.from(s.subarray(0,Math.min(s.length,16)))})()}return{pickedColors:s,decodePickingColor:l}}async _readTextureDataAsync(a,b,c){let{width:d,height:e}=b,f=a.computeMemoryLayout(b),g=this.device.createBuffer({byteLength:f.byteLength,usage:hF.Buffer.COPY_DST|hF.Buffer.MAP_READ});try{a.readBuffer(b,g);let h=await g.readAsync(0,f.byteLength),i=c.BYTES_PER_ELEMENT;if(f.bytesPerRow%i!=0)throw Error(`Texture readback row stride ${f.bytesPerRow} is not aligned to ${i}-byte elements.`);let j=new c(h.buffer,h.byteOffset,f.byteLength/i),k=4*d,l=f.bytesPerRow/i;if(l<k)throw Error(`Texture readback row stride ${l} is smaller than packed row length ${k}.`);let m=new c(d*e*4);for(let a=0;a<e;a++){let b=a*l;m.set(j.subarray(b,b+k),a*k)}return m}finally{g.destroy()}}_drawAndSample({layers:a,views:b,viewports:c,onViewportActive:d,deviceRect:e,cullRect:f,effects:g,pass:h},i=!1){let j=i?this.depthFBO:this.pickingFBO,k={layers:a,layerFilter:this.layerFilter,views:b,viewports:c,onViewportActive:d,pickingFBO:j,deviceRect:e,cullRect:f,effects:g,pass:h,pickZ:i,preRenderStats:{},isPicking:!0};for(let a of g)a.useInPicking&&(k.preRenderStats[a.id]=a.preRender(k));let{decodePickingColor:l,stats:m}=this.pickLayersPass.render(k);this._updateStats(m);let{x:n,y:o,width:p,height:q}=e,r=new(i?Float32Array:Uint8Array)(p*q*4);return this.device.readPixelsToArrayWebGL(j,{sourceX:n,sourceY:o,sourceWidth:p,sourceHeight:q,target:r}),{pickedColors:r,decodePickingColor:l}}_updateStats(a){if(!this.stats)return;let b=0;for(let{visibleCount:c}of a)b+=c;this.stats.get("Layers picked").addCount(b)}_getDepthLayers(a,b,c){if(!c||!this.depthFBO)return[];let{pickedLayer:d}=a,e=d?.state?.terrainDrawMode==="drape";return d&&!e?[d]:b.filter(a=>a.props.operation.includes("terrain"))}_getPickingRect({deviceX:a,deviceY:b,deviceRadius:c,deviceWidth:d,deviceHeight:e}){let f=Math.max(0,a-c),g=Math.max(0,b-c),h=Math.min(d,a+c+1)-f,i=Math.min(e,b+c+1)-g;return h<=0||i<=0?null:{x:f,y:g,width:h,height:i}}}let hO={"top-left":{top:0,left:0},"top-right":{top:0,right:0},"bottom-left":{bottom:0,left:0},"bottom-right":{bottom:0,right:0},fill:{top:0,left:0,bottom:0,right:0}},hP="root";class hQ{constructor({deck:a,parentElement:b}){this.defaultWidgets=[],this.widgets=[],this.resolvedWidgets=[],this.containers={},this.lastViewports={},this.deck=a,b?.classList.add("deck-widget-container"),this.parentElement=b}getWidgets(){return this.resolvedWidgets}setProps(a){if(a.widgets&&!gD(a.widgets,this.widgets,1)){let b=a.widgets.filter(Boolean);this._setWidgets(b)}}finalize(){for(let a of this.getWidgets())this._removeWidget(a);for(let a in this.defaultWidgets.length=0,this.resolvedWidgets.length=0,this.containers)this.containers[a].remove()}addDefault(a){this.defaultWidgets.find(b=>b.id===a.id)||(this._addWidget(a),this.defaultWidgets.push(a),this._setWidgets(this.widgets))}onRedraw({viewports:a,layers:b}){let c=a.reduce((a,b)=>(a[b.id]=b,a),{});for(let d of this.getWidgets()){let{viewId:e}=d;if(e){let a=c[e];a&&(d.onViewportChange&&d.onViewportChange(a),d.onRedraw?.({viewports:[a],layers:b}))}else{if(d.onViewportChange)for(let b of a)d.onViewportChange(b);d.onRedraw?.({viewports:a,layers:b})}}this.lastViewports=c,this._updateContainers()}onHover(a,b){for(let c of this.getWidgets()){let{viewId:d}=c;d&&d!==a.viewport?.id||c.onHover?.(a,b)}}onEvent(a,b){let c=db[b.type];if(c)for(let d of this.getWidgets()){let{viewId:e}=d;e&&e!==a.viewport?.id||d[c]?.(a,b)}}_setWidgets(a){let b={};for(let a of this.resolvedWidgets)b[a.id]=a;for(let a of(this.resolvedWidgets.length=0,this.defaultWidgets))b[a.id]=null,this.resolvedWidgets.push(a);for(let c of a){let a=b[c.id];a?a.viewId!==c.viewId||a.placement!==c.placement?(this._removeWidget(a),this._addWidget(c)):c!==a&&(a.setProps(c.props),c=a):this._addWidget(c),b[c.id]=null,this.resolvedWidgets.push(c)}for(let a in b){let c=b[a];c&&this._removeWidget(c)}this.widgets=a}_addWidget(a){let{viewId:b=null,placement:c="top-left"}=a,d=a.props._container??b;a.widgetManager=this,a.deck=this.deck,a.rootElement=a._onAdd({deck:this.deck,viewId:b}),a.rootElement&&this._getContainer(d,c).append(a.rootElement),a.updateHTML()}_removeWidget(a){a.onRemove?.(),a.rootElement&&a.rootElement.remove(),a.rootElement=void 0,a.deck=void 0,a.widgetManager=void 0}_getContainer(a,b){if(a&&"string"!=typeof a)return a;let c=a||hP,d=this.containers[c];d||((d=document.createElement("div")).style.pointerEvents="none",d.style.position="absolute",d.style.overflow="hidden",this.parentElement?.append(d),this.containers[c]=d);let e=d.querySelector(`.${b}`);return e||((e=globalThis.document.createElement("div")).className=b,e.style.position="absolute",e.style.zIndex="2",Object.assign(e.style,hO[b]),d.append(e)),e}_updateContainers(){let a=this.deck.width,b=this.deck.height;for(let c in this.containers){let d=this.lastViewports[c]||null,e=c===hP||d,f=this.containers[c];e?(f.style.display="block",f.style.left=`${d?d.x:0}px`,f.style.top=`${d?d.y:0}px`,f.style.width=`${d?d.width:a}px`,f.style.height=`${d?d.height:b}px`):f.style.display="none"}}}function hR(a,b){b&&Object.entries(b).map(([b,c])=>{b.startsWith("--")?a.style.setProperty(b,c):a.style[b]=c})}class hS{constructor(a){this.viewId=null,this.props={...this.constructor.defaultProps,...a},this.id=this.props.id}setProps(a){let b=this.props,c=this.rootElement;if(c&&b.className!==a.className&&(b.className&&c.classList.remove(b.className),a.className&&c.classList.add(a.className)),c&&!gD(b.style,a.style,1)){var d;(d=b.style)&&Object.keys(d).map(a=>{a.startsWith("--")?c.style.removeProperty(a):c.style[a]=""}),hR(c,a.style)}Object.assign(this.props,a),this.updateHTML()}updateHTML(){this.rootElement&&this.onRenderHTML(this.rootElement)}get viewIds(){return this.viewId?[this.viewId]:this.deck?.getViews().map(a=>a.id)??[]}getViewState(a){return this.deck?.viewManager?.getViewState(a)||{}}setViewState(a,b){this.deck?._onViewStateChange({viewId:a,viewState:b,interactionState:{}})}onCreateRootElement(){let a=["deck-widget",this.className,this.props.className],b=document.createElement("div");return a.filter(a=>"string"==typeof a&&a.length>0).forEach(a=>b.classList.add(a)),hR(b,this.props.style),b}_onAdd(a){return this.onAdd(a)??this.onCreateRootElement()}onAdd(a){}onRemove(){}onViewportChange(a){}onRedraw(a){}onHover(a,b){}onClick(a,b){}onDrag(a,b){}onDragStart(a,b){}onDragEnd(a,b){}}hS.defaultProps={id:"widget",style:{},_container:null,className:""};let hT={zIndex:"1",position:"absolute",pointerEvents:"none",color:"#a0a7b4",backgroundColor:"#29323c",padding:"10px",top:"0",left:"0",display:"none"};class hU extends hS{constructor(a={}){super(a),this.id="default-tooltip",this.placement="fill",this.className="deck-tooltip",this.isVisible=!1,this.setProps(a)}onCreateRootElement(){let a=document.createElement("div");return a.className=this.className,Object.assign(a.style,hT),a}onRenderHTML(a){}onViewportChange(a){this.isVisible&&a.id===this.lastViewport?.id&&!a.equals(this.lastViewport)&&this.setTooltip(null),this.lastViewport=a}onHover(a){let{deck:b}=this,c=b&&b.props.getTooltip;if(!c)return;let d=c(a);this.setTooltip(d,a.x,a.y)}setTooltip(a,b,c){let d=this.rootElement;if(d){if("string"==typeof a)d.innerText=a;else if(a)a.text&&(d.innerText=a.text),a.html&&(d.innerHTML=a.html),a.className&&(d.className=a.className);else{this.isVisible=!1,d.style.display="none";return}this.isVisible=!0,d.style.display="block",d.style.transform=`translate(${b}px, ${c}px)`,a&&"object"==typeof a&&"style"in a&&Object.assign(d.style,a.style)}}}hU.defaultProps={...hS.defaultProps};let hV=globalThis.loaders?.parseImageNode,hW="u">typeof Image,hX="u">typeof ImageBitmap,hY=!!fs||!!hV,hZ=/^data:image\/svg\+xml/,h$=/\.svg((\?|#).*)?$/;function h_(a){return a&&(hZ.test(a)||h$.test(a))}function h0(a,b){if(h_(b))throw Error("SVG cannot be parsed directly to imagebitmap");return new Blob([new Uint8Array(a)])}async function h1(a,b,c){let d=function(a,b){if(h_(b)){let b=new TextDecoder().decode(a);try{"function"==typeof unescape&&"function"==typeof encodeURIComponent&&(b=unescape(encodeURIComponent(b)))}catch(a){throw Error(a.message)}return`data:image/svg+xml;base64,${btoa(b)}`}return h0(a,b)}(a,c),e=self.URL||self.webkitURL,f="string"!=typeof d&&e.createObjectURL(d);try{return await h2(f||d,b)}finally{f&&e.revokeObjectURL(f)}}async function h2(a,b){let c=new Image;return(c.src=a,b.image&&b.image.decode&&c.decode)?(await c.decode(),c):await new Promise((a,b)=>{try{c.onload=()=>a(c),c.onerror=a=>{let c=a instanceof Error?a.message:"error";b(Error(c))}}catch(a){b(a)}})}let h3=!0;async function h4(a,b,c){let d;d=h_(c)?await h1(a,b,c):h0(a,c);let e=b&&b.imagebitmap;return await h5(d,e)}async function h5(a,b=null){if((function(a){if(!a)return!0;for(let b in a)if(Object.prototype.hasOwnProperty.call(a,b))return!1;return!0}(b)||!h3)&&(b=null),b)try{return await createImageBitmap(a,b)}catch(a){console.warn(a),h3=!1}return await createImageBitmap(a)}function h6(a){var b,c;let d,e,f,g,h=h7(a);return((d=h7(h)).byteLength>=24&&0x89504e47===d.getUint32(0,!1)?{mimeType:"image/png",width:d.getUint32(16,!1),height:d.getUint32(20,!1)}:null)||function(a){let b=h7(a);if(!(b.byteLength>=3&&65496===b.getUint16(0,!1)&&255===b.getUint8(2)))return null;let{tableMarkers:c,sofMarkers:d}=function(){let a=new Set([65499,65476,65484,65501,65534]);for(let b=65504;b<65520;++b)a.add(b);return{tableMarkers:a,sofMarkers:new Set([65472,65473,65474,65475,65477,65478,65479,65481,65482,65483,65485,65486,65487,65502])}}(),e=2;for(;e+9<b.byteLength;){let a=b.getUint16(e,!1);if(d.has(a))return{mimeType:"image/jpeg",height:b.getUint16(e+5,!1),width:b.getUint16(e+7,!1)};if(!c.has(a))break;e+=2,e+=b.getUint16(e,!1)}return null}(h)||((e=h7(h)).byteLength>=10&&0x47494638===e.getUint32(0,!1)?{mimeType:"image/gif",width:e.getUint16(6,!0),height:e.getUint16(8,!0)}:null)||((f=h7(h)).byteLength>=14&&16973===f.getUint16(0,!1)&&f.getUint32(2,!0)===f.byteLength?{mimeType:"image/bmp",width:f.getUint32(18,!0),height:f.getUint32(22,!0)}:null)||((g=!function(a,b,c=0){let d=[...b].map(a=>a.charCodeAt(0));for(let b=0;b<d.length;++b)if(d[b]!==a[b+c])return!1;return!0}(c=new Uint8Array((b=h)instanceof DataView?b.buffer:b),"ftyp",4)||(96&c[8])==0?null:function(a){switch(String.fromCharCode(...a.slice(8,12)).replace("\0"," ").trim()){case"avif":case"avis":return{extension:"avif",mimeType:"image/avif"};default:return null}}(c))?{mimeType:g.mimeType,width:0,height:0}:null)}function h7(a){if(a instanceof DataView)return a;if(ArrayBuffer.isView(a))return new DataView(a.buffer);if(a instanceof ArrayBuffer)return new DataView(a);throw Error("toDataView")}async function h8(a,b){let{mimeType:c}=h6(a)||{},d=globalThis.loaders?.parseImageNode;return e6(d),await d(a,c)}let h9={dataType:null,batchType:null,id:"image",module:"images",name:"Images",version:"4.4.3",mimeTypes:["image/png","image/jpeg","image/gif","image/webp","image/avif","image/bmp","image/vnd.microsoft.icon","image/svg+xml"],extensions:["png","jpg","jpeg","gif","webp","bmp","ico","svg","avif"],parse:async function a(a,b,c){let d,e=((b=b||{}).image||{}).type||"auto",{url:f}=c||{};switch(function(a){switch(a){case"auto":case"data":if(hX)return"imagebitmap";if(hW)return"image";if(hY)return"data";throw Error("Install '@loaders.gl/polyfills' to parse images under Node.js");default:return!function(a){switch(a){case"auto":return hX;case"imagebitmap":case"image":case"data":return;default:throw Error(`@loaders.gl/images: image ${a} not supported in this environment`)}}(a),a}}(e)){case"imagebitmap":d=await h4(a,b,f);break;case"image":d=await h1(a,b,f);break;case"data":d=await h8(a,b);break;default:e6(!1)}return"data"===e&&(d=function(a){switch(function(a){var b;let c=(b=a,"u">typeof ImageBitmap&&b instanceof ImageBitmap?"imagebitmap":"u">typeof Image&&b instanceof Image?"image":b&&"object"==typeof b&&b.data&&b.width&&b.height?"data":null);if(!c)throw Error("Not an image");return c}(a)){case"data":return a;case"image":case"imagebitmap":let b=document.createElement("canvas"),c=b.getContext("2d");if(!c)throw Error("getImageData");return b.width=a.width,b.height=a.height,c.drawImage(a,0,0),c.getImageData(0,0,a.width,a.height);default:throw Error("getImageData")}}(d)),d},tests:[a=>!!h6(new DataView(a))],options:{image:{type:"auto",decode:!0}}},ia={dataType:null,batchType:null,id:"JSON",name:"JSON",module:"",version:"",options:{},extensions:["json","geojson"],mimeTypes:["application/json","application/geo+json"],testText:function(a){let b=a[0],c=a[a.length-1];return"{"===b&&"}"===c||"["===b&&"]"===c},parseTextSync:JSON.parse},ib=function(){let a="9.3.7",b=globalThis.deck&&globalThis.deck.VERSION;if(b&&b!==a)throw Error(`deck.gl - multiple versions detected: ${b} vs ${a}`);if(!b){cs.log(1,`deck.gl ${a}`)(),globalThis.deck={...globalThis.deck,VERSION:a,version:a,log:cs,_registerLoggers:eX};var c=[ia,[h9,{imagebitmap:{premultiplyAlpha:"none"}}]];let b=f9();for(let a of c=Array.isArray(c)?c:[c]){let c=e8(a);b.find(a=>c===a)||b.unshift(c)}}return a}();var ic=a.i(59181),id=a.i(31486);let ie="No matching device found. Ensure `@luma.gl/webgl` and/or `@luma.gl/webgpu` modules are imported.";class ig{static defaultProps={...ic.Device.defaultProps,type:"best-available",adapters:void 0,waitForPageLoad:!0};stats=id.lumaStats;log=bB.log;VERSION="9.3.6";spector;preregisteredAdapters=new Map;constructor(){if(globalThis.luma){if(globalThis.luma.VERSION!==this.VERSION)throw bB.log.error(`Found luma.gl ${globalThis.luma.VERSION} while initialzing ${this.VERSION}`)(),bB.log.error("'yarn why @luma.gl/core' can help identify the source of the conflict")(),Error("luma.gl - multiple versions detected: see console log");bB.log.error("This version of luma.gl has already been initialized")()}bB.log.log(1,`${this.VERSION} - set luma.log.level=1 (or higher) to trace rendering`)(),globalThis.luma=this}async createDevice(a={}){let b={...ig.defaultProps,...a},c=this.selectAdapter(b.type,b.adapters);if(!c)throw Error(ie);return b.waitForPageLoad&&await c.pageLoaded,await c.create(b)}async attachDevice(a,b){let c=this._getTypeFromHandle(a,b.adapters),d=c&&this.selectAdapter(c,b.adapters);if(!d)throw Error(ie);return await d?.attach?.(a,b)}registerAdapters(a){for(let b of a)this.preregisteredAdapters.set(b.type,b)}getSupportedAdapters(a=[]){return Array.from(this._getAdapterMap(a)).map(([,a])=>a).filter(a=>a.isSupported?.()).map(a=>a.type)}getBestAvailableAdapterType(a=[]){let b=this._getAdapterMap(a);for(let a of["webgpu","webgl","null"])if(b.get(a)?.isSupported?.())return a;return null}selectAdapter(a,b=[]){let c=a;"best-available"===a&&(c=this.getBestAvailableAdapterType(b));let d=this._getAdapterMap(b);return c&&d.get(c)||null}enforceWebGL2(a=!0,b=[]){let c=this._getAdapterMap(b).get("webgl");c||bB.log.warn("enforceWebGL2: webgl adapter not found")(),c?.enforceWebGL2?.(a)}setDefaultDeviceProps(a){Object.assign(ig.defaultProps,a)}_getAdapterMap(a=[]){let b=new Map(this.preregisteredAdapters);for(let c of a)b.set(c.type,c);return b}_getTypeFromHandle(a,b=[]){return a instanceof WebGL2RenderingContext?"webgl":"u">typeof GPUDevice&&a instanceof GPUDevice||a?.queue?"webgpu":null===a?"null":(a instanceof WebGLRenderingContext?bB.log.warn("WebGL1 is not supported",a)():bB.log.warn("Unknown handle type",a)(),null)}}let ih=new ig;var ii=a.i(66934);class ij{get pageLoaded(){return il||(ik&&document.readyState,il=Promise.resolve()),il}}let ik=(0,ii.isBrowser)()&&"u">typeof document,il=null,im={WEBGL_depth_texture:{UNSIGNED_INT_24_8_WEBGL:34042},OES_element_index_uint:{},OES_texture_float:{},OES_texture_half_float:{HALF_FLOAT_OES:5131},EXT_color_buffer_float:{},OES_standard_derivatives:{FRAGMENT_SHADER_DERIVATIVE_HINT_OES:35723},EXT_frag_depth:{},EXT_blend_minmax:{MIN_EXT:32775,MAX_EXT:32776},EXT_shader_texture_lod:{}};var io=a.i(4533),ip=a.i(81796);let iq=new class extends ij{type="webgl";constructor(){super(),ic.Device.defaultProps={...ic.Device.defaultProps,...io.DEFAULT_SPECTOR_PROPS}}enforceWebGL2(a){!function(a=!0){let b=HTMLCanvasElement.prototype;if(!a&&b.originalGetContext){b.getContext=b.originalGetContext,b.originalGetContext=void 0;return}b.originalGetContext=b.getContext,b.getContext=function(a,b){if("webgl"===a||"experimental-webgl"===a){let a=this.originalGetContext("webgl2",b);return a instanceof HTMLElement&&function(a){a.getExtension("EXT_color_buffer_float");let b={...im,WEBGL_disjoint_timer_query:a.getExtension("EXT_disjoint_timer_query_webgl2"),WEBGL_draw_buffers:{drawBuffersWEBGL:b=>a.drawBuffers(b),COLOR_ATTACHMENT0_WEBGL:36064,COLOR_ATTACHMENT1_WEBGL:36065,COLOR_ATTACHMENT2_WEBGL:36066,COLOR_ATTACHMENT3_WEBGL:36067},OES_vertex_array_object:{VERTEX_ARRAY_BINDING_OES:34229,createVertexArrayOES:()=>a.createVertexArray(),deleteVertexArrayOES:b=>a.deleteVertexArray(b),isVertexArrayOES:b=>a.isVertexArray(b),bindVertexArrayOES:b=>a.bindVertexArray(b)},ANGLE_instanced_arrays:{VERTEX_ATTRIB_ARRAY_DIVISOR_ANGLE:35070,drawArraysInstancedANGLE:(...b)=>a.drawArraysInstanced(...b),drawElementsInstancedANGLE:(...b)=>a.drawElementsInstanced(...b),vertexAttribDivisorANGLE:(...b)=>a.vertexAttribDivisor(...b)}},c=a.getExtension;a.getExtension=function(d){let e=c.call(a,d);return e||(d in b?b[d]:null)};let d=a.getSupportedExtensions;a.getSupportedExtensions=function(){let c=d.apply(a)||[];return c?.concat(Object.keys(b))}}(a),a}return this.originalGetContext(a,b)}}(a)}isSupported(){return"u">typeof WebGL2RenderingContext}isDeviceHandle(a){return!!("u">typeof WebGL2RenderingContext&&a instanceof WebGL2RenderingContext)||("u">typeof WebGLRenderingContext&&a instanceof WebGLRenderingContext&&bB.log.warn("WebGL1 is not supported",a)(),!1)}async attach(b,c={}){var d;let{WebGLDevice:e}=await a.A(56290);if(b instanceof e)return b;let f=e.getDeviceFromContext(b);if(f)return f;if(d=b,!("u">typeof WebGL2RenderingContext&&d instanceof WebGL2RenderingContext)&&(!d||"function"!=typeof d.createVertexArray))throw Error("Invalid WebGL2RenderingContext");let g=!0===c.createCanvasContext?{}:c.createCanvasContext;return new e({...c,_handle:b,createCanvasContext:{canvas:b.canvas,autoResize:!1,...g}})}async create(b={}){let{WebGLDevice:c}=await a.A(56290),d=[];for(let a of((b.debugWebGL||b.debug)&&d.push((0,ip.loadWebGLDeveloperTools)()),b.debugSpectorJS&&d.push((0,io.loadSpectorJS)(b)),await Promise.allSettled(d)))"rejected"===a.status&&bB.log.error(`Failed to initialize debug libraries ${a.reason}`)();try{let a=new c(b);bB.log.groupCollapsed(1,`WebGLDevice ${a.id} created`)();let d=`\
${a._reused?"Reusing":"Created"} device with WebGL2 ${a.props.debug?"debug ":""}context: \
${a.info.vendor}, ${a.info.renderer} for canvas: ${a.canvasContext.id}`;return bB.log.probe(1,d)(),bB.log.table(1,a.info)(),a}finally{bB.log.groupEnd(1)(),bB.log.info(1,"%cWebGL call tracing: luma.log.set('debug-webgl') ","color: white; background: blue; padding: 2px 6px; border-radius: 3px;")()}}},ir=0;class is{static defaultAnimationLoopProps={device:null,onAddHTML:()=>"",onInitialize:async()=>null,onRender:()=>{},onFinalize:()=>{},onError:a=>console.error(a),stats:void 0,autoResizeViewport:!1};device=null;canvas=null;props;animationProps=null;timeline=null;stats;sharedStats;cpuTime;gpuTime;frameRate;display;_needsRedraw="initialized";_initialized=!1;_running=!1;_animationFrameId=null;_nextFramePromise=null;_resolveNextFrame=null;_cpuStartTime=0;_error=null;_lastFrameTime=0;constructor(a){if(this.props={...is.defaultAnimationLoopProps,...a},!(a=this.props).device)throw Error("No device provided");this.stats=a.stats||new e$.Stats({id:`animation-loop-${ir++}`}),this.sharedStats=ih.stats.get("Animation Loop"),this.frameRate=this.stats.get("Frame Rate"),this.frameRate.setSampleSize(1),this.cpuTime=this.stats.get("CPU Time"),this.gpuTime=this.stats.get("GPU Time"),this.setProps({autoResizeViewport:a.autoResizeViewport}),this.start=this.start.bind(this),this.stop=this.stop.bind(this),this._onMousemove=this._onMousemove.bind(this),this._onMouseleave=this._onMouseleave.bind(this)}destroy(){this.stop(),this._setDisplay(null),this.device?._disableDebugGPUTime()}delete(){this.destroy()}reportError(a){this.props.onError(a),this._error=a}setNeedsRedraw(a){return this._needsRedraw=this._needsRedraw||a,this}needsRedraw(){let a=this._needsRedraw;return this._needsRedraw=!1,a}setProps(a){return"autoResizeViewport"in a&&(this.props.autoResizeViewport=a.autoResizeViewport||!1),this}async start(){if(this._running)return this;this._running=!0;try{let a;if(!this._initialized){if(this._initialized=!0,await this._initDevice(),this._initialize(),!this._running)return null;await this.props.onInitialize(this._getAnimationProps())}if(!this._running)return null;return!1!==a&&(this._cancelAnimationFrame(),this._requestAnimationFrame()),this}catch(b){let a=b instanceof Error?b:Error("Unknown error");throw this.props.onError(a),a}}stop(){return this._running&&(this.animationProps&&!this._error&&this.props.onFinalize(this.animationProps),this._cancelAnimationFrame(),this._nextFramePromise=null,this._resolveNextFrame=null,this._running=!1,this._lastFrameTime=0),this}redraw(a){return this.device?.isLost||this._error||(this._beginFrameTimers(a),this._setupFrame(),this._updateAnimationProps(),this._renderFrame(this._getAnimationProps()),this._clearNeedsRedraw(),this._resolveNextFrame&&(this._resolveNextFrame(this),this._nextFramePromise=null,this._resolveNextFrame=null),this._endFrameTimers()),this}attachTimeline(a){return this.timeline=a,this.timeline}detachTimeline(){this.timeline=null}waitForRender(){return this.setNeedsRedraw("waitForRender"),this._nextFramePromise||(this._nextFramePromise=new Promise(a=>{this._resolveNextFrame=a})),this._nextFramePromise}async toDataURL(){if(this.setNeedsRedraw("toDataURL"),await this.waitForRender(),this.canvas instanceof HTMLCanvasElement)return this.canvas.toDataURL();throw Error("OffscreenCanvas")}_initialize(){this._startEventHandling(),this._initializeAnimationProps(),this._updateAnimationProps(),this._resizeViewport(),this.device?._enableDebugGPUTime()}_setDisplay(a){this.display&&(this.display.destroy(),this.display.animationLoop=null),a&&(a.animationLoop=this),this.display=a}_requestAnimationFrame(){if(this._running){var a;this._animationFrameId=(a=this._animationFrame.bind(this),setTimeout(()=>a("u">typeof performance?performance.now():Date.now()),1e3/60))}}_cancelAnimationFrame(){null!==this._animationFrameId&&(clearTimeout(this._animationFrameId),this._animationFrameId=null)}_animationFrame(a){this._running&&(this.redraw(a),this._requestAnimationFrame())}_renderFrame(a){this.display?this.display._renderFrame(a):(this.props.onRender(this._getAnimationProps()),this.device?.submit())}_clearNeedsRedraw(){this._needsRedraw=!1}_setupFrame(){this._resizeViewport()}_initializeAnimationProps(){let a=this.device?.getDefaultCanvasContext();if(!this.device||!a)throw Error("loop");let b=a?.canvas,c=a.props.useDevicePixels;this.animationProps={animationLoop:this,device:this.device,canvasContext:a,canvas:b,useDevicePixels:c,timeline:this.timeline,needsRedraw:!1,width:1,height:1,aspect:1,time:0,startTime:Date.now(),engineTime:0,tick:0,tock:0,_mousePosition:null}}_getAnimationProps(){if(!this.animationProps)throw Error("animationProps");return this.animationProps}_updateAnimationProps(){if(!this.animationProps)return;let{width:a,height:b,aspect:c}=this._getSizeAndAspect();(a!==this.animationProps.width||b!==this.animationProps.height)&&this.setNeedsRedraw("drawing buffer resized"),c!==this.animationProps.aspect&&this.setNeedsRedraw("drawing buffer aspect changed"),this.animationProps.width=a,this.animationProps.height=b,this.animationProps.aspect=c,this.animationProps.needsRedraw=this._needsRedraw,this.animationProps.engineTime=Date.now()-this.animationProps.startTime,this.timeline&&this.timeline.update(this.animationProps.engineTime),this.animationProps.tick=Math.floor(this.animationProps.time/1e3*60),this.animationProps.tock++,this.animationProps.time=this.timeline?this.timeline.getTime():this.animationProps.engineTime}async _initDevice(){if(this.device=await this.props.device,!this.device)throw Error("No device provided");this.canvas=this.device.getDefaultCanvasContext().canvas||null}_createInfoDiv(){if(this.canvas&&this.props.onAddHTML){let a=document.createElement("div");document.body.appendChild(a),a.style.position="relative";let b=document.createElement("div");b.style.position="absolute",b.style.left="10px",b.style.bottom="10px",b.style.width="300px",b.style.background="white",this.canvas instanceof HTMLCanvasElement&&a.appendChild(this.canvas),a.appendChild(b);let c=this.props.onAddHTML(b);c&&(b.innerHTML=c)}}_getSizeAndAspect(){if(!this.device)return{width:1,height:1,aspect:1};let[a,b]=this.device.getDefaultCanvasContext().getDrawingBufferSize();return{width:a,height:b,aspect:a>0&&b>0?a/b:1}}_resizeViewport(){this.props.autoResizeViewport&&this.device.gl&&this.device.gl.viewport(0,0,this.device.gl.drawingBufferWidth,this.device.gl.drawingBufferHeight)}_beginFrameTimers(a){let b=a??("u">typeof performance?performance.now():Date.now());if(this._lastFrameTime){let a=b-this._lastFrameTime;a>0&&this.frameRate.addTime(a)}this._lastFrameTime=b,this.device?._isDebugGPUTimeEnabled()&&this._consumeEncodedGpuTime(),this.cpuTime.timeStart()}_endFrameTimers(){this.device?._isDebugGPUTimeEnabled()&&this._consumeEncodedGpuTime(),this.cpuTime.timeEnd(),this._updateSharedStats()}_consumeEncodedGpuTime(){if(!this.device)return;let a=this.device.commandEncoder._gpuTimeMs;void 0!==a&&(this.gpuTime.addTime(a),this.device.commandEncoder._gpuTimeMs=void 0)}_updateSharedStats(){if(this.stats!==this.sharedStats){for(let a of Object.keys(this.sharedStats.stats))this.stats.stats[a]||delete this.sharedStats.stats[a];this.stats.forEach(a=>{let b=this.sharedStats.get(a.name,a.type);b.sampleSize=a.sampleSize,b.time=a.time,b.count=a.count,b.samples=a.samples,b.lastTiming=a.lastTiming,b.lastSampleTime=a.lastSampleTime,b.lastSampleCount=a.lastSampleCount,b._count=a._count,b._time=a._time,b._samples=a._samples,b._startTime=a._startTime,b._timerPending=a._timerPending})}}_startEventHandling(){this.canvas&&(this.canvas.addEventListener("mousemove",this._onMousemove.bind(this)),this.canvas.addEventListener("mouseleave",this._onMouseleave.bind(this)))}_onMousemove(a){a instanceof MouseEvent&&(this._getAnimationProps()._mousePosition=[a.offsetX,a.offsetY])}_onMouseleave(a){this._getAnimationProps()._mousePosition=null}}function it(){}let iu={id:"",width:"100%",height:"100%",style:null,viewState:null,initialViewState:null,pickingRadius:0,pickAsync:"auto",layerFilter:null,parameters:{},parent:null,device:null,deviceProps:{},gl:null,canvas:null,layers:[],effects:[],views:null,controller:null,useDevicePixels:!0,touchAction:"none",eventRecognizerOptions:{},_framebuffer:null,_animate:!1,_pickable:!0,_typedArrayManagerProps:{},_customRender:null,widgets:[],onDeviceInitialized:it,onWebGLInitialized:it,onResize:it,onViewStateChange:it,onInteractionStateChange:it,onBeforeRender:it,onAfterRender:it,onLoad:it,onError:a=>cs.error(a.message,a.cause)(),onHover:null,onClick:null,onDragStart:null,onDrag:null,onDragEnd:null,_onMetrics:null,getCursor:({isDragging:a})=>a?"grabbing":"grab",getTooltip:null,debug:!1,drawPickingColors:!1};class iv{constructor(a){this.width=0,this.height=0,this.userData={},this.device=null,this.canvas=null,this.viewManager=null,this.layerManager=null,this.effectManager=null,this.deckRenderer=null,this.deckPicker=null,this.eventManager=null,this.widgetManager=null,this.tooltip=null,this.animationLoop=null,this._canvasContext=null,this._deviceResizeHandler=null,this.cursorState={isHovering:!1,isDragging:!1},this.stats=new e$.Stats({id:"deck.gl"}),this.metrics={fps:0,setPropsTime:0,layersCount:0,drawLayersCount:0,updateLayersCount:0,updateAttributesCount:0,updateAttributesTime:0,framesRedrawn:0,pickTime:0,pickCount:0,pickLayersCount:0,gpuTime:0,gpuTimePerFrame:0,cpuTime:0,cpuTimePerFrame:0,bufferMemory:0,textureMemory:0,renderbufferMemory:0,gpuMemory:0},this._metricsCounter=0,this._hoverPickSequence=0,this._pointerDownPickSequence=0,this._needsRedraw="Initial render",this._pickRequest={mode:"hover",x:-1,y:-1,radius:0,event:null,unproject3D:!1},this._lastPointerDownInfo=null,this._lastPointerDownInfoPromise=null,this._onPointerMove=a=>{let{_pickRequest:b}=this;if("pointerleave"===a.type)b.x=-1,b.y=-1,b.radius=0;else{if(a.leftButton||a.rightButton)return;let c=a.offsetCenter;if(!c)return;b.x=c.x,b.y=c.y,b.radius=this.props.pickingRadius}this.layerManager&&(this.layerManager.context.mousePosition={x:b.x,y:b.y}),b.event=a},this._onEvent=a=>{let b=db[a.type],c=a.offsetCenter;if(!b||!c||!this.layerManager)return;let d=this.layerManager.getLayers(),e=this._getInternalPickingMode();if(e){if("sync"===e){let b="click"===a.type&&this._shouldUnproject3D(d)?this._getFirstPickedInfo(this._pickPointSync(this._getPointPickOptions(c.x,c.y,{unproject3D:!0},d))):this._getLastPointerDownPickingInfo(c.x,c.y,d);this._dispatchPickingEvent(b,a);return}(this._lastPointerDownInfoPromise||Promise.resolve(this._getLastPointerDownPickingInfo(c.x,c.y,d))).then(b=>{this._dispatchPickingEvent(b,a)}).catch(a=>this.props.onError?.(a))}},this._onPointerDown=a=>{let b=a.offsetCenter;if(!b)return;let c=this._getInternalPickingMode();if(!c)return;let d=this.layerManager?.getLayers()||[],e=++this._pointerDownPickSequence;if("sync"===c){let a=this._pickPointSync({x:b.x,y:b.y,radius:this.props.pickingRadius}),c=this._getFirstPickedInfo(a);this._lastPointerDownInfo=c,this._lastPointerDownInfoPromise=Promise.resolve(c);return}let f=this._pickPointAsync(this._getPointPickOptions(b.x,b.y,{},d)).then(a=>this._getFirstPickedInfo(a)).then(a=>(e===this._pointerDownPickSequence&&(this._lastPointerDownInfo=a),a)).catch(a=>{this.props.onError?.(a);let c=this.deckPicker&&this.viewManager?this._getLastPointerDownPickingInfo(b.x,b.y,d):{};return e===this._pointerDownPickSequence&&(this._lastPointerDownInfo=c),c});this._lastPointerDownInfo=null,this._lastPointerDownInfoPromise=f};const b=a;this.props={...iu,...a},(a=this.props).viewState&&a.initialViewState&&cs.warn("View state tracking is disabled. Use either `initialViewState` for auto update or `viewState` for manual update.")(),this.viewState=this.props.initialViewState,a.device&&(this.device=a.device,this._setDeviceCanvasContext(a.device));let c=this.device;!c&&a.gl&&(a.gl instanceof WebGLRenderingContext&&cs.error("WebGL1 context not supported.")(),c=iq.attach(a.gl,{_cacheShaders:!0,_cachePipelines:!0,...this.props.deviceProps})),c||(c=this._createDevice(a)),this.animationLoop=this._createAnimationLoop(c,a),this.setProps(b),a._typedArrayManagerProps&&gp.setOptions(a._typedArrayManagerProps),this.animationLoop.start()}finalize(){this._restoreDeviceResizeHandler(),this.animationLoop?.stop(),this.animationLoop?.destroy(),this.animationLoop=null,this._hoverPickSequence++,this._pointerDownPickSequence++,this._lastPointerDownInfo=null,this._lastPointerDownInfoPromise=null,this.layerManager?.finalize(),this.layerManager=null,this.viewManager?.finalize(),this.viewManager=null,this.effectManager?.finalize(),this.effectManager=null,this.deckRenderer?.finalize(),this.deckRenderer=null,this.deckPicker?.finalize(),this.deckPicker=null,this.eventManager?.destroy(),this.eventManager=null,this.widgetManager?.finalize(),this.widgetManager=null,this.props.canvas||this.props.device||this.props.gl||!this.canvas||(this.canvas.parentElement?.removeChild(this.canvas),this.canvas=null),this._canvasContext=null}setProps(a){this.stats.get("setProps Time").timeStart(),"onLayerHover"in a&&cs.removed("onLayerHover","onHover")(),"onLayerClick"in a&&cs.removed("onLayerClick","onClick")(),a.initialViewState&&!gD(this.props.initialViewState,a.initialViewState,3)&&(this.viewState=a.initialViewState),Object.assign(this.props,a),this._validateInternalPickingMode(),this._setCanvasSize(this.props);let b=Object.create(this.props);if(Object.assign(b,{views:this._getViews(),width:this.width,height:this.height,viewState:this._getViewState()}),a.device&&a.device.id!==this.device?.id){let b=a.device.getDefaultCanvasContext();this.animationLoop?.stop(),this.canvas!==b.canvas&&(this.canvas?.remove(),this.eventManager?.destroy(),this.canvas=null),this._setDeviceCanvasContext(a.device),cs.log(`recreating animation loop for new device! id=${a.device.id}`)(),this.animationLoop=this._createAnimationLoop(a.device,a),this.animationLoop.start()}this.animationLoop?.setProps(b),void 0!==a.useDevicePixels&&this._canvasContext?.setProps&&this._canvasContext.setProps({useDevicePixels:a.useDevicePixels}),this.layerManager&&(this.viewManager.setProps(b),this.layerManager.activateViewport(this.getViewports()[0]),this.layerManager.setProps(b),this.effectManager.setProps(b),this.deckRenderer.setProps(b),this.deckPicker.setProps(b),this.widgetManager.setProps(b)),this.stats.get("setProps Time").timeEnd()}needsRedraw(a={clearRedrawFlags:!1}){if(!this.layerManager)return!1;if(this.props._animate)return"Deck._animate";let b=this._needsRedraw;a.clearRedrawFlags&&(this._needsRedraw=!1);let c=this.viewManager.needsRedraw(a),d=this.layerManager.needsRedraw(a),e=this.effectManager.needsRedraw(a),f=this.deckRenderer.needsRedraw(a);return b||c||d||e||f}redraw(a){if(!this.layerManager)return;let b=this.needsRedraw({clearRedrawFlags:!0});(b=a||b)&&(this.stats.get("Redraw Count").incrementCount(),this.props._customRender?this.props._customRender(b):this._drawLayers(b))}get isInitialized(){return null!==this.viewManager}getViews(){return gT(this.viewManager),this.viewManager.views}getView(a){return gT(this.viewManager),this.viewManager.getView(a)}getViewports(a){return gT(this.viewManager),this.viewManager.getViewports(a)}getCanvas(){return this.canvas}async pickObjectAsync(a){let b=(await this._pickAsync("pickObjectAsync","pickObject Time",a)).result;return b.length?b[0]:null}async pickObjectsAsync(a){return await this._pickAsync("pickObjectsAsync","pickObjects Time",a)}pickObject(a){let b=this._pick("pickObject","pickObject Time",a).result;return b.length?b[0]:null}pickMultipleObjects(a){return a.depth=a.depth||10,this._pick("pickObject","pickMultipleObjects Time",a).result}pickObjects(a){return this._pick("pickObjects","pickObjects Time",a)}_pickPositionForController(a,b){return"sync"!==this._getInternalPickingMode()?null:this.pickObject({x:a,y:b,radius:0,unproject3D:!0})}_addResources(a,b=!1){for(let c in a)this.layerManager.resourceManager.add({resourceId:c,data:a[c],forceUpdate:b})}_removeResources(a){for(let b of a)this.layerManager.resourceManager.remove(b)}_addDefaultEffect(a){this.effectManager.addDefaultEffect(a)}_addDefaultShaderModule(a){this.layerManager.addDefaultShaderModule(a)}_removeDefaultShaderModule(a){this.layerManager?.removeDefaultShaderModule(a)}_resolveInternalPickingMode(){let{pickAsync:a}=this.props,b=this.device?.type||this.props.deviceProps?.type;if("auto"===a)return"webgpu"===b?"async":"sync";if("sync"===a&&"webgpu"===b)throw Error('`pickAsync: "sync"` is not supported when Deck is using a WebGPU device.');return a}_getInternalPickingMode(){try{return this._resolveInternalPickingMode()}catch(a){return this.props.onError?.(a),null}}_validateInternalPickingMode(){this._getInternalPickingMode()}_getFirstPickedInfo({result:a,emptyInfo:b}){return a[0]||b}_shouldUnproject3D(a=this.layerManager?.getLayers()||[]){return a.some(a=>"3d"===a.props.pickable)}_getPointPickOptions(a,b,c={},d=this.layerManager?.getLayers()||[]){return{x:a,y:b,radius:this.props.pickingRadius,unproject3D:this._shouldUnproject3D(d),...c}}_pickPointSync(a){return this._pick("pickObject","pickObject Time",a)}_pickPointAsync(a){return this._pickAsync("pickObjectAsync","pickObject Time",a)}_getLastPointerDownPickingInfo(a,b,c=this.layerManager?.getLayers()||[]){return this.deckPicker.getLastPickedObject({x:a,y:b,layers:c,viewports:this.getViewports({x:a,y:b})},this._lastPointerDownInfo)}_applyHoverCallbacks({result:a,emptyInfo:b},c){if(!this.widgetManager)return;this.cursorState.isHovering=a.length>0;let d=b,e=!1;for(let b of a)d=b,e=b.layer?.onHover(b,c)||e;e||(this.props.onHover?.(d,c),this.widgetManager.onHover(d,c))}_dispatchPickingEvent(a,b){if(!this.layerManager||!this.widgetManager)return;let c=db[b.type];if(!c)return;let{layer:d}=a,e=d&&(d[c]||d.props[c]),f=this.props[c],g=!1;e&&(g=e.call(d,a,b)),g||(f?.(a,b),this.widgetManager.onEvent(a,b))}_pickAsync(a,b,c){gT(this.deckPicker);let{stats:d}=this;d.get("Pick Count").incrementCount(),d.get(b).timeStart();let e=this.deckPicker[a]({layers:this.layerManager.getLayers(c),views:this.viewManager.getViews(),viewports:this.getViewports(c),onViewportActive:this.layerManager.activateViewport,effects:this.effectManager.getEffects(),...c,canvasContext:this._canvasContext||void 0});return d.get(b).timeEnd(),e}_pick(a,b,c){gT(this.deckPicker);let{stats:d}=this;d.get("Pick Count").incrementCount(),d.get(b).timeStart();let e=this.deckPicker[a]({layers:this.layerManager.getLayers(c),views:this.viewManager.getViews(),viewports:this.getViewports(c),onViewportActive:this.layerManager.activateViewport,effects:this.effectManager.getEffects(),...c,canvasContext:this._canvasContext||void 0});return d.get(b).timeEnd(),e}_createCanvas(a){let b=a.canvas;return"string"==typeof b&&gT(b=document.getElementById(b)),b||((b=document.createElement("canvas")).id=a.id||"deckgl-overlay",a.width&&"number"==typeof a.width&&(b.width=a.width),a.height&&"number"==typeof a.height&&(b.height=a.height),(a.parent||document.body).appendChild(b)),Object.assign(b.style,a.style),b}_setCanvasContext(a){this._canvasContext=a,"style"in a.canvas&&(this.canvas=a.canvas)}_setDeviceCanvasContext(a,b={}){let c=a.getDefaultCanvasContext();this._setCanvasContext(c),this._setDeviceResizeHandler(a,b)}_setDeviceResizeHandler(a,b={}){let c=!!b.syncDrawingBuffer;if(this._deviceResizeHandler?.device===a){this._deviceResizeHandler.syncDrawingBuffer=c;return}this._restoreDeviceResizeHandler();let d=a=>{a===this._canvasContext&&this._canvasContext&&this._onCanvasContextResize(this._canvasContext,{syncDrawingBuffer:this._deviceResizeHandler?.syncDrawingBuffer})};a.props.onResize=d,this._deviceResizeHandler={device:a,onResize:d,syncDrawingBuffer:c}}_restoreDeviceResizeHandler(){let a=this._deviceResizeHandler;a&&a.device.props?.onResize===a.onResize&&(a.device.props.onResize=it),this._deviceResizeHandler=null}_setCanvasSize(a){if(!this.canvas)return;let{width:b,height:c}=a;if(b||0===b){let a=Number.isFinite(b)?`${b}px`:b;this.canvas.style.width=a}if(c||0===c){let b=Number.isFinite(c)?`${c}px`:c;this.canvas.style.position=a.style?.position||"absolute",this.canvas.style.height=b}}_updateCanvasSize(a=this._canvasContext){let{canvas:b}=this,[c,d]=a?a.getCSSSize():[b?.clientWidth??b?.width??0,b?.clientHeight??b?.height??0];(c!==this.width||d!==this.height)&&(this.width=c,this.height=d,this.viewManager?.setProps({width:c,height:d}),this.layerManager?.activateViewport(this.getViewports()[0]),this.props.onResize({width:c,height:d},a||void 0))}_onCanvasContextResize(a,b={}){if(b.syncDrawingBuffer){let{width:b,height:c}=a.canvas;a.setDrawingBufferSize(b,c)}this._needsRedraw="Canvas resized",this._updateCanvasSize(a)}_createAnimationLoop(a,b){let{gl:c,onError:d}=b;return new is({device:a,autoResizeDrawingBuffer:!c,autoResizeViewport:!1,onInitialize:a=>this._setDevice(a.device),onRender:this._onRenderFrame.bind(this),onError:d})}_createDevice(a){let b=this.props.deviceProps?.createCanvasContext,c={adapters:[],_cacheShaders:!0,_cachePipelines:!0,...a.deviceProps};c.adapters.includes(iq)||c.adapters.push(iq);let d={alphaMode:this.props.deviceProps?.type==="webgpu"?"premultiplied":void 0};return ih.createDevice({_reuseDevices:!0,type:"webgl",...c,createCanvasContext:{...d,..."object"==typeof b?b:void 0,canvas:this._createCanvas(a),useDevicePixels:this.props.useDevicePixels,autoResize:!0}})}_getViewState(){return this.props.viewState||this.viewState}_getViews(){let{views:a}=this.props,b=Array.isArray(a)?a:a?[a]:[new hg({id:"default-view"})];return b.length&&this.props.controller&&(b[0].props.controller=this.props.controller),b}_onContextLost(){let{onError:a}=this.props;this.animationLoop&&a&&a(Error("WebGL context is lost"))}_pickAndCallback(){let{_pickRequest:a}=this;if(a.event){let b=a.event,c=this.layerManager?.getLayers()||[],d=this._getPointPickOptions(a.x,a.y,{radius:a.radius,mode:a.mode},c),e=this._getInternalPickingMode(),f=++this._hoverPickSequence;if(a.event=null,!e)return;if("sync"===e)return void this._applyHoverCallbacks(this._pickPointSync(d),b);this._pickPointAsync(d).then(({result:a,emptyInfo:c})=>{f===this._hoverPickSequence&&this._applyHoverCallbacks({result:a,emptyInfo:c},b)}).catch(a=>this.props.onError?.(a))}}_updateCursor(){let a=this.props.parent||this.canvas;a&&(a.style.cursor=this.props.getCursor(this.cursorState))}_setDevice(a){if(this.device=a,this._validateInternalPickingMode(),!this.animationLoop)return;this._setDeviceCanvasContext(a,{syncDrawingBuffer:!!(this.props.gl&&this.props.device!==a)}),this.canvas&&!this.canvas.isConnected&&this.props.parent&&this.props.parent.insertBefore(this.canvas,this.props.parent.firstChild),"webgl"===this.device.type&&this.device.setParametersWebGL({blend:!0,blendFunc:[770,771,1,771],polygonOffsetFill:!0,depthTest:!0,depthFunc:515}),this.props.onDeviceInitialized(this.device),"webgl"===this.device.type&&this.props.onWebGLInitialized(this.device.gl);let b=new aH;b.play(),this.animationLoop.attachTimeline(b);let c=this.props.parent?.querySelector(".deck-events-root")||this.canvas;for(let a in this.eventManager=new c7(c,{touchAction:this.props.touchAction,recognizers:Object.keys(dc).map(a=>{let[b,c,d,e]=dc[a],f=this.props.eventRecognizerOptions?.[a];return{recognizer:new b({...c,...f,event:a}),recognizeWith:d,requireFailure:e}}),events:{pointerdown:this._onPointerDown,pointermove:this._onPointerMove,pointerleave:this._onPointerMove}}),db)this.eventManager.on(a,this._onEvent);this.viewManager=new gE({timeline:b,eventManager:this.eventManager,onViewStateChange:this._onViewStateChange.bind(this),onInteractionStateChange:this._onInteractionStateChange.bind(this),pickPosition:this._pickPositionForController.bind(this),views:this._getViews(),viewState:this._getViewState(),width:this.width,height:this.height});let d=this.viewManager.getViewports()[0];this.layerManager=new gC(this.device,{deck:this,stats:this.stats,viewport:d,timeline:b}),this.effectManager=new hy({deck:this,device:this.device}),this.deckRenderer=new hE(this.device,{stats:this.stats}),this.deckPicker=new hN(this.device,{stats:this.stats});let e=this.props.parent?.querySelector(".deck-widgets-root")||this.canvas?.parentElement;this.widgetManager=new hQ({deck:this,parentElement:e}),this.widgetManager.addDefault(new hU),this.setProps({}),this._updateCanvasSize(this._canvasContext),this.props.onLoad()}_drawLayers(a,b){let{device:c,gl:d}=this.layerManager.context;this.props.onBeforeRender({device:c,gl:d});let e={target:this.props._framebuffer,layers:this.layerManager.getLayers(),viewports:this.viewManager.getViewports(),onViewportActive:this.layerManager.activateViewport,views:this.viewManager.getViews(),pass:"screen",effects:this.effectManager.getEffects(),...b};this.deckRenderer?.renderLayers(e),"screen"===e.pass&&this.widgetManager.onRedraw({viewports:e.viewports,layers:e.layers}),this.props.onAfterRender({device:c,gl:d})}_onRenderFrame(){this._getFrameStats(),this._metricsCounter++%60==0&&(this._getMetrics(),this.stats.reset(),cs.table(4,this.metrics)(),this.props._onMetrics&&this.props._onMetrics(this.metrics)),this._updateCursor(),this.layerManager.updateLayers(),this._pickAndCallback(),this.redraw(),this.viewManager&&this.viewManager.updateViewStates()}_onViewStateChange(a){let b=this.props.onViewStateChange(a)||a.viewState;this.viewState&&(this.viewState={...this.viewState,[a.viewId]:b},!this.props.viewState&&this.viewManager&&this.viewManager.setProps({viewState:this.viewState}))}_onInteractionStateChange(a){this.cursorState.isDragging=a.isDragging||!1,this.props.onInteractionStateChange(a)}_getFrameStats(){let{stats:a}=this;a.get("frameRate").timeEnd(),a.get("frameRate").timeStart();let b=this.animationLoop.stats;a.get("GPU Time").addTime(b.get("GPU Time").lastTiming),a.get("CPU Time").addTime(b.get("CPU Time").lastTiming)}_getMetrics(){let{metrics:a,stats:b}=this;a.fps=b.get("frameRate").getHz(),a.setPropsTime=b.get("setProps Time").time,a.updateAttributesTime=b.get("Update Attributes").time,a.framesRedrawn=b.get("Redraw Count").count,a.pickTime=b.get("pickObject Time").time+b.get("pickMultipleObjects Time").time+b.get("pickObjects Time").time,a.pickCount=b.get("Pick Count").count,a.layersCount=this.layerManager?.layers.length??0,a.drawLayersCount=b.get("Layers rendered").lastSampleCount,a.pickLayersCount=b.get("Layers picked").lastSampleCount,a.updateAttributesCount=b.get("Layers updated").count,a.updateAttributesCount=b.get("Attributes updated").count,a.gpuTime=b.get("GPU Time").time,a.cpuTime=b.get("CPU Time").time,a.gpuTimePerFrame=b.get("GPU Time").getAverageTime(),a.cpuTimePerFrame=b.get("CPU Time").getAverageTime();let c=ih.stats.get("GPU Time and Memory");a.bufferMemory=c.get("Buffer Memory").count,a.textureMemory=c.get("Texture Memory").count,a.renderbufferMemory=c.get("Renderbuffer Memory").count,a.gpuMemory=c.get("GPU Memory").count}}iv.defaultProps=iu,iv.VERSION=ib;let iw=s.useEffect;function ix(a,b){for(;a;){if(a===b)return!0;a=Object.getPrototypeOf(a)}return!1}var iy=a.i(41501),iz=a.i(68295),iA=a.i(24759);let iB=iz.dataTypeDecoder.getDataType.bind(iz.dataTypeDecoder);function iC(a,b,c){if(b.size>4)return null;let d="webgpu"===c&&"uint8"===b.type?"unorm8":b.type;return{attribute:a,format:b.size>1?`${d}x${b.size}`:b.type,byteOffset:b.offset||0}}function iD(a){return a.stride||a.size*a.bytesPerElement}function iE(a,b){b.offset&&cs.removed("shaderAttribute.offset","vertexOffset, elementOffset")();let c=iD(a),d=(void 0!==b.vertexOffset?b.vertexOffset:a.vertexOffset||0)*c+(b.elementOffset||0)*a.bytesPerElement+(a.offset||0);return{...b,offset:d,stride:c}}class iF{constructor(a,b,c){let d;this._buffer=null,this.device=a,this.id=b.id||"",this.size=b.size||1;const e=b.logicalType||b.type,f="float64"===e;let{defaultValue:g}=b;g=Number.isFinite(g)?[g]:g||Array(this.size).fill(0),d=f?"float32":!e&&b.isIndexed?"uint32":e||"float32";let h=function(a){switch(a){case"float64":return Float64Array;case"uint8":case"unorm8":return Uint8ClampedArray;default:return(0,iA.getTypedArrayConstructor)(a)}}(e||d);this.doublePrecision=f,f&&!1===b.fp64&&(h=Float32Array),this.value=null,this.settings={...b,defaultType:h,defaultValue:g,logicalType:e,type:d,normalized:d.includes("norm"),size:this.size,bytesPerElement:h.BYTES_PER_ELEMENT},this.state={...c,externalBuffer:null,bufferAccessor:this.settings,allocatedValue:null,numInstances:0,bounds:null,constant:!1}}get isConstant(){return this.state.constant}get buffer(){return this._buffer}get byteOffset(){let a=this.getAccessor();return a.vertexOffset?a.vertexOffset*iD(a):0}get numInstances(){return this.state.numInstances}set numInstances(a){this.state.numInstances=a}delete(){this._buffer&&(this._buffer.delete(),this._buffer=null),gp.release(this.state.allocatedValue)}getBuffer(){return this.state.constant?null:this.state.externalBuffer||this._buffer}getValue(a=this.id,b=null){let c={};if(this.state.constant){let d=this.value;if(b){let e=iE(this.getAccessor(),b),f=e.offset/d.BYTES_PER_ELEMENT,g=e.size||this.size;c[a]=d.subarray(f,f+g)}else c[a]=d}else c[a]=this.getBuffer();return this.doublePrecision&&(this.value instanceof Float64Array?c[`${a}64Low`]=c[a]:c[`${a}64Low`]=new Float32Array(this.size)),c}_getBufferLayout(a=this.id,b=null){let c=this.getAccessor(),d=[],e={name:this.id,byteStride:iD(c)};if(this.doublePrecision){let e,f={high:e=iE(c,b||{}),low:{...e,offset:e.offset+4*c.size}};d.push(iC(a,{...c,...f.high},this.device.type),iC(`${a}64Low`,{...c,...f.low},this.device.type))}else if(b){let e=iE(c,b);d.push(iC(a,{...c,...e},this.device.type))}else d.push(iC(a,c,this.device.type));return e.attributes=d.filter(Boolean),e}setAccessor(a){this.state.bufferAccessor=a}getAccessor(){return this.state.bufferAccessor}getBounds(){if(this.state.bounds)return this.state.bounds;let a=null;if(this.state.constant&&this.value){let b=Array.from(this.value);a=[b,b]}else{let{value:b,numInstances:c,size:d}=this,e=c*d;if(b&&e&&b.length>=e){let c=Array(d).fill(1/0),f=Array(d).fill(-1/0);for(let a=0;a<e;)for(let e=0;e<d;e++){let d=b[a++];d<c[e]&&(c[e]=d),d>f[e]&&(f[e]=d)}a=[c,f]}}return this.state.bounds=a,a}setData(a){let b,{state:c}=this;b=ArrayBuffer.isView(a)?{value:a}:a instanceof hF.Buffer?{buffer:a}:a;let d={...this.settings,...b};if(ArrayBuffer.isView(b.value)){if(!b.type)if(this.doublePrecision&&b.value instanceof Float64Array)d.type="float32";else{let a=iB(b.value);d.type=d.normalized?a.replace("int","norm"):a}d.bytesPerElement=b.value.BYTES_PER_ELEMENT,d.stride=iD(d)}if(c.bounds=null,b.constant){let a=b.value;if(a=this._normalizeValue(a,[],0),this.settings.normalized&&(a=this.normalizeConstant(a)),!(!c.constant||!this._areValuesEqual(a,this.value)))return!1;c.externalBuffer=null,c.constant=!0,this.value=ArrayBuffer.isView(a)?a:new Float32Array(a)}else if(b.buffer)c.externalBuffer=b.buffer,c.constant=!1,this.value=b.value||null;else if(b.value){this._checkExternalBuffer(b);let a=b.value;c.externalBuffer=null,c.constant=!1,this.value=a;let{buffer:e}=this,f=iD(d),g=(d.vertexOffset||0)*f;if(this.doublePrecision&&a instanceof Float64Array&&(a=gu(a,d)),this.settings.isIndexed){let b=this.settings.defaultType;a.constructor!==b&&(a=new b(a))}let h=a.byteLength+g+2*f;(!e||e.byteLength<h)&&(e=this._createBuffer(h)),e.write(a,g)}return this.setAccessor(d),!0}updateSubBuffer(a={}){this.state.bounds=null;let b=this.value,{startOffset:c=0,endOffset:d}=a;this.buffer.write(this.doublePrecision&&b instanceof Float64Array?gu(b,{size:this.size,startIndex:c,endIndex:d}):b.subarray(c,d),c*b.BYTES_PER_ELEMENT+this.byteOffset)}allocate(a,b=!1){let{state:c}=this,d=c.allocatedValue,e=gp.allocate(d,a+1,{size:this.size,type:this.settings.defaultType,copy:b});this.value=e;let{byteOffset:f}=this,{buffer:g}=this;return(!g||g.byteLength<e.byteLength+f)&&(g=this._createBuffer(e.byteLength+f),b&&d&&g.write(d instanceof Float64Array?gu(d,this):d,f)),c.allocatedValue=e,c.constant=!1,c.externalBuffer=null,this.setAccessor(this.settings),!0}_checkExternalBuffer(a){let{value:b}=a;if(!ArrayBuffer.isView(b))throw Error(`Attribute ${this.id} value is not TypedArray`);let c=this.settings.defaultType,d=!1;if(this.doublePrecision&&(d=b.BYTES_PER_ELEMENT<4),d)throw Error(`Attribute ${this.id} does not support ${b.constructor.name}`);b instanceof c||!this.settings.normalized||"normalized"in a||cs.warn(`Attribute ${this.id} is normalized`)()}normalizeConstant(a){switch(this.settings.type){case"snorm8":return new Float32Array(a).map(a=>(a+128)/255*2-1);case"snorm16":return new Float32Array(a).map(a=>(a+32768)/65535*2-1);case"unorm8":return new Float32Array(a).map(a=>a/255);case"unorm16":return new Float32Array(a).map(a=>a/65535);default:return a}}_normalizeValue(a,b,c){let{defaultValue:d,size:e}=this.settings;if(Number.isFinite(a))return b[c]=a,b;if(!a){let a=e;for(;--a>=0;)b[c+a]=d[a];return b}switch(e){case 4:b[c+3]=Number.isFinite(a[3])?a[3]:d[3];case 3:b[c+2]=Number.isFinite(a[2])?a[2]:d[2];case 2:b[c+1]=Number.isFinite(a[1])?a[1]:d[1];case 1:b[c+0]=Number.isFinite(a[0])?a[0]:d[0];break;default:let f=e;for(;--f>=0;)b[c+f]=Number.isFinite(a[f])?a[f]:d[f]}return b}_areValuesEqual(a,b){if(!a||!b)return!1;let{size:c}=this;for(let d=0;d<c;d++)if(a[d]!==b[d])return!1;return!0}_createBuffer(a){this._buffer&&this._buffer.destroy();let{isIndexed:b,type:c}=this.settings;return this._buffer=this.device.createBuffer({...this._buffer?.props,id:this.id,usage:(b?hF.Buffer.INDEX:hF.Buffer.VERTEX)|hF.Buffer.COPY_DST,indexType:b?c:void 0,byteLength:a}),this._buffer}}let iG=[],iH=[];function iI(a,b=0,c=1/0){let d=iG,e={index:-1,data:a,target:[]};return a?"function"==typeof a[Symbol.iterator]?d=a:a.length>0&&(iH.length=a.length,d=iH):d=iG,(b>0||Number.isFinite(c))&&(d=(Array.isArray(d)?d:Array.from(d)).slice(b,c),e.index=b-1),{iterable:d,objectInfo:e}}function iJ(a){return a&&a[Symbol.asyncIterator]}function iK(a,b){let{size:c,stride:d,offset:e,startIndices:f,nested:g}=b,h=a.BYTES_PER_ELEMENT,i=d?d/h:c,j=e?e/h:0,k=Math.floor((a.length-j)/i);return(b,{index:d,target:e})=>{let h;if(!f){let b=d*i+j;for(let d=0;d<c;d++)e[d]=a[b+d];return e}let l=f[d],m=f[d+1]||k;if(g){h=Array(m-l);for(let b=l;b<m;b++){let d=b*i+j;e=Array(c);for(let b=0;b<c;b++)e[b]=a[d+b];h[b-l]=e}}else if(i===c)h=a.subarray(l*c+j,m*c+j);else{h=new a.constructor((m-l)*c);let b=0;for(let d=l;d<m;d++){let e=d*i+j;for(let d=0;d<c;d++)h[b++]=a[e+d]}}return h}}let iL=[],iM=[[0,1/0]],iN={interpolation:{duration:0,easing:a=>a},spring:{stiffness:.05,damping:.5}};function iO(a,b){if(!a)return null;Number.isFinite(a)&&(a={type:"interpolation",duration:a});let c=a.type||"interpolation";return{...iN[c],...b,...a,type:c}}class iP extends iF{constructor(a,b){super(a,b,{startIndices:null,lastExternalBuffer:null,binaryValue:null,binaryAccessor:null,needsUpdate:!0,needsRedraw:!1,layoutChanged:!1,updateRanges:iM}),this.constant=!1,this.settings.update=b.update||(b.accessor?this._autoUpdater:void 0),Object.seal(this.settings),Object.seal(this.state),this._validateAttributeUpdaters()}get startIndices(){return this.state.startIndices}set startIndices(a){this.state.startIndices=a}needsUpdate(){return this.state.needsUpdate}needsRedraw({clearChangedFlags:a=!1}={}){let b=this.state.needsRedraw;return this.state.needsRedraw=b&&!a,b}layoutChanged(){return this.state.layoutChanged}setAccessor(a){var b,c;(b=this.state).layoutChanged||(c=this.getAccessor(),b.layoutChanged=a.type!==c.type||a.size!==c.size||iD(a)!==iD(c)||(a.offset||0)!==(c.offset||0)),super.setAccessor(a)}getUpdateTriggers(){let{accessor:a}=this.settings;return[this.id].concat("function"!=typeof a&&a||[])}supportsTransition(){return!!this.settings.transition}getTransitionSetting(a){if(!a||!this.supportsTransition())return null;let{accessor:b}=this.settings,c=this.settings.transition;return iO(Array.isArray(b)?a[b.find(b=>a[b])]:a[b],c)}setNeedsUpdate(a=this.id,b){if(this.state.needsUpdate=this.state.needsUpdate||a,this.setNeedsRedraw(a),b){let{startRow:a=0,endRow:c=1/0}=b;this.state.updateRanges=function(a,b){if(a===iM||(b[0]<0&&(b[0]=0),b[0]>=b[1]))return a;let c=[],d=a.length,e=0;for(let f=0;f<d;f++){let d=a[f];d[1]<b[0]?(c.push(d),e=f+1):d[0]>b[1]?c.push(d):b=[Math.min(d[0],b[0]),Math.max(d[1],b[1])]}return c.splice(e,0,b),c}(this.state.updateRanges,[a,c])}else this.state.updateRanges=iM}clearNeedsUpdate(){this.state.needsUpdate=!1,this.state.updateRanges=iL}setNeedsRedraw(a=this.id){this.state.needsRedraw=this.state.needsRedraw||a}allocate(a){let{state:b,settings:c}=this;return!c.noAlloc&&!!c.update&&(super.allocate(a,b.updateRanges!==iM),!0)}updateBuffer({numInstances:a,data:b,props:c,context:d}){if(!this.needsUpdate())return!1;let{state:{updateRanges:e},settings:{update:f,noAlloc:g}}=this,h=!0;if(f){for(let[g,h]of e)f.call(d,this,{data:b,startRow:g,endRow:h,props:c,numInstances:a});if(this.value)if(this.constant||!this.buffer||this.buffer.byteLength<this.value.byteLength+this.byteOffset)this.constant?this.setConstantValue(d,this.value):this.setData({value:this.value,constant:this.constant}),this.constant=!1;else for(let[b,c]of e){let d=Number.isFinite(b)?this.getVertexOffset(b):0,e=Number.isFinite(c)?this.getVertexOffset(c):g||!Number.isFinite(a)?this.value.length:a*this.size;super.updateSubBuffer({startOffset:d,endOffset:e})}this._checkAttributeArray()}else h=!1;return this.clearNeedsUpdate(),this.setNeedsRedraw(),h}setConstantValue(a,b){if(void 0===b||"function"==typeof b)return!1;let c=this.settings.transform&&a?this.settings.transform.call(a,b):b;return"webgpu"===this.device.type?this.setConstantBufferValue(c,this.numInstances):(this.setData({constant:!0,value:c})&&this.setNeedsRedraw(),this.clearNeedsUpdate(),!0)}setConstantBufferValue(a,b){let c=this.settings.defaultType,d=this._normalizeValue(a,new c(this.size),0);if(this._hasConstantBufferValue(d,b))return this.constant=!1,this.clearNeedsUpdate(),!1;let e=new c(Math.max(b,1)*this.size);for(let a=0;a<e.length;a+=this.size)e.set(d,a);let f=this.setData({value:e});return this.constant=!1,this.clearNeedsUpdate(),f&&this.setNeedsRedraw(),f}_hasConstantBufferValue(a,b){let c=this.value,d=Math.max(b,1)*this.size;if(!ArrayBuffer.isView(c)||c.length!==d||c.length%this.size!=0)return!1;for(let b=0;b<c.length;b+=this.size)for(let d=0;d<this.size;d++)if(c[b+d]!==a[d])return!1;return!0}setExternalBuffer(a){let{state:b}=this;return a?(this.clearNeedsUpdate(),b.lastExternalBuffer===a||(b.lastExternalBuffer=a,this.setNeedsRedraw(),this.setData(a),!0)):(b.lastExternalBuffer=null,!1)}setBinaryValue(a,b=null){let{state:c,settings:d}=this;if(!a)return c.binaryValue=null,c.binaryAccessor=null,!1;if(d.noAlloc)return!1;if(c.binaryValue===a)return this.clearNeedsUpdate(),!0;if(c.binaryValue=a,this.setNeedsRedraw(),d.transform||b!==this.startIndices){ArrayBuffer.isView(a)&&(a={value:a});let e=a;gT(ArrayBuffer.isView(e.value),`invalid ${d.accessor}`);let f=!!e.size&&e.size!==this.size;return c.binaryAccessor=iK(e.value,{size:e.size||this.size,stride:e.stride,offset:e.offset,startIndices:b,nested:f}),!1}return this.clearNeedsUpdate(),this.setData(a),!0}getVertexOffset(a){let{startIndices:b}=this;return(b?a<b.length?b[a]:this.numInstances:a)*this.size}getValue(){let a=this.settings.shaderAttributes,b=super.getValue();if(!a)return b;for(let c in a)Object.assign(b,super.getValue(c,a[c]));return b}getBufferLayout(a){this.state.layoutChanged=!1;let b=this.settings.shaderAttributes,c=super._getBufferLayout(),{stepMode:d}=this.settings;if("dynamic"===d?c.stepMode=a?a.isInstanced?"instance":"vertex":"instance":c.stepMode=d??"vertex",!b)return c;for(let a in b){let d=super._getBufferLayout(a,b[a]);c.attributes.push(...d.attributes)}return c}_autoUpdater(a,{data:b,startRow:c,endRow:d,props:e,numInstances:f}){let{settings:g,state:h,value:i,size:j,startIndices:k}=a,{accessor:l,transform:m}=g,n=h.binaryAccessor||("function"==typeof l?l:e[l]);gT("function"==typeof n,`accessor "${l}" is not a function`);let o=a.getVertexOffset(c),{iterable:p,objectInfo:q}=iI(b,c,d);for(let b of p){q.index++;let c=n(b,q);if(m&&(c=m.call(this,c)),k){let b=(q.index<k.length-1?k[q.index+1]:f)-k[q.index];if(c&&Array.isArray(c[0])){let b=o;for(let d of c)a._normalizeValue(d,i,b),b+=j}else c&&c.length>j?i.set(c,o):(a._normalizeValue(c,q.target,0),function({target:a,source:b,start:c=0,count:d=1}){let e=b.length,f=d*e,g=0;for(let d=c;g<e;g++)a[d++]=b[g];for(;g<f;)g<f-g?(a.copyWithin(c+g,c,c+g),g*=2):(a.copyWithin(c+g,c,c+f-g),g=f)}({target:i,source:q.target,start:o,count:b}));o+=b*j}else a._normalizeValue(c,i,o),o+=j}}_validateAttributeUpdaters(){let{settings:a}=this;if(!(a.noAlloc||"function"==typeof a.update))throw Error(`Attribute ${this.id} missing update or accessor`)}_checkAttributeArray(){let{value:a}=this,b=Math.min(4,this.size);if(a&&a.length>=b){let c=!0;switch(b){case 4:c=c&&Number.isFinite(a[3]);case 3:c=c&&Number.isFinite(a[2]);case 2:c=c&&Number.isFinite(a[1]);case 1:c=c&&Number.isFinite(a[0]);break;default:c=!1}if(!c)throw Error(`Illegal attribute generated for ${this.id}`)}}}let iQ=`\
out vec4 transform_output;
void main() {
  transform_output = vec4(0);
}`,iR=`#version 300 es
${iQ}`;function iS(a){let{input:b,inputChannels:c,output:d}=a||{};if(!b)return iR;if(!c)throw Error("inputChannels");let e=function(a){switch(a){case 1:return"float";case 2:return"vec2";case 3:return"vec3";case 4:return"vec4";default:throw Error(`invalid channels: ${a}`)}}(c),f=function(a,b){switch(b){case 1:return`vec4(${a}, 0.0, 0.0, 1.0)`;case 2:return`vec4(${a}, 0.0, 1.0)`;case 3:return`vec4(${a}, 1.0)`;case 4:return a;default:throw Error(`invalid channels: ${b}`)}}(b,c);return`\
#version 300 es
in ${e} ${b};
out vec4 ${d};
void main() {
  ${d} = ${f};
}`}var iT=a.i(81555),iU=a.i(30245),iV=a.i(52486),iW=a.i(79552);class iX extends iW.Resource{get[Symbol.toStringTag](){return"ComputePipeline"}hash="";shaderLayout;constructor(a,b){super(a,b,iX.defaultProps),this.shaderLayout=b.shaderLayout}static defaultProps={...iW.Resource.defaultProps,shader:void 0,entryPoint:void 0,constants:{},shaderLayout:void 0}}var iY=a.i(93869);class iZ{static defaultProps={...iV.RenderPipeline.defaultProps};static getDefaultPipelineFactory(a){let b=a.getModuleData("@luma.gl/core");return b.defaultPipelineFactory||=new iZ(a),b.defaultPipelineFactory}device;_hashCounter=0;_hashes={};_renderPipelineCache={};_computePipelineCache={};_sharedRenderPipelineCache={};get[Symbol.toStringTag](){return"PipelineFactory"}toString(){return`PipelineFactory(${this.device.id})`}constructor(a){this.device=a}createRenderPipeline(a){if(!this.device.props._cachePipelines)return this.device.createRenderPipeline(a);let b={...iV.RenderPipeline.defaultProps,...a},c=this._renderPipelineCache,d=this._hashRenderPipeline(b),e=c[d]?.resource;if(e)c[d].useCount++,this.device.props.debugFactories&&bB.log.log(3,`${this}: ${c[d].resource} reused, count=${c[d].useCount}, (id=${a.id})`)();else{let a="webgl"===this.device.type&&this.device.props._sharePipelines?this.createSharedRenderPipeline(b):void 0;(e=this.device.createRenderPipeline({...b,id:b.id?`${b.id}-cached`:(0,iY.uid)("unnamed-cached"),_sharedRenderPipeline:a})).hash=d,c[d]={resource:e,useCount:1},this.device.props.debugFactories&&bB.log.log(3,`${this}: ${e} created, count=${c[d].useCount}`)()}return e}createComputePipeline(a){if(!this.device.props._cachePipelines)return this.device.createComputePipeline(a);let b={...iX.defaultProps,...a},c=this._computePipelineCache,d=this._hashComputePipeline(b),e=c[d]?.resource;return e?(c[d].useCount++,this.device.props.debugFactories&&bB.log.log(3,`${this}: ${c[d].resource} reused, count=${c[d].useCount}, (id=${a.id})`)()):((e=this.device.createComputePipeline({...b,id:b.id?`${b.id}-cached`:void 0})).hash=d,c[d]={resource:e,useCount:1},this.device.props.debugFactories&&bB.log.log(3,`${this}: ${e} created, count=${c[d].useCount}`)()),e}release(a){if(!this.device.props._cachePipelines)return void a.destroy();let b=this._getCache(a),c=a.hash;b[c].useCount--,0===b[c].useCount?(this._destroyPipeline(a),this.device.props.debugFactories&&bB.log.log(3,`${this}: ${a} released and destroyed`)()):b[c].useCount<0?(bB.log.error(`${this}: ${a} released, useCount < 0, resetting`)(),b[c].useCount=0):this.device.props.debugFactories&&bB.log.log(3,`${this}: ${a} released, count=${b[c].useCount}`)()}createSharedRenderPipeline(a){let b=this._hashSharedRenderPipeline(a),c=this._sharedRenderPipelineCache[b];return c||(c={resource:this.device._createSharedRenderPipelineWebGL(a),useCount:0},this._sharedRenderPipelineCache[b]=c),c.useCount++,c.resource}releaseSharedRenderPipeline(a){if(!a.sharedRenderPipeline)return;let b=this._hashSharedRenderPipeline(a.sharedRenderPipeline.props),c=this._sharedRenderPipelineCache[b];c&&(c.useCount--,0===c.useCount&&(c.resource.destroy(),delete this._sharedRenderPipelineCache[b]))}_destroyPipeline(a){let b=this._getCache(a);return!!this.device.props._destroyPipelines&&(delete b[a.hash],a.destroy(),a instanceof iV.RenderPipeline&&this.releaseSharedRenderPipeline(a),!0)}_getCache(a){let b;if(a instanceof iX&&(b=this._computePipelineCache),a instanceof iV.RenderPipeline&&(b=this._renderPipelineCache),!b)throw Error(`${this}`);if(!b[a.hash])throw Error(`${this}: ${a} matched incorrect entry`);return b}_hashComputePipeline(a){let{type:b}=this.device,c=this._getHash(a.shader.source),d=this._getHash(JSON.stringify(a.shaderLayout));return`${b}/C/${c}SL${d}`}_hashRenderPipeline(a){let b=a.vs?this._getHash(a.vs.source):0,c=a.fs?this._getHash(a.fs.source):0,d=this._getWebGLVaryingHash(a),e=this._getHash(JSON.stringify(a.shaderLayout)),f=this._getHash(JSON.stringify(a.bufferLayout)),{type:g}=this.device;if("webgl"===g){let h=this._getHash(JSON.stringify(a.parameters));return`${g}/R/${b}/${c}V${d}T${a.topology}P${h}SL${e}BL${f}`}{let h=this._getHash(JSON.stringify({vertexEntryPoint:a.vertexEntryPoint,fragmentEntryPoint:a.fragmentEntryPoint})),i=this._getHash(JSON.stringify(a.parameters)),j=this._getWebGPUAttachmentHash(a);return`${g}/R/${b}/${c}V${d}T${a.topology}EP${h}P${i}SL${e}BL${f}A${j}`}}_hashSharedRenderPipeline(a){let b=a.vs?this._getHash(a.vs.source):0,c=a.fs?this._getHash(a.fs.source):0,d=this._getWebGLVaryingHash(a);return`webgl/S/${b}/${c}V${d}`}_getHash(a){return void 0===this._hashes[a]&&(this._hashes[a]=this._hashCounter++),this._hashes[a]}_getWebGLVaryingHash(a){let{varyings:b=[],bufferMode:c=null}=a;return this._getHash(JSON.stringify({varyings:b,bufferMode:c}))}_getWebGPUAttachmentHash(a){let b=a.colorAttachmentFormats??[this.device.preferredColorFormat],c=a.parameters?.depthWriteEnabled?a.depthStencilAttachmentFormat||this.device.preferredDepthFormat:null;return this._getHash(JSON.stringify({colorAttachmentFormats:b,depthStencilAttachmentFormat:c}))}}var i$=a.i(57652);class i_{static defaultProps={...i$.Shader.defaultProps};static getDefaultShaderFactory(a){let b=a.getModuleData("@luma.gl/core");return b.defaultShaderFactory||=new i_(a),b.defaultShaderFactory}device;_cache={};get[Symbol.toStringTag](){return"ShaderFactory"}toString(){return`${this[Symbol.toStringTag]}(${this.device.id})`}constructor(a){this.device=a}createShader(a){if(!this.device.props._cacheShaders)return this.device.createShader(a);let b=this._hashShader(a),c=this._cache[b];if(c)c.useCount++,this.device.props.debugFactories&&bB.log.log(3,`${this}: Reusing shader ${c.resource.id} count=${c.useCount}`)();else{let d=this.device.createShader({...a,id:a.id?`${a.id}-cached`:void 0});this._cache[b]=c={resource:d,useCount:1},this.device.props.debugFactories&&bB.log.log(3,`${this}: Created new shader ${d.id}`)()}return c.resource}release(a){if(!this.device.props._cacheShaders)return void a.destroy();let b=this._hashShader(a),c=this._cache[b];if(c)if(c.useCount--,0===c.useCount)this.device.props._destroyShaders&&(delete this._cache[b],c.resource.destroy(),this.device.props.debugFactories&&bB.log.log(3,`${this}: Releasing shader ${a.id}, destroyed`)());else if(c.useCount<0)throw Error(`ShaderFactory: Shader ${a.id} released too many times`);else this.device.props.debugFactories&&bB.log.log(3,`${this}: Releasing shader ${a.id} count=${c.useCount}`)()}_hashShader(a){return`${a.stage}:${a.source}`}}var i0=a.i(35763);function i1(a,b){let c=(0,i0.resolveVariableShaderTypeAlias)(a),d=(0,i0.getVariableShaderTypeInfo)(c),e=/^mat(\d)x(\d)<.+>$/.exec(c);if(e){var f,g;let a=Number(e[1]),h=Number(e[2]),i=i4(h,c,d.type,b),j=(f=i.size,g=i.alignment,"std140"===b?4:(0,iA.alignTo)(f,g));return{alignment:i.alignment,size:a*j,components:a*h,columns:a,rows:h,columnStride:j,shaderType:c,type:d.type}}let h=/^vec(\d)<.+>$/.exec(c);return h?i4(Number(h[1]),c,d.type,b):{alignment:1,size:1,components:1,columns:1,rows:1,columnStride:1,shaderType:c,type:d.type}}function i2(a){return!!a&&"object"==typeof a&&!Array.isArray(a)}function i3(a,b){var c;if("string"==typeof a)return i1(a,b).alignment;if(Array.isArray(a)){let c=i3(a[0],b);return i5(b)?Math.max(c,4):c}let d=1;for(let c of Object.values(a))d=Math.max(d,i3(c,b));return"std140"===(c=b)||"wgsl-uniform"===c?Math.max(d,4):d}function i4(a,b,c,d){return{alignment:2===a?2:4,size:3===a?3:a,components:a,columns:1,rows:a,columnStride:3===a?3:a,shaderType:b,type:c}}function i5(a){return"std140"===a||"wgsl-uniform"===a}function i6(a){return Array.isArray(a)?0===a.length||"number"==typeof a[0]:ArrayBuffer.isView(a)&&!(a instanceof DataView)}class i7{name;uniforms={};modifiedUniforms={};modified=!0;bindingLayout={};needsRedraw="initialized";constructor(a){if(this.name=a?.name||"unnamed",a?.name&&a?.shaderLayout){const b=a?.shaderLayout.bindings?.find(b=>"uniform"===b.type&&b.name===a?.name);if(!b)throw Error(a?.name);for(const a of b.uniforms||[])this.bindingLayout[a.name]=a}}setUniforms(a){for(let[b,c]of Object.entries(a))this._setUniform(b,c),this.needsRedraw||this.setNeedsRedraw(`${this.name}.${b}=${c}`)}setNeedsRedraw(a){this.needsRedraw=this.needsRedraw||a}getAllUniforms(){return this.modifiedUniforms={},this.needsRedraw=!1,this.uniforms||{}}_setUniform(a,b){!function(a,b,c=16){if(a===b)return!0;if(!i6(a)||!i6(b)||a.length!==b.length)return!1;let d=Math.min(c,128);if(a.length>d)return!1;for(let c=0;c<a.length;++c)if(b[c]!==a[c])return!1;return!0}(this.uniforms[a],b)&&(this.uniforms[a]=i6(b)?b.slice():b,this.modifiedUniforms[a]=!0,this.modified=!0)}}var i8=a.i(41656);class i9{layout;constructor(a){this.layout=a}has(a){return!!this.layout.fields[a]}get(a){let b=this.layout.fields[a];return b?{offset:b.offset,size:b.size}:void 0}getFlatUniformValues(a){let b={};for(let[c,d]of Object.entries(a)){let a=this.layout.uniformTypes[c];a?this._flattenCompositeValue(b,c,a,d):this.layout.fields[c]&&(b[c]=d)}return b}getData(a){let b=(0,i8.getScratchArrayBuffer)(this.layout.byteLength);new Uint8Array(b,0,this.layout.byteLength).fill(0);let c={i32:new Int32Array(b),u32:new Uint32Array(b),f32:new Float32Array(b),f16:new Uint16Array(b)};for(let[b,d]of Object.entries(this.getFlatUniformValues(a)))this._writeLeafValue(c,b,d);return new Uint8Array(b,0,this.layout.byteLength)}_flattenCompositeValue(a,b,c,d){if(void 0!==d){var e;if("string"==typeof c||this.layout.fields[b]){a[b]=d;return}if(Array.isArray(c)){let e=c[0],f=c[1];if(Array.isArray(e))throw Error(`Nested arrays are not supported for ${b}`);if("string"==typeof e&&i6(d))return void this._flattenPackedArray(a,b,e,f,d);if(!Array.isArray(d))return void bB.log.warn(`Unsupported uniform array value for ${b}:`,d)();for(let c=0;c<Math.min(d.length,f);c++){let f=d[c];void 0!==f&&this._flattenCompositeValue(a,`${b}[${c}]`,e,f)}return}if(i2(c)&&(e=d)&&"object"==typeof e&&!Array.isArray(e)&&!ArrayBuffer.isView(e)){for(let[e,f]of Object.entries(d)){if(void 0===f)continue;let d=`${b}.${e}`;this._flattenCompositeValue(a,d,c[e],f)}return}bB.log.warn(`Unsupported uniform value for ${b}:`,d)()}}_flattenPackedArray(a,b,c,d,e){let f=i1(c,this.layout.layout).components;for(let c=0;c<d;c++){var g,h,i;let d=c*f;if(d>=e.length)break;1===f?a[`${b}[${c}]`]=Number(e[d]):a[`${b}[${c}]`]=(g=e,h=d,i=d+f,Array.prototype.slice.call(g,h,i))}}_writeLeafValue(a,b,c){let d=this.layout.fields[b];if(!d)return void bB.log.warn(`Uniform ${b} not found in layout`)();let{type:e,components:f,columns:g,rows:h,offset:i,columnStride:j}=d,k=a[e];if(1===f){k[i]=Number(c);return}if(1===g){for(let a=0;a<f;a++)k[i+a]=Number(c[a]??0);return}let l=0;for(let a=0;a<g;a++){let b=i+a*j;for(let a=0;a<h;a++)k[b+a]=Number(c[l++]??0)}}}class ja{device;uniformBlocks=new Map;shaderBlockLayouts=new Map;shaderBlockWriters=new Map;uniformBuffers=new Map;constructor(a,b){for(const[c,d]of(this.device=a,Object.entries(b))){const b=function(a,b={}){let c={...a},d=b.layout??"std140",e={},f=0;for(let[a,b]of Object.entries(c))f=function a(b,c,d,e,f){if("string"==typeof d){let a=i1(d,f),g=(0,iA.alignTo)(e,a.alignment);return b[c]={offset:g,...a},g+a.size}if(Array.isArray(d)){if(Array.isArray(d[0]))throw Error(`Nested arrays are not supported for ${c}`);let g=d[0],h=d[1],i=function a(b,c){var d,e,f;return d=function b(c,d){if("string"==typeof c)return i1(c,d).size;if(Array.isArray(c)){let b=c[0],e=c[1];if(Array.isArray(b))throw Error("Nested arrays are not supported");return a(b,d)*e}let e=0;for(let a of Object.values(c))e=(0,iA.alignTo)(e,i3(a,d))+b(a,d);return(0,iA.alignTo)(e,i3(c,d))}(b,c),e=i3(b,c),f=c,(0,iA.alignTo)(d,i5(f)?4:e)}(g,f),j=(0,iA.alignTo)(e,i3(d,f));for(let d=0;d<h;d++)a(b,`${c}[${d}]`,g,j+d*i,f);return j+i*h}if(i2(d)){let g=i3(d,f),h=(0,iA.alignTo)(e,g);for(let[e,g]of Object.entries(d))h=a(b,`${c}.${e}`,g,h,f);return(0,iA.alignTo)(h,g)}throw Error(`Unsupported CompositeShaderType for ${c}`)}(e,a,b,f,d);return f=(0,iA.alignTo)(f,i3(c,d)),{layout:d,byteLength:4*f,uniformTypes:c,fields:e}}(d.uniformTypes??{},{layout:d.layout??function(a){return"webgpu"===a.type?"wgsl-uniform":"std140"}(a)}),e=new i9(b);this.shaderBlockLayouts.set(c,b),this.shaderBlockWriters.set(c,e);const f=new i7({name:c});f.setUniforms(e.getFlatUniformValues(d.defaultUniforms||{})),this.uniformBlocks.set(c,f)}}destroy(){for(let a of this.uniformBuffers.values())a.destroy()}setUniforms(a){for(let[b,c]of Object.entries(a)){let a=this.shaderBlockWriters.get(b),d=a?.getFlatUniformValues(c||{});this.uniformBlocks.get(b)?.setUniforms(d||{})}this.updateUniformBuffers()}getUniformBufferByteLength(a){return Math.max(this.shaderBlockLayouts.get(a)?.byteLength||0,1024)}getUniformBufferData(a){let b=this.uniformBlocks.get(a)?.getAllUniforms()||{},c=this.shaderBlockWriters.get(a);return c?.getData(b)||new Uint8Array(0)}createUniformBuffer(a,b){b&&this.setUniforms(b);let c=this.getUniformBufferByteLength(a),d=this.device.createBuffer({usage:hF.Buffer.UNIFORM|hF.Buffer.COPY_DST,byteLength:c}),e=this.getUniformBufferData(a);return d.write(e),d}getManagedUniformBuffer(a){if(!this.uniformBuffers.get(a)){let b=this.getUniformBufferByteLength(a),c=this.device.createBuffer({usage:hF.Buffer.UNIFORM|hF.Buffer.COPY_DST,byteLength:b});this.uniformBuffers.set(a,c)}return this.uniformBuffers.get(a)}updateUniformBuffers(){let a=!1;for(let b of this.uniformBlocks.keys()){let c=this.updateUniformBuffer(b);a||=c}return a&&bB.log.log(3,`UniformStore.updateUniformBuffers(): ${a}`)(),a}updateUniformBuffer(a){let b=this.uniformBlocks.get(a),c=this.uniformBuffers.get(a),d=!1;if(c&&b?.needsRedraw){d||=b.needsRedraw;let e=this.getUniformBufferData(a);c=this.uniformBuffers.get(a),c?.write(e);let f=this.uniformBlocks.get(a)?.getAllUniforms();bB.log.log(4,`Writing to uniform buffer ${String(a)}`,e,f)()}return d}}var jb=a.i(73007),jc=a.i(84978),jd=a.i(93245);let je={};function jf(a="id"){je[a]=je[a]||1;let b=je[a]++;return`${a}-${b}`}class jg{id;userData={};topology;bufferLayout=[];vertexCount;indices;attributes;constructor(a){if(this.id=a.id||jf("geometry"),this.topology=a.topology,this.indices=a.indices||null,this.attributes=a.attributes,this.vertexCount=a.vertexCount,this.bufferLayout=a.bufferLayout||[],this.indices&&!(this.indices.usage&hF.Buffer.INDEX))throw Error("Index buffer must have INDEX usage")}destroy(){for(let a of(this.indices?.destroy(),Object.values(this.attributes)))a.destroy()}getVertexCount(){return this.vertexCount}getAttributes(){return this.attributes}getIndexes(){return this.indices||null}_calculateVertexCount(a){return a.byteLength/12}}let jh="__debugFramebufferState";function ji(a,b){if(!a)return b;let c=Number.parseInt(a,10);return Number.isFinite(c)?c:b}class jj{bufferLayouts;constructor(a){this.bufferLayouts=a}getBufferLayout(a){return this.bufferLayouts.find(b=>b.name===a)||null}getAttributeNamesForBuffer(a){return a.attributes?a.attributes?.map(a=>a.attribute):[a.name]}mergeBufferLayouts(a,b){let c=[...a];for(let a of b){let b=c.findIndex(b=>b.name===a.name);b<0?c.push(a):c[b]=a}return c}getBufferIndex(a){let b=this.bufferLayouts.findIndex(b=>b.name===a);return -1===b&&bB.log.warn(`BufferLayout: Missing buffer for "${a}".`)(),b}}function jk(a,b){let c=1/0;for(let d of a){let a=b[d];void 0!==a&&(c=Math.min(c,a))}return c}function jl(a,b){if(!a||!b.some(a=>a.bindingLayout?.length))return a;let c={...a,bindings:a.bindings.map(a=>({...a}))};for(let d of("attributes"in(a||{})&&(c.attributes=a?.attributes||[]),b))for(let a of d.bindingLayout||[])for(let b of function(a){let b=new Set([a,`${a}Uniforms`]);return a.endsWith("Uniforms")||b.add(`${a}Sampler`),[...b]}(a.name)){let d=c.bindings.find(a=>a.name===b);d?.group===0&&(d.group=a.group)}return c}class jm{options={disableWarnings:!1};modules;moduleUniforms;moduleBindings;constructor(a,b){for(const c of(Object.assign(this.options,b),aS(Object.values(a).filter(jr))))a[c.name]=c;for(const[b,c]of(bB.log.log(1,"Creating ShaderInputs with modules",Object.keys(a))(),this.modules=a,this.moduleUniforms={},this.moduleBindings={},Object.entries(a)))c&&(this._addModule(c),c.name&&b!==c.name&&!this.options.disableWarnings&&bB.log.warn(`Module name: ${b} vs ${c.name}`)())}destroy(){}setProps(a){for(let b of Object.keys(a)){let c=a[b]||{},d=this.modules[b];if(d){let a=this.moduleUniforms[b],e=this.moduleBindings[b],{uniforms:f,bindings:g}=function(a,b={}){let c={bindings:{},uniforms:{}};return Object.keys(a).forEach(d=>{var e,f,g;let h=a[d];Object.prototype.hasOwnProperty.call(b,d)||ArrayBuffer.isView(g=f=e=h)&&!(g instanceof DataView)||Array.isArray(f)&&(0===f.length||"number"==typeof f[0])||"number"==typeof e||"boolean"==typeof e?c.uniforms[d]=h:c.bindings[d]=h}),c}(d.getUniforms?.(c,a)||c,d.uniformTypes);this.moduleUniforms[b]=jn(a,f,d.uniformTypes),this.moduleBindings[b]={...e,...g}}else this.options.disableWarnings||bB.log.warn(`Module ${b} not found`)()}}getModules(){return Object.values(this.modules)}getUniformValues(){return this.moduleUniforms}getBindingValues(){let a={};for(let b of Object.values(this.moduleBindings))Object.assign(a,b);return a}getDebugTable(){let a={};for(let[b,c]of Object.entries(this.moduleUniforms))for(let[d,e]of Object.entries(c))a[`${b}.${d}`]={type:this.modules[b].uniformTypes?.[d],value:String(e)};return a}_addModule(a){let b=a.name;this.moduleUniforms[b]=jn({},a.defaultUniforms||{},a.uniformTypes),this.moduleBindings[b]={}}}function jn(a={},b={},c={}){let d={...a};for(let[e,f]of Object.entries(b))void 0!==f&&(d[e]=function a(b,c,d){if(!d||"string"==typeof d)return jo(c);if(Array.isArray(d)){if(jp(c)||!Array.isArray(c))return jo(c);let e=Array.isArray(b)&&!jp(b)?[...b]:[],f=e.slice();for(let b=0;b<c.length;b++){let g=c[b];void 0!==g&&(f[b]=a(e[b],g,d[0]))}return f}if(!jq(c))return jo(c);let e=jq(b)?b:{},f={...e};for(let[b,g]of Object.entries(c))void 0!==g&&(f[b]=a(e[b],g,d[b]));return f}(a[e],f,c[e]));return d}function jo(a){return ArrayBuffer.isView(a)?Array.prototype.slice.call(a):Array.isArray(a)?jp(a)?a.slice():a.map(a=>void 0===a?void 0:jo(a)):jq(a)?Object.fromEntries(Object.entries(a).map(([a,b])=>[a,void 0===b?void 0:jo(b)])):a}function jp(a){return ArrayBuffer.isView(a)||Array.isArray(a)&&(0===a.length||"number"==typeof a[0])}function jq(a){return!!a&&"object"==typeof a&&!Array.isArray(a)&&!ArrayBuffer.isView(a)}function jr(a){return!!a?.dependencies}var js=a.i(61669);let jt={"+X":0,"-X":1,"+Y":2,"-Y":3,"+Z":4,"-Z":5};function ju(a){return a?Array.isArray(a)?a[0]??null:a:null}function jv(a){if((0,js.isExternalImage)(a))return(0,js.getExternalImageSize)(a);if("object"==typeof a&&"width"in a&&"height"in a)return{width:a.width,height:a.height};throw Error("Unsupported mip-level data")}function jw(a){let{textureFormat:b,format:c}=a;if(b&&c&&b!==c)throw Error(`Conflicting texture formats "${b}" and "${c}" provided for the same mip level`);return b??c}function jx(a){let b=jt[a];if(void 0===b)throw Error(`Invalid cube face: ${a}`);return b}function jy(a){throw Error("setTexture1DData not supported in WebGL.")}function jz(a,b,c,d){let e=Array.isArray(b)?b:[b],f=[];for(let b=0;b<e.length;b++){let g=e[b];if((0,js.isExternalImage)(g))f.push({type:"external-image",image:g,z:a,mipLevel:b});else if("object"==typeof g&&null!==g&&"data"in g&&"width"in g&&"height"in g)f.push({type:"texture-data",data:g,textureFormat:jw(g),z:a,mipLevel:b});else if(ArrayBuffer.isView(g)&&c)f.push({type:"texture-data",data:{data:g,width:Math.max(1,c.width>>b),height:Math.max(1,c.height>>b),...d?{format:d}:{}},textureFormat:d,z:a,mipLevel:b});else throw Error("Unsupported 2D mip-level payload")}return f}function jA(a){let b=[];for(let c=0;c<a.length;c++)b.push(...jz(c,a[c]));return b}function jB(a){let b=[];for(let c=0;c<a.length;c++)b.push(...jz(c,a[c]));return b}function jC(a){let b=[];for(let[c,d]of Object.entries(a)){let a=jx(c);b.push(...jz(a,d))}return b}function jD(a){let b=[];return a.forEach((a,c)=>{for(let[d,e]of Object.entries(a)){let a=6*c+jx(d);b.push(...jz(a,e))}}),b}class jE{device;id;props;_texture=null;_sampler=null;_view=null;ready;isReady=!1;destroyed=!1;resolveReady=()=>{};rejectReady=()=>{};get texture(){if(!this._texture)throw Error("Texture not initialized yet");return this._texture}get sampler(){if(!this._sampler)throw Error("Sampler not initialized yet");return this._sampler}get view(){if(!this._view)throw Error("View not initialized yet");return this._view}get[Symbol.toStringTag](){return"DynamicTexture"}toString(){let a=this._texture?.width??this.props.width??"?",b=this._texture?.height??this.props.height??"?";return`DynamicTexture:"${this.id}":${a}x${b}px:(${this.isReady?"ready":"loading..."})`}constructor(a,b){this.device=a;const c=jf("dynamic-texture");this.props={...jE.defaultProps,id:c,...b,data:null},this.id=this.props.id,this.ready=new Promise((a,b)=>{this.resolveReady=a,this.rejectReady=b}),this.initAsync(b)}async initAsync(a){try{let b=await this._loadAllData(a);this._checkNotDestroyed();let c=b.data?function(a){if(!a.data)return[];let b=a.width&&a.height?{width:a.width,height:a.height}:void 0,c="format"in a?a.format:void 0;switch(a.dimension){case"1d":return jy(a.data);case"2d":return jz(0,a.data,b,c);case"3d":return jA(a.data);case"2d-array":return jB(a.data);case"cube":return jC(a.data);case"cube-array":return jD(a.data);default:throw Error(`Unhandled dimension ${a.dimension}`)}}({...b,width:a.width,height:a.height,format:a.format}):[],d="format"in a&&void 0!==a.format,e="usage"in a&&void 0!==a.usage,f=(()=>{if(this.props.width&&this.props.height)return{width:this.props.width,height:this.props.height};let a=function(a){let{dimension:b,data:c}=a;if(!c)return null;switch(b){case"1d":{let a=ju(c);if(!a)return null;let{width:b}=jv(a);return{width:b,height:1}}case"2d":{let a=ju(c);return a?jv(a):null}case"3d":case"2d-array":{if(!Array.isArray(c)||0===c.length)return null;let a=ju(c[0]);return a?jv(a):null}case"cube":{let a=Object.keys(c)[0]??null;if(!a)return null;let b=ju(c[a]);return b?jv(b):null}case"cube-array":{if(!Array.isArray(c)||0===c.length)return null;let a=c[0],b=Object.keys(a)[0]??null;if(!b)return null;let d=ju(a[b]);return d?jv(d):null}default:return null}}(b);return a||{width:this.props.width||1,height:this.props.height||1}})();if(!f||f.width<=0||f.height<=0)throw Error(`${this} size could not be determined or was zero`);let g=function(a,b,c,d){if(0===b.length)return{subresources:b,mipLevels:1,format:d.format,hasExplicitMipChain:!1};let e=new Map;for(let a of b){let b=e.get(a.z)??[];b.push(a),e.set(a.z,b)}let f=b.some(a=>a.mipLevel>0),g=d.format,h=1/0,i=[];for(let[b,d]of e){let e=[...d].sort((a,b)=>a.mipLevel-b.mipLevel),f=e[0];if(!f||0!==f.mipLevel)throw Error(`DynamicTexture: slice ${b} is missing mip level 0`);let j=jG(a,f);if(j.width!==c.width||j.height!==c.height)throw Error(`DynamicTexture: slice ${b} base level dimensions ${j.width}x${j.height} do not match expected ${c.width}x${c.height}`);let k=jF(f);if(k){if(g&&g!==k)throw Error(`DynamicTexture: slice ${b} base level format "${k}" does not match texture format "${g}"`);g=k}let l=g&&a.isTextureFormatCompressed(g)?function(a,b,c,d){let{blockWidth:e=1,blockHeight:f=1}=a.getTextureFormatInfo(d),g=1;for(let a=1;;a++){let d=Math.max(1,b>>a),h=Math.max(1,c>>a);if(d<e||h<f)break;g++}return g}(a,j.width,j.height,g):a.getMipLevelCount(j.width,j.height),m=0;for(let b=0;b<e.length;b++){let c=e[b];if(!c||c.mipLevel!==b||b>=l)break;let d=jG(a,c),f=Math.max(1,j.width>>b),h=Math.max(1,j.height>>b);if(d.width!==f||d.height!==h)break;let k=jF(c);if(k&&(g||(g=k),k!==g))break;m++,i.push(c)}h=Math.min(h,m)}let j=Number.isFinite(h)?Math.max(1,h):1;return{subresources:i.filter(a=>a.mipLevel<j),mipLevels:j,format:g,hasExplicitMipChain:f}}(this.device,c,f,{format:d?a.format:void 0}),h=g.format??this.props.format,i={...this.props,...f,format:h,mipLevels:1,data:void 0};this.device.isTextureFormatCompressed(h)&&!e&&(i.usage=hG.Texture.SAMPLE|hG.Texture.COPY_DST);let j=this.props.mipmaps&&!g.hasExplicitMipChain&&!this.device.isTextureFormatCompressed(h);if("webgpu"===this.device.type&&j){let a="3d"===this.props.dimension?hG.Texture.SAMPLE|hG.Texture.STORAGE|hG.Texture.COPY_DST|hG.Texture.COPY_SRC:hG.Texture.SAMPLE|hG.Texture.RENDER|hG.Texture.COPY_DST|hG.Texture.COPY_SRC;i.usage|=a}let k=this.device.getMipLevelCount(i.width,i.height),l=g.hasExplicitMipChain?g.mipLevels:"auto"===this.props.mipLevels?k:Math.max(1,Math.min(k,this.props.mipLevels??1)),m={...i,mipLevels:l};this._texture=this.device.createTexture(m),this._sampler=this.texture.sampler,this._view=this.texture.view,g.subresources.length&&this._setTextureSubresources(g.subresources),!this.props.mipmaps||g.hasExplicitMipChain||j||bB.log.warn(`${this} skipping auto-generated mipmaps for compressed texture format`)(),j&&this.generateMipmaps(),this.isReady=!0,this.resolveReady(this.texture),bB.log.info(0,`${this} created`)()}catch(b){let a=b instanceof Error?b:Error(String(b));this.rejectReady(a)}}destroy(){this._texture&&(this._texture.destroy(),this._texture=null,this._sampler=null,this._view=null),this.destroyed=!0}generateMipmaps(){"webgl"===this.device.type?this.texture.generateMipmapsWebGL():"webgpu"===this.device.type?this.device.generateMipmapsWebGPU(this.texture):bB.log.warn(`${this} mipmaps not supported on ${this.device.type}`)}setSampler(a={}){this._checkReady();let b=a instanceof iU.Sampler?a:this.device.createSampler(a);this.texture.setSampler(b),this._sampler=b}async readBuffer(a={}){this.isReady||await this.ready;let b=a.width??this.texture.width,c=a.height??this.texture.height,d=a.depthOrArrayLayers??this.texture.depth,e=this.texture.computeMemoryLayout({width:b,height:c,depthOrArrayLayers:d}),f=this.device.createBuffer({byteLength:e.byteLength,usage:hF.Buffer.COPY_DST|hF.Buffer.MAP_READ});this.texture.readBuffer({...a,width:b,height:c,depthOrArrayLayers:d},f);let g=this.device.createFence();return await g.signaled,g.destroy(),f}async readAsync(a={}){this.isReady||await this.ready;let b=a.width??this.texture.width,c=a.height??this.texture.height,d=a.depthOrArrayLayers??this.texture.depth,e=this.texture.computeMemoryLayout({width:b,height:c,depthOrArrayLayers:d}),f=await this.readBuffer(a),g=await f.readAsync(0,e.byteLength);return f.destroy(),g.buffer}resize(a){if(this._checkReady(),a.width===this.texture.width&&a.height===this.texture.height)return!1;let b=this.texture;return this._texture=b.clone(a),this._sampler=this.texture.sampler,this._view=this.texture.view,b.destroy(),bB.log.info(`${this} resized`),!0}getCubeFaceIndex(a){let b=jt[a];if(void 0===b)throw Error(`Invalid cube face: ${a}`);return b}getCubeArrayFaceIndex(a,b){return 6*a+this.getCubeFaceIndex(b)}setTexture1DData(a){if(this._checkReady(),"1d"!==this.texture.props.dimension)throw Error(`${this} is not 1d`);let b=jy(a);this._setTextureSubresources(b)}setTexture2DData(a,b=0){if(this._checkReady(),"2d"!==this.texture.props.dimension)throw Error(`${this} is not 2d`);let c=jz(b,a);this._setTextureSubresources(c)}setTexture3DData(a){if("3d"!==this.texture.props.dimension)throw Error(`${this} is not 3d`);let b=jA(a);this._setTextureSubresources(b)}setTextureArrayData(a){if("2d-array"!==this.texture.props.dimension)throw Error(`${this} is not 2d-array`);let b=jB(a);this._setTextureSubresources(b)}setTextureCubeData(a){if("cube"!==this.texture.props.dimension)throw Error(`${this} is not cube`);let b=jC(a);this._setTextureSubresources(b)}setTextureCubeArrayData(a){if("cube-array"!==this.texture.props.dimension)throw Error(`${this} is not cube-array`);let b=jD(a);this._setTextureSubresources(b)}_setTextureSubresources(a){for(let b of a){let{z:a,mipLevel:c}=b;switch(b.type){case"external-image":let{image:d,flipY:e}=b;this.texture.copyExternalImage({image:d,z:a,mipLevel:c,flipY:e});break;case"texture-data":let{data:f,textureFormat:g}=b;if(g&&g!==this.texture.format)throw Error(`${this} mip level ${c} uses format "${g}" but texture format is "${this.texture.format}"`);this.texture.writeData(f.data,{x:0,y:0,z:a,width:f.width,height:f.height,depthOrArrayLayers:1,mipLevel:c});break;default:throw Error("Unsupported 2D mip-level payload")}}}async _loadAllData(a){let b=await jH(a.data);return{dimension:a.dimension??"2d",data:b??null}}_checkNotDestroyed(){this.destroyed&&bB.log.warn(`${this} already destroyed`)}_checkReady(){this.isReady||bB.log.warn(`${this} Cannot perform this operation before ready`)}static defaultProps={...hG.Texture.defaultProps,dimension:"2d",data:null,mipmaps:!1}}function jF(a){if("texture-data"===a.type)return a.textureFormat??jw(a.data)}function jG(a,b){switch(b.type){case"external-image":return a.getExternalImageSize(b.image);case"texture-data":return{width:b.data.width,height:b.data.height};default:throw Error("Unsupported texture subresource")}}async function jH(a){if(Array.isArray(a=await a))return await Promise.all(a.map(jH));if(a&&"object"==typeof a&&a.constructor===Object){let b=a,c=await Promise.all(Object.values(b).map(jH)),d=Object.keys(b),e={};for(let a=0;a<d.length;a++)e[d[a]]=c[a];return e}return a}let jI="render pipeline initialization failed";class jJ{static defaultProps={...iV.RenderPipeline.defaultProps,source:void 0,vs:null,fs:null,id:"unnamed",handle:void 0,userData:{},defines:{},modules:[],geometry:null,indexBuffer:null,attributes:{},constantAttributes:{},bindings:{},uniforms:{},varyings:[],isInstanced:void 0,instanceCount:0,vertexCount:0,shaderInputs:void 0,material:void 0,pipelineFactory:void 0,shaderFactory:void 0,transformFeedback:void 0,shaderAssembler:by.getDefaultShaderAssembler(),debugShaders:void 0,disableWarnings:void 0};device;id;source;vs;fs;pipelineFactory;shaderFactory;userData={};parameters;topology;bufferLayout;isInstanced=void 0;instanceCount=0;vertexCount;indexBuffer=null;bufferAttributes={};constantAttributes={};bindings={};vertexArray;transformFeedback=null;pipeline;shaderInputs;material=null;_uniformStore;_attributeInfos={};_gpuGeometry=null;props;_pipelineNeedsUpdate="newly created";_needsRedraw="initializing";_destroyed=!1;_lastDrawTimestamp=-1;_bindingTable=[];get[Symbol.toStringTag](){return"Model"}toString(){return`Model(${this.id})`}constructor(a,b){this.props={...jJ.defaultProps,...b},b=this.props,this.id=b.id||jf("model"),this.device=a,Object.assign(this.userData,b.userData),this.material=b.material||null;const c=Object.fromEntries(this.props.modules?.map(a=>[a.name,a])||[]),d=b.shaderInputs||new jm(c,{disableWarnings:this.props.disableWarnings});this.setShaderInputs(d);const e=function(a){return{type:a.type,shaderLanguage:a.info.shadingLanguage,shaderLanguageVersion:a.info.shadingLanguageVersion,gpu:a.info.gpu,features:a.features}}(a),f=(this.props.modules?.length>0?this.props.modules:this.shaderInputs?.getModules())||[];if(this.props.shaderLayout=jl(this.props.shaderLayout,f)||null,"webgpu"===this.device.type&&this.props.source){const{source:b,getUniforms:c,bindingTable:d}=this.props.shaderAssembler.assembleWGSLShader({platformInfo:e,...this.props,modules:f});this.source=b,this._getModuleUniforms=c,this._bindingTable=d;const g=a.getShaderLayout?.(this.source);this.props.shaderLayout=jl(this.props.shaderLayout||g||null,f)||null}else{const{vs:a,fs:b,getUniforms:c}=this.props.shaderAssembler.assembleGLSLShaderPair({platformInfo:e,...this.props,modules:f});this.vs=a,this.fs=b,this._getModuleUniforms=c,this._bindingTable=[]}this.vertexCount=this.props.vertexCount,this.instanceCount=this.props.instanceCount,this.topology=this.props.topology,this.bufferLayout=this.props.bufferLayout,this.parameters=this.props.parameters,b.geometry&&this.setGeometry(b.geometry),this.pipelineFactory=b.pipelineFactory||iZ.getDefaultPipelineFactory(this.device),this.shaderFactory=b.shaderFactory||i_.getDefaultShaderFactory(this.device),this.pipeline=this._updatePipeline(),this.vertexArray=a.createVertexArray({shaderLayout:this.pipeline.shaderLayout,bufferLayout:this.pipeline.bufferLayout}),this._gpuGeometry&&this._setGeometryAttributes(this._gpuGeometry),"isInstanced"in b&&(this.isInstanced=b.isInstanced),b.instanceCount&&this.setInstanceCount(b.instanceCount),b.vertexCount&&this.setVertexCount(b.vertexCount),b.indexBuffer&&this.setIndexBuffer(b.indexBuffer),b.attributes&&this.setAttributes(b.attributes),b.constantAttributes&&this.setConstantAttributes(b.constantAttributes),b.bindings&&this.setBindings(b.bindings),b.transformFeedback&&(this.transformFeedback=b.transformFeedback)}destroy(){this._destroyed||(this.pipelineFactory.release(this.pipeline),this.shaderFactory.release(this.pipeline.vs),this.pipeline.fs&&this.pipeline.fs!==this.pipeline.vs&&this.shaderFactory.release(this.pipeline.fs),this._uniformStore.destroy(),this._gpuGeometry?.destroy(),this._destroyed=!0)}needsRedraw(){this._getBindingsUpdateTimestamp()>this._lastDrawTimestamp&&this.setNeedsRedraw("contents of bound textures or buffers updated");let a=this._needsRedraw;return this._needsRedraw=!1,a}setNeedsRedraw(a){this._needsRedraw||=a}getBindingDebugTable(){return this._bindingTable}predraw(){this.updateShaderInputs(),this.pipeline=this._updatePipeline()}draw(a){let b,c=this._areBindingsLoading();if(c)return bB.log.info(2,`>>> DRAWING ABORTED ${this.id}: ${c} not loaded`)(),!1;try{a.pushDebugGroup(`${this}.predraw(${a})`),this.predraw()}finally{a.popDebugGroup()}let d=this.pipeline.isErrored;try{if(a.pushDebugGroup(`${this}.draw(${a})`),this._logDrawCallStart(),this.pipeline=this._updatePipeline(),d=this.pipeline.isErrored)bB.log.info(2,`>>> DRAWING ABORTED ${this.id}: ${jI}`)(),b=!1;else{let c=this._getBindings(),d=this._getBindGroups(),{indexBuffer:e}=this.vertexArray,f=e?e.byteLength/("uint32"===e.indexType?4:2):void 0;b=this.pipeline.draw({renderPass:a,vertexArray:this.vertexArray,isInstanced:this.isInstanced,vertexCount:this.vertexCount,instanceCount:this.instanceCount,indexCount:f,transformFeedback:this.transformFeedback||void 0,bindings:c,bindGroups:d,_bindGroupCacheKeys:this._getBindGroupCacheKeys(),uniforms:this.props.uniforms,parameters:this.parameters,topology:this.topology})}}finally{a.popDebugGroup(),this._logDrawCallEnd()}return this._logFramebuffer(a),b?(this._lastDrawTimestamp=this.device.timestamp,this._needsRedraw=!1):d?this._needsRedraw=jI:this._needsRedraw="waiting for resource initialization",b}setGeometry(a){this._gpuGeometry?.destroy();let b=a&&function(a,b){if(b instanceof jg)return b;let c=function(a,b){if(!b.indices)return;let c=b.indices.value;return a.createBuffer({usage:hF.Buffer.INDEX,data:c})}(a,b),{attributes:d,bufferLayout:e}=function(a,b){let c=[],d={};for(let[e,f]of Object.entries(b.attributes)){let b=e;switch(e){case"POSITION":b="positions";break;case"NORMAL":b="normals";break;case"TEXCOORD_0":b="texCoords";break;case"TEXCOORD_1":b="texCoords1";break;case"COLOR_0":b="colors"}if(f){d[b]=a.createBuffer({data:f.value,id:`${e}-buffer`});let{value:g,size:h,normalized:i}=f;if(void 0===h)throw Error(`Attribute ${e} is missing a size`);c.push({name:b,format:jd.vertexFormatDecoder.getVertexFormatFromAttribute(g,h,i)})}}return{attributes:d,bufferLayout:c,vertexCount:b._calculateVertexCount(b.attributes,b.indices)}}(a,b);return new jg({topology:b.topology||"triangle-list",bufferLayout:e,vertexCount:b.vertexCount,indices:c,attributes:d})}(this.device,a);if(b){this.setTopology(b.topology||"triangle-list");let a=new jj(this.bufferLayout);this.bufferLayout=a.mergeBufferLayouts(b.bufferLayout,this.bufferLayout),this.vertexArray&&this._setGeometryAttributes(b)}this._gpuGeometry=b}setTopology(a){a!==this.topology&&(this.topology=a,this._setPipelineNeedsUpdate("topology"))}setBufferLayout(a){let b=new jj(this.bufferLayout);this.bufferLayout=this._gpuGeometry?b.mergeBufferLayouts(a,this._gpuGeometry.bufferLayout):a,this._setPipelineNeedsUpdate("bufferLayout"),this.pipeline=this._updatePipeline(),this.vertexArray=this.device.createVertexArray({shaderLayout:this.pipeline.shaderLayout,bufferLayout:this.pipeline.bufferLayout}),this._gpuGeometry&&this._setGeometryAttributes(this._gpuGeometry)}setParameters(a){!function a(b,c,d){if(b===c)return!0;if(!d||!b||!c)return!1;if(Array.isArray(b)){if(!Array.isArray(c)||b.length!==c.length)return!1;for(let e=0;e<b.length;e++)if(!a(b[e],c[e],d-1))return!1;return!0}if(Array.isArray(c))return!1;if("object"==typeof b&&"object"==typeof c){let e=Object.keys(b),f=Object.keys(c);if(e.length!==f.length)return!1;for(let f of e)if(!c.hasOwnProperty(f)||!a(b[f],c[f],d-1))return!1;return!0}return!1}(a,this.parameters,2)&&(this.parameters=a,this._setPipelineNeedsUpdate("parameters"))}setInstanceCount(a){this.instanceCount=a,void 0===this.isInstanced&&a>0&&(this.isInstanced=!0),this.setNeedsRedraw("instanceCount")}setVertexCount(a){this.vertexCount=a,this.setNeedsRedraw("vertexCount")}setShaderInputs(a){for(let[b,c]of(this.shaderInputs=a,this._uniformStore=new ja(this.device,this.shaderInputs.modules),Object.entries(this.shaderInputs.modules)))if(c.uniformTypes&&!function(a){for(let b in a)return!1;return!0}(c.uniformTypes)&&!this.material?.ownsModule(b)){let a=this._uniformStore.getManagedUniformBuffer(b);this.bindings[`${b}Uniforms`]=a}this.setNeedsRedraw("shaderInputs")}setMaterial(a){this.material=a,this.setNeedsRedraw("material")}updateShaderInputs(){this._uniformStore.setUniforms(this.shaderInputs.getUniformValues()),this.setBindings(this._getNonMaterialBindings(this.shaderInputs.getBindingValues())),this.setNeedsRedraw("shaderInputs")}setBindings(a){Object.assign(this.bindings,a),this.setNeedsRedraw("bindings")}setTransformFeedback(a){this.transformFeedback=a,this.setNeedsRedraw("transformFeedback")}setIndexBuffer(a){this.vertexArray.setIndexBuffer(a),this.setNeedsRedraw("indexBuffer")}setAttributes(a,b){var c,d;let e,f,g=b?.disableWarnings??this.props.disableWarnings;a.indices&&bB.log.warn(`Model:${this.id} setAttributes() - indexBuffer should be set using setIndexBuffer()`)(),this.bufferLayout=(c=this.pipeline.shaderLayout,d=this.bufferLayout,e=Object.fromEntries(c.attributes.map(a=>[a.name,a.location])),(f=d.slice()).sort((a,b)=>{let c=a.attributes?a.attributes.map(a=>a.attribute):[a.name],d=b.attributes?b.attributes.map(a=>a.attribute):[b.name];return jk(c,e)-jk(d,e)}),f);let h=new jj(this.bufferLayout);for(let[b,c]of Object.entries(a)){let a=h.getBufferLayout(b);if(!a){g||bB.log.warn(`Model(${this.id}): Missing layout for buffer "${b}".`)();continue}let d=h.getAttributeNamesForBuffer(a),e=!1;for(let a of d){let b=this._attributeInfos[a];if(b){let a="webgpu"===this.device.type?h.getBufferIndex(b.bufferName):b.location;this.vertexArray.setBuffer(a,c),e=!0}}e||g||bB.log.warn(`Model(${this.id}): Ignoring buffer "${c.id}" for unknown attribute "${b}"`)()}this.setNeedsRedraw("attributes")}setConstantAttributes(a,b){for(let[c,d]of Object.entries(a)){let a=this._attributeInfos[c];a?this.vertexArray.setConstantWebGL(a.location,d):(b?.disableWarnings??this.props.disableWarnings)||bB.log.warn(`Model "${this.id}: Ignoring constant supplied for unknown attribute "${c}"`)()}this.setNeedsRedraw("constants")}_areBindingsLoading(){for(let a of Object.values(this.bindings))if(a instanceof jE&&!a.isReady)return a.id;for(let a of Object.values(this.material?.bindings||{}))if(a instanceof jE&&!a.isReady)return a.id;return!1}_getBindings(){let a={};for(let[b,c]of Object.entries(this.bindings))c instanceof jE?c.isReady&&(a[b]=c.texture):a[b]=c;return a}_getBindGroups(){let a=this.pipeline?.shaderLayout||this.props.shaderLayout||{bindings:[]},b=a.bindings.length?(0,jc.normalizeBindingsByGroup)(a,this._getBindings()):{0:this._getBindings()};if(!this.material)return b;for(let[a,c]of Object.entries(this.material.getBindingsByGroup())){let d=Number(a);b[d]={...b[d]||{},...c}}return b}_getBindGroupCacheKeys(){let a=this.material?.getBindGroupCacheKey(3);return a?{3:a}:{}}_getBindingsUpdateTimestamp(){let a=0;for(let b of Object.values(this.bindings))b instanceof iT.TextureView?a=Math.max(a,b.texture.updateTimestamp):b instanceof hF.Buffer||b instanceof hG.Texture?a=Math.max(a,b.updateTimestamp):b instanceof jE?a=b.texture?Math.max(a,b.texture.updateTimestamp):1/0:b instanceof iU.Sampler||(a=Math.max(a,b.buffer.updateTimestamp));return Math.max(a,this.material?.getBindingsUpdateTimestamp()||0)}_setGeometryAttributes(a){let b={...a.attributes};for(let[a]of Object.entries(b))this.pipeline.shaderLayout.attributes.find(b=>b.name===a)||"positions"===a||delete b[a];this.vertexCount=a.vertexCount,this.setIndexBuffer(a.indices||null),this.setAttributes(a.attributes,{disableWarnings:!0}),this.setAttributes(b,{disableWarnings:this.props.disableWarnings}),this.setNeedsRedraw("geometry attributes")}_setPipelineNeedsUpdate(a){this._pipelineNeedsUpdate||=a,this.setNeedsRedraw(a)}_updatePipeline(){if(this._pipelineNeedsUpdate){let a=null,b=null;this.pipeline&&(bB.log.log(1,`Model ${this.id}: Recreating pipeline because "${this._pipelineNeedsUpdate}".`)(),a=this.pipeline.vs,b=this.pipeline.fs),this._pipelineNeedsUpdate=!1;let c=this.shaderFactory.createShader({id:`${this.id}-vertex`,stage:"vertex",source:this.source||this.vs,debugShaders:this.props.debugShaders}),d=null;this.source?d=c:this.fs&&(d=this.shaderFactory.createShader({id:`${this.id}-fragment`,stage:"fragment",source:this.source||this.fs,debugShaders:this.props.debugShaders})),this.pipeline=this.pipelineFactory.createRenderPipeline({...this.props,bindings:void 0,bufferLayout:this.bufferLayout,topology:this.topology,parameters:this.parameters,bindGroups:this._getBindGroups(),vs:c,fs:d}),this._attributeInfos=(0,jb.getAttributeInfosFromLayouts)(this.pipeline.shaderLayout,this.bufferLayout),a&&this.shaderFactory.release(a),b&&b!==a&&this.shaderFactory.release(b)}return this.pipeline}_lastLogTime=0;_logOpen=!1;_logDrawCallStart(){let a=bB.log.level>3?0:1e4;bB.log.level<2||Date.now()-this._lastLogTime<a||(this._lastLogTime=Date.now(),this._logOpen=!0,bB.log.group(2,`>>> DRAWING MODEL ${this.id}`,{collapsed:bB.log.level<=2})())}_logDrawCallEnd(){if(this._logOpen){let a=function(a,b){let c={},d="Values";if(0===a.attributes.length&&!a.varyings?.length)return{"No attributes or varyings":{[d]:"N/A"}};for(let b of a.attributes)if(b){let a=`${b.location} ${b.name}: ${b.type}`;c[`in ${a}`]={[d]:b.stepMode||"vertex"}}for(let b of a.varyings||[]){let a=`${b.location} ${b.name}`;c[`out ${a}`]={[d]:JSON.stringify(b)}}return c}(this.pipeline.shaderLayout,this.id);bB.log.table(2,a)();let b=this.shaderInputs.getDebugTable();bB.log.table(2,b)();let c=this._getAttributeDebugTable();bB.log.table(2,this._attributeInfos)(),bB.log.table(2,c)(),bB.log.groupEnd(2)(),this._logOpen=!1}}_drawCount=0;_logFramebuffer(a){let b=this.device.props.debugFramebuffers;if(this._drawCount++,!b)return;let c=a.props.framebuffer;!function(a,b,c){var d;if("webgl"!==a.device.type)return;let e=(d=a.device,d.userData[jh]||={flushing:!1,queuedFramebuffers:[]},d.userData[jh]);if(!e.flushing){let d;if(!(d=a.props.framebuffer)||null===d.handle)return function(a,b,c){if(0===c.queuedFramebuffers.length)return;let{gl:d}=a.device,e=d.getParameter(36010),f=d.getParameter(36006),[g,h]=a.device.getDefaultCanvasContext().getDrawingBufferSize(),i=ji(b.top,8),j=ji(b.left,8);c.flushing=!0;try{for(let a of c.queuedFramebuffers){let[c,e,f,k,l]=function(a){let{framebuffer:b,targetWidth:c,targetHeight:d,topPx:e,leftPx:f,minimap:g}=a,h=g?Math.max(Math.floor(c/4),1):c,i=g?Math.max(Math.floor(d/4),1):d,j=Math.min(h/b.width,i/b.height),k=Math.max(Math.floor(b.width*j),1),l=Math.max(Math.floor(b.height*j),1),m=Math.max(d-e-l,0);return[f,m,f+k,m+l,l]}({framebuffer:a,targetWidth:g,targetHeight:h,topPx:i,leftPx:j,minimap:b.minimap});d.bindFramebuffer(36008,a.handle),d.bindFramebuffer(36009,null),d.blitFramebuffer(0,0,a.width,a.height,c,e,f,k,16384,9728),i+=l+8}}finally{d.bindFramebuffer(36008,e),d.bindFramebuffer(36009,f),c.flushing=!1}}(a,c,e);b&&"colorAttachments"in b&&null!==b.handle&&!e.queuedFramebuffers.includes(b)&&e.queuedFramebuffers.push(b)}}(a,c,{id:c?.id||`${this.id}-framebuffer`,minimap:!0})}_getAttributeDebugTable(){let a={};for(let[b,c]of Object.entries(this._attributeInfos)){let d=this.vertexArray.attributes[c.location];a[c.location]={name:b,type:c.shaderType,values:d?this._getBufferOrConstantValues(d,c.bufferDataType):"null"}}if(this.vertexArray.indexBuffer){let{indexBuffer:b}=this.vertexArray,c="uint32"===b.indexType?new Uint32Array(b.debugData):new Uint16Array(b.debugData);a.indices={name:"indices",type:b.indexType,values:c.toString()}}return a}_getBufferOrConstantValues(a,b){let c=iz.dataTypeDecoder.getTypedArrayConstructor(b);return(a instanceof hF.Buffer?new c(a.debugData):a).toString()}_getNonMaterialBindings(a){if(!this.material)return a;let b={};for(let[c,d]of Object.entries(a))this.material.ownsBinding(c)||(b[c]=d);return b}}class jK{device;model;transformFeedback;static defaultProps={...jJ.defaultProps,outputs:void 0,feedbackBuffers:void 0};static isSupported(a){return a?.info?.type==="webgl"}constructor(a,b=jK.defaultProps){if(!jK.isSupported(a))throw Error("BufferTransform not yet implemented on WebGPU");this.device=a,this.model=new jJ(this.device,{id:b.id||"buffer-transform-model",fs:b.fs||iS(),topology:b.topology||"point-list",varyings:b.outputs||b.varyings,...b}),this.transformFeedback=this.device.createTransformFeedback({layout:this.model.pipeline.shaderLayout,buffers:b.feedbackBuffers}),this.model.setTransformFeedback(this.transformFeedback),Object.seal(this)}destroy(){this.model&&this.model.destroy()}delete(){this.destroy()}run(a){a?.inputBuffers&&this.model.setAttributes(a.inputBuffers),a?.outputBuffers&&this.transformFeedback.setBuffers(a.outputBuffers);let b=this.device.beginRenderPass(a);this.model.draw(b),b.end()}getBuffer(a){return this.transformFeedback.getBuffer(a)}readAsync(a){let b=this.getBuffer(a);if(!b)throw Error("BufferTransform#getBuffer");if(b instanceof hF.Buffer)return b.readAsync();let{buffer:c,byteOffset:d=0,byteLength:e=c.byteLength}=b;return c.readAsync(d,e)}}function jL(a,b=[],c=0){let d=Math.fround(a),e=a-d;return b[c]=d,b[c+1]=e,b}let jM=`\

layout(std140) uniform fp64arithmeticUniforms {
  uniform float ONE;
  uniform float SPLIT;
} fp64;

/*
About LUMA_FP64_CODE_ELIMINATION_WORKAROUND

The purpose of this workaround is to prevent shader compilers from
optimizing away necessary arithmetic operations by swapping their sequences
or transform the equation to some 'equivalent' form.

These helpers implement Dekker/Veltkamp-style error tracking. If the compiler
folds constants or reassociates the arithmetic, the high/low split can stop
tracking the rounding error correctly. That failure mode tends to look fine in
simple coordinate setup, but then breaks down inside iterative arithmetic such
as fp64 Mandelbrot loops.

The method is to multiply an artifical variable, ONE, which will be known to
the compiler to be 1 only at runtime. The whole expression is then represented
as a polynomial with respective to ONE. In the coefficients of all terms, only one a
and one b should appear

err = (a + b) * ONE^6 - a * ONE^5 - (a + b) * ONE^4 + a * ONE^3 - b - (a + b) * ONE^2 + a * ONE
*/

float prevent_fp64_optimization(float value) {
#if defined(LUMA_FP64_CODE_ELIMINATION_WORKAROUND)
  return value + fp64.ONE * 0.0;
#else
  return value;
#endif
}

// Divide float number to high and low floats to extend fraction bits
vec2 split(float a) {
  // Keep SPLIT as a runtime uniform so the compiler cannot fold the Dekker
  // split into a constant expression and reassociate the recovery steps.
  float split = prevent_fp64_optimization(fp64.SPLIT);
  float t = prevent_fp64_optimization(a * split);
  float temp = t - a;
  float a_hi = t - temp;
  float a_lo = a - a_hi;
  return vec2(a_hi, a_lo);
}

// Divide float number again when high float uses too many fraction bits
vec2 split2(vec2 a) {
  vec2 b = split(a.x);
  b.y += a.y;
  return b;
}

// Special sum operation when a > b
vec2 quickTwoSum(float a, float b) {
#if defined(LUMA_FP64_CODE_ELIMINATION_WORKAROUND)
  float sum = (a + b) * fp64.ONE;
  float err = b - (sum - a) * fp64.ONE;
#else
  float sum = a + b;
  float err = b - (sum - a);
#endif
  return vec2(sum, err);
}

// General sum operation
vec2 twoSum(float a, float b) {
  float s = (a + b);
#if defined(LUMA_FP64_CODE_ELIMINATION_WORKAROUND)
  float v = (s * fp64.ONE - a) * fp64.ONE;
  float err = (a - (s - v) * fp64.ONE) * fp64.ONE * fp64.ONE * fp64.ONE + (b - v);
#else
  float v = s - a;
  float err = (a - (s - v)) + (b - v);
#endif
  return vec2(s, err);
}

vec2 twoSub(float a, float b) {
  float s = (a - b);
#if defined(LUMA_FP64_CODE_ELIMINATION_WORKAROUND)
  float v = (s * fp64.ONE - a) * fp64.ONE;
  float err = (a - (s - v) * fp64.ONE) * fp64.ONE * fp64.ONE * fp64.ONE - (b + v);
#else
  float v = s - a;
  float err = (a - (s - v)) - (b + v);
#endif
  return vec2(s, err);
}

vec2 twoSqr(float a) {
  float prod = a * a;
  vec2 a_fp64 = split(a);
#if defined(LUMA_FP64_CODE_ELIMINATION_WORKAROUND)
  float err = ((a_fp64.x * a_fp64.x - prod) * fp64.ONE + 2.0 * a_fp64.x *
    a_fp64.y * fp64.ONE * fp64.ONE) + a_fp64.y * a_fp64.y * fp64.ONE * fp64.ONE * fp64.ONE;
#else
  float err = ((a_fp64.x * a_fp64.x - prod) + 2.0 * a_fp64.x * a_fp64.y) + a_fp64.y * a_fp64.y;
#endif
  return vec2(prod, err);
}

vec2 twoProd(float a, float b) {
  float prod = a * b;
  vec2 a_fp64 = split(a);
  vec2 b_fp64 = split(b);
  // twoProd is especially sensitive because mul_fp64 and div_fp64 both depend
  // on the split terms and cross terms staying in the original evaluation
  // order. If the compiler folds or reassociates them, the low part tends to
  // collapse to zero or NaN on some drivers.
  float highProduct = prevent_fp64_optimization(a_fp64.x * b_fp64.x);
  float crossProduct1 = prevent_fp64_optimization(a_fp64.x * b_fp64.y);
  float crossProduct2 = prevent_fp64_optimization(a_fp64.y * b_fp64.x);
  float lowProduct = prevent_fp64_optimization(a_fp64.y * b_fp64.y);
#if defined(LUMA_FP64_CODE_ELIMINATION_WORKAROUND)
  float err1 = (highProduct - prod) * fp64.ONE;
  float err2 = crossProduct1 * fp64.ONE * fp64.ONE;
  float err3 = crossProduct2 * fp64.ONE * fp64.ONE * fp64.ONE;
  float err4 = lowProduct * fp64.ONE * fp64.ONE * fp64.ONE * fp64.ONE;
#else
  float err1 = highProduct - prod;
  float err2 = crossProduct1;
  float err3 = crossProduct2;
  float err4 = lowProduct;
#endif
  float err = ((err1 + err2) + err3) + err4;
  return vec2(prod, err);
}

vec2 sum_fp64(vec2 a, vec2 b) {
  vec2 s, t;
  s = twoSum(a.x, b.x);
  t = twoSum(a.y, b.y);
  s.y += t.x;
  s = quickTwoSum(s.x, s.y);
  s.y += t.y;
  s = quickTwoSum(s.x, s.y);
  return s;
}

vec2 sub_fp64(vec2 a, vec2 b) {
  vec2 s, t;
  s = twoSub(a.x, b.x);
  t = twoSub(a.y, b.y);
  s.y += t.x;
  s = quickTwoSum(s.x, s.y);
  s.y += t.y;
  s = quickTwoSum(s.x, s.y);
  return s;
}

vec2 mul_fp64(vec2 a, vec2 b) {
  vec2 prod = twoProd(a.x, b.x);
  // y component is for the error
  prod.y += a.x * b.y;
#if defined(LUMA_FP64_HIGH_BITS_OVERFLOW_WORKAROUND)
  prod = split2(prod);
#endif
  prod = quickTwoSum(prod.x, prod.y);
  prod.y += a.y * b.x;
#if defined(LUMA_FP64_HIGH_BITS_OVERFLOW_WORKAROUND)
  prod = split2(prod);
#endif
  prod = quickTwoSum(prod.x, prod.y);
  return prod;
}

vec2 div_fp64(vec2 a, vec2 b) {
  float xn = 1.0 / b.x;
#if defined(LUMA_FP64_HIGH_BITS_OVERFLOW_WORKAROUND)
  vec2 yn = mul_fp64(a, vec2(xn, 0));
#else
  vec2 yn = a * xn;
#endif
  float diff = (sub_fp64(a, mul_fp64(b, yn))).x;
  vec2 prod = twoProd(xn, diff);
  return sum_fp64(yn, prod);
}

vec2 sqrt_fp64(vec2 a) {
  if (a.x == 0.0 && a.y == 0.0) return vec2(0.0, 0.0);
  if (a.x < 0.0) return vec2(0.0 / 0.0, 0.0 / 0.0);

  float x = 1.0 / sqrt(a.x);
  float yn = a.x * x;
#if defined(LUMA_FP64_CODE_ELIMINATION_WORKAROUND)
  vec2 yn_sqr = twoSqr(yn) * fp64.ONE;
#else
  vec2 yn_sqr = twoSqr(yn);
#endif
  float diff = sub_fp64(a, yn_sqr).x;
  vec2 prod = twoProd(x * 0.5, diff);
#if defined(LUMA_FP64_HIGH_BITS_OVERFLOW_WORKAROUND)
  return sum_fp64(split(yn), prod);
#else
  return sum_fp64(vec2(yn, 0.0), prod);
#endif
}
`,jN={name:"fp64arithmetic",source:`\
struct Fp64ArithmeticUniforms {
  ONE: f32,
  SPLIT: f32,
};

@group(0) @binding(auto) var<uniform> fp64arithmetic : Fp64ArithmeticUniforms;

fn fp64_nan(seed: f32) -> f32 {
  let nanBits = 0x7fc00000u | select(0u, 1u, seed < 0.0);
  return bitcast<f32>(nanBits);
}

fn fp64_runtime_zero() -> f32 {
  return fp64arithmetic.ONE * 0.0;
}

fn prevent_fp64_optimization(value: f32) -> f32 {
#ifdef LUMA_FP64_CODE_ELIMINATION_WORKAROUND
  return value + fp64_runtime_zero();
#else
  return value;
#endif
}

fn split(a: f32) -> vec2f {
  let splitValue = prevent_fp64_optimization(fp64arithmetic.SPLIT + fp64_runtime_zero());
  let t = prevent_fp64_optimization(a * splitValue);
  let temp = prevent_fp64_optimization(t - a);
  let aHi = prevent_fp64_optimization(t - temp);
  let aLo = prevent_fp64_optimization(a - aHi);
  return vec2f(aHi, aLo);
}

fn split2(a: vec2f) -> vec2f {
  var b = split(a.x);
  b.y = b.y + a.y;
  return b;
}

fn quickTwoSum(a: f32, b: f32) -> vec2f {
#ifdef LUMA_FP64_CODE_ELIMINATION_WORKAROUND
  let sum = prevent_fp64_optimization((a + b) * fp64arithmetic.ONE);
  let err = prevent_fp64_optimization(b - (sum - a) * fp64arithmetic.ONE);
#else
  let sum = prevent_fp64_optimization(a + b);
  let err = prevent_fp64_optimization(b - (sum - a));
#endif
  return vec2f(sum, err);
}

fn twoSum(a: f32, b: f32) -> vec2f {
  let s = prevent_fp64_optimization(a + b);
#ifdef LUMA_FP64_CODE_ELIMINATION_WORKAROUND
  let v = prevent_fp64_optimization((s * fp64arithmetic.ONE - a) * fp64arithmetic.ONE);
  let err =
    prevent_fp64_optimization((a - (s - v) * fp64arithmetic.ONE) *
      fp64arithmetic.ONE *
      fp64arithmetic.ONE *
      fp64arithmetic.ONE) +
    prevent_fp64_optimization(b - v);
#else
  let v = prevent_fp64_optimization(s - a);
  let err = prevent_fp64_optimization(a - (s - v)) + prevent_fp64_optimization(b - v);
#endif
  return vec2f(s, err);
}

fn twoSub(a: f32, b: f32) -> vec2f {
  let s = prevent_fp64_optimization(a - b);
#ifdef LUMA_FP64_CODE_ELIMINATION_WORKAROUND
  let v = prevent_fp64_optimization((s * fp64arithmetic.ONE - a) * fp64arithmetic.ONE);
  let err =
    prevent_fp64_optimization((a - (s - v) * fp64arithmetic.ONE) *
      fp64arithmetic.ONE *
      fp64arithmetic.ONE *
      fp64arithmetic.ONE) -
    prevent_fp64_optimization(b + v);
#else
  let v = prevent_fp64_optimization(s - a);
  let err = prevent_fp64_optimization(a - (s - v)) - prevent_fp64_optimization(b + v);
#endif
  return vec2f(s, err);
}

fn twoSqr(a: f32) -> vec2f {
  let prod = prevent_fp64_optimization(a * a);
  let aFp64 = split(a);
  let highProduct = prevent_fp64_optimization(aFp64.x * aFp64.x);
  let crossProduct = prevent_fp64_optimization(2.0 * aFp64.x * aFp64.y);
  let lowProduct = prevent_fp64_optimization(aFp64.y * aFp64.y);
#ifdef LUMA_FP64_CODE_ELIMINATION_WORKAROUND
  let err =
    (prevent_fp64_optimization(highProduct - prod) * fp64arithmetic.ONE +
      crossProduct * fp64arithmetic.ONE * fp64arithmetic.ONE) +
    lowProduct * fp64arithmetic.ONE * fp64arithmetic.ONE * fp64arithmetic.ONE;
#else
  let err = ((prevent_fp64_optimization(highProduct - prod) + crossProduct) + lowProduct);
#endif
  return vec2f(prod, err);
}

fn twoProd(a: f32, b: f32) -> vec2f {
  let prod = prevent_fp64_optimization(a * b);
  let aFp64 = split(a);
  let bFp64 = split(b);
  let highProduct = prevent_fp64_optimization(aFp64.x * bFp64.x);
  let crossProduct1 = prevent_fp64_optimization(aFp64.x * bFp64.y);
  let crossProduct2 = prevent_fp64_optimization(aFp64.y * bFp64.x);
  let lowProduct = prevent_fp64_optimization(aFp64.y * bFp64.y);
#ifdef LUMA_FP64_CODE_ELIMINATION_WORKAROUND
  let err1 = (highProduct - prod) * fp64arithmetic.ONE;
  let err2 = crossProduct1 * fp64arithmetic.ONE * fp64arithmetic.ONE;
  let err3 = crossProduct2 * fp64arithmetic.ONE * fp64arithmetic.ONE * fp64arithmetic.ONE;
  let err4 =
    lowProduct *
    fp64arithmetic.ONE *
    fp64arithmetic.ONE *
    fp64arithmetic.ONE *
    fp64arithmetic.ONE;
#else
  let err1 = highProduct - prod;
  let err2 = crossProduct1;
  let err3 = crossProduct2;
  let err4 = lowProduct;
#endif
  let err12InputA = prevent_fp64_optimization(err1);
  let err12InputB = prevent_fp64_optimization(err2);
  let err12 = prevent_fp64_optimization(err12InputA + err12InputB);
  let err123InputA = prevent_fp64_optimization(err12);
  let err123InputB = prevent_fp64_optimization(err3);
  let err123 = prevent_fp64_optimization(err123InputA + err123InputB);
  let err1234InputA = prevent_fp64_optimization(err123);
  let err1234InputB = prevent_fp64_optimization(err4);
  let err = prevent_fp64_optimization(err1234InputA + err1234InputB);
  return vec2f(prod, err);
}

fn sum_fp64(a: vec2f, b: vec2f) -> vec2f {
  var s = twoSum(a.x, b.x);
  let t = twoSum(a.y, b.y);
  s.y = prevent_fp64_optimization(s.y + t.x);
  s = quickTwoSum(s.x, s.y);
  s.y = prevent_fp64_optimization(s.y + t.y);
  s = quickTwoSum(s.x, s.y);
  return s;
}

fn sub_fp64(a: vec2f, b: vec2f) -> vec2f {
  var s = twoSub(a.x, b.x);
  let t = twoSub(a.y, b.y);
  s.y = prevent_fp64_optimization(s.y + t.x);
  s = quickTwoSum(s.x, s.y);
  s.y = prevent_fp64_optimization(s.y + t.y);
  s = quickTwoSum(s.x, s.y);
  return s;
}

fn mul_fp64(a: vec2f, b: vec2f) -> vec2f {
  var prod = twoProd(a.x, b.x);
  let crossProduct1 = prevent_fp64_optimization(a.x * b.y);
  prod.y = prevent_fp64_optimization(prod.y + crossProduct1);
#ifdef LUMA_FP64_HIGH_BITS_OVERFLOW_WORKAROUND
  prod = split2(prod);
#endif
  prod = quickTwoSum(prod.x, prod.y);
  let crossProduct2 = prevent_fp64_optimization(a.y * b.x);
  prod.y = prevent_fp64_optimization(prod.y + crossProduct2);
#ifdef LUMA_FP64_HIGH_BITS_OVERFLOW_WORKAROUND
  prod = split2(prod);
#endif
  prod = quickTwoSum(prod.x, prod.y);
  return prod;
}

fn div_fp64(a: vec2f, b: vec2f) -> vec2f {
  let xn = prevent_fp64_optimization(1.0 / b.x);
  let yn = mul_fp64(a, vec2f(xn, fp64_runtime_zero()));
  let diff = prevent_fp64_optimization(sub_fp64(a, mul_fp64(b, yn)).x);
  let prod = twoProd(xn, diff);
  return sum_fp64(yn, prod);
}

fn sqrt_fp64(a: vec2f) -> vec2f {
  if (a.x == 0.0 && a.y == 0.0) {
    return vec2f(0.0, 0.0);
  }
  if (a.x < 0.0) {
    let nanValue = fp64_nan(a.x);
    return vec2f(nanValue, nanValue);
  }

  let x = prevent_fp64_optimization(1.0 / sqrt(a.x));
  let yn = prevent_fp64_optimization(a.x * x);
#ifdef LUMA_FP64_CODE_ELIMINATION_WORKAROUND
  let ynSqr = twoSqr(yn) * fp64arithmetic.ONE;
#else
  let ynSqr = twoSqr(yn);
#endif
  let diff = prevent_fp64_optimization(sub_fp64(a, ynSqr).x);
  let prod = twoProd(prevent_fp64_optimization(x * 0.5), diff);
#ifdef LUMA_FP64_HIGH_BITS_OVERFLOW_WORKAROUND
  return sum_fp64(split(yn), prod);
#else
  return sum_fp64(vec2f(yn, 0.0), prod);
#endif
}
`,fs:jM,vs:jM,defaultUniforms:{ONE:1,SPLIT:4097},uniformTypes:{ONE:"f32",SPLIT:"f32"},fp64ify:jL,fp64LowPart:function(a){return a-Math.fround(a)},fp64ifyMatrix4:function(a){let b=new Float32Array(32);for(let c=0;c<4;++c)for(let d=0;d<4;++d){let e=4*c+d;jL(a[4*d+c],b,2*e)}return b}};function jO(a){let{source:b,target:c,start:d=0,size:e,getData:f}=a,g=a.end||c.length,h=b.length,i=g-d;if(h>i)return void c.set(b.subarray(0,i),d);if(c.set(b,d),!f)return;let j=h;for(;j<i;){let a=f(j,b);for(let b=0;b<e;b++)c[d+j]=a[b]||0,j++}}function jP(a){switch(a){case 1:return"float";case 2:return"vec2";case 3:return"vec3";case 4:return"vec4";default:throw Error(`No defined attribute type for size "${a}"`)}}function jQ(a){switch(a){case 1:return"float32";case 2:return"float32x2";case 3:return"float32x3";case 4:return"float32x4";default:throw Error("invalid type size")}}function jR(a){a.push(a.shift())}function jS({device:a,source:b,target:c}){return(!c||c.byteLength<b.byteLength)&&(c?.destroy(),c=a.createBuffer({byteLength:b.byteLength,usage:b.usage})),c}function jT({device:a,buffer:b,attribute:c,fromLength:d,toLength:e,fromStartIndices:f,getData:g=a=>a}){let h=c.doublePrecision&&c.value instanceof Float64Array?2:1,i=c.size*h,j=c.byteOffset,k=c.settings.bytesPerElement<4?j/c.settings.bytesPerElement*4:j,l=c.startIndices,m=f&&l,n=c.isConstant;if(!m&&b&&d>=e)return b;let o=c.value instanceof Float64Array?Float32Array:c.value.constructor,p=n?c.value:new o(c.getBuffer().readSyncWebGL(j,e*o.BYTES_PER_ELEMENT).buffer);if(c.settings.normalized&&!n){let a=g;g=(b,d)=>c.normalizeConstant(a(b,d))}let q=n?(a,b)=>g(p,b):(a,b)=>g(p.subarray(a+j,a+j+i),b),r=new Float32Array(b?b.readSyncWebGL(k,4*d).buffer:0),s=new Float32Array(e);return!function({source:a,target:b,size:c,getData:d,sourceStartIndices:e,targetStartIndices:f}){if(!e||!f)return jO({source:a,target:b,size:c,getData:d});let g=0,h=0,i=d&&((a,b)=>d(a+h,b)),j=Math.min(e.length,f.length);for(let d=1;d<j;d++){let j=e[d]*c,k=f[d]*c;jO({source:a.subarray(g,j),target:b,start:h,end:k,size:c,getData:i}),g=j,h=k}h<b.length&&jO({source:[],target:b,start:h,size:c,getData:i})}({source:r,target:s,sourceStartIndices:f,targetStartIndices:l,size:i,getData:q}),(!b||b.byteLength<s.byteLength+k)&&(b?.destroy(),b=a.createBuffer({byteLength:s.byteLength+k,usage:35050})),b.write(s,k),b}class jU{constructor({device:a,attribute:b,timeline:c}){this.buffers=[],this.currentLength=0,this.device=a,this.transition=new gP(c),this.attribute=b,this.attributeInTransition=function(a){let{device:b,settings:c,value:d}=a,e=new iP(b,c);return e.setData({value:d instanceof Float64Array?new Float64Array(0):new Float32Array(0),normalized:c.normalized}),e}(b),this.currentStartIndices=b.startIndices}get inProgress(){return this.transition.inProgress}start(a,b,c=1/0){this.settings=a,this.currentStartIndices=this.attribute.startIndices,this.currentLength=function(a,b){let{doublePrecision:c,settings:d,value:e,size:f}=a,g=c&&e instanceof Float64Array?2:1,h=0,{shaderAttributes:i}=a.settings;if(i)for(let a of Object.values(i))h=Math.max(h,a.vertexOffset??0);return(d.noAlloc?e.length:(b+h)*f)*g}(this.attribute,b),this.transition.start({...a,duration:c})}update(){let a=this.transition.update();return a&&this.onUpdate(),a}setBuffer(a){this.attributeInTransition.setData({buffer:a,normalized:this.attribute.settings.normalized,value:this.attributeInTransition.value})}cancel(){this.transition.cancel()}delete(){for(let a of(this.cancel(),this.buffers))a.destroy();this.buffers.length=0}}let jV={name:"interpolation",vs:`\
layout(std140) uniform interpolationUniforms {
  float time;
} interpolation;
`,uniformTypes:{time:"f32"}},jW=`\
#version 300 es
#define SHADER_NAME interpolation-transition-vertex-shader

in ATTRIBUTE_TYPE aFrom;
in ATTRIBUTE_TYPE aTo;
out ATTRIBUTE_TYPE vCurrent;

void main(void) {
  vCurrent = mix(aFrom, aTo, interpolation.time);
  gl_Position = vec4(0.0);
}
`,jX=`\
#version 300 es
#define SHADER_NAME interpolation-transition-vertex-shader

in ATTRIBUTE_TYPE aFrom;
in ATTRIBUTE_TYPE aFrom64Low;
in ATTRIBUTE_TYPE aTo;
in ATTRIBUTE_TYPE aTo64Low;
out ATTRIBUTE_TYPE vCurrent;
out ATTRIBUTE_TYPE vCurrent64Low;

vec2 mix_fp64(vec2 a, vec2 b, float x) {
  vec2 range = sub_fp64(b, a);
  return sum_fp64(a, mul_fp64(range, vec2(x, 0.0)));
}

void main(void) {
  for (int i=0; i<ATTRIBUTE_SIZE; i++) {
    vec2 value = mix_fp64(vec2(aFrom[i], aFrom64Low[i]), vec2(aTo[i], aTo64Low[i]), interpolation.time);
    vCurrent[i] = value.x;
    vCurrent64Low[i] = value.y;
  }
  gl_Position = vec4(0.0);
}
`;function jY(a){return a.doublePrecision&&a.value instanceof Float64Array}let jZ={name:"spring",vs:`\
layout(std140) uniform springUniforms {
  float damping;
  float stiffness;
} spring;
`,uniformTypes:{damping:"f32",stiffness:"f32"}},j$=`\
#version 300 es
#define SHADER_NAME spring-transition-vertex-shader

#define EPSILON 0.00001

in ATTRIBUTE_TYPE aPrev;
in ATTRIBUTE_TYPE aCur;
in ATTRIBUTE_TYPE aTo;
out ATTRIBUTE_TYPE vNext;
out float vIsTransitioningFlag;

ATTRIBUTE_TYPE getNextValue(ATTRIBUTE_TYPE cur, ATTRIBUTE_TYPE prev, ATTRIBUTE_TYPE dest) {
  ATTRIBUTE_TYPE velocity = cur - prev;
  ATTRIBUTE_TYPE delta = dest - cur;
  ATTRIBUTE_TYPE force = delta * spring.stiffness;
  ATTRIBUTE_TYPE resistance = velocity * spring.damping;
  return force - resistance + velocity + cur;
}

void main(void) {
  bool isTransitioning = length(aCur - aPrev) > EPSILON || length(aTo - aCur) > EPSILON;
  vIsTransitioningFlag = isTransitioning ? 1.0 : 0.0;

  vNext = getNextValue(aCur, aPrev, aTo);
  gl_Position = vec4(0, 0, 0, 1);
  gl_PointSize = 100.0;
}
`,j_=`\
#version 300 es
#define SHADER_NAME spring-transition-is-transitioning-fragment-shader

in float vIsTransitioningFlag;

out vec4 fragColor;

void main(void) {
  if (vIsTransitioningFlag == 0.0) {
    discard;
  }
  fragColor = vec4(1.0);
}`,j0={interpolation:class extends jU{constructor({device:a,attribute:b,timeline:c}){super({device:a,attribute:b,timeline:c}),this.type="interpolation",this.transform=function(a,b){let c=b.size,d=jP(c),e=jQ(c),f=b.getBufferLayout();return jY(b)?new jK(a,{vs:jX,bufferLayout:[{name:"aFrom",byteStride:8*c,attributes:[{attribute:"aFrom",format:e,byteOffset:0},{attribute:"aFrom64Low",format:e,byteOffset:4*c}]},{name:"aTo",byteStride:8*c,attributes:[{attribute:"aTo",format:e,byteOffset:0},{attribute:"aTo64Low",format:e,byteOffset:4*c}]}],modules:[jN,jV],defines:{ATTRIBUTE_TYPE:d,ATTRIBUTE_SIZE:c},moduleSettings:{},varyings:["vCurrent","vCurrent64Low"],bufferMode:35980,disableWarnings:!0}):new jK(a,{vs:jW,bufferLayout:[{name:"aFrom",format:e},{name:"aTo",format:f.attributes[0].format}],modules:[jV],defines:{ATTRIBUTE_TYPE:d},varyings:["vCurrent"],disableWarnings:!0})}(a,b)}start(a,b){let c=this.currentLength,d=this.currentStartIndices;if(super.start(a,b,a.duration),a.duration<=0)return void this.transition.cancel();let{buffers:e,attribute:f}=this;jR(e),e[0]=jT({device:this.device,buffer:e[0],attribute:f,fromLength:c,toLength:this.currentLength,fromStartIndices:d,getData:a.enter}),e[1]=jS({device:this.device,source:e[0],target:e[1]}),this.setBuffer(e[1]);let{transform:g}=this,h=g.model,i=Math.floor(this.currentLength/f.size);jY(f)&&(i/=2),h.setVertexCount(i),f.isConstant?(h.setAttributes({aFrom:e[0]}),h.setConstantAttributes({aTo:f.value})):h.setAttributes({aFrom:e[0],aTo:f.getBuffer()}),g.transformFeedback.setBuffers({vCurrent:e[1]})}onUpdate(){let{duration:a,easing:b}=this.settings,{time:c}=this.transition,d=c/a;b&&(d=b(d));let{model:e}=this.transform,f={time:d};e.shaderInputs.setProps({interpolation:f}),this.transform.run({discard:!0})}delete(){super.delete(),this.transform.destroy()}},spring:class extends jU{constructor({device:a,attribute:b,timeline:c}){super({device:a,attribute:b,timeline:c}),this.type="spring",this.texture=function(a){return a.createTexture({data:new Uint8Array(4),format:"rgba8unorm",width:1,height:1})}(a),this.framebuffer=function(a,b){return a.createFramebuffer({id:"spring-transition-is-transitioning-framebuffer",width:1,height:1,colorAttachments:[b]})}(a,this.texture),this.transform=function(a,b){let c=jP(b.size),d=jQ(b.size);return new jK(a,{vs:j$,fs:j_,bufferLayout:[{name:"aPrev",format:d},{name:"aCur",format:d},{name:"aTo",format:b.getBufferLayout().attributes[0].format}],varyings:["vNext"],modules:[jZ],defines:{ATTRIBUTE_TYPE:c},parameters:{depthCompare:"always",blendColorOperation:"max",blendColorSrcFactor:"one",blendColorDstFactor:"one",blendAlphaOperation:"max",blendAlphaSrcFactor:"one",blendAlphaDstFactor:"one"}})}(a,b)}start(a,b){let c=this.currentLength,d=this.currentStartIndices;super.start(a,b);let{buffers:e,attribute:f}=this;for(let b=0;b<2;b++)e[b]=jT({device:this.device,buffer:e[b],attribute:f,fromLength:c,toLength:this.currentLength,fromStartIndices:d,getData:a.enter});e[2]=jS({device:this.device,source:e[0],target:e[2]}),this.setBuffer(e[1]);let{model:g}=this.transform;g.setVertexCount(Math.floor(this.currentLength/f.size)),f.isConstant?g.setConstantAttributes({aTo:f.value}):g.setAttributes({aTo:f.getBuffer()})}onUpdate(){let{buffers:a,transform:b,framebuffer:c,transition:d}=this,e=this.settings;b.model.setAttributes({aPrev:a[0],aCur:a[1]}),b.transformFeedback.setBuffers({vNext:a[2]});let f={stiffness:e.stiffness,damping:e.damping};b.model.shaderInputs.setProps({spring:f}),b.run({framebuffer:c,discard:!1,parameters:{viewport:[0,0,1,1]},clearColor:[0,0,0,0]}),jR(a),this.setBuffer(a[1]),this.device.readPixelsToArrayWebGL(c)[0]>0||d.end()}delete(){super.delete(),this.transform.destroy(),this.texture.destroy(),this.framebuffer.destroy()}}};class j1{constructor(a,{id:b,timeline:c}){if(!a)throw Error("AttributeTransitionManager is constructed without device");this.id=b,this.device=a,this.timeline=c,this.transitions={},this.needsRedraw=!1,this.numInstances=1}finalize(){for(let a in this.transitions)this._removeTransition(a)}update({attributes:a,transitions:b,numInstances:c}){for(let d in this.numInstances=c||1,a){let c=a[d],e=c.getTransitionSetting(b);e&&this._updateAttribute(d,c,e)}for(let c in this.transitions){let d=a[c];d&&d.getTransitionSetting(b)||this._removeTransition(c)}}hasAttribute(a){let b=this.transitions[a];return b&&b.inProgress}getAttributes(){let a={};for(let b in this.transitions){let c=this.transitions[b];c.inProgress&&(a[b]=c.attributeInTransition)}return a}run(){if(0===this.numInstances)return!1;for(let a in this.transitions)this.transitions[a].update()&&(this.needsRedraw=!0);let a=this.needsRedraw;return this.needsRedraw=!1,a}_removeTransition(a){this.transitions[a].delete(),delete this.transitions[a]}_updateAttribute(a,b,c){let d=this.transitions[a],e=!d||d.type!==c.type;if(e){d&&this._removeTransition(a);let f=j0[c.type];f?this.transitions[a]=new f({attribute:b,timeline:this.timeline,device:this.device}):(cs.error(`unsupported transition type '${c.type}'`)(),e=!1)}(e||b.needsRedraw())&&(this.needsRedraw=!0,this.transitions[a].start(c,this.numInstances))}}let j2="attributeManager.invalidate";class j3{constructor(a,{id:b="attribute-manager",stats:c,timeline:d}={}){this.mergeBoundsMemoized=dd(gv),this.id=b,this.device=a,this.attributes={},this.updateTriggers={},this.needsRedraw=!0,this.userData={},this.stats=c,this.attributeTransitionManager=new j1(a,{id:`${b}-transitions`,timeline:d}),Object.seal(this)}finalize(){for(let a in this.attributes)this.attributes[a].delete();this.attributeTransitionManager.finalize()}getNeedsRedraw(a={clearRedrawFlags:!1}){let b=this.needsRedraw;return this.needsRedraw=this.needsRedraw&&!a.clearRedrawFlags,b&&this.id}setNeedsRedraw(){this.needsRedraw=!0}add(a){this._add(a)}addInstanced(a){this._add(a,{stepMode:"instance"})}remove(a){for(let b of a)void 0!==this.attributes[b]&&(this.attributes[b].delete(),delete this.attributes[b])}invalidate(a,b){let c=this._invalidateTrigger(a,b);eY(j2,this,a,c)}invalidateAll(a){for(let b in this.attributes)this.attributes[b].setNeedsUpdate(b,a);eY(j2,this,"all")}update({data:a,numInstances:b,startIndices:c=null,transitions:d,props:e={},buffers:f={},context:g={}}){let h=!1;for(let d in eY("attributeManager.updateStart",this),this.stats&&this.stats.get("Update Attributes").timeStart(),this.attributes){let i=this.attributes[d],j=i.settings.accessor;i.startIndices=c,i.numInstances=b,e[d]&&cs.removed(`props.${d}`,`data.attributes.${d}`)(),i.setExternalBuffer(f[d])||i.setBinaryValue("string"==typeof j?f[j]:void 0,a.startIndices)||"string"==typeof j&&!f[j]&&i.setConstantValue(g,e[j])||i.needsUpdate()&&(h=!0,this._updateAttribute({attribute:i,numInstances:b,data:a,props:e,context:g})),this.needsRedraw=this.needsRedraw||i.needsRedraw()}h&&eY("attributeManager.updateEnd",this,b),this.stats&&(this.stats.get("Update Attributes").timeEnd(),h&&this.stats.get("Attributes updated").incrementCount()),this.attributeTransitionManager.update({attributes:this.attributes,numInstances:b,transitions:d})}updateTransition(){let{attributeTransitionManager:a}=this,b=a.run();return this.needsRedraw=this.needsRedraw||b,b}getAttributes(){return{...this.attributes,...this.attributeTransitionManager.getAttributes()}}getBounds(a){let b=a.map(a=>this.attributes[a]?.getBounds());return this.mergeBoundsMemoized(b)}getChangedAttributes(a={clearChangedFlags:!1}){let{attributes:b,attributeTransitionManager:c}=this,d={...c.getAttributes()};for(let e in b){let f=b[e];f.needsRedraw(a)&&!c.hasAttribute(e)&&(d[e]=f)}return d}getBufferLayouts(a){return Object.values(this.getAttributes()).map(b=>b.getBufferLayout(a))}_add(a,b){for(let c in a){let d=a[c],e={...d,id:c,size:d.isIndexed&&1||d.size||1,...b};this.attributes[c]=new iP(this.device,e)}this._mapUpdateTriggersToAttributes()}_mapUpdateTriggersToAttributes(){let a={};for(let b in this.attributes)this.attributes[b].getUpdateTriggers().forEach(c=>{a[c]||(a[c]=[]),a[c].push(b)});this.updateTriggers=a}_invalidateTrigger(a,b){let{attributes:c,updateTriggers:d}=this,e=d[a];return e&&e.forEach(a=>{let d=c[a];d&&d.setNeedsUpdate(d.id,b)}),e}_updateAttribute(a){let{attribute:b,numInstances:c}=a;(eY("attribute.updateStart",b),b.constant)?b.setConstantValue(a.context,b.value):(b.allocate(c)&&eY("attribute.allocate",b,c),b.updateBuffer(a)&&(this.needsRedraw=!0,eY("attribute.updateEnd",b,c)))}}function j4(a,b,c,d,e){let f=b-a;return(c-b)*e+-f*d+f+b}function j5(a,b){if(Array.isArray(a)){let c=0;for(let d=0;d<a.length;d++){let e=a[d]-b[d];c+=e*e}return Math.sqrt(c)}return Math.abs(a-b)}let j6={interpolation:class extends gP{get value(){return this._value}_onUpdate(){let{time:a,settings:{fromValue:b,toValue:c,duration:d,easing:e}}=this,f=e(a/d);this._value=dC(b,c,f)}},spring:class extends gP{get value(){return this._currValue}_onUpdate(){let{fromValue:a,toValue:b,damping:c,stiffness:d}=this.settings,{_prevValue:e=a,_currValue:f=a}=this,g=function(a,b,c,d,e){if(Array.isArray(c)){let f=[];for(let g=0;g<c.length;g++)f[g]=j4(a[g],b[g],c[g],d,e);return f}return j4(a,b,c,d,e)}(e,f,b,c,d),h=j5(g,b),i=j5(g,f);h<1e-5&&i<1e-5&&(g=b,this.end()),this._prevValue=f,this._currValue=g}}};class j7{constructor(a){this.transitions=new Map,this.timeline=a}get active(){return this.transitions.size>0}add(a,b,c,d){let{transitions:e}=this;if(e.has(a)){let c=e.get(a),{value:d=c.settings.fromValue}=c;b=d,this.remove(a)}if(!(d=iO(d)))return;let f=j6[d.type];if(!f)return void cs.error(`unsupported transition type '${d.type}'`)();let g=new f(this.timeline);g.start({...d,fromValue:b,toValue:c}),e.set(a,g)}remove(a){let{transitions:b}=this;b.has(a)&&(b.get(a).cancel(),b.delete(a))}update(){let a={};for(let[b,c]of this.transitions)c.update(),a[b]=c.value,c.inProgress||this.remove(b);return a}clear(){for(let a of this.transitions.keys())this.remove(a)}}function j8({newProps:a,oldProps:b,ignoreProps:c={},propTypes:d={},triggerName:e="props"}){if(b===a)return!1;if("object"!=typeof a||null===a||"object"!=typeof b||null===b)return`${e} changed shallowly`;for(let f of Object.keys(a))if(!(f in c)){if(!(f in b))return`${e}.${f} added`;let c=j9(a[f],b[f],d[f]);if(c)return`${e}.${f} ${c}`}for(let f of Object.keys(b))if(!(f in c)){if(!(f in a))return`${e}.${f} dropped`;if(!Object.hasOwnProperty.call(a,f)){let c=j9(a[f],b[f],d[f]);if(c)return`${e}.${f} ${c}`}}return!1}function j9(a,b,c){let d=c&&c.equal;return d&&!d(a,b,c)||!d&&(d=a&&b&&a.equals)&&!d.call(a,b)?"changed deeply":d||b===a?null:"changed shallowly"}function ka(a,b,c){let d=a.updateTriggers[c];d=null==d?{}:d;let e=b.updateTriggers[c];return j8({oldProps:e=null==e?{}:e,newProps:d,triggerName:c})}function kb(a,b){if(!b)return a;let c={...a,...b};if("defines"in b&&(c.defines={...a.defines,...b.defines}),"modules"in b&&(c.modules=(a.modules||[]).concat(b.modules),b.modules.some(a=>"project64"===a.name))){let a=c.modules.findIndex(a=>"project32"===a.name);a>=0&&c.modules.splice(a,1)}if("inject"in b)if(a.inject){let d={...a.inject};for(let a in b.inject)d[a]=(d[a]||"")+b.inject[a];c.inject=d}else c.inject=b.inject;return c}var el=el,cq=cq;let kc=[0,0,0];function kd(a,b,c=!1){let d=b.projectPosition(a);if(c&&b instanceof gO){let[c,e,f=0]=a,g=b.getDistanceScales([c,e]);d[2]=f*g.unitsPerMeter[2]}return d}function ke(a,{viewport:b,modelMatrix:c,coordinateSystem:d,coordinateOrigin:e,offsetMode:f}){let[g,h,i=0]=a;switch(c&&([g,h,i]=cq.transformMat4([],[g,h,i,1],c)),d){case"default":return ke(a,{viewport:b,modelMatrix:c,coordinateSystem:b.isGeospatial?"lnglat":"cartesian",coordinateOrigin:e,offsetMode:f});case"lnglat":return kd([g,h,i],b,f);case"lnglat-offsets":return kd([g+e[0],h+e[1],i+(e[2]||0)],b,f);case"meter-offsets":return kd(ev(e,[g,h,i]),b,f);case"cartesian":return b.isGeospatial?[g+e[0],h+e[1],i+e[2]]:b.projectPosition([g,h,i]);default:throw Error(`Invalid coordinateSystem: ${d}`)}}let kf={minFilter:"linear",mipmapFilter:"linear",magFilter:"linear",addressModeU:"clamp-to-edge",addressModeV:"clamp-to-edge"},kg={},kh={boolean:{validate:(a,b)=>!0,equal:(a,b,c)=>!!a==!!b},number:{validate:(a,b)=>Number.isFinite(a)&&(!("max"in b)||a<=b.max)&&(!("min"in b)||a>=b.min)},color:{validate:(a,b)=>b.optional&&!a||kj(a)&&(3===a.length||4===a.length),equal:(a,b,c)=>gD(a,b,1)},accessor:{validate(a,b){let c=kk(a);return"function"===c||c===kk(b.value)},equal:(a,b,c)=>"function"==typeof b||gD(a,b,1)},array:{validate:(a,b)=>b.optional&&!a||kj(a),equal(a,b,c){let{compare:d}=c,e=Number.isInteger(d)?d:+!!d;return d?gD(a,b,e):a===b}},object:{equal(a,b,c){if(c.ignore)return!0;let{compare:d}=c,e=Number.isInteger(d)?d:+!!d;return d?gD(a,b,e):a===b}},function:{validate:(a,b)=>b.optional&&!a||"function"==typeof a,equal:(a,b,c)=>!c.compare&&!1!==c.ignore||a===b},data:{transform:(a,b,c)=>{if(!a)return a;let{dataTransform:d}=c.props;return d?d(a):"string"==typeof a.shape&&a.shape.endsWith("-table")&&Array.isArray(a.data)?a.data:a}},image:{transform:(a,b,c)=>{let d=c.context;return d&&d.device?function(a,b,c,d){if(c instanceof hG.Texture)return c;c.constructor&&"Object"!==c.constructor.name&&(c={data:c});let e=null;c.compressed&&(e={minFilter:"linear",mipmapFilter:c.data.length>1?"nearest":"linear"});let{width:f,height:g}=c.data,h=b.createTexture({...c,sampler:{...kf,...e,...d},mipLevels:b.getMipLevelCount(f,g)});return"webgl"===b.type?h.generateMipmapsWebGL():"webgpu"===b.type&&b.generateMipmapsWebGPU(h),kg[h.id]=a,h}(c.id,d.device,a,{...b.parameters,...c.props.textureParameters}):null},release:(a,b,c)=>{var d;d=c.id,a&&a instanceof hG.Texture&&kg[a.id]===d&&(a.delete(),delete kg[a.id])}}};function ki(a,b){return"type"in b?{name:a,...kh[b.type],...b}:"value"in b?{name:a,type:kk(b.value),...b}:{name:a,type:"object",value:b}}function kj(a){return Array.isArray(a)||ArrayBuffer.isView(a)}function kk(a){return kj(a)?"array":null===a?"null":typeof a}function kl(a,b){return Object.prototype.hasOwnProperty.call(a,b)}let km=0;class kn{constructor(...a){this.props=function(a,b){let c;for(let a=b.length-1;a>=0;a--){let d=b[a];"extensions"in d&&(c=d.extensions)}let d=Object.create(function a(b,c){var d,e;if(!(b instanceof ko.constructor))return{};let f="_mergedDefaultProps";if(c)for(let a of c){let b=a.constructor;b&&(f+=`:${b.extensionName||b.name}`)}let g=kl(d=b,e=f)&&d[e];return g||(b[f]=function(b,c){var d;let e;if(!b.prototype)return null;let f=a(Object.getPrototypeOf(b)),g=function(a){let b={},c={},d={};for(let[e,f]of Object.entries(a)){let a=f?.deprecatedFor;if(a)d[e]=Array.isArray(a)?a:[a];else{let a=function(a,b){switch(kk(b)){case"object":return ki(a,b);case"array":return ki(a,{type:"array",value:b,compare:!1});case"boolean":return ki(a,{type:"boolean",value:b});case"number":return ki(a,{type:"number",value:b});case"function":return ki(a,{type:"function",value:b,compare:!0});default:return{name:a,type:"unknown",value:b}}}(e,f);b[e]=a,c[e]=a.value}}return{propTypes:b,defaultProps:c,deprecatedProps:d}}(function(a,b){return kl(a,b)&&a[b]}(b,"defaultProps")||{}),h=Object.assign(Object.create(null),f,g.defaultProps),i=Object.assign(Object.create(null),f?.[eR],g.propTypes),j=Object.assign(Object.create(null),f?.[eS],g.deprecatedProps);for(let b of c){let c=a(b.constructor);c&&(Object.assign(h,c),Object.assign(i,c[eR]),Object.assign(j,c[eS]))}return Object.defineProperties(h,{id:{writable:!0,value:((e=(d=b).componentName)||cs.warn(`${d.name}.componentName not specified`)(),e||d.name)}}),function(a,b){let c={},d={};for(let a in b){let e=b[a],{name:f,value:g}=e;e.async&&(c[f]=g,d[f]=function(a){return{enumerable:!0,set(b){"string"==typeof b||b instanceof Promise||iJ(b)?this[eU][a]=b:this[eV][a]=b},get(){if(this[eV]){if(a in this[eV])return this[eV][a]||this[eT][a];if(a in this[eU]){let b=this[eQ]&&this[eQ].internalState;if(b&&b.hasAsyncProp(a))return b.getAsyncProp(a)||this[eT][a]}}return this[eT][a]}}}(f))}a[eT]=c,a[eU]={},Object.defineProperties(a,d)}(h,i),function(a,b){for(let c in b)Object.defineProperty(a,c,{enumerable:!1,set(a){let d=`${this.id}: ${c}`;for(let d of b[c])kl(this,d)||(this[d]=a);cs.deprecated(d,b[c].join("/"))()}})}(h,j),h[eR]=i,h[eS]=j,0!==c.length||kl(b,"_propTypes")||(b._propTypes=i),h}(b,c||[]))}(a.constructor,c));d[eQ]=a,d[eU]={},d[eV]={};for(let a=0;a<b.length;++a){let c=b[a];for(let a in c)d[a]=c[a]}return Object.freeze(d),d}(this,a),this.id=this.props.id,this.count=km++}clone(a){let{props:b}=this,c={};for(let a in b[eT])a in b[eV]?c[a]=b[eV][a]:a in b[eU]&&(c[a]=b[eU][a]);return new this.constructor({...b,...c,...a})}}kn.componentName="Component",kn.defaultProps={};let ko=kn,kp=Object.freeze({});class kq{constructor(a){this.component=a,this.asyncProps={},this.onAsyncPropUpdated=()=>{},this.oldProps=null,this.oldAsyncProps=null}finalize(){for(let a in this.asyncProps){let b=this.asyncProps[a];b&&b.type&&b.type.release&&b.type.release(b.resolvedValue,b.type,this.component)}this.asyncProps={},this.component=null,this.resetOldProps()}getOldProps(){return this.oldAsyncProps||this.oldProps||kp}resetOldProps(){this.oldAsyncProps=null,this.oldProps=this.component?this.component.props:null}hasAsyncProp(a){return a in this.asyncProps}getAsyncProp(a){let b=this.asyncProps[a];return b&&b.resolvedValue}isAsyncPropLoading(a){if(a){let b=this.asyncProps[a];return!!(b&&b.pendingLoadCount>0&&b.pendingLoadCount!==b.resolvedLoadCount)}for(let a in this.asyncProps)if(this.isAsyncPropLoading(a))return!0;return!1}reloadAsyncProp(a,b){this._watchPromise(a,Promise.resolve(b))}setAsyncProps(a){this.component=a[eQ]||this.component;let b=a[eV]||{},c=a[eU]||a,d=a[eT]||{};for(let a in b){let c=b[a];this._createAsyncPropData(a,d[a]),this._updateAsyncProp(a,c),b[a]=this.getAsyncProp(a)}for(let a in c){let b=c[a];this._createAsyncPropData(a,d[a]),this._updateAsyncProp(a,b)}}_fetch(a,b){return null}_onResolve(a,b){}_onError(a,b){}_updateAsyncProp(a,b){if(this._didAsyncInputValueChange(a,b)){if("string"==typeof b&&(b=this._fetch(a,b)),b instanceof Promise)return void this._watchPromise(a,b);if(iJ(b))return void this._resolveAsyncIterable(a,b);this._setPropValue(a,b)}}_freezeAsyncOldProps(){if(!this.oldAsyncProps&&this.oldProps)for(let a in this.oldAsyncProps=Object.create(this.oldProps),this.asyncProps)Object.defineProperty(this.oldAsyncProps,a,{enumerable:!0,value:this.oldProps[a]})}_didAsyncInputValueChange(a,b){let c=this.asyncProps[a];return b!==c.resolvedValue&&b!==c.lastValue&&(c.lastValue=b,!0)}_setPropValue(a,b){this._freezeAsyncOldProps();let c=this.asyncProps[a];c&&(b=this._postProcessValue(c,b),c.resolvedValue=b,c.pendingLoadCount++,c.resolvedLoadCount=c.pendingLoadCount)}_setAsyncPropValue(a,b,c){let d=this.asyncProps[a];d&&c>=d.resolvedLoadCount&&void 0!==b&&(this._freezeAsyncOldProps(),d.resolvedValue=b,d.resolvedLoadCount=c,this.onAsyncPropUpdated(a,b))}_watchPromise(a,b){let c=this.asyncProps[a];if(c){c.pendingLoadCount++;let d=c.pendingLoadCount;b.then(b=>{this.component&&(b=this._postProcessValue(c,b),this._setAsyncPropValue(a,b,d),this._onResolve(a,b))}).catch(b=>{this._onError(a,b)})}}async _resolveAsyncIterable(a,b){if("data"!==a)return void this._setPropValue(a,b);let c=this.asyncProps[a];if(!c)return;c.pendingLoadCount++;let d=c.pendingLoadCount,e=[],f=0;for await(let c of b){if(!this.component)return;let{dataTransform:b}=this.component.props;Object.defineProperty(e=b?b(c,e):e.concat(c),"__diff",{enumerable:!1,value:[{startRow:f,endRow:e.length}]}),f=e.length,this._setAsyncPropValue(a,e,d)}this._onResolve(a,e)}_postProcessValue(a,b){let c=a.type;return c&&this.component&&(c.release&&c.release(a.resolvedValue,c,this.component),c.transform)?c.transform(b,c,this.component):b}_createAsyncPropData(a,b){if(!this.asyncProps[a]){let c=this.component&&this.component.props[eR];this.asyncProps[a]={type:c&&c[a],lastValue:null,resolvedValue:b,pendingLoadCount:0,resolvedLoadCount:0}}}}class kr extends kq{constructor({attributeManager:a,layer:b}){super(b),this.attributeManager=a,this.needsRedraw=!0,this.needsUpdate=!0,this.subLayers=null,this.usesPickingColorCache=!1}get layer(){return this.component}_fetch(a,b){let c=this.layer,d=c?.props.fetch;return d?d(b,{propName:a,layer:c}):super._fetch(a,b)}_onResolve(a,b){let c=this.layer;if(c){let d=c.props.onDataLoad;"data"===a&&d&&d(b,{propName:a,layer:c})}}_onError(a,b){let c=this.layer;c&&c.raiseError(b,`loading ${a} of ${this.layer}`)}}let ks=Object.freeze([]),kt=dd(({oldViewport:a,viewport:b})=>a.equals(b)),ku=new Uint8ClampedArray(0),kv={data:{type:"data",value:ks,async:!0},dataComparator:{type:"function",value:null,optional:!0},_dataDiff:{type:"function",value:a=>a&&a.__diff,optional:!0},dataTransform:{type:"function",value:null,optional:!0},onDataLoad:{type:"function",value:null,optional:!0},onError:{type:"function",value:null,optional:!0},fetch:{type:"function",value:(a,{propName:b,layer:c,loaders:d,loadOptions:e,signal:f})=>{let{resourceManager:g}=c.context;e=e||c.getLoadOptions(),d=d||c.props.loaders,f&&(e={...e,core:{...e?.core,fetch:{...e?.core?.fetch,signal:f}}});let h=g.contains(a);return(h||e||(g.add({resourceId:a,data:gm(a,d),persistent:!1}),h=!0),h)?g.subscribe({resourceId:a,onChange:a=>c.internalState?.reloadAsyncProp(b,a),consumerId:c.id,requestId:b}):gm(a,d,e)}},updateTriggers:{},visible:!0,pickable:!1,opacity:{type:"number",min:0,max:1,value:1},operation:"draw",onHover:{type:"function",value:null,optional:!0},onClick:{type:"function",value:null,optional:!0},onDragStart:{type:"function",value:null,optional:!0},onDrag:{type:"function",value:null,optional:!0},onDragEnd:{type:"function",value:null,optional:!0},coordinateSystem:"default",coordinateOrigin:{type:"array",value:[0,0,0],compare:!0},modelMatrix:{type:"array",value:null,compare:!0,optional:!0},wrapLongitude:!1,positionFormat:"XYZ",colorFormat:"RGBA",parameters:{type:"object",value:{},optional:!0,compare:2},loadOptions:{type:"object",value:null,optional:!0,ignore:!0},transitions:null,extensions:[],loaders:{type:"array",value:[],optional:!0,ignore:!0},getPolygonOffset:{type:"function",value:({layerIndex:a})=>[0,-(100*a)]},highlightedObjectIndex:null,autoHighlight:!1,highlightColor:{type:"accessor",value:[0,0,128,128]}};class kw extends ko{constructor(){super(...arguments),this.internalState=null,this.lifecycle="Awaiting state",this.parent=null}static get componentName(){return Object.prototype.hasOwnProperty.call(this,"layerName")?this.layerName:""}get root(){let a=this;for(;a.parent;)a=a.parent;return a}toString(){let a=this.constructor.layerName||this.constructor.name;return`${a}({id: '${this.props.id}'})`}project(a){gT(this.internalState);let b=this.internalState.viewport||this.context.viewport,[c,d,e]=ey(ke(a,{viewport:b,modelMatrix:this.props.modelMatrix,coordinateOrigin:this.props.coordinateOrigin,coordinateSystem:this.props.coordinateSystem}),b.pixelProjectionMatrix);return 2===a.length?[c,d]:[c,d,e]}unproject(a){return gT(this.internalState),(this.internalState.viewport||this.context.viewport).unproject(a)}projectPosition(a,b){return gT(this.internalState),function(a,b){let{viewport:c,coordinateSystem:d,coordinateOrigin:e,modelMatrix:f,fromCoordinateSystem:g,fromCoordinateOrigin:h}=function(a){let{viewport:b,modelMatrix:c,coordinateOrigin:d}=a,{coordinateSystem:e,fromCoordinateSystem:f,fromCoordinateOrigin:g}=a;return"default"===e&&(e=b.isGeospatial?"lnglat":"cartesian"),void 0===f?f=e:"default"===f&&(f=b.isGeospatial?"lnglat":"cartesian"),void 0===g&&(g=d),{viewport:b,coordinateSystem:e,coordinateOrigin:d,modelMatrix:c,fromCoordinateSystem:f,fromCoordinateOrigin:g}}(b),{autoOffset:i=!0}=b,{geospatialOrigin:j=kc,shaderCoordinateOrigin:k=kc,offsetMode:l=!1}=i?dm(c,d,e):{},m=ke(a,{viewport:c,modelMatrix:f,coordinateSystem:g,coordinateOrigin:h,offsetMode:l});if(l){let a=c.projectPosition(j||k);el.sub(m,m,a)}return m}(a,{viewport:this.internalState.viewport||this.context.viewport,modelMatrix:this.props.modelMatrix,coordinateOrigin:this.props.coordinateOrigin,coordinateSystem:this.props.coordinateSystem,...b})}get isComposite(){return!1}get isDrawable(){return!0}setState(a){this.setChangeFlags({stateChanged:!0}),Object.assign(this.state,a),this.setNeedsRedraw()}setNeedsRedraw(){this.internalState&&(this.internalState.needsRedraw=!0)}setNeedsUpdate(){this.internalState&&(this.context.layerManager.setNeedsUpdate(String(this)),this.internalState.needsUpdate=!0)}get isLoaded(){return!!this.internalState&&!this.internalState.isAsyncPropLoading()}get wrapLongitude(){return this.props.wrapLongitude}isPickable(){return this.props.pickable&&this.props.visible}getModels(){let a=this.state;return a&&(a.models||a.model&&[a.model])||[]}setShaderModuleProps(...a){for(let b of this.getModels())b.shaderInputs.setProps(...a)}getAttributeManager(){return this.internalState&&this.internalState.attributeManager}getCurrentLayer(){return this.internalState&&this.internalState.layer}getLoadOptions(){return this.props.loadOptions}use64bitPositions(){let{coordinateSystem:a}=this.props;return"default"===a||"lnglat"===a||"cartesian"===a}onHover(a,b){return!!this.props.onHover&&(this.props.onHover(a,b)||!1)}onClick(a,b){return!!this.props.onClick&&(this.props.onClick(a,b)||!1)}nullPickingColor(){return[0,0,0]}encodePickingColor(a,b=[]){return b[0]=a+1&255,b[1]=a+1>>8&255,b[2]=a+1>>8>>8&255,b}decodePickingColor(a){gT(a instanceof Uint8Array);let[b,c,d]=a;return b+256*c+65536*d-1}getNumInstances(){if(Number.isFinite(this.props.numInstances))return this.props.numInstances;if(this.state&&void 0!==this.state.numInstances)return this.state.numInstances;var a,b,c=this.props.data;if(null===(a=c)||"object"!=typeof a)throw Error("count(): argument not an object");if("function"==typeof c.count)return c.count();if(Number.isFinite(c.size))return c.size;if(Number.isFinite(c.length))return c.length;if(null!==(b=c)&&"object"==typeof b&&b.constructor===Object)return Object.keys(c).length;throw Error("count(): argument not a container")}getStartIndices(){return this.props.startIndices?this.props.startIndices:this.state&&this.state.startIndices?this.state.startIndices:null}getBounds(){return this.getAttributeManager()?.getBounds(["positions","instancePositions"])}getShaders(a){for(let b of(a=kb(a,{disableWarnings:!0,modules:this.context.defaultShaderModules}),this.props.extensions))a=kb(a,b.getShaders.call(this,b));return a}shouldUpdateState(a){return a.changeFlags.propsOrDataChanged}updateState(a){let b=this.getAttributeManager(),{dataChanged:c}=a.changeFlags;if(c&&b)if(Array.isArray(c))for(let a of c)b.invalidateAll(a);else b.invalidateAll();if(b){let{props:c}=a,d=this.internalState.hasPickingBuffer,e=Number.isInteger(c.highlightedObjectIndex)||!!c.pickable||c.extensions.some(a=>a.getNeedsPickingBuffer.call(this,a));if(d!==e){this.internalState.hasPickingBuffer=e;let{pickingColors:a,instancePickingColors:c}=b.attributes,d=a||c;d&&(e&&d.constant&&(d.constant=!1,b.invalidate(d.id)),d.value||e||(d.constant=!0,d.value=[0,0,0]))}}}finalizeState(a){for(let a of this.getModels())a.destroy();let b=this.getAttributeManager();b&&b.finalize(),this.context&&this.context.resourceManager.unsubscribe({consumerId:this.id}),this.internalState&&(this.internalState.uniformTransitions.clear(),this.internalState.finalize())}draw(a){for(let b of this.getModels())b.draw(a.renderPass)}getPickingInfo({info:a,mode:b,sourceLayer:c}){let{index:d}=a;return d>=0&&Array.isArray(this.props.data)&&(a.object=this.props.data[d]),a}raiseError(a,b){b&&(a=Error(`${b}: ${a.message}`,{cause:a})),this.props.onError?.(a)||this.context?.onError?.(a,this)}getNeedsRedraw(a={clearRedrawFlags:!1}){return this._getNeedsRedraw(a)}needsUpdate(){return!!this.internalState&&(this.internalState.needsUpdate||this.hasUniformTransition()||this.shouldUpdateState(this._getUpdateParams()))}hasUniformTransition(){return this.internalState?.uniformTransitions.active||!1}activateViewport(a){if(!this.internalState)return;let b=this.internalState.viewport;this.internalState.viewport=a,b&&kt({oldViewport:b,viewport:a})||(this.setChangeFlags({viewportChanged:!0}),this.isComposite?this.needsUpdate()&&this.setNeedsUpdate():this._update())}invalidateAttribute(a="all"){let b=this.getAttributeManager();b&&("all"===a?b.invalidateAll():b.invalidate(a))}updateAttributes(a){let b=!1;for(let c in a)a[c].layoutChanged()&&(b=!0);for(let c of this.getModels())this._setModelAttributes(c,a,b)}_updateAttributes(){let a=this.getAttributeManager();if(!a)return;let b=this.props,c=this.getNumInstances(),d=this.getStartIndices();a.update({data:b.data,numInstances:c,startIndices:d,props:b,transitions:b.transitions,buffers:b.data.attributes,context:this});let e=a.getChangedAttributes({clearChangedFlags:!0});this.updateAttributes(e)}_updateAttributeTransition(){let a=this.getAttributeManager();a&&a.updateTransition()}_updateUniformTransition(){let{uniformTransitions:a}=this.internalState;if(a.active){let b=a.update(),c=Object.create(this.props);for(let a in b)Object.defineProperty(c,a,{value:b[a]});return c}return this.props}calculateInstancePickingColors(a,{numInstances:b}){if(a.constant)return;let c=Math.floor(ku.length/4);this.internalState.usesPickingColorCache=!0;let d=b>0&&0===ku[0];if(c<b||d){b>0xffffff&&cs.warn("Layer has too many data objects. Picking might not be able to distinguish all objects.")();let a=Math.floor((ku=gp.allocate(ku,b,{size:4,copy:!0,maxCount:Math.max(b,0xffffff)})).length/4),e=[0,0,0],f=d?0:c;for(let b=f;b<a;b++)this.encodePickingColor(b,e),ku[4*b+0]=e[0],ku[4*b+1]=e[1],ku[4*b+2]=e[2],ku[4*b+3]=0}a.value=ku.subarray(0,4*b)}_setModelAttributes(a,b,c=!1){if(!Object.keys(b).length)return;if(c){let c=this.getAttributeManager();a.setBufferLayout(c.getBufferLayouts(a)),b=c.getAttributes()}let d=a.userData?.excludeAttributes||{},e={},f={};for(let c in b){if(d[c])continue;let g=b[c].getValue();for(let d in g){let h=g[d];h instanceof hF.Buffer?b[c].settings.isIndexed?a.setIndexBuffer(h):e[d]=h:h&&(f[d]=h)}}a.setAttributes(e),a.setConstantAttributes(f)}disablePickingIndex(a){let b=this.props.data;if(!("attributes"in b))return void this._disablePickingIndex(a);let{pickingColors:c,instancePickingColors:d}=this.getAttributeManager().attributes,e=c||d,f=e&&b.attributes&&b.attributes[e.id];if(f&&f.value){let c=f.value,d=this.encodePickingColor(a);for(let a=0;a<b.length;a++){let b=e.getVertexOffset(a);c[b]===d[0]&&c[b+1]===d[1]&&c[b+2]===d[2]&&this._disablePickingIndex(a)}}else this._disablePickingIndex(a)}_disablePickingIndex(a){let{pickingColors:b,instancePickingColors:c}=this.getAttributeManager().attributes,d=b||c;if(!d)return;let e=d.getVertexOffset(a),f=d.getVertexOffset(a+1);d.buffer.write(new Uint8Array(f-e),e)}restorePickingColors(){let{pickingColors:a,instancePickingColors:b}=this.getAttributeManager().attributes,c=a||b;c&&(this.internalState.usesPickingColorCache&&c.value.buffer!==ku.buffer&&(c.value=ku.subarray(0,c.value.length)),c.updateSubBuffer({startOffset:0}))}_initialize(){gT(!this.internalState),eY("layer.initialize",this);let a=this._getAttributeManager();for(let b of(a&&a.addInstanced({instancePickingColors:{type:"uint8",size:4,noAlloc:!0,update:this.calculateInstancePickingColors}}),this.internalState=new kr({attributeManager:a,layer:this}),this._clearChangeFlags(),this.state={},Object.defineProperty(this.state,"attributeManager",{get:()=>(cs.deprecated("layer.state.attributeManager","layer.getAttributeManager()")(),a)}),this.internalState.uniformTransitions=new j7(this.context.timeline),this.internalState.onAsyncPropUpdated=this._onAsyncPropUpdated.bind(this),this.internalState.setAsyncProps(this.props),this.initializeState(this.context),this.props.extensions))b.initializeState.call(this,this.context,b);this.setChangeFlags({dataChanged:"init",propsChanged:"init",viewportChanged:!0,extensionsChanged:!0}),this._update()}_transferState(a){eY("layer.matched",this,this===a);let{state:b,internalState:c}=a;this!==a&&(this.internalState=c,this.state=b,this.internalState.setAsyncProps(this.props),this._diffProps(this.props,this.internalState.getOldProps()))}_update(){let a=this.needsUpdate();if(eY("layer.update",this,a),!a)return;this.context.stats.get("Layer updates").incrementCount();let b=this.props,c=this.context,d=this.internalState,e=c.viewport,f=this._updateUniformTransition();d.propsInTransition=f,c.viewport=d.viewport||e,this.props=f;try{let a=this._getUpdateParams(),b=this.getModels();if(c.device)this.updateState(a);else try{this.updateState(a)}catch(a){}for(let b of this.props.extensions)b.updateState.call(this,a,b);this.setNeedsRedraw(),this._updateAttributes();let d=this.getModels()[0]!==b[0];this._postUpdate(a,d)}finally{c.viewport=e,this.props=b,this._clearChangeFlags(),d.needsUpdate=!1,d.resetOldProps()}}_finalize(){for(let a of(eY("layer.finalize",this),this.finalizeState(this.context),this.props.extensions))a.finalizeState.call(this,this.context,a)}_drawLayer({renderPass:a,shaderModuleProps:b=null,uniforms:c={},parameters:d={}}){this._updateAttributeTransition();let e=this.props,f=this.context;this.props=this.internalState.propsInTransition||e;try{b&&this.setShaderModuleProps(b);let{getPolygonOffset:e}=this.props,g=e&&e(c)||[0,0];f.device instanceof iy.WebGLDevice&&f.device.setParametersWebGL({polygonOffset:g});let h=f.device instanceof iy.WebGLDevice?null:function(a){let{blendConstant:b,...c}=a;return b?{pipelineParameters:c,renderPassParameters:{blendConstant:b}}:{pipelineParameters:c}}(d);if(function(a,b,c,d){for(let e of a)"webgpu"===e.device.type?(function(a,b){let c=b.props.framebuffer||(b.framebuffer??null);if(!c)return;let d=c.colorAttachments.map(a=>a?.texture?.format??null),e=c.depthStencilAttachment?.texture?.format;(!function(a,b){if(a===b)return!0;if(!a||!b||a.length!==b.length)return!1;for(let c=0;c<a.length;c++)if(a[c]!==b[c])return!1;return!0}(a.props.colorAttachmentFormats,d)||a.props.depthStencilAttachmentFormat!==e)&&(a.props.colorAttachmentFormats=d,a.props.depthStencilAttachmentFormat=e,a._setPipelineNeedsUpdate("attachment formats"))}(e,b),e.setParameters({...e.parameters,...d?.pipelineParameters})):e.setParameters(c)}(this.getModels(),a,d,h),f.device instanceof iy.WebGLDevice)f.device.withParametersWebGL(d,()=>{let e={renderPass:a,shaderModuleProps:b,uniforms:c,parameters:d,context:f};for(let a of this.props.extensions)a.draw.call(this,e,a);this.draw(e)});else{h?.renderPassParameters&&a.setParameters(h.renderPassParameters);let e={renderPass:a,shaderModuleProps:b,uniforms:c,parameters:d,context:f};for(let a of this.props.extensions)a.draw.call(this,e,a);this.draw(e)}}finally{this.props=e}}getChangeFlags(){return this.internalState?.changeFlags}setChangeFlags(a){if(!this.internalState)return;let{changeFlags:b}=this.internalState;for(let c in a)if(a[c]){let d=!1;if("dataChanged"===c){let e=a[c],f=b[c];e&&Array.isArray(f)&&(b.dataChanged=Array.isArray(e)?f.concat(e):e,d=!0)}b[c]||(b[c]=a[c],d=!0),d&&eY("layer.changeFlag",this,c,a)}let c=!!(b.dataChanged||b.updateTriggersChanged||b.propsChanged||b.extensionsChanged);b.propsOrDataChanged=c,b.somethingChanged=c||b.viewportChanged||b.stateChanged}_clearChangeFlags(){this.internalState.changeFlags={dataChanged:!1,propsChanged:!1,updateTriggersChanged:!1,viewportChanged:!1,stateChanged:!1,extensionsChanged:!1,propsOrDataChanged:!1,somethingChanged:!1}}_diffProps(a,b){let c,d,e,f=(c=j8({newProps:a,oldProps:b,propTypes:a[eR],ignoreProps:{data:null,updateTriggers:null,extensions:null,transitions:null}}),d=function(a,b){if(null===b)return"oldProps is null, initial diff";let c=!1,{dataComparator:d,_dataDiff:e}=a;return d?d(a.data,b.data)||(c="Data comparator detected a change"):a.data!==b.data&&(c="A new data container was supplied"),c&&e&&(c=e(a.data,b.data)||c),c}(a,b),e=!1,d||(e=function(a,b){if(null===b||"all"in a.updateTriggers&&ka(a,b,"all"))return{all:!0};let c={},d=!1;for(let e in a.updateTriggers)"all"!==e&&ka(a,b,e)&&(c[e]=!0,d=!0);return!!d&&c}(a,b)),{dataChanged:d,propsChanged:c,updateTriggersChanged:e,extensionsChanged:function(a,b){if(null===b)return!0;let c=b.extensions,{extensions:d}=a;if(d===c)return!1;if(!c||!d||d.length!==c.length)return!0;for(let a=0;a<d.length;a++)if(!d[a].equals(c[a]))return!0;return!1}(a,b),transitionsChanged:function(a,b){if(!a.transitions)return!1;let c={},d=a[eR],e=!1;for(let f in a.transitions){let g=d[f],h=g&&g.type;("number"===h||"color"===h||"array"===h)&&j9(a[f],b[f],g)&&(c[f]=!0,e=!0)}return!!e&&c}(a,b)});if(f.updateTriggersChanged)for(let a in f.updateTriggersChanged)f.updateTriggersChanged[a]&&this.invalidateAttribute(a);if(f.transitionsChanged)for(let c in f.transitionsChanged)this.internalState.uniformTransitions.add(c,b[c],a[c],a.transitions?.[c]);return this.setChangeFlags(f)}validateProps(){!function(a){let b=a[eR];for(let c in b){let d=b[c],{validate:e}=d;if(e&&!e(a[c],d))throw Error(`Invalid prop ${c}: ${a[c]}`)}}(this.props)}updateAutoHighlight(a){this.props.autoHighlight&&!Number.isInteger(this.props.highlightedObjectIndex)&&this._updateAutoHighlight(a)}_updateAutoHighlight(a){let b={highlightedObjectColor:a.picked?a.color:null},{highlightColor:c}=this.props;a.picked&&"function"==typeof c&&(b.highlightColor=c(a)),this.setShaderModuleProps({picking:b}),this.setNeedsRedraw()}_getAttributeManager(){let a=this.context;return new j3(a.device,{id:this.props.id,stats:a.stats,timeline:a.timeline})}_postUpdate(a,b){let{props:c,oldProps:d}=a,e=this.state.model;e?.isInstanced&&e.setInstanceCount(this.getNumInstances());let{autoHighlight:f,highlightedObjectIndex:g,highlightColor:h}=c;if(b||d.autoHighlight!==f||d.highlightedObjectIndex!==g||d.highlightColor!==h){let a={};Array.isArray(h)&&(a.highlightColor=h),(b||d.autoHighlight!==f||g!==d.highlightedObjectIndex)&&(a.highlightedObjectColor=Number.isFinite(g)&&g>=0?this.encodePickingColor(g):null),this.setShaderModuleProps({picking:a})}}_getUpdateParams(){return{props:this.props,oldProps:this.internalState.getOldProps(),context:this.context,changeFlags:this.internalState.changeFlags}}_getNeedsRedraw(a){if(!this.internalState)return!1;let b=!1;b=this.internalState.needsRedraw&&this.id;let c=this.getAttributeManager(),d=!!c&&c.getNeedsRedraw(a);if(b=b||d)for(let a of this.props.extensions)a.onNeedsRedraw.call(this,a);return this.internalState.needsRedraw=this.internalState.needsRedraw&&!a.clearRedrawFlags,b}_onAsyncPropUpdated(){this._diffProps(this.props,this.internalState.getOldProps()),this.setNeedsUpdate()}}kw.defaultProps=kv,kw.layerName="Layer";let kx=kw,ky={position:"absolute",zIndex:-1};function kz(a){return s.isValidElement(a)}let kA=(0,s.createContext)(),kB={mixBlendMode:null};function kC(a){a.redrawReason&&(a.deck._drawLayers(a.redrawReason),a.redrawReason=null)}let kD=s.forwardRef(function(a,b){let[c,d]=(0,s.useState)(0),e=(0,s.useRef)({control:null,version:c,forceUpdate:()=>d(a=>a+1)}).current,f=(0,s.useRef)(null),g=(0,s.useRef)(null),h=(0,s.useMemo)(()=>(function({children:a,layers:b=[],views:c}){let d=[],e=[],f={};return s.Children.forEach(function a(b){if("function"==typeof b)return(0,s.createElement)(gL,{},b);if(Array.isArray(b))return b.map(a);if(kz(b)){if(b.type===s.Fragment)return a(b.props.children);ix(b.type,gL)}return b}(a),a=>{if(kz(a)){let b=a.type;if(ix(b,kx)){let c=function(a,b){let c={},d=a.defaultProps||{};for(let a in b)d[a]!==b[a]&&(c[a]=b[a]);return new a(c)}(b,a.props);e.push(c)}else d.push(a);if(ix(b,gL)&&b!==gL&&a.props.id){let c=new b(a.props);f[c.id]=c}}else a&&d.push(a)}),Object.keys(f).length>0&&(Array.isArray(c)?c.forEach(a=>{f[a.id]=a}):c&&(f[c.id]=c),c=Object.values(f)),{layers:b=e.length>0?[e,b]:b,children:d,views:c}})(a),[a.layers,a.views,a.children]),i=!0,j=b=>i&&a.viewState?(e.viewStateUpdateRequested=b,null):(e.viewStateUpdateRequested=null,a.onViewStateChange?.(b)),k=b=>{i?e.interactionStateUpdateRequested=b:(e.interactionStateUpdateRequested=null,a.onInteractionStateChange?.(b))},l=(0,s.useMemo)(()=>{let b={widgets:[],...a,style:null,width:"100%",height:"100%",parent:f.current,canvas:g.current,layers:h.layers,onViewStateChange:j,onInteractionStateChange:k};return h.views&&(b.views=h.views),delete b._customRender,e.deck&&(e.deck.setProps(b),e.deck.isInitialized&&(e.lastRenderedViewports=e.deck.getViewports())),b},[a]);(0,s.useEffect)(()=>{var b;let c;return c=new(a.Deck||iv)({...b={...l,parent:f.current,canvas:g.current},_customRender:b.deviceProps?.adapters?.[0]?.type==="webgpu"?void 0:a=>{e.redrawReason=a;let b=c.getViewports();e.lastRenderedViewports!==b?e.forceUpdate():kC(e)}}),e.deck=c,()=>e.deck?.finalize()},[]),iw(()=>{kC(e);let{viewStateUpdateRequested:a,interactionStateUpdateRequested:b}=e;a&&j(a),b&&k(b),e.deck?.isInitialized&&e.deck.redraw("Initial render")}),(0,s.useImperativeHandle)(b,()=>({get deck(){return e.deck},pickObjectAsync:a=>e.deck.pickObjectAsync(a),pickObjectsAsync:a=>e.deck.pickObjectsAsync(a),pickObject:a=>e.deck.pickObject(a),pickMultipleObjects:a=>e.deck.pickMultipleObjects(a),pickObjects:a=>e.deck.pickObjects(a)}),[]);let m=e.deck&&e.deck.isInitialized?e.deck.getViewports():void 0,{ContextProvider:n,width:o="100%",height:p="100%",id:q,style:r}=a,{containerStyle:t,canvasStyle:u}=(0,s.useMemo)(()=>(function({width:a,height:b,style:c}){let d={position:"absolute",zIndex:0,left:0,top:0,width:a,height:b},e={left:0,top:0};if(c)for(let a in c)a in kB?e[a]=c[a]:d[a]=c[a];return{containerStyle:d,canvasStyle:e}})({width:o,height:p,style:r}),[o,p,r]);if(!e.viewStateUpdateRequested&&e.lastRenderedViewports===m||e.version!==c){e.lastRenderedViewports=m,e.version=c;let a=function({children:a,deck:b,ContextProvider:c=kA.Provider}){let{viewManager:d}=b||{};if(!d||!d.views.length)return[];let e={},f=d.views[0].id;for(let b of a){let a=f,c=b;kz(b)&&ix(b.type,gL)&&(a=b.props.id||f,c=b.props.children);let g=d.getViewport(a),h=d.getViewState(a);if(g){h.padding=g.padding;let{x:b,y:d,width:f,height:i}=g;c=function a(b,c){if("function"==typeof b)return b(c);if(Array.isArray(b))return b.map(b=>a(b,c));if(kz(b)){var d;if(d=b,d.props?.mapStyle)return c.style=ky,(0,s.cloneElement)(b,c);if(function(a){let b=a.type;return b&&b.deckGLViewProps}(b))return(0,s.cloneElement)(b,c)}return b}(c,{x:b,y:d,width:f,height:i,viewport:g,viewState:h}),e[a]||(e[a]={viewport:g,children:[]}),e[a].children.push(c)}}return Object.keys(e).map(a=>{let{viewport:d,children:f}=e[a],{x:g,y:h,width:i,height:j}=d,k=`view-${a}`,l=(0,s.createElement)("div",{key:k,id:k,style:{position:"absolute",left:g,top:h,width:i,height:j}},...f),m={deck:b,viewport:d,container:b.canvas.offsetParent,eventManager:b.eventManager,onViewStateChange:c=>{c.viewId=a,b._onViewStateChange(c)},widgets:[]},n=`view-${a}-context`;return(0,s.createElement)(c,{key:n,value:m},l)})}({children:h.children,deck:e.deck,ContextProvider:n}),b=(0,s.createElement)("canvas",{key:"canvas",id:q||"deckgl-overlay",ref:g,style:u}),d=(0,s.createElement)("div",{key:"deck-events-root",className:"deck-events-root",style:{width:o,height:p}},[b,a]),i=(0,s.createElement)("div",{key:"deck-widgets-root",className:"deck-widgets-root"});e.control=(0,s.createElement)("div",{id:`${q||"deckgl"}-wrapper`,ref:f,style:t},[d,i])}return i=!1,e.control});class kE{id;topology;vertexCount;indices;attributes;userData={};constructor(a){const{attributes:b={},indices:c=null,vertexCount:d=null}=a;for(const[d,e]of(this.id=a.id||jf("geometry"),this.topology=a.topology,c&&(this.indices=ArrayBuffer.isView(c)?{value:c,size:1}:c),this.attributes={},Object.entries(b))){const a=ArrayBuffer.isView(e)?{value:e}:e;if(!ArrayBuffer.isView(a.value))throw Error(`${this._print(d)}: must be typed array or object with value as typed array`);if("POSITION"!==d&&"positions"!==d||a.size||(a.size=3),"indices"===d){if(this.indices)throw Error("Multiple indices detected");this.indices=a}else this.attributes[d]=a}this.indices&&void 0!==this.indices.isIndexed&&(this.indices=Object.assign({},this.indices),delete this.indices.isIndexed),this.vertexCount=d||this._calculateVertexCount(this.attributes,this.indices)}getVertexCount(){return this.vertexCount}getAttributes(){return this.indices?{indices:this.indices,...this.attributes}:this.attributes}_print(a){return`Geometry ${this.id} attribute ${a}`}_setAttributes(a,b){return this}_calculateVertexCount(a,b){if(b)return b.value.length;let c=1/0;for(let b of Object.values(a)){let{value:a,size:d,constant:e}=b;!e&&a&&void 0!==d&&d>=1&&(c=Math.min(c,a.length/d))}return c}}let kF=`\
layout(std140) uniform scatterplotUniforms {
  float radiusScale;
  float radiusMinPixels;
  float radiusMaxPixels;
  float lineWidthScale;
  float lineWidthMinPixels;
  float lineWidthMaxPixels;
  float stroked;
  float filled;
  bool antialiasing;
  bool billboard;
  highp int radiusUnits;
  highp int lineWidthUnits;
} scatterplot;
`,kG={name:"scatterplot",vs:kF,fs:kF,source:"",uniformTypes:{radiusScale:"f32",radiusMinPixels:"f32",radiusMaxPixels:"f32",lineWidthScale:"f32",lineWidthMinPixels:"f32",lineWidthMaxPixels:"f32",stroked:"f32",filled:"f32",antialiasing:"f32",billboard:"f32",radiusUnits:"i32",lineWidthUnits:"i32"}},kH=`\
#version 300 es
#define SHADER_NAME scatterplot-layer-vertex-shader
in vec3 positions;
in vec3 instancePositions;
in vec3 instancePositions64Low;
in float instanceRadius;
in float instanceLineWidths;
in vec4 instanceFillColors;
in vec4 instanceLineColors;
in vec3 instancePickingColors;
in vec2 instancePixelOffset;
out vec4 vFillColor;
out vec4 vLineColor;
out vec2 unitPosition;
out float innerUnitRadius;
out float outerRadiusPixels;
void main(void) {
geometry.worldPosition = instancePositions;
outerRadiusPixels = clamp(
project_size_to_pixel(scatterplot.radiusScale * instanceRadius, scatterplot.radiusUnits),
scatterplot.radiusMinPixels, scatterplot.radiusMaxPixels
);
float lineWidthPixels = clamp(
project_size_to_pixel(scatterplot.lineWidthScale * instanceLineWidths, scatterplot.lineWidthUnits),
scatterplot.lineWidthMinPixels, scatterplot.lineWidthMaxPixels
);
outerRadiusPixels += scatterplot.stroked * lineWidthPixels / 2.0;
float edgePadding = scatterplot.antialiasing ? (outerRadiusPixels + SMOOTH_EDGE_RADIUS) / outerRadiusPixels : 1.0;
unitPosition = edgePadding * positions.xy;
geometry.uv = unitPosition;
geometry.pickingColor = instancePickingColors;
innerUnitRadius = 1.0 - scatterplot.stroked * lineWidthPixels / outerRadiusPixels;
if (scatterplot.billboard) {
gl_Position = project_position_to_clipspace(instancePositions, instancePositions64Low, vec3(0.0), geometry.position);
DECKGL_FILTER_GL_POSITION(gl_Position, geometry);
vec3 offset = edgePadding * positions * outerRadiusPixels;
offset.xy += instancePixelOffset;
DECKGL_FILTER_SIZE(offset, geometry);
gl_Position.xy += project_pixel_size_to_clipspace(offset.xy);
} else {
vec3 offset = edgePadding * positions * project_pixel_size(outerRadiusPixels);
offset.xy += project_pixel_size(instancePixelOffset);
DECKGL_FILTER_SIZE(offset, geometry);
gl_Position = project_position_to_clipspace(instancePositions, instancePositions64Low, offset, geometry.position);
DECKGL_FILTER_GL_POSITION(gl_Position, geometry);
}
vFillColor = vec4(instanceFillColors.rgb, instanceFillColors.a * layer.opacity);
DECKGL_FILTER_COLOR(vFillColor, geometry);
vLineColor = vec4(instanceLineColors.rgb, instanceLineColors.a * layer.opacity);
DECKGL_FILTER_COLOR(vLineColor, geometry);
}
`,kI=`\
#version 300 es
#define SHADER_NAME scatterplot-layer-fragment-shader
precision highp float;
in vec4 vFillColor;
in vec4 vLineColor;
in vec2 unitPosition;
in float innerUnitRadius;
in float outerRadiusPixels;
out vec4 fragColor;
void main(void) {
geometry.uv = unitPosition;
float distToCenter = length(unitPosition) * outerRadiusPixels;
float inCircle = scatterplot.antialiasing ?
smoothedge(distToCenter, outerRadiusPixels) :
step(distToCenter, outerRadiusPixels);
if (inCircle == 0.0) {
discard;
}
if (scatterplot.stroked > 0.5) {
float isLine = scatterplot.antialiasing ?
smoothedge(innerUnitRadius * outerRadiusPixels, distToCenter) :
step(innerUnitRadius * outerRadiusPixels, distToCenter);
if (scatterplot.filled > 0.5) {
fragColor = mix(vFillColor, vLineColor, isLine);
} else {
if (isLine == 0.0) {
discard;
}
fragColor = vec4(vLineColor.rgb, vLineColor.a * isLine);
}
} else if (scatterplot.filled < 0.5) {
discard;
} else {
fragColor = vFillColor;
}
fragColor.a *= inCircle;
DECKGL_FILTER_COLOR(fragColor, geometry);
}
`,kJ=`\
// Main shaders

struct ScatterplotUniforms {
  radiusScale: f32,
  radiusMinPixels: f32,
  radiusMaxPixels: f32,
  lineWidthScale: f32,
  lineWidthMinPixels: f32,
  lineWidthMaxPixels: f32,
  stroked: f32,
  filled: i32,
  antialiasing: i32,
  billboard: i32,
  radiusUnits: i32,
  lineWidthUnits: i32,
};

struct ConstantAttributeUniforms {
 instancePositions: vec3<f32>,
 instancePositions64Low: vec3<f32>,
 instanceRadius: f32,
 instanceLineWidths: f32,
 instanceFillColors: vec4<f32>,
 instanceLineColors: vec4<f32>,
 instancePickingColors: vec3<f32>,
 instancePixelOffset: vec2<f32>,

 instancePositionsConstant: i32,
 instancePositions64LowConstant: i32,
 instanceRadiusConstant: i32,
 instanceLineWidthsConstant: i32,
 instanceFillColorsConstant: i32,
 instanceLineColorsConstant: i32,
 instancePickingColorsConstant: i32,
 instancePixelOffsetConstant: i32
};

@group(0) @binding(0) var<uniform> scatterplot: ScatterplotUniforms;

struct ConstantAttributes {
  instancePositions: vec3<f32>,
  instancePositions64Low: vec3<f32>,
  instanceRadius: f32,
  instanceLineWidths: f32,
  instanceFillColors: vec4<f32>,
  instanceLineColors: vec4<f32>,
  instancePickingColors: vec3<f32>,
  instancePixelOffset: vec2<f32>
};

const constants = ConstantAttributes(
  vec3<f32>(0.0),
  vec3<f32>(0.0),
  0.0,
  0.0,
  vec4<f32>(0.0, 0.0, 0.0, 1.0),
  vec4<f32>(0.0, 0.0, 0.0, 1.0),
  vec3<f32>(0.0),
  vec2<f32>(0.0)
);

struct Attributes {
  @builtin(instance_index) instanceIndex : u32,
  @builtin(vertex_index) vertexIndex : u32,
  @location(0) positions: vec3<f32>,
  @location(1) instancePositions: vec3<f32>,
  @location(2) instancePositions64Low: vec3<f32>,
  @location(3) instanceRadius: f32,
  @location(4) instanceLineWidths: f32,
  @location(5) instanceFillColors: vec4<f32>,
  @location(6) instanceLineColors: vec4<f32>,
  @location(7) instancePickingColors: vec3<f32>,
  @location(8) instancePixelOffset: vec2<f32>
};

struct Varyings {
  @builtin(position) position: vec4<f32>,
  @location(0) vFillColor: vec4<f32>,
  @location(1) vLineColor: vec4<f32>,
  @location(2) unitPosition: vec2<f32>,
  @location(3) innerUnitRadius: f32,
  @location(4) outerRadiusPixels: f32,
  @location(5) pickingColor: vec3<f32>,
};

@vertex
fn vertexMain(attributes: Attributes) -> Varyings {
  var varyings: Varyings;

  // Draw an inline geometry constant array clip space triangle to verify that rendering works.
  // var positions = array<vec2<f32>, 3>(vec2(0.0, 0.5), vec2(-0.5, -0.5), vec2(0.5, -0.5));
  // if (attributes.instanceIndex == 0) {
  //   varyings.position = vec4<f32>(positions[attributes.vertexIndex], 0.0, 1.0);
  //   return varyings;
  // }

  geometry.worldPosition = attributes.instancePositions;

  // Multiply out radius and clamp to limits
  varyings.outerRadiusPixels = clamp(
    project_unit_size_to_pixel(scatterplot.radiusScale * attributes.instanceRadius, scatterplot.radiusUnits),
    scatterplot.radiusMinPixels, scatterplot.radiusMaxPixels
  );

  // Multiply out line width and clamp to limits
  let lineWidthPixels = clamp(
    project_unit_size_to_pixel(scatterplot.lineWidthScale * attributes.instanceLineWidths, scatterplot.lineWidthUnits),
    scatterplot.lineWidthMinPixels, scatterplot.lineWidthMaxPixels
  );

  // outer radius needs to offset by half stroke width
  varyings.outerRadiusPixels += scatterplot.stroked * lineWidthPixels / 2.0;
  // Expand geometry to accommodate edge smoothing
  let edgePadding = select(
    (varyings.outerRadiusPixels + SMOOTH_EDGE_RADIUS) / varyings.outerRadiusPixels,
    1.0,
    scatterplot.antialiasing != 0
  );

  // position on the containing square in [-1, 1] space
  varyings.unitPosition = edgePadding * attributes.positions.xy;
  geometry.uv = varyings.unitPosition;
  geometry.pickingColor = attributes.instancePickingColors;

  varyings.innerUnitRadius = 1.0 - scatterplot.stroked * lineWidthPixels / varyings.outerRadiusPixels;

  if (scatterplot.billboard != 0) {
    varyings.position = project_position_to_clipspace(attributes.instancePositions, attributes.instancePositions64Low, vec3<f32>(0.0)); // TODO , geometry.position);
    // DECKGL_FILTER_GL_POSITION(varyings.position, geometry);
    var offset = edgePadding * attributes.positions * varyings.outerRadiusPixels;
    offset = vec3<f32>(offset.xy + attributes.instancePixelOffset, offset.z);
    // DECKGL_FILTER_SIZE(offset, geometry);
    let clipPixels = project_pixel_size_to_clipspace(offset.xy);
    varyings.position = vec4<f32>(varyings.position.x + clipPixels.x, varyings.position.y + clipPixels.y, varyings.position.z, varyings.position.w);
  } else {
    var offset = edgePadding * attributes.positions * project_pixel_size_float(varyings.outerRadiusPixels);
    offset = vec3<f32>(offset.xy + project_pixel_size_vec2(attributes.instancePixelOffset), offset.z);
    // DECKGL_FILTER_SIZE(offset, geometry);
    varyings.position = project_position_to_clipspace(attributes.instancePositions, attributes.instancePositions64Low, offset); // TODO , geometry.position);
    // DECKGL_FILTER_GL_POSITION(varyings.position, geometry);
  }

  // Apply opacity to instance color, or return instance picking color
  varyings.vFillColor = vec4<f32>(attributes.instanceFillColors.rgb, attributes.instanceFillColors.a * layer.opacity);
  // DECKGL_FILTER_COLOR(varyings.vFillColor, geometry);
  varyings.vLineColor = vec4<f32>(attributes.instanceLineColors.rgb, attributes.instanceLineColors.a * layer.opacity);
  // DECKGL_FILTER_COLOR(varyings.vLineColor, geometry);
  varyings.pickingColor = attributes.instancePickingColors;

  return varyings;
}

@fragment
fn fragmentMain(varyings: Varyings) -> @location(0) vec4<f32> {
  // var geometry: Geometry;
  // geometry.uv = unitPosition;

  let distToCenter = length(varyings.unitPosition) * varyings.outerRadiusPixels;
  let inCircle = select(
    smoothedge(distToCenter, varyings.outerRadiusPixels),
    step(distToCenter, varyings.outerRadiusPixels),
    scatterplot.antialiasing != 0
  );

  if (inCircle == 0.0) {
    discard;
  }

  var fragColor: vec4<f32>;

  if (scatterplot.stroked != 0) {
    let isLine = select(
      smoothedge(varyings.innerUnitRadius * varyings.outerRadiusPixels, distToCenter),
      step(varyings.innerUnitRadius * varyings.outerRadiusPixels, distToCenter),
      scatterplot.antialiasing != 0
    );

    if (scatterplot.filled != 0) {
      fragColor = mix(varyings.vFillColor, varyings.vLineColor, isLine);
    } else {
      if (isLine == 0.0) {
        discard;
      }
      fragColor = vec4<f32>(varyings.vLineColor.rgb, varyings.vLineColor.a * isLine);
    }
  } else if (scatterplot.filled == 0) {
    discard;
  } else {
    fragColor = varyings.vFillColor;
  }

  fragColor.a *= inCircle;

  if (picking.isActive > 0.5) {
    if (!picking_isColorValid(varyings.pickingColor)) {
      discard;
    }
    return vec4<f32>(varyings.pickingColor, 1.0);
  }

  if (picking.isHighlightActive > 0.5) {
    let highlightedObjectColor = picking_normalizeColor(picking.highlightedObjectColor);
    if (picking_isColorZero(abs(varyings.pickingColor - highlightedObjectColor))) {
      let highLightAlpha = picking.highlightColor.a;
      let blendedAlpha = highLightAlpha + fragColor.a * (1.0 - highLightAlpha);
      if (blendedAlpha > 0.0) {
        let highLightRatio = highLightAlpha / blendedAlpha;
        fragColor = vec4<f32>(
          mix(fragColor.rgb, picking.highlightColor.rgb, highLightRatio),
          blendedAlpha
        );
      } else {
        fragColor = vec4<f32>(fragColor.rgb, 0.0);
      }
    }
  }

  // Apply premultiplied alpha as required by transparent canvas
  fragColor = deckgl_premultiplied_alpha(fragColor);

  return fragColor;
  // return vec4<f32>(0, 0, 1, 1);
}
`,kK=[0,0,0,255],kL={radiusUnits:"meters",radiusScale:{type:"number",min:0,value:1},radiusMinPixels:{type:"number",min:0,value:0},radiusMaxPixels:{type:"number",min:0,value:Number.MAX_SAFE_INTEGER},lineWidthUnits:"meters",lineWidthScale:{type:"number",min:0,value:1},lineWidthMinPixels:{type:"number",min:0,value:0},lineWidthMaxPixels:{type:"number",min:0,value:Number.MAX_SAFE_INTEGER},stroked:!1,filled:!0,billboard:!1,antialiasing:!0,getPosition:{type:"accessor",value:a=>a.position},getRadius:{type:"accessor",value:1},getFillColor:{type:"accessor",value:kK},getLineColor:{type:"accessor",value:kK},getLineWidth:{type:"accessor",value:1},getPixelOffset:{type:"accessor",value:[0,0]},strokeWidth:{deprecatedFor:"getLineWidth"},outline:{deprecatedFor:"stroked"},getColor:{deprecatedFor:["getFillColor","getLineColor"]}};class kM extends kx{getShaders(){return super.getShaders({vs:kH,fs:kI,source:kJ,modules:[dy,bR,eM,kG]})}initializeState(){this.getAttributeManager().addInstanced({instancePositions:{size:3,type:"float64",fp64:this.use64bitPositions(),transition:!0,accessor:"getPosition"},instanceRadius:{size:1,transition:!0,accessor:"getRadius",defaultValue:1},instanceFillColors:{size:this.props.colorFormat.length,transition:!0,type:"unorm8",accessor:"getFillColor",defaultValue:[0,0,0,255]},instanceLineColors:{size:this.props.colorFormat.length,transition:!0,type:"unorm8",accessor:"getLineColor",defaultValue:[0,0,0,255]},instanceLineWidths:{size:1,transition:!0,accessor:"getLineWidth",defaultValue:1},instancePixelOffset:{size:2,transition:!0,accessor:"getPixelOffset"}})}updateState(a){super.updateState(a),a.changeFlags.extensionsChanged&&(this.state.model?.destroy(),this.state.model=this._getModel(),this.getAttributeManager().invalidateAll())}draw({uniforms:a}){let{radiusUnits:b,radiusScale:c,radiusMinPixels:d,radiusMaxPixels:e,stroked:f,filled:g,billboard:h,antialiasing:i,lineWidthUnits:j,lineWidthScale:k,lineWidthMinPixels:l,lineWidthMaxPixels:m}=this.props,n={stroked:f,filled:g,billboard:h,antialiasing:i,radiusUnits:da[b],radiusScale:c,radiusMinPixels:d,radiusMaxPixels:e,lineWidthUnits:da[j],lineWidthScale:k,lineWidthMinPixels:l,lineWidthMaxPixels:m},o=this.state.model;o.shaderInputs.setProps({scatterplot:n}),o.draw(this.context.renderPass)}_getModel(){return new jJ(this.context.device,{...this.getShaders(),id:this.props.id,bufferLayout:this.getAttributeManager().getBufferLayouts(),geometry:new kE({topology:"triangle-strip",attributes:{positions:{size:3,value:new Float32Array([-1,-1,0,1,-1,0,-1,1,0,1,1,0])}}}),isInstanced:!0})}}kM.defaultProps=kL,kM.layerName="ScatterplotLayer";class kN{constructor(a){this.indexStarts=[0],this.vertexStarts=[0],this.vertexCount=0,this.instanceCount=0;const{attributes:b={}}=a;this.typedArrayManager=gp,this.attributes={},this._attributeDefs=b,this.opts=a,this.updateGeometry(a)}updateGeometry(a){Object.assign(this.opts,a);let{data:b,buffers:c={},getGeometry:d,geometryBuffer:e,positionFormat:f,dataChanged:g,normalize:h=!0}=this.opts;if(this.data=b,this.getGeometry=d,this.positionSize=e&&e.size||("XY"===f?2:3),this.buffers=c,this.normalize=h,e&&(gT(b.startIndices),this.getGeometry=this.getGeometryFromBuffer(e),h||(c.vertexPositions=e)),this.geometryBuffer=c.vertexPositions,Array.isArray(g))for(let a of g)this._rebuildGeometry(a);else this._rebuildGeometry()}updatePartialGeometry({startRow:a,endRow:b}){this._rebuildGeometry({startRow:a,endRow:b})}getGeometryFromBuffer(a){let b=a.value||a;return ArrayBuffer.isView(b)?iK(b,{size:this.positionSize,offset:a.offset,stride:a.stride,startIndices:this.data.startIndices}):null}_allocate(a,b){let{attributes:c,buffers:d,_attributeDefs:e,typedArrayManager:f}=this;for(let g in e)if(g in d)f.release(c[g]),c[g]=null;else{let d=e[g];d.copy=b,c[g]=f.allocate(c[g],a,d)}}_forEachGeometry(a,b,c){let{data:d,getGeometry:e}=this,{iterable:f,objectInfo:g}=iI(d,b,c);for(let b of f)g.index++,a(e?e(b,g):null,g.index)}_rebuildGeometry(a){if(!this.data)return;let{indexStarts:b,vertexStarts:c,instanceCount:d}=this,{data:e,geometryBuffer:f}=this,{startRow:g=0,endRow:h=1/0}=a||{},i={};if(a||(b=[0],c=[0]),this.normalize||!f)this._forEachGeometry((a,b)=>{let d=a&&this.normalizeGeometry(a);i[b]=d,c[b+1]=c[b]+(d?this.getGeometrySize(d):0)},g,h),d=c[c.length-1];else if(d=(c=e.startIndices)[e.length]||0,ArrayBuffer.isView(f))d=d||f.length/this.positionSize;else if(f instanceof hF.Buffer){let a=4*this.positionSize;d=d||f.byteLength/a}else if(f.buffer){let a=f.stride||4*this.positionSize;d=d||f.buffer.byteLength/a}else if(f.value){let a=f.value,b=f.stride/a.BYTES_PER_ELEMENT||this.positionSize;d=d||a.length/b}this._allocate(d,!!a),this.indexStarts=b,this.vertexStarts=c,this.instanceCount=d;let j={};this._forEachGeometry((a,e)=>{let f=i[e]||a;j.vertexStart=c[e],j.indexStart=b[e],j.geometrySize=(e<c.length-1?c[e+1]:d)-c[e],j.geometryIndex=e,this.updateGeometryAttributes(f,j)},g,h),this.vertexCount=b[b.length-1]}}function kO(a,b,c,d,e=[]){let f,g;if(8&c)f=(d[3]-a[1])/(b[1]-a[1]),g=3;else if(4&c)f=(d[1]-a[1])/(b[1]-a[1]),g=1;else if(2&c)f=(d[2]-a[0])/(b[0]-a[0]),g=2;else{if(!(1&c))return null;f=(d[0]-a[0])/(b[0]-a[0]),g=0}for(let c=0;c<a.length;c++)e[c]=(1&g)===c?d[g]:f*(b[c]-a[c])+a[c];return e}function kP(a,b){let c=0;return a[0]<b[0]?c|=1:a[0]>b[2]&&(c|=2),a[1]<b[1]?c|=4:a[1]>b[3]&&(c|=8),c}function kQ(a,b){let c=b.length,d=a.length;if(d>0){let e=!0;for(let f=0;f<c;f++)if(a[d-c+f]!==b[f]){e=!1;break}if(e)return!1}for(let e=0;e<c;e++)a[d+e]=b[e];return!0}function kR(a,b){let c=b.length;for(let d=0;d<c;d++)a[d]=b[d]}function kS(a,b,c,d,e=[]){let f=d+b*c;for(let b=0;b<c;b++)e[b]=a[f+b];return e}function kT(a,b){var c,d,e,f,g,h,i;let j,k,l,m,{size:n=2,broken:o=!1,gridResolution:p=10,gridOffset:q=[0,0],startIndex:r=0,endIndex:s=a.length}=b||{},t=(s-r)/n,u=[],v=[u],w=kS(a,0,n,r),x=(c=w,d=p,e=q,f=[],j=Math.floor((c[0]-e[0])/d)*d+e[0],k=Math.floor((c[1]-e[1])/d)*d+e[1],f[0]=j,f[1]=k,f[2]=j+d,f[3]=k+d,f),y=[];kQ(u,w);for(let b=1;b<t;b++){for(m=kP(l=kS(a,b,n,r,l),x);m;){kO(w,l,m,x,y);let a=kP(y,x);a&&(kO(w,y,a,x,y),m=a),kQ(u,y),kR(w,y),g=x,h=p,8&(i=m)?(g[1]+=h,g[3]+=h):4&i?(g[1]-=h,g[3]-=h):2&i?(g[0]+=h,g[2]+=h):1&i&&(g[0]-=h,g[2]-=h),o&&u.length>n&&(u=[],v.push(u),kQ(u,w)),m=kP(l,x)}kQ(u,l),kR(w,l)}return o?v:v[0]}class kU extends kN{constructor(a){super({...a,attributes:{positions:{size:3,padding:18,initialize:!0,type:a.fp64?Float64Array:Float32Array},segmentTypes:{size:1,type:Uint8ClampedArray}}})}get(a){return this.attributes[a]}getGeometryFromBuffer(a){return this.normalize?super.getGeometryFromBuffer(a):null}normalizeGeometry(a){return this.normalize?function(a,b,c,d){let e;if(Array.isArray(a[0])){e=Array(a.length*b);for(let c=0;c<a.length;c++)for(let d=0;d<b;d++)e[c*b+d]=a[c][d]||0}else e=a;return c?kT(e,{size:b,gridResolution:c}):d?function(a,b){let{size:c=2,startIndex:d=0,endIndex:e=a.length,normalize:f=!0}=b||{},g=a.slice(d,e);!function(a,b,c){let d,e=a[0];for(let f=0;f<c;f+=b){let b=(d=a[f])-e;(b>180||b<-180)&&(d-=360*Math.round(b/360)),a[f]=e=d}}(g,c,e-d);let h=kT(g,{size:c,broken:!0,gridResolution:360,gridOffset:[-180,-180]});if(f)for(let a of h)!function(a,b){let c,d=a.length/b;for(let e=0;e<d&&((c=a[e*b])+180)%360==0;e++);let e=-(360*Math.round(c/360));if(0!==e)for(let c=0;c<d;c++)a[c*b]+=e}(a,c);return h}(e,{size:b}):e}(a,this.positionSize,this.opts.resolution,this.opts.wrapLongitude):a}getGeometrySize(a){if(kV(a)){let b=0;for(let c of a)b+=this.getGeometrySize(c);return b}let b=this.getPathLength(a);return b<2?0:this.isClosed(a)?b<3?0:b+2:b}updateGeometryAttributes(a,b){if(0!==b.geometrySize)if(a&&kV(a))for(let c of a){let a=this.getGeometrySize(c);b.geometrySize=a,this.updateGeometryAttributes(c,b),b.vertexStart+=a}else this._updateSegmentTypes(a,b),this._updatePositions(a,b)}_updateSegmentTypes(a,b){let c=this.attributes.segmentTypes,d=!!a&&this.isClosed(a),{vertexStart:e,geometrySize:f}=b;c.fill(0,e,e+f),d?(c[e]=4,c[e+f-2]=4):(c[e]+=1,c[e+f-2]+=2),c[e+f-1]=4}_updatePositions(a,b){let{positions:c}=this.attributes;if(!c||!a)return;let{vertexStart:d,geometrySize:e}=b,f=[,,,];for(let b=d,g=0;g<e;b++,g++)this.getPointOnPath(a,g,f),c[3*b]=f[0],c[3*b+1]=f[1],c[3*b+2]=f[2]}getPathLength(a){return a.length/this.positionSize}getPointOnPath(a,b,c=[]){let{positionSize:d}=this;b*d>=a.length&&(b+=1-a.length/d);let e=b*d;return c[0]=a[e],c[1]=a[e+1],c[2]=3===d&&a[e+2]||0,c}isClosed(a){if(!this.normalize)return!!this.opts.loop;let{positionSize:b}=this,c=a.length-b;return a[0]===a[c]&&a[1]===a[c+1]&&(2===b||a[2]===a[c+2])}}function kV(a){return Array.isArray(a[0])}let kW=`\
layout(std140) uniform pathUniforms {
  float widthScale;
  float widthMinPixels;
  float widthMaxPixels;
  float jointType;
  float capType;
  float miterLimit;
  bool billboard;
  highp int widthUnits;
} path;
`,kX={name:"path",vs:kW,fs:kW,uniformTypes:{widthScale:"f32",widthMinPixels:"f32",widthMaxPixels:"f32",jointType:"f32",capType:"f32",miterLimit:"f32",billboard:"f32",widthUnits:"i32"}},kY=`\
#version 300 es
#define SHADER_NAME path-layer-vertex-shader
in vec2 positions;
in float instanceTypes;
in vec3 instanceStartPositions;
in vec3 instanceEndPositions;
in vec3 instanceLeftPositions;
in vec3 instanceRightPositions;
in vec3 instanceLeftPositions64Low;
in vec3 instanceStartPositions64Low;
in vec3 instanceEndPositions64Low;
in vec3 instanceRightPositions64Low;
in float instanceStrokeWidths;
in vec4 instanceColors;
in vec3 instancePickingColors;
uniform float opacity;
out vec4 vColor;
out vec2 vCornerOffset;
out float vMiterLength;
out vec2 vPathPosition;
out float vPathLength;
out float vJointType;
const float EPSILON = 0.001;
const vec3 ZERO_OFFSET = vec3(0.0);
float flipIfTrue(bool flag) {
return -(float(flag) * 2. - 1.);
}
vec3 getLineJoinOffset(
vec3 prevPoint, vec3 currPoint, vec3 nextPoint,
vec2 width
) {
bool isEnd = positions.x > 0.0;
float sideOfPath = positions.y;
float isJoint = float(sideOfPath == 0.0);
vec3 deltaA3 = (currPoint - prevPoint);
vec3 deltaB3 = (nextPoint - currPoint);
mat3 rotationMatrix;
bool needsRotation = !path.billboard && project_needs_rotation(currPoint, rotationMatrix);
if (needsRotation) {
deltaA3 = deltaA3 * rotationMatrix;
deltaB3 = deltaB3 * rotationMatrix;
}
vec2 deltaA = deltaA3.xy / width;
vec2 deltaB = deltaB3.xy / width;
float lenA = length(deltaA);
float lenB = length(deltaB);
vec2 dirA = lenA > 0. ? normalize(deltaA) : vec2(0.0, 0.0);
vec2 dirB = lenB > 0. ? normalize(deltaB) : vec2(0.0, 0.0);
vec2 perpA = vec2(-dirA.y, dirA.x);
vec2 perpB = vec2(-dirB.y, dirB.x);
vec2 tangent = dirA + dirB;
tangent = length(tangent) > 0. ? normalize(tangent) : perpA;
vec2 miterVec = vec2(-tangent.y, tangent.x);
vec2 dir = isEnd ? dirA : dirB;
vec2 perp = isEnd ? perpA : perpB;
float L = isEnd ? lenA : lenB;
float sinHalfA = abs(dot(miterVec, perp));
float cosHalfA = abs(dot(dirA, miterVec));
float turnDirection = flipIfTrue(dirA.x * dirB.y >= dirA.y * dirB.x);
float cornerPosition = sideOfPath * turnDirection;
float miterSize = 1.0 / max(sinHalfA, EPSILON);
miterSize = mix(
min(miterSize, max(lenA, lenB) / max(cosHalfA, EPSILON)),
miterSize,
step(0.0, cornerPosition)
);
vec2 offsetVec = mix(miterVec * miterSize, perp, step(0.5, cornerPosition))
* (sideOfPath + isJoint * turnDirection);
bool isStartCap = lenA == 0.0 || (!isEnd && (instanceTypes == 1.0 || instanceTypes == 3.0));
bool isEndCap = lenB == 0.0 || (isEnd && (instanceTypes == 2.0 || instanceTypes == 3.0));
bool isCap = isStartCap || isEndCap;
if (isCap) {
offsetVec = mix(perp * sideOfPath, dir * path.capType * 4.0 * flipIfTrue(isStartCap), isJoint);
vJointType = path.capType;
} else {
vJointType = path.jointType;
}
vPathLength = L;
vCornerOffset = offsetVec;
vMiterLength = dot(vCornerOffset, miterVec * turnDirection);
vMiterLength = isCap ? isJoint : vMiterLength;
vec2 offsetFromStartOfPath = vCornerOffset + deltaA * float(isEnd);
vPathPosition = vec2(
dot(offsetFromStartOfPath, perp),
dot(offsetFromStartOfPath, dir)
);
geometry.uv = vPathPosition;
float isValid = step(instanceTypes, 3.5);
vec3 offset = vec3(offsetVec * width * isValid, 0.0);
if (needsRotation) {
offset = rotationMatrix * offset;
}
return offset;
}
void clipLine(inout vec4 position, vec4 refPosition) {
if (position.w < EPSILON) {
float r = (EPSILON - refPosition.w) / (position.w - refPosition.w);
position = refPosition + (position - refPosition) * r;
}
}
void main() {
geometry.pickingColor = instancePickingColors;
vColor = vec4(instanceColors.rgb, instanceColors.a * layer.opacity);
float isEnd = positions.x;
vec3 prevPosition = mix(instanceLeftPositions, instanceStartPositions, isEnd);
vec3 prevPosition64Low = mix(instanceLeftPositions64Low, instanceStartPositions64Low, isEnd);
vec3 currPosition = mix(instanceStartPositions, instanceEndPositions, isEnd);
vec3 currPosition64Low = mix(instanceStartPositions64Low, instanceEndPositions64Low, isEnd);
vec3 nextPosition = mix(instanceEndPositions, instanceRightPositions, isEnd);
vec3 nextPosition64Low = mix(instanceEndPositions64Low, instanceRightPositions64Low, isEnd);
geometry.worldPosition = currPosition;
vec2 widthPixels = vec2(clamp(
project_size_to_pixel(instanceStrokeWidths * path.widthScale, path.widthUnits),
path.widthMinPixels, path.widthMaxPixels) / 2.0);
vec3 width;
if (path.billboard) {
vec4 prevPositionScreen = project_position_to_clipspace(prevPosition, prevPosition64Low, ZERO_OFFSET);
vec4 currPositionScreen = project_position_to_clipspace(currPosition, currPosition64Low, ZERO_OFFSET, geometry.position);
vec4 nextPositionScreen = project_position_to_clipspace(nextPosition, nextPosition64Low, ZERO_OFFSET);
clipLine(prevPositionScreen, currPositionScreen);
clipLine(nextPositionScreen, currPositionScreen);
clipLine(currPositionScreen, mix(nextPositionScreen, prevPositionScreen, isEnd));
width = vec3(widthPixels, 0.0);
DECKGL_FILTER_SIZE(width, geometry);
vec3 offset = getLineJoinOffset(
prevPositionScreen.xyz / prevPositionScreen.w,
currPositionScreen.xyz / currPositionScreen.w,
nextPositionScreen.xyz / nextPositionScreen.w,
project_pixel_size_to_clipspace(width.xy)
);
DECKGL_FILTER_GL_POSITION(currPositionScreen, geometry);
gl_Position = vec4(currPositionScreen.xyz + offset * currPositionScreen.w, currPositionScreen.w);
} else {
prevPosition = project_position(prevPosition, prevPosition64Low);
currPosition = project_position(currPosition, currPosition64Low);
nextPosition = project_position(nextPosition, nextPosition64Low);
width = vec3(project_pixel_size(widthPixels), 0.0);
DECKGL_FILTER_SIZE(width, geometry);
vec3 offset = getLineJoinOffset(prevPosition, currPosition, nextPosition, width.xy);
geometry.position = vec4(currPosition + offset, 1.0);
gl_Position = project_common_position_to_clipspace(geometry.position);
DECKGL_FILTER_GL_POSITION(gl_Position, geometry);
}
DECKGL_FILTER_COLOR(vColor, geometry);
}
`,kZ=`\
#version 300 es
#define SHADER_NAME path-layer-fragment-shader
precision highp float;
in vec4 vColor;
in vec2 vCornerOffset;
in float vMiterLength;
in vec2 vPathPosition;
in float vPathLength;
in float vJointType;
out vec4 fragColor;
void main(void) {
geometry.uv = vPathPosition;
if (vPathPosition.y < 0.0 || vPathPosition.y > vPathLength) {
if (vJointType > 0.5 && length(vCornerOffset) > 1.0) {
discard;
}
if (vJointType < 0.5 && vMiterLength > path.miterLimit + 1.0) {
discard;
}
}
fragColor = vColor;
DECKGL_FILTER_COLOR(fragColor, geometry);
}
`,k$=[0,0,0,255],k_={widthUnits:"meters",widthScale:{type:"number",min:0,value:1},widthMinPixels:{type:"number",min:0,value:0},widthMaxPixels:{type:"number",min:0,value:Number.MAX_SAFE_INTEGER},jointRounded:!1,capRounded:!1,miterLimit:{type:"number",min:0,value:4},billboard:!1,_pathType:null,getPath:{type:"accessor",value:a=>a.path},getColor:{type:"accessor",value:k$},getWidth:{type:"accessor",value:1},rounded:{deprecatedFor:["jointRounded","capRounded"]}},k0={enter:(a,b)=>b.length?b.subarray(b.length-a.length):a};class k1 extends kx{getShaders(){return super.getShaders({vs:kY,fs:kZ,modules:[dy,eM,kX]})}get wrapLongitude(){return!1}getBounds(){return this.getAttributeManager()?.getBounds(["vertexPositions"])}initializeState(){this.getAttributeManager().addInstanced({vertexPositions:{size:3,vertexOffset:1,type:"float64",fp64:this.use64bitPositions(),transition:k0,accessor:"getPath",update:this.calculatePositions,noAlloc:!0,shaderAttributes:{instanceLeftPositions:{vertexOffset:0},instanceStartPositions:{vertexOffset:1},instanceEndPositions:{vertexOffset:2},instanceRightPositions:{vertexOffset:3}}},instanceTypes:{size:1,type:"uint8",update:this.calculateSegmentTypes,noAlloc:!0},instanceStrokeWidths:{size:1,accessor:"getWidth",transition:k0,defaultValue:1},instanceColors:{size:this.props.colorFormat.length,type:"unorm8",accessor:"getColor",transition:k0,defaultValue:k$},instancePickingColors:{size:4,type:"uint8",accessor:(a,{index:b,target:c})=>this.encodePickingColor(a&&a.__source?a.__source.index:b,c)}}),this.setState({pathTesselator:new kU({fp64:this.use64bitPositions()})})}updateState(a){super.updateState(a);let{props:b,changeFlags:c}=a,d=this.getAttributeManager();if(c.dataChanged||c.updateTriggersChanged&&(c.updateTriggersChanged.all||c.updateTriggersChanged.getPath)){let{pathTesselator:a}=this.state,e=b.data.attributes||{};a.updateGeometry({data:b.data,geometryBuffer:e.getPath,buffers:e,normalize:!b._pathType,loop:"loop"===b._pathType,getGeometry:b.getPath,positionFormat:b.positionFormat,wrapLongitude:b.wrapLongitude,resolution:this.context.viewport.resolution,dataChanged:c.dataChanged}),this.setState({numInstances:a.instanceCount,startIndices:a.vertexStarts}),c.dataChanged||d.invalidateAll()}c.extensionsChanged&&(this.state.model?.destroy(),this.state.model=this._getModel(),d.invalidateAll())}getPickingInfo(a){let b=super.getPickingInfo(a),{index:c}=b,d=this.props.data;return d[0]&&d[0].__source&&(b.object=d.find(a=>a.__source.index===c)),b}disablePickingIndex(a){let b=this.props.data;if(b[0]&&b[0].__source)for(let c=0;c<b.length;c++)b[c].__source.index===a&&this._disablePickingIndex(c);else super.disablePickingIndex(a)}draw({uniforms:a}){let{jointRounded:b,capRounded:c,billboard:d,miterLimit:e,widthUnits:f,widthScale:g,widthMinPixels:h,widthMaxPixels:i}=this.props,j=this.state.model,k={jointType:Number(b),capType:Number(c),billboard:d,widthUnits:da[f],widthScale:g,miterLimit:e,widthMinPixels:h,widthMaxPixels:i};j.shaderInputs.setProps({path:k}),j.draw(this.context.renderPass)}_getModel(){return new jJ(this.context.device,{...this.getShaders(),id:this.props.id,bufferLayout:this.getAttributeManager().getBufferLayouts(),geometry:new kE({topology:"triangle-list",attributes:{indices:new Uint16Array([0,1,2,1,4,2,1,3,4,3,5,4]),positions:{value:new Float32Array([0,0,0,-1,0,1,1,-1,1,1,1,0]),size:2}}}),isInstanced:!0})}calculatePositions(a){let{pathTesselator:b}=this.state;a.startIndices=b.vertexStarts,a.value=b.get("positions")}calculateSegmentTypes(a){let{pathTesselator:b}=this.state;a.startIndices=b.vertexStarts,a.value=b.get("segmentTypes")}}k1.defaultProps=k_,k1.layerName="PathLayer";let k2=new Float32Array(12);function k3(a,b=2){let c=0;for(let d of a)for(let a=0;a<b;a++)k2[c++]=d[a]||0;return k2}class k4{device;model;sampler;currentIndex=0;samplerTextureMap=null;bindings=[];resources={};constructor(a,b){this.device=a,this.sampler=a.createSampler({addressModeU:"clamp-to-edge",addressModeV:"clamp-to-edge",minFilter:"nearest",magFilter:"nearest",mipmapFilter:"nearest"}),this.model=new jJ(this.device,{id:b.id||jf("texture-transform-model"),fs:b.fs||iS({input:b.targetTextureVarying,inputChannels:b.targetTextureChannels,output:"transform_output"}),vertexCount:b.vertexCount,...b}),this._initialize(b),Object.seal(this)}destroy(){for(let a of(this.model.destroy(),this.bindings))a.framebuffer?.destroy()}delete(){this.destroy()}run(a){let{framebuffer:b}=this.bindings[this.currentIndex],c=this.device.beginRenderPass({framebuffer:b,...a});this.model.draw(c),c.end(),this.device.submit()}getTargetTexture(){let{targetTexture:a}=this.bindings[this.currentIndex];return a}getFramebuffer(){return this.bindings[this.currentIndex].framebuffer}_initialize(a){this._updateBindings(a)}_updateBindings(a){this.bindings[this.currentIndex]=this._updateBinding(this.bindings[this.currentIndex],a)}_updateBinding(a,{sourceBuffers:b,sourceTextures:c,targetTexture:d}){if(a||(a={sourceBuffers:{},sourceTextures:{},targetTexture:null}),Object.assign(a.sourceTextures,c),Object.assign(a.sourceBuffers,b),d){a.targetTexture=d;let{width:b,height:c}=d;a.framebuffer&&a.framebuffer.destroy(),a.framebuffer=this.device.createFramebuffer({id:"transform-framebuffer",width:b,height:c,colorAttachments:[d]}),a.framebuffer.resize({width:b,height:c})}return a}_setSourceTextureParameters(){let a=this.currentIndex,{sourceTextures:b}=this.bindings[a];for(let a in b)b[a].sampler=this.sampler}}let k5=`\
#version 300 es
#define SHADER_NAME heatp-map-layer-vertex-shader
uniform sampler2D maxTexture;
in vec3 positions;
in vec2 texCoords;
out vec2 vTexCoords;
out float vIntensityMin;
out float vIntensityMax;
void main(void) {
gl_Position = project_position_to_clipspace(positions, vec3(0.0), vec3(0.0));
vTexCoords = texCoords;
vec4 maxTexture = texture(maxTexture, vec2(0.5));
float maxValue = triangle.aggregationMode < 0.5 ? maxTexture.r : maxTexture.g;
float minValue = maxValue * triangle.threshold;
if (triangle.colorDomain[1] > 0.) {
maxValue = triangle.colorDomain[1];
minValue = triangle.colorDomain[0];
}
vIntensityMax = triangle.intensity / maxValue;
vIntensityMin = triangle.intensity / minValue;
}
`,k6=`\
#version 300 es
#define SHADER_NAME triangle-layer-fragment-shader
precision highp float;
uniform sampler2D weightsTexture;
uniform sampler2D colorTexture;
in vec2 vTexCoords;
in float vIntensityMin;
in float vIntensityMax;
out vec4 fragColor;
vec4 getLinearColor(float value) {
float factor = clamp(value * vIntensityMax, 0., 1.);
vec4 color = texture(colorTexture, vec2(factor, 0.5));
color.a *= min(value * vIntensityMin, 1.0);
return color;
}
void main(void) {
vec4 weights = texture(weightsTexture, vTexCoords);
float weight = weights.r;
if (triangle.aggregationMode > 0.5) {
weight /= max(1.0, weights.a);
}
if (weight <= 0.) {
discard;
}
vec4 linearColor = getLinearColor(weight);
linearColor.a *= layer.opacity;
fragColor = linearColor;
}
`,k7=`\
layout(std140) uniform triangleUniforms {
  float aggregationMode;
  vec2 colorDomain;
  float intensity;
  float threshold;
} triangle;
`,k8={name:"triangle",vs:k7,fs:k7,uniformTypes:{aggregationMode:"f32",colorDomain:"vec2<f32>",intensity:"f32",threshold:"f32"}};class k9 extends kx{getShaders(){return super.getShaders({vs:k5,fs:k6,modules:[dy,k8]})}initializeState({device:a}){this.setState({model:this._getModel(a)})}_getModel(a){let{vertexCount:b,data:c}=this.props;return new jJ(a,{...this.getShaders(),id:this.props.id,attributes:c.attributes,bufferLayout:[{name:"positions",format:"float32x3"},{name:"texCoords",format:"float32x2"}],topology:"triangle-strip",vertexCount:b})}draw(){let{model:a}=this.state,{aggregationMode:b,colorDomain:c,intensity:d,threshold:e,colorTexture:f,maxTexture:g,weightsTexture:h}=this.props;a.shaderInputs.setProps({triangle:{aggregationMode:b,colorDomain:c,intensity:d,threshold:e,colorTexture:f,maxTexture:g,weightsTexture:h}}),a.draw(this.context.renderPass)}}k9.layerName="TriangleLayer";class la extends kx{get isComposite(){return!0}get isDrawable(){return!1}get isLoaded(){return super.isLoaded&&this.getSubLayers().every(a=>a.isLoaded)}getSubLayers(){return this.internalState&&this.internalState.subLayers||[]}initializeState(a){}setState(a){super.setState(a),this.setNeedsUpdate()}getPickingInfo({info:a}){let{object:b}=a;return b&&b.__source&&b.__source.parent&&b.__source.parent.id===this.id&&(a.object=b.__source.object,a.index=b.__source.index),a}filterSubLayer(a){return!0}shouldRenderSubLayer(a,b){return b&&b.length}getSubLayerClass(a,b){let{_subLayerProps:c}=this.props;return c&&c[a]&&c[a].type||b}getSubLayerRow(a,b,c){return a.__source={parent:this,object:b,index:c},a}getSubLayerAccessor(a){if("function"==typeof a){let b={index:-1,data:this.props.data,target:[]};return(c,d)=>c&&c.__source?(b.index=c.__source.index,a(c.__source.object,b)):a(c,d)}return a}getSubLayerProps(a={}){let{opacity:b,pickable:c,visible:d,parameters:e,getPolygonOffset:f,highlightedObjectIndex:g,autoHighlight:h,highlightColor:i,coordinateSystem:j,coordinateOrigin:k,wrapLongitude:l,positionFormat:m,modelMatrix:n,extensions:o,fetch:p,operation:q,_subLayerProps:r}=this.props,s={id:"",updateTriggers:{},opacity:b,pickable:c,visible:d,parameters:e,getPolygonOffset:f,highlightedObjectIndex:g,autoHighlight:h,highlightColor:i,coordinateSystem:j,coordinateOrigin:k,wrapLongitude:l,positionFormat:m,modelMatrix:n,extensions:o,fetch:p,operation:q},t=r&&a.id&&r[a.id],u=t&&t.updateTriggers,v=a.id||"sublayer";if(t){let b=this.props[eR],c=a.type?a.type._propTypes:{};for(let a in t){let d=c[a]||b[a];d&&"accessor"===d.type&&(t[a]=this.getSubLayerAccessor(t[a]))}}for(let b of(Object.assign(s,a,t),s.id=`${this.props.id}-${v}`,s.updateTriggers={all:this.props.updateTriggers?.all,...a.updateTriggers,...u},o)){let a=b.getSubLayerProps.call(this,b);a&&Object.assign(s,a,{updateTriggers:Object.assign(s.updateTriggers,a.updateTriggers)})}return s}_updateAutoHighlight(a){for(let b of this.getSubLayers())b.updateAutoHighlight(a)}_getAttributeManager(){return null}_postUpdate(a,b){let c=this.internalState.subLayers,d=!c||this.needsUpdate();for(let a of(d&&(c=eZ(this.renderLayers(),Boolean),this.internalState.subLayers=c),eY("compositeLayer.renderLayers",this,d,c),c))a.parent=this}}la.layerName="CompositeLayer";let lb=la;class lc extends lb{initializeAggregationLayer(a){super.initializeState(this.context),this.setState({ignoreProps:function(a,b){let c={};for(let d in a)b.includes(d)||(c[d]=a[d]);return c}(this.constructor._propTypes,a.data.props),dimensions:a})}updateState(a){super.updateState(a);let{changeFlags:b}=a;if(b.extensionsChanged){let a=this.getShaders({});a&&a.defines&&(a.defines.NON_INSTANCED_MODEL=1),this.updateShaders(a)}this._updateAttributes()}updateAttributes(a){this.setState({changedAttributes:a})}getAttributes(){return this.getAttributeManager().getAttributes()}getModuleSettings(){let{viewport:a,mousePosition:b,device:c}=this.context;return Object.assign(Object.create(this.props),{viewport:a,mousePosition:b,picking:{isActive:0},devicePixelRatio:c.canvasContext.cssToDeviceRatio()})}updateShaders(a){}isAggregationDirty(a,b={}){let{props:c,oldProps:d,changeFlags:e}=a,{compareAll:f=!1,dimension:g}=b,{ignoreProps:h}=this.state,{props:i,accessors:j=[]}=g,{updateTriggersChanged:k}=e;if(e.dataChanged)return!0;if(k){if(k.all)return!0;for(let a of j)if(k[a])return!0}if(f)return!!e.extensionsChanged||j8({oldProps:d,newProps:c,ignoreProps:h,propTypes:this.constructor._propTypes});for(let a of i)if(c[a]!==d[a])return!0;return!1}isAttributeChanged(a){let{changedAttributes:b}=this.state;return a?b&&void 0!==b[a]:!function(a){let b=!0;for(let c in a){b=!1;break}return b}(b)}_getAttributeManager(){return new j3(this.context.device,{id:this.props.id,stats:this.context.stats})}}lc.layerName="AggregationLayer";let ld=lc,le=`\
#version 300 es
in vec3 positions;
in vec3 positions64Low;
in float weights;
out vec4 weightsTexture;
void main()
{
weightsTexture = vec4(weights * weight.weightsScale, 0., 0., 1.);
float radiusTexels = project_pixel_size(weight.radiusPixels) * weight.textureWidth / (weight.commonBounds.z - weight.commonBounds.x);
gl_PointSize = radiusTexels * 2.;
vec3 commonPosition = project_position(positions, positions64Low);
gl_Position.xy = (commonPosition.xy - weight.commonBounds.xy) / (weight.commonBounds.zw - weight.commonBounds.xy) ;
gl_Position.xy = (gl_Position.xy * 2.) - (1.);
gl_Position.w = 1.0;
}
`,lf=`\
#version 300 es
in vec4 weightsTexture;
out vec4 fragColor;
float gaussianKDE(float u){
return pow(2.71828, -u*u/0.05555)/(1.77245385*0.166666);
}
void main()
{
float dist = length(gl_PointCoord - vec2(0.5, 0.5));
if (dist > 0.5) {
discard;
}
fragColor = weightsTexture * gaussianKDE(2. * dist);
DECKGL_FILTER_COLOR(fragColor, geometry);
}
`,lg=`\
#version 300 es
uniform sampler2D inTexture;
out vec4 outTexture;
void main()
{
int yIndex = gl_VertexID / int(maxWeight.textureSize);
int xIndex = gl_VertexID - (yIndex * int(maxWeight.textureSize));
vec2 uv = (0.5 + vec2(float(xIndex), float(yIndex))) / maxWeight.textureSize;
outTexture = texture(inTexture, uv);
gl_Position = vec4(0.0, 0.0, 0.0, 1.0);
gl_PointSize = 1.0;
}
`,lh=`\
#version 300 es
in vec4 outTexture;
out vec4 fragColor;
void main() {
fragColor = outTexture;
fragColor.g = outTexture.r / max(1.0, outTexture.a);
}
`,li={name:"weight",vs:`\
layout(std140) uniform weightUniforms {
  vec4 commonBounds;
  float radiusPixels;
  float textureWidth;
  float weightsScale;
} weight;
`,uniformTypes:{commonBounds:"vec4<f32>",radiusPixels:"f32",textureWidth:"f32",weightsScale:"f32"}},lj={name:"maxWeight",vs:`\
layout(std140) uniform maxWeightUniforms {
  float textureSize;
} maxWeight;
`,uniformTypes:{textureSize:"f32"}},lk={format:"rgba8unorm",dimension:"2d",width:1,height:1,sampler:{minFilter:"linear",magFilter:"linear",addressModeU:"clamp-to-edge",addressModeV:"clamp-to-edge"}},ll=[0,0],lm={SUM:0,MEAN:1},ln=["float32-renderable-webgl","texture-blend-float-webgl"],lo={data:{props:["radiusPixels"]}};class lp extends ld{getShaders(a){let b=[dy];return a.modules&&(b=[...b,...a.modules]),super.getShaders({...a,modules:b})}initializeState(){super.initializeAggregationLayer(lo),this.setState({colorDomain:ll}),this._setupTextureParams(),this._setupAttributes(),this._setupResources()}shouldUpdateState({changeFlags:a}){return a.somethingChanged}updateState(a){super.updateState(a),this._updateHeatmapState(a)}_updateHeatmapState(a){let{props:b,oldProps:c}=a,d=this._getChangeFlags(a);if((d.dataChanged||d.viewportChanged)&&(d.boundsChanged=this._updateBounds(d.dataChanged),this._updateTextureRenderingBounds()),d.dataChanged||d.boundsChanged){if(clearTimeout(this.state.updateTimer),this.setState({isWeightMapDirty:!0}),d.dataChanged){let a=this.getShaders({vs:le,fs:lf});this._createWeightsTransform(a)}}else d.viewportZoomChanged&&this._debouncedUpdateWeightmap();b.colorRange!==c.colorRange&&this._updateColorTexture(a),this.state.isWeightMapDirty&&this._updateWeightmap(),this.setState({zoom:a.context.viewport.zoom})}renderLayers(){let{weightsTexture:a,triPositionBuffer:b,triTexCoordBuffer:c,maxWeightsTexture:d,colorTexture:e,colorDomain:f}=this.state,{updateTriggers:g,intensity:h,threshold:i,aggregation:j}=this.props;return new(this.getSubLayerClass("triangle",k9))(this.getSubLayerProps({id:"triangle-layer",updateTriggers:g}),{coordinateSystem:"default",data:{attributes:{positions:b,texCoords:c}},vertexCount:4,maxTexture:d,colorTexture:e,aggregationMode:lm[j]||0,weightsTexture:a,intensity:h,threshold:i,colorDomain:f})}finalizeState(a){super.finalizeState(a);let{weightsTransform:b,weightsTexture:c,maxWeightTransform:d,maxWeightsTexture:e,triPositionBuffer:f,triTexCoordBuffer:g,colorTexture:h,updateTimer:i}=this.state;b?.destroy(),c?.destroy(),d?.destroy(),e?.destroy(),f?.destroy(),g?.destroy(),h?.destroy(),i&&clearTimeout(i)}_getAttributeManager(){return new j3(this.context.device,{id:this.props.id,stats:this.context.stats})}_getChangeFlags(a){let b={},{dimensions:c}=this.state;b.dataChanged=this.isAttributeChanged()&&"attribute changed"||this.isAggregationDirty(a,{compareAll:!0,dimension:c.data})&&"aggregation is dirty",b.viewportChanged=a.changeFlags.viewportChanged;let{zoom:d}=this.state;return a.context.viewport&&a.context.viewport.zoom===d||(b.viewportZoomChanged=!0),b}_createTextures(){let{textureSize:a,format:b}=this.state;this.setState({weightsTexture:this.context.device.createTexture({...lk,width:a,height:a,format:b}),maxWeightsTexture:this.context.device.createTexture({...lk,width:1,height:1,format:b})})}_setupAttributes(){this.getAttributeManager().add({positions:{size:3,type:"float64",accessor:"getPosition"},weights:{size:1,accessor:"getWeight"}}),this.setState({positionAttributeName:"positions"})}_setupTextureParams(){let{device:a}=this.context,{weightsTextureSize:b}=this.props,c=Math.min(b,a.limits.maxTextureDimension2D),d=ln.every(b=>a.features.has(b));this.setState({textureSize:c,format:d?"rgba32float":"rgba8unorm",weightsScale:d?1:1/255}),d||cs.warn(`HeatmapLayer: ${this.id} rendering to float texture not supported, falling back to low precision format`)()}_createWeightsTransform(a){let{weightsTransform:b}=this.state,{weightsTexture:c}=this.state,d=this.getAttributeManager();b?.destroy(),b=new k4(this.context.device,{id:`${this.id}-weights-transform`,...a,bufferLayout:d.getBufferLayouts(),vertexCount:1,targetTexture:c,parameters:{depthWriteEnabled:!1,blend:!0,blendColorOperation:"add",blendColorSrcFactor:"one",blendColorDstFactor:"one",blendAlphaSrcFactor:"one",blendAlphaDstFactor:"one"},topology:"point-list",modules:[...a.modules,li]}),this.setState({weightsTransform:b})}_setupResources(){this._createTextures();let{device:a}=this.context,{textureSize:b,weightsTexture:c,maxWeightsTexture:d}=this.state,e=this.getShaders({vs:le,fs:lf});this._createWeightsTransform(e);let f=this.getShaders({vs:lg,fs:lh,modules:[lj]}),g=new k4(a,{id:`${this.id}-max-weights-transform`,targetTexture:d,...f,vertexCount:b*b,topology:"point-list",parameters:{depthWriteEnabled:!1,blend:!0,blendColorOperation:"max",blendAlphaOperation:"max",blendColorSrcFactor:"one",blendColorDstFactor:"one",blendAlphaSrcFactor:"one",blendAlphaDstFactor:"one"}});g.model.shaderInputs.setProps({maxWeight:{inTexture:c,textureSize:b}}),this.setState({weightsTexture:c,maxWeightsTexture:d,maxWeightTransform:g,zoom:null,triPositionBuffer:a.createBuffer({byteLength:48}),triTexCoordBuffer:a.createBuffer({byteLength:48})})}updateShaders(a){this._createWeightsTransform({vs:le,fs:lf,...a})}_updateMaxWeightValue(){let{maxWeightTransform:a}=this.state;a.run({parameters:{viewport:[0,0,1,1]},clearColor:[0,0,0,0]})}_updateBounds(a=!1){var b;let c,d,e,f,{viewport:g}=this.context,h=[g.unproject([0,0]),g.unproject([g.width,0]),g.unproject([0,g.height]),g.unproject([g.width,g.height])].map(a=>a.map(Math.fround)),i=(c=h.map(a=>a[0]),d=h.map(a=>a[1]),e=Math.min.apply(null,c),f=Math.max.apply(null,c),[e,Math.min.apply(null,d),f,Math.max.apply(null,d)]),j={visibleWorldBounds:i,viewportCorners:h},k=!1;if(a||!this.state.worldBounds||(b=this.state.worldBounds,(!(i[0]>=b[0])||!(i[2]<=b[2])||!(i[1]>=b[1])||!(i[3]<=b[3]))&&1)){let a=this._worldToCommonBounds(i),b=this._commonToWorldBounds(a);"lnglat"===this.props.coordinateSystem&&(b[1]=Math.max(b[1],-85.051129),b[3]=Math.min(b[3],85.051129),b[0]=Math.max(b[0],-360),b[2]=Math.min(b[2],360));let c=this._worldToCommonBounds(b);j.worldBounds=b,j.normalizedCommonBounds=c,k=!0}return this.setState(j),k}_updateTextureRenderingBounds(){let{triPositionBuffer:a,triTexCoordBuffer:b,normalizedCommonBounds:c,viewportCorners:d}=this.state,{viewport:e}=this.context;a.write(k3(d,3));let f=d.map(a=>(function(a,b){let[c,d,e,f]=b;return[(a[0]-c)/(e-c),(a[1]-d)/(f-d)]})(e.projectPosition(a),c));b.write(k3(f,2))}_updateColorTexture(a){let{colorRange:b}=a.props,{colorTexture:c}=this.state,d=function(a,b=!1,c=Float32Array){let d;if(Number.isFinite(a[0]))d=new c(a);else{d=new c(4*a.length);let b=0;for(let c=0;c<a.length;c++){let e=a[c];d[b++]=e[0],d[b++]=e[1],d[b++]=e[2],d[b++]=Number.isFinite(e[3])?e[3]:255}}if(b)for(let a=0;a<d.length;a++)d[a]/=255;return d}(b,!1,Uint8Array);c?.destroy(),c=this.context.device.createTexture({...lk,data:d,width:b.length,height:1}),this.setState({colorTexture:c})}_updateWeightmap(){let{radiusPixels:a,colorDomain:b,aggregation:c}=this.props,{worldBounds:d,textureSize:e,weightsScale:f,weightsTexture:g}=this.state,h=this.state.weightsTransform;this.state.isWeightMapDirty=!1;let i=this._worldToCommonBounds(d,{useLayerCoordinateSystem:!0});if(b&&"SUM"===c){let{viewport:a}=this.context,c=a.distanceScales.metersPerUnit[2]*(i[2]-i[0])/e;this.state.colorDomain=[b[0]*c*f,b[1]*c*f]}else this.state.colorDomain=b||ll;let j=this.getAttributeManager().getAttributes(),k=this.getModuleSettings();this._setModelAttributes(h.model,j),h.model.setVertexCount(this.getNumInstances());let{viewport:l,devicePixelRatio:m,coordinateSystem:n,coordinateOrigin:o}=k,{modelMatrix:p}=this.props;h.model.shaderInputs.setProps({project:{viewport:l,devicePixelRatio:m,modelMatrix:p,coordinateSystem:n,coordinateOrigin:o},weight:{radiusPixels:a,commonBounds:i,textureWidth:e,weightsScale:f,weightsTexture:g}}),h.run({parameters:{viewport:[0,0,e,e]},clearColor:[0,0,0,0]}),this._updateMaxWeightValue()}_debouncedUpdateWeightmap(a=!1){let{updateTimer:b}=this.state,{debounceTimeout:c}=this.props;a?(b=null,this._updateBounds(!0),this._updateTextureRenderingBounds(),this.setState({isWeightMapDirty:!0})):(this.setState({isWeightMapDirty:!1}),clearTimeout(b),b=setTimeout(this._debouncedUpdateWeightmap.bind(this,!0),c)),this.setState({updateTimer:b})}_worldToCommonBounds(a,b={}){let c,d,{useLayerCoordinateSystem:e=!1}=b,[f,g,h,i]=a,{viewport:j}=this.context,{textureSize:k}=this.state,{coordinateSystem:l}=this.props,m=e&&("lnglat-offsets"===l||"meter-offsets"===l),n=m?j.projectPosition(this.props.coordinateOrigin):[0,0],o=2*k/j.scale;return e&&!m?(c=this.projectPosition([f,g,0]),d=this.projectPosition([h,i,0])):(c=j.projectPosition([f,g,0]),d=j.projectPosition([h,i,0])),function(a,b,c){let[d,e,f,g]=a,h=f-d,i=g-e,j=h,k=i;h/i<b/c?j=b/c*i:k=c/b*h,j<b&&(j=b,k=c);let l=(f+d)/2,m=(g+e)/2;return[l-j/2,m-k/2,l+j/2,m+k/2]}([c[0]-n[0],c[1]-n[1],d[0]-n[0],d[1]-n[1]],o,o)}_commonToWorldBounds(a){let[b,c,d,e]=a,{viewport:f}=this.context,g=f.unprojectPosition([b,c]),h=f.unprojectPosition([d,e]);return g.slice(0,2).concat(h.slice(0,2))}}lp.layerName="HeatmapLayer",lp.defaultProps={getPosition:{type:"accessor",value:a=>a.position},getWeight:{type:"accessor",value:1},intensity:{type:"number",min:0,value:1},radiusPixels:{type:"number",min:1,max:100,value:50},colorRange:[[255,255,178],[254,217,118],[254,178,76],[253,141,60],[240,59,32],[189,0,38]],threshold:{type:"number",min:0,max:1,value:.05},colorDomain:{type:"array",value:null,optional:!0},aggregation:"SUM",weightsTextureSize:{type:"number",min:128,max:2048,value:2048},debounceTimeout:{type:"number",min:0,max:1e3,value:500}};var lq=a.i(68803);let lr=C("external-link",[["path",{d:"M15 3h6v6",key:"1q9fwt"}],["path",{d:"M10 14 21 3",key:"gplh6r"}],["path",{d:"M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6",key:"a6xqqp"}]]),ls=[{key:"pins",label:"Pins",icon:Y},{key:"paths",label:"Paths",icon:Z},{key:"heatmap",label:"Heat",icon:$},{key:"time",label:"Time",icon:_}];function lt(){let{mapMode:a,setMapMode:b,filteredEvents:c,filters:d}=x(),e=d.dateFrom||(c[0]?.start_eastern.slice(0,10)??"—"),f=d.dateTo||(c[c.length-1]?.end_eastern.slice(0,10)??"—");return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsxs)("div",{className:"absolute top-3 left-3 z-20 flex flex-col gap-2",children:[(0,r.jsx)("div",{className:"bg-surface/90 backdrop-blur border border-border rounded-[var(--r)] shadow-[var(--shadow)] p-1 flex flex-col gap-1",children:ls.map(c=>{let d=c.icon;return(0,r.jsx)("button",{onClick:()=>b(c.key),title:c.label,className:`flex items-center justify-center w-8 h-8 rounded-[var(--r-sm)] transition ${a===c.key?"bg-signal text-white":"text-muted hover:text-ink hover:bg-surface-2"}`,children:(0,r.jsx)(d,{className:"w-4 h-4"})},c.key)})}),(0,r.jsxs)("button",{onClick:()=>alert("Kepler.gl pop-out is a later-phase stub."),className:"bg-surface/90 backdrop-blur border border-border rounded-[var(--r)] shadow-[var(--shadow)] px-2.5 py-1.5 text-xs font-medium text-muted hover:text-ink flex items-center gap-1.5",children:[(0,r.jsx)(lr,{className:"w-3.5 h-3.5"})," Kepler"]})]}),(0,r.jsxs)("div",{className:"absolute top-3 right-14 z-20 bg-surface/90 backdrop-blur border border-border rounded-[var(--r)] shadow-[var(--shadow)] px-3 py-2",children:[(0,r.jsx)("div",{className:"text-[10px] uppercase tracking-wide font-mono text-muted mb-0.5",children:"Events"}),(0,r.jsx)("div",{className:"text-sm font-semibold text-ink",children:c.length.toLocaleString()}),(0,r.jsxs)("div",{className:"text-[10px] font-mono text-faint mt-0.5",children:[e," → ",f]})]})]})}let lu=[{label:"Visit",color:"var(--signal)"},{label:"Activity",color:"var(--concerning)"},{label:"Path",color:"var(--faint)"},{label:"Selected",color:"#3db8cc"}],lv=[{label:"Night (20-06)",color:"var(--overnight)"},{label:"Day (06-20)",color:"var(--concerning)"}];function lw(){let{mapMode:a}=x();return"heatmap"===a?null:(0,r.jsxs)("div",{className:"absolute bottom-12 left-3 z-20 bg-surface/90 backdrop-blur border border-border rounded-[var(--r)] shadow-[var(--shadow)] px-3 py-2",children:[(0,r.jsx)("div",{className:"text-[10px] uppercase tracking-wide font-mono text-muted mb-1.5",children:"time"===a?"Time of day":"Pins"}),(0,r.jsx)("div",{className:"flex flex-col gap-1",children:("time"===a?lv:lu).map(a=>(0,r.jsxs)("div",{className:"flex items-center gap-2",children:[(0,r.jsx)("span",{className:"w-2.5 h-2.5 rounded-full",style:{background:a.color}}),(0,r.jsx)("span",{className:"text-[11px] text-ink",children:a.label})]},a.label))})]})}function lx(){let{filteredEvents:a,filters:b,setFilters:c}=x(),d=(0,s.useMemo)(()=>(function(a){if(!a.length)return{min:"",max:""};let b=[...a].sort((a,b)=>a.start_eastern.localeCompare(b.start_eastern));return{min:b[0].start_eastern.slice(0,10),max:b[b.length-1].end_eastern.slice(0,10)}})(a),[a]),e=d.min,f=d.max,g=b.dateFrom||e,h=b.dateTo||f;return e&&f?(0,r.jsx)("div",{className:"absolute bottom-3 left-3 right-3 z-20 bg-surface/95 backdrop-blur border border-border rounded-[var(--r)] shadow-[var(--shadow)] px-3 py-2",children:(0,r.jsxs)("div",{className:"flex items-center gap-3",children:[(0,r.jsx)("span",{className:"text-[10px] font-mono text-muted whitespace-nowrap",children:g}),(0,r.jsxs)("div",{className:"flex-1 relative h-6",children:[(0,r.jsx)("input",{type:"range",min:0,max:ly(e,f),value:ly(e,g),onChange:a=>c({...b,dateFrom:lz(e,Number(a.target.value))}),className:"absolute w-full accent-signal h-1 top-2.5 opacity-70"}),(0,r.jsx)("input",{type:"range",min:0,max:ly(e,f),value:ly(e,h),onChange:a=>c({...b,dateTo:lz(e,Number(a.target.value))}),className:"absolute w-full accent-signal h-1 top-2.5"})]}),(0,r.jsx)("span",{className:"text-[10px] font-mono text-muted whitespace-nowrap",children:h})]})}):null}function ly(a,b){let c=new Date(a).getTime();return Math.max(0,Math.round((new Date(b).getTime()-c)/864e5))}function lz(a,b){let c=new Date(a);return c.setDate(c.getDate()+b),c.toISOString().slice(0,10)}function lA(){let{filteredEvents:a,mapMode:b,selectedEvent:c,setSelectedEvent:d,theme:e}=x(),f=(0,s.useRef)(null),g=(0,s.useMemo)(()=>({longitude:-84.5,latitude:42.7,zoom:6.5,pitch:0,bearing:0}),[]);(0,s.useEffect)(()=>{c&&f.current&&f.current.flyTo({center:[c.lng,c.lat],zoom:10,duration:700})},[c]);let h="dark"===e?[61,184,204,255]:[20,125,140,255],i="dark"===e?[146,133,224,220]:[106,90,192,220],j="dark"===e?[217,154,43,220]:[182,125,22,220],k=(0,s.useMemo)(()=>{let e={id:"events",data:a,pickable:!0};return"pins"===b?[new kM({...e,id:"pins",getPosition:a=>[a.lng,a.lat],getFillColor:a=>a.event_id===c?.event_id?[61,184,204,255]:"overnight"===a.overnight_simple?i:h,getRadius:a=>a.event_id===c?.event_id?14:7,radiusMinPixels:3,radiusMaxPixels:30,onClick:a=>d(a.object||null)})]:"paths"===b?[new k1({id:"path",data:[{path:[...a].sort((a,b)=>new Date(a.start_utc).getTime()-new Date(b.start_utc).getTime()).map(a=>[a.lng,a.lat])}],getPath:a=>a.path,getColor:h,getWidth:3,widthMinPixels:2}),new kM({...e,id:"path-pins",getPosition:a=>[a.lng,a.lat],getFillColor:h,getRadius:4,radiusMinPixels:2,onClick:a=>d(a.object||null)})]:"heatmap"===b?[new lp({...e,id:"heat",getPosition:a=>[a.lng,a.lat],getWeight:a=>a.probability,radiusPixels:25,intensity:1,threshold:.05})]:[new kM({...e,id:"time",getPosition:a=>[a.lng,a.lat],getFillColor:a=>{let b=new Date(a.start_eastern).getHours();return b<6||b>=20?i:j},getRadius:7,radiusMinPixels:3,radiusMaxPixels:24,onClick:a=>d(a.object||null)})]},[a,b,c,d,h,i,j]);return(0,r.jsxs)("div",{className:"relative flex-1 h-full overflow-hidden",children:[(0,r.jsx)(kD,{initialViewState:g,controller:!0,layers:k,getTooltip:({object:a})=>a?{text:`${a.serial_display} \xb7 ${a.event_type}
${a.place_id}
${a.start_eastern.slice(0,16)}`}:null,style:{position:"absolute",inset:"0",zIndex:"1"},children:(0,r.jsx)(aE,{ref:f,mapLib:lq,mapStyle:"dark"===e?"https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json":"https://basemaps.cartocdn.com/gl/positron-gl-style/style.json",style:{width:"100%",height:"100%"},attributionControl:!1,children:(0,r.jsx)(aC,{position:"top-right"})})}),(0,r.jsx)(lt,{}),(0,r.jsx)(lw,{}),(0,r.jsx)(lx,{})]})}let lB=C("arrow-up-down",[["path",{d:"m21 16-4 4-4-4",key:"f6ql7i"}],["path",{d:"M17 20V4",key:"1ejh1v"}],["path",{d:"m3 8 4-4 4 4",key:"11wl7u"}],["path",{d:"M7 4v16",key:"1glfcx"}]]),lC=C("hash",[["line",{x1:"4",x2:"20",y1:"9",y2:"9",key:"4lhtct"}],["line",{x1:"4",x2:"20",y1:"15",y2:"15",key:"vyu0kd"}],["line",{x1:"10",x2:"8",y1:"3",y2:"21",key:"1ggp8o"}],["line",{x1:"16",x2:"14",y1:"3",y2:"21",key:"weycgp"}]]),lD=C("circle-alert",[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["line",{x1:"12",x2:"12",y1:"8",y2:"12",key:"1pkeuh"}],["line",{x1:"12",x2:"12.01",y1:"16",y2:"16",key:"4dfq90"}]]);function lE(){let{filteredEvents:a,selectedEvent:b,setSelectedEvent:c,filters:d}=x(),[e,f]=(0,s.useState)({key:"start_eastern",dir:"desc"}),g=(0,s.useMemo)(()=>{let b=[...a];return b.sort((a,b)=>{let c=a[e.key],d=b[e.key];return("start_eastern"===e.key?(c=new Date(c).getTime(),d=new Date(d).getTime()):"duration"===e.key&&(c=lF(a.start_eastern,a.end_eastern),d=lF(b.start_eastern,b.end_eastern)),c<d)?"asc"===e.dir?-1:1:c>d?"asc"===e.dir?1:-1:0}),b},[a,e]),h=({k:a,children:b,className:c=""})=>(0,r.jsxs)("button",{onClick:()=>f({key:a,dir:e.key===a&&"asc"===e.dir?"desc":"asc"}),className:`flex items-center gap-1 text-left text-[10.5px] uppercase tracking-wide font-mono font-medium text-muted hover:text-ink ${c}`,children:[b," ",(0,r.jsx)(lB,{className:"w-3 h-3"})]});return(0,r.jsxs)("div",{className:"flex flex-col h-full w-full min-w-[280px] max-w-[420px] bg-surface border-l border-border",children:[(0,r.jsxs)("div",{className:"px-4 py-3 border-b border-border flex items-center justify-between",children:[(0,r.jsxs)("div",{className:"text-sm font-medium text-ink flex items-center gap-2",children:[(0,r.jsx)(lC,{className:"w-4 h-4"})," Results"]}),(0,r.jsxs)("div",{className:"flex items-center gap-2",children:[(0,r.jsxs)("span",{className:"text-xs text-muted",children:[a.length," events"]}),(0,r.jsx)("button",{onClick:()=>c(b?null:a[0]??null),className:"p-1.5 rounded-md border border-border text-muted hover:text-ink hover:bg-surface-2",title:"Open in map",children:(0,r.jsx)(Y,{className:"w-3.5 h-3.5"})}),(0,r.jsx)("button",{onClick:()=>alert("Analytics view stub"),className:"p-1.5 rounded-md border border-border text-muted hover:text-ink hover:bg-surface-2",title:"Open in analytics",children:(0,r.jsx)(lr,{className:"w-3.5 h-3.5"})})]})]}),(0,r.jsxs)("div",{className:"flex-1 overflow-auto",children:[(0,r.jsxs)("table",{className:"w-full text-left border-collapse",children:[(0,r.jsx)("thead",{className:"sticky top-0 bg-surface z-10",children:(0,r.jsxs)("tr",{className:"border-b border-border",children:[(0,r.jsx)("th",{className:"px-3 py-2",children:(0,r.jsx)(h,{k:"serial_display",children:"ID"})}),(0,r.jsx)("th",{className:"px-3 py-2",children:(0,r.jsx)(h,{k:"place_id",children:"Place"})}),(0,r.jsx)("th",{className:"px-3 py-2",children:(0,r.jsx)(h,{k:"start_eastern",children:"Start"})}),(0,r.jsx)("th",{className:"px-3 py-2",children:(0,r.jsx)(h,{k:"overnight_simple",children:"Ovn"})}),(0,r.jsx)("th",{className:"px-3 py-2 text-right",children:(0,r.jsx)(h,{k:"duration",className:"justify-end",children:"Dur"})})]})}),(0,r.jsx)("tbody",{children:g.map(a=>{let d=b?.event_id===a.event_id,e=lF(a.start_eastern,a.end_eastern);return(0,r.jsxs)("tr",{onClick:()=>c(a),className:`text-xs border-b border-border cursor-pointer transition ${d?"bg-signal-soft":"hover:bg-surface-2"}`,children:[(0,r.jsx)("td",{className:"px-3 py-2 font-mono text-muted",children:a.serial_display}),(0,r.jsx)("td",{className:"px-3 py-2 max-w-[120px] truncate",title:a.place_id,children:a.place_id}),(0,r.jsxs)("td",{className:"px-3 py-2 font-mono text-muted",children:[a.start_eastern.slice(0,10),(0,r.jsx)("br",{}),(0,r.jsx)("span",{className:"text-[10px]",children:a.start_eastern.slice(11,16)})]}),(0,r.jsx)("td",{className:"px-3 py-2",children:"overnight"===a.overnight_simple?(0,r.jsx)("span",{className:"inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-overnight/10 text-overnight border border-overnight/20",children:"ON"}):(0,r.jsx)("span",{className:"text-faint",children:"—"})}),(0,r.jsx)("td",{className:"px-3 py-2 text-right font-mono text-muted",children:function(a){if(a<60)return`${a}m`;let b=Math.floor(a/60),c=a%60;return c?`${b}h ${c}m`:`${b}h`}(e)})]},a.event_id)})})]}),0===g.length&&(0,r.jsxs)("div",{className:"flex flex-col items-center justify-center h-40 text-muted text-sm gap-2",children:[(0,r.jsx)(lD,{className:"w-5 h-5"})," No events match filters."]})]})]})}function lF(a,b){return Math.max(0,Math.round((new Date(b).getTime()-new Date(a).getTime())/6e4))}let lG=C("send",[["path",{d:"M14.536 21.686a.5.5 0 0 0 .937-.024l6.5-19a.496.496 0 0 0-.635-.635l-19 6.5a.5.5 0 0 0-.024.937l7.93 3.18a2 2 0 0 1 1.112 1.11z",key:"1ffxy3"}],["path",{d:"m21.854 2.147-10.94 10.939",key:"12cjpa"}]]),lH=C("bot",[["path",{d:"M12 8V4H8",key:"hb8ula"}],["rect",{width:"16",height:"12",x:"4",y:"8",rx:"2",key:"enze0r"}],["path",{d:"M2 14h2",key:"vft8re"}],["path",{d:"M20 14h2",key:"4cs60a"}],["path",{d:"M15 13v2",key:"1xurst"}],["path",{d:"M9 13v2",key:"rq6x2g"}]]),lI=C("receipt",[["path",{d:"M4 2v20l2-1 2 1 2-1 2 1 2-1 2 1 2-1 2 1V2l-2 1-2-1-2 1-2-1-2 1-2-1-2 1Z",key:"q3az6g"}],["path",{d:"M16 8h-6a2 2 0 1 0 0 4h4a2 2 0 1 1 0 4H8",key:"1h4pet"}],["path",{d:"M12 17.5v-11",key:"1jc1ny"}]]),lJ=C("user",[["path",{d:"M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2",key:"975kel"}],["circle",{cx:"12",cy:"7",r:"4",key:"17ys0d"}]]),lK=C("sparkles",[["path",{d:"M9.937 15.5A2 2 0 0 0 8.5 14.063l-6.135-1.582a.5.5 0 0 1 0-.962L8.5 9.936A2 2 0 0 0 9.937 8.5l1.582-6.135a.5.5 0 0 1 .963 0L14.063 8.5A2 2 0 0 0 15.5 9.937l6.135 1.581a.5.5 0 0 1 0 .964L15.5 14.063a2 2 0 0 0-1.437 1.437l-1.582 6.135a.5.5 0 0 1-.963 0z",key:"4pj2yx"}],["path",{d:"M20 3v4",key:"1olli1"}],["path",{d:"M22 5h-4",key:"1gvqau"}],["path",{d:"M4 17v2",key:"vumght"}],["path",{d:"M5 18H3",key:"zchphs"}]]);function lL(){let{mode:a,filters:b,selectedEvent:c,setSelectedEvent:d}=x(),[e,f]=(0,s.useState)([{id:"intro",role:"assistant",content:"Ask me about the timeline. I'll cite rows as receipts."}]),[g,h]=(0,s.useState)(""),i=async a=>{if(a?.preventDefault(),!g.trim())return;let c=g.trim();h("");let d={id:crypto.randomUUID(),role:"user",content:c},e={id:crypto.randomUUID(),role:"assistant",content:"",loading:!0};f(a=>[...a,d,e]);let i=await U.askAgent(c,b);f(a=>a.map(a=>a.id===e.id?{...a,content:i.answer,receipts:i.receipts,loading:!1}:a))};return(0,r.jsxs)("div",{className:"flex flex-col h-full border-t border-border bg-surface",children:[(0,r.jsxs)("div",{className:"flex items-center justify-between px-4 py-2 border-b border-border",children:[(0,r.jsxs)("div",{className:"flex items-center gap-2 text-xs font-medium text-muted",children:[(0,r.jsx)(lH,{className:"w-4 h-4"})," Assistant"]}),(0,r.jsx)("span",{className:`text-[10px] px-2 py-0.5 rounded-full border ${"agent"===a?"bg-signal text-white border-signal":"bg-surface-2 text-muted border-border"}`,children:"agent"===a?"Agent mode":"Manual mode"})]}),(0,r.jsx)("div",{className:"flex-1 overflow-y-auto p-3 space-y-3",children:e.map(a=>(0,r.jsxs)("div",{className:`flex gap-2 ${"user"===a.role?"flex-row-reverse":""}`,children:[(0,r.jsx)("div",{className:`w-6 h-6 rounded-full flex items-center justify-center shrink-0 ${"user"===a.role?"bg-surface-2 text-muted":"bg-signal-soft text-signal"}`,children:"user"===a.role?(0,r.jsx)(lJ,{className:"w-3.5 h-3.5"}):(0,r.jsx)(lK,{className:"w-3.5 h-3.5"})}),(0,r.jsxs)("div",{className:`max-w-[85%] px-3 py-2 rounded-xl text-xs border ${"user"===a.role?"bg-surface-2 border-border text-ink rounded-br-none":"bg-surface text-ink border-border rounded-bl-none"}`,children:[a.loading?(0,r.jsx)("span",{className:"animate-pulse",children:"Thinking…"}):a.content,a.receipts&&a.receipts.length>0&&(0,r.jsx)("div",{className:"mt-2 flex flex-wrap gap-1",children:a.receipts.map(a=>(0,r.jsxs)("button",{onClick:()=>U.getEvent(a).then(a=>a&&d(a)),className:"inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-signal-soft border border-signal/20 text-signal text-[10px] hover:bg-signal hover:text-white transition",children:[(0,r.jsx)(lI,{className:"w-3 h-3"}),a.slice(0,8),"…"]},a))})]})]},a.id))}),(0,r.jsxs)("form",{onSubmit:i,className:"p-3 border-t border-border flex gap-2",children:[(0,r.jsx)("input",{value:g,onChange:a=>h(a.target.value),placeholder:"agent"===a?"Ask the agent…":"Chat disabled in manual mode",disabled:"agent"!==a,className:"flex-1 px-3 py-2 text-xs bg-surface-2 border border-border rounded-lg focus:outline-none focus:border-signal disabled:opacity-50"}),(0,r.jsx)("button",{type:"submit",disabled:"agent"!==a||!g.trim(),className:"px-3 py-2 rounded-lg bg-signal text-white disabled:opacity-50",children:(0,r.jsx)(lG,{className:"w-4 h-4"})})]})]})}function lM(){let{tab:a,setFilteredEvents:b}=x();return((0,s.useEffect)(()=>{U.searchEvents({}).then(b)},[b]),"explore"!==a)?(0,r.jsxs)("div",{className:"flex flex-col h-full",children:[(0,r.jsx)(M,{}),(0,r.jsx)("div",{className:"flex-1 flex items-center justify-center text-muted",children:(0,r.jsxs)("div",{className:"text-center",children:[(0,r.jsx)("h2",{className:"text-lg font-medium text-ink mb-1",children:a.charAt(0).toUpperCase()+a.slice(1)}),(0,r.jsxs)("p",{className:"text-sm",children:["Placeholder page for ",a," module."]})]})})]}):(0,r.jsxs)("div",{className:"flex flex-col h-full",children:[(0,r.jsx)(M,{}),(0,r.jsxs)("div",{className:"flex-1 flex overflow-hidden",children:[(0,r.jsx)("div",{className:"flex h-full",children:(0,r.jsxs)("div",{className:"flex flex-col h-full",children:[(0,r.jsx)("div",{className:"flex-1 overflow-hidden",children:(0,r.jsx)(ae,{})}),(0,r.jsx)("div",{className:"h-[260px] shrink-0",children:(0,r.jsx)(lL,{})})]})}),(0,r.jsx)(lA,{}),(0,r.jsx)(lE,{})]})]})}a.s(["default",0,function(){return(0,r.jsx)(w,{children:(0,r.jsx)(lM,{})})}],83844)}];

//# sourceMappingURL=%5Broot-of-the-server%5D__04vm9p7._.js.map