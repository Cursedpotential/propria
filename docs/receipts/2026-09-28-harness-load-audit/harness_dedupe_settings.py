"""Harness dedupe: backup-first edits to Claude Code and Codex config (2026-09-28 harness load audit).

Byline: Claude Code · Opus 5.5 · 2026-09-28
Report: E:/AI_Workspace/Projects/Propria/docs/receipts/2026-09-28-harness-load-audit.md

Default run applies only the mechanical fixes (M1-M3). `--with-defaults` also applies the
owner-decision defaults (D1-D3) named in the report. `--dry-run` prints the plan and writes nothing.
Every file is copied to `<file>.bak-20260928-harness-dedupe` before it is written. Idempotent.
Restart Claude Code / Codex afterwards: settings and hooks load at session start.
"""
import json
import os
import shutil
import sys

HOME = os.path.expanduser("~")
CLAUDE = os.path.join(HOME, ".claude")
SETTINGS = os.path.join(CLAUDE, "settings.json")
INSTALLED = os.path.join(CLAUDE, "plugins", "installed_plugins.json")
CODEX_HOOKS = os.path.join(HOME, ".codex", "hooks.json")
CODEX_HOOK_DIR = os.path.join(HOME, ".codex", "hooks")
CLAUDE_HOOK_DIR = os.path.join(CLAUDE, "hooks")
QUARANTINE = os.path.join(HOME, ".codex", "to_be_deleted", "2026-09-28-harness-dedupe", "hooks")
SUFFIX = ".bak-20260928-harness-dedupe"

DRY = "--dry-run" in sys.argv
DEFAULTS = "--with-defaults" in sys.argv
changes = []


def load(path):
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def save(path, data):
    if DRY:
        return
    if not os.path.exists(path + SUFFIX):
        shutil.copy2(path, path + SUFFIX)
    tmp = path + ".tmp-harness-dedupe"
    with open(tmp, "w", encoding="utf-8", newline="\n") as f:
        json.dump(data, f, indent=2, ensure_ascii=False)
        f.write("\n")
    os.replace(tmp, path)


# ---- settings.json -------------------------------------------------------------------
s = load(SETTINGS)
ep = s.setdefault("enabledPlugins", {})
before = json.dumps(ep, sort_keys=True)

# M1: enabled but never installed, and its marketplace is not registered -> dead entry.
if "context-mode@context-mode" in ep:
    del ep["context-mode@context-mode"]
    changes.append("M1 settings.json: removed enabledPlugins['context-mode@context-mode'] (not installed, marketplace unknown)")

# M2: installed ffmpeg-master 3.6.0 is the old monolith; the five split sub-plugins it duplicates are
# enabled. Upstream 4.0.0 turned ffmpeg-master into an empty meta-bundle.
if ep.get("ffmpeg-master@claude-plugin-marketplace") is not False:
    ep["ffmpeg-master@claude-plugin-marketplace"] = False
    changes.append("M2 settings.json: ffmpeg-master@claude-plugin-marketplace -> false (duplicates ffmpeg-core/effects/platforms/python/social-video)")

if DEFAULTS:
    # D1: same-named plugin from the old anthropics/claude-code marketplace; the official one stays.
    if ep.get("code-review@claude-code-plugins") is not False:
        ep["code-review@claude-code-plugins"] = False
        changes.append("D1 settings.json: code-review@claude-code-plugins -> false (keep code-review@claude-plugins-official)")
    # D2: ~/.claude/skills/mineru is both a skill (root SKILL.md, richer) and a plugin (skills/mineru).
    if ep.get("mineru@skills-dir") is not False:
        ep["mineru@skills-dir"] = False
        changes.append("D2 settings.json: mineru@skills-dir -> false (keep the root mineru skill)")

if json.dumps(ep, sort_keys=True) != before:
    save(SETTINGS, s)

# ---- installed_plugins.json ------------------------------------------------------------
# M3: drop records of disabled installs whose install folder or project folder no longer exists.
inst = load(INSTALLED)
dropped = []
for key in list(inst.get("plugins", {})):
    keep = []
    for rec in inst["plugins"][key]:
        ip = rec.get("installPath", "")
        pp = rec.get("projectPath")
        gone = (ip and not os.path.isdir(ip)) or (pp and not os.path.isdir(pp))
        if gone and ep.get(key) is not True:
            dropped.append(f"{key} [{rec.get('scope')}] {pp or ip}")
        else:
            keep.append(rec)
    if keep:
        inst["plugins"][key] = keep
    else:
        del inst["plugins"][key]
if dropped:
    save(INSTALLED, inst)
    changes += [f"M3 installed_plugins.json: dropped stale record {d}" for d in dropped]

# ---- Codex hooks.json ------------------------------------------------------------------
if DEFAULTS and os.path.exists(CODEX_HOOKS):
    # D3: three compaction hooks are byte-identical copies of the Claude Code hooks; point Codex at
    # the primary copies in ~/.claude/hooks and quarantine the Codex copies.
    names = ["precompact_handoff.py", "postcompact_summary.py", "sessionstart_compact_handoff.py"]
    hk = load(CODEX_HOOKS)
    text = json.dumps(hk)
    new = text
    for n in names:
        src = os.path.join(CODEX_HOOK_DIR, n)
        dst = os.path.join(CLAUDE_HOOK_DIR, n)
        if os.path.exists(dst):
            new = new.replace(json.dumps(src)[1:-1], json.dumps(dst)[1:-1])
    if new != text:
        save(CODEX_HOOKS, json.loads(new))
        changes.append("D3 codex hooks.json: compaction hooks now run the ~/.claude/hooks copies")
        for n in names:
            src = os.path.join(CODEX_HOOK_DIR, n)
            if os.path.exists(src):
                if not DRY:
                    os.makedirs(QUARANTINE, exist_ok=True)
                    shutil.move(src, os.path.join(QUARANTINE, n))
                changes.append(f"D3 moved {src} -> {QUARANTINE}")

print(("DRY RUN - nothing written\n" if DRY else "") + ("\n".join(changes) if changes else "nothing to change (already applied)"))
if changes and not DRY:
    print(f"\nBackups: *{SUFFIX} beside each file. Restart Claude Code and Codex to load the change.")
