-- =====================================================
-- SUPABASE PRODUCTION SCHEMA
-- Salem v. Kinzel Evidence Management System
-- =====================================================
-- 
-- This schema goes in your Supabase project (oflqpddqaecotsdsxbzp)
-- Run this via Supabase SQL Editor
--
-- Tables:
--   documents          - Source files and their metadata
--   conversations      - Chat/messaging conversations
--   messages           - Individual messages within conversations
--   entities           - People, places, organizations
--   entity_mentions    - Where entities appear in messages
--   behaviors          - Detected behavioral patterns
--   evidence_items     - Court-relevant evidence pieces
--   timeline_events    - Chronological event tracking
--   mcl_factors        - MCL 722.23 factor reference
--   factor_citations   - Links evidence to legal factors

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For text search

-- =====================================================
-- REFERENCE TABLES
-- =====================================================

-- MCL 722.23 Best Interest Factors
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

-- Behavior categories reference
CREATE TABLE behavior_categories (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    severity_default VARCHAR(20) DEFAULT 'medium',
    mcl_factors VARCHAR(20)[],  -- Array of relevant factors
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert behavior categories (from your YAML configs)
INSERT INTO behavior_categories (id, name, mcl_factors) VALUES
('gaslighting', 'Gaslighting', ARRAY['F', 'G', 'K']),
('blame_shifting', 'Blame Shifting', ARRAY['F', 'J']),
('minimizing', 'Minimizing', ARRAY['F', 'K']),
('love_bombing', 'Love Bombing', ARRAY['F']),
('stonewalling', 'Stonewalling', ARRAY['J']),
('parental_alienation', 'Parental Alienation', ARRAY['J', 'K']),
('coercive_control', 'Coercive Control', ARRAY['F', 'K']),
('financial_abuse', 'Financial Abuse/Control', ARRAY['C', 'F', 'K']),
('substance_weaponization', 'Substance Use Weaponization', ARRAY['F', 'G']),
('reactive_abuse', 'Reactive Abuse Provocation', ARRAY['F', 'K']),
('darvo', 'DARVO Pattern', ARRAY['F', 'K']),
('character_assassination', 'Character Assassination', ARRAY['F', 'J']);

-- =====================================================
-- CORE DATA TABLES
-- =====================================================

-- Source documents
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    filename VARCHAR(500) NOT NULL,
    file_type VARCHAR(50),  -- 'sms_xml', 'pdf', 'image', 'json', etc.
    file_hash VARCHAR(64),  -- SHA-256 for deduplication
    file_size BIGINT,
    storage_path TEXT,  -- R2/GCS path
    
    -- Processing status
    status VARCHAR(20) DEFAULT 'pending',  -- pending, processing, completed, failed
    processed_at TIMESTAMPTZ,
    
    -- Extracted content
    raw_text TEXT,
    page_count INTEGER,
    
    -- Metadata
    source_device VARCHAR(100),
    date_range_start TIMESTAMPTZ,
    date_range_end TIMESTAMPTZ,
    record_count INTEGER,
    
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Conversations (groups messages together)
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID REFERENCES documents(id) ON DELETE SET NULL,
    
    -- Platform info
    platform VARCHAR(50) NOT NULL,  -- 'sms', 'facebook', 'claude', 'email', etc.
    platform_id VARCHAR(255),  -- External conversation ID
    
    -- Participants
    participants TEXT[],  -- Array of participant identifiers
    participant_count INTEGER,
    
    -- Conversation metadata
    title VARCHAR(500),
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    message_count INTEGER DEFAULT 0,
    
    -- Analysis summary
    behavior_summary JSONB,  -- Aggregated behavior counts
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(platform, platform_id)
);

-- Individual messages
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
    document_id UUID REFERENCES documents(id) ON DELETE SET NULL,
    
    -- Message data
    external_id VARCHAR(100),  -- Original message ID
    timestamp TIMESTAMPTZ NOT NULL,
    sender VARCHAR(255),
    sender_normalized VARCHAR(255),  -- Standardized (e.g., phone number format)
    recipient VARCHAR(255),
    
    -- Content
    content TEXT,
    content_lower TEXT GENERATED ALWAYS AS (LOWER(content)) STORED,  -- For search
    
    -- Message metadata
    direction VARCHAR(10),  -- 'inbound', 'outbound'
    message_type VARCHAR(20),  -- 'text', 'mms', 'voice', etc.
    status VARCHAR(20),  -- 'sent', 'delivered', 'read', 'failed'
    is_read BOOLEAN,
    
    -- Attachments
    has_attachments BOOLEAN DEFAULT FALSE,
    attachment_count INTEGER DEFAULT 0,
    
    -- Analysis flags
    has_behaviors BOOLEAN DEFAULT FALSE,
    behavior_count INTEGER DEFAULT 0,
    max_severity VARCHAR(20),
    
    -- Raw preservation
    raw_data JSONB,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(conversation_id, external_id)
);

-- Entities (people, places, organizations)
CREATE TABLE entities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    entity_type VARCHAR(50) NOT NULL,  -- 'person', 'location', 'organization', 'phone', etc.
    name VARCHAR(255) NOT NULL,
    normalized_name VARCHAR(255),  -- Standardized version
    
    -- For people
    aliases TEXT[],
    phone_numbers TEXT[],
    email_addresses TEXT[],
    
    -- For locations
    address TEXT,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    
    -- Classification
    role VARCHAR(50),  -- 'party', 'witness', 'child', 'attorney', etc.
    is_party BOOLEAN DEFAULT FALSE,  -- Is this a party to the case
    
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(entity_type, normalized_name)
);

-- Entity mentions in messages
CREATE TABLE entity_mentions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
    entity_id UUID REFERENCES entities(id) ON DELETE CASCADE,
    
    -- Position in text
    mention_text TEXT,
    start_char INTEGER,
    end_char INTEGER,
    
    -- Extraction info
    confidence DECIMAL(3,2),
    extraction_method VARCHAR(50),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Detected behaviors
CREATE TABLE behaviors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
    
    -- Classification
    category VARCHAR(50) NOT NULL REFERENCES behavior_categories(id),
    subcategory VARCHAR(100),
    
    -- Evidence
    matched_pattern TEXT,
    matched_text TEXT,
    start_char INTEGER,
    end_char INTEGER,
    
    -- Scoring
    confidence DECIMAL(3,2),
    severity VARCHAR(20) NOT NULL,  -- 'low', 'medium', 'high', 'critical'
    
    -- Detection
    detection_method VARCHAR(50),
    rule_name VARCHAR(100),
    
    -- Verification
    is_verified BOOLEAN DEFAULT FALSE,
    verified_by VARCHAR(100),
    verified_at TIMESTAMPTZ,
    verification_notes TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Evidence items (court-ready)
CREATE TABLE evidence_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Source
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    document_id UUID REFERENCES documents(id) ON DELETE SET NULL,
    
    -- Evidence content
    title VARCHAR(500) NOT NULL,
    description TEXT,
    quote TEXT,  -- The relevant quote
    context TEXT,  -- Surrounding context
    
    -- Classification
    evidence_type VARCHAR(50),  -- 'communication', 'document', 'photo', 'record'
    category VARCHAR(100),
    
    -- Dates
    evidence_date TIMESTAMPTZ,
    date_precision VARCHAR(20),  -- 'exact', 'approximate', 'range'
    
    -- Legal relevance
    relevance_score DECIMAL(3,2),
    
    -- Exhibit info
    exhibit_number VARCHAR(50),
    is_exhibit BOOLEAN DEFAULT FALSE,
    
    -- Verification
    is_authenticated BOOLEAN DEFAULT FALSE,
    authentication_method TEXT,
    chain_of_custody TEXT,
    
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Link evidence to MCL factors
CREATE TABLE factor_citations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    evidence_id UUID REFERENCES evidence_items(id) ON DELETE CASCADE,
    factor_id VARCHAR(2) REFERENCES mcl_factors(id) ON DELETE CASCADE,
    
    -- How it supports/contradicts
    supports_factor BOOLEAN DEFAULT TRUE,  -- TRUE = supports, FALSE = contradicts
    strength VARCHAR(20),  -- 'weak', 'moderate', 'strong', 'decisive'
    explanation TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(evidence_id, factor_id)
);

-- Timeline events (for chronological analysis)
CREATE TABLE timeline_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Timing
    event_date TIMESTAMPTZ NOT NULL,
    event_date_end TIMESTAMPTZ,  -- For events with duration
    date_precision VARCHAR(20) DEFAULT 'exact',
    
    -- Event info
    title VARCHAR(500) NOT NULL,
    description TEXT,
    event_type VARCHAR(50),  -- 'communication', 'incident', 'filing', 'visit', etc.
    
    -- Sources
    message_ids UUID[],  -- Array of related message IDs
    document_ids UUID[],  -- Array of related document IDs
    evidence_ids UUID[],  -- Array of related evidence IDs
    
    -- Location
    location TEXT,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    
    -- Participants
    participants TEXT[],
    
    -- Legal relevance
    mcl_factors VARCHAR(2)[],
    
    -- Flags
    is_verified BOOLEAN DEFAULT FALSE,
    is_disputed BOOLEAN DEFAULT FALSE,
    
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- INDEXES
-- =====================================================

-- Messages
CREATE INDEX idx_messages_conversation ON messages(conversation_id);
CREATE INDEX idx_messages_timestamp ON messages(timestamp);
CREATE INDEX idx_messages_sender ON messages(sender_normalized);
CREATE INDEX idx_messages_content_search ON messages USING gin(content_lower gin_trgm_ops);
CREATE INDEX idx_messages_has_behaviors ON messages(has_behaviors) WHERE has_behaviors = TRUE;

-- Behaviors
CREATE INDEX idx_behaviors_message ON behaviors(message_id);
CREATE INDEX idx_behaviors_category ON behaviors(category);
CREATE INDEX idx_behaviors_severity ON behaviors(severity);
CREATE INDEX idx_behaviors_verified ON behaviors(is_verified);

-- Entities
CREATE INDEX idx_entities_type ON entities(entity_type);
CREATE INDEX idx_entities_name ON entities(normalized_name);

-- Evidence
CREATE INDEX idx_evidence_date ON evidence_items(evidence_date);
CREATE INDEX idx_evidence_type ON evidence_items(evidence_type);

-- Timeline
CREATE INDEX idx_timeline_date ON timeline_events(event_date);
CREATE INDEX idx_timeline_type ON timeline_events(event_type);

-- Factor citations
CREATE INDEX idx_factor_citations_evidence ON factor_citations(evidence_id);
CREATE INDEX idx_factor_citations_factor ON factor_citations(factor_id);

-- =====================================================
-- VIEWS
-- =====================================================

-- Messages with behavior summary
CREATE VIEW v_messages_analyzed AS
SELECT 
    m.id,
    m.timestamp,
    m.sender,
    m.recipient,
    m.content,
    m.direction,
    c.platform,
    m.has_behaviors,
    m.behavior_count,
    m.max_severity,
    ARRAY_AGG(DISTINCT b.category) FILTER (WHERE b.category IS NOT NULL) as behavior_categories,
    ARRAY_AGG(DISTINCT bc.mcl_factors) FILTER (WHERE bc.mcl_factors IS NOT NULL) as related_factors
FROM messages m
LEFT JOIN conversations c ON m.conversation_id = c.id
LEFT JOIN behaviors b ON m.id = b.message_id
LEFT JOIN behavior_categories bc ON b.category = bc.id
GROUP BY m.id, m.timestamp, m.sender, m.recipient, m.content, m.direction, 
         c.platform, m.has_behaviors, m.behavior_count, m.max_severity;

-- Evidence by MCL factor
CREATE VIEW v_evidence_by_factor AS
SELECT 
    f.id as factor_id,
    f.name as factor_name,
    fc.supports_factor,
    fc.strength,
    e.id as evidence_id,
    e.title,
    e.quote,
    e.evidence_date,
    fc.explanation
FROM mcl_factors f
LEFT JOIN factor_citations fc ON f.id = fc.factor_id
LEFT JOIN evidence_items e ON fc.evidence_id = e.id
ORDER BY f.id, e.evidence_date;

-- Daily behavior summary
CREATE VIEW v_daily_behavior_summary AS
SELECT 
    DATE(m.timestamp) as date,
    COUNT(DISTINCT m.id) as total_messages,
    COUNT(DISTINCT b.id) as total_behaviors,
    COUNT(DISTINCT CASE WHEN b.severity = 'critical' THEN b.id END) as critical_count,
    COUNT(DISTINCT CASE WHEN b.severity = 'high' THEN b.id END) as high_count,
    ARRAY_AGG(DISTINCT b.category) FILTER (WHERE b.category IS NOT NULL) as categories
FROM messages m
LEFT JOIN behaviors b ON m.id = b.message_id
GROUP BY DATE(m.timestamp)
ORDER BY date;

-- =====================================================
-- FUNCTIONS
-- =====================================================

-- Update message behavior counts
CREATE OR REPLACE FUNCTION update_message_behavior_counts()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messages SET
        has_behaviors = TRUE,
        behavior_count = (SELECT COUNT(*) FROM behaviors WHERE message_id = NEW.message_id),
        max_severity = (
            SELECT severity FROM behaviors 
            WHERE message_id = NEW.message_id 
            ORDER BY 
                CASE severity 
                    WHEN 'critical' THEN 1 
                    WHEN 'high' THEN 2 
                    WHEN 'medium' THEN 3 
                    WHEN 'low' THEN 4 
                END
            LIMIT 1
        )
    WHERE id = NEW.message_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_behavior_insert
AFTER INSERT ON behaviors
FOR EACH ROW
EXECUTE FUNCTION update_message_behavior_counts();

-- Update conversation message count
CREATE OR REPLACE FUNCTION update_conversation_message_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE conversations SET
        message_count = (SELECT COUNT(*) FROM messages WHERE conversation_id = NEW.conversation_id),
        updated_at = NOW()
    WHERE id = NEW.conversation_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_message_insert
AFTER INSERT ON messages
FOR EACH ROW
EXECUTE FUNCTION update_conversation_message_count();

-- =====================================================
-- ROW LEVEL SECURITY (Optional - enable if needed)
-- =====================================================

-- ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE behaviors ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE evidence_items ENABLE ROW LEVEL SECURITY;

COMMENT ON SCHEMA public IS 'Salem v. Kinzel Evidence Management - Production Schema';
