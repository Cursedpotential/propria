-- N8N Local Processing Database Schema
-- These tables are for TEMPORARY processing before pushing to production (Supabase)
-- 
-- Tables:
--   staging_messages     - Raw messages before classification
--   staging_entities     - Extracted entities pending review
--   staging_behaviors    - Detected behavioral patterns
--   processing_jobs      - Track processing status
--   processing_errors    - Log errors for debugging

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Processing job tracking
CREATE TABLE processing_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_file VARCHAR(500) NOT NULL,
    source_type VARCHAR(50) NOT NULL,  -- 'sms_xml', 'claude_json', 'facebook_json', etc.
    status VARCHAR(20) DEFAULT 'pending',  -- pending, processing, completed, failed
    total_records INTEGER DEFAULT 0,
    processed_records INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);

-- Staging table for raw messages
CREATE TABLE staging_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID REFERENCES processing_jobs(id) ON DELETE CASCADE,
    
    -- Core message data
    external_id VARCHAR(100),  -- Original ID from source
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    sender VARCHAR(255),
    recipient VARCHAR(255),
    content TEXT,
    
    -- Source info
    source_type VARCHAR(50),  -- 'sms', 'mms', 'claude', 'facebook', etc.
    source_platform VARCHAR(50),  -- 'android', 'ios', 'web', etc.
    direction VARCHAR(10),  -- 'inbound', 'outbound'
    
    -- Processing status
    is_processed BOOLEAN DEFAULT FALSE,
    is_pushed BOOLEAN DEFAULT FALSE,  -- Pushed to production
    
    -- Raw data preservation
    raw_data JSONB,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Staging table for extracted entities
CREATE TABLE staging_entities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID REFERENCES staging_messages(id) ON DELETE CASCADE,
    
    entity_type VARCHAR(50) NOT NULL,  -- 'person', 'location', 'date', 'money', 'phone', etc.
    entity_value TEXT NOT NULL,
    normalized_value TEXT,  -- Standardized version
    
    -- Source grounding (character positions)
    start_char INTEGER,
    end_char INTEGER,
    
    -- Confidence and extraction method
    confidence DECIMAL(3,2),
    extraction_method VARCHAR(50),  -- 'llamaextract', 'regex', 'spacy', etc.
    
    is_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Staging table for behavioral patterns
CREATE TABLE staging_behaviors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID REFERENCES staging_messages(id) ON DELETE CASCADE,
    
    -- Behavior classification
    behavior_category VARCHAR(100) NOT NULL,  -- 'gaslighting', 'blame_shifting', 'love_bombing', etc.
    behavior_subcategory VARCHAR(100),
    
    -- Evidence
    matched_pattern TEXT,  -- The regex/pattern that matched
    matched_text TEXT,  -- The actual text that matched
    start_char INTEGER,
    end_char INTEGER,
    
    -- Scoring
    confidence DECIMAL(3,2),
    severity VARCHAR(20),  -- 'low', 'medium', 'high', 'critical'
    
    -- Legal relevance
    mcl_factor VARCHAR(10),  -- MCL 722.23 factor (A-L)
    legal_relevance TEXT,
    
    -- Detection method
    detection_method VARCHAR(50),  -- 'yaml_rules', 'llamaextract', 'manual'
    rule_name VARCHAR(100),  -- Which YAML rule matched
    
    is_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Processing errors log
CREATE TABLE processing_errors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID REFERENCES processing_jobs(id) ON DELETE CASCADE,
    message_id UUID REFERENCES staging_messages(id) ON DELETE SET NULL,
    
    error_type VARCHAR(100),
    error_message TEXT,
    stack_trace TEXT,
    
    -- Context
    source_data JSONB,
    processing_stage VARCHAR(50),  -- 'parsing', 'extraction', 'classification', 'push'
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_staging_messages_job ON staging_messages(job_id);
CREATE INDEX idx_staging_messages_timestamp ON staging_messages(timestamp);
CREATE INDEX idx_staging_messages_processed ON staging_messages(is_processed, is_pushed);
CREATE INDEX idx_staging_entities_message ON staging_entities(message_id);
CREATE INDEX idx_staging_entities_type ON staging_entities(entity_type);
CREATE INDEX idx_staging_behaviors_message ON staging_behaviors(message_id);
CREATE INDEX idx_staging_behaviors_category ON staging_behaviors(behavior_category);
CREATE INDEX idx_staging_behaviors_mcl ON staging_behaviors(mcl_factor);
CREATE INDEX idx_processing_jobs_status ON processing_jobs(status);

-- View: Messages with behavior counts
CREATE VIEW v_messages_with_behaviors AS
SELECT 
    m.id,
    m.timestamp,
    m.sender,
    m.content,
    m.source_type,
    COUNT(DISTINCT b.id) as behavior_count,
    ARRAY_AGG(DISTINCT b.behavior_category) FILTER (WHERE b.behavior_category IS NOT NULL) as behaviors,
    ARRAY_AGG(DISTINCT b.mcl_factor) FILTER (WHERE b.mcl_factor IS NOT NULL) as mcl_factors
FROM staging_messages m
LEFT JOIN staging_behaviors b ON m.id = b.message_id
GROUP BY m.id, m.timestamp, m.sender, m.content, m.source_type;

-- View: Job summary
CREATE VIEW v_job_summary AS
SELECT 
    j.id,
    j.source_file,
    j.source_type,
    j.status,
    j.total_records,
    j.processed_records,
    j.error_count,
    j.started_at,
    j.completed_at,
    EXTRACT(EPOCH FROM (COALESCE(j.completed_at, NOW()) - j.started_at)) as duration_seconds,
    (SELECT COUNT(*) FROM staging_messages WHERE job_id = j.id AND is_pushed = TRUE) as pushed_count,
    (SELECT COUNT(DISTINCT behavior_category) FROM staging_behaviors b 
     JOIN staging_messages m ON b.message_id = m.id WHERE m.job_id = j.id) as unique_behaviors_found
FROM processing_jobs j;

COMMENT ON TABLE staging_messages IS 'Temporary storage for messages during processing pipeline';
COMMENT ON TABLE staging_behaviors IS 'Detected behavioral patterns before verification and push to production';
COMMENT ON TABLE staging_entities IS 'Extracted entities (people, places, dates) for relationship mapping';
