<!-- Byline: Codex · GPT-5 · 2026-08-27. -->
@AGENTS.md
@AGENT_MEMORY.md
@..\AGENTS.md
@..\AGENT_MEMORY.md
@..\..\AGENTS.md
@..\..\AGENT_MEMORY.md
activate /karpathy-guidelines   /think:tm-graph-thinking /think:tm-thinking-systems /hyperfocus:hyperfocus
/think:tm-thinking-socratic  THIS TYPE PF THOUGHT PROCESS NEEDS TO GUIDE YOUR DESIGN AND DECISION MAKING IN GENERAL, ITS DELIVERABLES ARE NECCISARY BUT APPLICATION OF ITS PRINCIPALS ARE!
consider a  /think:workflows IF ONE FITS THE TASK YOU MUST LOAD IT

## Paths (verified live 2026-09-10)

> _Byline: Claude Code · Opus 5 (1M) · 2026-09-10 — owner order 06:47 "add path guidance to the Claude MD file". Placed here because this file loads for sessions started in `Probata` AND inside `Probata\probata`._

**Local (desktop)**

- **Probata repo, git root:** `E:\AI_Workspace\Projects\Propria\Probata\probata`. Commit only from here, staging by explicit path. Other sessions share this index and stage hundreds of their own files.
- ~~**Start sessions in:** `E:\AI_Workspace\Projects\Propria\Probata`. Its auto-memory store is `C:\Users\matts\.claude\projects\E--AI-Workspace-Projects-Propria-Probata\memory` (191 memories, moved here 2026-09-10). A session started inside `Probata\probata` gets an empty store.~~ **Corrected 2026-09-10 (Claude Code · Opus 5): the owner moved the parent's memory folders into the repo.**
- **Start sessions in:** `E:\AI_Workspace\Projects\Propria\Probata\probata`. Its auto-memory store is `C:\Users\matts\.claude\projects\E--AI-Workspace-Projects-Propria-Probata-probata\memory` (all 191 memories copied there 2026-09-10; the old `…-Propria-Probata` store is left in place, unwritten). `.claude\` and `.remember\` live inside the repo; the merge of the parent copies is logged in `probata\to_be_deleted\2026-09-10-dotfolder-merge\merge-log.txt`.
- **memsearch (shared agent memory):** ONE folder for every agent and project, `C:\Users\matts\.memsearch\memory`, and ONE Milvus collection, `agent_session_memory_nemotron3`, pinned by `C:\Users\matts\.memsearch\.collection`. Claude gets it from `MEMSEARCH_DIR` in `~\.claude\settings.json`; Codex from `~\.codex\hooks\memsearch_codex_hook.py`. Codex's own memory store is imported under `memory\codex\<project>\`. Per-project `.memsearch\` folders are retired (left in place, no longer written). Both plugin copies carry a local patch that honors `.collection`; re-apply it after a memsearch plugin update.
- **Worktrees:** `E:\AI_Workspace\Projects\Propria\_worktrees`. New Propria-owned linked worktrees belong here; relocate existing linked worktrees only with `git worktree move` after their owner is paused and state is captured.
- **Running TODO:** `probata\docs\planning\<date>-TODO.md`. Current file is `2026-09-08-TODO.md`.
- **Handoffs:** `probata\docs\handoffs\HANDOFF-<date>-<topic>.md`.
- **Secrets:** `C:\Users\matts\.secrets`. Parse with a regex and never `source` these files.
- **rclone:** the binary is the scoop shim `C:\Users\matts\scoop\shims\rclone.exe`, and the config is `C:\Users\matts\scoop\apps\rclone\current\rclone.conf`.

**VPS (tailnet only)**

- **ovh-files** is `100.91.190.107`. **ovh-app** is `100.72.169.40`. Connect with `ssh -i ~/.ssh/ovh root@<ip>`.
- **Host roots:** `/data/probata/volumes`, `/data/probata/config`, `/data/probata/secrets`, `/data/probata/tsnet`.
- **Coolify render dir:** `/data/coolify/applications/<uuid>/` holds only `docker-compose.yaml` and `.env`, with no repo checkout. A relative bind of a repo file becomes an empty directory. Mount config files from absolute host paths under `/data/probata/config/<app>/`.
- **Service URLs:** `https://<name>.tilapia-skilift.ts.net`, served by each host's own tailscaled as a Tailscale Service. Adding one takes three steps: register, `tailscale serve --service`, approve the host.

**Path syntax in the Bash tool (Git-Bash)**

- The tool collapses `\\` to `\` before bash runs. A Windows path inside a Python or JSON string then turns `\t` into a tab and `\r` into a carriage return. This corrupted a handoff and a memory note on 2026-09-10.
- In Bash, write Windows paths with forward slashes (`E:/AI_Workspace/...`). In scripts, build a backslash with `chr(92)`. For file content that contains backslashes, use the Write or Edit tool.
- Prefix commands that pass `/unix/paths` to `ssh` or `docker` with `MSYS_NO_PATHCONV=1`.
