from .base import BaseParser
from typing import List
import re

class MarkdownParser(BaseParser):
    def get_name(self) -> str:
        return "Markdown / Text"
    
    def can_parse(self, file_extension: str) -> bool:
        return file_extension in ['.md', '.markdown', '.txt']
    
    def parse(self, content: str) -> List[str]:
        sections = []
        current = ''
        in_code_block = False
        
        date_pattern = re.compile(
            r'\b(\d{1,2}[\/\-]\d{1,2}[\/\-]\d{2,4}|\d{4}[\/\-]\d{1,2}[\/\-]\d{1,2}|'
            r'(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{1,2},?\s+\d{4}|'
            r'\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{4})\b',
            re.IGNORECASE
        )
        
        for line in content.split('\n'):
            if line.strip().startswith('```'):
                in_code_block = not in_code_block
            
            if not in_code_block and (line.startswith('#') or date_pattern.search(line)):
                if current.strip():
                    sections.append(current)
                    current = ''
            
            current += line + '\n'
        
        if current.strip():
            sections.append(current)
        
        return [s for s in sections if s.strip()]
