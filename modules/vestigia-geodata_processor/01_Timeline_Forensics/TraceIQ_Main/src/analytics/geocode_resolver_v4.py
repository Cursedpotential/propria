from __future__ import annotations
import json, sqlite3, hashlib, time, os
from pathlib import Path
from typing import Dict, Any, Optional
from scripts.providers.google import get_place_details_by_id
from scripts.providers.radar import reverse_geocode

RAW_API_DIR = Path(os.environ.get("RAW_API_DIR", "/mnt/data/raw_api"))
def _safe_key(s: str) -> str: return hashlib.sha256(s.encode("utf-8")).hexdigest()
def _cache_path(provider: str, mode: str, key: str) -> Path:
    d = RAW_API_DIR / provider / mode; d.mkdir(parents=True, exist_ok=True); return d / f"{_safe_key(key)}.json"
def load_cache(provider: str, mode: str, key: str):
    p = _cache_path(provider, mode, key); 
    if p.exists():
        try: return json.loads(p.read_text(encoding="utf-8"))
        except Exception: return None
    return None
def save_cache(provider: str, mode: str, key: str, data):
    p = _cache_path(provider, mode, key)
    try: p.write_text(json.dumps(data, indent=2), encoding='utf-8')
    except Exception: pass
def norm_latlng(s: str|None) -> str|None:
    if not isinstance(s, str) or ',' not in s: return None
    return s.replace(' ','')
def location_uuid(latlng: str) -> str: return hashlib.sha256(latlng.encode('utf-8')).hexdigest()
def upsert(conn, table: str, row: Dict[str, Any], pk: str):
    cols=list(row.keys()); placeholders=','.join(['?']*len(cols)); collist=','.join(cols)
    sql=f"INSERT INTO {table} ({collist}) VALUES ({placeholders}) ON CONFLICT({pk}) DO UPDATE SET "+','.join([f"{c}=excluded.{c}" for c in cols if c!=pk])
    conn.execute(sql, tuple(row.get(c) for c in cols))

def resolve(requests_dir: Path, db_path: Path, google_key: str|None=None, radar_key: str|None=None):
    conn = sqlite3.connect(db_path)
    gfile = requests_dir/'google_placeid_requests.json'
    rfile = requests_dir/'radar_latlng_requests.json'
    g_reqs = json.loads(gfile.read_text())['requests'] if gfile.exists() else []
    r_reqs = json.loads(rfile.read_text())['requests'] if rfile.exists() else []

    # Google by place_id
    for req in g_reqs:
        pid = req.get('place_id'); 
        if not pid: continue
        rid = hashlib.sha256(f'google:{pid}'.encode()).hexdigest()
        upsert(conn, 'geocode_request', {
            'request_uuid': rid, 'location_uuid': None, 'latlng_r4': None, 'latlng_r5': None, 'latlng_exact6': None, 'status': 'requested'
        }, pk='request_uuid')
        res = load_cache('google','place_details', pid); src='cache'
        if not res and google_key:
            try: res = get_place_details_by_id(pid, google_key); src='live'
            except Exception: res=None
        if res:
            save_cache('google','place_details', pid, res)
            upsert(conn, 'geocode_result_google', {
                'result_uuid': hashlib.sha256(f'google:res:{pid}'.encode()).hexdigest(),
                'request_uuid': rid,
                'place_id': res.get('place_id') or pid,
                'formatted_address': res.get('formatted_address'),
                'confidence': res.get('confidence'),
                'bounds': json.dumps(res.get('bounds')) if isinstance(res.get('bounds'), dict) else None,
                'raw_json': json.dumps(res.get('raw_json')) if res.get('raw_json') is not None else None
            }, pk='result_uuid')
            conn.execute('UPDATE geocode_request SET status=? WHERE request_uuid=?', (src, rid))

    # Radar by lat/lng
    for req in r_reqs:
        ll = req.get('latlng') or req.get('centroid_latlng'); s = norm_latlng(ll)
        if not s: continue
        lat, lng = map(float, s.split(','))
        rid = hashlib.sha256(f'radar:{s}'.encode()).hexdigest()
        upsert(conn, 'geocode_request', {
            'request_uuid': rid, 'location_uuid': location_uuid(s),
            'latlng_r4': req.get('latlng_r4') or req.get('key_r4'),
            'latlng_r5': req.get('latlng_r5'), 'latlng_exact6': req.get('latlng_exact6'),
            'status': 'requested'
        }, pk='request_uuid')
        res = load_cache('radar','reverse_geocode', s); src='cache'
        if not res and radar_key:
            try: res = reverse_geocode(lat, lng, radar_key); src='live'
            except Exception: res=None
        if res:
            save_cache('radar','reverse_geocode', s, res)
            upsert(conn, 'geocode_result_radar', {
                'result_uuid': hashlib.sha256(f'radar:res:{s}'.encode()).hexdigest(),
                'request_uuid': rid,
                'place_id': res.get('place_id'),
                'formatted_address': res.get('formatted_address'),
                'confidence': res.get('confidence'),
                'bounds': json.dumps(res.get('bounds')) if isinstance(res.get('bounds'), dict) else None,
                'raw_json': json.dumps(res.get('raw_json')) if res.get('raw_json') is not None else None
            }, pk='result_uuid')
            conn.execute('UPDATE geocode_request SET status=? WHERE request_uuid=?', (src, rid))

    conn.commit(); conn.close()

if __name__=='__main__':
    import argparse
    ap=argparse.ArgumentParser()
    ap.add_argument('--requests-dir', required=True)
    ap.add_argument('--db', required=True)
    ap.add_argument('--google-key', default=None)
    ap.add_argument('--radar-key', default=None)
    a=ap.parse_args()
    resolve(Path(a.requests_dir), Path(a.db), a.google_key, a.radar_key)
