"""
Universal Pandoc Converter - One tool to rule them all.
Handles MD, DOCX, HTML → PDF and vice versa.

Usage:
    python pandoc_converter.py <input_directory> [--format pdf]
"""

import sys
import subprocess
import argparse
from pathlib import Path

# Fix Windows console encoding
if sys.platform == 'win32':
    import codecs
    sys.stdout = codecs.getwriter('utf-8')(sys.stdout.buffer, 'ignore')
    sys.stderr = codecs.getwriter('utf-8')(sys.stderr.buffer, 'ignore')


class PandocConverter:
    """Universal converter using pandoc."""

    # Try PDF engines in order of preference
    PDF_ENGINES = [
        'wkhtmltopdf',  # Best for HTML/MD
        'weasyprint',   # Python-based
        'prince',       # Commercial but good
        'context',      # ConTeXt
        'pdfroff',      # Groff
        'pdflatex',     # LaTeX (most common fallback)
        'xelatex',      # XeLaTeX
        'lualatex',     # LuaLaTeX
    ]

    def __init__(self, input_dir: str, output_format: str = 'pdf'):
        self.input_dir = Path(input_dir)
        self.output_format = output_format
        self.output_dir = self.input_dir.parent / f"{self.input_dir.name}_{output_format}"
        self.output_dir.mkdir(exist_ok=True)
        self.working_engine = None

    def _format_size(self, size_bytes: int) -> str:
        """Format byte size."""
        for unit in ['B', 'KB', 'MB', 'GB']:
            if size_bytes < 1024.0:
                return f"{size_bytes:.1f} {unit}"
            size_bytes /= 1024.0
        return f"{size_bytes:.1f} TB"

    def _find_working_engine(self):
        """Find the first working PDF engine."""
        if self.working_engine:
            return self.working_engine

        for engine in self.PDF_ENGINES:
            try:
                # Test if engine is available
                result = subprocess.run(
                    ['pandoc', '--pdf-engine=' + engine, '--version'],
                    capture_output=True,
                    timeout=5
                )
                if result.returncode == 0 or 'not found' not in result.stderr.decode():
                    self.working_engine = engine
                    print(f"Using PDF engine: {engine}")
                    return engine
            except:
                continue

        return None

    def convert_file(self, input_file: Path):
        """Convert a single file using pandoc."""
        print(f"Converting {input_file.name}...", end=' ')

        try:
            # Determine output filename
            output_file = self.output_dir / f"{input_file.stem}.{self.output_format}"

            # Build pandoc command
            cmd = ['pandoc', str(input_file), '-o', str(output_file)]

            # Add PDF engine if converting to PDF
            if self.output_format == 'pdf':
                engine = self._find_working_engine()
                if engine:
                    cmd.extend(['--pdf-engine=' + engine])

                # Add nice PDF options
                cmd.extend([
                    '-V', 'geometry:margin=1in',
                    '-V', 'fontsize=11pt',
                    '--metadata', f'title={input_file.stem}',
                ])

            # Run pandoc
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=120
            )

            if result.returncode != 0:
                print(f"FAILED: {result.stderr[:100]}")
                return False

            if output_file.exists():
                size = output_file.stat().st_size
                print(f"OK ({self._format_size(size)})")
                return True
            else:
                print("FAILED: No output")
                return False

        except subprocess.TimeoutExpired:
            print("TIMEOUT")
            return False
        except Exception as e:
            print(f"ERROR: {e}")
            return False

    def convert_all(self):
        """Convert all files in directory."""
        # Find all convertible files
        patterns = ['*.md', '*.markdown', '*.html', '*.htm', '*.docx', '*.txt']
        files = []
        for pattern in patterns:
            files.extend(self.input_dir.glob(pattern))

        files = sorted(set(files))

        if not files:
            print(f"No convertible files found in {self.input_dir}")
            return

        print(f"\nUniversal Pandoc Converter")
        print(f"Input: {self.input_dir}")
        print(f"Output: {self.output_dir}")
        print(f"Format: {self.output_format.upper()}")
        print(f"Files: {len(files)}")
        print("\n" + "="*60)

        success = 0
        failed = 0

        for file in files:
            if self.convert_file(file):
                success += 1
            else:
                failed += 1

        print("="*60)
        print(f"Complete! Success: {success} | Failed: {failed}")
        if success > 0:
            print(f"Output saved to: {self.output_dir}")
        if failed > 0 and self.output_format == 'pdf':
            print(f"\nNote: Install a PDF engine for better results:")
            print(f"  choco install wkhtmltopdf")
        print("="*60)


def main():
    parser = argparse.ArgumentParser(
        description='Universal converter using pandoc',
        epilog="""
Examples:
  python pandoc_converter.py ./markdown_folder
  python pandoc_converter.py ./markdown_folder --format pdf
  python pandoc_converter.py ./docx_folder --format pdf
  python pandoc_converter.py ./html_folder --format docx

Supported formats: pdf, docx, html, markdown, and more
        """
    )

    parser.add_argument('input_dir', help='Directory containing files to convert')
    parser.add_argument(
        '--format',
        type=str,
        default='pdf',
        help='Output format (default: pdf)'
    )

    args = parser.parse_args()

    try:
        converter = PandocConverter(
            input_dir=args.input_dir,
            output_format=args.format
        )
        converter.convert_all()
    except Exception as e:
        print(f"\nError: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()
