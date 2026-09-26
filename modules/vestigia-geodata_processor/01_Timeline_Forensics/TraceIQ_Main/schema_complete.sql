-- =========================================
-- TRACEIQ COMPLETE SQLITE SCHEMA
-- All indexes, views, and forensic features compatible with SQLite
-- =========================================

-- Enable foreign keys
PRAGMA foreign_keys = ON;

-- Enable JSON functions
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;

-- =========================================
-- CORE TABLES
-- =========================================

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
    processed_at TEXT,
    timestamp_utc TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    
    FOREIGN KEY (google_api_cache_id) REFERENCES google_api_cache(cache_id),
    FOREIGN KEY (radar_api_cache_id) REFERENCES radar_api_cache(cache_id)
);

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
    processed_at TEXT DEFAULT (datetime('now')),
    data_source TEXT DEFAULT 'google_timeline',
    meta_json TEXT
);

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
    processed_at TEXT DEFAULT (datetime('now')),
    data_source TEXT DEFAULT 'google_timeline',
    meta_json TEXT
);

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
    processed_at TEXT DEFAULT (datetime('now')),
    data_source TEXT DEFAULT 'google_timeline',
    meta_json TEXT
);

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
    processed_at TEXT DEFAULT (datetime('now')),
    data_source TEXT DEFAULT 'google_timeline',
    meta_json TEXT
);

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
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    last_validated TEXT
);

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
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

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
    last_attempt_at TEXT,
    error_message TEXT,
    status TEXT DEFAULT 'pending',
    created_at TEXT DEFAULT (datetime('now')),
    completed_at TEXT,
    FOREIGN KEY (event_id) REFERENCES timeline_enriched(event_id)
);

CREATE TABLE data_quality_metrics (
    metric_id TEXT PRIMARY KEY,
    check_date TEXT NOT NULL,
    table_name TEXT NOT NULL,
    metric_name TEXT NOT NULL,
    metric_value REAL,
    threshold_min REAL,
    threshold_max REAL,
    status TEXT CHECK (status IN ('PASS', 'WARN', 'FAIL')),
    details TEXT,
    created_at TEXT DEFAULT (datetime('now'))
);

-- =========================================
-- INDEXES (SQLite-Compatible)
-- =========================================

-- Primary keys already indexed
-- Foreign keys manually indexed

CREATE INDEX idx_timeline_serial ON timeline_enriched(serial_id);
CREATE INDEX idx_timeline_event_type ON timeline_enriched(event_type);
CREATE INDEX idx_timeline_date_us ON timeline_enriched(date_us);
CREATE INDEX idx_timeline_timestamp_utc ON timeline_enriched(timestamp_utc);
CREATE INDEX idx_timeline_location_fuzzy ON timeline_enriched(location_fuzzy);
CREATE INDEX idx_timeline_address_city ON timeline_enriched(address_city);
CREATE INDEX idx_timeline_google_place_id ON timeline_enriched(google_place_id);
CREATE INDEX idx_timeline_overnight ON timeline_enriched(overnight_flag);
CREATE INDEX idx_timeline_probability ON timeline_enriched(probability);
CREATE INDEX idx_timeline_device ON timeline_enriched(device);
CREATE INDEX idx_timeline_source_visit ON timeline_enriched(source_visit_id);
CREATE INDEX idx_timeline_source_activity ON timeline_enriched(source_activity_id);
CREATE INDEX idx_timeline_created ON timeline_enriched(created_at);

-- Composite indexes for common queries
CREATE INDEX idx_timeline_date_city ON timeline_enriched(date_us, address_city);
CREATE INDEX idx_timeline_city_event ON timeline_enriched(address_city, event_type);
CREATE INDEX idx_timeline_event_date ON timeline_enriched(event_type, date_us);
CREATE INDEX idx_timeline_fuzzy_date ON timeline_enriched(location_fuzzy, date_us);
CREATE INDEX idx_timeline_device_serial ON timeline_enriched(device, serial_id);

-- API Cache indexes
CREATE INDEX idx_google_cache_place_id ON google_api_cache(place_id);
CREATE INDEX idx_google_cache_fuzzy ON google_api_cache(location_fuzzy);
CREATE UNIQUE INDEX idx_google_cache_place_unique ON google_api_cache(place_id);

CREATE INDEX idx_radar_cache_fuzzy ON radar_api_cache(request_coords_fuzzy);
CREATE INDEX idx_radar_cache_city ON radar_api_cache(city);
CREATE INDEX idx_radar_cache_google_id ON radar_api_cache(google_place_id);

-- Queue indexes
CREATE INDEX idx_queue_status ON enrichment_queue(status);
CREATE INDEX idx_queue_priority ON enrichment_queue(priority);
CREATE INDEX idx_queue_event_date ON enrichment_queue(event_date);

-- =========================================
-- ANALYTICAL VIEWS
-- =========================================

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
        map_link_place,
        
        -- Convert duration to minutes
        (CAST(substr(duration, 1, instr(duration, ':')-1) AS INTEGER) * 1440 +
         CAST(substr(duration, instr(duration, ':')+1, 2) AS INTEGER) * 60 +
         CAST(substr(duration, instr(duration, ':')+4, 2) AS INTEGER)) as duration_minutes,
         
        -- Extract hour for analysis
        CASE
            WHEN time_12h LIKE '%PM' AND substr(time_12h, 1, 2) != '12' 
            THEN CAST(substr(time_12h, 1, instr(time_12h, ':')-1) AS INTEGER) + 12
            WHEN time_12h LIKE '%AM' AND substr(time_12h, 1, 2) = '12' THEN 0
            ELSE CAST(substr(time_12h, 1, instr(time_12h, ':')-1) AS INTEGER)
        END as arrival_hour
    FROM timeline_enriched
    WHERE event_type = 'VISIT_START' AND location_fuzzy IS NOT NULL
)
SELECT
    location_fuzzy,
    place_name,
    place_type,
    address,
    address_city,
    COUNT(*) as total_visits,
    COUNT(DISTINCT date_us) as unique_days_visited,
    ROUND(AVG(duration_minutes), 0) as avg_minutes_per_visit,
    MIN(duration_minutes) as shortest_visit_minutes,
    MAX(duration_minutes) as longest_visit_minutes,
    MIN(date_us) as first_visit_date,
    MAX(date_us) as last_visit_date,
    (julianday(MAX(date_us)) - julianday(MIN(date_us))) as days_span,
    MIN(time_12h) as earliest_arrival,
    MAX(time_12h) as latest_arrival,
    MIN(arrival_hour) as earliest_arrival_hour,
    MAX(arrival_hour) as latest_arrival_hour,
    GROUP_CONCAT(DISTINCT day_of_week) as days_of_week_visited,
    SUM(CASE WHEN overnight_flag = 1 THEN 1 ELSE 0 END) as overnight_visits,
    ROUND(CAST(SUM(CASE WHEN overnight_flag = 1 THEN 1 ELSE 0 END) AS REAL) / COUNT(*) * 100, 1) as overnight_percentage,
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
    prev_place_name || ' → ' || radar_place_name as route,
    prev_city || ' → ' || address_city as city_route,
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

CREATE VIEW vw_overnight_activity AS
SELECT
    date_us,
    time_12h,
    event_type,
    label,
    duration,
    overnight_type,
    google_place_name,
    radar_address,
    address_city,
    map_link_coords,
    CASE
        WHEN duration > '1:00:00' THEN 'Long Overnight'
        WHEN overnight_type = 'spans' THEN 'Spans Overnight Window'
        ELSE 'Partial Overnight'
    END as overnight_category
FROM timeline_enriched
WHERE overnight_flag = 1
ORDER BY date_us, time_12h;

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

-- =========================================
-- DATA QUALITY FUNCTIONS
-- =========================================

CREATE TABLE IF NOT EXISTS data_quality_metrics (
    metric_id TEXT PRIMARY KEY,
    check_date TEXT NOT NULL,
    table_name TEXT NOT NULL,
    metric_name TEXT NOT NULL,
    metric_value REAL,
    threshold_min REAL,
    threshold_max REAL,
    status TEXT CHECK (status IN ('PASS', 'WARN', 'FAIL')),
    details TEXT,
    created_at TEXT DEFAULT (datetime('now'))
);

-- Quality check trigger
CREATE TRIGGER IF NOT EXISTS trig_quality_check
AFTER INSERT ON timeline_enriched
BEGIN
    INSERT INTO data_quality_metrics (metric_id, check_date, table_name, metric_name, metric_value, threshold_min, threshold_max, status, details)
    VALUES (
        lower(hex(randomblob(16))),
        date('now'),
        'timeline_enriched',
        'missing_coordinates_pct',
        (SELECT COUNT(CASE WHEN location_geopair IS NULL THEN 1 END) * 100.0 / COUNT(*) FROM timeline_enriched WHERE date(created_at) = date('now')),
        0,
        5,
        CASE WHEN (SELECT COUNT(CASE WHEN location_geopair IS NULL THEN 1 END) * 100.0 / COUNT(*) FROM timeline_enriched WHERE date(created_at) = date('now')) > 5 THEN 'FAIL' ELSE 'PASS' END,
        'Percentage of records with missing coordinates today'
    );
END;

-- =========================================
-- FORENSIC EVIDENCE VIEW
-- =========================================

CREATE VIEW vw_forensic_evidence_package AS
SELECT 
    event_id as evidence_id,
    serial_id as sequence_number,
    timestamp_utc as utc_timestamp,
    date_us || ' ' || time_12h as local_timestamp,
    COALESCE(google_place_name, radar_place_name, 'Unknown') as location_name,
    radar_address as verified_address,
    address_city as city,
    location_geopair as coordinates,
    map_link_coords as map_link,
    probability as confidence_score,
    CASE 
        WHEN google_place_id IS NOT NULL AND radar_place_name IS NOT NULL THEN 'HIGH'
        WHEN google_place_id IS NOT NULL OR radar_place_name IS NOT NULL THEN 'MEDIUM'
        ELSE 'LOW'
    END as location_confidence,
    event_type,
    label as event_label,
    duration as duration_display,
    delta_duration_seconds as duration_seconds,
    overnight_flag as is_overnight,
    overnight_type as overnight_classification,
    source_visit_id,
    source_activity_id,
    data_source,
    processed_at as processing_timestamp,
    original_json as raw_evidence
    
FROM timeline_enriched
WHERE probability > 0.6
ORDER BY serial_id;

