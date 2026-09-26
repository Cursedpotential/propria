# ADDENDUM PART 2: Geocoding Deep Dive & Constraints

**Append after Addendum Part 1**

---

## G. EXISTING GOOGLE PLACE ID CACHE

A cache of Google Place ID → address lookups already exists. **Check this BEFORE any Google API call.**

**Expected schema:**
```
place_id,formatted_address,name,types,lat,lng,cached_at
```

**Lookup protocol:**
```python
def resolve_place_id(place_id: str, cache_conn, google_api_key: str = None) -> dict:
    """
    1. Check local cache first
    2. Only hit Google API if cache miss
    3. Cache new results immediately
    """
    if not place_id or place_id == "":
        return None
    
    # Cache check
    cached = cache_conn.execute(
        "SELECT * FROM geocoding_cache WHERE cache_type = 'google_place' AND lookup_key = ?",
        (place_id,)
    ).fetchone()
    
    if cached:
        return dict(cached)
    
    # API call only if cache miss AND key provided
    if google_api_key:
        result = google_place_details(place_id, google_api_key)
        # Cache immediately
        cache_result(cache_conn, 'google_place', place_id, result)
        return result
    
    return {'place_id': place_id, 'status': 'uncached_no_api_key'}
```

---

## H. GOOGLE SNAP TO ROADS API

**Purpose:** Clean up GPS drift in driving routes. Raw waypoints zigzag; snapped points follow actual roads.

**When to use:**
- Activity type = `IN_PASSENGER_VEHICLE`, `MOTORCYCLING`, `IN_BUS`
- NOT for walking, cycling, or stationary

**API endpoint:**
```
https://roads.googleapis.com/v1/snapToRoads
```

**Cost:** $0.01 per call (up to 100 points per call)

**Implementation:**
```python
def snap_to_roads(waypoints: list[dict], api_key: str) -> list[dict]:
    """
    Snap GPS points to nearest roads.
    Max 100 points per call — batch if needed.
    Returns snapped coordinates + placeId for road segment.
    """
    if len(waypoints) > 100:
        # Batch into chunks of 100
        results = []
        for i in range(0, len(waypoints), 100):
            chunk = waypoints[i:i+100]
            results.extend(snap_to_roads(chunk, api_key))
        return results
    
    path = "|".join([f"{wp['latitude']},{wp['longitude']}" for wp in waypoints])
    
    resp = requests.get(
        "https://roads.googleapis.com/v1/snapToRoads",
        params={
            "path": path,
            "interpolate": "true",  # Fill gaps
            "key": api_key
        }
    )
    
    data = resp.json()
    snapped = []
    
    for point in data.get('snappedPoints', []):
        snapped.append({
            'latitude': point['location']['latitude'],
            'longitude': point['location']['longitude'],
            'original_index': point.get('originalIndex'),  # Which input point this came from
            'place_id': point.get('placeId'),  # Road segment ID
        })
    
    return snapped
```

**Schema addition for snapped waypoints:**
```sql
ALTER TABLE waypoints ADD COLUMN snapped_latitude REAL;
ALTER TABLE waypoints ADD COLUMN snapped_longitude REAL;
ALTER TABLE waypoints ADD COLUMN road_place_id TEXT;  -- Google's road segment ID
ALTER TABLE waypoints ADD COLUMN snap_source TEXT;    -- 'google_roads' or NULL
```

**When NOT to snap:**
- Walking routes (sidewalks not in roads DB)
- Indoor locations
- Already-accurate points (< 10m from road)

---

## I. GEOCODING CONSTRAINTS & EXPECTATIONS

### Hard Rules

| Rule | Rationale |
|------|-----------|
| **Cache before API** | Never duplicate a call. Check all caches first. |
| **Radar first, Google second** | Cost. Radar = free. Google = $17/1K. |
| **Batch API calls** | Snap to Roads takes 100 pts/call. Use it. |
| **Preserve original coordinates** | Snapped coords go in separate columns. Never overwrite raw GPS. |
| **Log all API calls** | Forensic audit requires knowing when/what was fetched. |

### Accuracy Expectations

| Source | Typical Accuracy | Notes |
|--------|------------------|-------|
| Google Timeline raw | 10-50m | Phone GPS, varies by conditions |
| Snap to Roads | Road-level | Good for "which street" not "which lane" |
| Radar reverse geocode | Address-level | Usually correct street address |
| Google Place Details | Exact | For named places with placeId |

### Known Issues

1. **Empty placeId:** Google sometimes returns `""` for placeId. Treat as NULL.
2. **Stale place data:** Businesses close, addresses change. Cache includes `cached_at` for reason.
3. **Coordinate precision loss:** Don't round. Store full precision from source.
4. **Timezone in coordinates:** Some exports include timezone in metadata — use it for local time display.

---

## J. MULTI-SOURCE VALIDATION MATRIX

For forensically significant locations (homes, schools, exchange points):

| Check | Pass Criteria |
|-------|---------------|
| Radar address matches Google | Street name + number match |
| Coordinates within 50m of claimed location | Haversine distance check |
| Timestamp plausible | Not in future, not before phone existed |
| No impossible travel | < 200 km/h between consecutive points |

Flag but don't discard failures — anomalies may be evidence.

---

## K. IMPLEMENTING AI: EXPLORE THESE

You have context about:
- Michigan custody case needing location proof
- Cost constraints (free tiers preferred)
- Existing caches (Radar CSV, Google Place ID)
- Snap to Roads for driving routes

**Explore and recommend:**

1. **What validation would YOU add?** Spoofing detection? Impossible travel flagging?
2. **What's missing for court admissibility?** Chain of custody gaps?
3. **Better overnight detection?** Current logic is simple (crosses midnight or 22:00-08:00 + >1hr). Improve it.
4. **Location clustering?** Auto-detect "home" vs "work" vs "frequently visited"?
5. **Export format for attorneys?** What do lawyers actually need to see?
6. **Correlation opportunities?** How would you tie this to SMS/photo timestamps?

Don't just implement the spec — improve it where you see opportunity.

---

*End of Addendum Part 2*
