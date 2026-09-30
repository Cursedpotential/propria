"""Push the documents that are deliberately NOT in git to the Docstore's /extras mount.

Owner ruling 2026-09-28: "index them so that they're searchable but make sure they don't make it
to GitHub." Same line as dev-resources on 2026-09-26 -- indexing is local, publication is not.

The Docstore image is built from a git clone, so a gitignored document cannot be in it. The 14
already in the store therefore read as documents whose sources had vanished, and every sync
stopped at `degraded` with cdc_verified false (872 sources expected, 886 documents observed).
This copies them to a host directory the container mounts read-only at /extras, where service.py
merges them over the git-built docs tree, additively.

WHAT IT SENDS, and what it does not:

  sends    markdown under one of the seven docs roots that `git check-ignore` rejects
  skips    anything git tracks -- the image already has it, and git stays authoritative
  refuses  private/, to_be_deleted/, .docstore*/ -- the registry's mandatory exclusions
  refuses  donor and parts-bin trees (original-context/, donor*/, _archive/). AGENTS.md keeps
           imported third-party material in its own ccc collection, not the Docstore. Including
           them would send 566 files instead of 112, 454 of them one imported custody-guide
           tree. Owner confirmed 2026-09-28: "seems ok".

Nothing here writes to git, and the destination is not a git repository, so a file pushed this
way cannot reach GitHub by this route.

Usage:
    python tools/push-local-docstore-sources.py --repo <checkout>            # dry run
    python tools/push-local-docstore-sources.py --repo <checkout> --push     # copy to the VPS

Byline: Claude Code · Opus 5 · 2026-09-28
"""
from __future__ import annotations

import argparse
import shutil
import subprocess
import sys
from pathlib import Path

# parents[4] is the repository root. The default points at this file's own tree, but a LINKED
# WORKTREE contains tracked files only -- the gitignored documents this tool exists to push are
# not in it. Run it against the main checkout, with --repo if that is somewhere else.
DEFAULT_REPO = Path(__file__).resolve().parents[4]
HOST = "root@100.91.190.107"
KEY = str(Path.home() / ".ssh" / "ovh")
DESTINATION = "/data/probata/volumes/docstore-extras"

# published path under docs/  ->  the module directory that supplies it
ROOTS = {
    "": "docs",
    "probata": "modules/Probata/probata/docs",
    "consignatio": "modules/Consignatio/docs",
    "consignatio-intake": "modules/Consignatio/Intake/docs",
    "advocatio": "modules/Legal-desktop/docs",
    "vestigia": "modules/vestigia-geodata_processor/traceiq-rebuild/docs",
    "family-court": "modules/FL-MCP/docs",
}

# Propria/docs holds a junction per project into that module's docs directory, and rglob follows
# them. Scanning the propria root would pull every other root's files in under a second path, so
# the junction names are skipped there; each is scanned once, as its own root.
JUNCTIONS = {"probata", "consignatio", "consignatio-intake", "advocatio", "vestigia",
             "family-court"}

REFUSED = ("private", "to_be_deleted", ".docstore", ".docstore-control", "node_modules",
           "__pycache__", "original-context", "_archive")

# docs/probata/adr/generated/ is written by adr.refresh_projections INSIDE the container on every
# sync, which is why it is gitignored. Pushing the desktop's 95 copies would hand the generator a
# second source of the same documents and make them churn on every run.
GENERATED = ("adr/generated",)


def refused(relative: Path) -> bool:
    parts = [part.lower() for part in relative.parts[:-1]]
    if any(part in REFUSED or part.startswith("donor") for part in parts):
        return True
    posix = relative.as_posix().lower()
    return any(f"{marker}/" in posix for marker in GENERATED)


def ignored_markdown(repo: Path, directory: Path, skip_junctions: bool) -> list[Path]:
    """Markdown under `directory` that git refuses to track. One git call, not one per file."""
    candidates: list[Path] = []
    for item in sorted(directory.rglob("*.md")):
        if not item.is_file():
            continue
        relative = item.relative_to(directory)
        if refused(relative):
            continue
        if skip_junctions and relative.parts and relative.parts[0] in JUNCTIONS:
            continue
        candidates.append(item)
    if not candidates:
        return []
    # -z gives NUL-separated, unquoted paths. Without it git quotes anything unusual and the
    # Windows line ending rides along, so the returned path no longer resolves.
    probe = subprocess.run(
        ["git", "-C", str(repo), "check-ignore", "-z", "--stdin"],
        input="\0".join(str(item) for item in candidates),
        capture_output=True, text=True,
    )
    # check-ignore exits 1 when nothing matches, which is an answer, not a failure.
    if probe.returncode not in (0, 1):
        raise SystemExit(f"git check-ignore failed: {probe.stderr.strip()[:300]}")
    return [Path(entry) for entry in probe.stdout.split("\0") if entry.strip()]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--push", action="store_true", help="copy to the VPS (default: dry run)")
    parser.add_argument("--repo", default=str(DEFAULT_REPO),
                        help="checkout holding the gitignored documents (not a linked worktree)")
    args = parser.parse_args()
    repo = Path(args.repo).resolve()

    staged: list[tuple[str, Path]] = []
    for published, module in ROOTS.items():
        directory = repo / module
        if not directory.is_dir():
            print(f"  {published or '(root)':24} {module:56} absent")
            continue
        files = ignored_markdown(repo, directory, skip_junctions=(published == ""))
        for absolute in files:
            relative = absolute.relative_to(directory)
            staged.append((str(Path(published) / relative).replace("\\", "/"), absolute))
        print(f"  {published or '(root)':24} {module:56} {len(files)}")

    published_paths = [path for path, _ in staged]
    if len(set(published_paths)) != len(published_paths):
        raise SystemExit("refusing: two source files claim the same published path")

    print(f"\n  {len(staged)} gitignored markdown files to index")
    if not args.push:
        for path, _ in staged[:10]:
            print(f"    {path}")
        if len(staged) > 10:
            print(f"    … and {len(staged) - 10} more")
        print("\n  dry run; pass --push to copy")
        return 0

    staging = Path(repo / ".git" / "docstore-extras-staging")
    if staging.exists():
        shutil.rmtree(staging)
    for path, absolute in staged:
        target = staging / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(absolute.read_bytes())

    print(f"  copying {len(staged)} files to {HOST}:{DESTINATION}")
    subprocess.run(["ssh", "-o", "BatchMode=yes", "-i", KEY, HOST,
                    f"install -d -m 0755 {DESTINATION}"], check=True)
    result = subprocess.run(["scp", "-q", "-o", "BatchMode=yes", "-i", KEY, "-r",
                             f"{staging}/.", f"{HOST}:{DESTINATION}/"])
    shutil.rmtree(staging, ignore_errors=True)
    if result.returncode != 0:
        return result.returncode

    count = subprocess.run(["ssh", "-o", "BatchMode=yes", "-i", KEY, HOST,
                            f"find {DESTINATION} -name '*.md' | wc -l"],
                           capture_output=True, text=True).stdout.strip()
    print(f"  {DESTINATION} now holds {count} markdown files")
    print("  redeploy the Docstore app so service.py merges them at start-up")
    return 0


if __name__ == "__main__":
    sys.exit(main())
