"""
Process Real Claude Code Session with NLTK Preprocessing
Compares token usage between sending full session vs. NLTK-preprocessed data
"""

import json
import nltk
from nltk.sentiment import SentimentIntensityAnalyzer
import re
from pathlib import Path

# Ensure NLTK data is available
try:
    nltk.data.find('vader_lexicon')
except LookupError:
    nltk.download('vader_lexicon', quiet=True)
    nltk.download('punkt', quiet=True)

def count_tokens(text):
    """Simple token estimation (4 chars ≈ 1 token)"""
    return len(text) // 4

def load_session(filepath):
    """Load JSONL session file and extract messages"""
    messages = []
    with open(filepath, 'r', encoding='utf-8') as f:
        for line in f:
            try:
                entry = json.loads(line.strip())
                messages.append(entry)
            except json.JSONDecodeError:
                continue
    return messages

def extract_user_messages(messages):
    """Extract only user messages from session"""
    user_messages = []
    for msg in messages:
        # Look for user message type
        if msg.get('type') == 'user':
            # Content is nested under message.content
            message_obj = msg.get('message', {})
            content = message_obj.get('content', '')
            if content:
                user_messages.append({
                    'text': content,
                    'timestamp': msg.get('timestamp', ''),
                    'messageId': msg.get('uuid', '')
                })
    return user_messages

def nltk_analysis(user_messages):
    """Run NLTK sentiment and pattern analysis"""
    sia = SentimentIntensityAnalyzer()

    results = {
        'total_messages': len(user_messages),
        'negative_messages': [],
        'correction_patterns': [],
        'technical_mentions': []
    }

    correction_keywords = [
        r'\bdon\'t\b', r'\bdon\'t\b', r'\bstop\b', r'\bavoid\b', r'\bnever\b',
        r'\bwhy did you\b', r'\byou just\b', r'\bagain\b',
        r'\bwrong\b', r'\bmistake\b', r'\bfix\b', r'\bfucking\b', r'\bjesus\b'
    ]

    for msg in user_messages:
        text = msg['text']

        # Handle if content is a list (sometimes it's structured content)
        if isinstance(text, list):
            text = ' '.join([str(item) for item in text if isinstance(item, (str, dict))])
        text = str(text)

        # Sentiment analysis
        sentiment = sia.polarity_scores(text)
        if sentiment['compound'] < -0.3:
            results['negative_messages'].append({
                'text': text[:100],
                'score': sentiment['compound']
            })

        # Check for correction signals
        for pattern in correction_keywords:
            if re.search(pattern, text, re.IGNORECASE):
                results['correction_patterns'].append({
                    'text': text[:80],
                    'pattern': pattern
                })
                break

        # Extract technical mentions
        if re.search(r'\b(Read|Write|Edit|Bash|Grep|Task|Skill)\b', text):
            results['technical_mentions'].append(text[:60])

    return results

def main():
    session_file = Path("C:/Users/matts/.claude/projects/C--Users-matts/6c19d9d4-1913-46d7-a67e-2332b8fe794e.jsonl")

    if not session_file.exists():
        print(f"Session file not found: {session_file}")
        return

    print("="*70)
    print("REAL SESSION ANALYSIS - NLTK vs Full LLM")
    print("="*70)

    # Load session
    print(f"\nLoading session from: {session_file.name}")
    messages = load_session(session_file)
    print(f"Total entries in session: {len(messages)}")

    # Extract user messages
    user_messages = extract_user_messages(messages)
    print(f"User messages found: {len(user_messages)}")

    if not user_messages:
        print("\nNo user messages found. Checking message types...")
        types = {}
        for msg in messages[:10]:
            msg_type = msg.get('type', 'unknown')
            types[msg_type] = types.get(msg_type, 0) + 1
        print("Sample message types:", types)
        return

    # Calculate full conversation size
    full_conversation = "\n\n".join([
        f"User: {msg['text']}"
        for msg in user_messages
    ])

    print(f"\nFull conversation size: {len(full_conversation)} characters")

    # Approach 1: Send full conversation
    system_prompt_size = 500  # Approximate system prompt for conversation analyzer
    full_tokens = count_tokens(full_conversation) + system_prompt_size

    print("\n" + "="*70)
    print("APPROACH 1: Send Full User Messages to Agent")
    print("="*70)
    print(f"Tokens: {full_tokens}")
    print(f"Cost (@$3/M): ${full_tokens * 3 / 1_000_000:.6f}")

    # Approach 2: NLTK preprocessing
    print("\n" + "="*70)
    print("APPROACH 2: NLTK Preprocessing (Local - 0 tokens)")
    print("="*70)

    analysis = nltk_analysis(user_messages)

    print(f"[OK] Analyzed {analysis['total_messages']} user messages")
    print(f"[OK] Found {len(analysis['negative_messages'])} negative sentiment messages")
    print(f"[OK] Found {len(analysis['correction_patterns'])} correction signals")
    print(f"[OK] Found {len(analysis['technical_mentions'])} technical tool mentions")

    # Create condensed prompt
    condensed = f"""Conversation Analysis Results:

Total user messages: {analysis['total_messages']}
Negative sentiment detected: {len(analysis['negative_messages'])}

Correction signals ({len(analysis['correction_patterns'])}):
{chr(10).join([f"- {c['text']}..." for c in analysis['correction_patterns'][:5]])}

Technical patterns mentioned ({len(analysis['technical_mentions'])}):
{chr(10).join([f"- {t}..." for t in analysis['technical_mentions'][:5]])}

Generate hookify rules for these patterns."""

    condensed_tokens = count_tokens(condensed)

    print(f"\nCondensed prompt tokens: {condensed_tokens}")
    print(f"Cost (@$3/M): ${condensed_tokens * 3 / 1_000_000:.6f}")

    # Comparison
    print("\n" + "="*70)
    print("RESULTS")
    print("="*70)
    print(f"Full approach:  {full_tokens:6d} tokens")
    print(f"NLTK approach:  {condensed_tokens:6d} tokens")
    print(f"Savings:        {full_tokens - condensed_tokens:6d} tokens ({(1 - condensed_tokens/full_tokens)*100:.1f}%)")
    print(f"Cost savings:   ${(full_tokens - condensed_tokens) * 3 / 1_000_000:.6f}")

    # Show some examples
    if analysis['correction_patterns']:
        print("\n" + "="*70)
        print("SAMPLE CORRECTION SIGNALS FOUND")
        print("="*70)
        for i, corr in enumerate(analysis['correction_patterns'][:3], 1):
            print(f"\n{i}. {corr['text']}...")
            print(f"   Pattern: {corr['pattern']}")

    if analysis['negative_messages']:
        print("\n" + "="*70)
        print("SAMPLE NEGATIVE SENTIMENT MESSAGES")
        print("="*70)
        for i, neg in enumerate(analysis['negative_messages'][:3], 1):
            print(f"\n{i}. {neg['text']}...")
            print(f"   Sentiment score: {neg['score']:.2f}")

if __name__ == "__main__":
    main()
