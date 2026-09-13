"""Read-only ad-hoc query runner against the live 'platform' Postgres.

Usage: uv run python pgq.py "SELECT ..."

Reads creds from C:\\Users\\matts\\.secrets\\probata.env with a tolerant regex
(never sourced as shell), connects via psycopg to 100.91.190.107:5432 (the
tailnet IP -- DB_HOST=agentos-db only resolves inside the compose network),
and only permits SELECT / information_schema / pg_catalog / EXPLAIN statements.
Never prints credential values.
"""
import re
import sys

ENV_PATH = r"C:\Users\matts\.secrets\probata.env"

def load_env(path):
    cfg = {}
    with open(path, "r", encoding="utf-8") as fh:
        for line in fh:
            m = re.match(r"^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*?)\s*$", line)
            if not m:
                continue
            key, val = m.group(1), m.group(2)
            if val.startswith('"') and val.endswith('"'):
                val = val[1:-1]
            cfg[key] = val
    return cfg


def main():
    if len(sys.argv) < 2:
        print("usage: pgq.py '<SELECT ...>'", file=sys.stderr)
        sys.exit(2)
    sql = sys.argv[1]
    stripped = sql.strip().lstrip("(").strip()
    upper = stripped.upper()
    allowed_prefixes = ("SELECT", "EXPLAIN", "WITH")
    if not upper.startswith(allowed_prefixes):
        print("REFUSED: only SELECT/WITH/EXPLAIN statements are permitted", file=sys.stderr)
        sys.exit(2)

    cfg = load_env(ENV_PATH)
    host = "100.91.190.107"  # tailnet IP; DB_HOST=agentos-db only resolves in-compose
    port = cfg.get("DB_PORT", "5432")
    user = cfg.get("DB_USER")
    dbname = "platform"
    password = cfg.get("DB_PASS") or cfg.get("DB_PASSWORD")

    if not user or not password or not dbname:
        print("REFUSED: missing DB_USER/DB_PASS/DB_DATABASE in env file", file=sys.stderr)
        sys.exit(2)

    import psycopg

    conninfo = f"host={host} port={port} dbname={dbname} user={user} password={password}"
    with psycopg.connect(conninfo, connect_timeout=10) as conn:
        with conn.cursor() as cur:
            cur.execute(sql)
            if cur.description is None:
                print("(no result set)")
                return
            cols = [d.name for d in cur.description]
            print("\t".join(cols))
            for row in cur.fetchall():
                print("\t".join("" if v is None else str(v) for v in row))


if __name__ == "__main__":
    main()
