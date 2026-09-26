from abc import ABC, abstractmethod
from typing import List, Dict

class BaseExporter(ABC):
    @abstractmethod
    def export(self, chunks: List[Dict[str, any]], output_path: str, base_name: str = "chunks", llm_instruction: str = "") -> None:
        """Export chunks to file"""
        pass
    
    @abstractmethod
    def get_extension(self) -> str:
        """Return file extension for this exporter"""
        pass
