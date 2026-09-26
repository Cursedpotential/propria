from abc import ABC, abstractmethod
from typing import List

class BaseParser(ABC):
    def get_name(self) -> str:
        """Return descriptive name for this parser"""
        return self.__class__.__name__
    
    @abstractmethod
    def can_parse(self, file_extension: str) -> bool:
        """Return True if this parser can handle the file extension"""
        pass
    
    @abstractmethod
    def parse(self, content: str) -> List[str]:
        """Parse content into sections"""
        pass
