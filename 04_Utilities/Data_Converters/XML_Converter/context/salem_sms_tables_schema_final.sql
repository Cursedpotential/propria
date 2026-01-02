-- =====================================================
-- SALEM SMS/MESSAGING TABLES - FINAL SCHEMA
-- Consolidated from all iterations for court admissibility
-- Forensic-grade chain of custody and coercive control detection
-- =====================================================
--
-- Date: December 27, 2025
-- Database: Supabase PostgreSQL  
-- Case: Salem v. Kinzel (2025-53985-DC)
-- Purpose: SMS, MMS, and multi-platform message storage with behavioral analysis
--
-- =====================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For text search
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";  -- For query analysis

-- =====================================================
-- CORE MESSAGING TABLES
-- =====================================================

-- Source documents (XML exports, JSON files, PDFs, screenshots)
CREATE TABLE messaging_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- File identification
    filename VARCHAR(500) NOT NULL,
    file_type VARCHAR(50) NOT NULL,  -- 'sms_xml', 'mms_xml', 'facebook_json', 'screenshots', 'pdf', etc.
    file_hash VARCHAR(64) NOT NULL UNIQUE,  -- SHA-256 for chain of custody & deduplication
    file_size BIGINT NOT NULL,
    
    -- Storage
    storage_path TEXT NOT NULL,  -- R2/GCS/S3 path
    storage_provider VARCHAR(50),  -- 'cloudflare_r2', 'google_drive', 'local'
    
    -- Processing status
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'verified')),
    processed_at TIMESTAMPTZ,
    processing_duration_ms INTEGER,
    
    -- Extracted summary
    raw_text TEXT,  -- Full text extraction for search
    message_count INTEGER DEFAULT 0,
    page_count INTEGER,  -- For PDFs/screenshots
    
    -- Source metadata
    source_device VARCHAR(100),  -- Phone model, computer, etc.
    source_platform VARCHAR(50),  -- 'android', 'ios', 'facebook', 'snapchat', etc.
    export_date TIMESTAMPTZ,  -- When export was created
    date_range_start TIMESTAMPTZ,  -- Earliest message date in file
    date_range_end TIMESTAMPTZ,  -- Latest message date in file
    
    -- Chain of custody
    acquired_by VARCHAR(100),  -- Who obtained this evidence
    acquired_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acquisition_method TEXT,  -- How it was obtained
    verified_by VARCHAR(100),  -- Who verified authenticity
    verified_date TIMESTAMPTZ,
    
    -- Additional metadata
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Conversations (groups messages together)
CREATE TABLE messaging_conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID REFERENCES messaging_documents(id) ON DELETE SET NULL,
    
    -- Platform identification
    platform VARCHAR(50) NOT NULL,  -- 'sms', 'mms', 'facebook', 'snapchat', 'instagram', 'whatsapp', etc.
    platform_id VARCHAR(255),  -- External conversation/thread ID
    platform_name VARCHAR(200),  -- Display name from platform
    
    -- Participants
    participants TEXT[] NOT NULL,  -- Array of participant identifiers
    participant_count INTEGER NOT NULL,
    primary_participant VARCHAR(255),  -- The "other" person in 1-on-1 conversations
    
    -- Temporal info
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    last_message_at TIMESTAMPTZ,
    message_count INTEGER DEFAULT 0,
    
    -- Thread metadata
    is_group BOOLEAN DEFAULT FALSE,
    group_name VARCHAR(500),
    
    -- Analysis summary (updated by triggers)
    behavior_summary JSONB DEFAULT '{}'::JSONB,  -- Aggregated behavior counts
    total_words INTEGER DEFAULT 0,
    avg_message_length INTEGER,
    
    -- Court relevance
    is_evidence BOOLEAN DEFAULT FALSE,
    exhibit_number VARCHAR(50),
    relevance_score DECIMAL(3,2),  -- 0.00 to 1.00
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(platform, platform_id)
);

-- Individual messages (the core forensic record)
CREATE TABLE messaging_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES messaging_conversations(id) ON DELETE CASCADE,
    document_id UUID REFERENCES messaging_documents(id) ON DELETE SET NULL,
    
    -- Message identification
    external_id VARCHAR(200),  -- Original message ID from platform
    serial_number INTEGER,  -- Sequential number within conversation
    
    -- Temporal data
    timestamp TIMESTAMPTZ NOT NULL,
    timestamp_precision VARCHAR(20) DEFAULT 'exact' CHECK (timestamp_precision IN ('exact', 'minute', 'hour', 'day', 'approximate')),
    timezone VARCHAR(50),  -- Original timezone if known
    date_us TEXT,  -- US format for readability (MM/DD/YYYY)
    time_12h TEXT,  -- 12-hour format for readability
    
    -- Sender/Recipient
    sender VARCHAR(255) NOT NULL,
    sender_normalized VARCHAR(255),  -- Standardized (E.164 for phones)
    sender_name VARCHAR(255),  -- Display name if available
    recipient VARCHAR(255),
    recipient_normalized VARCHAR(255),
    recipient_name VARCHAR(255),
    
    -- Content
    content TEXT,
    content_lower TEXT GENERATED ALWAYS AS (LOWER(content)) STORED,  -- For search
    content_hash VARCHAR(64),  -- SHA-256 of content for deduplication
    word_count INTEGER,
    char_count INTEGER,
    
    -- Message metadata
    direction VARCHAR(10) CHECK (direction IN ('inbound', 'outbound', 'unknown')),
    message_type VARCHAR(20) DEFAULT 'text',  -- 'text', 'mms', 'voice', 'image', 'video', 'file'
    status VARCHAR(20),  -- 'sent', 'delivered', 'read', 'failed'
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    
    -- Attachments
    has_attachments BOOLEAN DEFAULT FALSE,
    attachment_count INTEGER DEFAULT 0,
    attachment_types TEXT[],  -- Array of MIME types
    attachment_urls TEXT[],  -- Storage paths
    
    -- Analysis flags (updated by triggers)
    has_behaviors BOOLEAN DEFAULT FALSE,
    behavior_count INTEGER DEFAULT 0,
    max_severity VARCHAR(20),  -- 'low', 'medium', 'high', 'critical'
    behavior_categories TEXT[],  -- Array of detected categories
    
    -- Entity extraction
    has_entities BOOLEAN DEFAULT FALSE,
    entity_count INTEGER DEFAULT 0,
    mentioned_people TEXT[],  -- Array of person names/numbers
    mentioned_locations TEXT[],  -- Array of places
    
    -- Forensic preservation
    raw_data JSONB,  -- Complete original message data
    extraction_method VARCHAR(50),  -- How message was extracted
    extraction_timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    -- Court relevance
    is_evidence BOOLEAN DEFAULT FALSE,
    evidence_notes TEXT,
    mcl_factors VARCHAR(2)[],  -- Array of MCL 722.23 factors (A-L)
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(conversation_id, external_id)
);

-- =====================================================
-- ENTITY EXTRACTION TABLES
-- =====================================================

-- Entities (people, places, organizations)
CREATE TABLE messaging_entities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    entity_type VARCHAR(50) NOT NULL CHECK (entity_type IN ('person', 'location', 'organization', 'phone', 'email', 'date', 'url')),
    name VARCHAR(255) NOT NULL,
    normalized_name VARCHAR(255),  -- Standardized version
    
    -- For people
    aliases TEXT[],
    phone_numbers TEXT[],
    email_addresses TEXT[],
    social_handles JSONB DEFAULT '{}'::JSONB,
    
    -- For locations
    address TEXT,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    
    -- Classification
    role VARCHAR(50),  -- 'party', 'witness', 'child', 'attorney', 'third_party', etc.
    is_party BOOLEAN DEFAULT FALSE,
    is_witness BOOLEAN DEFAULT FALSE,
    
    -- Frequency tracking
    mention_count INTEGER DEFAULT 0,
    first_mentioned_at TIMESTAMPTZ,
    last_mentioned_at TIMESTAMPTZ,
    
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(entity_type, normalized_name)
);

-- Entity mentions in messages
CREATE TABLE messaging_entity_mentions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messaging_messages(id) ON DELETE CASCADE,
    entity_id UUID NOT NULL REFERENCES messaging_entities(id) ON DELETE CASCADE,
    
    -- Position in text
    mention_text TEXT NOT NULL,
    start_char INTEGER,
    end_char INTEGER,
    
    -- Context
    surrounding_text TEXT,  -- 50 chars before/after for context
    
    -- Extraction info
    confidence DECIMAL(3,2),  -- 0.00 to 1.00
    extraction_method VARCHAR(50),  -- 'regex', 'nlp', 'manual', 'llm'
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(message_id, entity_id, start_char)
);

-- =====================================================
-- BEHAVIORAL ANALYSIS TABLES
-- =====================================================

-- Behavior categories reference
CREATE TABLE messaging_behavior_categories (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    severity_default VARCHAR(20) DEFAULT 'medium' CHECK (severity_default IN ('low', 'medium', 'high', 'critical')),
    mcl_factors VARCHAR(2)[],  -- Relevant MCL 722.23 factors
    pattern_indicators TEXT[],  -- Common text patterns
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert behavior categories
INSERT INTO messaging_behavior_categories (id, name, mcl_factors, pattern_indicators) VALUES
('gaslighting', 'Gaslighting', ARRAY['F', 'G', 'K'], ARRAY['never happened', 'you''re crazy', 'making things up', 'imagining things']),
('blame_shifting', 'Blame Shifting', ARRAY['F', 'J'], ARRAY['your fault', 'you made me', 'because of you', 'you caused']),
('minimizing', 'Minimizing', ARRAY['F', 'K'], ARRAY['not a big deal', 'overreacting', 'too sensitive', 'dramatic']),
('love_bombing', 'Love Bombing', ARRAY['F'], ARRAY['i love you so much', 'can''t live without', 'soulmate', 'meant to be']),
('stonewalling', 'Stonewalling', ARRAY['J'], ARRAY['not talking about', 'done discussing', 'conversation over', 'nothing to say']),
('parental_alienation', 'Parental Alienation', ARRAY['J', 'K'], ARRAY['doesn''t care about you', 'never wanted', 'bad parent', 'doesn''t love']),
('coercive_control', 'Coercive Control', ARRAY['F', 'K'], ARRAY['have to', 'must', 'better', 'or else', 'consequences']),
('financial_abuse', 'Financial Abuse/Control', ARRAY['C', 'F', 'K'], ARRAY['your money', 'can''t afford', 'pay for', 'broke']),
('substance_weaponization', 'Substance Use Weaponization', ARRAY['F', 'G'], ARRAY['drunk again', 'high', 'addict', 'rehab', 'sober']),
('reactive_abuse', 'Reactive Abuse Provocation', ARRAY['F', 'K'], ARRAY['calm down', 'angry', 'out of control', 'losing it']),
('darvo', 'DARVO Pattern', ARRAY['F', 'K'], ARRAY['attacking me', 'i''m the victim', 'you''re abusive', 'hurting me']),
('character_assassination', 'Character Assassination', ARRAY['F', 'J'], ARRAY['crazy', 'unstable', 'unfit', 'dangerous', 'bad']);

-- Detected behaviors in messages
CREATE TABLE messaging_behaviors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messaging_messages(id) ON DELETE CASCADE,
    
    -- Classification
    category VARCHAR(50) NOT NULL REFERENCES messaging_behavior_categories(id),
    subcategory VARCHAR(100),
    
    -- Evidence
    matched_pattern TEXT,  -- The pattern that triggered detection
    matched_text TEXT,  -- The actual text from message
    start_char INTEGER,
    end_char INTEGER,
    context_before TEXT,  -- Text before match
    context_after TEXT,  -- Text after match
    
    -- Scoring
    confidence DECIMAL(3,2) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),  -- 0.00 to 1.00
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    
    -- Detection
    detection_method VARCHAR(50) NOT NULL,  -- 'regex', 'llm', 'manual', 'nlp'
    rule_name VARCHAR(100),
    model_version VARCHAR(50),  -- For LLM detections
    
    -- Verification
    is_verified BOOLEAN DEFAULT FALSE,
    verified_by VARCHAR(100),
    verified_at TIMESTAMPTZ,
    verification_notes TEXT,
    is_false_positive BOOLEAN DEFAULT FALSE,
    
    -- Related evidence
    related_message_ids UUID[],  -- Array of related message IDs showing pattern
    pattern_frequency INTEGER,  -- How many times this pattern appears
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- EVIDENCE & LEGAL TABLES
-- =====================================================

-- Evidence items (court-ready extracts)
CREATE TABLE messaging_evidence_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Source
    message_id UUID REFERENCES messaging_messages(id) ON DELETE SET NULL,
    conversation_id UUID REFERENCES messaging_conversations(id) ON DELETE SET NULL,
    document_id UUID REFERENCES messaging_documents(id) ON DELETE SET NULL,
    
    -- Evidence content
    title VARCHAR(500) NOT NULL,
    description TEXT,
    quote TEXT NOT NULL,  -- The relevant quote
    context TEXT,  -- Surrounding context for understanding
    
    -- Classification
    evidence_type VARCHAR(50) NOT NULL,  -- 'communication', 'pattern', 'admission', 'threat', etc.
    category VARCHAR(100),
    
    -- Dates
    evidence_date TIMESTAMPTZ NOT NULL,
    date_precision VARCHAR(20) DEFAULT 'exact',
    
    -- Legal relevance
    relevance_score DECIMAL(3,2) CHECK (relevance_score >= 0 AND relevance_score <= 1),  -- 0.00 to 1.00
    mcl_factors VARCHAR(2)[],  -- MCL 722.23 factors supported
    
    -- Exhibit info
    exhibit_number VARCHAR(50),
    exhibit_letter VARCHAR(10),
    is_exhibit BOOLEAN DEFAULT FALSE,
    exhibit_date TIMESTAMPTZ,
    
    -- Verification
    is_authenticated BOOLEAN DEFAULT FALSE,
    authentication_method TEXT,
    chain_of_custody TEXT,
    
    -- Related items
    related_evidence_ids UUID[],
    supporting_behavior_ids UUID[],
    
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- MCL 722.23 Best Interest Factors Reference
CREATE TABLE mcl_factors (
    id VARCHAR(2) PRIMARY KEY,  -- A through L
    name VARCHAR(200) NOT NULL,
    description TEXT,
    statutory_text TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert MCL 722.23 factors
INSERT INTO mcl_factors (id, name, description) VALUES
('A', 'Love and Affection', 'The love, affection, and other emotional ties existing between the parties involved and the child'),
('B', 'Capacity to Provide', 'The capacity and disposition of the parties involved to give the child love, affection, and guidance and to continue the education and raising of the child in his or her religion or creed'),
('C', 'Capacity for Necessities', 'The capacity and disposition of the parties involved to provide the child with food, clothing, medical care or other remedial care'),
('D', 'Home Environment', 'The length of time the child has lived in a stable, satisfactory environment, and the desirability of maintaining continuity'),
('E', 'Permanence of Family Unit', 'The permanence, as a family unit, of the existing or proposed custodial home'),
('F', 'Moral Fitness', 'The moral fitness of the parties involved'),
('G', 'Mental and Physical Health', 'The mental and physical health of the parties involved'),
('H', 'Home, School, Community Record', 'The home, school, and community record of the child'),
('I', 'Child Preference', 'The reasonable preference of the child, if the court considers the child to be of sufficient age'),
('J', 'Willingness to Facilitate Relationship', 'The willingness and ability of each of the parties to facilitate and encourage a close and continuing parent-child relationship between the child and the other parent'),
('K', 'Domestic Violence', 'Domestic violence, regardless of whether the violence was directed against or witnessed by the child'),
('L', 'Other Factors', 'Any other factor considered by the court to be relevant to a particular child custody dispute');

-- Link evidence to MCL factors
CREATE TABLE messaging_factor_citations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    evidence_id UUID NOT NULL REFERENCES messaging_evidence_items(id) ON DELETE CASCADE,
    factor_id VARCHAR(2) NOT NULL REFERENCES mcl_factors(id) ON DELETE CASCADE,
    
    -- How it supports/contradicts
    supports_factor BOOLEAN DEFAULT TRUE,  -- TRUE = supports, FALSE = contradicts
    strength VARCHAR(20) CHECK (strength IN ('weak', 'moderate', 'strong', 'decisive')),
    explanation TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(evidence_id, factor_id)
);

-- =====================================================
-- TIMELINE INTEGRATION
-- =====================================================

-- Timeline events (connects to TraceIQ geospatial data)
CREATE TABLE messaging_timeline_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Timing
    event_date TIMESTAMPTZ NOT NULL,
    event_date_end TIMESTAMPTZ,
    date_precision VARCHAR(20) DEFAULT 'exact',
    
    -- Event info
    title VARCHAR(500) NOT NULL,
    description TEXT,
    event_type VARCHAR(50) NOT NULL,  -- 'message_sent', 'conversation_started', 'pattern_detected', etc.
    
    -- Sources
    message_ids UUID[],
    conversation_ids UUID[],
    evidence_ids UUID[],
    behavior_ids UUID[],
    
    -- Location (if available from TraceIQ correlation)
    location TEXT,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    location_confidence VARCHAR(20),  -- 'exact', 'approximate', 'inferred'
    
    -- Participants
    participants TEXT[],
    
    -- Legal relevance
    mcl_factors VARCHAR(2)[],
    is_significant BOOLEAN DEFAULT FALSE,
    significance_notes TEXT,
    
    -- Flags
    is_verified BOOLEAN DEFAULT FALSE,
    is_disputed BOOLEAN DEFAULT FALSE,
    dispute_notes TEXT,
    
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- INDEXES FOR PERFORMANCE
-- =====================================================

-- Document indexes
CREATE INDEX idx_messaging_documents_hash ON messaging_documents(file_hash);
CREATE INDEX idx_messaging_documents_status ON messaging_documents(status);
CREATE INDEX idx_messaging_documents_platform ON messaging_documents(source_platform);

-- Conversation indexes
CREATE INDEX idx_messaging_conversations_platform ON messaging_conversations(platform);
CREATE INDEX idx_messaging_conversations_participant ON messaging_conversations(primary_participant);
CREATE INDEX idx_messaging_conversations_dates ON messaging_conversations(started_at, ended_at);
CREATE INDEX idx_messaging_conversations_evidence ON messaging_conversations(is_evidence) WHERE is_evidence = TRUE;

-- Message indexes (most critical for performance)
CREATE INDEX idx_messaging_messages_conversation ON messaging_messages(conversation_id);
CREATE INDEX idx_messaging_messages_timestamp ON messaging_messages(timestamp DESC);
CREATE INDEX idx_messaging_messages_sender ON messaging_messages(sender_normalized);
CREATE INDEX idx_messaging_messages_recipient ON messaging_messages(recipient_normalized);
CREATE INDEX idx_messaging_messages_content_search ON messaging_messages USING gin(content_lower gin_trgm_ops);
CREATE INDEX idx_messaging_messages_has_behaviors ON messaging_messages(has_behaviors) WHERE has_behaviors = TRUE;
CREATE INDEX idx_messaging_messages_is_evidence ON messaging_messages(is_evidence) WHERE is_evidence = TRUE;
CREATE INDEX idx_messaging_messages_direction ON messaging_messages(direction);
CREATE INDEX idx_messaging_messages_date_us ON messaging_messages(date_us);

-- Composite indexes for common queries
CREATE INDEX idx_messaging_messages_conv_timestamp ON messaging_messages(conversation_id, timestamp DESC);
CREATE INDEX idx_messaging_messages_sender_timestamp ON messaging_messages(sender_normalized, timestamp DESC);

-- Entity indexes
CREATE INDEX idx_messaging_entities_type ON messaging_entities(entity_type);
CREATE INDEX idx_messaging_entities_name ON messaging_entities(normalized_name);
CREATE INDEX idx_messaging_entities_party ON messaging_entities(is_party) WHERE is_party = TRUE;

-- Entity mention indexes
CREATE INDEX idx_messaging_entity_mentions_message ON messaging_entity_mentions(message_id);
CREATE INDEX idx_messaging_entity_mentions_entity ON messaging_entity_mentions(entity_id);

-- Behavior indexes
CREATE INDEX idx_messaging_behaviors_message ON messaging_behaviors(message_id);
CREATE INDEX idx_messaging_behaviors_category ON messaging_behaviors(category);
CREATE INDEX idx_messaging_behaviors_severity ON messaging_behaviors(severity);
CREATE INDEX idx_messaging_behaviors_verified ON messaging_behaviors(is_verified);
CREATE INDEX idx_messaging_behaviors_false_positive ON messaging_behaviors(is_false_positive) WHERE is_false_positive = FALSE;

-- Evidence indexes
CREATE INDEX idx_messaging_evidence_date ON messaging_evidence_items(evidence_date DESC);
CREATE INDEX idx_messaging_evidence_type ON messaging_evidence_items(evidence_type);
CREATE INDEX idx_messaging_evidence_exhibit ON messaging_evidence_items(is_exhibit) WHERE is_exhibit = TRUE;
CREATE INDEX idx_messaging_evidence_message ON messaging_evidence_items(message_id);

-- Factor citation indexes
CREATE INDEX idx_messaging_factor_citations_evidence ON messaging_factor_citations(evidence_id);
CREATE INDEX idx_messaging_factor_citations_factor ON messaging_factor_citations(factor_id);

-- Timeline indexes
CREATE INDEX idx_messaging_timeline_date ON messaging_timeline_events(event_date DESC);
CREATE INDEX idx_messaging_timeline_type ON messaging_timeline_events(event_type);
CREATE INDEX idx_messaging_timeline_significant ON messaging_timeline_events(is_significant) WHERE is_significant = TRUE;

-- =====================================================
-- VIEWS FOR ANALYSIS
-- =====================================================

-- Messages with behavior summary
CREATE VIEW v_messaging_messages_analyzed AS
SELECT 
    m.id,
    m.timestamp,
    m.date_us,
    m.time_12h,
    m.sender,
    m.sender_name,
    m.recipient,
    m.recipient_name,
    m.content,
    m.direction,
    c.platform,
    m.has_behaviors,
    m.behavior_count,
    m.max_severity,
    m.behavior_categories,
    m.has_entities,
    m.mentioned_people,
    m.is_evidence,
    m.mcl_factors,
    ARRAY_AGG(DISTINCT b.category) FILTER (WHERE b.category IS NOT NULL) as detected_behaviors,
    COUNT(DISTINCT b.id) as total_behavior_detections
FROM messaging_messages m
LEFT JOIN messaging_conversations c ON m.conversation_id = c.id
LEFT JOIN messaging_behaviors b ON m.id = b.message_id AND b.is_false_positive = FALSE
GROUP BY m.id, m.timestamp, m.date_us, m.time_12h, m.sender, m.sender_name, 
         m.recipient, m.recipient_name, m.content, m.direction, c.platform,
         m.has_behaviors, m.behavior_count, m.max_severity, m.behavior_categories,
         m.has_entities, m.mentioned_people, m.is_evidence, m.mcl_factors;

-- Daily behavior summary
CREATE VIEW v_messaging_daily_behavior_summary AS
SELECT 
    m.date_us,
    COUNT(DISTINCT m.id) as total_messages,
    COUNT(DISTINCT CASE WHEN m.direction = 'inbound' THEN m.id END) as inbound_messages,
    COUNT(DISTINCT CASE WHEN m.direction = 'outbound' THEN m.id END) as outbound_messages,
    COUNT(DISTINCT b.id) FILTER (WHERE b.is_false_positive = FALSE) as total_behaviors,
    COUNT(DISTINCT CASE WHEN b.severity = 'critical' THEN b.id END) as critical_behaviors,
    COUNT(DISTINCT CASE WHEN b.severity = 'high' THEN b.id END) as high_behaviors,
    COUNT(DISTINCT CASE WHEN b.severity = 'medium' THEN b.id END) as medium_behaviors,
    ARRAY_AGG(DISTINCT b.category) FILTER (WHERE b.category IS NOT NULL) as categories_detected,
    COUNT(DISTINCT CASE WHEN m.is_evidence THEN m.id END) as evidence_messages
FROM messaging_messages m
LEFT JOIN messaging_behaviors b ON m.id = b.message_id
WHERE m.date_us IS NOT NULL
GROUP BY m.date_us
ORDER BY m.date_us DESC;

-- Evidence by MCL factor
CREATE VIEW v_messaging_evidence_by_factor AS
SELECT 
    f.id as factor_id,
    f.name as factor_name,
    f.description as factor_description,
    COUNT(DISTINCT fc.evidence_id) as evidence_count,
    COUNT(DISTINCT CASE WHEN fc.supports_factor = TRUE THEN fc.evidence_id END) as supporting_count,
    COUNT(DISTINCT CASE WHEN fc.supports_factor = FALSE THEN fc.evidence_id END) as contradicting_count,
    ARRAY_AGG(DISTINCT fc.strength) FILTER (WHERE fc.strength IS NOT NULL) as strength_levels,
    COUNT(DISTINCT CASE WHEN fc.strength = 'decisive' THEN fc.evidence_id END) as decisive_count,
    COUNT(DISTINCT CASE WHEN fc.strength = 'strong' THEN fc.evidence_id END) as strong_count
FROM mcl_factors f
LEFT JOIN messaging_factor_citations fc ON f.id = fc.factor_id
GROUP BY f.id, f.name, f.description
ORDER BY f.id;

-- Behavior patterns over time
CREATE VIEW v_messaging_behavior_timeline AS
SELECT 
    m.date_us,
    m.timestamp,
    b.category,
    bc.name as category_name,
    b.severity,
    m.sender,
    m.sender_name,
    m.content,
    b.matched_text,
    b.confidence,
    b.is_verified,
    b.mcl_factors as related_factors
FROM messaging_behaviors b
JOIN messaging_messages m ON b.message_id = m.id
JOIN messaging_behavior_categories bc ON b.category = bc.id
WHERE b.is_false_positive = FALSE
ORDER BY m.timestamp DESC;

-- Conversation statistics
CREATE VIEW v_messaging_conversation_stats AS
SELECT 
    c.id,
    c.platform,
    c.primary_participant,
    c.message_count,
    c.started_at,
    c.ended_at,
    EXTRACT(EPOCH FROM (c.ended_at - c.started_at))::INTEGER as duration_seconds,
    COUNT(DISTINCT b.id) FILTER (WHERE b.is_false_positive = FALSE) as total_behaviors,
    COUNT(DISTINCT CASE WHEN b.severity IN ('high', 'critical') THEN b.id END) as high_severity_behaviors,
    ARRAY_AGG(DISTINCT b.category) FILTER (WHERE b.category IS NOT NULL) as behavior_categories,
    COUNT(DISTINCT e.id) as evidence_item_count,
    c.is_evidence,
    c.exhibit_number
FROM messaging_conversations c
LEFT JOIN messaging_messages m ON c.id = m.conversation_id
LEFT JOIN messaging_behaviors b ON m.id = b.message_id
LEFT JOIN messaging_evidence_items e ON m.id = e.message_id
GROUP BY c.id, c.platform, c.primary_participant, c.message_count, 
         c.started_at, c.ended_at, c.is_evidence, c.exhibit_number;

-- =====================================================
-- TRIGGERS FOR AUTO-UPDATES
-- =====================================================

-- Update message behavior counts when behavior inserted
CREATE OR REPLACE FUNCTION update_message_behavior_counts()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messaging_messages SET
        has_behaviors = TRUE,
        behavior_count = (
            SELECT COUNT(*) 
            FROM messaging_behaviors 
            WHERE message_id = NEW.message_id 
              AND is_false_positive = FALSE
        ),
        max_severity = (
            SELECT severity 
            FROM messaging_behaviors 
            WHERE message_id = NEW.message_id 
              AND is_false_positive = FALSE
            ORDER BY 
                CASE severity 
                    WHEN 'critical' THEN 1 
                    WHEN 'high' THEN 2 
                    WHEN 'medium' THEN 3 
                    WHEN 'low' THEN 4 
                END
            LIMIT 1
        ),
        behavior_categories = (
            SELECT ARRAY_AGG(DISTINCT category)
            FROM messaging_behaviors
            WHERE message_id = NEW.message_id
              AND is_false_positive = FALSE
        )
    WHERE id = NEW.message_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_messaging_behavior_insert
AFTER INSERT OR UPDATE ON messaging_behaviors
FOR EACH ROW
EXECUTE FUNCTION update_message_behavior_counts();

-- Update conversation message count when message inserted
CREATE OR REPLACE FUNCTION update_conversation_message_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messaging_conversations SET
        message_count = (SELECT COUNT(*) FROM messaging_messages WHERE conversation_id = NEW.conversation_id),
        last_message_at = (SELECT MAX(timestamp) FROM messaging_messages WHERE conversation_id = NEW.conversation_id),
        updated_at = NOW()
    WHERE id = NEW.conversation_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_messaging_message_insert
AFTER INSERT ON messaging_messages
FOR EACH ROW
EXECUTE FUNCTION update_conversation_message_count();

-- Update entity mention counts
CREATE OR REPLACE FUNCTION update_entity_mention_counts()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messaging_entities SET
        mention_count = (SELECT COUNT(*) FROM messaging_entity_mentions WHERE entity_id = NEW.entity_id),
        last_mentioned_at = NOW(),
        updated_at = NOW()
    WHERE id = NEW.entity_id;
    
    IF (SELECT first_mentioned_at FROM messaging_entities WHERE id = NEW.entity_id) IS NULL THEN
        UPDATE messaging_entities SET first_mentioned_at = NOW() WHERE id = NEW.entity_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_messaging_entity_mention_insert
AFTER INSERT ON messaging_entity_mentions
FOR EACH ROW
EXECUTE FUNCTION update_entity_mention_counts();

-- Update document processing status
CREATE OR REPLACE FUNCTION update_document_message_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messaging_documents SET
        message_count = (SELECT COUNT(*) FROM messaging_messages WHERE document_id = NEW.document_id),
        updated_at = NOW()
    WHERE id = NEW.document_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_messaging_document_message_insert
AFTER INSERT ON messaging_messages
FOR EACH ROW
WHEN (NEW.document_id IS NOT NULL)
EXECUTE FUNCTION update_document_message_count();

-- =====================================================
-- FORENSIC INTEGRITY FUNCTIONS
-- =====================================================

-- Generate SHA-256 hash of message content
CREATE OR REPLACE FUNCTION generate_content_hash()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.content IS NOT NULL THEN
        NEW.content_hash = encode(digest(NEW.content, 'sha256'), 'hex');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_messaging_message_hash
BEFORE INSERT OR UPDATE OF content ON messaging_messages
FOR EACH ROW
EXECUTE FUNCTION generate_content_hash();

-- Audit log for evidence modifications
CREATE TABLE messaging_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    table_name VARCHAR(100) NOT NULL,
    record_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL CHECK (action IN ('INSERT', 'UPDATE', 'DELETE')),
    changed_by VARCHAR(100),
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    old_values JSONB,
    new_values JSONB,
    change_reason TEXT
);

CREATE OR REPLACE FUNCTION log_evidence_changes()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        INSERT INTO messaging_audit_log (table_name, record_id, action, old_values)
        VALUES (TG_TABLE_NAME, OLD.id, 'DELETE', row_to_json(OLD));
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO messaging_audit_log (table_name, record_id, action, old_values, new_values)
        VALUES (TG_TABLE_NAME, NEW.id, 'UPDATE', row_to_json(OLD), row_to_json(NEW));
        RETURN NEW;
    ELSIF TG_OP = 'INSERT' THEN
        INSERT INTO messaging_audit_log (table_name, record_id, action, new_values)
        VALUES (TG_TABLE_NAME, NEW.id, 'INSERT', row_to_json(NEW));
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_messaging_evidence_audit
AFTER INSERT OR UPDATE OR DELETE ON messaging_evidence_items
FOR EACH ROW
EXECUTE FUNCTION log_evidence_changes();

-- =====================================================
-- COMMENTS FOR DOCUMENTATION
-- =====================================================

COMMENT ON SCHEMA public IS 'Salem v. Kinzel SMS/Messaging Evidence System - Production Schema';

COMMENT ON TABLE messaging_documents IS 'Source files: XML exports, JSON, PDFs, screenshots. Maintains chain of custody.';
COMMENT ON TABLE messaging_conversations IS 'Groups messages into conversations/threads across platforms.';
COMMENT ON TABLE messaging_messages IS 'Core forensic record: individual messages with full metadata and analysis flags.';
COMMENT ON TABLE messaging_entities IS 'People, places, organizations extracted from messages.';
COMMENT ON TABLE messaging_entity_mentions IS 'Where entities appear in messages with position data.';
COMMENT ON TABLE messaging_behavior_categories IS 'Reference table: coercive control and manipulation patterns.';
COMMENT ON TABLE messaging_behaviors IS 'Detected behavioral patterns in messages with confidence scoring.';
COMMENT ON TABLE messaging_evidence_items IS 'Court-ready evidence extracts with authentication and exhibit tracking.';
COMMENT ON TABLE mcl_factors IS 'MCL 722.23 Best Interest Factors reference.';
COMMENT ON TABLE messaging_factor_citations IS 'Links evidence to specific MCL factors with strength ratings.';
COMMENT ON TABLE messaging_timeline_events IS 'Chronological events for timeline reconstruction and geospatial correlation.';
COMMENT ON TABLE messaging_audit_log IS 'Forensic audit trail for all evidence modifications.';

-- =====================================================
-- SAMPLE QUERIES FOR COMMON USE CASES
-- =====================================================

/*
-- Find all high-severity behaviors in December 2024
SELECT m.timestamp, m.sender_name, m.content, b.category, b.matched_text
FROM messaging_messages m
JOIN messaging_behaviors b ON m.id = b.message_id
WHERE b.severity IN ('high', 'critical')
  AND b.is_false_positive = FALSE
  AND m.timestamp >= '2024-12-01'
  AND m.timestamp < '2025-01-01'
ORDER BY m.timestamp DESC;

-- Evidence supporting MCL Factor K (Domestic Violence)
SELECT e.title, e.quote, e.evidence_date, fc.strength
FROM messaging_evidence_items e
JOIN messaging_factor_citations fc ON e.id = fc.evidence_id
WHERE fc.factor_id = 'K'
  AND fc.supports_factor = TRUE
ORDER BY e.evidence_date DESC;

-- Conversation analysis for specific person
SELECT 
    c.platform,
    c.started_at,
    c.ended_at,
    c.message_count,
    COUNT(DISTINCT b.id) as behavior_count,
    ARRAY_AGG(DISTINCT b.category) as detected_patterns
FROM messaging_conversations c
JOIN messaging_messages m ON c.id = m.conversation_id
LEFT JOIN messaging_behaviors b ON m.id = b.message_id AND b.is_false_positive = FALSE
WHERE c.primary_participant = '+15555551234'  -- Replace with actual number
GROUP BY c.id, c.platform, c.started_at, c.ended_at, c.message_count;

-- Timeline of parental alienation attempts
SELECT 
    m.date_us,
    m.timestamp,
    m.sender_name,
    m.content,
    b.matched_text,
    b.confidence
FROM messaging_messages m
JOIN messaging_behaviors b ON m.id = b.message_id
WHERE b.category = 'parental_alienation'
  AND b.is_false_positive = FALSE
ORDER BY m.timestamp;

-- Export court-ready evidence package
SELECT 
    e.exhibit_number,
    e.title,
    e.evidence_date,
    e.quote,
    e.context,
    f.name as mcl_factor,
    fc.strength,
    fc.explanation
FROM messaging_evidence_items e
JOIN messaging_factor_citations fc ON e.id = fc.evidence_id
JOIN mcl_factors f ON fc.factor_id = f.id
WHERE e.is_exhibit = TRUE
  AND e.is_authenticated = TRUE
ORDER BY e.exhibit_number, f.id;
*/

-- =====================================================
-- END OF SCHEMA
-- =====================================================