from .base import BaseParser
from typing import List
from bs4 import BeautifulSoup
from schemas import load_schema, list_schemas

class HTMLParser(BaseParser):
    def __init__(self, schema_name: str = None):
        self.schema_name = schema_name
        self.schema = load_schema(schema_name) if schema_name else None
    
    def get_name(self) -> str:
        return self.schema['name'] if self.schema else 'HTML Parser'
    
    def can_parse(self, file_extension: str) -> bool:
        return file_extension in ['.html', '.htm']
    
    def parse(self, content: str) -> List[str]:
        soup = BeautifulSoup(content, 'html.parser')
        sections = []
        
        # Try all schemas if none specified
        schemas_to_try = [self.schema] if self.schema else [load_schema(s) for s in list_schemas()]
        
        for schema in schemas_to_try:
            if not schema:
                continue
            
            selectors = schema.get('selectors', {})
            container_sel = selectors.get('container')
            
            if not container_sel:
                continue
            
            containers = soup.select(container_sel)
            if not containers:
                continue
            
            # Found matching schema
            output_fmt = schema.get('output_format', '{content}\n')
            
            for container in containers:
                data = {'content': container.get_text().strip()}
                
                # Extract fields from selectors
                for field, selector in selectors.items():
                    if field == 'container':
                        continue
                    elem = container.select_one(selector)
                    data[field] = elem.get_text().strip() if elem else ''
                
                # Format output
                try:
                    formatted = output_fmt.format(**data)
                    sections.append(formatted)
                except KeyError:
                    sections.append(data['content'] + '\n')
            
            if sections:
                return sections
        
        # Fallback: generic text extraction
        return [soup.get_text()]
