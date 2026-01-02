from __future__ import annotations
import re, math
from typing import Optional, Dict

M_PER_DEG_LAT = 111_320.0
_GEOHASH_LEN_TO_CELL_M = {
    1:(5_000_000.0,5_000_000.0), 2:(1_250_000.0,625_000.0),
    3:(156_000.0,156_000.0), 4:(39_100.0,19_500.0),
    5:(4_890.0,4_890.0), 6:(1_220.0,610.0),
    7:(153.0,153.0), 8:(38.2,19.1),
    9:(4.77,4.77), 10:(1.19,0.596),
    11:(0.149,0.149), 12:(0.0372,0.0186),
}
_DEC_RE = re.compile(r"^-?\d+\.(\d+)$")

def _decimals_of(s: str) -> Optional[int]:
    if not isinstance(s, str): return None
    m = _DEC_RE.match(s.strip())
    return len(m.group(1)) if m else None

def precision_from_latlng(latlng: str) -> Optional[Dict[str, float]]:
    if not isinstance(latlng, str) or "," not in latlng: return None
    try:
        lat_s, lng_s = [x.strip() for x in latlng.split(",", 1)]
        lat = float(lat_s); lng = float(lng_s)
    except Exception:
        return None
    d_lat = _decimals_of(lat_s); d_lng = _decimals_of(lng_s)
    if d_lat is None and d_lng is None: return None
    m_per_deg_lon = M_PER_DEG_LAT * math.cos(math.radians(lat))
    lat_m = M_PER_DEG_LAT * (10 ** (-(d_lat if d_lat is not None else d_lng)))
    lon_m = m_per_deg_lon * (10 ** (-(d_lng if d_lng is not None else d_lat)))
    return {"lat_m": float(lat_m), "lon_m": float(lon_m), "d_lat": float(d_lat or d_lng or 0), "d_lng": float(d_lng or d_lat or 0)}

def precision_from_geohash_len(geohash: str, lat_for_lon: Optional[float]=None) -> Optional[Dict[str, float]]:
    if not isinstance(geohash, str) or not geohash: return None
    gh_len = len(geohash)
    base = _GEOHASH_LEN_TO_CELL_M.get(gh_len)
    if base is None:
        keys = sorted(_GEOHASH_LEN_TO_CELL_M.keys())
        gh_len = keys[0] if gh_len < keys[0] else keys[-1]
        base = _GEOHASH_LEN_TO_CELL_M[gh_len]
    lon_m, lat_m = base
    if lat_for_lon is not None:
        lon_m *= max(0.0, math.cos(math.radians(lat_for_lon)))
    return {"lat_m": float(lat_m), "lon_m": float(lon_m), "gh_len": float(gh_len)}

def choose_precision(latlng: Optional[str]=None, geohash: Optional[str]=None) -> Optional[Dict[str, float]]:
    lat_for_lon = None
    if isinstance(latlng, str) and "," in latlng:
        try: lat_for_lon = float(latlng.split(",",1)[0].strip())
        except Exception: pass
    if isinstance(geohash, str) and geohash:
        gh = precision_from_geohash_len(geohash, lat_for_lon=lat_for_lon)
        if gh:
            return {"precision_method":"geohash_len","precision_m_lat_est":gh["lat_m"],"precision_m_lon_est":gh["lon_m"],"precision_m_est":max(gh["lat_m"], gh["lon_m"])}
    dec = precision_from_latlng(latlng) if isinstance(latlng, str) else None
    if dec:
        return {"precision_method":"latlng_decimals","precision_m_lat_est":dec["lat_m"],"precision_m_lon_est":dec["lon_m"],"precision_m_est":max(dec["lat_m"], dec["lon_m"])}
    return None
