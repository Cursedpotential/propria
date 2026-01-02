# Salem Forensic NLP Service

Local Python service for entity extraction and behavioral pattern detection using spaCy and regex patterns.

## Quick Start

### Windows

```bash
cd python-backend
start_nlp_service.bat
```

This will:
1. Create a Python virtual environment (if needed)
2. Install dependencies (FastAPI, spaCy, etc.)
3. Download spaCy English model
4. Start the service on http://localhost:8000

### Manual Setup

```bash
# Create virtual environment
python -m venv venv

# Activate (Windows)
venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt

# Download spaCy model
python -m spacy download en_core_web_sm

# Start service
python nlp_service.py
```

## API Endpoints

### POST /analyze/batch

Analyze a batch of messages for entities and behavioral patterns.

**Request:**
```json
{
  "messages": [
    {
      "id": "msg-123",
      "content": "You're crazy, that never happened!",
      "sender": "+15551234567",
      "recipient": "+15559876543",
      "timestamp": "2024-01-15T10:30:00Z"
    }
  ]
}
```

**Response:**
```json
{
  "analyses": [
    {
      "message_id": "msg-123",
      "entities": [
        {
          "entity_type": "phone",
          "name": "+15551234567",
          "normalized_name": "15551234567",
          "mention_text": "+15551234567",
          "start_char": 0,
          "end_char": 12,
          "confidence": 0.95
        }
      ],
      "behaviors": [
        {
          "category": "gaslighting",
          "matched_pattern": "\\b(never happened|you're crazy)\\b",
          "matched_text": "You're crazy, that never happened",
          "start_char": 0,
          "end_char": 33,
          "context_before": "",
          "context_after": "!",
          "confidence": 0.8,
          "severity": "high",
          "detection_method": "regex"
        }
      ],
      "linguistic_markers": {
        "contains_apology": false,
        "contains_blame": false,
        "contains_threat": false,
        "contains_minimizing": false,
        "question_count": 0,
        "exclamation_count": 1,
        "caps_ratio": 0.03
      },
      "word_count": 5,
      "character_count": 34
    }
  ],
  "processing_time_ms": 45
}
```

### GET /health

Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "spacy_loaded": true,
  "behavior_patterns": 10
}
```

## Behavioral Patterns Detected

The service detects 10 coercive control and manipulation patterns:

1. **Gaslighting** (MCL Factors: F, G, K) - "never happened", "you're crazy", "making it up"
2. **Blame Shifting** (F, J) - "your fault", "you made me", "because of you"
3. **Minimizing** (F, K) - "not a big deal", "overreacting", "too sensitive"
4. **Love Bombing** (F) - "can't live without", "soulmate", "meant to be"
5. **Stonewalling** (J) - "not talking about", "done discussing", "conversation over"
6. **Parental Alienation** (J, K) - "doesn't care about you", "bad parent"
7. **Coercive Control** (F, K) - "have to", "must", "or else"
8. **Financial Abuse** (C, F, K) - "your money", "can't afford", "pay for"
9. **DARVO Pattern** (F, K) - "I'm the victim", "you're abusive"
10. **Character Assassination** (F, J) - "crazy", "unstable", "unfit"

Each pattern is mapped to relevant Michigan Child Custody Act (MCL 722.23) Best Interest Factors.

## Entity Types Extracted

- **Persons** - Using spaCy NER (PERSON entities)
- **Locations** - Geographic locations (GPE, LOC, FAC)
- **Organizations** - Companies, agencies (ORG)
- **Phone Numbers** - Regex extraction with E.164 normalization
- **Email Addresses** - Regex extraction with lowercase normalization

## Performance

- Processes ~100 messages/second (depends on content length)
- Low memory footprint (~200MB with spaCy model loaded)
- No API costs - runs entirely locally

## Integration with Browser App

The browser app automatically calls this service during XML processing if it's running on localhost:8000. Enable/disable in settings.
