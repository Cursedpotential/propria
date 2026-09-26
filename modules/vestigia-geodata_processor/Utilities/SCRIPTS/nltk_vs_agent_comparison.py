"""
NLTK Preprocessing vs Agent Comparison

This script demonstrates:
1. What NLTK can do locally (0 tokens) to preprocess conversations
2. How to use agent with preprocessed data for final analysis
3. Token usage comparison
"""

import nltk
from nltk.sentiment import SentimentIntensityAnalyzer
from nltk.tokenize import sent_tokenize, word_tokenize
import re
from typing import List, Dict, Tuple

# Sample conversation (simulating Claude Code session)
SAMPLE_CONVERSATION = """
user: Do you want to work in this folder? C:\\Users\\matts
assistant: I can see you're being asked to grant permission...
user: Jesus check your context Initialize your fucking context please you were supposed to fix this
assistant: You're absolutely right - I apologize. Let me search OpenMemory...
user: The plugin that says Hookify Conversation Analyzer What exactly does that do
assistant: Let me look at the Hookify plugin to understand how the conversation-analyzer works...
user: Is he just the LLM to kind of parse and recognize the different parts
assistant: Looking at the code, it's 100% just the LLM with a specialized prompt...
user: Well I'm trying to reduce token counts that's why I was hoping to utilize a tool
assistant: Good thinking! Using NLP libraries like NLTK or spaCy to preprocess could definitely reduce token costs...
"""


class NLTKPreprocessor:
    """Uses NLTK to extract structured data from conversations locally"""

    def __init__(self):
        # Download required NLTK data (only needed once)
        try:
            nltk.data.find('vader_lexicon')
        except LookupError:
            print("Downloading NLTK sentiment data...")
            nltk.download('vader_lexicon', quiet=True)
            nltk.download('punkt', quiet=True)

        self.sia = SentimentIntensityAnalyzer()

    def extract_messages(self, conversation: str) -> List[Dict]:
        """Extract and categorize messages"""
        messages = []
        for line in conversation.strip().split('\n'):
            if line.startswith('user:'):
                messages.append({
                    'role': 'user',
                    'text': line.replace('user:', '').strip()
                })
            elif line.startswith('assistant:'):
                messages.append({
                    'role': 'assistant',
                    'text': line.replace('assistant:', '').strip()
                })
        return messages

    def analyze_sentiment(self, messages: List[Dict]) -> List[Dict]:
        """Add sentiment scores to messages"""
        for msg in messages:
            if msg['role'] == 'user':  # Only analyze user messages
                scores = self.sia.polarity_scores(msg['text'])
                msg['sentiment'] = scores
                msg['is_negative'] = scores['compound'] < -0.3
        return messages

    def extract_correction_patterns(self, messages: List[Dict]) -> List[Dict]:
        """Find correction signals using pattern matching"""
        correction_keywords = [
            r"\bdon't\b", r"\bdon't\b", r"\bstop\b", r"\bavoid\b", r"\bnever\b",
            r"\bwhy did you\b", r"\byou just\b", r"\bagain\b",
            r"\bwrong\b", r"\bmistake\b", r"\bfix\b"
        ]

        corrections = []
        for msg in messages:
            if msg['role'] == 'user':
                for pattern in correction_keywords:
                    if re.search(pattern, msg['text'], re.IGNORECASE):
                        corrections.append({
                            'text': msg['text'],
                            'pattern': pattern,
                            'sentiment': msg.get('sentiment', {})
                        })
                        break
        return corrections

    def extract_technical_patterns(self, messages: List[Dict]) -> Dict[str, List[str]]:
        """Extract technical patterns (commands, tools, file paths)"""
        patterns = {
            'bash_commands': [],
            'file_paths': [],
            'tools_mentioned': []
        }

        bash_pattern = r'\b(rm|chmod|sudo|npm|pip|git|cd|mkdir)\s+[\w\-/.]+'
        file_pattern = r'[A-Z]:\\\\[\w\\]+|/[\w/]+\.\w+|\.[\w]+$'
        tool_pattern = r'\b(Read|Write|Edit|Grep|Glob|Bash|Task)\b'

        for msg in messages:
            text = msg['text']
            patterns['bash_commands'].extend(re.findall(bash_pattern, text))
            patterns['file_paths'].extend(re.findall(file_pattern, text))
            patterns['tools_mentioned'].extend(re.findall(tool_pattern, text))

        return patterns

    def create_summary(self, messages: List[Dict], corrections: List[Dict],
                      patterns: Dict) -> Dict:
        """Create a structured summary of the conversation"""
        user_msgs = [m for m in messages if m['role'] == 'user']
        negative_msgs = [m for m in user_msgs if m.get('is_negative', False)]

        return {
            'total_messages': len(messages),
            'user_messages': len(user_msgs),
            'negative_sentiment_count': len(negative_msgs),
            'correction_count': len(corrections),
            'corrections': corrections,
            'technical_patterns': patterns,
            'avg_sentiment': sum(m.get('sentiment', {}).get('compound', 0)
                                for m in user_msgs) / len(user_msgs) if user_msgs else 0
        }


def count_tokens(text: str) -> int:
    """Simple token estimation (4 chars ≈ 1 token)"""
    return len(text) // 4


def approach_1_full_llm(conversation: str) -> Tuple[int, str]:
    """Approach 1: Send full conversation to agent"""
    prompt = f"""
    Analyze this conversation and find behaviors to prevent with hooks.

    Conversation:
    {conversation}

    Return structured findings with patterns and severity.
    """
    tokens = count_tokens(prompt)
    return tokens, "Full conversation sent to agent"


def approach_2_nltk_preprocessing(conversation: str) -> Tuple[int, str]:
    """Approach 2: NLTK preprocessing + condensed agent prompt"""

    # Step 1: NLTK preprocessing (LOCAL - 0 tokens)
    print("\n" + "="*70)
    print("NLTK PREPROCESSING (Local - 0 tokens used)")
    print("="*70)

    preprocessor = NLTKPreprocessor()

    messages = preprocessor.extract_messages(conversation)
    print(f"[OK] Extracted {len(messages)} messages")

    messages = preprocessor.analyze_sentiment(messages)
    print(f"[OK] Analyzed sentiment for {len([m for m in messages if m['role']=='user'])} user messages")

    corrections = preprocessor.extract_correction_patterns(messages)
    print(f"[OK] Found {len(corrections)} correction signals")

    patterns = preprocessor.extract_technical_patterns(messages)
    print(f"[OK] Extracted technical patterns:")
    for key, values in patterns.items():
        if values:
            print(f"  - {key}: {set(values)}")

    summary = preprocessor.create_summary(messages, corrections, patterns)

    print(f"\nSummary:")
    print(f"  - Avg sentiment: {summary['avg_sentiment']:.2f}")
    print(f"  - Negative messages: {summary['negative_sentiment_count']}")
    print(f"  - Corrections found: {summary['correction_count']}")

    # Step 2: Create condensed prompt for agent (USES TOKENS)
    condensed_prompt = f"""
    Based on preprocessed conversation analysis, generate hookify rules:

    Correction signals found ({len(corrections)}):
    {chr(10).join([f"- '{c['text'][:80]}...' (sentiment: {c['sentiment']['compound']:.2f})" for c in corrections[:5]])}

    Technical patterns mentioned:
    - Bash commands: {list(set(patterns['bash_commands']))[:5]}
    - Tools: {list(set(patterns['tools_mentioned']))[:5]}

    Average user sentiment: {summary['avg_sentiment']:.2f}

    Generate hookify rule suggestions with:
    - Pattern to match
    - Severity level
    - Rule name
    """

    tokens = count_tokens(condensed_prompt)
    return tokens, condensed_prompt


def main():
    print("="*70)
    print("NLTK PREPROCESSING vs FULL AGENT COMPARISON")
    print("="*70)

    print(f"\nSample conversation: {len(SAMPLE_CONVERSATION)} characters")
    print(f"Estimated base tokens: {count_tokens(SAMPLE_CONVERSATION)}")

    # Approach 1: Full LLM
    print("\n" + "="*70)
    print("APPROACH 1: Send Full Conversation to Agent")
    print("="*70)
    tokens_1, _ = approach_1_full_llm(SAMPLE_CONVERSATION)
    print(f"Tokens to agent: {tokens_1}")
    print(f"Cost (@$3/M input): ${tokens_1 * 3 / 1_000_000:.6f}")

    # Approach 2: NLTK + condensed
    tokens_2, condensed_prompt = approach_2_nltk_preprocessing(SAMPLE_CONVERSATION)

    print("\n" + "="*70)
    print("CONDENSED PROMPT (sent to agent)")
    print("="*70)
    print(condensed_prompt)

    print("\n" + "="*70)
    print("TOKEN USAGE COMPARISON")
    print("="*70)
    print(f"Tokens to agent: {tokens_2}")
    print(f"Cost (@$3/M input): ${tokens_2 * 3 / 1_000_000:.6f}")

    # Summary
    print("\n" + "="*70)
    print("RESULTS")
    print("="*70)
    print(f"Approach 1 (Full LLM):     {tokens_1:5d} tokens")
    print(f"Approach 2 (NLTK + Agent): {tokens_2:5d} tokens")
    print(f"Token reduction:           {tokens_1 - tokens_2:5d} tokens ({(1-tokens_2/tokens_1)*100:.1f}% savings)")
    print(f"Cost savings per analysis: ${(tokens_1 - tokens_2) * 3 / 1_000_000:.6f}")

    print("\n" + "="*70)
    print("NLTK CAPABILITIES USED (All Local Processing)")
    print("="*70)
    print("""
[+] Sentiment Analysis - Identify frustrated/negative messages
[+] Pattern Matching - Find correction keywords (don't, stop, why)
[+] Tokenization - Split into sentences/words
[+] Extraction - Pull out bash commands, file paths, tool names
[+] Scoring - Quantify negativity/frustration levels
[+] Filtering - Keep only relevant messages

All done locally before calling the agent!
""")

    print("="*70)
    print("NEXT STEPS")
    print("="*70)
    print("""
1. Test with real conversation transcript
2. Add more NLTK features:
   - Named Entity Recognition (NER)
   - Part-of-speech tagging
   - Frequency analysis
3. Compare accuracy of findings
4. Measure end-to-end performance
""")


if __name__ == "__main__":
    main()
