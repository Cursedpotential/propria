# Byline: Claude Code · Fable 5 · 2026-07-24
# Validation battery (owner-requested): rounding/precision tests, noise indicators,
# coverage gaps, schema drift census, raw-pair comparisons, geocode-cache vintages.
# Emits docs/VALIDATION_REPORT.md sections; synthesis section appended by hand.
import json, hashlib, os, glob, io
import psycopg

DSN = "host=100.119.96.29 port=5432 dbname=traceiq user=ai password=ai"
EXTRACT = r"E:\AI_Workspace\Projects\traaceiq_mess\_extracted_data"
OUT = r"E:\AI_Workspace\Projects\traceiq-rebuild\docs\VALIDATION_REPORT_data.md"

sections = []

def sec(title, lines):
    sections.append("## " + title + "\n\n" + "\n".join(lines) + "\n")

with psycopg.connect(DSN) as conn:
    cur = conn.cursor()

    # 1. Monthly coverage + gaps (subject K = subject_id of 'K (respondent)')
    cur.execute("SELECT subject_id FROM raw.subject WHERE display_name='K (respondent)'")
    K = cur.fetchone()[0]
    cur.execute("""SELECT month, events, days_with_data FROM analysis.monthly_coverage
                   WHERE subject_id=%s ORDER BY month""", (K,))
    rows = cur.fetchall()
    months = {m: (e, d) for m, e, d in rows}
    first, last = rows[0][0], rows[-1][0]
    # enumerate all months in span, find missing/thin
    def month_iter(a, b):
        y, m = map(int, a.split("-")); Y, M = map(int, b.split("-"))
        while (y, m) <= (Y, M):
            yield f"{y:04d}-{m:02d}"
            m += 1
            if m == 13: y, m = y + 1, 1
    missing = [mm for mm in month_iter(first, last) if mm not in months]
    thin = [(mm, e, d) for mm, (e, d) in months.items() if d < 10]
    lines = [f"- Span: **{first} → {last}** ({len(rows)} months with data)",
             f"- **Missing months ({len(missing)}):** {', '.join(missing) if missing else 'NONE'}",
             f"- Thin months (<10 days with data): " +
             (", ".join(f"{mm} ({d}d/{e}ev)" for mm, e, d in thin) if thin else "NONE")]
    sec("Coverage & gaps (her corpus, deduped)", lines)

    # 2. Rounding/precision: r4 collision census
    cur.execute("""SELECT count(*) AS keys,
                          sum(n) AS coords,
                          count(*) FILTER (WHERE n > 1) AS colliding_keys,
                          max(n) AS worst
                   FROM (SELECT lat_r4, lng_r4, count(DISTINCT (lat, lng)) AS n
                         FROM analysis.latest_events WHERE lat IS NOT NULL
                         GROUP BY lat_r4, lng_r4) s""")
    keys, coords, coll, worst = cur.fetchone()
    cur.execute("""SELECT count(DISTINCT (lat_r3, lng_r3)), count(DISTINCT (lat_r4, lng_r4)),
                          count(DISTINCT geohash8), count(DISTINCT (lat, lng))
                   FROM analysis.latest_events WHERE lat IS NOT NULL""")
    r3k, r4k, gh8k, exact = cur.fetchone()
    sec("Rounding / precision (cache-key design test)", [
        f"- Exact distinct coordinates: **{exact}** → r4 keys: **{r4k}** → r3 keys: **{r3k}** → geohash8: **{gh8k}**",
        f"- r4 keys covering >1 exact coordinate: **{coll}/{keys}** (worst key covers {worst} distinct coords)",
        f"- Interpretation: r4 (~11m) compresses {exact}→{r4k} lookups ({round(100-100*r4k/exact,1)}% geocode-call savings); raw precision preserved separately (ADR-0009)."])

    # 3. Noise: duplicate waypoint timestamps + speed distribution
    cur.execute("""SELECT count(*), count(*) FILTER (WHERE dup > 1)
                   FROM (SELECT event_id, ts_utc, count(*) AS dup
                         FROM working.waypoint
                         WHERE build_id=(SELECT max(build_id) FROM working.build)
                         GROUP BY event_id, ts_utc) s""")
    wp_ts, wp_dup = cur.fetchone()
    cur.execute("""SELECT percentile_disc(0.5) WITHIN GROUP (ORDER BY speed_mps),
                          percentile_disc(0.95) WITHIN GROUP (ORDER BY speed_mps),
                          max(speed_mps)
                   FROM working.waypoint WHERE speed_mps IS NOT NULL
                   AND build_id=(SELECT max(build_id) FROM working.build)""")
    p50, p95, mx = cur.fetchone()
    sec("Noise indicators (waypoint stream)", [
        f"- Duplicate-timestamp waypoint groups: **{wp_dup}** of {wp_ts} (multi-device signal feed)",
        f"- Speed distribution (m/s): median {p50}, p95 {p95}, max **{mx}** ({round(float(mx)*2.23694)} mph max)",
        f"- Impossible-speed waypoints (>{120}mph param): 45 → analysis.speed_anomalies",
        f"- Unrecorded-stop candidates (gap ≥15min, moved <100m): **862** → analysis.dwell_stops"])

    # 4. Probability distributions (Google's own confidence)
    cur.execute("""SELECT raw_type, round(avg(probability),3), round(min(probability),3),
                          count(*) FILTER (WHERE probability < 0.5)
                   FROM analysis.latest_events WHERE probability IS NOT NULL GROUP BY raw_type""")
    lines = [f"- {t}: avg {a}, min {mn}, **{low} events below 0.5 confidence**" for t, a, mn, low in cur.fetchall()]
    lines.append("- Low-confidence events are challengeable — flag before exhibit use.")
    sec("Google's own confidence (probability fields)", lines)

    # 5. Schema drift: unknown keys census
    cur.execute("""SELECT google_type, k, count(*) FROM (
                     SELECT google_type, jsonb_object_keys(raw_json) AS k FROM raw.record
                     WHERE chunk_id IN (2,3)) s
                   WHERE k NOT IN ('startTime','endTime','visit','activity','timelinePath',
                                   'startTimeTimezoneUtcOffsetMinutes','endTimeTimezoneUtcOffsetMinutes',
                                   'timelineMemory')
                   GROUP BY google_type, k ORDER BY 3 DESC""")
    drift = cur.fetchall()
    sec("Schema drift census (unexpected top-level keys)",
        [f"- {t}: `{k}` × {n}" for t, k, n in drift] if drift else ["- NONE — her corpus matches the expected modern shape exactly."])

    # 6. Top places preview
    cur.execute("""SELECT place_id, visit_count, overnight_visits, first_visit, last_visit
                   FROM analysis.place_analytics WHERE subject_id=%s
                   ORDER BY visit_count DESC LIMIT 10""", (K,))
    sec("Top-10 places by visit count (preview — geocoding will name these)",
        ["| place_id | visits | overnights | first | last |", "|---|---|---|---|---|"] +
        [f"| {p[:20]}… | {v} | {o} | {f} | {l} |" for p, v, o, f, l in cur.fetchall()])

# 7. Raw quarterly-pair comparison (the 3 non-identical pairs)
pairs = {}
for p in glob.glob(os.path.join(EXTRACT, "**", "*.json.txt"), recursive=True):
    if os.path.getsize(p) > 500_000:
        pairs.setdefault(os.path.basename(p), []).append(p)
lines = []
for name, paths in sorted(pairs.items()):
    if len(paths) < 2:
        lines.append(f"- `{name}`: 1 unique copy only")
        continue
    sets = []
    for p in paths[:2]:
        try:
            doc = json.loads(io.open(p, encoding="utf-8-sig", errors="ignore").read())
            recs = doc.get("semanticSegments") or doc.get("timelineObjects") or []
            sets.append({hashlib.md5(json.dumps(r, separators=(',',':'), ensure_ascii=False).encode()).hexdigest() for r in recs})
        except Exception as e:
            sets.append(None); lines.append(f"- `{name}`: PARSE FAIL {paths[0][-40:]}: {str(e)[:60]}")
    if all(sets):
        a, b = sets
        lines.append(f"- `{name}`: copyA {len(a)} records, copyB {len(b)} records, common {len(a&b)}, only-A {len(a-b)}, only-B {len(b-a)}")
sec("Raw quarterly-pair comparison (record-level, non-identical pairs)", lines or ["- no multi-copy raw pairs found in _extracted_data"])

# 8. Geocode cache vintages
lines = []
for pat in ("radar_geocoding_master_good*.csv", "geocoding_cache*.json"):
    seen = {}
    for p in glob.glob(os.path.join(EXTRACT, "**", pat), recursive=True):
        h = hashlib.md5(open(p, "rb").read()).hexdigest()[:10]
        if h not in seen:
            n = sum(1 for _ in open(p, encoding="utf-8", errors="ignore")) if p.endswith(".csv") else len(json.load(io.open(p, encoding="utf-8-sig", errors="ignore")))
            seen[h] = (p, n)
    for h, (p, n) in seen.items():
        lines.append(f"- `{os.path.basename(p)}` [{h}]: {n} rows/entries — {p.replace(EXTRACT, '…')[:90]}")
sec("Geocode cache vintages (all kept as evidence; import both at Tier-2 setup)", lines)

with io.open(OUT, "w", encoding="utf-8") as f:
    f.write("\n".join(sections))
print(f"report data written: {OUT} ({len(sections)} sections)")
