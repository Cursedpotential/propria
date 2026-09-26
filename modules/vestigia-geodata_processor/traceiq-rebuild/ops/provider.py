# Byline: Claude Code · Fable 5 · 2026-07-24
# Provider management CLI (ADR-0014): add providers, keys, and API call templates as
# DATA — no code per provider. Every executed call persists to geo.api_call (evidence).
#
#   python provider.py add locationiq --functions geocode_reverse --base-url https://us1.locationiq.com
#   python provider.py key locationiq MAIN_KEY            (value read from stdin, never argv)
#   python provider.py template locationiq reverse GET "/v1/reverse?key={key}&lat={lat}&lon={lng}&format=json"
#   python provider.py call locationiq reverse lat=43.0126 lng=-83.6875
#   python provider.py list
import argparse, getpass, json, os, sys, urllib.request, urllib.error
import psycopg
from provider_http import build_request

DSN = os.environ.get("TRACEIQ_DSN_KV",
                     "host=100.119.96.29 port=5432 dbname=traceiq user=ai password=ai")


p = argparse.ArgumentParser()
sub = p.add_subparsers(dest="cmd", required=True)
pa = sub.add_parser("add"); pa.add_argument("name"); pa.add_argument("--functions", required=True)
pa.add_argument("--base-url"); pa.add_argument("--notes")
pk = sub.add_parser("key"); pk.add_argument("provider"); pk.add_argument("key_name")
pt = sub.add_parser("template"); pt.add_argument("provider"); pt.add_argument("tname")
pt.add_argument("method"); pt.add_argument("url_template"); pt.add_argument("--cost", type=float, default=0)
pt.add_argument("--header", action="append", metavar="'Name: value'",
                help="auth/extra header; {key} is substituted from the stored credential. "
                     "Repeatable. Needed for Radar (Authorization: <key>).")
pc = sub.add_parser("call"); pc.add_argument("provider"); pc.add_argument("tname")
pc.add_argument("kv", nargs="*", help="lat=.. lng=.. etc")
pb = sub.add_parser("batch", help="run template once per row of a SQL query")
pb.add_argument("provider"); pb.add_argument("tname")
pb.add_argument("--sql", required=True, help="query whose column names feed the {placeholders}")
pb.add_argument("--limit", type=int, default=100)
pb.add_argument("--execute", action="store_true", help="without this: dry-run (row count + cost estimate only)")
sub.add_parser("list")
a = p.parse_args()

with psycopg.connect(DSN) as conn, conn.cursor() as cur:
    if a.cmd == "add":
        cur.execute("""INSERT INTO geo.provider (name, functions, base_url, notes)
                       VALUES (%s, string_to_array(%s, ','), %s, %s)
                       ON CONFLICT (name) DO UPDATE SET functions=EXCLUDED.functions,
                         base_url=EXCLUDED.base_url, notes=COALESCE(EXCLUDED.notes, geo.provider.notes)
                       RETURNING provider_id""", (a.name, a.functions, a.base_url, a.notes))
        print(f"provider '{a.name}' -> id {cur.fetchone()[0]}")
    elif a.cmd == "key":
        val = getpass.getpass(f"paste {a.key_name} for {a.provider} (hidden): ") if sys.stdin.isatty() else sys.stdin.readline().strip()
        cur.execute("""INSERT INTO geo.provider_credential (provider_id, key_name, key_value)
                       SELECT provider_id, %s, %s FROM geo.provider WHERE name=%s
                       ON CONFLICT (provider_id, key_name)
                       DO UPDATE SET key_value=EXCLUDED.key_value, status='active', rotated_at=now()
                       RETURNING credential_id""", (a.key_name, val, a.provider))
        cid = cur.fetchone()[0]
        cur.execute("INSERT INTO geo.credential_audit (credential_id, changed_by, change) VALUES (%s,'owner-cli','set/rotated')", (cid,))
        print(f"credential {a.key_name} stored for {a.provider} (value not echoed)")
    elif a.cmd == "template":
        # --header 'Name: value' may repeat; {key} inside a header value is substituted at
        # call time. Radar authenticates via `Authorization: <raw key>` (no Bearer prefix),
        # which cannot be expressed as a URL param — HERE/Google take ?apiKey=/?key= instead,
        # so auth style stays DATA per ADR-0014 rather than code per provider.
        hdrs = {}
        for h in (a.header or []):
            if ":" not in h:
                sys.exit(f"--header must be 'Name: value', got {h!r}")
            hk, hv = h.split(":", 1)
            hdrs[hk.strip()] = hv.strip()
        tpl_body = {"method": a.method.upper(), "url": a.url_template, "cost_usd": a.cost}
        if hdrs:
            tpl_body["headers"] = hdrs
        tpl_patch = json.dumps({a.tname: tpl_body})
        cur.execute("""UPDATE geo.provider SET call_templates = call_templates || %s::jsonb
                       WHERE name=%s RETURNING name""", (tpl_patch, a.provider))
        print("template saved:", cur.fetchone()[0], "/", a.tname)
    elif a.cmd == "call":
        cur.execute("""SELECT p.provider_id, p.base_url, p.call_templates->%s,
                              (SELECT key_value FROM geo.provider_credential c
                               WHERE c.provider_id=p.provider_id AND c.status='active' LIMIT 1)
                       FROM geo.provider p WHERE p.name=%s AND p.enabled""", (a.tname, a.provider))
        row = cur.fetchone()
        if not row or not row[2]:
            sys.exit(f"no enabled provider '{a.provider}' with template '{a.tname}'")
        pid, base, tpl, key = row
        args = dict(kv.split("=", 1) for kv in a.kv)
        req = build_request(base, tpl, key, args)
        status, body = None, None
        try:
            with urllib.request.urlopen(req, timeout=30) as r:
                status, body = r.status, r.read().decode("utf-8", "replace")
        except urllib.error.HTTPError as e:
            status, body = e.code, e.read().decode("utf-8", "replace")
        try:
            body_json = json.dumps(json.loads(body))
        except Exception:
            body_json = json.dumps({"raw_text": body[:20000]})
        cur.execute("""INSERT INTO geo.api_call (provider_id, template, request, response, http_status, cache_level, cost_usd)
                       VALUES (%s,%s,%s,%s,%s,'live',%s) RETURNING api_call_id""",
                    (pid, a.tname, json.dumps({"url_template": tpl["url"], "args": args}),
                     body_json, status, tpl.get("cost_usd", 0)))
        print(f"api_call {cur.fetchone()[0]} recorded (HTTP {status}); response preview:")
        print(body[:400])
    elif a.cmd == "batch":
        cur.execute("""SELECT p.provider_id, p.base_url, p.call_templates->%s,
                              (SELECT key_value FROM geo.provider_credential c
                               WHERE c.provider_id=p.provider_id AND c.status='active' LIMIT 1)
                       FROM geo.provider p WHERE p.name=%s AND p.enabled""", (a.tname, a.provider))
        row = cur.fetchone()
        if not row or not row[2]:
            sys.exit(f"no enabled provider '{a.provider}' with template '{a.tname}'")
        pid, base, tpl, key = row
        cur.execute(f"SELECT * FROM ({a.sql}) q LIMIT %s", (a.limit,))
        cols = [d.name for d in cur.description]
        rows = cur.fetchall()
        est = len(rows) * float(tpl.get("cost_usd", 0))
        print(f"query returns {len(rows)} rows (limit {a.limit}); template {a.provider}/{a.tname}; est cost ${est:.4f}")
        if not a.execute:
            print("DRY-RUN (default). Re-run with --execute to make the calls."); sys.exit(0)
        ok = fail = 0
        for r in rows:
            args = dict(zip(cols, (str(v) for v in r)))
            try:
                with urllib.request.urlopen(build_request(base, tpl, key, args), timeout=30) as resp:
                    status, body = resp.status, resp.read().decode("utf-8", "replace")
            except urllib.error.HTTPError as e:
                status, body = e.code, e.read().decode("utf-8", "replace")
            except Exception as e:
                status, body = 0, str(e)
            try:
                body_json = json.dumps(json.loads(body))
            except Exception:
                body_json = json.dumps({"raw_text": body[:20000]})
            cur.execute("""INSERT INTO geo.api_call (provider_id, template, request, response, http_status, cache_level, cost_usd)
                           VALUES (%s,%s,%s,%s,%s,'live',%s)""",
                        (pid, a.tname, json.dumps({"url_template": tpl["url"], "args": args}),
                         body_json, status, tpl.get("cost_usd", 0)))
            ok, fail = (ok + 1, fail) if status and 200 <= status < 300 else (ok, fail + 1)
            conn.commit()   # each call is evidence the moment it happens
        print(f"batch done: {ok} ok, {fail} failed — all recorded in geo.api_call")
    else:
        for r in cur.execute("""SELECT p.name, p.enabled, array_to_string(p.functions, '+'),
                                       COALESCE(jsonb_object_keys_agg.k, '-') AS templates,
                                       COALESCE(cr.n, 0) AS keys,
                                       COALESCE(ac.n, 0) AS calls_made
                                FROM geo.provider p
                                LEFT JOIN LATERAL (SELECT string_agg(k, ',') AS k FROM jsonb_object_keys(p.call_templates) k) jsonb_object_keys_agg ON true
                                LEFT JOIN (SELECT provider_id, count(*) n FROM geo.provider_credential WHERE status='active' GROUP BY 1) cr USING (provider_id)
                                LEFT JOIN (SELECT provider_id, count(*) n FROM geo.api_call GROUP BY 1) ac USING (provider_id)
                                ORDER BY p.name""").fetchall():
            print(" | ".join(str(x) for x in r))
    conn.commit()
