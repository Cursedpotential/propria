"""0.8.1-r6 release test: the memory duplicate guard runs the real fn::remember on embedded SurrealDB.

Byline: Claude Code · Opus 5.5 · 2026-09-28. Installed as tests/test_memory_duplicate_guard.py by apply_r6.py.
The rule (scripts/docstore/schema/2026-09-28-memory-duplicate-guard-lexical.surql): a stored row conflicts if
cosine distance <= 0.10, or distance <= 0.20 with word Jaccard >= 0.35, or BM25 finds every claim word.
Cases mirror the live measurements of 2026-09-28: an unrelated claim at 0.17-0.19 with overlap 0.05-0.10 was
wrongly refused under the old cosine-only 0.20 rule; a reworded duplicate at 0.184 has overlap 0.56.
"""
import math
from pathlib import Path

import pytest
from surrealdb import AsyncSurreal

ROOT = Path(__file__).resolve().parents[1]
SCHEMA = ROOT / 'scripts/docstore/schema'
BASE = {'kind': 'preference', 'evidence': 'test', 'agent': 'test', 'scope': 'propria'}
STORED = "Owner rule: search with the owner's tools, never grep; check every memory lane first."


def at(distance, axis):
    """A unit vector at the given cosine distance from [1, 0, 0], leaning into one of two other axes."""
    c = 1 - distance
    s = math.sqrt(1 - c * c)
    return {'y': [c, s, 0.0], '-y': [c, -s, 0.0], 'z': [c, 0.0, s]}[axis]


def value(result):
    while isinstance(result, list) and len(result) == 1:
        result = result[0]
    return result


@pytest.fixture
async def memory():
    db = AsyncSurreal('mem://')
    await db.connect()
    await db.use('test', 'memory')
    await db.query('''
        DEFINE ANALYZER doc_text TOKENIZERS blank, class FILTERS lowercase;
        DEFINE TABLE memory SCHEMALESS;
        DEFINE INDEX memory_claim_ft ON memory FIELDS claim FULLTEXT ANALYZER doc_text BM25;
        DEFINE INDEX memory_vec ON memory FIELDS embedding HNSW DIMENSION 3 DIST COSINE;
        DEFINE TABLE decision_log SCHEMALESS;
    ''')
    for name in ('2026-09-27-memory-remember-guard.surql', '2026-09-28-memory-duplicate-guard-lexical.surql'):
        await db.query((SCHEMA / name).read_text())
    await db.query('CREATE memory SET claim = $c, scope = "propria", status = "active", embedding = [1.0, 0.0, 0.0];',
                   {'c': STORED})

    async def remember(**payload):
        return value(await db.query('RETURN fn::remember($p);', {'p': {**BASE, **payload}}))
    yield remember
    await db.close()


async def test_claim_words_lowercase_split_and_drop_short_words():
    db = AsyncSurreal('mem://')
    await db.connect()
    await db.use('test', 'words')
    await db.query((SCHEMA / '2026-09-28-memory-duplicate-guard-lexical.surql').read_text())
    words = value(await db.query('RETURN fn::claim_words($t);', {'t': "Owner's tools (read-memories/DuckDB), OK ok tools."}))
    await db.close()
    assert sorted(words) == ['duckdb', 'memories', 'owner', 'read', 'tools']


async def test_unrelated_claim_in_the_old_false_positive_band_is_written(memory):
    result = await memory(claim='When a permission classifier blocks an approved change, give the owner one command in a code block.',
                          embedding=at(0.17, 'z'))
    assert result['written'] and result['conflicts'] == []


async def test_reworded_duplicate_is_refused_with_distance_and_overlap(memory):
    result = await memory(claim="Owner rule: use the owner's search tools instead of grep and check each memory lane before deciding.",
                          embedding=at(0.18, 'y'))
    assert result['written'] is None
    conflict = result['conflicts'][0]
    assert conflict['claim'] == STORED and 0.17 < conflict['dist'] < 0.19 and conflict['overlap'] >= 0.35


async def test_same_meaning_in_other_words_is_refused_below_010(memory):
    result = await memory(claim='Agents must query their approved lookup utilities rather than raw pattern matching.',
                          embedding=at(0.08, '-y'))
    assert result['written'] is None and result['conflicts'][0]['overlap'] < 0.35


async def test_force_keeps_both(memory):
    result = await memory(claim="Owner rule: use the owner's search tools instead of grep and check each memory lane before deciding.",
                          embedding=at(0.18, 'y'), force=True)
    assert result['written'] and result['conflicts'] == []
