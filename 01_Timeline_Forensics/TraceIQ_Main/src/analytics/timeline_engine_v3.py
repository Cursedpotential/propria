
from __future__ import annotations
import pandas as pd
from pathlib import Path
from dateutil import parser as dp
from scripts.helpers import haversine_m, parse_latlng
from datetime import timedelta
TOL = timedelta(minutes=5)

def _infer_path_times(df):
    import pandas as pd
    out = df.copy()
    out['__start']=pd.to_datetime(out.get('start_time'),errors='coerce')
    out['__end']=pd.to_datetime(out.get('end_time'),errors='coerce')
    out['__pt']=pd.to_datetime(out.get('point_time'),errors='coerce')
    path_mask=out['event_type'].eq('timeline_path')
    point_mask=out['event_type'].eq('point')
    segs=out[path_mask].sort_values('__start').copy()
    visits=out[out['event_type'].isin(['visit','activity'])].sort_values('__start')
    v=visits[['__start','__end','start_latlng','end_latlng','event_id']]
    for idx,row in segs.iterrows():
        st=row['__start']; en=row['__end']
        pts=out[(point_mask)&(out.get('_parent_seg_idx')==row.get('_seg_idx'))].sort_values('__pt')
        first_pt_time=pts['__pt'].min(); last_pt_time=pts['__pt'].max()
        first_pt_ll=pts['start_latlng'].iloc[0] if len(pts) else None
        last_pt_ll=pts['start_latlng'].iloc[-1] if len(pts) else None
        use_points=True; src='points_first_last'
        new_start, new_end = first_pt_time, last_pt_time
        new_start_ll, new_end_ll = first_pt_ll, last_pt_ll
        prev=v[v['__end'] <= (st if pd.notna(st) else pd.NaT) + TOL].tail(1)
        nxt=v[v['__start'] >= (en if pd.notna(en) else pd.NaT) - TOL].head(1)
        if not prev.empty and not nxt.empty and pd.notna(st) and pd.notna(en):
            if (abs(st - prev['__end'].iloc[0]) <= TOL) and (abs(nxt['__start'].iloc[0] - en) <= TOL):
                use_points=False; src='adjacent_events'
                new_start=prev['__end'].iloc[0]; new_end=nxt['__start'].iloc[0]
                new_start_ll=prev['end_latlng'].iloc[0]; new_end_ll=nxt['start_latlng'].iloc[0]
        out.at[idx,'path_time_fallback']=bool(use_points)
        out.at[idx,'path_time_source']=src
        if pd.notna(new_start): out.at[idx,'start_time']=new_start
        if pd.notna(new_end): out.at[idx,'end_time']=new_end
        if new_start_ll: out.at[idx,'start_latlng']=new_start_ll
        if new_end_ll: out.at[idx,'end_latlng']=new_end_ll
    return out


def _is_overnight_local(ts):
    # Coarse local-clock check: 22:00–07:00
    if pd.isna(ts): 
        return False
    hour = ts.hour
    return hour >= 22 or hour < 7

def compute_analytics(df:
    df = _infer_path_times(df)
 pd.DataFrame) -> pd.DataFrame:
    df = df.copy()
    # parse times
    df["__start"] = pd.to_datetime(df["start_time"], errors="coerce")
    df["__end"]   = pd.to_datetime(df["end_time"], errors="coerce")
    df["__pt"]    = pd.to_datetime(df["point_time"], errors="coerce")

    # segments only
    seg_mask = df["event_type"].isin(["visit","activity","timeline_path","trip"])

    # duration + hhmm
    df.loc[seg_mask, "duration_sec"] = (df.loc[seg_mask, "__end"] - df.loc[seg_mask, "__start"]).dt.total_seconds().astype("Int64")
    df.loc[seg_mask, "duration_hhmm"] = df.loc[seg_mask, "duration_sec"].apply(lambda x: f"{int(x//3600):02}:{int((x%3600)//60):02}" if pd.notna(x) else None)

    # gap to previous segment
    segs = df[seg_mask].sort_values(["__start"]).copy()
    segs["__prev_end"] = segs["__end"].shift(1)
    segs["gap_to_prev_sec"] = (segs["__start"] - segs["__prev_end"]).dt.total_seconds().astype("Int64")
    df.loc[segs.index, "gap_to_prev_sec"] = segs["gap_to_prev_sec"]

    # overnight for segments
    df.loc[seg_mask, "overnight_flag"] = [(_is_overnight_local(s) or _is_overnight_local(e)) for s,e in zip(df.loc[seg_mask,"__start"], df.loc[seg_mask,"__end"])]

    # point-to-point durations (within same parent prefix)
    pts = df[~seg_mask].copy()
    pts["__parent_eid"] = pts["event_id"].str.replace(r"\.P\d+$","", regex=True)
    for parent, g in pts.sort_values("__pt").groupby("__parent_eid"):
        prev = None
        for idx, row in g.iterrows():
            if prev is None:
                df.at[idx, "duration_sec"] = None
            else:
                df.at[idx, "duration_sec"] = int((row["__pt"] - prev).total_seconds())
            prev = row["__pt"]

    # path distance meters for timeline_path parents (sum of haversine across their points)
    for parent_idx, prow in df[df["event_type"]=="timeline_path"].iterrows():
        parent_eid = prow["event_id"]
        pts = df[df["event_id"].str.startswith(parent_eid + ".P", na=False)].sort_values("__pt")
        dist = 0.0
        prev_latlng = None
        for _,r in pts.iterrows():
            if pd.isna(r["start_latlng"]): 
                continue
            lat, lng = parse_latlng(r["start_latlng"])
            if prev_latlng is not None:
                dist += haversine_m(prev_latlng[0], prev_latlng[1], lat, lng)
            prev_latlng = (lat, lng)
        if dist>0:
            df.at[parent_idx, "path_distance_meters"] = dist

    # NEW: Overnight for points only if their parent path is overnight AND the point_time is in overnight window
    # Build set of overnight parent event_ids for timeline paths
    overnight_path_ids = set(df[(df["event_type"]=="timeline_path") & (df["overnight_flag"]==True)]["event_id"].dropna().tolist())
    # mark points
    pts_idx = df.index[~seg_mask]
    df.loc[pts_idx, "overnight_flag"] = False  # baseline
    # Only if parent path is overnight and point time is overnight by clock
    for idx,row in df.loc[pts_idx].iterrows():
        eid = row.get("event_id") or ""
        parent_eid = eid.split(".P")[0] if ".P" in eid else None
        if parent_eid and parent_eid in overnight_path_ids and _is_overnight_local(row["__pt"]):
            df.at[idx, "overnight_flag"] = True

    # cleanup
    return df.drop(columns=["__start","__end","__pt"], errors="ignore")

if __name__ == "__main__":
    import argparse
    ap = argparse.ArgumentParser()
    ap.add_argument("--src", required=True)
    ap.add_argument("--out", required=True)
    args = ap.parse_args()
    df = pd.read_csv(Path(args.src))
    df = compute_analytics(df)
    df.to_csv(Path(args.out), index=False)
