# Chart and visualization libraries — owner list

> _Byline: Claude Code · Opus 5 · 2026-09-14 — owner-supplied list (23:32 EDT) during the portal board → native Homepage widgets rework. Same intent as `shadcn-block-libraries-2026-09-14.md`: use these instead of hand-rolling charts._

**Context.** Owner rulings 2026-09-14 23:26–23:27 EDT: portal dashboards must use native, glanceable widgets, graphs, Gantt charts and to-do lists, not framed pages. These are the libraries to build them with. A license is recorded only after it's checked at the source. Commercial-licensed options are marked as such, not assumed free.

| # | Library | Notes for our use (verify before relying) |
|---|---|---|
| 1 | FusionCharts Suite — https://www.fusioncharts.com/ | Broad chart and Gantt set. Commercial license, check before use. |
| 2 | D3.js | Low-level SVG/data binding. The base for custom visuals; highest effort. |
| 3 | Chart.js | Canvas charts, small, common. No native Gantt. |
| 4 | Taucharts | Grammar-of-graphics charts (D3-based). Check maintenance status. |
| 5 | Two.js | 2D drawing API (SVG/canvas/WebGL). Drawing, not charts. |
| 6 | Psst.js | Listed by owner. Identify the project and license before use. |
| 7 | Raphael.js | Legacy SVG drawing. Check maintenance status. |
| 8 | Anime.js | Animation, not charts. Useful for widget transitions. |
| 9 | Recharts | React + D3 charts. Fits React surfaces such as Intake and Probata. |
| 10 | Trading Vue.js | Vue financial and time-series charts. |
| 11 | Highcharts | Full suite including Gantt. Commercial license, check before use. |
| 12 | Chartkick | One-line wrapper over Chart.js, Google Charts and Highcharts. |
| 13 | Pixi.js | WebGL 2D renderer. For large or animated scenes. |
| 14 | Three.js | 3D WebGL. |
| 15 | Zdog | Pseudo-3D flat illustration engine. |
| — | ApexCharts — https://apexcharts.com/ | SVG charts with a rangeBar/timeline (Gantt-style) type. Check the current license terms at the source. |
| — | CanvasJS — https://canvasjs.com/ | Canvas charts. Commercial license, check before use. |
| — | Container-Aware Plotly Chart | Plotly chart pattern that resizes to its container. Suits widget cards. |
| — | Alpine.js and Tailwind Radial Chart | Lightweight radial/gauge widget pattern, no framework. |
| — | Oculus II | Listed by owner. Identify it before use. |
| — | TimelineJS | Narrative timelines (Knight Lab). Candidate for project and case timelines. |
| — | amCharts 5: SerpentineChart and CurveColumnSeries (owner 23:33 EDT) | Timeline package: winding "serpentine" timelines and curved column series, for roadmap- or case-timeline-style visuals. Check licensing at the source: the free build carries an amCharts credit, removed with a commercial license. |

## Frontend JS library example galleries (owner 23:33 EDT)

Source index: freefrontend.com, with a curated example gallery per library. Use these to find a ready example before building a widget, interaction or effect.

| Area | Libraries (gallery links) |
|---|---|
| Charts / data viz | [amcharts.js](https://freefrontend.com/amcharts-js/) · [apexcharts.js](https://freefrontend.com/apexcharts-js/) · [chart.js](https://freefrontend.com/chart-js/) · [d3.js](https://freefrontend.com/d3-js/) |
| Lightweight UI / reactivity | [alpine.js](https://freefrontend.com/alpine-js/) · [tippy.js](https://freefrontend.com/tippy-js/) (tooltips) · [swiper.js](https://freefrontend.com/swiper-js/) (carousels) · [fancybox.js](https://freefrontend.com/fancybox-js/) · [lightgallery.js](https://freefrontend.com/lightgallery-js/) (lightboxes) |
| Drag, gestures, layout motion | [dragula.js](https://freefrontend.com/dragula-js/) · [hammer.js](https://freefrontend.com/hammer-js/) · [flip.js](https://freefrontend.com/flip-js/) · [lenis.js](https://freefrontend.com/lenis-js/) (smooth scroll) |
| Animation | [anime.js](https://freefrontend.com/anime-js-examples/) · [motion.js](https://freefrontend.com/motion-js/) · [lottie.js](https://freefrontend.com/lottie-js/) · [polymorph.js](https://freefrontend.com/polymorph-js/) · [splitting.js](https://freefrontend.com/splitting-js/) · [vfx.js](https://freefrontend.com/vfx-js/) · [tsparticles.js](https://freefrontend.com/tsparticles-js/) |
| GSAP and plugins | [gsap.js](https://freefrontend.com/gsap-js/): [customease](https://freefrontend.com/customease-js/), [draggable](https://freefrontend.com/draggable-js/), [drawSVGPlugin](https://freefrontend.com/draw-svg-plugin-js/), [observer](https://freefrontend.com/observer-js/), [scrambleTextPlugin](https://freefrontend.com/scramble-text-plugin-js/), [scrollTrigger](https://freefrontend.com/scroll-trigger-js/), [splitText](https://freefrontend.com/split-text-js/), [tweenMax](https://freefrontend.com/tweenmax-js/) |
| Data helpers | [date-fns.js](https://freefrontend.com/date-fns/) (dates) · [faker.js](https://freefrontend.com/faker-js/) (fake data; demos only, never canonical) · [rgba.js](https://freefrontend.com/rgba-js/) (colour) |
| Maps / media / 3D / creative | [leaflet.js](https://freefrontend.com/leaflet-js/) (maps) · [howler.js](https://freefrontend.com/howler-js/) (audio) · [model-viewer.js](https://freefrontend.com/model-viewer-js/) · [three.js](https://freefrontend.com/three-js/) · [three.js games](https://freefrontend.com/three-js-games/) · [p5.js](https://freefrontend.com/p5-js/) · [zim.js](https://freefrontend.com/zim-js/) |

## To-do / task components (owner 23:35 EDT)

| Library | Notes (checked 2026-09-14) |
|---|---|
| [TodoMVC React example](https://github.com/tastejs/todomvc/tree/master/examples/react) | A simple reference To-Do MVC in React. A pattern to copy, not a package. |
| [react-beautiful-dnd](https://github.com/atlassian/react-beautiful-dnd) | **Deprecated. Atlassian announced the deprecation in October 2024 and archived the repository read-only on April 30, 2025** ([repo](https://github.com/atlassian/react-beautiful-dnd), [issue #2672](https://github.com/atlassian/react-beautiful-dnd/issues/2672)). Don't start new work on it. |
| [Pragmatic drag and drop](https://atlassian.design/components/pragmatic-drag-and-drop/optional-packages/react-beautiful-dnd-migration) | Atlassian's successor to react-beautiful-dnd, with a migration package. Built for Jira/Trello-scale boards. |
| @dnd-kit | Actively maintained, small, accessible React drag-and-drop ([2026 comparison](https://www.pkgpulse.com/guides/dnd-kit-vs-react-beautiful-dnd-vs-pragmatic-drag-drop-2026)). Default pick for sortable to-do lists and kanban columns. |

For drag-and-drop task boards in React surfaces (Intake, Probata, the progress board), use @dnd-kit or Pragmatic drag and drop, not react-beautiful-dnd.

## Calendars and schedulers (owner 23:37 EDT)

Notes are from the owner's source article unless marked. Check license and maintenance at each repo before adopting.

| # | Library | Notes |
|---|---|---|
| 1 | FullCalendar.js; [Modernize admin template](https://adminmart.com/product/modernize-bootstrap-5-admin-template/) (AdminMart, Bootstrap 5) | Feature-rich event calendar. The Modernize template bundles FullCalendar plus dashboards, email, chat and kanban apps, and 50+ page templates. Check the template's price and license, and whether any FullCalendar features we want are premium plugins. |
| 2 | [Calendar.js](https://github.com/williamtroup/Calendar.js) ([review](https://medevel.com/calendar-js/)) | Responsive drag-and-drop event calendar. Runs as a library or a self-hosted app; imports and exports events from other calendars. |
| 3 | [TOAST UI Calendar](https://ui.toast.com/tui-calendar) | Enterprise-grade open-source calendar. Monthly, weekly, daily, 2-week, 3-week and no-weekend views, drag to create or edit, time zones, themes. |
| 4 | [jsCalendar](https://github.com/GramThanos/jsCalendar) | Lightweight; multi-language, themes. Usable as a date picker or a simple event manager. |
| 5 | [Event Calendar (vkurko)](https://github.com/vkurko/calendar) | Full-sized drag-and-drop calendar with resource view. Svelte core, ES6 modules, works with React/Vue/Angular. About 37 KB compressed, zero dependencies. A FullCalendar-like API, so the lightweight alternative. |
| 6 | [Caleandar](https://github.com/jackducasse/caleandar) | About 7.5 KB, no dependencies, optional themes, click handlers on events. |
| 7 | [Evo Calendar](https://github.com/edlynvillegas/evo-calendar) | Modern-looking responsive event calendar; event types (event, holiday, birthday), add/remove/view events. |
| 8 | [CalenStyle](https://github.com/nehakadam/CalenStyle) | jQuery drag-and-drop calendar: planners, timeline agenda, date-time picker. Depends on jQuery. |
| 9 | [dhtmlxScheduler](https://github.com/DHTMLX/scheduler) | Google-style scheduler with 10 views (day, week, month, year, agenda, timeline…), recurring and multiday events, map locations. The GitHub edition is **GPL**; commercial licenses exist. |
| 10 | [calendar-javascript-lib](https://github.com/nizarmah/calendar-javascript-lib) | Calendar and organizer. **Archived, no updates in ~3 years (per source). Don't adopt.** |
| 11 | [jquery-calendar-bs4](https://github.com/ArrobeFr/jquery-calendar-bs4) | Responsive jQuery scheduler on Bootstrap 4 + moment.js, MIT (per source). jQuery/Bootstrap-bound. |

For the portal, Intake and Probata React/TanStack surfaces, the first candidates are Event Calendar (vkurko, lightweight, resource view) and TOAST UI Calendar. Avoid jQuery-bound and archived options.

**How to use:**
- Pick from this list for any dashboard, portal widget, Gantt, timeline or chart work.
- Prefer a library with the needed chart type built in (Gantt/timeline, radial, sparkline) over custom D3.
- Serve assets locally from our own hosts; no CDN.
- Tables stay Glide Data Grid per the settled stack. These libraries are for charts and visuals only.
