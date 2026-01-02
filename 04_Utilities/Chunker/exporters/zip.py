from .base import BaseExporter
from typing import List, Dict
import zipfile

class ZipExporter(BaseExporter):
    def export(self, chunks: List[Dict[str, any]], output_path: str, base_name: str = "chunks", llm_instruction: str = "") -> None:
        total = len(chunks)
        with zipfile.ZipFile(output_path, 'w') as zf:
            for chunk in chunks:
                label = chunk.get('label', '')
                if label:
                    filename = f"{base_name}_{chunk['num']}_of_{total}_{label}.txt"
                else:
                    filename = f"{base_name}_{chunk['num']}_of_{total}.txt"
                if llm_instruction and chunk['num'] == 1:
                    content = f"INSTRUCTIONS: {llm_instruction}\n\n"
                else:
                    content = ""
                
                if label:
                    content += f"[{chunk['num']}/{total} - {label}]\n\n"
                else:
                    content += f"[{chunk['num']}/{total}]\n\n"
                
                content += chunk['content']
                
                # End marker
                if chunk['num'] < total:
                    content += "\n\n[END - MORE CHUNKS FOLLOW]"
                else:
                    content += "\n\n[END OF DOCUMENT]"
                zf.writestr(filename, content)
    
    def get_extension(self) -> str:
        return '.zip'
