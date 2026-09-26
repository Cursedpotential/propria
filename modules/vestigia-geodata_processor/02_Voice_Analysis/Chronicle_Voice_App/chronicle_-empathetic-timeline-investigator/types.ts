
export interface EvidenceTag {
  id: string;
  name: string; 
  type: string; 
  fileReference?: string; 
}

export type EventCategory = 
  | 'Childhood Memory' 
  | 'Career/Professional' 
  | 'Historical Trauma' 
  | 'Current Case Incident' 
  | 'Legal Milestone' 
  | 'Personal Growth' 
  | 'Strength/Resilience'
  | 'Medical/Psychiatric' 
  | 'Financial Abuse' 
  | 'Relationship Phase';

// PROVENANCE: This ensures we know exactly which app generated this data.
export type DataOrigin = 'Chronicle' | 'Sentinel' | 'Anamnesis' | 'External_Import';
export type VerificationStatus = 'Verified' | 'Proposed' | 'Contested' | 'Conflicted';

export interface ForensicAnalysis {
  sourceType: 'Surveillance Video' | 'Audio Recording' | 'Transcript' | 'Document' | 'AI Chat Log';
  sourceFileRef?: string; 
  deceptionScore?: number; 
  sentiment?: 'Hostile' | 'Neutral' | 'Supportive' | 'Distressed' | 'Deceptive';
  microExpressions?: string[]; 
  voiceStressAnalysis?: string; 
  transcriptExcerpt?: string;
  processedAt: number;
}

export interface TimelineEvent {
  id: string;
  date: string;
  category: EventCategory;
  description: string;
  
  // PROVENANCE FIELDS
  origin: DataOrigin; 
  verificationStatus: VerificationStatus;
  conflictId?: string; 
  
  rawQuotes?: string[];
  mood?: string;
  
  location?: string;
  childPresence: 'Unknown' | 'Not Present' | 'Present - Asleep' | 'Present - Awake' | 'Directly Involved';
  witnesses?: string[];
  involvedEntities: string[];

  emotionalContext?: string;
  reactionContext?: string;
  vulnerability?: string;
  aiAnalysis?: string;
  
  forensicAnalysis?: ForensicAnalysis;
  
  historicalTraumaLink?: string;
  manipulationPattern?: string;
  weaponization?: string; 
  
  courtExplanation?: string;
  evidenceStatus: 'Have' | 'Need';
  evidenceStrength: 'Weak' | 'Moderate' | 'Strong' | 'Conclusive';
  evidenceTags: EvidenceTag[];
  evidenceLinks?: string[];
  
  isSignificant: boolean;
  createdAt: number;
}

export interface Entity {
  id: string;
  name: string;
  type: 'person' | 'location' | 'organization' | 'workplace' | 'school' | 'medical' | 'other';
  role?: string; 
  relationship?: string; 
  impactOnCase?: 'high' | 'medium' | 'low';
  frequency?: string; 
  notes: string;
  
  // PROVENANCE
  origin: DataOrigin; 
}

export interface EntityRelationship {
  id: string;
  sourceId: string;
  targetId: string;
  label: string; 
  type: 'familial' | 'romantic' | 'professional' | 'conflict' | 'support';
  startDate?: string;
  endDate?: string;
  notes?: string;
  
  // PROVENANCE
  origin: DataOrigin; 
}

export interface ContextFact {
  id: string;
  category: 'Personal History' | 'Medical' | 'Family Dynamics' | 'Financial Context';
  content: string;
  source: string;
  relevance: string;
  origin: DataOrigin;
}

export interface ContextPhase {
  id: string;
  name: string; 
  category: 'Life Era' | 'Relationship' | 'Professional' | 'Medical' | 'Legal' | 'Geographic';
  startDate: string; 
  endDate: string; 
  emotionalBaseline: string; 
  description: string;
  notes: string;
}

export interface GlobalNote {
  id: string;
  title: string;
  content: string;
  category: 'Strategy' | 'Theory' | 'Observation' | 'Draft' | 'Reflection' | 'Memory' | 'General' | 'Strength' | 'Vulnerability';
  createdAt: number;
  lastUpdated: number;
}

export interface AISettings {
  focus: 'Legal Strategy' | 'Empathetic Listener' | 'Fact Extraction' | 'Self Discovery';
  strictness: 'Strict/Literal' | 'Interpretive';
  liveModel: string;
  analysisModel: string;
}

export interface ConnectionSettings {
  weaviateUrl?: string;
  weaviateApiKey?: string;
  neo4jUrl?: string;
  neo4jUser?: string;
  neo4jPassword?: string;
}

export interface AppState {
  events: TimelineEvent[];
  entities: Entity[];
  entityRelationships: EntityRelationship[]; 
  contextFacts: ContextFact[];
  contextPhases: ContextPhase[]; 
  globalNotes: GlobalNote[];
  isSessionActive: boolean;
  caseContext: string;
  currentTab: 'recall' | 'context' | 'timeline' | 'graph' | 'reports' | 'vault';
  transcripts: string[];
  aiSettings: AISettings;
  connectionSettings: ConnectionSettings;
}
