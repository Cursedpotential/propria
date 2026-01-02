"""
Salem Forensic NLP Service
FastAPI backend for entity extraction and behavioral pattern detection
Uses spaCy and regex patterns to avoid expensive LLM calls
"""

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import List, Dict, Any, Optional
import spacy
import re
from datetime import datetime

app = FastAPI(title="Salem Forensic NLP Service")

# CORS for browser access
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Lock this down in production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Load spaCy model (install: python -m spacy download en_core_web_sm)
try:
    nlp = spacy.load("en_core_web_sm")
except:
    print("⚠️  spaCy model not found. Install with: python -m spacy download en_core_web_sm")
    nlp = None

# Behavioral Pattern Definitions (from Salem schema)
BEHAVIOR_PATTERNS = {
    'gaslighting': {
        'patterns': [
            r'\b(never happened|didn\'?t happen|you\'?re crazy|making (it|things) up|imagining (it|things))\b',
            r'\b(not real|in your head|paranoid|delusional)\b'
        ],
        'mcl_factors': ['F', 'G', 'K'],
        'severity_default': 'high'
    },
    'blame_shifting': {
        'patterns': [
            r'\b(your fault|you made me|because of you|you caused|you\'?re the reason)\b',
            r'\b(if you (hadn\'?t|didn\'?t)|you should have)\b'
        ],
        'mcl_factors': ['F', 'J'],
        'severity_default': 'medium'
    },
    'minimizing': {
        'patterns': [
            r'\b(not a big deal|overreacting|too sensitive|being dramatic)\b',
            r'\b(calm down|relax|chill|stop crying)\b'
        ],
        'mcl_factors': ['F', 'K'],
        'severity_default': 'medium'
    },
    'love_bombing': {
        'patterns': [
            r'\b(can\'?t live without|soulmate|meant to be|perfect (for|together))\b',
            r'\b(love you so much|everything to me|my world)\b'
        ],
        'mcl_factors': ['F'],
        'severity_default': 'low'
    },
    'stonewalling': {
        'patterns': [
            r'\b(not talking about|done discussing|conversation over|nothing to say)\b',
            r'\b(leave me alone|stop texting|don\'?t want to hear)\b'
        ],
        'mcl_factors': ['J'],
        'severity_default': 'medium'
    },
    'parental_alienation': {
        'patterns': [
            r'\b(doesn\'?t care about you|never wanted|bad (parent|mom|dad))\b',
            r'\b(doesn\'?t love|abandoned you|left us)\b'
        ],
        'mcl_factors': ['J', 'K'],
        'severity_default': 'critical'
    },
    'coercive_control': {
        'patterns': [
            r'\b(have to|must|better|or else|consequences)\b',
            r'\b(you will|you\'?re going to|don\'?t (make me|test me))\b'
        ],
        'mcl_factors': ['F', 'K'],
        'severity_default': 'high'
    },
    'financial_abuse': {
        'patterns': [
            r'\b(your money|can\'?t afford|pay for|broke|need money)\b',
            r'\b(bills|rent|expenses|credit card)\b'
        ],
        'mcl_factors': ['C', 'F', 'K'],
        'severity_default': 'high'
    },
    'darvo': {
        'patterns': [
            r'\b(i\'?m the victim|you\'?re abusive|attacking me|hurting me)\b',
            r'\b(you\'?re the (problem|abuser)|i\'?m defending myself)\b'
        ],
        'mcl_factors': ['F', 'K'],
        'severity_default': 'critical'
    },
    'character_assassination': {
        'patterns': [
            r'\b(crazy|unstable|unfit|dangerous|bad (parent|mom|dad))\b',
            r'\b(psycho|mental|insane|losing it)\b'
        ],
        'mcl_factors': ['F', 'J'],
        'severity_default': 'high'
    }
}

class Message(BaseModel):
    id: str
    content: Optional[str]
    sender: Optional[str]
    recipient: Optional[str]
    timestamp: Optional[str]

class BatchRequest(BaseModel):
    messages: List[Message]

class Entity(BaseModel):
    entity_type: str  # 'person', 'location', 'organization', 'phone', 'email'
    name: str
    normalized_name: str
    mention_text: str
    start_char: int
    end_char: int
    confidence: float

class Behavior(BaseModel):
    category: str
    matched_pattern: str
    matched_text: str
    start_char: int
    end_char: int
    context_before: str
    context_after: str
    confidence: float
    severity: str
    detection_method: str

class MessageAnalysis(BaseModel):
    message_id: str
    entities: List[Entity]
    behaviors: List[Behavior]
    linguistic_markers: Dict[str, Any]
    word_count: int
    character_count: int

class BatchResponse(BaseModel):
    analyses: List[MessageAnalysis]
    processing_time_ms: int

def extract_entities(text: str, message_id: str) -> List[Entity]:
    """Extract named entities using spaCy NER"""
    if not nlp or not text:
        return []

    entities = []
    doc = nlp(text)

    for ent in doc.ents:
        entity_type_map = {
            'PERSON': 'person',
            'GPE': 'location',
            'LOC': 'location',
            'ORG': 'organization',
            'FAC': 'location'
        }

        entity_type = entity_type_map.get(ent.label_, None)
        if entity_type:
            entities.append(Entity(
                entity_type=entity_type,
                name=ent.text,
                normalized_name=ent.text.strip().title(),
                mention_text=ent.text,
                start_char=ent.start_char,
                end_char=ent.end_char,
                confidence=0.85  # spaCy doesn't provide scores for base model
            ))

    # Extract phone numbers (regex)
    phone_pattern = r'\b\d{3}[-.]?\d{3}[-.]?\d{4}\b|\b\(\d{3}\)\s?\d{3}[-.]?\d{4}\b'
    for match in re.finditer(phone_pattern, text):
        entities.append(Entity(
            entity_type='phone',
            name=match.group(),
            normalized_name=re.sub(r'\D', '', match.group()),
            mention_text=match.group(),
            start_char=match.start(),
            end_char=match.end(),
            confidence=0.95
        ))

    # Extract emails (regex)
    email_pattern = r'\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b'
    for match in re.finditer(email_pattern, text):
        entities.append(Entity(
            entity_type='email',
            name=match.group(),
            normalized_name=match.group().lower(),
            mention_text=match.group(),
            start_char=match.start(),
            end_char=match.end(),
            confidence=0.95
        ))

    return entities

def detect_behaviors(text: str, message_id: str) -> List[Behavior]:
    """Detect behavioral patterns using regex"""
    if not text:
        return []

    behaviors = []
    text_lower = text.lower()

    for category, config in BEHAVIOR_PATTERNS.items():
        for pattern_str in config['patterns']:
            pattern = re.compile(pattern_str, re.IGNORECASE)

            for match in pattern.finditer(text):
                # Get context (50 chars before/after)
                start = max(0, match.start() - 50)
                end = min(len(text), match.end() + 50)
                context_before = text[start:match.start()]
                context_after = text[match.end():end]

                # Calculate confidence based on pattern strength
                confidence = 0.7  # Base confidence for regex match
                if len(match.group()) > 10:
                    confidence += 0.1  # Longer matches more reliable
                if re.search(r'\b(always|never|every time)\b', text_lower):
                    confidence += 0.1  # Absolutist language increases confidence

                confidence = min(1.0, confidence)

                behaviors.append(Behavior(
                    category=category,
                    matched_pattern=pattern_str,
                    matched_text=match.group(),
                    start_char=match.start(),
                    end_char=match.end(),
                    context_before=context_before,
                    context_after=context_after,
                    confidence=round(confidence, 2),
                    severity=config['severity_default'],
                    detection_method='regex'
                ))

    return behaviors

def analyze_linguistic_markers(text: str) -> Dict[str, Any]:
    """Analyze linguistic patterns"""
    if not text:
        return {
            'contains_apology': False,
            'contains_blame': False,
            'contains_threat': False,
            'contains_minimizing': False,
            'question_count': 0,
            'exclamation_count': 0,
            'caps_ratio': 0.0
        }

    return {
        'contains_apology': bool(re.search(r'\b(sorry|apologize|my bad)\b', text, re.IGNORECASE)),
        'contains_blame': bool(re.search(r'\b(your fault|you made|because of you)\b', text, re.IGNORECASE)),
        'contains_threat': bool(re.search(r'\b(or else|you\'?ll (regret|see)|watch out)\b', text, re.IGNORECASE)),
        'contains_minimizing': bool(re.search(r'\b(not a big deal|overreacting|too sensitive)\b', text, re.IGNORECASE)),
        'question_count': text.count('?'),
        'exclamation_count': text.count('!'),
        'caps_ratio': round(sum(1 for c in text if c.isupper()) / len(text) if text else 0, 2)
    }

@app.post("/analyze/batch", response_model=BatchResponse)
async def analyze_batch(request: BatchRequest):
    """Analyze a batch of messages for entities and behavioral patterns"""
    start_time = datetime.now()

    analyses = []

    for msg in request.messages:
        content = msg.content or ""

        analysis = MessageAnalysis(
            message_id=msg.id,
            entities=extract_entities(content, msg.id),
            behaviors=detect_behaviors(content, msg.id),
            linguistic_markers=analyze_linguistic_markers(content),
            word_count=len(content.split()) if content else 0,
            character_count=len(content) if content else 0
        )

        analyses.append(analysis)

    processing_time = (datetime.now() - start_time).total_seconds() * 1000

    return BatchResponse(
        analyses=analyses,
        processing_time_ms=int(processing_time)
    )

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "spacy_loaded": nlp is not None,
        "behavior_patterns": len(BEHAVIOR_PATTERNS)
    }

if __name__ == "__main__":
    import uvicorn
    print("🚀 Starting Salem Forensic NLP Service on http://localhost:8000")
    print("📊 Loaded behavior patterns:", list(BEHAVIOR_PATTERNS.keys()))
    uvicorn.run(app, host="0.0.0.0", port=8000)
