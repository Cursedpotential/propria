from abc import ABC, abstractmethod
from typing import List, Dict

class BaseChunker(ABC):
    @abstractmethod
    def chunk(self, sections: List[str], chunk_size: int, overlap: int = 0) -> List[Dict[str, any]]:
        """Chunk sections into chunks with metadata"""
        pass
