import flet as ft
from pathlib import Path
from parsers import get_parser, get_all_parsers
from chunkers import DEFAULT_CHUNKER
from exporters import EXPORTERS
from utils import get_stats
from schemas import list_schemas
from ui.schema_editor import SchemaEditor

class ChunkerApp:
    def __init__(self, page: ft.Page):
        self.page = page
        self.page.title = "Smart Chunker"
        self.page.theme_mode = ft.ThemeMode.DARK
        self.page.padding = 20
        
        self.content = ""
        self.chunks = []
        self.current_chunk_idx = 0
        self.file_ext = '.md'
        self.selected_schema = None
        self.schema_editor = SchemaEditor(page, self.refresh_parsers)
        
        self.setup_ui()
    
    def setup_ui(self):
        # Theme toggle
        self.theme_btn = ft.IconButton(
            icon=ft.icons.DARK_MODE,
            on_click=self.toggle_theme
        )
        
        # File picker
        self.file_picker = ft.FilePicker(on_result=self.on_file_picked)
        self.page.overlay.append(self.file_picker)
        
        # Parser/Schema selector
        self.parser_dropdown = ft.Dropdown(
            label="Parser / Schema",
            options=[ft.dropdown.Option(p.get_name()) for p in get_all_parsers()],
            value="Markdown / Text",
            width=250,
            on_change=self.on_parser_changed
        )
        
        self.edit_schema_btn = ft.IconButton(
            icon=ft.icons.EDIT,
            tooltip="Edit Schema",
            on_click=self.edit_current_schema,
            visible=False
        )
        
        self.new_schema_btn = ft.IconButton(
            icon=ft.icons.ADD,
            tooltip="New Schema",
            on_click=lambda e: self.schema_editor.show_editor()
        )
        
        # Input area
        self.input_field = ft.TextField(
            multiline=True,
            min_lines=10,
            max_lines=20,
            hint_text="Paste content or pick a file...",
            on_change=self.update_stats
        )
        
        # Stats
        self.stats_text = ft.Text("0 chars | 0 words | 0 tokens")
        
        # Controls
        self.chunk_size = ft.TextField(label="Chunk Size", value="8000", width=150)
        self.overlap_size = ft.TextField(label="Overlap", value="0", width=150)
        
        # LLM Instructions
        self.llm_instruction = ft.TextField(
            label="LLM Processing Instructions (System Prompt)",
            value="You are analyzing a large document that has been broken into sequential chunks. Do not attempt to summarize or conclude until you have been presented with all chunks. Use the provided context to guide your final analysis.",
            multiline=True,
            min_lines=3,
            max_lines=5
        )
        
        # Buttons
        self.pick_btn = ft.ElevatedButton("Pick File", on_click=lambda _: self.file_picker.pick_files(
            allowed_extensions=['md', 'txt', 'html', 'htm', 'csv', 'tsv']
        ))
        self.process_btn = ft.ElevatedButton("Process", on_click=self.process)
        
        # Output
        self.chunk_count_text = ft.Text("0 chunks")
        self.output_field = ft.TextField(
            multiline=True,
            min_lines=15,
            max_lines=25,
            read_only=True
        )
        
        # Navigation
        self.chunk_nav = ft.Row([
            ft.IconButton(icon=ft.icons.FIRST_PAGE, on_click=lambda _: self.jump_chunk(0)),
            ft.IconButton(icon=ft.icons.NAVIGATE_BEFORE, on_click=lambda _: self.jump_chunk(self.current_chunk_idx - 1)),
            ft.Text("Chunk: "),
            ft.TextField(value="1", width=80, on_submit=lambda e: self.jump_chunk(int(e.control.value) - 1)),
            ft.Text(" / "),
            self.chunk_count_text,
            ft.IconButton(icon=ft.icons.NAVIGATE_NEXT, on_click=lambda _: self.jump_chunk(self.current_chunk_idx + 1)),
            ft.IconButton(icon=ft.icons.LAST_PAGE, on_click=lambda _: self.jump_chunk(len(self.chunks) - 1)),
        ], visible=False)
        
        # Export
        self.output_filename = ft.TextField(
            label="Output Filename",
            value="chunks",
            width=200,
            hint_text="Base filename for output"
        )
        self.export_dropdown = ft.Dropdown(
            label="Export Format",
            options=[ft.dropdown.Option(k) for k in EXPORTERS.keys()],
            value="TXT",
            width=150
        )
        self.export_btn = ft.ElevatedButton("Export", on_click=self.export, disabled=True)
        
        # Layout
        self.page.add(
            ft.Row([ft.Text("Smart Chunker", size=24, weight=ft.FontWeight.BOLD), self.theme_btn]),
            ft.Divider(),
            ft.Row([self.pick_btn, self.parser_dropdown, self.edit_schema_btn, self.new_schema_btn]),
            ft.Row([self.stats_text]),
            self.input_field,
            ft.Row([self.chunk_size, self.overlap_size, self.process_btn]),
            self.llm_instruction,
            ft.Divider(),
            self.chunk_nav,
            self.output_field,
            ft.Row([self.output_filename, self.export_dropdown, self.export_btn])
        )
    
    def toggle_theme(self, e):
        self.page.theme_mode = ft.ThemeMode.LIGHT if self.page.theme_mode == ft.ThemeMode.DARK else ft.ThemeMode.DARK
        self.theme_btn.icon = ft.icons.LIGHT_MODE if self.page.theme_mode == ft.ThemeMode.LIGHT else ft.icons.DARK_MODE
        self.page.update()
    
    def on_file_picked(self, e: ft.FilePickerResultEvent):
        if e.files:
            file_path = e.files[0].path
            self.file_ext = Path(file_path).suffix
            with open(file_path, 'r', encoding='utf-8') as f:
                self.content = f.read()
            self.input_field.value = self.content
            self.update_stats(None)
            self.page.update()
    
    def update_stats(self, e):
        text = self.input_field.value or ""
        stats = get_stats(text)
        self.stats_text.value = f"{stats['chars']} chars | {stats['words']} words | {stats['tokens']} tokens"
        self.page.update()
    
    def on_parser_changed(self, e):
        parser_name = self.parser_dropdown.value
        # Check if it's a schema-based parser
        for schema_name in list_schemas():
            parser = get_parser('.html', schema_name)
            if parser.get_name() == parser_name:
                self.selected_schema = schema_name
                self.edit_schema_btn.visible = True
                self.page.update()
                return
        self.selected_schema = None
        self.edit_schema_btn.visible = False
        self.page.update()
    
    def edit_current_schema(self, e):
        if self.selected_schema:
            self.schema_editor.show_editor(self.selected_schema)
    
    def refresh_parsers(self):
        self.parser_dropdown.options = [ft.dropdown.Option(p.get_name()) for p in get_all_parsers()]
        self.page.update()
    
    def process(self, e):
        content = self.input_field.value
        if not content:
            return
        
        chunk_size = int(self.chunk_size.value or 8000)
        overlap = int(self.overlap_size.value or 0)
        
        # Parse
        parser = get_parser(self.file_ext, self.selected_schema)
        sections = parser.parse(content)
        
        # Chunk
        self.chunks = DEFAULT_CHUNKER.chunk(sections, chunk_size, overlap)
        
        # Display
        self.chunk_count_text.value = f"{len(self.chunks)} chunks"
        self.chunk_nav.visible = len(self.chunks) > 0
        self.export_btn.disabled = len(self.chunks) == 0
        
        if self.chunks:
            self.jump_chunk(0)
        
        self.page.update()
    
    def jump_chunk(self, idx):
        if 0 <= idx < len(self.chunks):
            self.current_chunk_idx = idx
            chunk = self.chunks[idx]
            self.output_field.value = f"--- Chunk {chunk['num']}/{len(self.chunks)} ---\n\n{chunk['content']}"
            self.page.update()
    
    def export(self, e):
        if not self.chunks:
            return
        
        format_name = self.export_dropdown.value
        exporter = EXPORTERS[format_name]
        base_name = self.output_filename.value or "chunks"
        llm_instruction = self.llm_instruction.value or ""
        
        output_path = f"{base_name}{exporter.get_extension()}"
        exporter.export(self.chunks, output_path, base_name, llm_instruction)
        
        self.page.snack_bar = ft.SnackBar(ft.Text(f"Exported to {output_path}"))
        self.page.snack_bar.open = True
        self.page.update()

def main(page: ft.Page):
    ChunkerApp(page)

if __name__ == "__main__":
    ft.app(target=main)
