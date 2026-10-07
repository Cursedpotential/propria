"""Independent seven-root manifests and exact-plan/retraction regression proof.

Byline: Codex / GPT-6.1 / 2026-10-07. Isolated fixtures only, no live store.
"""
import hashlib
import json
import sys
from contextlib import nullcontext
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import source_sync


def entry(project='propria', path='new.md', content='# new\n'):
    return {'project':project, 'path':path, 'content':content,
            'sha256':hashlib.sha256(content.encode()).hexdigest()}


def test_explicit_roots_allow_empty_and_held_projects():
    roots=list(source_sync.ROOTS)
    assert source_sync.validated([], roots)==[]
    assert source_sync.validated([entry()], roots)==[entry()]


@pytest.mark.parametrize('roots', [[], ['propria'], 'propria',
    ['propria']*7, [*list(source_sync.ROOTS)[:-1], 'unapproved'],
    [*list(source_sync.ROOTS), 'propria'], [None]*7])
def test_invalid_root_manifest_rejected(roots):
    with pytest.raises(ValueError, match='roots'):
        source_sync.validated([], roots)


def test_legacy_files_only_still_requires_every_root():
    files=[entry(project) for project in source_sync.ROOTS]
    assert source_sync.validated(files)==files
    with pytest.raises(ValueError):
        source_sync.validated(files[:-1])
    files[-1]=entry('propria', 'different.md')
    with pytest.raises(ValueError, match='Every docs root'):
        source_sync.validated(files)


@pytest.mark.parametrize('bad', [None, 'bad', {'path':'new.md'},
    {'project':[], 'path':'new.md'}, {'project':'propria','path':[]},
    entry(path='../outside.md'), entry(path='private/no.md'),
    {**entry(), 'sha256':'wrong'}])
def test_source_validation_remains_strict(bad):
    with pytest.raises(ValueError):
        source_sync.validated([bad], list(source_sync.ROOTS))


@pytest.fixture
def mirror(tmp_path, monkeypatch):
    root=tmp_path/'mirror'
    root.mkdir()
    projects=[]
    for project,(relative,prefix) in source_sync.ROOTS.items():
        (root/relative).mkdir(parents=True, exist_ok=True)
        projects.append({'project_id':project, 'source_root':relative,
            'canonical_prefix':prefix, 'registration_status':'active',
            'required':True, 'excluded_patterns':['private/**','**/to_be_deleted/**']})
    registry=tmp_path/'registry.json'
    registry.write_text(json.dumps({'monorepo_root':str(root),'projects':projects}))
    monkeypatch.setenv('DOCSTORE_PROJECT_REGISTRY', str(registry))
    monkeypatch.setenv('DOCSTORE_SYNC_LOCK', str(tmp_path/'sync.lock'))
    monkeypatch.setattr(source_sync,'worker_lock',lambda _:nullcontext())
    return root


def test_plan_nonmutating_and_apply_quarantines_only_named_retraction(mirror):
    old=mirror/'docs'/'held.md'
    old.write_bytes(b'# held\n')
    payload={'roots':list(source_sync.ROOTS), 'files':[entry()]}
    plan=source_sync.operation('plan', payload)
    assert plan['roots']==sorted(source_sync.ROOTS)
    assert plan['retracted_sources']==['propria/held.md']
    assert old.read_bytes()==b'# held\n'
    assert not (mirror/'docs'/'new.md').exists()
    with pytest.raises(ValueError, match='without being named'):
        source_sync.operation('apply', {**payload,'plan_id':plan['plan_id']})
    assert old.read_bytes()==b'# held\n'
    receipt=source_sync.operation('apply', {**payload,'plan_id':plan['plan_id'],
        'retract':['propria/held.md']})
    assert receipt['verified'] and not receipt['index_started']
    assert (mirror/'docs'/'new.md').read_bytes()==b'# new\n'
    saved=mirror/'to_be_deleted'/('sync-'+receipt['generation'])/'propria'/'held.md'
    assert saved.read_bytes()==b'# held\n'


def test_hold_preserved_by_including_mirror_body(mirror):
    (mirror/'docs'/'held.md').write_bytes(b'# held\n')
    payload={'roots':list(source_sync.ROOTS),
             'files':[entry(),entry(path='held.md',content='# held\n')]}
    plan=source_sync.operation('plan', payload)
    assert not plan['retracted_sources']
    receipt=source_sync.operation('apply',{**payload,'plan_id':plan['plan_id']})
    assert receipt['verified']
    assert (mirror/'docs'/'held.md').read_bytes()==b'# held\n'


def test_roots_bound_into_exact_plan(mirror):
    files=[entry(project) for project in source_sync.ROOTS]
    explicit={'roots':list(source_sync.ROOTS),'files':files}
    plan=source_sync.operation('plan',explicit)
    with pytest.raises(ValueError,match='plan changed'):
        source_sync.operation('apply',{'files':files,'plan_id':plan['plan_id']})
