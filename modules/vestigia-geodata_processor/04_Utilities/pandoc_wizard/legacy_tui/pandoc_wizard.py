import os
import subprocess
import questionary
from rich.console import Console
from rich.panel import Panel
from rich.text import Text

console = Console()

LUA_FILTERS_PATH = r"C:\Users\matts\AI Workspace\pandoc-lua-filters"

def get_lua_filters():
    filters = []
    if os.path.exists(LUA_FILTERS_PATH):
        for root, dirs, files in os.walk(LUA_FILTERS_PATH):
            for file in files:
                if file.endswith(".lua"):
                    filters.append(os.path.join(root, file))
    return filters

def run_pandoc_command(cmd):
    console.print(Panel(f"[bold green]Running Command:[/bold green]\n{cmd}", title="Execution"))
    try:
        subprocess.run(cmd, shell=True, check=True)
        console.print("[bold blue]Success![/bold blue]")
    except subprocess.CalledProcessError as e:
        console.print(f"[bold red]Error:[/bold red] {e}")

def html_chunking_wizard():
    console.print("[bold cyan]HTML Chunking Module[/bold cyan]")
    console.print("[dim]Uses 'chunkedhtml' format to split document into multiple files based on headers.[/dim]")
    
    # 1. Input File
    input_file = questionary.path("Select the input file (HTML/MD/etc):").ask()
    if not input_file or not os.path.exists(input_file):
        console.print("[red]File not found![/red]")
        return

    # 2. Output Directory
    output_dir = questionary.path("Select/Create output directory (files will be placed here):", only_directories=True).ask()
    if not output_dir:
        return
    if not os.path.exists(output_dir):
        try:
            os.makedirs(output_dir)
            console.print(f"[green]Created directory: {output_dir}[/green]")
        except OSError as e:
            console.print(f"[red]Error creating directory: {e}[/red]")
            return

    # 3. Chunking Options
    split_level = questionary.select(
        "Split at header level (files created for headers at this level and above):",
        choices=["1", "2", "3", "4", "5", "6"],
        default="1"
    ).ask()

    # 4. Lua Filters
    available_filters = get_lua_filters()
    selected_filters = []
    if available_filters:
        # Highlight potentially useful filters for chunking context
        console.print("\n[bold]Available Lua Filters:[/bold]")
        console.print("[dim]Filters run before chunking. Use them to modify content (e.g. 'pagebreak', 'include-files').[/dim]")
        
        filter_choices = [
            questionary.Choice(os.path.basename(f), value=f) for f in available_filters
        ]
        selected_filters = questionary.checkbox(
            "Select Lua filters to apply:",
            choices=filter_choices
        ).ask()

    # 5. Construct Command
    # pandoc input.html -t chunkedhtml --split-level=X --lua-filter=... -o output_dir
    
    cmd_parts = ["pandoc", f'"{input_file}"']
    
    # Use the specific chunkedhtml writer
    cmd_parts.append("-t chunkedhtml") 
    cmd_parts.append(f"--split-level={split_level}")
    
    for f in selected_filters:
        cmd_parts.append(f'--lua-filter="{f}"')
        
    # For chunkedhtml, -o is the directory (or a zip file)
    # We will output to the directory logic. 
    # Note: If -o ends in .zip, it zips it. If it has no extension, it treats as dir?
    # Actually, standard usage often requires just pointing to the dir if using a template, 
    # but strictly speaking `pandoc -t chunkedhtml -o output.zip` is the most robust cross-platform way 
    # to guarantee containerization without messy file spew.
    # However, user likely wants a folder.
    # Let's ask preference.
    
    output_mode = questionary.select(
        "Output format:",
        choices=["Directory (folder of HTML files)", "Zip Archive (.zip)"]
    ).ask()
    
    if output_mode == "Zip Archive (.zip)":
        output_path = os.path.join(output_dir, "output.zip")
    else:
        # Pandoc usually infers directory output if no extension, or explicitly using specific flags in older versions.
        # But for `chunkedhtml`, providing a directory path *as* the output argument works in modern versions.
        # Let's verify by just passing the dir path (ensure it doesn't have an extension).
        output_path = output_dir

    cmd_parts.append(f'-o "{output_path}"')

    final_cmd = " ".join(cmd_parts)
    
    console.print(Panel(f"[bold yellow]Command Construction:[/bold yellow]\n{final_cmd}", title="Review"))
    
    if questionary.confirm("Run this command?").ask():
        run_pandoc_command(final_cmd)
def main():
    console.print(Panel("[bold magenta]Pandoc Wizard[/bold magenta]", subtitle="Modular CLI Tool"))
    
    action = questionary.select(
        "What would you like to do?",
        choices=[
            "HTML Chunking",
            "Exit"
        ]
    ).ask()

    if action == "HTML Chunking":
        html_chunking_wizard()
    else:
        console.print("Goodbye!")

if __name__ == "__main__":
    main()
