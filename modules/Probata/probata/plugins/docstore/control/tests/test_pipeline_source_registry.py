"""The worker source registry preserves current IDs and gates multi-root activation."""
from __future__ import annotations

import importlib.util
import json
import sys
from pathlib import Path

import pytest


MODULE_PATH = Path(__file__).parents[4] / "scripts" / "docstore" / "source_registry.py"
# scope.py sits beside it and owns the root set. Import it here rather than inside the fixture,
# so this file does not depend on another suite having already put scripts/docstore on sys.path.
sys.path.insert(0, str(MODULE_PATH.parent))
from scope import ROOTS

SPEC = importlib.util.spec_from_file_location("docstore_pipeline_source_registry", MODULE_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)
load_sources = MODULE.load_sources


def registry(tmp_path: Path) -> tuple[Path, Path]:
    root=tmp_path/'propria'; root.mkdir()
    projects=[]
    for identity,(relative,prefix) in ROOTS.items():
        (root/relative).mkdir(parents=True,exist_ok=True)
        projects.append({'project_id':identity,'title':identity,'source_root':relative,'canonical_prefix':prefix,
                         'domains':['docs'],'included_patterns':['**/*.md'],
                         'excluded_patterns':['private/**','**/to_be_deleted/**'],
                         'registration_status':'active','ingestion_status':'current-full-source' if identity=='probata' else 'pending-multi-root-cdc'})
    path=tmp_path/'registry.json'
    path.write_text(json.dumps({'schema':'propria-docstore-source-registry-v1','monorepo_root':str(root),'projects':projects}))
    return path,root/ROOTS['probata'][0]


def test_legacy_mode_is_rejected_for_complete_scope(tmp_path):
    with pytest.raises(ValueError, match='complete governed registry'):
        load_sources(None,tmp_path,multi_root_enabled=False)


def test_registry_without_activation_is_rejected(tmp_path):
    path,current=registry(tmp_path)
    with pytest.raises(ValueError, match='complete governed registry'):
        load_sources(path,current,multi_root_enabled=False)


def test_registry_activation_declares_complete_source_set(tmp_path):
    path, current = registry(tmp_path)
    sources, fingerprint = load_sources(path, current, multi_root_enabled=True)
    assert {source.project_id for source in sources} == set(ROOTS)  # never a literal: this listed 5 while scope.py had 7
    assert len(fingerprint) == 64


def test_multi_root_activation_requires_explicit_registry(tmp_path):
    with pytest.raises(ValueError, match="complete governed registry"):
        load_sources(None, tmp_path, multi_root_enabled=True)


def test_docs_registry_rejects_code_file_classes(tmp_path):
    path, current = registry(tmp_path)
    payload = json.loads(path.read_text(encoding="utf-8"))
    payload["projects"][0]["included_patterns"] = ["**/*.py"]
    path.write_text(json.dumps(payload), encoding="utf-8")
    with pytest.raises(ValueError, match="only Markdown"):
        load_sources(path, current, multi_root_enabled=True)
