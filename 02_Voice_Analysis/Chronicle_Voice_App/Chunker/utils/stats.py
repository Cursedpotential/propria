import re

def count_tokens(text: str) -> int:
    """Approximate token count (GPT-style: ~4 chars per token)"""
    return len(text) // 4

def count_words(text: str) -> int:
    return len(re.findall(r'\b\w+\b', text))

def count_chars(text: str) -> int:
    return len(text)

def get_stats(text: str) -> dict:
    return {
        'chars': count_chars(text),
        'words': count_words(text),
        'tokens': count_tokens(text),
    }
