"""
Lightweight Markdown to PDF - Pure Python, no external services.
Uses reportlab for PDF generation - works offline, no Docker needed.

Usage:
    python lightweight_md_to_pdf.py <markdown_directory>
"""

import re
import sys
import argparse
from pathlib import Path
from reportlab.lib.pagesizes import letter
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib.units import inch
from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, PageBreak
from reportlab.lib.enums import TA_LEFT, TA_CENTER
from reportlab.lib.colors import HexColor

# Fix Windows console encoding
if sys.platform == 'win32':
    import codecs
    sys.stdout = codecs.getwriter('utf-8')(sys.stdout.buffer, 'ignore')
    sys.stderr = codecs.getwriter('utf-8')(sys.stderr.buffer, 'ignore')


class LightweightMDToPDF:
    """Lightweight markdown to PDF converter."""

    def __init__(self, markdown_dir: str):
        self.markdown_dir = Path(markdown_dir)
        self.pdf_dir = self.markdown_dir.parent / f"{self.markdown_dir.name}_pdf"
        self.pdf_dir.mkdir(exist_ok=True)
        self.setup_styles()

    def setup_styles(self):
        """Setup PDF styles."""
        self.styles = getSampleStyleSheet()

        # Customize styles (use unique names)
        self.styles.add(ParagraphStyle(
            name='CustomTitle',
            parent=self.styles['Heading1'],
            fontSize=24,
            textColor=HexColor('#2c3e50'),
            spaceAfter=20,
            alignment=TA_CENTER
        ))

        self.styles.add(ParagraphStyle(
            name='CustomH2',
            parent=self.styles['Heading2'],
            fontSize=16,
            textColor=HexColor('#34495e'),
            spaceBefore=15,
            spaceAfter=10
        ))

        self.styles.add(ParagraphStyle(
            name='CustomH3',
            parent=self.styles['Heading3'],
            fontSize=12,
            textColor=HexColor('#16a085'),
            spaceBefore=10,
            spaceAfter=6,
            fontName='Helvetica-Bold'
        ))

    def _format_size(self, size_bytes: int) -> str:
        """Format byte size."""
        for unit in ['B', 'KB', 'MB', 'GB']:
            if size_bytes < 1024.0:
                return f"{size_bytes:.1f} {unit}"
            size_bytes /= 1024.0
        return f"{size_bytes:.1f} TB"

    def parse_markdown(self, md_text: str) -> list:
        """Parse markdown and return list of flowable elements."""
        elements = []
        lines = md_text.split('\n')
        i = 0

        while i < len(lines):
            line = lines[i].rstrip()

            # Skip empty lines
            if not line:
                i += 1
                continue

            # H1
            if line.startswith('# '):
                text = line[2:].strip()
                elements.append(Paragraph(text, self.styles['CustomTitle']))
                elements.append(Spacer(1, 0.2*inch))
                i += 1
                continue

            # H2
            if line.startswith('## '):
                text = line[3:].strip()
                elements.append(Paragraph(text, self.styles['CustomH2']))
                elements.append(Spacer(1, 0.1*inch))
                i += 1
                continue

            # H3
            if line.startswith('### '):
                text = line[4:].strip()
                elements.append(Paragraph(text, self.styles['CustomH3']))
                elements.append(Spacer(1, 0.05*inch))
                i += 1
                continue

            # Horizontal rule
            if line.strip() in ['---', '===', '___']:
                elements.append(Spacer(1, 0.1*inch))
                i += 1
                continue

            # Strip ALL HTML tags to avoid XML parsing issues
            line = re.sub(r'<[^>]+>', '', line)

            # Remove markdown formatting (render as plain text)
            line = re.sub(r'\*\*(.*?)\*\*', r'\1', line)  # Bold
            line = re.sub(r'__(.*?)__', r'\1', line)  # Bold
            line = re.sub(r'\*(.*?)\*', r'\1', line)  # Italic
            line = re.sub(r'_(.*?)_', r'\1', line)  # Italic

            # Escape XML special characters AFTER stripping HTML
            line = line.replace('&', '&amp;')
            line = line.replace('<', '&lt;')
            line = line.replace('>', '&gt;')

            # Regular paragraph
            if line:
                # Handle multi-line paragraphs
                para_lines = [line]
                i += 1
                while i < len(lines) and lines[i].strip() and not lines[i].startswith('#'):
                    para_lines.append(lines[i].rstrip())
                    i += 1

                text = ' '.join(para_lines)
                elements.append(Paragraph(text, self.styles['Normal']))
                elements.append(Spacer(1, 0.1*inch))
                continue

            i += 1

        return elements

    def convert_file(self, md_file: Path):
        """Convert a single markdown file to PDF."""
        print(f"Converting {md_file.name}...", end=' ')

        try:
            # Read markdown
            with open(md_file, 'r', encoding='utf-8') as f:
                md_content = f.read()

            # Create PDF
            pdf_file = self.pdf_dir / f"{md_file.stem}.pdf"

            doc = SimpleDocTemplate(
                str(pdf_file),
                pagesize=letter,
                rightMargin=inch,
                leftMargin=inch,
                topMargin=inch,
                bottomMargin=inch
            )

            # Parse markdown and build PDF
            elements = self.parse_markdown(md_content)
            doc.build(elements)

            size = pdf_file.stat().st_size
            print(f"OK ({self._format_size(size)})")
            return True

        except Exception as e:
            print(f"ERROR: {e}")
            return False

    def convert_all(self):
        """Convert all markdown files."""
        md_files = sorted(self.markdown_dir.glob('*.md'))

        if not md_files:
            print(f"No markdown files found in {self.markdown_dir}")
            return

        print(f"\nLightweight Markdown to PDF Converter")
        print(f"Pure Python - No Docker, No Services")
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
        print(f"PDFs saved to: {self.pdf_dir}")
        print("="*60)


def main():
    parser = argparse.ArgumentParser(
        description='Lightweight markdown to PDF converter (pure Python)',
        epilog="""
Examples:
  python lightweight_md_to_pdf.py ./my_markdown_folder
  python lightweight_md_to_pdf.py "perplexity_chats_clean_markdown"
        """
    )

    parser.add_argument('markdown_dir', help='Directory containing markdown files')

    args = parser.parse_args()

    try:
        converter = LightweightMDToPDF(markdown_dir=args.markdown_dir)
        converter.convert_all()
    except Exception as e:
        print(f"\nError: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()
