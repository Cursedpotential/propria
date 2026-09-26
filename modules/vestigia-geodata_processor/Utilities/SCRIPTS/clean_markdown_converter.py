"""
Clean Markdown Converter - No escape sequences, proper formatting.
Creates clean markdown suitable for any PDF converter.

Usage:
    python clean_markdown_converter.py <chunks_directory>
"""

import re
import sys
import argparse
from pathlib import Path
from datetime import datetime

# Fix Windows console encoding
if sys.platform == 'win32':
    import codecs
    sys.stdout = codecs.getwriter('utf-8')(sys.stdout.buffer, 'ignore')
    sys.stderr = codecs.getwriter('utf-8')(sys.stderr.buffer, 'ignore')


class CleanMarkdownConverter:
    """Convert conversations to clean markdown - no escape sequences."""

    def __init__(self, chunks_dir: str, output_prefix: str = "perplexity_chats"):
        self.chunks_dir = Path(chunks_dir)
        self.output_prefix = output_prefix
        self.output_dir = self.chunks_dir.parent / f"{output_prefix}_clean_markdown"
        self.output_dir.mkdir(exist_ok=True)

    def clean_text(self, text: str) -> str:
        """Clean text - remove escapes and normalize."""
        if not text:
            return ""

        # Unescape JSON sequences
        replacements = {
            '\\n': '\n',
            '\\"': '"',
            "\\'": "'",
            '\\\\': '\\',
            '\\t': '    ',
            '\\r': '',
        }

        for old, new in replacements.items():
            text = text.replace(old, new)

        # Remove any remaining backslash escapes
        text = re.sub(r'\\(.)', r'\1', text)

        # Normalize whitespace
        lines = []
        for line in text.split('\n'):
            line = line.rstrip()
            lines.append(line)

        # Join and clean up excessive blank lines
        text = '\n'.join(lines)

        # Replace 3+ newlines with just 2
        while '\n\n\n' in text:
            text = text.replace('\n\n\n', '\n\n')

        return text.strip()

    def extract_conversations(self, content: str) -> list:
        """Extract conversations using pattern matching."""
        conversations = []
        parts = re.split(r'"context_uuid"\s*:', content)

        for part in parts[1:]:
            try:
                # Extract title
                title_match = re.search(r'"context_title"\s*:\s*"([^"]*(?:\\.[^"]*)*)"', part)
                title = title_match.group(1) if title_match else "Untitled"
                title = self.clean_text(title)

                # Extract date
                created_match = re.search(r'"created_at"\s*:\s*"([^"]+)"', part)
                created = ""
                if created_match:
                    try:
                        dt = datetime.fromisoformat(created_match.group(1).replace('Z', '+00:00'))
                        created = dt.strftime('%B %d, %Y at %I:%M %p')
                    except:
                        created = created_match.group(1)[:10]

                # Extract Q&A entries
                entries = []
                qa_matches = re.finditer(
                    r'"query"\s*:\s*"([^"]*(?:\\.[^"]*)*)".*?"answer"\s*:\s*"([^"]*(?:\\.[^"]*)*)"',
                    part,
                    re.DOTALL
                )

                for match in qa_matches:
                    query = self.clean_text(match.group(1))
                    answer = self.clean_text(match.group(2))

                    if query:
                        entries.append({'role': 'user', 'text': query})
                    if answer:
                        entries.append({'role': 'assistant', 'text': answer})

                if entries:
                    conversations.append({
                        'title': title,
                        'date': created,
                        'entries': entries
                    })

            except Exception:
                continue

        return conversations

    def format_markdown(self, conversations: list, chunk_num: int) -> str:
        """Format conversations as clean markdown."""
        lines = []

        # Header
        lines.append(f"# {self.output_prefix.replace('_', ' ').title()} - Part {chunk_num}")
        lines.append("")
        lines.append(f"Contains {len(conversations)} conversations")
        lines.append("")
        lines.append("=" * 80)
        lines.append("")
        lines.append("")

        # Each conversation
        for i, conv in enumerate(conversations, 1):
            lines.append(f"## Conversation {i}: {conv['title']}")
            lines.append("")

            if conv['date']:
                lines.append(f"**Date:** {conv['date']}")
                lines.append("")

            lines.append("---")
            lines.append("")

            # Entries
            for entry in conv['entries']:
                if entry['role'] == 'user':
                    lines.append("### User")
                    lines.append("")
                    lines.append(entry['text'])
                    lines.append("")
                else:
                    lines.append("### Assistant")
                    lines.append("")
                    lines.append(entry['text'])
                    lines.append("")

                lines.append("---")
                lines.append("")

            lines.append("")
            lines.append("")

        return '\n'.join(lines)

    def process_chunk(self, chunk_file: Path, chunk_num: int):
        """Process a single chunk."""
        print(f"Processing {chunk_file.name}...", end=' ')

        try:
            with open(chunk_file, 'r', encoding='utf-8', errors='ignore') as f:
                content = f.read()

            conversations = self.extract_conversations(content)

            if not conversations:
                print("No conversations found")
                return 0

            # Write clean markdown
            markdown = self.format_markdown(conversations, chunk_num)
            output_file = self.output_dir / f"{self.output_prefix}_{chunk_num:04d}.md"

            with open(output_file, 'w', encoding='utf-8', newline='\n') as f:
                f.write(markdown)

            file_size = output_file.stat().st_size / 1024
            print(f"OK - {len(conversations)} conversations ({file_size:.1f} KB)")

            return len(conversations)

        except Exception as e:
            print(f"ERROR: {e}")
            return 0

    def process_all(self):
        """Process all chunks."""
        chunk_files = sorted(self.chunks_dir.glob('chunk_*.json'))

        if not chunk_files:
            print(f"No chunk files found in {self.chunks_dir}")
            return

        print(f"\nClean Markdown Converter (No escape sequences)")
        print(f"Input: {self.chunks_dir}")
        print(f"Output: {self.output_dir}")
        print(f"Files: {len(chunk_files)}")
        print("\n" + "="*60)

        total = 0
        for i, chunk_file in enumerate(chunk_files, 1):
            total += self.process_chunk(chunk_file, i)

        print("="*60)
        print(f"Complete! {total} conversations converted to clean markdown")
        print(f"Location: {self.output_dir}")
        print("="*60)
        print("\nReady for Stirling PDF conversion!")


def main():
    parser = argparse.ArgumentParser(
        description='Convert to clean markdown (no escape sequences)',
        epilog="""
Examples:
  python clean_markdown_converter.py ./chunks
  python clean_markdown_converter.py ./chunks --output-prefix "perplexity"
        """
    )

    parser.add_argument('chunks_dir', help='Directory with chunk files')
    parser.add_argument(
        '--output-prefix',
        type=str,
        default='perplexity_chats',
        help='Prefix for output files'
    )

    args = parser.parse_args()

    try:
        converter = CleanMarkdownConverter(
            chunks_dir=args.chunks_dir,
            output_prefix=args.output_prefix
        )
        converter.process_all()
    except Exception as e:
        print(f"\nError: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()
