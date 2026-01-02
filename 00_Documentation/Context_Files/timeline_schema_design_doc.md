# Timeline Ingestion System - Database Schema Design Document

## Table of Contents
1. [Overview](#overview)
2. [Architecture Principles](#architecture-principles)
3. [Component Breakdown](#component-breakdown)
4. [Design Decisions & Trade-offs](#design-decisions--trade-offs)
5. [Performance Optimization](#performance-optimization)
6. [Scalability Considerations](#scalability-considerations)
7. [Query Patterns](#query-patterns)
8. [Migration & Maintenance](#migration--maintenance)

---

## Overview

This database schema implements a comprehensive Timeline Ingestion System with three integrated components:

1. **Generic Timeline Ingestion**: Universal event storage for any temporal data
2. **Chat Extraction Timeline**: Specialized storage for conversation history with rich metadata
3. **Personal History Timeline**: Life events, relationships, and personal historical data

### Key Statistics
- **Tables**: 32 core tables
- **Indexes**: 100+ optimized indexes
- **Triggers**: 8 automatic data maintenance triggers
- **Functions**: 3 utility functions for complex queries
- **Materialized Views**: 2 for performance optimization

---

## Architecture Principles

### 1. **Inheritance Pattern with timeline_events**
The `timeline_events` table serves as the universal parent for all temporal data. This enables:
- **Unified querying** across all event types
- **Consistent temporal indexing**
- **Shared metadata** (tags, attachments, associations)
- **Polymorphic relationships** via foreign keys

**Trade-off**: Slightly more complex joins, but massive benefits in query flexibility and data consistency.

### 2. **Normalization vs. Denormalization Balance**

**Normalized**:
- Entity relationships (clean relational model)
- Source definitions
- Category hierarchies

**Denormalized**:
- `raw_data` JSONB column preserves original source data
- `processed_data` JSONB for extracted/enriched fields
- Calculated counts (reaction_count, reply_count, participant_count)

**Rationale**: Preserve source fidelity while optimizing read performance for common aggregations.

### 3. **JSONB for Schema Flexibility**
Used extensively for:
- Source-specific metadata
- Platform-specific configurations
- Extensible custom fields
- Original API responses

**Benefits**:
- No schema migrations for new source types
- Preserve all source data without loss
- GIN indexes enable efficient JSONB queries

**Trade-offs**:
- Less type safety than columns
- Must handle null checks in application
- Slightly slower than native columns

### 4. **Soft Deletes**
Implemented via `is_deleted` and `deleted_at` flags rather than hard deletes.

**Rationale**:
- Audit trail preservation
- Ability to recover accidentally deleted data
- Temporal queries can exclude deleted items via WHERE clauses
- Compliance requirements (data retention)

**Trade-off**: Database size grows, requires periodic archival strategy.

---

## Component Breakdown

### Component 1: Generic Timeline Ingestion

#### Core Tables
1. **sources** - External systems providing data
   - Tracks sync status, frequency, metadata
   - One source can provide multiple event types

2. **timeline_events** - Universal event storage
   - Parent table for all temporal data
   - Rich temporal data (start/end timestamps, timezone, precision)
   - Full-text search via tsvector
   - Importance scoring for filtering

3. **event_attachments** - Files, images, links
   - Supports cloud storage paths
   - MIME type tracking
   - Metadata for preview generation

4. **event_tags** - Flexible categorization
   - Many-to-many with confidence scores
   - Supports auto-tagging systems
   - Usage tracking for tag suggestions

#### Key Design Decisions

**Why UUIDs instead of serial integers?**
- Distributed systems compatibility
- No collision risk when merging data from multiple sources
- Can generate IDs client-side
- 128-bit uniqueness across all databases

**Why separate timeline_events.event_timestamp and event_end_timestamp?**
- Supports both point-in-time and duration events
- Enables range queries with GiST indexes
- Calendar/meeting events need duration
- Messages are typically point-in-time

**Why importance_score column?**
- Enables filtering by significance
- Can be ML-generated or user-defined
- Supports "highlight reel" queries
- Normalized 0.0-1.0 for consistency

### Component 2: Chat Extraction Timeline

#### Core Tables
1. **chat_platforms** → **chat_channels** → **chat_messages**
   - Hierarchical organization: platform > channel > message
   - Supports Slack, Discord, Teams, WhatsApp, etc.

2. **chat_participants** - Who's in which channel
   - Temporal tracking (joined_at, left_at)
   - Role-based access (owner, admin, member, guest)

3. **message_reactions** - Emoji reactions
   - Unique constraint prevents duplicate reactions
   - Auto-updates reaction_count trigger

4. **message_mentions** - @-mentions tracking
   - Critical for notification systems
   - Supports different mention types (@here, @channel)

5. **message_links** - URL extraction
   - Enables link preview generation
   - Domain tracking for analytics
   - Metadata for rich embeds

#### Key Design Decisions

**Why extend timeline_events for chat_messages?**
- Unified temporal queries across all data
- Share tagging, attachment, search infrastructure
- Consistent audit trail
- Join chat data with life events

**Why separate message_text and message_html?**
- Preserve formatting for rendering
- message_text for search and ML processing
- message_html for UI display
- Some platforms provide both

**Why track thread_root_id separately from parent_message_id?**
- `parent_message_id`: Direct reply relationship
- `thread_root_id`: Top-level thread identifier
- Enables efficient "show all replies in thread" queries
- Slack/Discord style threading support

**Why store external_message_id?**
- Deduplication during sync
- References back to source platform
- Enables incremental updates
- Supports "view in platform" links

### Component 3: Personal History Timeline

#### Core Tables
1. **life_events** - Major personal events
   - Extends timeline_events
   - Category hierarchy support
   - Significance and emotional valence
   - Privacy levels

2. **relationships** - Entity connections
   - Temporal tracking (started_at, ended_at)
   - Relationship strength metric
   - Bidirectional tracking
   - Support for multiple relationship types

3. **locations** - Geographic data
   - Full address decomposition
   - Coordinates for mapping
   - Location types (residence, workplace, venue)

4. **life_event_participants** - Who was involved
   - Many-to-many linking
   - Role tracking (organizer, participant, witness)

#### Key Design Decisions

**Why date_precision column?**
- Historical events may only have year/month
- "Graduated college in May 2015" vs "May 15, 2015 at 2:00 PM"
- Enables accurate historical timelines
- Supports approximate dates with is_approximate flag

**Why emotional_valence tracking?**
- Enables sentiment analysis of life trajectory
- Filter for "positive milestones" or "challenges overcome"
- Supports mental health applications
- Data for personal analytics

**Why privacy_level column?**
- Multi-user systems need access control
- Sharing personal timelines with family/friends
- Public profile generation
- GDPR compliance (user controls what's shared)

**Why separate locations table vs. storing in life_events?**
- Location reuse (multiple events at same place)
- Geocoding once, reference many times
- Analytics by location
- Future: "Show me all events in New York"

**Why relationship strength metric?**
- Not all relationships are equal
- Enables network analysis (close friends vs. acquaintances)
- Can be ML-generated from interaction frequency
- Supports "most important relationships" queries

---

## Design Decisions & Trade-offs

### 1. **Single timeline_events Table vs. Separate Tables**

**Decision**: Single inheritance table with type-specific extensions

**Alternatives Considered**:
- Fully denormalized (all event types in one massive table)
- Completely separate tables per type (no shared structure)

**Why This Approach**:
✅ Unified temporal queries across all event types
✅ Shared infrastructure (tags, attachments, search)
✅ Consistent audit trail
✅ Type-specific optimizations via child tables

**Trade-offs**:
❌ More complex joins for type-specific queries
❌ Cannot have NOT NULL constraints on type-specific fields in parent
❌ Slightly larger indexes due to heterogeneous data

### 2. **JSONB vs. Additional Columns**

**Decision**: Use JSONB for source-specific and extensible data

**Why**:
- 50+ different chat platforms, each with unique metadata
- API responses change over time
- No schema migrations for new fields
- Preserves complete source fidelity

**When NOT to use JSONB**:
- Frequently filtered/sorted columns → Use native types
- Foreign key relationships → Use proper table references
- Data with strict validation → Use columns with constraints

**Example**:
```sql
-- Bad: Frequently queried data in JSONB
SELECT * FROM timeline_events WHERE raw_data->>'importance' > '0.8';

-- Good: Dedicated column with index
SELECT * FROM timeline_events WHERE importance_score > 0.8;
```

### 3. **Normalization: Tags Table vs. Text Array**

**Decision**: Separate tags table with many-to-many relationship

**Alternatives**:
- `tags TEXT[]` column on timeline_events
- No tags, just free-text labels in JSONB

**Why Separate Table**:
✅ Tag reuse (store once, reference many)
✅ Tag hierarchy support (parent_tag_id)
✅ Usage tracking for autocomplete
✅ Consistent naming (avoid "work" vs "Work" vs "WORK")
✅ Batch tag operations (rename, merge, delete)

**Trade-offs**:
❌ Additional join for queries
❌ More complex INSERT operations

**Performance Impact**: Minimal with proper indexing. The GIN index on tag arrays would be similar in size.

### 4. **Soft vs. Hard Deletes**

**Decision**: Soft deletes with is_deleted flag

**Why**:
- Regulatory compliance (data retention requirements)
- User error recovery ("I didn't mean to delete that")
- Audit trail completeness
- Temporal queries can analyze deleted data

**Implementation**:
```sql
-- Application enforces soft delete
UPDATE timeline_events SET is_deleted = true, deleted_at = NOW() WHERE event_id = ?;

-- Queries filter deleted by default
SELECT * FROM timeline_events WHERE is_deleted = false;

-- Partial index excludes deleted from common queries
CREATE INDEX idx_timeline_active ON timeline_events(event_timestamp) WHERE is_deleted = false;
```

**Trade-offs**:
❌ Database grows indefinitely
❌ Must remember to filter is_deleted in queries
❌ Requires archival strategy

**Mitigation**: Scheduled archival job moves records older than N years to archive tables.

### 5. **Materialized Views for Performance**

**Decision**: Pre-compute common aggregations

**Examples**:
- `daily_event_summary`: Events per day/source/type
- `entity_activity_summary`: Per-user activity metrics

**Why**:
- Complex aggregations take 10+ seconds on millions of rows
- Summary queries are common (dashboards, analytics)
- Data doesn't need to be real-time (daily refresh OK)

**Trade-offs**:
❌ Storage overhead (duplicated data)
❌ Refresh time (locks during REFRESH)
❌ Stale data between refreshes

**Best Practices**:
```sql
-- Concurrent refresh (doesn't lock reads)
REFRESH MATERIALIZED VIEW CONCURRENTLY daily_event_summary;

-- Scheduled refresh via pg_cron
SELECT cron.schedule('refresh-daily', '0 3 * * *', $$
    REFRESH MATERIALIZED VIEW CONCURRENTLY daily_event_summary;
$$);
```

### 6. **UUID vs. BIGSERIAL for Primary Keys**

**Decision**: UUID for all primary keys

**Why**:
✅ Distributed system friendly (no coordination needed)
✅ Client-side ID generation
✅ Merge data from multiple databases without conflicts
✅ External API safety (no sequential ID enumeration)
✅ Sharding-friendly

**Trade-offs**:
❌ 16 bytes vs. 8 bytes (double the storage)
❌ Slightly slower index lookups (non-sequential)
❌ Less human-readable

**Performance Impact**:
- Index size: ~2x larger
- Query speed: ~10-20% slower for PK lookups
- Worth it for scalability and security

**When to use BIGSERIAL**: Small, internal-only tables with high insert rates where sequential ordering helps.

### 7. **Triggers for Derived Data**

**Decision**: Auto-update counts and vectors via triggers

**Examples**:
- `reaction_count` updates when reactions added/removed
- `message_vector` auto-generates from message_text
- `updated_at` timestamps

**Why**:
✅ Application doesn't need to remember
✅ Data consistency guaranteed
✅ Eliminates application bugs
✅ Works across all clients (APIs, admin tools, SQL clients)

**Trade-offs**:
❌ Insert/update performance cost
❌ Hidden behavior (less transparent)
❌ Harder to debug

**When NOT to use triggers**:
- Complex business logic (belongs in application)
- Cross-database operations
- Long-running operations (use queue instead)

**Performance Impact**:
- Full-text vector generation: ~5-10ms per row
- Count updates: <1ms per row
- Acceptable for OLTP workloads

---

## Performance Optimization

### Index Strategy

#### 1. **Temporal Queries (Most Critical)**
```sql
-- Timestamp DESC for timeline queries
CREATE INDEX idx_timeline_events_timestamp
    ON timeline_events(event_timestamp DESC)
    WHERE is_deleted = false;

-- Composite index for filtered temporal queries
CREATE INDEX idx_timeline_events_source
    ON timeline_events(source_id, event_timestamp DESC);
```

**Why DESC?** Timeline queries almost always want newest-first.

#### 2. **Full-Text Search**
```sql
-- GIN index for tsvector (fast text search)
CREATE INDEX idx_timeline_events_content_vector
    ON timeline_events USING GIN(content_vector);

-- Trigram index for fuzzy/partial matching
CREATE INDEX idx_chat_messages_text_trgm
    ON chat_messages USING GIN(message_text gin_trgm_ops);
```

**Use Cases**:
- `content_vector`: "Find events about 'project planning'"
- `gin_trgm_ops`: "Find messages containing 'proj plan'" (typos, abbreviations)

#### 3. **JSONB Queries**
```sql
-- GIN index for JSONB containment/existence
CREATE INDEX idx_timeline_events_raw_data
    ON timeline_events USING GIN(raw_data);
```

**Supports**:
- `WHERE raw_data @> '{"status": "completed"}'`
- `WHERE raw_data ? 'priority'`
- `WHERE raw_data->>'user_id' = '123'`

#### 4. **Partial Indexes (Critical for Performance)**
```sql
-- Only index active records
CREATE INDEX idx_sources_last_sync
    ON sources(last_sync_at DESC)
    WHERE is_active = true;

-- Only index current relationships
CREATE INDEX idx_relationships_current
    ON relationships(is_current)
    WHERE is_current = true;
```

**Benefits**:
- 50-90% smaller indexes (faster, less storage)
- Matches WHERE clauses in queries (index-only scans)
- Significantly faster updates (fewer index entries)

#### 5. **GiST for Range Queries**
```sql
-- Overlapping time range queries
CREATE INDEX idx_timeline_events_range
    ON timeline_events USING GIST(
        tstzrange(event_timestamp, event_end_timestamp)
    );
```

**Use Case**: "Find all events that overlap with this time period"
```sql
SELECT * FROM timeline_events
WHERE tstzrange(event_timestamp, event_end_timestamp)
    && tstzrange('2024-01-01', '2024-01-31');
```

### Query Performance Best Practices

#### 1. **Use Covering Indexes**
```sql
-- Bad: Requires table lookup
CREATE INDEX idx_events_type ON timeline_events(event_type);

-- Good: Index includes needed columns
CREATE INDEX idx_events_type_covering
    ON timeline_events(event_type, event_timestamp, title)
    WHERE is_deleted = false;
```

#### 2. **Leverage Partial Indexes in WHERE Clauses**
```sql
-- Takes advantage of partial index
SELECT * FROM timeline_events
WHERE is_deleted = false
    AND event_timestamp > NOW() - INTERVAL '30 days'
ORDER BY event_timestamp DESC;
```

#### 3. **Use EXPLAIN ANALYZE**
```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM timeline_events
WHERE event_type = 'message'
ORDER BY event_timestamp DESC
LIMIT 100;
```

Watch for:
- Sequential scans on large tables (add index)
- Index scans with high row counts (refine WHERE)
- Nested loop joins with large row counts (consider hash join)

---

## Scalability Considerations

### 1. **Table Partitioning**

**When to Partition**: timeline_events exceeds 10M rows

**Strategy**: Partition by time range
```sql
-- Parent table
CREATE TABLE timeline_events (...) PARTITION BY RANGE (event_timestamp);

-- Partitions
CREATE TABLE timeline_events_2024_q1
    PARTITION OF timeline_events
    FOR VALUES FROM ('2024-01-01') TO ('2024-04-01');

CREATE TABLE timeline_events_2024_q2
    PARTITION OF timeline_events
    FOR VALUES FROM ('2024-04-01') TO ('2024-07-01');
```

**Benefits**:
- Queries on recent data only scan relevant partitions
- Easy archival (detach old partitions)
- Parallel query execution across partitions
- Faster VACUUM/ANALYZE on smaller partitions

**Trade-offs**:
- More complex schema management
- Global indexes slower
- Application must handle partition boundaries

### 2. **TimescaleDB Extension**

**Alternative to manual partitioning**: Use TimescaleDB for time-series data

```sql
-- Convert to hypertable (automatic partitioning)
SELECT create_hypertable('timeline_events', 'event_timestamp');

-- Automatic compression of old data
ALTER TABLE timeline_events SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'source_id,event_type'
);

-- Auto-compress data older than 90 days
SELECT add_compression_policy('timeline_events', INTERVAL '90 days');
```

**Benefits**:
- Automatic partition management
- Compression (10x space savings on old data)
- Continuous aggregates (better than materialized views)
- Time-series specific optimizations

### 3. **Sharding Strategy**

**When**: Database exceeds single-server capacity (100M+ events)

**Sharding Key Options**:
1. **entity_id**: All data for a user on one shard (good for user queries)
2. **source_id**: All data from a source on one shard (good for ingestion)
3. **time range**: Recent data on fast servers, old on slow (cost optimization)

**Recommended**: Shard by entity_id for user-facing queries

**Implementation**: Postgres Citus extension or application-level routing

### 4. **Read Replicas**

**Setup**: 1 primary (writes), N replicas (reads)

**Use Cases**:
- Analytics queries → replica
- Dashboard queries → replica
- API writes → primary
- API reads → replica

**Connection Routing**:
```python
# Application code
if query_type == 'read':
    conn = read_replica_pool.get_connection()
else:
    conn = primary_db_pool.get_connection()
```

### 5. **Archival Strategy**

**Problem**: Database grows indefinitely with soft deletes

**Solution**: Age-based archival
```sql
-- Create archive table (identical structure)
CREATE TABLE timeline_events_archive (LIKE timeline_events);

-- Move old deleted events (>1 year)
INSERT INTO timeline_events_archive
SELECT * FROM timeline_events
WHERE is_deleted = true
    AND deleted_at < NOW() - INTERVAL '1 year';

DELETE FROM timeline_events
WHERE is_deleted = true
    AND deleted_at < NOW() - INTERVAL '1 year';

-- Vacuum to reclaim space
VACUUM FULL timeline_events;
```

**Schedule**: Monthly via pg_cron

**Alternative**: Use PostgreSQL table partitioning and simply drop old partitions.

---

## Query Patterns

### 1. **Timeline Queries**

#### Recent Events
```sql
SELECT
    te.event_id,
    te.title,
    te.event_timestamp,
    s.source_name,
    te.event_type
FROM timeline_events te
JOIN sources s ON te.source_id = s.source_id
WHERE te.is_deleted = false
    AND te.event_timestamp > NOW() - INTERVAL '7 days'
ORDER BY te.event_timestamp DESC
LIMIT 100;
```

#### Filtered Timeline
```sql
SELECT *
FROM timeline_events
WHERE is_deleted = false
    AND event_type IN ('message', 'meeting')
    AND importance_score > 0.7
    AND event_timestamp BETWEEN '2024-01-01' AND '2024-12-31'
ORDER BY event_timestamp DESC;
```

### 2. **Search Queries**

#### Full-Text Search
```sql
SELECT
    event_id,
    title,
    description,
    event_timestamp,
    ts_rank(content_vector, query) as relevance
FROM timeline_events,
     websearch_to_tsquery('english', 'project planning') as query
WHERE content_vector @@ query
    AND is_deleted = false
ORDER BY relevance DESC, event_timestamp DESC
LIMIT 50;
```

#### Fuzzy Search (Typo-Tolerant)
```sql
SELECT
    message_id,
    message_text,
    similarity(message_text, 'proj plan') as similarity_score
FROM chat_messages
WHERE message_text % 'proj plan'  -- % is similarity operator
ORDER BY similarity_score DESC
LIMIT 20;
```

### 3. **Aggregation Queries**

#### Daily Activity Summary
```sql
SELECT
    DATE(event_timestamp) as day,
    event_type,
    COUNT(*) as event_count,
    AVG(importance_score) as avg_importance
FROM timeline_events
WHERE event_timestamp > NOW() - INTERVAL '30 days'
    AND is_deleted = false
GROUP BY DATE(event_timestamp), event_type
ORDER BY day DESC, event_count DESC;
```

#### Top Chat Participants
```sql
SELECT
    e.display_name,
    COUNT(cm.message_id) as message_count,
    COUNT(DISTINCT DATE(te.event_timestamp)) as active_days,
    MAX(te.event_timestamp) as last_message_at
FROM chat_messages cm
JOIN entities e ON cm.sender_id = e.entity_id
JOIN timeline_events te ON cm.message_id = te.event_id
WHERE cm.channel_id = 'channel-uuid'
    AND te.event_timestamp > NOW() - INTERVAL '30 days'
GROUP BY e.entity_id, e.display_name
ORDER BY message_count DESC
LIMIT 10;
```

### 4. **Relationship Queries**

#### Direct Relationships
```sql
SELECT
    e.display_name,
    r.relationship_type,
    r.relationship_subtype,
    r.strength,
    r.started_at
FROM relationships r
JOIN entities e ON (
    CASE
        WHEN r.entity_id_1 = 'my-uuid' THEN r.entity_id_2
        ELSE r.entity_id_1
    END = e.entity_id
)
WHERE (r.entity_id_1 = 'my-uuid' OR r.entity_id_2 = 'my-uuid')
    AND r.is_current = true
ORDER BY r.strength DESC;
```

#### Network Analysis (2nd Degree Connections)
```sql
WITH my_connections AS (
    SELECT
        CASE
            WHEN entity_id_1 = 'my-uuid' THEN entity_id_2
            ELSE entity_id_1
        END as connection_id
    FROM relationships
    WHERE (entity_id_1 = 'my-uuid' OR entity_id_2 = 'my-uuid')
        AND is_current = true
)
SELECT
    e.display_name,
    COUNT(*) as mutual_connections
FROM relationships r
JOIN entities e ON (
    CASE
        WHEN r.entity_id_1 IN (SELECT connection_id FROM my_connections)
        THEN r.entity_id_2
        ELSE r.entity_id_1
    END = e.entity_id
)
WHERE (r.entity_id_1 IN (SELECT connection_id FROM my_connections)
    OR r.entity_id_2 IN (SELECT connection_id FROM my_connections))
    AND r.is_current = true
    AND e.entity_id != 'my-uuid'
    AND e.entity_id NOT IN (SELECT connection_id FROM my_connections)
GROUP BY e.entity_id, e.display_name
ORDER BY mutual_connections DESC;
```

### 5. **Conversation Thread Queries**

#### Entire Thread with Replies
```sql
WITH RECURSIVE thread AS (
    -- Root message
    SELECT
        cm.message_id,
        cm.message_text,
        cm.parent_message_id,
        te.event_timestamp,
        e.display_name,
        0 as depth,
        ARRAY[cm.message_id] as path
    FROM chat_messages cm
    JOIN timeline_events te ON cm.message_id = te.event_id
    JOIN entities e ON cm.sender_id = e.entity_id
    WHERE cm.message_id = 'root-message-uuid'

    UNION ALL

    -- Recursive: replies
    SELECT
        cm.message_id,
        cm.message_text,
        cm.parent_message_id,
        te.event_timestamp,
        e.display_name,
        t.depth + 1,
        t.path || cm.message_id
    FROM chat_messages cm
    JOIN timeline_events te ON cm.message_id = te.event_id
    JOIN entities e ON cm.sender_id = e.entity_id
    JOIN thread t ON cm.parent_message_id = t.message_id
    WHERE NOT cm.message_id = ANY(t.path)  -- Prevent cycles
)
SELECT * FROM thread
ORDER BY event_timestamp;
```

### 6. **Life Event Queries**

#### Life Timeline with Categories
```sql
SELECT
    le.event_title,
    le.event_description,
    lec.category_name,
    te.event_timestamp,
    le.significance_level,
    le.emotional_valence,
    ARRAY_AGG(e.display_name) as participants
FROM life_events le
JOIN timeline_events te ON le.life_event_id = te.event_id
LEFT JOIN life_event_categories lec ON le.category_id = lec.category_id
LEFT JOIN life_event_participants lep ON le.life_event_id = lep.life_event_id
LEFT JOIN entities e ON lep.entity_id = e.entity_id
WHERE le.entity_id = 'my-uuid'
    AND te.event_timestamp > '2020-01-01'
GROUP BY le.life_event_id, le.event_title, le.event_description,
         lec.category_name, te.event_timestamp, le.significance_level,
         le.emotional_valence
ORDER BY te.event_timestamp DESC;
```

#### Shared Experiences (Events with Common Participants)
```sql
SELECT
    le1.event_title as my_event,
    le2.event_title as their_event,
    COUNT(*) as shared_participants
FROM life_event_participants lep1
JOIN life_event_participants lep2 ON lep1.entity_id = lep2.entity_id
JOIN life_events le1 ON lep1.life_event_id = le1.life_event_id
JOIN life_events le2 ON lep2.life_event_id = le2.life_event_id
WHERE le1.entity_id = 'my-uuid'
    AND le2.entity_id = 'their-uuid'
    AND le1.life_event_id != le2.life_event_id
GROUP BY le1.life_event_id, le2.life_event_id,
         le1.event_title, le2.event_title
HAVING COUNT(*) >= 2
ORDER BY shared_participants DESC;
```

---

## Migration & Maintenance

### Initial Setup

```bash
# 1. Create database
createdb timeline_ingestion

# 2. Run schema
psql timeline_ingestion < timeline_ingestion_schema.sql

# 3. Verify
psql timeline_ingestion -c "\dt"  # List tables
psql timeline_ingestion -c "\di"  # List indexes
```

### Regular Maintenance

#### Daily
```sql
-- Refresh materialized views
REFRESH MATERIALIZED VIEW CONCURRENTLY daily_event_summary;
REFRESH MATERIALIZED VIEW CONCURRENTLY entity_activity_summary;
```

#### Weekly
```sql
-- Update statistics for query planner
ANALYZE timeline_events;
ANALYZE chat_messages;
ANALYZE life_events;
```

#### Monthly
```sql
-- Reclaim space from deleted rows
VACUUM (ANALYZE) timeline_events;
VACUUM (ANALYZE) chat_messages;

-- Check index bloat
SELECT schemaname, tablename,
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Reindex if needed (slow, use CONCURRENTLY in production)
REINDEX INDEX CONCURRENTLY idx_timeline_events_timestamp;
```

#### Quarterly
```sql
-- Archive old deleted records
INSERT INTO timeline_events_archive
SELECT * FROM timeline_events
WHERE is_deleted = true AND deleted_at < NOW() - INTERVAL '1 year';

DELETE FROM timeline_events
WHERE is_deleted = true AND deleted_at < NOW() - INTERVAL '1 year';

VACUUM FULL timeline_events;
```

### Monitoring Queries

#### Table Sizes
```sql
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) AS table_size,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) -
                   pg_relation_size(schemaname||'.'||tablename)) AS index_size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

#### Slow Queries
```sql
SELECT
    query,
    calls,
    total_time,
    mean_time,
    max_time
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 20;
```

#### Index Usage
```sql
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan as index_scans,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched
FROM pg_stat_user_indexes
WHERE idx_scan = 0  -- Unused indexes
    AND schemaname = 'public'
ORDER BY pg_relation_size(indexrelid) DESC;
```

### Backup Strategy

#### Full Backup (Daily)
```bash
pg_dump -Fc timeline_ingestion > timeline_$(date +%Y%m%d).dump
```

#### Point-in-Time Recovery Setup
```bash
# Enable WAL archiving in postgresql.conf
archive_mode = on
archive_command = 'cp %p /backup/wal/%f'
wal_level = replica

# Base backup
pg_basebackup -D /backup/base -Fp -Xs -P
```

#### Restore
```bash
# From dump
pg_restore -d timeline_ingestion timeline_20240101.dump

# Point-in-time
# 1. Restore base backup
# 2. Create recovery.conf with recovery target time
# 3. Start PostgreSQL
```

---

## Additional Considerations

### Security

1. **Row-Level Security (RLS)**
```sql
-- Enable RLS
ALTER TABLE life_events ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only see their own events
CREATE POLICY life_events_isolation ON life_events
    USING (entity_id = current_setting('app.current_user_id')::UUID);
```

2. **Sensitive Data**
- Encrypt `message_text` for private channels
- Hash external IDs for privacy
- Audit trail for data access

### Compliance

1. **GDPR Right to Erasure**
```sql
-- Function to anonymize user data
CREATE FUNCTION anonymize_entity(entity_uuid UUID) RETURNS void AS $$
BEGIN
    UPDATE entities SET
        display_name = 'Deleted User',
        email = NULL,
        metadata = '{}'
    WHERE entity_id = entity_uuid;

    UPDATE chat_messages SET
        message_text = '[deleted]',
        message_html = '[deleted]'
    WHERE sender_id = entity_uuid;

    -- Mark timeline events as deleted
    UPDATE timeline_events SET is_deleted = true
    WHERE entity_id = entity_uuid OR event_id IN (
        SELECT message_id FROM chat_messages WHERE sender_id = entity_uuid
    );
END;
$$ LANGUAGE plpgsql;
```

2. **Data Retention**
- Automated archival after N days
- Legal hold flags to prevent deletion
- Audit log retention (7 years typical)

### Future Enhancements

1. **Graph Database Integration**
- Export relationships to Neo4j for complex network analysis
- 6+ degrees of separation queries
- Community detection

2. **Vector Embeddings for Semantic Search**
```sql
ALTER TABLE timeline_events ADD COLUMN embedding vector(1536);
CREATE INDEX ON timeline_events USING ivfflat (embedding vector_cosine_ops);

-- Query similar events
SELECT * FROM timeline_events
ORDER BY embedding <=> '[0.1, 0.2, ...]'::vector
LIMIT 10;
```

3. **Event Streaming**
- Kafka integration for real-time ingestion
- Change Data Capture (CDC) for downstream systems
- Event sourcing pattern

---

## Conclusion

This schema provides a solid foundation for a comprehensive timeline ingestion system. Key strengths:

✅ **Flexible**: Supports any event type via JSONB and inheritance
✅ **Scalable**: Partitioning, sharding, and caching strategies included
✅ **Performant**: 100+ optimized indexes for all query patterns
✅ **Maintainable**: Clear structure, documentation, utility functions
✅ **Production-Ready**: Triggers, constraints, audit trails, soft deletes

### Recommended Roadmap

**Phase 1**: Core Implementation (Weeks 1-4)
- Deploy schema
- Build ingestion pipeline for 1-2 sources
- Basic API for timeline queries

**Phase 2**: Chat Integration (Weeks 5-8)
- Slack/Discord connectors
- Full-text search implementation
- Thread reconstruction

**Phase 3**: Personal History (Weeks 9-12)
- Manual life event entry UI
- Relationship management
- Location integration

**Phase 4**: Optimization (Weeks 13-16)
- Partitioning for large datasets
- Materialized view tuning
- Query performance optimization

**Phase 5**: Advanced Features (Weeks 17+)
- ML-powered tagging
- Semantic search
- Timeline analytics dashboard
