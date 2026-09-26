# Intake desktop runtime boundary

The Intake dev build uses `tauri.intake.conf.json`, not the upstream app profile.
The parent launcher owns the frontend/build environment and must supply:

- `XPLORER_INTAKE_MODE=1`
- `XPLORER_RUNTIME_DIR=E:\AI_Workspace\.intake-dev\runtime`
- Isolated `TEMP`, `TMP`, Cargo/npm/toolchain/cache directories on E:.
- Frontend at `http://127.0.0.1:5176`.

From the `apps` directory the Tauri config argument is
`--config src-tauri/tauri.intake.conf.json`. Do not start the marketplace.

The native host validates the runtime root before Tauri startup, requires the
matching `com.propria.intake.dev` identifier, and holds an OS file lock for its
lifetime. A second process using the same runtime fails rather than sharing it.
The lock marker is retained; the operating system releases its lock on exit.

The main window is created explicitly with the absolute runtime `webview` data
directory. JSON `dataDirectory` alone is not used because the installed Tauri
schema describes that property as relative to the OS application data directory.
App storage, legacy dirs-based storage, default chat files and credential service
names are isolated. Windows known-folder environment overrides are not relied on.

Legacy Xplorer search/index commands return an explicit disabled/not-connected
error. Engine startup and rebuild paths also guard against automatic indexing.
This does not disable filesystem listing or nonrecursive visible-folder watches,
and does not replace the future CocoIndex/remote-filesystem integration.

The old destructive extension migration and automatic DevTools opening are
skipped in Intake mode. Native agent chat rejects its local-model fallback.
The separate frontend/AI workstream owns selection manifests and remote chat.

## Verification receipt — 2026-09-11

- JSON parsed; distinct identifier, port and manual window creation confirmed.
- 23 public legacy search commands each have a disabled guard.
- Direct isolated Rustfmt parsed/formatted the runtime helper, main and credential
  module successfully; no Rustup, downloads or app startup were triggered.
- Native diff whitespace check passed (preexisting CLAUDE whitespace excluded).
- Two pure runtime-path unit tests added; not executed by this workstream.
- Full native typecheck, runtime directory observation, duplicate-instance test,
  WebView startup and file-selection/chat smoke test remain parent build gates.

This is an application-managed state guarantee, not a promise that Windows or
WebView system services never perform incidental writes on the system drive.
Existing nested junctions inside an already-used runtime require review; the
startup check validates the root and each top-level managed directory only.

## Filesystem search HTTP bridge

`filesystem_index_search(query, mode, limit)` calls the native-launcher-only
`INTAKE_FILESYSTEM_API_URL` base plus `/filesystem/search`. Optional
`INTAKE_FILESYSTEM_API_TOKEN` stays native. No endpoint or credential is accepted
from the renderer. Missing configuration is a structured `not_configured` error,
not an empty successful search. The response retains coverage and target-vector
identity from Weaviate's filesystem projection, separate from evidence search.

Limits: two concurrent requests, 25-second total/5-second connect timeout, no
redirects, 2 MiB streamed response cap, query length 1–4096, result limit 1–100.
Three URL/input tests added; execution remains pending native toolchain recovery.
Rustfmt parsed the bridge successfully. No live search request was sent here.

## Portkey discovery receipt

Read-only source lookup found the separate Probata repository's
`deploy/docker/gateway/portkey/README.md` and `configs/chat.json` plus
`configs/graphiti-llm.json`. README records the stateless gateway at
`http://100.72.169.40:8787`, using an inline `x-portkey-config` header, not a
verified named model alias. Current JSON differs from the old README's models;
therefore do not treat the historical route description as current deployment.

A bounded GET to `/health` on 2026-09-11 returned HTTP 404: HTTP listener reachable,
not a model/health success. Provider key presence (not values) was verified in
`C:\Users\matts\.secrets\probata.env`: NVIDIA_API_KEY and OLLAMA_API_KEY. No
credentials were printed/copied/changed and no model request was issued.
The parent/provider lane must choose and live-test the actual Portkey route.

## Weaviate readiness receipt — 2026-09-11

Probata's `docs/plans/WEAVIATE-NATIVE-EVIDENCE-CUTOVER-RUNBOOK-2026-08-18.md`
identifies the two existing endpoints on ovh-files. Bounded anonymous GET probes:

| Endpoint | `/v1/.well-known/ready` | `/v1/meta` version |
|---|---|---|
| `http://100.91.190.107:8081` | 200 | 200; 1.38.7 |
| `http://100.91.190.107:8082` | 200 | 200; 1.38.7 |

No authentication was sent. This proves access to these readiness/meta routes,
not authorization for collection reads/writes. Tracked deploy configs enable
anonymous access and `DEFAULT_VECTORIZER_MODULE: none`. No WEAVIATE-prefixed
variables were found in either inspected local probata/legacy platform env file.

Important unresolved ownership: Probata's `docs/MASTER-TODO-2026-09-09.md` T-007
and T-030 still hold the physical-instance/cutover choice. Intake needs its own
explicit collection, not an existing evidence collection. These probes did not
inspect schemas/objects, select a canonical instance, create a collection, alter
aliases, or deploy anything. Parent must resolve the physical target before writes.

## Launcher review receipt — 2026-09-11

`scripts/start-intake.ps1` requires PowerShell 7.4+ (host verified 7.6.5) and uses
the documented Start-Process `-Environment` override to remove credential-like
variables from the Vite child without changing the native host's environment.
Provider-key parsing accepts quoted values and trailing comments; three synthetic
parser fixtures passed without touching credentials. All three Tauri configs were
checked: no `beforeDevCommand` currently exists. Launcher now rejects a future
conflicting command instead of spawning duplicate Vite instances.

Launcher validates local node/CLI dependencies, requires loopback and absolute
Vite-path process ownership, waits up to 15 seconds for Vite readiness, and checks
the newly started process still owns the listener. Unrecognized existing processes
(including relative-path commands whose working directory cannot be proven) are
left untouched. Logs stay on E:. Native launch is not attempted on readiness failure.

`intake-env.ps1` requires Windows/E: checkout, checks the pinned Rust executables
and E: SDK headers/resource compiler, and reasserts E: temporary paths after the
Visual Studio environment setup. Both scripts parsed with zero PowerShell errors;
all three checked SDK component paths exist. No launcher/build/process stop was
executed during this script review.

## Per-pane selection receipt — 2026-09-11

Intake mode now uses a parent-owned keyed selection store: each pane has its own
selected paths and preview file, scoped to its active tab/current directory.
Changing focus between panes does not clear or overwrite either selection. The
existing toolbar/chat/sidebar projection follows the active pane. Actual tab or
directory navigation retires only the affected pane's selection; closing a pane
retires its state. Delayed setters from retired scopes cannot re-add stale paths.

`SplitContainer` passes pane-bound setters directly, avoiding save/restore effects
and the inactive-pane first-click race. `EditorGroupPane` skips the old global
clear effect only for these controlled Intake panes. Its shift-click anchor resets
on path/tab navigation, not focus. Non-Intake mode retains legacy global selection.

Verification: nine scoped hook tests passed, TypeScript `--noEmit` passed, seven
affected source/test files passed ESLint with zero warnings. Tests cover focus
switching, independent groups/previews, inactive-pane setters, navigation, closed
groups, stale events, layout-only changes, tab scope and updater isolation.
Native UI range-selection/clipboard/chat interaction still requires desktop smoke
testing; no native build or process was started by this selection workstream.

## Updater isolation receipt — 2026-09-11

`useUpdater` now schedules no startup/periodic checks in Intake mode. Manual check
and install entrypoints return before calling the upstream updater plugin;
download/install and application relaunch cannot run through this hook in Intake.
The non-Intake updater behavior remains unchanged. Four scoped tests passed,
including absent Intake timers/network plugin calls and retained legacy checks
and installation. The native owner also guarded plugin registration in `main.rs`
to block direct plugin invocation independently. That native gate takes effect
after rebuild/next launch; the already running desktop uses the earlier binary.
No external updater request or native process/build was started by this change.
