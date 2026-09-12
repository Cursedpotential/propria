# SDA-04 Family Court surface design adoption receipt

> _Byline: OpenAI Codex · GPT-5 · 2026-09-12._

## Outcome

Family Court Workbench now consumes a repository-owned, pinned copy of the Propria
Carbon-Linen-Seal surface contract. Dark and light modes use the same semantic roles as the other
Propria workspaces without changing this product into advocatio or treating Probata's experience
tier as a theme.

The application remains a distinct **Family Court Console** with the explicit context:

- Scope: local case workspace.
- Authority: planning aid.
- AI surface: Claude Agent SDK chat constrained to the Family Court console's existing tool policy.

No deployment or shared infrastructure change was made in SDA-04.

## Pinned design package

- Package: `@propria/design-contract` `1.0.0`.
- Source commit: `c6da141`.
- Source subtree: `resources/design`.
- Source subtree tree ID: `9dcb135efedfb05b423e8e7e4b64529b09b556a9`.
- Local copy: `vendor/propria-design-contract/1.0.0/`.
- Pin receipt: `vendor/propria-design-contract/PIN.md`.
- Runtime rule: application CSS imports only the vendored package, never the live router-level
  `resources/design` path.
- Drift rule: `tokens.json` is authoritative; `tokens.css` is generated and must not be hand-edited.

The initial archive extraction converted the generated CSS to CRLF on Windows. Running the
package's canonical builder restored LF and produced the exact pinned Git blob
`728ce008b8f91191f7e9cb160d41813bad6b113d`; the strict verifier then passed.

## Applied surface contract

- Dark canvas/surface/action/focus: `#161a18`, `#202622`, `#f27479`, `#f0b45a`.
- Light canvas/surface/action/focus: `#f3f0e8`, `#fffdf8`, `#9f303b`, `#7d5200`.
- UI type: Instrument Sans with Segoe UI fallback.
- Data/code type: IBM Plex Mono with Cascadia Code fallback.
- Family Court aliases map to shared semantic tokens rather than maintaining a second palette.
- Seal red is reserved for primary action/selection; brass is the keyboard focus signal; teal is
  informational/status only.
- Buttons, inputs, badges, cards, navigation, the Glide docket grid, unavailable/retry messaging,
  and Storybook stories now resolve through the shared contract.
- The theme provider synchronizes legacy `data-theme` with `data-pr-theme` before canvas-based
  consumers read computed variables.
- A compact context strip states product, scope, and authority on the working surface.
- The shell reflows below 720 CSS pixels: navigation becomes a horizontally scrollable top rail,
  the header wraps, and the document itself does not gain horizontal overflow.

## Stack alignment

No framework migration was necessary; the installed application stack is already the requested
React/TanStack/Storybook/Glide family:

- React and React DOM `19.2.3`.
- TanStack Query `5.102.8`, Router `1.170.33`, Table `8.21.3`, Virtual `3.14.11`.
- Storybook and `@storybook/react-vite` `10.6.0`.
- Glide Data Grid `6.0.3`.
- Claude Agent SDK `0.3.263`.

SDA-04 added design build/verification scripts and corrected the existing Tailwind mono-font bridge;
it did not introduce a competing component or state framework.

## Verification evidence

All checks were run from `projects/family-court-workbench` on 2026-09-12.

| Check | Result |
|---|---|
| `npm.cmd run design:build` | Passed; generated `tokens.css` from `tokens.json`. |
| `npm.cmd run design:verify` | Passed exact token/density parity, adapters, sample integration, CSS policy, and core contrast pairs. |
| `npm.cmd run lint` | Passed with zero errors; two existing warnings remain (theme-provider fast-refresh export shape and TanStack Table compiler compatibility). |
| `npm.cmd run test:ui -- --maxWorkers=1` | Passed: 5 files, 13 tests. The first parallel run timed out before any worker started; the isolated rerun passed. |
| `npm.cmd run test:sidecar` | Passed: 14 tests. |
| `npm.cmd run build` | Passed Vite production build and `tsc -b`. |
| `npm.cmd run build-storybook` | Passed; Storybook emitted its existing deprecation and large-chunk warnings. |
| Browser dark mode | Passed at `http://127.0.0.1:5183/`; computed values matched the dark contract and no console error/warning was captured. |
| Browser light mode | Passed; computed values matched the light contract and both theme attributes/classes changed together. |
| Selected navigation contrast | Passed after browser-led correction: 13.92:1 in light mode and 11.71:1 in dark mode. |
| Keyboard/focus | Passed with keyboard Tab; the focused retry control showed the brass solid focus outline. |
| 200% equivalent reflow | Passed at a 640×360 CSS viewport (half the 1280×720 baseline): top-rail navigation, wrapped header, and `scrollWidth === clientWidth` for the document. |
| Unavailable/retry | Passed: after the bounded 6-second store timeout the UI announced “Case store unavailable,” explained the timeout, exposed “Try again,” and returned to the connecting state when invoked. |
| AI panel | Passed visual/semantic inspection in dark mode; the Claude panel inherited the contract and retained its Family Court-only authority text. No prompt was transmitted. |

## External availability and auth inventory

The final owner ingress contract has two separate lanes:

1. Enrolled owner systems keep their existing unrestricted Tailscale routes unchanged.
2. Outside browsers use ordinary public HTTPS through the existing Coolify/Traefik reverse proxy,
   then Authentik, and reach only an authorized human UI.

Current product facts:

- No public hostname is assigned in this repository.
- No Coolify, Traefik, Authentik, OIDC, forward-auth, or public deployment configuration exists in
  this repository.
- Vite development is intentionally bound to `127.0.0.1:5183`.
- The Fastify sidecar is intentionally bound to `127.0.0.1:4177` for browser development and to a
  loopback ephemeral port under Tauri.
- The sidecar holds the Claude credential boundary and accesses the local SurrealDB service; it is
  an internal API and must not be exposed through public HTTPS.
- The current browser client assumes a same-machine loopback sidecar. A public browser cannot use
  that seam safely or correctly.

Remaining live gates, owned outside SDA-04:

- Assign the Family Court human-UI hostname and portal entry.
- Provide a deployable human-UI runtime behind direct Coolify/Traefik HTTPS.
- Apply and verify the shared Authentik browser session/authorization architecture at that edge.
- Design a server-side, authenticated UI-to-service seam that keeps the sidecar, Claude token,
  SurrealDB, workers, tool runtime, and gateway private; never point a public browser at loopback or
  publish those services.
- Preserve and separately verify the current Tailscale owner route without forcing it through
  Authentik or narrowing it.
- Run authenticated external-browser, unauthorized-access, session-expiry, direct-origin bypass,
  and live data-path tests after the shared ingress work exists.

Until those gates are implemented and proven, this receipt establishes local/browser design proof,
not external availability or production readiness.
