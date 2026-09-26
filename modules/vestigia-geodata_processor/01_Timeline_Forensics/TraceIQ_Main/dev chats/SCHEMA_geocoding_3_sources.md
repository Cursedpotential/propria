# Geocoding Schema Design - 3 Sources (Google, Radar, Geodata)

## Design Philosophy:
- **Separate table for each API provider** - Different response structures
- **Unified request tracking** - Single table tracks all requests regardless of source
- **Multi-source resolution** - Compare results when multiple providers used
- **Full audit trail** - Track every API call for cost/forensic purposes

---

## Table 1: `geocode_request` (Unified Request Tracking)

### Purpose:
Track ALL geocoding requests regardless of which provider(s) are used.

### Schema:
```
request_uuid         TEXT PRIMARY KEY         -- UUIDv7
location_uuid        TEXT FK → location_key   -- Links to deduplicated location
request_lat          REAL NOT NULL            -- Exact lat from timeline
request_lng          REAL NOT NULL            -- Exact lng from timeline
latlng_r3            TEXT                     -- Rounded to 3 decimals (±111m)
latlng_r4            TEXT                     -- Rounded to 4 decimals (±11m) - PRIMARY LOOKUP KEY
latlng_r5            TEXT                     -- Rounded to 5 decimals (±1.1m)
geohash8             TEXT                     -- Geohash precision 8 (clustering)
geohash9             TEXT                     -- Geohash precision 9 (clustering)
providers_called     TEXT[]                   -- Array: ['radar', 'google', 'geodata']
status               TEXT                     -- 'pending', 'processing', 'completed', 'failed'
cache_hit            BOOLEAN DEFAULT FALSE    -- TRUE if satisfied by cache
priority             INTEGER DEFAULT 0        -- Queue priority (higher = sooner)
requested_at         TIMESTAMPTZ DEFAULT now()
completed_at         TIMESTAMPTZ
error_message        TEXT
created_at           TIMESTAMPTZ DEFAULT now()
```

### Indexes:
```
CREATE INDEX idx_geocode_req_r4 ON geocode_request(latlng_r4);  -- Cache lookups
CREATE INDEX idx_geocode_req_status ON geocode_request(status);
CREATE INDEX idx_geocode_req_location ON geocode_request(location_uuid);
```

---

## Table 2: `geocode_result_radar` (Radar API Responses)

### Purpose:
Store Radar reverse geocoding API responses (Matt's primary provider - 50K+ cached)

### Schema (Matches radar_geocoding_master_good.csv):
```
result_uuid                  TEXT PRIMARY KEY         -- UUIDv7
request_uuid                 TEXT FK → geocode_request
request_lat                  REAL                     -- Original request coords
request_lng                  REAL
response_lat                 REAL                     -- Radar's snapped coords
response_lng                 REAL
geocode_accuracy_meters      REAL                     -- Distance between request/response
geocode_variance_flag        BOOLEAN                  -- TRUE if accuracy > threshold
is_good_match                BOOLEAN                  -- Radar confidence flags
is_questionable              BOOLEAN
is_bad_match                 BOOLEAN
needs_re_enrichment          BOOLEAN
label                        TEXT                     -- e.g., "Residence", "Business"
label_type                   TEXT
layer                        TEXT                     -- e.g., "address", "place", "neighborhood"
top_type                     TEXT
types                        TEXT                     -- Comma-separated types
street_number                TEXT
street                       TEXT
city                         TEXT
state                        TEXT
state_code                   TEXT                     -- e.g., "MI"
postal_code                  TEXT
formatted_address            TEXT                     -- Full address string
place_label                  TEXT
address_label                TEXT
distance_from_request        REAL                     -- Meters from request to result
timezone_id                  TEXT                     -- e.g., "America/Detroit"
timezone_name                TEXT                     -- e.g., "Eastern Daylight Time"
timezone_code                TEXT                     -- e.g., "EDT"
google_place_id              TEXT                     -- If Radar found matching Google place
google_place_id_found        BOOLEAN
problematic_poi              TEXT
problematic_notes            TEXT
manually_verified            BOOLEAN DEFAULT FALSE
custom_label                 TEXT
batch_file                   TEXT                     -- Tracking info
batch_timestamp              TEXT
api_metadata                 JSONB                    -- Full raw API response
created_at                   TIMESTAMPTZ DEFAULT now()
```

### Indexes:
```
CREATE INDEX idx_radar_request ON geocode_result_radar(request_uuid);
CREATE UNIQUE INDEX idx_radar_coords ON geocode_result_radar(response_lat, response_lng);
CREATE INDEX idx_radar_address ON geocode_result_radar(formatted_address);
CREATE INDEX idx_radar_city ON geocode_result_radar(city, state_code);
```

---

## Table 3: `geocode_result_google` (Google Places API Responses)

### Purpose:
Store Google Places API responses (better for commercial locations, place names)

### Schema:
```
result_uuid                  TEXT PRIMARY KEY         -- UUIDv7
request_uuid                 TEXT FK → geocode_request
place_id                     TEXT UNIQUE NOT NULL     -- Google's place identifier
place_name                   TEXT                     -- Business/location name
place_types                  TEXT[]                   -- Array of types: ['restaurant', 'food']
formatted_address            TEXT
street_number                TEXT
street                       TEXT
city                         TEXT
state                        TEXT
state_code                   TEXT
postal_code                  TEXT
country                      TEXT
country_code                 TEXT                     -- e.g., "US"
latitude                     REAL
longitude                    REAL
viewport_ne_lat              REAL                     -- Bounding box
viewport_ne_lng              REAL
viewport_sw_lat              REAL
viewport_sw_lng              REAL
rating                       REAL                     -- Google rating (0.0-5.0)
user_ratings_total           INTEGER
price_level                  INTEGER                  -- 0-4 ($-$$$$)
business_status              TEXT                     -- "OPERATIONAL", "CLOSED_TEMPORARILY"
opening_hours                JSONB                    -- Full hours object
phone_number                 TEXT
website                      TEXT
icon_url                     TEXT
photos                       JSONB                    -- Array of photo references
confidence                   REAL                     -- Google's confidence score
raw_json                     JSONB                    -- Full API response
api_called_at                TIMESTAMPTZ
created_at                   TIMESTAMPTZ DEFAULT now()
```

### Indexes:
```
CREATE UNIQUE INDEX idx_google_place ON geocode_result_google(place_id);
CREATE INDEX idx_google_request ON geocode_result_google(request_uuid);
CREATE INDEX idx_google_coords ON geocode_result_google(latitude, longitude);
CREATE INDEX idx_google_name ON geocode_result_google(place_name);
CREATE INDEX idx_google_types ON geocode_result_google USING GIN(place_types);  -- Array index
```

---

## Table 4: `geocode_result_geodata` (Geodata API Responses - NEW 3rd Source)

### Purpose:
Backup geocoding provider for:
- International addresses (Radar/Google weak outside US)
- Radar API failures
- Cost optimization (cheaper than Google for some use cases)

### Schema:
```
result_uuid                  TEXT PRIMARY KEY         -- UUIDv7
request_uuid                 TEXT FK → geocode_request
provider_id                  TEXT                     -- Geodata's internal ID
latitude                     REAL
longitude                    REAL
formatted_address            TEXT
street_number                TEXT
street                       TEXT
city                         TEXT
state                        TEXT
state_code                   TEXT
postal_code                  TEXT
country                      TEXT
country_code                 TEXT
administrative_area_1        TEXT                     -- State/province
administrative_area_2        TEXT                     -- County
locality                     TEXT                     -- City/town
sublocality                  TEXT                     -- Neighborhood
accuracy_level               TEXT                     -- 'ROOFTOP', 'RANGE_INTERPOLATED', 'GEOMETRIC_CENTER', 'APPROXIMATE'
confidence                   REAL                     -- 0.0-1.0
place_type                   TEXT                     -- Primary type
place_types                  TEXT[]                   -- All types
timezone                     TEXT
distance_from_request        REAL
raw_json                     JSONB                    -- Full API response
api_called_at                TIMESTAMPTZ
created_at                   TIMESTAMPTZ DEFAULT now()
```

### Indexes:
```
CREATE INDEX idx_geodata_request ON geocode_result_geodata(request_uuid);
CREATE INDEX idx_geodata_coords ON geocode_result_geodata(latitude, longitude);
CREATE INDEX idx_geodata_address ON geocode_result_geodata(formatted_address);
```

---

## Table 5: `geocode_resolution` (Multi-Source Conflict Resolution)

### Purpose:
When multiple providers return different results for the same location, track which one we chose and why.

### Schema:
```
resolution_uuid              TEXT PRIMARY KEY         -- UUIDv7
request_uuid                 TEXT FK → geocode_request
radar_result_uuid            TEXT FK → geocode_result_radar (nullable)
google_result_uuid           TEXT FK → geocode_result_google (nullable)
geodata_result_uuid          TEXT FK → geocode_result_geodata (nullable)
preferred_provider           TEXT NOT NULL            -- 'radar', 'google', 'geodata'
preferred_result_uuid        TEXT NOT NULL            -- Which result we chose
resolution_method            TEXT NOT NULL            -- How we decided
disagreement_flag            BOOLEAN DEFAULT FALSE    -- TRUE if providers conflicted
disagreement_type            TEXT                     -- 'address_mismatch', 'coords_differ', 'type_conflict'
disagreement_severity        TEXT                     -- 'minor', 'moderate', 'major'
distance_between_results_m   REAL                     -- Meters between provider coords
address_similarity_score     REAL                     -- 0.0-1.0 (Levenshtein distance)
tie_break_reason             TEXT                     -- Why we chose this provider
manual_override              BOOLEAN DEFAULT FALSE    -- TRUE if human picked
override_reason              TEXT
override_by                  TEXT                     -- User who made override
resolved_at                  TIMESTAMPTZ DEFAULT now()
created_at                   TIMESTAMPTZ DEFAULT now()
```

### Resolution Logic (Automatic):
```
IF radar.confidence > 0.9 AND google.confidence > 0.9:
    IF distance_between < 20m AND address_similarity > 0.95:
        → preferred = 'radar' (cheaper, Matt's default)
        → disagreement_flag = FALSE
    ELSE:
        → preferred = provider with higher confidence
        → disagreement_flag = TRUE
        → disagreement_type = 'address_mismatch' OR 'coords_differ'

IF only one provider has result:
    → preferred = that provider
    → disagreement_flag = FALSE

IF all providers failed:
    → flag for manual review
```

### Indexes:
```
CREATE INDEX idx_resolution_request ON geocode_resolution(request_uuid);
CREATE INDEX idx_resolution_disagreement ON geocode_resolution(disagreement_flag);
CREATE INDEX idx_resolution_manual ON geocode_resolution(manual_override);
```

---

## Table 6: `geocode_audit` (Forensic Audit Trail)

### Purpose:
Track EVERY geocoding operation for:
- Cost tracking (how many API calls, which provider)
- Forensic documentation (when/how addresses were obtained)
- Error debugging
- Performance monitoring

### Schema:
```
audit_id                     BIGSERIAL PRIMARY KEY    -- Auto-increment
request_uuid                 TEXT                     -- Which request
event_uuid                   TEXT                     -- Which timeline event (if linked)
action                       TEXT NOT NULL            -- 'cache_hit', 'api_call', 'resolution', 'manual_override'
provider                     TEXT                     -- 'radar', 'google', 'geodata', 'cache'
status                       TEXT                     -- 'success', 'failure', 'partial'
api_cost_usd                 REAL                     -- Estimated cost of API call
response_time_ms             INTEGER                  -- API latency
cache_hit_level              TEXT                     -- 'r3', 'r4', 'r5', 'geohash8', 'exact', NULL
details                      JSONB                    -- Full context
error_code                   TEXT
error_message                TEXT
performed_by                 TEXT                     -- 'system', 'user:matt', 'batch_job:123'
created_at                   TIMESTAMPTZ DEFAULT now()
```

### Example Audit Entries:
```
# Cache hit
action: 'cache_hit'
provider: 'radar'
cache_hit_level: 'r4'
api_cost_usd: 0.00
response_time_ms: 2

# Live API call
action: 'api_call'
provider: 'radar'
status: 'success'
api_cost_usd: 0.006
response_time_ms: 342

# Multi-source resolution
action: 'resolution'
provider: 'multiple'
details: {'radar_conf': 0.92, 'google_conf': 0.88, 'chosen': 'radar'}
api_cost_usd: 0.012  # Both called

# Manual override
action: 'manual_override'
provider: 'google'
performed_by: 'user:matt'
details: {'reason': 'Radar returned wrong building, Google correct'}
```

### Indexes:
```
CREATE INDEX idx_audit_request ON geocode_audit(request_uuid);
CREATE INDEX idx_audit_event ON geocode_audit(event_uuid);
CREATE INDEX idx_audit_action ON geocode_audit(action);
CREATE INDEX idx_audit_time ON geocode_audit(created_at);
```

---

## Supporting Tables (From Archive Schema)

### Table 7: `location_key` (Deduplicated Location Registry)

### Purpose:
One entry per unique physical location, regardless of how many times visited.

### Schema:
```
location_uuid                TEXT PRIMARY KEY         -- UUIDv7
latlng_exact                 TEXT UNIQUE NOT NULL     -- r6 precision (forensic)
lat_r3                       REAL                     -- Neighborhood level
lng_r3                       REAL
lat_r4                       REAL                     -- Building level (MAIN KEY)
lng_r4                       REAL
lat_r5                       REAL                     -- Exact location
lng_r5                       REAL
geohash8                     TEXT                     -- Clustering
geohash9                     TEXT
visit_count                  INTEGER DEFAULT 0        -- How many times visited
first_seen                   TIMESTAMPTZ
last_seen                    TIMESTAMPTZ
created_at                   TIMESTAMPTZ DEFAULT now()
```

### Indexes:
```
CREATE UNIQUE INDEX idx_location_exact ON location_key(latlng_exact);
CREATE INDEX idx_location_r4 ON location_key(lat_r4, lng_r4);
CREATE INDEX idx_location_geohash8 ON location_key(geohash8);
CREATE INDEX idx_location_geohash9 ON location_key(geohash9);
```

---

### Table 8: `event_geokey` (Event-Specific Coordinate Rounding)

### Purpose:
For EACH timeline event, store multi-precision versions of its coordinates for cache lookups.

### Schema:
```
event_uuid                   TEXT PRIMARY KEY         -- UUIDv7, FK to visits/activities/paths
start_latlng_exact           TEXT                     -- r6 precision
start_lat_r3                 REAL
start_lng_r3                 REAL
start_lat_r4                 REAL
start_lng_r4                 REAL
start_lat_r5                 REAL
start_lng_r5                 REAL
start_geohash8               TEXT
start_geohash9               TEXT
end_latlng_exact             TEXT                     -- For activities (have end location)
end_lat_r3                   REAL
end_lng_r3                   REAL
end_lat_r4                   REAL
end_lng_r4                   REAL
end_lat_r5                   REAL
end_lng_r5                   REAL
end_geohash8                 TEXT
end_geohash9                 TEXT
created_at                   TIMESTAMPTZ DEFAULT now()
```

### Indexes:
```
CREATE INDEX idx_geokey_start_r4 ON event_geokey(start_lat_r4, start_lng_r4);
CREATE INDEX idx_geokey_end_r4 ON event_geokey(end_lat_r4, end_lng_r4);
CREATE INDEX idx_geokey_start_gh8 ON event_geokey(start_geohash8);
CREATE INDEX idx_geokey_end_gh8 ON event_geokey(end_geohash8);
```

---

## Workflow Example: Processing a New Timeline Event

### Step 1: Event Arrives
```
Visit #1234:
  latitude: 43.0123456
  longitude: -83.6876543
```

### Step 2: Create `event_geokey` Entry
```
Compute all precision levels:
  start_latlng_exact: 43.0123456,-83.6876543
  start_lat_r3: 43.012
  start_lng_r3: -83.688
  start_lat_r4: 43.0123
  start_lng_r4: -83.6877
  start_lat_r5: 43.01235
  start_lng_r5: -83.68765
  start_geohash8: dpshw7g2
  start_geohash9: dpshw7g2h
```

### Step 3: Check `location_key`
```
Lookup by latlng_exact (r6):
  → NOT FOUND

Lookup by lat_r4, lng_r4:
  → FOUND: location_uuid = abc-123
  → This is a known location (visited before)
  → Increment visit_count
  → Update last_seen
```

### Step 4: Create `geocode_request`
```
request_uuid: xyz-789
location_uuid: abc-123 (from step 3)
request_lat: 43.0123456
request_lng: -83.6876543
latlng_r4: 43.0123,-83.6877 (PRIMARY LOOKUP KEY)
status: 'pending'
```

### Step 5: Cache Lookup (Tier 1)
```
Query radar_geocoding_master_good.csv:
  WHERE request_lat = 43.0123 AND request_lng = -83.6877  (r4 lookup)
  → FOUND!

Result:
  formatted_address: "4431 E Mount Morris Rd, Mount Morris, MI 48458 US"
  city: "Mount Morris"
  state: "Michigan"

Update geocode_request:
  status: 'completed'
  cache_hit: TRUE

Create audit entry:
  action: 'cache_hit'
  provider: 'radar'
  cache_hit_level: 'r4'
  api_cost_usd: 0.00
```

### Step 6: Link to Event
```
Update visit #1234:
  geocoded_address: "4431 E Mount Morris Rd, Mount Morris, MI 48458 US"
  geocoded_city: "Mount Morris"
  geocoded_state: "Michigan"
  geocode_source: "radar_cache"
  geocode_confidence: 0.95
```

---

## Cache Miss Scenario (Tier 3 - Live API)

### If cache lookup fails at ALL precision levels:

### Step 1: Create API Request
```
Call Radar API:
  POST /geocode/reverse
  coords: 43.0123456,-83.6876543
```

### Step 2: Store Result
```
Insert into geocode_result_radar:
  result_uuid: new-456
  request_uuid: xyz-789
  formatted_address: "123 New St, Burton, MI 48519 US"
  city: "Burton"
  state: "Michigan"
  ...all 37 columns...
```

### Step 3: Update Cache File
```
Append to radar_geocoding_master_good.csv:
  43.0123456,-83.6876543,43.012345,-83.687654,0.5,False,True,...
```

### Step 4: Audit
```
Insert into geocode_audit:
  action: 'api_call'
  provider: 'radar'
  api_cost_usd: 0.006
  response_time_ms: 342
```

### Step 5: Future Requests
```
Next time we visit this location:
  → r4 lookup: 43.0123,-83.6877
  → CACHE HIT (from new entry)
  → $0 cost
```

---

## Summary: 8 Geocoding Tables

| Table | Purpose | Rows (approx) |
|-------|---------|---------------|
| geocode_request | Unified request tracking | ~100K (one per unique location visited) |
| geocode_result_radar | Radar API responses | ~50K (Matt's cache) |
| geocode_result_google | Google Places responses | ~10K (commercial locations) |
| geocode_result_geodata | Geodata responses (new) | ~5K (backup/international) |
| geocode_resolution | Multi-source conflict resolution | ~1K (only when providers disagree) |
| geocode_audit | Full audit trail | ~200K (every operation logged) |
| location_key | Deduplicated locations | ~75K (one per physical location) |
| event_geokey | Event coordinate rounding | ~300K (one per timeline event) |

**Total: 8 tables, ~740K rows across geocoding system**
