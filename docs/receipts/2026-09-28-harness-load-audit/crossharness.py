# Byline: Claude Code · Opus 5.5 · 2026-09-28 (harness load audit; read-only)
import collections
import glob
import hashlib
import json
import os

H = os.path.expanduser("~").replace("\\", "/")
roots = {"claude": f"{H}/.claude/skills", "agents": f"{H}/.agents/skills", "codex": f"{H}/.codex/skills",
         "gemini": f"{H}/.gemini/skills", "opencode": f"{H}/.config/opencode/skills"}


def norm(b):
    return b.replace(b"\r\n", b"\n")


def dirhash(d):
    hs = hashlib.sha256()
    for p in sorted(glob.glob(d + "/**/*", recursive=True)):
        pp = p.replace("\\", "/")
        if os.path.isfile(p) and "/.git" not in pp and "node_modules" not in pp and "__pycache__" not in pp:
            hs.update(os.path.relpath(p, d).replace("\\", "/").encode())
            hs.update(norm(open(p, "rb").read()))
    return hs.hexdigest()[:12]


by = collections.defaultdict(dict)
for k, r in roots.items():
    for s in glob.glob(r + "/*/SKILL.md"):
        d = os.path.dirname(s)
        by[os.path.basename(d)][k] = (dirhash(d), hashlib.sha256(norm(open(s, "rb").read())).hexdigest()[:12], os.path.getmtime(s))
out = {}
for n, m in sorted(by.items()):
    if len(m) > 1:
        out[n] = m
        skillsame = len({v[1] for v in m.values()}) == 1
        dirsame = len({v[0] for v in m.values()}) == 1
        newest = max(m, key=lambda k: m[k][2])
        print(f"{n:32} {'SKILL=' if skillsame else 'SKILL~'} {'DIR=' if dirsame else 'DIR~'} newest={newest:8} " + " ".join(f"{k}:{v[1][:6]}/{v[0][:6]}" for k, v in m.items()))
print(len(out), "names present in 2+ harness skill roots")
json.dump(out, open(os.path.join(os.path.dirname(os.path.abspath(__file__)), "crossharness.json"), "w"), indent=1)
