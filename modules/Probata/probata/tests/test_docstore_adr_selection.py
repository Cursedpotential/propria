"""Offline ADR selector contract tests; no live corpus, DB writes or file writes."""

import asyncio
import importlib.util
from pathlib import Path

import pytest


@pytest.fixture
def adr(monkeypatch):
    """Load the implementation with its sibling imports, without opening a store."""
    directory = Path(__file__).resolve().parents[1] / 'scripts' / 'docstore'
    monkeypatch.syspath_prepend(str(directory))
    spec = importlib.util.spec_from_file_location('docstore_adr_selection', directory / 'adr.py')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def record(adr_number, **overrides):
    """Construct an in-memory unit-test row, never a canonical source record."""
    return {'id': f'adr:propria_{adr_number:04d}', 'number': adr_number, 'project': 'propria',
            'title': 'Selector contract', 'decision': 'Keep selection exact',
            'status': 'proposed', **overrides}


class ReadOnlyDB:
    """Reject every query except exact reads and explicitly tested legacy browsing."""

    def __init__(self, records, allow_browse=False):
        self.records = {r['id']: r for r in records}
        self.allow_browse = allow_browse
        self.calls = []
        self.closed = 0

    async def query(self, sql, params=None):
        self.calls.append((sql, params))
        if sql == 'SELECT * FROM type::record($id);':
            row = self.records.get(params['id'])
            return [row] if row else []
        if sql.startswith('SELECT id FROM document WHERE source_path='):
            return []
        if self.allow_browse and sql == 'SELECT * FROM adr WHERE project="propria" ORDER BY number LIMIT $limit;':
            return sorted(self.records.values(), key=lambda r: r['number'])[:params['limit']]
        raise AssertionError(f'Unexpected broad query or mutation: {sql}')

    async def close(self):
        self.closed += 1


def connect_stub(adr, monkeypatch, db):
    """Replace only the connection boundary with the offline read-only double."""
    async def connect(*args):
        assert args == ('docs', 'probata', 'docs')
        return db
    monkeypatch.setattr(adr.sq, 'connect', connect)


@pytest.mark.parametrize('selector', [
    {'id': 'adr:propria_0100'}, {'ids': ['adr:propria_0100']},
    {'number': 100}, {'numbers': [100]},
])
@pytest.mark.parametrize('action', ['list', 'projections', 'verify'])
def test_selected_adr100_only_uses_actual_record_lookup(adr, monkeypatch, selector, action):
    db = ReadOnlyDB([record(1), record(100), record(200)])
    connect_stub(adr, monkeypatch, db)
    result = asyncio.run(adr.operation(action, selector))
    if action == 'list':
        assert [r['id'] for r in result['results']] == ['adr:propria_0100']
    else:
        assert [p['adr_id'] for p in result['projections']] == ['adr:propria_0100']
        assert '# ADR-0100:' in result['projections'][0]['content']
        assert result['projection_sync_required'] is True
    assert result['missing_ids'] == []
    assert db.calls[0] == ('SELECT * FROM type::record($id);', {'id': 'adr:propria_0100'})
    assert db.closed == 1


@pytest.mark.parametrize('action', ['list', 'projections', 'verify'])
def test_selection_union_deduplicates_and_reports_missing_actual_ids(adr, monkeypatch, action):
    db = ReadOnlyDB([record(100), record(2)])
    connect_stub(adr, monkeypatch, db)
    result = asyncio.run(adr.operation(action, {
        'id': 'adr:propria_0100', 'ids': ['adr:propria_0002'],
        'numbers': [100, 3], 'limit': 3,
    }))
    key, identity = ('results', 'id') if action == 'list' else ('projections', 'adr_id')
    assert [row[identity] for row in result[key]] == ['adr:propria_0100', 'adr:propria_0002']
    assert result['missing_ids'] == ['adr:propria_0003']
    reads = [params['id'] for sql, params in db.calls if sql == 'SELECT * FROM type::record($id);']
    assert reads == ['adr:propria_0100', 'adr:propria_0002', 'adr:propria_0003']


@pytest.mark.parametrize('action', ['list', 'projections'])
def test_migration_legacy_actual_id_is_preserved(adr, monkeypatch, action):
    legacy_id = 'adr:legacy_' + 'a1' * 16
    db = ReadOnlyDB([record(100, id=legacy_id)])
    connect_stub(adr, monkeypatch, db)
    selected = asyncio.run(adr.operation(action, {'id': legacy_id}))
    key, identity = ('results', 'id') if action == 'list' else ('projections', 'adr_id')
    assert selected[key][0][identity] == legacy_id
    assert selected['missing_ids'] == []
    # A numeric selector must never silently select a legacy row with that number.
    canonical = asyncio.run(adr.operation(action, {'number': 100}))
    assert canonical[key] == []
    assert canonical['missing_ids'] == ['adr:propria_0100']


@pytest.mark.parametrize('payload', [
    {'id': None}, {'id': True}, {'id': ''}, {'id': 'document:propria_0100'},
    {'id': 'adr:other_0100'}, {'id': 'adr:propria_100'},
    {'id': 'adr:propria_00100'}, {'id': 'adr:propria_0000'},
    {'id': 'adr:propria_1000000'}, {'id': 'adr:propria_0100; SELECT * FROM adr'},
    {'id': 'adr:legacy_' + 'A' * 32}, {'id': 'adr:legacy_' + 'a' * 31},
    {'ids': []}, {'ids': 'adr:propria_0100'}, {'ids': [100]},
    {'number': True}, {'number': False}, {'number': 0}, {'number': 1000000},
    {'number': '100'}, {'number': 100.0}, {'numbers': []}, {'numbers': [True]},
    {'numbers': tuple([100])}, {'numbers': list(range(1, 52))},
    {'ids': ['adr:propria_0100'] * 51},
    {'id': 'adr:propria_0100', 'numbers': list(range(1, 51))},
    {'number': 100, 'ids': []}, {'number': 100, 'id': 'bad'},
    {'limit': True}, {'limit': 0}, {'limit': 51}, {'limit': None}, {'limit': '1'},
    {'numbers': [1, 100], 'limit': 1},
])
@pytest.mark.parametrize('action', ['list', 'projections', 'verify'])
def test_invalid_selection_fails_before_connect(adr, monkeypatch, payload, action):
    async def forbidden(*args):
        raise AssertionError('Invalid selection opened a database connection')
    monkeypatch.setattr(adr.sq, 'connect', forbidden)
    with pytest.raises(ValueError):
        asyncio.run(adr.operation(action, payload))


@pytest.mark.parametrize('overrides', [
    {'project': 'other'}, {'number': True}, {'number': 0}, {'number': 101},
])
def test_selected_row_identity_is_verified(adr, monkeypatch, overrides):
    db = ReadOnlyDB([record(100, **overrides)])
    connect_stub(adr, monkeypatch, db)
    with pytest.raises(ValueError):
        asyncio.run(adr.operation('list', {'number': 100}))
    assert db.closed == 1


@pytest.mark.parametrize('found', [
    [record(101)], [record(100), record(100)],
])
def test_lookup_rejects_wrong_id_or_multiple_rows(adr, monkeypatch, found):
    db = ReadOnlyDB([])
    async def invalid_lookup(sql, params=None):
        assert sql == 'SELECT * FROM type::record($id);'
        return found
    monkeypatch.setattr(db, 'query', invalid_lookup)
    connect_stub(adr, monkeypatch, db)
    with pytest.raises(ValueError):
        asyncio.run(adr.operation('list', {'number': 100}))
    assert db.closed == 1


def test_selector_boundaries_and_fifty_exact_records(adr, monkeypatch):
    db = ReadOnlyDB([record(n) for n in range(1, 51)] + [record(999999)])
    connect_stub(adr, monkeypatch, db)
    result = asyncio.run(adr.operation('list', {'numbers': list(range(1, 51)), 'limit': 50}))
    assert len(result['results']) == len(db.calls) == 50
    assert adr.selection({'id': 'adr:propria_999999'}) == (['adr:propria_999999'], None)
    assert adr.selection({'number': 1}) == (['adr:propria_0001'], None)


@pytest.mark.parametrize('action,default_limit', [('list', 200), ('projections', 5001), ('verify', 5001)])
def test_unselected_legacy_bound_and_explicit_small_limit(adr, monkeypatch, action, default_limit):
    db = ReadOnlyDB([record(1), record(100)], allow_browse=True)
    connect_stub(adr, monkeypatch, db)
    legacy = asyncio.run(adr.operation(action))
    assert db.calls[0][1] == {'limit': default_limit}
    key = 'results' if action == 'list' else 'projections'
    assert len(legacy[key]) == 2
    assert 'missing_ids' not in legacy
    db.calls.clear()
    bounded = asyncio.run(adr.operation(action, {'limit': 1}))
    assert db.calls[0][1] == {'limit': 1}
    assert len(bounded[key]) == 1


def test_unselected_projection_corpus_limit_is_preserved(adr, monkeypatch):
    db = ReadOnlyDB([record(n) for n in range(1, 5002)], allow_browse=True)
    connect_stub(adr, monkeypatch, db)
    with pytest.raises(ValueError, match='Projection limit exceeded'):
        asyncio.run(adr.refresh_projections())
    assert db.closed == 1
