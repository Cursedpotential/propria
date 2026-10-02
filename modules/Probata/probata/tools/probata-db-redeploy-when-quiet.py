"""Redeploy probata-db (Coolify app data-pg-files) only when no import is running, then check it came back.

> _Byline: Claude Code · Opus 5.5 · 2026-10-02_

WHY. A redeploy restarts PostgreSQL, which drops every client's connection (Temporal, Infisical,
ContextForge, the Probata services) and aborts any transaction in flight. A Proffer/context import
holds row locks on dozens of platform tables for many minutes (2026-10-02: 51 and 75 tables, 18+
minutes), so the restart waits for a quiet moment. Since 2026-10-02 `ai` refuses network logins, so
the check uses pg_locks, which every login may read, instead of pg_stat_activity's transaction times.

WHAT IT DOES
  1. Every minute, for up to --wait N minutes (default 40): counts sessions holding more than 10 table
     locks in `platform`. Zero means quiet.
  2. Deploys data-pg-files through the coolify-write plugin's own functions (the owner's rule: Coolify
     operations go through that plugin), then follows the deployment to finished or failed.
  3. Checks: the app reports running:healthy; `platform_dba` logs in; `duckdb.postgres_role` and the
     login rules survived; clients have reconnected; the container log since the restart has no
     "database ai does not exist" lines.

RUN (the plugin's environment, plus psycopg2):
  uv run --quiet --project E:/AI_Workspace/plugins/plugins/coolify-write --with psycopg2-binary \
      python tools/probata-db-redeploy-when-quiet.py [--wait N]
"""
from __future__ import annotations

import importlib.util
import pathlib
import re
import sys
import time

import psycopg2

APP = "w10gg3an43jvry4y79n6sxi1"  # Coolify app data-pg-files = container probata-db
HOST, PORT = "100.91.190.107", 5432
PLUGIN = pathlib.Path("E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py")
SECRETS = pathlib.Path.home() / ".secrets" / "probata-db.env"

BUSY = """
select count(*) from (
  select l.pid from pg_locks l join pg_database d on d.oid = l.database
   where d.datname = 'platform' and l.relation is not null and l.pid <> pg_backend_pid()
   group by l.pid having count(*) > 10) busy
"""


def secret(name: str) -> str:
    for line in SECRETS.read_text(encoding="utf-8").splitlines():
        m = re.match(r"^\s*([A-Z_]+)\s*=\s*(.+?)\s*$", line)
        if m and m.group(1) == name:
            return m.group(2)
    raise KeyError(name)


def dba(db: str = "platform"):
    c = psycopg2.connect(host=HOST, port=PORT, user="platform_dba", password=secret("PLATFORM_DBA_PASSWORD"),
                         dbname=db, connect_timeout=40)
    c.autocommit = True
    return c


def plugin():
    spec = importlib.util.spec_from_file_location("coolify_write", PLUGIN)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def main(wait_minutes: int) -> int:
    deadline = time.time() + wait_minutes * 60
    while True:
        with dba() as c, c.cursor() as cur:
            cur.execute(BUSY)
            busy = cur.fetchone()[0]
        if not busy:
            break
        if time.time() >= deadline:
            print(f"  still busy after {wait_minutes} min ({busy} session(s) holding many platform tables); not deployed")
            return 1
        time.sleep(60)
    print(f"  quiet at {time.strftime('%H:%M:%S')}: no import holding platform tables", flush=True)

    cw = plugin()
    started = time.time()
    answer = cw.deploy_application(APP)
    dep = (answer.get("deployments") or [{}])[0].get("deployment_uuid") if isinstance(answer, dict) else None
    print(f"  deployment {dep or answer}", flush=True)
    status = None
    while dep and time.time() - started < 1200:
        time.sleep(15)
        status = (cw.get_deployment(dep) or {}).get("status")
        if status in ("finished", "failed", "cancelled", "cancelled-by-user"):
            break
    print(f"  deployment status after {time.time() - started:.0f}s: {status}", flush=True)
    if status != "finished":
        return 1

    app_status = None
    for _ in range(20):
        app_status = cw.get_application(APP).get("status")
        if app_status == "running:healthy":
            break
        time.sleep(6)
    print(f"  app status: {app_status}")

    with dba("postgres") as c, c.cursor() as cur:
        cur.execute("select pg_postmaster_start_time()::timestamp(0), current_setting('duckdb.postgres_role', true)")
        start, role = cur.fetchone()
        cur.execute("select rolvaliduntil from pg_roles where rolname = 'ai'")
        valid = cur.fetchone()[0]
        cur.execute("select usename, count(*) from pg_stat_activity where usename is not null group by 1 order by 1")
        sessions = cur.fetchall()
    print(f"  server started {start} UTC; duckdb.postgres_role = {role}; ai password valid until {valid}")
    print(f"  sessions by login: {sessions}")

    time.sleep(20)
    log = (cw.get_application_logs(APP, lines=400).get("logs") or "").splitlines()
    since = [l for l in log if l[:19] >= str(start)]
    noisy = sum("database \"ai\" does not exist" in l for l in since)
    fatal = [l.split("] ", 1)[-1][:110] for l in since if " FATAL: " in l and "database \"ai\" does not exist" not in l]
    print(f"  log since restart: {len(since)} lines, 'database ai does not exist' x{noisy}, other FATAL: {fatal[:4] or 'none'}")
    return 0 if app_status == "running:healthy" and noisy == 0 else 1


if __name__ == "__main__":
    wait = int(sys.argv[sys.argv.index("--wait") + 1]) if "--wait" in sys.argv else 40
    sys.exit(main(wait))
