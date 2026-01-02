"""
Convert JSON conversation chunks to clean Markdown format.

Usage:
    python json_to_markdown.py <chunks_directory> [--output-prefix NAME]
"""

import json
import sys
import argparse
from pathlib import Path
from datetime import datetime

# Fix Windows console encoding
if sys.platform == 'win32':
    import codecs
    sys.stdout = codecs.getwriter('utf-8')(sys.stdout.buffer, 'ignore')
    sys.stderr = codecs.getwriter('utf-8')(sys.stderr.buffer, 'ignore')


class JSONToMarkdown:
    """Convert JSON conversation chunks to Markdown."""

    def __init__(self, chunks_dir: str, output_prefix: str = "conversations"):
        self.chunks_dir = Path(chunks_dir)
        self.output_prefix = output_prefix
        self.output_dir = self.chunks_dir.parent / f"{output_prefix}_markdown"
        self.output_dir.mkdir(exist_ok=True)

    def _format_timestamp(self, timestamp_str: str) -> str:
        """Format ISO timestamp to readable format."""
        try:
            dt = datetime.fromisoformat(timestamp_str.replace('Z', '+00:00'))
            return dt.strftime('%Y-%m-%d %I:%M %p')
        except:
            return timestamp_str

    def _clean_text(self, text: str) -> str:
        """Clean up text for markdown."""
        if not text:
            return ""
        # Replace excessive newlines
        text = '\n'.join(line.rstrip() for line in text.split('\n'))
        # Remove excessive blank lines
        while '\n\n\n' in text:
            text = text.replace('\n\n\n', '\n\n')
        return text.strip()

    def _convert_entry(self, entry: dict) -> str:
        """Convert a single conversation entry to markdown."""
        role = entry.get('role', 'unknown')
        content = entry.get('content', '')

        if role == 'user':
            return f"**👤 User:**\n\n{self._clean_text(content)}\n"
        elif role == 'assistant':
            return f"**🤖 Assistant:**\n\n{self._clean_text(content)}\n"
        else:
            return f"**{role.title()}:**\n\n{self._clean_text(content)}\n"

    def _convert_conversation(self, conv: dict, conv_num: int) -> str:
        """Convert a single conversation to markdown."""
        title = conv.get('context_title', 'Untitled Conversation')
        created = self._format_timestamp(conv.get('created_at', ''))
        updated = self._format_timestamp(conv.get('updated_at', ''))
        entries = conv.get('entries', [])

        # Build markdown
        md = f"## Conversation #{conv_num}: {title}\n\n"
        md += f"📅 Created: {created}\n"
        if updated and updated != created:
            md += f"📅 Updated: {updated}\n"
        md += "\n---\n\n"

        # Add entries
        for entry in entries:
            md += self._convert_entry(entry)
            md += "\n---\n\n"

        return md

    def convert_chunk(self, chunk_file: Path, chunk_num: int):
        """Convert a single chunk file to markdown."""
        print(f"📝 Converting {chunk_file.name}...", end=' ')

        with open(chunk_file, 'r', encoding='utf-8') as f:
            data = json.load(f)

        conversations = data.get('conversations', [])

        if not conversations:
            print("⚠️  No conversations found")
            return

        # Create markdown output
        output_file = self.output_dir / f"{self.output_prefix}_{chunk_num:04d}.md"

        with open(output_file, 'w', encoding='utf-8') as f:
            f.write(f"# {self.output_prefix.replace('_', ' ').title()} - Part {chunk_num}\n\n")
            f.write(f"📊 Contains {len(conversations)} conversations\n\n")
            f.write("=" * 80 + "\n\n")

            for i, conv in enumerate(conversations, 1):
                conv_md = self._convert_conversation(conv, i)
                f.write(conv_md)
                f.write("\n\n")

        file_size = output_file.stat().st_size / 1024
        print(f"✓ ({file_size:.1f} KB)")

    def convert_all(self):
        """Convert all chunk files."""
        chunks = sorted(self.chunks_dir.glob('chunk_*.json'))

        if not chunks:
            print(f"❌ No chunk files found in {self.chunks_dir}")
            return

        print(f"\n🔄 Converting {len(chunks)} JSON chunks to Markdown")
        print(f"📁 Input: {self.chunks_dir}")
        print(f"📁 Output: {self.output_dir}")
        print(f"\n{'='*60}")

        for i, chunk_file in enumerate(chunks, 1):
            try:
                self.convert_chunk(chunk_file, i)
            except Exception as e:
                print(f"❌ Error: {e}")

        print(f"{'='*60}")
        print(f"✅ Conversion complete!")
        print(f"📂 Markdown files saved to: {self.output_dir}")
        print(f"{'='*60}\n")


def main():
    parser = argparse.ArgumentParser(
        description='Convert JSON conversation chunks to Markdown',
        epilog="""
Examples:
  python json_to_markdown.py ./conversation_chunks
  python json_to_markdown.py ./chunks --output-prefix "perplexity_chats"
        """
    )

    parser.add_argument('chunks_dir', help='Directory containing JSON chunk files')
    parser.add_argument(
        '--output-prefix',
        type=str,
        default='conversations',
        help='Prefix for output files (default: conversations)'
    )

    args = parser.parse_args()

    try:
        converter = JSONToMarkdown(
            chunks_dir=args.chunks_dir,
            output_prefix=args.output_prefix
        )
        converter.convert_all()
    except Exception as e:
        print(f"\n❌ Error: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()
