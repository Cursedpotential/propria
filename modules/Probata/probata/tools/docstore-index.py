#!/usr/bin/env python3
"""Run the Docstore documentation index: live-watch by default, or one catch-up pass.

Byline: Claude Code / Opus 5 / 2026-09-26. Owner ruling 2026-09-26: documentation
indexing belongs to Docstore, and it is continuous and automatic. A new or edited
document must be queryable immediately -- no commit, no schedule, and nobody
running a sync by hand.

WHY THIS FILE EXISTS
    `flow_docs.py` cannot simply be run. Two of the variables it needs are read at
    MODULE IMPORT -- DOCSTORE_PROJECT_REGISTRY and DOCSTORE_MULTI_ROOT_ENABLED --
    which is before its own `.docstore/.env` loading happens, so putting them in
    that file cannot work and the run dies with "0.8 requires the complete
    five-root registry and multi-root mode". This launcher sets the environment
    first and imports second. That ordering is the whole point; do not reorder it.

    It also removes the need for a `.docstore/.env` holding copies of secrets:
    values are regex-parsed straight out of ~/.secrets into this process. Those
    files are never sourced -- several use `KEY = value` spacing, which makes a
    shell execute the value.

WHAT RUNS WHERE
    Local: this watcher and the change-detection state. Documents are read where
    they sit, so an uncommitted edit is picked up. Remote: embeddings are API
    calls and the index is the hosted store. Nothing is embedded or stored
    locally, and reading the index happens through ctl, never from here.

    Docstore is its own CocoIndex application on its own environment and state
    database. It is not `ccc` and it is not run through the general vigil runner:
    one store, one owner.

USAGE
    python docstore-index.py                 # live: catch up, then watch
    python docstore-index.py --catch-up      # one pass, then exit
    python docstore-index.py --propria-root E:/path/to/Propria
"""

from __future__ import annotations

import argparse
import os
import pathlib
import re
import runpy
import sys

SECRET_LINE = re.compile(r"^\s*(?:export\s+)?([A-Za-z0-9_]+)\s*=\s*(.+?)\s*$")

# Regex-parsed at run time; never printed, never copied to disk, never sourced.
SECRET_FILES = ("probata-docstore.env", "probata.env")
SECRET_KEYS = ("SURREAL_DOCS_URL", "SURREAL_DOCS_USER", "SURREAL_DOCS_PASS", "NVIDIA_API_KEY")

DEFAULT_PROPRIA_ROOT = "E:/AI_Workspace/Projects/Propria"


def read_secrets(secrets_dir: pathlib.Path) -> dict[str, str]:
    found: dict[str, str] = {}
    for name in SECRET_FILES:
        path = secrets_dir / name
        if not path.exists():
            continue
        for line in path.read_text(encoding="utf-8", errors="ignore").splitlines():
            m = SECRET_LINE.match(line)
            if m and m.group(1) in SECRET_KEYS and m.group(1) not in found:
                found[m.group(1)] = m.group(2).strip().strip('"').strip("'")
    missing = set(SECRET_KEYS) - set(found)
    if missing:
        raise SystemExit(
            f"docstore-index: missing {', '.join(sorted(missing))} in "
            f"{secrets_dir}/{{{','.join(SECRET_FILES)}}}"
        )
    return found


def main() -> int:
    ap = argparse.ArgumentParser(prog="docstore-index")
    ap.add_argument("--catch-up", action="store_true",
                    help="one pass then exit; the default is to stay up and watch")
    ap.add_argument("--full-reprocess", action="store_true",
                    help="re-embed everything, ignoring change detection")
    ap.add_argument("--propria-root", default=os.environ.get("PROPRIA_ROOT", DEFAULT_PROPRIA_ROOT))
    ap.add_argument("--secrets-dir", default=os.environ.get(
        "SECRETS_DIR", str(pathlib.Path.home() / ".secrets")))
    args = ap.parse_args()

    root = pathlib.Path(args.propria_root)
    registry = root / "docs" / "docstore-source-registry.json"
    flow = root / "modules" / "Probata" / "probata" / "scripts" / "docstore" / "flow_docs.py"
    for label, path in (("registry", registry), ("flow", flow)):
        if not path.exists():
            raise SystemExit(f"docstore-index: {label} not found: {path}\n"
                             f"  pass --propria-root or set PROPRIA_ROOT")

    env = {
        # Read at import by flow_docs.py -- must be set before it loads.
        "DOCSTORE_PROJECT_REGISTRY": str(registry),
        "DOCSTORE_MULTI_ROOT_ENABLED": "1",
        "DOCSTORE_LIVE": "" if args.catch_up else "1",
    }
    if args.full_reprocess:
        env["DOCSTORE_FULL_REPROCESS"] = "1"
    env.update(read_secrets(pathlib.Path(args.secrets_dir)))
    os.environ.update(env)

    sys.path.insert(0, str(flow.parent))
    print(f"docstore-index: mode={'catch-up' if args.catch_up else 'live'} root={root}", flush=True)
    runpy.run_path(str(flow), run_name="__main__")
    return 0


if __name__ == "__main__":
    sys.exit(main())
