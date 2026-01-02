from .base import BaseExporter
from typing import List, Dict
import json

class JsonExporter(BaseExporter):
    def export(self, chunks: List[Dict[str, any]], output_path: str, base_name: str = "chunks", llm_instruction: str = "") -> None:
        output = {
            'base_name': base_name,
            'llm_instruction': llm_instruction,
            'total': len(chunks),
            'chunks': [{
                'num': c['num'],
                'label': c.get('label', ''),
                'size': len(c['content']),
                'content': c['content']
            } for c in chunks]
        }
        with open(output_path, 'w', encoding='utf-8') as f:
            json.dump(output, f, indent=2)
    
    def get_extension(self) -> str:
        return '.json'
