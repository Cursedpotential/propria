/**
 * Browser-Native NLP Service - MICRO-ANALYSIS ONLY
 *
 * Lightweight tagging during ingestion for basic flags.
 * Macro-level pattern analysis (gaslighting, DARVO, etc.) happens LATER
 * after all data is centralized.
 *
 * Libraries:
 * - compromise: Fast entity extraction + simple word matching
 * - nlp.js: Sentiment analysis (AFINN + negation)
 *
 * What we tag:
 * - Sentiment (positive/negative/neutral)
 * - Entities (people, places, orgs mentioned)
 * - Simple content flags (apology, blame, threat, minimizing)
 * - Linguistic markers (questions, exclamations, caps, profanity)
 * - Basic stats (word count, char count)
 */

import nlp from 'compromise';
import { containerBootstrap } from '@nlpjs/core';
import { Nlp } from '@nlpjs/nlp';
import { LangEn } from '@nlpjs/lang-en-min';
import { RulesLoader, BehaviorRule } from './rulesLoader';

// Transformers.js for advanced mode (lazy loaded)
let transformersJs: any = null;

export type AnalysisMode = 'basic' | 'standard' | 'advanced';

export interface AnalysisModeConfig {
    mode: AnalysisMode;
    // Advanced mode options
    useTransformers?: boolean; // Enable BERT-based deep learning
    transformersModel?: string; // Default: 'Xenova/distilbert-base-uncased-finetuned-sst-2-english'
}

export interface Entity {
    entity_type: 'person' | 'location' | 'organization' | 'phone' | 'email' | 'date';
    name: string;
    normalized_name: string;
    mention_text: string;
    start_char: number;
    end_char: number;
    confidence: number;
}

export interface BehaviorMatch {
    category: string; // e.g., 'character_assassination', 'parental_alienation'
    matched_pattern: string; // The rule pattern that matched
    matched_text: string; // ACTUAL words used (e.g., "motherfucker", "psycho")
    start_char: number; // Position in message
    end_char: number;
    context_before: string; // 50 chars before
    context_after: string; // 50 chars after
    confidence: number;
    severity: 'low' | 'medium' | 'high' | 'critical';
    detection_method: 'rule_based' | 'regex' | 'nlp';
}

export interface LinguisticMarkers {
    contains_apology: boolean;
    contains_blame: boolean;
    contains_threat: boolean;
    contains_minimizing: boolean;
    question_count: number;
    exclamation_count: number;
    caps_ratio: number;
    sentiment: 'positive' | 'negative' | 'neutral';
}

export interface MessageAnalysis {
    message_id: string;
    entities: Entity[];
    behaviors: BehaviorMatch[];
    linguistic_markers: LinguisticMarkers;
    word_count: number;
    character_count: number;
}

// Behavioral Pattern Definitions (Salem v. Kinzel coercive control patterns)
const BEHAVIOR_PATTERNS = {
    gaslighting: {
        patterns: [
            /\b(never happened|didn'?t happen|you'?re crazy|making (it|things) up|imagining (it|things))\b/gi,
            /\b(not real|in your head|paranoid|delusional|losing it)\b/gi
        ],
        mcl_factors: ['F', 'G', 'K'],
        severity: 'high' as const
    },
    blame_shifting: {
        patterns: [
            /\b(your fault|you made me|because of you|you caused|you'?re the reason)\b/gi,
            /\b(if you (hadn'?t|didn'?t)|you should have|you shouldn'?t have)\b/gi
        ],
        mcl_factors: ['F', 'J'],
        severity: 'medium' as const
    },
    minimizing: {
        patterns: [
            /\b(not a big deal|overreacting|too sensitive|being dramatic)\b/gi,
            /\b(calm down|relax|chill out?|stop crying)\b/gi
        ],
        mcl_factors: ['F', 'K'],
        severity: 'medium' as const
    },
    love_bombing: {
        patterns: [
            /\b(can'?t live without|soulmate|meant to be|perfect (for|together))\b/gi,
            /\b(love you so much|everything to me|my (whole )?world)\b/gi
        ],
        mcl_factors: ['F'],
        severity: 'low' as const
    },
    stonewalling: {
        patterns: [
            /\b(not talking about|done discussing|conversation over|nothing to say)\b/gi,
            /\b(leave me alone|stop texting|don'?t want to hear)\b/gi
        ],
        mcl_factors: ['J'],
        severity: 'medium' as const
    },
    parental_alienation: {
        patterns: [
            /\b(doesn'?t care about you|never wanted|bad (parent|mom|dad|mother|father))\b/gi,
            /\b(doesn'?t love|abandoned you|left us|walked out)\b/gi
        ],
        mcl_factors: ['J', 'K'],
        severity: 'critical' as const
    },
    coercive_control: {
        patterns: [
            /\b(have to|must|better|or else|consequences)\b/gi,
            /\b(you will|you'?re going to|don'?t (make me|test me))\b/gi
        ],
        mcl_factors: ['F', 'K'],
        severity: 'high' as const
    },
    financial_abuse: {
        patterns: [
            /\b(your money|can'?t afford|pay for|you'?re broke)\b/gi,
            /\b(bills|rent|expenses|credit card|debt)\b/gi
        ],
        mcl_factors: ['C', 'F', 'K'],
        severity: 'high' as const
    },
    darvo: {
        patterns: [
            /\b(i'?m the victim|you'?re abusive|attacking me|hurting me)\b/gi,
            /\b(you'?re the (problem|abuser)|i'?m defending myself)\b/gi
        ],
        mcl_factors: ['F', 'K'],
        severity: 'critical' as const
    },
    character_assassination: {
        patterns: [
            /\b(crazy|unstable|unfit|dangerous|bad (parent|mom|dad))\b/gi,
            /\b(psycho|mental|insane|losing it|nuts)\b/gi
        ],
        mcl_factors: ['F', 'J'],
        severity: 'high' as const
    }
};

// Simple profanity wordlist for abusive language detection
const PROFANITY_PATTERNS = /\b(fuck|shit|damn|bitch|asshole|bastard|cunt|dick|pussy|whore|slut)\b/gi;

export class BrowserNLPService {
    private enabled: boolean = false; // Disabled by default - not used in current stage
    private rulesLoader: RulesLoader;
    private nlp: any = null;
    private mode: AnalysisMode = 'standard'; // Default: compromise + @nlpjs
    private transformersPipeline: any = null;

    constructor(rulesPath?: string, mode: AnalysisMode = 'standard') {
        // Build and ready but not initialized yet
        this.rulesLoader = RulesLoader.getInstance(rulesPath);
        this.mode = mode;
        console.log(`🧠 NLP Mode: ${mode}`);
    }

    /**
     * Set analysis mode
     * - basic: compromise only (fastest)
     * - standard: compromise + @nlpjs sentiment (balanced) - DEFAULT
     * - advanced: + Transformers.js BERT models (slow, high accuracy)
     */
    setMode(mode: AnalysisMode) {
        this.mode = mode;
        console.log(`🧠 NLP Mode changed to: ${mode}`);
    }

    getMode(): AnalysisMode {
        return this.mode;
    }

    async initialize() {
        // Standard mode: Initialize @nlpjs for sentiment
        if (this.mode === 'standard' || this.mode === 'advanced') {
            try {
                const container = await containerBootstrap();
                container.use(Nlp);
                container.use(LangEn);
                this.nlp = container.get('nlp');
                this.nlp.settings.autoSave = false;
                await this.nlp.train();
                console.log('✅ @nlpjs sentiment ready');
            } catch (error) {
                console.warn('⚠️  @nlpjs failed, using fallback:', error);
                this.nlp = null;
            }
        }

        // Advanced mode: Initialize Transformers.js (lazy loaded)
        if (this.mode === 'advanced') {
            try {
                console.log('🚀 Loading Transformers.js (this may take a moment)...');
                if (!transformersJs) {
                    transformersJs = await import('@xenova/transformers');
                }

                // Initialize sentiment analysis pipeline with DistilBERT
                this.transformersPipeline = await transformersJs.pipeline(
                    'sentiment-analysis',
                    'Xenova/distilbert-base-uncased-finetuned-sst-2-english'
                );

                console.log('✅ Transformers.js BERT model loaded (Advanced Mode)');
            } catch (error) {
                console.warn('⚠️  Transformers.js failed, falling back to standard:', error);
                this.transformersPipeline = null;
            }
        }

        console.log(`✅ NLP service ready (Mode: ${this.mode})`);
    }

    /**
     * Extract entities using compromise.js
     */
    extractEntities(text: string): Entity[] {
        if (!text) return [];

        const entities: Entity[] = [];
        const doc = nlp(text);

        // Extract people
        const people = doc.people().json();
        people.forEach((person: any) => {
            entities.push({
                entity_type: 'person',
                name: person.text,
                normalized_name: person.text.trim(),
                mention_text: person.text,
                start_char: person.offset?.start || 0,
                end_char: person.offset?.start + person.text.length || person.text.length,
                confidence: 0.85
            });
        });

        // Extract places
        const places = doc.places().json();
        places.forEach((place: any) => {
            entities.push({
                entity_type: 'location',
                name: place.text,
                normalized_name: place.text.trim(),
                mention_text: place.text,
                start_char: place.offset?.start || 0,
                end_char: place.offset?.start + place.text.length || place.text.length,
                confidence: 0.85
            });
        });

        // Extract organizations
        const orgs = doc.organizations().json();
        orgs.forEach((org: any) => {
            entities.push({
                entity_type: 'organization',
                name: org.text,
                normalized_name: org.text.trim(),
                mention_text: org.text,
                start_char: org.offset?.start || 0,
                end_char: org.offset?.start + org.text.length || org.text.length,
                confidence: 0.85
            });
        });

        // Extract dates
        const dates = doc.dates().json();
        dates.forEach((date: any) => {
            entities.push({
                entity_type: 'date',
                name: date.text,
                normalized_name: date.text.trim(),
                mention_text: date.text,
                start_char: date.offset?.start || 0,
                end_char: date.offset?.start + date.text.length || date.text.length,
                confidence: 0.90
            });
        });

        // Extract phone numbers (regex)
        const phonePattern = /\b\d{3}[-.]?\d{3}[-.]?\d{4}\b|\b\(\d{3}\)\s?\d{3}[-.]?\d{4}\b/g;
        let phoneMatch;
        while ((phoneMatch = phonePattern.exec(text)) !== null) {
            entities.push({
                entity_type: 'phone',
                name: phoneMatch[0],
                normalized_name: phoneMatch[0].replace(/\D/g, ''),
                mention_text: phoneMatch[0],
                start_char: phoneMatch.index,
                end_char: phoneMatch.index + phoneMatch[0].length,
                confidence: 0.95
            });
        }

        // Extract emails (regex)
        const emailPattern = /\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b/g;
        let emailMatch;
        while ((emailMatch = emailPattern.exec(text)) !== null) {
            entities.push({
                entity_type: 'email',
                name: emailMatch[0],
                normalized_name: emailMatch[0].toLowerCase(),
                mention_text: emailMatch[0],
                start_char: emailMatch.index,
                end_char: emailMatch.index + emailMatch[0].length,
                confidence: 0.95
            });
        }

        return entities;
    }

    /**
     * Detect simple content flags (NOT complex behavioral patterns)
     * Just quick micro-level tags for later macro analysis
     */
    private detectSimpleFlags(text: string): {
        contains_apology: boolean;
        contains_blame: boolean;
        contains_threat: boolean;
        contains_minimizing: boolean;
        contains_profanity: boolean;
    } {
        if (!text) {
            return {
                contains_apology: false,
                contains_blame: false,
                contains_threat: false,
                contains_minimizing: false,
                contains_profanity: false
            };
        }

        return {
            contains_apology: /\b(sorry|apologize|my bad|my fault)\b/i.test(text),
            contains_blame: /\b(your fault|you made|because of you)\b/i.test(text),
            contains_threat: /\b(or else|you'?ll (regret|see)|watch out)\b/i.test(text),
            contains_minimizing: /\b(not a big deal|overreacting|too sensitive|dramatic)\b/i.test(text),
            contains_profanity: PROFANITY_PATTERNS.test(text)
        };
    }

    /**
     * Analyze linguistic markers using compromise.js
     */
    async analyzeLinguisticMarkers(text: string): Promise<LinguisticMarkers> {
        if (!text) {
            return {
                contains_apology: false,
                contains_blame: false,
                contains_threat: false,
                contains_minimizing: false,
                question_count: 0,
                exclamation_count: 0,
                caps_ratio: 0,
                sentiment: 'neutral'
            };
        }

        return {
            contains_apology: /\b(sorry|apologize|my bad|my fault)\b/i.test(text),
            contains_blame: /\b(your fault|you made|because of you)\b/i.test(text),
            contains_threat: /\b(or else|you'?ll (regret|see)|watch out|i'?ll make you)\b/i.test(text),
            contains_minimizing: /\b(not a big deal|overreacting|too sensitive|dramatic)\b/i.test(text),
            question_count: (text.match(/\?/g) || []).length,
            exclamation_count: (text.match(/!/g) || []).length,
            caps_ratio: Math.round((text.split('').filter(c => /[A-Z]/.test(c)).length / text.length) * 100) / 100,
            sentiment: await this.detectSentiment(text)
        };
    }

    /**
     * Multi-tier sentiment detection
     * - Basic: Regex patterns only
     * - Standard: @nlpjs AFINN + negation
     * - Advanced: Transformers.js BERT model
     */
    private async detectSentiment(text: string): Promise<'positive' | 'negative' | 'neutral'> {
        if (!text) return 'neutral';

        // ADVANCED MODE: Use Transformers.js BERT model
        if (this.mode === 'advanced' && this.transformersPipeline) {
            try {
                const result = await this.transformersPipeline(text);
                const label = result[0]?.label?.toLowerCase();
                const score = result[0]?.score || 0;

                // BERT returns 'positive' or 'negative' with confidence
                if (label === 'positive' && score > 0.6) return 'positive';
                if (label === 'negative' && score > 0.6) return 'negative';
                return 'neutral';
            } catch (error) {
                console.warn('Transformers.js sentiment failed, falling back');
            }
        }

        // STANDARD MODE: Use @nlpjs (AFINN + negation)
        if ((this.mode === 'standard' || this.mode === 'advanced') && this.nlp) {
            try {
                const result = await this.nlp.process('en', text);
                const score = result.sentiment?.score || 0;

                if (score > 0.1) return 'positive';
                if (score < -0.1) return 'negative';
                return 'neutral';
            } catch (error) {
                console.warn('@nlpjs sentiment failed, using basic fallback');
            }
        }

        // BASIC MODE (or fallback): Custom regex patterns
        const positive = /\b(love|happy|great|good|wonderful|amazing|perfect|thank|appreciate|excited|joy|fantastic|excellent|beautiful|pleased|glad|better|best)\b/gi;
        const negative = /\b(hate|angry|terrible|awful|worst|stupid|idiot|disgusting|never|nothing|horrible|bad|sad|mad|upset|annoyed|frustrated|pathetic|useless|worthless)\b/gi;
        const strongNegative = /\b(fuck|shit|damn|bitch|asshole|bastard|cunt|motherfucker)\b/gi;

        let score = 0;
        score += (text.match(positive) || []).length;
        score -= (text.match(negative) || []).length;
        score -= (text.match(strongNegative) || []).length * 2;

        // Handle negation
        const negation = /\b(not|no|never)\s+(good|great|happy|wonderful|amazing|perfect|bad|terrible|awful|horrible|worst)\b/gi;
        const negMatches = text.match(negation) || [];
        score -= negMatches.length;

        if (score >= 2) return 'positive';
        if (score <= -2) return 'negative';
        return 'neutral';
    }

    /**
     * Detect behavioral patterns using rules from ConflictAnalysisApp
     * Captures ACTUAL matched text (e.g., "motherfucker", "crackhead")
     */
    async detectBehaviors(text: string): Promise<BehaviorMatch[]> {
        if (!text) return [];

        const matches: BehaviorMatch[] = [];
        const behaviorPatterns = await this.rulesLoader.getBehaviorPatterns();

        // Severity mapping (based on Salem case context)
        const severityMap: Record<string, 'low' | 'medium' | 'high' | 'critical'> = {
            manipulation_gaslighting: 'high',
            projection_blameshift: 'medium',
            victimhood_narrative: 'medium',
            certainty_absolutism: 'low',
            character_assassination: 'high',
            coercive_control: 'critical',
            stonewalling_blocking: 'medium',
            last_minute_plan_change: 'medium',
            parental_alienation: 'critical',
            financial_abuse: 'high',
            healthcare_denial: 'high',
            sexual_weaponization: 'high',
            love_bombing: 'low',
            feigned_incompetence: 'low',
            social_media_deception: 'medium',
            infidelity_betrayal: 'high',
            substance_weaponization: 'critical',
            stimulant_control: 'high',
            alcohol_risk_child: 'critical',
            reactive_abuse_gotcha: 'high',
            dismissiveness: 'low'
        };

        for (const [category, patterns] of Object.entries(behaviorPatterns)) {
            const result = this.rulesLoader.matchesPattern(text, patterns);

            if (result.matched && result.matchedText) {
                const lowerText = text.toLowerCase();
                const matchIndex = lowerText.indexOf(result.matchedText.toLowerCase());

                if (matchIndex !== -1) {
                    const startChar = matchIndex;
                    const endChar = matchIndex + result.matchedText.length;

                    // Extract context (50 chars before/after)
                    const contextStart = Math.max(0, startChar - 50);
                    const contextEnd = Math.min(text.length, endChar + 50);
                    const contextBefore = text.substring(contextStart, startChar);
                    const contextAfter = text.substring(endChar, contextEnd);

                    matches.push({
                        category,
                        matched_pattern: result.matchedText,
                        matched_text: text.substring(startChar, endChar), // Preserve original case
                        start_char: startChar,
                        end_char: endChar,
                        context_before: contextBefore,
                        context_after: contextAfter,
                        confidence: 0.90, // Rule-based matches are high confidence
                        severity: severityMap[category] || 'medium',
                        detection_method: typeof patterns[0] === 'string' ? 'rule_based' : 'regex'
                    });
                }
            }
        }

        return matches;
    }

    /**
     * Analyze a single message
     */
    async analyzeMessage(messageId: string, content: string): Promise<MessageAnalysis> {
        return {
            message_id: messageId,
            entities: this.extractEntities(content),
            behaviors: await this.detectBehaviors(content),
            linguistic_markers: await this.analyzeLinguisticMarkers(content),
            word_count: content ? content.split(/\s+/).length : 0,
            character_count: content ? content.length : 0
        };
    }

    /**
     * Analyze a batch of messages (async for large batches)
     */
    async analyzeBatch(messages: { id: string; content?: string }[]): Promise<MessageAnalysis[]> {
        const results: MessageAnalysis[] = [];

        for (const msg of messages) {
            results.push(await this.analyzeMessage(msg.id, msg.content || ''));
        }

        return results;
    }

    setEnabled(enabled: boolean) {
        this.enabled = enabled;
    }

    isEnabled(): boolean {
        return this.enabled;
    }
}
