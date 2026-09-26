"""
Generate demo results by processing the example audio file.
"""
import os
from pathlib import Path
from gemini_processor import (
    process_audio_with_gemini,
    create_markdown_file,
    create_json_file
)
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

def main():
    # Get API key from environment
    api_key = os.getenv('GEMINI_API')
    if not api_key:
        raise ValueError("GEMINI_API not found in .env file")

    # Path to example audio
    audio_path = "example-data/movie-prefs.opus"

    print(f"Processing {audio_path}...")

    # Process with Gemini (using "user" identification)
    context_markdown, human_readable_name, snake_case_filename = process_audio_with_gemini(
        audio_path,
        api_key,
        user_name=None  # Use "the user" format
    )

    print(f"Extracted context: {human_readable_name}")

    # Create output files
    md_filename, md_content = create_markdown_file(
        context_markdown,
        human_readable_name,
        snake_case_filename
    )

    json_filename, json_content = create_json_file(
        context_markdown,
        human_readable_name,
        snake_case_filename
    )

    # Create demo-results directory
    demo_dir = Path("demo-results")
    demo_dir.mkdir(exist_ok=True)

    # Write files
    md_path = demo_dir / md_filename
    json_path = demo_dir / json_filename

    with open(md_path, 'w') as f:
        f.write(md_content)
    print(f"Saved: {md_path}")

    with open(json_path, 'w') as f:
        f.write(json_content)
    print(f"Saved: {json_path}")

    print("\nDemo results generated successfully!")

if __name__ == "__main__":
    main()
