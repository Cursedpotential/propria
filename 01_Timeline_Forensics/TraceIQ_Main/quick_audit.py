#!/usr/bin/env python3
from __future__ import annotations
import os, re, json, argparse
from pathlib import Path
import pandas as pd

def load_csv(path):
    try: return pd.read_csv(path)
    except: return None

def is_iso8601_with_offset(s):
    if not isinstance(s, str): return False
    return bool(re.match(r"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+\-]\d{2}:\d{2})$", s))

def check_exists(p, label, problems):
    if not p.exists(): problems.append(f"FAIL: missing {label} at {p}"); return False
    return True

def check_master(root, problems, notes):
    master = root / "output_data/master_machine.csv"
    if not check_exists(master, "master_machine.csv", problems): return
    df = load_csv(master)
    if df is None or df.empty: problems.append("FAIL: master_machine.csv missing/empty"); return
    notes.append(f"OK: master rows={len(df)}")
    allowed = {"visit","activity","trip","timeline_path","point"}
    if "event_type" not in df.columns: problems.append("FAIL: missing event_type")
    else:
        bad = df[~df["event_type"].isin(list(allowed))]
        if len(bad): problems.append(f"FAIL: invalid event_type values (n={len(bad)})")
    if "event_uuid" not in df.columns: problems.append("FAIL: missing event_uuid")
    else:
        if df["event_uuid"].isna().any(): problems.append("FAIL: some rows missing event_uuid")
        if df["event_uuid"].duplicated().any(): problems.append("FAIL: duplicate event_uuid")
    if "custom_id" not in df.columns: problems.append("FAIL: missing custom_id")
    if "event_type" in df.columns and "parent_event_uuid" in df.columns:
        pts = df[df["event_type"]=="point"]
        if not pts.empty and pts["parent_event_uuid"].isna().any(): problems.append("FAIL: points missing parent_event_uuid")
    tpls = df[df["event_type"]=="timeline_path"]
    if not tpls.empty:
        for col in ["start_time","end_time","start_latlng","end_latlng","path_time_source"]:
            if col not in tpls.columns: problems.append(f"FAIL: timeline_path missing {col}")
        if "start_time" in tpls.columns and "end_time" in tpls.columns:
            bad_time = sum(1 for s,e in zip(tpls["start_time"], tpls["end_time"]) if not (isinstance(s,str) and isinstance(e,str) and is_iso8601_with_offset(s) and is_iso8601_with_offset(e)))
            if bad_time: problems.append(f"FAIL: {bad_time} timeline_path times not ISO8601 w/ offset")
        if not {"path_time_source","path_time_fallback"}.issubset(set(tpls.columns)): problems.append("FAIL: provenance columns missing")
    r_cols = [c for c in df.columns if c.endswith("_lat_r3") or c.endswith("_lng_r3") or c.endswith("_lat_r4") or c.endswith("_lng_r4")]
    gh_cols = [c for c in df.columns if "geohash8" in c or "geohash9" in c]
    if not r_cols: problems.append("FAIL: r3/r4 rounding columns not found")
    if not gh_cols: problems.append("FAIL: geohash8/9 columns not found")
    p_cols = {"precision_method","precision_m_lat_est","precision_m_lon_est","precision_m_est"}
    if not p_cols.issubset(set(df.columns)): problems.append("FAIL: precision_* columns missing")

def check_orders(root, problems, notes):
    g = root / "geo_requests/google_placeid_requests.json"
    r = root / "geo_requests/radar_latlng_requests.json"
    if g.exists():
        try: d = json.load(open(g, "r")); n = len(d.get("place_ids", [])); 
        except: problems.append("FAIL: google orders unreadable"); n=0
        if n == 0: problems.append("FAIL: google_placeid_requests has 0 ids")
        else: notes.append(f"OK: google orders count={n}")
    else: problems.append("FAIL: google_placeid_requests.json missing")
    if r.exists():
        try: d = json.load(open(r, "r")); n = len(d.get("latlng_keys", [])); 
        except: problems.append("FAIL: radar orders unreadable"); n=0
        if n == 0: problems.append("FAIL: radar_latlng_requests has 0 keys")
        else: notes.append(f"OK: radar orders count={n}")
    else: problems.append("FAIL: radar_latlng_requests.json missing")

def check_resolver(root, problems, notes):
    g = root / "output_data/api_csv/google_results.csv"
    r = root / "output_data/api_csv/radar_results.csv"
    st = root / "state/resolver_progress.json"
    lg = root / "output_data/logs/resolver.log"
    if not g.exists(): problems.append("FAIL: google_results.csv missing")
    if not r.exists(): problems.append("FAIL: radar_results.csv missing")
    if not st.exists(): problems.append("FAIL: resolver_progress.json missing")
    if not lg.exists(): problems.append("FAIL: resolver.log missing")

def check_human(root, problems, notes):
    human = root / "output_data/human"
    if not human.exists(): problems.append("FAIL: human export folder missing"); return
    files = list(human.glob("human_*.csv"))
    if not files: problems.append("FAIL: no human_YYYY.csv"); return
    import pandas as pd
    df = pd.read_csv(files[0])
    needed = {"date_local","weekday_local","start_time_et_12h","end_time_et_12h",
              "place_name","place_type_primary","street_short","city_short",
              "google_address","radar_address","overnight_flag","party_or_bar_flag",
              "maps_link_start","maps_link_end","maps_link_point","google_place_url"}
    miss = needed - set(df.columns)
    if miss: problems.append(f"FAIL: human CSV missing columns: {sorted(miss)}")

def main():
    ap = argparse.ArgumentParser(); ap.add_argument("--root", required=True)
    args = ap.parse_args(); root = Path(os.path.expanduser(args.root)).resolve()
    probs=[]; notes=[]
    for sub in ["working_data","output_data","geo_requests","raw_api","state"]:
        if not (root/sub).exists(): probs.append(f"FAIL: missing folder {sub}")
    check_master(root, probs, notes)
    check_orders(root, probs, notes)
    check_resolver(root, probs, notes)
    check_human(root, probs, notes)
    print("=== QUICK AUDIT REPORT ===")
    for n in notes: print(n)
    if probs:
        print("\n--- PROBLEMS ---"); 
        for p in probs: print(p)
    else:
        print("\nNo problems detected.")
if __name__=="__main__": main()
