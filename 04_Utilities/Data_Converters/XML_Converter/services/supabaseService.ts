
import { createClient, SupabaseClient } from '@supabase/supabase-js';

export class SupabaseService {
    private client: SupabaseClient | null = null;
    private table: string = 'messages';
    private entityTable: string = 'entities';
    private attachmentTable: string = 'attachments';
    private callsTable: string = 'calls';
    private projectRef: string = '';

    constructor(url?: string, key?: string, table?: string, entityTable?: string, attachmentTable?: string, callsTable?: string) {
        if (url && key) {
            this.client = createClient(url, key, {
                auth: { persistSession: false },
                db: { schema: 'public' }
            });
            this.table = table || 'messages';
            this.entityTable = entityTable || 'entities';
            this.attachmentTable = attachmentTable || 'attachments';
            this.callsTable = callsTable || 'calls';

            // Extract project ref from URL (https://<ref>.supabase.co)
            const match = url.match(/https:\/\/([^.]+)\./);
            if (match) this.projectRef = match[1];
        }
    }

    isConnected(): boolean {
        return !!this.client;
    }

    async getTableData(tableName: string, page: number = 0, pageSize: number = 50) {
        if (!this.client) return { data: [], error: 'Not connected' };
        
        const from = page * pageSize;
        const to = from + pageSize - 1;

        // Dynamic sorting based on table type
        const sortCol = tableName === this.table ? 'date_iso' : 'message_date';

        return await this.client
            .from(tableName)
            .select('*')
            .order(sortCol, { ascending: false })
            .range(from, to);
    }

    async getStorageFileUrl(bucket: string, path: string) {
        if (!this.client) return null;
        const { data } = this.client.storage.from(bucket).getPublicUrl(path);
        return data.publicUrl;
    }

    async deploySchema(pat: string, columns: string[]): Promise<{ success: boolean; message: string }> {
        if (!this.projectRef) return { success: false, message: 'Could not determine Project Ref from URL. Ensure URL format is https://<ref>.supabase.co' };

        // Basic validation: PATs usually start with sbp_ or similar, JWTs start with eyJ
        if (pat.startsWith('eyJ')) {
            return { success: false, message: 'Invalid Token Type: You provided a Project API Key (JWT). This operation requires a Personal Access Token (starts with "sbp_") from your Account Settings.' };
        }

        const sql = this.generateFullSchemaSQL(columns);

        try {
            const response = await fetch(`https://api.supabase.com/v1/projects/${this.projectRef}/query`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${pat}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ query: sql })
            });

            if (!response.ok) {
                let errDetail = response.statusText;
                try {
                    const errJson = await response.json();
                    errDetail = errJson.message || errJson.error || JSON.stringify(errJson);
                } catch (e) { /* ignore json parse error */ }

                if (response.status === 401) {
                    return { success: false, message: `Unauthorized (401): Your Personal Access Token is invalid or expired. Detail: ${errDetail}` };
                }
                if (response.status === 404) {
                    return { success: false, message: `Project Not Found (404): Check if your Project URL is correct. Ref: ${this.projectRef}` };
                }
                return { success: false, message: `Deployment Failed (${response.status}): ${errDetail}` };
            }

            return { success: true, message: 'Schema & Functions deployed successfully!' };

        } catch (e: any) {
            return { success: false, message: `Network/CORS Error: ${e.message}. (Note: The Management API might be blocked by browser privacy extensions)` };
        }
    }

    async createBucket(bucketName: string): Promise<{ success: boolean; message: string }> {
        if (!this.client) return { success: false, message: 'Not connected' };
        
        try {
            const { data, error } = await this.client.storage.createBucket(bucketName, {
                public: false, 
                fileSizeLimit: 52428800, // 50MB
            });
            
            if (error && error.message.includes('already exists')) {
                 return { success: true, message: `Bucket '${bucketName}' ready.` };
            }
            if (error) return { success: false, message: error.message };
            return { success: true, message: `Bucket '${bucketName}' created.` };
        } catch (e: any) {
             return { success: false, message: e.message };
        }
    }

    async uploadFileToStorage(bucket: string, path: string, fileData: Uint8Array, contentType: string): Promise<string | null> {
        if (!this.client) return null;
        
        const { data, error } = await this.client.storage
            .from(bucket)
            .upload(path, fileData, {
                contentType: contentType,
                upsert: true
            });
        
        if (error) {
            console.error("Storage upload failed:", error);
            return null;
        }
        return data?.path || null;
    }

    async testConnection(): Promise<{ success: boolean; message: string }> {
        if (!this.client) return { success: false, message: 'Supabase client not initialized' };
        try {
            const { error } = await this.client.from(this.table).select('uuid_v7').limit(1);
            if (error) {
                if (error.code === '42P01') {
                    return { success: false, message: `Table "${this.table}" does not exist. Deploy Schema first.` };
                }
                return { success: false, message: `Connection failed: ${error.message}` };
            }
            return { success: true, message: `Connected to "${this.table}"` };
        } catch (err: any) {
            return { success: false, message: `Client Error: ${err.message}` };
        }
    }

    /**
     * Create a document record in messaging_documents table
     * Establishes chain of custody for forensic evidence
     */
    async createDocument(documentData: {
        id: string;
        filename: string;
        file_hash: string;
        acquired_by?: string;
        acquired_date?: string;
        acquisition_method?: string;
        source_label?: string;
        file_size_bytes?: number;
        notes?: string;
    }): Promise<string | null> {
        if (!this.client) return null;

        try {
            const { data, error } = await this.client
                .from('messaging_documents')
                .insert({
                    id: documentData.id,
                    filename: documentData.filename,
                    file_hash: documentData.file_hash,
                    acquired_by: documentData.acquired_by || 'Device Owner',
                    acquired_date: documentData.acquired_date || new Date().toISOString(),
                    acquisition_method: documentData.acquisition_method || 'XML Export',
                    source_label: documentData.source_label,
                    file_size_bytes: documentData.file_size_bytes,
                    notes: documentData.notes
                })
                .select('id')
                .single();

            if (error) {
                console.error('Document creation failed:', error);
                return null;
            }

            return data?.id || documentData.id;
        } catch (err) {
            console.error('Document creation exception:', err);
            return null;
        }
    }

    /**
     * Create or find existing conversation record
     * Groups messages by participants and platform
     */
    async createConversation(conversationData: {
        document_id: string;
        platform: string;
        participants: string[];
        thread_id?: string;
        start_date?: string;
    }): Promise<string | null> {
        if (!this.client) return null;

        try {
            // Check if conversation already exists
            const { data: existing, error: findError } = await this.client
                .from('messaging_conversations')
                .select('id')
                .eq('document_id', conversationData.document_id)
                .eq('platform', conversationData.platform)
                .contains('participants', conversationData.participants)
                .limit(1)
                .single();

            if (existing && !findError) {
                return existing.id;
            }

            // Create new conversation
            const { data, error } = await this.client
                .from('messaging_conversations')
                .insert({
                    document_id: conversationData.document_id,
                    platform: conversationData.platform,
                    participants: conversationData.participants,
                    thread_id: conversationData.thread_id,
                    start_date: conversationData.start_date,
                    message_count: 0,
                    behavior_count: 0
                })
                .select('id')
                .single();

            if (error) {
                console.error('Conversation creation failed:', error);
                return null;
            }

            return data?.id || null;
        } catch (err) {
            console.error('Conversation creation exception:', err);
            return null;
        }
    }

    async uploadBatch(
        records: any[],
        documentId: string,
        conversationId: string
    ): Promise<{ error: any; count: number }> {
        if (!this.client) return { error: 'Not initialized', count: 0 };

        // Separate records by type
        const messages: any[] = [];
        const calls: any[] = [];
        const attachments: any[] = [];
        const behaviors: any[] = [];
        const entities: any[] = [];

        for (const record of records) {
            // Route to calls table if it's a call log
            if (record.type === 'call' || record.call_type) {
                calls.push({
                    id: record.uuid_v7,
                    conversation_id: conversationId,
                    document_id: documentId,
                    phone_number: record.address || record.phone_number,
                    call_type: record.call_type || (record.type === '1' ? 'incoming' : 'outgoing'),
                    timestamp: record.date_iso || record.date,
                    duration: record.duration ? parseInt(record.duration) : null,
                    contact_name: record.contact_name,
                    notes: record.body // Some call logs have notes
                });
            } else {
                // Route to messages table
                messages.push({
                    id: record.uuid_v7,
                    conversation_id: conversationId,
                    document_id: documentId,
                    timestamp: record.date_iso || record.date,
                    sender: record.address,
                    recipient: record.type === '1' ? 'ME' : record.address,
                    message_type: record.type === '1' ? 'received' : 'sent',
                    content: record.body,
                    content_hash: record.record_hash,
                    platform: record.platform || 'sms',

                    // NLP markers (if available)
                    contains_apology: record.nlp_markers?.contains_apology || false,
                    contains_blame: record.nlp_markers?.contains_blame || false,
                    contains_threat: record.nlp_markers?.contains_threat || false,
                    contains_minimizing: record.nlp_markers?.contains_minimizing || false,
                    question_count: record.nlp_markers?.question_count || 0,
                    exclamation_count: record.nlp_markers?.exclamation_count || 0,
                    caps_ratio: record.nlp_markers?.caps_ratio || 0,
                    word_count: record.word_count || 0,
                    character_count: record.character_count || 0,
                    sentiment: record.nlp_markers?.sentiment || 'neutral',

                    // Metadata
                    contact_name: record.contact_name,
                    read_status: record.read === '1',
                    source_id: record.source_id
                });

                // Extract behaviors if detected
                if (record.behaviors && Array.isArray(record.behaviors)) {
                    record.behaviors.forEach((behavior: any) => {
                        behaviors.push({
                            message_id: record.uuid_v7,
                            category: behavior.category,
                            matched_text: behavior.matched_text, // ACTUAL words
                            matched_pattern: behavior.matched_pattern,
                            start_char: behavior.start_char,
                            end_char: behavior.end_char,
                            context_before: behavior.context_before,
                            context_after: behavior.context_after,
                            confidence: behavior.confidence,
                            severity: behavior.severity,
                            detection_method: behavior.detection_method
                        });
                    });
                }

                // Extract entities if detected
                if (record.entities && Array.isArray(record.entities)) {
                    record.entities.forEach((entity: any) => {
                        entities.push({
                            message_id: record.uuid_v7,
                            entity_type: entity.entity_type,
                            name: entity.name,
                            normalized_name: entity.normalized_name,
                            mention_text: entity.mention_text,
                            start_char: entity.start_char,
                            end_char: entity.end_char,
                            confidence: entity.confidence
                        });
                    });
                }

                // Extract MMS attachments
                if (record._parts && Array.isArray(record._parts)) {
                    record._parts.forEach((part: any, idx: number) => {
                        if (part.ct || part.data) {
                            attachments.push({
                                message_id: record.uuid_v7,
                                content_type: part.ct || 'application/octet-stream',
                                filename: part.name || `attachment_${idx}`,
                                storage_path: part.storage_path || record.image_files,
                                file_size_bytes: part.data ? part.data.length : null,
                                sequence_number: idx
                            });
                        }
                    });
                }
            }
        }

        // Batch insert to forensic tables
        let errorCount = 0;
        let totalInserted = 0;

        // Insert messages
        if (messages.length > 0) {
            const { error, count } = await this.client
                .from('messaging_messages')
                .upsert(messages, {
                    onConflict: 'id',
                    ignoreDuplicates: true
                });
            if (error) {
                console.error('Messages insert error:', error);
                errorCount++;
            } else {
                totalInserted += messages.length;
            }
        }

        // Insert calls
        if (calls.length > 0) {
            const { error } = await this.client
                .from('messaging_calls')
                .upsert(calls, { onConflict: 'id', ignoreDuplicates: true });
            if (error) {
                console.error('Calls insert error:', error);
                errorCount++;
            } else {
                totalInserted += calls.length;
            }
        }

        // Insert attachments
        if (attachments.length > 0) {
            const { error } = await this.client
                .from('messaging_attachments')
                .insert(attachments);
            if (error) console.warn('Attachments insert error:', error);
        }

        // Insert behaviors
        if (behaviors.length > 0) {
            const { error } = await this.client
                .from('messaging_behaviors')
                .insert(behaviors);
            if (error) console.warn('Behaviors insert error:', error);
        }

        // Insert entities
        if (entities.length > 0) {
            const { error } = await this.client
                .from('messaging_entities')
                .insert(entities);
            if (error) console.warn('Entities insert error:', error);
        }

        return {
            error: errorCount > 0 ? `${errorCount} table insert failures` : null,
            count: totalInserted
        };
    }

    generateFullSchemaSQL(columns: string[]): string {
        // Salem v. Kinzel Forensic Evidence Schema
        return `
-- =====================================================
-- SALEM SMS/MESSAGING TABLES - FORENSIC SCHEMA
-- Court-admissible chain of custody and behavioral analysis
-- =====================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- =====================================================
-- SOURCE DOCUMENTS (Chain of Custody)
-- =====================================================

CREATE TABLE IF NOT EXISTS messaging_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- File identification
    filename VARCHAR(500) NOT NULL,
    file_type VARCHAR(50) NOT NULL,
    file_hash VARCHAR(64) NOT NULL UNIQUE,
    file_size BIGINT NOT NULL,

    -- Storage
    storage_path TEXT NOT NULL,
    storage_provider VARCHAR(50),

    -- Processing status
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'verified')),
    processed_at TIMESTAMPTZ,
    processing_duration_ms INTEGER,

    -- Summary
    raw_text TEXT,
    message_count INTEGER DEFAULT 0,
    page_count INTEGER,

    -- Source metadata
    source_device VARCHAR(100),
    source_platform VARCHAR(50),
    export_date TIMESTAMPTZ,
    date_range_start TIMESTAMPTZ,
    date_range_end TIMESTAMPTZ,

    -- Chain of custody
    acquired_by VARCHAR(100),
    acquired_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acquisition_method TEXT,
    verified_by VARCHAR(100),
    verified_date TIMESTAMPTZ,

    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_messaging_documents_hash ON messaging_documents(file_hash);
CREATE INDEX idx_messaging_documents_status ON messaging_documents(status);
CREATE INDEX idx_messaging_documents_platform ON messaging_documents(source_platform);

-- =====================================================
-- CONVERSATIONS
-- =====================================================

CREATE TABLE IF NOT EXISTS messaging_conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID REFERENCES messaging_documents(id) ON DELETE SET NULL,

    -- Platform identification
    platform VARCHAR(50) NOT NULL,
    platform_id VARCHAR(255),
    platform_name VARCHAR(200),

    -- Participants
    participants TEXT[] NOT NULL,
    participant_count INTEGER NOT NULL,
    primary_participant VARCHAR(255),
    primary_participant_normalized VARCHAR(255),

    -- Temporal info
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    last_message_at TIMESTAMPTZ,
    message_count INTEGER DEFAULT 0,

    -- Thread metadata
    is_group BOOLEAN DEFAULT FALSE,
    group_name VARCHAR(500),

    -- Analysis summary
    behavior_summary JSONB DEFAULT '{}'::JSONB,
    total_words INTEGER DEFAULT 0,
    avg_message_length INTEGER,

    -- Court relevance
    is_evidence BOOLEAN DEFAULT FALSE,
    exhibit_number VARCHAR(50),
    relevance_score DECIMAL(3,2),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(platform, platform_id)
);

CREATE INDEX idx_messaging_conversations_platform ON messaging_conversations(platform);
CREATE INDEX idx_messaging_conversations_participant ON messaging_conversations(primary_participant);
CREATE INDEX idx_messaging_conversations_dates ON messaging_conversations(started_at, ended_at);
CREATE INDEX idx_messaging_conversations_evidence ON messaging_conversations(is_evidence) WHERE is_evidence = TRUE;

-- =====================================================
-- MESSAGES (Core Forensic Record)
-- =====================================================

CREATE TABLE IF NOT EXISTS messaging_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID REFERENCES messaging_conversations(id) ON DELETE CASCADE,
    document_id UUID REFERENCES messaging_documents(id) ON DELETE SET NULL,

    -- Message identification
    external_id VARCHAR(200),
    serial_number INTEGER,

    -- Temporal data
    timestamp TIMESTAMPTZ NOT NULL,
    timestamp_precision VARCHAR(20) DEFAULT 'exact',
    timezone VARCHAR(50),
    date_us TEXT,
    time_12h TEXT,

    -- Sender/Recipient
    sender VARCHAR(255) NOT NULL,
    sender_normalized VARCHAR(255),
    sender_name VARCHAR(255),
    recipient VARCHAR(255),
    recipient_normalized VARCHAR(255),
    recipient_name VARCHAR(255),

    -- Content
    content TEXT,
    content_lower TEXT GENERATED ALWAYS AS (LOWER(content)) STORED,
    content_hash VARCHAR(64),
    word_count INTEGER,
    character_count INTEGER,

    -- Message metadata
    direction VARCHAR(10) CHECK (direction IN ('inbound', 'outbound', 'unknown')),
    message_type VARCHAR(20) DEFAULT 'text',
    status VARCHAR(20),
    is_read BOOLEAN,
    read_timestamp TIMESTAMPTZ,

    -- Attachments
    has_attachments BOOLEAN DEFAULT FALSE,
    attachment_count INTEGER DEFAULT 0,
    attachment_types TEXT[],

    -- Context preservation
    previous_message_id UUID,
    next_message_id UUID,
    time_since_previous_seconds INTEGER,

    -- Analysis flags
    has_behaviors BOOLEAN DEFAULT FALSE,
    behavior_count INTEGER DEFAULT 0,
    behavior_categories TEXT[],
    max_severity VARCHAR(20),

    -- Linguistic markers
    contains_apology BOOLEAN DEFAULT FALSE,
    contains_blame BOOLEAN DEFAULT FALSE,
    contains_threat BOOLEAN DEFAULT FALSE,
    contains_minimizing BOOLEAN DEFAULT FALSE,
    question_count INTEGER DEFAULT 0,
    exclamation_count INTEGER DEFAULT 0,
    caps_ratio DECIMAL(3,2),

    -- Court metadata
    is_evidence BOOLEAN DEFAULT FALSE,
    evidence_item_id UUID,
    is_redacted BOOLEAN DEFAULT FALSE,
    redaction_reason TEXT,

    -- Raw preservation
    raw_data JSONB,

    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(conversation_id, external_id)
);

CREATE INDEX idx_messaging_messages_conversation ON messaging_messages(conversation_id);
CREATE INDEX idx_messaging_messages_timestamp ON messaging_messages(timestamp DESC);
CREATE INDEX idx_messaging_messages_sender ON messaging_messages(sender_normalized);
CREATE INDEX idx_messaging_messages_recipient ON messaging_messages(recipient_normalized);
CREATE INDEX idx_messaging_messages_content_search ON messaging_messages USING gin(content_lower gin_trgm_ops);
CREATE INDEX idx_messaging_messages_has_behaviors ON messaging_messages(has_behaviors) WHERE has_behaviors = TRUE;
CREATE INDEX idx_messaging_messages_is_evidence ON messaging_messages(is_evidence) WHERE is_evidence = TRUE;
CREATE INDEX idx_messaging_messages_direction ON messaging_messages(direction);
CREATE INDEX idx_messaging_messages_conv_timestamp ON messaging_messages(conversation_id, timestamp DESC);
CREATE INDEX idx_messaging_messages_sender_timestamp ON messaging_messages(sender_normalized, timestamp DESC);

-- =====================================================
-- ATTACHMENTS
-- =====================================================

CREATE TABLE IF NOT EXISTS messaging_attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messaging_messages(id) ON DELETE CASCADE,

    filename VARCHAR(500),
    file_type VARCHAR(50) NOT NULL,
    mime_type VARCHAR(100),
    file_hash VARCHAR(64),
    file_size BIGINT,
    part_index INTEGER,

    storage_provider VARCHAR(50),
    storage_path TEXT,
    storage_url TEXT,

    -- Content (for text attachments)
    text_content TEXT,
    content_id VARCHAR(200),

    -- Location data (for location shares)
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    location_name TEXT,

    -- Metadata
    width INTEGER,
    height INTEGER,
    duration_seconds INTEGER,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_messaging_attachments_message ON messaging_attachments(message_id);
CREATE INDEX idx_messaging_attachments_type ON messaging_attachments(file_type);

-- =====================================================
-- CALL LOGS (Forensic Call Records)
-- =====================================================

CREATE TABLE IF NOT EXISTS messaging_calls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID REFERENCES messaging_documents(id) ON DELETE SET NULL,

    -- Call identification
    external_id VARCHAR(200),

    -- Temporal data
    timestamp TIMESTAMPTZ NOT NULL,
    timestamp_precision VARCHAR(20) DEFAULT 'exact',
    timezone VARCHAR(50),
    date_us TEXT,
    time_12h TEXT,
    duration INTEGER,

    -- Participants
    phone_number VARCHAR(255) NOT NULL,
    phone_number_normalized VARCHAR(255),
    contact_name VARCHAR(255),

    -- Call metadata
    call_type VARCHAR(20) NOT NULL CHECK (call_type IN ('incoming', 'outgoing', 'missed', 'voicemail', 'rejected', 'blocked')),
    presentation VARCHAR(20) CHECK (presentation IN ('allowed', 'restricted', 'unknown', 'payphone')),
    subscription_id VARCHAR(50),

    -- Analysis flags
    is_suspicious BOOLEAN DEFAULT FALSE,
    is_pattern BOOLEAN DEFAULT FALSE,
    pattern_notes TEXT,

    -- Court metadata
    is_evidence BOOLEAN DEFAULT FALSE,
    evidence_item_id UUID,

    -- Raw preservation
    raw_data JSONB,
    content_hash VARCHAR(64),

    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(document_id, external_id)
);

CREATE INDEX idx_messaging_calls_timestamp ON messaging_calls(timestamp DESC);
CREATE INDEX idx_messaging_calls_phone ON messaging_calls(phone_number_normalized);
CREATE INDEX idx_messaging_calls_type ON messaging_calls(call_type);
CREATE INDEX idx_messaging_calls_document ON messaging_calls(document_id);
CREATE INDEX idx_messaging_calls_evidence ON messaging_calls(is_evidence) WHERE is_evidence = TRUE;
CREATE INDEX idx_messaging_calls_suspicious ON messaging_calls(is_suspicious) WHERE is_suspicious = TRUE;

-- Auto-hash call content
CREATE OR REPLACE FUNCTION generate_call_hash()
RETURNS TRIGGER AS $$
BEGIN
    NEW.content_hash = encode(digest(
        COALESCE(NEW.phone_number, '') ||
        COALESCE(NEW.timestamp::text, '') ||
        COALESCE(NEW.call_type, '') ||
        COALESCE(NEW.duration::text, ''),
        'sha256'
    ), 'hex');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_messaging_call_hash
BEFORE INSERT OR UPDATE ON messaging_calls
FOR EACH ROW
EXECUTE FUNCTION generate_call_hash();

-- =====================================================
-- BEHAVIORAL ANALYSIS
-- =====================================================

CREATE TABLE IF NOT EXISTS messaging_behavior_categories (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    severity_default VARCHAR(20) DEFAULT 'medium',
    mcl_factors VARCHAR(2)[],
    pattern_indicators TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert behavior categories
INSERT INTO messaging_behavior_categories (id, name, mcl_factors, pattern_indicators) VALUES
('gaslighting', 'Gaslighting', ARRAY['F', 'G', 'K'], ARRAY['never happened', 'you''re crazy', 'making things up']),
('blame_shifting', 'Blame Shifting', ARRAY['F', 'J'], ARRAY['your fault', 'you made me', 'because of you']),
('minimizing', 'Minimizing', ARRAY['F', 'K'], ARRAY['not a big deal', 'overreacting', 'too sensitive']),
('love_bombing', 'Love Bombing', ARRAY['F'], ARRAY['can''t live without', 'soulmate', 'meant to be']),
('stonewalling', 'Stonewalling', ARRAY['J'], ARRAY['not talking about', 'done discussing']),
('parental_alienation', 'Parental Alienation', ARRAY['J', 'K'], ARRAY['doesn''t care about you', 'bad parent']),
('coercive_control', 'Coercive Control', ARRAY['F', 'K'], ARRAY['have to', 'must', 'or else']),
('financial_abuse', 'Financial Abuse', ARRAY['C', 'F', 'K'], ARRAY['your money', 'can''t afford']),
('darvo', 'DARVO Pattern', ARRAY['F', 'K'], ARRAY['i''m the victim', 'you''re abusive']),
('character_assassination', 'Character Assassination', ARRAY['F', 'J'], ARRAY['crazy', 'unstable', 'unfit'])
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS messaging_behaviors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messaging_messages(id) ON DELETE CASCADE,

    category VARCHAR(50) NOT NULL REFERENCES messaging_behavior_categories(id),
    subcategory VARCHAR(100),

    matched_pattern TEXT,
    matched_text TEXT,
    start_char INTEGER,
    end_char INTEGER,
    context_before TEXT,
    context_after TEXT,

    confidence DECIMAL(3,2) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    severity VARCHAR(20) NOT NULL,

    detection_method VARCHAR(50) NOT NULL,
    rule_name VARCHAR(100),
    model_version VARCHAR(50),

    is_verified BOOLEAN DEFAULT FALSE,
    verified_by VARCHAR(100),
    verified_at TIMESTAMPTZ,
    is_false_positive BOOLEAN DEFAULT FALSE,

    related_message_ids UUID[],
    pattern_frequency INTEGER,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_messaging_behaviors_message ON messaging_behaviors(message_id);
CREATE INDEX idx_messaging_behaviors_category ON messaging_behaviors(category);
CREATE INDEX idx_messaging_behaviors_severity ON messaging_behaviors(severity);

-- =====================================================
-- MCL 722.23 FACTORS
-- =====================================================

CREATE TABLE IF NOT EXISTS mcl_factors (
    id VARCHAR(2) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    statutory_text TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO mcl_factors (id, name, description) VALUES
('A', 'Love and Affection', 'Emotional ties between parties and child'),
('B', 'Capacity to Provide', 'Capacity to give love, affection, and guidance'),
('C', 'Capacity for Necessities', 'Capacity to provide food, clothing, medical care'),
('D', 'Home Environment', 'Stability and continuity of environment'),
('E', 'Permanence of Family Unit', 'Permanence of existing or proposed custodial home'),
('F', 'Moral Fitness', 'Moral fitness of the parties involved'),
('G', 'Mental and Physical Health', 'Mental and physical health of parties'),
('H', 'Home, School, Community Record', 'Record of child in home, school, community'),
('I', 'Child Preference', 'Reasonable preference of child if sufficient age'),
('J', 'Willingness to Facilitate', 'Willingness to facilitate parent-child relationship'),
('K', 'Domestic Violence', 'Domestic violence directed at or witnessed by child'),
('L', 'Other Factors', 'Any other factor relevant to custody dispute')
ON CONFLICT (id) DO NOTHING;

-- =====================================================
-- AUDIT LOG
-- =====================================================

CREATE TABLE IF NOT EXISTS messaging_audit_log (
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

-- =====================================================
-- TRIGGERS
-- =====================================================

-- Auto-hash content
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

-- Update message behavior counts
CREATE OR REPLACE FUNCTION update_message_behavior_counts()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messaging_messages SET
        has_behaviors = TRUE,
        behavior_count = (SELECT COUNT(*) FROM messaging_behaviors WHERE message_id = NEW.message_id AND is_false_positive = FALSE),
        max_severity = (
            SELECT severity FROM messaging_behaviors
            WHERE message_id = NEW.message_id AND is_false_positive = FALSE
            ORDER BY CASE severity WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 END
            LIMIT 1
        ),
        behavior_categories = (SELECT ARRAY_AGG(DISTINCT category) FROM messaging_behaviors WHERE message_id = NEW.message_id AND is_false_positive = FALSE)
    WHERE id = NEW.message_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_messaging_behavior_insert
AFTER INSERT OR UPDATE ON messaging_behaviors
FOR EACH ROW
EXECUTE FUNCTION update_message_behavior_counts();

-- Update conversation counts
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

-- =====================================================
-- SCHEMA DEPLOYMENT COMPLETE
-- =====================================================
`;
    }
}
