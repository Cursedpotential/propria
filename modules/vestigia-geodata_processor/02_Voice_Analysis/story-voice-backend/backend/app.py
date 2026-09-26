import gradio as gr
import httpx
import json
import os
from datetime import datetime
from typing import Optional
import hashlib

# ============ CONFIG ============
GEMINI_3_PRO_MODEL = "gemini-3-pro"  # Update if different
SUPABASE_URL = os.getenv("SUPABASE_URL", "")
SUPABASE_KEY = os.getenv("SUPABASE_KEY", "")
QDRANT_URL = os.getenv("QDRANT_URL", "")
QDRANT_API_KEY = os.getenv("QDRANT_API_KEY", "")
R2_ENDPOINT = os.getenv("R2_ENDPOINT", "")
R2_ACCESS_KEY = os.getenv("R2_ACCESS_KEY", "")
R2_SECRET_KEY = os.getenv("R2_SECRET_KEY", "")

EXTRACTION_PROMPT = """Analyze this conversation transcript and extract structured information.

TRANSCRIPT:
{transcript}

Extract the following in JSON format:

{{
  "entities": {{
    "people": [{{ "name": "...", "relationship": "...", "mentions": [...turn indices...] }}],
    "places": [{{ "name": "...", "context": "...", "mentions": [...] }}],
    "organizations": [{{ "name": "...", "type": "...", "mentions": [...] }}]
  }},
  "timeline": [
    {{ "event": "...", "date_mentioned": "...", "date_estimated": "YYYY-MM-DD or null", "confidence": 0-100, "turn": N }}
  ],
  "patterns": [
    {{ "pattern": "...", "evidence": ["quote1", "quote2"], "frequency": "recurring/isolated", "significance": "..." }}
  ],
  "themes": ["theme1", "theme2"],
  "emotional_markers": [
    {{ "turn": N, "emotion": "...", "trigger": "..." }}
  ],
  "follow_up_questions": ["question1", "question2"],
  "summary": "2-3 sentence factual summary of what was discussed"
}}

Be thorough. Extract ALL entities mentioned. Note ALL timeline references even if vague ("a few years ago").
For patterns, look for: repeated behaviors, "always/never" statements, recurring themes.
Do NOT add interpretations or judgments - extract only what was explicitly stated."""

CLEAN_TRANSCRIPT_PROMPT = """Clean this conversation transcript for documentation.

TRANSCRIPT:
{transcript}

Rules:
1. Light grammar/spelling fixes only
2. Replace profanity with [expletive] 
3. Keep all factual content intact
4. Maintain speaker's voice and meaning
5. Do NOT summarize or paraphrase
6. Do NOT add any content
7. Preserve timeline references exactly as stated
8. Keep emotional expressions (they're evidence)

Output the cleaned transcript in markdown format with speaker labels."""


async def call_gemini_3_pro(prompt: str, token: str) -> dict:
    """Call Gemini 3 Pro for analysis."""
    async with httpx.AsyncClient(timeout=120) as client:
        resp = await client.post(
            f"https://generativelanguage.googleapis.com/v1beta/models/{GEMINI_3_PRO_MODEL}:generateContent",
            headers={
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/json"
            },
            json={
                "contents": [{"role": "user", "parts": [{"text": prompt}]}],
                "generationConfig": {"maxOutputTokens": 4096, "temperature": 0.2}
            }
        )
        resp.raise_for_status()
        data = resp.json()
        text = data.get("candidates", [{}])[0].get("content", {}).get("parts", [{}])[0].get("text", "")
        return text


async def save_to_supabase(session_id: str, data: dict):
    """Save extracted data to Supabase."""
    if not SUPABASE_URL or not SUPABASE_KEY:
        print("Supabase not configured")
        return
    
    async with httpx.AsyncClient() as client:
        # Save session
        await client.post(
            f"{SUPABASE_URL}/rest/v1/story_sessions",
            headers={
                "apikey": SUPABASE_KEY,
                "Authorization": f"Bearer {SUPABASE_KEY}",
                "Content-Type": "application/json",
                "Prefer": "return=minimal"
            },
            json={
                "id": session_id,
                "created_at": datetime.utcnow().isoformat(),
                "raw_transcript": data.get("raw_transcript"),
                "clean_transcript": data.get("clean_transcript"),
                "extraction": data.get("extraction"),
                "summary": data.get("extraction", {}).get("summary")
            }
        )
        
        # Save entities
        entities = data.get("extraction", {}).get("entities", {})
        for entity_type, items in entities.items():
            for item in items:
                await client.post(
                    f"{SUPABASE_URL}/rest/v1/story_entities",
                    headers={
                        "apikey": SUPABASE_KEY,
                        "Authorization": f"Bearer {SUPABASE_KEY}",
                        "Content-Type": "application/json",
                        "Prefer": "return=minimal"
                    },
                    json={
                        "session_id": session_id,
                        "entity_type": entity_type,
                        "name": item.get("name"),
                        "metadata": item
                    }
                )
        
        # Save timeline events
        for event in data.get("extraction", {}).get("timeline", []):
            await client.post(
                f"{SUPABASE_URL}/rest/v1/story_timeline",
                headers={
                    "apikey": SUPABASE_KEY,
                    "Authorization": f"Bearer {SUPABASE_KEY}",
                    "Content-Type": "application/json",
                    "Prefer": "return=minimal"
                },
                json={
                    "session_id": session_id,
                    "event_description": event.get("event"),
                    "date_mentioned": event.get("date_mentioned"),
                    "date_estimated": event.get("date_estimated"),
                    "confidence": event.get("confidence"),
                    "turn_index": event.get("turn")
                }
            )


async def save_to_qdrant(session_id: str, transcript: list, extraction: dict, token: str):
    """Generate embeddings and save to Qdrant."""
    if not QDRANT_URL:
        print("Qdrant not configured")
        return
    
    # Generate embedding for the session summary
    summary = extraction.get("summary", "")
    themes = " ".join(extraction.get("themes", []))
    embed_text = f"{summary} {themes}"
    
    # Use Gemini embedding API (or switch to another embedding model)
    async with httpx.AsyncClient() as client:
        # Get embedding
        embed_resp = await client.post(
            "https://generativelanguage.googleapis.com/v1beta/models/text-embedding-004:embedContent",
            headers={
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/json"
            },
            json={
                "model": "models/text-embedding-004",
                "content": {"parts": [{"text": embed_text}]}
            }
        )
        
        if embed_resp.status_code == 200:
            embedding = embed_resp.json().get("embedding", {}).get("values", [])
            
            # Save to Qdrant
            headers = {"Content-Type": "application/json"}
            if QDRANT_API_KEY:
                headers["api-key"] = QDRANT_API_KEY
            
            await client.put(
                f"{QDRANT_URL}/collections/story_sessions/points",
                headers=headers,
                json={
                    "points": [{
                        "id": hashlib.md5(session_id.encode()).hexdigest()[:16],
                        "vector": embedding,
                        "payload": {
                            "session_id": session_id,
                            "summary": summary,
                            "themes": extraction.get("themes", []),
                            "entity_names": [e.get("name") for e in extraction.get("entities", {}).get("people", [])],
                            "created_at": datetime.utcnow().isoformat()
                        }
                    }]
                }
            )


async def process_session(session_id: str, transcript: list, token: str) -> dict:
    """Main processing pipeline."""
    # Format transcript for prompts
    formatted = "\n".join([
        f"[Turn {i}] {'USER' if t['role']=='user' else 'AI'}: {t['text']}"
        for i, t in enumerate(transcript)
    ])
    
    # 1. Extract entities/timeline with Gemini 3 Pro
    extraction_text = await call_gemini_3_pro(
        EXTRACTION_PROMPT.format(transcript=formatted),
        token
    )
    
    # Parse JSON from response
    try:
        # Find JSON in response
        start = extraction_text.find("{")
        end = extraction_text.rfind("}") + 1
        extraction = json.loads(extraction_text[start:end])
    except:
        extraction = {"error": "Failed to parse extraction", "raw": extraction_text}
    
    # 2. Clean transcript
    clean_text = await call_gemini_3_pro(
        CLEAN_TRANSCRIPT_PROMPT.format(transcript=formatted),
        token
    )
    
    result = {
        "session_id": session_id,
        "raw_transcript": transcript,
        "clean_transcript": clean_text,
        "extraction": extraction,
        "processed_at": datetime.utcnow().isoformat()
    }
    
    # 3. Save to databases
    await save_to_supabase(session_id, result)
    await save_to_qdrant(session_id, transcript, extraction, token)
    
    return result


# ============ GRADIO API ============
def process_endpoint(data: str) -> str:
    """Gradio endpoint for processing."""
    import asyncio
    
    try:
        payload = json.loads(data)
        session_id = payload.get("sessionId")
        transcript = payload.get("transcript", [])
        token = payload.get("token")
        
        if not session_id or not transcript or not token:
            return json.dumps({"error": "Missing required fields"})
        
        result = asyncio.run(process_session(session_id, transcript, token))
        return json.dumps(result)
    
    except Exception as e:
        return json.dumps({"error": str(e)})


# Simple Gradio interface
with gr.Blocks() as app:
    gr.Markdown("# Story Capture Backend")
    gr.Markdown("Processing endpoint for voice sessions.")
    
    with gr.Row():
        input_json = gr.Textbox(label="Input JSON", lines=10)
        output_json = gr.Textbox(label="Output", lines=10)
    
    process_btn = gr.Button("Process")
    process_btn.click(process_endpoint, inputs=input_json, outputs=output_json)
    
    # Also expose as API
    app.load(lambda: None, api_name="health")


# For direct API access
@app.post("/process")
async def api_process(request):
    data = await request.json()
    result = await process_session(
        data.get("sessionId"),
        data.get("transcript", []),
        data.get("token")
    )
    return result


if __name__ == "__main__":
    app.launch()
