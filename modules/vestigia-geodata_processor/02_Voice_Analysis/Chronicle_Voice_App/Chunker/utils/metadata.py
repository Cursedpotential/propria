import re
from typing import Optional

DATE_PATTERN = re.compile(
    r'\b(\d{1,2}[\/\-]\d{1,2}[\/\-]\d{2,4}|\d{4}[\/\-]\d{1,2}[\/\-]\d{1,2}|'
    r'(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{1,2},?\s+\d{4}|'
    r'\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{4})\b',
    re.IGNORECASE
)

def extract_date(text: str) -> Optional[str]:
    """Extract first date from text"""
    match = DATE_PATTERN.search(text)
    return match.group(0) if match else None

def extract_header(text: str) -> Optional[str]:
    """Extract first markdown header"""
    lines = text.split('\n')
    for line in lines:
        if line.strip().startswith('#'):
            return line.strip('#').strip()[:50]
    return None

def extract_label(text: str) -> str:
    """Extract smart label from chunk content"""
    # Try date first
    date = extract_date(text[:500])
    if date:
        return date.replace('/', '-').replace(' ', '_')
    
    # Try header
    header = extract_header(text[:500])
    if header:
        return header.replace(' ', '_')[:30]
    
    # Fallback to first words
    words = text.strip().split()[:5]
    return '_'.join(words)[:30] if words else 'chunk'
