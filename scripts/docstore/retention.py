"""Guard the pinned CocoIndex connector's SQL at the canonical write boundary.

Only generated document DELETE statements are rewritten. Any unrecognized
document deletion or schema removal fails closed before submission.
"""
from __future__ import annotations
import re

_DELETE = re.compile(r'^DELETE (document:(?:[a-zA-Z0-9_]+|`[^`\n]+`));$')


def retained_sql(sql: str) -> str:
    output = []
    for line in sql.splitlines():
        statement = line.strip()
        match = _DELETE.fullmatch(statement)
        if match:
            rid = match[1]
            output.append(f'UPDATE {rid} SET status = "retracted";')
        elif re.match(r'(DELETE\s+(?:FROM\s+)?document\b|REMOVE\s+TABLE)', statement, re.I):
            raise ValueError('Canonical document deletion/schema removal prohibited')
        else:
            output.append(line)
    return '\n'.join(output)


class RetainingConnection:
    def __init__(self, connection):
        self.connection = connection

    async def query(self, sql, *args, **kwargs):
        return await self.connection.query(retained_sql(sql), *args, **kwargs)

    def __getattr__(self, key):
        return getattr(self.connection, key)


class RetainingFactory:
    def __init__(self, factory):
        self.factory = factory

    async def acquire(self):
        return RetainingConnection(await self.factory.acquire())
