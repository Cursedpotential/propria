"""
Conversation JSON Splitter

Specialized splitter for conversation export files with nested structure.
Splits by number of conversations rather than file size.

Usage:
    python conversation_splitter.py <input_file> [--conversations-per-chunk N]
"""

import json
import sys
import argparse
from pathlib import Path

# Fix Windows console encoding
if sys.platform == 'win32':
    import codecs
    sys.stdout = codecs.getwriter('utf-8')(sys.stdout.buffer, 'ignore')
    sys.stderr = codecs.getwriter('utf-8')(sys.stderr.buffer, 'ignore')


class ConversationSplitter:
    """Split conversation export files intelligently."""

    def __init__(self, input_file: str, convs_per_chunk: int = 50, output_dir: str = None):
        self.input_file = Path(input_file)
        self.convs_per_chunk = convs_per_chunk

        if output_dir:
            self.output_dir = Path(output_dir)
        else:
            self.output_dir = self.input_file.parent / f"{self.input_file.stem}_chunks"

        self.output_dir.mkdir(exist_ok=True)

    def _format_size(self, size_bytes: int) -> str:
        """Format byte size to human-readable."""
        for unit in ['B', 'KB', 'MB', 'GB']:
            if size_bytes < 1024.0:
                return f"{size_bytes:.2f} {unit}"
            size_bytes /= 1024.0
        return f"{size_bytes:.2f} TB"

    def split(self):
        """Split the conversation file."""
        print(f"\n🔍 Loading: {self.input_file.name}")

        # Load the entire JSON (conversation files are usually manageable in memory)
        with open(self.input_file, 'r', encoding='utf-8') as f:
            data = json.load(f)

        # Extract conversations array
        if isinstance(data, dict) and 'conversations' in data:
            conversations = data['conversations']
            wrapper_key = 'conversations'
        elif isinstance(data, list):
            conversations = data
            wrapper_key = None
        else:
            raise ValueError("Unable to find conversations array in JSON structure")

        total_convs = len(conversations)
        total_chunks = (total_convs + self.convs_per_chunk - 1) // self.convs_per_chunk

        print(f"📊 Total conversations: {total_convs}")
        print(f"📦 Conversations per chunk: {self.convs_per_chunk}")
        print(f"🗂️  Chunks to create: {total_chunks}")
        print(f"📁 Output directory: {self.output_dir}")
        print(f"\n{'='*60}")

        # Split into chunks
        for chunk_num in range(total_chunks):
            start_idx = chunk_num * self.convs_per_chunk
            end_idx = min(start_idx + self.convs_per_chunk, total_convs)

            chunk_convs = conversations[start_idx:end_idx]

            # Create chunk data
            if wrapper_key:
                chunk_data = {wrapper_key: chunk_convs}
            else:
                chunk_data = chunk_convs

            # Save chunk
            output_file = self.output_dir / f"chunk_{chunk_num + 1:04d}.json"

            with open(output_file, 'w', encoding='utf-8') as f:
                json.dump(chunk_data, f, indent=2, ensure_ascii=False)

            file_size = output_file.stat().st_size

            print(f"✓ Chunk {chunk_num + 1:04d}: "
                  f"conversations {start_idx + 1}-{end_idx} "
                  f"({len(chunk_convs)} convs, {self._format_size(file_size)})")

        print(f"{'='*60}")
        print(f"✅ Splitting complete!")
        print(f"📂 Location: {self.output_dir}")
        print(f"{'='*60}\n")


def main():
    parser = argparse.ArgumentParser(
        description='Split conversation export files by number of conversations.',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python conversation_splitter.py conversations.json
  python conversation_splitter.py conversations.json --conversations-per-chunk 100
  python conversation_splitter.py conversations.json --conversations-per-chunk 25 --output-dir ./chunks
        """
    )

    parser.add_argument('input_file', help='Path to conversation JSON file')
    parser.add_argument(
        '--conversations-per-chunk',
        type=int,
        default=50,
        help='Number of conversations per chunk (default: 50)'
    )
    parser.add_argument(
        '--output-dir',
        type=str,
        help='Output directory for chunks'
    )

    args = parser.parse_args()

    try:
        splitter = ConversationSplitter(
            input_file=args.input_file,
            convs_per_chunk=args.conversations_per_chunk,
            output_dir=args.output_dir
        )
        splitter.split()
    except Exception as e:
        print(f"\n❌ Error: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()
