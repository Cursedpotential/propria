#!/usr/bin/env python3
"""
Pandoc Toolkit - Complete CLI with all the options you need.
Fancy menus, chunking, Lua filters, the works.

Usage:
    python pandoc_toolkit.py
"""

import sys
import subprocess
from pathlib import Path
from typing import List, Dict
import questionary
from questionary import Choice
from rich.console import Console
from rich.panel import Panel
from rich.table import Table
from rich import print as rprint

# Fix Windows console encoding
if sys.platform == 'win32':
    import codecs
    sys.stdout = codecs.getwriter('utf-8')(sys.stdout.buffer, 'ignore')
    sys.stderr = codecs.getwriter('utf-8')(sys.stderr.buffer, 'ignore')

console = Console()

LUA_FILTERS_DIR = Path(__file__).parent / "pandoc-lua-filters"


class PandocToolkit:
    """Complete pandoc toolkit with interactive CLI."""

    FORMATS = {
        'pdf': 'PDF Document',
        'docx': 'Microsoft Word',
        'html': 'HTML Document',
        'html5': 'HTML5 Document',
        'markdown': 'Markdown',
        'gfm': 'GitHub-Flavored Markdown',
        'latex': 'LaTeX',
        'epub': 'EPUB eBook',
        'rst': 'reStructuredText',
        'txt': 'Plain Text',
    }

    PDF_ENGINES = [
        'wkhtmltopdf',
        'weasyprint',
        'pdflatex',
        'xelatex',
        'lualatex',
        'context',
    ]

    LUA_FILTERS = {
        'wordcount': 'Count words in document',
        'pagebreak': 'Insert page breaks',
        'include-files': 'Include external files',
        'table-filter': 'Enhanced table processing',
        'section-refs': 'Section references',
        'scholarly-metadata': 'Academic metadata',
        'diagram-generator': 'Generate diagrams',
        'task-list': 'Task list support',
    }

    def __init__(self):
        self.config = {
            'input_path': None,
            'output_format': 'pdf',
            'output_path': None,
            'pdf_engine': 'wkhtmltopdf',
            'chunk': False,
            'chunk_level': 2,
            'lua_filters': [],
            'toc': False,
            'numbered': False,
            'standalone': True,
            'template': None,
        }

    def show_banner(self):
        """Show fancy banner."""
        banner = """
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║              🐼 PANDOC TOOLKIT 🐼                        ║
║                                                           ║
║        Universal Document Converter & Processor          ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
        """
        rprint(Panel(banner, style="bold cyan"))

    def main_menu(self):
        """Show main menu."""
        while True:
            self.show_banner()

            choice = questionary.select(
                "What do you want to do?",
                choices=[
                    Choice("📄 Convert Document", value="convert"),
                    Choice("✂️  Split/Chunk Document", value="chunk"),
                    Choice("🔧 Batch Process Directory", value="batch"),
                    Choice("📚 Facebook Chat HTML Processor", value="facebook"),
                    Choice("⚙️  Configure Settings", value="settings"),
                    Choice("ℹ️  Show Current Config", value="show_config"),
                    Choice("🚪 Exit", value="exit"),
                ]
            ).ask()

            if choice == "exit":
                console.print("\n👋 Goodbye!\n", style="bold green")
                break
            elif choice == "convert":
                self.convert_workflow()
            elif choice == "chunk":
                self.chunk_workflow()
            elif choice == "batch":
                self.batch_workflow()
            elif choice == "facebook":
                self.facebook_workflow()
            elif choice == "settings":
                self.settings_menu()
            elif choice == "show_config":
                self.show_config()

    def convert_workflow(self):
        """Single file conversion workflow."""
        console.print("\n📄 Document Conversion", style="bold cyan")

        # Get input file
        input_path = questionary.path(
            "Input file path:",
            only_files=True
        ).ask()

        if not input_path:
            return

        self.config['input_path'] = Path(input_path)

        # Select output format
        format_choices = [Choice(f"{v} (.{k})", value=k) for k, v in self.FORMATS.items()]
        output_format = questionary.select(
            "Output format:",
            choices=format_choices
        ).ask()

        self.config['output_format'] = output_format

        # Get output path
        default_output = self.config['input_path'].with_suffix(f'.{output_format}')
        output_path = questionary.text(
            "Output file path:",
            default=str(default_output)
        ).ask()

        self.config['output_path'] = Path(output_path)

        # PDF-specific options
        if output_format == 'pdf':
            self.pdf_options()

        # Lua filters
        if questionary.confirm("Use Lua filters?", default=False).ask():
            self.select_lua_filters()

        # Document options
        self.config['toc'] = questionary.confirm("Generate Table of Contents?", default=False).ask()
        self.config['numbered'] = questionary.confirm("Number sections?", default=False).ask()

        # Execute conversion
        self.execute_conversion()

    def chunk_workflow(self):
        """Document chunking workflow."""
        console.print("\n✂️  Document Chunking", style="bold cyan")

        input_path = questionary.path(
            "Input file path:",
            only_files=True
        ).ask()

        if not input_path:
            return

        self.config['input_path'] = Path(input_path)
        self.config['chunk'] = True

        # Chunk level
        self.config['chunk_level'] = questionary.select(
            "Chunk at which heading level?",
            choices=[
                Choice("# Heading 1", value=1),
                Choice("## Heading 2", value=2),
                Choice("### Heading 3", value=3),
            ]
        ).ask()

        # Output format
        format_choices = [Choice(f"{v} (.{k})", value=k) for k, v in self.FORMATS.items()]
        self.config['output_format'] = questionary.select(
            "Output format for chunks:",
            choices=format_choices
        ).ask()

        # Output directory
        output_dir = questionary.text(
            "Output directory:",
            default=str(self.config['input_path'].stem + "_chunks")
        ).ask()

        self.config['output_path'] = Path(output_dir)

        # Execute chunking
        self.execute_chunking()

    def batch_workflow(self):
        """Batch processing workflow."""
        console.print("\n🔧 Batch Processing", style="bold cyan")

        input_dir = questionary.path(
            "Input directory:",
            only_directories=True
        ).ask()

        if not input_dir:
            return

        # File pattern
        pattern = questionary.text(
            "File pattern (e.g., *.md, *.html):",
            default="*.md"
        ).ask()

        # Output format
        format_choices = [Choice(f"{v} (.{k})", value=k) for k, v in self.FORMATS.items()]
        output_format = questionary.select(
            "Output format:",
            choices=format_choices
        ).ask()

        # Execute batch
        self.execute_batch(Path(input_dir), pattern, output_format)

    def facebook_workflow(self):
        """Facebook chat HTML processor."""
        console.print("\n📚 Facebook Chat HTML Processor", style="bold cyan")
        console.print("Perfect for those massive multi-year chat exports!\n")

        input_path = questionary.path(
            "Facebook HTML file:",
            only_files=True
        ).ask()

        if not input_path:
            return

        action = questionary.select(
            "What do you want to do?",
            choices=[
                Choice("Split into chunks (by date/size)", value="chunk"),
                Choice("Convert to clean format", value="convert"),
                Choice("Extract & organize media links", value="media"),
                Choice("All of the above", value="all"),
            ]
        ).ask()

        if action in ["chunk", "all"]:
            chunk_method = questionary.select(
                "Chunking method:",
                choices=[
                    Choice("By heading level (recommended)", value="heading"),
                    Choice("By file size", value="size"),
                    Choice("By date/year", value="date"),
                ]
            ).ask()

            self.execute_facebook_processing(Path(input_path), chunk_method, action)

    def pdf_options(self):
        """Configure PDF-specific options."""
        engine = questionary.select(
            "PDF Engine:",
            choices=[Choice(e, value=e) for e in self.PDF_ENGINES]
        ).ask()

        self.config['pdf_engine'] = engine

    def select_lua_filters(self):
        """Select Lua filters."""
        if not LUA_FILTERS_DIR.exists():
            console.print("⚠️  Lua filters not found. Clone them first!", style="yellow")
            return

        filter_choices = [
            Choice(f"{k}: {v}", value=k)
            for k, v in self.LUA_FILTERS.items()
        ]

        selected = questionary.checkbox(
            "Select Lua filters:",
            choices=filter_choices
        ).ask()

        self.config['lua_filters'] = selected

    def settings_menu(self):
        """Settings configuration menu."""
        console.print("\n⚙️  Settings", style="bold cyan")

        # Show current settings
        table = Table(title="Current Settings")
        table.add_column("Setting", style="cyan")
        table.add_column("Value", style="green")

        for key, value in self.config.items():
            if value is not None:
                table.add_row(key, str(value))

        console.print(table)

        questionary.press_any_key_to_continue("Press any key to continue...").ask()

    def show_config(self):
        """Display current configuration."""
        table = Table(title="📋 Current Configuration", show_header=True)
        table.add_column("Setting", style="cyan", no_wrap=True)
        table.add_column("Value", style="green")

        for key, value in self.config.items():
            table.add_row(key.replace('_', ' ').title(), str(value))

        console.print("\n")
        console.print(table)
        console.print("\n")

        questionary.press_any_key_to_continue().ask()

    def build_pandoc_command(self) -> List[str]:
        """Build pandoc command from config."""
        cmd = ['pandoc', str(self.config['input_path'])]

        # Output
        if not self.config['chunk']:
            cmd.extend(['-o', str(self.config['output_path'])])

        # Format
        cmd.extend(['-t', self.config['output_format']])

        # PDF engine
        if self.config['output_format'] == 'pdf' and self.config['pdf_engine']:
            cmd.append(f"--pdf-engine={self.config['pdf_engine']}")
            cmd.extend([
                '-V', 'geometry:margin=1in',
                '-V', 'fontsize=11pt',
            ])

        # Chunking
        if self.config['chunk']:
            cmd.append(f"--split-level={self.config['chunk_level']}")
            cmd.append('--split-at-heading')

        # TOC
        if self.config['toc']:
            cmd.append('--toc')

        # Numbering
        if self.config['numbered']:
            cmd.append('--number-sections')

        # Standalone
        if self.config['standalone']:
            cmd.append('--standalone')

        # Lua filters
        for filter_name in self.config['lua_filters']:
            filter_path = LUA_FILTERS_DIR / filter_name / f"{filter_name}.lua"
            if filter_path.exists():
                cmd.extend(['--lua-filter', str(filter_path)])

        return cmd

    def execute_conversion(self):
        """Execute the conversion."""
        cmd = self.build_pandoc_command()

        console.print("\n🚀 Executing conversion...\n", style="bold green")
        console.print(f"Command: {' '.join(cmd)}\n", style="dim")

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=300
            )

            if result.returncode == 0:
                console.print("✅ Conversion successful!", style="bold green")
                if self.config['output_path']:
                    size = self.config['output_path'].stat().st_size / 1024
                    console.print(f"📄 Output: {self.config['output_path']} ({size:.1f} KB)\n")
            else:
                console.print("❌ Conversion failed!", style="bold red")
                console.print(f"Error: {result.stderr}\n", style="red")

        except Exception as e:
            console.print(f"❌ Error: {e}", style="bold red")

        questionary.press_any_key_to_continue().ask()

    def execute_chunking(self):
        """Execute document chunking."""
        output_dir = self.config['output_path']
        output_dir.mkdir(exist_ok=True)

        # Pandoc chunking command
        cmd = [
            'pandoc',
            str(self.config['input_path']),
            '-t', self.config['output_format'],
            f"--split-level={self.config['chunk_level']}",
            '-o', str(output_dir / f"chunk_%n.{self.config['output_format']}")
        ]

        console.print("\n✂️  Chunking document...\n", style="bold green")
        console.print(f"Command: {' '.join(cmd)}\n", style="dim")

        try:
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=300)

            if result.returncode == 0:
                chunks = list(output_dir.glob(f"chunk_*.{self.config['output_format']}"))
                console.print(f"✅ Created {len(chunks)} chunks!", style="bold green")
                console.print(f"📂 Output: {output_dir}\n")
            else:
                console.print("❌ Chunking failed!", style="bold red")
                console.print(f"Error: {result.stderr}\n", style="red")

        except Exception as e:
            console.print(f"❌ Error: {e}", style="bold red")

        questionary.press_any_key_to_continue().ask()

    def execute_batch(self, input_dir: Path, pattern: str, output_format: str):
        """Execute batch conversion."""
        files = list(input_dir.glob(pattern))

        if not files:
            console.print(f"❌ No files found matching {pattern}", style="red")
            return

        output_dir = input_dir / f"{output_format}_output"
        output_dir.mkdir(exist_ok=True)

        console.print(f"\n🔧 Processing {len(files)} files...\n", style="bold green")

        success = 0
        failed = 0

        for file in files:
            output_file = output_dir / f"{file.stem}.{output_format}"
            cmd = ['pandoc', str(file), '-o', str(output_file)]

            try:
                result = subprocess.run(cmd, capture_output=True, timeout=60)
                if result.returncode == 0:
                    console.print(f"✅ {file.name}", style="green")
                    success += 1
                else:
                    console.print(f"❌ {file.name}", style="red")
                    failed += 1
            except:
                console.print(f"❌ {file.name} (timeout)", style="red")
                failed += 1

        console.print(f"\n📊 Complete! Success: {success} | Failed: {failed}")
        console.print(f"📂 Output: {output_dir}\n")

        questionary.press_any_key_to_continue().ask()

    def execute_facebook_processing(self, input_file: Path, chunk_method: str, action: str):
        """Process Facebook chat HTML."""
        console.print("\n📚 Processing Facebook chat HTML...\n", style="bold green")

        output_dir = input_file.parent / f"{input_file.stem}_processed"
        output_dir.mkdir(exist_ok=True)

        # Build command based on method
        if chunk_method == "heading":
            cmd = [
                'pandoc',
                str(input_file),
                '-t', 'html',
                '--split-level=2',
                '-o', str(output_dir / 'chunk_%n.html')
            ]
        elif chunk_method == "size":
            # Convert to markdown first, then chunk
            cmd = [
                'pandoc',
                str(input_file),
                '-t', 'markdown',
                '-o', str(output_dir / 'messages.md')
            ]

        try:
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=600)

            if result.returncode == 0:
                console.print("✅ Processing complete!", style="bold green")
                console.print(f"📂 Output: {output_dir}\n")
            else:
                console.print("❌ Processing failed!", style="bold red")
                console.print(f"Error: {result.stderr}\n", style="red")

        except Exception as e:
            console.print(f"❌ Error: {e}", style="bold red")

        questionary.press_any_key_to_continue().ask()


def main():
    try:
        toolkit = PandocToolkit()
        toolkit.main_menu()
    except KeyboardInterrupt:
        console.print("\n\n👋 Interrupted. Goodbye!\n", style="bold yellow")
    except Exception as e:
        console.print(f"\n❌ Error: {e}", style="bold red")
        import traceback
        traceback.print_exc()


if __name__ == '__main__':
    main()
