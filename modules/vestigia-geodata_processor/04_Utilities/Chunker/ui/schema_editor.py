import flet as ft
from schemas import load_schema, save_schema, list_schemas
import json

class SchemaEditor:
    def __init__(self, page: ft.Page, on_save_callback=None):
        self.page = page
        self.on_save_callback = on_save_callback
        self.current_schema = None
        self.schema_name = None
        
    def show_editor(self, schema_name: str = None):
        self.schema_name = schema_name
        self.current_schema = load_schema(schema_name) if schema_name else {
            "name": "New Schema",
            "description": "",
            "file_pattern": "*.html",
            "selectors": {"container": ""},
            "output_format": "{content}\\n"
        }
        
        # Fields
        name_field = ft.TextField(label="Schema Name", value=self.current_schema.get('name', ''))
        desc_field = ft.TextField(label="Description", value=self.current_schema.get('description', ''))
        pattern_field = ft.TextField(label="File Pattern", value=self.current_schema.get('file_pattern', '*.html'))
        
        # Selectors
        selectors = self.current_schema.get('selectors', {})
        selector_fields = []
        
        def add_selector_field(key='', value=''):
            row = ft.Row([
                ft.TextField(label="Field Name", value=key, width=200),
                ft.TextField(label="CSS Selector", value=value, width=300),
                ft.IconButton(icon=ft.icons.DELETE, on_click=lambda e: remove_selector(row))
            ])
            selector_fields.append(row)
            return row
        
        def remove_selector(row):
            selector_fields.remove(row)
            selectors_column.controls.remove(row)
            self.page.update()
        
        selectors_column = ft.Column([add_selector_field(k, v) for k, v in selectors.items()])
        
        add_selector_btn = ft.ElevatedButton(
            "Add Selector",
            on_click=lambda e: (selectors_column.controls.append(add_selector_field()), self.page.update())
        )
        
        output_format_field = ft.TextField(
            label="Output Format (use {field_name} placeholders)",
            value=self.current_schema.get('output_format', '{content}\\n'),
            multiline=True,
            min_lines=3
        )
        
        def save_schema_data(e):
            # Build schema
            new_schema = {
                "name": name_field.value,
                "description": desc_field.value,
                "file_pattern": pattern_field.value,
                "selectors": {},
                "output_format": output_format_field.value
            }
            
            for row in selector_fields:
                key = row.controls[0].value
                value = row.controls[1].value
                if key and value:
                    new_schema['selectors'][key] = value
            
            # Save
            save_name = self.schema_name or name_field.value.lower().replace(' ', '_')
            save_schema(save_name, new_schema)
            
            if self.on_save_callback:
                self.on_save_callback()
            
            dialog.open = False
            self.page.update()
            self.page.snack_bar = ft.SnackBar(ft.Text(f"Schema '{new_schema['name']}' saved!"))
            self.page.snack_bar.open = True
            self.page.update()
        
        dialog = ft.AlertDialog(
            title=ft.Text("Schema Editor"),
            content=ft.Container(
                content=ft.Column([
                    name_field,
                    desc_field,
                    pattern_field,
                    ft.Divider(),
                    ft.Text("CSS Selectors", weight=ft.FontWeight.BOLD),
                    selectors_column,
                    add_selector_btn,
                    ft.Divider(),
                    output_format_field,
                ], scroll=ft.ScrollMode.AUTO, height=500, width=600),
                padding=10
            ),
            actions=[
                ft.TextButton("Cancel", on_click=lambda e: (setattr(dialog, 'open', False), self.page.update())),
                ft.ElevatedButton("Save", on_click=save_schema_data)
            ]
        )
        
        self.page.dialog = dialog
        dialog.open = True
        self.page.update()
