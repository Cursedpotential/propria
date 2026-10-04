"""Exercise catalog streaming EOF and incremental consumption without external services."""

import asyncio

import pyarrow as pa
import pytest

from casebible_index import catalog_source


@pytest.mark.asyncio
@pytest.mark.parametrize("rows", [[], [{"key": "proof/a.png", "size": 7, "sha1": "a" * 40}]])
async def test_catalog_stream_reaches_eof_and_closes_connection(monkeypatch, rows):
    schema = pa.schema([("key", pa.string()), ("size", pa.int64()), ("sha1", pa.string())])
    table = pa.Table.from_pylist(rows, schema=schema)

    class Connection:
        closed = False

        def execute(self, query):
            return self

        def fetch_record_batch(self, batch_rows):
            assert batch_rows == 1
            return table.to_reader(max_chunksize=batch_rows)

        def close(self):
            self.closed = True

    connection = Connection()
    monkeypatch.setattr(catalog_source.duckdb, "connect", lambda: connection)
    monkeypatch.setattr(catalog_source, "_attach", lambda *_: None)

    async def consume():
        return [
            item
            async for item in catalog_source.iter_catalog_objects("fixture", "SELECT", batch_rows=1)
        ]

    objects = await asyncio.wait_for(consume(), timeout=2)
    assert [item.key for item in objects] == [row["key"] for row in rows]
    assert connection.closed
