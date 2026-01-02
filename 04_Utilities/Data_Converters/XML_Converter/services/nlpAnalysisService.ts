/**
 * NLP Analysis Service
 * Connects browser to local Python NLP service for entity extraction and behavioral analysis
 */

export interface Entity {
    entity_type: 'person' | 'location' | 'organization' | 'phone' | 'email';
    name: string;
    normalized_name: string;
    mention_text: string;
    start_char: number;
    end_char: number;
    confidence: number;
}

export interface Behavior {
    category: string;
    matched_pattern: string;
    matched_text: string;
    start_char: number;
    end_char: number;
    context_before: string;
    context_after: string;
    confidence: number;
    severity: 'low' | 'medium' | 'high' | 'critical';
    detection_method: string;
}

export interface LinguisticMarkers {
    contains_apology: boolean;
    contains_blame: boolean;
    contains_threat: boolean;
    contains_minimizing: boolean;
    question_count: number;
    exclamation_count: number;
    caps_ratio: number;
}

export interface MessageAnalysis {
    message_id: string;
    entities: Entity[];
    behaviors: Behavior[];
    linguistic_markers: LinguisticMarkers;
    word_count: number;
    character_count: number;
}

export interface BatchAnalysisRequest {
    messages: {
        id: string;
        content?: string;
        sender?: string;
        recipient?: string;
        timestamp?: string;
    }[];
}

export interface BatchAnalysisResponse {
    analyses: MessageAnalysis[];
    processing_time_ms: number;
}

export class NLPAnalysisService {
    private baseUrl: string;
    private enabled: boolean;

    constructor(baseUrl: string = 'http://localhost:8000', enabled: boolean = true) {
        this.baseUrl = baseUrl;
        this.enabled = enabled;
    }

    async healthCheck(): Promise<boolean> {
        if (!this.enabled) return false;

        try {
            const response = await fetch(`${this.baseUrl}/health`, {
                method: 'GET',
                signal: AbortSignal.timeout(3000)
            });

            if (response.ok) {
                const data = await response.json();
                return data.status === 'healthy' && data.spacy_loaded === true;
            }
            return false;
        } catch (error) {
            console.warn('NLP service not available:', error);
            return false;
        }
    }

    async analyzeBatch(request: BatchAnalysisRequest): Promise<BatchAnalysisResponse | null> {
        if (!this.enabled) return null;

        try {
            const response = await fetch(`${this.baseUrl}/analyze/batch`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(request),
                signal: AbortSignal.timeout(30000) // 30 second timeout for batch
            });

            if (!response.ok) {
                console.error('NLP analysis failed:', response.statusText);
                return null;
            }

            return await response.json();
        } catch (error) {
            console.warn('NLP analysis error:', error);
            return null;
        }
    }

    setEnabled(enabled: boolean) {
        this.enabled = enabled;
    }

    isEnabled(): boolean {
        return this.enabled;
    }
}
