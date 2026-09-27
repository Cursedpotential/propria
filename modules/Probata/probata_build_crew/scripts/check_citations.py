"""Mechanical citation check for crew outputs: does every `path:line` citation point at a real file and line?

Why: on run-08e the gatekeeper LLM reported `docs/DECISION_LOG.md:1200 VERIFIED` on a 331-line file. An LLM must
not be the verifier of record. This script parses every citation of the forms
    path:line · path:line-line · [file: path] · [file: path:line] · `path:line`
from a markdown document, resolves it under the repository root, and reports EXISTS / LINE-IN-RANGE / MISSING.
When a citation sits inside a table row or bullet, the row's other backticked identifiers are searched within
+/- 40 lines of the cited line so obviously wrong anchors surface as WEAK.

Usage:
    uv run python scripts/check_citations.py <document.md> [--repo E:/.../probata] [--append]
`--append` writes a '## Mechanical citation check' section to the end of the document.

Byline: Claude Code · Fable 5.1 · 2026-09-06
"""

import argparse
import io
import os
import re
import sys
from collections import Counter
from pathlib import Path

CITE = re.compile(
    r"(?P<path>(?:[A-Za-z0-9_.\-]+/)*[A-Za-z0-9_.\-]+\.(?:go|py|sql|md|yaml|yml|toml|json|jsonc|txt|sh|ts|tsx|js))"
    r"(?::(?P<line>\d+)(?:-(?P<end>\d+))?)?"
)
IDENT = re.compile(r"`([A-Za-z_][A-Za-z0-9_.]{3,})`")


def check(doc_path: Path, repo: Path) -> tuple[list[dict], Counter]:
    text = io.open(doc_path, encoding="utf-8").read()
    results: list[dict] = []
    seen: set[tuple[str, int | None]] = set()
    for ln, row in enumerate(text.splitlines(), 1):
        for m in CITE.finditer(row):
            rel = m.group("path")
            line = int(m.group("line")) if m.group("line") else None
            key = (rel, line)
            if key in seen or rel.startswith("http"):
                continue
            seen.add(key)
            target = repo / rel
            rec = {"doc_line": ln, "cite": m.group(0), "status": "", "note": ""}
            if not target.is_file():
                # tolerate citations of files that live outside the repo root (absolute paths, other repos)
                if Path(rel).is_file():
                    target = Path(rel)
                else:
                    rec["status"] = "MISSING"
                    rec["note"] = "file not found under repo root"
                    results.append(rec)
                    continue
            if line is None:
                rec["status"] = "EXISTS"
                results.append(rec)
                continue
            try:
                lines = io.open(target, encoding="utf-8", errors="replace").read().splitlines()
            except OSError as exc:
                rec["status"] = "MISSING"
                rec["note"] = f"unreadable: {exc}"
                results.append(rec)
                continue
            if line < 1 or line > len(lines):
                rec["status"] = "OUT-OF-RANGE"
                rec["note"] = f"file has {len(lines)} lines"
                results.append(rec)
                continue
            idents = [i for i in IDENT.findall(row) if i not in rel]
            if idents:
                lo, hi = max(0, line - 41), min(len(lines), line + 40)
                window = "\n".join(lines[lo:hi])
                hits = [i for i in idents if i.split(".")[-1] in window]
                if hits:
                    rec["status"] = "IN-RANGE+KEYWORD"
                    rec["note"] = f"'{hits[0]}' within ±40 lines"
                else:
                    rec["status"] = "WEAK"
                    rec["note"] = f"none of {idents[:3]} within ±40 lines of {line}"
            else:
                rec["status"] = "IN-RANGE"
            results.append(rec)
    return results, Counter(r["status"] for r in results)


def render(results: list[dict], counts: Counter, repo: Path) -> str:
    total = sum(counts.values())
    out = ["## Mechanical citation check", "",
           f"_Checked by scripts/check_citations.py against `{repo}`: {total} distinct citations._", "",
           "| Status | Count |", "|---|---|"]
    out += [f"| {k} | {v} |" for k, v in counts.most_common()]
    bad = [r for r in results if r["status"] in {"MISSING", "OUT-OF-RANGE", "WEAK"}]
    if bad:
        out += ["", "### Citations needing attention", "", "| Doc line | Citation | Status | Note |", "|---|---|---|---|"]
        out += [f"| {r['doc_line']} | `{r['cite']}` | {r['status']} | {r['note']} |" for r in bad]
    return "\n".join(out) + "\n"


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("document")
    ap.add_argument("--repo", default=os.environ.get("PROBATA_REPO_ROOT", "E:/AI_Workspace/Projects/the-platform-workspace/probata"))
    ap.add_argument("--append", action="store_true")
    a = ap.parse_args(argv)
    doc = Path(a.document)
    results, counts = check(doc, Path(a.repo))
    report = render(results, counts, Path(a.repo))
    print(report)
    if a.append:
        text = io.open(doc, encoding="utf-8").read()
        if "## Mechanical citation check" in text:
            text = text[: text.index("## Mechanical citation check")].rstrip() + "\n\n"
        io.open(doc, "w", encoding="utf-8", newline="\n").write(text.rstrip() + "\n\n---\n\n" + report)
        print(f"appended to {doc}")
    return 0 if not counts.get("MISSING") and not counts.get("OUT-OF-RANGE") else 1


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
