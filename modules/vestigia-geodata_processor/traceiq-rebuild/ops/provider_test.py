# Byline: Claude Code · Opus 5 · 2026-07-25
# Test surface: probe every registered provider x every endpoint template.
#
# Answers "does each API actually work, and is the response shaped how we think?" without
# per-provider code — placeholders are discovered from the template itself and filled from
# REAL corpus fixtures, so a probe exercises the same input shape production will use.
#
# Cost safety: DRY-RUN IS THE DEFAULT. Nothing is called until --execute. Every executed
# probe is recorded in geo.api_call like any other call (ADR-0014 ledger).
#
#   python ops/provider_test.py                      # plan only: what would be called
#   python ops/provider_test.py --execute             # probe everything enabled
#   python ops/provider_test.py --execute --provider here
#   python ops/provider_test.py --execute --provider radar --template reverse
#   python ops/provider_test.py --execute --report ops/PROVIDER_TEST_REPORT.md
import argparse, json, os, re, sys, time, urllib.error, urllib.request
from datetime import datetime, timezone

import psycopg
from provider_http import build_request

DSN = os.environ.get("TRACEIQ_DSN_KV",
                     "host=100.119.96.29 port=5432 dbname=traceiq user=ai password=ai")

PLACEHOLDER = re.compile(r"\{(\w+)\}")

# Response fields we EXPECT per provider+template. Missing ones are reported, not fatal —
# a shape change is exactly what this surface exists to catch.
EXPECTED = {
    ("radar", "reverse"):  ["addresses"],
    ("here",  "reverse"):  ["items"],
    ("google", "place_details"): ["result", "status"],
    ("geoapify", "reverse"): ["results"],
}
# Where the useful payload lives, for a readable one-line summary per probe.
SUMMARY = {
    ("radar", "reverse"):  lambda d: (d.get("addresses") or [{}])[0].get("formattedAddress"),
    ("here",  "reverse"):  lambda d: (d.get("items") or [{}])[0].get("title"),
    ("google", "place_details"): lambda d: (d.get("result") or {}).get("formatted_address"),
    ("geoapify", "reverse"): lambda d: (d.get("results") or [{}])[0].get("formatted"),
}


def fixtures(cur):
    """Real corpus values for template placeholders — never invented coordinates."""
    f = {}
    cur.execute("""SELECT req_lat, req_lng FROM src.radar_api_batch
                   WHERE req_lat IS NOT NULL ORDER BY id LIMIT 1""")
    r = cur.fetchone()
    f["lat"], f["lng"] = (str(r[0]), str(r[1])) if r else ("43.0140430", "-83.7115422")

    cur.execute("""SELECT place_id FROM src.google_place_cache
                   WHERE NOT is_empty AND place_id LIKE 'ChIJ%' LIMIT 1""")
    r = cur.fetchone()
    f["place_id"] = r[0] if r else "ChIJezMgkaaHI4gRGrmYlbwztOg"

    # A real travelled path for route-match / directions probes: waypoints of one path event.
    cur.execute("""SELECT string_agg(lat || ',' || lng, '|' ORDER BY seq)
                   FROM (SELECT lat, lng, seq FROM working.waypoint
                         WHERE event_id = (SELECT event_id FROM working.waypoint
                                           GROUP BY event_id HAVING count(*) BETWEEN 5 AND 12
                                           LIMIT 1)
                         ORDER BY seq) t""")
    r = cur.fetchone()
    f["path"] = r[0] if r and r[0] else f"{f['lat']},{f['lng']}"
    pts = (f["path"] or "").split("|")
    f["origin"], f["destination"] = pts[0], pts[-1]
    f["query"] = "starbucks"
    f["mode"] = "car"
    f["units"] = "imperial"          # viewing standard
    return f


def redact(text, key):
    return text.replace(key, "<KEY>") if key else text


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--execute", action="store_true", help="actually call (default: plan only)")
    ap.add_argument("--provider")
    ap.add_argument("--template")
    ap.add_argument("--report", help="write a markdown report to this path")
    ap.add_argument("--include-disabled", action="store_true")
    a = ap.parse_args()

    rows_out = []
    with psycopg.connect(DSN) as conn, conn.cursor() as cur:
        fx = fixtures(cur)
        q = """SELECT p.provider_id, p.name, p.enabled, p.base_url, p.call_templates,
                      (SELECT count(*) FROM geo.provider_credential c
                       WHERE c.provider_id = p.provider_id AND c.status = 'active'),
                      (SELECT key_value FROM geo.provider_credential c
                       WHERE c.provider_id = p.provider_id AND c.status = 'active' LIMIT 1)
               FROM geo.provider p"""
        if not a.include_disabled:
            q += " WHERE p.enabled"
        q += " ORDER BY p.name"
        cur.execute(q)
        providers = cur.fetchall()

        print(f"fixtures: lat={fx['lat']} lng={fx['lng']} place_id={fx['place_id'][:18]}… "
              f"path={len(fx['path'].split('|'))} pts")
        print(f"{'provider':10s} {'template':14s} {'status':>7s} {'ms':>6s}  detail")
        print("-" * 92)

        for pid, name, enabled, base, tpls, ncred, key in providers:
            if a.provider and name != a.provider:
                continue
            tpls = tpls or {}
            if not tpls:
                rows_out.append((name, "—", "NO_TEMPLATE", None,
                                 "no call templates registered"))
                print(f"{name:10s} {'—':14s} {'SKIP':>7s} {'—':>6s}  no call templates registered")
                continue
            for tname, tpl in sorted(tpls.items()):
                if a.template and tname != a.template:
                    continue
                needed = set(PLACEHOLDER.findall(tpl.get("url", "")))
                for hv in (tpl.get("headers") or {}).values():
                    needed |= set(PLACEHOLDER.findall(str(hv)))
                needs_key = "key" in needed
                missing = sorted(n for n in needed if n != "key" and n not in fx)
                args = {k: v for k, v in fx.items() if k in needed}

                if missing:
                    detail = f"no fixture for {missing}"
                    rows_out.append((name, tname, "NO_FIXTURE", None, detail))
                    print(f"{name:10s} {tname:14s} {'SKIP':>7s} {'—':>6s}  {detail}")
                    continue
                if needs_key and not ncred:
                    detail = "template needs {key} but no active credential stored"
                    rows_out.append((name, tname, "NO_CRED", None, detail))
                    print(f"{name:10s} {tname:14s} {'SKIP':>7s} {'—':>6s}  {detail}")
                    continue
                if not a.execute:
                    shown = redact(tpl["url"].format(key="<KEY>", **args), None)
                    hdrs = ",".join((tpl.get("headers") or {}).keys()) or "-"
                    print(f"{name:10s} {tname:14s} {'PLAN':>7s} {'—':>6s}  "
                          f"{tpl.get('method','GET')} {shown[:60]} hdr[{hdrs}]")
                    rows_out.append((name, tname, "PLAN", None, shown[:120]))
                    continue

                t0 = time.time()
                try:
                    with urllib.request.urlopen(build_request(base, tpl, key, args), timeout=30) as r:
                        status, body = r.status, r.read().decode("utf-8", "replace")
                except urllib.error.HTTPError as e:
                    status, body = e.code, e.read().decode("utf-8", "replace")
                except Exception as e:                    # network/DNS/timeout
                    status, body = 0, str(e)
                ms = int((time.time() - t0) * 1000)

                try:
                    d = json.loads(body)
                    parsed = True
                except ValueError:
                    d, parsed = None, False

                notes = []
                if not parsed:
                    notes.append("NON-JSON")
                else:
                    exp = EXPECTED.get((name, tname), [])
                    absent = [f for f in exp if f not in d]
                    if absent:
                        notes.append(f"missing fields {absent}")
                    fn = SUMMARY.get((name, tname))
                    if fn:
                        try:
                            notes.append(f"-> {fn(d)!r}")
                        except Exception:
                            notes.append("-> (summary failed)")

                ok = 200 <= (status or 0) < 300 and parsed and not any(
                    n.startswith("missing") for n in notes)
                verdict = "PASS" if ok else "FAIL"
                detail = redact("; ".join(notes) or body[:80], key)
                rows_out.append((name, tname, f"{verdict} {status}", ms, detail))
                print(f"{name:10s} {tname:14s} {status:>7} {ms:>6}  {verdict}  {detail[:60]}")

                # cache_level describes WHERE the data came from (file|db|live), not why we
                # called — a probe is a live call, and its probe-ness is flagged in request.
                cur.execute("""INSERT INTO geo.api_call
                               (provider_id, template, request, response, http_status,
                                cache_level, cost_usd)
                               VALUES (%s,%s,%s,%s,%s,'live',%s)""",
                            (pid, tname,
                             json.dumps({"url_template": tpl["url"], "args": args,
                                         "probe": True}),
                             json.dumps(d if parsed else {"raw_text": body[:20000]}),
                             status, tpl.get("cost_usd", 0)))
                conn.commit()

    npass = sum(1 for r in rows_out if str(r[2]).startswith("PASS"))
    nfail = sum(1 for r in rows_out if str(r[2]).startswith("FAIL"))
    nskip = sum(1 for r in rows_out if r[2] in ("NO_CRED", "NO_FIXTURE", "NO_TEMPLATE"))
    print("-" * 92)
    print(f"{npass} pass, {nfail} fail, {nskip} skipped"
          + ("" if a.execute else "   (PLAN ONLY — re-run with --execute)"))

    if a.report:
        with open(a.report, "w", encoding="utf-8") as fh:
            fh.write("# Provider Endpoint Test Report\n\n")
            fh.write(f"Generated: {datetime.now(timezone.utc):%Y-%m-%d %H:%M} UTC  \n")
            fh.write(f"Mode: {'EXECUTE' if a.execute else 'PLAN ONLY'}\n\n")
            fh.write("| provider | template | result | ms | detail |\n|---|---|---|---:|---|\n")
            for name, tname, verdict, ms, detail in rows_out:
                safe = str(detail).replace("|", "\\|")[:160]
                fh.write(f"| {name} | {tname} | {verdict} | {ms if ms is not None else ''} | {safe} |\n")
            fh.write(f"\n**{npass} pass, {nfail} fail, {nskip} skipped**\n")
        print(f"report -> {a.report}")
    return 1 if nfail else 0


if __name__ == "__main__":
    sys.exit(main())
