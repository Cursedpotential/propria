"""
Batch JSON Splitter

Process multiple JSON files at once with the same settings.

Usage:
    python batch_json_splitter.py <directory> [--chunk-size MB] [--pattern PATTERN]
"""

import argparse
import subprocess
import sys
from pathlib import Path
from typing import List


class BatchJSONSplitter:
    """Batch process multiple JSON files."""

    def __init__(self, directory: str, chunk_size: float = 50, pattern: str = "*.json"):
        self.directory = Path(directory)
        self.chunk_size = chunk_size
        self.pattern = pattern
        self.splitter_script = Path(__file__).parent / "json_splitter.py"

    def find_json_files(self) -> List[Path]:
        """Find all JSON files matching the pattern."""
        if not self.directory.exists():
            raise FileNotFoundError(f"Directory not found: {self.directory}")

        files = list(self.directory.glob(self.pattern))

        # Filter out already processed chunks
        files = [f for f in files if not f.stem.startswith('chunk_')]

        return sorted(files)

    def split_file(self, json_file: Path) -> bool:
        """Split a single JSON file."""
        print(f"\n{'='*60}")
        print(f"📄 Processing: {json_file.name}")
        print(f"{'='*60}")

        try:
            cmd = [
                sys.executable,
                str(self.splitter_script),
                str(json_file),
                "--chunk-size", str(self.chunk_size)
            ]

            result = subprocess.run(cmd, capture_output=False, text=True)

            if result.returncode == 0:
                print(f"✅ Successfully split {json_file.name}")
                return True
            else:
                print(f"❌ Failed to split {json_file.name}")
                return False

        except Exception as e:
            print(f"❌ Error processing {json_file.name}: {e}")
            return False

    def process_all(self):
        """Process all JSON files in the directory."""
        if not self.splitter_script.exists():
            raise FileNotFoundError(
                f"json_splitter.py not found at {self.splitter_script}\n"
                "Make sure batch_json_splitter.py is in the same directory as json_splitter.py"
            )

        files = self.find_json_files()

        if not files:
            print(f"⚠️  No JSON files found matching pattern '{self.pattern}' in {self.directory}")
            return

        print(f"\n{'='*60}")
        print(f"🔍 Batch JSON Splitter")
        print(f"{'='*60}")
        print(f"📁 Directory: {self.directory}")
        print(f"🔎 Pattern: {self.pattern}")
        print(f"📦 Chunk size: {self.chunk_size} MB")
        print(f"📊 Files found: {len(files)}")
        print(f"{'='*60}")

        # Show files to be processed
        print("\n📝 Files to process:")
        for i, file in enumerate(files, 1):
            size = file.stat().st_size / (1024 * 1024)
            print(f"   {i}. {file.name} ({size:.2f} MB)")

        # Confirm
        response = input(f"\n❓ Process {len(files)} file(s)? [Y/n]: ").strip().lower()
        if response and response not in ['y', 'yes']:
            print("❌ Cancelled by user")
            return

        # Process files
        success_count = 0
        fail_count = 0

        for i, file in enumerate(files, 1):
            print(f"\n\n{'#'*60}")
            print(f"# File {i}/{len(files)}")
            print(f"{'#'*60}")

            if self.split_file(file):
                success_count += 1
            else:
                fail_count += 1

        # Summary
        print(f"\n\n{'='*60}")
        print(f"📊 BATCH PROCESSING COMPLETE")
        print(f"{'='*60}")
        print(f"✅ Successful: {success_count}")
        print(f"❌ Failed: {fail_count}")
        print(f"📁 Total: {len(files)}")
        print(f"{'='*60}\n")


def main():
    parser = argparse.ArgumentParser(
        description='Batch process multiple JSON files with the splitter.',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Split all JSON files in current directory
  python batch_json_splitter.py .

  # Split all JSON files in specific directory
  python batch_json_splitter.py C:\\data\\json_files

  # Split with custom chunk size
  python batch_json_splitter.py ./data --chunk-size 100

  # Split only files matching pattern
  python batch_json_splitter.py ./data --pattern "export_*.json"
        """
    )

    parser.add_argument(
        'directory',
        help='Directory containing JSON files to process'
    )
    parser.add_argument(
        '--chunk-size',
        type=float,
        default=50,
        help='Target size for each chunk in MB (default: 50)'
    )
    parser.add_argument(
        '--pattern',
        type=str,
        default='*.json',
        help='Glob pattern for files to process (default: *.json)'
    )

    args = parser.parse_args()

    try:
        processor = BatchJSONSplitter(
            directory=args.directory,
            chunk_size=args.chunk_size,
            pattern=args.pattern
        )
        processor.process_all()
    except Exception as e:
        print(f"\n❌ Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
