# SALEM FORENSIC TRINITY: UNIFIED DEPLOYMENT v3

**Purpose:** Reconcile Chronicle (therapeutic capture) + Command Center (evidence processing) into one coherent system
**Date:** December 26, 2025

---

## STACK SUMMARY

| Layer | Service | Purpose |
|-------|---------|---------|
| **Database** | Supabase PostgreSQL | Primary data store (direct connect via IPv6) |
| **ORM** | Drizzle | Type-safe queries, migrations |
| **Admin UI** | Directus | Visual data browser, file management |
| **File Storage** | Cloudflare R2 | Blob storage for evidence files |
| **Graph** | Neo4j Aura | Entity relationships |
| **Vector** | Qdrant Cloud | Semantic search (NOT pgvector, NOT Weaviate) |
| **Cache/Queue** | Dragonfly | Redis-compatible for N8N queues |
| **Workflows** | N8N | Evidence processing automation |
| **Memory** | Mem0 | AI conversation memory |
| **Knowledge Graph API** | Graphiti | Graph operations layer |

---

## SECTION 1: DOCKER COMPOSE (FIXED)

```yaml
# Salem Processing Stack v3
# Fixed: Added Directus, removed pgvector refs, aligned services

services:
  # ===========================================
  # REVERSE PROXY (if not using Coolify's)
  # ===========================================
  traefik:
    image: traefik:v3.0
    container_name: salem-traefik
    restart: unless-stopped
    command:
      - "--api.dashboard=true"
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.http.address=:80"
      - "--entrypoints.https.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge=true"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=http"
      - "--certificatesresolvers.letsencrypt.acme.email=${ACME_EMAIL}"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - traefik_certs:/letsencrypt
    networks:
      - salem-net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.traefik.rule=Host(`traefik.mitechconsult.com`)"
      - "traefik.http.routers.traefik.entrypoints=https"
      - "traefik.http.routers.traefik.tls.certresolver=letsencrypt"
      - "traefik.http.routers.traefik.service=api@internal"

  # ===========================================
  # CACHE/QUEUE (Dragonfly - Redis replacement)
  # ===========================================
  dragonfly:
    image: docker.dragonflydb.io/dragonflydb/dragonfly:latest
    container_name: salem-dragonfly
    restart: unless-stopped
    ulimits:
      memlock: -1
    volumes:
      - dragonfly_data:/data
    networks:
      - salem-net
    labels:
      - "traefik.enable=false"

  # ===========================================
  # ADMIN UI (Directus)
  # ===========================================
  directus:
    image: directus/directus:latest
    container_name: salem-directus
    restart: unless-stopped
    environment:
      KEY: ${DIRECTUS_KEY}
      SECRET: ${DIRECTUS_SECRET}
      ADMIN_EMAIL: ${DIRECTUS_ADMIN_EMAIL}
      ADMIN_PASSWORD: ${DIRECTUS_ADMIN_PASSWORD}
      DB_CLIENT: pg
      DB_HOST: ${SUPABASE_HOST}
      DB_PORT: 5432
      DB_DATABASE: postgres
      DB_USER: ${SUPABASE_USER}
      DB_PASSWORD: ${SUPABASE_PASSWORD}
      DB_SSL: "true"
      STORAGE_LOCATIONS: "r2"
      STORAGE_R2_DRIVER: "s3"
      STORAGE_R2_KEY: ${R2_ACCESS_KEY}
      STORAGE_R2_SECRET: ${R2_SECRET_KEY}
      STORAGE_R2_BUCKET: ${R2_BUCKET}
      STORAGE_R2_ENDPOINT: ${R2_ENDPOINT}
      STORAGE_R2_REGION: auto
    volumes:
      - directus_uploads:/directus/uploads
      - directus_extensions:/directus/extensions
    expose:
      - "8055"
    networks:
      - salem-net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.directus.rule=Host(`admin.mitechconsult.com`)"
      - "traefik.http.routers.directus.entrypoints=https"
      - "traefik.http.routers.directus.tls=true"
      - "traefik.http.routers.directus.tls.certresolver=letsencrypt"
      - "traefik.http.services.directus.loadbalancer.server.port=8055"

  # ===========================================
  # WORKFLOW AUTOMATION (N8N)
  # ===========================================
  n8n:
    image: n8nio/n8n:latest
    container_name: salem-n8n
    restart: unless-stopped
    environment:
      - DB_TYPE=postgresdb
      - DB_POSTGRESDB_HOST=${SUPABASE_HOST}
      - DB_POSTGRESDB_PORT=5432
      - DB_POSTGRESDB_DATABASE=postgres
      - DB_POSTGRESDB_USER=${SUPABASE_USER}
      - DB_POSTGRESDB_PASSWORD=${SUPABASE_PASSWORD}
      - DB_POSTGRESDB_SSL_REJECT_UNAUTHORIZED=false
      - N8N_HOST=0.0.0.0
      - N8N_PORT=5678
      - N8N_PROTOCOL=https
      - WEBHOOK_URL=https://n8n.mitechconsult.com
      - GENERIC_TIMEZONE=America/Detroit
      - TZ=America/Detroit
      - N8N_COMMUNITY_PACKAGES_ENABLED=true
      - QUEUE_BULL_REDIS_HOST=dragonfly
      - QUEUE_BULL_REDIS_PORT=6379
    volumes:
      - n8n_data:/home/node/.n8n
      - /mnt/r2:/mnt/r2
    expose:
      - "5678"
    depends_on:
      - dragonfly
    networks:
      - salem-net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.n8n.rule=Host(`n8n.mitechconsult.com`)"
      - "traefik.http.routers.n8n.entrypoints=https"
      - "traefik.http.routers.n8n.tls=true"
      - "traefik.http.routers.n8n.tls.certresolver=letsencrypt"
      - "traefik.http.services.n8n.loadbalancer.server.port=5678"

  # ===========================================
  # DOCUMENT PROCESSING (Unstructured)
  # ===========================================
  unstructured:
    image: quay.io/unstructured-io/unstructured-api:latest
    container_name: salem-unstructured
    restart: unless-stopped
    volumes:
      - /mnt/r2:/mnt/r2
    expose:
      - "8000"
    networks:
      - salem-net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.unstructured.rule=Host(`unstructured.mitechconsult.com`)"
      - "traefik.http.routers.unstructured.entrypoints=https"
      - "traefik.http.routers.unstructured.tls=true"
      - "traefik.http.routers.unstructured.tls.certresolver=letsencrypt"
      - "traefik.http.services.unstructured.loadbalancer.server.port=8000"

  # ===========================================
  # AI MEMORY (Mem0 - uses Qdrant Cloud, not local)
  # ===========================================
  mem0:
    image: mem0ai/mem0:latest
    container_name: salem-mem0
    restart: unless-stopped
    environment:
      - MEM0_API_KEY=${MEM0_API_KEY}
      - QDRANT_URL=${QDRANT_URL}
      - QDRANT_API_KEY=${QDRANT_API_KEY}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    volumes:
      - mem0_data:/app/data
    expose:
      - "8000"
    networks:
      - salem-net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.mem0.rule=Host(`mem0.mitechconsult.com`)"
      - "traefik.http.routers.mem0.entrypoints=https"
      - "traefik.http.routers.mem0.tls=true"
      - "traefik.http.routers.mem0.tls.certresolver=letsencrypt"
      - "traefik.http.services.mem0.loadbalancer.server.port=8000"

  # ===========================================
  # KNOWLEDGE GRAPH API (Graphiti - connects to Neo4j Aura)
  # ===========================================
  graphiti:
    image: zepai/graphiti:latest
    container_name: salem-graphiti
    restart: unless-stopped
    environment:
      - NEO4J_URI=${NEO4J_URI}
      - NEO4J_USER=${NEO4J_USER:-neo4j}
      - NEO4J_PASSWORD=${NEO4J_PASSWORD}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    expose:
      - "8000"
    networks:
      - salem-net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.graphiti.rule=Host(`graphiti.mitechconsult.com`)"
      - "traefik.http.routers.graphiti.entrypoints=https"
      - "traefik.http.routers.graphiti.tls=true"
      - "traefik.http.routers.graphiti.tls.certresolver=letsencrypt"
      - "traefik.http.services.graphiti.loadbalancer.server.port=8000"

  # ===========================================
  # CHAT UI (Open WebUI)
  # ===========================================
  open-webui:
    image: ghcr.io/open-webui/open-webui:main
    container_name: salem-openwebui
    restart: unless-stopped
    environment:
      - DATABASE_URL=${SUPABASE_DB_URL}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - OPENAI_API_BASE_URL=${OPENAI_API_BASE_URL:-https://api.openai.com/v1}
      - WEBUI_SECRET_KEY=${WEBUI_SECRET_KEY}
      - WEBUI_AUTH=true
      - ENABLE_SIGNUP=false
    volumes:
      - openwebui_data:/app/backend/data
      - /mnt/r2:/mnt/r2:ro
    expose:
      - "8080"
    networks:
      - salem-net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.openwebui.rule=Host(`chat.mitechconsult.com`)"
      - "traefik.http.routers.openwebui.entrypoints=https"
      - "traefik.http.routers.openwebui.tls=true"
      - "traefik.http.routers.openwebui.tls.certresolver=letsencrypt"
      - "traefik.http.services.openwebui.loadbalancer.server.port=8080"

volumes:
  traefik_certs:
  dragonfly_data:
  directus_uploads:
  directus_extensions:
  n8n_data:
  mem0_data:
  openwebui_data:

networks:
  salem-net:
    name: salem-network
```

---

## SECTION 2: UNIFIED DATABASE SCHEMA

This schema serves BOTH Chronicle (voice capture) AND Command Center (evidence processing).

### 2.1 Drizzle Schema

```typescript
// src/db/schema.ts
import { 
  pgTable, pgSchema,
  uuid, text, timestamp, boolean, jsonb, decimal, integer,
  index, uniqueIndex, primaryKey
} from 'drizzle-orm/pg-core';
import { relations } from 'drizzle-orm';

// Reference Supabase auth schema
const authSchema = pgSchema('auth');
export const authUsers = authSchema.table('users', {
  id: uuid('id').primaryKey(),
  email: text('email'),
});

// ============================================
// CORE: CASES (container for everything)
// ============================================
export const cases = pgTable('cases', {
  id: uuid('id').defaultRandom().primaryKey(),
  userId: uuid('user_id').notNull().references(() => authUsers.id),
  
  // Case identification
  name: text('name').notNull().default('Primary Case'),
  caseNumber: text('case_number'),                    // '2025-53985-DC'
  jurisdiction: text('jurisdiction'),                  // 'Genesee County 7th Circuit'
  judge: text('judge'),                               // 'Dawn M. Weier'
  
  // Status
  status: text('status').default('active'),
  
  // Legacy JSON blob (Chronicle's original pattern)
  caseData: jsonb('case_data').default({}),
  
  // Timestamps
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
}, (table) => ({
  userIdx: index('cases_user_idx').on(table.userId),
}));

// ============================================
// FILES (R2 blob references - Directus manages these)
// ============================================
export const files = pgTable('files', {
  id: uuid('id').defaultRandom().primaryKey(),
  userId: uuid('user_id').notNull().references(() => authUsers.id),
  caseId: uuid('case_id').references(() => cases.id, { onDelete: 'cascade' }),
  
  // R2 reference
  r2Key: text('r2_key').notNull(),
  r2Bucket: text('r2_bucket').notNull().default('salem-legal-evidence'),
  
  // Metadata
  fileName: text('file_name').notNull(),
  fileType: text('file_type').notNull(),              // 'video', 'audio', 'image', 'pdf', 'xml', 'html'
  mimeType: text('mime_type'),
  fileSize: integer('file_size'),
  
  // Integrity
  sha256Hash: text('sha256_hash'),                    // Chain of custody
  
  // Classification
  category: text('category'),                         // 'evidence', 'court_filing', 'communication', 'media'
  platform: text('platform'),                         // 'sms', 'facebook', 'snapchat', 'screenshot'
  description: text('description'),
  tags: text('tags').array(),
  
  // Processing status
  processingStatus: text('processing_status').default('pending'),  // 'pending', 'processing', 'complete', 'failed'
  processedAt: timestamp('processed_at', { withTimezone: true }),
  
  // Source tracking
  sourceApp: text('source_app').notNull().default('manual'),
  
  // Timestamps
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
}, (table) => ({
  userIdx: index('files_user_idx').on(table.userId),
  caseIdx: index('files_case_idx').on(table.caseId),
  hashIdx: uniqueIndex('files_hash_idx').on(table.sha256Hash),
  statusIdx: index('files_status_idx').on(table.processingStatus),
}));

// ============================================
// ENTITIES (people, places, organizations)
// Replaces: master_person (Command Center)
// ============================================
export const entities = pgTable('entities', {
  id: uuid('id').defaultRandom().primaryKey(),
  userId: uuid('user_id').notNull().references(() => authUsers.id),
  caseId: uuid('case_id').references(() => cases.id, { onDelete: 'cascade' }),
  
  // Identity
  name: text('name').notNull(),
  canonicalName: text('canonical_name').notNull(),
  aliases: text('aliases').array().default([]),
  
  // Classification
  type: text('type').notNull(),                       // 'person', 'place', 'organization'
  subtype: text('subtype'),                           // 'family', 'workplace', 'court', 'witness'
  
  // Contact info (for entity resolution)
  phoneNumbers: text('phone_numbers').array().default([]),   // E.164 format
  emails: text('emails').array().default([]),
  socialHandles: jsonb('social_handles').default({}),        // {facebook: "...", snapchat: "..."}
  
  // Relationship to user
  relationshipToUser: text('relationship_to_user'),
  
  // Case relevance
  impactOnCase: text('impact_on_case').default('unknown'),
  roleInCase: text('role_in_case'),
  
  // Details
  notes: text('notes'),
  profile: text('profile'),
  firstAppearanceDate: text('first_appearance_date'),
  
  // Document links
  documentKeys: text('document_keys').array().default([]),
  
  // Source tracking
  sourceApp: text('source_app').notNull().default('chronicle'),
  
  // Timestamps
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
}, (table) => ({
  userIdx: index('entities_user_idx').on(table.userId),
  caseIdx: index('entities_case_idx').on(table.caseId),
  canonicalIdx: index('entities_canonical_idx').on(table.canonicalName),
  typeIdx: index('entities_type_idx').on(table.type),
}));

// ============================================
// ENTITY RELATIONSHIPS (graph edges)
// ============================================
export const entityRelationships = pgTable('entity_relationships', {
  id: uuid('id').defaultRandom().primaryKey(),
  userId: uuid('user_id').notNull().references(() => authUsers.id),
  caseId: uuid('case_id').references(() => cases.id, { onDelete: 'cascade' }),
  
  sourceEntityId: uuid('source_entity_id').notNull().references(() => entities.id, { onDelete: 'cascade' }),
  targetEntityId: uuid('target_entity_id').notNull().references(() => entities.id, { onDelete: 'cascade' }),
  
  label: text('label').notNull(),                     // 'married_to', 'employed_by', 'threatened'
  relationshipType: text('relationship_type').notNull(),  // 'familial', 'romantic', 'professional', 'conflict'
  
  startDate: text('start_date'),
  endDate: text('end_date'),
  isCurrent: boolean('is_current').default(true),
  
  notes: text('notes'),
  sourceApp: text('source_app').notNull().default('chronicle'),
  
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
}, (table) => ({
  sourceIdx: index('entity_rel_source_idx').on(table.sourceEntityId),
  targetIdx: index('entity_rel_target_idx').on(table.targetEntityId),
}));

// ============================================
// TIMELINE EVENTS
// Replaces: events (Command Center) + timeline_events (Chronicle)
// ============================================
export const timelineEvents = pgTable('timeline_events', {
  id: uuid('id').defaultRandom().primaryKey(),
  userId: uuid('user_id').notNull().references(() => authUsers.id),
  caseId: uuid('case_id').references(() => cases.id, { onDelete: 'cascade' }),
  
  // Temporal
  dateRaw: text('date_raw').notNull(),                // Original: "Summer 2005", "Oct 12, 2024"
  dateParsed: timestamp('date_parsed', { withTimezone: true }),
  datePrecision: text('date_precision').default('unknown'),  // 'exact', 'day', 'month', 'year', 'approximate'
  
  // Content
  description: text('description').notNull(),
  category: text('category').notNull(),               // See enum below
  
  // Location & Witnesses
  location: text('location'),
  witnesses: text('witnesses').array().default([]),
  
  // Linked entities
  involvedEntities: uuid('involved_entities').array().default([]),
  
  // Analysis flags
  isSignificant: boolean('is_significant').default(false),
  manipulationPattern: text('manipulation_pattern'),  // 'DARVO', 'gaslighting', 'triangulation'
  evidenceStrength: text('evidence_strength').default('unknown'),
  evidenceLinks: uuid('evidence_links').array().default([]),  // file IDs
  
  // Legal tagging (MCL 722.23)
  legalFactors: text('legal_factors').array().default([]),    // ['K', 'J', 'G']
  legalRelevance: decimal('legal_relevance'),                  // 0.0 - 1.0
  keyQuote: text('key_quote'),
  
  // Emotional/Psychological (Chronicle)
  mood: text('mood'),
  emotionalState: text('emotional_state'),
  vulnerability: text('vulnerability'),
  triggerContext: text('trigger_context'),
  rawQuotes: text('raw_quotes').array().default([]),
  
  // Child presence (critical for custody)
  childPresent: boolean('child_present'),
  childImpact: text('child_impact'),
  
  // AI analysis
  aiAnalysis: text('ai_analysis'),
  
  // Source tracking
  sourceApp: text('source_app').notNull().default('chronicle'),
  sourceFileId: uuid('source_file_id').references(() => files.id),
  
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
}, (table) => ({
  userIdx: index('timeline_user_idx').on(table.userId),
  caseIdx: index('timeline_case_idx').on(table.caseId),
  dateIdx: index('timeline_date_idx').on(table.dateParsed),
  categoryIdx: index('timeline_category_idx').on(table.category),
  significantIdx: index('timeline_significant_idx').on(table.isSignificant),
}));

// Event categories enum (reference)
// 'childhood_memory', 'historical_trauma', 'current_case_incident', 'career_professional',
// 'relationship_event', 'legal_court_event', 'medical_mental_health', 'financial_event',
// 'strength_resilience', 'communication', 'witness_account', 'video_evidence', 'ai_chat_extract'

// ============================================
// COMMUNICATIONS (parsed messages)
// For Command Center's SMS/Facebook parsing
// ============================================
export const communications = pgTable('communications', {
  id: uuid('id').defaultRandom().primaryKey(),
  userId: uuid('user_id').notNull().references(() => authUsers.id),
  caseId: uuid('case_id').references(() => cases.id, { onDelete: 'cascade' }),
  
  // Message content
  content: text('content').notNull(),
  contentHash: text('content_hash').notNull(),        // Deduplication
  
  // Participants
  senderId: uuid('sender_id').references(() => entities.id),
  recipientId: uuid('recipient_id').references(() => entities.id),
  senderRaw: text('sender_raw'),                      // Original before entity resolution
  recipientRaw: text('recipient_raw'),
  
  // Temporal
  timestamp: timestamp('timestamp', { withTimezone: true }).notNull(),
  timestampRaw: text('timestamp_raw'),                // Original format
  timezone: text('timezone'),
  
  // Platform
  platform: text('platform').notNull(),               // 'sms', 'facebook', 'snapchat', etc.
  threadId: text('thread_id'),
  
  // Message type
  messageType: integer('message_type'),               // SMS: 1=recv, 2=sent, etc.
  direction: text('direction'),                       // 'inbound', 'outbound'
  
  // Attachments
  hasAttachments: boolean('has_attachments').default(false),
  attachments: jsonb('attachments').default([]),
  
  // Analysis
  sentiment: decimal('sentiment'),                    // -1.0 to 1.0
  legalFactors: text('legal_factors').array().default([]),
  
  // Provenance
  sourceFileId: uuid('source_file_id').references(() => files.id),
  byteOffset: integer('byte_offset'),                 // Position in source file
  
  // Link to timeline event (if promoted)
  timelineEventId: uuid('timeline_event_id').references(() => timelineEvents.id),
  
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
}, (table) => ({
  userIdx: index('comms_user_idx').on(table.userId),
  caseIdx: index('comms_case_idx').on(table.caseId),
  timestampIdx: index('comms_timestamp_idx').on(table.timestamp),
  platformIdx: index('comms_platform_idx').on(table.platform),
  hashIdx: uniqueIndex('comms_hash_idx').on(table.contentHash),
  senderIdx: index('comms_sender_idx').on(table.senderId),
}));

// ============================================
// CONTEXT PHASES (life eras)
// ============================================
export const contextPhases = pgTable('context_phases', {
  id: uuid('id').defaultRandom().primaryKey(),
  userId: uuid('user_id').notNull().references(() => authUsers.id),
  caseId: uuid('case_id').references(() => cases.id, { onDelete: 'cascade' }),
  
  name: text('name').notNull(),
  category: text('category').notNull(),               // 'life_era', 'relationship', 'career', 'trauma'
  
  startDate: text('start_date').notNull(),
  endDate: text('end_date'),
  isCurrent: boolean('is_current').default(false),
  
  description: text('description'),
  emotionalBaseline: text('emotional_baseline'),
  livingSituation: text('living_situation'),
  employmentStatus: text('employment_status'),
  relationshipStatus: text('relationship_status'),
  
  primaryStressors: text('primary_stressors').array().default([]),
  primarySupports: text('primary_supports').array().default([]),
  activePatterns: text('active_patterns').array().default([]),
  
  sourceApp: text('source_app').notNull().default('chronicle'),
  
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
}, (table) => ({
  userIdx: index('phases_user_idx').on(table.userId),
  caseIdx: index('phases_case_idx').on(table.caseId),
}));

// ============================================
// NOTES (global observations)
// ============================================
export const notes = pgTable('notes', {
  id: uuid('id').defaultRandom().primaryKey(),
  userId: uuid('user_id').notNull().references(() => authUsers.id),
  caseId: uuid('case_id').references(() => cases.id, { onDelete: 'cascade' }),
  
  title: text('title'),
  content: text('content').notNull(),
  category: text('category').notNull(),               // 'strategy', 'theory', 'observation', 'vulnerability', 'strength', 'reflection'
  
  linkedEvents: uuid('linked_events').array().default([]),
  linkedEntities: uuid('linked_entities').array().default([]),
  tags: text('tags').array().default([]),
  
  sourceApp: text('source_app').notNull().default('chronicle'),
  
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
}, (table) => ({
  userIdx: index('notes_user_idx').on(table.userId),
  caseIdx: index('notes_case_idx').on(table.caseId),
  categoryIdx: index('notes_category_idx').on(table.category),
}));

// ============================================
// LEGAL CITATIONS
// ============================================
export const legalCitations = pgTable('legal_citations', {
  id: uuid('id').defaultRandom().primaryKey(),
  userId: uuid('user_id').notNull().references(() => authUsers.id),
  caseId: uuid('case_id').references(() => cases.id, { onDelete: 'cascade' }),
  
  citation: text('citation').notNull(),               // 'Ireland v. Smith, 451 Mich. 457 (1996)'
  citationType: text('citation_type'),                // 'case', 'statute', 'rule'
  
  // Shepardizing
  status: text('status').default('unchecked'),        // 'good', 'caution', 'overruled', 'unchecked'
  lastChecked: timestamp('last_checked', { withTimezone: true }),
  courtListenerId: text('court_listener_id'),
  
  // Usage
  usedFor: text('used_for'),
  notes: text('notes'),
  
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
}, (table) => ({
  userIdx: index('citations_user_idx').on(table.userId),
  citationIdx: uniqueIndex('citations_citation_idx').on(table.citation),
}));

// ============================================
// FORENSIC ARTIFACTS (Video Analyzer / Chat Miner output)
// ============================================
export const forensicArtifacts = pgTable('forensic_artifacts', {
  id: uuid('id').defaultRandom().primaryKey(),
  userId: uuid('user_id').notNull().references(() => authUsers.id),
  caseId: uuid('case_id').references(() => cases.id, { onDelete: 'cascade' }),
  
  sourceApp: text('source_app').notNull(),            // 'video_analyzer', 'chat_miner'
  sourceFileId: uuid('source_file_id').references(() => files.id),
  sourceType: text('source_type').notNull(),          // 'video', 'audio', 'chat_transcript'
  
  // Video Analyzer outputs
  transcript: text('transcript'),
  sentimentScore: decimal('sentiment_score'),
  sentimentLabel: text('sentiment_label'),
  deceptionIndicators: jsonb('deception_indicators').default({}),
  speakerSegments: jsonb('speaker_segments').default([]),
  
  // Chat Miner outputs
  aiPlatform: text('ai_platform'),                    // 'chatgpt', 'claude', 'gemini'
  conversationDate: timestamp('conversation_date', { withTimezone: true }),
  extractedFacts: jsonb('extracted_facts').default([]),
  extractedEntities: jsonb('extracted_entities').default([]),
  traumaIndicators: jsonb('trauma_indicators').default([]),
  
  // Link to created timeline event
  linkedEventId: uuid('linked_event_id').references(() => timelineEvents.id),
  
  // Processing metadata
  processingModel: text('processing_model'),
  confidenceScore: decimal('confidence_score'),
  processedAt: timestamp('processed_at', { withTimezone: true }).defaultNow(),
  
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
}, (table) => ({
  userIdx: index('forensic_user_idx').on(table.userId),
  sourceIdx: index('forensic_source_idx').on(table.sourceApp),
}));

// ============================================
// COMMAND LOG (workflow executions)
// ============================================
export const commandLog = pgTable('command_log', {
  id: uuid('id').defaultRandom().primaryKey(),
  userId: uuid('user_id').notNull().references(() => authUsers.id),
  
  command: text('command').notNull(),
  source: text('source'),                             // 'command_center', 'n8n', 'chronicle'
  agent: text('agent'),
  prompt: text('prompt'),
  fileRefs: uuid('file_refs').array().default([]),
  
  status: text('status').default('pending'),          // 'pending', 'running', 'complete', 'failed'
  response: text('response'),
  errorMessage: text('error_message'),
  executionTimeMs: integer('execution_time_ms'),
  
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  completedAt: timestamp('completed_at', { withTimezone: true }),
}, (table) => ({
  userIdx: index('cmd_log_user_idx').on(table.userId),
  statusIdx: index('cmd_log_status_idx').on(table.status),
}));

// ============================================
// AUDIT LOG (every modification)
// ============================================
export const auditLog = pgTable('audit_log', {
  id: uuid('id').defaultRandom().primaryKey(),
  userId: uuid('user_id').references(() => authUsers.id),
  
  action: text('action').notNull(),                   // 'insert', 'update', 'delete'
  tableName: text('table_name').notNull(),
  recordId: uuid('record_id').notNull(),
  
  oldValue: jsonb('old_value'),
  newValue: jsonb('new_value'),
  
  sourceApp: text('source_app'),
  
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
}, (table) => ({
  tableIdx: index('audit_table_idx').on(table.tableName),
  recordIdx: index('audit_record_idx').on(table.recordId),
  createdIdx: index('audit_created_idx').on(table.createdAt),
}));

// ============================================
// SYNC LOG (cross-app coordination)
// ============================================
export const syncLog = pgTable('sync_log', {
  id: uuid('id').defaultRandom().primaryKey(),
  userId: uuid('user_id').notNull().references(() => authUsers.id),
  caseId: uuid('case_id').references(() => cases.id),
  
  sourceApp: text('source_app').notNull(),
  operation: text('operation').notNull(),
  tableName: text('table_name').notNull(),
  recordId: uuid('record_id').notNull(),
  
  changes: jsonb('changes').default({}),
  previousState: jsonb('previous_state').default({}),
  
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
}, (table) => ({
  sourceIdx: index('sync_source_idx').on(table.sourceApp),
  createdIdx: index('sync_created_idx').on(table.createdAt),
}));
```

### 2.2 Drizzle Config

```typescript
// drizzle.config.ts
import type { Config } from 'drizzle-kit';

export default {
  schema: './src/db/schema.ts',
  out: './drizzle',
  driver: 'pg',
  dbCredentials: {
    connectionString: process.env.SUPABASE_DIRECT_URL!,
  },
} satisfies Config;
```

### 2.3 Migration Commands

```bash
# Generate migration from schema
npx drizzle-kit generate:pg

# Push to database (development)
npx drizzle-kit push:pg

# Run migrations (production)
npx drizzle-kit migrate
```

---

## SECTION 3: NEO4J GRAPH SCHEMA

Managed by Graphiti service. All apps write through Graphiti API.

### Node Labels

```cypher
// Constraints
CREATE CONSTRAINT person_id IF NOT EXISTS FOR (p:Person) REQUIRE p.id IS UNIQUE;
CREATE CONSTRAINT place_id IF NOT EXISTS FOR (p:Place) REQUIRE p.id IS UNIQUE;
CREATE CONSTRAINT org_id IF NOT EXISTS FOR (o:Organization) REQUIRE o.id IS UNIQUE;
CREATE CONSTRAINT event_id IF NOT EXISTS FOR (e:Event) REQUIRE e.id IS UNIQUE;
```

### Relationship Types

```cypher
// Familial
(:Person)-[:PARENT_OF {start_date, source_app}]->(:Person)
(:Person)-[:SIBLING_OF]->(:Person)

// Romantic
(:Person)-[:DATED {start_date, end_date}]->(:Person)
(:Person)-[:MARRIED_TO]->(:Person)
(:Person)-[:AFFAIR_WITH]->(:Person)

// Professional
(:Person)-[:EMPLOYED_BY {start_date, end_date}]->(:Organization)
(:Person)-[:COWORKER_OF]->(:Person)

// Conflict
(:Person)-[:THREATENED {date, source_app}]->(:Person)
(:Person)-[:MANIPULATED {pattern}]->(:Person)
(:Person)-[:ALIENATED_FROM]->(:Person)

// Video Analyzer specific
(:Person)-[:SPOKE_TO {timestamp, sentiment}]->(:Person)
(:Person)-[:PRESENT_IN_FOOTAGE {file_id}]->(:Event)
(:Person)-[:DECEPTIVE_STATEMENT_ABOUT {confidence}]->(:Person)
```

---

## SECTION 4: QDRANT COLLECTIONS

Managed by Mem0 service. Used for semantic search.

```json
{
  "collections": [
    {
      "name": "timeline_events",
      "vectors": {
        "size": 1536,
        "distance": "Cosine"
      },
      "payload_schema": {
        "event_id": "uuid",
        "user_id": "uuid",
        "date_raw": "keyword",
        "category": "keyword",
        "source_app": "keyword"
      }
    },
    {
      "name": "communications",
      "vectors": {
        "size": 1536,
        "distance": "Cosine"
      },
      "payload_schema": {
        "comm_id": "uuid",
        "user_id": "uuid",
        "platform": "keyword",
        "timestamp": "datetime"
      }
    },
    {
      "name": "context_chunks",
      "vectors": {
        "size": 1536,
        "distance": "Cosine"
      },
      "payload_schema": {
        "chunk_id": "uuid",
        "user_id": "uuid",
        "source_app": "keyword",
        "source_type": "keyword"
      }
    }
  ]
}
```

---

## SECTION 5: ENVIRONMENT VARIABLES

```bash
# ===========================================
# SUPABASE (Direct connection via IPv6)
# ===========================================
SUPABASE_HOST=db.oflqpddqaecotsdsxbzp.supabase.co
SUPABASE_USER=postgres
SUPABASE_PASSWORD=<your_password>
SUPABASE_DB_URL=postgresql://postgres:<password>@db.oflqpddqaecotsdsxbzp.supabase.co:5432/postgres
SUPABASE_DIRECT_URL=postgresql://postgres:<password>@db.oflqpddqaecotsdsxbzp.supabase.co:5432/postgres

# ===========================================
# DIRECTUS
# ===========================================
DIRECTUS_KEY=<random-uuid>
DIRECTUS_SECRET=<random-secret>
DIRECTUS_ADMIN_EMAIL=matt@mitechconsult.com
DIRECTUS_ADMIN_PASSWORD=<admin_password>

# ===========================================
# CLOUDFLARE R2
# ===========================================
R2_ENDPOINT=https://<account_id>.r2.cloudflarestorage.com
R2_ACCESS_KEY=<access_key>
R2_SECRET_KEY=<secret_key>
R2_BUCKET=salem-legal-evidence

# ===========================================
# NEO4J AURA
# ===========================================
NEO4J_URI=neo4j+s://<instance>.databases.neo4j.io
NEO4J_USER=neo4j
NEO4J_PASSWORD=<neo4j_password>

# ===========================================
# QDRANT CLOUD
# ===========================================
QDRANT_URL=https://<cluster>.qdrant.io:6333
QDRANT_API_KEY=<qdrant_api_key>

# ===========================================
# AI KEYS
# ===========================================
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
OPENROUTER_API_KEY=sk-or-...

# ===========================================
# SERVICE KEYS
# ===========================================
MEM0_API_KEY=<mem0_key>
WEBUI_SECRET_KEY=<random_secret>

# ===========================================
# TRAEFIK
# ===========================================
ACME_EMAIL=matt@mitechconsult.com
```

---

## SECTION 6: DEPLOYMENT CHECKLIST

### Phase 1: Database

- [ ] Get IPv6 on Hetzner VPS
- [ ] Whitelist IPv6 in Supabase
- [ ] Test direct Postgres connection
- [ ] Run Drizzle migrations
- [ ] Verify all tables created

### Phase 2: Docker Services

- [ ] Deploy Traefik (if not using Coolify's)
- [ ] Deploy Dragonfly
- [ ] Deploy Directus
- [ ] Deploy N8N
- [ ] Deploy Unstructured
- [ ] Deploy Mem0
- [ ] Deploy Graphiti
- [ ] Deploy Open WebUI

### Phase 3: DNS

- [ ] admin.mitechconsult.com → Directus
- [ ] n8n.mitechconsult.com → N8N
- [ ] mem0.mitechconsult.com → Mem0
- [ ] graphiti.mitechconsult.com → Graphiti
- [ ] chat.mitechconsult.com → Open WebUI
- [ ] unstructured.mitechconsult.com → Unstructured

### Phase 4: Verification

- [ ] Directus sees all tables
- [ ] Directus can upload to R2
- [ ] N8N can connect to Postgres
- [ ] Graphiti can connect to Neo4j
- [ ] Mem0 can connect to Qdrant

---

## SECTION 7: APP INTEGRATION RULES

### Chronicle (Voice Capture)

**Writes to:** `cases`, `timeline_events`, `entities`, `entity_relationships`, `context_phases`, `notes`, `sync_log`
**Reads from:** All tables
**Graph:** Creates Person/Place nodes, relationship edges via Graphiti API
**Vector:** Embeds timeline events via Mem0 API

### Video Analyzer

**Writes to:** `files`, `forensic_artifacts`, `timeline_events`, `entities`, `communications`, `sync_log`
**Graph:** Creates PRESENT_IN_FOOTAGE, SPOKE_TO, DECEPTIVE_STATEMENT_ABOUT edges
**Vector:** Embeds transcript chunks

### Chat Miner

**Writes to:** `files`, `forensic_artifacts`, `timeline_events`, `entities`, `context_phases`, `notes`, `sync_log`
**Graph:** Creates historical relationship edges with date ranges
**Vector:** Embeds context chunks

### Command Center

**Writes to:** `command_log`, `files`
**Reads from:** All tables
**Triggers:** N8N workflows via webhooks

---

**END UNIFIED DEPLOYMENT**
