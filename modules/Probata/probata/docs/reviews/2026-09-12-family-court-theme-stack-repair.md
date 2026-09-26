# Family Court theme and stack repair

> _Byline: OpenAI Codex · GPT-5 · 2026-09-12._

## Outcome

The Family Court desktop workbench and its eight MCP App widgets now share one restrained graphite/warm-paper visual contract. Dark mode is driven by the actual application or MCP host, not an unrelated operating-system media query. The desktop no longer presents an empty graphite window while the external case store is unavailable.

The existing product stack was retained because it already matches the owner direction: React 19, TanStack Router/Query/Table/Virtual, Glide Data Grid, Storybook, Vite, Tailwind, and Tauri. The AI surface remains the official Anthropic Claude Agent SDK through the local sidecar boundary. No second AI framework or duplicate state layer was introduced.

## Implemented

- Changed the initial TanStack route from a blocking suspense loader to non-blocking cache prefetch plus an explicit loading/error/retry state.
- Added a six-second sidecar request timeout so an unavailable SurrealDB/plugin store fails closed instead of suspending the entire application indefinitely.
- Added explicit dark/light Glide Data Grid themes aligned to the workbench palette.
- Standardized the typography on the installed CaskaydiaCove Nerd Font family, using the proportional face for reading and the mono face for operational metadata.
- Refined the type system for sustained reading: 14px/1.55 body copy, stronger muted-text contrast, 13-15px component hierarchy, less tiny all-caps metadata, and matching Glide metrics.
- Added one build-time widget theme and one MCP host-context bridge for all eight Family Court widgets.
- Applied initial and live MCP host theme/style/font updates with `applyDocumentTheme`, `applyHostStyleVariables`, `applyHostFonts`, `getHostContext`, and `onhostcontextchanged`.
- Removed legacy OS dark-media-query behavior from generated widgets and made the widget framing contract explicit with `prefersBorder: false`.
- Rebuilt the MCP App bundle, refreshed the active Claude plugin cache, and synchronized the reviewed `dist/server.js` into the Probata Coolify build input.
- Added the existing TanStack progressive-disclosure package to the deliberate Codex `casebible-local` allowlist and enabled it for the next Codex reload.
- Ported the same host-aware theme and readable Nerd Font contract into the canonical Codex-local Family Court package, fixed the generated MCP connection transform so it cannot recurse into itself, and enabled the package in Codex configuration.

## Verification

- Desktop production build: passed (`vite build` and `tsc -b`).
- Desktop sidecar tests: 14 passed.
- Desktop UI tests: 8 passed.
- Desktop Storybook static build: passed.
- Desktop ESLint: passed with two warnings in unchanged files and no errors.
- MCP widget contract tests: 9 passed.
- MCP TypeScript typecheck: passed.
- Codex-local Family Court package: build passed, 20 tests passed, and TypeScript typecheck passed.
- Generated widget connection proof: each of eight widgets calls the shared wrapper, and each injected wrapper still calls the real `app.connect()` exactly once; the recursive form is absent.
- Browser inspection: dark and light themes both rendered; the previously empty initial view now showed navigation, header, loading state, then a bounded case-store-unavailable state with retry.
- Browser computed-style inspection: dark background `rgb(19, 21, 25)` and body font `CaskaydiaCove Nerd Font Propo` were active.
- Readability computed-style inspection: body type was `14px` with `21.7px` line height in both themes; light secondary text resolved to `rgb(77, 86, 98)`.
- Live desktop inspection reported `Claude connected`; the independent case store remained offline and correctly produced the bounded retry state.
- Runtime parity: Claude source/cache/Probata bundles share SHA-256 `946B46F1FE65E32E4615BE7B58F5A6C23B159F072EC7DA569B3690695267EADE`; Codex source and both installed cache copies share `71EDAEB58E77EFEF74906450DECD4AAB19F9F1437FB865B8EE759A901EA2178D`.
- Codex plugin inventory reported `family-court-toolkit-codex@custody-guide-codex-local` installed and enabled at version `2.0.0`.

The complete MCP test suite reported 83 passing and two failures outside the edited UI paths: one dry-run content-store isolation assertion and one missing `skills/family-court-toolkit/references/court-language/EXAMPLES.md` fixture. The targeted widget suite and typecheck passed.

## Preservation and deployment boundary

The replaced cache and deployment bundle were preserved under `to_be_deleted` locations; nothing was permanently deleted. The heavily dirty and diverged Probata worktree was not staged, committed, pushed, or deployed. The synchronized Coolify build input is local source proof only, not a live production deployment receipt.
