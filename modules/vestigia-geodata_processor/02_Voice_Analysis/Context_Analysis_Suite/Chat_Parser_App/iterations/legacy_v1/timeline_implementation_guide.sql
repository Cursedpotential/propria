-- ============================================================================
-- TIMELINE INGESTION SYSTEM - IMPLEMENTATION GUIDE
-- ============================================================================
-- Sample data, common queries, and usage examples
-- ============================================================================

-- ============================================================================
-- SECTION 1: SAMPLE DATA INSERTION
-- ============================================================================

-- Insert sample sources
INSERT INTO sources (source_id, source_name, source_type, source_metadata, sync_frequency_minutes)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'Slack Workspace', 'chat', '{"workspace_id": "T12345", "api_endpoint": "https://slack.com/api"}'::jsonb, 15),
    ('22222222-2222-2222-2222-222222222222', 'Discord Server', 'chat', '{"guild_id": "987654321", "bot_token_ref": "vault:discord_bot"}'::jsonb, 10),
    ('33333333-3333-3333-3333-333333333333', 'Google Calendar', 'calendar', '{"calendar_id": "primary", "oauth_ref": "vault:google_oauth"}'::jsonb, 30),
    ('44444444-4444-4444-4444-444444444444', 'Manual Entry', 'manual', '{}'::jsonb, NULL),
    ('55555555-5555-5555-5555-555555555555', 'WhatsApp Export', 'chat', '{"export_format": "txt", "language": "en"}'::jsonb, NULL);

-- Insert sample entities (users)
INSERT INTO entities (entity_id, entity_type, external_id, display_name, email, metadata)
VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'user', 'slack:U12345', 'Alice Johnson', 'alice@example.com', '{"timezone": "America/New_York", "avatar_url": "https://..."}'::jsonb),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'user', 'slack:U67890', 'Bob Smith', 'bob@example.com', '{"timezone": "America/Los_Angeles"}'::jsonb),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', 'user', 'discord:123456789', 'Charlie Davis', 'charlie@example.com', '{}'::jsonb),
    ('dddddddd-dddd-dddd-dddd-dddddddddddd', 'user', NULL, 'Diana Martinez', 'diana@example.com', '{"preferred_name": "Di"}'::jsonb),
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 'bot', 'slack:B99999', 'SlackBot', NULL, '{"bot_type": "assistant"}'::jsonb);

-- Insert tags
INSERT INTO tags (tag_id, tag_name, tag_category, tag_color)
VALUES
    ('tag11111-1111-1111-1111-111111111111', 'work', 'context', '#3B82F6'),
    ('tag22222-2222-2222-2222-222222222222', 'personal', 'context', '#10B981'),
    ('tag33333-3333-3333-3333-333333333333', 'urgent', 'priority', '#EF4444'),
    ('tag44444-4444-4444-4444-444444444444', 'meeting', 'type', '#8B5CF6'),
    ('tag55555-5555-5555-5555-555555555555', 'project-alpha', 'project', '#F59E0B'),
    ('tag66666-6666-6666-6666-666666666666', 'question', 'sentiment', '#EC4899'),
    ('tag77777-7777-7777-7777-777777777777', 'decision', 'type', '#6366F1');

-- ============================================================================
-- SECTION 2: CHAT DATA EXAMPLES
-- ============================================================================

-- Insert chat platform
INSERT INTO chat_platforms (platform_id, source_id, platform_name, platform_type, metadata)
VALUES
    ('plat1111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'Slack - Engineering', 'team', '{"workspace_name": "Acme Corp Engineering"}'::jsonb),
    ('plat2222-2222-2222-2222-222222222222', '22222222-2222-2222-2222-222222222222', 'Discord - Game Dev', 'public', '{"server_name": "Game Developers"}'::jsonb);

-- Insert chat channels
INSERT INTO chat_channels (channel_id, platform_id, external_channel_id, channel_name, channel_type, is_private, participant_count)
VALUES
    ('chan1111-1111-1111-1111-111111111111', 'plat1111-1111-1111-1111-111111111111', 'C12345', '#general', 'channel', false, 0),
    ('chan2222-2222-2222-2222-222222222222', 'plat1111-1111-1111-1111-111111111111', 'C67890', '#project-alpha', 'channel', false, 0),
    ('chan3333-3333-3333-3333-333333333333', 'plat1111-1111-1111-1111-111111111111', 'D11111', 'Alice & Bob DM', 'direct', true, 0);

-- Insert chat participants
INSERT INTO chat_participants (channel_id, entity_id, role)
VALUES
    ('chan1111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'member'),
    ('chan1111-1111-1111-1111-111111111111', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'member'),
    ('chan1111-1111-1111-1111-111111111111', 'dddddddd-dddd-dddd-dddd-dddddddddddd', 'admin'),
    ('chan2222-2222-2222-2222-222222222222', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'owner'),
    ('chan2222-2222-2222-2222-222222222222', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'member'),
    ('chan3333-3333-3333-3333-333333333333', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'member'),
    ('chan3333-3333-3333-3333-333333333333', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'member');

-- Insert timeline events (for chat messages)
INSERT INTO timeline_events (event_id, source_id, entity_id, event_timestamp, event_type, event_category, title, description, raw_data, importance_score)
VALUES
    ('msg11111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '2024-01-15 10:30:00+00', 'message', 'chat', 'Project Alpha Kickoff', 'Hey team! Let''s discuss the Project Alpha timeline today.', '{"platform": "slack", "channel": "C67890", "thread_ts": null}'::jsonb, 0.8),
    ('msg22222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '2024-01-15 10:32:00+00', 'message', 'chat', 'Re: Project Alpha Kickoff', 'Sounds good! I can meet at 2pm.', '{"platform": "slack", "channel": "C67890", "thread_ts": "1705315800.123456"}'::jsonb, 0.6),
    ('msg33333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '2024-01-15 10:35:00+00', 'message', 'chat', 'Meeting scheduled', 'Perfect! Created a calendar invite for 2pm.', '{"platform": "slack", "channel": "C67890", "thread_ts": "1705315800.123456"}'::jsonb, 0.7),
    ('msg44444-4444-4444-4444-444444444444', '11111111-1111-1111-1111-111111111111', 'dddddddd-dddd-dddd-dddd-dddddddddddd', '2024-01-15 14:00:00+00', 'message', 'chat', 'Project Alpha Meeting Notes', 'Key decisions from today''s meeting:\n1. Timeline: 3 months\n2. Budget: $50k\n3. Team: Alice (lead), Bob (dev), Diana (PM)', '{"platform": "slack", "channel": "C67890", "has_code_block": true}'::jsonb, 0.9);

-- Insert chat messages (extending timeline events)
INSERT INTO chat_messages (message_id, channel_id, sender_id, parent_message_id, thread_root_id, message_text, external_message_id)
VALUES
    ('msg11111-1111-1111-1111-111111111111', 'chan2222-2222-2222-2222-222222222222', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', NULL, NULL, 'Hey team! Let''s discuss the Project Alpha timeline today.', '1705315800.123456'),
    ('msg22222-2222-2222-2222-222222222222', 'chan2222-2222-2222-2222-222222222222', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'msg11111-1111-1111-1111-111111111111', 'msg11111-1111-1111-1111-111111111111', 'Sounds good! I can meet at 2pm.', '1705315920.234567'),
    ('msg33333-3333-3333-3333-333333333333', 'chan2222-2222-2222-2222-222222222222', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'msg22222-2222-2222-2222-222222222222', 'msg11111-1111-1111-1111-111111111111', 'Perfect! Created a calendar invite for 2pm.', '1705316100.345678'),
    ('msg44444-4444-4444-4444-444444444444', 'chan2222-2222-2222-2222-222222222222', 'dddddddd-dddd-dddd-dddd-dddddddddddd', NULL, NULL, E'Key decisions from today\'s meeting:\n1. Timeline: 3 months\n2. Budget: $50k\n3. Team: Alice (lead), Bob (dev), Diana (PM)', '1705330800.456789');

-- Insert message reactions
INSERT INTO message_reactions (message_id, reactor_id, reaction_type)
VALUES
    ('msg11111-1111-1111-1111-111111111111', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '👍'),
    ('msg11111-1111-1111-1111-111111111111', 'dddddddd-dddd-dddd-dddd-dddddddddddd', '👍'),
    ('msg44444-4444-4444-4444-444444444444', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '📝'),
    ('msg44444-4444-4444-4444-444444444444', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '✅');

-- Insert message mentions
INSERT INTO message_mentions (message_id, mentioned_entity_id, mention_type)
VALUES
    ('msg11111-1111-1111-1111-111111111111', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'direct'),
    ('msg11111-1111-1111-1111-111111111111', 'dddddddd-dddd-dddd-dddd-dddddddddddd', 'direct');

-- Tag some events
INSERT INTO event_tags (event_id, tag_id, confidence_score)
VALUES
    ('msg11111-1111-1111-1111-111111111111', 'tag11111-1111-1111-1111-111111111111', 1.0),  -- work
    ('msg11111-1111-1111-1111-111111111111', 'tag55555-5555-5555-5555-555555555555', 1.0),  -- project-alpha
    ('msg44444-4444-4444-4444-444444444444', 'tag11111-1111-1111-1111-111111111111', 1.0),  -- work
    ('msg44444-4444-4444-4444-444444444444', 'tag55555-5555-5555-5555-555555555555', 1.0),  -- project-alpha
    ('msg44444-4444-4444-4444-444444444444', 'tag77777-7777-7777-7777-777777777777', 0.9);  -- decision

-- ============================================================================
-- SECTION 3: LIFE EVENTS DATA EXAMPLES
-- ============================================================================

-- Insert timeline events for life events
INSERT INTO timeline_events (event_id, source_id, entity_id, event_timestamp, event_type, event_category, title, description, importance_score)
VALUES
    ('life1111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444444', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '2015-05-15 14:00:00+00', 'life_event', 'education', 'Graduated from University', 'Received Bachelor of Science in Computer Science from State University', 0.95),
    ('life2222-2222-2222-2222-222222222222', '44444444-4444-4444-4444-444444444444', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '2016-06-01 09:00:00+00', 'life_event', 'career', 'Started first job at TechCorp', 'Began working as Junior Software Engineer', 0.85),
    ('life3333-3333-3333-3333-333333333333', '44444444-4444-4444-4444-444444444444', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '2018-12-25 00:00:00+00', 'life_event', 'relationship', 'Got engaged', 'Engaged to partner at Christmas dinner', 1.0),
    ('life4444-4444-4444-4444-444444444444', '44444444-4444-4444-4444-444444444444', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '2019-08-10 16:00:00+00', 'life_event', 'relationship', 'Wedding Day', 'Married at Sunset Gardens', 1.0),
    ('life5555-5555-5555-5555-555555555555', '44444444-4444-4444-4444-444444444444', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '2020-03-15 00:00:00+00', 'life_event', 'residence', 'Moved to San Francisco', 'Relocated for new job opportunity', 0.75);

-- Get category IDs (from pre-inserted data)
DO $$
DECLARE
    cat_education UUID;
    cat_career UUID;
    cat_relationship UUID;
    cat_residence UUID;
BEGIN
    SELECT category_id INTO cat_education FROM life_event_categories WHERE category_name = 'Education';
    SELECT category_id INTO cat_career FROM life_event_categories WHERE category_name = 'Career';
    SELECT category_id INTO cat_relationship FROM life_event_categories WHERE category_name = 'Relationships';
    SELECT category_id INTO cat_residence FROM life_event_categories WHERE category_name = 'Residence';

    -- Insert life events
    INSERT INTO life_events (life_event_id, entity_id, category_id, event_title, event_description, date_precision, significance_level, emotional_valence, privacy_level)
    VALUES
        ('life1111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', cat_education, 'Graduated from University', 'Received Bachelor of Science in Computer Science from State University', 'day', 'critical', 'positive', 'public'),
        ('life2222-2222-2222-2222-222222222222', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', cat_career, 'Started first job at TechCorp', 'Began working as Junior Software Engineer', 'day', 'high', 'positive', 'public'),
        ('life3333-3333-3333-3333-333333333333', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', cat_relationship, 'Got engaged', 'Engaged to partner at Christmas dinner', 'day', 'critical', 'positive', 'friends'),
        ('life4444-4444-4444-4444-444444444444', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', cat_relationship, 'Wedding Day', 'Married at Sunset Gardens', 'hour', 'critical', 'positive', 'friends'),
        ('life5555-5555-5555-5555-555555555555', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', cat_residence, 'Moved to San Francisco', 'Relocated for new job opportunity', 'month', 'high', 'neutral', 'public');
END $$;

-- Insert relationships
INSERT INTO relationships (entity_id_1, entity_id_2, relationship_type, relationship_subtype, started_at, strength)
VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'colleague', 'coworker', '2016-06-01'::timestamptz, 0.7),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'dddddddd-dddd-dddd-dddd-dddddddddddd', 'colleague', 'manager', '2020-03-15'::timestamptz, 0.8),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'cccccccc-cccc-cccc-cccc-cccccccccccc', 'friend', NULL, '2010-01-01'::timestamptz, 0.9);

-- Insert locations
INSERT INTO locations (location_id, location_name, location_type, city, state_province, country, latitude, longitude)
VALUES
    ('loc11111-1111-1111-1111-111111111111', 'State University', 'venue', 'College Town', 'NY', 'USA', 42.4440, -76.5019),
    ('loc22222-2222-2222-2222-222222222222', 'Sunset Gardens', 'venue', 'Portland', 'OR', 'USA', 45.5152, -122.6784),
    ('loc33333-3333-3333-3333-333333333333', 'San Francisco Apartment', 'residence', 'San Francisco', 'CA', 'USA', 37.7749, -122.4194);

-- Link locations to life events
INSERT INTO life_event_locations (life_event_id, location_id, location_role)
VALUES
    ('life1111-1111-1111-1111-111111111111', 'loc11111-1111-1111-1111-111111111111', 'primary'),
    ('life4444-4444-4444-4444-444444444444', 'loc22222-2222-2222-2222-222222222222', 'primary'),
    ('life5555-5555-5555-5555-555555555555', 'loc33333-3333-3333-3333-333333333333', 'destination');

-- ============================================================================
-- SECTION 4: COMMON QUERY PATTERNS
-- ============================================================================

-- ====================
-- 4.1 Timeline Queries
-- ====================

-- Get recent timeline events (last 30 days)
SELECT
    te.event_id,
    te.event_type,
    te.title,
    te.event_timestamp,
    s.source_name,
    e.display_name as entity_name,
    te.importance_score
FROM timeline_events te
JOIN sources s ON te.source_id = s.source_id
LEFT JOIN entities e ON te.entity_id = e.entity_id
WHERE te.is_deleted = false
    AND te.event_timestamp > NOW() - INTERVAL '30 days'
ORDER BY te.event_timestamp DESC
LIMIT 50;

-- Get events for a specific entity
SELECT
    te.event_type,
    te.title,
    te.event_timestamp,
    s.source_name
FROM timeline_events te
JOIN sources s ON te.source_id = s.source_id
WHERE te.entity_id = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
    AND te.is_deleted = false
ORDER BY te.event_timestamp DESC;

-- Get events by tag
SELECT
    te.event_id,
    te.title,
    te.event_timestamp,
    t.tag_name,
    et.confidence_score
FROM timeline_events te
JOIN event_tags et ON te.event_id = et.event_id
JOIN tags t ON et.tag_id = t.tag_id
WHERE t.tag_name = 'project-alpha'
    AND te.is_deleted = false
ORDER BY te.event_timestamp DESC;

-- ====================
-- 4.2 Chat Queries
-- ====================

-- Get all messages in a channel
SELECT
    e.display_name as sender,
    cm.message_text,
    te.event_timestamp,
    cm.reaction_count,
    cm.reply_count
FROM chat_messages cm
JOIN entities e ON cm.sender_id = e.entity_id
JOIN timeline_events te ON cm.message_id = te.event_id
WHERE cm.channel_id = 'chan2222-2222-2222-2222-222222222222'
    AND te.is_deleted = false
ORDER BY te.event_timestamp;

-- Get conversation thread
WITH RECURSIVE thread AS (
    -- Root message
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
    WHERE cm.message_id = 'msg11111-1111-1111-1111-111111111111'

    UNION ALL

    -- Replies
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
SELECT
    REPEAT('  ', depth) || display_name as sender,
    message_text,
    event_timestamp,
    depth
FROM thread
ORDER BY event_timestamp;

-- Get most active participants in a channel
SELECT
    e.display_name,
    COUNT(cm.message_id) as message_count,
    MAX(te.event_timestamp) as last_message_at,
    COUNT(DISTINCT DATE(te.event_timestamp)) as active_days
FROM chat_messages cm
JOIN entities e ON cm.sender_id = e.entity_id
JOIN timeline_events te ON cm.message_id = te.event_id
WHERE cm.channel_id = 'chan2222-2222-2222-2222-222222222222'
    AND te.event_timestamp > NOW() - INTERVAL '90 days'
GROUP BY e.entity_id, e.display_name
ORDER BY message_count DESC;

-- Search messages by text
SELECT
    e.display_name,
    cm.message_text,
    te.event_timestamp,
    cc.channel_name,
    ts_rank(cm.message_vector, query) as relevance
FROM chat_messages cm
JOIN entities e ON cm.sender_id = e.entity_id
JOIN timeline_events te ON cm.message_id = te.event_id
JOIN chat_channels cc ON cm.channel_id = cc.channel_id,
     websearch_to_tsquery('english', 'project timeline') as query
WHERE cm.message_vector @@ query
ORDER BY relevance DESC, te.event_timestamp DESC
LIMIT 20;

-- ====================
-- 4.3 Life Event Queries
-- ====================

-- Get life timeline for a person
SELECT
    le.event_title,
    le.event_description,
    lec.category_name,
    te.event_timestamp,
    le.significance_level,
    le.emotional_valence,
    l.location_name
FROM life_events le
JOIN timeline_events te ON le.life_event_id = te.event_id
LEFT JOIN life_event_categories lec ON le.category_id = lec.category_id
LEFT JOIN life_event_locations lel ON le.life_event_id = lel.life_event_id
LEFT JOIN locations l ON lel.location_id = l.location_id
WHERE le.entity_id = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
    AND te.is_deleted = false
ORDER BY te.event_timestamp;

-- Get significant life events (critical importance)
SELECT
    le.event_title,
    le.significance_level,
    le.emotional_valence,
    te.event_timestamp,
    lec.category_name
FROM life_events le
JOIN timeline_events te ON le.life_event_id = te.event_id
JOIN life_event_categories lec ON le.category_id = lec.category_id
WHERE le.entity_id = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
    AND le.significance_level = 'critical'
ORDER BY te.event_timestamp;

-- Get life events by category
SELECT
    le.event_title,
    te.event_timestamp,
    le.emotional_valence
FROM life_events le
JOIN timeline_events te ON le.life_event_id = te.event_id
JOIN life_event_categories lec ON le.category_id = lec.category_id
WHERE lec.category_name = 'Career'
    AND le.entity_id = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
ORDER BY te.event_timestamp;

-- ====================
-- 4.4 Relationship Queries
-- ====================

-- Get all relationships for a person
SELECT
    CASE
        WHEN r.entity_id_1 = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
        THEN e2.display_name
        ELSE e1.display_name
    END as related_person,
    r.relationship_type,
    r.relationship_subtype,
    r.strength,
    r.started_at,
    r.is_current
FROM relationships r
JOIN entities e1 ON r.entity_id_1 = e1.entity_id
JOIN entities e2 ON r.entity_id_2 = e2.entity_id
WHERE (r.entity_id_1 = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
    OR r.entity_id_2 = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')
ORDER BY r.strength DESC, r.started_at DESC;

-- Get relationship network (using the utility function)
SELECT * FROM get_relationship_network('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 2);

-- ====================
-- 4.5 Cross-Component Queries
-- ====================

-- Get unified timeline (chat + life events)
SELECT
    'chat' as source_type,
    te.event_id,
    te.title,
    cm.message_text as content,
    te.event_timestamp,
    e.display_name as person
FROM timeline_events te
JOIN chat_messages cm ON te.event_id = cm.message_id
JOIN entities e ON cm.sender_id = e.entity_id
WHERE te.is_deleted = false
    AND te.event_timestamp > NOW() - INTERVAL '90 days'

UNION ALL

SELECT
    'life_event' as source_type,
    te.event_id,
    te.title,
    le.event_description as content,
    te.event_timestamp,
    e.display_name as person
FROM timeline_events te
JOIN life_events le ON te.event_id = le.life_event_id
JOIN entities e ON le.entity_id = e.entity_id
WHERE te.is_deleted = false
    AND te.event_timestamp > NOW() - INTERVAL '90 days'

ORDER BY event_timestamp DESC;

-- Activity heatmap (events by hour and day of week)
SELECT
    TO_CHAR(event_timestamp, 'Day') as day_of_week,
    EXTRACT(HOUR FROM event_timestamp) as hour_of_day,
    COUNT(*) as event_count,
    event_type
FROM timeline_events
WHERE event_timestamp > NOW() - INTERVAL '90 days'
    AND is_deleted = false
GROUP BY TO_CHAR(event_timestamp, 'Day'), EXTRACT(HOUR FROM event_timestamp), event_type
ORDER BY
    EXTRACT(DOW FROM event_timestamp),
    hour_of_day,
    event_type;

-- ====================
-- 4.6 Analytics Queries
-- ====================

-- Event volume over time
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

-- Most used tags
SELECT
    t.tag_name,
    t.tag_category,
    t.usage_count,
    COUNT(et.event_id) as current_usage
FROM tags t
LEFT JOIN event_tags et ON t.tag_id = et.tag_id
GROUP BY t.tag_id, t.tag_name, t.tag_category, t.usage_count
ORDER BY t.usage_count DESC
LIMIT 20;

-- Source activity summary
SELECT
    s.source_name,
    s.source_type,
    COUNT(te.event_id) as total_events,
    MAX(te.event_timestamp) as latest_event,
    s.last_sync_at
FROM sources s
LEFT JOIN timeline_events te ON s.source_id = te.source_id
WHERE s.is_active = true
GROUP BY s.source_id, s.source_name, s.source_type, s.last_sync_at
ORDER BY total_events DESC;

-- ============================================================================
-- SECTION 5: MAINTENANCE QUERIES
-- ============================================================================

-- Check for orphaned records
SELECT 'chat_messages without timeline_events' as issue, COUNT(*) as count
FROM chat_messages cm
LEFT JOIN timeline_events te ON cm.message_id = te.event_id
WHERE te.event_id IS NULL

UNION ALL

SELECT 'life_events without timeline_events' as issue, COUNT(*) as count
FROM life_events le
LEFT JOIN timeline_events te ON le.life_event_id = te.event_id
WHERE te.event_id IS NULL;

-- Find duplicate external IDs
SELECT
    source_id,
    external_id,
    COUNT(*) as duplicate_count
FROM timeline_events
WHERE external_id IS NOT NULL
GROUP BY source_id, external_id
HAVING COUNT(*) > 1;

-- Table size report
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) AS table_size,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) -
                   pg_relation_size(schemaname||'.'||tablename)) AS index_size,
    (SELECT COUNT(*) FROM information_schema.columns
     WHERE table_schema = schemaname AND table_name = tablename) as column_count
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Recent sync activity
SELECT
    s.source_name,
    sl.sync_started_at,
    sl.sync_completed_at,
    sl.sync_status,
    sl.records_processed,
    sl.records_created,
    sl.records_updated,
    sl.records_failed
FROM sync_logs sl
JOIN sources s ON sl.source_id = s.source_id
ORDER BY sl.sync_started_at DESC
LIMIT 20;

-- ============================================================================
-- SECTION 6: PERFORMANCE TESTING
-- ============================================================================

-- Generate sample load test data
DO $$
DECLARE
    i INTEGER;
    source_uuid UUID := '11111111-1111-1111-1111-111111111111';
    entity_uuid UUID := 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa';
    channel_uuid UUID := 'chan1111-1111-1111-1111-111111111111';
BEGIN
    FOR i IN 1..1000 LOOP
        -- Insert timeline event
        INSERT INTO timeline_events (
            source_id,
            entity_id,
            event_timestamp,
            event_type,
            title,
            description,
            importance_score
        ) VALUES (
            source_uuid,
            entity_uuid,
            NOW() - (i || ' minutes')::INTERVAL,
            'message',
            'Test message ' || i,
            'This is test message content for performance testing',
            RANDOM()
        );

        -- Note: In real scenario, would also insert into chat_messages
        -- Skipped here for simplicity
    END LOOP;
END $$;

-- Test query performance
EXPLAIN ANALYZE
SELECT
    te.event_id,
    te.title,
    te.event_timestamp
FROM timeline_events te
WHERE te.is_deleted = false
    AND te.event_timestamp > NOW() - INTERVAL '30 days'
    AND te.event_type = 'message'
ORDER BY te.event_timestamp DESC
LIMIT 100;

-- ============================================================================
-- SECTION 7: DATA EXPORT EXAMPLES
-- ============================================================================

-- Export timeline to JSON
SELECT json_agg(timeline_data)
FROM (
    SELECT
        te.event_id,
        te.event_type,
        te.title,
        te.description,
        te.event_timestamp,
        te.importance_score,
        s.source_name,
        e.display_name as entity_name,
        COALESCE(
            json_agg(DISTINCT t.tag_name) FILTER (WHERE t.tag_name IS NOT NULL),
            '[]'::json
        ) as tags
    FROM timeline_events te
    JOIN sources s ON te.source_id = s.source_id
    LEFT JOIN entities e ON te.entity_id = e.entity_id
    LEFT JOIN event_tags et ON te.event_id = et.event_id
    LEFT JOIN tags t ON et.tag_id = t.tag_id
    WHERE te.is_deleted = false
        AND te.event_timestamp > NOW() - INTERVAL '30 days'
    GROUP BY te.event_id, s.source_name, e.display_name
    ORDER BY te.event_timestamp DESC
    LIMIT 100
) timeline_data;

-- Export chat conversation to CSV format
COPY (
    SELECT
        cc.channel_name,
        e.display_name as sender,
        te.event_timestamp,
        cm.message_text,
        cm.reaction_count
    FROM chat_messages cm
    JOIN chat_channels cc ON cm.channel_id = cc.channel_id
    JOIN entities e ON cm.sender_id = e.entity_id
    JOIN timeline_events te ON cm.message_id = te.event_id
    WHERE cc.channel_id = 'chan2222-2222-2222-2222-222222222222'
        AND te.is_deleted = false
    ORDER BY te.event_timestamp
) TO '/tmp/chat_export.csv' WITH CSV HEADER;

-- ============================================================================
-- END OF IMPLEMENTATION GUIDE
-- ============================================================================
