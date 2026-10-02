"""Give the platform database's objects a non-superuser owner, so schema work stops needing `ai`.

> _Byline: Claude Code · Opus 5.5 · 2026-10-02_

WHY. Owner 2026-10-02 (option A, then "first make sure everyone calling it has a user that gives
them all the permissions they need"): `ai` on probata-db is the bootstrap superuser and should
only be used from inside the database container. Every schema, table, view, function and type in
the `platform` database was owned by `ai`, so migrations and agent test work needed `ai` over the
network. The role design already had an owner role for this, `platform_migrator` (NOLOGIN, unused).

WHAT IT DOES (idempotent; re-running changes nothing that is already done):
  1. checks nothing depends on superuser rights it would lose (views reading extension tables
     without public access, publications);
  2. creates login `platform_dba` (CREATEDB, not a superuser, member of `platform_migrator`, and its
     sessions SET ROLE platform_migrator so new objects get the right owner); the password goes to
     ~/.secrets/probata-db.env before the role exists;
  3. gives `platform_migrator` CONNECT/CREATE/TEMPORARY on the database and `platform_dba` the
     throwaway test databases;
  4. in ONE server-side statement (a DO block, lock_timeout 3 s, so it either finishes at once or
     changes nothing): moves every non-extension schema, relation, function and type owned by `ai`
     to `platform_migrator`, and copies `ai`'s default privileges to `platform_migrator`, so tables
     created later still grant to platform_app as before;
  5. proves it: logs in as platform_dba, creates a probe table and reads its owner and grants,
     then rolls back.

Extension objects (pg_duckdb, vector, ...) stay with `ai`.

USAGE. `python tools/probata-db-platform-ownership.py [--wait N]`. A long import can hold row locks
on dozens of platform tables for many minutes; with `--wait N` the tool checks once a minute, for up
to N minutes, and moves ownership as soon as no transaction older than a minute holds platform tables.
"""
from __future__ import annotations

import pathlib
import re
import secrets
import sys
import time

import psycopg2
import psycopg2.errors

HOST, PORT = "100.91.190.107", 5432
SECRETS = pathlib.Path.home() / ".secrets" / "probata-db.env"
TEST_DATABASES = ("investigation_test_advocatio_20261002", "platform_baseline_test", "platform_preburn_20260830")

PRECHECK = """
select (select count(*) from pg_class v join pg_rewrite rw on rw.ev_class = v.oid
          join pg_depend d on d.classid = 'pg_rewrite'::regclass and d.objid = rw.oid and d.refclassid = 'pg_class'::regclass
          join pg_class t on t.oid = d.refobjid
         where v.relkind in ('v','m') and t.oid <> v.oid and pg_get_userbyid(v.relowner) = 'ai'
           and exists (select 1 from pg_depend e where e.classid = 'pg_class'::regclass and e.objid = t.oid and e.deptype = 'e')
           and not has_table_privilege('public', t.oid, 'SELECT')),
       (select count(*) from pg_publication)
"""

MOVE = """
DO $move$
DECLARE r record; n_s int := 0; n_r int := 0; n_f int := 0; n_t int := 0; n_d int := 0;
BEGIN
  PERFORM set_config('lock_timeout', '3s', true);
  FOR r IN SELECT n.nspname FROM pg_namespace n
            WHERE pg_get_userbyid(n.nspowner) = 'ai' AND n.nspname !~ '^(pg_|information_schema)'
              AND NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid = 'pg_namespace'::regclass AND d.objid = n.oid AND d.deptype = 'e')
  LOOP EXECUTE format('ALTER SCHEMA %I OWNER TO platform_migrator', r.nspname); n_s := n_s + 1; END LOOP;

  FOR r IN SELECT c.relkind, n.nspname, c.relname FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
            WHERE pg_get_userbyid(c.relowner) = 'ai' AND n.nspname !~ '^(pg_|information_schema)'
              AND c.relkind IN ('r','p','v','m','S','f')
              AND NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid = 'pg_class'::regclass AND d.objid = c.oid AND d.deptype = 'e')
              AND NOT (c.relkind = 'S' AND EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid = 'pg_class'::regclass
                                                     AND d.objid = c.oid AND d.deptype IN ('a','i')))
  LOOP EXECUTE format('ALTER %s %I.%I OWNER TO platform_migrator',
         CASE r.relkind WHEN 'v' THEN 'VIEW' WHEN 'm' THEN 'MATERIALIZED VIEW' WHEN 'S' THEN 'SEQUENCE'
                        WHEN 'f' THEN 'FOREIGN TABLE' ELSE 'TABLE' END, r.nspname, r.relname);
       n_r := n_r + 1; END LOOP;

  FOR r IN SELECT p.oid::regprocedure AS sig, p.prokind FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace
            WHERE pg_get_userbyid(p.proowner) = 'ai' AND n.nspname !~ '^(pg_|information_schema)'
              AND NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid = 'pg_proc'::regclass AND d.objid = p.oid AND d.deptype = 'e')
  LOOP EXECUTE format('ALTER %s %s OWNER TO platform_migrator',
         CASE r.prokind WHEN 'p' THEN 'PROCEDURE' WHEN 'a' THEN 'AGGREGATE' ELSE 'FUNCTION' END, r.sig);
       n_f := n_f + 1; END LOOP;

  FOR r IN SELECT t.oid::regtype AS typ, t.typtype FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
            WHERE pg_get_userbyid(t.typowner) = 'ai' AND n.nspname !~ '^(pg_|information_schema)'
              AND t.typrelid = 0 AND t.typelem = 0 AND t.typtype IN ('c','d','e','r','m')
              AND NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid = 'pg_type'::regclass AND d.objid = t.oid AND d.deptype = 'e')
  LOOP EXECUTE format('ALTER %s %s OWNER TO platform_migrator', CASE r.typtype WHEN 'd' THEN 'DOMAIN' ELSE 'TYPE' END, r.typ);
       n_t := n_t + 1; END LOOP;

  FOR r IN SELECT d.defaclnamespace AS nsp, d.defaclobjtype AS objtype, a.grantee,
                  string_agg(DISTINCT a.privilege_type, ', ') AS privs
             FROM pg_default_acl d, aclexplode(d.defaclacl) a
            WHERE d.defaclrole = 'ai'::regrole AND a.grantee <> 'ai'::regrole
            GROUP BY 1, 2, 3
  LOOP EXECUTE format('ALTER DEFAULT PRIVILEGES FOR ROLE platform_migrator %s GRANT %s ON %s TO %s',
         CASE WHEN r.nsp = 0 THEN '' ELSE format('IN SCHEMA %s', r.nsp::regnamespace) END,
         r.privs,
         CASE r.objtype WHEN 'r' THEN 'TABLES' WHEN 'S' THEN 'SEQUENCES' WHEN 'f' THEN 'FUNCTIONS'
                        WHEN 'T' THEN 'TYPES' ELSE 'SCHEMAS' END,
         CASE WHEN r.grantee = 0 THEN 'PUBLIC' ELSE quote_ident(r.grantee::regrole::text) END);
       n_d := n_d + 1; END LOOP;

  RAISE NOTICE 'moved % schemas, % tables/views/sequences, % functions, % types; copied % default grants',
               n_s, n_r, n_f, n_t, n_d;
END $move$;
"""

BUSY = """
select count(distinct a.pid) from pg_locks l join pg_stat_activity a on a.pid = l.pid
 where l.database = (select oid from pg_database where datname = 'platform') and l.relation is not null
   and a.pid <> pg_backend_pid() and a.xact_start < now() - interval '1 minute'
"""

LEFT = """
select count(*) from pg_class c join pg_namespace n on n.oid = c.relnamespace
 where pg_get_userbyid(c.relowner) = 'ai' and n.nspname !~ '^(pg_|information_schema)' and c.relkind in ('r','p','v','m','f')
   and not exists (select 1 from pg_depend d where d.classid = 'pg_class'::regclass and d.objid = c.oid and d.deptype = 'e')
"""


def read_secrets() -> dict[str, str]:
    out = {}
    for line in SECRETS.read_text(encoding="utf-8").splitlines():
        m = re.match(r"^\s*([A-Z_]+)\s*=\s*(.+?)\s*$", line)
        if m:
            out[m.group(1)] = m.group(2)
    return out


def connect(db: str, user: str, password: str):
    c = psycopg2.connect(host=HOST, port=PORT, user=user, password=password, dbname=db, connect_timeout=40)
    c.autocommit = True
    return c


def main(wait_minutes: int = 0) -> int:
    sec = read_secrets()
    admin = sec["PROBATA_DB_ADMIN_PASSWORD"]

    plat = connect("platform", "ai", admin)
    cur = plat.cursor()
    cur.execute(PRECHECK)
    risky, pubs = cur.fetchone()
    print(f"  pre-check: views reading extension tables without public access {risky}, publications {pubs}")
    if risky or pubs:
        print("  stopping: these need a closer look before ownership moves")
        return 1

    if "PLATFORM_DBA_PASSWORD" not in sec:
        pw = secrets.token_urlsafe(32)
        raw = SECRETS.read_bytes()
        nl = b"\r\n" if b"\r\n" in raw else b"\n"
        SECRETS.write_bytes(raw.rstrip(b"\r\n") + nl + nl.join([
            b"# Schema changes and test databases on probata-db. Sessions run as platform_migrator,"
            b" which owns the platform objects.",
            b"PLATFORM_DBA_USER=platform_dba",
            b"PLATFORM_DBA_PASSWORD=" + pw.encode(),
            f"PLATFORM_DBA_URL=postgresql://platform_dba:{pw}@{HOST}:{PORT}/platform".encode(),
        ]) + nl)
        sec["PLATFORM_DBA_PASSWORD"] = pw
        print("  platform_dba login saved to ~/.secrets/probata-db.env")
    pw = sec["PLATFORM_DBA_PASSWORD"]

    root = connect("postgres", "ai", admin)
    rc = root.cursor()
    rc.execute("select 1 from pg_roles where rolname = 'platform_dba'")
    if not rc.fetchone():
        rc.execute("CREATE ROLE platform_dba LOGIN NOSUPERUSER CREATEDB NOCREATEROLE IN ROLE platform_migrator PASSWORD %s", (pw,))
    rc.execute("ALTER ROLE platform_dba SET role = 'platform_migrator'")
    rc.execute("GRANT CONNECT, CREATE, TEMPORARY ON DATABASE platform TO platform_migrator")
    rc.execute("GRANT CONNECT ON DATABASE platform TO platform_dba")
    for db in TEST_DATABASES:
        rc.execute("select 1 from pg_database where datname = %s", (db,))
        if rc.fetchone():
            rc.execute(f'ALTER DATABASE "{db}" OWNER TO platform_dba')
    print("  login platform_dba ready (member of platform_migrator); test databases belong to it")

    # A long import holds row locks on dozens of platform tables for many minutes (2026-10-02: two
    # platform_runtime transactions, 18+ minutes, 51 and 75 tables). Ownership changes need each table
    # to itself for a moment, so wait until no transaction older than a minute holds platform locks.
    deadline = time.time() + wait_minutes * 60
    attempt = 0
    while True:
        cur.execute(BUSY)
        busy = cur.fetchone()[0]
        if busy:
            if time.time() >= deadline:
                print(f"  still {busy} long transaction(s) holding platform tables; nothing changed")
                return 1
            time.sleep(60)
            continue
        attempt += 1
        try:
            t0 = time.time()
            cur.execute(MOVE)
            note = plat.notices[-1].strip().replace("NOTICE:  ", "") if plat.notices else "done"
            print(f"  {note} ({time.time() - t0:.1f}s, attempt {attempt})", flush=True)
            break
        except psycopg2.errors.LockNotAvailable:
            print(f"  attempt {attempt}: a table stayed busy for 3 s; nothing changed", flush=True)
            if time.time() >= deadline:
                return 1
            time.sleep(30)
    cur.execute(LEFT)
    print(f"  platform tables/views still owned by ai: {cur.fetchone()[0]}")

    probe = psycopg2.connect(host=HOST, port=PORT, user="platform_dba", password=pw, dbname="platform", connect_timeout=40)
    pc = probe.cursor()
    pc.execute("select session_user, current_user")
    print(f"  platform_dba logs in; works as {pc.fetchone()}")
    pc.execute("create table working._dba_probe_20261002(x int)")
    pc.execute("select pg_get_userbyid(relowner), relacl::text from pg_class where oid = 'working._dba_probe_20261002'::regclass")
    owner, acl = pc.fetchone()
    print(f"  probe table owner {owner}, grants {acl} (rolled back)")
    probe.rollback()
    probe.close()
    return 0


if __name__ == "__main__":
    # --wait N: keep checking once a minute, for up to N minutes, until no long transaction holds platform tables.
    wait = int(sys.argv[sys.argv.index("--wait") + 1]) if "--wait" in sys.argv else 0
    sys.exit(main(wait))
