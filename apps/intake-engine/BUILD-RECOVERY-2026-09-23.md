# Hosted Intake engine build recovery — 2026-09-23

Byline: Codex · GPT-6 · 2026-09-23

## Planned change — before modification

The tracked tree is clean on `feat/hosted-intake-engine`, except for the existing
untracked `to_be_deleted/` quarantine. This receipt and the two Cargo files are
the only planned tracked changes. No quarantined material will be altered.

`apps/intake-engine` is a standalone Cargo package with its own manifest and
lockfile. Its lockfile is byte-identical to the donor desktop's
`apps/src-tauri/Cargo.lock` (SHA-256
`4A04DDF48E6A9DB5A381D0D3C7199DDA78D360AA29FE5B796469B07D4A163565`).
It still names the donor `xplorer` root and omits the engine's
`deadpool-postgres` dependency. The engine therefore needs a reconciled lock
generated from its own manifest. The donor lock is outside this change.

The engine reuses the donor Rust library. Its Windows-only shell integration
imports `winreg`; the donor manifest supplies `winreg = "0.55"` under the
Windows target, but the engine manifest does not. Add that same target-specific
dependency, then resolve only the engine lockfile. Preserve compatible locked
versions where Cargo permits; inspect every package addition and removal before
building. Use the pinned E: toolchain, cache, target, and temporary directories.

## Checkpoint — before modification

- Repository: `E:/AI_Workspace/Projects/Propria/modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot`.
- Branch: `feat/hosted-intake-engine`; HEAD `34aa2ba7`.
- Existing untracked `to_be_deleted/` belongs to the owner and remains intact.
- Back up the original engine manifest and lockfile under a new, isolated
  `to_be_deleted/2026-09-23-hosted-engine-build-recovery/` directory before edits.
- Verification target: Cargo metadata, locked engine check, narrow engine tests,
  and static Linux/container semantics if this Windows host cannot run Linux.

## Checkpoint — resolved lock and dependency delta

Byline: Codex · GPT-6 · 2026-09-23

Cargo metadata resolved the existing engine manifest after the Windows target
dependency was added. The resulting lock has `intake-engine 0.1.0` as its root,
the two local `tauri-headless` packages, and the hosted PostgreSQL dependencies.
The donor `xplorer 0.99.1` root and Tauri/webview/plugin packages are absent.

Package-version entries changed from 772 to 551: 277 old entries removed and
56 entries added. Of the 56, 49 represent new package names/versions for the
hosted engine; seven replace old versions of `libc`, `libredox`, `rand`,
`rand_core`, `redox_syscall`, `syn`, and `wasi`. The new hosted branches include
Axum, Deadpool/PostgreSQL, metadata parsing, HTTP compression, and their
transitives. Most removals are the desktop Tauri, GTK/WebKit/WebView, plugin,
platform, and build-only branches. Cargo retained compatible shared package
versions wherever its existing lock permitted.

To check the version drift, offline `cargo update --precise` attempts for the
older `libc 0.2.182`, `libredox 0.1.12`, and `redox_syscall 0.7.2` all failed
without changing the lock. `whoami 2.1.3`, brought in by `tokio-postgres`,
requires `libc ^0.2.186` and `libredox ^0.1.17`; the resolved `libredox`
requires `redox_syscall ^0.9.2`. The remaining changed major versions are new
transitive branches, not updates of retained engine dependencies.

The engine lock SHA-256 after resolution is
`1E0B23410228B4F0D0E4EBED267D153FE0C28F934C9C4B41A4644A38ABE09D82`.
The donor lock remains at its original hash. The backups in the isolated
quarantine folder remain intact.

## Verification and limits

Byline: Codex · GPT-6 · 2026-09-23

- Pinned toolchain: Cargo/Rust 1.91.1 from E:. The launcher set E: for
  Cargo home, Rustup home, target, TEMP/TMP, and runtime state.
- `cargo metadata --manifest-path apps/intake-engine/Cargo.toml --format-version 1`:
  passed and reconciled the engine lock.
- `cargo metadata --locked --manifest-path apps/intake-engine/Cargo.toml
  --format-version 1 --no-deps`: passed.
- `cargo check --locked --manifest-path apps/intake-engine/Cargo.toml`:
  passed. It emitted 11 unused-import/dead-code warnings in reused donor code.
- `cargo test --locked --manifest-path apps/intake-engine/Cargo.toml
  --bin intake-engine name_search::`: 14 passed, 0 failed.
- Linux-target `cargo metadata --locked --filter-platform
  x86_64-unknown-linux-gnu` passed. Reverse dependency trees for both locked
  `winreg` versions print no Linux normal dependencies, confirming the direct
  dependency stays Windows-only in the resolved graph.

The Dockerfile copies `apps/intake-engine`, including its engine-owned lock,
and builds from that package directory. This Windows host did not run the
container or compile a Linux binary; Linux runtime behavior and deployment
remain unverified. No authentication, portal, donor manifest/lock, frontend,
SDK, or deployment files were changed.
