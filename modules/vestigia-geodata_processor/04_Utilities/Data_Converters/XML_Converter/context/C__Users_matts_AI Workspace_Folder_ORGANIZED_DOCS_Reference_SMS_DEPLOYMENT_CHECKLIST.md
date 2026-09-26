# SMS TABLES DEPLOYMENT CHECKLIST
**Date**: December 27, 2025
**Target**: Supabase Project `oflqpddqaecotsdsxbzp`
**Schema**: SMS_TABLES_FINAL.sql (900+ lines)

---

## PRE-DEPLOYMENT

### 1. Verify Supabase Project
- [ ] Confirm project ID: `oflqpddqaecotsdsxbzp`
- [ ] Verify PostgreSQL version: 14+ required
- [ ] Check current database size/quota
- [ ] Backup existing data (if any)

### 2. Review Schema File
- [ ] Open `SMS_TABLES_FINAL.sql` in editor
- [ ] Verify all table names follow convention
- [ ] Check no hardcoded sensitive data
- [ ] Confirm extension requirements:
  - [ ] `uuid-ossp` (UUID generation)
  - [ ] `pg_trgm` (fuzzy text search)
  - [ ] `pg_stat_statements` (query analysis)

### 3. Prepare Test Data
- [ ] Gather sample SMS XML export (10-50 messages)
- [ ] Prepare test phone numbers (fake/anonymized)
- [ ] Create test conversation scenarios:
  - [ ] 1-on-1 conversation
  - [ ] Group chat
  - [ ] MMS with attachments
  - [ ] Messages with behaviors (gaslighting, blame-shifting)

---

## DEPLOYMENT STEPS

### Step 1: Access Supabase SQL Editor
```
1. Go to https://supabase.com/dashboard/project/oflqpddqaecotsdsxbzp
2. Navigate to: SQL Editor (left sidebar)
3. Click: "+ New query"
```

### Step 2: Enable Extensions
```sql
-- Run this first to enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
```
- [ ] Extensions enabled successfully
- [ ] No errors in output panel

### Step 3: Deploy Core Tables (Section by Section)
**IMPORTANT**: Deploy in order to respect foreign key dependencies

#### 3a. Deploy Reference Tables First
```sql
-- From SMS_TABLES_FINAL.sql, copy and run:
-- Line ~40-120: mcl_factors table + inserts
-- Line ~120-180: behavior_categories table + inserts
```
- [ ] `mcl_factors` created (12 rows inserted)
- [ ] `behavior_categories` created (18 rows inserted)
- [ ] Verify: `SELECT COUNT(*) FROM mcl_factors;` returns 12
- [ ] Verify: `SELECT COUNT(*) FROM behavior_categories;` returns 18

#### 3b. Deploy Core Messaging Tables
```sql
-- From SMS_TABLES_FINAL.sql, copy and run:
-- Line ~180-280: messaging_documents
-- Line ~280-380: messaging_conversations
-- Line ~380-520: messaging_messages
-- Line ~520-600: messaging_attachments
```
- [ ] `messaging_documents` created
- [ ] `messaging_conversations` created
- [ ] `messaging_messages` created
- [ ] `messaging_attachments` created
- [ ] Verify: `\d messaging_messages` shows all columns

#### 3c. Deploy Behavioral Analysis Tables
```sql
-- From SMS_TABLES_FINAL.sql, copy and run:
-- Line ~600-680: messaging_behaviors
-- Line ~680-760: messaging_behavior_patterns
```
- [ ] `messaging_behaviors` created
- [ ] `messaging_behavior_patterns` created

#### 3d. Deploy Entity Tables
```sql
-- From SMS_TABLES_FINAL.sql, copy and run:
-- Line ~760-820: messaging_entities
-- Line ~820-870: messaging_entity_mentions
```
- [ ] `messaging_entities` created
- [ ] `messaging_entity_mentions` created

#### 3e. Deploy Evidence Tables
```sql
-- From SMS_TABLES_FINAL.sql, copy and run:
-- Line ~870-920: messaging_evidence_items
-- Line ~920-970: messaging_factor_citations
```
- [ ] `messaging_evidence_items` created
- [ ] `messaging_factor_citations` created

#### 3f. Deploy Timeline Integration
```sql
-- From SMS_TABLES_FINAL.sql, copy and run:
-- Line ~970-1020: messaging_timeline_events
```
- [ ] `messaging_timeline_events` created

### Step 4: Deploy Indexes
```sql
-- From SMS_TABLES_FINAL.sql, copy and run:
-- Line ~1020-1150: All CREATE INDEX statements
```
- [ ] All indexes created (40+ indexes)
- [ ] No errors (ignore warnings about already-indexed PKs)
- [ ] Verify: Check "Index" tab in table browser

### Step 5: Deploy Triggers
```sql
-- From SMS_TABLES_FINAL.sql, copy and run:
-- Line ~1150-1300: Trigger functions + CREATE TRIGGER statements
```
- [ ] All trigger functions created (4 functions)
- [ ] All triggers created (4 triggers)
- [ ] Verify: `\df` shows functions
- [ ] Verify: `SELECT * FROM pg_trigger;` shows triggers

### Step 6: Deploy Views
```sql
-- From SMS_TABLES_FINAL.sql, copy and run:
-- Line ~1300-1450: All CREATE VIEW statements
```
- [ ] `v_messaging_analyzed` created
- [ ] `v_messaging_evidence_by_factor` created
- [ ] `v_messaging_daily_behavior_summary` created
- [ ] `v_messaging_conversation_summary` created
- [ ] Verify: Can query views without errors

### Step 7: Deploy Functions
```sql
-- From SMS_TABLES_FINAL.sql, copy and run:
-- Line ~1450-1500: Forensic integrity functions
```
- [ ] `generate_content_hash()` created
- [ ] `validate_message_chain()` created
- [ ] Verify: `SELECT generate_content_hash('test');` returns hash

---

## POST-DEPLOYMENT VALIDATION

### 1. Table Count Verification
```sql
SELECT COUNT(*) AS table_count
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name LIKE 'messaging_%' OR table_name LIKE 'mcl_%' OR table_name = 'behavior_categories';
```
**Expected**: 15 tables
- [ ] Count matches expected

### 2. Foreign Key Integrity
```sql
SELECT conname, conrelid::regclass AS table_name, confrelid::regclass AS referenced_table
FROM pg_constraint
WHERE contype = 'f' AND connamespace = 'public'::regnamespace
  AND (conrelid::regclass::text LIKE 'messaging_%' OR confrelid::regclass::text LIKE 'messaging_%');
```
- [ ] All foreign keys visible
- [ ] No orphaned references

### 3. Index Count Verification
```sql
SELECT schemaname, tablename, COUNT(*) AS index_count
FROM pg_indexes
WHERE schemaname = 'public' AND tablename LIKE 'messaging_%'
GROUP BY schemaname, tablename
ORDER BY tablename;
```
**Expected**: 40+ indexes across all tables
- [ ] Index counts look reasonable

### 4. Trigger Verification
```sql
SELECT event_object_table, trigger_name, action_timing, event_manipulation
FROM information_schema.triggers
WHERE event_object_schema = 'public'
  AND event_object_table LIKE 'messaging_%'
ORDER BY event_object_table;
```
**Expected**: 4 triggers
- [ ] All triggers present

---

## FUNCTIONAL TESTING

### Test 1: Insert Document
```sql
INSERT INTO messaging_documents (
    filename, file_type, file_hash, file_size, storage_path, 
    source_platform, date_range_start, date_range_end
) VALUES (
    'test_sms_export.xml',
    'sms_xml',
    'abc123def456', -- SHA-256 placeholder
    1024000,
    '/uploads/test_sms_export.xml',
    'android',
    '2024-01-01 00:00:00+00',
    '2024-12-31 23:59:59+00'
) RETURNING id, filename, status;
```
- [ ] Document inserted successfully
- [ ] `status` defaults to 'pending'
- [ ] Copy `id` for next test: `________________`

### Test 2: Insert Conversation
```sql
INSERT INTO messaging_conversations (
    document_id, platform, platform_id, participants, 
    participant_count, primary_participant, started_at
) VALUES (
    '[PASTE_DOCUMENT_ID]', -- From Test 1
    'sms',
    'conv_123',
    ARRAY['Matt Salem', 'Test Contact'],
    2,
    'Test Contact',
    '2024-06-01 10:00:00+00'
) RETURNING id, platform, primary_participant;
```
- [ ] Conversation inserted successfully
- [ ] Copy `id` for next test: `________________`

### Test 3: Insert Messages (with trigger test)
```sql
INSERT INTO messaging_messages (
    conversation_id, timestamp, sender, sender_normalized,
    recipient, content, direction, message_type
) VALUES 
(
    '[PASTE_CONVERSATION_ID]', -- From Test 2
    '2024-06-01 10:05:00+00',
    'Matt Salem',
    '+15551234567',
    'Test Contact',
    'This is a test message to verify the system works correctly.',
    'outbound',
    'text'
),
(
    '[PASTE_CONVERSATION_ID]',
    '2024-06-01 10:10:00+00',
    'Test Contact',
    '+15559876543',
    'Matt Salem',
    'I received your message. Everything looks good!',
    'inbound',
    'text'
) RETURNING id, timestamp, direction, content;
```
- [ ] Both messages inserted successfully
- [ ] Auto-generated fields populated (id, created_at, content_lower)
- [ ] Check conversation updated:
```sql
SELECT message_count, last_message_at 
FROM messaging_conversations 
WHERE id = '[PASTE_CONVERSATION_ID]';
```
- [ ] `message_count` = 2
- [ ] `last_message_at` matches latest message timestamp

### Test 4: Insert Behavior (with auto-update trigger)
```sql
INSERT INTO messaging_behaviors (
    message_id, category, matched_text, confidence, severity, detection_method
) VALUES (
    (SELECT id FROM messaging_messages ORDER BY timestamp LIMIT 1),
    'blame_shifting',
    'your message',
    0.75,
    'medium',
    'manual'
) RETURNING id, category, message_id;
```
- [ ] Behavior inserted successfully
- [ ] Check message updated:
```sql
SELECT has_behaviors, behavior_count, behavior_categories, max_severity
FROM messaging_messages
WHERE id = (SELECT message_id FROM messaging_behaviors LIMIT 1);
```
- [ ] `has_behaviors` = TRUE
- [ ] `behavior_count` = 1
- [ ] `behavior_categories` contains 'blame_shifting'
- [ ] `max_severity` = 'medium'

### Test 5: Full-Text Search
```sql
-- Search messages by content
SELECT id, content, timestamp, sender
FROM messaging_messages
WHERE content_lower LIKE '%test%'
ORDER BY timestamp DESC;
```
- [ ] Returns test messages
- [ ] Search is case-insensitive

```sql
-- Fuzzy search using pg_trgm
SELECT id, content, similarity(content, 'sistem') AS similarity_score
FROM messaging_messages
WHERE similarity(content, 'sistem') > 0.3
ORDER BY similarity_score DESC;
```
- [ ] Returns messages with similar words (e.g., "system")

### Test 6: View Verification
```sql
-- Check analyzed messages view
SELECT * FROM v_messaging_analyzed
WHERE has_behaviors = TRUE;
```
- [ ] Returns message with behavior
- [ ] Shows detected_behaviors array
- [ ] Shows related_mcl_factors array

```sql
-- Check daily summary view
SELECT * FROM v_messaging_daily_behavior_summary
WHERE date = CURRENT_DATE;
```
- [ ] Returns today's summary (may be empty if no messages today)

### Test 7: Chain Integrity Function
```sql
-- Test message chain validation
SELECT * FROM validate_message_chain(
    (SELECT id FROM messaging_conversations LIMIT 1)
);
```
- [ ] Function executes without errors
- [ ] Returns is_valid, broken_links, message_count

### Test 8: Evidence Linking
```sql
-- Create evidence item
INSERT INTO messaging_evidence_items (
    message_id, title, quote, evidence_date, mcl_factors, relevance_score
) VALUES (
    (SELECT id FROM messaging_messages LIMIT 1),
    'Test Evidence - Blame Shifting Pattern',
    'your message',
    '2024-06-01 10:05:00+00',
    ARRAY['F', 'J'],
    0.85
) RETURNING id, title;
```
- [ ] Evidence item created
- [ ] Copy `id` for next test: `________________`

```sql
-- Link evidence to MCL factors
INSERT INTO messaging_factor_citations (
    evidence_id, factor_id, supports_factor, strength, explanation
) VALUES
(
    '[PASTE_EVIDENCE_ID]',
    'F',
    TRUE,
    'moderate',
    'Demonstrates pattern of shifting blame to other parent'
),
(
    '[PASTE_EVIDENCE_ID]',
    'J',
    FALSE,
    'weak',
    'Indicates unwillingness to facilitate relationship'
) RETURNING evidence_id, factor_id, strength;
```
- [ ] Factor citations created

```sql
-- Verify evidence by factor view
SELECT * FROM v_messaging_evidence_by_factor
WHERE evidence_id = '[PASTE_EVIDENCE_ID]';
```
- [ ] Shows both factor citations
- [ ] Displays factor names correctly

---

## CLEANUP TEST DATA

```sql
-- Remove all test data (cascades due to ON DELETE CASCADE)
DELETE FROM messaging_documents WHERE filename = 'test_sms_export.xml';
```
- [ ] All test data deleted
- [ ] Verify tables are empty:
```sql
SELECT 
    (SELECT COUNT(*) FROM messaging_documents) AS docs,
    (SELECT COUNT(*) FROM messaging_conversations) AS convos,
    (SELECT COUNT(*) FROM messaging_messages) AS messages,
    (SELECT COUNT(*) FROM messaging_behaviors) AS behaviors,
    (SELECT COUNT(*) FROM messaging_evidence_items) AS evidence;
```
- [ ] All counts = 0

---

## PERFORMANCE BASELINE

### 1. Table Size Check
```sql
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) AS table_size,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) - pg_relation_size(schemaname||'.'||tablename)) AS index_size
FROM pg_tables
WHERE schemaname = 'public'
  AND tablename LIKE 'messaging_%'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```
- [ ] Record baseline sizes (should be minimal with empty tables)

### 2. Query Plan Analysis
```sql
EXPLAIN ANALYZE
SELECT *
FROM messaging_messages m
JOIN messaging_conversations c ON m.conversation_id = c.id
WHERE c.platform = 'sms'
  AND m.timestamp >= CURRENT_DATE - INTERVAL '30 days'
ORDER BY m.timestamp DESC
LIMIT 100;
```
- [ ] Query plan looks reasonable
- [ ] Uses appropriate indexes
- [ ] No sequential scans on large tables (when data present)

---

## SECURITY REVIEW

### 1. RLS Policies (Optional - Enable Later)
Currently all RLS policies are commented out in schema.
- [ ] Decide if multi-user access is needed
- [ ] If yes, uncomment and customize RLS policies
- [ ] Test with different user roles

### 2. Sensitive Data Audit
```sql
-- Check for any exposed PII in test data
SELECT id, sender, recipient, content
FROM messaging_messages
WHERE content ILIKE '%[0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9][0-9][0-9]%' -- SSN pattern
   OR content ILIKE '%password%'
LIMIT 10;
```
- [ ] No sensitive data found
- [ ] Production: Enable encryption for PII fields if needed

---

## INTEGRATION READINESS

### 1. Neo4j Aura Connection
- [ ] Create Neo4j user credentials in Supabase secrets
- [ ] Test connection from Supabase Functions
- [ ] Verify entity sync works

### 2. Qdrant Cloud Connection
- [ ] Create Qdrant API key in Supabase secrets
- [ ] Test vector embedding generation
- [ ] Verify similarity search works

### 3. Cloudflare R2 Storage
- [ ] Create R2 bucket: `salem-case-attachments`
- [ ] Generate R2 access keys
- [ ] Store credentials in Supabase secrets
- [ ] Test file upload/download

---

## DOCUMENTATION UPDATES

- [ ] Update README.md with deployment date
- [ ] Document any schema modifications made during deployment
- [ ] Record Supabase project settings (region, plan, etc.)
- [ ] Add connection strings to secure password manager
- [ ] Update MASTER_TIMELINE_CONSOLIDATED.md with new tables

---

## SIGN-OFF

**Deployed By**: ________________
**Date**: ________________
**Supabase Project**: oflqpddqaecotsdsxbzp
**Schema Version**: SMS_TABLES_FINAL.sql (Dec 27, 2025)
**Status**: 
- [ ] ✅ PRODUCTION READY
- [ ] ⚠️ NEEDS FIXES (list below)
- [ ] ❌ DEPLOYMENT FAILED (list issues below)

**Notes**:
_______________________________________________________________
_______________________________________________________________
_______________________________________________________________

---

## ROLLBACK PROCEDURE (if needed)

### Emergency Rollback
```sql
-- Drop all messaging tables (nuclear option)
DROP TABLE IF EXISTS messaging_timeline_events CASCADE;
DROP TABLE IF EXISTS messaging_factor_citations CASCADE;
DROP TABLE IF EXISTS messaging_evidence_items CASCADE;
DROP TABLE IF EXISTS messaging_entity_mentions CASCADE;
DROP TABLE IF EXISTS messaging_entities CASCADE;
DROP TABLE IF EXISTS messaging_behavior_patterns CASCADE;
DROP TABLE IF EXISTS messaging_behaviors CASCADE;
DROP TABLE IF EXISTS messaging_attachments CASCADE;
DROP TABLE IF EXISTS messaging_messages CASCADE;
DROP TABLE IF EXISTS messaging_conversations CASCADE;
DROP TABLE IF EXISTS messaging_documents CASCADE;
DROP TABLE IF EXISTS behavior_categories CASCADE;
DROP TABLE IF EXISTS mcl_factors CASCADE;

-- Drop views
DROP VIEW IF EXISTS v_messaging_conversation_summary;
DROP VIEW IF EXISTS v_messaging_daily_behavior_summary;
DROP VIEW IF EXISTS v_messaging_evidence_by_factor;
DROP VIEW IF EXISTS v_messaging_analyzed;

-- Drop functions
DROP FUNCTION IF EXISTS validate_message_chain;
DROP FUNCTION IF EXISTS generate_content_hash;
DROP FUNCTION IF EXISTS populate_behavior_mcl_factors;
DROP FUNCTION IF EXISTS update_document_message_count;
DROP FUNCTION IF EXISTS update_conversation_message_count;
DROP FUNCTION IF EXISTS update_message_behavior_counts;
```
- [ ] Rollback completed
- [ ] Database restored to pre-deployment state

**Rollback Reason**: _______________________________________________

---

**Checklist Version**: 1.0
**Last Updated**: December 27, 2025
