# TraceIQ Database Schemas - Production Ready

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` lands with the workspace directory-rename step; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


## Database Tables

### 1. TIMELINE_ENRICHED TABLE

```sql
CREATE TABLE timeline_enriched (
    event_id TEXT PRIMARY KEY,
    serial_id TEXT NOT NULL UNIQUE,
    device TEXT,
    event_type TEXT NOT NULL,
    label TEXT,
    date_us TEXT,
    time_12h TEXT,
    day_of_week TEXT,
    duration TEXT,
    google_place_id TEXT,
    google_place_name TEXT,
    google_place_types TEXT,
    google_address TEXT,
    radar_place_name TEXT,
    radar_place_type TEXT,
    radar_address TEXT,
    address_display TEXT,
    map_link_coords TEXT,
    map_link_place TEXT,
    distance_miles REAL,
    delta_duration TEXT,
    overnight_flag INTEGER,
    overnight_type TEXT,
    address_street TEXT,
    address_city TEXT,
    location_geopair TEXT,
    location_fuzzy TEXT,
    delta_distance_meters REAL,
    delta_duration_seconds INTEGER,
    semantic_type_probability REAL,
    activity_type_probability REAL,
    probability REAL,
    google_api_cache_id TEXT,
    radar_api_cache_id TEXT,
    source_visit_id TEXT,
    source_activity_id TEXT,
    source_path_id TEXT,
    source_memory_id TEXT,
    original_json TEXT,
    data_source TEXT,
    processed_at TIMESTAMP,
    timestamp_utc TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 2. RADAR_API_CACHE TABLE

```sql
CREATE TABLE radar_api_cache (
    cache_id TEXT PRIMARY KEY,
    request_lat REAL NOT NULL,
    request_lng REAL NOT NULL,
    response_lat REAL,
    response_lng REAL,
    geocode_accuracy_meters REAL,
    label TEXT,
    label_type TEXT,
    layer TEXT,
    top_type TEXT,
    types TEXT,
    street_number TEXT,
    street TEXT,
    city TEXT,
    state TEXT,
    state_code TEXT,
    postal_code TEXT,
    formatted_address TEXT,
    place_label TEXT,
    address_label TEXT,
    distance_from_request REAL,
    timezone_id TEXT,
    timezone_name TEXT,
    timezone_code TEXT,
    google_place_id TEXT,
    google_place_id_found INTEGER,
    problematic_poi INTEGER,
    problematic_notes TEXT,
    manually_verified INTEGER,
    custom_label TEXT,
    batch_file TEXT,
    batch_timestamp TEXT,
    api_metadata TEXT,
    request_coords_fuzzy TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 3. GOOGLE_API_CACHE TABLE

```sql
CREATE TABLE google_api_cache (
    cache_id TEXT PRIMARY KEY,
    place_id TEXT UNIQUE NOT NULL,
    name TEXT,
    formatted_address TEXT,
    types TEXT,
    location_lat REAL,
    location_lng REAL,
    location_fuzzy TEXT,
    viewport_ne_lat REAL,
    viewport_ne_lng REAL,
    viewport_sw_lat REAL,
    viewport_sw_lng REAL,
    maps_url TEXT,
    api_response_json TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_validated TIMESTAMP
);
```

### 4. VISITS TABLE

```sql
CREATE TABLE visits (
    visit_id TEXT PRIMARY KEY,
    event_serial_id TEXT NOT NULL UNIQUE,
    hierarchy_level INTEGER,
    start_timestamp_raw TEXT NOT NULL,
    end_timestamp_raw TEXT NOT NULL,
    start_timestamp_utc TEXT NOT NULL,
    end_timestamp_utc TEXT NOT NULL,
    start_timezone_offset_minutes INTEGER,
    end_timezone_offset_minutes INTEGER,
    duration_seconds INTEGER,
    visit_detection_probability REAL,
    semantic_type TEXT,
    semantic_type_probability REAL,
    visit_place_id TEXT,
    visit_geopair TEXT,
    parent_id TEXT,
    memory_id TEXT,
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data_source TEXT DEFAULT 'google_timeline',
    meta_json TEXT
);
```

### 5. ACTIVITIES TABLE

```sql
CREATE TABLE activities (
    activity_id TEXT PRIMARY KEY,
    event_serial_id TEXT NOT NULL UNIQUE,
    start_timestamp_raw TEXT NOT NULL,
    end_timestamp_raw TEXT NOT NULL,
    start_timestamp_utc TEXT NOT NULL,
    end_timestamp_utc TEXT NOT NULL,
    start_timezone_offset_minutes INTEGER,
    end_timezone_offset_minutes INTEGER,
    duration_seconds INTEGER,
    activity_type TEXT NOT NULL,
    activity_type_probability REAL,
    distance_meters REAL,
    activity_start_geopair TEXT NOT NULL,
    activity_end_geopair TEXT NOT NULL,
    activity_place_id_start TEXT,
    activity_place_id_end TEXT,
    parent_id TEXT,
    memory_id TEXT,
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data_source TEXT DEFAULT 'google_timeline',
    meta_json TEXT
);
```

### 6. TIMELINE_PATHS TABLE

```sql
CREATE TABLE timeline_paths (
    path_id TEXT NOT NULL,
    point_id TEXT PRIMARY KEY,
    parent_id TEXT,
    path_type TEXT,
    start_timestamp_raw TEXT,
    end_timestamp_raw TEXT,
    start_timestamp_utc TEXT,
    end_timestamp_utc TEXT,
    start_timezone_offset_minutes INTEGER,
    end_timezone_offset_minutes INTEGER,
    duration_seconds INTEGER,
    path_start_geopair TEXT,
    path_end_geopair TEXT,
    waypoints_count INTEGER,
    point_sequence INTEGER,
    point_geopair TEXT,
    point_timestamp_raw TEXT,
    point_timestamp_utc TEXT,
    multi_device_split INTEGER DEFAULT 0,
    device_index INTEGER,
    split_from_segment INTEGER,
    aligned_activity_id TEXT,
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data_source TEXT DEFAULT 'google_timeline',
    meta_json TEXT
);
```

### 7. MEMORIES_TRIPS TABLE

```sql
CREATE TABLE memories_trips (
    memory_id TEXT PRIMARY KEY,
    event_serial_id TEXT NOT NULL UNIQUE,
    start_timestamp_raw TEXT NOT NULL,
    end_timestamp_raw TEXT NOT NULL,
    start_timestamp_utc TEXT NOT NULL,
    end_timestamp_utc TEXT NOT NULL,
    start_timezone_offset_minutes INTEGER,
    end_timezone_offset_minutes INTEGER,
    duration_seconds INTEGER,
    trip_distance_from_origin_km INTEGER,
    trip_destination_place_ids TEXT,
    parent_id TEXT,
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data_source TEXT DEFAULT 'google_timeline',
    meta_json TEXT
);
```

### 8. ENRICHMENT_QUEUE TABLE

```sql
CREATE TABLE enrichment_queue (
    queue_id TEXT PRIMARY KEY,
    event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    request_lat REAL NOT NULL,
    request_lng REAL NOT NULL,
    request_coords_fuzzy TEXT NOT NULL,
    needs_radar INTEGER DEFAULT 1,
    needs_google INTEGER DEFAULT 1,
    priority INTEGER DEFAULT 5,
    event_date TEXT,
    event_city TEXT,
    attempt_count INTEGER DEFAULT 0,
    last_attempt_at TIMESTAMP,
    error_message TEXT,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);
```

## Database Indexes

### Timeline_Enriched Indexes

```sql
CREATE INDEX idx_serial_id ON timeline_enriched(serial_id);
CREATE INDEX idx_event_type_serial_id ON timeline_enriched(event_type, serial_id);
CREATE INDEX idx_date_us_time_12h ON timeline_enriched(date_us, time_12h);
CREATE INDEX idx_location_geopair ON timeline_enriched(location_geopair);
CREATE INDEX idx_location_fuzzy ON timeline_enriched(location_fuzzy);
CREATE INDEX idx_address_city ON timeline_enriched(address_city);
CREATE INDEX idx_address_street ON timeline_enriched(address_street);
CREATE INDEX idx_google_place_id ON timeline_enriched(google_place_id);
CREATE INDEX idx_radar_place_name ON timeline_enriched(radar_place_name);
CREATE INDEX idx_google_api_cache_id ON timeline_enriched(google_api_cache_id);
CREATE INDEX idx_radar_api_cache_id ON timeline_enriched(radar_api_cache_id);
CREATE INDEX idx_source_visit_id ON timeline_enriched(source_visit_id);
CREATE INDEX idx_source_activity_id ON timeline_enriched(source_activity_id);
CREATE INDEX idx_source_path_id ON timeline_enriched(source_path_id);
CREATE INDEX idx_source_memory_id ON timeline_enriched(source_memory_id);
CREATE INDEX idx_overnight_flag ON timeline_enriched(overnight_flag);
CREATE INDEX idx_distance_miles ON timeline_enriched(distance_miles);
CREATE INDEX idx_probability ON timeline_enriched(probability);
CREATE INDEX idx_processed_at ON timeline_enriched(processed_at);
CREATE INDEX idx_created_at ON timeline_enriched(created_at);
CREATE INDEX idx_timestamp_utc ON timeline_enriched(timestamp_utc);
CREATE INDEX idx_date_city ON timeline_enriched(date_us, address_city);
CREATE INDEX idx_city_event_type ON timeline_enriched(address_city, event_type);
CREATE INDEX idx_event_type_date ON timeline_enriched(event_type, date_us);
CREATE INDEX idx_fuzzy_coords_date ON timeline_enriched(location_fuzzy, date_us);
CREATE INDEX idx_overnight_date ON timeline_enriched(overnight_flag, date_us);
CREATE INDEX idx_device_serial ON timeline_enriched(device, serial_id);
```

### API Cache Indexes

```sql
CREATE INDEX idx_google_cache_place_id ON google_api_cache(place_id);
CREATE INDEX idx_google_cache_location_fuzzy ON google_api_cache(location_fuzzy);
CREATE UNIQUE INDEX idx_google_cache_place_id_unique ON google_api_cache(place_id);
CREATE INDEX idx_radar_cache_fuzzy_coords ON radar_api_cache(request_coords_fuzzy);
CREATE INDEX idx_radar_cache_city ON radar_api_cache(city);
CREATE INDEX idx_radar_cache_google_place_id ON radar_api_cache(google_place_id);
CREATE INDEX idx_radar_cache_batch_id ON radar_api_cache(batch_file);
CREATE INDEX idx_radar_cache_quality ON radar_api_cache(problematic_poi, manually_verified);
```

## Analytical Views

### vw_place_analytics

```sql
CREATE VIEW vw_place_analytics AS
WITH visit_details AS (
    SELECT
        location_fuzzy,
        COALESCE(google_place_name, radar_place_name, 'Unnamed Location') as place_name,
        COALESCE(google_place_types, radar_place_type) as place_type,
        radar_address as address,
        address_city,
        date_us,
        time_12h,
        duration,
        overnight_flag,
        day_of_week,
        map_link_coords,
        map_link_place
    FROM timeline_enriched
    WHERE event_type = 'VISIT_START'
      AND location_fuzzy IS NOT NULL
)
SELECT
    location_fuzzy,
    place_name,
    place_type,
    address,
    address_city,
    COUNT(*) as total_visits,
    COUNT(DISTINCT date_us) as unique_days_visited,
    MIN(date_us) as first_visit_date,
    MAX(date_us) as last_visit_date,
    MIN(time_12h) as earliest_arrival,
    MAX(time_12h) as latest_arrival,
    GROUP_CONCAT(DISTINCT day_of_week) as days_of_week_visited,
    SUM(CASE WHEN overnight_flag = 1 THEN 1 ELSE 0 END) as overnight_visits,
    CASE
        WHEN COUNT(*) >= 50 THEN 'Very Frequent (50+)'
        WHEN COUNT(*) >= 20 THEN 'Frequent (20-49)'
        WHEN COUNT(*) >= 10 THEN 'Regular (10-19)'
        WHEN COUNT(*) >= 5 THEN 'Occasional (5-9)'
        ELSE 'Rare (1-4)'
    END as frequency_category,
    MAX(map_link_coords) as map_link_coords,
    MAX(map_link_place) as map_link_place
FROM visit_details
GROUP BY location_fuzzy, place_name, place_type, address, address_city
ORDER BY total_visits DESC;
```

### vw_bouncy_trips

```sql
CREATE VIEW vw_bouncy_trips AS
WITH trip_sequences AS (
    SELECT
        event_id,
        event_type,
        date_us,
        time_12h,
        address_city,
        location_geopair,
        duration,
        LAG(address_city) OVER (ORDER BY serial_id) as prev_city,
        LAG(location_geopair) OVER (ORDER BY serial_id) as prev_geopair,
        LAG(time_12h) OVER (ORDER BY serial_id) as prev_time,
        LEAD(address_city) OVER (ORDER BY serial_id) as next_city,
        LEAD(location_geopair) OVER (ORDER BY serial_id) as next_geopair,
        ROW_NUMBER() OVER (ORDER BY serial_id) as event_position
    FROM timeline_enriched
    WHERE event_type IN ('VISIT_START', 'ACTIVITY_START')
)
SELECT
    event_id,
    date_us,
    time_12h,
    prev_city,
    address_city,
    next_city,
    event_type,
    CASE
        WHEN prev_city IS NOT NULL AND prev_city = address_city AND next_city IS NOT NULL AND next_city != address_city THEN 'BOUNCY_LEFT'
        WHEN prev_city IS NOT NULL AND prev_city != address_city AND address_city = next_city THEN 'BOUNCY_RETURN'
        WHEN prev_city IS NOT NULL AND prev_city IS NOT NULL AND address_city IS NOT NULL
            AND prev_city != address_city AND address_city != next_city AND next_city IS NOT NULL
            AND prev_city = next_city THEN 'BOUNCY_PING_PONG'
        ELSE 'NORMAL'
    END as bounce_pattern,
    CASE
        WHEN (prev_city IS NOT NULL AND prev_city != address_city AND address_city != next_city) THEN 1
        ELSE 0
    END as is_anomalous
FROM trip_sequences
WHERE prev_city IS NOT NULL OR next_city IS NOT NULL;
```

### vw_route_patterns

```sql
CREATE VIEW vw_route_patterns AS
WITH ordered_events AS (
    SELECT
        location_fuzzy,
        address_city,
        radar_place_name,
        serial_id,
        LAG(location_fuzzy) OVER (ORDER BY serial_id) as prev_location,
        LAG(address_city) OVER (ORDER BY serial_id) as prev_city,
        LAG(radar_place_name) OVER (ORDER BY serial_id) as prev_place_name,
        distance_miles,
        date_us
    FROM timeline_enriched
    WHERE event_type IN ('VISIT_START', 'ACTIVITY_START')
)
SELECT
    prev_place_name || ' to ' || radar_place_name as route,
    prev_city || ' to ' || address_city as city_route,
    COUNT(*) as times_traveled,
    AVG(distance_miles) as avg_distance_miles,
    MIN(date_us) as first_traveled,
    MAX(date_us) as last_traveled
FROM ordered_events
WHERE prev_location IS NOT NULL
  AND prev_location != location_fuzzy
GROUP BY route, city_route
HAVING times_traveled > 1
ORDER BY times_traveled DESC;
```

### vw_city_summary

```sql
CREATE VIEW vw_city_summary AS
SELECT
    address_city,
    COUNT(DISTINCT date_us) as days_visited,
    COUNT(*) as total_events,
    MIN(date_us) as first_visit,
    MAX(date_us) as last_visit,
    SUM(CASE WHEN overnight_flag = 1 THEN 1 ELSE 0 END) as overnight_events,
    ROUND(SUM(distance_miles), 1) as total_miles_traveled
FROM timeline_enriched
WHERE address_city IS NOT NULL
GROUP BY address_city
ORDER BY total_events DESC;
```
