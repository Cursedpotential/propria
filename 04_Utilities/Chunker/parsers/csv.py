from .base import BaseParser
from typing import List

class CSVParser(BaseParser):
    def get_name(self) -> str:
        return "CSV (Comma-Separated)"
    
    def can_parse(self, file_extension: str) -> bool:
        return file_extension == '.csv'
    
    def parse(self, content: str) -> List[str]:
        lines = [l for l in content.split('\n') if l.strip()]
        if not lines:
            return []
        
        header = lines[0]
        return [f"{header}\n{line}\n" for line in lines[1:]]
