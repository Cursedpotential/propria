---
tags: [ui-components, shadcn, blocks, design-system, charts, dashboards, frontend]
---

# shadcn/ui block and component libraries — reference

> _Byline: Claude Code · Fable 5.1 · 2026-09-14 20:58 EDT — saved on owner order ("save the sites mentioned here… idk why we are hand-rolling shit like this"). Source: [AdminLTE, "shadcn/ui block libraries", 2026-09-07](https://adminlte.io/blog/shadcn-ui-block-libraries/). Facts below are the article's claims as of its date; verify counts and pricing before relying on them._

## Owner intent

Stop hand-rolling UI chrome. For every drawer, tab, table shell, dashboard tile, form, sidebar or chart, reach for an existing shadcn-compatible block first and adapt it to the Propria tokens. Tables and grids stay **Glide Data Grid + TanStack Table** (settled 2026-09-12); routing/queries stay TanStack; these libraries supply the chrome around them.

## The libraries

| Library | URL | What it offers | License / price | shadcn CLI | Notes |
|---|---|---|---|---|---|
| **shadcn/ui Charts** | https://ui.shadcn.com/charts | Area, bar, line, pie, radar, radial; tooltip presets (Recharts) | Free, MIT | yes | Official; no third-party version skew |
| **Origin UI → COSS** | https://coss.com/ui | 484 "particles": accordion, alert dialog, autocomplete, breadcrumb, avatar, badge, **timeline**, … | Free, MIT | copy/paste | Base UI backend; rebranded 2026; ~9.7k stars |
| **ReUI** | https://reui.io | 1,003+ components: forms, navigation, data display, feedback, overlays; dual Base UI + Radix | Free MIT + Pro | yes | Shadcn Create compatible; ~2.9k stars |
| **Shadcnblocks** | https://shadcnblocks.com | 1,557 blocks, 1,189 components, 15 templates; Next/Astro/Vue/Svelte | Freemium | yes | Figma kit V2 (2026-03); visual Builder |
| **shadcn.io** | https://shadcn.io | 6,000+ blocks in 56 categories: dashboards, auth, CRM, pricing; 59 themes; icons | Freemium | registry | Largest marketplace; community-run, not official |
| **21st.dev** | https://21st.dev | Community components + agent templates; AI Agents SDK | Free + paid tiers | yes | YC-backed |
| **Cult UI** | https://cult-ui.com | 75+ animated components + 100+ AI-SDK agent/chat patterns | Free MIT + Pro | yes | Bridges UI and agent workflows; ~3.8k stars |
| **Magic UI** | https://magicui.design | 150+ animated components: terminal, marquee, bento grids, kinetic text | Free MIT + Pro ($199) | yes | ~20.8k stars |
| **Aceternity UI** | https://ui.aceternity.com | 200+ motion-rich components: hero, bento, parallax, canvas cards | Free core + all-access ($249) | copy/paste | Landing-page oriented |
| **Tailark** | https://tailark.com | 300+ marketing blocks; 4 themes | Freemium | copy/paste | Landing-page conversion focus |
| **Velora UI** | https://github.com/ColorlibHQ/velora-ui | 64 components + multi-page template; animated backgrounds, cards, nav, forms | Free, MIT | yes | 33 blocks with no runtime deps |
| **Skiper UI** | https://skiper-ui.com | Uncommon form interactions and layouts | Freemium | `npx shadcn add @skiper-ui/…` | Pro unlocks full library |
| **MynaUI** | https://mynaui.com | 70+ elements, 12k+ icon set, Figma/React parity | Free + Pro | copy/paste | Design-systems-first |

## Where each fits Propria

- **Intake search workbench** (`Consignatio/Intake/docs/SEARCH-SURFACE-DIRECTIVE-2026-09-14.md`): drawers, tabs, sidebars, parameter forms, empty/loading states → ReUI or Origin/COSS particles; charts for coverage/freshness → shadcn Charts; result grids → Glide (not from these libraries).
- **Progress board / portal dashboards**: dashboard blocks from Shadcnblocks or shadcn.io; KPI tiles and charts from shadcn Charts.
- **Selection-aware chat / agent panels**: Cult UI agent patterns.
- **Timelines** (Advocatio, Intake provenance): Origin/COSS timeline particle as a starting chrome; engines remain react-calendar-timeline / vis-timeline / Timesketch per the stack reconciliation.
- **Marketing/landing kits** (Tailark, Aceternity, Magic UI, Velora): not needed for internal surfaces; listed for completeness.

## Constraints that still apply

- Propria tokens are the single source (`design-contract/tokens.json` → `tokens.css`; `verify.mjs` drift/contrast). Blocks are re-themed, never the other way round.
- One platform-wide chrome library decision is still open for the Codex design lane (shadcn classic vs Astryx); every library above is shadcn-registry compatible, several also ship Base UI variants, so they remain usable under either answer.
- Copy-paste means we own the code: pin the source version in a provenance comment in each imported block.
