# Byline: Claude Code · Fable 5 · 2026-07-24
# Owner flagging CLI (interim until the thin web UI): label places, flag anomalies/
# concerning locations, annotate anything. Writes only to ref.* (never raw/working).
#
#   python flag.py place ChIJezMgkaaHI4gRGrmY... --label "Address X" --category residence
#   python flag.py place --r4 43.0126,-83.6875 --label "The bar" --category bar --concerning --severity 4
#   python flag.py note event 2403151122a --category anomaly_confirmed --note "confirmed: GPS glitch"
#   python flag.py note day 2020-06-15 --category question --note "why no data this month?"
#   python flag.py list            # show all labels + annotations
import argparse, psycopg

DSN = "host=100.119.96.29 port=5432 dbname=traceiq user=ai password=ai"

p = argparse.ArgumentParser()
sub = p.add_subparsers(dest="cmd", required=True)
pp = sub.add_parser("place")
pp.add_argument("place_id", nargs="?")
pp.add_argument("--r4")
pp.add_argument("--label", required=True)
pp.add_argument("--business")
pp.add_argument("--tags", help="comma list: home_base,work,relative,personal_interest,bar,...")
pp.add_argument("--concerning", action="store_true")
pp.add_argument("--watch", action="store_true")
pp.add_argument("--severity", type=int)
pp.add_argument("--radius", type=int, help="alert radius meters")
pp.add_argument("--note")
pn = sub.add_parser("note")
pn.add_argument("kind", choices=["event", "waypoint", "day", "place", "route", "other"])
pn.add_argument("ref")
pn.add_argument("--category", required=True)
pn.add_argument("--label")
pn.add_argument("--note")
sub.add_parser("list")
a = p.parse_args()

with psycopg.connect(DSN) as conn, conn.cursor() as cur:
    if a.cmd == "place":
        lat = lng = None
        if a.r4:
            lat, lng = (x.strip() for x in a.r4.split(","))
        tags = [t.strip() for t in (a.tags or "").split(",") if t.strip()]
        cur.execute("""INSERT INTO ref.known_place
            (place_id, lat_r4, lng_r4, label, business_name, tags, is_concerning, is_watch,
             severity, alert_radius_m, note)
            VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s) RETURNING known_place_id""",
            (a.place_id, lat, lng, a.label, a.business, tags, a.concerning, a.watch,
             a.severity, a.radius, a.note))
        print("known place", cur.fetchone()[0], "saved:", a.label, tags)
    elif a.cmd == "note":
        cur.execute("""INSERT INTO ref.annotation (target_kind, target_ref, category, label, note)
            VALUES (%s,%s,%s,%s,%s) RETURNING annotation_id""",
            (a.kind, a.ref, a.category, a.label, a.note))
        print("annotation", cur.fetchone()[0], "saved on", a.kind, a.ref)
    else:
        for r in cur.execute("""SELECT 'label', place_label_id::text, COALESCE(place_id, lat_r4||','||lng_r4),
                                       label || COALESCE(' ['||category||']',''), concerning::text
                                FROM ref.place_label
                                UNION ALL
                                SELECT 'note', annotation_id::text, target_kind||':'||target_ref,
                                       category || COALESCE(': '||note,''), ''
                                FROM ref.annotation ORDER BY 1,2""").fetchall():
            print(" | ".join(x or "" for x in r))
    conn.commit()
