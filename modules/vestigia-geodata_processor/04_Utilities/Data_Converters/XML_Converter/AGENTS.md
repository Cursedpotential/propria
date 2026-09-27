# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a browser-based XML to CSV converter built with React, TypeScript, and Vite. It processes multi-gigabyte XML files (particularly SMS/message exports) using streaming technologies to avoid memory constraints. The app supports local file conversion, Google Drive integration, and direct streaming to Supabase databases.

**Key Capabilities:**
- Memory-safe streaming parser for gigabyte+ XML files
- AI-powered context analysis via Gemini API
- Multiple export formats (CSV, SQLite, PostgreSQL, Neo4j)
- Deduplication strategies (hash-based, fuzzy, ID-based)
- Attachment handling (inline, separate files, cloud storage)
- Real-time data preview and SQL query builder

## Development Commands

**Start development server:**
```bash
npm run dev
```
Server runs at http://localhost:3000 (configured in vite.config.ts)

**Build for production:**
```bash
npm run build
```

**Preview production build:**
```bash
npm run preview
```

## Environment Configuration

Required environment variables in `.env.local`:

- `VITE_GEMINI_API_KEY` - Required for AI-powered analysis features
- `VITE_SUPABASE_URL` - For Supabase streaming/storage features
- `VITE_SUPABASE_SERVICE_ROLE_KEY` - Service role key for backend operations
- `VITE_GOOGLE_DRIVE_CLIENT_ID` - For Google Drive file source integration

**Note:** The `.env` file in the repository contains actual credentials and should be treated as sensitive. In production, these should be moved to `.env.local` and `.env` should be gitignored.

## Architecture

### Core Processing Pipeline

The main data flow follows this path:

1. **Source Selection** (`App.tsx`) - File upload or Google Drive picker
2. **XML Streaming** (`services/xmlStreamService.ts`) - Chunked file reader with tag parsing
3. **Schema Discovery** - Auto-detects XML structure and available columns
4. **Configuration** (`ConversionConfig` in `types.ts`) - User defines export options, deduplication, splitting
5. **Streaming Conversion** - Processes XML in chunks, writes to configured outputs
6. **Export** - Local download, SQLite file, Supabase streaming, or Neo4j graph export

### Key Services

**`xmlStreamService.ts`** - Core streaming processor
- Uses ReadableStream API for memory-efficient file processing
- Supports local File objects and Google Drive streams via `GoogleDriveService`
- Implements chunked parsing with 1MB buffer (`CHUNK_SIZE`)
- Handles nested XML elements (`_parts`, `_addrs` arrays)
- Manages splitting logic (by row count, file size, or date ranges)
- Coordinates with Supabase, SQLite, and Neo4j exporters

**`supabaseService.ts`** - Supabase integration
- Batch inserts (500 records per batch, `SUPABASE_BATCH_SIZE`)
- Schema deployment via Supabase Management API (requires Personal Access Token)
- Handles messages, entities, and attachments tables
- Storage bucket integration for base64 attachments

**`geminiService.ts`** - AI-powered analysis
- Uses Google Generative AI SDK
- Analyzes conversation context, participants, topics, date ranges
- Requires `VITE_GEMINI_API_KEY` environment variable

**`sqliteService.ts`** - Client-side SQLite export using sql.js
- Generates in-memory SQLite database
- Downloads as `.db` file via browser File System Access API

**Hash & UUID utilities:**
- `sha256.ts` - Custom SHA-256 implementation for deduplication
- `uuid.ts` - UUIDv7 generator (time-ordered UUIDs)

### Component Architecture

**Main App** (`App.tsx`) - Single-page application with tab-based navigation:
- `convert` - File upload and conversion configuration
- `data` - Browse processed data (when connected to Supabase)
- `query` - SQL query builder interface
- `graph` - Neo4j graph visualization/export configuration
- `export` - Export management panel
- `settings` - Configuration and credentials

**Core Components:**
- `FileDropZone` - Drag-and-drop upload with Google Drive picker integration
- `SettingsPanel` - Conversion configuration (deduplication, splitting, export targets)
- `ColumnManager` - Select/reorder discovered XML columns
- `PreviewTable` - Display first 50 rows during schema discovery
- `AnalysisPanel` - AI-powered conversation analysis results
- `DataBrowser` - Paginated table viewer for Supabase data
- `QueryBuilder` - Visual SQL query builder with table joins
- `GraphBuilder` - Configure Neo4j node/edge mappings

## Important Types & Enums

**`Base64Option`** - How to handle base64-encoded attachments:
- `INLINE` - Keep in message column
- `SEPARATE_COLUMN` - Extract to dedicated column
- `SEPARATE_FILE` - Download as individual files
- `EXPORT_IMAGES` - Decode and save image files
- `UPLOAD_TO_STORAGE` - Upload to Supabase Storage bucket
- `SKIP` - Ignore base64 data

**`DeduplicationStrategy`**:
- `STRICT_HASH` - SHA-256 hash of (date + sender + body)
- `FUZZY_BODY` - Normalized body text matching
- `ID_ONLY` - Use source XML ID field only

**`SplitMethod`**:
- `NONE` - Single output file
- `ROW_COUNT` - Split every N rows
- `FILE_SIZE_MB` - Split when file reaches N megabytes
- `DATE` - Split by month/year based on message dates

**`StreamSource`** - Polymorphic file source:
```typescript
{ type: 'FILE', file: File } |
{ type: 'DRIVE', url: string, name: string, token: string, size: number }
```

## Critical Implementation Notes

### Streaming Parser Architecture

The XML parser in `xmlStreamService.ts` processes files in 1MB chunks without loading the entire file into memory. It maintains a state machine for tag parsing and uses buffers for batch processing:

- **Message Buffer**: Accumulates 500 messages before batch insert
- **Entity Buffer**: Map-based deduplication for participants
- **Attachment Buffer**: Separate handling for binary data
- **Neo4j Buffers**: Dynamic node/edge maps organized by label

### Deduplication Logic

When `calculateHash` is enabled, the processor generates SHA-256 hashes based on the selected `DeduplicationStrategy`. The hash is computed in `xmlStreamService.ts` during message processing and can be used for:
- Preventing duplicate imports
- Identifying message threads
- Forensic analysis tracking

### File Splitting

The `filenamePattern` configuration supports template variables:
- `{Source}` - Source label from config
- `{Date}` - Current timestamp or message date range
- `{Seq}` - Sequential split number

When `splitMethod` is `DATE`, files are split by `dateSplitGranularity` (MONTH or YEAR), with one file per time period.

### Supabase Schema Deployment

The app can auto-deploy database schemas via `deploySchema()` in `supabaseService.ts`. This requires:
1. A **Personal Access Token** (starts with `sbp_`), NOT the project API key
2. Project ref extracted from Supabase URL (`https://<ref>.supabase.co`)
3. Sends SQL DDL to Supabase Management API

Schema includes:
- `messages` table with discovered columns
- `entities` table for participants
- `attachments` table for binary data references
- Storage bucket for file uploads
- RLS policies and indexes

### File System Access API

The app uses the modern File System Access API for:
- Directory selection (`window.showDirectoryPicker()`)
- File save dialogs (`window.showSaveFilePicker()`)
- Stream writing without buffering entire file in memory

This API is only available in Chromium-based browsers and requires user permission.

## Development Patterns

**State Management**: Uses React useState and useRef hooks. No external state library. Main state lives in `App.tsx` and is passed to components via props.

**Styling**: Tailwind CSS via `index.css`. Uses `tailwind-merge` utility and `clsx` for conditional classes.

**Path Aliases**: `@/` resolves to project root (configured in `vite.config.ts` and `tsconfig.json`)

**Type Safety**: All components use TypeScript with strict types defined in `types.ts`. Note that `experimentalDecorators` is enabled in tsconfig.json.

## Testing & Debugging

The app logs progress updates via callback functions passed to service classes. Monitor these in the browser console during conversion.

**Common Issues:**
- **Large files fail**: Check browser memory limits. Enable file splitting.
- **Supabase schema deployment fails**: Verify you're using a Personal Access Token, not the service role JWT.
- **Google Drive integration not working**: Ensure `VITE_GOOGLE_DRIVE_CLIENT_ID` is set and OAuth consent screen is configured.
- **Gemini analysis fails**: Check `VITE_GEMINI_API_KEY` is valid and has appropriate quota.

## Security Considerations

**Current State**: The `.env` file contains actual API keys and should NOT be committed. Move sensitive credentials to `.env.local` (already in `.gitignore`).

**Service Role Key Exposure**: The `VITE_SUPABASE_SERVICE_ROLE_KEY` is exposed to the client. This is a security risk in production. Consider using Row Level Security policies and the anon key instead, or implement a backend proxy for sensitive operations.

**File Upload Safety**: The app processes user-provided XML files. While it doesn't execute code from XML, be cautious with malformed or malicious XML structures that could cause denial-of-service via excessive memory usage.
