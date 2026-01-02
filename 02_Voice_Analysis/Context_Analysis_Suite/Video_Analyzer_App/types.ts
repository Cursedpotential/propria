
// --- Core Data Schema from Hand-Off Packet ---

export interface TimelineEvent {
  id: string;             // UUID
  date: string;           // Flexible string: "1999", "Oct 12, 2024", "Summer 2005"
  description: string;    // Main narrative
  category: string;       // Enum: 'Childhood Memory' | 'Current Case Incident' | 'Historical Trauma' | etc.
  
  // CRITICAL DATA POINTS
  location: string;       // MANDATORY. Specific place (e.g., "Kitchen", "Courtroom").
  witnesses: string[];    // Array of names present.
  involvedEntities: string[]; // IDs of Entity objects involved.
  
  // ANALYSIS FLAGS
  isSignificant: boolean; // "The Gold Star" - Major turning point.
  manipulationPattern: string; // e.g., "DARVO", "Gaslighting".
  evidenceStrength: 'Weak' | 'Moderate' | 'Strong' | 'Conclusive';
  
  // AI DERIVED DATA
  aiAnalysis: string;     // Stored micro-analysis from Gemini.
  forensicAnalysis?: ForensicAnalysis; // From this app
}

export interface Entity {
  id: string;             // UUID
  name: string;           // Display Name
  type: 'person' | 'location' | 'organization';
  relationship: string;   // Relation to the Subject (User).
  impactOnCase: 'high' | 'medium' | 'low';
  notes: string;          // Character profile/background.
}

export interface EntityRelationship {
  id: string;
  sourceId: string;       // Entity UUID
  targetId: string;       // Entity UUID
  label: string;          // e.g., "Married", "Employed By", "Abused"
  type: 'familial' | 'romantic' | 'professional' | 'conflict';
  startDate?: string;
  endDate?: string;
}

export interface GlobalNote {
  id: string;
  title: string;
  content: string;
  category: 'Strategy' | 'Theory' | 'Observation' | 'Vulnerability';
}

export interface ForensicAnalysis {
  sourceType: 'Surveillance Video' | 'Audio Recording' | 'Transcript' | 'Document';
  sourceFileRef?: string; // ID or URL of the original file
  deceptionScore?: number; // 0-100 Confidence Score
  sentiment?: 'Hostile' | 'Neutral' | 'Supportive' | 'Distressed' | 'Deceptive';
  microExpressions?: string[]; // e.g. ["Contempt", "Fear", "Duping Delight"]
  voiceStressAnalysis?: string; // e.g. "High stress detected at 04:32"
  transcriptExcerpt?: string; // Key quote associated with this event
  processedAt: number;
}

export interface CaseData {
    events: TimelineEvent[];
    entities: Entity[];
    entityRelationships: EntityRelationship[];
    globalNotes: GlobalNote[];
}


// --- Internal Application Types ---

export interface TranscriptEntry {
  speaker: string;
  text: string;
  timestamp: string;
}

export interface AnalysisReport {
  summary: string;
  entities: {
    name: string;
    description: string;
  }[];
  moodAnalysis: {
    overallMood: string;
    confidence: number;
    bodyLanguage: string;
    emotionalConsistency: string;
  };
  events: {
    timestamp: string;
    description: string;
  }[];
  transcript?: TranscriptEntry[];
}

export type LoadingPhase = 
  | 'idle'
  | 'initializing'
  | 'extractingFrames'
  | 'extractingAudio'
  | 'analyzing'
  | 'advancedAnalysis'
  | 'compilingReport'
  | 'done';

export type VideoStatus = 'Pending' | 'Processing' | 'Analyzed' | 'Verified' | 'Error';

export interface StagedVideo {
  id: string;
  file: File;
  videoUrl: string;
  status: VideoStatus;
  report?: AnalysisReport; // Raw AI output
  
  // Editable fields for the final TimelineEvent
  description: string;
  category: string;
  location: string;
  isSignificant: boolean;
  manipulationPattern: string;
  evidenceStrength: 'Weak' | 'Moderate' | 'Strong' | 'Conclusive';
  forensicAnalysis?: ForensicAnalysis;
}

// --- Settings Types ---
export interface ApiKeys {
  google: string;
  openai: string; // Placeholder for future extension
}

export interface ModelPreferences {
  basic: string; // Model for initial, "naive" analysis
  advanced: string; // Model for deep, forensic analysis
}

export interface AppSettings {
  apiKeys: ApiKeys;
  modelPreferences: ModelPreferences;
}
