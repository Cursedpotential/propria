from .base import BaseChunker
from typing import List, Dict
from utils.metadata import extract_label

class SmartChunker(BaseChunker):
    def chunk(self, sections: List[str], chunk_size: int, overlap: int = 0) -> List[Dict[str, any]]:
        chunks = []
        current = ''
        chunk_num = 1
        
        for section in sections:
            if len(current) + len(section) > chunk_size and current.strip():
                label = extract_label(current)
                chunks.append({'content': current, 'num': chunk_num, 'label': label})
                current = current[-overlap:] if overlap > 0 else ''
                chunk_num += 1
            current += section
        
        if current.strip():
            label = extract_label(current)
            chunks.append({'content': current, 'num': chunk_num, 'label': label})
        
        return chunks
