"""Byline: Codex / GPT-6, 2026-09-20. Windows adapter encoding regression."""
import importlib.util
import json
from pathlib import Path
from types import SimpleNamespace
import pytest

path=Path(__file__).resolve().parents[1]/'plugins/docstore/claude/federation.py'
spec=importlib.util.spec_from_file_location('federation_adapter_test',path)
federation=importlib.util.module_from_spec(spec)
spec.loader.exec_module(federation)


@pytest.mark.parametrize('valid',[True,False])
def test_adapter_encoding_is_isolated_from_local_sources(tmp_path,monkeypatch,valid):
    monkeypatch.delenv('DOCSTORE_CONTROL_MCP_URL',raising=False)
    monkeypatch.setattr(federation,'file_search',lambda source,*args,**kwargs: (
        ([{'id':'local','source':source,'snippet':'retained local context'}] if source=='claude' else []),
        {'available':source=='claude','queried':source=='claude'}))
    payload=json.dumps({'results':[{'id':'remote','snippet':'Résumé – document'}]},ensure_ascii=False).encode() if valid else b'\x97'
    monkeypatch.setattr(federation.subprocess,'run',lambda *args,**kwargs:SimpleNamespace(returncode=0,stdout=payload))
    result=federation.recall('document',tmp_path,adapters={'read-memories':['adapter']})
    assert result['sources']['read-memories']['available'] is valid
    assert any(row['snippet']=='retained local context' for row in result['results'])
    assert any(row['snippet']=='Résumé – document' for row in result['results']) is valid
