"""
Convert DOCX files to PDF.

Usage:
    python docx_to_pdf.py <docx_directory>
"""

import sys
import argparse
from pathlib import Path
from docx2pdf import convert

# Fix Windows console encoding
if sys.platform == 'win32':
    import codecs
    sys.stdout = codecs.getwriter('utf-8')(sys.stdout.buffer, 'ignore')
    sys.stderr = codecs.getwriter('utf-8')(sys.stderr.buffer, 'ignore')


class DOCXToPDF:
    """Convert DOCX files to PDF."""

    def __init__(self, docx_dir: str):
        self.docx_dir = Path(docx_dir)
        self.pdf_dir = self.docx_dir.parent / f"{self.docx_dir.name}_pdf"
        self.pdf_dir.mkdir(exist_ok=True)

    def _format_size(self, size_bytes: int) -> str:
        """Format byte size."""
        for unit in ['B', 'KB', 'MB', 'GB']:
            if size_bytes < 1024.0:
                return f"{size_bytes:.1f} {unit}"
            size_bytes /= 1024.0
        return f"{size_bytes:.1f} TB"

    def convert_file(self, docx_file: Path):
        """Convert a single DOCX to PDF."""
        print(f"📄 Converting {docx_file.name}...", end=' ')

        try:
            pdf_file = self.pdf_dir / f"{docx_file.stem}.pdf"

            # Convert
            convert(str(docx_file), str(pdf_file))

            # Check result
            if pdf_file.exists():
                size = pdf_file.stat().st_size
                print(f"✓ ({self._format_size(size)})")
                return True
            else:
                print("❌ Failed")
                return False

        except Exception as e:
            print(f"❌ Error: {e}")
            return False

    def convert_all(self):
        """Convert all DOCX files."""
        docx_files = sorted(self.docx_dir.glob('*.docx'))

        if not docx_files:
            print(f"❌ No DOCX files found in {self.docx_dir}")
            return

        # Filter out temp files
        docx_files = [f for f in docx_files if not f.name.startswith('~$')]

        print(f"\n📑 Converting {len(docx_files)} DOCX files to PDF")
        print(f"📁 Input: {self.docx_dir}")
        print(f"📁 Output: {self.pdf_dir}")
        print(f"\n{'='*60}")

        success = 0
        failed = 0

        for docx_file in docx_files:
            if self.convert_file(docx_file):
                success += 1
            else:
                failed += 1

        print(f"{'='*60}")
        print(f"✅ Conversion complete!")
        print(f"📊 Success: {success} | Failed: {failed}")
        print(f"📂 PDFs saved to: {self.pdf_dir}")
        print(f"{'='*60}\n")


def main():
    parser = argparse.ArgumentParser(
        description='Convert DOCX files to PDF',
        epilog="""
Examples:
  python docx_to_pdf.py ./my_docx_folder
  python docx_to_pdf.py "C:\\Users\\matts\\AI Workspace\\perplexity_chats_docx"
        """
    )

    parser.add_argument('docx_dir', help='Directory containing DOCX files')

    args = parser.parse_args()

    try:
        converter = DOCXToPDF(docx_dir=args.docx_dir)
        converter.convert_all()
    except Exception as e:
        print(f"\n❌ Error: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()
