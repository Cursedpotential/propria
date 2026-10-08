"""Prove selected policy invalidation against CocoIndex with three tiny source files and no targets.

Inputs: a new scratch directory. Outputs: JSON counters and retained scratch
tracking state. Side effects: tiny fixture files and CocoIndex bookkeeping only;
no database/index targets, remote providers, or production state are used.
"""
from __future__ import annotations

import argparse
import asyncio
import importlib.metadata
import json
import sys
from pathlib import Path
from dataclasses import dataclass

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import cocoindex as coco
from cocoindex.connectors import localfs
from context_sources import contextual_items

CONFIG = {}
CALLS = []
CHUNKS = []
BASE = coco.ContextKey[Path]("contextual_memo_proof_sources")


async def table_target_identity() -> dict:
    """Compare real connector memo identities for raw and expanded table schemas.

    Inputs: installed connector only. Output: identity/serialization checks. Side
    effects: none; descriptors are not mounted and no database connection opens.
    Use to verify schema additions do not change file-target memo arguments.
    """
    try:
        from cocoindex.connectors import surrealdb
    except ImportError:
        return {"verified": False, "reason": "SurrealDB connector dependency unavailable"}
    from cocoindex._internal.memo_fingerprint import memo_fingerprint

    @dataclass
    class Raw:
        id: str
        text: str

        @property
        def description(self) -> str:
            return ""

    @dataclass
    class Expanded(Raw):
        description: str = ""

    raw_schema = await surrealdb.TableSchema.from_class(Raw)
    expanded_schema = await surrealdb.TableSchema.from_class(Expanded)
    key = coco.ContextKey("contextual_memo_proof_target_identity")
    raw_state = surrealdb.table_target(key, "chunk", raw_schema, managed_by="user")
    expanded_state = surrealdb.table_target(key, "chunk", expanded_schema, managed_by="user")
    # Descriptor introspection only; target keys must exclude the changed schema.
    key_equal = memo_fingerprint(raw_state._key).as_bytes() == memo_fingerprint(expanded_state._key).as_bytes()
    raw_target = surrealdb.TableTarget(raw_state._provider, raw_schema, "chunk")
    expanded_target = surrealdb.TableTarget(expanded_state._provider, expanded_schema, "chunk")
    memo_equal = memo_fingerprint(raw_target).as_bytes() == memo_fingerprint(expanded_target).as_bytes()
    legacy_serialized = expanded_target._row_to_dict(Raw("a", "original"))
    if not key_equal or not memo_equal or legacy_serialized != {"id": "a", "text": "original", "description": ""}:
        raise AssertionError("Expanded table schema changed memo identity or legacy serialization")
    return {"verified": True, "table_key_equal": key_equal, "table_target_memo_equal": memo_equal,
            "legacy_row_serializes": True, "mounted_targets": 0}


@coco.fn
async def raw_group(name: str) -> None:
    """Record one tiny legacy group execution without declaring targets.

    Input: source name. Output: none. Side effects: in-memory counter. Use for baseline.
    """
    CHUNKS.append((name, "raw"))


@coco.fn
async def contextual_group(name: str) -> None:
    """Record one tiny contextual group execution without provider calls.

    Input: source name. Output: none. Side effects: in-memory counter. Use for selected proof.
    """
    CHUNKS.append((name, "context"))


@coco.fn(memo=True, version=8)
async def process_file(file) -> None:
    """Read one source under its original memo version and stable group subpath.

    Input: FileLike or policy proxy. Output: none. Side effects: source read/counters,
    child mount only. Use to model the production file/group memo topology.
    """
    await file.read()
    name = file.file_path.path.as_posix()
    CALLS.append(name)
    if getattr(file, "context_policy", None) is None:
        handle = await coco.mount_each(raw_group, [(0, name)])
    else:
        handle = await coco.mount_each(coco.component_subpath(coco.Symbol("raw_group")),
                                       contextual_group, [(0, name)])
    await handle.ready()


@coco.fn
async def app_main() -> None:
    """Mount the complete three-file live source map on every proof phase.

    Input: fixture source and phase config. Output: none. Side effects: source watch
    and existing component bookkeeping. Every phase declares every original key.
    """
    files = localfs.walk_dir(BASE, live=True, recursive=True)
    handle = await coco.mount_each(process_file, contextual_items(files.items(), "docs/", CONFIG))
    await handle.ready()


async def run(directory: Path) -> dict:
    """Execute baseline, selection, stable-cache and model-change proof phases.

    Input: fresh scratch directory. Output: counts with exact SDK version. Side
    effects: tiny fixture/tracking files only. Pick this before production activation.
    """
    if directory.exists():
        raise ValueError("Proof scratch directory must be new; existing state is preserved")
    source = directory / "sources"
    source.mkdir(parents=True)
    for name in ("a.md", "b.md", "c.md"):
        (source / name).write_bytes(("# " + name + "\r\nTiny fixture.\r\n").encode())
    env = coco.Environment(coco.Settings.from_env(db_path=directory / "tracking.db"), name="contextual-memo-proof")
    env.context_provider.provide(BASE, source)
    app = coco.App(coco.AppConfig(name="ContextualMemoProof", environment=env, max_inflight_components=4), app_main)
    phases = []
    for phase in ("baseline", "selected-a", "same-policy", "model-change-a"):
        if phase == "selected-a":
            CONFIG.update(DOCSTORE_CONTEXT_ENABLED="1", DOCSTORE_CONTEXT_SOURCES='["docs/a.md"]',
                          DOCSTORE_LLM_MODEL="proof/model-v1", DOCSTORE_LLM_BASE_URL="https://example.invalid/v1")
        if phase == "model-change-a":
            CONFIG["DOCSTORE_LLM_MODEL"] = "proof/model-v2"
        CALLS.clear()
        CHUNKS.clear()
        handle = app.update()
        await handle.result()
        stats = handle.stats()
        if stats is None or stats.total.num_errors:
            raise RuntimeError("CocoIndex proof phase failed")
        calls = sorted(CALLS)
        expected = ["a.md", "b.md", "c.md"] if phase == "baseline" else [] if phase == "same-policy" else ["a.md"]
        if calls != expected:
            raise AssertionError((phase, calls, expected))
        phases.append({"phase": phase, "file_executions": calls, "group_executions": sorted(CHUNKS)})
    return {"sdk_version": importlib.metadata.version("cocoindex"), "phases": phases,
            "table_identity": await table_target_identity(),
            "source_membership": ["a.md", "b.md", "c.md"], "target_writes": 0,
            "provider_calls": 0, "scratch_preserved": str(directory)}


def main() -> None:
    """Run the targetless memo proof with an explicit fresh scratch directory.

    Input: --scratch path. Output: JSON proof. Side effects: those documented by run.
    Pick an ignored quarantine/scratch location; this script never deletes files.
    """
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--scratch", type=Path, required=True)
    args = parser.parse_args()
    print(json.dumps(asyncio.run(run(args.scratch)), indent=2))


if __name__ == "__main__":
    main()
