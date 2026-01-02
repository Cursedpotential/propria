"""
Memory-Efficient JSON File Splitter

Splits large JSON files into smaller chunks while ensuring clean bracket separation.
Supports both JSON arrays and objects. Chunks are saved to a subfolder.

Usage:
    python json_splitter.py <input_file> [--chunk-size MB] [--output-dir DIR]
"""

import json
import os
import sys
import argparse
from pathlib import Path
from typing import TextIO, Optional

# Fix Windows console encoding for emojis
if sys.platform == 'win32':
    import codecs
    sys.stdout = codecs.getwriter('utf-8')(sys.stdout.buffer, 'ignore')
    sys.stderr = codecs.getwriter('utf-8')(sys.stderr.buffer, 'ignore')


class JSONSplitter:
    """Memory-efficient JSON file splitter with streaming support."""

    def __init__(self, input_file: str, chunk_size_mb: float = 50, output_dir: Optional[str] = None):
        self.input_file = Path(input_file)
        self.chunk_size_bytes = int(chunk_size_mb * 1024 * 1024)

        # Create output directory
        if output_dir:
            self.output_dir = Path(output_dir)
        else:
            self.output_dir = self.input_file.parent / f"{self.input_file.stem}_chunks"

        self.output_dir.mkdir(exist_ok=True)

        self.bracket_depth = 0
        self.current_chunk = []
        self.current_size = 0
        self.chunk_number = 1
        self.is_array = None
        self.first_line = True

    def _detect_json_type(self, file_handle: TextIO) -> str:
        """Detect if JSON is an array or object."""
        file_handle.seek(0)
        for line in file_handle:
            stripped = line.strip()
            if stripped:
                if stripped.startswith('['):
                    return 'array'
                elif stripped.startswith('{'):
                    return 'object'
        file_handle.seek(0)
        return 'array'  # default

    def _get_bracket_depth(self, char: str) -> int:
        """Track bracket/brace depth."""
        if char in '{[':
            return 1
        elif char in '}]':
            return -1
        return 0

    def _is_clean_split_point(self, line: str) -> bool:
        """
        Determine if this is a clean point to split.
        Clean points are between complete JSON elements (depth = 1 for arrays, 0 for top-level).
        """
        stripped = line.strip()

        # For arrays, we want to split between elements (after '],' or '},')
        if self.is_array == 'array':
            # Check if we're at depth 1 (inside main array but between elements)
            if self.bracket_depth == 1 and (stripped.endswith(',') or stripped == ']'):
                return True

        # For objects, split between top-level properties
        elif self.is_array == 'object':
            if self.bracket_depth == 1 and stripped.endswith(','):
                return True

        return False

    def _save_chunk(self):
        """Save current chunk to file."""
        if not self.current_chunk:
            return

        output_file = self.output_dir / f"chunk_{self.chunk_number:04d}.json"

        with open(output_file, 'w', encoding='utf-8') as f:
            # Write opening bracket/brace
            if self.is_array == 'array':
                f.write('[\n')
            else:
                f.write('{\n')

            # Write chunk content
            for i, line in enumerate(self.current_chunk):
                # Remove trailing comma from last item
                if i == len(self.current_chunk) - 1:
                    line = line.rstrip().rstrip(',')
                f.write(line)
                if not line.endswith('\n'):
                    f.write('\n')

            # Write closing bracket/brace
            if self.is_array == 'array':
                f.write(']\n')
            else:
                f.write('}\n')

        print(f"✓ Saved chunk {self.chunk_number}: {output_file.name} ({self._format_size(self.current_size)})")

        self.chunk_number += 1
        self.current_chunk = []
        self.current_size = 0

    def _format_size(self, size_bytes: int) -> str:
        """Format byte size to human-readable format."""
        for unit in ['B', 'KB', 'MB', 'GB']:
            if size_bytes < 1024.0:
                return f"{size_bytes:.2f} {unit}"
            size_bytes /= 1024.0
        return f"{size_bytes:.2f} TB"

    def split(self):
        """Main splitting logic with streaming."""
        if not self.input_file.exists():
            raise FileNotFoundError(f"Input file not found: {self.input_file}")

        print(f"\n🔍 Analyzing JSON file: {self.input_file.name}")
        print(f"📁 Output directory: {self.output_dir}")
        print(f"📦 Target chunk size: {self._format_size(self.chunk_size_bytes)}")
        print(f"\n{'='*60}")

        with open(self.input_file, 'r', encoding='utf-8') as f:
            # Detect JSON type
            self.is_array = self._detect_json_type(f)
            print(f"📋 JSON type detected: {self.is_array.upper()}")
            print(f"{'='*60}\n")

            line_number = 0
            for line in f:
                line_number += 1

                # Skip opening/closing brackets of main structure
                stripped = line.strip()
                if line_number == 1 and stripped in ['[', '{']:
                    continue
                if stripped in [']', '}'] and line_number > 1:
                    # This might be the final closing bracket
                    continue

                # Update bracket depth
                for char in line:
                    self.bracket_depth += self._get_bracket_depth(char)

                # Add line to current chunk
                line_size = len(line.encode('utf-8'))
                self.current_chunk.append(line)
                self.current_size += line_size

                # Check if we should split
                if self.current_size >= self.chunk_size_bytes and self._is_clean_split_point(line):
                    self._save_chunk()

        # Save any remaining content
        if self.current_chunk:
            self._save_chunk()

        print(f"\n{'='*60}")
        print(f"✅ Splitting complete!")
        print(f"📊 Total chunks created: {self.chunk_number - 1}")
        print(f"📂 Location: {self.output_dir}")
        print(f"{'='*60}\n")


def main():
    parser = argparse.ArgumentParser(
        description='Split large JSON files into smaller chunks with clean bracket separation.',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python json_splitter.py large_file.json
  python json_splitter.py data.json --chunk-size 100
  python json_splitter.py data.json --chunk-size 25 --output-dir ./my_chunks
        """
    )

    parser.add_argument('input_file', help='Path to the input JSON file')
    parser.add_argument(
        '--chunk-size',
        type=float,
        default=50,
        help='Target size for each chunk in MB (default: 50)'
    )
    parser.add_argument(
        '--output-dir',
        type=str,
        help='Output directory for chunks (default: <filename>_chunks)'
    )

    args = parser.parse_args()

    try:
        splitter = JSONSplitter(
            input_file=args.input_file,
            chunk_size_mb=args.chunk_size,
            output_dir=args.output_dir
        )
        splitter.split()
    except Exception as e:
        print(f"\n❌ Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
