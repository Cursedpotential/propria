"""Mandatory bounded DuckDB normalization after retrieval; never a vector store."""
from __future__ import annotations
import hashlib
import json
import math
import duckdb


def pack(rows: list[dict], *, limit: int = 20, budget: int = 8000) -> dict:
    if not 1 <= limit <= 200 or not 256 <= budget <= 64000 or len(rows) > 1000:
        raise ValueError('Context bounds exceeded')
    normalized = []
    for ordinal, source in enumerate(rows):
        row = {k: v for k, v in source.items() if k not in {'embedding', 'vector'}}
        text = str(row.get('snippet') or row.get('text') or row.get('content') or row.get('body') or '')
        text = ' '.join(text.split())[:4000]
        for key in ('body', 'content', 'text'):
            row.pop(key, None)
        row['snippet'] = text
        origin = str(row.get('source') or 'docstore')
        identity = str(row.get('source_record_id') or row.get('id') or '')
        # Equal prose from independent sources is corroboration, not duplicate identity.
        key = origin + ':' + identity if identity else origin + ':' + hashlib.sha256(json.dumps(row, sort_keys=True, default=str).encode()).hexdigest()
        row.setdefault('provenance', {'source': origin, 'record_id': identity, 'path': row.get('path')})
        encoded = json.dumps(row, ensure_ascii=False, default=str)
        score = float(row.get('score') or 0)
        if not math.isfinite(score):
            score = 0.0
        normalized.append((ordinal, key, str(row.get('path') or identity or key), score, encoded, len(encoded.encode('utf-8'))))
    if sum(r[5] for r in normalized) > 8 * 1024 * 1024:
        raise ValueError('Context input exceeds 8 MiB')
    with duckdb.connect(':memory:', config={'threads': '1', 'memory_limit': '64MB', 'enable_external_access': 'false'}) as db:
        db.execute('CREATE TABLE candidates (ordinal INTEGER, identity VARCHAR, source_path VARCHAR, score DOUBLE, payload VARCHAR, cost INTEGER)')
        if normalized:
            db.executemany('INSERT INTO candidates VALUES (?, ?, ?, ?, ?, ?)', normalized)
        selected = db.execute('''WITH unique_rows AS (
            SELECT * FROM candidates QUALIFY row_number() OVER (PARTITION BY identity ORDER BY score DESC, ordinal) = 1
        ), diverse AS (
            SELECT *, row_number() OVER (PARTITION BY source_path ORDER BY score DESC, ordinal) AS source_rank FROM unique_rows
        ) SELECT payload, cost FROM diverse ORDER BY source_rank, score DESC, ordinal''').fetchall()
    results, consumed = [], 0
    for payload, cost in selected:
        if consumed + cost <= budget and len(results) < limit:
            results.append(json.loads(payload))
            consumed += cost
    return {'results': results, 'packing': {'engine': 'duckdb', 'input_rows': len(rows), 'returned_rows': len(results),
            'omitted_rows': len(rows) - len(results), 'context_bytes': consumed, 'budget_bytes': budget,
            'estimated_tokens': (consumed + 3) // 4, 'token_estimate_only': True}}
