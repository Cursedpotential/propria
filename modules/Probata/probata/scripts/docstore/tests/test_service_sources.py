"""Verify the code-only Docstore image's staged source contract."""
# Byline: Codex · GPT-6.1-sol · 2026-10-07.
import json
import sys
from pathlib import Path

import pytest

PIPELINE = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(PIPELINE))
import scope
import service


@pytest.fixture
def staged(tmp_path, monkeypatch):
    """Build seven small source roots and a registry with production identities.

    Inputs: isolated tmp path. Output: root, extras and registry payload. Effects:
    writes fixture files only. Choose for startup contract tests, never live data.
    """
    root = tmp_path / 'app'
    docs = root / 'docs'
    docs.mkdir(parents=True)
    projects = []
    for project, (relative, prefix) in scope.ROOTS.items():
        (root / relative).mkdir(parents=True, exist_ok=True)
        projects.append({'project_id': project, 'source_root': relative,
                         'canonical_prefix': prefix, 'registration_status': 'active',
                         'ingestion_status': 'current-full-source' if project == 'probata' else 'pending-multi-root-cdc',
                         'required': True, 'domains': ['docs'], 'included_patterns': ['**/*.md'],
                         'excluded_patterns': ['private/**', '**/to_be_deleted/**']})
    payload = {'schema': 'propria-docstore-source-registry-v1',
               'monorepo_root': str(root), 'projects': projects}
    registry = docs / 'docstore-source-registry.json'
    registry.write_text(json.dumps(payload), encoding='utf-8')
    extras = tmp_path / 'extras'
    (extras / 'probata').mkdir(parents=True)
    (extras / 'probata' / 'private.md').write_bytes(b'# private\n')
    (docs / 'probata' / 'private.md').write_bytes(b'# private\n')
    monkeypatch.setenv('DOCSTORE_PROJECT_REGISTRY', str(registry))
    monkeypatch.setenv('DOCSTORE_LOCAL_SOURCES', str(extras))
    return root, extras, payload


def test_staged_sources_validate_exact_roots_and_private_bytes(staged):
    root, _, _ = staged
    assert service.validate_staged_sources(root)['roots'] == 7
    assert service.verify_local_sources(root)['verified'] == 1


def test_staged_sources_reject_missing_root_and_identity_change(staged):
    root, _, payload = staged
    registry = root / 'docs/docstore-source-registry.json'
    (root / 'docs/family-court').rename(root / 'docs/family-court-held')
    with pytest.raises(ValueError, match='missing'):
        service.validate_staged_sources(root)
    (root / 'docs/family-court-held').rename(root / 'docs/family-court')
    payload['projects'][0]['canonical_prefix'] = 'changed/docs/'
    registry.write_text(json.dumps(payload), encoding='utf-8')
    with pytest.raises(ValueError, match='prefix'):
        service.validate_staged_sources(root)


def test_staged_sources_reject_wrong_monorepo_root(staged):
    root, _, payload = staged
    payload['monorepo_root'] = '/other'
    (root / 'docs/docstore-source-registry.json').write_text(json.dumps(payload), encoding='utf-8')
    with pytest.raises(ValueError, match='monorepo_root'):
        service.validate_staged_sources(root)


def test_staged_startup_requires_durable_quarantine_mount(staged, monkeypatch):
    root, _, _ = staged
    docs = root / 'docs'
    monkeypatch.setattr(Path, 'is_mount', lambda path: path == docs)
    with pytest.raises(ValueError, match='to_be_deleted'):
        service.validate_staged_sources(root, require_mount=True)


def test_staged_private_sources_fail_on_missing_or_changed_staging(staged):
    root, extras, _ = staged
    target = root / 'docs/probata/private.md'
    target.write_bytes(b'# different\n')
    with pytest.raises(ValueError, match='differs'):
        service.verify_local_sources(root)
    target.rename(target.with_suffix('.held'))
    with pytest.raises(ValueError, match='missing'):
        service.verify_local_sources(root)
    assert (extras / 'probata/private.md').read_bytes() == b'# private\n'


def test_legacy_writable_overlay_remains_additive(staged):
    root, extras, _ = staged
    (extras / 'probata/new.md').write_bytes(b'# new\n')
    report = service.overlay_local_sources(root)
    assert report['added'] == 1
    assert report['already_in_git'] == 1
    assert (root / 'docs/probata/private.md').read_bytes() == b'# private\n'
    assert (root / 'docs/probata/new.md').read_bytes() == b'# new\n'


def test_release_binds_writable_sources_and_durable_quarantine_without_baking_documents():
    """Assert the code-only image has staged source and quarantine binds.

    Inputs: checked-in release definitions. Output: contract assertions. Effects:
    reads files only. Choose as a focused guard for this release boundary.
    """
    module = PIPELINE.parents[1]
    dockerfile = (module / 'deploy/docker/docstore/Dockerfile').read_text(encoding='utf-8')
    compose = (module / 'deploy/docstore.yaml').read_text(encoding='utf-8')
    assert 'COPY docs/' not in dockerfile
    assert 'COPY modules/Probata/probata/docs/' not in dockerfile
    assert 'COPY --from=docs' not in dockerfile
    source_mount = compose.split('source: /data/probata/volumes/docstore-sources', 1)[1].split('source: /data/probata/volumes/docstore-quarantine', 1)[0]
    quarantine_mount = compose.split('source: /data/probata/volumes/docstore-quarantine', 1)[1].split('# The private additions', 1)[0]
    assert 'target: /app/docs' in source_mount
    assert 'read_only: true' not in source_mount
    assert 'create_host_path: false' in source_mount
    assert 'target: /app/to_be_deleted' in quarantine_mount
    assert 'create_host_path: false' in quarantine_mount
    assert "DOCSTORE_SOURCES_STAGED: '1'" in compose
    assert 'DOCSTORE_SOURCES_READ_ONLY' not in compose
