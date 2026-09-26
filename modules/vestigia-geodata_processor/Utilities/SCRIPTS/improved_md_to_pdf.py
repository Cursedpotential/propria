"""
Improved Markdown to PDF using markdown-it-py + reportlab.
Better markdown parsing than standard library.

Usage:
    python improved_md_to_pdf.py <markdown_directory>
"""

import re
import sys
import argparse
from pathlib import Path
from markdown_it import MarkdownIt
from mdit_py_plugins.footnote import footnote_plugin
from mdit_py_plugins.deflist import deflist_plugin
from reportlab.lib.pagesizes import letter
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib.units import inch
from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, PageBreak
from reportlab.lib.colors import HexColor

# Fix Windows console encoding
if sys.platform == 'win32':
    import codecs
    sys.stdout = codecs.getwriter('utf-8')(sys.stdout.buffer, 'ignore')
    sys.stderr = codecs.getwriter('utf-8')(sys.stderr.buffer, 'ignore')


class ImprovedMDToPDF:
    """Improved markdown to PDF using markdown-it-py."""

    def __init__(self, markdown_dir: str):
        self.markdown_dir = Path(markdown_dir)
        self.pdf_dir = self.markdown_dir.parent / f"{self.markdown_dir.name}_pdf"
        self.pdf_dir.mkdir(exist_ok=True)
        self.setup_styles()

        # Setup markdown-it with plugins
        self.md = (
            MarkdownIt()
            .enable('table')
            .use(footnote_plugin)
            .use(deflist_plugin)
        )

    def setup_styles(self):
        """Setup PDF styles."""
        self.styles = getSampleStyleSheet()

        self.styles.add(ParagraphStyle(
            name='CustomTitle',
            parent=self.styles['Heading1'],
            fontSize=24,
            textColor=HexColor('#2c3e50'),
            spaceAfter=20,
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
        ))

    def _format_size(self, size_bytes: int) -> str:
        """Format byte size."""
        for unit in ['B', 'KB', 'MB', 'GB']:
            if size_bytes < 1024.0:
                return f"{size_bytes:.1f} {unit}"
            size_bytes /= 1024.0
        return f"{size_bytes:.1f} TB"

    def _escape_xml(self, text: str) -> str:
        """Escape XML special characters."""
        return (text
                .replace('&', '&amp;')
                .replace('<', '&lt;')
                .replace('>', '&gt;'))

    def _process_token(self, token, tokens, idx) -> list:
        """Process a markdown-it token into reportlab elements."""
        elements = []

        if token.type == 'heading_open':
            level = int(token.tag[1])
            # Get the inline content
            content_token = tokens[idx + 1]
            if content_token.type == 'inline':
                text = self._escape_xml(content_token.content)

                if level == 1:
                    elements.append(Paragraph(text, self.styles['CustomTitle']))
                    elements.append(Spacer(1, 0.2*inch))
                elif level == 2:
                    elements.append(Paragraph(text, self.styles['CustomH2']))
                    elements.append(Spacer(1, 0.1*inch))
                elif level == 3:
                    elements.append(Paragraph(text, self.styles['CustomH3']))
                    elements.append(Spacer(1, 0.05*inch))

        elif token.type == 'paragraph_open':
            # Get paragraph content
            content_token = tokens[idx + 1]
            if content_token.type == 'inline':
                text = self._escape_xml(content_token.content)
                if text.strip():
                    elements.append(Paragraph(text, self.styles['Normal']))
                    elements.append(Spacer(1, 0.1*inch))

        elif token.type == 'hr':
            elements.append(Spacer(1, 0.1*inch))

        return elements

    def convert_file(self, md_file: Path):
        """Convert a single markdown file to PDF."""
        print(f"Converting {md_file.name}...", end=' ')

        try:
            # Read markdown
            with open(md_file, 'r', encoding='utf-8') as f:
                md_content = f.read()

            # Parse with markdown-it
            tokens = self.md.parse(md_content)

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

            # Process tokens into elements
            elements = []
            i = 0
            while i < len(tokens):
                token_elements = self._process_token(tokens[i], tokens, i)
                elements.extend(token_elements)
                i += 1

            # Build PDF
            doc.build(elements)

            size = pdf_file.stat().st_size
            print(f"OK ({self._format_size(size)})")
            return True

        except Exception as e:
            print(f"ERROR: {e}")
            import traceback
            traceback.print_exc()
            return False

    def convert_all(self):
        """Convert all markdown files."""
        md_files = sorted(self.markdown_dir.glob('*.md'))

        if not md_files:
            print(f"No markdown files found in {self.markdown_dir}")
            return

        print(f"\nImproved Markdown to PDF Converter")
        print(f"Using: markdown-it-py + reportlab")
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
        description='Improved markdown to PDF using markdown-it-py',
        epilog="""
Examples:
  python improved_md_to_pdf.py ./my_markdown_folder
  python improved_md_to_pdf.py "perplexity_chats_clean_markdown"
        """
    )

    parser.add_argument('markdown_dir', help='Directory containing markdown files')

    args = parser.parse_args()

    try:
        converter = ImprovedMDToPDF(markdown_dir=args.markdown_dir)
        converter.convert_all()
    except Exception as e:
        print(f"\nError: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()
