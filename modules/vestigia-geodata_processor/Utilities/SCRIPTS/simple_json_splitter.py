"""
Simple JSON Splitter - Split at bracket+comma boundaries

Super simple streaming splitter that splits whenever it finds:
  },  or  ],
and the chunk has reached the target size.

Usage:
    python simple_json_splitter.py <input_file> [--chunk-size-mb N]
"""

import sys
import argparse
from pathlib import Path

# Fix Windows console encoding
if sys.platform == 'win32':
    import codecs
    sys.stdout = codecs.getwriter('utf-8')(sys.stdout.buffer, 'ignore')
    sys.stderr = codecs.getwriter('utf-8')(sys.stderr.buffer, 'ignore')


class SimpleJSONSplitter:
    """Simple line-by-line JSON splitter."""

    def __init__(self, input_file: str, chunk_size_mb: float = 2, output_dir: str = None):
        self.input_file = Path(input_file)
        self.chunk_size_bytes = int(chunk_size_mb * 1024 * 1024)

        if output_dir:
            self.output_dir = Path(output_dir)
        else:
            self.output_dir = self.input_file.parent / f"{self.input_file.stem}_simple_chunks"

        self.output_dir.mkdir(exist_ok=True)
        self.chunk_num = 1

    def _format_size(self, size_bytes: int) -> str:
        """Format byte size."""
        for unit in ['B', 'KB', 'MB', 'GB']:
            if size_bytes < 1024.0:
                return f"{size_bytes:.2f} {unit}"
            size_bytes /= 1024.0
        return f"{size_bytes:.2f} TB"

    def _is_split_point(self, line: str) -> bool:
        """Check if line ends with }, or ],"""
        stripped = line.rstrip()
        return stripped.endswith('},') or stripped.endswith('],')

    def _save_chunk(self, lines: list):
        """Save a chunk."""
        if not lines:
            return

        output_file = self.output_dir / f"chunk_{self.chunk_num:04d}.json"

        # Calculate size
        total_size = sum(len(line.encode('utf-8')) for line in lines)

        with open(output_file, 'w', encoding='utf-8') as f:
            # Write all lines
            f.writelines(lines)

        print(f"✓ Chunk {self.chunk_num:04d}: {len(lines):,} lines ({self._format_size(total_size)})")

        self.chunk_num += 1

    def split(self):
        """Split the file."""
        print(f"\n🔍 Processing: {self.input_file.name}")
        print(f"📦 Target chunk size: {self._format_size(self.chunk_size_bytes)}")
        print(f"📁 Output directory: {self.output_dir}")
        print(f"\n{'='*60}")

        current_chunk = []
        current_size = 0

        with open(self.input_file, 'r', encoding='utf-8') as f:
            for line in f:
                line_size = len(line.encode('utf-8'))
                current_chunk.append(line)
                current_size += line_size

                # Check if we should split
                if current_size >= self.chunk_size_bytes and self._is_split_point(line):
                    self._save_chunk(current_chunk)
                    current_chunk = []
                    current_size = 0

        # Save remaining
        if current_chunk:
            self._save_chunk(current_chunk)

        print(f"{'='*60}")
        print(f"✅ Complete! Created {self.chunk_num - 1} chunks")
        print(f"📂 Location: {self.output_dir}")
        print(f"{'='*60}\n")


def main():
    parser = argparse.ArgumentParser(
        description='Simple JSON splitter - splits at bracket+comma boundaries',
        epilog="""
Examples:
  python simple_json_splitter.py data.json
  python simple_json_splitter.py data.json --chunk-size-mb 5
        """
    )

    parser.add_argument('input_file', help='JSON file to split')
    parser.add_argument(
        '--chunk-size-mb',
        type=float,
        default=2,
        help='Target chunk size in MB (default: 2)'
    )
    parser.add_argument(
        '--output-dir',
        type=str,
        help='Output directory'
    )

    args = parser.parse_args()

    try:
        splitter = SimpleJSONSplitter(
            input_file=args.input_file,
            chunk_size_mb=args.chunk_size_mb,
            output_dir=args.output_dir
        )
        splitter.split()
    except Exception as e:
        print(f"\n❌ Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
