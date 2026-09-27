"""Assemble output/work-package.md deterministically from the stage outputs.

Why: asking the gatekeeper LLM to re-copy the work package, the gap table and the review table verbatim (~30 KB)
before its own section exhausted the completion budget (run-08d, 2026-09-06) and invited copy drift. Now the
gatekeeper writes only `## Gatekeeper review` and this script stitches the deliverable together.

Inputs (files): constraints brief, code review, gap analysis, work package draft, gatekeeper review.
Output: output/work-package.md = draft + '## Gap analysis' (gap table) + '## Code review summary' (findings table)
        + gatekeeper review, with a provenance header.

Usage:
    uv run python scripts/assemble_package.py <brief.md> <review.md> <gaps.md> <draft.md> <gate.md> [out.md]

Byline: Claude Code · Fable 5.1 · 2026-09-06
"""

import datetime as dt
import io
import re
import sys
from pathlib import Path


def _section(text: str, start_pat: str, end_pat: str) -> str:
    """Return the slice from the first heading matching start_pat up to (not including) the next matching end_pat."""
    m = re.search(start_pat, text, flags=re.M)
    if not m:
        return ""
    rest = text[m.start():]
    e = re.search(end_pat, rest[m.end() - m.start():], flags=re.M)
    return rest if not e else rest[: (m.end() - m.start()) + e.start()]


def assemble(brief: str, review: str, gaps: str, draft: str, gate: str) -> str:
    gap_table = _section(gaps, r"^##\s+Gap Table", r"^##\s+") or gaps[:6000]
    findings = _section(review, r"^##\s+2\s+Findings Table", r"^##\s+") or _section(review, r"^##.*Findings Table", r"^##\s+") or "(findings table not found in review output)"
    gate_body = gate.strip()
    if not re.search(r"^##\s+Gatekeeper review", gate_body, flags=re.M):
        gate_body = "## Gatekeeper review\n\n" + gate_body
    stamp = dt.datetime.now().strftime("%Y-%m-%d %H:%M")
    header = (
        f"<!-- assembled by scripts/assemble_package.py at {stamp}; sources: task outputs 1-4 + gatekeeper review. "
        "Do not edit by hand - edit the stage outputs and re-assemble. -->\n\n"
    )
    return (
        header
        + draft.strip()
        + "\n\n---\n\n## Gap analysis\n\n"
        + gap_table.strip()
        + "\n\n---\n\n## Code review summary\n\n"
        + findings.strip()
        + "\n\n---\n\n"
        + gate_body
        + "\n"
    )


def main(argv: list[str]) -> int:
    if len(argv) not in (5, 6):
        print(__doc__)
        return 2
    paths = [Path(p) for p in argv[:5]]
    texts = [io.open(p, encoding="utf-8").read() for p in paths]
    out = Path(argv[5]) if len(argv) == 6 else Path("output/work-package.md")
    out.parent.mkdir(parents=True, exist_ok=True)
    result = assemble(*texts)
    io.open(out, "w", encoding="utf-8", newline="\n").write(result)
    print(f"assembled {out} ({len(result)} bytes)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
