from .base import BaseExporter
from typing import List, Dict
from datetime import datetime

class TxtExporter(BaseExporter):
    def export(self, chunks: List[Dict[str, any]], output_path: str, base_name: str = "chunks", llm_instruction: str = "") -> None:
        with open(output_path, 'w', encoding='utf-8') as f:
            total = len(chunks)
            timestamp = datetime.now().isoformat()
            
            # Minimal header
            if llm_instruction:
                f.write(f"INSTRUCTIONS: {llm_instruction}\n\n")
            
            for chunk in chunks:
                # Minimal chunk header
                label = chunk.get('label', '')
                if label:
                    f.write(f"\n[{chunk['num']}/{total} - {label}]\n\n")
                else:
                    f.write(f"\n[{chunk['num']}/{total}]\n\n")
                
                # Content
                f.write(chunk['content'])
                
                # End marker
                if chunk['num'] < total:
                    f.write("\n\n[END - MORE CHUNKS FOLLOW]\n")
                else:
                    f.write("\n\n[END OF DOCUMENT]\n")
    
    def get_extension(self) -> str:
        return '.txt'
