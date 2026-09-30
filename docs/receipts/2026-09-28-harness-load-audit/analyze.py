# Byline: Claude Code · Opus 5.5 · 2026-09-28 (harness load audit; read-only)
import collections
import json
import os
import sys

D = json.load(open(os.path.join(os.path.dirname(os.path.abspath(__file__)), "inventory.json"), encoding="utf-8"))
rows = [r for r in D["rows"] if r.get("kind") in ("skill", "command", "agent", "prompt")]
harness = sys.argv[1]
kinds = sys.argv[2].split(",") if len(sys.argv) > 2 else ["skill"]
R = [r for r in rows if r["harness"] == harness and r["kind"] in kinds and "DISABLED" not in r["origin"]]
by = collections.defaultdict(list)
for r in R:
    by[(r["name"].lower())].append(r)
n = 0
for name, lst in sorted(by.items()):
    reals = {r["real"] for r in lst}
    if len(lst) < 2 or len(reals) < 2:
        continue
    n += 1
    hashes = {r["hash"] for r in lst}
    print(f"{name}  [{'SAME' if len(hashes) == 1 else 'DIFF'}]")
    for r in lst:
        print(f"    {r['hash']} {r['kind']:7} {r['origin'][:48]:48} {('ns=' + r['ns']) if r['ns'] else ''}")
print("dup names:", n, "of", len(by))
# frontmatter problems
bad = [r for r in R if r["kind"] == "skill" and (r["nofm"] or not r["has_desc"] or (r["fm_name"] and r["fm_name"] != r["name"]))]
print("\nfrontmatter issues:", len(bad))
for r in bad:
    print("   ", r["origin"][:40], r["name"], "nofm" if r["nofm"] else "", "nodesc" if not r["has_desc"] else "", f"fm_name={r['fm_name']}" if r["fm_name"] and r["fm_name"] != r["name"] else "")
