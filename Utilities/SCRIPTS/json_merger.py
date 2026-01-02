"""
JSON Chunk Merger

Merges JSON chunks created by json_splitter.py back into a single file.
Memory-efficient streaming merge.

Usage:
    python json_merger.py <chunks_directory> [--output FILE]
"""

import json
import os
import sys
import argparse
from pathlib import Path
from typing import List


class JSONMerger:
    """Memory-efficient JSON chunk merger."""

    def __init__(self, chunks_dir: str, output_file: str = None):
        self.chunks_dir = Path(chunks_dir)

        if output_file:
            self.output_file = Path(output_file)
        else:
            # Default: create merged file in parent of chunks directory
            parent = self.chunks_dir.parent
            base_name = self.chunks_dir.name.replace('_chunks', '')
            self.output_file = parent / f"{base_name}_merged.json"

        self.json_type = None

    def _get_chunk_files(self) -> List[Path]:
        """Get sorted list of chunk files."""
        chunks = sorted(self.chunks_dir.glob('chunk_*.json'))
        if not chunks:
            raise ValueError(f"No chunk files found in {self.chunks_dir}")
        return chunks

    def _detect_json_type(self, first_chunk: Path) -> str:
        """Detect if chunks contain arrays or objects."""
        with open(first_chunk, 'r', encoding='utf-8') as f:
            first_line = f.readline().strip()
            if first_line.startswith('['):
                return 'array'
            elif first_line.startswith('{'):
                return 'object'
        return 'array'

    def _process_chunk_lines(self, chunk_file: Path, is_first: bool, is_last: bool):
        """Generator that yields lines from a chunk, excluding wrapper brackets."""
        with open(chunk_file, 'r', encoding='utf-8') as f:
            lines = f.readlines()

            # Skip first line (opening bracket) and last line (closing bracket)
            content_lines = lines[1:-1]

            for i, line in enumerate(content_lines):
                # Remove trailing comma from last line of non-last chunks
                if not is_last and i == len(content_lines) - 1:
                    stripped = line.rstrip()
                    if not stripped.endswith(','):
                        line = stripped + ',\n'

                yield line

    def merge(self):
        """Merge all chunks into a single JSON file."""
        if not self.chunks_dir.exists():
            raise FileNotFoundError(f"Chunks directory not found: {self.chunks_dir}")

        chunks = self._get_chunk_files()
        print(f"\n🔍 Found {len(chunks)} chunk files")
        print(f"📁 Input directory: {self.chunks_dir}")
        print(f"📝 Output file: {self.output_file}")

        # Detect JSON type from first chunk
        self.json_type = self._detect_json_type(chunks[0])
        print(f"📋 JSON type: {self.json_type.upper()}")
        print(f"\n{'='*60}")

        total_size = 0

        with open(self.output_file, 'w', encoding='utf-8') as out_f:
            # Write opening bracket
            if self.json_type == 'array':
                out_f.write('[\n')
            else:
                out_f.write('{\n')

            # Process each chunk
            for idx, chunk_file in enumerate(chunks):
                is_first = (idx == 0)
                is_last = (idx == len(chunks) - 1)

                print(f"⚙️  Processing {chunk_file.name}...", end=' ')

                chunk_size = chunk_file.stat().st_size
                total_size += chunk_size

                # Stream lines from chunk
                for line in self._process_chunk_lines(chunk_file, is_first, is_last):
                    out_f.write(line)

                print(f"✓ ({self._format_size(chunk_size)})")

            # Write closing bracket
            if self.json_type == 'array':
                out_f.write(']\n')
            else:
                out_f.write('}\n')

        output_size = self.output_file.stat().st_size

        print(f"{'='*60}")
        print(f"✅ Merge complete!")
        print(f"📊 Chunks processed: {len(chunks)}")
        print(f"📦 Output size: {self._format_size(output_size)}")
        print(f"💾 Saved to: {self.output_file}")
        print(f"{'='*60}\n")

    def _format_size(self, size_bytes: int) -> str:
        """Format byte size to human-readable format."""
        for unit in ['B', 'KB', 'MB', 'GB']:
            if size_bytes < 1024.0:
                return f"{size_bytes:.2f} {unit}"
            size_bytes /= 1024.0
        return f"{size_bytes:.2f} TB"


def main():
    parser = argparse.ArgumentParser(
        description='Merge JSON chunks back into a single file.',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python json_merger.py ./data_chunks
  python json_merger.py ./data_chunks --output merged_data.json
        """
    )

    parser.add_argument('chunks_dir', help='Directory containing chunk files')
    parser.add_argument(
        '--output',
        type=str,
        help='Output file path (default: <dirname>_merged.json in parent directory)'
    )

    args = parser.parse_args()

    try:
        merger = JSONMerger(
            chunks_dir=args.chunks_dir,
            output_file=args.output
        )
        merger.merge()
    except Exception as e:
        print(f"\n❌ Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
