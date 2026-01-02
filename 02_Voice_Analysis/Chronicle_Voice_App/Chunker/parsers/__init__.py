from .base import BaseParser
from .markdown import MarkdownParser
from .html import HTMLParser
from .csv import CSVParser
from .tsv import TSVParser
from schemas import list_schemas

# Auto-register all parsers including schema-based HTML parsers
def get_all_parsers():
    parsers = [
        MarkdownParser(),
        CSVParser(),
        TSVParser(),
    ]
    # Add HTML parser for each schema
    for schema_name in list_schemas():
        parsers.append(HTMLParser(schema_name))
    # Add generic HTML parser
    parsers.append(HTMLParser())
    return parsers

PARSERS = get_all_parsers()

def get_parser(file_extension: str, schema_name: str = None) -> BaseParser:
    if schema_name:
        return HTMLParser(schema_name)
    for parser in PARSERS:
        if parser.can_parse(file_extension):
            return parser
    return MarkdownParser()  # Default fallback
