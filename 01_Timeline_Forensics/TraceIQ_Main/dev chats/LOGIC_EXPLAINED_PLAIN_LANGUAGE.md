# Critical Logic Explanations (Plain Language)

## 1. Multi-Device Detection Logic

### The Problem:
Google Timeline sometimes records the same person in two places at the same time because:
- They have multiple devices (phone + tablet + watch)
- Data sync issues create duplicate timestamps
- GPS drift causes the same device to report slightly different locations

### How We Detect It:
**Step 1:** Look at ALL waypoints (GPS breadcrumbs) within a single path segment

**Step 2:** Group waypoints that have the EXACT SAME timestamp (down to the second)

**Step 3:** For each group of same-timestamp waypoints:
- Calculate the distance between them using haversine formula (spherical earth distance)
- If distance > 100 meters → This is physically impossible for one person
- Therefore: TWO DEVICES must be recording

**Step 4:** When we detect multi-device:
- Split the waypoints into TWO separate paths
- Label one as `device_0` and the other as `device_1`
- Tag both with `multi_device_split = TRUE`
- Record the original segment_index they came from
- Document the `split_reason = "duplicate_timestamp_>100m"`

### Why This Matters for Court:
- Shows we caught data anomalies
- Prevents "impossible travel" accusations (being in two places at once)
- Documents which device recorded which location
- Proves data integrity checking was performed

### Example Scenario:
```
Path segment #47 has waypoints:
- 2:15:00 PM → Flint, MI (lat: 43.012, lng: -83.687)
- 2:15:00 PM → Detroit, MI (lat: 42.331, lng: -83.045)
- 2:15:30 PM → Flint, MI (lat: 43.013, lng: -83.688)

Distance between first two points = 90km (way more than 100m)
→ MULTI-DEVICE DETECTED
→ Split into:
   - path_47_device0: Flint timestamps
   - path_47_device1: Detroit timestamps
```

---

## 2. Timeline Path Timestamp Logic (Fixing Google's Garbage Timestamps)

### The Problem:
Google gives paths (driving/walking segments) a **2-hour window timestamp** instead of precise start/end times.

Example:
- Path says: "Activity occurred between 2:00 PM - 4:00 PM"
- But the actual drive was 2:17 PM - 2:31 PM (14 minutes)

Google does this because paths are just containers for waypoints, and Google's UI doesn't care about precision.

### Our Solution: 3-Tier Priority System

#### **PRIORITY 1: Use Waypoint Times (Most Accurate)**
- Look at all waypoints (GPS breadcrumbs) in the path
- First waypoint timestamp = START of path
- Last waypoint timestamp = END of path
- Tag as: `path_time_source = "points_first_last"`
- This is the BEST source because waypoints are actual GPS pings

#### **PRIORITY 2: Use Adjacent Event Times (Second Best)**
- Look at the visit/activity BEFORE the path
- Look at the visit/activity AFTER the path
- If timestamps align (within 5 minutes tolerance):
  - Path START = end time of previous visit
  - Path END = start time of next visit
- Tag as: `path_time_source = "adjacent_events"`
- This works when waypoints are missing but events bookend the path

#### **PRIORITY 3: Container Timestamp (Worst - Fallback Only)**
- Use Google's 2-hour window timestamp
- Tag as: `path_time_source = "container_fallback"`
- This is GARBAGE DATA but better than nothing
- Clearly flagged so attorneys know it's approximate

### Why We Tag the Source:
- **Forensic transparency**: Court can see where each timestamp came from
- **Data quality indicator**: "points_first_last" = high confidence, "container_fallback" = low confidence
- **Challenge preparation**: If opposing counsel questions a timestamp, we can show it came from actual GPS pings, not estimates

### Example Scenario:
```
Event Sequence:
- Visit #52: Home (2:00 PM - 2:15 PM)
- Path #53: Driving (Google says "2:00 PM - 4:00 PM" - useless)
  └── Waypoints: 2:17 PM, 2:19 PM, 2:24 PM, 2:28 PM, 2:31 PM
- Visit #54: Store (2:33 PM - 3:10 PM)

Our Processing:
1. Check waypoints → Found 5 waypoints
2. Path START = 2:17 PM (first waypoint)
3. Path END = 2:31 PM (last waypoint)
4. Tag: path_time_source = "points_first_last"
5. Verify: 2:31 PM + 2 min = 2:33 PM (next visit start) ✓ Makes sense
```

### Edge Case - Missing Waypoints:
```
Event Sequence:
- Visit #52: Home (2:00 PM - 2:15 PM)
- Path #53: Driving (NO WAYPOINTS - phone was off?)
- Visit #54: Store (2:33 PM - 3:10 PM)

Our Processing:
1. Check waypoints → None found
2. Check adjacent events:
   - Previous visit ended: 2:15 PM
   - Next visit started: 2:33 PM
   - Gap = 18 minutes (within tolerance)
3. Path START = 2:15 PM (previous visit end)
4. Path END = 2:33 PM (next visit start)
5. Tag: path_time_source = "adjacent_events"
6. Flag: path_time_fallback = TRUE (not from GPS)
```

---

## 3. Overnight Detection Logic

### Simple Version (Current):
**Rule:** If a visit or activity happens between 10:00 PM (22:00) and 7:00 AM (07:00), flag it as overnight.

**How it works:**
- Extract the hour from the timestamp (ignore timezone for now - just look at the clock time)
- If hour >= 22 OR hour < 7 → `overnight_flag = TRUE`

**Why this matters:**
- Custody cases care about "overnight stays"
- Shows where someone was sleeping
- Helps establish primary residence
- Can prove/disprove claims about where child stayed

### Problems with Simple Version:
1. **Timezone issues:** 11 PM EST might be 8 PM in California (not actually overnight)
2. **Late night != overnight stay:** Being at a bar at 11 PM doesn't mean you stayed there overnight
3. **No multi-day detection:** Doesn't distinguish between "was there late" vs "stayed 3 days"

### Improved Version (What We Should Do):

#### **Step 1: Use Actual Timezone**
- Don't just look at clock hour
- Use `timezone_offset` field from timeline data
- Convert to TRUE local time
- Then apply 22:00-07:00 rule

#### **Step 2: Detect Multi-Day Stays**
- If visit START date ≠ END date → This crossed into a new day
- Calculate: `days_stayed = (end_date - start_date).days`
- If days_stayed >= 1 → TRUE overnight stay
- If days_stayed == 0 but hour in 22:00-07:00 window → POSSIBLE overnight (needs verification)

#### **Step 3: Location Context**
- Cross-reference with `home_base` table
- If location matches known home address → Likely overnight
- If location is bar/restaurant/entertainment → Unlikely overnight
- If location is hotel/friend's house → Likely overnight

### Example Scenarios:

**Scenario A: True Overnight**
```
Visit to "123 Main St":
- Start: 2023-09-15 21:30:00 (9:30 PM)
- End: 2023-09-16 07:45:00 (7:45 AM next day)
- Days stayed: 1
- Overnight flag: TRUE
- Reason: Multi-day stay + late start + early end
```

**Scenario B: Late Night (Not Overnight)**
```
Visit to "Bar & Grill":
- Start: 2023-09-15 22:15:00 (10:15 PM)
- End: 2023-09-16 00:30:00 (12:30 AM - technically next day)
- Days stayed: 0 (same calendar date when considering duration)
- Overnight flag: QUESTIONABLE
- Reason: Late hour but short duration, bar location
```

**Scenario C: Multi-Day Stay**
```
Visit to "Hotel Residence":
- Start: 2023-09-15 15:00:00 (3:00 PM)
- End: 2023-09-17 10:00:00 (10:00 AM two days later)
- Days stayed: 2
- Overnight flag: TRUE (definitely)
- Reason: 2-night hotel stay
```

---

## 4. Geocoding Cache Handling Strategy

### The Goal:
Get human-readable addresses (street, city, state) for lat/lng coordinates **without spending money on API calls every time**.

Matt has 50,000+ locations already cached. We want 99%+ cache hit rate = $0 API costs.

### How It Works (3-Tier System):

#### **Tier 1: File System Cache (Fastest)**
**Storage:** `data/caches/radar_geocoding_master_good.csv` (7 MB, 37 columns, 50K+ records)

**Lookup process:**
1. Take a coordinate: `43.012345, -83.687654`
2. Search cache for EXACT match
3. If found → Return cached address
4. If not found → Go to Tier 2

**Why this works:**
- CSV file loaded into memory at startup
- In-memory hash lookup is instant
- No network call needed
- Zero cost

#### **Tier 2: Multi-Precision Lookup (Smart Fallback)**
**Problem:** Exact match fails if GPS drifted by 1 meter

**Solution:** Try MULTIPLE precision levels:
1. Try r6 (6 decimals): `43.012345, -83.687654` - exact (±11cm)
2. Try r5 (5 decimals): `43.01235, -83.68765` - rounded (±1.1m)
3. Try r4 (4 decimals): `43.0123, -83.6877` - rounded (±11m)
4. Try r3 (3 decimals): `43.012, -83.688` - rounded (±111m)

**Plus geohash clustering:**
- Try geohash9: `dpshw7g2h` (±4.8m x 4.8m cell)
- Try geohash8: `dpshw7g2` (±38m x 19m cell)

**If ANY of these match → Use cached result**

**Why this works:**
- GPS coordinates are never EXACTLY the same
- A location visited twice will have slightly different lat/lng
- But r4 rounding (±11m) will match if it's the same building
- Massively increases cache hit rate

#### **Tier 3: Live API Call (Last Resort)**
**Only if** all cache lookups fail:
1. Call Radar API (or Google Places API)
2. Get fresh geocoded address
3. **SAVE TO CACHE** for next time
4. Update `radar_geocoding_master_good.csv`
5. Log to `geocode_audit` table

**Cost tracking:**
- Record in `geocode_request` table
- Status: `"cache"` (Tier 1/2) or `"live"` (Tier 3)
- Track API provider, timestamp, cost

### Multi-Source Strategy (3 Providers):

**Provider 1: Radar (Primary)**
- Best for US addresses
- Cheapest API
- Already have 50K+ cached
- Use for: residential addresses, streets

**Provider 2: Google Places (Secondary)**
- Best for commercial locations (restaurants, stores)
- More expensive
- Use for: place_id lookups, business names

**Provider 3: Geodata (Tertiary - NEW)**
- Backup provider
- Use for: international addresses, Radar failures

**Resolution Logic:**
- If Radar AND Google both return results → Compare
- If addresses match → High confidence
- If addresses differ → Flag as `disagreement_flag = TRUE`
- Use `geocode_resolution` table to pick preferred result
- Document tie-break reason

---

## 5. Coordinate Rounding & Precision Strategy

### Why Rounding Matters:
GPS coordinates look precise but aren't:
- `43.0123456789, -83.6876543210` (10 decimals)
- But GPS accuracy is only ±5-10 meters
- Storing extra decimals is useless
- Rounding increases cache hits

### Precision Levels (What Each Means):

| Decimals | Precision | Use Case | Example |
|----------|-----------|----------|---------|
| **r3 (3)** | ±111 meters | Neighborhood lookup | `43.012, -83.688` |
| **r4 (4)** | ±11 meters | Building/address lookup | `43.0123, -83.6877` |
| **r5 (5)** | ±1.1 meters | Exact location | `43.01235, -83.68765` |
| **r6 (6)** | ±11 centimeters | Forensic precision | `43.012345, -83.687654` |

### How We Use Each:

**r3 (Neighborhood level):**
- Use for: "Were they in the same neighborhood?"
- Cache lookup: Very broad match
- Example: All of "Downtown Flint" maps to same r3 coordinate

**r4 (Building level) - MOST COMMON:**
- Use for: "Were they at this address?"
- Cache lookup: Building-level accuracy
- Example: All GPS points at "123 Main St" round to same r4
- **This is our PRIMARY geocoding cache key**

**r5 (Exact location):**
- Use for: "Were they at this exact spot in the building?"
- Cache lookup: Room/parking spot level
- Example: Front door vs back door of building

**r6 (Forensic precision):**
- Use for: Original data preservation
- Never used for cache lookups (too precise)
- Stored in `location_key` table as `latlng_exact`

### The Multi-Precision Lookup Table:

**Table: `event_geokey`**
- Stores ALL precision levels for EVERY event
- Example row:

```
event_uuid: abc-123-def
start_latlng_exact: 43.012345,-83.687654    (r6 - original)
start_lat_r3: 43.012                         (r3 - neighborhood)
start_lng_r3: -83.688
start_lat_r4: 43.0123                        (r4 - building) ← PRIMARY KEY FOR CACHE
start_lng_r4: -83.6877
start_geohash8: dpshw7g2                     (clustering)
start_geohash9: dpshw7g2h                    (clustering)
```

**Cache Lookup Process:**
1. Try r6 (exact) first
2. If miss → try r5
3. If miss → try r4 **← 90% of hits happen here**
4. If miss → try r3
5. If miss → try geohash9
6. If miss → try geohash8
7. If ALL miss → API call

### Geohash for Location Clustering:

**What is geohash:**
- Divides earth into grid cells
- Each cell has a string code: `dpshw7g2h`
- Longer string = smaller cell
- Nearby locations share the same geohash prefix

**How we use it:**
- geohash8 (`dpshw7g2`) = 38m x 19m cell
- geohash9 (`dpshw7g2h`) = 4.8m x 4.8m cell
- Use for: "Find all visits within 50m of this location"
- SQL query: `WHERE geohash8 = 'dpshw7g2'` (instant, no distance calculation)

**Example - Detecting Home Location:**
```
Query: Find locations visited 50+ times (likely home)

SELECT geohash8, COUNT(*) as visits
FROM event_geokey
GROUP BY geohash8
HAVING COUNT(*) > 50

Result: geohash8 = dpshw7g2 (123 visits)
→ This is likely home address
→ All 123 visits are within 38m x 19m cell
→ Don't need exact lat/lng to know it's "home"
```

---

## Summary of Critical Logic

### Multi-Device Detection:
- Same timestamp + distance > 100m = TWO DEVICES
- Split into device_0 and device_1
- Forensic flag for court transparency

### Path Timestamps:
- Priority 1: Waypoint times (GPS pings)
- Priority 2: Adjacent event times (bookend logic)
- Priority 3: Container fallback (Google's garbage)
- Always tag the source for transparency

### Overnight Detection:
- Simple: 22:00-07:00 hour window
- Better: True timezone + multi-day detection + location context
- Cross-reference with home_base for validation

### Geocoding Cache:
- Tier 1: Exact match in CSV (instant)
- Tier 2: Multi-precision + geohash (smart fallback)
- Tier 3: Live API (last resort, then cache)
- 99% hit rate = $0 API costs

### Coordinate Rounding:
- r3 = neighborhood (±111m)
- r4 = building (±11m) ← MAIN CACHE KEY
- r5 = exact (±1.1m)
- r6 = forensic precision (±11cm) ← STORED BUT NOT USED FOR LOOKUP
- geohash = clustering & proximity queries

---

## Verification Questions for Matt:

1. **Multi-Device:** Is 100m threshold correct? Should it vary by context (urban vs rural)?

2. **Path Timestamps:** Is 5-minute tolerance for adjacent events correct? Should it be tighter (2 min) or looser (10 min)?

3. **Overnight:** Should we use the improved version (timezone + multi-day + location context) or keep it simple (just hour window)?

4. **Cache:** Confirm radar_geocoding_master_good.csv has these 37 columns (I verified from file):
   - request_lat, request_lng, response_lat, response_lng
   - geocode_accuracy_meters, geocode_variance_flag
   - is_good_match, is_questionable, is_bad_match, needs_re_enrichment
   - label, label_type, layer, top_type, types
   - street_number, street, city, state, state_code, postal_code
   - formatted_address, place_label, address_label
   - distance_from_request, timezone_id, timezone_name, timezone_code
   - google_place_id, google_place_id_found
   - problematic_poi, problematic_notes
   - manually_verified, custom_label
   - batch_file, batch_timestamp, api_metadata

5. **Rounding:** Is r4 (±11m) the right primary key? Or should we use r5 (±1.1m) for tighter matching?
