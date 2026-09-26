
export interface ParsedMessage {
  [key: string]: string | any;
  _parts?: ParsedMessage[];
  _addrs?: ParsedMessage[];
  
  // Post-Processing Fields (Placeholder for future AI Agents)
  abuse_score?: number;
  sentiment_label?: string;
  manipulation_tags?: string[]; // JSON string or array
  ai_analysis_notes?: string;
}

export enum Base64Option {
  INLINE = 'INLINE',
  SEPARATE_COLUMN = 'SEPARATE_COLUMN',
  SEPARATE_FILE = 'SEPARATE_FILE',
  EXPORT_IMAGES = 'EXPORT_IMAGES',
  UPLOAD_TO_STORAGE = 'UPLOAD_TO_STORAGE', 
  SKIP = 'SKIP',
}

export enum SplitMethod {
  NONE = 'NONE',
  ROW_COUNT = 'ROW_COUNT',
  FILE_SIZE_MB = 'FILE_SIZE_MB',
  DATE = 'DATE' // New split method
}

export enum DeduplicationStrategy {
  STRICT_HASH = 'STRICT_HASH', // Date + Sender + Body (Exact)
  FUZZY_BODY = 'FUZZY_BODY',   // Normalized Body (ignore whitespace/case)
  ID_ONLY = 'ID_ONLY'          // Only use Source ID (if available)
}

export enum TargetDb {
  SQLITE = 'SQLITE',
  POSTGRES = 'POSTGRES'
}

// Graph Configuration Types
export interface GraphNodeConfig {
  id: string;
  label: string;
  sourceColumn: string; // which XML column maps to the ID
  properties: string[]; // XML columns to include as props
}

export interface GraphEdgeConfig {
  id: string;
  type: string;
  sourceNodeId: string;
  targetNodeId: string;
  condition?: string; // Optional JS-like condition (e.g. "type == '1'")
  properties: string[];
}

export interface GraphConfig {
  nodes: GraphNodeConfig[];
  edges: GraphEdgeConfig[];
}

export type StreamSource = { type: 'FILE', file: File } | { type: 'DRIVE', url: string, name: string, token: string, size: number };

export interface ProjectMetadata {
  caseId?: string;
  investigator?: string;
  description?: string;
  customTags?: string[];
}

export interface ConversionConfig {
  // ETL Source Control
  sourceLabel: string; 

  base64Option: Base64Option;
  
  // Local Export
  exportLocalFile: boolean;
  exportSqliteFile: boolean; 
  exportNeo4j: boolean; 
  localExportFormat: TargetDb; 
  
  // Naming & splitting
  filenamePattern: string; // e.g., "{Source}_{Date}_{Seq}"
  splitMethod: SplitMethod;
  splitThreshold: number; 
  dateSplitGranularity: 'MONTH' | 'YEAR';

  // Cloud Stream
  streamToSupabase: boolean;
  
  // Supabase Credentials
  supabaseUrl?: string; 
  supabaseKey?: string; 
  supabaseAccessToken?: string; 
  
  // Google Drive Credentials
  googleDriveClientId?: string;

  // Table Names
  tableName: string;
  entityTable: string;
  attachmentTable: string;
  callsTable: string;
  separateCallsTable: boolean;
  storageBucket?: string; 

  // Integrity & Validation
  calculateHash: boolean;
  deduplicationStrategy: DeduplicationStrategy;
  generateUuid: boolean; 
  
  // Timezone
  humanReadableTime: boolean; 
  
  // Tag detection
  itemTag?: string;

  // Column Management
  columns: string[]; 

  // Graph Config
  graphConfig?: GraphConfig;

  // Project Level Metadata
  projectMetadata?: ProjectMetadata;
}

export interface SavedQuery {
  id: string;
  name: string;
  tables: string[];
  joins: {from: string, to: string, on: string}[];
  createdAt: number;
}

export interface AnalysisResult {
  summary: string;
  participants: string[];
  dateRange: string;
  topics: string[];
}

export interface ProcessingMetadata {
  originalFileName: string;
  fileSize: number;
  lastModified: number;
  startTime: string;
  endTime?: string;
  recordCount: number;
  minDate?: number;
  maxDate?: number;
  hash?: string;
  participants: Set<string>;
  validationErrors: number;
}

// File System Access API Types
export interface FileSystemHandle {
  kind: 'file' | 'directory';
  name: string;
}

export interface FileSystemDirectoryHandle extends FileSystemHandle {
  getFileHandle(name: string, options?: { create?: boolean }): Promise<FileSystemFileHandle>;
  getDirectoryHandle(name: string, options?: { create?: boolean }): Promise<FileSystemDirectoryHandle>;
}

export interface FileSystemFileHandle extends FileSystemHandle {
  createWritable(): Promise<FileSystemWritableFileStream>;
  getFile(): Promise<File>;
}

export interface FileSystemWritableFileStream extends WritableStream {
  write(data: string | BufferSource | Blob): Promise<void>;
  close(): Promise<void>;
}

declare global {
  interface Window {
    showDirectoryPicker(): Promise<FileSystemDirectoryHandle>;
    showSaveFilePicker(options?: any): Promise<FileSystemFileHandle>;
    showOpenFilePicker(options?: any): Promise<FileSystemFileHandle[]>;
    gapi: any;
    google: any;
  }
}
