"""
Markdown to PDF converter using standard libraries.
Uses markdown + weasyprint for professional PDF output.

Usage:
    python markdown_to_pdf.py <markdown_directory>
"""

import sys
import argparse
from pathlib import Path
import markdown
from weasyprint import HTML, CSS

# Fix Windows console encoding
if sys.platform == 'win32':
    import codecs
    sys.stdout = codecs.getwriter('utf-8')(sys.stdout.buffer, 'ignore')
    sys.stderr = codecs.getwriter('utf-8')(sys.stderr.buffer, 'ignore')


# Professional CSS styling
PDF_CSS = """
@page {
    size: Letter;
    margin: 1in;
}

body {
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    font-size: 11pt;
    line-height: 1.6;
    color: #333;
}

h1 {
    color: #2c3e50;
    border-bottom: 3px solid #3498db;
    padding-bottom: 10px;
    font-size: 24pt;
}

h2 {
    color: #34495e;
    border-bottom: 2px solid #95a5a6;
    padding-bottom: 5px;
    margin-top: 30px;
    font-size: 18pt;
}

h3 {
    color: #16a085;
    font-size: 14pt;
    margin-top: 20px;
}

code {
    background-color: #f4f4f4;
    padding: 2px 6px;
    border-radius: 3px;
    font-family: 'Consolas', 'Monaco', monospace;
    font-size: 10pt;
}

pre {
    background-color: #f4f4f4;
    padding: 15px;
    border-radius: 5px;
    border-left: 4px solid #3498db;
    overflow-x: auto;
}

pre code {
    background-color: transparent;
    padding: 0;
}

hr {
    border: none;
    border-top: 1px solid #ddd;
    margin: 20px 0;
}

blockquote {
    border-left: 4px solid #3498db;
    padding-left: 15px;
    margin-left: 0;
    color: #555;
    font-style: italic;
}

strong {
    color: #2c3e50;
}

a {
    color: #3498db;
    text-decoration: none;
}

a:hover {
    text-decoration: underline;
}

table {
    border-collapse: collapse;
    width: 100%;
    margin: 20px 0;
}

th, td {
    border: 1px solid #ddd;
    padding: 8px;
    text-align: left;
}

th {
    background-color: #3498db;
    color: white;
}

tr:nth-child(even) {
    background-color: #f9f9f9;
}
"""


class MarkdownToPDF:
    """Convert markdown files to PDF using markdown + weasyprint."""

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

    def convert_file(self, md_file: Path):
        """Convert a single markdown file to PDF."""
        print(f"Converting {md_file.name}...", end=' ')

        try:
            # Read markdown
            with open(md_file, 'r', encoding='utf-8') as f:
                md_content = f.read()

            # Convert to HTML
            html_content = markdown.markdown(
                md_content,
                extensions=[
                    'extra',  # Tables, fenced code, etc.
                    'codehilite',  # Code highlighting
                    'nl2br',  # Newline to <br>
                ]
            )

            # Wrap in HTML document
            full_html = f"""
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>{md_file.stem}</title>
</head>
<body>
    {html_content}
</body>
</html>
"""

            # Convert to PDF
            pdf_file = self.pdf_dir / f"{md_file.stem}.pdf"

            HTML(string=full_html).write_pdf(
                pdf_file,
                stylesheets=[CSS(string=PDF_CSS)]
            )

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

        print(f"\nMarkdown to PDF Converter")
        print(f"Using: markdown + weasyprint")
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
        description='Convert markdown files to PDF',
        epilog="""
Examples:
  python markdown_to_pdf.py ./my_markdown_folder
  python markdown_to_pdf.py "C:\\Users\\matts\\AI Workspace\\perplexity_chats_clean_markdown"
        """
    )

    parser.add_argument('markdown_dir', help='Directory containing markdown files')

    args = parser.parse_args()

    try:
        converter = MarkdownToPDF(markdown_dir=args.markdown_dir)
        converter.convert_all()
    except Exception as e:
        print(f"\nError: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()
