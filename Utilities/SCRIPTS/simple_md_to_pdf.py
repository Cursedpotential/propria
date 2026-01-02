"""
Simple Markdown to PDF converter using pandoc.
Lightweight backup - no Docker, no services, just pandoc.

Usage:
    python simple_md_to_pdf.py <markdown_directory>
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


class SimpleMDToPDF:
    """Simple markdown to PDF using pandoc."""

    def __init__(self, markdown_dir: str):
        self.markdown_dir = Path(markdown_dir)
        self.pdf_dir = self.markdown_dir.parent / f"{self.markdown_dir.name}_pdf"
        self.pdf_dir.mkdir(exist_ok=True)

    def _format_size(self, size_bytes: int) -> str:
        """Format byte size."""
        for unit in ['B', 'KB', 'MB', 'GB']:
            if size_bytes < 1024.0:
                return f"{size_bytes:.1f} {unit}"
            size_bytes /= 1024.0
        return f"{size_bytes:.1f} TB"

    def check_pandoc(self):
        """Check if pandoc is available."""
        try:
            result = subprocess.run(
                ['pandoc', '--version'],
                capture_output=True,
                text=True
            )
            return result.returncode == 0
        except FileNotFoundError:
            return False

    def convert_file(self, md_file: Path):
        """Convert a single markdown file to PDF using pandoc."""
        print(f"Converting {md_file.name}...", end=' ')

        try:
            pdf_file = self.pdf_dir / f"{md_file.stem}.pdf"

            # Use pandoc with good defaults
            cmd = [
                'pandoc',
                str(md_file),
                '-o', str(pdf_file),
                '--pdf-engine=wkhtmltopdf',  # Try wkhtmltopdf first
                '--metadata', f'title={md_file.stem}',
                '-V', 'geometry:margin=1in',
            ]

            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=120
            )

            if result.returncode != 0:
                # If wkhtmltopdf not found, try other engines
                for engine in ['pdflatex', 'xelatex', 'lualatex', 'context', 'pdfroff']:
                    cmd[3] = f'--pdf-engine={engine}'
                    result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
                    if result.returncode == 0:
                        break

            if result.returncode != 0:
                # Last resort: convert to HTML first, then print
                print(f"SKIP (no PDF engine available)")
                return False

            if pdf_file.exists():
                size = pdf_file.stat().st_size
                print(f"OK ({self._format_size(size)})")
                return True
            else:
                print("FAILED")
                return False

        except subprocess.TimeoutExpired:
            print("TIMEOUT")
            return False
        except Exception as e:
            print(f"ERROR: {e}")
            return False

    def convert_all(self):
        """Convert all markdown files."""
        if not self.check_pandoc():
            print("\nERROR: pandoc not found!")
            print("Install pandoc from: https://pandoc.org/installing.html")
            return

        md_files = sorted(self.markdown_dir.glob('*.md'))

        if not md_files:
            print(f"No markdown files found in {self.markdown_dir}")
            return

        print(f"\nSimple Markdown to PDF Converter")
        print(f"Using: pandoc")
        print(f"Input: {self.markdown_dir}")
        print(f"Output: {self.pdf_dir}")
        print(f"Files: {len(md_files)}")
        print("\n" + "="*60)

        success = 0
        failed = 0

        for md_file in md_files:
            if self.convert_file(md_file):
                success += 1
            else:
                failed += 1

        print("="*60)
        print(f"Complete! Success: {success} | Failed: {failed}")
        if success > 0:
            print(f"PDFs saved to: {self.pdf_dir}")
        print("="*60)


def main():
    parser = argparse.ArgumentParser(
        description='Convert markdown files to PDF using pandoc',
        epilog="""
Examples:
  python simple_md_to_pdf.py ./my_markdown_folder
  python simple_md_to_pdf.py "perplexity_chats_clean_markdown"
        """
    )

    parser.add_argument('markdown_dir', help='Directory containing markdown files')

    args = parser.parse_args()

    try:
        converter = SimpleMDToPDF(markdown_dir=args.markdown_dir)
        converter.convert_all()
    except Exception as e:
        print(f"\nError: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()
