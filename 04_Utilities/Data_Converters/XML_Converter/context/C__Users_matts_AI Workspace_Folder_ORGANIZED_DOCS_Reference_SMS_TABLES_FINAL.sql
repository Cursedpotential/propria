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
-- MCL 722.23 Focus: Factors K (domestic violence), J (facilitation), G (mental/physical health)
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
    acquired_by VARCHAR(100) DEFAULT 'Matt Salem',
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
    primary_participant_normalized VARCHAR(255),  -- E.164 format for phones
    
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
    
    -- Content (CRITICAL: Preserve nuance)
    content TEXT,  -- Original message body
    content_lower TEXT GENERATED ALWAYS AS (LOWER(content)) STORED,  -- For search
    content_hash VARCHAR(64),  -- SHA-256 of content for deduplication
    word_count INTEGER,
    character_count INTEGER,
    
    -- Message metadata
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('inbound', 'outbound', 'unknown')),
    message_type VARCHAR(20) DEFAULT 'text' CHECK (message_type IN ('text', 'mms', 'voice', 'video', 'sticker', 'reaction', 'system')),
    status VARCHAR(20),  -- 'sent', 'delivered', 'read', 'failed'
    is_read BOOLEAN,
    read_timestamp TIMESTAMPTZ,
    
    -- Attachments
    has_attachments BOOLEAN DEFAULT FALSE,
    attachment_count INTEGER DEFAULT 0,
    attachment_types TEXT[],  -- ['image', 'video', 'audio', 'document']
    
    -- Context preservation (THE NUANCE IS THE ABUSE)
    previous_message_id UUID,  -- Link to prior message for context
    next_message_id UUID,  -- Link to next message
    time_since_previous_seconds INTEGER,  -- Response time analysis
    
    -- Analysis flags (updated by triggers)
    has_behaviors BOOLEAN DEFAULT FALSE,
    behavior_count INTEGER DEFAULT 0,
    behavior_categories TEXT[],  -- Quick lookup array
    max_severity VARCHAR(20),  -- 'low', 'medium', 'high', 'critical'
    
    -- Linguistic markers (for pattern detection)
    contains_apology BOOLEAN DEFAULT FALSE,
    contains_blame BOOLEAN DEFAULT FALSE,
    contains_threat BOOLEAN DEFAULT FALSE,
    contains_minimizing BOOLEAN DEFAULT FALSE,
    question_count INTEGER DEFAULT 0,
    exclamation_count INTEGER DEFAULT 0,
    caps_ratio DECIMAL(3,2),  -- Percentage of uppercase characters
    
    -- Court metadata
    is_evidence BOOLEAN DEFAULT FALSE,
    evidence_item_id UUID,  -- Link to evidence_items table
    is_redacted BOOLEAN DEFAULT FALSE,
    redaction_reason TEXT,
    
    -- Raw preservation (forensic integrity)
    raw_data JSONB,  -- Original XML/JSON structure
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(conversation_id, external_id)
);

-- MMS/Media attachments
CREATE TABLE messaging_attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messaging_messages(id) ON DELETE CASCADE,
    
    -- File identification
    filename VARCHAR(500),
    file_type VARCHAR(50) NOT NULL,  -- 'image', 'video', 'audio', 'document', 'location', 'contact'
    mime_type VARCHAR(100),
    file_hash VARCHAR(64),  -- SHA-256
    file_size BIGINT,
    
    -- Storage
    storage_path TEXT,  -- R2/GCS path
    thumbnail_path TEXT,
    
    -- Media metadata
    width INTEGER,
    height INTEGER,
    duration_seconds INTEGER,  -- For audio/video
    
    -- OCR/Extraction
    ocr_text TEXT,  -- Extracted text from images
    transcription TEXT,  -- For audio/video
    
    -- Content analysis
    contains_faces BOOLEAN,
    face_count INTEGER,
    is_screenshot BOOLEAN,
    
    -- Forensic
    exif_data JSONB,  -- Image metadata (location, device, timestamp)
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- BEHAVIORAL ANALYSIS TABLES
-- =====================================================

-- Behavior categories reference (coercive control patterns)
CREATE TABLE behavior_categories (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    severity_default VARCHAR(20) DEFAULT 'medium',
    mcl_factors VARCHAR(2)[],  -- Array of relevant MCL 722.23 factors
    
    -- Detection patterns
    keyword_patterns TEXT[],  -- Array of regex patterns
    context_required BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert behavior categories (coercive control taxonomy)
INSERT INTO behavior_categories (id, name, description, mcl_factors, severity_default) VALUES
('gaslighting', 'Gaslighting', 'Denial of reality, making victim question their perceptions', ARRAY['F', 'G', 'K'], 'high'),
('blame_shifting', 'Blame Shifting', 'Transferring responsibility for abuse to victim', ARRAY['F', 'J', 'K'], 'medium'),
('minimizing', 'Minimizing', 'Downplaying abuse or harm caused', ARRAY['F', 'K'], 'medium'),
('love_bombing', 'Love Bombing', 'Excessive affection to manipulate or control', ARRAY['F'], 'medium'),
('stonewalling', 'Stonewalling', 'Refusing to communicate, withholding interaction', ARRAY['J'], 'medium'),
('parental_alienation', 'Parental Alienation', 'Undermining child-parent relationship', ARRAY['J', 'K'], 'critical'),
('coercive_control', 'Coercive Control', 'Pattern of controlling behaviors limiting autonomy', ARRAY['F', 'K'], 'critical'),
('financial_control', 'Financial Abuse/Control', 'Controlling access to money/resources', ARRAY['C', 'F', 'K'], 'high'),
('substance_weaponization', 'Substance Use Weaponization', 'Using past substance use as manipulation tool', ARRAY['F', 'G'], 'high'),
('reactive_abuse', 'Reactive Abuse Provocation', 'Provoking victim then framing their reaction as abuse', ARRAY['F', 'K'], 'high'),
('darvo', 'DARVO Pattern', 'Deny, Attack, Reverse Victim and Offender', ARRAY['F', 'K'], 'critical'),
('character_assassination', 'Character Assassination', 'Systematic destruction of victim reputation', ARRAY['F', 'J'], 'high'),
('triangulation', 'Triangulation', 'Using third parties to manipulate or control', ARRAY['F', 'J'], 'medium'),
('isolation', 'Isolation', 'Cutting off victim from support systems', ARRAY['F', 'K'], 'high'),
('monitoring_surveillance', 'Monitoring/Surveillance', 'Tracking location, communications, activities', ARRAY['F', 'K'], 'high'),
('threats', 'Threats', 'Direct or implied threats of harm', ARRAY['K'], 'critical'),
('guilt_tripping', 'Guilt Tripping', 'Manipulating through false guilt', ARRAY['F'], 'medium'),
('double_bind', 'Double Bind', 'No-win situations, conflicting demands', ARRAY['F', 'K'], 'medium');

-- Detected behaviors in messages
CREATE TABLE messaging_behaviors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messaging_messages(id) ON DELETE CASCADE,
    
    -- Classification
    category VARCHAR(50) NOT NULL REFERENCES behavior_categories(id),
    subcategory VARCHAR(100),
    
    -- Evidence within message
    matched_pattern TEXT,  -- The regex/pattern that matched
    matched_text TEXT,  -- The actual text that matched
    start_char INTEGER,
    end_char INTEGER,
    context_before TEXT,  -- 100 chars before
    context_after TEXT,  -- 100 chars after
    
    -- Scoring
    confidence DECIMAL(3,2) NOT NULL,  -- 0.00 to 1.00
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    
    -- Detection metadata
    detection_method VARCHAR(50) NOT NULL,  -- 'keyword', 'llm', 'rule_based', 'manual'
    rule_name VARCHAR(100),
    model_name VARCHAR(100),  -- If LLM detected
    
    -- Verification
    is_verified BOOLEAN DEFAULT FALSE,
    verified_by VARCHAR(100),
    verified_at TIMESTAMPTZ,
    verification_notes TEXT,
    is_disputed BOOLEAN DEFAULT FALSE,
    
    -- Court relevance
    mcl_factors VARCHAR(2)[],  -- Computed from category
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Behavioral patterns (multi-message analysis)
CREATE TABLE messaging_behavior_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES messaging_conversations(id) ON DELETE CASCADE,
    
    -- Pattern identification
    pattern_type VARCHAR(50) NOT NULL,  -- 'escalation', 'cycle', 'triggered_response', 'time_based'
    pattern_name VARCHAR(200),
    description TEXT,
    
    -- Messages involved
    message_ids UUID[] NOT NULL,
    message_count INTEGER NOT NULL,
    
    -- Temporal span
    start_timestamp TIMESTAMPTZ NOT NULL,
    end_timestamp TIMESTAMPTZ NOT NULL,
    duration_hours INTEGER,
    
    -- Pattern characteristics
    behavior_categories TEXT[],
    severity_progression TEXT,  -- 'escalating', 'stable', 'de-escalating'
    
    -- Analysis
    confidence DECIMAL(3,2),
    evidence_summary TEXT,
    
    -- Court relevance
    is_evidence BOOLEAN DEFAULT FALSE,
    mcl_factors VARCHAR(2)[],
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- ENTITY TABLES
-- =====================================================

-- People mentioned in messages
CREATE TABLE messaging_entities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    entity_type VARCHAR(50) NOT NULL CHECK (entity_type IN ('person', 'location', 'organization', 'phone', 'email')),
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
    role VARCHAR(50),  -- 'party', 'witness', 'child', 'attorney', 'family', 'friend', etc.
    is_party BOOLEAN DEFAULT FALSE,  -- Party to Salem v. Kinzel case
    relationship_to_petitioner TEXT,
    relationship_to_respondent TEXT,
    
    -- Case relevance
    is_flagged BOOLEAN DEFAULT FALSE,
    flag_reason TEXT,
    
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
    
    -- Extraction info
    confidence DECIMAL(3,2),
    extraction_method VARCHAR(50),  -- 'regex', 'ner', 'llm', 'manual'
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- EVIDENCE & COURT TABLES
-- =====================================================

-- Court-ready evidence items
CREATE TABLE messaging_evidence_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Source
    message_id UUID REFERENCES messaging_messages(id) ON DELETE SET NULL,
    pattern_id UUID REFERENCES messaging_behavior_patterns(id) ON DELETE SET NULL,
    document_id UUID REFERENCES messaging_documents(id) ON DELETE SET NULL,
    
    -- Evidence content
    title VARCHAR(500) NOT NULL,
    description TEXT,
    quote TEXT,  -- The relevant quote/excerpt
    context TEXT,  -- Surrounding messages for context
    
    -- Classification
    evidence_type VARCHAR(50),  -- 'communication', 'pattern', 'screenshot', 'recording'
    category VARCHAR(100),
    
    -- Dates
    evidence_date TIMESTAMPTZ NOT NULL,
    date_precision VARCHAR(20) DEFAULT 'exact',
    
    -- Legal relevance
    mcl_factors VARCHAR(2)[] NOT NULL,  -- Which MCL 722.23 factors this supports
    relevance_score DECIMAL(3,2),  -- 0.00 to 1.00
    strength VARCHAR(20),  -- 'weak', 'moderate', 'strong', 'decisive'
    
    -- Exhibit management
    exhibit_number VARCHAR(50),
    is_exhibit BOOLEAN DEFAULT FALSE,
    is_filed BOOLEAN DEFAULT FALSE,
    filed_date DATE,
    
    -- Authentication
    is_authenticated BOOLEAN DEFAULT FALSE,
    authentication_method TEXT,
    chain_of_custody TEXT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- MCL 722.23 Best Interest Factors (reference table)
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

-- Link messages to timeline events
CREATE TABLE messaging_timeline_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Timing
    event_date TIMESTAMPTZ NOT NULL,
    event_date_end TIMESTAMPTZ,  -- For events with duration
    date_precision VARCHAR(20) DEFAULT 'exact',
    
    -- Event info
    title VARCHAR(500) NOT NULL,
    description TEXT,
    event_type VARCHAR(50) NOT NULL,  -- 'communication', 'incident', 'pattern', 'visit', etc.
    
    -- Sources
    message_ids UUID[],  -- Array of related message IDs
    pattern_ids UUID[],  -- Array of related pattern IDs
    evidence_ids UUID[],  -- Array of related evidence IDs
    
    -- Location (if applicable)
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
-- INDEXES (Performance Critical)
-- =====================================================

-- Documents
CREATE INDEX idx_messaging_documents_hash ON messaging_documents(file_hash);
CREATE INDEX idx_messaging_documents_status ON messaging_documents(status);
CREATE INDEX idx_messaging_documents_date_range ON messaging_documents(date_range_start, date_range_end);

-- Conversations
CREATE INDEX idx_messaging_conversations_platform ON messaging_conversations(platform);
CREATE INDEX idx_messaging_conversations_participant ON messaging_conversations(primary_participant_normalized);
CREATE INDEX idx_messaging_conversations_dates ON messaging_conversations(started_at, ended_at);

-- Messages (CRITICAL - Most queried)
CREATE INDEX idx_messaging_messages_conversation ON messaging_messages(conversation_id);
CREATE INDEX idx_messaging_messages_timestamp ON messaging_messages(timestamp);
CREATE INDEX idx_messaging_messages_sender ON messaging_messages(sender_normalized);
CREATE INDEX idx_messaging_messages_direction ON messaging_messages(direction);
CREATE INDEX idx_messaging_messages_has_behaviors ON messaging_messages(has_behaviors) WHERE has_behaviors = TRUE;
CREATE INDEX idx_messaging_messages_evidence ON messaging_messages(is_evidence) WHERE is_evidence = TRUE;
CREATE INDEX idx_messaging_messages_content_search ON messaging_messages USING gin(content_lower gin_trgm_ops);
CREATE INDEX idx_messaging_messages_hash ON messaging_messages(content_hash);

-- Composite indexes for common queries
CREATE INDEX idx_messaging_messages_conv_timestamp ON messaging_messages(conversation_id, timestamp);
CREATE INDEX idx_messaging_messages_sender_timestamp ON messaging_messages(sender_normalized, timestamp);

-- Attachments
CREATE INDEX idx_messaging_attachments_message ON messaging_attachments(message_id);
CREATE INDEX idx_messaging_attachments_type ON messaging_attachments(file_type);
CREATE INDEX idx_messaging_attachments_hash ON messaging_attachments(file_hash);

-- Behaviors
CREATE INDEX idx_messaging_behaviors_message ON messaging_behaviors(message_id);
CREATE INDEX idx_messaging_behaviors_category ON messaging_behaviors(category);
CREATE INDEX idx_messaging_behaviors_severity ON messaging_behaviors(severity);
CREATE INDEX idx_messaging_behaviors_verified ON messaging_behaviors(is_verified);
CREATE INDEX idx_messaging_behaviors_mcl ON messaging_behaviors USING gin(mcl_factors);

-- Behavior patterns
CREATE INDEX idx_messaging_patterns_conversation ON messaging_behavior_patterns(conversation_id);
CREATE INDEX idx_messaging_patterns_timestamp ON messaging_behavior_patterns(start_timestamp, end_timestamp);
CREATE INDEX idx_messaging_patterns_evidence ON messaging_behavior_patterns(is_evidence) WHERE is_evidence = TRUE;

-- Entities
CREATE INDEX idx_messaging_entities_type ON messaging_entities(entity_type);
CREATE INDEX idx_messaging_entities_name ON messaging_entities(normalized_name);
CREATE INDEX idx_messaging_entities_party ON messaging_entities(is_party) WHERE is_party = TRUE;
CREATE INDEX idx_messaging_entities_flagged ON messaging_entities(is_flagged) WHERE is_flagged = TRUE;

-- Entity mentions
CREATE INDEX idx_messaging_entity_mentions_message ON messaging_entity_mentions(message_id);
CREATE INDEX idx_messaging_entity_mentions_entity ON messaging_entity_mentions(entity_id);

-- Evidence
CREATE INDEX idx_messaging_evidence_date ON messaging_evidence_items(evidence_date);
CREATE INDEX idx_messaging_evidence_type ON messaging_evidence_items(evidence_type);
CREATE INDEX idx_messaging_evidence_mcl ON messaging_evidence_items USING gin(mcl_factors);
CREATE INDEX idx_messaging_evidence_exhibit ON messaging_evidence_items(is_exhibit) WHERE is_exhibit = TRUE;

-- Factor citations
CREATE INDEX idx_messaging_factor_citations_evidence ON messaging_factor_citations(evidence_id);
CREATE INDEX idx_messaging_factor_citations_factor ON messaging_factor_citations(factor_id);

-- Timeline
CREATE INDEX idx_messaging_timeline_date ON messaging_timeline_events(event_date);
CREATE INDEX idx_messaging_timeline_type ON messaging_timeline_events(event_type);
CREATE INDEX idx_messaging_timeline_mcl ON messaging_timeline_events USING gin(mcl_factors);

-- =====================================================
-- TRIGGERS (Automatic Updates)
-- =====================================================

-- Update message behavior counts when behavior is added
CREATE OR REPLACE FUNCTION update_message_behavior_counts()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messaging_messages SET
        has_behaviors = TRUE,
        behavior_count = (SELECT COUNT(*) FROM messaging_behaviors WHERE message_id = NEW.message_id),
        behavior_categories = (SELECT ARRAY_AGG(DISTINCT category) FROM messaging_behaviors WHERE message_id = NEW.message_id),
        max_severity = (
            SELECT severity FROM messaging_behaviors 
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

CREATE TRIGGER trg_messaging_behavior_insert
AFTER INSERT ON messaging_behaviors
FOR EACH ROW
EXECUTE FUNCTION update_message_behavior_counts();

-- Update conversation message count when message is added
CREATE OR REPLACE FUNCTION update_conversation_message_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messaging_conversations SET
        message_count = (SELECT COUNT(*) FROM messaging_messages WHERE conversation_id = NEW.conversation_id),
        last_message_at = GREATEST(COALESCE(last_message_at, NEW.timestamp), NEW.timestamp),
        updated_at = NOW()
    WHERE id = NEW.conversation_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_messaging_message_insert
AFTER INSERT ON messaging_messages
FOR EACH ROW
EXECUTE FUNCTION update_conversation_message_count();

-- Update document message count
CREATE OR REPLACE FUNCTION update_document_message_count()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.document_id IS NOT NULL THEN
        UPDATE messaging_documents SET
            message_count = (SELECT COUNT(*) FROM messaging_messages WHERE document_id = NEW.document_id),
            updated_at = NOW()
        WHERE id = NEW.document_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_messaging_doc_message_insert
AFTER INSERT ON messaging_messages
FOR EACH ROW
EXECUTE FUNCTION update_document_message_count();

-- Auto-populate MCL factors in behaviors from category
CREATE OR REPLACE FUNCTION populate_behavior_mcl_factors()
RETURNS TRIGGER AS $$
BEGIN
    NEW.mcl_factors := (SELECT mcl_factors FROM behavior_categories WHERE id = NEW.category);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_messaging_behavior_mcl
BEFORE INSERT ON messaging_behaviors
FOR EACH ROW
EXECUTE FUNCTION populate_behavior_mcl_factors();

-- =====================================================
-- VIEWS (Convenient Queries)
-- =====================================================

-- Messages with full behavioral analysis
CREATE VIEW v_messaging_analyzed AS
SELECT 
    m.id,
    m.timestamp,
    m.date_us,
    m.time_12h,
    m.sender,
    m.sender_name,
    m.recipient,
    m.direction,
    m.content,
    m.word_count,
    c.platform,
    c.primary_participant,
    m.has_behaviors,
    m.behavior_count,
    m.behavior_categories,
    m.max_severity,
    ARRAY_AGG(DISTINCT b.category) FILTER (WHERE b.category IS NOT NULL) as detected_behaviors,
    ARRAY_AGG(DISTINCT bc.mcl_factors) FILTER (WHERE bc.mcl_factors IS NOT NULL) as related_mcl_factors
FROM messaging_messages m
LEFT JOIN messaging_conversations c ON m.conversation_id = c.id
LEFT JOIN messaging_behaviors b ON m.id = b.message_id
LEFT JOIN behavior_categories bc ON b.category = bc.id
GROUP BY m.id, m.timestamp, m.date_us, m.time_12h, m.sender, m.sender_name, 
         m.recipient, m.direction, m.content, m.word_count, c.platform, 
         c.primary_participant, m.has_behaviors, m.behavior_count, 
         m.behavior_categories, m.max_severity;

-- Evidence by MCL factor
CREATE VIEW v_messaging_evidence_by_factor AS
SELECT 
    f.id as factor_id,
    f.name as factor_name,
    fc.supports_factor,
    fc.strength,
    e.id as evidence_id,
    e.title,
    e.quote,
    e.evidence_date,
    e.exhibit_number,
    e.is_filed,
    fc.explanation
FROM mcl_factors f
LEFT JOIN messaging_factor_citations fc ON f.id = fc.factor_id
LEFT JOIN messaging_evidence_items e ON fc.evidence_id = e.id
ORDER BY f.id, e.evidence_date DESC;

-- Daily behavior summary
CREATE VIEW v_messaging_daily_behavior_summary AS
SELECT 
    DATE(m.timestamp) as date,
    COUNT(DISTINCT m.id) as total_messages,
    COUNT(DISTINCT CASE WHEN m.direction = 'inbound' THEN m.id END) as inbound_count,
    COUNT(DISTINCT CASE WHEN m.direction = 'outbound' THEN m.id END) as outbound_count,
    COUNT(DISTINCT b.id) as total_behaviors,
    COUNT(DISTINCT CASE WHEN b.severity = 'critical' THEN b.id END) as critical_count,
    COUNT(DISTINCT CASE WHEN b.severity = 'high' THEN b.id END) as high_count,
    COUNT(DISTINCT CASE WHEN b.severity = 'medium' THEN b.id END) as medium_count,
    COUNT(DISTINCT CASE WHEN b.severity = 'low' THEN b.id END) as low_count,
    ARRAY_AGG(DISTINCT b.category) FILTER (WHERE b.category IS NOT NULL) as categories_detected
FROM messaging_messages m
LEFT JOIN messaging_behaviors b ON m.id = b.message_id
GROUP BY DATE(m.timestamp)
ORDER BY date DESC;

-- Conversation summary with behavior stats
CREATE VIEW v_messaging_conversation_summary AS
SELECT 
    c.id,
    c.platform,
    c.primary_participant,
    c.started_at,
    c.ended_at,
    c.message_count,
    COUNT(DISTINCT b.id) as total_behaviors,
    COUNT(DISTINCT b.category) as unique_behavior_types,
    COUNT(DISTINCT CASE WHEN b.severity = 'critical' THEN b.id END) as critical_behaviors,
    c.is_evidence,
    c.exhibit_number,
    ARRAY_AGG(DISTINCT b.category) FILTER (WHERE b.category IS NOT NULL) as behavior_categories
FROM messaging_conversations c
LEFT JOIN messaging_messages m ON c.id = m.conversation_id
LEFT JOIN messaging_behaviors b ON m.id = b.message_id
GROUP BY c.id, c.platform, c.primary_participant, c.started_at, c.ended_at, 
         c.message_count, c.is_evidence, c.exhibit_number
ORDER BY c.started_at DESC;

-- =====================================================
-- FORENSIC INTEGRITY FUNCTIONS
-- =====================================================

-- Generate SHA-256 hash for content
CREATE OR REPLACE FUNCTION generate_content_hash(content TEXT)
RETURNS VARCHAR(64) AS $$
BEGIN
    RETURN encode(digest(content, 'sha256'), 'hex');
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Validate message chain integrity
CREATE OR REPLACE FUNCTION validate_message_chain(conv_id UUID)
RETURNS TABLE(is_valid BOOLEAN, broken_links INTEGER, message_count INTEGER) AS $$
BEGIN
    RETURN QUERY
    WITH chain_check AS (
        SELECT 
            m.id,
            m.previous_message_id,
            m.next_message_id,
            LAG(m.id) OVER (ORDER BY m.timestamp) as expected_prev,
            LEAD(m.id) OVER (ORDER BY m.timestamp) as expected_next
        FROM messaging_messages m
        WHERE m.conversation_id = conv_id
    )
    SELECT 
        COUNT(*) FILTER (WHERE previous_message_id != expected_prev OR next_message_id != expected_next) = 0 as is_valid,
        COUNT(*) FILTER (WHERE previous_message_id != expected_prev OR next_message_id != expected_next)::INTEGER as broken_links,
        COUNT(*)::INTEGER as message_count
    FROM chain_check;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- COMMENTS (Documentation)
-- =====================================================

COMMENT ON SCHEMA public IS 'Salem v. Kinzel Evidence Management - SMS/Messaging Module';
COMMENT ON TABLE messaging_documents IS 'Source files (XML exports, screenshots, PDFs) - Chain of custody tracking';
COMMENT ON TABLE messaging_conversations IS 'Conversation threads grouping related messages';
COMMENT ON TABLE messaging_messages IS 'Individual messages - Core forensic evidence with nuance preservation';
COMMENT ON TABLE messaging_attachments IS 'MMS attachments, media files, screenshots';
COMMENT ON TABLE behavior_categories IS 'Coercive control behavior taxonomy for detection';
COMMENT ON TABLE messaging_behaviors IS 'Detected abusive/manipulative behaviors in messages';
COMMENT ON TABLE messaging_behavior_patterns IS 'Multi-message patterns (escalation, cycles, etc.)';
COMMENT ON TABLE messaging_entities IS 'People, places, organizations mentioned in messages';
COMMENT ON TABLE messaging_evidence_items IS 'Court-ready evidence pieces linked to MCL 722.23 factors';
COMMENT ON TABLE mcl_factors IS 'MCL 722.23 Best Interest Factors reference';
COMMENT ON TABLE messaging_factor_citations IS 'Links evidence to legal factors';
COMMENT ON TABLE messaging_timeline_events IS 'Chronological events for master timeline';

-- =====================================================
-- USAGE NOTES
-- =====================================================

/*
DEPLOYMENT STEPS:
1. Run this schema in Supabase SQL Editor
2. Verify all tables/indexes created successfully
3. Test triggers with sample data
4. Configure RLS policies if multi-user access needed
5. Import existing SMS XML exports using Chat Miner parser
6. Run behavioral analysis pipeline
7. Generate evidence reports

INTEGRATION POINTS:
- Chat Miner: Parses SMS XML → messaging_documents + messaging_messages
- TraceIQ: Links location data to messages via timeline_events
- Chronicle: Voice narratives reference message evidence
- Video Analyzer: Screenshot attachments link to messaging_attachments
- Neo4j Aura: Entity relationship graph from messaging_entities
- Qdrant Cloud: Vector embeddings of message content for semantic search

COURT ADMISSIBILITY:
- Chain of custody tracked via messaging_documents
- SHA-256 hashes prevent tampering
- Timestamps preserved with precision indicators
- Original raw data preserved in JSONB fields
- Behavioral analysis transparent via detection_method
- MCL 722.23 factor links documented

MCL 722.23 PRIORITY FACTORS:
- Factor K (Domestic Violence): Coercive control, threats, patterns
- Factor J (Facilitation): Parental alienation, stonewalling
- Factor G (Mental/Physical Health): Substance weaponization, stress
*/
