# Smart Chunker

Modular, cross-platform document chunker built with Flet. Intelligently splits large documents into AI-friendly chunks with metadata, smart labels, and custom schemas.

## Features

### Core Functionality
- **Smart Chunking** - Respects code blocks, lists, headers, and date boundaries
- **Date Detection** - Automatically detects and uses dates as chunk boundaries
- **Auto-Labeling** - Chunks labeled with dates, headers, or content preview
- **Configurable Overlap** - Context continuity between chunks
- **LLM Instructions** - Embed system prompts in chunk headers

### File Format Support
- **Markdown/Text** - Smart header and code block detection
- **HTML** - Schema-based parsing (Facebook, Snapchat, custom)
- **CSV/TSV** - Row-based chunking with header preservation

### Export Formats
- **TXT** - Single file with AI-friendly headers and metadata
- **JSON** - Structured format with chunk metadata
- **ZIP** - Individual files per chunk (e.g., `messages_1_of_10_Jan_05_2024.txt`)

### UI Features
- **Schema Editor** - Create/edit HTML parsing schemas on-the-fly
- **Parser Selector** - Choose specific parsers (Facebook Messages, CSV, etc.)
- **Dark/Light Theme** - Persistent theme preference
- **File Picker** - Native file selection
- **Chunk Navigation** - Browse chunks before export
- **Custom Filenames** - Set base name for output files

### Cross-Platform
- Windows (.exe)
- macOS (.app)
- Linux (binary)
- Web (PWA)
- iOS/Android (mobile apps)

## Installation

```bash
cd chunker
pip install -r requirements.txt
```

## Usage

### Run Application
```bash
python main.py
```

### Build Standalone Executables

```bash
# Windows (creates .exe in build/windows/)
flet build windows

# macOS (creates .app in build/macos/)
flet build macos

# Linux (creates binary in build/linux/)
flet build linux

# Web (creates PWA in build/web/)
flet build web

# Android (creates .apk)
flet build apk

# iOS (creates .ipa)
flet build ipa
```

## Architecture

```
chunker/
├── main.py              # Flet UI application
├── parsers/             # File format parsers
│   ├── base.py         # BaseParser interface
│   ├── markdown.py     # Markdown/text parser
│   ├── html.py         # Schema-based HTML parser
│   ├── csv.py          # CSV parser
│   └── tsv.py          # TSV parser
├── chunkers/            # Chunking strategies
│   ├── base.py         # BaseChunker interface
│   └── smart.py        # Smart chunker with metadata
├── exporters/           # Export formats
│   ├── base.py         # BaseExporter interface
│   ├── txt.py          # Text exporter
│   ├── json.py         # JSON exporter
│   └── zip.py          # ZIP archive exporter
├── schemas/             # HTML parsing schemas
│   ├── facebook_messages.json
│   ├── snapchat_messages.json
│   └── generic_html.json
├── ui/                  # UI components
│   └── schema_editor.py # Schema editor dialog
└── utils/               # Utilities
    ├── stats.py        # Token/word/char counting
    └── metadata.py     # Date/header extraction
```

## Extending Functionality

### Add New Parser

1. Create `parsers/pdf.py`:
```python
from .base import BaseParser
from typing import List

class PDFParser(BaseParser):
    def get_name(self) -> str:
        return "PDF Documents"
    
    def can_parse(self, file_extension: str) -> bool:
        return file_extension == '.pdf'
    
    def parse(self, content: str) -> List[str]:
        # Your PDF parsing logic
        sections = []
        # ... extract text, split by pages/sections
        return sections
```

2. Register in `parsers/__init__.py`:
```python
from .pdf import PDFParser

def get_all_parsers():
    parsers = [
        MarkdownParser(),
        PDFParser(),  # Add here
        CSVParser(),
        # ...
    ]
```

### Add New Chunker

Create `chunkers/semantic.py`:
```python
from .base import BaseChunker
from typing import List, Dict

class SemanticChunker(BaseChunker):
    def chunk(self, sections: List[str], chunk_size: int, overlap: int = 0) -> List[Dict[str, any]]:
        # Your semantic chunking logic (e.g., sentence embeddings)
        chunks = []
        # ... semantic similarity grouping
        return chunks
```

### Add New Exporter

Create `exporters/docx.py`:
```python
from .base import BaseExporter
from typing import List, Dict
from docx import Document

class DocxExporter(BaseExporter):
    def export(self, chunks: List[Dict[str, any]], output_path: str, base_name: str = "chunks", llm_instruction: str = "") -> None:
        doc = Document()
        for chunk in chunks:
            doc.add_heading(f"Chunk {chunk['num']}", level=1)
            doc.add_paragraph(chunk['content'])
        doc.save(output_path)
    
    def get_extension(self) -> str:
        return '.docx'
```

Register in `exporters/__init__.py`:
```python
from .docx import DocxExporter

EXPORTERS = {
    'TXT': TxtExporter(),
    'JSON': JsonExporter(),
    'ZIP': ZipExporter(),
    'DOCX': DocxExporter(),  # Add here
}
```

### Add New HTML Schema

#### Via UI:
1. Click **+** button in main window
2. Enter schema name (e.g., "Instagram DMs")
3. Add CSS selectors:
   - `container`: `.message-container`
   - `sender`: `.sender-name`
   - `content`: `.message-text`
   - `timestamp`: `.timestamp`
4. Set output format: `**{sender}** ({timestamp})\n{content}\n`
5. Save

#### Via JSON:
Create `schemas/instagram_dms.json`:
```json
{
  "name": "Instagram DMs",
  "description": "Instagram direct message export format",
  "file_pattern": "messages_*.html",
  "selectors": {
    "container": "div.message-container",
    "sender": ".sender-name",
    "content": ".message-text",
    "timestamp": ".timestamp"
  },
  "output_format": "**{sender}** ({timestamp})\n{content}\n"
}
```

## Output Examples

### TXT Export
```
================================================================================
DOCUMENT CHUNKS - AI PROCESSING FORMAT
Generated: 2024-01-15T10:30:00
Total Chunks: 5
Base Name: facebook_messages
Output File: facebook_messages.txt
================================================================================

LLM INSTRUCTIONS:
You are analyzing a large document that has been broken into sequential chunks...
================================================================================

================================================================================
CHUNK 1 of 5
Label: Jan_05_2024
Position: 1/5 (20.0%)
Size: 7850 characters

INSTRUCTIONS: You are analyzing a large document...
================================================================================

**John Doe** (Jan 5, 2024 10:30 AM)
Hey, how are you?
...
```

### ZIP Export
```
facebook_messages.zip
├── facebook_messages_1_of_5_Jan_05_2024.txt
├── facebook_messages_2_of_5_Feb_12_2024.txt
├── facebook_messages_3_of_5_Mar_20_2024.txt
├── facebook_messages_4_of_5_Apr_15_2024.txt
└── facebook_messages_5_of_5_May_30_2024.txt
```

### JSON Export
```json
{
  "base_name": "facebook_messages",
  "llm_instruction": "You are analyzing...",
  "total": 5,
  "chunks": [
    {
      "num": 1,
      "label": "Jan_05_2024",
      "size": 7850,
      "content": "**John Doe** (Jan 5, 2024 10:30 AM)\nHey, how are you?\n..."
    }
  ]
}
```

## Use Cases

- **LLM Context Windows** - Split large documents to fit token limits
- **Message Archive Analysis** - Parse Facebook/Snapchat exports
- **Legal Discovery** - Chunk depositions by date/speaker
- **Research Papers** - Split by sections with overlap
- **Code Documentation** - Preserve code blocks and structure
- **CSV Processing** - Chunk large datasets with header preservation

## License

MIT

## Contributing

Pull requests welcome! Please follow the modular architecture:
- New parsers in `parsers/`
- New chunkers in `chunkers/`
- New exporters in `exporters/`
- New schemas in `schemas/`
