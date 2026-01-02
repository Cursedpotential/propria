-- ============================================================================
-- TIMELINE INGESTION SYSTEM - COMPREHENSIVE DATABASE SCHEMA
-- ============================================================================
-- PostgreSQL 14+ required for advanced features
-- Extensions: uuid-ossp, pg_trgm, btree_gist, timescaledb (optional)
-- ============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For fuzzy text search
CREATE EXTENSION IF NOT EXISTS "btree_gist"; -- For exclusion constraints
CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- For encryption if needed

-- ============================================================================
-- SHARED INFRASTRUCTURE TABLES
-- ============================================================================

-- Source systems that feed data into the timeline
CREATE TABLE sources (
    source_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_name VARCHAR(255) NOT NULL UNIQUE,
    source_type VARCHAR(100) NOT NULL, -- 'chat', 'calendar', 'email', 'social', 'manual', etc.
    source_metadata JSONB DEFAULT '{}', -- API endpoints, auth configs, etc.
    is_active BOOLEAN DEFAULT true,
    sync_frequency_minutes INTEGER, -- NULL for manual-only sources
    last_sync_at TIMESTAMPTZ,
    last_sync_status VARCHAR(50), -- 'success', 'failed', 'partial'
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_source_type CHECK (source_type IN (
        'chat', 'email', 'calendar', 'social', 'document',
        'location', 'health', 'finance', 'manual', 'other'
    ))
);

CREATE INDEX idx_sources_type ON sources(source_type) WHERE is_active = true;
CREATE INDEX idx_sources_last_sync ON sources(last_sync_at DESC) WHERE is_active = true;

-- User/entity management (for multi-user support)
CREATE TABLE entities (
    entity_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type VARCHAR(50) NOT NULL, -- 'user', 'organization', 'service'
    external_id VARCHAR(255), -- ID from external system
    display_name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_entity_type CHECK (entity_type IN ('user', 'organization', 'service', 'bot'))
);

CREATE UNIQUE INDEX idx_entities_external_id ON entities(external_id) WHERE external_id IS NOT NULL;
CREATE INDEX idx_entities_email ON entities(email) WHERE email IS NOT NULL;
CREATE INDEX idx_entities_type ON entities(entity_type) WHERE is_active = true;

-- Tags for categorization across all timelines
CREATE TABLE tags (
    tag_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tag_name VARCHAR(100) NOT NULL UNIQUE,
    tag_category VARCHAR(100), -- 'topic', 'sentiment', 'priority', 'project', etc.
    tag_color VARCHAR(7), -- Hex color for UI
    parent_tag_id UUID REFERENCES tags(tag_id) ON DELETE SET NULL,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tags_category ON tags(tag_category);
CREATE INDEX idx_tags_parent ON tags(parent_tag_id) WHERE parent_tag_id IS NOT NULL;
CREATE INDEX idx_tags_usage ON tags(usage_count DESC);

-- ============================================================================
-- COMPONENT 1: TIMELINE INGESTION SYSTEM (Generic Timeline Storage)
-- ============================================================================

-- Core timeline events table
CREATE TABLE timeline_events (
    event_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_id UUID NOT NULL REFERENCES sources(source_id) ON DELETE RESTRICT,
    entity_id UUID REFERENCES entities(entity_id) ON DELETE SET NULL,

    -- Temporal data
    event_timestamp TIMESTAMPTZ NOT NULL,
    event_end_timestamp TIMESTAMPTZ, -- For duration events
    event_timezone VARCHAR(50) DEFAULT 'UTC',

    -- Event classification
    event_type VARCHAR(100) NOT NULL, -- 'message', 'meeting', 'transaction', 'activity', etc.
    event_category VARCHAR(100), -- Domain-specific categorization

    -- Core content
    title VARCHAR(500),
    description TEXT,
    raw_data JSONB NOT NULL DEFAULT '{}', -- Original data from source
    processed_data JSONB DEFAULT '{}', -- Extracted/enriched data

    -- Metadata
    external_id VARCHAR(255), -- ID from source system
    parent_event_id UUID REFERENCES timeline_events(event_id) ON DELETE CASCADE,

    -- Search and classification
    content_vector tsvector, -- For full-text search
    importance_score NUMERIC(3,2) DEFAULT 0.5, -- 0.0 to 1.0

    -- Audit
    ingested_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN DEFAULT false,
    deleted_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT valid_timestamp_range CHECK (
        event_end_timestamp IS NULL OR event_end_timestamp >= event_timestamp
    ),
    CONSTRAINT valid_importance CHECK (importance_score >= 0 AND importance_score <= 1),
    CONSTRAINT valid_external_id UNIQUE (source_id, external_id)
);

-- Indexes for timeline_events
CREATE INDEX idx_timeline_events_timestamp ON timeline_events(event_timestamp DESC)
    WHERE is_deleted = false;
CREATE INDEX idx_timeline_events_source ON timeline_events(source_id, event_timestamp DESC);
CREATE INDEX idx_timeline_events_entity ON timeline_events(entity_id, event_timestamp DESC)
    WHERE entity_id IS NOT NULL;
CREATE INDEX idx_timeline_events_type ON timeline_events(event_type, event_timestamp DESC);
CREATE INDEX idx_timeline_events_parent ON timeline_events(parent_event_id)
    WHERE parent_event_id IS NOT NULL;

-- GiST index for overlapping time ranges
CREATE INDEX idx_timeline_events_range ON timeline_events
    USING GIST (tstzrange(event_timestamp, event_end_timestamp));

-- GIN indexes for JSONB and full-text search
CREATE INDEX idx_timeline_events_raw_data ON timeline_events USING GIN(raw_data);
CREATE INDEX idx_timeline_events_processed_data ON timeline_events USING GIN(processed_data);
CREATE INDEX idx_timeline_events_content_vector ON timeline_events USING GIN(content_vector);

-- Partitioning strategy (optional - comment in if needed)
-- CREATE TABLE timeline_events_2024_q1 PARTITION OF timeline_events
--     FOR VALUES FROM ('2024-01-01') TO ('2024-04-01');

-- Event attachments (files, images, links)
CREATE TABLE event_attachments (
    attachment_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES timeline_events(event_id) ON DELETE CASCADE,

    attachment_type VARCHAR(50) NOT NULL, -- 'file', 'image', 'link', 'video'
    file_name VARCHAR(500),
    file_path TEXT, -- Local or cloud storage path
    file_url TEXT,
    file_size_bytes BIGINT,
    mime_type VARCHAR(100),

    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_attachment_type CHECK (
        attachment_type IN ('file', 'image', 'video', 'audio', 'link', 'document')
    )
);

CREATE INDEX idx_event_attachments_event ON event_attachments(event_id);
CREATE INDEX idx_event_attachments_type ON event_attachments(attachment_type);

-- Event tags (many-to-many)
CREATE TABLE event_tags (
    event_id UUID NOT NULL REFERENCES timeline_events(event_id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(tag_id) ON DELETE CASCADE,
    tagged_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    confidence_score NUMERIC(3,2) DEFAULT 1.0, -- For auto-tagging

    PRIMARY KEY (event_id, tag_id)
);

CREATE INDEX idx_event_tags_tag ON event_tags(tag_id);
CREATE INDEX idx_event_tags_confidence ON event_tags(confidence_score DESC);

-- ============================================================================
-- COMPONENT 2: CHAT EXTRACTION TIMELINE
-- ============================================================================

-- Chat platforms/channels
CREATE TABLE chat_platforms (
    platform_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_id UUID NOT NULL REFERENCES sources(source_id) ON DELETE RESTRICT,
    platform_name VARCHAR(255) NOT NULL, -- 'Slack', 'Discord', 'WhatsApp', etc.
    platform_type VARCHAR(50) NOT NULL, -- 'team', 'personal', 'public'

    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_platform_type CHECK (
        platform_type IN ('team', 'personal', 'public', 'private')
    )
);

CREATE INDEX idx_chat_platforms_source ON chat_platforms(source_id);

-- Chat channels/conversations
CREATE TABLE chat_channels (
    channel_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    platform_id UUID NOT NULL REFERENCES chat_platforms(platform_id) ON DELETE CASCADE,

    external_channel_id VARCHAR(255) NOT NULL,
    channel_name VARCHAR(255) NOT NULL,
    channel_type VARCHAR(50) NOT NULL, -- 'direct', 'group', 'channel', 'thread'

    is_private BOOLEAN DEFAULT false,
    participant_count INTEGER DEFAULT 0,

    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    archived_at TIMESTAMPTZ,

    CONSTRAINT valid_channel_type CHECK (
        channel_type IN ('direct', 'group', 'channel', 'thread', 'broadcast')
    ),
    CONSTRAINT unique_external_channel UNIQUE (platform_id, external_channel_id)
);

CREATE INDEX idx_chat_channels_platform ON chat_channels(platform_id);
CREATE INDEX idx_chat_channels_type ON chat_channels(channel_type);
CREATE INDEX idx_chat_channels_active ON chat_channels(created_at DESC)
    WHERE archived_at IS NULL;

-- Chat participants (users in channels)
CREATE TABLE chat_participants (
    participant_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    channel_id UUID NOT NULL REFERENCES chat_channels(channel_id) ON DELETE CASCADE,
    entity_id UUID NOT NULL REFERENCES entities(entity_id) ON DELETE CASCADE,

    role VARCHAR(50) DEFAULT 'member', -- 'owner', 'admin', 'member', 'guest'
    joined_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    left_at TIMESTAMPTZ,

    CONSTRAINT unique_channel_participant UNIQUE (channel_id, entity_id)
);

CREATE INDEX idx_chat_participants_channel ON chat_participants(channel_id);
CREATE INDEX idx_chat_participants_entity ON chat_participants(entity_id);
CREATE INDEX idx_chat_participants_active ON chat_participants(channel_id)
    WHERE left_at IS NULL;

-- Chat messages (extends timeline_events)
CREATE TABLE chat_messages (
    message_id UUID PRIMARY KEY REFERENCES timeline_events(event_id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES chat_channels(channel_id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES entities(entity_id) ON DELETE RESTRICT,

    -- Message structure
    parent_message_id UUID REFERENCES chat_messages(message_id) ON DELETE SET NULL,
    thread_root_id UUID REFERENCES chat_messages(message_id) ON DELETE SET NULL,
    reply_count INTEGER DEFAULT 0,

    -- Content
    message_text TEXT,
    message_html TEXT, -- Rich formatted version

    -- Message metadata
    is_edited BOOLEAN DEFAULT false,
    edited_at TIMESTAMPTZ,
    is_deleted BOOLEAN DEFAULT false,
    deleted_at TIMESTAMPTZ,

    -- Reactions and engagement
    reaction_count INTEGER DEFAULT 0,

    -- Search
    message_vector tsvector,

    -- External reference
    external_message_id VARCHAR(255),

    CONSTRAINT unique_channel_external_msg UNIQUE (channel_id, external_message_id)
);

CREATE INDEX idx_chat_messages_channel ON chat_messages(channel_id);
CREATE INDEX idx_chat_messages_sender ON chat_messages(sender_id);
CREATE INDEX idx_chat_messages_thread ON chat_messages(thread_root_id)
    WHERE thread_root_id IS NOT NULL;
CREATE INDEX idx_chat_messages_parent ON chat_messages(parent_message_id)
    WHERE parent_message_id IS NOT NULL;
CREATE INDEX idx_chat_messages_vector ON chat_messages USING GIN(message_vector);

-- Trigram index for fuzzy search
CREATE INDEX idx_chat_messages_text_trgm ON chat_messages USING GIN(message_text gin_trgm_ops);

-- Message reactions
CREATE TABLE message_reactions (
    reaction_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES chat_messages(message_id) ON DELETE CASCADE,
    reactor_id UUID NOT NULL REFERENCES entities(entity_id) ON DELETE CASCADE,

    reaction_type VARCHAR(100) NOT NULL, -- emoji or reaction name
    reacted_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_message_reactor_type UNIQUE (message_id, reactor_id, reaction_type)
);

CREATE INDEX idx_message_reactions_message ON message_reactions(message_id);
CREATE INDEX idx_message_reactions_reactor ON message_reactions(reactor_id);
CREATE INDEX idx_message_reactions_type ON message_reactions(reaction_type);

-- Message mentions
CREATE TABLE message_mentions (
    mention_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES chat_messages(message_id) ON DELETE CASCADE,
    mentioned_entity_id UUID NOT NULL REFERENCES entities(entity_id) ON DELETE CASCADE,

    mention_type VARCHAR(50) DEFAULT 'direct', -- 'direct', 'channel', 'here', 'everyone'
    mentioned_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_message_mention UNIQUE (message_id, mentioned_entity_id)
);

CREATE INDEX idx_message_mentions_message ON message_mentions(message_id);
CREATE INDEX idx_message_mentions_entity ON message_mentions(mentioned_entity_id);

-- Message links/URLs
CREATE TABLE message_links (
    link_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES chat_messages(message_id) ON DELETE CASCADE,

    url TEXT NOT NULL,
    url_domain VARCHAR(255),
    url_title VARCHAR(500),
    url_description TEXT,
    preview_image_url TEXT,

    metadata JSONB DEFAULT '{}',
    extracted_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_message_links_message ON message_links(message_id);
CREATE INDEX idx_message_links_domain ON message_links(url_domain);

-- ============================================================================
-- COMPONENT 3: PERSONAL HISTORY TIMELINE
-- ============================================================================

-- Life event categories
CREATE TABLE life_event_categories (
    category_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category_name VARCHAR(100) NOT NULL UNIQUE,
    parent_category_id UUID REFERENCES life_event_categories(category_id) ON DELETE SET NULL,
    icon VARCHAR(50),
    color VARCHAR(7),
    sort_order INTEGER DEFAULT 0
);

CREATE INDEX idx_life_event_categories_parent ON life_event_categories(parent_category_id);

-- Insert common categories
INSERT INTO life_event_categories (category_name, icon, color) VALUES
    ('Career', '💼', '#3B82F6'),
    ('Education', '🎓', '#10B981'),
    ('Relationships', '❤️', '#EF4444'),
    ('Health', '🏥', '#F59E0B'),
    ('Travel', '✈️', '#8B5CF6'),
    ('Achievements', '🏆', '#F59E0B'),
    ('Family', '👨‍👩‍👧‍👦', '#EC4899'),
    ('Residence', '🏠', '#6366F1'),
    ('Financial', '💰', '#059669'),
    ('Hobbies', '🎨', '#14B8A6');

-- Personal life events (extends timeline_events)
CREATE TABLE life_events (
    life_event_id UUID PRIMARY KEY REFERENCES timeline_events(event_id) ON DELETE CASCADE,
    entity_id UUID NOT NULL REFERENCES entities(entity_id) ON DELETE CASCADE,
    category_id UUID REFERENCES life_event_categories(category_id) ON DELETE SET NULL,

    -- Event details
    event_title VARCHAR(500) NOT NULL,
    event_description TEXT,
    event_location VARCHAR(500),

    -- Temporal precision
    date_precision VARCHAR(20) DEFAULT 'day', -- 'year', 'month', 'day', 'hour', 'minute'
    is_approximate BOOLEAN DEFAULT false,

    -- Significance
    significance_level VARCHAR(20) DEFAULT 'medium', -- 'low', 'medium', 'high', 'critical'
    emotional_valence VARCHAR(20), -- 'positive', 'negative', 'neutral', 'mixed'

    -- Privacy
    privacy_level VARCHAR(20) DEFAULT 'private', -- 'private', 'family', 'friends', 'public'

    -- Rich content
    notes TEXT,
    metadata JSONB DEFAULT '{}',

    CONSTRAINT valid_date_precision CHECK (
        date_precision IN ('year', 'month', 'day', 'hour', 'minute')
    ),
    CONSTRAINT valid_significance CHECK (
        significance_level IN ('low', 'medium', 'high', 'critical')
    ),
    CONSTRAINT valid_valence CHECK (
        emotional_valence IN ('positive', 'negative', 'neutral', 'mixed')
    ),
    CONSTRAINT valid_privacy CHECK (
        privacy_level IN ('private', 'family', 'friends', 'public')
    )
);

CREATE INDEX idx_life_events_entity ON life_events(entity_id);
CREATE INDEX idx_life_events_category ON life_events(category_id);
CREATE INDEX idx_life_events_significance ON life_events(significance_level);
CREATE INDEX idx_life_events_privacy ON life_events(privacy_level);

-- Relationships between entities
CREATE TABLE relationships (
    relationship_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_id_1 UUID NOT NULL REFERENCES entities(entity_id) ON DELETE CASCADE,
    entity_id_2 UUID NOT NULL REFERENCES entities(entity_id) ON DELETE CASCADE,

    relationship_type VARCHAR(100) NOT NULL, -- 'family', 'friend', 'colleague', 'romantic', etc.
    relationship_subtype VARCHAR(100), -- 'parent', 'sibling', 'spouse', 'manager', etc.

    -- Temporal
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    is_current BOOLEAN DEFAULT true,

    -- Metadata
    strength NUMERIC(3,2) DEFAULT 0.5, -- Relationship closeness 0.0 to 1.0
    notes TEXT,
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_relationship_entities CHECK (entity_id_1 != entity_id_2),
    CONSTRAINT valid_relationship_dates CHECK (
        ended_at IS NULL OR ended_at >= started_at
    ),
    CONSTRAINT valid_strength CHECK (strength >= 0 AND strength <= 1),
    CONSTRAINT unique_relationship UNIQUE (entity_id_1, entity_id_2, relationship_type)
);

CREATE INDEX idx_relationships_entity1 ON relationships(entity_id_1);
CREATE INDEX idx_relationships_entity2 ON relationships(entity_id_2);
CREATE INDEX idx_relationships_type ON relationships(relationship_type);
CREATE INDEX idx_relationships_current ON relationships(is_current) WHERE is_current = true;

-- Life event participants (people involved in life events)
CREATE TABLE life_event_participants (
    participant_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    life_event_id UUID NOT NULL REFERENCES life_events(life_event_id) ON DELETE CASCADE,
    entity_id UUID NOT NULL REFERENCES entities(entity_id) ON DELETE CASCADE,

    role VARCHAR(100), -- 'participant', 'witness', 'organizer', etc.
    notes TEXT,

    CONSTRAINT unique_event_participant UNIQUE (life_event_id, entity_id)
);

CREATE INDEX idx_life_event_participants_event ON life_event_participants(life_event_id);
CREATE INDEX idx_life_event_participants_entity ON life_event_participants(entity_id);

-- Locations (for travel, residence, events)
CREATE TABLE locations (
    location_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    location_name VARCHAR(500) NOT NULL,
    location_type VARCHAR(50), -- 'residence', 'workplace', 'venue', 'city', etc.

    -- Address components
    street_address VARCHAR(500),
    city VARCHAR(200),
    state_province VARCHAR(200),
    country VARCHAR(100),
    postal_code VARCHAR(20),

    -- Coordinates
    latitude NUMERIC(10, 7),
    longitude NUMERIC(10, 7),

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_coordinates CHECK (
        (latitude IS NULL AND longitude IS NULL) OR
        (latitude BETWEEN -90 AND 90 AND longitude BETWEEN -180 AND 180)
    )
);

CREATE INDEX idx_locations_city ON locations(city);
CREATE INDEX idx_locations_country ON locations(country);
CREATE INDEX idx_locations_type ON locations(location_type);

-- Spatial index for coordinates (requires PostGIS extension)
-- CREATE INDEX idx_locations_coords ON locations USING GIST(
--     ll_to_earth(latitude, longitude)
-- );

-- Life event locations (many-to-many)
CREATE TABLE life_event_locations (
    life_event_id UUID NOT NULL REFERENCES life_events(life_event_id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(location_id) ON DELETE CASCADE,

    location_role VARCHAR(50) DEFAULT 'primary', -- 'primary', 'secondary', 'origin', 'destination'

    PRIMARY KEY (life_event_id, location_id)
);

CREATE INDEX idx_life_event_locations_location ON life_event_locations(location_id);

-- ============================================================================
-- CROSS-COMPONENT LINKING TABLES
-- ============================================================================

-- Link chat messages to life events (e.g., planning a wedding)
CREATE TABLE chat_message_life_events (
    message_id UUID NOT NULL REFERENCES chat_messages(message_id) ON DELETE CASCADE,
    life_event_id UUID NOT NULL REFERENCES life_events(life_event_id) ON DELETE CASCADE,

    link_type VARCHAR(50) DEFAULT 'related', -- 'planning', 'discussing', 'documenting', 'related'
    linked_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (message_id, life_event_id)
);

CREATE INDEX idx_chat_msg_life_events_life ON chat_message_life_events(life_event_id);

-- Entity timeline view linking (what entities are associated with timeline events)
CREATE TABLE entity_timeline_associations (
    association_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_id UUID NOT NULL REFERENCES entities(entity_id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES timeline_events(event_id) ON DELETE CASCADE,

    association_type VARCHAR(100), -- 'creator', 'participant', 'mentioned', 'related'
    association_strength NUMERIC(3,2) DEFAULT 1.0,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_association_strength CHECK (
        association_strength >= 0 AND association_strength <= 1
    ),
    CONSTRAINT unique_entity_event_type UNIQUE (entity_id, event_id, association_type)
);

CREATE INDEX idx_entity_timeline_entity ON entity_timeline_associations(entity_id);
CREATE INDEX idx_entity_timeline_event ON entity_timeline_associations(event_id);
CREATE INDEX idx_entity_timeline_type ON entity_timeline_associations(association_type);

-- ============================================================================
-- AUDIT AND SYNC TRACKING
-- ============================================================================

-- Sync logs for tracking ingestion runs
CREATE TABLE sync_logs (
    sync_log_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_id UUID NOT NULL REFERENCES sources(source_id) ON DELETE CASCADE,

    sync_started_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    sync_completed_at TIMESTAMPTZ,
    sync_status VARCHAR(50) NOT NULL, -- 'running', 'success', 'failed', 'partial'

    records_processed INTEGER DEFAULT 0,
    records_created INTEGER DEFAULT 0,
    records_updated INTEGER DEFAULT 0,
    records_failed INTEGER DEFAULT 0,

    error_message TEXT,
    error_details JSONB,

    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_sync_logs_source ON sync_logs(source_id, sync_started_at DESC);
CREATE INDEX idx_sync_logs_status ON sync_logs(sync_status, sync_started_at DESC);

-- Audit trail for important changes
CREATE TABLE audit_trail (
    audit_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    table_name VARCHAR(100) NOT NULL,
    record_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL, -- 'INSERT', 'UPDATE', 'DELETE'

    changed_by UUID REFERENCES entities(entity_id) ON DELETE SET NULL,
    changed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    old_values JSONB,
    new_values JSONB,

    change_reason TEXT,

    CONSTRAINT valid_action CHECK (action IN ('INSERT', 'UPDATE', 'DELETE'))
);

CREATE INDEX idx_audit_trail_table ON audit_trail(table_name, changed_at DESC);
CREATE INDEX idx_audit_trail_record ON audit_trail(record_id, changed_at DESC);
CREATE INDEX idx_audit_trail_changed_by ON audit_trail(changed_by);

-- ============================================================================
-- MATERIALIZED VIEWS FOR PERFORMANCE
-- ============================================================================

-- Daily event summary
CREATE MATERIALIZED VIEW daily_event_summary AS
SELECT
    DATE(event_timestamp) as event_date,
    source_id,
    event_type,
    COUNT(*) as event_count,
    AVG(importance_score) as avg_importance
FROM timeline_events
WHERE is_deleted = false
GROUP BY DATE(event_timestamp), source_id, event_type;

CREATE UNIQUE INDEX idx_daily_event_summary_unique
    ON daily_event_summary(event_date, source_id, event_type);

-- Entity activity summary
CREATE MATERIALIZED VIEW entity_activity_summary AS
SELECT
    e.entity_id,
    e.display_name,
    COUNT(DISTINCT cm.message_id) as message_count,
    COUNT(DISTINCT le.life_event_id) as life_event_count,
    MAX(te.event_timestamp) as last_activity_at
FROM entities e
LEFT JOIN chat_messages cm ON e.entity_id = cm.sender_id
LEFT JOIN life_events le ON e.entity_id = le.entity_id
LEFT JOIN timeline_events te ON (
    te.event_id = cm.message_id OR te.event_id = le.life_event_id
)
GROUP BY e.entity_id, e.display_name;

CREATE UNIQUE INDEX idx_entity_activity_summary_unique
    ON entity_activity_summary(entity_id);

-- ============================================================================
-- TRIGGERS FOR MAINTAINING DATA INTEGRITY
-- ============================================================================

-- Auto-update updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_sources_updated_at BEFORE UPDATE ON sources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_entities_updated_at BEFORE UPDATE ON entities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_timeline_events_updated_at BEFORE UPDATE ON timeline_events
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_relationships_updated_at BEFORE UPDATE ON relationships
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Auto-update full-text search vectors
CREATE OR REPLACE FUNCTION update_timeline_event_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.content_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER timeline_event_vector_update BEFORE INSERT OR UPDATE ON timeline_events
    FOR EACH ROW EXECUTE FUNCTION update_timeline_event_vector();

-- Auto-update chat message search vector
CREATE OR REPLACE FUNCTION update_chat_message_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.message_vector := to_tsvector('english', COALESCE(NEW.message_text, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER chat_message_vector_update BEFORE INSERT OR UPDATE ON chat_messages
    FOR EACH ROW EXECUTE FUNCTION update_chat_message_vector();

-- Update tag usage count
CREATE OR REPLACE FUNCTION update_tag_usage_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE tags SET usage_count = usage_count + 1 WHERE tag_id = NEW.tag_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE tags SET usage_count = usage_count - 1 WHERE tag_id = OLD.tag_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_tag_usage_on_event_tag AFTER INSERT OR DELETE ON event_tags
    FOR EACH ROW EXECUTE FUNCTION update_tag_usage_count();

-- Update chat channel participant count
CREATE OR REPLACE FUNCTION update_channel_participant_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE chat_channels
        SET participant_count = participant_count + 1
        WHERE channel_id = NEW.channel_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE chat_channels
        SET participant_count = participant_count - 1
        WHERE channel_id = OLD.channel_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_participant_count AFTER INSERT OR DELETE ON chat_participants
    FOR EACH ROW EXECUTE FUNCTION update_channel_participant_count();

-- Update message reaction count
CREATE OR REPLACE FUNCTION update_message_reaction_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE chat_messages
        SET reaction_count = reaction_count + 1
        WHERE message_id = NEW.message_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE chat_messages
        SET reaction_count = reaction_count - 1
        WHERE message_id = OLD.message_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_reaction_count AFTER INSERT OR DELETE ON message_reactions
    FOR EACH ROW EXECUTE FUNCTION update_message_reaction_count();

-- Update message reply count
CREATE OR REPLACE FUNCTION update_message_reply_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' AND NEW.parent_message_id IS NOT NULL THEN
        UPDATE chat_messages
        SET reply_count = reply_count + 1
        WHERE message_id = NEW.parent_message_id;
    ELSIF TG_OP = 'DELETE' AND OLD.parent_message_id IS NOT NULL THEN
        UPDATE chat_messages
        SET reply_count = reply_count - 1
        WHERE message_id = OLD.parent_message_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_reply_count AFTER INSERT OR DELETE ON chat_messages
    FOR EACH ROW EXECUTE FUNCTION update_message_reply_count();

-- Soft delete handler for timeline events
CREATE OR REPLACE FUNCTION handle_timeline_event_delete()
RETURNS TRIGGER AS $$
BEGIN
    NEW.is_deleted = true;
    NEW.deleted_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Note: This would be used in application logic, not as a trigger
-- to prevent accidental hard deletes

-- ============================================================================
-- UTILITY FUNCTIONS
-- ============================================================================

-- Function to search timeline events with full-text search
CREATE OR REPLACE FUNCTION search_timeline_events(
    search_query TEXT,
    start_date TIMESTAMPTZ DEFAULT NULL,
    end_date TIMESTAMPTZ DEFAULT NULL,
    event_types TEXT[] DEFAULT NULL,
    limit_count INTEGER DEFAULT 100
)
RETURNS TABLE (
    event_id UUID,
    title VARCHAR(500),
    description TEXT,
    event_timestamp TIMESTAMPTZ,
    event_type VARCHAR(100),
    relevance REAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        te.event_id,
        te.title,
        te.description,
        te.event_timestamp,
        te.event_type,
        ts_rank(te.content_vector, websearch_to_tsquery('english', search_query)) as relevance
    FROM timeline_events te
    WHERE
        te.is_deleted = false
        AND te.content_vector @@ websearch_to_tsquery('english', search_query)
        AND (start_date IS NULL OR te.event_timestamp >= start_date)
        AND (end_date IS NULL OR te.event_timestamp <= end_date)
        AND (event_types IS NULL OR te.event_type = ANY(event_types))
    ORDER BY relevance DESC, te.event_timestamp DESC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get entity timeline
CREATE OR REPLACE FUNCTION get_entity_timeline(
    p_entity_id UUID,
    start_date TIMESTAMPTZ DEFAULT NULL,
    end_date TIMESTAMPTZ DEFAULT NULL,
    limit_count INTEGER DEFAULT 100
)
RETURNS TABLE (
    event_id UUID,
    event_type VARCHAR(100),
    title VARCHAR(500),
    event_timestamp TIMESTAMPTZ,
    source_name VARCHAR(255)
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        te.event_id,
        te.event_type,
        te.title,
        te.event_timestamp,
        s.source_name
    FROM timeline_events te
    JOIN sources s ON te.source_id = s.source_id
    WHERE
        te.is_deleted = false
        AND (te.entity_id = p_entity_id OR
             EXISTS (
                 SELECT 1 FROM entity_timeline_associations eta
                 WHERE eta.event_id = te.event_id AND eta.entity_id = p_entity_id
             ))
        AND (start_date IS NULL OR te.event_timestamp >= start_date)
        AND (end_date IS NULL OR te.event_timestamp <= end_date)
    ORDER BY te.event_timestamp DESC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get relationship network
CREATE OR REPLACE FUNCTION get_relationship_network(
    p_entity_id UUID,
    max_depth INTEGER DEFAULT 2
)
RETURNS TABLE (
    entity_id UUID,
    entity_name VARCHAR(255),
    relationship_path TEXT[],
    depth INTEGER
) AS $$
BEGIN
    RETURN QUERY
    WITH RECURSIVE relationship_tree AS (
        -- Base case: direct relationships
        SELECT
            CASE
                WHEN r.entity_id_1 = p_entity_id THEN r.entity_id_2
                ELSE r.entity_id_1
            END as entity_id,
            ARRAY[r.relationship_type] as relationship_path,
            1 as depth
        FROM relationships r
        WHERE (r.entity_id_1 = p_entity_id OR r.entity_id_2 = p_entity_id)
            AND r.is_current = true

        UNION ALL

        -- Recursive case: relationships of relationships
        SELECT
            CASE
                WHEN r.entity_id_1 = rt.entity_id THEN r.entity_id_2
                ELSE r.entity_id_1
            END as entity_id,
            rt.relationship_path || r.relationship_type,
            rt.depth + 1
        FROM relationship_tree rt
        JOIN relationships r ON (r.entity_id_1 = rt.entity_id OR r.entity_id_2 = rt.entity_id)
        WHERE rt.depth < max_depth
            AND r.is_current = true
            AND NOT (CASE WHEN r.entity_id_1 = rt.entity_id THEN r.entity_id_2 ELSE r.entity_id_1 END = p_entity_id)
    )
    SELECT DISTINCT
        rt.entity_id,
        e.display_name,
        rt.relationship_path,
        rt.depth
    FROM relationship_tree rt
    JOIN entities e ON rt.entity_id = e.entity_id
    ORDER BY rt.depth, e.display_name;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- PERFORMANCE OPTIMIZATION: VACUUM AND ANALYZE POLICIES
-- ============================================================================

-- These should be run periodically via cron or pg_cron extension

-- Example: Schedule daily vacuum analyze for large tables
-- SELECT cron.schedule('daily-vacuum-timeline', '0 2 * * *',
--     'VACUUM ANALYZE timeline_events, chat_messages, life_events;');

-- Refresh materialized views daily
-- SELECT cron.schedule('refresh-daily-summary', '0 3 * * *',
--     'REFRESH MATERIALIZED VIEW CONCURRENTLY daily_event_summary;');

-- SELECT cron.schedule('refresh-entity-summary', '0 3 * * *',
--     'REFRESH MATERIALIZED VIEW CONCURRENTLY entity_activity_summary;');

-- ============================================================================
-- COMMENTS FOR DOCUMENTATION
-- ============================================================================

COMMENT ON TABLE sources IS 'External systems that provide timeline data';
COMMENT ON TABLE entities IS 'Users, organizations, and services that interact with the system';
COMMENT ON TABLE timeline_events IS 'Core timeline event storage for all event types';
COMMENT ON TABLE chat_messages IS 'Chat messages extending timeline_events with chat-specific data';
COMMENT ON TABLE life_events IS 'Personal life events extending timeline_events';
COMMENT ON TABLE relationships IS 'Relationships between entities (people, organizations)';

COMMENT ON COLUMN timeline_events.content_vector IS 'Full-text search vector for title and description';
COMMENT ON COLUMN timeline_events.importance_score IS 'Calculated importance from 0.0 (low) to 1.0 (high)';
COMMENT ON COLUMN timeline_events.raw_data IS 'Original unprocessed data from source system';
COMMENT ON COLUMN timeline_events.processed_data IS 'Enriched and extracted structured data';

COMMENT ON COLUMN chat_messages.message_vector IS 'Full-text search vector for message content';
COMMENT ON COLUMN life_events.date_precision IS 'Granularity of the event date (year/month/day/hour/minute)';

-- ============================================================================
-- SAMPLE QUERIES FOR COMMON USE CASES
-- ============================================================================

/*
-- 1. Search all timeline events for a keyword
SELECT * FROM search_timeline_events('project planning', '2024-01-01'::timestamptz, '2024-12-31'::timestamptz);

-- 2. Get all chat messages in a channel with participants
SELECT
    cm.message_id,
    e.display_name as sender,
    cm.message_text,
    te.event_timestamp
FROM chat_messages cm
JOIN entities e ON cm.sender_id = e.entity_id
JOIN timeline_events te ON cm.message_id = te.event_id
WHERE cm.channel_id = 'some-uuid'
    AND te.is_deleted = false
ORDER BY te.event_timestamp DESC;

-- 3. Get life events for a person within a date range
SELECT
    le.event_title,
    le.event_description,
    lec.category_name,
    te.event_timestamp,
    le.significance_level
FROM life_events le
JOIN timeline_events te ON le.life_event_id = te.event_id
LEFT JOIN life_event_categories lec ON le.category_id = lec.category_id
WHERE le.entity_id = 'some-uuid'
    AND te.event_timestamp BETWEEN '2020-01-01' AND '2024-12-31'
ORDER BY te.event_timestamp DESC;

-- 4. Get all relationships for an entity
SELECT
    e.display_name,
    r.relationship_type,
    r.relationship_subtype,
    r.started_at,
    r.is_current
FROM relationships r
JOIN entities e ON (r.entity_id_2 = e.entity_id)
WHERE r.entity_id_1 = 'some-uuid'
    AND r.is_current = true;

-- 5. Get conversation thread with all replies
WITH RECURSIVE thread AS (
    SELECT
        cm.message_id,
        cm.message_text,
        cm.parent_message_id,
        te.event_timestamp,
        e.display_name,
        0 as depth
    FROM chat_messages cm
    JOIN timeline_events te ON cm.message_id = te.event_id
    JOIN entities e ON cm.sender_id = e.entity_id
    WHERE cm.message_id = 'root-message-uuid'

    UNION ALL

    SELECT
        cm.message_id,
        cm.message_text,
        cm.parent_message_id,
        te.event_timestamp,
        e.display_name,
        t.depth + 1
    FROM chat_messages cm
    JOIN timeline_events te ON cm.message_id = te.event_id
    JOIN entities e ON cm.sender_id = e.entity_id
    JOIN thread t ON cm.parent_message_id = t.message_id
)
SELECT * FROM thread ORDER BY event_timestamp;

-- 6. Find most active chat participants
SELECT
    e.display_name,
    COUNT(cm.message_id) as message_count,
    COUNT(DISTINCT DATE(te.event_timestamp)) as active_days
FROM chat_messages cm
JOIN entities e ON cm.sender_id = e.entity_id
JOIN timeline_events te ON cm.message_id = te.event_id
WHERE cm.channel_id = 'some-uuid'
    AND te.event_timestamp > NOW() - INTERVAL '30 days'
GROUP BY e.entity_id, e.display_name
ORDER BY message_count DESC
LIMIT 10;

-- 7. Get timeline heatmap by hour and day of week
SELECT
    EXTRACT(DOW FROM event_timestamp) as day_of_week,
    EXTRACT(HOUR FROM event_timestamp) as hour_of_day,
    COUNT(*) as event_count
FROM timeline_events
WHERE event_timestamp > NOW() - INTERVAL '90 days'
    AND is_deleted = false
GROUP BY day_of_week, hour_of_day
ORDER BY day_of_week, hour_of_day;

-- 8. Find related life events by participants
SELECT
    le1.event_title as event1,
    le2.event_title as event2,
    COUNT(DISTINCT lep.entity_id) as shared_participants
FROM life_events le1
JOIN life_event_participants lep1 ON le1.life_event_id = lep1.life_event_id
JOIN life_event_participants lep2 ON lep1.entity_id = lep2.entity_id
JOIN life_events le2 ON lep2.life_event_id = le2.life_event_id
WHERE le1.life_event_id < le2.life_event_id
GROUP BY le1.life_event_id, le2.life_event_id, le1.event_title, le2.event_title
HAVING COUNT(DISTINCT lep1.entity_id) >= 2
ORDER BY shared_participants DESC;
*/

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================
