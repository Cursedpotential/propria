"""
Robust conversation extractor - handles broken JSON with fuzzy pattern matching.
Doesn't trust brackets, uses text patterns instead.

Usage:
    python robust_conversation_extractor.py <chunks_directory>
"""

import re
import sys
import argparse
from pathlib import Path

# Fix Windows console encoding
if sys.platform == 'win32':
    import codecs
    sys.stdout = codecs.getwriter('utf-8')(sys.stdout.buffer, 'ignore')
    sys.stderr = codecs.getwriter('utf-8')(sys.stderr.buffer, 'ignore')


class RobustExtractor:
    """Extract conversations from broken JSON using pattern matching."""

    def __init__(self, chunks_dir: str, output_prefix: str = "perplexity_chats"):
        self.chunks_dir = Path(chunks_dir)
        self.output_prefix = output_prefix
        self.output_dir = self.chunks_dir.parent / f"{output_prefix}_markdown"
        self.output_dir.mkdir(exist_ok=True)

    def _clean_text(self, text: str) -> str:
        """Clean extracted text."""
        # Remove JSON escaping
        text = text.replace('\\n', '\n')
        text = text.replace('\\"', '"')
        text = text.replace('\\/', '/')
        text = text.replace('\\\\', '\\')

        # Remove excessive whitespace
        lines = [line.rstrip() for line in text.split('\n')]
        text = '\n'.join(lines)

        # Remove excessive blank lines
        while '\n\n\n' in text:
            text = text.replace('\n\n\n', '\n\n')

        return text.strip()

    def _extract_field(self, text: str, field_name: str) -> str:
        """Extract a field value using pattern matching."""
        # Try to find: "field_name": "value"
        patterns = [
            rf'"{field_name}"\s*:\s*"([^"]*(?:\\.[^"]*)*)"',  # With quotes
            rf'"{field_name}"\s*:\s*([^,\}}\]]+)',  # Without quotes
        ]

        for pattern in patterns:
            match = re.search(pattern, text, re.DOTALL)
            if match:
                return self._clean_text(match.group(1))

        return ""

    def _find_conversations(self, content: str) -> list:
        """Find conversation-like patterns in the text."""
        conversations = []

        # Split on patterns that look like conversation boundaries
        # Look for: "context_title" or "context_uuid" as markers
        splits = re.split(r'(?="context_uuid"\s*:)', content)

        for chunk in splits:
            if not chunk.strip() or len(chunk) < 100:
                continue

            # Extract fields using fuzzy matching
            title = self._extract_field(chunk, 'context_title') or "Untitled"
            created = self._extract_field(chunk, 'created_at')
            updated = self._extract_field(chunk, 'updated_at')

            # Try to find entries/messages
            entries = self._extract_entries(chunk)

            if entries:
                conversations.append({
                    'title': title,
                    'created': created,
                    'updated': updated,
                    'entries': entries
                })

        return conversations

    def _extract_entries(self, text: str) -> list:
        """Extract conversation entries using pattern matching."""
        entries = []

        # Look for role and content patterns
        # Pattern: "role": "user" ... "content": "message"
        entry_patterns = [
            r'"role"\s*:\s*"(user|assistant)".*?"content"\s*:\s*"([^"]*(?:\\.[^"]*)*)"',
        ]

        for pattern in entry_patterns:
            matches = re.finditer(pattern, text, re.DOTALL)
            for match in matches:
                role = match.group(1)
                content = self._clean_text(match.group(2))
                if content:
                    entries.append({'role': role, 'content': content})

        return entries

    def _format_conversation(self, conv: dict, num: int) -> str:
        """Format conversation as markdown."""
        md = f"## Conversation #{num}: {conv['title']}\n\n"

        if conv['created']:
            md += f"📅 {conv['created']}\n\n"

        md += "---\n\n"

        for entry in conv['entries']:
            role = entry['role']
            content = entry['content']

            if role == 'user':
                md += f"**👤 User:**\n\n{content}\n\n"
            else:
                md += f"**🤖 Assistant:**\n\n{content}\n\n"

            md += "---\n\n"

        return md

    def process_chunk(self, chunk_file: Path, chunk_num: int):
        """Process a single chunk file."""
        print(f"📖 Reading {chunk_file.name}...", end=' ')

        # Read as raw text
        with open(chunk_file, 'r', encoding='utf-8', errors='ignore') as f:
            content = f.read()

        # Extract conversations
        conversations = self._find_conversations(content)

        if not conversations:
            print("⚠️  No conversations extracted")
            return

        # Write markdown
        output_file = self.output_dir / f"{self.output_prefix}_{chunk_num:04d}.md"

        with open(output_file, 'w', encoding='utf-8') as f:
            f.write(f"# {self.output_prefix.replace('_', ' ').title()} - Part {chunk_num}\n\n")
            f.write(f"📊 Extracted {len(conversations)} conversations\n\n")
            f.write("=" * 80 + "\n\n")

            for i, conv in enumerate(conversations, 1):
                f.write(self._format_conversation(conv, i))
                f.write("\n\n")

        file_size = output_file.stat().st_size / 1024
        print(f"✓ {len(conversations)} conversations ({file_size:.1f} KB)")

    def process_all(self):
        """Process all chunks."""
        # Look for any JSON-like files
        chunk_files = list(self.chunks_dir.glob('chunk_*.json'))

        if not chunk_files:
            chunk_files = list(self.chunks_dir.glob('*.json'))

        if not chunk_files:
            print(f"❌ No files found in {self.chunks_dir}")
            return

        chunk_files = sorted(chunk_files)

        print(f"\n🔧 Robust extraction mode (handles broken JSON)")
        print(f"📁 Input: {self.chunks_dir}")
        print(f"📁 Output: {self.output_dir}")
        print(f"📄 Files: {len(chunk_files)}")
        print(f"\n{'='*60}")

        total_conversations = 0

        for i, chunk_file in enumerate(chunk_files, 1):
            try:
                before = total_conversations
                self.process_chunk(chunk_file, i)
                # Count from file if it exists
                output_file = self.output_dir / f"{self.output_prefix}_{i:04d}.md"
                if output_file.exists():
                    with open(output_file, 'r', encoding='utf-8') as f:
                        content = f.read()
                        count = content.count('## Conversation #')
                        total_conversations += count
            except Exception as e:
                print(f"❌ Error: {e}")

        print(f"{'='*60}")
        print(f"✅ Extraction complete!")
        print(f"📊 Total conversations extracted: {total_conversations}")
        print(f"📂 Saved to: {self.output_dir}")
        print(f"{'='*60}\n")


def main():
    parser = argparse.ArgumentParser(
        description='Robust conversation extractor for broken JSON',
        epilog="""
Examples:
  python robust_conversation_extractor.py ./chunks
  python robust_conversation_extractor.py ./simple_chunks --output-prefix "perplexity"
        """
    )

    parser.add_argument('chunks_dir', help='Directory with chunk files')
    parser.add_argument(
        '--output-prefix',
        type=str,
        default='perplexity_chats',
        help='Prefix for output files (default: perplexity_chats)'
    )

    args = parser.parse_args()

    try:
        extractor = RobustExtractor(
            chunks_dir=args.chunks_dir,
            output_prefix=args.output_prefix
        )
        extractor.process_all()
    except Exception as e:
        print(f"\n❌ Error: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()
