# Byline: Claude Code · Opus 5.5 · 2026-09-28 (harness load audit; read-only)
"""Harness load inventory: collect every skill/command/agent each harness can load, with origin and content hash.

Writes inventory.json next to this file. Read-only.
"""
import glob
import hashlib
import json
import os
import re
import tomllib

H = os.path.expanduser("~").replace("\\", "/")
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "inventory.json")
rows = []


def fm(path):
    try:
        t = open(path, encoding="utf-8", errors="replace").read()
    except OSError as e:
        return {"_err": str(e)}, ""
    m = re.match(r"^\ufeff?---\s*\n(.*?)\n---", t, re.S)
    meta = {}
    if not m:
        meta["_nofm"] = True
    else:
        for line in m.group(1).splitlines():
            mm = re.match(r"^([A-Za-z_-]+):\s*(.*)$", line)
            if mm:
                meta[mm.group(1)] = mm.group(2).strip().strip("'\"")
    return meta, t


def h(t):
    return hashlib.sha256(t.replace("\r\n", "\n").encode("utf-8")).hexdigest()[:16]


def add(harness, kind, name, origin, path, ns=""):
    meta, t = fm(path) if path.endswith(".md") else ({}, open(path, encoding="utf-8", errors="replace").read())
    rows.append({
        "harness": harness, "kind": kind, "name": name, "ns": ns, "origin": origin,
        "path": path.replace("\\", "/"), "hash": h(t) if t else "",
        "fm_name": meta.get("name", ""), "nofm": bool(meta.get("_nofm")),
        "has_desc": bool(meta.get("description")),
        "real": os.path.realpath(path).replace("\\", "/"),
    })


def scan_skills(harness, root, origin, ns="", recursive=False):
    if not os.path.isdir(root):
        return
    pat = f"{root}/**/SKILL.md" if recursive else f"{root}/*/SKILL.md"
    for p in glob.glob(pat, recursive=recursive):
        p = p.replace("\\", "/")
        if "/node_modules/" in p or "/.git/" in p:
            continue
        name = os.path.basename(os.path.dirname(p))
        add(harness, "skill", name, origin, p, ns)


def scan_md(harness, kind, root, origin, ns="", recursive=True):
    if not os.path.isdir(root):
        return
    for p in glob.glob(f"{root}/**/*.md" if recursive else f"{root}/*.md", recursive=recursive):
        p = p.replace("\\", "/")
        rel = os.path.relpath(p, root).replace("\\", "/")[:-3]
        if os.path.basename(p).upper() in ("README.MD",):
            continue
        add(harness, kind, rel.replace("/", ":"), origin, p, ns)


def plugin_components(harness, base, ns, origin):
    pj = {}
    for cand in (f"{base}/.claude-plugin/plugin.json", f"{base}/.codex-plugin/plugin.json"):
        if os.path.exists(cand):
            try:
                pj = json.load(open(cand, encoding="utf-8"))
            except Exception as e:
                rows.append({"harness": harness, "kind": "error", "name": ns, "origin": origin, "path": cand, "err": str(e)})
            break

    def paths(key, default):
        v = pj.get(key)
        out = [f"{base}/{default}"]
        if isinstance(v, str):
            out.append(f"{base}/{v.lstrip('./')}")
        elif isinstance(v, list):
            out += [f"{base}/{x.lstrip('./')}" for x in v if isinstance(x, str)]
        return out

    for d in paths("skills", "skills"):
        if d.endswith("SKILL.md") and os.path.exists(d):
            add(harness, "skill", os.path.basename(os.path.dirname(d)), origin, d, ns)
        elif os.path.exists(f"{d}/SKILL.md"):
            add(harness, "skill", os.path.basename(d), origin, f"{d}/SKILL.md", ns)
        else:
            scan_skills(harness, d, origin, ns)
    for d in paths("commands", "commands"):
        if d.endswith(".md") and os.path.exists(d):
            add(harness, "command", os.path.basename(d)[:-3], origin, d, ns)
        else:
            scan_md(harness, "command", d, origin, ns)
    for d in paths("agents", "agents"):
        if d.endswith(".md") and os.path.exists(d):
            add(harness, "agent", os.path.basename(d)[:-3], origin, d, ns)
        else:
            scan_md(harness, "agent", d, origin, ns, recursive=False)
    hk = f"{base}/hooks/hooks.json"
    if os.path.exists(hk) or pj.get("hooks"):
        rows.append({"harness": harness, "kind": "hooks", "name": ns, "ns": ns, "origin": origin, "path": hk if os.path.exists(hk) else "plugin.json:hooks"})
    mcp = f"{base}/.mcp.json"
    if os.path.exists(mcp) or pj.get("mcpServers"):
        try:
            servers = list(json.load(open(mcp, encoding="utf-8")).get("mcpServers", {})) if os.path.exists(mcp) else list(pj["mcpServers"]) if isinstance(pj.get("mcpServers"), dict) else ["(ref)"]
        except Exception:
            servers = ["(unparsed)"]
        for s in servers:
            rows.append({"harness": harness, "kind": "mcp", "name": s, "ns": ns, "origin": origin, "path": mcp})


# ---------------- Claude Code ----------------
C = f"{H}/.claude"
settings = json.load(open(f"{C}/settings.json", encoding="utf-8"))
ep = settings.get("enabledPlugins", {})
scan_skills("claude", f"{C}/skills", "user")
scan_md("claude", "command", f"{C}/commands", "user")
scan_md("claude", "agent", f"{C}/agents", "user", recursive=False)
inst = json.load(open(f"{C}/plugins/installed_plugins.json", encoding="utf-8"))["plugins"]
missing_enabled = []
for key, val in ep.items():
    if not val:
        continue
    if key.endswith("@synced"):
        continue
    entries = [e for e in inst.get(key, []) if e.get("scope") == "user"]
    if not entries:
        missing_enabled.append(key)
        continue
    base = entries[0]["installPath"].replace("\\", "/")
    if not os.path.isdir(base):
        missing_enabled.append(key + " (installPath missing)")
        continue
    plugin_components("claude", base, key.split("@")[0], f"plugin:{key}")
SYN = glob.glob(f"{C}/plugins/synced/*/manifest.json")
synced_plugins = []
for man in SYN:
    b = os.path.dirname(man).replace("\\", "/")
    for p in json.load(open(man, encoding="utf-8"))["plugins"]:
        dirs = [d for d in os.listdir(b) if (d == p["name"] or d.startswith(p["name"] + "~")) and os.path.isdir(f"{b}/{d}")]
        disabled = ep.get(f"{p['name']}@synced") is False
        synced_plugins.append({"name": p["name"], "marketplace": p["marketplaceName"], "disabled_flag": disabled, "dirs": dirs})
        for d in dirs:
            plugin_components("claude", f"{b}/{d}", p["name"], "synced-plugin" + (":DISABLED" if disabled else ""))
for man in glob.glob(f"{C}/skills/synced/*/manifest.json"):
    b = os.path.dirname(man).replace("\\", "/")
    scan_skills("claude", b, "synced-skill", "anthropic-skills")
# projects scanned for Propria cwd chain
for proj in ("E:/AI_Workspace", "E:/AI_Workspace/Projects", "E:/AI_Workspace/Projects/Propria"):
    scan_skills("claude", f"{proj}/.claude/skills", f"project:{proj}")
    scan_md("claude", "command", f"{proj}/.claude/commands", f"project:{proj}")
    scan_md("claude", "agent", f"{proj}/.claude/agents", f"project:{proj}", recursive=False)

# ---------------- Codex ----------------
X = f"{H}/.codex"
cfg = tomllib.load(open(f"{X}/config.toml", "rb"))
scan_skills("codex", f"{X}/skills", "codex-user")
scan_skills("codex", f"{X}/skills/.system", "codex-system")
scan_skills("codex", f"{H}/.agents/skills", "agents-user")
scan_skills("codex", "E:/AI_Workspace/.agents/skills", "agents-repo:E:/AI_Workspace")
scan_md("codex", "prompt", f"{X}/prompts", "codex-prompts", recursive=False)
codex_missing = []
for key, v in cfg.get("plugins", {}).items():
    if not v.get("enabled"):
        continue
    name, mk = key.split("@", 1)
    vers = sorted(glob.glob(f"{X}/plugins/cache/{mk}/{name}/*/"), key=os.path.getmtime)
    if not vers:
        codex_missing.append(key)
        continue
    plugin_components("codex", vers[-1].rstrip("/\\").replace("\\", "/"), name, f"plugin:{key}")
dead_skill_cfg = []
for ent in cfg.get("skills", {}).get("config", []):
    p = ent.get("path")
    if p and not os.path.exists(p):
        dead_skill_cfg.append(p)

# ---------------- OpenCode ----------------
O = f"{H}/.config/opencode"
for root, org in ((f"{O}/skills", "oc-global"), (f"{O}/skill", "oc-global-singular"),
                  (f"{H}/.opencode/skills", "oc-dot-opencode"), (f"{H}/.claude/skills", "claude-compat"),
                  (f"{H}/.agents/skills", "agents-user")):
    scan_skills("opencode", root, org)
for root, org in ((f"{O}/commands", "oc-global"), (f"{O}/command", "oc-global-singular"), (f"{H}/.opencode/commands", "oc-dot-opencode")):
    scan_md("opencode", "command", root, org, recursive=False)
for root, org in ((f"{O}/agents", "oc-global"), (f"{O}/agent", "oc-global-singular"), (f"{H}/.opencode/agents", "oc-dot-opencode")):
    scan_md("opencode", "agent", root, org, recursive=False)

# ---------------- Gemini ----------------
G = f"{H}/.gemini"
scan_skills("gemini", f"{G}/skills", "gemini-user")
scan_skills("gemini", f"{H}/.agents/skills", "agents-user")
for ext in glob.glob(f"{G}/extensions/*/"):
    ext = ext.rstrip("/\\").replace("\\", "/")
    scan_skills("gemini", f"{ext}/skills", f"extension:{os.path.basename(ext)}")

json.dump({"rows": rows, "claude_missing_enabled": missing_enabled, "synced_plugins": synced_plugins,
           "codex_missing_enabled": codex_missing, "codex_dead_skill_config": dead_skill_cfg}, open(OUT, "w", encoding="utf-8"), indent=1)
print(len(rows), "rows ->", OUT)
print("claude enabled-not-installed:", missing_enabled)
print("codex enabled-without-cache:", codex_missing)
print("codex dead skills.config paths:", len(dead_skill_cfg))
