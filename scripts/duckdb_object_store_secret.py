"""Emit SQL that gives pg_duckdb one NAMED, bucket-SCOPED S3 secret for an object store.

Byline: Claude Code · Fable 5.1 · 2026-09-20

Why: pg_duckdb reads only ``s3://`` URLs. Every store the engine is configured
for (``OBJECT_STORES_JSON``: r2, b2, ...) therefore needs a DuckDB secret, and
two S3-compatible providers can only coexist when each secret is scoped to its
own bucket — DuckDB picks the secret whose scope is the longest prefix of the
URL. The 94 unnamed ``simple_s3_secret[_N]`` servers found live on 2026-09-20
all carry the generic scope ``s3://`` and point at R2; a bucket-scoped secret
wins over them for its bucket and leaves them untouched.

A pg_duckdb secret is a FOREIGN SERVER (endpoint/region/scope) plus one USER
MAPPING per PostgreSQL role (key id/secret). The mapping is per role, so every
role that runs DuckDB reads must be listed (the engine connects as
``platform_runtime``, not as the owner role).

The script prints SQL on STDOUT and nothing else; it never prints to a
terminal by accident because the caller pipes it straight into psql:

    python3 scripts/duckdb_object_store_secret.py \
        --name platform_b2_salem_data --bucket salem-data \
        --credentials /data/probata/secrets/casebible-b2.json \
        --role ai --role platform_runtime --role platform_api \
      | docker exec -i <probata-db> psql -U ai -d platform -v ON_ERROR_STOP=1 -q

Re-running replaces the secret (key rotation = re-run). Credential file shape
is the one the engine already reads: ``access_key_id``, ``secret_access_key``,
``endpoint_url``, ``region``.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from urllib.parse import urlparse


def _lit(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def _ident(value: str) -> str:
    if not re.fullmatch(r"[a-z_][a-z0-9_]{0,62}", value):
        raise SystemExit(f"not a plain SQL identifier: {value!r}")
    return value


def build_sql(name: str, bucket: str, credentials: dict[str, str], roles: list[str]) -> str:
    endpoint = urlparse(credentials["endpoint_url"])
    host = endpoint.netloc or endpoint.path
    use_ssl = "false" if endpoint.scheme == "http" else "true"
    server = _ident(name)
    lines = [
        "BEGIN;",
        "SET LOCAL log_statement = 'none';",
        f"DROP SERVER IF EXISTS {server} CASCADE;",
        f"CREATE SERVER {server} TYPE 's3' FOREIGN DATA WRAPPER duckdb OPTIONS ("
        f"endpoint {_lit(host)}, region {_lit(credentials['region'])}, url_style 'path', "
        f"use_ssl {_lit(use_ssl)}, scope {_lit('s3://' + bucket)});",
    ]
    for role in roles:
        role = _ident(role)
        lines.append(f"GRANT USAGE ON FOREIGN SERVER {server} TO {role};")
        lines.append(
            f"CREATE USER MAPPING FOR {role} SERVER {server} OPTIONS ("
            f"key_id {_lit(credentials['access_key_id'])}, secret {_lit(credentials['secret_access_key'])});"
        )
    lines.append("COMMIT;")
    return "\n".join(lines) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.split("\n", 1)[0])
    parser.add_argument("--name", required=True)
    parser.add_argument("--bucket", required=True)
    parser.add_argument("--credentials", required=True)
    parser.add_argument("--role", action="append", required=True)
    args = parser.parse_args()
    if sys.stdout.isatty():
        raise SystemExit("refusing to print credentials to a terminal; pipe into psql")
    with open(args.credentials, encoding="utf-8") as handle:
        credentials = json.load(handle)
    sys.stdout.write(build_sql(args.name, args.bucket, credentials, args.role))


if __name__ == "__main__":
    main()
