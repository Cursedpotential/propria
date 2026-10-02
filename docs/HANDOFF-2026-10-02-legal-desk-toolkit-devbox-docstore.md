# HANDOFF — legal desk + Family Law Toolkit, devbox/Kasm, docs-store memory (2026-10-02)

> _Byline: Claude Code · Opus 5.5 · 2026-10-02_
STATUS: PARTIAL
BUILD_STATUS: UNKNOWN (toolkit mcp-app: typecheck passes, tests 88/92 with the same 4 failures before and after today's change; desk: 5 unrelated failures reported by the builder; nothing else run)

Previous handoff for this session: `docs/HANDOFF-2026-10-01-contextforge-coolify-surreal.md` (its Codex-model, Morph and rename items are still open).

## Verified-live state (do not re-derive)

| Thing | State |
|---|---|
| Legal desk (Advocatio) `legal-workspace` `gvghzivfmctev8dloetfssnj` | API under `https://legal.tilapia-skilift.ts.net/api/legal/...`. Toolkit listing uncapped (was `LIMIT 200`): 323→now 321 references, 193 sources (`7a19caf7`). Builder `df34bd44` (deployment `i252mowmvanp9v11794hh50x`): `/v1/mcp/connections`, `/v1/mcp/tools` (28 tools), `/v1/mcp/invocations`; web `/toolkit/tools` (schema forms, call shown before Run, writes need a "reviewed" tick); factors `/toolkit?table=factor` (12). `search_guide` live call returned 12 results. Writes classed: 9 tools incl. `case_query` (it allows CREATE/UPDATE). |
| Toolkit store `surreal-case` fct/case | 321 references, 193 sources (every source has a file or an official/alternate URL; ICWA and Sullivan v Gray official_url restored), 12 factors, 1 court, 0 case_status (probe removed). No case data yet. Removed rows saved in `modules/Legal-desktop/docs/receipts/2026-10-01-toolkit-store-test-rows-removed.json`. |
| Toolkit content vs originals | Originals: `F:/Users/matts/Downloads/custodyguide_v1complete_20260812` (+ donor copy, + `plugins/_stale/family-court-toolkit-2.1.0-pre-rebuild-20260907`). `verification_ledger.md` and `sources/README.md` restored from 08-12 (were older) and rewritten into their store rows. Everything else is equal or a later edit. The plugin's `content/` is gitignored (no history). |
| Hosted toolkit console `family-court-console` `sokv65ibdq2y8xdaqmd6p4rq` | Store login fixed 01:54 EDT (toolkit 3.2.4 `4443005`, build copy `b322c9b2`, deployment `m10iwrd487j1p0iht0p9y7cy`). Verified through the desk: `case_summary` reads the store; a marked test note was written, read back and purged (0 before, 0 after). |
| Docs store `surreal-docs` `r13ehbwuw3xji8x9mibeypp0` | RocksDB block cache 2 GiB + 2×64 MiB write buffers (`542efd1c`, deployment `mp1md6u4s1713kzyn4tk6crf`). RSS 7.5 GiB → 105 MiB after restart; ovh-files available RAM 7→14 GB. Docstore health ok, store up. Growth should stop near ~2.5 GiB; not yet observed over time. |
| Devbox `pd3xc78ahqkfswq12bpfqgy1` (Kasm agent) | `fae5ff12`: xrdp → 13389, ttyd 1.7.7 on 7681 into `claude` in `~/work`, tailscale sidecar removed, `/root` on the volume, `deploy/devbox/pre_redeploy_check.py` (ran: 501 files rescued to `~/rescued/2026-10-02T053415Z/`, verified). `~/.claude.json` restored from the 09-25 backup; Claude Code not signed in. 09-28 deploy had already destroyed container-layer content (two agent work dirs ~1.1 GB, `.npm`, `.duckdb`, `portal-work`) — no copy exists. Owner triggered the force redeploy 01:36 EDT; agent watching. |

## Findings / work done

1. **Desk hid a third of the toolkit** (list cap). Fixed, live.
2. **Toolkit regressions vs 08-12 originals**: two documents restored; two missing official links restored; agent probe row + 2 test fixtures removed; loader skips `toolkit/tests/` (`propria-plugins` `9541682`, toolkit 3.2.3).
3. **Desk now runs toolkit tools and shows factors** (builder). Store-touching tools fail until the console auth fix ships.
4. **Docs store memory**: RocksDB sized from host RAM; capped by env in compose.
5. **Chat model decision (owner 01:14–01:17)**: B + C. General chat on kimi-k3 (NIM via Portkey); "consult/escalate to Claude" through the official, unmodified Claude Code on the devbox, signed in with the owner's own Max account (Anthropic legal page allows the unmodified binary with the user's own subscription, including hosted; the desk never holds the token). Listener reserved as a second service in `deploy/devbox.yaml` (`svc:devbox-claude`), not built.
6. **Rules recorded today** (local memory + Docstore memory): Coolify auto-deploy is OFF on purpose (owner 09-20), never "fix" it; toolkit data is live — test writes only as marked ephemeral rows purged at once (owner 01:07).

## UNRESOLVED (mandatory)

- Devbox redeploy in flight; owner must sign in to Claude Code there afterwards (`claude`, then `/login`).
- Kasm Workspaces install/registration/exposure (brief steps 2–6) not started; `deploy/kasm/` being written.
- Desk chat: wire kimi-k3 + toolkit skills/agents/tools; build the devbox Claude listener + "Ask Claude".
- Legal desk case facts (owner picked A: parties, children, court, judge, case number) — not loaded yet; source = memory note `case-identity-salem-v-kinzel` and its source documents.
- Probata Workbench: redesign proposal (N-07) and `/api/tools` → ContextForge — proposed, not started (owner was mid-question; "Desk" there is the Workbench's own `/` page, not the legal desk).
- coolify-write `get_application` does not redact `VNC_PW` in rendered compose — plugin bug to fix.
- Codex copy of family-court-toolkit stale (a Codex process held the folder); run `python3 tools/sync_installs.py` in `E:/AI_Workspace/plugins`.
- Toolkit tests: 4 failures pre-exist (same count without today's change); desk tests: 5 unrelated failures (Motion writer labels, pip-less wheel build).

## Pending owner decisions

- Auto-guard before every devbox redeploy — A (default): coolify-write deploy guard runs the check and refuses unsafe deploys (covers tool deploys only); B: Coolify pre_deployment_command (unproven on 4.1.2). Recommendation A now.
- Allow rule for `coolify-write-deploy-application` in `~/.claude/settings.json` (script given to the owner; MCP rules cannot be limited to one app uuid).

## Next steps (work in order)

1. Follow the devbox redeploy to healthy; owner signs in; verify ttyd and `claude auth status`.
3. Load legal-desk case facts (option A) from the case-identity record — real data, read back.
4. Desk chat B (kimi-k3 + skills/agents/tools), then C (devbox listener + Ask Claude).
5. Kasm steps 2–6 per `modules/Probata/probata/docs/planning/2026-09-27-TODO.md` brief.
6. Workbench N-07 proposal and `/api/tools`.

## Owner working-style contract

- Structured replies: bullets, labeled blocks, white space, answer-first; plain words, no jargon.
- Confirm before changes; never hard-delete (quarantine); byline every artifact; verify live before claiming done; never call something complete before checking it.
